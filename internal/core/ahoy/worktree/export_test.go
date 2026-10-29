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

// StartForTest reads the start record add kept for the worktree name in l.
func StartForTest(l Lane, name string) (branch, start string, ok bool) {
	rec, ok := readStart(l, name)
	return rec.Branch, rec.Start, ok
}

// SwapOwnerForTest replaces the owner lookup for one test, so a test can stand
// in for a worktree another account owns, which it cannot make without root.
func SwapOwnerForTest(fn func(path string) (uint32, error)) (restore func()) {
	prev := ownerOf
	ownerOf = fn
	return func() { ownerOf = prev }
}

// SwapIgnoredForTest replaces the ignored-path listing for one test, so a test
// can stand in for a git that refuses it.
func SwapIgnoredForTest(fn func(root, rel string) ([]string, error)) (restore func()) {
	prev := ignoredUnder
	ignoredUnder = fn
	return func() { ignoredUnder = prev }
}
