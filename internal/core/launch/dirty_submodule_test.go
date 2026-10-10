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

// TestDirtyTreeFilesRunsNoSubmoduleFilter is iss-2610091935327982: a diff
// against HEAD starts a status inside each checked-out submodule, which reads
// the submodule's own config, so a clean filter the submodule names runs
// although the superproject's filters are blanked. The check passes
// --ignore-submodules=dirty: content inside a submodule no longer makes the
// tree dirty, and a moved submodule pointer still does.
func TestDirtyTreeFilesRunsNoSubmoduleFilter(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sentinel script")
	}
	r := gittest.NewRepo(t)
	r.Write("a.txt", "a\n")
	r.Commit("seed")
	gitDir := r.AddSubmodule("sub")
	mark := filepath.Join(t.TempDir(), "sub-filter-ran")
	script := filepath.Join(t.TempDir(), "sub-clean.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho ran >> "+mark+"\ncat\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.Git("config", "--file", filepath.Join(gitDir, "config"), "filter.subevil.clean", script)
	if err := os.MkdirAll(filepath.Join(gitDir, "info"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "info", "attributes"), []byte("* filter=subevil\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// submodule.<name>.ignore=none in the superproject would beat a
	// diff.ignoreSubmodules config; the flag beats both.
	r.Git("config", "submodule.sub.ignore", "none")
	stale := func(age time.Duration) {
		old := time.Now().Add(-age)
		if err := os.Chtimes(filepath.Join(r.Root(), "sub", "s.txt"), old, old); err != nil {
			t.Fatal(err)
		}
	}
	stale(48 * time.Hour)

	cmd := exec.Command("git", "-C", r.Root(), "diff", "--name-only", "HEAD")
	cmd.Env = r.Env()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixture diff: %v: %s", err, out)
	}
	if !markRan(t, mark) {
		t.Fatal("fixture: a plain git diff did not run the submodule's clean filter, so this test proves nothing")
	}
	if err := os.Remove(mark); err != nil {
		t.Fatal(err)
	}
	stale(24 * time.Hour)

	dirty, err := DirtyTreeFiles(r.Root())
	if err != nil {
		t.Fatalf("DirtyTreeFiles: %v", err)
	}
	if markRan(t, mark) {
		t.Fatal("DirtyTreeFiles ran the submodule's own clean filter")
	}
	if len(dirty) != 0 {
		t.Fatalf("an unchanged submodule must read clean, got %v", dirty)
	}

	// Uncommitted content inside the submodule is not the superproject's dirt.
	if err := os.WriteFile(filepath.Join(r.Root(), "sub", "s.txt"), []byte("edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if dirty, err = DirtyTreeFiles(r.Root()); err != nil || len(dirty) != 0 {
		t.Fatalf("an edit inside a submodule must not dirty the tree: %v %v", dirty, err)
	}
	if markRan(t, mark) {
		t.Fatal("DirtyTreeFiles ran the submodule's own clean filter over an edited file")
	}

	// A moved pointer is.
	move := exec.Command("git", "-C", filepath.Join(r.Root(), "sub"),
		"-c", "user.name=F", "-c", "user.email=f@example.invalid", "-c", "commit.gpgsign=false",
		"commit", "-q", "--allow-empty", "-m", "moved")
	move.Env = r.Env()
	if out, err := move.CombinedOutput(); err != nil {
		t.Fatalf("moving the submodule: %v\n%s", err, out)
	}
	_ = os.Remove(mark)
	dirty, err = DirtyTreeFiles(r.Root())
	if err != nil {
		t.Fatalf("DirtyTreeFiles after a move: %v", err)
	}
	if markRan(t, mark) {
		t.Fatal("DirtyTreeFiles ran the submodule's own clean filter over a moved pointer")
	}
	if strings.Join(dirty, ",") != "sub" {
		t.Fatalf("a moved submodule pointer must still be listed, got %v", dirty)
	}
}
