package main

import (
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"sync"
	"time"
)

// tokenBucket is a classic way to limit how often something happens.
//
// Picture a bucket that holds up to `capacity` tokens and is refilled at a
// steady rate. Each action takes one token. When the bucket is empty, the
// action has to wait for the next token (or is refused). This allows short
// bursts (up to capacity) while keeping the long-run average at the refill
// rate.
//
// In the worst case, a full bucket is used up and then every refilled
// token is used too, so the most actions in any one minute is
// capacity + perMinute. Keep that sum under the real limit.
//
// tokenBucket is not safe to use from several goroutines at once; the code
// that owns it must hold a mutex around every call.
type tokenBucket struct {
	capacity  float64
	perSecond float64 // refill rate
	tokens    float64 // can dip below 0 when tokens are reserved ahead of time
	updated   time.Time
}

// newTokenBucket returns a full bucket.
func newTokenBucket(capacity int, perMinute float64, now time.Time) *tokenBucket {
	return &tokenBucket{
		capacity:  float64(capacity),
		perSecond: perMinute / 60,
		tokens:    float64(capacity),
		updated:   now,
	}
}

// refill adds the tokens earned since the last update, up to capacity.
// Instead of a timer adding tokens in the background, we work out how many
// have "dripped in" whenever the bucket is used. Same result, less code.
func (b *tokenBucket) refill(now time.Time) {
	elapsed := now.Sub(b.updated).Seconds()
	if elapsed <= 0 {
		return
	}
	b.tokens = min(b.capacity, b.tokens+elapsed*b.perSecond)
	b.updated = now
}

// reserve takes one token if one is available now, or will be within
// maxWait. It returns how long the caller must wait before acting
// (0 means right away).
//
// If the wait would be longer than maxWait, nothing is taken, ok is false,
// and wait says how long the caller would have had to wait (useful for a
// Retry-After header).
func (b *tokenBucket) reserve(now time.Time, maxWait time.Duration) (wait time.Duration, ok bool) {
	b.refill(now)

	if b.tokens >= 1 {
		b.tokens--
		return 0, true
	}

	missing := 1 - b.tokens
	wait = time.Duration(missing / b.perSecond * float64(time.Second))
	if wait > maxWait {
		return wait, false
	}

	// Take the token now, even though it hasn't arrived yet. The count goes
	// below zero, so the next caller sees it's taken and waits even longer.
	b.tokens--
	return wait, true
}

// retryAfterSeconds turns a wait into whole seconds for a Retry-After
// header, rounding up (a 0.2s wait becomes 1s, never 0s).
func retryAfterSeconds(wait time.Duration) int {
	seconds := int(wait / time.Second)
	if wait%time.Second != 0 {
		seconds++
	}
	return max(seconds, 1)
}

// Per-visitor limits. These protect us (and our shared AniList allowance)
// from any single visitor sending too many requests, e.g. a script.
// Someone pressing Enter on searches and clicking through title pages
// never gets close.
const (
	perIPPerMinute = 30
	perIPBurst     = 10

	// maxTrackedIPs caps how many visitors we remember at once
	// (roughly 100 bytes each, so about 1 MB at most).
	maxTrackedIPs = 10_000

	// ipIdleTimeout: visitors not seen for this long are forgotten.
	// Forgetting them loses nothing: after 20 seconds of quiet their bucket
	// would be full again anyway, exactly like a brand-new visitor's.
	ipIdleTimeout = 5 * time.Minute

	// ipCleanupInterval is how often we look for idle visitors to forget.
	ipCleanupInterval = time.Minute
)

// ipLimiter gives every visitor (by IP address) their own token bucket.
// It's shared by all requests, so a mutex guards it.
type ipLimiter struct {
	mu          sync.Mutex
	visitors    map[string]*visitor
	maxVisitors int
	lastCleanup time.Time

	// now is replaceable so tests can control time.
	now func() time.Time
}

// visitor is one IP address's bucket, plus when we last saw it.
type visitor struct {
	bucket   *tokenBucket
	lastSeen time.Time
}

func newIPLimiter() *ipLimiter {
	return &ipLimiter{
		visitors:    make(map[string]*visitor),
		maxVisitors: maxTrackedIPs,
		lastCleanup: time.Now(),
		now:         time.Now,
	}
}

// allow reports whether the visitor identified by key may make a request
// now. If not, retryAfter says how long until they may.
func (l *ipLimiter) allow(key string) (ok bool, retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if now.Sub(l.lastCleanup) >= ipCleanupInterval {
		l.forgetIdle(now)
	}

	v, found := l.visitors[key]
	if !found {
		if len(l.visitors) >= l.maxVisitors {
			l.forgetIdle(now)
		}
		if len(l.visitors) >= l.maxVisitors {
			// We're tracking as many visitors as we allow and none are idle:
			// likely an attack from very many addresses. Let this one through
			// untracked rather than locking out real people; our AniList
			// limiter (anilist.go) still keeps AniList safe either way.
			slog.Warn("per-IP limiter is full; allowing an untracked visitor", "tracked", len(l.visitors))
			return true, 0
		}
		v = &visitor{bucket: newTokenBucket(perIPBurst, perIPPerMinute, now)}
		l.visitors[key] = v
	}

	v.lastSeen = now
	wait, ok := v.bucket.reserve(now, 0) // 0: never wait, answer right away
	return ok, wait
}

// forgetIdle removes visitors we haven't seen for ipIdleTimeout.
// The caller must already hold l.mu.
func (l *ipLimiter) forgetIdle(now time.Time) {
	for key, v := range l.visitors {
		if now.Sub(v.lastSeen) >= ipIdleTimeout {
			delete(l.visitors, key)
		}
	}
	l.lastCleanup = now
}

// limitPerIP wraps a handler so each visitor can only call it at the
// per-IP rate. Visitors over the limit get 429 with a Retry-After header.
func limitPerIP(limiter *ipLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok, wait := limiter.allow(clientKey(r))
		if !ok {
			w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds(wait)))
			writeError(w, http.StatusTooManyRequests, "too many requests, try again shortly")
			return
		}
		next(w, r)
	}
}

// clientKey works out which "visitor" a request comes from.
//
// IPv4 addresses are used as they are. IPv6 addresses are grouped by
// their first half (the "/64"): a single home connection is usually given
// a whole /64 block (billions of billions of addresses), so limiting each
// IPv6 address separately would be trivial to get around.
//
// IMPORTANT before deploying: r.RemoteAddr is whoever connected to us
// directly. Behind a hosting provider's proxy or load balancer, that's
// the proxy, so every visitor would share one limit. We'll then need to
// read the visitor's address from the X-Forwarded-For header, but only
// when the request really came from that trusted proxy, since anyone
// can send a fake X-Forwarded-For header.
func clientKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr) // "1.2.3.4:5678" -> "1.2.3.4"
	if err != nil {
		host = r.RemoteAddr
	}

	addr, err := netip.ParseAddr(host)
	if err != nil {
		return host // not an IP address; use it as is
	}

	// Unmap turns an IPv4 address written in IPv6 form (::ffff:1.2.3.4)
	// back into plain 1.2.3.4, so it's treated like any other IPv4 visitor.
	addr = addr.Unmap()
	if addr.Is6() {
		prefix, err := addr.Prefix(64)
		if err == nil {
			return prefix.String() // e.g. "2001:db8:1:2::/64"
		}
	}
	return addr.String()
}
