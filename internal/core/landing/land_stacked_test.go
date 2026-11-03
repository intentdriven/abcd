package landing

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// A branch cut from another pull request's branch carries that one's commits,
// and the merge queue removes it when that one is queued or armed: on
// 2026-11-03 land armed #1376, whose branch carried #1375's two commits while
// #1375 was queued, and the queue removed #1376 as invalid_merge_commit 16
// seconds after it entered (iss-2611031719581061). land defers that arming.

// listedPR is one open pull request into main as the forge answers
// claim.ArmedPullRequestsQuery.
func listedPR(n int, head, ref string, queued, armed bool) string {
	auto := "null"
	if armed {
		auto = `{"mergeMethod":"MERGE"}`
	}
	return fmt.Sprintf(`{"number":%d,"id":"PR_kwDO%d","headRefOid":%q,"headRefName":%q,"isCrossRepository":false,"isInMergeQueue":%t,"autoMergeRequest":%s}`,
		n, n, head, ref, queued, auto)
}

func listed(nodes ...string) string { return "[" + strings.Join(nodes, ",") + "]" }

// stack puts a second commit on feat/x, so the branch carries the fixture's
// first one beneath its own, and returns that first commit: the head of the
// pull request feat/x was cut from.
func (f *fixture) stack() string {
	f.t.Helper()
	base := f.head()
	f.repo.Write("stacked.txt", "the stacked change\n")
	f.repo.Commit("the stacked change")
	return base
}

func TestABranchCarryingAQueuedOrArmedPullRequestsCommitsIsNotArmed(t *testing.T) {
	for _, c := range []struct {
		name          string
		queued, armed bool
		says          string
	}{
		{"queued", true, true, "in the merge queue"},
		{"armed", false, true, "armed"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t, queueRuleset)
			base := f.stack()
			f.forge.prs = f.prFor(f.head())
			f.forge.views = []string{viewBlocked}
			f.forge.armed = listed(listedPR(5, base, "feat/base", c.queued, c.armed))
			res, err := f.land(f.options())
			if err != nil {
				t.Fatal(err)
			}
			if f.forge.called("pr", "merge") != 0 {
				t.Fatalf("forge calls %v: want no arming while #5 is on its way to the merge", f.forge.calls)
			}
			s := step(t, res, "enqueue")
			if s.Status != StatusStopped {
				t.Fatalf("enqueue step %+v, want stopped", s)
			}
			for _, want := range []string{"#7", "#5 (feat/base, " + c.says, "nothing was armed", "after #5 merges", "rebase feat/x onto origin/main"} {
				if !strings.Contains(s.Note, want) {
					t.Errorf("enqueue note %q does not say %q", s.Note, want)
				}
			}
			if f.onOrigin("feat/x") != f.head() {
				t.Fatalf("origin holds feat/x at %q, want the push to stand at %s", f.onOrigin("feat/x"), f.head())
			}
		})
	}
}

// A pull request that moved on after the branch was cut from it still has the
// commits the branch carries: its head is not in this checkout, so land
// fetches it from origin to read its commits.
func TestABranchCarryingCommitsOfAPullRequestThatMovedOnIsNotArmed(t *testing.T) {
	f := newFixture(t, queueRuleset)
	base := f.head()
	f.repo.Git("push", "-q", "origin", base+":refs/heads/feat/base")
	other := filepath.Join(t.TempDir(), "other")
	f.repo.Git("clone", "-q", "--branch", "feat/base", f.bare, other)
	if err := os.WriteFile(filepath.Join(other, "more.txt"), []byte("the base moves on\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.repo.Git("-C", other, "add", "-A")
	f.repo.Git("-C", other, "commit", "-q", "-m", "the base moves on")
	moved := f.repo.Git("-C", other, "rev-parse", "HEAD")
	f.repo.Git("-C", other, "push", "-q", "origin", "HEAD:refs/heads/feat/base", "HEAD:refs/pull/5/head")
	if _, err := gitutil.Run(f.repo.Root(), "cat-file", "-e", moved+"^{commit}"); err == nil {
		t.Fatal("the checkout already holds the moved head; the test needs one it lacks")
	}
	f.stack()
	f.forge.prs = f.prFor(f.head())
	f.forge.views = []string{viewBlocked}
	f.forge.armed = listed(listedPR(5, moved, "feat/base", true, false))
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if f.forge.called("pr", "merge") != 0 {
		t.Fatalf("forge calls %v: want no arming", f.forge.calls)
	}
	if s := step(t, res, "enqueue"); s.Status != StatusStopped || !strings.Contains(s.Note, "#5 (feat/base, in the merge queue)") {
		t.Fatalf("enqueue step %+v, want stopped naming #5", s)
	}
	fetched := false
	for _, c := range f.gitCalls {
		if len(c) > 0 && c[0] == "fetch" && strings.Contains(strings.Join(c, " "), "refs/pull/5/head") {
			fetched = true
		}
	}
	if !fetched {
		t.Fatalf("git calls %v: want #5's head fetched from origin", f.gitCalls)
	}
}

// A queued pull request on a branch of its own, one sharing the branch's
// commits that is neither armed nor queued, and the branch's own pull request
// in the listing hold nothing: the branch is armed.
func TestOnlyAQueuedOrArmedPullRequestsCommitsHoldTheArming(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.repo.Git("checkout", "-q", "-b", "feat/other", "main")
	f.repo.Write("other.txt", "another change\n")
	f.repo.Commit("another change")
	other := f.head()
	f.repo.Git("checkout", "-q", "feat/x")
	base := f.stack()
	f.forge.prs = f.prFor(f.head())
	f.forge.views = []string{viewBlocked, viewQueued}
	f.forge.armed = listed(listedPR(4, other, "feat/other", true, true), listedPR(5, base, "feat/base", false, false),
		listedPR(7, f.head(), "feat/x", false, true))
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if f.forge.called("pr", "merge", "7", "--auto", "--merge", "--match-head-commit="+f.head()) != 1 {
		t.Fatalf("forge calls %v, want one arming", f.forge.calls)
	}
	if s := step(t, res, "enqueue"); s.Status != StatusDone || !strings.Contains(s.Note, "in the merge queue") {
		t.Fatalf("enqueue step %+v, want armed and queued", s)
	}
}

// A listing the forge cannot give proves nothing, and nothing is armed.
func TestAnUnreadableListingOfArmedPullRequestsArmsNothing(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.forge.views = []string{viewBlocked}
	f.forge.armedErr = errors.New("HTTP 502")
	_, err := f.land(f.options())
	wantRefusal(t, err, "HTTP 502")
	wantRefusal(t, err, "nothing was armed")
	if f.forge.called("pr", "merge") != 0 {
		t.Fatalf("armed: %v", f.forge.calls)
	}
}

// A queued pull request whose head origin cannot give leaves land unable to
// read its commits, and nothing is armed.
func TestAQueuedHeadOriginCannotGiveArmsNothing(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.forge.views = []string{viewBlocked}
	f.forge.armed = listed(listedPR(5, strings.Repeat("ab", 20), "feat/gone", true, false))
	_, err := f.land(f.options())
	wantRefusal(t, err, "#5")
	wantRefusal(t, err, "nothing was armed")
	if f.forge.called("pr", "merge") != 0 {
		t.Fatalf("armed: %v", f.forge.calls)
	}
}
