package loop

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestTheConflictProbeRenormalisesNothing: the probe predicts the merge the
// forge computes, and the forge does not renormalise. A checkout configuring
// merge.renormalize and a content filter, with an attribute naming the filter
// only in the lane's commit, makes a plain merge-tree run both of the filter's
// commands over the conflicted path; the probe runs neither, and still names
// the path.
func TestTheConflictProbeRenormalisesNothing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sentinel script")
	}
	r := gittest.NewRepo(t)
	r.Write("f.txt", "a\nb\nc\nd\ne\nf\ng\n")
	r.Commit("base")
	r.Git("checkout", "-q", "-b", "side")
	r.Write("f.txt", "a\nTHEIRS\nc\nd\ne\nf\ng\n")
	r.Write(".gitattributes", "f.txt filter=evil\n")
	r.Commit("side")
	head := r.Git("rev-parse", "HEAD")
	r.Git("checkout", "-q", "main")
	r.Write("f.txt", "a\nOURS\nc\nd\ne\nf\ng\n")
	r.Commit("main")

	marks := t.TempDir()
	filter := filepath.Join(t.TempDir(), "evil-filter.sh")
	if err := os.WriteFile(filter, []byte("#!/bin/sh\ntouch \""+marks+"/$1\"\nexec cat\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.Git("config", "merge.renormalize", "true")
	r.Git("config", "filter.evil.clean", filter+" clean")
	r.Git("config", "filter.evil.smudge", filter+" smudge")
	ran := func() []string {
		t.Helper()
		es, err := os.ReadDir(marks)
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, e := range es {
			names = append(names, e.Name())
		}
		return names
	}

	// The fixture is live: a plain merge-tree under this config runs the filter.
	cmd := exec.Command("git", "-C", r.Root(), "-c", "core.hooksPath=/dev/null",
		"merge-tree", "--write-tree", "--name-only", "--no-messages", "-z", "refs/heads/main", head)
	cmd.Env = gitutil.ScrubbedEnv()
	var ee *exec.ExitError
	if err := cmd.Run(); err != nil && !errors.As(err, &ee) {
		t.Fatal(err)
	}
	if got := ran(); !slices.Contains(got, "clean") || !slices.Contains(got, "smudge") {
		t.Fatalf("fixture: a plain merge-tree ran %v of the filter, not both commands, so this test proves nothing", got)
	}
	for _, n := range ran() {
		if err := os.Remove(filepath.Join(marks, n)); err != nil {
			t.Fatal(err)
		}
	}

	f := localConflict(r.Root(), "refs/heads/main", head)
	if got := ran(); len(got) != 0 {
		t.Fatalf("the conflict probe ran the repository's filter (%v)", got)
	}
	if f.err != nil {
		t.Fatalf("the conflict probe failed: %v", f.err)
	}
	if !slices.Equal(f.paths, []string{"f.txt"}) {
		t.Fatalf("the conflict probe names f.txt, got %q", f.paths)
	}
}
