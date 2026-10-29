package claim

import (
	"slices"
	"testing"
	"time"
)

// A holder id names no account the forge's record of pushes could be matched
// against, so a release's reading holds one of the reader's own ids to the
// reader's login (namesPusher): a commit another account pushed naming the
// reader's id is not the reader's push, as a commit naming their login is not
// (iss-2610290809526033).

// forgedOverNaming is forgedOver with the commits naming holder, as holder and
// as the withdrawer of #11.
func (w *world) forgedOverNaming(id, holder string) (record, tomb string) {
	w.t.Helper()
	wd := []Withdrawal{{Number: 11, ID: "PR_eleven", Form: FormAutoSquash, Head: headA, By: holder, State: WithdrawnPending}}
	rec := message{Record: id, Holder: holder, Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: t0, ExpiresAt: t0.Add(time.Hour), Withdrawn: wd}
	record = w.pushChild(id, w.remoteTip(id), rec.text(), t0)
	ts := rec
	ts.ReleasedAt, ts.ExpiresAt = t0, t0
	tomb = w.pushChild(id, record, ts.text(), t0)
	return record, tomb
}

// TestACommitAnotherLoginPushedNamingTheReadersHolderIDIsNeverReArmed: commits
// naming octo's holder id that the record says mal pushed lie above octo's
// newest push, so octo's next act reads nothing from them, re-arms nothing and
// counts them; the same commits pushed by octo are octo's record, and are
// re-armed.
func TestACommitAnotherLoginPushedNamingTheReadersHolderIDIsNeverReArmed(t *testing.T) {
	w, f, octo := releasedWithNothingArmed(t)
	record, tomb := w.forgedOverNaming("rel-v0.16.0", octo.keptHolderID())
	f.pusherOf[record], f.pusherOf[tomb] = "mal", "mal"
	res, err := octo.reserve("iss-8", "o.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(f.mutations()) != 0 || f.pr(11).method != "" {
		t.Fatalf("mutations %v, #11 %+v; want nothing re-armed for a record mal pushed naming octo's id", f.mutations(), f.pr(11))
	}
	if len(res.Settles) != 1 || !slices.Equal(res.Settles[0].NotRead, []PushCount{{Pusher: "mal", Commits: 2}}) || res.Settles[0].Commit != "" {
		t.Fatalf("settles = %+v; want mal's two commits counted and nothing pushed", res.Settles)
	}
	delete(f.pusherOf, record)
	delete(f.pusherOf, tomb)
	forgetVerdicts(t, octo.dir)
	if _, err := octo.reserve("iss-9", "p.go"); err != nil {
		t.Fatal(err)
	}
	if got := f.mutations(); !slices.Equal(got, []string{"enablePullRequestAutoMerge #11"}) {
		t.Fatalf("mutations = %v; want #11 re-armed once octo's own push recorded it", got)
	}
}
