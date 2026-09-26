package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestCaptureRootBoundsARepositoryGitWillNotAnswerFor is iss-2609020224230967:
// captureRoot fell back to the working directory on any git failure, so a
// repository git will not answer for (git absent from PATH here; an ownership
// refusal or a corrupt .git alike) was treated as no repository at all, and a
// subdirectory became the root the scanner reads its per-repo override from.
// It resolves three-state instead: git's toplevel, else the checkout the .git
// marker names (through the rules root's plausibility and ownership gates),
// else the working directory.
func TestCaptureRootBoundsARepositoryGitWillNotAnswerFor(t *testing.T) {
	repo := gittest.NewRepo(t).Root()
	sub := filepath.Join(repo, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.EvalSymlinks(repo)

	if got, _ := filepath.EvalSymlinks(captureRoot(sub)); got != want {
		t.Fatalf("control: captureRoot(sub) = %q, want the toplevel %q", got, want)
	}

	t.Setenv("PATH", t.TempDir()) // git is absent: it answers for nothing
	if got, _ := filepath.EvalSymlinks(captureRoot(sub)); got != want {
		t.Fatalf("captureRoot(sub) with git unable to answer = %q, want the checkout %q", got, want)
	}

	plain := t.TempDir()
	if got := captureRoot(plain); got != plain {
		t.Fatalf("outside any repository captureRoot = %q, want the working directory", got)
	}
}
