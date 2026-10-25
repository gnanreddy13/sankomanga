package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// assertAbout checks that a duration is within a millisecond of want.
// (Token maths uses fractions, so 3s may come out as 2.9999999s.)
func assertAbout(t *testing.T, got, want time.Duration) {
	t.Helper()
	if diff := got - want; diff < -time.Millisecond || diff > time.Millisecond {
		t.Errorf("wait = %v, want about %v", got, want)
	}
}

func TestTokenBucketBurstThenRefill(t *testing.T) {
	clock := newFakeClock()
	bucket := newTokenBucket(5, 20, clock.now()) // 20/min = one token every 3s

	// A full bucket allows a burst of 5 right away.
	for i := range 5 {
		if wait, ok := bucket.reserve(clock.now(), 0); !ok || wait != 0 {
			t.Fatalf("call %d: wait=%v ok=%v, want immediate", i+1, wait, ok)
		}
	}

	// The 6th must wait 3s for the next token; with maxWait 0 it's refused.
	wait, ok := bucket.reserve(clock.now(), 0)
	if ok {
		t.Fatal("6th call allowed, but the bucket is empty")
	}
	assertAbout(t, wait, 3*time.Second)

	// 3 seconds later, one token has dripped back in.
	clock.advance(3 * time.Second)
	if _, ok := bucket.reserve(clock.now(), 0); !ok {
		t.Error("call refused after the bucket refilled a token")
	}
}

func TestTokenBucketNeverHoldsMoreThanCapacity(t *testing.T) {
	clock := newFakeClock()
	bucket := newTokenBucket(5, 20, clock.now())

	clock.advance(time.Hour) // a long quiet spell must not save up 1,200 tokens

	for range 5 {
		bucket.reserve(clock.now(), 0)
	}
	if _, ok := bucket.reserve(clock.now(), 0); ok {
		t.Error("bucket allowed more than its capacity after a long idle period")
	}
}

func TestTokenBucketReserveAhead(t *testing.T) {
	clock := newFakeClock()
	bucket := newTokenBucket(1, 20, clock.now())
	bucket.reserve(clock.now(), 0) // empty the bucket

	// Next token is 3s away, within maxWait: reserved, wait 3s.
	wait, ok := bucket.reserve(clock.now(), 3*time.Second)
	if !ok {
		t.Fatal("reservation within maxWait was refused")
	}
	assertAbout(t, wait, 3*time.Second)

	// The one after that is 6s away, past maxWait: refused, and nothing taken.
	wait, ok = bucket.reserve(clock.now(), 3*time.Second)
	if ok {
		t.Fatal("reservation beyond maxWait was allowed")
	}
	assertAbout(t, wait, 6*time.Second)

	// Since the refused call took nothing, 6s later a token is available now.
	clock.advance(6 * time.Second)
	if wait, ok := bucket.reserve(clock.now(), 0); !ok || wait != 0 {
		t.Errorf("wait=%v ok=%v, want an immediate token", wait, ok)
	}
}

func TestRetryAfterSeconds(t *testing.T) {
	tests := []struct {
		wait time.Duration
		want int
	}{
		{0, 1}, // never tell a client "retry after 0"
		{200 * time.Millisecond, 1},
		{3 * time.Second, 3},
		{3*time.Second + time.Millisecond, 4}, // rounds up
		{time.Minute, 60},
	}
	for _, tt := range tests {
		if got := retryAfterSeconds(tt.wait); got != tt.want {
			t.Errorf("retryAfterSeconds(%v) = %d, want %d", tt.wait, got, tt.want)
		}
	}
}

// newTestIPLimiter returns a per-IP limiter with a fake clock.
func newTestIPLimiter(maxVisitors int) (*ipLimiter, *fakeClock) {
	clock := newFakeClock()
	limiter := newIPLimiter()
	limiter.now = clock.now
	limiter.lastCleanup = clock.now()
	limiter.maxVisitors = maxVisitors
	return limiter, clock
}

func TestIPLimiterBurstAndRefill(t *testing.T) {
	limiter, clock := newTestIPLimiter(maxTrackedIPs)

	for i := range perIPBurst {
		if ok, _ := limiter.allow("192.0.2.1"); !ok {
			t.Fatalf("request %d refused, want a burst of %d allowed", i+1, perIPBurst)
		}
	}

	ok, wait := limiter.allow("192.0.2.1")
	if ok {
		t.Fatal("request over the burst was allowed")
	}
	assertAbout(t, wait, 2*time.Second) // 30/min = one every 2s

	// Another visitor is not affected.
	if ok, _ := limiter.allow("192.0.2.2"); !ok {
		t.Error("a different visitor was refused")
	}

	clock.advance(2 * time.Second)
	if ok, _ := limiter.allow("192.0.2.1"); !ok {
		t.Error("request refused after waiting for a refill")
	}
}

func TestIPLimiterForgetsIdleVisitors(t *testing.T) {
	limiter, clock := newTestIPLimiter(maxTrackedIPs)

	limiter.allow("192.0.2.1")
	clock.advance(ipIdleTimeout)
	limiter.allow("192.0.2.2") // cleanup is due, so 192.0.2.1 gets forgotten

	if _, found := limiter.visitors["192.0.2.1"]; found {
		t.Error("idle visitor was not forgotten")
	}
	if len(limiter.visitors) != 1 {
		t.Errorf("tracking %d visitors, want 1", len(limiter.visitors))
	}
}

func TestIPLimiterWhenFull(t *testing.T) {
	captureLogs(t) // keep the expected warning out of the test output
	limiter, clock := newTestIPLimiter(2)

	limiter.allow("a")
	clock.advance(30 * time.Second)
	limiter.allow("b")

	// Full, and nobody is idle: a new visitor is let through, untracked.
	if ok, _ := limiter.allow("c"); !ok {
		t.Error("new visitor refused when the limiter was full")
	}
	if _, found := limiter.visitors["c"]; found || len(limiter.visitors) != 2 {
		t.Errorf("limiter grew past its cap: %d visitors", len(limiter.visitors))
	}

	// Once "a" has gone idle, a new visitor takes its place and is tracked.
	clock.advance(ipIdleTimeout - 30*time.Second)
	limiter.allow("d")
	if _, found := limiter.visitors["d"]; !found {
		t.Error("new visitor not tracked after an idle one was forgotten")
	}
	if _, found := limiter.visitors["a"]; found {
		t.Error("idle visitor should have been forgotten to make room")
	}
}

func TestClientKey(t *testing.T) {
	tests := []struct {
		remoteAddr string
		want       string
	}{
		{"192.0.2.1:1234", "192.0.2.1"},
		// Two addresses in the same IPv6 /64 count as one visitor...
		{"[2001:db8:1:2:aaaa::1]:443", "2001:db8:1:2::/64"},
		{"[2001:db8:1:2:bbbb::9]:443", "2001:db8:1:2::/64"},
		// ...but a different /64 is a different visitor.
		{"[2001:db8:1:3::1]:443", "2001:db8:1:3::/64"},
		// IPv4 written in IPv6 form is treated as plain IPv4.
		{"[::ffff:192.0.2.1]:80", "192.0.2.1"},
		// Not an IP address at all: used as is.
		{"not-an-address", "not-an-address"},
	}
	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = tt.remoteAddr
		if got := clientKey(req, ""); got != tt.want {
			t.Errorf("clientKey(%q) = %q, want %q", tt.remoteAddr, got, tt.want)
		}
	}
}

// TestClientKeyBehindProxy checks that, behind a proxy, visitors are told
// apart by the address in the proxy's header, not by the proxy's own
// address (which every request shares).
func TestClientKeyBehindProxy(t *testing.T) {
	const proxy = "10.0.0.1:5000" // every request arrives from the proxy

	tests := []struct {
		name   string
		header string
		want   string
	}{
		{"visitor's address", "203.0.113.7", "203.0.113.7"},
		{"first of a list is the visitor", "203.0.113.7, 10.0.0.2", "203.0.113.7"},
		{"spaces are ignored", "  203.0.113.7 ", "203.0.113.7"},
		{"IPv6 grouped by /64 as usual", "2001:db8:1:2::5", "2001:db8:1:2::/64"},
		// Missing or nonsense values fall back to the connection's address.
		{"header missing", "", "10.0.0.1"},
		{"not an address", "hello", "10.0.0.1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = proxy
			if tt.header != "" {
				req.Header.Set("X-Real-IP", tt.header)
			}
			if got := clientKey(req, "X-Real-IP"); got != tt.want {
				t.Errorf("clientKey = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestClientKeyIgnoresHeaderWithoutProxy checks that when no proxy header
// is configured (running locally, or a host without a proxy), a made-up
// header can't change who a visitor counts as.
func TestClientKeyIgnoresHeaderWithoutProxy(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	req.Header.Set("X-Real-IP", "203.0.113.7")
	if got := clientKey(req, ""); got != "192.0.2.1" {
		t.Errorf("clientKey = %q, want the connection's address 192.0.2.1", got)
	}
}

// TestRoutesPerIPLimit checks the limit through the real router.
func TestRoutesPerIPLimit(t *testing.T) {
	router := newRouter(&app{source: &fakeSource{}})

	get := func(path, remoteAddr string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.RemoteAddr = remoteAddr
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	// Search and title pages share one allowance.
	for i := range perIPBurst {
		path := "/api/search?q=x"
		if i%2 == 1 {
			path = "/api/manga/1"
		}
		if rec := get(path, "192.0.2.1:1000"); rec.Code != http.StatusOK {
			t.Fatalf("request %d: status %d, want 200", i+1, rec.Code)
		}
	}

	rec := get("/api/manga/1", "192.0.2.1:1000")
	assertJSON(t, rec, http.StatusTooManyRequests, `{"error":"too many requests, try again shortly"}`)
	if got := rec.Header().Get("Retry-After"); got != "2" {
		t.Errorf("Retry-After = %q, want \"2\"", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}

	// Health checks are never limited, and other visitors are unaffected.
	if rec := get("/api/health", "192.0.2.1:1000"); rec.Code != http.StatusOK {
		t.Errorf("health check status %d, want 200", rec.Code)
	}
	if rec := get("/api/search?q=x", "192.0.2.2:1000"); rec.Code != http.StatusOK {
		t.Errorf("other visitor status %d, want 200", rec.Code)
	}
}
