package main

// This file serves the website itself: the HTML pages, CSS and JavaScript
// in the web/ folder. They're served by the same Go server as the API, so
// the browser sees one site and no CORS setup is needed.

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

// webFiles holds everything in the web/ folder. The //go:embed line below
// tells Go to copy those files *into the compiled program*, so deploying
// is still a single file: there's no separate folder to upload.
//
//go:embed web
var webFiles embed.FS

// siteFiles returns the web/ folder as a file system rooted at web/, so
// "index.html" means web/index.html.
func siteFiles() fs.FS {
	site, err := fs.Sub(webFiles, "web")
	if err != nil {
		// Only possible if the embed line above were wrong, which the
		// compiler would already have caught; so this never happens.
		panic(err)
	}
	return site
}

// cacheControlSite: browsers must check with us before reusing a page or
// file. While the site is being built that means changes always show up
// on reload. (Once the frontend settles, files can get longer caching.)
const cacheControlSite = "no-cache"

// servePage returns a handler that sends one HTML page from the site.
func servePage(site fs.FS, name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", cacheControlSite)
		http.ServeFileFS(w, r, site, name)
	}
}

// serveStatic returns a handler for the CSS, JavaScript and other files
// under /static/. Go's file server would show a list of files for folder
// paths like /static/js/; we answer those with 404 instead, since there's
// no reason to advertise what's in there.
func serveStatic(site fs.FS) http.HandlerFunc {
	files := http.FileServerFS(site)
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", cacheControlSite)
		files.ServeHTTP(w, r)
	}
}
