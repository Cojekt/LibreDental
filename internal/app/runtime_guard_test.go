package app

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type nextRecorder struct {
	calls int
	body  []byte
	req   *http.Request
}

func (n *nextRecorder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	n.calls++
	n.body, _ = io.ReadAll(r.Body)
	n.req = r
	w.WriteHeader(http.StatusTeapot)
}

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func newTestGuard() (*runtimeGuard, http.Handler, *nextRecorder, *clock) {
	return newTestGuardWith(application.ServerOptions{})
}

func newTestGuardWith(server application.ServerOptions) (*runtimeGuard, http.Handler, *nextRecorder, *clock) {
	c := &clock{t: time.Unix(1_700_000_000, 0)}
	g := newRuntimeGuard(log.New(io.Discard, "", 0), server, c.now)
	next := &nextRecorder{}
	return g, g.middleware(next), next, c
}

func send(h http.Handler, method, target string, body []byte, header map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func sendChunk(h http.Handler, id string, index, total int, body string) *httptest.ResponseRecorder {
	return send(h, http.MethodPost, runtimePath, []byte(body), map[string]string{
		chunkIDHeader:    id,
		chunkIndexHeader: strconv.Itoa(index),
		chunkTotalHeader: strconv.Itoa(total),
	})
}

func TestRuntimeGuard_RefusesApplicationObject(t *testing.T) {
	cases := []struct {
		name, method, target, body string
		origin                     bool
	}{
		{"quit", http.MethodPost, runtimePath, `{"object":2,"method":2}`, false},
		{"quit with origin", http.MethodPost, runtimePath, `{"object":2,"method":2}`, true},
		{"hide", http.MethodPost, runtimePath, `{"object":2,"method":0}`, true},
		{"show", http.MethodPost, runtimePath, `{"object":2,"method":1}`, false},
		{"unknown method", http.MethodPost, runtimePath, `{"object":2,"method":99}`, false},
		{"case-insensitive key", http.MethodPost, runtimePath, `{"OBJECT":2,"method":2}`, false},
		{"last duplicate wins", http.MethodPost, runtimePath, `{"object":0,"object":2,"method":2}`, false},
		{"query", http.MethodPost, runtimePath + "?object=2&method=2", ``, false},
		{"query with plus", http.MethodPost, runtimePath + "?object=%2B2&method=2", ``, false},
		{"query with leading zero", http.MethodPost, runtimePath + "?object=02&method=2", ``, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, h, next, _ := newTestGuard()
			header := map[string]string{}
			if tc.origin {
				header["Origin"] = "http://example.com"
			}
			rec := send(h, tc.method, tc.target, []byte(tc.body), header)
			if rec.Code != http.StatusForbidden {
				t.Errorf("status = %d, want 403", rec.Code)
			}
			if next.calls != 0 {
				t.Errorf("next called %d times", next.calls)
			}
		})
	}
}

func TestRuntimeGuard_PassesThrough(t *testing.T) {
	cases := []struct{ name, target, body string }{
		{"service call", runtimePath, `{"object":0,"method":0,"args":{"name":"x"}}`},
		{"body wins over query", runtimePath + "?object=2&method=2", `{"object":0,"method":0}`},
		{"invalid JSON", runtimePath, `{"object":2`},
		{"missing object", runtimePath, `{"method":2}`},
		{"other path", "/health", `{"object":2,"method":2}`},
		{"trailing slash", runtimePath + "/", `{"object":2,"method":2}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, h, next, _ := newTestGuard()
			send(h, http.MethodPost, tc.target, []byte(tc.body), nil)
			if next.calls != 1 {
				t.Fatalf("next called %d times, want 1", next.calls)
			}
			if string(next.body) != tc.body {
				t.Errorf("next read %q, want %q", next.body, tc.body)
			}
		})
	}
}

func TestRuntimeGuard_ChunkedApplicationCallRefused(t *testing.T) {
	for _, order := range [][]int{{0, 1}, {1, 0}} {
		t.Run(fmt.Sprint(order), func(t *testing.T) {
			_, h, next, _ := newTestGuard()
			parts := []string{`{"object":2,`, `"method":2}`}
			if rec := sendChunk(h, "a", order[0], 2, parts[order[0]]); rec.Code != http.StatusOK {
				t.Fatalf("first chunk status = %d, want 200", rec.Code)
			}
			if rec := sendChunk(h, "a", order[1], 2, parts[order[1]]); rec.Code != http.StatusForbidden {
				t.Errorf("completing chunk status = %d, want 403", rec.Code)
			}
			if next.calls != 0 {
				t.Errorf("next called %d times", next.calls)
			}
		})
	}
}

func TestRuntimeGuard_ChunkedServiceCallForwardedWhole(t *testing.T) {
	g, h, next, _ := newTestGuard()
	sendChunk(h, "a", 0, 2, `{"object":0,`)
	if next.calls != 0 {
		t.Fatal("an incomplete chunk reached next")
	}
	sendChunk(h, "a", 1, 2, `"method":0}`)
	if next.calls != 1 {
		t.Fatalf("next called %d times, want 1", next.calls)
	}
	want := `{"object":0,"method":0}`
	if string(next.body) != want {
		t.Errorf("next read %q, want %q", next.body, want)
	}
	if next.req.ContentLength != int64(len(want)) {
		t.Errorf("ContentLength = %d, want %d", next.req.ContentLength, len(want))
	}
	for _, hdr := range []string{chunkIDHeader, chunkIndexHeader, chunkTotalHeader} {
		if next.req.Header.Get(hdr) != "" {
			t.Errorf("forwarded request still has %s", hdr)
		}
	}
	if len(g.pending) != 0 || g.pendingBytes != 0 {
		t.Errorf("pending = %d sets, %d bytes after completion", len(g.pending), g.pendingBytes)
	}
}

func TestRuntimeGuard_RepeatedIndexOverwrites(t *testing.T) {
	g, h, next, _ := newTestGuard()
	sendChunk(h, "a", 0, 2, `{"object":2,`)
	sendChunk(h, "a", 0, 2, `{"object":0,`)
	if g.pendingBytes != len(`{"object":0,`) {
		t.Errorf("pendingBytes = %d after overwrite", g.pendingBytes)
	}
	sendChunk(h, "a", 1, 2, `"method":0}`)
	if next.calls != 1 || string(next.body) != `{"object":0,"method":0}` {
		t.Errorf("next calls = %d body = %q", next.calls, next.body)
	}
}

func TestRuntimeGuard_InvalidChunks(t *testing.T) {
	cases := []struct {
		name         string
		index, total string
	}{
		{"zero total", "0", "0"},
		{"total over limit", "0", strconv.Itoa(maxChunkTotal + 1)},
		{"index out of range", "2", "2"},
		{"negative index", "-1", "2"},
		{"non-numeric", "x", "2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, h, next, _ := newTestGuard()
			rec := send(h, http.MethodPost, runtimePath, []byte(`{"object":0}`), map[string]string{
				chunkIDHeader: "a", chunkIndexHeader: tc.index, chunkTotalHeader: tc.total,
			})
			if rec.Code != http.StatusUnprocessableEntity {
				t.Errorf("status = %d, want 422", rec.Code)
			}
			if next.calls != 0 {
				t.Errorf("next called %d times", next.calls)
			}
		})
	}
}

func TestRuntimeGuard_MismatchedTotalDropsSet(t *testing.T) {
	g, h, next, _ := newTestGuard()
	sendChunk(h, "a", 0, 3, `{"object":0,`)
	if rec := sendChunk(h, "a", 1, 2, `"method":0}`); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
	if _, ok := g.pending["a"]; ok || g.pendingBytes != 0 {
		t.Error("set survived a mismatched total")
	}
	if rec := sendChunk(h, "a", 1, 2, `"method":0}`); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("chunk after a dropped set status = %d, want 503", rec.Code)
	}
	if next.calls != 0 {
		t.Errorf("next called %d times", next.calls)
	}
}

func TestRuntimeGuard_NonPOSTRefused(t *testing.T) {
	cases := []struct {
		name   string
		target string
		header map[string]string
	}{
		{"application object", runtimePath + "?object=2&method=2", nil},
		{"service call", runtimePath + "?object=0&method=0", nil},
		{"chunk", runtimePath, map[string]string{chunkIDHeader: "a", chunkIndexHeader: "0", chunkTotalHeader: "2"}},
	}
	for _, tc := range cases {
		for _, method := range []string{http.MethodGet, http.MethodPut} {
			t.Run(tc.name+" "+method, func(t *testing.T) {
				g, h, next, _ := newTestGuard()
				rec := send(h, method, tc.target, []byte(`{"object":0}`), tc.header)
				if rec.Code != http.StatusMethodNotAllowed {
					t.Errorf("status = %d, want 405", rec.Code)
				}
				if next.calls != 0 || len(g.pending) != 0 {
					t.Errorf("next calls = %d, pending sets = %d", next.calls, len(g.pending))
				}
			})
		}
	}
}

type readFlag struct {
	io.Reader
	read bool
}

func (r *readFlag) Read(p []byte) (int, error) {
	r.read = true
	return r.Reader.Read(p)
}

func TestRuntimeGuard_OriginCheckedBeforeBuffering(t *testing.T) {
	cases := []struct {
		name   string
		header map[string]string
	}{
		{"cross-origin", map[string]string{"Origin": "http://evil.test"}},
		{"invalid origin", map[string]string{"Origin": "null"}},
		{"cross-site fetch", map[string]string{"Sec-Fetch-Site": "cross-site"}},
		{"cross-origin chunk", map[string]string{
			"Origin": "http://evil.test", chunkIDHeader: "a", chunkIndexHeader: "0", chunkTotalHeader: "2",
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g, h, next, _ := newTestGuard()
			body := &readFlag{Reader: strings.NewReader(`{"object":0}`)}
			req := httptest.NewRequest(http.MethodPost, runtimePath, body)
			for k, v := range tc.header {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusForbidden {
				t.Errorf("status = %d, want 403", rec.Code)
			}
			if body.read || next.calls != 0 || len(g.pending) != 0 {
				t.Errorf("body read = %v, next calls = %d, pending sets = %d", body.read, next.calls, len(g.pending))
			}
		})
	}
}

func TestRuntimeGuard_OriginPolicyMatchesServerOptions(t *testing.T) {
	cases := []struct {
		name   string
		server application.ServerOptions
		origin string
	}{
		{"same host", application.ServerOptions{}, "http://example.com"},
		{"host pattern", application.ServerOptions{WebSocketOriginPatterns: []string{"*.lan.test"}}, "http://desk.lan.test"},
		{"scheme pattern", application.ServerOptions{WebSocketOriginPatterns: []string{"https://lan.test"}}, "https://lan.test"},
		{"allow all", application.ServerOptions{WebSocketAllowAllOrigins: true}, "http://evil.test"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, h, next, _ := newTestGuardWith(tc.server)
			send(h, http.MethodPost, runtimePath, []byte(`{"object":0}`), map[string]string{"Origin": tc.origin})
			if next.calls != 1 {
				t.Errorf("next called %d times, want 1", next.calls)
			}
		})
	}
}

func TestRuntimeGuard_SizeLimits(t *testing.T) {
	_, h, next, _ := newTestGuard()
	if rec := sendChunk(h, "a", 0, 2, strings.Repeat("x", maxChunkBodyBytes+1)); rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("oversized chunk status = %d, want 413", rec.Code)
	}
	big := append([]byte(`{"object":0,"args":"`), bytes.Repeat([]byte("x"), maxAssembledBytes)...)
	if rec := send(h, http.MethodPost, runtimePath, big, nil); rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("oversized body status = %d, want 413", rec.Code)
	}
	if next.calls != 0 {
		t.Errorf("next called %d times", next.calls)
	}
}

// Before the guard reassembled chunks itself, evicting a partly sent set from the guard
// while Wails still held it let the completing chunk through.
func TestRuntimeGuard_EvictionCannotCompleteRefusedCall(t *testing.T) {
	g, h, next, c := newTestGuard()
	sendChunk(h, "quit", 0, 2, `{"object":2,`)
	for i := range maxPendingSets {
		c.t = c.t.Add(time.Millisecond)
		sendChunk(h, fmt.Sprintf("junk-%d", i), 0, 2, `x`)
	}
	if _, ok := g.pending["quit"]; ok {
		t.Fatal("oldest set was not evicted")
	}
	if len(g.pending) > maxPendingSets {
		t.Errorf("%d pending sets, limit %d", len(g.pending), maxPendingSets)
	}
	if rec := sendChunk(h, "quit", 1, 2, `"method":2}`); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("final chunk status = %d, want 503", rec.Code)
	}
	if next.calls != 0 {
		t.Errorf("next called %d times", next.calls)
	}
}

// The browser runtime treats an empty 200 on the final chunk as a successful call, so an
// upload that lost its earlier chunks must fail instead of starting a new set.
func TestRuntimeGuard_InterruptedUploadFails(t *testing.T) {
	t.Run("expired", func(t *testing.T) {
		g, h, next, c := newTestGuard()
		sendChunk(h, "a", 0, 3, `{"object":0,`)
		c.t = c.t.Add(chunkTTL + time.Second)
		for i := 1; i < 3; i++ {
			rec := sendChunk(h, "a", i, 3, `"method":0}`)
			if rec.Code != http.StatusServiceUnavailable || rec.Header().Get("Retry-After") == "" {
				t.Errorf("chunk %d status = %d, want retryable 503", i, rec.Code)
			}
		}
		if next.calls != 0 || len(g.pending) != 0 || g.pendingBytes != 0 {
			t.Errorf("next calls = %d, pending = %d sets, %d bytes", next.calls, len(g.pending), g.pendingBytes)
		}
	})
	t.Run("evicted", func(t *testing.T) {
		_, h, next, c := newTestGuard()
		sendChunk(h, "a", 0, 2, `{"object":0,`)
		for i := range maxPendingSets {
			c.t = c.t.Add(time.Millisecond)
			sendChunk(h, fmt.Sprintf("other-%d", i), 0, 2, `x`)
		}
		if rec := sendChunk(h, "a", 1, 2, `"method":0}`); rec.Code != http.StatusServiceUnavailable {
			t.Errorf("final chunk status = %d, want 503", rec.Code)
		}
		if next.calls != 0 {
			t.Errorf("next called %d times", next.calls)
		}
	})
	t.Run("lost ids expire", func(t *testing.T) {
		g, h, _, c := newTestGuard()
		sendChunk(h, "a", 0, 2, `{"object":0,`)
		c.t = c.t.Add(chunkTTL + time.Second)
		sendChunk(h, "b", 0, 2, `x`)
		c.t = c.t.Add(chunkTTL + time.Second)
		sendChunk(h, "c", 0, 2, `x`)
		if _, ok := g.lost["a"]; ok {
			t.Error("lost id kept past its TTL")
		}
	})
}

func TestRuntimeGuard_ByteBudgetEvictsOldestFirst(t *testing.T) {
	chunk := strings.Repeat("x", maxChunkBodyBytes)

	t.Run("new set over budget", func(t *testing.T) {
		g, h, _, _ := newTestGuard()
		for i := range 40 {
			sendChunk(h, "a", i, 64, chunk)
		}
		for i := range 30 {
			if rec := sendChunk(h, "b", i, 64, chunk); rec.Code != http.StatusOK {
				t.Fatalf("chunk %d of newer set status = %d", i, rec.Code)
			}
		}
		if g.pendingBytes > maxPendingBytes {
			t.Errorf("pendingBytes = %d, budget %d", g.pendingBytes, maxPendingBytes)
		}
		if _, ok := g.pending["a"]; ok {
			t.Error("older set kept past the byte budget")
		}
		if set := g.pending["b"]; set == nil || len(set.chunks) != 30 {
			t.Error("newer set lost chunks")
		}
	})

	t.Run("oldest set grows over budget", func(t *testing.T) {
		g, h, _, _ := newTestGuard()
		sendChunk(h, "a", 0, 64, chunk)
		for i := range 63 {
			sendChunk(h, "b", i, 64, chunk)
		}
		if rec := sendChunk(h, "a", 1, 64, chunk); rec.Code != http.StatusServiceUnavailable {
			t.Errorf("status = %d, want 503 for the evicted oldest set", rec.Code)
		}
		if _, ok := g.pending["a"]; ok {
			t.Error("oldest set kept past the byte budget")
		}
		if set := g.pending["b"]; set == nil || len(set.chunks) != 63 {
			t.Error("newer set was evicted instead of the oldest")
		}
		if g.pendingBytes > maxPendingBytes {
			t.Errorf("pendingBytes = %d, budget %d", g.pendingBytes, maxPendingBytes)
		}
	})
}
