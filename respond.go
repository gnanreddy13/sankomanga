package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// errorResponse is the single JSON shape used for every error: {"error": "..."}.
type errorResponse struct {
	Error string `json:"error"`
}

// writeJSON sends v as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	// Encode first, before writing anything. If encoding failed halfway through
	// writing, the client would get a 200 status with a broken body.
	body, err := json.Marshal(v)
	if err != nil {
		slog.Error("encoding JSON response", "err", err)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal error"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	w.Write(body)
}

// Cache-Control header values. This header tells browsers (and any proxy
// in between) whether, and for how long, they may reuse a response
// without asking us again.
const (
	// cacheControlPublic: any cache may keep the answer for 5 minutes.
	// Only for responses that are the same for every visitor. Future
	// user-specific responses must use "private" or "no-store" instead.
	cacheControlPublic = "public, max-age=300"

	// cacheControlNoStore: never keep the answer. Used for errors (a
	// "try again shortly" must not be replayed for 5 minutes) and for the
	// health check (which must always reflect the server right now).
	cacheControlNoStore = "no-store"
)

// writeError sends an error in the standard {"error": "..."} shape.
// Errors are never cached by browsers or proxies.
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Cache-Control", cacheControlNoStore)
	writeJSON(w, status, errorResponse{Error: message})
}
