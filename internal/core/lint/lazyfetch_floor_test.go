package lint

import (
	"errors"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The per-file walk's ignore pruning reports the lazy-fetch floor instead of
// silently not pruning (coordinator ruling on iss-2610091935324732). Below git
// 2.44 a partial clone refuses the `ls-files --ignored` that names what to
// prune; IgnoredUnder used to fold that into "nothing ignored", so the lint
// read and reported the ignored tree as if it were documentation. Now Lint,
// DocumentsInRoots and PrunedInRoots refuse with the floor named, and at the
// floor they prune as before.
func TestLintReportsTheLazyFetchFloorInsteadOfNotPruning(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write(".gitignore", "docs/cache/\n")
	repo.Write("docs/README.md", "# Docs\n")
	repo.Commit("docs")
	repo.Write("docs/cache/clone/x.md", "[far](../../elsewhere/y.md)\n")
	repo.Git("config", "remote.origin.promisor", "true")
	root := repo.Root()
	cfg := Config{
		Roots: []string{"docs"},
		Rules: map[string]RuleConfig{"links_resolve": {Enabled: true, Severity: "blocker"}},
	}

	restore := gitutil.SwapGitVersionForTest(func() (string, error) { return "git version 2.39.5 (Apple Git-154)", nil })
	defer restore()
	if _, err := Lint(cfg, root); !errors.Is(err, gitutil.ErrLazyFetchFloor) {
		t.Errorf("Lint: want ErrLazyFetchFloor, got %v", err)
	}
	if _, err := DocumentsInRoots(cfg, root); !errors.Is(err, gitutil.ErrLazyFetchFloor) {
		t.Errorf("DocumentsInRoots: want ErrLazyFetchFloor, got %v", err)
	}
	if _, err := PrunedInRoots(cfg, root); !errors.Is(err, gitutil.ErrLazyFetchFloor) {
		t.Errorf("PrunedInRoots: want ErrLazyFetchFloor, got %v", err)
	}

	restore2 := gitutil.SwapGitVersionForTest(func() (string, error) { return "git version 2.44.0", nil })
	defer restore2()
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatalf("Lint at the floor: %v", err)
	}
	if len(fs) != 0 {
		t.Errorf("Lint at the floor did not prune the ignored cache: %+v", fs)
	}
}
