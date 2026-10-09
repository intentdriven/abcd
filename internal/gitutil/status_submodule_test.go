package gitutil_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// submoduleFilterFixture is a superproject with a checked-out submodule whose
// OWN config names a clean filter for every path, the filter a script that
// records each run. The submodule's file has a stale stat, so a status that
// looks inside the submodule re-hashes it through that filter. The
// superproject configures no filter, so FilterOverrides has nothing to blank.
func submoduleFilterFixture(t *testing.T) (r *gittest.Repo, mark string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sentinel script")
	}
	t.Setenv("HOME", t.TempDir())
	r = gittest.NewRepo(t)
	r.Write("a.txt", "a\n")
	r.Commit("seed")
	gitDir := r.AddSubmodule("sub")
	mark = filepath.Join(t.TempDir(), "sub-filter-ran")
	script := filepath.Join(t.TempDir(), "sub-clean.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ntouch "+mark+"\ncat\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.Git("config", "--file", filepath.Join(gitDir, "config"), "filter.subevil.clean", script)
	if err := os.MkdirAll(filepath.Join(gitDir, "info"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "info", "attributes"), []byte("* filter=subevil\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// submodule.<name>.ignore=none in the superproject beats a
	// diff.ignoreSubmodules config; the flag beats both.
	r.Git("config", "submodule.sub.ignore", "none")
	staleSub(t, r)
	return r, mark
}

func staleSub(t *testing.T, r *gittest.Repo) {
	t.Helper()
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(filepath.Join(r.Root(), "sub", "s.txt"), later, later); err != nil {
		t.Fatal(err)
	}
}

// TestStatusRunsNoSubmoduleContentFilter is iss-2610091935327982: without
// --ignore-submodules git status starts a status inside each checked-out
// submodule, which reads the submodule's own config, so a clean filter the
// submodule names runs although Status blanked every superproject filter. A
// changed submodule pointer is still listed.
func TestStatusRunsNoSubmoduleContentFilter(t *testing.T) {
	r, mark := submoduleFilterFixture(t)

	// The fixture is live: a plain git status runs the submodule's filter.
	plain := exec.Command("git", "-C", r.Root(), "status", "--porcelain")
	plain.Env = r.Env()
	if out, err := plain.CombinedOutput(); err != nil {
		t.Fatalf("fixture: git status: %v\n%s", err, out)
	}
	if !ran(mark) {
		t.Fatal("fixture: a plain git status did not run the submodule's clean filter, so this test proves nothing")
	}
	if err := os.Remove(mark); err != nil {
		t.Fatal(err)
	}
	staleSub(t, r)

	entries, err := gitutil.Status(r.Root(), 1<<20, gitutil.StatusOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if ran(mark) {
		t.Fatal("Status ran the submodule's own clean filter")
	}
	if len(entries) != 0 {
		t.Fatalf("an unchanged submodule reads as unchanged: %+v", entries)
	}

	// A new commit checked out inside the submodule moves its pointer, which
	// the superproject still sees.
	sub := exec.Command("git", "-C", filepath.Join(r.Root(), "sub"),
		"-c", "user.name=F", "-c", "user.email=f@example.invalid", "-c", "commit.gpgsign=false",
		"-c", "filter.subevil.clean=", "commit", "-q", "--allow-empty", "-m", "moved")
	sub.Env = r.Env()
	if out, err := sub.CombinedOutput(); err != nil {
		t.Fatalf("moving the submodule: %v\n%s", err, out)
	}
	_ = os.Remove(mark)
	entries, err = gitutil.Status(r.Root(), 1<<20, gitutil.StatusOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if ran(mark) {
		t.Fatal("Status ran the submodule's own clean filter over a moved pointer")
	}
	if len(entries) != 1 || entries[0].Path != "sub" {
		t.Fatalf("a moved submodule pointer must still be listed: %+v", entries)
	}
}
