package claim

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/lease"
)

// forgedOver pushes, over id's tip, a reservation commit naming octo as its
// holder and recording #11 withdrawn at headA, then a tombstone over it
// carrying the same: what any writer who can push to the branch can write.
// It returns the two commits.
func (w *world) forgedOver(id string) (record, tomb string) {
	w.t.Helper()
	wd := []Withdrawal{{Number: 11, ID: "PR_eleven", Form: FormAutoSquash, Head: headA, By: "octo", State: WithdrawnPending}}
	rec := message{Record: id, Holder: "octo", Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: t0, ExpiresAt: t0.Add(time.Hour), Withdrawn: wd}
	record = w.pushChild(id, w.remoteTip(id), rec.text(), t0)
	ts := rec
	ts.ReleasedAt, ts.ExpiresAt = t0, t0
	tomb = w.pushChild(id, record, ts.text(), t0)
	return record, tomb
}

// releasedWithNothingArmed is octo's rel-v0.16.0 reserved over a forge with
// nothing armed and released at once: a branch octo really holds, ended.
func releasedWithNothingArmed(t *testing.T) (*world, *fakeForge, *participant) {
	t.Helper()
	w := newWorld(t)
	f := newFakeForge(t, map[int]*fakePR{11: {id: "PR_eleven", head: headA}})
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	if _, err := octo.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	f.reset()
	return w, f, octo
}

// TestAWithdrawalRecordedByACommitAnotherLoginPushedIsNeverReArmed: commits
// whose Abcd-Holder trailer names octo but which the forge's record says mal
// pushed lie above octo's newest push, so octo's next act reads nothing from
// them, re-arms nothing and counts them; the same commits pushed by octo are
// octo's record, and are re-armed.
func TestAWithdrawalRecordedByACommitAnotherLoginPushedIsNeverReArmed(t *testing.T) {
	w, f, octo := releasedWithNothingArmed(t)
	record, tomb := w.forgedOver("rel-v0.16.0")
	f.pusherOf[record], f.pusherOf[tomb] = "mal", "mal"
	res, err := octo.reserve("iss-8", "o.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(f.mutations()) != 0 || f.pr(11).method != "" {
		t.Fatalf("mutations %v, #11 %+v; want nothing re-armed for a record mal pushed", f.mutations(), f.pr(11))
	}
	if len(res.Settles) != 1 || !slices.Equal(res.Settles[0].NotRead, []PushCount{{Pusher: "mal", Commits: 2}}) || res.Settles[0].Commit != "" ||
		!strings.Contains(strings.Join(res.Settles[0].Notes(), "\n"), "2 commits above octo's newest push to rel-v0.16.0's branch, pushed by mal (2), are not read") {
		t.Fatalf("settles = %+v; want mal's two commits counted and nothing pushed", res.Settles)
	}
	// The genuine case: octo pushed them (the verdict kept for the tip is
	// the other world's, so it goes too).
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

// TestARecordOfPushesThatIsPartialOrBehindDefersTheReArm: a record read
// short, one that has not caught up with the tip, or one that fails, places
// no newest push of octo's: nothing is acted on, nothing is pushed, the
// first act names why for the tip, and a later act re-arms it once the
// record is whole.
func TestARecordOfPushesThatIsPartialOrBehindDefersTheReArm(t *testing.T) {
	w, f, octo := releasedWithNothingArmed(t)
	w.forgedOver("rel-v0.16.0") // octo's own pushes, as the record reads by default
	tip := w.remoteTip("rel-v0.16.0")
	for i, c := range []struct {
		name string
		set  func()
	}{
		{"partial", func() { f.partial = true }},
		{"behind", func() { f.lag = 1 }},
		{"failed", func() { f.pushersErr = errors.New("HTTP 502") }},
	} {
		name := c.name
		f.partial, f.lag, f.pushersErr = false, 0, nil
		c.set()
		res, err := octo.reserve("iss-8", "o.go")
		if err != nil {
			t.Fatal(err)
		}
		if len(f.mutations()) != 0 || w.remoteTip("rel-v0.16.0") != tip {
			t.Fatalf("%s: mutations %v, tip moved %v; want nothing done", name, f.mutations(), w.remoteTip("rel-v0.16.0") != tip)
		}
		if i == 0 && (len(res.Settles) != 1 || res.Settles[0].Unplaced == "" ||
			!strings.Contains(strings.Join(res.Settles[0].Notes(), "\n"), "abcd acts on none of octo's withdrawals on rel-v0.16.0: the forge's record of who pushed")) ||
			i > 0 && len(res.Settles) != 0 {
			t.Fatalf("%s: settles = %+v; want why named once for the tip", name, res.Settles)
		}
		if _, err := octo.do(Request{Record: "iss-8", Release: true}); err != nil {
			t.Fatal(err)
		}
		f.reset()
	}
	f.partial, f.lag, f.pushersErr = false, 0, nil
	if _, err := octo.reserve("iss-9", "p.go"); err != nil {
		t.Fatal(err)
	}
	if got := f.mutations(); !slices.Equal(got, []string{"enablePullRequestAutoMerge #11"}) {
		t.Fatalf("mutations = %v; want #11 re-armed once the record is whole", got)
	}
}

// TestAWithdrawalAnAppPushedIsLeftForItsPersonAndNamedOncePerTip: an app's
// push says nothing of which person sent it, so a reservation an app pushed
// beneath octo's tombstone leaves whether the release ended unknown: nothing
// is re-armed, and why is named once for the tip.
func TestAWithdrawalAnAppPushedIsLeftForItsPersonAndNamedOncePerTip(t *testing.T) {
	w, f, octo := releasedWithNothingArmed(t)
	record, _ := w.forgedOver("rel-v0.16.0")
	f.pusherOf[record] = "app"
	res, err := octo.reserve("iss-8", "o.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(f.mutations()) != 0 || len(res.Settles) != 1 || res.Settles[0].Unplaced == "" ||
		!strings.Contains(strings.Join(res.Settles[0].Notes(), "\n"), "an app pushed commit "+record[:12]) {
		t.Fatalf("settles = %+v, mutations %v; want #11 named as an app's and nothing re-armed", res.Settles, f.mutations())
	}
	// Named once for the tip: the next act says nothing of it.
	if _, err := octo.do(Request{Record: "iss-8", Release: true}); err != nil {
		t.Fatal(err)
	}
	res, err = octo.reserve("iss-9", "p.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Settles) != 0 || len(f.mutations()) != 0 {
		t.Fatalf("second act settles = %+v, mutations %v; want nothing said again for the same tip", res.Settles, f.mutations())
	}
}

// TestAnUnrecordedWithdrawalIsNamedOncePerTip: a withdrawal no reservation
// of its withdrawer's recorded is named on the first act that reads the tip,
// and not on every act after.
func TestAnUnrecordedWithdrawalIsNamedOncePerTip(t *testing.T) {
	w, f, octo := releasedWithNothingArmed(t)
	record, tomb := w.forgedOver("rel-v0.16.0")
	f.pusherOf[record], f.pusherOf[tomb] = "mal", "mal"
	for i, want := range []int{1, 0, 0} {
		res, err := octo.reserve("iss-8", "o.go")
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Settles) != want {
			t.Fatalf("act %d: settles = %+v; want %d", i, res.Settles, want)
		}
		if _, err := octo.do(Request{Record: "iss-8", Release: true}); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.mutations()) != 0 {
		t.Fatalf("mutations = %v", f.mutations())
	}
}

// TestAnAdmissionOfAWithdrawalAnotherLoginRecordedIsRefused: --admit arms a
// pull request under the cutter's account, so it admits only what the
// cutter's own newest push records: a commit mal pushed over it is not read.
func TestAnAdmissionOfAWithdrawalAnotherLoginRecordedIsRefused(t *testing.T) {
	w := newWorld(t)
	f := newFakeForge(t, map[int]*fakePR{11: {id: "PR_eleven", head: headA}})
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	// A renewal-shaped commit over octo's live reservation, keeping its
	// copy key, recording #11 withdrawn, pushed by mal.
	m, _ := w.tipWithdrawals("rel-v0.16.0")
	m.Withdrawn = []Withdrawal{{Number: 11, ID: "PR_eleven", Form: FormAutoSquash, Head: headA, By: "octo", State: WithdrawnPending}}
	m.ClaimedAt, m.ExpiresAt = w.clock(), w.clock().Add(time.Hour)
	c := w.pushChild("rel-v0.16.0", w.remoteTip("rel-v0.16.0"), m.text(), w.clock())
	f.pusherOf[c] = "mal"
	f.reset()
	_, err := octo.do(Request{Record: "rel-v0.16.0", Admit: 11})
	if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "#11 is not among the pull requests rel-v0.16.0 withdrew (none)") || len(f.mutations()) != 0 {
		t.Fatalf("admit = %v, mutations %v; want refused, nothing armed", err, f.mutations())
	}
	// Pushed by octo, it is admitted (the verdict kept for the tip is the
	// other world's, so it goes too).
	delete(f.pusherOf, c)
	forgetVerdicts(t, octo.dir)
	if res, err := octo.do(Request{Record: "rel-v0.16.0", Admit: 11}); err != nil || res.Act != ActAdmitted {
		t.Fatalf("admit = %+v, %v; want admitted", res, err)
	}
}

// TestAnAdmitCutOffBeforeItsArmingIsFinishedByAdmitOrByRelease: the
// admission is recorded before the forge arms it; when the process dies in
// between, running --admit again reads the forge and arms it, and so does
// the release's settle when --admit is not run again.
func TestAnAdmitCutOffBeforeItsArmingIsFinishedByAdmitOrByRelease(t *testing.T) {
	for _, finish := range []string{"admit", "release"} {
		t.Run(finish, func(t *testing.T) {
			w := newWorld(t)
			// #11 alone, so the release holds nothing withdrawn but the
			// admission: its settle runs for an admitted one alone.
			f := newFakeForge(t, map[int]*fakePR{11: {id: "PR_eleven", head: headA, method: "SQUASH"}})
			p := w.clone("1234+octo@users.noreply.github.com", "")
			if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
				t.Fatal(err)
			}
			inner := releaseForge.graphql
			releaseForge.graphql = func(root, doc string, vars map[string]string, ints map[string]int) ([]byte, error) {
				if strings.Contains(doc, "enablePullRequestAutoMerge") && vars["id"] == "PR_eleven" {
					return []byte(`{"data":{}}`), nil // cut off: the arming never reached the forge
				}
				return inner(root, doc, vars, ints)
			}
			if _, err := p.do(Request{Record: "rel-v0.16.0", Admit: 11}); err != nil {
				t.Fatal(err)
			}
			releaseForge.graphql = inner
			if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != WithdrawnAdmitted || f.pr(11).method != "" {
				t.Fatalf("after the cut-off admit: #11 recorded %q, armed %q", stateOf(ws, 11), f.pr(11).method)
			}
			f.reset()
			if finish == "admit" {
				if _, err := p.do(Request{Record: "rel-v0.16.0", Admit: 11}); err != nil {
					t.Fatalf("admit again = %v; want #11 armed", err)
				}
			} else if _, err := p.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
				t.Fatal(err)
			}
			if f.pr(11).method != "SQUASH" || !slices.Contains(f.mutations(), "enablePullRequestAutoMerge #11") {
				t.Fatalf("mutations %v, #11 %+v; want #11 armed at its admitted head", f.mutations(), f.pr(11))
			}
			if finish == "release" {
				if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != WithdrawnRearmed {
					t.Fatalf("the tombstone records #11 %q; want rearmed", stateOf(ws, 11))
				}
			}
		})
	}
}

// branchLength is the first-parent length of id's branch on origin.
func (w *world) branchLength(id string) int {
	w.t.Helper()
	return len(strings.Fields(runGit(w.t, w.bare, "rev-list", "--first-parent", RefPrefix+id)))
}

// TestAReArmTheForgeKeepsRefusingWritesNothingUntilItSettles: a re-arm the
// forge refuses is tried again on later acts without a tombstone each time,
// so the branch does not grow with the retries; once the forge accepts, it
// is re-armed and one tombstone records it.
func TestAReArmTheForgeKeepsRefusingWritesNothingUntilItSettles(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	f.refuse["enqueuePullRequest #12"] = "Pull request is not ready to be queued"
	if _, err := p.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	before := w.branchLength("rel-v0.16.0")
	// Two more refusals (three in all, under the bound), one an act.
	w.set(w.clock().Add(time.Minute))
	if _, err := p.reserve("iss-3", "a.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.do(Request{Record: "iss-3", Release: true}); err != nil {
		t.Fatal(err)
	}
	if got := w.branchLength("rel-v0.16.0"); got != before {
		t.Fatalf("the branch grew from %d to %d commits over refused retries; want none added", before, got)
	}
	delete(f.refuse, "enqueuePullRequest #12")
	if _, err := p.reserve("iss-4", "b.go"); err != nil {
		t.Fatal(err)
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 12) != WithdrawnRearmed || !f.pr(12).queued {
		t.Fatalf("#12 recorded %q, %+v; want re-armed once the forge accepts", stateOf(ws, 12), f.pr(12))
	}
	if got := w.branchLength("rel-v0.16.0"); got != before+1 {
		t.Fatalf("branch length %d; want %d, one tombstone recording the re-arm", got, before+1)
	}
}

// TestAReArmRefusedFiveTimesIsLeftToItsOwner: past maxRearmRefusals the
// withdrawal is settled refused, named for its owner, and no copy asks the
// forge again; seventy acts leave the branch bounded.
func TestAReArmRefusedFiveTimesIsLeftToItsOwner(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	f.refuse["enqueuePullRequest #12"] = "GraphQL: Pull request is not mergeable: required status check is pending (enqueuePullRequest)"
	if _, err := p.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	before := w.branchLength("rel-v0.16.0")
	var lines []string
	for i := 0; i < 35; i++ {
		w.set(w.clock().Add(time.Minute))
		res, err := p.reserve("iss-3", "a.go")
		if err != nil {
			t.Fatalf("act %d: %v", i, err)
		}
		rel, err := p.do(Request{Record: "iss-3", Release: true})
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range append(res.Settles, rel.Settles...) {
			if s.Unplaced != "" || len(s.NotRead) > 0 {
				t.Fatalf("act %d: #12 reads as %+v", i, s)
			}
			for _, wd := range s.Settled {
				lines = append(lines, SettledLine(s.Subject, wd))
			}
		}
	}
	tries := 0
	for _, c := range f.calls {
		if c == "enqueuePullRequest #12" {
			tries++
		}
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 12) != WithdrawnRefused || tries != maxRearmRefusals {
		t.Fatalf("#12 recorded %q after %d tries; want refused after %d", stateOf(ws, 12), tries, maxRearmRefusals)
	}
	if got := w.branchLength("rel-v0.16.0"); got != before+1 {
		t.Fatalf("branch length %d after seventy acts; want %d", got, before+1)
	}
	if len(lines) != 1 || !strings.Contains(lines[0], "#12 was not re-armed after rel-v0.16.0: the forge refused its re-arm 5 times") {
		t.Fatalf("settled lines = %q", lines)
	}
}

// TestAWithdrawalTheForgeRefusedIsNotReportedLost: the correction after a
// refused withdrawal keeps the entry, as kept, so the walk for lost
// withdrawals never reads its absence as a later commit's drop.
func TestAWithdrawalTheForgeRefusedIsNotReportedLost(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	f.refuse["disablePullRequestAutoMerge #11"] = "GraphQL: octo does not have permission to disable auto-merge"
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	for _, at := range []time.Time{w.clock(), w.clock().Add(time.Hour)} {
		w.set(at)
		if _, err := Fetch(p.dir, p.opts()); err != nil {
			t.Fatal(err)
		}
		if lost := LostReleaseWithdrawals(p.dir, w.clock()); len(lost) != 0 {
			t.Fatalf("lost = %q; want none: #11 is still armed, never withdrawn", lost[0].Line())
		}
	}
	// Nor after a renewal, which tries it again and, refused again, carries
	// it kept.
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != WithdrawnKept {
		t.Fatalf("the renewal carries #11 as %q; want kept", stateOf(ws, 11))
	}
	if _, err := Fetch(p.dir, p.opts()); err != nil {
		t.Fatal(err)
	}
	if lost := LostReleaseWithdrawals(p.dir, w.clock()); len(lost) != 0 {
		t.Fatalf("lost after the renewal = %q", lost[0].Line())
	}
}

// TestALapsedReleaseThatWithdrewNothingDoesNotBlockTheNextCut: an abandoned
// release reservation that holds nothing is neither named lapsed by the
// launch reading nor in the way of the next release; one that lapsed still
// holding withdrawals is named.
func TestALapsedReleaseThatWithdrewNothingDoesNotBlockTheNextCut(t *testing.T) {
	w := newWorld(t)
	f := newFakeForge(t, map[int]*fakePR{14: {id: "PR_fourteen", head: headA}})
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	w.set(t0.Add(lease.Max + 48*time.Hour))
	if _, err := p.do(Request{Record: "rel-v0.17.0"}); err != nil {
		t.Fatalf("reserving rel-v0.17.0 over an abandoned rel-v0.16.0 = %v", err)
	}
	a, err := ArmedMergesForCut(p.dir, p.opts())
	if err != nil {
		t.Fatal(err)
	}
	if a.Release != "rel-v0.17.0" || a.Lapsed != "" {
		t.Fatalf("release=%q lapsed=%q; want rel-v0.17.0 and no lapse named", a.Release, a.Lapsed)
	}
	// rel-v0.17.0 withdraws #14 and lapses: the reading names it.
	_ = f
	w2 := newWorld(t)
	newFakeForge(t, map[int]*fakePR{14: {id: "PR_fourteen", head: headA, method: "MERGE"}})
	q := w2.clone("1234+octo@users.noreply.github.com", "")
	if _, err := q.do(Request{Record: "rel-v0.17.0"}); err != nil {
		t.Fatal(err)
	}
	w2.set(t0.Add(lease.Max + 48*time.Hour))
	if _, err := Fetch(q.dir, q.opts()); err != nil {
		t.Fatal(err)
	}
	a, err = ArmedMergesForCut(q.dir, q.opts())
	if err != nil {
		t.Fatal(err)
	}
	if a.Release != "" || a.Lapsed != "rel-v0.17.0" {
		t.Fatalf("release=%q lapsed=%q; want rel-v0.17.0 named lapsed", a.Release, a.Lapsed)
	}
}

// TestALapsedReleaseStillBeingCutBlocksASecondRelease: a lapse ends nothing,
// so a lapsed release still holding what it withdrew and admitted blocks
// another person's release as a live one does, with a back-off saying how
// the first one ends; the first cutter can still renew it.
func TestALapsedReleaseStillBeingCutBlocksASecondRelease(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	a := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := a.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.do(Request{Record: "rel-v0.16.0", Admit: 11}); err != nil {
		t.Fatal(err)
	}
	w.set(t0.Add(lease.Max + TakeoverMargin + time.Hour))
	b := w.clone("99+quinn@users.noreply.github.com", "")
	f.reset()
	_, err := b.do(Request{Record: "rel-v0.17.0"})
	if !errors.Is(err, ErrBackOff) || !strings.Contains(err.Error(), "rel-v0.16.0 is still being cut by octo") ||
		!strings.Contains(err.Error(), "abcd reserve rel-v0.16.0 --release") || len(f.mutations()) != 0 {
		t.Fatalf("B's reserve = %v, mutations %v; want a back-off naming how rel-v0.16.0 ends", err, f.mutations())
	}
	if f.pr(11).method != "SQUASH" {
		t.Fatalf("#11 = %+v; want A's admission left armed", f.pr(11))
	}
	if res, err := a.do(Request{Record: "rel-v0.16.0"}); err != nil || res.Act != ActRenewed {
		t.Fatalf("A renews = %+v, %v", res, err)
	}
	if _, err := a.do(Request{Record: "rel-v0.16.0", Admit: 12}); err != nil {
		t.Fatalf("A admits #12 = %v", err)
	}
}

// TestPeersNeverCallsAnUnrecordedWithdrawalDue: the listing reads the
// reader's own withdrawals on an ended release against the branch's
// history, so one no reservation of theirs recorded is not "re-armed by
// your next act".
func TestPeersNeverCallsAnUnrecordedWithdrawalDue(t *testing.T) {
	w := newWorld(t)
	newFakeForge(t, map[int]*fakePR{11: {id: "PR_eleven", head: headA}})
	other := message{Record: "rel-v0.17.0", Holder: "mal", Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: t0, ExpiresAt: t0, ReleasedAt: t0,
		Withdrawn: []Withdrawal{{Number: 11, ID: "PR_eleven", Form: FormAutoMerge, Head: headA, By: "octo", State: WithdrawnPending}}}
	w.pushHostile("rel-v0.17.0", other.text(), t0)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := Fetch(octo.dir, octo.opts()); err != nil {
		t.Fatal(err)
	}
	snap, err := View(octo.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	if got := snap.ReleaseWithdrawals(octo.dir, "octo", w.clock()); len(got) != 0 {
		t.Fatalf("listing = %+v; want nothing of octo's on a branch octo never pushed to", got)
	}
}

// TestALiveRenewalWithdrawsAgainWhatItRecordsWithdrawnButFindsArmed: a
// process cut off between the record push and the forge, or an arming made
// by hand since, leaves a merge recorded withdrawn and armed; the next live
// renewal withdraws it again, and touches nothing it does not record.
func TestALiveRenewalWithdrawsAgainWhatItRecordsWithdrawnButFindsArmed(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	f.prs[11].method = "SQUASH" // armed again
	f.prs[14].method = "MERGE"  // armed since, not recorded
	f.reset()
	w.set(t0.Add(time.Hour))
	res, err := p.do(Request{Record: "rel-v0.16.0"})
	if err != nil || res.Act != ActRenewed {
		t.Fatalf("renew = %+v, %v", res, err)
	}
	if got := f.mutations(); !slices.Equal(got, []string{"disablePullRequestAutoMerge #11"}) || len(res.Withdrew) != 1 {
		t.Fatalf("mutations = %v, withdrew %v; want #11 withdrawn again and #14 untouched", got, numbersOf(res.Withdrew))
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != WithdrawnPending || stateOf(ws, 14) != "" {
		t.Fatalf("recorded %+v", ws)
	}
}

// TestTheLaunchReadingExemptsOnlyAnAdmissionTheReaderPushed: the launch
// check's exemption is read from the reader's own newest push: an admission a
// commit mal pushed over it records is not exempt, and an app's push over it
// leaves none exempt, naming why.
func TestTheLaunchReadingExemptsOnlyAnAdmissionTheReaderPushed(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.do(Request{Record: "rel-v0.16.0", Admit: 11}); err != nil {
		t.Fatal(err)
	}
	a, err := ArmedMergesForCut(p.dir, p.opts())
	if err != nil || len(a.Admitted) != 1 || len(a.Unverified) != 0 {
		t.Fatalf("reading = %+v, %v; want #11 admitted", a, err)
	}
	// A renewal-shaped commit mal pushes, recording #12 admitted too.
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	for i := range ws {
		if ws[i].Number == 12 {
			ws[i].State = WithdrawnAdmitted
		}
	}
	m.Withdrawn = ws
	c := w.pushChild("rel-v0.16.0", w.remoteTip("rel-v0.16.0"), m.text(), w.clock())
	f.pusherOf[c] = "mal"
	if _, err := Fetch(p.dir, p.opts()); err != nil {
		t.Fatal(err)
	}
	a, err = ArmedMergesForCut(p.dir, p.opts())
	if err != nil || !slices.Equal(numbersOf(a.Admitted), []int{11}) || len(a.Unverified) != 0 {
		t.Fatalf("mal's commit over it: reading = %+v, %v; want only #11, octo's own, exempt", a, err)
	}
	f.pusherOf[c] = "app"
	forgetVerdicts(t, p.dir)
	a, err = ArmedMergesForCut(p.dir, p.opts())
	if err != nil || len(a.Admitted) != 0 || len(a.Unverified) != 2 || !strings.Contains(a.UnverifiedWhy, "an app pushed commit "+c[:12]) {
		t.Fatalf("an app's commit over it: reading = %+v, %v; want none exempt and why named", a, err)
	}
}

// TestFirstRecordMustBeAReservationRecordingItWithdrawn: the chain vouches
// only for a first record that is a reservation, not a tombstone, of the
// withdrawer's, recording it withdrawn; a tip carrying it again over a
// commit that settled it is its own first record.
func TestFirstRecordMustBeAReservationRecordingItWithdrawn(t *testing.T) {
	w := Withdrawal{Number: 11, ID: "PR_eleven", Form: FormAutoMerge, Head: headA, By: "octo", State: WithdrawnPending}
	with := func(state string) []Withdrawal { x := w; x.State = state; return []Withdrawal{x} }
	res := func(state string) chainLink {
		return chainLink{commit: "r" + state, readable: true, m: message{Holder: "octo", Withdrawn: with(state)}}
	}
	tomb := func(state string) chainLink {
		return chainLink{commit: "t" + state, readable: true, m: message{Holder: "octo", ReleasedAt: t0, Withdrawn: with(state)}}
	}
	for name, c := range map[string]struct {
		links []chainLink
		want  bool
	}{
		"reservation recording it withdrawn": {[]chainLink{tomb(WithdrawnPending), res(WithdrawnPending)}, true},
		"first record a tombstone":           {[]chainLink{tomb(WithdrawnPending), tomb(WithdrawnPending)}, false},
		"first record admitted":              {[]chainLink{tomb(WithdrawnPending), res(WithdrawnAdmitted)}, false},
		"carried again over a settled one":   {[]chainLink{tomb(WithdrawnPending), tomb(WithdrawnRearmed), res(WithdrawnPending)}, false},
		"another holder's first record":      {[]chainLink{tomb(WithdrawnPending), {commit: "q", readable: true, m: message{Holder: "quinn", Withdrawn: with(WithdrawnPending)}}}, false},
		"admitted over its withdrawn record": {[]chainLink{res(WithdrawnAdmitted), res(WithdrawnPending)}, true},
		"re-withdrawn over a kept one":       {[]chainLink{res(WithdrawnPending), res(WithdrawnKept), res(WithdrawnPending)}, true},
	} {
		if _, got := firstRecord(c.links, true, w); got != c.want {
			t.Errorf("%s: vouched %v; want %v", name, got, c.want)
		}
	}
}

// TestLostNeedsTheTipToLackTheNumber: an entry an ancestor held withdrawn
// and a later commit dropped is not lost while the tip carries its number
// again.
func TestLostNeedsTheTipToLackTheNumber(t *testing.T) {
	w := Withdrawal{Number: 11, ID: "PR_eleven", Form: FormAutoMerge, Head: headA, By: "octo", State: WithdrawnPending}
	res := func(c string, ws ...Withdrawal) chainLink {
		return chainLink{commit: c, readable: true, m: message{Holder: "octo", Withdrawn: ws}}
	}
	moved := w
	moved.Head = headB
	if lost := lostWithdrawals([]chainLink{res("c", moved), res("b"), res("a", w)}, true); len(lost) != 0 {
		t.Fatalf("lost = %+v; want none while the tip carries #11", lost)
	}
	if lost := lostWithdrawals([]chainLink{res("c"), res("b"), res("a", w)}, true); len(lost) != 1 {
		t.Fatalf("lost = %+v; want #11", lost)
	}
}

// TestLostWithdrawalsAreLookedForOnlyWithinTheWindow: a release tip claimed
// longer than lostWindow ago is not walked.
func TestLostWithdrawalsAreLookedForOnlyWithinTheWindow(t *testing.T) {
	w := newWorld(t)
	armedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	// An older binary's renewal, which drops the trailers.
	m, _ := w.tipWithdrawals("rel-v0.16.0")
	m.Withdrawn = nil
	w.pushChild("rel-v0.16.0", w.remoteTip("rel-v0.16.0"), m.text(), w.clock())
	if _, err := Fetch(p.dir, p.opts()); err != nil {
		t.Fatal(err)
	}
	if lost := LostReleaseWithdrawals(p.dir, m.ClaimedAt.Add(lostWindow-time.Hour)); len(lost) != 1 {
		t.Fatalf("within the window: lost = %+v; want rel-v0.16.0's", lost)
	}
	if lost := LostReleaseWithdrawals(p.dir, m.ClaimedAt.Add(lostWindow+time.Hour)); len(lost) != 0 {
		t.Fatalf("past the window: lost = %+v; want none", lost)
	}
}

// TestTombstoneFitIsMeasuredWithTheLongestLogin: the fit is measured with the
// longest GitHub username, not only the longest label: a list that fits a
// short login under the longest label, and not the longest login, does not
// fit.
func TestTombstoneFitIsMeasuredWithTheLongestLogin(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	pr, err := prepare(p.dir, Request{Record: "rel-v0.16.0"}, p.opts())
	if err != nil {
		t.Fatal(err)
	}
	entry := func(n, idLen int) Withdrawal {
		return Withdrawal{Number: 100000000 + n, ID: strings.Repeat("A", idLen), Form: FormAutoRebase, Head: headA, By: "octo", State: WithdrawnPending}
	}
	fits := func(ws []Withdrawal, holder string) bool {
		m := pr.tombstone(nil)
		m.Holder, m.AssistedBy = holder, longestLabel
		for _, w := range ws {
			w.State = longestState()
			m.Withdrawn = append(m.Withdrawn, w)
		}
		return checkMessage(p.dir, m.text()) == nil
	}
	var ws []Withdrawal
	for n := 1; fits(append(slices.Clone(ws), entry(n, 100)), "octo"); n++ {
		ws = append(ws, entry(n, 100))
	}
	// Shorten the entries one id at a time until the list fits the short
	// login and not the longest.
	for i := len(ws) - 1; i >= 0; i-- {
		for l := 100; l >= 1; l-- {
			ws[i] = entry(i+1, l)
			next := append(slices.Clone(ws), entry(len(ws)+1, 100))
			if !fits(next, "octo") || fits(next, longestLogin) {
				continue
			}
			if pr.tombstoneFits(next) {
				t.Fatalf("a list that fits only a short login reads as fitting (%d entries)", len(next))
			}
			return
		}
	}
	t.Fatal("no list fits the short login and not the longest")
}

// TestATakeOverByTheSamePersonDoesNotWithdrawAnAdmittedMergeAgain: an
// admission stands across the same person's take-over from another copy.
func TestATakeOverByTheSamePersonDoesNotWithdrawAnAdmittedMergeAgain(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	a := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := a.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.do(Request{Record: "rel-v0.16.0", Admit: 11}); err != nil {
		t.Fatal(err)
	}
	b := w.clone("1234+octo@users.noreply.github.com", "")
	f.reset()
	res, err := b.do(Request{Record: "rel-v0.16.0", TakeOver: true})
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(f.mutations(), "disablePullRequestAutoMerge #11") || f.pr(11).method != "SQUASH" {
		t.Fatalf("take-over = %+v, mutations %v; want admitted #11 left armed", res.Act, f.mutations())
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != WithdrawnAdmitted {
		t.Fatalf("#11 recorded %q; want admitted", stateOf(ws, 11))
	}
}

// TestTheSweepTouchesNothingWhenGhActsAsAnotherAccount: before claiming an
// ended release, the sweep compares gh's account with the holder, and on a
// mismatch pushes nothing.
func TestTheSweepTouchesNothingWhenGhActsAsAnotherAccount(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	w.tagOrigin("v0.16.0")
	tip := w.remoteTip("rel-v0.16.0")
	f.logins[p.dir] = "quinn"
	f.reset()
	res, err := p.reserve("iss-8", "o.go")
	if err != nil {
		t.Fatal(err)
	}
	if w.remoteTip("rel-v0.16.0") != tip || len(f.mutations()) != 0 {
		t.Fatalf("tip moved %v, mutations %v; want nothing pushed or armed", w.remoteTip("rel-v0.16.0") != tip, f.mutations())
	}
	if len(res.Settles) != 1 || !strings.Contains(res.Settles[0].Failed, "gh acts as quinn") {
		t.Fatalf("settles = %+v; want the mismatch named", res.Settles)
	}
}
