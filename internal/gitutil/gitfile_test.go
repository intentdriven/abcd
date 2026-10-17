package gitutil_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// GitfileTarget names where a git file points from the file itself, whether
// or not anything is there: an absolute target as written, a relative one
// joined to the worktree. A .git that is a directory, a symlink, or a file
// whose first line is no "gitdir: " line names nothing.
func TestGitfileTargetReadsWhereTheGitFilePoints(t *testing.T) {
	write := func(t *testing.T, body string) string {
		t.Helper()
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, ".git"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	nowhere := filepath.Join(t.TempDir(), "nowhere")
	if got, ok := gitutil.GitfileTarget(write(t, "gitdir: "+nowhere+"\n")); !ok || got != nowhere {
		t.Errorf("an absolute target: got %q, %v; want %q", got, ok, nowhere)
	}
	rel := write(t, "gitdir: ../main/.git/worktrees/wt\r\nmore\n")
	if got, ok := gitutil.GitfileTarget(rel); !ok || got != filepath.Join(filepath.Dir(rel), "main", ".git", "worktrees", "wt") {
		t.Errorf("a relative target is joined to the worktree: got %q, %v", got, ok)
	}
	for name, dir := range map[string]string{
		"no gitdir line":         write(t, "ref: refs/heads/main\n"),
		"an empty path":          write(t, "gitdir: \n"),
		"an oversize first line": write(t, "gitdir: /"+strings.Repeat("x", 8<<10)+"\n"),
	} {
		if got, ok := gitutil.GitfileTarget(dir); ok {
			t.Errorf("%s names %q", name, got)
		}
	}
	asDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(asDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, ok := gitutil.GitfileTarget(asDir); ok {
		t.Errorf("a .git directory names %q", got)
	}
	linked := t.TempDir()
	if err := os.Symlink(filepath.Join(write(t, "gitdir: "+nowhere+"\n"), ".git"), filepath.Join(linked, ".git")); err != nil {
		t.Fatal(err)
	}
	if got, ok := gitutil.GitfileTarget(linked); ok {
		t.Errorf("a symlinked .git names %q", got)
	}
}
