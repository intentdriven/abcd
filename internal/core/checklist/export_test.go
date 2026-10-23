package checklist

import "time"

// SetLockTimeout shortens Update's wait for the lock in a test and returns
// the restore.
func SetLockTimeout(d time.Duration) (restore func()) {
	was := lockTimeout
	lockTimeout = d
	return func() { lockTimeout = was }
}
