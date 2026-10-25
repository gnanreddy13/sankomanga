package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebRoutes(t *testing.T) {
	tests := []struct {
		name            string
		method          string
		path            string
		wantStatus      int
		wantContentType string // checked with "starts with"
		wantBodyHas     string
	}{
		{"home page", http.MethodGet, "/", http.StatusOK, "text/html", `action="/search"`},
		{"results page", http.MethodGet, "/search?q=berserk", http.StatusOK, "text/html", "search.js"},
		{"results script", http.MethodGet, "/static/js/search.js", http.StatusOK, "text/javascript", "renderResults"},
		{"title page", http.MethodGet, "/manga/53390", http.StatusOK, "text/html", "manga.js"},
		// Unusual ids still get the page; manga.js then shows "not found".
		{"title page with odd id", http.MethodGet, "/manga/abc", http.StatusOK, "text/html", "manga.js"},
		{"title script", http.MethodGet, "/static/js/manga.js", http.StatusOK, "text/javascript", "renderTitle"},
		{"stylesheet", http.MethodGet, "/static/css/style.css", http.StatusOK, "text/css", "--bg"},
		{"home script", http.MethodGet, "/static/js/home.js", http.StatusOK, "text/javascript", "createVolume"},
		{"config script", http.MethodGet, "/static/js/config.js", http.StatusOK, "text/javascript", "APP_NAME"},
		{"search overlay script", http.MethodGet, "/static/js/search-overlay.js", http.StatusOK, "text/javascript", "setUpSearchOverlay"},
		{"no folder listings", http.MethodGet, "/static/js/", http.StatusNotFound, "", ""},
		{"missing static file", http.MethodGet, "/static/nope.css", http.StatusNotFound, "", ""},
		// Unknown addresses get the styled "Page not found" page, with a 404.
		{"unknown page", http.MethodGet, "/nope", http.StatusNotFound, "text/html", "Page not found"},
		{"unknown nested page", http.MethodGet, "/a/b/c", http.StatusNotFound, "text/html", "Page not found"},
		// Not a page of the site either (browsers never POST here).
		{"home with wrong method", http.MethodPost, "/", http.StatusNotFound, "text/html", "Page not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(&fakeSource{}, tt.method, tt.path)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, tt.wantContentType) {
				t.Errorf("Content-Type = %q, want it to start with %q", got, tt.wantContentType)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBodyHas) {
				t.Errorf("body doesn't contain %q", tt.wantBodyHas)
			}
			if tt.wantStatus == http.StatusOK && rec.Header().Get("Cache-Control") != "no-cache" {
				t.Errorf("Cache-Control = %q, want no-cache", rec.Header().Get("Cache-Control"))
			}
		})
	}
}

func TestFavicon(t *testing.T) {
	rec := serve(&fakeSource{}, http.MethodGet, "/favicon.ico")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "image/svg+xml" {
		t.Errorf("Content-Type = %q, want image/svg+xml", got)
	}
	if !strings.HasPrefix(rec.Body.String(), "<svg") {
		t.Errorf("body isn't an SVG image")
	}
}

// TestAPIStillJSON makes sure adding the website routes didn't change how
// unknown API paths are answered.
func TestAPIStillJSON(t *testing.T) {
	rec := serve(&fakeSource{}, http.MethodGet, "/api/nope")
	assertJSON(t, rec, http.StatusNotFound, `{"error":"not found"}`)
}

// TestPagePlaceholdersFilled checks that the pages' placeholders are filled
// in by the server, so apps that don't run JavaScript (link previews) see
// the real name and address.
func TestPagePlaceholdersFilled(t *testing.T) {
	t.Setenv("SITE_URL", "https://example.test/")
	t.Setenv("RAILWAY_PUBLIC_DOMAIN", "")

	for _, path := range []string{"/", "/search?q=one", "/manga/1", "/nope"} {
		body := serve(&fakeSource{}, http.MethodGet, path).Body.String()
		if strings.Contains(body, "{{") {
			t.Errorf("%s still has a placeholder", path)
		}
	}

	home := serve(&fakeSource{}, http.MethodGet, "/").Body.String()
	for _, want := range []string{
		"<title>" + appName + "</title>",
		`<meta property="og:title" content="` + appName + `">`,
		// The trailing "/" of SITE_URL is dropped, so there's no "//".
		`<meta property="og:image" content="https://example.test/og-image.png">`,
		`<link rel="apple-touch-icon" href="/apple-touch-icon.png">`,
	} {
		if !strings.Contains(home, want) {
			t.Errorf("home page is missing %s", want)
		}
	}
}

// TestSiteURL checks where the site's address comes from.
func TestSiteURL(t *testing.T) {
	tests := []struct {
		name, siteURL, railwayDomain, want string
	}{
		{"SITE_URL wins", "https://sankomanga.example/", "app.up.railway.app", "https://sankomanga.example"},
		{"Railway's address", "", "app.up.railway.app", "https://app.up.railway.app"},
		{"running locally", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SITE_URL", tt.siteURL)
			t.Setenv("RAILWAY_PUBLIC_DOMAIN", tt.railwayDomain)
			if got := siteURL(); got != tt.want {
				t.Errorf("siteURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestImages checks the images browsers and apps ask for on their own.
func TestImages(t *testing.T) {
	tests := []struct {
		path, contentType string
	}{
		{"/apple-touch-icon.png", "image/png"},
		{"/og-image.png", "image/png"},
	}
	for _, tt := range tests {
		rec := serve(&fakeSource{}, http.MethodGet, tt.path)
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", tt.path, rec.Code)
			continue
		}
		if got := rec.Header().Get("Content-Type"); got != tt.contentType {
			t.Errorf("%s: Content-Type = %q, want %q", tt.path, got, tt.contentType)
		}
		// Every PNG file starts with these eight bytes.
		if !strings.HasPrefix(rec.Body.String(), "\x89PNG\r\n\x1a\n") {
			t.Errorf("%s: body isn't a PNG image", tt.path)
		}
	}
}

// TestRepeatVisitsNotModified checks that pages and files carry an ETag,
// and that a browser sending it back gets "304 Not Modified" (no body)
// instead of the whole thing again.
func TestRepeatVisitsNotModified(t *testing.T) {
	router := newRouter(&app{source: &fakeSource{}})

	for _, path := range []string{"/", "/search", "/static/css/style.css", "/static/js/home.js"} {
		first := httptest.NewRecorder()
		router.ServeHTTP(first, httptest.NewRequest(http.MethodGet, path, nil))
		etag := first.Header().Get("ETag")
		if first.Code != http.StatusOK || etag == "" {
			t.Errorf("%s: status %d, ETag %q; want 200 with an ETag", path, first.Code, etag)
			continue
		}

		repeat := httptest.NewRequest(http.MethodGet, path, nil)
		repeat.Header.Set("If-None-Match", etag)
		second := httptest.NewRecorder()
		router.ServeHTTP(second, repeat)
		if second.Code != http.StatusNotModified || second.Body.Len() != 0 {
			t.Errorf("%s: repeat visit got %d with %d bytes; want 304 with none", path, second.Code, second.Body.Len())
		}
	}
}
