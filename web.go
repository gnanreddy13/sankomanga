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

// serveNotFoundPage returns a handler that answers with the site's styled
// "Page not found" page (web/404.html) and the 404 status code, for any
// address that isn't a page of the site. (http.ServeFileFS always answers
// 200, so we send the file ourselves.)
func serveNotFoundPage(site fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := fs.ReadFile(site, "404.html")
		if err != nil {
			http.NotFound(w, r) // only if the file were missing from the build
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", cacheControlSite)
		w.WriteHeader(http.StatusNotFound)
		w.Write(page)
	}
}

// cacheControlFavicon: the tab icon rarely changes, so browsers may keep
// it for a day instead of asking for it on every page.
const cacheControlFavicon = "public, max-age=86400"

// serveFavicon answers /favicon.ico, which browsers ask for on their own.
// Our icon is an SVG (sharp at every size); modern browsers accept it at
// this address as long as it's labelled as SVG.
func serveFavicon(site fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		icon, err := fs.ReadFile(site, "favicon.svg")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", cacheControlFavicon)
		w.Write(icon)
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
