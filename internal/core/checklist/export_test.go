package checklist

import (
	"context"
	"os"
	"time"
)

// SetLockTimeout shortens Update's wait for the lock in a test and returns
// the restore.
func SetLockTimeout(d time.Duration) (restore func()) {
	was := lockTimeout
	lockTimeout = d
	return func() { lockTimeout = was }
}

// SwapOwnerForTest substitutes the owner lookup Read applies to each file,
// since a test process can make only its own files, and returns the restore.
func SwapOwnerForTest(fn func(os.FileInfo) (uint32, error)) (restore func()) {
	was := ownerOf
	ownerOf = fn
	return func() { ownerOf = was }
}

// SwapGitForTest substitutes the git runner the tracked check uses, so a git
// that fails inside a checkout can be provoked, and returns the restore.
func SwapGitForTest(fn func(string, ...string) (string, error)) (restore func()) {
	was := gitRun
	gitRun = func(_ context.Context, root string, args ...string) (string, error) { return fn(root, args...) }
	return func() { gitRun = was }
}

// Decode is decode, for the schema tests.
var Decode = decode
