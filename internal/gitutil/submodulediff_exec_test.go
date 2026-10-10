package gitutil_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestIsolatedDiffStartsNoInnerSubmoduleDiff is iss-2610091935325886: with
// diff.submodule=diff in the superproject, a patch diff over a moved submodule
// pointer starts a second git diff inside the submodule, and git passes it
// none of the parent's --no-ext-diff/--no-textconv, so a diff.external the
// submodule's own config names runs and writes into the text a check parses.
// ExecPins forces diff.submodule=short on every isolated command: the
// submodule reads as the pointer change alone.
func TestIsolatedDiffStartsNoInnerSubmoduleDiff(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sentinel script")
	}
	t.Setenv("HOME", t.TempDir())
	r := gittest.NewRepo(t)
	r.Write("a.txt", "a\n")
	r.Commit("seed")
	gitDir := r.AddSubmodule("sub")
	before := r.Git("rev-parse", "HEAD")

	// Move the submodule to a new commit and record the move.
	sub := filepath.Join(r.Root(), "sub")
	if err := os.WriteFile(filepath.Join(sub, "s.txt"), []byte("s2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"add", "s.txt"},
		{"-c", "user.name=F", "-c", "user.email=f@example.invalid", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "moved"},
	} {
		c := exec.Command("git", append([]string{"-C", sub}, args...)...)
		c.Env = r.Env()
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v in the submodule: %v\n%s", args, err, out)
		}
	}
	r.Commit("move submodule")

	mark := filepath.Join(t.TempDir(), "ext-diff-ran")
	script := filepath.Join(t.TempDir(), "ext-diff.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ntouch "+mark+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.Git("config", "--file", filepath.Join(gitDir, "config"), "diff.external", script)
	r.Git("config", "diff.submodule", "diff")

	// The fixture is live: the same diff outside the isolated runner starts
	// the submodule's external diff.
	plain := exec.Command("git", "-C", r.Root(), "diff", "--no-ext-diff", "--no-textconv", before, "HEAD")
	plain.Env = r.Env()
	if out, err := plain.CombinedOutput(); err != nil {
		t.Fatalf("fixture diff: %v\n%s", err, out)
	}
	if !ran(mark) {
		t.Fatal("fixture: diff.submodule=diff did not start the submodule's external diff, so this test proves nothing")
	}
	if err := os.Remove(mark); err != nil {
		t.Fatal(err)
	}

	out, err := gitutil.Run(r.Root(), "diff", "--no-ext-diff", "--no-textconv", before, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if ran(mark) {
		t.Fatal("an isolated diff started the submodule's own diff.external")
	}
	if want := "Subproject commit"; !strings.Contains(out, want) {
		t.Fatalf("the submodule must read as a pointer change (%q), got:\n%s", want, out)
	}
}
