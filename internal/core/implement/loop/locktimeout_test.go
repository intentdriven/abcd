package loop

import (
	"testing"
	"time"
)

// shortLockTimeout shortens how long a mutation waits for the run's lock
// before it reports contention, for the rest of t. A test that holds a lock
// to see the waiter back off proves the backoff, not the length of the wait,
// and at the shipped three seconds a waiter that retries spent most of its
// test asleep.
func shortLockTimeout(t *testing.T) {
	t.Helper()
	was := lockTimeout
	lockTimeout = 100 * time.Millisecond
	t.Cleanup(func() { lockTimeout = was })
}
