package claim

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/github"
)

// TestAFormAnotherLoginRewroteIsNeverReArmed: the form a withdrawal is
// re-armed in is part of what its withdrawer recorded. A tombstone another
// login pushes over the genuine record, changing the forms, records each
// changed entry for the first time, so it is named unrecorded and never
// re-armed; an entry it carried unchanged is re-armed in its own form.
func TestAFormAnotherLoginRewroteIsNeverReArmed(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	for i := range ws {
		switch ws[i].Number {
		case 11:
			ws[i].Form = FormAutoRebase // withdrawn as auto:squash
		case 12:
			ws[i].Form = FormAutoMerge // withdrawn as queue
		}
	}
	m.Withdrawn = ws
	m.ReleasedAt, m.ExpiresAt = t0, t0
	tomb := w.pushChild("rel-v0.16.0", w.remoteTip("rel-v0.16.0"), m.text(), t0)
	f.pusherOf[tomb] = "mal"
	f.reset()
	w.set(t0.Add(time.Minute))
	res, err := octo.reserve("iss-8", "o.go")
	if err != nil {
		t.Fatal(err)
	}
	if pr := f.pr(11); pr.method != "" {
		t.Fatalf("#11 = %+v; want it left withdrawn, never armed in the form mal wrote", pr)
	}
	if pr := f.pr(12); pr.method != "" || pr.queued {
		t.Fatalf("#12 = %+v; want it left withdrawn, never armed in the form mal wrote", pr)
	}
	if got := f.mutations(); !slices.Equal(got, []string{"enqueuePullRequest #13"}) {
		t.Fatalf("mutations = %v; want only #13, carried unchanged, re-armed in its own form", got)
	}
	if len(res.Settles) != 1 || !slices.Equal(numbersOf(res.Settles[0].Unrecorded), []int{11, 12}) {
		t.Fatalf("settles = %+v; want #11 and #12 named unrecorded", res.Settles)
	}
}

// TestAnAdmissionArmsOnlyTheFormItsWithdrawerRecorded: a renewal-shaped
// commit another login pushes over the live reservation, changing a
// withdrawal's form, is that entry's first record, so --admit refuses it and
// arms nothing; abcd's own correction of a form, pushed by the holder, still
// vouches.
func TestAnAdmissionArmsOnlyTheFormItsWithdrawerRecorded(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	f.refuse["dequeuePullRequest #13"] = "network unreachable" // a partial withdrawal: corrected to auto:merge
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	if stateOf(ws, 13) != WithdrawnPending || ws[2].Form != FormAutoMerge {
		t.Fatalf("recorded %+v; want #13 corrected to auto:merge", ws)
	}
	for i := range ws {
		if ws[i].Number == 11 {
			ws[i].Form = FormAutoRebase
		}
	}
	m.Withdrawn = ws
	m.ClaimedAt, m.ExpiresAt = w.clock(), w.clock().Add(time.Hour)
	c := w.pushChild("rel-v0.16.0", w.remoteTip("rel-v0.16.0"), m.text(), w.clock())
	f.pusherOf[c] = "mal"
	f.reset()
	_, err := octo.do(Request{Record: "rel-v0.16.0", Admit: 11})
	if err == nil || !strings.Contains(err.Error(), "recorded its withdrawal first") || len(f.mutations()) != 0 {
		t.Fatalf("admit #11 = %v, mutations %v; want refused, nothing armed", err, f.mutations())
	}
	if _, err := octo.do(Request{Record: "rel-v0.16.0", Admit: 13}); err != nil {
		t.Fatalf("admit #13 = %v; want the holder's own correction to vouch", err)
	}
	// #13 is still queued, so the admission finds it armed and records it.
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 13) != WithdrawnAdmitted || stateOf(ws, 11) != WithdrawnPending {
		t.Fatalf("recorded %+v; want #13 admitted and #11 still withdrawn", ws)
	}
}

// TestALiveRenewalWithdrawsAgainOnlyWhatItsWithdrawerRecorded: a commit
// another login pushes over the live reservation, reading as the holder's
// own renewal and recording a withdrawal the holder never made, of a pull
// request armed after the reservation, never makes the holder's renewal
// withdraw that pull request: the entry is named unrecorded, and the release
// that follows leaves it armed.
func TestALiveRenewalWithdrawsAgainOnlyWhatItsWithdrawerRecorded(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	f.prs[14].method = "MERGE" // armed by its author after the reservation
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	m.Withdrawn = append(ws, Withdrawal{Number: 14, ID: "PR_fourteen", Form: FormAutoMerge, Head: headA, By: "octo", State: WithdrawnPending})
	m.ClaimedAt, m.ExpiresAt = w.clock(), w.clock().Add(time.Hour)
	planted := w.pushChild("rel-v0.16.0", w.remoteTip("rel-v0.16.0"), m.text(), w.clock())
	f.pusherOf[planted] = "mal"
	f.reset()
	w.set(t0.Add(time.Minute))
	res, err := octo.do(Request{Record: "rel-v0.16.0"})
	if err != nil || res.Act != ActRenewed || !res.RenewedLive {
		t.Fatalf("renewal = %+v, %v; want a live renewal", res, err)
	}
	if got := f.mutations(); len(got) != 0 || f.pr(14).method != "MERGE" {
		t.Fatalf("mutations = %v, #14 %+v; want #14 left armed", got, f.pr(14))
	}
	if len(res.Settles) != 1 || !slices.Equal(numbersOf(res.Settles[0].Unrecorded), []int{14}) {
		t.Fatalf("settles = %+v; want #14 named unrecorded", res.Settles)
	}
	f.reset()
	w.set(t0.Add(2 * time.Hour))
	if _, err := octo.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(f.mutations(), "disablePullRequestAutoMerge #14") || f.pr(14).method != "MERGE" {
		t.Fatalf("release mutations = %v, #14 %+v; want #14 untouched", f.mutations(), f.pr(14))
	}
}

// TestALiveRenewalWithdrawsAgainNothingTheRecordDoesNotSettle: the
// withdrawal a live renewal would make again is held to the forge's record
// of pushes as a re-arm is: a record behind the tip defers it, and an app's
// push leaves it for its person.
func TestALiveRenewalWithdrawsAgainNothingTheRecordDoesNotSettle(t *testing.T) {
	for name, c := range map[string]struct {
		set   func(f *fakeForge, record string)
		check func(s Settle) bool
	}{
		"behind": {func(f *fakeForge, _ string) { f.lag = 2 }, func(s Settle) bool { return len(s.Unverified) == 1 }},
		"app":    {func(f *fakeForge, record string) { f.pusherOf[record] = "app" }, func(s Settle) bool { return len(s.AppPushed) == 1 }},
	} {
		w := newWorld(t)
		f := newFakeForge(t, map[int]*fakePR{11: {id: "PR_eleven", head: headA, method: "SQUASH"}})
		octo := w.clone("1234+octo@users.noreply.github.com", "")
		if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
			t.Fatal(err)
		}
		record := w.remoteTip("rel-v0.16.0")
		f.prs[11].method = "SQUASH" // armed again
		c.set(f, record)
		f.reset()
		w.set(t0.Add(time.Hour))
		res, err := octo.do(Request{Record: "rel-v0.16.0"})
		if err != nil || res.Act != ActRenewed {
			t.Fatalf("%s: renewal = %+v, %v", name, res, err)
		}
		if len(f.mutations()) != 0 || len(res.Settles) != 1 || !c.check(res.Settles[0]) {
			t.Fatalf("%s: mutations %v, settles %+v; want nothing withdrawn again and #11 named", name, f.mutations(), res.Settles)
		}
	}
}

// TestAWithdrawalTheForgeRefusedIsTriedAgainByALiveRenewal: a pull request
// the forge refused to withdraw is recorded kept, carried by the holder's
// own renewals, and withdrawn on the first live renewal the forge accepts it
// on; the release then re-arms it as any other.
func TestAWithdrawalTheForgeRefusedIsTriedAgainByALiveRenewal(t *testing.T) {
	w := newWorld(t)
	f := newFakeForge(t, map[int]*fakePR{11: {id: "PR_eleven", head: headA, method: "SQUASH"}})
	f.refuse["disablePullRequestAutoMerge #11"] = "HTTP 502"
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != WithdrawnKept {
		t.Fatalf("recorded %+v; want #11 kept", ws)
	}
	// Still refused: the renewal tries, and keeps it kept.
	w.set(t0.Add(time.Minute))
	if res, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil || len(res.ForgeFailures) != 1 {
		t.Fatalf("renewal = %+v, %v; want #11 tried again and refused", res, err)
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != WithdrawnKept {
		t.Fatalf("recorded %+v; want #11 still kept", ws)
	}
	delete(f.refuse, "disablePullRequestAutoMerge #11")
	f.reset()
	w.set(t0.Add(2 * time.Minute))
	res, err := octo.do(Request{Record: "rel-v0.16.0"})
	if err != nil || len(res.Withdrew) != 1 || !slices.Equal(f.mutations(), []string{"disablePullRequestAutoMerge #11"}) {
		t.Fatalf("renewal = %+v, %v, mutations %v; want #11 withdrawn", res, err, f.mutations())
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != WithdrawnPending {
		t.Fatalf("recorded %+v; want #11 withdrawn", ws)
	}
	f.reset()
	if _, err := octo.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	if got := f.mutations(); !slices.Equal(got, []string{"enablePullRequestAutoMerge #11"}) || f.pr(11).method != "SQUASH" {
		t.Fatalf("release mutations = %v, #11 %+v; want #11 re-armed", got, f.pr(11))
	}
}

// TestAKeptWithdrawalIsVouchedForByTheRecordBeneathIt: a tip entry recorded
// kept is vouched for by the withdrawn record beneath the commits carrying
// it kept, and by nothing else.
func TestAKeptWithdrawalIsVouchedForByTheRecordBeneathIt(t *testing.T) {
	w := Withdrawal{Number: 11, ID: "PR_eleven", Form: FormAutoMerge, Head: headA, By: "octo", State: WithdrawnKept}
	link := func(c, holder, state string) chainLink {
		x := w
		x.State = state
		return chainLink{commit: c, readable: true, m: message{Holder: holder, Withdrawn: []Withdrawal{x}}}
	}
	for name, c := range map[string]struct {
		links []chainLink
		want  string
	}{
		"kept over its record":            {[]chainLink{link("c", "octo", WithdrawnKept), link("b", "octo", WithdrawnKept), link("a", "octo", WithdrawnPending)}, "a"},
		"kept with no record beneath":     {[]chainLink{link("b", "octo", WithdrawnKept), link("a", "octo", WithdrawnKept)}, ""},
		"kept over another's record":      {[]chainLink{link("b", "octo", WithdrawnKept), link("a", "quinn", WithdrawnPending)}, ""},
		"kept over a settled one":         {[]chainLink{link("b", "octo", WithdrawnKept), link("a", "octo", WithdrawnRearmed)}, ""},
		"withdrawn, kept, then withdrawn": {[]chainLink{link("c", "octo", WithdrawnKept), link("b", "octo", WithdrawnPending), link("a", "octo", WithdrawnKept)}, "b"},
	} {
		got, ok := firstRecord(c.links, true, w)
		if got != c.want || ok != (c.want != "") {
			t.Errorf("%s: first record %q (%v); want %q", name, got, ok, c.want)
		}
	}
	if got := numbersOf(carried([]Withdrawal{w}, true)); !slices.Equal(got, []int{11}) {
		t.Fatalf("carried on one's own = %v; want the kept one", got)
	}
	if got := numbersOf(carried([]Withdrawal{w}, false)); len(got) != 0 {
		t.Fatalf("carried from another tip = %v; want none", got)
	}
}

// TestPeersNeverCallsAWithdrawalAnAppOrAnotherLoginPushedDue: the listing
// reads the reader's own withdrawals on an ended release against the forge's
// record of pushes as the act does, so one an app or another account pushed
// is never "re-armed by your next act"; and once an act has given that
// verdict for the tip, neither the next act nor the listing asks the forge
// again.
func TestPeersNeverCallsAWithdrawalAnAppOrAnotherLoginPushedDue(t *testing.T) {
	for pusher, field := range map[string]func(ReleaseWithdrawals) []Withdrawal{
		"app": func(r ReleaseWithdrawals) []Withdrawal { return r.AppPushed },
		"mal": func(r ReleaseWithdrawals) []Withdrawal { return r.Unrecorded },
	} {
		w, f, octo := releasedWithNothingArmed(t)
		record, _ := w.forgedOver("rel-v0.16.0")
		f.pusherOf[record] = pusher
		if _, err := Fetch(octo.dir, octo.opts()); err != nil {
			t.Fatal(err)
		}
		listing := func() ReleaseWithdrawals {
			snap, err := View(octo.dir, w.clock())
			if err != nil {
				t.Fatal(err)
			}
			got := snap.ReleaseWithdrawals(octo.dir, "octo", w.clock())
			if len(got) != 1 {
				t.Fatalf("%s: listing = %+v; want rel-v0.16.0", pusher, got)
			}
			return got[0]
		}
		// The listing before any act: asked once, and not due.
		got := listing()
		if len(got.Due) != 0 || len(field(got)) != 1 || strings.Contains(strings.Join(got.Lines(), "\n"), "your next abcd act") {
			t.Fatalf("%s: listing = %+v %q; want #11 not due", pusher, got, got.Lines())
		}
		asks := f.pushAsks
		// The act and the listing after it read the verdict kept for the tip.
		if _, err := octo.reserve("iss-8", "o.go"); err != nil {
			t.Fatal(err)
		}
		if _, err := octo.do(Request{Record: "iss-8", Release: true}); err != nil {
			t.Fatal(err)
		}
		got = listing()
		if len(got.Due) != 0 || len(field(got)) != 1 || f.pushAsks != asks || len(f.mutations()) != 0 {
			t.Fatalf("%s: listing = %+v, asks %d then %d, mutations %v; want the kept verdict and no ask", pusher, got, asks, f.pushAsks, f.mutations())
		}
	}
}

// TestAVerdictKeptForATipIsNotReadForAnother: the kept verdicts are the
// tip's they were given at, and only the ones that leave a withdrawal alone
// are kept.
func TestAVerdictKeptForATipIsNotReadForAnother(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, tierRelDir), 0o700); err != nil {
		t.Fatal(err)
	}
	k := readVerdicts(root, "rel-v0.16.0", headA)
	k.keep(headB, "Octo", pushedByApp)
	k.keep(headC, "octo", pushedByThem)
	k.keep(headC, "quinn", pushNotVerified)
	k.save()
	again := readVerdicts(root, "rel-v0.16.0", headA)
	if v, ok := again.of(headB, "octo"); !ok || v != pushedByApp {
		t.Fatalf("kept verdict = %v, %v; want the app's", v, ok)
	}
	if _, ok := again.of(headC, "octo"); ok {
		t.Fatal("a verdict that acts was kept")
	}
	if _, ok := again.of(headC, "quinn"); ok {
		t.Fatal("a verdict that can change was kept")
	}
	if _, ok := readVerdicts(root, "rel-v0.16.0", headC).of(headB, "octo"); ok {
		t.Fatal("a verdict kept for one tip was read for another")
	}
}

// TestAReArmTheForgeDidNotAnswerIsNotCountedAsARefusal: a re-arm that fails
// on the way to the forge (a 5xx, the network) is tried again on later acts
// without counting toward maxRearmRefusals, and is re-armed once the forge
// answers.
func TestAReArmTheForgeDidNotAnswerIsNotCountedAsARefusal(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	f.refuse["enqueuePullRequest #12"] = "exit status 1 (HTTP 502: Bad Gateway (https://api.github.com/graphql))"
	if _, err := p.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxRearmRefusals+2; i++ {
		w.set(w.clock().Add(time.Minute))
		res, err := p.reserve("iss-3", "a.go")
		if err != nil {
			t.Fatalf("act %d: %v", i, err)
		}
		if len(res.Settles) != 1 || len(res.Settles[0].Failures) != 1 || !strings.Contains(res.Settles[0].Failures[0].Reason, "not counted as a refusal") {
			t.Fatalf("act %d: settles = %+v; want #12 named as not answered", i, res.Settles)
		}
		if _, err := p.do(Request{Record: "iss-3", Release: true}); err != nil {
			t.Fatal(err)
		}
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 12) != WithdrawnPending {
		t.Fatalf("#12 recorded %q after %d failures the forge did not answer; want still withdrawn", stateOf(ws, 12), maxRearmRefusals+2)
	}
	delete(f.refuse, "enqueuePullRequest #12")
	if _, err := p.reserve("iss-4", "b.go"); err != nil {
		t.Fatal(err)
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 12) != WithdrawnRearmed || !f.pr(12).queued {
		t.Fatalf("#12 recorded %q, %+v; want re-armed once the forge answers", stateOf(ws, 12), f.pr(12))
	}
}

// TestForgeRefusedIsOnlyTheForgesAnswer: a refusal the forge's answer
// carries counts; a request that did not reach it, or was not answered,
// does not.
func TestForgeRefusedIsOnlyTheForgesAnswer(t *testing.T) {
	for err, want := range map[error]bool{
		answerError("Pull request is not mergeable"):                                  true,
		errors.New("exit status 1 (GraphQL: Pull request is in unstable status (x))"): true,
		errors.New("exit status 1 (HTTP 502: Bad Gateway)"):                           false,
		errors.New("dial tcp: lookup api.github.com: no such host"):                   false,
		fmt.Errorf("%w within 30s", github.ErrTimeout):                                false,
	} {
		if got := forgeRefused(err); got != want {
			t.Errorf("forgeRefused(%q) = %v; want %v", err, got, want)
		}
	}
}

// TestAnAdmittedMergeFoundArmedAtANewHeadCountsAsDone: the forge keeps an
// arming across pushes to the head, so an admitted pull request its owner
// pushed to during the cut is still armed after it: the settle counts it
// done, touches nothing, and never names it for its owner to re-arm.
func TestAnAdmittedMergeFoundArmedAtANewHeadCountsAsDone(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.do(Request{Record: "rel-v0.16.0", Admit: 11}); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	f.prs[11].head = headB // its owner pushed; auto-merge stays on
	f.mu.Unlock()
	f.reset()
	res, err := p.do(Request{Record: "rel-v0.16.0", Release: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range f.mutations() {
		if strings.HasSuffix(m, "#11") {
			t.Fatalf("mutations = %v; want #11 untouched", f.mutations())
		}
	}
	for _, s := range res.Settles {
		for _, wd := range s.Settled {
			if wd.Number == 11 {
				t.Fatalf("settled %+v; want #11 not named", s.Settled)
			}
		}
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != WithdrawnFoundArmed {
		t.Fatalf("#11 recorded %q; want armed", stateOf(ws, 11))
	}
	snap, err := View(p.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	for _, rw := range snap.ReleaseWithdrawals(p.dir, "octo", w.clock()) {
		if slices.Contains(numbersOf(rw.Skipped), 11) {
			t.Fatalf("listing skips #11 for its owner: %+v", rw)
		}
	}
}

// TestAnAdmissionTheRecordDoesNotSettleIsRefused: --admit arms under the
// cutter's account, so a record of pushes that has not caught up, or an
// app's push of the commit that recorded the withdrawal, refuses it with
// nothing armed.
func TestAnAdmissionTheRecordDoesNotSettleIsRefused(t *testing.T) {
	for name, c := range map[string]struct {
		set  func(f *fakeForge, record string)
		want string
	}{
		"behind": {func(f *fakeForge, _ string) { f.lag = 1 }, "run --admit 11 again in a minute"},
		"app":    {func(f *fakeForge, record string) { f.pusherOf[record] = "app" }, "an app pushed the commit that recorded its withdrawal"},
	} {
		w := newWorld(t)
		f := newFakeForge(t, map[int]*fakePR{11: {id: "PR_eleven", head: headA, method: "SQUASH"}})
		octo := w.clone("1234+octo@users.noreply.github.com", "")
		if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
			t.Fatal(err)
		}
		c.set(f, w.remoteTip("rel-v0.16.0"))
		tip := w.remoteTip("rel-v0.16.0")
		f.reset()
		_, err := octo.do(Request{Record: "rel-v0.16.0", Admit: 11})
		if err == nil || !strings.Contains(err.Error(), c.want) || len(f.mutations()) != 0 || w.remoteTip("rel-v0.16.0") != tip {
			t.Fatalf("%s: admit = %v, mutations %v; want refused with %q and nothing pushed or armed", name, err, f.mutations(), c.want)
		}
	}
}

// TestTheTagSweepClaimsTheEndBeforeItReArms: a release its tag ended is
// claimed with a tombstone over its tip, every withdrawal as it stood,
// before anything is re-armed; a second tombstone then records the re-arm.
func TestTheTagSweepClaimsTheEndBeforeItReArms(t *testing.T) {
	w := newWorld(t)
	f := newFakeForge(t, map[int]*fakePR{11: {id: "PR_eleven", head: headA, method: "SQUASH"}})
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	reserved := w.remoteTip("rel-v0.16.0")
	w.tagOrigin("v0.16.0")
	f.reset()
	if _, err := p.reserve("iss-8", "o.go"); err != nil {
		t.Fatal(err)
	}
	final := w.remoteTip("rel-v0.16.0")
	claim := w.parentOf(final)
	if w.parentOf(claim) != reserved {
		t.Fatalf("the settle's tombstone is not two commits over the reservation: %s -> %s -> %s", reserved[:8], claim[:8], final[:8])
	}
	cm, why := parseMessage("rel-v0.16.0", w.remoteMessage(claim))
	if why != "" || !cm.released() || stateOf(cm.Withdrawn, 11) != WithdrawnPending {
		t.Fatalf("the claim reads %+v (%q); want a tombstone recording #11 still withdrawn", cm, why)
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != WithdrawnRearmed {
		t.Fatalf("#11 recorded %q; want re-armed", stateOf(ws, 11))
	}
}

// settleAfter runs octo's next act and returns its one settle of
// rel-v0.16.0, failing unless there is exactly one.
func settleAfter(t *testing.T, octo *participant) Settle {
	t.Helper()
	res, err := octo.reserve("iss-8", "o.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Settles) != 1 {
		t.Fatalf("settles = %+v; want one", res.Settles)
	}
	return res.Settles[0]
}

// pushIndex is the index of the push to after in ps, -1 when none.
func pushIndex(ps []Push, after string) int {
	return slices.IndexFunc(ps, func(p Push) bool { return p.After == after })
}

// TestAPushCarryingSeveralCommitsIsPlacedOnEachOfThem: one push that moved
// the branch over two commits is placed on both, so another login's push of
// a withdrawal record together with the tombstone over it vouches for
// nothing.
func TestAPushCarryingSeveralCommitsIsPlacedOnEachOfThem(t *testing.T) {
	w, f, octo := releasedWithNothingArmed(t)
	record, tomb := w.forgedOver("rel-v0.16.0")
	f.edit = func(ps []Push) []Push {
		i, j := pushIndex(ps, tomb), pushIndex(ps, record)
		one := Push{Before: ps[j].Before, After: tomb, Pusher: "mal"}
		return slices.Concat(ps[:i], []Push{one}, ps[j+1:])
	}
	s := settleAfter(t, octo)
	if len(f.mutations()) != 0 || !slices.Equal(numbersOf(s.Unrecorded), []int{11}) {
		t.Fatalf("settle = %+v, mutations %v; want #11 unrecorded", s, f.mutations())
	}
}

// TestAForcedPushIsPlacedOnEveryCommitBeneathIt: a push whose earlier tip is
// not on the branch's line (a forced one) may have carried any commit
// beneath its own, so another login's forced push leaves every withdrawal
// beneath it unrecorded, never re-armed.
func TestAForcedPushIsPlacedOnEveryCommitBeneathIt(t *testing.T) {
	w, f, octo := releasedWithNothingArmed(t)
	_, tomb := w.forgedOver("rel-v0.16.0") // octo's own pushes
	m, _ := w.tipWithdrawals("rel-v0.16.0")
	forced := w.pushChild("rel-v0.16.0", tomb, m.text()+"\n", t0)
	if _, err := Fetch(octo.dir, octo.opts()); err != nil {
		t.Fatal(err)
	}
	// A commit over the tombstone that the forced push moved the branch
	// away from, held here.
	tree := strings.TrimSpace(runGit(t, octo.dir, "hash-object", "-t", "tree", "/dev/null"))
	away := strings.TrimSpace(runGit(t, octo.dir, "commit-tree", tree, "-p", tomb, "-m", "away"))
	f.edit = func(ps []Push) []Push {
		i := pushIndex(ps, forced)
		return slices.Concat(ps[:i], []Push{{Before: away, After: forced, Pusher: "mal"}, {Before: tomb, After: away, Pusher: "mal"}}, ps[i+1:])
	}
	s := settleAfter(t, octo)
	if len(f.mutations()) != 0 || !slices.Equal(numbersOf(s.Unrecorded), []int{11}) {
		t.Fatalf("settle = %+v, mutations %v; want #11 unrecorded", s, f.mutations())
	}
}

// TestAPushIsPlacedOnlyOnTheCommitsAboveItsEarlierTip: another login's push
// over a branch octo's own pushes recorded carries only what it added, so
// octo's withdrawal beneath it is re-armed.
func TestAPushIsPlacedOnlyOnTheCommitsAboveItsEarlierTip(t *testing.T) {
	w, f, octo := releasedWithNothingArmed(t)
	_, tomb := w.forgedOver("rel-v0.16.0") // octo's own pushes
	m, _ := w.tipWithdrawals("rel-v0.16.0")
	later := w.pushChild("rel-v0.16.0", tomb, m.text()+"\n", t0)
	f.pusherOf[later] = "mal"
	settleAfter(t, octo)
	if got := f.mutations(); !slices.Equal(got, []string{"enablePullRequestAutoMerge #11"}) {
		t.Fatalf("mutations = %v; want #11 re-armed", got)
	}
}

// TestAPushThisCheckoutCannotPlaceIsPlacedOnTheWholeLine: another login's
// push to a commit this checkout does not hold may have carried any commit
// it read, so its pusher is placed on all of them and the withdrawal reads
// unrecorded, never merely not yet verified.
func TestAPushThisCheckoutCannotPlaceIsPlacedOnTheWholeLine(t *testing.T) {
	w, f, octo := releasedWithNothingArmed(t)
	_, tomb := w.forgedOver("rel-v0.16.0") // octo's own pushes
	m, _ := w.tipWithdrawals("rel-v0.16.0")
	back := w.pushChild("rel-v0.16.0", tomb, m.text()+"\n", t0) // octo's, as the record reads by default
	// Two pushes of mal's this checkout never saw, then octo's back to the
	// line: the newer of mal's has neither end on the line.
	unseen, unseen2 := strings.Repeat("ab", 20), strings.Repeat("cd", 20)
	f.edit = func(ps []Push) []Push {
		i := pushIndex(ps, back)
		return slices.Concat(ps[:i], []Push{{Before: unseen, After: back, Pusher: "octo"}, {Before: unseen2, After: unseen, Pusher: "mal"},
			{Before: tomb, After: unseen2, Pusher: "mal"}}, ps[i+1:])
	}
	s := settleAfter(t, octo)
	if len(f.mutations()) != 0 || !slices.Equal(numbersOf(s.Unrecorded), []int{11}) {
		t.Fatalf("settle = %+v, mutations %v; want #11 unrecorded", s, f.mutations())
	}
}

// TestARecordWithAPushMissingSettlesNothing: the forge's activity record
// passes over an entry without a pusher; a record whose pushes do not run
// unbroken down to the push that made the branch says nothing of who carried
// every commit, so the withdrawal is not yet verified, never re-armed. The
// case: the branch rewound under octo's push, which then carried a commit
// whose own push the record dropped.
func TestARecordWithAPushMissingSettlesNothing(t *testing.T) {
	w, f, octo := releasedWithNothingArmed(t)
	before := w.remoteTip("rel-v0.16.0")
	record, tomb := w.forgedOver("rel-v0.16.0")
	f.edit = func(ps []Push) []Push {
		i, j := pushIndex(ps, tomb), pushIndex(ps, record)
		return slices.Concat(ps[:i], []Push{{Before: before, After: tomb, Pusher: "octo"}, {Before: record, After: before, Pusher: "mal"}}, ps[j+1:])
	}
	s := settleAfter(t, octo)
	if len(f.mutations()) != 0 || !slices.Equal(numbersOf(s.Unverified), []int{11}) || !strings.Contains(s.UnverifiedWhy, "a push missing") {
		t.Fatalf("settle = %+v, mutations %v; want #11 not yet verified", s, f.mutations())
	}
	// Run unbroken, the same pushes settle it (another account's rewind
	// carried nothing it recorded).
	f.edit = nil
	settleAfter(t, octo)
	if got := f.mutations(); !slices.Equal(got, []string{"enablePullRequestAutoMerge #11"}) {
		t.Fatalf("mutations = %v; want #11 re-armed", got)
	}
}

// TestUnbrokenNeedsEveryLinkDownToTheBranchsFirstPush and the verdict's
// backstop: a commit the record places no push on is not verified.
func TestUnbrokenNeedsEveryLinkDownToTheBranchsFirstPush(t *testing.T) {
	a, b, c := headA, headB, headC
	for name, c := range map[string]struct {
		ps   []Push
		want bool
	}{
		"whole":            {[]Push{{Before: b, After: c}, {Before: a, After: b}, {Before: zeroCommit, After: a}}, true},
		"made anew":        {[]Push{{Before: zeroCommit, After: c}, {Before: a, After: b}, {Before: zeroCommit, After: a}}, true},
		"a link missing":   {[]Push{{Before: b, After: c}, {Before: zeroCommit, After: a}}, false},
		"no first push":    {[]Push{{Before: b, After: c}, {Before: a, After: b}}, false},
		"nothing recorded": {nil, true},
	} {
		if got := unbroken(c.ps); got != c.want {
			t.Errorf("%s: unbroken = %v; want %v", name, got, c.want)
		}
	}
	rec := pushRecord{by: map[string][]Push{headA: {{After: headA, Pusher: "octo"}}}}
	if got := rec.verdict(headB, "octo"); got != pushNotVerified {
		t.Fatalf("a commit no push is placed on reads %v; want not verified", got)
	}
	if got := rec.verdict(headA, "octo"); got != pushedByThem {
		t.Fatalf("octo's own push reads %v", got)
	}
}

// TestARecordOfNoPushesSettlesNothing: an empty record of pushes, read in
// full, vouches for no commit.
func TestARecordOfNoPushesSettlesNothing(t *testing.T) {
	w, f, octo := releasedWithNothingArmed(t)
	w.forgedOver("rel-v0.16.0")
	f.edit = func([]Push) []Push { return nil }
	s := settleAfter(t, octo)
	if len(f.mutations()) != 0 || !slices.Equal(numbersOf(s.Unverified), []int{11}) {
		t.Fatalf("settle = %+v, mutations %v; want #11 not yet verified", s, f.mutations())
	}
}

// TestTheLaunchReadingExemptsNoAdmissionPastTheWalk: an admission carried by
// every commit the walk reads lies past it, unread, so it is not exempt.
func TestTheLaunchReadingExemptsNoAdmissionPastTheWalk(t *testing.T) {
	w := newWorld(t)
	armedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.do(Request{Record: "rel-v0.16.0", Admit: 11}); err != nil {
		t.Fatal(err)
	}
	m, _ := w.tipWithdrawals("rel-v0.16.0")
	w.pushChildren("rel-v0.16.0", w.remoteTip("rel-v0.16.0"), m, maxChain)
	if _, err := Fetch(p.dir, p.opts()); err != nil {
		t.Fatal(err)
	}
	a, err := ArmedMergesForCut(p.dir, p.opts())
	if err != nil || len(a.Admitted) != 0 || len(a.Unverified) != 1 || !strings.Contains(a.UnverifiedWhy, "lies past the") {
		t.Fatalf("reading = %+v, %v; want #11 not exempt", a, err)
	}
}

// pushChildren pushes n renewals of m, one over the other from parent, in
// one push.
func (w *world) pushChildren(id, parent string, m message, n int) string {
	w.t.Helper()
	dir := filepath.Join(w.t.TempDir(), "children")
	w.seed.Git("clone", "-q", w.bare, dir)
	runGit(w.t, dir, "fetch", "-q", "origin", RefPrefix+id)
	tree := strings.TrimSpace(runGit(w.t, dir, "hash-object", "-w", "-t", "tree", "/dev/null"))
	for i := 0; i < n; i++ {
		path := filepath.Join(w.t.TempDir(), "msg")
		if err := os.WriteFile(path, []byte(m.text()), 0o600); err != nil {
			w.t.Fatal(err)
		}
		parent = strings.TrimSpace(runGit(w.t, dir, "-c", "user.name=Pat Example", "-c", "user.email=1234+octo@users.noreply.github.com",
			"commit-tree", tree, "-p", parent, "-F", path))
	}
	runGit(w.t, dir, "push", "-q", "origin", parent+":"+RefPrefix+id)
	return parent
}

// forgetVerdicts removes the verdicts kept in the checkout at root, for a
// test that has the forge's record say otherwise of the same commits.
func forgetVerdicts(t *testing.T, root string) {
	t.Helper()
	if err := os.RemoveAll(filepath.Join(root, filepath.FromSlash(verdictsDirRel))); err != nil {
		t.Fatal(err)
	}
}
