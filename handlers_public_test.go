package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

// fakeSource stands in for AniList in handler tests. It returns whatever
// the test puts in it, and remembers what the handler asked for.
type fakeSource struct {
	// What to return.
	searchResult anilistSearchResult
	media        anilistMedia
	err          error

	// What the handler asked for (zero values mean "not called").
	gotSearch string
	gotPage   int
	gotID     int
}

func (f *fakeSource) searchManga(ctx context.Context, search string, page int) (anilistSearchResult, error) {
	f.gotSearch, f.gotPage = search, page
	return f.searchResult, f.err
}

func (f *fakeSource) getManga(ctx context.Context, id int) (anilistMedia, error) {
	f.gotID = id
	return f.media, f.err
}

// serve sends one fake request through the real router and returns the
// recorded response. httptest lets us do this without starting a server.
func serve(source mangaSource, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	newRouter(&app{source: source}).ServeHTTP(rec, req)
	return rec
}

// assertJSON checks a response's status, content type, and body. Bodies
// are compared as JSON values, so spacing and key order don't matter, but
// every key and value (including nulls) does.
func assertJSON(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantBody string) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Errorf("status = %d, want %d", rec.Code, wantStatus)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want JSON", got)
	}

	var got, want any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v\n%s", err, rec.Body.String())
	}
	if err := json.Unmarshal([]byte(wantBody), &want); err != nil {
		t.Fatalf("bad wantBody in test: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("body mismatch\n got: %s\nwant: %s", rec.Body.String(), wantBody)
	}
}

func TestHealthAndUnknownRoutes(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{"health check", http.MethodGet, "/api/health", http.StatusOK, `{"status":"ok"}`},
		{"health check with wrong method", http.MethodPost, "/api/health", http.StatusMethodNotAllowed, `{"error":"method not allowed"}`},
		{"search with wrong method", http.MethodDelete, "/api/search?q=x", http.StatusMethodNotAllowed, `{"error":"method not allowed"}`},
		{"unknown api path", http.MethodGet, "/api/does-not-exist", http.StatusNotFound, `{"error":"not found"}`},
		{"manga with no id", http.MethodGet, "/api/manga/", http.StatusNotFound, `{"error":"not found"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(&fakeSource{}, tt.method, tt.path)
			assertJSON(t, rec, tt.wantStatus, tt.wantBody)
		})
	}
}

func TestSearchSuccess(t *testing.T) {
	source := &fakeSource{
		searchResult: anilistSearchResult{
			HasNextPage: true,
			Media: []anilistMedia{
				{
					ID:           30013,
					Title:        anilistTitle{Romaji: ptr("ONE PIECE"), English: ptr("One Piece"), Native: ptr("ONE PIECE")},
					CoverImage:   anilistCoverImage{Large: ptr("https://example.com/medium/op.jpg"), Color: ptr("#f1935d")},
					StartDate:    anilistDate{Year: ptr(1997)},
					Format:       "MANGA",
					AverageScore: ptr(91),
				},
				{
					// A sparse one-shot: no English title, colour, or score.
					ID:         94261,
					Title:      anilistTitle{Romaji: ptr("Android wa Furimukanai"), Native: ptr("アンドロイドはふり向かない")},
					CoverImage: anilistCoverImage{Large: ptr("https://example.com/medium/a.jpg")},
					StartDate:  anilistDate{Year: ptr(2002)},
					Format:     "ONE_SHOT",
				},
			},
		},
	}

	// "%20" is a space: the query should arrive trimmed.
	rec := serve(source, http.MethodGet, "/api/search?q=%20one%20piece%20&page=2")

	assertJSON(t, rec, http.StatusOK, `{
		"query": "one piece",
		"page": 2,
		"hasNextPage": true,
		"results": [
			{"id": 30013, "title": "ONE PIECE", "coverUrl": "https://example.com/medium/op.jpg",
			 "coverColor": "#f1935d", "year": 1997, "format": "Manga", "score": 9.1},
			{"id": 94261, "title": "Android wa Furimukanai", "coverUrl": "https://example.com/medium/a.jpg",
			 "coverColor": null, "year": 2002, "format": "One Shot", "score": null}
		]
	}`)

	if source.gotSearch != "one piece" || source.gotPage != 2 {
		t.Errorf("source asked for (%q, %d), want (\"one piece\", 2)", source.gotSearch, source.gotPage)
	}
}

func TestSearchNoResults(t *testing.T) {
	rec := serve(&fakeSource{}, http.MethodGet, "/api/search?q=xqzv")

	// No results is a normal 200 with an empty list, not an error.
	assertJSON(t, rec, http.StatusOK, `{"query":"xqzv","page":1,"hasNextPage":false,"results":[]}`)
}

func TestSearchValidation(t *testing.T) {
	const (
		emptyQuery = `{"error":"query must not be empty"}`
		longQuery  = `{"error":"query must be at most 100 characters"}`
		badPage    = `{"error":"page must be a whole number from 1 to 50"}`
	)

	tests := []struct {
		name       string
		query      string // raw query string after "?"
		wantStatus int
		wantError  string // "" means the request should succeed
	}{
		{"missing q", "", http.StatusBadRequest, emptyQuery},
		{"empty q", "q=", http.StatusBadRequest, emptyQuery},
		{"only spaces", "q=%20%20%20", http.StatusBadRequest, emptyQuery},
		{"101 characters", "q=" + strings.Repeat("a", 101), http.StatusBadRequest, longQuery},
		{"100 characters", "q=" + strings.Repeat("a", 100), http.StatusOK, ""},
		// 100 Japanese characters are 300 bytes, but still only 100 characters.
		{"100 Japanese characters", "q=" + url.QueryEscape(strings.Repeat("漫", 100)), http.StatusOK, ""},
		{"page 0", "q=x&page=0", http.StatusBadRequest, badPage},
		{"page 51", "q=x&page=51", http.StatusBadRequest, badPage},
		{"page 50", "q=x&page=50", http.StatusOK, ""},
		{"page not a number", "q=x&page=abc", http.StatusBadRequest, badPage},
		{"page with decimals", "q=x&page=1.5", http.StatusBadRequest, badPage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &fakeSource{}
			rec := serve(source, http.MethodGet, "/api/search?"+tt.query)

			if tt.wantError == "" {
				if rec.Code != tt.wantStatus {
					t.Errorf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
				}
				return
			}

			assertJSON(t, rec, tt.wantStatus, tt.wantError)
			// A rejected request must never reach AniList.
			if source.gotSearch != "" {
				t.Errorf("source was called for an invalid request")
			}
		})
	}
}

func TestMangaDetailSuccess(t *testing.T) {
	source := &fakeSource{
		media: anilistMedia{
			ID:           53390,
			Title:        anilistTitle{Romaji: ptr("Shingeki no Kyojin"), English: ptr("Attack on Titan"), Native: ptr("進撃の巨人")},
			CoverImage:   anilistCoverImage{ExtraLarge: ptr("https://example.com/large/aot.jpg"), Color: ptr("#d6431a")},
			StartDate:    anilistDate{Year: ptr(2009)},
			Format:       "MANGA",
			Status:       ptr("FINISHED"),
			Chapters:     ptr(141),
			Volumes:      ptr(34),
			AverageScore: ptr(84),
			Genres:       []string{"Action", "Drama", "Fantasy", "Mystery"},
			Description: ptr("Paragraph one...\n<br><br>\n(Source: Kodansha USA)\n<br><br>\n" +
				"<i>Paragraph two...</i>"),
			SiteURL: "https://anilist.co/manga/53390",
		},
	}

	rec := serve(source, http.MethodGet, "/api/manga/53390")

	// This is the "Target detail response" from CLAUDE.md.
	assertJSON(t, rec, http.StatusOK, `{
		"id": 53390, "title": "Shingeki no Kyojin", "subtitle": "Attack on Titan",
		"format": "Manga", "status": "Finished", "year": 2009,
		"chapters": 141, "volumes": 34, "score": 8.4,
		"genres": ["Action", "Drama", "Fantasy", "Mystery"],
		"synopsis": ["Paragraph one...", "Paragraph two..."],
		"coverUrl": "https://example.com/large/aot.jpg", "coverColor": "#d6431a",
		"url": "https://anilist.co/manga/53390"
	}`)

	if source.gotID != 53390 {
		t.Errorf("source asked for id %d, want 53390", source.gotID)
	}
}

func TestMangaDetailNulls(t *testing.T) {
	// A sparse, ongoing title: almost everything AniList can leave out is missing.
	source := &fakeSource{
		media: anilistMedia{
			ID:      32688,
			Title:   anilistTitle{Romaji: ptr("Karappo na Bokutachi")},
			Format:  "MANGA",
			Status:  ptr("RELEASING"),
			SiteURL: "https://anilist.co/manga/32688",
		},
	}

	rec := serve(source, http.MethodGet, "/api/manga/32688")

	// Every key is present; missing values are null and missing lists are [].
	assertJSON(t, rec, http.StatusOK, `{
		"id": 32688, "title": "Karappo na Bokutachi", "subtitle": null,
		"format": "Manga", "status": "Releasing", "year": null,
		"chapters": null, "volumes": null, "score": null,
		"genres": [], "synopsis": [],
		"coverUrl": null, "coverColor": null,
		"url": "https://anilist.co/manga/32688"
	}`)
}

func TestMangaDetailInvalidID(t *testing.T) {
	for _, id := range []string{"abc", "0", "-5", "1.5", "99999999999"} {
		t.Run(id, func(t *testing.T) {
			source := &fakeSource{}
			rec := serve(source, http.MethodGet, "/api/manga/"+id)

			assertJSON(t, rec, http.StatusBadRequest, `{"error":"invalid id"}`)
			if source.gotID != 0 {
				t.Errorf("source was called for an invalid id")
			}
		})
	}
}

// TestUpstreamErrors checks that each kind of error from the manga source
// becomes the right status and message, for both endpoints.
func TestUpstreamErrors(t *testing.T) {
	const unavailable = `{"error":"upstream service unavailable, try again shortly"}`

	tests := []struct {
		name           string
		err            error
		wantStatus     int
		wantBody       string
		wantRetryAfter string
	}{
		{"not found", errNotFound, http.StatusNotFound, `{"error":"manga not found"}`, ""},
		{"rate limited with wait", &rateLimitedError{RetryAfterSeconds: 30}, http.StatusServiceUnavailable, unavailable, "30"},
		{"rate limited without wait", &rateLimitedError{}, http.StatusServiceUnavailable, unavailable, ""},
		{"unavailable", fmt.Errorf("%w: timeout", errUpstreamUnavailable), http.StatusServiceUnavailable, unavailable, ""},
		{"bad response", fmt.Errorf("%w: status 500", errUpstreamBad), http.StatusBadGateway, `{"error":"upstream service error"}`, ""},
		{"unexpected", errors.New("something odd"), http.StatusInternalServerError, `{"error":"internal error"}`, ""},
	}

	for _, tt := range tests {
		for _, path := range []string{"/api/manga/1", "/api/search?q=x"} {
			t.Run(tt.name+" "+path, func(t *testing.T) {
				rec := serve(&fakeSource{err: tt.err}, http.MethodGet, path)

				assertJSON(t, rec, tt.wantStatus, tt.wantBody)
				if got := rec.Header().Get("Retry-After"); got != tt.wantRetryAfter {
					t.Errorf("Retry-After = %q, want %q", got, tt.wantRetryAfter)
				}
			})
		}
	}
}
