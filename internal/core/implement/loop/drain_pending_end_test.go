package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// A pending hand-back ends only when it is published, or when it is no longer
// due: the record changed, origin holds no record of it or several, or the
// tombstone would pass the cap. The move says which, and a default branch it
// could not read keeps the hand-back pending.

// editOnOrigin changes the body of id's open record on origin's main, as a
// person settling the hand-back would.
func editOnOrigin(t *testing.T, bare, id string) {
	t.Helper()
	pushFromOtherCopy(t, bare, "settle "+id, func(dir string) {
		matches, _ := filepath.Glob(filepath.Join(dir, ".abcd", "work", "issues", "open", id+"-*.md"))
		if len(matches) != 1 {
			t.Fatalf("origin holds %d open records of %s", len(matches), id)
		}
		f, err := os.OpenFile(matches[0], os.O_APPEND|os.O_WRONLY, 0)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if _, err := f.WriteString("\nA person settled the trust-rule question: keep the boundary as it is.\n"); err != nil {
			t.Fatal(err)
		}
	})
}

// TestAPendingHandBackOverAChangedRecordIsDroppedNotPublished: a yes whose
// hold was renewed by hand keeps its hand-back pending; a person then changes
// the record on origin and releases the issue. The next move publishes
// nothing, ends the pending hand-back, and says in its next step and in the
// route that the hand-back was dropped, and why, never that it was published.
func TestAPendingHandBackOverAChangedRecordIsDroppedNotPublished(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil || first.Judging == nil || first.Judging.Issue != secondIssue {
		t.Fatalf("the first move asks over %s: %+v %v", secondIssue, first.Judging, err)
	}
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Footprint: []string{"commands"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	next := judge(t, repo, Options{}, first, JudgementYes, HandBackTrustRule, "the remedy moves a trust boundary")
	if _, ok := pendingFor(next.PendingHandBacks, secondIssue); !ok {
		t.Fatalf("the yes keeps no pending hand-back: %+v", next.PendingHandBacks)
	}

	editOnOrigin(t, bare, secondIssue)
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Release: true}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	tomb := refOf(t, bare, secondIssue)
	poll, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if refOf(t, bare, secondIssue) != tomb {
		if kind, _ := tombstoneTrailers(t, bare, secondIssue); kind != "" {
			t.Fatalf("the poll published a %s hand-back over a changed record", kind)
		}
	}
	if _, ok := pendingFor(poll.PendingHandBacks, secondIssue); ok {
		t.Fatalf("a hand-back over a changed record is still pending: %+v", poll.PendingHandBacks)
	}
	if strings.Contains(poll.Next, "published the hand-back") || !strings.Contains(poll.Next, "dropped the hand-back of "+secondIssue) ||
		!strings.Contains(poll.Next, "has changed since the hand-back was judged") {
		t.Fatalf("the poll's next step = %q; want the hand-back said dropped, and why", poll.Next)
	}
	r := routeOf(t, poll.HandBacks, secondIssue)
	if strings.Contains(r.Wrote, "published the hand-back") || !strings.Contains(r.Wrote, "dropped the hand-back of "+secondIssue) {
		t.Fatalf("the route wrote %q; want the hand-back said dropped", r.Wrote)
	}
}

// TestAPendingHandBackOverAnUnreadDefaultBranchStaysPending: a pending
// hand-back over a plain release, in a checkout that cannot read origin's
// default branch, is not published and not dropped: it is still due, and is
// published once the branch reads again.
func TestAPendingHandBackOverAnUnreadDefaultBranchStaysPending(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	blob := originBlob(t, bare, secondIssue)
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Footprint: []string{"commands"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Release: true}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	tomb := refOf(t, bare, secondIssue)
	now := time.Now()
	view, err := claim.View(repo.Root(), now)
	if err != nil {
		t.Fatal(err)
	}
	pending := DrainPendingHandBack{Issue: secondIssue, Kind: HandBackTrustRule, Blob: blob, Why: "the hold moved on", At: now}
	st := DrainState{PendingHandBacks: []DrainPendingHandBack{pending},
		HandBacks: []DrainRoute{{Issue: secondIssue, From: DrainFromJudgement, Wrote: "nothing: a flag in this summary"}}}

	repo.Git("symbolic-ref", "--delete", "refs/remotes/origin/HEAD")
	repo.Git("update-ref", "-d", "refs/remotes/origin/main")
	if out := publishPending(repo.Root(), &st, &view, now); len(out) != 0 {
		t.Fatalf("the move ended the pending hand-back over an unread default branch: %q", out)
	}
	if _, ok := pendingFor(st.PendingHandBacks, secondIssue); !ok {
		t.Fatalf("the hand-back is no longer pending: %+v", st.PendingHandBacks)
	}
	if refOf(t, bare, secondIssue) != tomb {
		t.Fatal("the move pushed over the release while the default branch could not be read")
	}

	repo.Git("fetch", "-q", "origin", "+refs/heads/main:refs/remotes/origin/main")
	repo.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	out := publishPending(repo.Root(), &st, &view, now)
	if len(out) != 1 || !strings.Contains(out[0], "published the hand-back of "+secondIssue) {
		t.Fatalf("once the branch reads, the move says %q; want the hand-back published", out)
	}
	if kind, got := tombstoneTrailers(t, bare, secondIssue); kind != HandBackTrustRule || got != blob {
		t.Fatalf("origin's tip marks %q over %q; want the trust-rule hand-back over %s", kind, got, blob)
	}
}
