package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// These tests never contact the real AniList. Each one starts a fake
// AniList server on the test machine (httptest.NewServer) that sends back
// a canned answer, and points our client at it. That keeps tests fast,
// repeatable, and free of AniList's rate limit.

// newFakeAniList starts a fake AniList server that runs handler for every
// request, and returns a client pointed at it. The server is shut down
// automatically when the test finishes.
func newFakeAniList(t *testing.T, handler http.HandlerFunc) *anilistClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := newAniListClient()
	client.url = server.URL
	return client
}

// respondWith returns a fake-server handler that always sends the given
// status code and body.
func respondWith(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		io.WriteString(w, body)
	}
}

// sentRequest is what our client sent to the fake server.
type sentRequest struct {
	Method string
	Body   graphqlRequest
}

// recordAndRespond returns a fake-server handler that records the request
// our client sent, passes it to the test through the sent channel, and
// then answers with body.
//
// Why a channel: the fake server runs on a different goroutine from the
// test, and a channel is the safe way to hand data between goroutines.
func recordAndRespond(t *testing.T, sent chan<- sentRequest, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// t.Errorf (not t.Fatalf) because we're not on the test's goroutine.
			t.Errorf("fake server could not decode request: %v", err)
		}
		sent <- sentRequest{Method: r.Method, Body: req}
		respondWith(http.StatusOK, body)(w, r)
	}
}

func TestSearchMangaSuccess(t *testing.T) {
	// Trimmed from a real AniList response (build step 1), including nulls.
	const body = `{"data":{"Page":{
		"pageInfo":{"hasNextPage":true},
		"media":[
			{"id":30013,"title":{"romaji":"ONE PIECE","english":"One Piece","native":"ONE PIECE"},
			 "coverImage":{"large":"https://example.com/medium/op.jpg","color":"#f1935d"},
			 "startDate":{"year":1997},"format":"MANGA","averageScore":91},
			{"id":94261,"title":{"romaji":"Android wa Furimukanai","english":null,"native":"アンドロイドはふり向かない"},
			 "coverImage":{"large":"https://example.com/medium/a.jpg","color":null},
			 "startDate":{"year":2002},"format":"ONE_SHOT","averageScore":null}
		]}}}`

	requests := make(chan sentRequest, 1)
	client := newFakeAniList(t, recordAndRespond(t, requests, body))

	result, err := client.searchManga(context.Background(), "one piece", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check what we sent to AniList.
	sent := <-requests
	if sent.Method != http.MethodPost {
		t.Errorf("method = %s, want POST", sent.Method)
	}
	if !strings.Contains(sent.Body.Query, mangaFilters) {
		t.Errorf("query is missing the manga filters:\n%s", sent.Body.Query)
	}
	if !strings.Contains(sent.Body.Query, "SEARCH_MATCH") {
		t.Errorf("query is missing SEARCH_MATCH sort")
	}
	// JSON numbers decode as float64, hence 2.0 and 20.0 here.
	if sent.Body.Variables["search"] != "one piece" || sent.Body.Variables["page"] != 2.0 || sent.Body.Variables["perPage"] != 20.0 {
		t.Errorf("variables = %v", sent.Body.Variables)
	}

	// Check what we got back.
	if !result.HasNextPage {
		t.Errorf("HasNextPage = false, want true")
	}
	if len(result.Media) != 2 {
		t.Fatalf("got %d results, want 2", len(result.Media))
	}
	first := result.Media[0]
	if first.ID != 30013 || *first.Title.Romaji != "ONE PIECE" || *first.AverageScore != 91 || *first.StartDate.Year != 1997 {
		t.Errorf("first result decoded wrongly: %+v", first)
	}
	second := result.Media[1]
	if second.Title.English != nil || second.CoverImage.Color != nil || second.AverageScore != nil {
		t.Errorf("null fields should decode as nil: %+v", second)
	}
}

func TestSearchMangaNoResults(t *testing.T) {
	client := newFakeAniList(t, respondWith(http.StatusOK,
		`{"data":{"Page":{"pageInfo":{"hasNextPage":false},"media":[]}}}`))

	result, err := client.searchManga(context.Background(), "xqzv", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Media) != 0 || result.HasNextPage {
		t.Errorf("want empty result, got %+v", result)
	}
}

func TestGetMangaSuccess(t *testing.T) {
	// Trimmed from AniList's real answer for Attack on Titan.
	const body = `{"data":{"Media":{
		"id":53390,
		"title":{"romaji":"Shingeki no Kyojin","english":"Attack on Titan","native":"進撃の巨人"},
		"coverImage":{"extraLarge":"https://example.com/large/aot.jpg","color":"#d6431a"},
		"startDate":{"year":2009},"format":"MANGA","status":"FINISHED",
		"chapters":141,"volumes":34,"averageScore":84,
		"genres":["Action","Drama","Fantasy","Mystery"],
		"description":"In this post-apocalyptic sci-fi story...<br><br>(Source: Kodansha USA)",
		"siteUrl":"https://anilist.co/manga/53390"}}}`

	requests := make(chan sentRequest, 1)
	client := newFakeAniList(t, recordAndRespond(t, requests, body))

	media, err := client.getManga(context.Background(), 53390)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sent := <-requests
	if !strings.Contains(sent.Body.Query, mangaFilters) {
		t.Errorf("query is missing the manga filters:\n%s", sent.Body.Query)
	}
	if sent.Body.Variables["id"] != 53390.0 {
		t.Errorf("id variable = %v, want 53390", sent.Body.Variables["id"])
	}

	if media.ID != 53390 || *media.Status != "FINISHED" || *media.Chapters != 141 ||
		*media.CoverImage.ExtraLarge != "https://example.com/large/aot.jpg" ||
		len(media.Genres) != 4 || media.SiteURL != "https://anilist.co/manga/53390" {
		t.Errorf("decoded wrongly: %+v", media)
	}
}

// TestAniListErrors checks that every kind of AniList failure turns into
// the right one of our errors.
func TestAniListErrors(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantErr error
	}{
		{
			name: "unknown or filtered title (real AniList 404 body)",
			handler: respondWith(http.StatusNotFound,
				`{"errors":[{"message":"Not Found.","status":404}],"data":{"Media":null}}`),
			wantErr: errNotFound,
		},
		{
			name:    "200 with null media",
			handler: respondWith(http.StatusOK, `{"data":{"Media":null}}`),
			wantErr: errNotFound,
		},
		{
			name: "rate limited",
			handler: respondWith(http.StatusTooManyRequests,
				`{"data":null,"errors":[{"message":"Too Many Requests.","status":429}]}`),
			wantErr: errUpstreamUnavailable,
		},
		{
			name:    "server error",
			handler: respondWith(http.StatusInternalServerError, `oops`),
			wantErr: errUpstreamBad,
		},
		{
			name:    "graphql error inside a 200",
			handler: respondWith(http.StatusOK, `{"data":null,"errors":[{"message":"Syntax Error"}]}`),
			wantErr: errUpstreamBad,
		},
		{
			name:    "unreadable JSON",
			handler: respondWith(http.StatusOK, `<html>not json</html>`),
			wantErr: errUpstreamBad,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newFakeAniList(t, tt.handler)

			_, err := client.getManga(context.Background(), 1)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestRateLimitedCarriesRetryAfter(t *testing.T) {
	client := newFakeAniList(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	})

	_, err := client.getManga(context.Background(), 1)

	// errors.As finds a *rateLimitedError inside err, if there is one,
	// so we can read the wait time AniList asked for.
	var rateErr *rateLimitedError
	if !errors.As(err, &rateErr) {
		t.Fatalf("error = %v, want a rateLimitedError", err)
	}
	if rateErr.RetryAfterSeconds != 30 {
		t.Errorf("RetryAfterSeconds = %d, want 30", rateErr.RetryAfterSeconds)
	}
}

func TestTimeout(t *testing.T) {
	// The fake server never answers; it just waits until the client gives up.
	client := newFakeAniList(t, func(w http.ResponseWriter, r *http.Request) {
		// Read the request body first: Go's server only notices that the
		// client hung up once the body has been read. Without this line the
		// handler would wait forever and the test would hang.
		io.Copy(io.Discard, r.Body)
		<-r.Context().Done()
	})
	// Shorten the timeout so the test takes milliseconds, not 10 seconds.
	client.httpClient.Timeout = 50 * time.Millisecond

	_, err := client.getManga(context.Background(), 1)

	if !errors.Is(err, errUpstreamUnavailable) {
		t.Errorf("error = %v, want errUpstreamUnavailable", err)
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := map[string]int{
		"30":  30,
		"0":   0,
		"":    0, // header missing
		"abc": 0, // not a number
		"-5":  0, // negative makes no sense
	}
	for header, want := range tests {
		if got := parseRetryAfter(header); got != want {
			t.Errorf("parseRetryAfter(%q) = %d, want %d", header, got, want)
		}
	}
}
