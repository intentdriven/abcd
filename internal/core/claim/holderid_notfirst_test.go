package claim

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// A copy never withdraws a pull request whose entry under its own id it did
// not first record (iss-2610290809526033). An entry naming the reader's kept
// id that a commit another login pushed first recorded is not the reader's,
// so their copy never re-arms it; were their copy to withdraw that pull
// request, carrying the entry unchanged, nothing would ever re-arm it. So the
// withdrawal is refused: the forge is left as it is, nothing is recorded, the
// pull request stays armed, and the launch check keeps naming it until it is
// withdrawn from the merge path by hand or merges.

// sevenForge is a forge with #7 open, armed for nothing, and its comment
// writer set.
func sevenForge(t *testing.T) *fakeForge {
	t.Helper()
	f := newFakeForge(t, map[int]*fakePR{7: {id: "PR_seven", head: headA}})
	releaseForge.comments = f
	return f
}

// armedReading is what the launch check reads for p, fetched first.
func armedReading(t *testing.T, p *participant) ArmedMerges {
	t.Helper()
	if _, err := Fetch(p.dir, p.opts()); err != nil {
		t.Fatal(err)
	}
	a, err := ArmedMergesForCut(p.dir, p.opts())
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// namesArmed reports whether a reading lists pull request n armed and does
// not exempt it: not admitted, not recorded kept, not awaiting the record.
func namesArmed(a ArmedMerges, n int) bool {
	is := func(w Withdrawal) bool { return w.Number == n }
	return slices.ContainsFunc(a.Armed, func(pr ArmedPR) bool { return pr.Number == n }) &&
		!slices.ContainsFunc(a.Admitted, is) && !slices.ContainsFunc(a.Kept, is) && !slices.ContainsFunc(a.Unverified, is)
}

// TestAnEntryUnderTheReadersIDAnotherLoginFirstRecordedIsNeverWithdrawn:
// mal pushes a reservation commit on rel-v0.16.0 recording #7 withdrawn
// under octo's kept id, and it lapses; #7 is then armed. octo's next act on
// the release works from that commit, as the tip it is written over, and
// carries mal's #7 entry unchanged: a renewal, when mal's commit is the
// branch's first and is held under octo's id and copy key, and a take-over,
// when it is held under another id over octo's own lapsed cut. Either act
// withdraws every merge armed, but leaves #7 armed, touching nothing on the
// forge, and names it with the step to take by hand; the launch check names
// #7 armed until it is withdrawn by hand; and octo's release re-arms nothing.
func TestAnEntryUnderTheReadersIDAnotherLoginFirstRecordedIsNeverWithdrawn(t *testing.T) {
	for name, c := range map[string]struct {
		seed func(w *world, f *fakeForge, octo *participant) message
		act  Act
	}{
		"a renewal of the commit held under octo's id": {seed: func(w *world, f *fakeForge, octo *participant) message {
			m, _ := w.tipWithdrawals("iss-5") // octo's copy key, as any trailer of his shows it
			return message{Record: "rel-v0.16.0", Holder: octo.keptHolderID(), Copy: m.Copy, ClaimedAt: w.clock(), ExpiresAt: w.clock().Add(time.Hour), AssistedBy: "abcd:0.0.0"}
		}, act: ActRenewed},
		"a take-over of the commit held under another id": {seed: func(w *world, f *fakeForge, octo *participant) message {
			if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
				t.Fatal(err)
			}
			return message{Record: "rel-v0.16.0", Holder: quinnID, Copy: "1f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: w.clock(), ExpiresAt: w.clock().Add(time.Hour), AssistedBy: "abcd:0.0.0"}
		}, act: ActTakenOver},
	} {
		t.Run(name, func(t *testing.T) {
			w := newWorld(t)
			f := sevenForge(t)
			octo := w.clone("1234+octo@users.noreply.github.com", "")
			if _, err := octo.reserve("iss-5", "a.go"); err != nil {
				t.Fatal(err)
			}
			m := c.seed(w, f, octo)
			theirs := Withdrawal{Number: 7, ID: "PR_seven", Form: FormAutoSquash, Head: headA, By: octo.keptHolderID(), State: WithdrawnPending}
			m.Withdrawn = []Withdrawal{theirs}
			if w.remoteTip("rel-v0.16.0") == "" {
				f.pusherOf[w.pushTipAt("rel-v0.16.0", m.text(), w.clock())] = "mal"
			} else {
				w.pushAs(f, "rel-v0.16.0", "mal", m.text())
			}
			f.prs[7].method = "SQUASH" // armed by its author since
			f.reset()
			w.set(w.clock().Add(2 * time.Hour))
			res, err := octo.do(Request{Record: "rel-v0.16.0"})
			if err != nil || res.Act != c.act || res.RenewedLive {
				t.Fatalf("act = %+v, %v; want %s afresh", res, err, c.act)
			}
			if got := actedOn(f, 7); len(got) != 0 || f.pr(7).method != "SQUASH" {
				t.Fatalf("#7 was acted on: %v, now %+v; want it left armed", got, f.pr(7))
			}
			i := slices.IndexFunc(res.ForgeFailures, func(x ForgeFailure) bool { return x.Number == 7 })
			if i < 0 || res.ForgeFailures[i].Act != ForgeWithdraw {
				t.Fatalf("failures = %+v; want #7's withdrawal refused", res.ForgeFailures)
			}
			line := res.ForgeFailures[i].Line()
			for _, want := range []string{"#7 was not withdrawn", "another login pushed", "withdraw #7 from the merge path by hand", "launch check refuses the cut while it is armed"} {
				if !strings.Contains(line, want) {
					t.Fatalf("the refusal = %q; want it to say %q", line, want)
				}
			}
			noLogin(t, "the refusal", line)
			if strings.Contains(line, "mal") {
				t.Fatalf("the refusal names the login that pushed: %q", line)
			}
			if e, ok := withdrawalOf(tipEntries(t, w, "rel-v0.16.0"), 7); !ok || e != theirs {
				t.Fatalf("tip records %+v; want mal's #7 entry carried unchanged, nothing recorded over it", tipEntries(t, w, "rel-v0.16.0"))
			}
			if a := armedReading(t, octo); a.Release != "rel-v0.16.0" || !namesArmed(a, 7) {
				t.Fatalf("launch reading = %+v; want #7 named armed against octo's cut", a)
			}
			f.prs[7].method = "" // withdrawn from the merge path by hand
			if a := armedReading(t, octo); a.Release != "rel-v0.16.0" || slices.ContainsFunc(a.Armed, func(pr ArmedPR) bool { return pr.Number == 7 }) {
				t.Fatalf("launch reading = %+v; want #7 no longer armed once withdrawn by hand", a)
			}
			w.set(w.clock().Add(30 * time.Minute))
			if _, err := octo.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
				t.Fatal(err)
			}
			if got := actedOn(f, 7); len(got) != 0 || f.pr(7).method != "" {
				t.Fatalf("#7 was acted on: %v, now %+v; want nothing re-armed", got, f.pr(7))
			}
		})
	}
}

// TestTheReadersOwnEntryIsWithdrawnAgainOnAFreshRenewal: octo's reservation
// withdraws #7, recording it under his kept id in his own push; #7's author
// arms it again; octo's lapsed reservation is renewed afresh. His own entry
// is his to act on, so #7 is withdrawn again, with no failure.
func TestTheReadersOwnEntryIsWithdrawnAgainOnAFreshRenewal(t *testing.T) {
	w := newWorld(t)
	f := sevenForge(t)
	f.prs[7].method = "SQUASH"
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	if len(ws) != 1 || ws[0].Number != 7 || ws[0].By != octo.keptHolderID() || f.pr(7).method != "" {
		t.Fatalf("reservation records %+v, #7 %+v; want #7 withdrawn under octo's id", ws, f.pr(7))
	}
	f.prs[7].method = "SQUASH" // armed again by its author
	f.reset()
	w.set(m.ExpiresAt.Add(2 * time.Minute))
	res, err := octo.do(Request{Record: "rel-v0.16.0"})
	if err != nil || res.Act != ActRenewed || res.RenewedLive {
		t.Fatalf("renewal = %+v, %v; want a renewal of a lapsed reservation", res, err)
	}
	if got := actedOn(f, 7); !slices.Contains(got, "disablePullRequestAutoMerge #7") || f.pr(7).method != "" || len(res.ForgeFailures) != 0 {
		t.Fatalf("#7 acted on %v, now %+v, failures %+v; want it withdrawn again", got, f.pr(7), res.ForgeFailures)
	}
}
