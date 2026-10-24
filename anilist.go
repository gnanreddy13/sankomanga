package main

// This file is the only place that talks to AniList. Everything else in the
// app calls searchManga or getManga and never sees AniList's URL, its query
// language, or its raw HTTP responses. That way, if AniList changes (or we
// switch data source), only this file needs to change.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	anilistURL = "https://graphql.anilist.co"

	// anilistTimeout is the most we wait for a single AniList call.
	// AniList normally answers in well under a second, so 10s only kicks in
	// when something is really wrong. It must stay shorter than the server's
	// WriteTimeout (30s, in main.go) so we still have time to send an error.
	anilistTimeout = 10 * time.Second

	// maxAniListResponseBytes caps how much of a response we read (1 MB).
	// Our largest real response is a few KB; the cap protects our memory
	// if AniList ever sends something broken or huge.
	maxAniListResponseBytes = 1 << 20

	// searchPageSize is how many results one search page returns.
	searchPageSize = 20

	// How many calls we allow ourselves to make to AniList. AniList allows
	// 30 per minute right now (normally 90; their docs say the API is
	// temporarily "degraded"), plus an unpublished burst limit. At most
	// anilistBurst + anilistPerMinute = 25 calls can happen in any one
	// minute, which leaves a safety margin below 30.
	// When AniList is back to 90/min, these can go up to about 70 and 10.
	anilistPerMinute = 20
	anilistBurst     = 5

	// anilistMaxWait is the longest a request waits for its turn to call
	// AniList before we give up and answer 503. At 20 calls per minute a
	// new turn frees up every 3 seconds, so 3s lets one request queue up.
	anilistMaxWait = 3 * time.Second

	// anilistDefaultPause is how long we stop calling AniList after a 429
	// that doesn't say how long to wait. AniList's docs describe a
	// 1-minute timeout.
	anilistDefaultPause = time.Minute
)

// mangaFilters is added to every AniList query, so search and detail can
// never disagree about what counts as "Japanese manga, not adult".
// The argument names were confirmed against AniList's schema in build step 1.
const mangaFilters = `type: MANGA, countryOfOrigin: "JP", format_in: [MANGA, ONE_SHOT], isAdult: false`

// searchQuery asks for one page of search results, best match first.
const searchQuery = `
query ($search: String, $page: Int, $perPage: Int) {
  Page(page: $page, perPage: $perPage) {
    pageInfo { hasNextPage }
    media(search: $search, ` + mangaFilters + `, sort: [SEARCH_MATCH]) {
      id
      title { romaji english native }
      coverImage { large color }
      startDate { year }
      format
      status
      averageScore
    }
  }
}`

// detailQuery asks for one title by id. If the id is unknown, or the title
// fails our filters, AniList answers with HTTP 404.
const detailQuery = `
query ($id: Int) {
  Media(id: $id, ` + mangaFilters + `) {
    id
    title { romaji english native }
    coverImage { extraLarge color }
    startDate { year }
    format
    status
    chapters
    volumes
    averageScore
    genres
    description
    siteUrl
  }
}`

// showcaseQuery asks for the most popular titles, for the decorative covers
// on the home page. Ecchi is excluded here only (not in search), so the
// first thing visitors see is never fan-service art. The genre_not_in
// argument was confirmed against AniList's schema.
const showcaseQuery = `
query ($perPage: Int) {
  Page(page: 1, perPage: $perPage) {
    media(` + mangaFilters + `, genre_not_in: ["Ecchi"], sort: [POPULARITY_DESC]) {
      id
      title { romaji english native }
      coverImage { extraLarge color }
    }
  }
}`

// showcaseSize is how many covers the home page showcase gets: enough for
// two rows of 15 different covers each, so no cover appears twice on the
// screen at once. Still a single request, cached for hours.
const showcaseSize = 30

// These errors describe what went wrong in a way the HTTP handlers can act
// on, without knowing anything about AniList. Handlers check them with
// errors.Is, e.g. errors.Is(err, errNotFound).
var (
	// errNotFound: the title doesn't exist or is excluded by our filters.
	errNotFound = errors.New("anilist: not found")

	// errUpstreamUnavailable: AniList is busy or unreachable (rate limit,
	// timeout, connection failure). Trying again later may work.
	errUpstreamUnavailable = errors.New("anilist: unavailable")

	// errUpstreamBad: AniList answered, but the answer was broken
	// (server error, unreadable JSON, or a GraphQL error).
	errUpstreamBad = errors.New("anilist: bad response")
)

// rateLimitedError means we can't call AniList right now: either AniList
// answered 429 Too Many Requests, or our own limiter (anilistLimiter)
// stopped the call before it was sent. It carries how long to wait.
// Its Unwrap method makes errors.Is(err, errUpstreamUnavailable) true, so
// handlers that don't care about the wait time can treat it like any
// other "unavailable" error.
type rateLimitedError struct {
	RetryAfterSeconds int  // 0 if unknown
	ByUs              bool // true: our own limiter; false: AniList said 429
}

func (e *rateLimitedError) Error() string {
	if e.ByUs {
		return fmt.Sprintf("anilist: holding back to stay under the rate limit (retry after %ds)", e.RetryAfterSeconds)
	}
	return fmt.Sprintf("anilist: rate limited by AniList (retry after %ds)", e.RetryAfterSeconds)
}

func (e *rateLimitedError) Unwrap() error {
	return errUpstreamUnavailable
}

// The types below mirror AniList's JSON exactly. A pointer (*string, *int)
// means the field can be null: nil means "AniList sent null". Turning these
// into our own clean API shape happens elsewhere (build steps 5 and 6).

type anilistTitle struct {
	Romaji  *string `json:"romaji"`
	English *string `json:"english"`
	Native  *string `json:"native"`
}

type anilistCoverImage struct {
	Large      *string `json:"large"`
	ExtraLarge *string `json:"extraLarge"`
	Color      *string `json:"color"`
}

type anilistDate struct {
	Year *int `json:"year"`
}

// anilistMedia is one manga. Search results only fill in some fields;
// the rest stay nil/empty.
type anilistMedia struct {
	ID           int               `json:"id"`
	Title        anilistTitle      `json:"title"`
	CoverImage   anilistCoverImage `json:"coverImage"`
	StartDate    anilistDate       `json:"startDate"`
	Format       string            `json:"format"`
	Status       *string           `json:"status"`
	Chapters     *int              `json:"chapters"`
	Volumes      *int              `json:"volumes"`
	AverageScore *int              `json:"averageScore"`
	Genres       []string          `json:"genres"`
	Description  *string           `json:"description"`
	SiteURL      string            `json:"siteUrl"`
}

// anilistSearchResult is one page of search results.
type anilistSearchResult struct {
	HasNextPage bool
	Media       []anilistMedia
}

// anilistClient sends queries to AniList.
type anilistClient struct {
	httpClient *http.Client
	// url is a field (not just the constant) so tests can point the
	// client at a fake AniList server running on the test machine.
	url string
	// limiter makes sure we stay under AniList's rate limit.
	limiter *anilistLimiter
}

// newAniListClient returns a client for the real AniList API.
func newAniListClient() *anilistClient {
	return &anilistClient{
		httpClient: &http.Client{Timeout: anilistTimeout},
		url:        anilistURL,
		limiter:    newAniListLimiter(),
	}
}

// anilistLimiter decides when we may call AniList. It's shared by every
// request, so a mutex guards it (like the cache in cache.go).
//
// It does two things:
//  1. Spreads our calls out with a token bucket (ratelimit.go), so we
//     never send more than AniList allows. A request whose turn is near
//     waits for it; one whose turn is far off gets a 503 instead.
//  2. After AniList answers 429, pauses ALL calls until AniList's
//     Retry-After time has passed. AniList's docs say calls made during
//     that timeout are refused too, so sending them would be wasted.
type anilistLimiter struct {
	mu          sync.Mutex
	bucket      *tokenBucket
	pausedUntil time.Time

	// Replaceable in tests, so they control time instead of waiting.
	now   func() time.Time
	sleep func(ctx context.Context, d time.Duration) error
}

func newAniListLimiter() *anilistLimiter {
	return &anilistLimiter{
		bucket: newTokenBucket(anilistBurst, anilistPerMinute, time.Now()),
		now:    time.Now,
		sleep:  sleepContext,
	}
}

// wait returns once it's our turn to call AniList. It returns a
// *rateLimitedError instead if the turn is too far away (or AniList told
// us to pause), and an "unavailable" error if the visitor gives up
// (disconnects) while waiting.
func (l *anilistLimiter) wait(ctx context.Context) error {
	l.mu.Lock()
	now := l.now()

	if now.Before(l.pausedUntil) {
		remaining := l.pausedUntil.Sub(now)
		l.mu.Unlock()
		return &rateLimitedError{RetryAfterSeconds: retryAfterSeconds(remaining), ByUs: true}
	}

	wait, ok := l.bucket.reserve(now, anilistMaxWait)
	l.mu.Unlock() // unlock before sleeping, so other requests aren't blocked

	if !ok {
		return &rateLimitedError{RetryAfterSeconds: retryAfterSeconds(wait), ByUs: true}
	}
	if wait > 0 {
		if err := l.sleep(ctx, wait); err != nil {
			return fmt.Errorf("%w: gave up waiting for our turn: %w", errUpstreamUnavailable, err)
		}
	}
	return nil
}

// pause stops all AniList calls for the given number of seconds (or
// anilistDefaultPause if AniList didn't say). It never shortens an
// existing pause.
func (l *anilistLimiter) pause(seconds int) {
	duration := anilistDefaultPause
	if seconds > 0 {
		duration = time.Duration(seconds) * time.Second
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if until := l.now().Add(duration); until.After(l.pausedUntil) {
		l.pausedUntil = until
	}
}

// sleepContext waits for d, but stops early (returning an error) if ctx is
// cancelled, e.g. because the visitor closed the page.
func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// searchManga returns one page of manga matching the search text.
func (c *anilistClient) searchManga(ctx context.Context, search string, page int) (anilistSearchResult, error) {
	variables := map[string]any{
		"search":  search,
		"page":    page,
		"perPage": searchPageSize,
	}

	// This struct matches the shape of the "data" part of the response.
	var data struct {
		Page struct {
			PageInfo struct {
				HasNextPage bool `json:"hasNextPage"`
			} `json:"pageInfo"`
			Media []anilistMedia `json:"media"`
		} `json:"Page"`
	}

	if err := c.query(ctx, searchQuery, variables, &data); err != nil {
		return anilistSearchResult{}, err
	}

	return anilistSearchResult{
		HasNextPage: data.Page.PageInfo.HasNextPage,
		Media:       data.Page.Media,
	}, nil
}

// getManga returns one manga by its AniList id.
func (c *anilistClient) getManga(ctx context.Context, id int) (anilistMedia, error) {
	variables := map[string]any{"id": id}

	var data struct {
		Media *anilistMedia `json:"Media"`
	}

	if err := c.query(ctx, detailQuery, variables, &data); err != nil {
		return anilistMedia{}, err
	}

	// AniList normally answers 404 for a missing title (handled in query),
	// but if it ever sends 200 with "Media": null, treat it the same way.
	if data.Media == nil {
		return anilistMedia{}, errNotFound
	}
	return *data.Media, nil
}

// popularManga returns the most popular titles (minus ecchi) for the
// home page showcase.
func (c *anilistClient) popularManga(ctx context.Context) ([]anilistMedia, error) {
	variables := map[string]any{"perPage": showcaseSize}

	var data struct {
		Page struct {
			Media []anilistMedia `json:"media"`
		} `json:"Page"`
	}

	if err := c.query(ctx, showcaseQuery, variables, &data); err != nil {
		return nil, err
	}
	return data.Page.Media, nil
}

// graphqlRequest is the JSON body every GraphQL API expects.
type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

// graphqlResponse is the outer shape of every GraphQL answer: the useful
// part is in "data", and anything that went wrong is listed in "errors".
type graphqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// query sends one GraphQL query to AniList and decodes the "data" part of
// the answer into data. Every failure comes back as one of the errors
// defined above, so callers never have to inspect HTTP details.
func (c *anilistClient) query(ctx context.Context, query string, variables map[string]any, data any) error {
	// Wait for our turn, so we stay under AniList's rate limit.
	if err := c.limiter.wait(ctx); err != nil {
		return err
	}

	reqBody, err := json.Marshal(graphqlRequest{Query: query, Variables: variables})
	if err != nil {
		return fmt.Errorf("encoding anilist request: %w", err)
	}

	// Passing ctx means the call is cancelled automatically if our visitor
	// disconnects, so we don't keep waiting on AniList for nobody.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("building anilist request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", appName)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Timeouts, DNS failures, refused connections, cancelled requests.
		return fmt.Errorf("%w: %w", errUpstreamUnavailable, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxAniListResponseBytes))
	if err != nil {
		// The connection broke (or timed out) while the answer was arriving.
		return fmt.Errorf("%w: reading response: %w", errUpstreamUnavailable, err)
	}

	if err := checkAniListStatus(resp); err != nil {
		// If AniList says we've sent too many requests, stop all calls
		// until it says we may continue.
		var rateErr *rateLimitedError
		if errors.As(err, &rateErr) {
			c.limiter.pause(rateErr.RetryAfterSeconds)
		}
		return err
	}

	var envelope graphqlResponse
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("%w: decoding response: %w", errUpstreamBad, err)
	}
	if len(envelope.Errors) > 0 {
		return fmt.Errorf("%w: graphql error: %s", errUpstreamBad, envelope.Errors[0].Message)
	}
	if err := json.Unmarshal(envelope.Data, data); err != nil {
		return fmt.Errorf("%w: decoding data: %w", errUpstreamBad, err)
	}
	return nil
}

// checkAniListStatus turns AniList's HTTP status code into one of our
// errors, or nil if the status is 200 OK.
func checkAniListStatus(resp *http.Response) error {
	switch {
	case resp.StatusCode == http.StatusOK:
		return nil
	case resp.StatusCode == http.StatusNotFound:
		return errNotFound
	case resp.StatusCode == http.StatusTooManyRequests:
		return &rateLimitedError{RetryAfterSeconds: parseRetryAfter(resp.Header.Get("Retry-After"))}
	default:
		return fmt.Errorf("%w: status %d", errUpstreamBad, resp.StatusCode)
	}
}

// parseRetryAfter reads AniList's Retry-After header, which is a number of
// seconds (e.g. "30"). It returns 0 if the header is missing or not a
// positive number.
func parseRetryAfter(header string) int {
	seconds, err := strconv.Atoi(header)
	if err != nil || seconds < 0 {
		return 0
	}
	return seconds
}
