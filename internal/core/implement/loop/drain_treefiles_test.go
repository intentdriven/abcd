package loop

import (
	"strings"
	"testing"
)

// How a drain tells an issue naming a file from one naming a directory on
// origin (spc-2610161044461448 step 2): each path is asked of the commit on
// its own, so a directory asked beside a path beneath it is still a
// directory, and a path the commit does not hold is not a file.

// TestTreeFilesTellsADirectoryAskedBesideAPathBeneathIt: internal/x asked
// with internal/x/a.go in one call is a directory, not a file, as is docs/
// with its trailing slash; a path the commit does not hold, and a
// trailing-slash directory it does not hold, are not files either; only
// internal/x/a.go is.
func TestTreeFilesTellsADirectoryAskedBesideAPathBeneathIt(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	repo.Write("internal/x/a.go", "package x\n")
	repo.Write("docs/page.md", "# page\n")
	repo.Commit("a tree")
	v := originView{ref: "refs/heads/main", commit: gitIn(t, repo.Root(), "rev-parse", "HEAD")}
	got, err := treeFiles(repo.Root(), v)([]string{"internal/x", "internal/x/a.go", "gone/away.go", "docs/", "internal/y/"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"internal/x/a.go": true}
	for _, p := range []string{"internal/x", "internal/x/a.go", "gone/away.go", "docs/", "internal/y/"} {
		if got[p] != want[p] {
			t.Errorf("%s: file %v; want %v", p, got[p], want[p])
		}
	}
}

// TestTreeFilesTellsAGitlinkIsNoFile: a found_at naming a submodule's
// gitlink, whose commit this repository lacks, is not a file, and asking it
// never errors the plan.
func TestTreeFilesTellsAGitlinkIsNoFile(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	repo.Git("update-index", "--add", "--cacheinfo", "160000,"+strings.Repeat("0123456789", 4)+",vendor/sub")
	repo.Git("-c", "commit.gpgsign=false", "commit", "-q", "-m", "a gitlink")
	v := originView{ref: "refs/heads/main", commit: gitIn(t, repo.Root(), "rev-parse", "HEAD")}
	got, err := treeFiles(repo.Root(), v)([]string{"vendor/sub", "vendor/sub/a.go"})
	if err != nil {
		t.Fatalf("a gitlink errored the plan: %v", err)
	}
	if got["vendor/sub"] || got["vendor/sub/a.go"] {
		t.Fatalf("a gitlink, or a path beneath it, counted as a file: %v", got)
	}
}
