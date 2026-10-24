package main

// A middleware is a function that wraps a handler: it runs some code
// before and after every request, without the handlers knowing about it.
// This file has the one that logs each request and adds the Server-Timing
// header, so you can see backend time in the browser's DevTools.

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"time"
)

// Values for the cache part of logs and the Server-Timing header.
const (
	cacheHit  = "hit"
	cacheMiss = "miss"
)

// requestInfo collects facts about one request while it's being handled,
// so the middleware can report them at the end. The cache (step 7c) fills
// in cacheResult through markCacheResult.
type requestInfo struct {
	cacheResult string // "", cacheHit, or cacheMiss
}

// requestInfoKey is the key for storing a *requestInfo in a request's
// context. Using our own private type (rather than a string like "info")
// means no other package can accidentally use the same key.
type requestInfoKey struct{}

// markCacheResult records whether this request was answered from the cache.
// It's safe to call even outside the middleware (e.g. in tests): if the
// context has no requestInfo, it does nothing.
func markCacheResult(ctx context.Context, result string) {
	if info, ok := ctx.Value(requestInfoKey{}).(*requestInfo); ok {
		info.cacheResult = result
	}
}

// logRequests wraps a handler so that every request:
//   - gets a Server-Timing header with how long our backend took, and
//     whether the cache was hit, e.g. `app;dur=3.1, cache;desc="hit"`
//   - is logged as one line, e.g. `method=GET path=/api/manga/1 status=200 dur_ms=3.1 cache=hit`
//
// Visitor IP addresses are deliberately not logged: we don't need them to
// debug, and leaving them out keeps personal data out of our logs.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Attach a fresh requestInfo to this request's context, so code
		// further in (the cache) can record facts about it.
		info := &requestInfo{}
		ctx := context.WithValue(r.Context(), requestInfoKey{}, info)

		rec := &timingRecorder{ResponseWriter: w, start: start, info: info, status: http.StatusOK}
		next.ServeHTTP(rec, r.WithContext(ctx))

		logRequest(r, rec.status, time.Since(start), info)
	})
}

// logRequest writes the one log line for a finished request. Server errors
// are logged at ERROR level so they stand out; everything else is INFO.
func logRequest(r *http.Request, status int, duration time.Duration, info *requestInfo) {
	level := slog.LevelInfo
	if status >= 500 {
		level = slog.LevelError
	}

	attrs := []any{
		"method", r.Method,
		// RequestURI includes the query string, e.g. /api/search?q=berserk
		"path", r.URL.RequestURI(),
		"status", status,
		"dur_ms", milliseconds(duration),
	}
	if info.cacheResult != "" {
		attrs = append(attrs, "cache", info.cacheResult)
	}

	slog.Log(r.Context(), level, "request", attrs...)
}

// timingRecorder wraps the real http.ResponseWriter so we can:
//   - remember the status code (for the log line), and
//   - add the Server-Timing header at the last possible moment.
//
// Headers can only be changed until the status code is sent, so the
// header is added inside WriteHeader, just before the real one is called.
// That also means Server-Timing measures the time until our answer was
// ready, which is exactly the backend time we want to see.
type timingRecorder struct {
	http.ResponseWriter
	start       time.Time
	info        *requestInfo
	status      int
	wroteHeader bool
}

func (t *timingRecorder) WriteHeader(status int) {
	if t.wroteHeader {
		return
	}
	t.wroteHeader = true
	t.status = status
	t.Header().Set("Server-Timing", serverTiming(time.Since(t.start), t.info.cacheResult))
	t.ResponseWriter.WriteHeader(status)
}

// Write sends part of the body. A handler may call Write without calling
// WriteHeader first, which means "200 OK", so we make sure our
// WriteHeader runs first in that case too.
func (t *timingRecorder) Write(b []byte) (int, error) {
	if !t.wroteHeader {
		t.WriteHeader(http.StatusOK)
	}
	return t.ResponseWriter.Write(b)
}

// Unwrap gives Go's http.ResponseController access to the real writer
// underneath, so standard features keep working through our wrapper.
func (t *timingRecorder) Unwrap() http.ResponseWriter {
	return t.ResponseWriter
}

// serverTiming builds the Server-Timing header value. The format is
// defined by a web standard, so browsers' DevTools can display it:
// "app;dur=3.1" means a step named "app" took 3.1 milliseconds.
func serverTiming(duration time.Duration, cacheResult string) string {
	value := fmt.Sprintf("app;dur=%.1f", milliseconds(duration))
	if cacheResult != "" {
		value += fmt.Sprintf(`, cache;desc=%q`, cacheResult)
	}
	return value
}

// milliseconds converts a duration to milliseconds, rounded to one decimal.
func milliseconds(d time.Duration) float64 {
	ms := float64(d) / float64(time.Millisecond)
	return math.Round(ms*10) / 10
}
