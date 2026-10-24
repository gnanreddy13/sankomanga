package main

import "time"

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
