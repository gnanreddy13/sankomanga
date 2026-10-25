package main

// This file serves the website itself: the HTML pages, CSS and JavaScript
// in the web/ folder. They're served by the same Go server as the API, so
// the browser sees one site and no CORS setup is needed.

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"io/fs"
	"net/http"
	"strings"
	"time"
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
// file, so a new version always shows up straight away. That check is
// cheap: each page and file carries an ETag (a fingerprint of its
// contents, see contentETag), the browser sends it back, and if nothing
// changed we answer "304 Not Modified" with no body, instead of sending
// the whole file again.
const cacheControlSite = "no-cache"

// contentETag returns an ETag for some content: a short fingerprint
// (the start of its SHA-256 hash) that changes whenever the content does.
// It's marked weak (W/) because compress.go may send the content gzipped,
// and a weak ETag means "the same content", whatever the encoding.
func contentETag(content []byte) string {
	sum := sha256.Sum256(content)
	return `W/"` + hex.EncodeToString(sum[:8]) + `"`
}

// fillPlaceholders fills in the two placeholders the HTML pages use where
// they can't rely on JavaScript (link previews are read by apps that
// don't run it): {{APP_NAME}} and {{SITE_URL}} (see siteURL in config.go).
func fillPlaceholders(page []byte) []byte {
	page = bytes.ReplaceAll(page, []byte("{{APP_NAME}}"), []byte(appName))
	page = bytes.ReplaceAll(page, []byte("{{SITE_URL}}"), []byte(siteURL()))
	return page
}

// servePage returns a handler that sends one HTML page from the site. The
// page is read and its placeholders filled in once, when the server
// starts; every visit then gets the same bytes, with an ETag so repeat
// visits can be answered with "304 Not Modified".
func servePage(site fs.FS, name string) http.HandlerFunc {
	page, err := fs.ReadFile(site, name)
	page = fillPlaceholders(page)
	etag := contentETag(page)

	return func(w http.ResponseWriter, r *http.Request) {
		if err != nil {
			http.NotFound(w, r) // only if the file were missing from the build
			return
		}
		w.Header().Set("Cache-Control", cacheControlSite)
		w.Header().Set("ETag", etag)
		// ServeContent picks the content type from the name (".html") and
		// answers "304 Not Modified" when the browser's ETag matches.
		http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(page))
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
		page = fillPlaceholders(page)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", cacheControlSite)
		w.WriteHeader(http.StatusNotFound)
		w.Write(page)
	}
}

// cacheControlImage: the tab icon, the home-screen icon and the link
// preview picture rarely change, so browsers may keep them for a day
// instead of asking for them on every page.
const cacheControlImage = "public, max-age=86400"

// serveImage returns a handler that sends one image from the site, with
// the given content type. Used for addresses browsers and apps ask for on
// their own:
//   - /favicon.ico: the tab icon. Ours is an SVG (sharp at every size);
//     browsers accept it at this address as long as it's labelled as SVG.
//   - /apple-touch-icon.png: the icon an iPhone uses when the site is
//     added to the home screen (a PNG, which iPhones require).
//   - /og-image.png: the picture shown in link previews.
func serveImage(site fs.FS, name, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		image, err := fs.ReadFile(site, name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", cacheControlImage)
		w.Write(image)
	}
}

// serveStatic returns a handler for the CSS, JavaScript and other files
// under /static/. Go's file server would show a list of files for folder
// paths like /static/js/; we answer those with 404 instead, since there's
// no reason to advertise what's in there.
//
// Every file gets an ETag (worked out once, when the server starts), so a
// browser that already has a file gets "304 Not Modified" instead of the
// whole file again.
func serveStatic(site fs.FS) http.HandlerFunc {
	files := http.FileServerFS(site)
	etags := staticETags(site)
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", cacheControlSite)
		if etag, ok := etags[strings.TrimPrefix(r.URL.Path, "/")]; ok {
			w.Header().Set("ETag", etag)
		}
		files.ServeHTTP(w, r)
	}
}

// staticETags works out the ETag of every file under static/, keyed by
// its path ("static/css/base.css"). The files are built into the
// program, so they can't change while it runs.
func staticETags(site fs.FS) map[string]string {
	etags := make(map[string]string)
	fs.WalkDir(site, "static", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if content, err := fs.ReadFile(site, path); err == nil {
			etags[path] = contentETag(content)
		}
		return nil
	})
	return etags
}
