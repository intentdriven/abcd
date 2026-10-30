//go:build unix

package gitutil

import "testing"

// StopGrace is how long a stopped commit has to end before the WaitDelay's
// SIGKILL.
const StopGrace = stopGrace

// StopWithDescendants is the stop a bounded commit installs on its git.
var StopWithDescendants = stopWithDescendants

// SetAfterSnapshotForTest runs f once the stop has read and frozen git's
// descendants, before it signals them, for the length of t.
func SetAfterSnapshotForTest(t *testing.T, f func()) {
	t.Helper()
	afterSnapshot = f
	t.Cleanup(func() { afterSnapshot = nil })
}

// SetSweepSeamsForTest runs stop as the sweep reaches its deadline, before it
// freezes the survivors, and kill before its SIGKILL, for the length of t.
func SetSweepSeamsForTest(t *testing.T, stop, kill func()) {
	t.Helper()
	beforeSweepStop, beforeSweepKill = stop, kill
	t.Cleanup(func() { beforeSweepStop, beforeSweepKill = nil, nil })
}

// SetPSPathsForTest makes the process-table read try paths in place of ps,
// for the length of t.
func SetPSPathsForTest(t *testing.T, paths []string) {
	t.Helper()
	was := psPaths
	psPaths = paths
	t.Cleanup(func() { psPaths = was })
}
