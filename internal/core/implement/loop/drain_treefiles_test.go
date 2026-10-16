package loop

import (
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
