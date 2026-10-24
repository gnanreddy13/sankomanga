package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// serveCompressed sends one request through compressResponses wrapped
// around handler, optionally saying the browser accepts gzip.
func serveCompressed(handler http.HandlerFunc, acceptGzip bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if acceptGzip {
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	}
	rec := httptest.NewRecorder()
	compressResponses(handler).ServeHTTP(rec, req)
	return rec
}

func TestCompressesWhenAccepted(t *testing.T) {
	body := strings.Repeat(`{"title":"Shingeki no Kyojin"},`, 200) // repetitive, like real JSON

	rec := serveCompressed(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "12345") // must be removed
		writeJSON(w, http.StatusOK, body)
	}, true)

	if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", got)
	}
	if rec.Header().Get("Content-Length") != "" {
		t.Errorf("Content-Length should be removed for compressed responses")
	}
	if got := rec.Header().Get("Vary"); got != "Accept-Encoding" {
		t.Errorf("Vary = %q, want Accept-Encoding", got)
	}
	if rec.Body.Len() >= len(body) {
		t.Errorf("compressed body (%d bytes) isn't smaller than the original (%d)", rec.Body.Len(), len(body))
	}

	// Decompressing must give back exactly the original JSON.
	reader, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("body isn't valid gzip: %v", err)
	}
	unzipped, _ := io.ReadAll(reader)
	if !strings.Contains(string(unzipped), "Shingeki no Kyojin") || len(unzipped) < len(body) {
		t.Errorf("decompressed body is wrong: %d bytes", len(unzipped))
	}
}

func TestNoCompressionWhenNotAccepted(t *testing.T) {
	rec := serveCompressed(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
	}, false)

	if rec.Header().Get("Content-Encoding") != "" {
		t.Errorf("compressed for a browser that didn't ask for it")
	}
	if rec.Body.String() != `{"status":"ok"}` {
		t.Errorf("body = %s, want plain JSON", rec.Body.String())
	}
	// Even uncompressed answers must say they vary, for caches.
	if got := rec.Header().Get("Vary"); got != "Accept-Encoding" {
		t.Errorf("Vary = %q, want Accept-Encoding", got)
	}
}

func TestNoCompressionWithoutBody(t *testing.T) {
	rec := serveCompressed(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotModified)
	}, true)

	if rec.Header().Get("Content-Encoding") != "" {
		t.Errorf("a 304 has no body, so it must not claim to be gzip")
	}
	if rec.Body.Len() != 0 {
		t.Errorf("a 304 must have no body, got %d bytes", rec.Body.Len())
	}
}

func TestNoCompressionForRangeRequests(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Range", "bytes=0-10")
	rec := httptest.NewRecorder()

	compressResponses(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "some file contents")
	})).ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "" {
		t.Errorf("range requests must not be compressed")
	}
}
