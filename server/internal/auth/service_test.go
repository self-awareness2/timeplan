package auth

import (
	"testing"
	"time"
)

func TestAttemptLimiterBlocksAndResets(t *testing.T) {
	limiter := newAttemptLimiter(2, time.Minute)
	if !limiter.Allow("127.0.0.1") || !limiter.Allow("127.0.0.1") {
		t.Fatal("expected attempts within the limit to pass")
	}
	if limiter.Allow("127.0.0.1") {
		t.Fatal("expected attempt above the limit to be blocked")
	}
	limiter.Reset("127.0.0.1")
	if !limiter.Allow("127.0.0.1") {
		t.Fatal("expected reset to allow a new attempt window")
	}
}
