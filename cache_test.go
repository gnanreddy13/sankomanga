package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// fakeClock is a clock that only moves when a test tells it to, so expiry
// can be tested instantly instead of by waiting 10 minutes.
type fakeClock struct {
	current time.Time
}

func (c *fakeClock) now() time.Time          { return c.current }
func (c *fakeClock) advance(d time.Duration) { c.current = c.current.Add(d) }

func newFakeClock() *fakeClock {
	return &fakeClock{current: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
}

// newTestCache returns a cache with a fake clock.
func newTestCache(maxEntries int) (*ttlCache, *fakeClock) {
	clock := newFakeClock()
	cache := newTTLCache(maxEntries)
	cache.now = clock.now
	return cache, clock
}

// newTestCachingSource returns a cachingSource in front of a fakeSource,
// with a fake clock, plus a context that records hit/miss like the
// logging middleware does.
func newTestCachingSource(source *fakeSource) (*cachingSource, *fakeClock) {
	cs := newCachingSource(source)
	clock := newFakeClock()
	cs.cache.now = clock.now
	return cs, clock
}

// recordingContext returns a context that captures markCacheResult calls,
// the same way the logging middleware sets one up.
func recordingContext() (context.Context, *requestInfo) {
	info := &requestInfo{}
	return context.WithValue(context.Background(), requestInfoKey{}, info), info
}

func TestTTLCacheGetSetAndExpiry(t *testing.T) {
	cache, clock := newTestCache(10)

	if _, ok := cache.get("a"); ok {
		t.Fatal("empty cache returned an entry")
	}

	cache.set("a", "value", nil, time.Minute)

	clock.advance(59 * time.Second)
	if entry, ok := cache.get("a"); !ok || entry.value != "value" {
		t.Fatalf("entry missing before expiry: %+v %v", entry, ok)
	}

	clock.advance(1 * time.Second) // now exactly at the expiry time
	if _, ok := cache.get("a"); ok {
		t.Fatal("entry still returned at its expiry time")
	}
	if len(cache.entries) != 0 {
		t.Errorf("expired entry was not removed from the map")
	}
}

func TestTTLCacheSizeCapDropsExpiredFirst(t *testing.T) {
	cache, clock := newTestCache(3)

	cache.set("short", 1, nil, time.Minute)
	cache.set("long1", 2, nil, time.Hour)
	cache.set("long2", 3, nil, time.Hour)
	clock.advance(2 * time.Minute) // "short" has expired

	cache.set("new", 4, nil, time.Hour) // cache is full: must make room

	if len(cache.entries) != 3 {
		t.Errorf("cache has %d entries, want 3", len(cache.entries))
	}
	for _, key := range []string{"long1", "long2", "new"} {
		if _, ok := cache.get(key); !ok {
			t.Errorf("%q was dropped, but the expired entry should have gone first", key)
		}
	}
}

func TestTTLCacheSizeCapDropsSoonestToExpire(t *testing.T) {
	cache, _ := newTestCache(3)

	cache.set("a", 1, nil, 3*time.Hour)
	cache.set("b", 2, nil, 1*time.Hour) // expires soonest
	cache.set("c", 3, nil, 2*time.Hour)

	cache.set("d", 4, nil, time.Hour) // nothing expired, so drop "b"

	if _, ok := cache.get("b"); ok {
		t.Errorf("b should have been dropped (it expires soonest)")
	}
	for _, key := range []string{"a", "c", "d"} {
		if _, ok := cache.get(key); !ok {
			t.Errorf("%q should still be cached", key)
		}
	}
}

func TestTTLCacheUpdatingKeyDoesNotEvict(t *testing.T) {
	cache, _ := newTestCache(2)
	cache.set("a", 1, nil, time.Hour)
	cache.set("b", 2, nil, time.Hour)

	cache.set("a", 10, nil, time.Hour) // replaces "a"; the cache doesn't grow

	if entry, _ := cache.get("a"); entry.value != 10 {
		t.Errorf("a = %v, want 10", entry.value)
	}
	if _, ok := cache.get("b"); !ok {
		t.Errorf("b was dropped, but updating a key shouldn't need room")
	}
}

func TestTTLCacheNeverGrowsPastCap(t *testing.T) {
	cache, _ := newTestCache(50)
	for i := range 500 {
		cache.set(fmt.Sprintf("key%d", i), i, nil, time.Hour)
	}
	if len(cache.entries) != 50 {
		t.Errorf("cache has %d entries, want 50", len(cache.entries))
	}
}

// TestTTLCacheConcurrentUse has many goroutines use the cache at once, the
// way simultaneous requests do. On its own it only checks for crashes;
// run with `go test -race` and Go also reports any unsafe map access.
func TestTTLCacheConcurrentUse(t *testing.T) {
	cache := newTTLCache(20)
	var wg sync.WaitGroup
	for g := range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 200 {
				key := fmt.Sprintf("key%d", (g*7+i)%40)
				cache.set(key, i, nil, time.Minute)
				cache.get(key)
			}
		}()
	}
	wg.Wait()

	if len(cache.entries) > 20 {
		t.Errorf("cache grew to %d entries, cap is 20", len(cache.entries))
	}
}

func TestCachingSourceSearch(t *testing.T) {
	source := &fakeSource{searchResult: anilistSearchResult{HasNextPage: true, Media: []anilistMedia{{ID: 1}}}}
	cs, clock := newTestCachingSource(source)

	// 1st search: miss, so the source is asked.
	ctx, info := recordingContext()
	result, err := cs.searchManga(ctx, "One Piece", 1)
	if err != nil || len(result.Media) != 1 || info.cacheResult != cacheMiss {
		t.Fatalf("first search: %+v, %v, cache=%q", result, err, info.cacheResult)
	}

	// Same search with different case and spacing: hit.
	ctx, info = recordingContext()
	result, err = cs.searchManga(ctx, "  one   PIECE ", 1)
	if err != nil || len(result.Media) != 1 || !result.HasNextPage || info.cacheResult != cacheHit {
		t.Fatalf("repeat search: %+v, %v, cache=%q", result, err, info.cacheResult)
	}
	if source.searchCalls != 1 {
		t.Errorf("source called %d times, want 1", source.searchCalls)
	}

	// A different page is a different search: miss.
	cs.searchManga(context.Background(), "one piece", 2)
	if source.searchCalls != 2 {
		t.Errorf("page 2 should have asked the source")
	}

	// After 10 minutes the entry has expired: miss again.
	clock.advance(searchCacheTTL)
	cs.searchManga(context.Background(), "one piece", 1)
	if source.searchCalls != 3 {
		t.Errorf("expired search should have asked the source again")
	}
}

func TestCachingSourceDetail(t *testing.T) {
	source := &fakeSource{media: anilistMedia{ID: 53390}}
	cs, clock := newTestCachingSource(source)

	cs.getManga(context.Background(), 53390)

	clock.advance(detailCacheTTL - time.Second) // just before expiry
	ctx, info := recordingContext()
	media, err := cs.getManga(ctx, 53390)
	if err != nil || media.ID != 53390 || info.cacheResult != cacheHit {
		t.Fatalf("got %+v, %v, cache=%q; want a cache hit", media, err, info.cacheResult)
	}
	if source.getCalls != 1 {
		t.Errorf("source called %d times, want 1", source.getCalls)
	}

	clock.advance(time.Second) // now expired
	cs.getManga(context.Background(), 53390)
	if source.getCalls != 2 {
		t.Errorf("expired title should have asked the source again")
	}
}

func TestCachingSourceRemembersNotFound(t *testing.T) {
	source := &fakeSource{err: errNotFound}
	cs, clock := newTestCachingSource(source)

	cs.getManga(context.Background(), 999)

	ctx, info := recordingContext()
	_, err := cs.getManga(ctx, 999)
	if !errors.Is(err, errNotFound) || info.cacheResult != cacheHit {
		t.Fatalf("got %v, cache=%q; want a cached not-found", err, info.cacheResult)
	}
	if source.getCalls != 1 {
		t.Errorf("source called %d times, want 1", source.getCalls)
	}

	clock.advance(notFoundCacheTTL)
	cs.getManga(context.Background(), 999)
	if source.getCalls != 2 {
		t.Errorf("expired not-found should have asked the source again")
	}
}

func TestCachingSourceNeverCachesUpstreamErrors(t *testing.T) {
	for _, upstreamErr := range []error{errUpstreamUnavailable, errUpstreamBad, &rateLimitedError{RetryAfterSeconds: 30}} {
		t.Run(upstreamErr.Error(), func(t *testing.T) {
			source := &fakeSource{err: upstreamErr}
			cs, _ := newTestCachingSource(source)

			cs.getManga(context.Background(), 1)
			cs.getManga(context.Background(), 1)
			cs.searchManga(context.Background(), "x", 1)
			cs.searchManga(context.Background(), "x", 1)

			if source.getCalls != 2 || source.searchCalls != 2 {
				t.Errorf("errors were cached: get=%d search=%d calls, want 2 each", source.getCalls, source.searchCalls)
			}
		})
	}
}

func TestSearchCacheKey(t *testing.T) {
	if searchCacheKey("One  Piece ", 1) != searchCacheKey("one piece", 1) {
		t.Errorf("case and spacing should not matter")
	}
	if searchCacheKey("one piece", 1) == searchCacheKey("one piece", 2) {
		t.Errorf("different pages must have different keys")
	}
}
