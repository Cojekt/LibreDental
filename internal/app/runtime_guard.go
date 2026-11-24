package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Mirrors pkg/application/transport_http.go, transport_http_server.go and
// messageprocessor.go in Wails v3.2.0-nightly.20261103. Re-check these when bumping Wails.
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
	maxLostIDs      = 1024
)

// RejectApplicationRuntime refuses the Wails runtime Application object (Quit, Hide, Show)
// on /wails/runtime. Those calls are not bound service methods, so they skip the session
// checks every service enforces, and Quit stops the LAN server for every workstation.
//
// The guard reassembles chunked requests itself and forwards only whole requests, so its
// view of a call can never drift from the one Wails dispatches on. server must be the
// options passed to Wails, so the origin policy matches the transport's.
func RejectApplicationRuntime(logger *log.Logger, server application.ServerOptions) application.Middleware {
	return newRuntimeGuard(logger, server, time.Now).middleware
}

type runtimeGuard struct {
	logger *log.Logger
	server application.ServerOptions
	now    func() time.Time

	mu           sync.Mutex
	seq          uint64
	pending      map[string]*pendingChunks
	pendingBytes int
	// lost holds ids whose partly received upload was dropped, so the rest of that
	// upload fails instead of starting a new set that would end in an empty 200.
	lost map[string]time.Time
}

type pendingChunks struct {
	total   int
	chunks  map[int][]byte
	size    int
	created time.Time
	seq     uint64
}

func newRuntimeGuard(logger *log.Logger, server application.ServerOptions, now func() time.Time) *runtimeGuard {
	if logger == nil {
		logger = log.Default()
	}
	return &runtimeGuard{
		logger:  logger,
		server:  server,
		now:     now,
		pending: map[string]*pendingChunks{},
		lost:    map[string]time.Time{},
	}
}

func (g *runtimeGuard) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != runtimePath {
			next.ServeHTTP(w, r)
			return
		}
		// Wails only dispatches POST. Refusing everything else here, chunked or not, keeps a
		// later transport change from reopening the Application object or chunk handling.
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		// Same policy and order as Wails: no body is read or buffered for a refused origin.
		if !g.server.WebSocketAllowAllOrigins && !originAllowed(r, g.server.WebSocketOriginPatterns) {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		if r.Header.Get(chunkIDHeader) != "" {
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

// originAllowed mirrors Wails' authorizeRuntimeOrigin: requests without Origin are
// allowed unless Sec-Fetch-Site marks them cross-origin, the request host is always
// permitted, and patterns follow path.Match syntax.
func originAllowed(r *http.Request, patterns []string) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		switch r.Header.Get("Sec-Fetch-Site") {
		case "cross-site", "same-site":
			return false
		}
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	if strings.EqualFold(r.Host, u.Host) {
		return true
	}
	for _, pattern := range patterns {
		target := u.Host
		if strings.Contains(pattern, "://") {
			target = u.Scheme + "://" + u.Host
		}
		if matched, _ := path.Match(strings.ToLower(pattern), strings.ToLower(target)); matched {
			return true
		}
	}
	return false
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
	switch {
	case status == http.StatusServiceUnavailable:
		w.Header().Set("Retry-After", "1")
		http.Error(w, "upload interrupted, retry", status)
		return
	case status != 0:
		http.Error(w, http.StatusText(status), status)
		return
	case !complete:
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
// non-zero status means the request is refused; 503 means an upload this guard had
// partly accepted was dropped and must be retried from the start.
func (g *runtimeGuard) store(id string, total, index int, chunk []byte) ([]byte, bool, int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := g.now()
	for k, set := range g.pending {
		if now.Sub(set.created) > chunkTTL {
			g.drop(k, true)
		}
	}
	for k, at := range g.lost {
		if now.Sub(at) > chunkTTL {
			delete(g.lost, k)
		}
	}
	if _, ok := g.lost[id]; ok {
		return nil, false, http.StatusServiceUnavailable
	}

	set, exists := g.pending[id]
	if exists && set.total != total {
		g.drop(id, true)
		return nil, false, http.StatusUnprocessableEntity
	}
	if !exists {
		g.seq++
		set = &pendingChunks{total: total, chunks: map[int][]byte{}, created: now, seq: g.seq}
	}
	growth := len(chunk) - len(set.chunks[index])
	if set.size+growth > maxAssembledBytes {
		if exists {
			g.drop(id, true)
		}
		return nil, false, http.StatusRequestEntityTooLarge
	}
	g.pending[id] = set
	for len(g.pending) > maxPendingSets || g.pendingBytes+growth > maxPendingBytes {
		if g.dropOldest() == id {
			return nil, false, http.StatusServiceUnavailable
		}
	}

	set.chunks[index] = chunk
	set.size += growth
	g.pendingBytes += growth
	if len(set.chunks) < set.total {
		return nil, false, 0
	}

	g.drop(id, false)
	body := make([]byte, 0, set.size)
	for i := range set.total {
		body = append(body, set.chunks[i]...)
	}
	return body, true, 0
}

func (g *runtimeGuard) dropOldest() string {
	oldest := ""
	for k, set := range g.pending {
		if oldest == "" || set.seq < g.pending[oldest].seq {
			oldest = k
		}
	}
	g.drop(oldest, true)
	return oldest
}

func (g *runtimeGuard) drop(id string, lost bool) {
	set, ok := g.pending[id]
	if !ok {
		return
	}
	g.pendingBytes -= set.size
	delete(g.pending, id)
	if !lost {
		return
	}
	if len(g.lost) >= maxLostIDs {
		for k := range g.lost {
			delete(g.lost, k)
			break
		}
	}
	g.lost[id] = g.now()
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
