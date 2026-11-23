package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Mirrors pkg/application/transport_http.go and messageprocessor.go in Wails
// v3.2.0-nightly.20261103. Re-check these when bumping Wails.
const (
	runtimePath       = "/wails/runtime"
	applicationObject = 2

	chunkIDHeader    = "x-wails-chunk-id"
	chunkIndexHeader = "x-wails-chunk-index"
	chunkTotalHeader = "x-wails-chunk-total"
	chunkTTL         = 30 * time.Second

	maxChunkTotal     = 1024
	maxChunkBodyBytes = 1024 * 1024
	maxAssembledBytes = 64 * 1024 * 1024

	maxPendingSets  = 32
	maxPendingBytes = maxAssembledBytes
)

// RejectApplicationRuntime refuses the Wails runtime Application object (Quit, Hide, Show)
// on /wails/runtime. Those calls are not bound service methods, so they skip the session
// checks every service enforces, and Quit stops the LAN server for every workstation.
//
// The guard reassembles chunked requests itself and forwards only whole requests, so its
// view of a call can never drift from the one Wails dispatches on.
func RejectApplicationRuntime(logger *log.Logger) application.Middleware {
	return newRuntimeGuard(logger, time.Now).middleware
}

type runtimeGuard struct {
	logger *log.Logger
	now    func() time.Time

	mu           sync.Mutex
	pending      map[string]*pendingChunks
	pendingBytes int
}

type pendingChunks struct {
	total   int
	chunks  map[int][]byte
	size    int
	created time.Time
}

func newRuntimeGuard(logger *log.Logger, now func() time.Time) *runtimeGuard {
	if logger == nil {
		logger = log.Default()
	}
	return &runtimeGuard{logger: logger, now: now, pending: map[string]*pendingChunks{}}
}

func (g *runtimeGuard) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != runtimePath {
			next.ServeHTTP(w, r)
			return
		}
		// Wails rejects non-POST before looking at chunk headers, so only POST is buffered.
		if r.Method == http.MethodPost && r.Header.Get(chunkIDHeader) != "" {
			g.serveChunk(w, r, next)
			return
		}
		body, ok := readLimited(w, r, maxAssembledBytes)
		if !ok {
			return
		}
		g.dispatch(w, r, body, next)
	})
}

func (g *runtimeGuard) dispatch(w http.ResponseWriter, r *http.Request, body []byte, next http.Handler) {
	if obj, ok := runtimeObject(r, body); ok && obj == applicationObject {
		g.logger.Printf("refused application runtime call from %s", r.RemoteAddr)
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	r.ContentLength = int64(len(body))
	next.ServeHTTP(w, r)
}

// runtimeObject resolves the object id exactly as Wails' processBody does: the JSON body
// when there is one (encoding/json, so keys match case-insensitively and the last
// duplicate wins), otherwise the query string via strconv.Atoi.
func runtimeObject(r *http.Request, body []byte) (int, bool) {
	if len(body) > 0 {
		var req struct {
			Object *int `json:"object"`
		}
		if json.Unmarshal(body, &req) != nil || req.Object == nil {
			return 0, false
		}
		return *req.Object, true
	}
	obj, err := strconv.Atoi(r.URL.Query().Get("object"))
	if err != nil {
		return 0, false
	}
	return obj, true
}

func (g *runtimeGuard) serveChunk(w http.ResponseWriter, r *http.Request, next http.Handler) {
	total, err := strconv.Atoi(r.Header.Get(chunkTotalHeader))
	if err != nil || total <= 0 || total > maxChunkTotal {
		http.Error(w, "invalid chunk total", http.StatusUnprocessableEntity)
		return
	}
	index, err := strconv.Atoi(r.Header.Get(chunkIndexHeader))
	if err != nil || index < 0 || index >= total {
		http.Error(w, "invalid chunk index", http.StatusUnprocessableEntity)
		return
	}
	chunk, ok := readLimited(w, r, maxChunkBodyBytes)
	if !ok {
		return
	}

	body, complete, status := g.store(r.Header.Get(chunkIDHeader), total, index, chunk)
	if status != 0 {
		http.Error(w, http.StatusText(status), status)
		return
	}
	if !complete {
		w.WriteHeader(http.StatusOK)
		return
	}

	whole := r.Clone(r.Context())
	whole.Header.Del(chunkIDHeader)
	whole.Header.Del(chunkIndexHeader)
	whole.Header.Del(chunkTotalHeader)
	whole.TransferEncoding = nil
	g.dispatch(w, whole, body, next)
}

// store records one chunk and returns the assembled body once the set is complete. A
// non-zero status means the request is refused and nothing was stored.
func (g *runtimeGuard) store(id string, total, index int, chunk []byte) ([]byte, bool, int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := g.now()
	for k, set := range g.pending {
		if now.Sub(set.created) > chunkTTL {
			g.drop(k)
		}
	}

	set, exists := g.pending[id]
	if exists && set.total != total {
		g.drop(id)
		return nil, false, http.StatusUnprocessableEntity
	}
	if !exists {
		set = &pendingChunks{total: total, chunks: map[int][]byte{}, created: now}
	}
	growth := len(chunk) - len(set.chunks[index])
	if set.size+growth > maxAssembledBytes {
		if exists {
			g.drop(id)
		}
		return nil, false, http.StatusRequestEntityTooLarge
	}
	g.pending[id] = set
	for len(g.pending) > maxPendingSets || g.pendingBytes+growth > maxPendingBytes {
		if !g.evictOldest(id) {
			g.drop(id)
			return nil, false, http.StatusRequestEntityTooLarge
		}
	}

	set.chunks[index] = chunk
	set.size += growth
	g.pendingBytes += growth
	if len(set.chunks) < set.total {
		return nil, false, 0
	}

	g.drop(id)
	body := make([]byte, 0, set.size)
	for i := range set.total {
		body = append(body, set.chunks[i]...)
	}
	return body, true, 0
}

func (g *runtimeGuard) evictOldest(keep string) bool {
	oldest := ""
	for k, set := range g.pending {
		if k != keep && (oldest == "" || set.created.Before(g.pending[oldest].created)) {
			oldest = k
		}
	}
	if oldest == "" {
		return false
	}
	g.drop(oldest)
	return true
}

func (g *runtimeGuard) drop(id string) {
	if set, ok := g.pending[id]; ok {
		g.pendingBytes -= set.size
		delete(g.pending, id)
	}
}

func readLimited(w http.ResponseWriter, r *http.Request, limit int64) ([]byte, bool) {
	if r.Body == nil {
		return nil, true
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limit))
	if err != nil {
		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
		} else {
			http.Error(w, "unable to read request body", http.StatusUnprocessableEntity)
		}
		return nil, false
	}
	return body, true
}
