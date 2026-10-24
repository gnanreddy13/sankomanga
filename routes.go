package main

import "net/http"

// newRouter connects URL paths to handlers. It lives in its own function
// (instead of inside main) so tests can use the exact same routes.
func newRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", onlyGET(handleHealth))

	// "/api/" ending in a slash matches every path under /api/ that no
	// more specific route above claimed.
	mux.HandleFunc("/api/", handleAPINotFound)

	return mux
}
