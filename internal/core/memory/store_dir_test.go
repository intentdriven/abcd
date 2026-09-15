package memory

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestMemoryDirIsOneWalker pins the GHSA-72rp-qxm2-r8vq guard to a single
// implementation: the read side (never creates) and the write side (creates
// when absent) resolve the store through one walker, and that walker always
// hands back the canonical Dir(repoRoot) — present or not — so no caller has
// to re-derive the path it withheld. Two copies of the segment walk meant a
// hardening fix to one left the other open.
func TestMemoryDirIsOneWalker(t *testing.T) {
	t.Run("absent store still resolves to the canonical path", func(t *testing.T) {
		repo := t.TempDir()
		mem, present, err := safeMemoryDir(repo)
		if err != nil {
			t.Fatal(err)
		}
		if present {
			t.Fatal("an absent store reported present")
		}
		if mem != Dir(repo) {
			t.Errorf("safeMemoryDir(absent) = %q, want the canonical %q", mem, Dir(repo))
		}
		if _, err := os.Lstat(Dir(repo)); !os.IsNotExist(err) {
			t.Errorf("the read side materialised the store: %v", err)
		}
	})

	t.Run("both sides refuse a symlinked store identically", func(t *testing.T) {
		repo := t.TempDir()
		outside := t.TempDir()
		if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(repo, ".abcd", "memory")); err != nil {
			t.Skipf("symlink unsupported: %v", err)
		}
		_, _, readErr := safeMemoryDir(repo)
		_, writeErr := validatedMemoryDir(repo)
		var readUnsafe, writeUnsafe *UnsafeStorePathError
		if !errors.As(readErr, &readUnsafe) {
			t.Fatalf("read side returned %T (%v); want *UnsafeStorePathError", readErr, readErr)
		}
		if !errors.As(writeErr, &writeUnsafe) {
			t.Fatalf("write side returned %T (%v); want *UnsafeStorePathError", writeErr, writeErr)
		}
		if readUnsafe.Msg != writeUnsafe.Msg {
			t.Errorf("the two sides refuse with different messages:\n read: %s\nwrite: %s", readUnsafe.Msg, writeUnsafe.Msg)
		}
	})
}
