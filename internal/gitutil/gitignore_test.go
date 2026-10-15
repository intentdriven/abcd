package gitutil_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// runGit runs an isolated git command in repo and returns its combined output.
func runGit(t *testing.T, repo string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	cmd.Env = gittest.Env(t)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// newRepo makes an isolated git repo with the given .gitignore body. Global and
// system config are neutralised so a developer's ~/.gitignore cannot change the
// result.
func newRepo(t *testing.T, gitignore string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	cmd.Env = gittest.Env(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	if gitignore != "" {
		if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(gitignore), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestIsIgnored(t *testing.T) {
	repo := newRepo(t, ".abcd/.work.local/\n*.log\n")

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"ignored directory", ".abcd/.work.local/", true},
		{"ignored by glob", "debug.log", true},
		{"tracked path", ".abcd/work/decisions/README.md", false},
		{"tracked source", "internal/core/repolint.go", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gitutil.IsIgnored(repo, tt.path); got != tt.want {
				t.Errorf("IsIgnored(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// A negation pattern un-ignores a path: `!keep.log` must NOT read as ignored,
// even though `*.log` matches it. git reports the match with a leading `!`.
func TestIsIgnoredNegationIsNotIgnored(t *testing.T) {
	repo := newRepo(t, "*.log\n!keep.log\n")

	if gitutil.IsIgnored(repo, "keep.log") {
		t.Error("IsIgnored(keep.log) = true; a negation pattern must not count as ignored")
	}
	if !gitutil.IsIgnored(repo, "other.log") {
		t.Error("IsIgnored(other.log) = false, want true")
	}
}

// A tracked (committed) file is never ignored, even when it was force-added
// against a matching .gitignore pattern: git consults the index for its real
// ignore decision, and CheckIgnored must not invert that answer (which would
// falsely flag a committed file as non-durable / drop it from a release bundle).
func TestCheckIgnoredTrackedFileNotIgnored(t *testing.T) {
	repo := newRepo(t, ".abcd/\n")

	dir := filepath.Join(repo, ".abcd", "work", "decisions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Force-add the path even though .abcd/ is ignored, then commit it.
	if out, err := runGit(t, repo, "add", "-f", ".abcd/work/decisions/README.md"); err != nil {
		t.Fatalf("git add -f: %v: %s", err, out)
	}
	commitAll(t, repo)

	if gitutil.IsIgnored(repo, ".abcd/work/decisions/README.md") {
		t.Error("IsIgnored(tracked force-added file) = true; git never ignores a tracked file")
	}
	// A genuinely untracked path under the same pattern is still ignored.
	if !gitutil.IsIgnored(repo, ".abcd/scratch.txt") {
		t.Error("IsIgnored(untracked ignored path) = false, want true")
	}
}

func TestCheckIgnoredBatch(t *testing.T) {
	repo := newRepo(t, "ignored/\n")

	got := gitutil.CheckIgnored(repo, []string{"ignored/a.txt", "kept/b.txt"})
	if _, ok := got["ignored/a.txt"]; !ok {
		t.Error("CheckIgnored: ignored/a.txt missing from the ignored set")
	}
	if _, ok := got["kept/b.txt"]; ok {
		t.Error("CheckIgnored: kept/b.txt must not be in the ignored set")
	}
}

// Fails open: outside a git repo nothing is reported ignored, and no error is
// raised. An audit rule must not claim "not gitignored" is a violation just
// because git is absent.
func TestCheckIgnoredOutsideRepoFailsOpen(t *testing.T) {
	dir := t.TempDir() // not a git repo

	got := gitutil.CheckIgnored(dir, []string{"anything.txt"})
	if len(got) != 0 {
		t.Errorf("CheckIgnored outside a repo = %v, want empty", got)
	}
}

func TestCheckIgnoredEmptyCandidates(t *testing.T) {
	repo := newRepo(t, "*.log\n")

	got := gitutil.CheckIgnored(repo, nil)
	if len(got) != 0 {
		t.Errorf("CheckIgnored(nil) = %v, want empty", got)
	}
}
