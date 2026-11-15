package landing

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestTheLandVerbsCleanTreeReadStartsNothingInASubmodule: the land verb's
// read of whether the checkout holds a change (dirtyTree) passes
// --ignore-submodules=dirty (gitutil.StatusArgs), so git starts no status
// inside a checked-out submodule and the content filter the submodule's own
// settings name does not run; a submodule whose checked-out commit moved
// still makes the tree dirty. Watched failing on the branch before, where
// the read ran the submodule's filter.
func TestTheLandVerbsCleanTreeReadStartsNothingInASubmodule(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sentinel script")
	}
	t.Setenv("HOME", t.TempDir())
	r := gittest.NewRepo(t)
	r.Write("a.txt", "a\n")
	r.Commit("seed")
	gitDir := r.AddSubmodule("sub")
	mark := filepath.Join(t.TempDir(), "sub-filter-ran")
	script := filepath.Join(t.TempDir(), "sub-clean.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ntouch "+mark+"\ncat\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.Git("config", "--file", filepath.Join(gitDir, "config"), "filter.subown.clean", script)
	if err := os.MkdirAll(filepath.Join(gitDir, "info"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "info", "attributes"), []byte("* filter=subown\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stale := func() {
		later := time.Now().Add(time.Hour)
		if err := os.Chtimes(filepath.Join(r.Root(), "sub", "s.txt"), later, later); err != nil {
			t.Fatal(err)
		}
	}
	stale()
	plain := exec.Command("git", "-C", r.Root(), "status", "--porcelain")
	plain.Env = r.Env()
	if out, err := plain.CombinedOutput(); err != nil {
		t.Fatalf("fixture: git status: %v\n%s", err, out)
	}
	if _, err := os.Stat(mark); err != nil {
		t.Fatal("fixture: a plain git status did not run the submodule's clean filter, so this test proves nothing")
	}
	for _, moved := range []bool{false, true} {
		if moved {
			r.Git("-C", "sub", "-c", "filter.subown.clean=", "commit", "-q", "--allow-empty", "-m", "moved")
		}
		_ = os.Remove(mark)
		stale()
		dirty, err := dirtyTree(r.Root())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(mark); err == nil {
			t.Errorf("submodule moved %v: the clean-tree read started a status inside the submodule, and its own clean filter ran", moved)
		}
		if dirty != moved {
			t.Errorf("submodule moved %v: dirty = %v", moved, dirty)
		}
	}
}
