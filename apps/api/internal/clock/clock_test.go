package clock_test

import (
	"testing"
	"time"

	"github.com/leotime/leotime/apps/api/internal/clock"
)

func TestInitFromEnvPinsNow(t *testing.T) {
	t.Setenv("LEOTIME_SEED_NOW", "2026-07-11T12:00:00Z")
	if err := clock.InitFromEnv(); err != nil {
		t.Fatalf("InitFromEnv: %v", err)
	}

	got := clock.Now()
	want := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("clock.Now() = %s, want %s", got, want)
	}
}

func TestInitFromEnvIgnoresEmptyValue(t *testing.T) {
	t.Setenv("LEOTIME_SEED_NOW", "")
	if err := clock.InitFromEnv(); err != nil {
		t.Fatalf("InitFromEnv: %v", err)
	}

	before := time.Now().UTC()
	got := clock.Now()
	after := time.Now().UTC()
	if got.Before(before.Add(-time.Second)) || got.After(after.Add(time.Second)) {
		t.Fatalf("clock.Now() = %s, expected near real time between %s and %s", got, before, after)
	}
}
