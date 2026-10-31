package claim

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// A copy refuses to withdraw a pull request only where its push would not
// become the entry's first record (iss-2610290809526033): the record it
// writes over holds that pull request still withdrawn or admitted, as the
// same withdrawal the copy would record, and the copy does not act on that
// entry. A kept entry, and an entry under another id of the reader's, is
// withdrawn over: the copy's push is the new entry's first record, so its
// own release re-arms it.

// keptSeven is #7 recorded kept, the forge having refused its withdrawal, at
// headA in the auto-squash form, under by.
func keptSeven(by string) Withdrawal {
	return Withdrawal{Number: 7, ID: "PR_seven", Form: FormAutoSquash, Head: headA, By: by, State: WithdrawnKept}
}

// rewithdrewSeven fails t unless the act res withdrew #7 again, with no
// failure for it, and the tip records it withdrawn under by.
func rewithdrewSeven(t *testing.T, w *world, f *fakeForge, res Result, by string) {
	t.Helper()
	if got := actedOn(f, 7); !slices.Contains(got, "disablePullRequestAutoMerge #7") || f.pr(7).method != "" {
		t.Fatalf("#7 acted on %v, now %+v; want it withdrawn again", got, f.pr(7))
	}
	if i := slices.IndexFunc(res.ForgeFailures, func(x ForgeFailure) bool { return x.Number == 7 }); i >= 0 {
		t.Fatalf("failures = %+v; want none for #7", res.ForgeFailures)
	}
	if e, ok := withdrawalOf(tipEntries(t, w, "rel-v0.16.0"), 7); !ok || e.By != by || e.State != WithdrawnPending {
		t.Fatalf("tip records %+v; want #7 withdrawn under the act's own id", tipEntries(t, w, "rel-v0.16.0"))
	}
}

// TestAKeptEntryOfTheReadersOtherHomeIsWithdrawnAgainOnATakeOver: maya
// keeps two homes that declare each other's ids under one login. Home A's
// cut records #7 kept, the forge refusing its withdrawal, and lapses; home
// B takes the cut over once the forge allows it. B's push is the first
// record of #7 withdrawn under B's own id, so #7 is withdrawn again, with no
// failure, and B's release re-arms it.
func TestAKeptEntryOfTheReadersOtherHomeIsWithdrawnAgainOnATakeOver(t *testing.T) {
	w := newWorld(t)
	f := sevenForge(t)
	f.prs[7].method = "SQUASH"
	f.refuse["disablePullRequestAutoMerge #7"] = "GraphQL: maya does not have permission to disable auto-merge"
	a := w.clone("5678+maya@users.noreply.github.com", t.TempDir())
	declareHolderIDs(t, a.home, studioID+"\n")
	if _, err := a.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	if e, ok := withdrawalOf(ws, 7); !ok || e != keptSeven(a.keptHolderID()) {
		t.Fatalf("home A's cut records %+v; want #7 kept under its id", ws)
	}
	b := w.clone("5678+maya@users.noreply.github.com", t.TempDir())
	keepHolderID(t, b.home, studioID)
	declareHolderIDs(t, b.home, a.keptHolderID()+"\n")
	delete(f.refuse, "disablePullRequestAutoMerge #7")
	f.reset()
	w.set(m.ExpiresAt.Add(2 * time.Minute))
	res, err := b.do(Request{Record: "rel-v0.16.0", TakeOver: true})
	if err != nil || res.Act != ActTakenOver {
		t.Fatalf("home B's act = %+v, %v; want a take-over", res, err)
	}
	rewithdrewSeven(t, w, f, res, studioID)
	f.reset()
	if _, err := b.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	if f.pr(7).method != "SQUASH" {
		t.Fatalf("#7 after home B's release = %+v, mutations %v; want it re-armed", f.pr(7), f.mutations())
	}
}

// TestAKeptEntryUnderTheReadersIDAnotherLoginRecordedIsWithdrawnAgain: mal
// pushes a commit recording #7 kept under octo's kept id; #7 is armed.
// octo's next act withdraws #7, its own push the first record of the
// withdrawn entry, with no failure, and octo's release re-arms it.
func TestAKeptEntryUnderTheReadersIDAnotherLoginRecordedIsWithdrawnAgain(t *testing.T) {
	w := newWorld(t)
	f := sevenForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	cm, _ := w.tipWithdrawals("iss-5") // octo's copy key, as any trailer of his shows it
	m := message{Record: "rel-v0.16.0", Holder: octo.keptHolderID(), Copy: cm.Copy, ClaimedAt: w.clock(), ExpiresAt: w.clock().Add(time.Hour),
		Withdrawn: []Withdrawal{keptSeven(octo.keptHolderID())}, AssistedBy: "abcd:0.0.0"}
	f.pusherOf[w.pushTipAt("rel-v0.16.0", m.text(), w.clock())] = "mal"
	f.prs[7].method = "SQUASH"
	f.reset()
	w.set(w.clock().Add(2 * time.Hour))
	res, err := octo.do(Request{Record: "rel-v0.16.0"})
	if err != nil {
		t.Fatal(err)
	}
	rewithdrewSeven(t, w, f, res, octo.keptHolderID())
	f.reset()
	w.set(w.clock().Add(30 * time.Minute))
	if _, err := octo.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	if f.pr(7).method != "SQUASH" {
		t.Fatalf("#7 after octo's release = %+v, mutations %v; want it re-armed", f.pr(7), f.mutations())
	}
}

// TestAnAdmissionUnderTheReadersIDAnotherLoginRecordedIsRefusedAndNamed:
// mal pushes a commit recording #7 admitted under octo's kept id; #7 is
// armed. octo's next act leaves #7 armed, touching nothing on the forge,
// and names it with the step to take by hand, as it names a withdrawn entry
// another login recorded; the launch check names #7 armed, not admitted.
func TestAnAdmissionUnderTheReadersIDAnotherLoginRecordedIsRefusedAndNamed(t *testing.T) {
	w := newWorld(t)
	f := sevenForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	cm, _ := w.tipWithdrawals("iss-5")
	theirs := keptSeven(octo.keptHolderID())
	theirs.State = WithdrawnAdmitted
	m := message{Record: "rel-v0.16.0", Holder: octo.keptHolderID(), Copy: cm.Copy, ClaimedAt: w.clock(), ExpiresAt: w.clock().Add(time.Hour),
		Withdrawn: []Withdrawal{theirs}, AssistedBy: "abcd:0.0.0"}
	f.pusherOf[w.pushTipAt("rel-v0.16.0", m.text(), w.clock())] = "mal"
	f.prs[7].method = "SQUASH"
	f.reset()
	w.set(w.clock().Add(2 * time.Hour))
	res, err := octo.do(Request{Record: "rel-v0.16.0"})
	if err != nil {
		t.Fatal(err)
	}
	if got := actedOn(f, 7); len(got) != 0 || f.pr(7).method != "SQUASH" {
		t.Fatalf("#7 was acted on: %v, now %+v; want it left armed", got, f.pr(7))
	}
	i := slices.IndexFunc(res.ForgeFailures, func(x ForgeFailure) bool { return x.Number == 7 })
	if i < 0 || res.ForgeFailures[i].Act != ForgeWithdraw {
		t.Fatalf("failures = %+v; want #7's withdrawal refused and named", res.ForgeFailures)
	}
	line := res.ForgeFailures[i].Line()
	for _, want := range []string{"#7 was not withdrawn", "another login pushed", "withdraw #7 from the merge path by hand"} {
		if !strings.Contains(line, want) {
			t.Fatalf("the refusal = %q; want it to say %q", line, want)
		}
	}
	noLogin(t, "the refusal", line)
	if a := armedReading(t, octo); a.Release != "rel-v0.16.0" || !namesArmed(a, 7) {
		t.Fatalf("launch reading = %+v; want #7 named armed against octo's cut", a)
	}
}
