package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)

// This file holds the public handlers: every visitor gets the same answer,
// so their responses are safe to cache and share. Any future user-specific
// handlers (accounts, settings) belong in a separate file and must never
// use the shared cache.

const (
	// maxQueryLength is the longest search query we accept, in characters.
	maxQueryLength = 100

	// maxPage is the last search page we serve. It stops anyone from
	// crawling AniList's catalog through us; real users never page this far.
	maxPage = 50
)

// mangaSource is anything that can search for and look up manga.
//
// It's an interface (a list of methods) rather than the AniList client
// itself, so that:
//   - tests can use a small fake that returns canned data, and
//   - the cache (step 7) can sit between the handlers and AniList
//     without either side changing.
//
// The real anilistClient already has both methods, so it fits as is.
type mangaSource interface {
	searchManga(ctx context.Context, search string, page int) (anilistSearchResult, error)
	getManga(ctx context.Context, id int) (anilistMedia, error)
}

// app holds what the handlers need. The handlers are methods on it, so
// they can reach the manga source without any global variables.
type app struct {
	source mangaSource
}

// healthResponse is the body of GET /api/health.
type healthResponse struct {
	Status string `json:"status"`
}

// handleHealth reports that the server is running. It deliberately does not
// call AniList: a health check should say whether *our* server is up, and
// it shouldn't spend AniList's rate limit every time a monitor pings it.
func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

// handleSearch answers GET /api/search?q=...&page=...
func (a *app) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeError(w, http.StatusBadRequest, "query must not be empty")
		return
	}
	// Count characters, not bytes: a Japanese character takes 3 bytes,
	// and we don't want a 40-character Japanese query rejected as "too long".
	if utf8.RuneCountInString(query) > maxQueryLength {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("query must be at most %d characters", maxQueryLength))
		return
	}

	page, ok := parsePage(r.URL.Query().Get("page"))
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("page must be a whole number from 1 to %d", maxPage))
		return
	}

	result, err := a.source.searchManga(r.Context(), query, page)
	if err != nil {
		writeUpstreamError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toSearchResponse(query, page, result))
}

// handleMangaDetail answers GET /api/manga/{id}
func (a *app) handleMangaDetail(w http.ResponseWriter, r *http.Request) {
	// PathValue reads the {id} part of the route pattern in routes.go.
	id, ok := parseID(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	media, err := a.source.getManga(r.Context(), id)
	if err != nil {
		writeUpstreamError(w, err)
		return
	}

	detail := toDetail(media)

	// If AniList ever adds a status we don't know, we show null. Log it so
	// we notice and can add the new status to statusText in display.go.
	if media.Status != nil && detail.Status == nil {
		slog.Warn("unknown AniList status", "status", *media.Status, "id", media.ID)
	}

	writeJSON(w, http.StatusOK, detail)
}

// handleAPINotFound answers any /api/... path we don't recognise, so that
// even mistakes come back in our standard JSON error shape.
func handleAPINotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not found")
}

// parsePage reads the optional page parameter. A missing page means page 1.
// ok is false if the value isn't a whole number from 1 to maxPage.
func parsePage(raw string) (page int, ok bool) {
	if raw == "" {
		return 1, true
	}
	page, err := strconv.Atoi(raw)
	if err != nil || page < 1 || page > maxPage {
		return 0, false
	}
	return page, true
}

// parseID reads a manga id from the URL. ok is false unless it's a positive
// whole number that fits in 32 bits, which is the size of AniList's ids.
// Without the size limit, a huge id would reach AniList and come back as a
// confusing "upstream error" instead of a simple "invalid id".
func parseID(raw string) (id int, ok bool) {
	parsed, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || parsed < 1 {
		return 0, false
	}
	return int(parsed), true
}

// writeUpstreamError turns an error from the manga source into the right
// HTTP response. For server-side problems (5xx), the details are logged
// for us, and the visitor only sees a short, generic message.
func writeUpstreamError(w http.ResponseWriter, err error) {
	const unavailable = "upstream service unavailable, try again shortly"

	// errors.As fills in rateErr if err is (or wraps) a *rateLimitedError.
	var rateErr *rateLimitedError

	switch {
	case errors.Is(err, errNotFound):
		writeError(w, http.StatusNotFound, "manga not found")

	case errors.As(err, &rateErr):
		slog.Warn("AniList rate limited us", "err", err)
		// Pass AniList's requested wait on to the visitor, if it gave one.
		if rateErr.RetryAfterSeconds > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(rateErr.RetryAfterSeconds))
		}
		writeError(w, http.StatusServiceUnavailable, unavailable)

	case errors.Is(err, errUpstreamUnavailable):
		slog.Error("AniList unavailable", "err", err)
		writeError(w, http.StatusServiceUnavailable, unavailable)

	case errors.Is(err, errUpstreamBad):
		slog.Error("AniList sent a bad response", "err", err)
		writeError(w, http.StatusBadGateway, "upstream service error")

	default:
		slog.Error("unexpected error", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

// onlyGET wraps a handler so that it only accepts GET (and HEAD, which is
// GET without the body and is what some uptime monitors use). Anything else
// gets a 405 in our JSON error shape. Go's router can filter methods itself,
// but its 405 response is plain text, which would break our error contract.
func onlyGET(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			// The Allow header tells the client which methods would work.
			w.Header().Set("Allow", "GET, HEAD")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		next(w, r)
	}
}
