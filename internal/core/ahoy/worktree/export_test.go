package worktree

import "os"

// SwapFlushForTest replaces the archive's flush for one test, so the test can
// watch what is made durable and when; restore puts the real one back.
func SwapFlushForTest(fn func(*os.File) error) (restore func()) {
	prev := flush
	flush = fn
	return func() { flush = prev }
}

// SwapRemoveForTest replaces prune's removal of a worktree for one test.
func SwapRemoveForTest(fn func(repoRoot, path string) error) (restore func()) {
	prev := removeWorktree
	removeWorktree = fn
	return func() { removeWorktree = prev }
}
