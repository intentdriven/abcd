package site

import (
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The site build asks git two questions before it touches its output
// directory: the checkout's root (the symlink rule's boundary) and the
// repository's root commit (the identity its marker carries). On a heavily
// loaded machine a git that answered can still miss the runner's WaitDelay,
// and both lookups once read that miss as an answer: "git cannot name its
// root", and an empty root commit that refused to remove the build's own
// output (iss-2610100846469473). gittest.SlowPipeGit reproduces the miss
// without load: git answers, and a process it leaves behind holds the pipe.
// Neither lookup has a deadline, so the build waits for git's answer.

// TestBuildSurvivesAGitHoldingItsPipeOnTheRootLookup: a root lookup whose
// output is held open on every call is waited for, and the build succeeds
// rather than refusing that git cannot name the checkout's root.
func TestBuildSurvivesAGitHoldingItsPipeOnTheRootLookup(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	gittest.SlowPipeGit(t, "--show-toplevel", -1)

	if _, err := Build(Request{RepoRoot: f.Root(), OutDir: out, Stamp: fixtureStamp}); err != nil {
		t.Fatalf("a slow root lookup failed the build: %v", err)
	}
}

// TestRebuildSurvivesAGitHoldingItsPipeOnTheRootCommit: rebuilding in place
// with the root-commit lookup's output held open on every call, the build
// still recognises its own output instead of refusing to remove it as a
// repository with no root commit.
func TestRebuildSurvivesAGitHoldingItsPipeOnTheRootCommit(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	buildFixture(t, f, out)
	gittest.SlowPipeGit(t, "--max-parents=0", -1)

	if _, err := Build(Request{RepoRoot: f.Root(), OutDir: out, Stamp: fixtureStamp}); err != nil {
		t.Fatalf("a slow root-commit lookup failed the rebuild: %v", err)
	}
}
