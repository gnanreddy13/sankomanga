package main

// This file keeps recent AniList answers in memory for a short time, so
// repeated searches and title pages don't call AniList again. That makes
// them much faster and saves AniList's rate limit (30 requests/minute).
//
// AniList's terms forbid using their API as a data store or mass-collecting
// it, so this cache is deliberately small and short-lived: every entry
// expires, and there's a hard cap on how many entries it holds.
//
// It's a shared cache: everyone gets the same cached answers. Only public
// data may ever go in it, never anything user-specific.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	// searchCacheTTL is how long search results are kept. People repeat
	// popular searches, and results barely change within 10 minutes.
	searchCacheTTL = 10 * time.Minute

	// detailCacheTTL is how long a title page is kept. Chapter counts and
	// scores change slowly, and people often go back and forth between
	// search results and title pages.
	detailCacheTTL = time.Hour

	// notFoundCacheTTL is how long we remember that an id doesn't exist
	// (or is filtered out). This stops anyone from spending our AniList
	// budget by requesting the same missing id over and over.
	notFoundCacheTTL = 10 * time.Minute

	// maxCacheEntries caps the cache's size. Each entry is a few KB at
	// most, so 1,000 entries is roughly 5 MB: tiny next to AniList's
	// catalog of about 100,000 manga, and never a growing copy of it.
	maxCacheEntries = 1000
)

// cacheEntry is one remembered answer.
type cacheEntry struct {
	value   any   // an anilistSearchResult or an anilistMedia
	err     error // set instead of value for a remembered "not found"
	expires time.Time
}

// ttlCache is a size-capped map whose entries expire ("TTL" = time to live).
//
// Many requests are handled at the same time, each on its own goroutine,
// and Go maps must not be changed by two goroutines at once. The mutex
// (mu) makes sure only one goroutine uses the map at a time.
type ttlCache struct {
	mu         sync.Mutex
	entries    map[string]cacheEntry
	maxEntries int

	// now returns the current time. It's a field so tests can replace it
	// with a fake clock and test expiry without actually waiting.
	now func() time.Time
}

func newTTLCache(maxEntries int) *ttlCache {
	return &ttlCache{
		entries:    make(map[string]cacheEntry),
		maxEntries: maxEntries,
		now:        time.Now,
	}
}

// get returns the entry for key, if there is one and it hasn't expired.
func (c *ttlCache) get(key string) (cacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		return cacheEntry{}, false
	}
	if !c.now().Before(entry.expires) {
		delete(c.entries, key) // expired: forget it now
		return cacheEntry{}, false
	}
	return entry, true
}

// set stores an answer (value, or err for "not found") for ttl.
func (c *ttlCache) set(key string, value any, err error, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Only a brand-new key can make the cache grow, so only then do we
	// need to check the size cap.
	if _, exists := c.entries[key]; !exists && len(c.entries) >= c.maxEntries {
		c.makeRoom()
	}
	c.entries[key] = cacheEntry{value: value, err: err, expires: c.now().Add(ttl)}
}

// makeRoom frees at least one slot. It first drops every expired entry;
// if none had expired, it drops the entry that would expire soonest.
// It looks through every entry, which is fine for 1,000 entries (well
// under a millisecond). The caller must already hold c.mu.
func (c *ttlCache) makeRoom() {
	now := c.now()
	soonestKey := ""
	var soonest time.Time

	for key, entry := range c.entries {
		if !now.Before(entry.expires) {
			delete(c.entries, key)
			continue
		}
		if soonestKey == "" || entry.expires.Before(soonest) {
			soonestKey, soonest = key, entry.expires
		}
	}

	if len(c.entries) >= c.maxEntries {
		delete(c.entries, soonestKey)
	}
}

// cachingSource is a mangaSource that answers from the cache when it can,
// and otherwise asks the next source (the AniList client) and remembers
// the answer. Because it has the same two methods as the AniList client,
// the handlers can't tell the difference.
type cachingSource struct {
	next  mangaSource
	cache *ttlCache
}

func newCachingSource(next mangaSource) *cachingSource {
	return &cachingSource{next: next, cache: newTTLCache(maxCacheEntries)}
}

// Cached values are shared between requests, so code that reads them
// (toCard, toDetail, ...) must never change them. It only reads today.

func (c *cachingSource) searchManga(ctx context.Context, search string, page int) (anilistSearchResult, error) {
	key := searchCacheKey(search, page)

	if entry, ok := c.cache.get(key); ok {
		markCacheResult(ctx, cacheHit)
		// .(anilistSearchResult) says "this value is a search result".
		// That's guaranteed: only search results are stored under "search:" keys.
		return entry.value.(anilistSearchResult), nil
	}
	markCacheResult(ctx, cacheMiss)

	result, err := c.next.searchManga(ctx, search, page)
	if err != nil {
		return anilistSearchResult{}, err // errors are never cached: next time, try again
	}
	c.cache.set(key, result, nil, searchCacheTTL)
	return result, nil
}

func (c *cachingSource) getManga(ctx context.Context, id int) (anilistMedia, error) {
	key := fmt.Sprintf("manga:%d", id)

	if entry, ok := c.cache.get(key); ok {
		markCacheResult(ctx, cacheHit)
		if entry.err != nil {
			return anilistMedia{}, entry.err // a remembered "not found"
		}
		return entry.value.(anilistMedia), nil
	}
	markCacheResult(ctx, cacheMiss)

	media, err := c.next.getManga(ctx, id)
	switch {
	case errors.Is(err, errNotFound):
		// Remember "not found", but only briefly.
		c.cache.set(key, nil, errNotFound, notFoundCacheTTL)
		return anilistMedia{}, err
	case err != nil:
		return anilistMedia{}, err // AniList trouble is never cached
	}
	c.cache.set(key, media, nil, detailCacheTTL)
	return media, nil
}

// searchCacheKey builds the cache key for a search. Upper/lower case and
// extra spaces are ignored, so "One Piece" and "one  piece" share an entry
// (AniList's search ignores case too). The page comes first, so a query
// can never be mistaken for part of the page number.
func searchCacheKey(search string, page int) string {
	normalized := strings.ToLower(strings.Join(strings.Fields(search), " "))
	return fmt.Sprintf("search:%d:%s", page, normalized)
}
