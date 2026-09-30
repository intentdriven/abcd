package fsutil

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// WithDirLock refuses what is not a real directory — a symlink to one, a
// regular file — as ErrLockPathUnsafe, without following or creating
// anything, and an absent directory is the open's own not-exist error, so a
// caller can tell "no store" from "a store it may not lock".
func TestWithDirLockRefusesWhatIsNotARealDirectory(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "real")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(base, "file")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, path, says string
	}{
		{"a symlink to a directory", link, "symlink"},
		{"a regular file", file, "not a directory"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ran := false
			err := WithDirLock(tc.path, time.Second, func() error { ran = true; return nil })
			if !errors.Is(err, ErrLockPathUnsafe) || ran {
				t.Fatalf("ran=%v err=%v; want ErrLockPathUnsafe with fn not run", ran, err)
			}
			if !strings.Contains(err.Error(), tc.says) {
				t.Errorf("the refusal %q does not say %q", err, tc.says)
			}
		})
	}
	t.Run("an absent directory", func(t *testing.T) {
		absent := filepath.Join(base, "absent")
		ran := false
		err := WithDirLock(absent, time.Second, func() error { ran = true; return nil })
		if !errors.Is(err, fs.ErrNotExist) || !errors.Is(err, syscall.ENOENT) || ran {
			t.Fatalf("ran=%v err=%v; want the open's ENOENT with fn not run", ran, err)
		}
		if _, statErr := os.Lstat(absent); !errors.Is(statErr, fs.ErrNotExist) {
			t.Errorf("the lock planted %s: %v", absent, statErr)
		}
	})
}

// fn's own error passes through WithDirLock unwrapped, as WithFileLock's does.
func TestWithDirLockReturnsFnsErrorUnwrapped(t *testing.T) {
	want := errors.New("fn failed")
	if err := WithDirLock(t.TempDir(), time.Second, func() error { return want }); err != want {
		t.Fatalf("WithDirLock = %v; want fn's own error, unwrapped", err)
	}
}
