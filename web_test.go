package main

import (
	"net/http"
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
		{"home script", http.MethodGet, "/static/js/home.js", http.StatusOK, "text/javascript", "buildColumns"},
		{"config script", http.MethodGet, "/static/js/config.js", http.StatusOK, "text/javascript", "APP_NAME"},
		{"no folder listings", http.MethodGet, "/static/js/", http.StatusNotFound, "", ""},
		{"missing static file", http.MethodGet, "/static/nope.css", http.StatusNotFound, "", ""},
		{"unknown page", http.MethodGet, "/nope", http.StatusNotFound, "", ""},
		{"home with wrong method", http.MethodPost, "/", http.StatusMethodNotAllowed, "", ""},
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

// TestAPIStillJSON makes sure adding the website routes didn't change how
// unknown API paths are answered.
func TestAPIStillJSON(t *testing.T) {
	rec := serve(&fakeSource{}, http.MethodGet, "/api/nope")
	assertJSON(t, rec, http.StatusNotFound, `{"error":"not found"}`)
}
