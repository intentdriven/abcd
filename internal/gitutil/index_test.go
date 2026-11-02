package gitutil_test

import (
	"os"
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

// A scratch index is written whole, never split, so a repository set to
// core.splitIndex gains no shared index file in its git directory from one; and
// StreamWithIndex hands git's answer to the caller's writer as git writes it.
func TestAScratchIndexWritesNoSharedIndexIntoTheRepository(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write("a.txt", "a\n")
	repo.Commit("init")
	repo.Git("config", "core.splitIndex", "true")
	src, err := os.ReadFile(filepath.Join(repo.Root(), ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	index := filepath.Join(t.TempDir(), "index")
	if err := os.WriteFile(index, src, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := gitutil.RunWithIndex(repo.Root(), index, 4096, "write-tree"); err != nil {
		t.Fatal(err)
	}
	var listing strings.Builder
	if err := gitutil.StreamWithIndex(repo.Root(), index, &listing, "ls-files", "--stage"); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(listing.String(), "\ta.txt\n") {
		t.Fatalf("the listing reaches the writer: %q", listing.String())
	}
	shared, err := filepath.Glob(filepath.Join(repo.Root(), ".git", "sharedindex.*"))
	if err != nil || len(shared) != 0 {
		t.Fatalf("no shared index is written into the git directory: %v %v", shared, err)
	}
}
