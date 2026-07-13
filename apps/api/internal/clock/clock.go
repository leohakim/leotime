package clock

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	mu    sync.RWMutex
	nowFn = func() time.Time { return time.Now().UTC() }
)

func Now() time.Time {
	mu.RLock()
	defer mu.RUnlock()
	return nowFn()
}

func SetNow(fn func() time.Time) {
	mu.Lock()
	defer mu.Unlock()
	nowFn = fn
}

func InitFromEnv() error {
	raw := strings.TrimSpace(os.Getenv("LEOTIME_SEED_NOW"))
	if raw == "" {
		SetNow(func() time.Time { return time.Now().UTC() })
		return nil
	}

	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return fmt.Errorf("invalid LEOTIME_SEED_NOW %q: %w", raw, err)
	}

	fixed := parsed.UTC()
	SetNow(func() time.Time { return fixed })
	return nil
}
