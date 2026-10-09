package gitutil_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// RunWithIndex points git at an index file of the caller's, so a whole-tree
// staging (`add -A`) and the diff of it leave the repository's own index —
// what the person staged — exactly as it was. An ambient GIT_INDEX_FILE is
// overridden by the caller's, never inherited.
func TestRunWithIndexLeavesTheRealIndexAlone(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write("a.txt", "a\n")
	repo.Commit("init")
	repo.Write("staged.txt", "s\n")
	repo.Git("add", "--", "staged.txt")
	repo.Write("a.txt", "a\nchanged\n")
	repo.Write("new.txt", "n\n")
	t.Setenv("GIT_INDEX_FILE", filepath.Join(t.TempDir(), "ambient"))

	index := filepath.Join(t.TempDir(), "index")
	for _, args := range [][]string{{"read-tree", "HEAD"}, {"add", "-A"}} {
		if _, err := gitutil.RunWithIndex(repo.Root(), index, 1<<20, args...); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	names, err := gitutil.RunWithIndex(repo.Root(), index, 1<<20, "diff", "--cached", "--name-only", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Fields(string(names)); strings.Join(got, ",") != "a.txt,new.txt,staged.txt" {
		t.Fatalf("the caller's index holds every change: %v", got)
	}
	if status := repo.Git("status", "--porcelain"); !strings.Contains(status, "A  staged.txt") || !strings.Contains(status, "M a.txt") && !strings.Contains(status, "M  a.txt") || !strings.Contains(status, "?? new.txt") {
		t.Fatalf("the repository's own index is untouched:\n%s", status)
	}
	if _, err := gitutil.RunWithIndex(repo.Root(), index, 4, "diff", "--cached", "HEAD"); err == nil {
		t.Fatal("output past the cap is refused, never truncated")
	}
}
