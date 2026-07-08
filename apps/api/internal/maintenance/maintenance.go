package maintenance

import (
	"sync/atomic"
)

var active atomic.Bool

// Enter blocks API writes and background jobs until Leave is called or the process restarts.
func Enter() {
	active.Store(true)
}

// Leave clears maintenance mode after a failed restore or tests.
func Leave() {
	active.Store(false)
}

// Enabled reports whether the server is in maintenance mode.
func Enabled() bool {
	return active.Load()
}
