package landing

import (
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// standStoredParents gives commit the parents named as git presents it,
// through a refs/replace entry or, with graftFile, the .git/info/grafts
// file; the stored commit is not changed.
func standStoredParents(t *testing.T, r *gittest.Repo, graftFile bool, commit string, parents ...string) {
	t.Helper()
	if graftFile {
		writeStoredGraft(t, r, strings.Join(append([]string{commit}, parents...), " "))
		return
	}
	r.Git(append([]string{"replace", "--graft", commit}, parents...)...)
}

// presentedOn reports whether git, as it presents history, walks from head to
// commit.
func presentedOn(r *gittest.Repo, commit, head string) bool {
	return slices.Contains(strings.Fields(r.Git("rev-list", head)), commit)
}

// storedBase is a repository with one commit, base, and its tree.
func storedBase(t *testing.T) (r *gittest.Repo, base string) {
	t.Helper()
	r = gittest.NewRepo(t)
	r.Write("README.md", "a project\n")
	r.Commit("start")
	return r, r.Git("rev-parse", "HEAD")
}

// TestABranchSharesAQueuedPullRequestsCommitsAsStored: a commit the branch and
// a queued pull request both carry, which the default branch does not store,
// is shared, whatever parents a replace ref or a graft gives the default
// branch's tip.
func TestABranchSharesAQueuedPullRequestsCommitsAsStored(t *testing.T) {
	for name, graftFile := range map[string]bool{"a replace ref": false, "a graft file": true} {
		t.Run(name, func(t *testing.T) {
			r, base := storedBase(t)
			shared := r.Git("commit-tree", base+"^{tree}", "-p", base, "-m", "shared")
			head := r.Git("commit-tree", base+"^{tree}", "-p", shared, "-m", "own")
			tracking := r.Git("commit-tree", base+"^{tree}", "-p", base, "-m", "main moves on")
			standStoredParents(t, r, graftFile, tracking, base, shared)
			if !presentedOn(r, shared, tracking) {
				t.Fatal("git presents the shared commit on the default branch")
			}
			got, err := sharesCommits(Options{}, r.Root(), tracking, armedNode{Number: 7, HeadRefOid: shared}, head)
			if err != nil || !got {
				t.Fatalf("sharesCommits = %v, %v; want the stored shared commit found", got, err)
			}
		})
	}
}

// TestAPushMovesTheForgesHeadOnlyWhenTheLocalHeadStoresIt: the push is let
// through only when the local head stores the forge's head; a replace ref or a
// graft giving the local head that commit as a parent is refused.
func TestAPushMovesTheForgesHeadOnlyWhenTheLocalHeadStoresIt(t *testing.T) {
	for name, graftFile := range map[string]bool{"a replace ref": false, "a graft file": true} {
		t.Run(name, func(t *testing.T) {
			r, base := storedBase(t)
			forge := r.Git("commit-tree", base+"^{tree}", "-p", base, "-m", "on the forge")
			local := r.Git("commit-tree", base+"^{tree}", "-p", base, "-m", "local")
			standStoredParents(t, r, graftFile, local, base, forge)
			if !presentedOn(r, forge, local) {
				t.Fatal("git presents the forge's head on the local head")
			}
			res := &Result{Branch: "topic", Head: local}
			if err := movedByPush(res, r.Root(), "7", forge, ""); err == nil {
				t.Fatal("a push the stored history does not fast-forward was let through")
			}
			if err := movedByPush(res, r.Root(), "7", base, ""); err != nil {
				t.Fatalf("a push over a stored ancestor was refused: %v", err)
			}
		})
	}
}
