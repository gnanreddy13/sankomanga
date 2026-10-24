package main

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

// captureLogs sends all slog output to a buffer for the rest of the test,
// so the test can check what was logged. The normal logger is put back
// when the test finishes.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &buf
}

// serveThroughMiddleware runs one GET request through logRequests
// wrapped around the given handler.
func serveThroughMiddleware(path string, handler http.HandlerFunc) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	logRequests(handler).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

// assertLogContains checks that every expected key=value piece is in the log.
func assertLogContains(t *testing.T, logs string, want ...string) {
	t.Helper()
	for _, piece := range want {
		if !strings.Contains(logs, piece) {
			t.Errorf("log is missing %q\nlog: %s", piece, logs)
		}
	}
}

func TestLogRequestsWithoutCache(t *testing.T) {
	logs := captureLogs(t)

	rec := serveThroughMiddleware("/api/manga/7", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "manga not found")
	})

	// e.g. "app;dur=0.1" and nothing about the cache.
	if got := rec.Header().Get("Server-Timing"); !regexp.MustCompile(`^app;dur=\d+\.\d$`).MatchString(got) {
		t.Errorf("Server-Timing = %q, want app;dur=<ms>", got)
	}
	// The handler's own response must pass through untouched.
	if rec.Code != http.StatusNotFound || rec.Body.String() != `{"error":"manga not found"}` {
		t.Errorf("response changed: %d %s", rec.Code, rec.Body.String())
	}

	assertLogContains(t, logs.String(), "level=INFO", "msg=request", "method=GET", "path=/api/manga/7", "status=404", "dur_ms=")
	if strings.Contains(logs.String(), "cache=") {
		t.Errorf("log mentions the cache, but nothing used it: %s", logs.String())
	}
}

func TestLogRequestsWithCacheResult(t *testing.T) {
	for _, result := range []string{cacheHit, cacheMiss} {
		t.Run(result, func(t *testing.T) {
			logs := captureLogs(t)

			rec := serveThroughMiddleware("/api/manga/7", func(w http.ResponseWriter, r *http.Request) {
				markCacheResult(r.Context(), result) // what the cache will do in step 7c
				writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
			})

			want := regexp.MustCompile(`^app;dur=\d+\.\d, cache;desc="` + result + `"$`)
			if got := rec.Header().Get("Server-Timing"); !want.MatchString(got) {
				t.Errorf("Server-Timing = %q, want app;dur=<ms>, cache;desc=%q", got, result)
			}
			assertLogContains(t, logs.String(), "status=200", "cache="+result)
		})
	}
}

func TestLogRequestsServerErrorIsLoggedAsError(t *testing.T) {
	logs := captureLogs(t)

	serveThroughMiddleware("/api/search?q=x", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusBadGateway, "upstream service error")
	})

	assertLogContains(t, logs.String(), "level=ERROR", "status=502")
}

func TestLogRequestsWriteWithoutWriteHeader(t *testing.T) {
	logs := captureLogs(t)

	// A handler that only calls Write means "200 OK".
	rec := serveThroughMiddleware("/x", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello")
	})

	if rec.Header().Get("Server-Timing") == "" {
		t.Errorf("Server-Timing header missing")
	}
	assertLogContains(t, logs.String(), "status=200")
}

func TestMarkCacheResultWithoutMiddleware(t *testing.T) {
	// Must not crash when there's no requestInfo in the context.
	markCacheResult(context.Background(), cacheHit)
}

func TestServerTiming(t *testing.T) {
	tests := []struct {
		duration    time.Duration
		cacheResult string
		want        string
	}{
		{1234567 * time.Nanosecond, "", "app;dur=1.2"},
		{250 * time.Millisecond, cacheMiss, `app;dur=250.0, cache;desc="miss"`},
		{80 * time.Microsecond, cacheHit, `app;dur=0.1, cache;desc="hit"`},
	}
	for _, tt := range tests {
		if got := serverTiming(tt.duration, tt.cacheResult); got != tt.want {
			t.Errorf("serverTiming(%v, %q) = %q, want %q", tt.duration, tt.cacheResult, got, tt.want)
		}
	}
}
