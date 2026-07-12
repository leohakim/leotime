package main

import (
	"fmt"
	"os"
	"time"

	"github.com/leotime/leotime/apps/api/internal/seed"
	"github.com/leotime/leotime/apps/api/internal/store"
)

func seedServiceForCommand(st *store.Store) (*seed.Service, error) {
	raw := os.Getenv("LEOTIME_SEED_NOW")
	if raw == "" {
		return seed.New(st), nil
	}

	fixed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, fmt.Errorf("invalid LEOTIME_SEED_NOW %q: %w", raw, err)
	}

	return seed.NewWithNow(st, func() time.Time {
		return fixed.UTC()
	}), nil
}
