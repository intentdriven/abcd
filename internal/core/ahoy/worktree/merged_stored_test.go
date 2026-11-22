package worktree_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestMergedJudgementReadsAncestryAsStored: a graft or a refs/replace entry
// that gives the default branch's tip an unmerged branch's tip as a parent
// makes that branch an ancestor of the default branch for a read that
// honours them. The merged judgement reads the commits as stored, so the
// branch stays unmerged.
func TestMergedJudgementReadsAncestryAsStored(t *testing.T) {
	setup := func(t *testing.T) (r *gittest.Repo, start, feat, main string) {
		t.Helper()
		r = gittest.NewRepo(t)
		r.Write("README.md", "a project\n")
		r.Commit("start")
		start = r.Git("rev-parse", "HEAD")
		r.Git("switch", "-q", "-c", "feat")
		r.Write("f.txt", "the feature\n")
		r.Commit("the feature")
		feat = r.Git("rev-parse", "HEAD")
		r.Git("switch", "-q", "main")
		r.Write("m.txt", "the main line\n")
		r.Commit("the main line")
		return r, start, feat, r.Git("rev-parse", "HEAD")
	}
	judge := func(t *testing.T, r *gittest.Repo, feat string) {
		t.Helper()
		r.Git("-c", "advice.graftFileDeprecated=false", "merge-base", "--is-ancestor", feat, "main")
		if merged, by := worktree.Merged(r.Root(), "refs/heads/main", "refs/heads/feat", feat, nil); merged {
			t.Fatalf("Merged = true by %q; feat's commit is on no stored line of main", by)
		}
	}
	t.Run("a graft", func(t *testing.T) {
		r, start, feat, main := setup(t)
		info := filepath.Join(r.Root(), ".git", "info")
		if err := os.MkdirAll(info, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(info, "grafts"), []byte(main+" "+start+" "+feat+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		judge(t, r, feat)
	})
	t.Run("a refs/replace entry", func(t *testing.T) {
		r, start, feat, main := setup(t)
		r.Git("replace", main, r.Git("commit-tree", main+"^{tree}", "-p", start, "-p", feat, "-m", "the main line"))
		judge(t, r, feat)
	})
}
