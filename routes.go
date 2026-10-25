package main

import "net/http"

// newRouter connects URL paths to handlers. It lives in its own function
// (instead of inside main) so tests can use the exact same routes.
func newRouter(a *app) http.Handler {
	mux := http.NewServeMux()

	// --- The website (web.go) ---
	site := siteFiles()
	// "/{$}" means exactly "/", not "everything starting with /".
	mux.HandleFunc("GET /{$}", servePage(site, "index.html"))
	// The results page. The query (?q=...) is read by search.js in the
	// browser, which then calls /api/search.
	mux.HandleFunc("GET /search", servePage(site, "search.html"))
	// A title page. The id is read by manga.js in the browser, which then
	// calls /api/manga/{id} (and shows "not found" for unknown ids).
	mux.HandleFunc("GET /manga/{id}", servePage(site, "manga.html"))
	// CSS, JavaScript, and other files: /static/css/base.css is the file
	// web/static/css/base.css.
	mux.HandleFunc("GET /static/", serveStatic(site))
	mux.HandleFunc("GET /favicon.ico", serveImage(site, "favicon.svg", "image/svg+xml"))
	mux.HandleFunc("GET /apple-touch-icon.png", serveImage(site, "apple-touch-icon.png", "image/png"))
	mux.HandleFunc("GET /og-image.png", serveImage(site, "og-image.png", "image/png"))
	// "/" matches every address no other route claims: answer those with
	// the styled "Page not found" page.
	mux.HandleFunc("/", serveNotFoundPage(site))

	// --- The JSON API ---

	// One per-IP limiter shared by all data endpoints, so a visitor's
	// allowance covers them together. Health checks aren't limited.
	limiter := newIPLimiter()

	mux.HandleFunc("/api/health", onlyGET(handleHealth))
	mux.HandleFunc("/api/search", onlyGET(limitPerIP(limiter, a.handleSearch)))
	// {id} is a path parameter: /api/manga/53390 sets it to "53390".
	mux.HandleFunc("/api/manga/{id}", onlyGET(limitPerIP(limiter, a.handleMangaDetail)))
	mux.HandleFunc("/api/showcase", onlyGET(limitPerIP(limiter, a.handleShowcase)))

	// "/api/" ending in a slash matches every path under /api/ that no
	// more specific route above claimed.
	mux.HandleFunc("/api/", handleAPINotFound)

	return mux
}
