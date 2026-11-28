package loop

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The conflict probe merges the default branch and the lane's head from the
// merge base their stored parents give, as the forge does: a refs/replace
// entry or a .git/info/grafts line that gives the head other parents moves
// the merge base git presents, and changes neither the paths the probe names
// nor whether it names any.

// conflictFixture is a repository whose base commit holds f.txt, with
// tracking (refs/heads/main) and head each changing the same line of it.
// With headOnTracking the head's stored parent is the tracking tip, so the
// two merge cleanly; otherwise both stand on the base, so they conflict.
func conflictFixture(t *testing.T, headOnTracking bool) (repo *gittest.Repo, base, tracking, head string) {
	t.Helper()
	repo = gittest.NewRepo(t)
	repo.Write("f.txt", "a\nBASE\nc\n")
	repo.Commit("base")
	base = repo.Git("rev-parse", "HEAD")
	repo.Write("f.txt", "a\nTRACKING\nc\n")
	repo.Commit("tracking")
	tracking = repo.Git("rev-parse", "HEAD")
	parent := base
	if headOnTracking {
		parent = tracking
	}
	repo.Git("checkout", "-q", "-b", "lane", parent)
	repo.Write("f.txt", "a\nHEAD\nc\n")
	repo.Commit("head")
	head = repo.Git("rev-parse", "HEAD")
	repo.Git("checkout", "-q", "main")
	return repo, base, tracking, head
}

// TestTheConflictProbeMergesFromTheStoredMergeBase: a head that conflicts
// with the default branch as stored is named as conflicting when a replace
// ref or a graft parents it on the default branch's tip, and a head that
// merges cleanly as stored is named as clean when one parents it on the base
// instead.
func TestTheConflictProbeMergesFromTheStoredMergeBase(t *testing.T) {
	for name, graftFile := range map[string]bool{"a replace ref": false, "a graft file": true} {
		t.Run(name, func(t *testing.T) {
			t.Run("a stored conflict", func(t *testing.T) {
				repo, base, tracking, head := conflictFixture(t, false)
				standParents(t, repo, graftFile, head, base, tracking)
				if got := repo.Git("-c", "advice.graftFileDeprecated=false", "merge-base", tracking, head); got != tracking {
					t.Fatalf("fixture: git presents merge base %s, not the tracking tip", got)
				}
				f := localConflict(repo.Root(), "refs/heads/main", head)
				if f.err != nil || !slices.Equal(f.paths, []string{"f.txt"}) {
					t.Fatalf("the probe names f.txt as conflicting, got %q (%v)", f.paths, f.err)
				}
			})
			t.Run("a stored clean merge", func(t *testing.T) {
				repo, base, tracking, head := conflictFixture(t, true)
				standParents(t, repo, graftFile, head, base)
				if got := repo.Git("-c", "advice.graftFileDeprecated=false", "merge-base", tracking, head); got != base {
					t.Fatalf("fixture: git presents merge base %s, not the base", got)
				}
				f := localConflict(repo.Root(), "refs/heads/main", head)
				if f.err != nil || len(f.paths) != 0 {
					t.Fatalf("the probe names no conflict, got %q (%v)", f.paths, f.err)
				}
			})
		})
	}
}

// TestAnArmedConflictQuotesTheArmingByTheMethodTheGateReadsAtTheTip: the
// refusal of an armed pull request that conflicts quotes its re-arming by the
// method the loop arms by, the merge gate's at the tip just fetched, never the
// lane's base's.
func TestAnArmedConflictQuotesTheArmingByTheMethodTheGateReadsAtTheTip(t *testing.T) {
	f, _ := rearmFixture(t)
	f.mirrorAtTip(t, "SQUASH")
	f.repo.Git("fetch", "-q", "origin")
	lane := currentLane(t, f.repo, f.runID)
	c := Context{RepoRoot: f.repo.Root(), Now: time.Now()}
	r := conflictRefusal(c, lane, "main", "refs/remotes/origin/main", mergeability{Mergeable: MergeableConflicting, MergeStateStatus: "DIRTY"})
	if !strings.Contains(r.Remedy, "`gh pr merge 7 --auto --squash`") || strings.Contains(r.Remedy, "--auto --merge") {
		t.Fatalf("the re-arming is quoted by the tip's method: %s", r.Remedy)
	}
}
