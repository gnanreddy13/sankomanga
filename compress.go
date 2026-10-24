package main

// This file compresses responses with gzip, which makes our JSON, HTML,
// CSS and JavaScript roughly 3–5× smaller to send. That matters most on
// phones with slow mobile data. Browsers decompress it automatically.

import (
	"compress/gzip"
	"net/http"
	"strings"
)

// compressResponses wraps a handler so its responses are gzip-compressed
// for browsers that say they can handle it.
func compressResponses(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Tell caches that the response differs depending on whether the
		// browser accepts gzip, so they never hand compressed data to a
		// browser that can't read it.
		w.Header().Add("Vary", "Accept-Encoding")

		// Range requests ask for a slice of a file (e.g. resuming a
		// download); compressing would make the slice positions wrong.
		if !acceptsGzip(r) || r.Header.Get("Range") != "" {
			next.ServeHTTP(w, r)
			return
		}

		cw := &compressWriter{ResponseWriter: w}
		defer cw.finish()
		next.ServeHTTP(cw, r)
	})
}

// acceptsGzip reports whether the browser said it can read gzip.
func acceptsGzip(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
}

// compressWriter sits between a handler and the real response. Whatever
// the handler writes is compressed before it's sent on.
type compressWriter struct {
	http.ResponseWriter
	gz          *gzip.Writer // nil until we know the response has a body
	wroteHeader bool
}

// WriteHeader decides whether to compress: only responses that have a body.
// (204 No Content and 304 Not Modified never do.)
func (c *compressWriter) WriteHeader(status int) {
	if c.wroteHeader {
		return
	}
	c.wroteHeader = true

	if status != http.StatusNoContent && status != http.StatusNotModified {
		h := c.Header()
		h.Set("Content-Encoding", "gzip")
		// The original length no longer matches the compressed body.
		h.Del("Content-Length")
		c.gz = gzip.NewWriter(c.ResponseWriter)
	}
	c.ResponseWriter.WriteHeader(status)
}

// Write compresses the body (or passes it through when not compressing).
// Writing without calling WriteHeader first means "200 OK".
func (c *compressWriter) Write(b []byte) (int, error) {
	if !c.wroteHeader {
		c.WriteHeader(http.StatusOK)
	}
	if c.gz == nil {
		return c.ResponseWriter.Write(b)
	}
	return c.gz.Write(b)
}

// finish writes the end of the compressed data. It must run after the
// handler is done, or the browser would get a cut-off response.
func (c *compressWriter) finish() {
	if c.gz != nil {
		c.gz.Close()
	}
}

// Unwrap gives Go's http.ResponseController access to the real writer
// underneath, so standard features keep working through our wrapper.
func (c *compressWriter) Unwrap() http.ResponseWriter {
	return c.ResponseWriter
}
