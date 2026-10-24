package main

import "net/http"

// This file holds the public handlers: every visitor gets the same answer,
// so their responses are safe to cache and share. Any future user-specific
// handlers (accounts, settings) belong in a separate file and must never
// use the shared cache.

// healthResponse is the body of GET /api/health.
type healthResponse struct {
	Status string `json:"status"`
}

// handleHealth reports that the server is running. It deliberately does not
// call AniList: a health check should say whether *our* server is up, and
// it shouldn't spend AniList's rate limit every time a monitor pings it.
func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

// handleAPINotFound answers any /api/... path we don't recognise, so that
// even mistakes come back in our standard JSON error shape.
func handleAPINotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not found")
}

// onlyGET wraps a handler so that it only accepts GET (and HEAD, which is
// GET without the body and is what some uptime monitors use). Anything else
// gets a 405 in our JSON error shape. Go's router can filter methods itself,
// but its 405 response is plain text, which would break our error contract.
func onlyGET(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			// The Allow header tells the client which methods would work.
			w.Header().Set("Allow", "GET, HEAD")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		next(w, r)
	}
}
