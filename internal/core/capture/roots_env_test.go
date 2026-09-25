package capture

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestDiscoverRepoRootIgnoresInheritedWorkTree is the attack-input test for the
// discoverRepoRoot env scrub: an inherited GIT_WORK_TREE overrides cwd-based
// `rev-parse --show-toplevel` and would redirect repo-root discovery (and thus
// where capture reads/writes the issue ledger) at an attacker-chosen tree.
func TestDiscoverRepoRootIgnoresInheritedWorkTree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	repo := t.TempDir()
	gitInit := exec.Command("git", "-C", repo, "init")
	gitInit.Env = gittest.Env(t)
	if out, err := gitInit.CombinedOutput(); err != nil {
		t.Skipf("git init unavailable: %v (%s)", err, out)
	}
	other := t.TempDir()
	t.Setenv("GIT_WORK_TREE", other)

	got := discoverRepoRoot(repo)
	gotResolved, _ := filepath.EvalSymlinks(got)
	repoResolved, _ := filepath.EvalSymlinks(repo)
	if gotResolved != repoResolved {
		t.Errorf("discoverRepoRoot(%q) = %q under inherited GIT_WORK_TREE; want the real repo root %q (discovery was redirected)", repo, got, repoResolved)
	}
}

// TestDiscoverRepoRootNeverAcceptsAMarkerGitWillNotAnswerFor is
// iss-2609090947359464's detector. Where git cannot name a repository, the
// fallback used to walk upward and accept any directory whose .git entry merely
// existed — an empty marker planted in a shared ancestor, or a real repository
// another uid laid there, bounded the ledger root with neither the shape check
// nor the ownership gate the rules-root resolver grew. The walk is gone:
// discovery is git's answer or no answer, so neither plant resolves a root.
func TestDiscoverRepoRootNeverAcceptsAMarkerGitWillNotAnswerFor(t *testing.T) {
	shared := t.TempDir()
	if err := os.Mkdir(filepath.Join(shared, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	below := filepath.Join(shared, "a", "b")
	if err := os.MkdirAll(below, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := discoverRepoRoot(below); got != "" {
		t.Fatalf("discoverRepoRoot accepted the empty marker at %q as a repository root", got)
	}
	if _, _, err := resolveRoots("", filepath.Join(below, "issues")); err == nil {
		t.Fatal("resolveRoots resolved a ledger root from an empty marker git will not answer for")
	}
}
