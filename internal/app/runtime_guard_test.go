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
	c := &clock{t: time.Unix(1_700_000_000, 0)}
	g := newRuntimeGuard(log.New(io.Discard, "", 0), c.now)
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
		{"non-POST", http.MethodGet, runtimePath + "?object=2&method=2", ``, false},
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
	if next.calls != 0 {
		t.Errorf("next called %d times", next.calls)
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
	sendChunk(h, "quit", 1, 2, `"method":2}`)
	if next.calls != 0 {
		t.Errorf("next called %d times", next.calls)
	}
}

func TestRuntimeGuard_ExpiredSetIsNotCompleted(t *testing.T) {
	g, h, next, c := newTestGuard()
	sendChunk(h, "a", 0, 2, `{"object":0,`)
	c.t = c.t.Add(chunkTTL + time.Second)
	if rec := sendChunk(h, "a", 1, 2, `"method":0}`); rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (stored as a new set)", rec.Code)
	}
	if next.calls != 0 {
		t.Errorf("next called %d times", next.calls)
	}
	if g.pendingBytes != len(`"method":0}`) {
		t.Errorf("pendingBytes = %d, expired chunk not released", g.pendingBytes)
	}
}

func TestRuntimeGuard_GlobalByteBudget(t *testing.T) {
	g, h, _, c := newTestGuard()
	chunk := strings.Repeat("x", maxChunkBodyBytes)
	for i := range 40 {
		sendChunk(h, "a", i, 64, chunk)
	}
	c.t = c.t.Add(time.Millisecond)
	for i := range 30 {
		sendChunk(h, "b", i, 64, chunk)
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
}
