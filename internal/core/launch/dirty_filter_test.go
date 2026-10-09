package launch

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// filterRepo is a committed checkout whose own config names a content filter
// for every path, the filter being a script that records each run and passes
// its input through. The file's mtime is moved after the commit, so the index
// stat no longer matches and git re-hashes the file — what a copied checkout
// does.
func filterRepo(t *testing.T, keys ...string) (*gittest.Repo, string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sentinel script")
	}
	r := gittest.NewRepo(t)
	r.Write("a.txt", "hello\n")
	r.Commit("seed")
	mark := filepath.Join(t.TempDir(), "filter-ran")
	script := filepath.Join(t.TempDir(), "evil-filter.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho ran >> "+mark+"\ncat\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, k := range keys {
		r.Git("config", k, script)
	}
	if err := os.WriteFile(filepath.Join(r.Root(), ".git", "info", "attributes"), []byte("* filter=evil\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(filepath.Join(r.Root(), "a.txt"), old, old); err != nil {
		t.Fatal(err)
	}
	return r, mark
}

func markRan(t *testing.T, mark string) bool {
	t.Helper()
	_, err := os.Stat(mark)
	return err == nil
}

// TestDirtyTreeFilesRunsNoRepoFilter is iss-2610090821548169: the launch dirty
// check diffs the working tree against HEAD, and over a tree whose index stat
// no longer matches git re-hashes each file through the repository's clean
// filter. The check must compare bytes without running it, still report a
// byte-identical tree clean, and still list a real edit.
func TestDirtyTreeFilesRunsNoRepoFilter(t *testing.T) {
	r, mark := filterRepo(t, "filter.evil.clean")

	// The fixture is live: a plain diff under the same environment runs it.
	cmd := exec.Command("git", "-C", r.Root(), "diff", "--name-only", "HEAD")
	cmd.Env = r.Env()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixture diff: %v: %s", err, out)
	}
	if !markRan(t, mark) {
		t.Fatal("fixture: a plain git diff did not run the clean filter, so this test proves nothing")
	}
	if err := os.Remove(mark); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-24 * time.Hour)
	if err := os.Chtimes(filepath.Join(r.Root(), "a.txt"), old, old); err != nil {
		t.Fatal(err)
	}

	dirty, err := DirtyTreeFiles(r.Root())
	if err != nil {
		t.Fatalf("DirtyTreeFiles: %v", err)
	}
	if markRan(t, mark) {
		t.Fatal("DirtyTreeFiles ran the repository's clean filter")
	}
	if len(dirty) != 0 {
		t.Fatalf("a byte-identical tree must read clean, got %v", dirty)
	}

	r.Write("a.txt", "edited\n")
	dirty, err = DirtyTreeFiles(r.Root())
	if err != nil {
		t.Fatalf("DirtyTreeFiles after an edit: %v", err)
	}
	if markRan(t, mark) {
		t.Fatal("DirtyTreeFiles ran the repository's clean filter over an edited file")
	}
	if strings.Join(dirty, ",") != "a.txt" {
		t.Fatalf("a real edit must still be listed, got %v", dirty)
	}
}

// TestDirtyTreeFilesRunsNoRepoFilterProcess is the same refusal for the
// long-running filter protocol: blanking clean alone leaves process live.
func TestDirtyTreeFilesRunsNoRepoFilterProcess(t *testing.T) {
	r, mark := filterRepo(t, "filter.evil.process")
	if _, err := DirtyTreeFiles(r.Root()); err != nil {
		t.Fatalf("DirtyTreeFiles: %v", err)
	}
	if markRan(t, mark) {
		t.Fatal("DirtyTreeFiles started the repository's filter process")
	}
}

// TestDirtyTreeFilesFailsClosedOnARequiredFilter: a filter git insists on,
// with its commands blanked, makes git refuse; the check reports the tree
// unreadable, never clean, and still runs nothing.
func TestDirtyTreeFilesFailsClosedOnARequiredFilter(t *testing.T) {
	r, mark := filterRepo(t, "filter.evil.clean")
	r.Git("config", "filter.evil.required", "true")
	if dirty, err := DirtyTreeFiles(r.Root()); err == nil {
		t.Fatalf("a required filter that cannot run must not read as a tree state, got %v", dirty)
	}
	if markRan(t, mark) {
		t.Fatal("DirtyTreeFiles ran the repository's required clean filter")
	}
}
