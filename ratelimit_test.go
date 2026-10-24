package main

import (
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
