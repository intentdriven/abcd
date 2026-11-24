//go:build unix

package matrix

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestACloneWhoseShallowFileIsAPipeIsRefusedAtOnce: a named pipe where a
// clone's shallow file stands is refused as no copy's clone, without waiting
// for anything to write to it.
func TestACloneWhoseShallowFileIsAPipeIsRefusedAtOnce(t *testing.T) {
	src, frozen := cloneSource(t)
	home := t.TempDir()
	key := TrialKey{Task: "widget", Cell: "cheap", Repeat: 1}
	dir, err := makeClone(src.Root(), cloneStore(home, gittest.RootCommit(t, src.Root())), cloneRun, frozen, key, keyAlias(key))
	if err != nil {
		t.Fatal(err)
	}
	shallow := filepath.Join(dir, ".git", "shallow")
	if err := os.Remove(shallow); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(shallow, 0o600); err != nil {
		t.Skipf("no named pipe can be made here: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- CopyClone(home, dir, frozen, "build/", true) }()
	select {
	case err := <-done:
		if !errors.Is(err, ErrNotACopyClone) {
			t.Fatalf("a clone whose shallow file is a pipe is refused: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("judging a clone whose shallow file is a pipe waited on the pipe")
	}
}
