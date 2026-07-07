package outbox

import (
	"math/rand"
	"testing"
	"time"
)

func TestRetryDelay(t *testing.T) {
	policy := DefaultRetryPolicy(time.Minute, 6*time.Hour)

	cases := []struct {
		attempts int
		want     time.Duration
	}{
		{attempts: 1, want: time.Minute},
		{attempts: 2, want: 2 * time.Minute},
		{attempts: 3, want: 4 * time.Minute},
		{attempts: 4, want: 8 * time.Minute},
		{attempts: 10, want: 6 * time.Hour},
	}

	for _, tc := range cases {
		got := RetryDelay(policy, tc.attempts)
		if got != tc.want {
			t.Fatalf("attempts=%d got %s want %s", tc.attempts, got, tc.want)
		}
	}
}

func TestNextRetryAtAddsJitterWithinBounds(t *testing.T) {
	policy := DefaultRetryPolicy(time.Minute, 6*time.Hour)
	now := time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC)
	rng := rand.New(rand.NewSource(42))

	next := NextRetryAt(policy, 1, now, rng)
	min := now.Add(time.Minute)
	max := now.Add(time.Minute + time.Duration(float64(time.Minute)*policy.JitterRatio))

	if next.Before(min) || next.After(max) {
		t.Fatalf("expected retry between %s and %s, got %s", min, max, next)
	}
}
