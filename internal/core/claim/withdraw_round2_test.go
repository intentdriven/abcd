package claim

// The release hold's second round (iss-2610222014550692): the end trigger
// (dcn-2610230328220298), the re-arm bound to its node id, the withdrawer
// vouched for by the branch's history, the holder checked against gh's
// account, one release at a time, the release pull request left alone, the
// queue-less re-arm (dcn-2610230328230096), the withdrawals an older binary
// dropped (dcn-2610230328242929), and the bounds.

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/lease"
	"github.com/intentdriven/abcd/internal/gittest"
)

// numbersOf is the pull request numbers of ws.
func numbersOf(ws []Withdrawal) []int {
	out := make([]int, len(ws))
	for i, w := range ws {
		out[i] = w.Number
	}
	return out
}

// pushChild pushes msg as a commit over parent on id's reservation branch,
// written outside abcd (an older binary, or a hand).
func (w *world) pushChild(id, parent, msg string, at time.Time) string {
	w.t.Helper()
	dir := filepath.Join(w.t.TempDir(), "child")
	w.seed.Git("clone", "-q", w.bare, dir)
	runGit(w.t, dir, "fetch", "-q", "origin", RefPrefix+id)
	path := filepath.Join(w.t.TempDir(), "msg")
	if err := os.WriteFile(path, []byte(msg), 0o600); err != nil {
		w.t.Fatal(err)
	}
	tree := runGit(w.t, dir, "hash-object", "-w", "-t", "tree", "/dev/null")
	date := "@" + strconv.FormatInt(at.Unix(), 10) + " +0000"
	cmd := exec.Command("git", "-C", dir, "-c", "user.name=Pat Example", "-c", "user.email=1234+octo@users.noreply.github.com",
		"-c", "commit.gpgsign=false", "commit-tree", tree, "-p", parent, "-F", path)
	cmd.Env = append(gittest.Env(w.t), "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
	out, err := cmd.Output()
	if err != nil {
		w.t.Fatal(err)
	}
	sha := strings.TrimSpace(string(out))
	runGit(w.t, dir, "push", "-q", "origin", sha+":"+RefPrefix+id)
	return sha
}

// TestALapsedReleaseEndsForItsHolderAndReArmsWhatItWithdrew
// (dcn-2610271926369425): a release cut whose lease lapses with no tag on
// origin ends its hold once the take-over margin has passed. Before then
// nothing is re-armed, a cut renewed in time keeps every merge withdrawn, and
// another person's copy re-arms nothing of it; past the margin, the next act
// of any copy of the cut's own person ends it with a tombstone and re-arms
// exactly what it recorded withdrawn, each with one comment saying the cut
// lapsed.
func TestALapsedReleaseEndsForItsHolderAndReArmsWhatItWithdrew(t *testing.T) {
	w := newWorld(t)
	f := commentedForge(t)
	cutter := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := cutter.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	expiry := func() time.Time {
		t.Helper()
		snap, err := View(cutter.dir, w.clock())
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range snap.Reservations {
			if r.Record == "rel-v0.16.0" {
				return r.EffectiveExpiry
			}
		}
		t.Fatal("no rel-v0.16.0 in the view")
		return time.Time{}
	}
	f.reset()

	// Lapsed, but inside the take-over margin: nothing is re-armed yet.
	w.set(expiry().Add(time.Minute))
	if s, err := SettleEndedReleases(cutter.dir, cutter.opts()); err != nil || len(s) != 0 {
		t.Fatalf("settles inside the margin = %+v, %v; want none", s, err)
	}
	snap, err := View(cutter.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	if got := snap.ReleaseWithdrawals(cutter.dir, cutter.self(), w.clock()); len(got) != 1 || !got[0].Lapsed ||
		!strings.Contains(strings.Join(got[0].Lines(), "\n"), "the next act of a copy of octo's ends it and re-arms what it withdrew, while the cut's tip is their own newest push") {
		t.Fatalf("listing inside the margin = %+v; want the lapse and when it ends", got)
	}
	// The cut renews in time, so its merges stay withdrawn past the old margin.
	if res, err := cutter.do(Request{Record: "rel-v0.16.0"}); err != nil || res.Act != ActRenewed {
		t.Fatalf("renewal = %+v, %v", res, err)
	}
	renewed := expiry()
	laptop := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := laptop.reserve("iss-9", "l.go"); err != nil {
		t.Fatal(err)
	}
	if got := f.mutations(); len(got) != 0 {
		t.Fatalf("mutations while the cut is live = %v; want none", got)
	}

	// Past the renewed lease and its margin, another person's act ends nothing.
	w.set(renewed.Add(TakeoverMargin + time.Hour))
	other := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := other.reserve("iss-6", "q.go"); err != nil {
		t.Fatal(err)
	}
	if got := f.mutations(); len(got) != 0 {
		t.Fatalf("mutations by another person = %v; want none", got)
	}
	if m, _ := w.tipWithdrawals("rel-v0.16.0"); m.released() {
		t.Fatal("another person's act ended octo's lapsed cut")
	}

	// The cut's own person's other copy ends it and re-arms what it withdrew.
	if _, err := laptop.reserve("iss-8", "k.go"); err != nil {
		t.Fatal(err)
	}
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	if !m.released() || stateOf(ws, 11) != WithdrawnRearmed || stateOf(ws, 12) != WithdrawnRearmed || stateOf(ws, 13) != WithdrawnRearmed {
		t.Fatalf("tip = %+v %+v; want a tombstone with #11-#13 re-armed", m, ws)
	}
	var lapsed []string
	for _, c := range f.posted() {
		if strings.Contains(c, "is re-armed;") {
			lapsed = append(lapsed, c)
		}
	}
	want := "#11 abcd release hold: the merge of #11 is re-armed; the cut of release v0.16.0 lapsed with no tag on origin, so its hold has ended."
	if len(lapsed) != 3 || lapsed[0] != want {
		t.Fatalf("re-arm comments = %q; want three, the first %q", lapsed, want)
	}

	// The cutting copy's next act finds nothing more to do.
	f.reset()
	if _, err := cutter.reserve("iss-7", "c.go"); err != nil {
		t.Fatal(err)
	}
	if got := f.mutations(); len(got) != 0 {
		t.Fatalf("mutations after it was settled = %v; want none", got)
	}
}

// TestALapseAnotherLoginForgedEndsNothing (dcn-2610271926369425 review): a
// commit another login pushes over a live cut, naming its holder with a lease
// long past, is no lapse of theirs: the tip is not the holder's own newest
// push, so their next act re-arms nothing and pushes no tombstone.
func TestALapseAnotherLoginForgedEndsNothing(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	m, _ := w.tipWithdrawals("rel-v0.16.0")
	past := t0.Add(-48 * time.Hour)
	m.ClaimedAt, m.ExpiresAt = past, past.Add(time.Hour)
	forged := w.pushChild("rel-v0.16.0", w.remoteTip("rel-v0.16.0"), m.text(), past)
	f.pusherOf[forged] = "mal"
	f.reset()
	w.set(t0.Add(time.Minute))
	if _, err := octo.reserve("iss-8", "o.go"); err != nil {
		t.Fatal(err)
	}
	if got := f.mutations(); len(got) != 0 {
		t.Fatalf("mutations = %v; want nothing re-armed on a lapse another login wrote", got)
	}
	if tip, _ := w.tipWithdrawals("rel-v0.16.0"); tip.released() || w.remoteTip("rel-v0.16.0") != forged {
		t.Fatal("octo's act pushed a tombstone over the forged lapse")
	}
}

// TestALapseRePushedByAnotherLoginEndsNothingAndSaysSo: another login
// force-pushes the branch back to the holder's own lapsed reservation, so the
// forge places two accounts' pushes on it. The holder's next act re-arms
// nothing, pushes no tombstone, and does not report the hold as ended.
func TestALapseRePushedByAnotherLoginEndsNothingAndSaysSo(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	lapsedTip := w.remoteTip("rel-v0.16.0")
	m, _ := w.tipWithdrawals("rel-v0.16.0")
	f.edit = func(ps []Push) []Push {
		return append([]Push{{Before: headC, After: lapsedTip, Pusher: "mal"}}, ps...)
	}
	f.reset()
	w.set(m.ExpiresAt.Add(lease.Max + TakeoverMargin))
	s := settleAfter(t, octo)
	if got := f.mutations(); len(got) != 0 {
		t.Fatalf("mutations = %v; want nothing re-armed on a lapse another login re-pushed", got)
	}
	if w.remoteTip("rel-v0.16.0") != lapsedTip {
		t.Fatal("octo's act pushed a tombstone over a lapse another login re-pushed")
	}
	if s.LapsedAt != nil || strings.Contains(strings.Join(s.Notes(), "\n"), "hold has ended") {
		t.Fatalf("settle = %+v, notes %q; want the hold not reported as ended", s, s.Notes())
	}
}

// TestReleasePushesTheTombstoneBeforeItReArms: at the first re-arm the forge
// sees, origin's tip is already the tombstone, still recording every one
// withdrawn: the push is the compare-and-swap that decides which copy acts.
func TestReleasePushesTheTombstoneBeforeItReArms(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	cutter := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := cutter.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	var tombstoned, pending bool
	seen := false
	inner := releaseForge.graphql
	releaseForge.graphql = func(root, doc string, vars map[string]string, ints map[string]int) ([]byte, error) {
		if strings.HasPrefix(doc, "mutation") && !seen {
			seen = true
			m, ws := w.tipWithdrawals("rel-v0.16.0")
			tombstoned, pending = m.released(), len(ws) == 3 && ws[0].Pending()
		}
		return inner(root, doc, vars, ints)
	}
	f.reset()
	res, err := cutter.do(Request{Record: "rel-v0.16.0", Release: true})
	if err != nil {
		t.Fatal(err)
	}
	if !tombstoned || !pending {
		t.Fatalf("at the first re-arm origin's tip was tombstone=%v, withdrawals pending=%v; want both", tombstoned, pending)
	}
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	if !m.released() || stateOf(ws, 11) != WithdrawnRearmed || res.Commit != w.remoteTip("rel-v0.16.0") || res.Settles[0].Commit != res.Commit {
		t.Fatalf("tip = %+v %+v, result commit %s; want the second tombstone recording the re-arms", m, ws, res.Commit)
	}
}

// TestASweepThatLosesTheTombstoneRaceReArmsNothing: the sweep's first push is
// the compare-and-swap; when another copy moved the tip first, it re-arms
// nothing.
func TestASweepThatLosesTheTombstoneRaceReArmsNothing(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	cutter := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := cutter.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	w.tagOrigin("v0.16.0")
	laptop := w.clone("1234+octo@users.noreply.github.com", "")
	for _, c := range []*participant{laptop, cutter} {
		if _, err := Fetch(c.dir, c.opts()); err != nil {
			t.Fatal(err)
		}
	}
	// Another copy settles first: its tombstone lands over the tip the
	// laptop read.
	if s, err := SettleEndedReleases(cutter.dir, cutter.opts()); err != nil || len(s) != 1 || s[0].Commit == "" {
		t.Fatalf("the first copy's settle = %+v, %v", s, err)
	}
	f.reset()
	tree, err := emptyTree(laptop.dir)
	if err != nil {
		t.Fatal(err)
	}
	h, _ := resolveHolder(laptop.dir, identity{Name: "Pat Example", Email: "1234+octo@users.noreply.github.com"}, laptop.opts(), false)
	a := actor{root: laptop.dir, who: identity{Name: "Pat Example", Email: "1234+octo@users.noreply.github.com"}, holder: h.Login,
		copy: ownCopyKey(laptop.dir), now: w.clock(), label: "abcd:v0.0.0"}
	// The forge's record ends at the other copy's tombstone, not at the tip
	// the laptop read, so nothing is verified and nothing is done.
	if s, _ := settleEnded(a, tree, "rel-v0.16.0", false); s.Commit != "" || len(s.Settled) != 0 || len(f.mutations()) != 0 {
		t.Fatalf("settle = %+v, mutations %v; want nothing done by the copy that lost", s, f.mutations())
	}
}

// TestAWithdrawalNoReservationOfItsWithdrawerRecordedIsNeverActedOn: a
// tombstone of another holder's, naming octo as the withdrawer of a pull
// request under another's node id, touches nothing on the forge: octo pushed
// nothing to that branch, so octo's record there holds nothing to act on.
func TestAWithdrawalNoReservationOfItsWithdrawerRecordedIsNeverActedOn(t *testing.T) {
	w := newWorld(t)
	f := newFakeForge(t, map[int]*fakePR{
		11: {id: "PR_eleven", head: headA},
		14: {id: "PR_fourteen", head: headA},
	})
	other := message{Record: "rel-v0.17.0", Holder: "mal", Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: t0, ExpiresAt: t0, ReleasedAt: t0,
		Withdrawn: []Withdrawal{{Number: 11, ID: "PR_fourteen", Form: FormAutoMerge, Head: headA, By: "octo", State: WithdrawnPending}}}
	w.pushHostile("rel-v0.17.0", other.text(), t0)
	f.reset()
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	res, err := octo.reserve("iss-8", "o.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(f.calls) != 0 || f.pr(14).method != "" {
		t.Fatalf("calls = %v, #14 = %+v; want the forge untouched", f.calls, f.pr(14))
	}
	if len(res.Settles) != 0 {
		t.Fatalf("settles = %+v; want nothing of octo's on a branch octo never pushed to", res.Settles)
	}
}

// TestAReArmIsBoundToItsNodeID: a re-arm reads the pull request by number and
// changes it by node id, so a record whose node id is not the forge's for the
// number is skipped as a mismatch, and no mutation is sent.
func TestAReArmIsBoundToItsNodeID(t *testing.T) {
	newWorld(t)
	f := newFakeForge(t, map[int]*fakePR{
		11: {id: "PR_eleven", head: headA},
		14: {id: "PR_fourteen", head: headA},
	})
	state, err := releaseForge.rearm("", "acme", "widgets", Withdrawal{Number: 11, ID: "PR_fourteen", Form: FormAutoMerge, Head: headA, By: "octo", State: WithdrawnPending})
	if err != nil || state != WithdrawnMismatch || len(f.mutations()) != 0 || f.pr(14).method != "" {
		t.Fatalf("rearm = %q, %v, mutations %v; want a mismatch and nothing sent", state, err, f.mutations())
	}
	if !strings.Contains(SettledLine("rel-v0.16.0", Withdrawal{Number: 11, State: WithdrawnMismatch}), "node id recorded for it is not the forge's") {
		t.Fatal("the mismatch is not named")
	}
}

// TestGhActingAsAnotherAccountTouchesNothing: when gh's token acts as another
// account than the reservation's holder, a release reserved anew withdraws
// nothing (named), --admit and --release are refused, and nothing reaches the
// forge's mutations.
func TestGhActingAsAnotherAccountTouchesNothing(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	f.logins[p.dir] = "quinn"
	res, err := p.do(Request{Record: "rel-v0.16.0"})
	if err != nil || res.Act != ActReserved {
		t.Fatalf("reserve = %+v, %v", res, err)
	}
	if len(res.Withdrew) != 0 || !strings.Contains(res.ForgeUnchecked, "gh acts as quinn, but the reservation's holder is octo") || len(f.mutations()) != 0 {
		t.Fatalf("result = %+v, mutations %v; want nothing withdrawn and the mismatch named", res, f.mutations())
	}
	// Withdrawn under the right account, then gh switches.
	delete(f.logins, p.dir)
	if _, err := p.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.do(Request{Record: "rel-v0.16.1"}); err != nil {
		t.Fatal(err)
	}
	f.reset()
	f.logins[p.dir] = "quinn"
	if _, err := p.do(Request{Record: "rel-v0.16.1", Admit: 11}); err == nil || !strings.Contains(err.Error(), "gh acts as quinn") {
		t.Fatalf("admit = %v; want refused on the mismatch", err)
	}
	if _, err := p.do(Request{Record: "rel-v0.16.1", Release: true}); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "was not released: gh acts as quinn") {
		t.Fatalf("release = %v; want refused on the mismatch", err)
	}
	if got := f.mutations(); len(got) != 0 {
		t.Fatalf("mutations = %v; want none under another account", got)
	}
	if m, _ := w.tipWithdrawals("rel-v0.16.1"); m.released() {
		t.Fatal("the refused release pushed a tombstone")
	}
}

// TestOneReleaseIsCutAtATime: a second release reserved while another is
// live backs off, naming the first; once that one ends, it is reserved.
func TestOneReleaseIsCutAtATime(t *testing.T) {
	w := newWorld(t)
	armedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	quinn := w.clone("99+quinn@users.noreply.github.com", "")
	for _, p := range []*participant{quinn, octo} {
		if _, err := p.do(Request{Record: "rel-v0.17.0"}); !errors.Is(err, ErrBackOff) || !strings.Contains(err.Error(), "rel-v0.16.0 is being cut by octo") {
			t.Fatalf("second release = %v; want a back-off naming the first", err)
		}
	}
	if _, err := octo.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := quinn.do(Request{Record: "rel-v0.17.0"}); err != nil {
		t.Fatalf("after the first ended: %v", err)
	}
}

// TestTheReleasePullRequestIsNeverWithdrawn: a pull request from the branch
// this checkout is on, in this repository, is the release pull request: a
// late re-reserve leaves it armed and unrecorded.
func TestTheReleasePullRequestIsNeverWithdrawn(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	p.git("checkout", "-q", "-b", "b12")
	res, err := p.do(Request{Record: "rel-v0.16.0"})
	if err != nil {
		t.Fatal(err)
	}
	if !f.pr(12).queued || slices.Contains(numbersOf(res.Withdrawals), 12) || len(res.Withdrew) != 2 {
		t.Fatalf("withdrew %+v, #12 = %+v; want the release pull request left queued and unrecorded", res.Withdrew, f.pr(12))
	}
}

// TestAReArmTheForgeFindsCleanNeedsAManualMerge: on a repository with no
// merge queue, a pull request whose checks passed while it was withdrawn
// cannot be armed again ("clean status"); it settles as needing a manual
// merge, named, abcd merges nothing, and no act tries it again.
func TestAReArmTheForgeFindsCleanNeedsAManualMerge(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	cutter := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := cutter.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	f.answer["enablePullRequestAutoMerge #11"] = "Pull request Pull request is in clean status"
	f.reset()
	if _, err := cutter.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	_, ws := w.tipWithdrawals("rel-v0.16.0")
	if stateOf(ws, 11) != WithdrawnManual || stateOf(ws, 12) != WithdrawnRearmed {
		t.Fatalf("tombstone records %+v; want #11 needing a manual merge", ws)
	}
	for _, c := range f.mutations() {
		if strings.HasPrefix(c, "mergePullRequest") {
			t.Fatalf("abcd merged a pull request: %v", f.mutations())
		}
	}
	snap, err := View(cutter.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	if got := snap.ReleaseWithdrawals(cutter.dir, cutter.self(), w.clock()); len(got) != 1 || len(got[0].Skipped) != 1 ||
		!strings.Contains(got[0].Lines()[0], "#11 was not re-armed after rel-v0.16.0: it can merge now and there is no merge queue to put it in, so it needs a manual merge by its owner") {
		t.Fatalf("listing = %+v; want #11 named for a manual merge", got)
	}
	f.reset()
	if _, err := cutter.reserve("iss-3", "a.go"); err != nil || len(f.calls) != 0 {
		t.Fatalf("next act = %v, calls %v; want no retry", err, f.calls)
	}
}

// TestWithdrawalsAnOlderBinaryDroppedAreReportedLost: a renewal written
// without the Abcd-Withdrawn trailers, as an older abcd writes one, drops
// the record from the tip; the walk of the branch's history names each lost
// one, and a later act of the new binary does not bring them back silently.
func TestWithdrawalsAnOlderBinaryDroppedAreReportedLost(t *testing.T) {
	w := newWorld(t)
	armedForge(t)
	cutter := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := cutter.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	m, _ := w.tipWithdrawals("rel-v0.16.0")
	old := message{Record: "rel-v0.16.0", Holder: "octo", Copy: m.Copy, ClaimedAt: t0.Add(time.Hour), ExpiresAt: t0.Add(lease.Max), AssistedBy: "abcd:v0.15.1"}
	w.pushChild("rel-v0.16.0", w.remoteTip("rel-v0.16.0"), old.text(), t0.Add(time.Hour))
	w.set(t0.Add(2 * time.Hour))
	if _, err := Fetch(cutter.dir, cutter.opts()); err != nil {
		t.Fatal(err)
	}
	lost := LostReleaseWithdrawals(cutter.dir, w.clock())
	if len(lost) != 1 || lost[0].Subject != "rel-v0.16.0" || !slices.Equal(numbersOf(lost[0].Lost), []int{11, 12, 13}) ||
		!strings.Contains(lost[0].Line(), "#11, #12, #13 withdrawn for rel-v0.16.0 by octo are lost") {
		t.Fatalf("lost = %+v; want #11, #12, #13 named", lost)
	}
	// The launch check's reading names them too.
	a, err := ArmedMergesForCut(cutter.dir, cutter.opts())
	if err != nil || len(a.Lost) != 1 || len(a.Lost[0].Lost) != 3 {
		t.Fatalf("armed merges = %+v, %v; want the lost ones", a, err)
	}
	// A settled record is not lost: none on a fresh release.
	if l := lostWithdrawals(nil, true); l != nil {
		t.Fatalf("lost on nothing = %+v", l)
	}
}

// TestATipOverTheWithdrawalCapIsUnreadable: more Abcd-Withdrawn trailers than
// the cap make the tip unreadable, never a list carried forward.
func TestATipOverTheWithdrawalCapIsUnreadable(t *testing.T) {
	base := "abcd reservation: rel-v0.16.0\n\nAbcd-Reservation: rel-v0.16.0\nAbcd-Holder: octo\nAbcd-Copy: 9f2c41d07be35a6c80e1f4b2a7d39c55\n" +
		"Abcd-Claimed-At: 2026-10-03T14:00:00Z\nAbcd-Expires-At: 2026-10-03T15:00:00Z\nAbcd-Released-At: 2026-10-03T15:00:00Z\n"
	var b strings.Builder
	for n := 1; n <= maxWithdrawals+1; n++ {
		b.WriteString("Abcd-Withdrawn: " + strconv.Itoa(n) + " P queue " + headA + " o rearmed\n")
	}
	if len(base+b.String()) > MaxMessageBytes {
		t.Fatalf("the probe is %d bytes, past the message cap: shorten it", len(base+b.String()))
	}
	if _, why := parseMessage("rel-v0.16.0", base+b.String()); why != reasonWithdrawn {
		t.Fatalf("a tip with %d withdrawals read with reason %q; want %q", maxWithdrawals+1, why, reasonWithdrawn)
	}
}

// TestTombstoneFitIsMeasuredForAnyHolder: whether a withdrawal can be
// recorded does not hang on the cutter's own login and label: the tombstone
// any copy of any person writes must carry it.
func TestTombstoneFitIsMeasuredForAnyHolder(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	pr, err := prepare(p.dir, Request{Record: "rel-v0.16.0"}, p.opts())
	if err != nil {
		t.Fatal(err)
	}
	entry := func(n, idLen int) Withdrawal {
		return Withdrawal{Number: 100000000 + n, ID: strings.Repeat("A", idLen), Form: FormAutoRebase, Head: headA, By: "octo", State: WithdrawnPending}
	}
	fitsAsOcto := func(ws []Withdrawal) bool {
		m := pr.tombstone(nil)
		m.Withdrawn = ws
		return checkMessage(p.dir, m.text()) == nil
	}
	// Fill with full-length entries while octo's own tombstone has room, then
	// size the last one so octo's still fits and the longest holder's does not.
	var ws []Withdrawal
	for n := 1; fitsAsOcto(append(slices.Clone(ws), entry(n, 100))); n++ {
		ws = append(ws, entry(n, 100))
	}
	for l := 100; l >= 1; l-- {
		next := append(slices.Clone(ws), entry(len(ws)+1, l))
		if !fitsAsOcto(next) {
			continue
		}
		if pr.tombstoneFits(next) {
			t.Fatalf("a list that only fits under octo's short login reads as fitting (%d entries, last id %d long)", len(next), l)
		}
		return
	}
	// No length fits under octo's own: drop one and try again.
	ws = ws[:len(ws)-1]
	for l := 100; l >= 1; l-- {
		next := append(slices.Clone(ws), entry(len(ws)+1, 100), entry(len(ws)+2, l))
		if fitsAsOcto(next) && pr.tombstoneFits(next) {
			continue
		}
		if fitsAsOcto(next) {
			return
		}
	}
	t.Fatal("found no list that fits under octo's login and not under the longest")
}

// The mutations round 1 left alive.

// TestReArmMineSkipsAnotherPersonsEntryOnTheSameTip: rearmMine re-arms only
// the acting person's withdrawals on a tip that carries another's too.
func TestReArmMineSkipsAnotherPersonsEntryOnTheSameTip(t *testing.T) {
	w := newWorld(t)
	f := newFakeForge(t, map[int]*fakePR{11: {id: "PR_eleven", head: headA}, 12: {id: "PR_twelve", head: headB}})
	p := w.clone("1234+octo@users.noreply.github.com", "")
	ws := []Withdrawal{
		{Number: 11, ID: "PR_eleven", Form: FormAutoMerge, Head: headA, By: "octo", State: WithdrawnPending},
		{Number: 12, ID: "PR_twelve", Form: FormAutoMerge, Head: headB, By: "quinn", State: WithdrawnPending},
	}
	out, settled, _, fails, why := rearmMine(actor{root: p.dir, holder: "octo", subject: "rel-v0.16.0"}, ws)
	if why != "" || len(fails) != 0 || len(settled) != 1 || settled[0].Number != 11 || out[1].State != WithdrawnPending ||
		!slices.Equal(f.mutations(), []string{"enablePullRequestAutoMerge #11"}) {
		t.Fatalf("rearmMine = %+v %+v %+v %q, mutations %v; want only octo's #11", out, settled, fails, why, f.mutations())
	}
}

// TestAdmitIsRefusedOnALapsedOrReleasedReservation: --admit needs the
// cutting copy's live reservation.
func TestAdmitIsRefusedOnALapsedOrReleasedReservation(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	f.reset()
	w.set(t0.Add(lease.Max + time.Minute))
	if _, err := p.do(Request{Record: "rel-v0.16.0", Admit: 11}); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "your reservation on rel-v0.16.0 lapsed at") {
		t.Fatalf("admit on a lapse = %v", err)
	}
	if _, err := p.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	f.reset()
	if _, err := p.do(Request{Record: "rel-v0.16.0", Admit: 12}); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "was released at") {
		t.Fatalf("admit on a release = %v", err)
	}
	if len(f.mutations()) != 0 {
		t.Fatalf("a refused admission touched the forge: %v", f.mutations())
	}
}

// TestAdmitRecordsBeforeItReArms: the admission reaches origin before the
// forge arms the pull request, and a forge that refuses the arming puts the
// record back to withdrawn.
func TestAdmitRecordsBeforeItReArms(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	var atArm string
	inner := releaseForge.graphql
	releaseForge.graphql = func(root, doc string, vars map[string]string, ints map[string]int) ([]byte, error) {
		if strings.HasPrefix(doc, "mutation") && atArm == "" {
			_, ws := w.tipWithdrawals("rel-v0.16.0")
			atArm = stateOf(ws, 11)
		}
		return inner(root, doc, vars, ints)
	}
	if _, err := p.do(Request{Record: "rel-v0.16.0", Admit: 11}); err != nil {
		t.Fatal(err)
	}
	if atArm != WithdrawnAdmitted {
		t.Fatalf("origin recorded #11 as %q when it was armed; want admitted first", atArm)
	}
	f.refuse["enqueuePullRequest #12"] = "the forge is down"
	if _, err := p.do(Request{Record: "rel-v0.16.0", Admit: 12}); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "the forge is down") {
		t.Fatalf("admit on a refused arming = %v", err)
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 12) != WithdrawnPending || stateOf(ws, 11) != WithdrawnAdmitted {
		t.Fatalf("recorded %+v; want #12 put back to withdrawn", ws)
	}
}

// TestParseWithdrawalRefusesAnUnknownState: on a tombstone, where settled
// states are read, a state outside the vocabulary is refused.
func TestParseWithdrawalRefusesAnUnknownState(t *testing.T) {
	if _, ok := parseWithdrawal("12 PR_x queue " + headA + " octo gone"); ok {
		t.Fatal("an unknown state parsed")
	}
	if w, ok := parseWithdrawal("12 PR_x queue " + headA + " octo manual"); !ok || w.State != WithdrawnManual {
		t.Fatalf("manual = %+v, %v", w, ok)
	}
	tomb := "abcd reservation: rel-v0.16.0\n\nAbcd-Reservation: rel-v0.16.0\nAbcd-Holder: octo\nAbcd-Copy: 9f2c41d07be35a6c80e1f4b2a7d39c55\n" +
		"Abcd-Claimed-At: 2026-10-03T14:00:00Z\nAbcd-Expires-At: 2026-10-03T15:00:00Z\nAbcd-Released-At: 2026-10-03T15:00:00Z\n" +
		"Abcd-Withdrawn: 12 PR_x queue " + headA + " octo gone\n"
	if _, why := parseMessage("rel-v0.16.0", tomb); why != reasonWithdrawn {
		t.Fatalf("a tombstone with an unknown state read with %q", why)
	}
}

// TestCarriedKeepsOnlyWhatIsStillToSettle and the stranded filter.
func TestCarriedKeepsOnlyWhatIsStillToSettle(t *testing.T) {
	ws := []Withdrawal{
		{Number: 1, State: WithdrawnPending, By: "octo"},
		{Number: 2, State: WithdrawnAdmitted, By: "octo"},
		{Number: 3, State: WithdrawnRearmed, By: "octo"},
		{Number: 4, State: WithdrawnMoved, By: "octo"},
		{Number: 5, State: WithdrawnPending, By: "quinn"},
	}
	if got := numbersOf(carried(ws, false)); !slices.Equal(got, []int{1, 5}) {
		t.Fatalf("carried from another tip = %v; want the pending ones", got)
	}
	if got := numbersOf(carried(ws, true)); !slices.Equal(got, []int{1, 2, 5}) {
		t.Fatalf("carried from one's own = %v; want pending and admitted", got)
	}
	if got := numbersOf(stranded(ws, Self{Login: "octo"})); !slices.Equal(got, []int{5}) {
		t.Fatalf("stranded for octo = %v; want quinn's", got)
	}
	if got := numbersOf(stranded(ws, Self{Login: "OCTO"})); !slices.Equal(got, []int{5}) {
		t.Fatalf("stranded for OCTO = %v; want quinn's (holders compare without case)", got)
	}
}

// TestVouchedAndLostReadTheChain: a withdrawal is vouched for only by a
// reservation of its own withdrawer that first recorded it, withdrawn; one
// first seen on a tombstone, under another holder, past the walk's bound or
// beneath an unreadable commit is not. A later commit that drops one is
// lost; one replaced at a new head for the same number is not.
func TestVouchedAndLostReadTheChain(t *testing.T) {
	w11 := Withdrawal{Number: 11, ID: "PR_eleven", Form: FormQueue, Head: headA, By: "octo", State: WithdrawnPending}
	w11b := Withdrawal{Number: 11, ID: "PR_eleven", Form: FormQueue, Head: headB, By: "octo", State: WithdrawnPending}
	w12 := Withdrawal{Number: 12, ID: "PR_twelve", Form: FormQueue, Head: headB, By: "octo", State: WithdrawnPending}
	res := func(holder string, released bool, ws ...Withdrawal) chainLink {
		m := message{Record: "rel-v0.16.0", Holder: holder, Withdrawn: ws}
		if released {
			m.ReleasedAt = t0
		}
		return chainLink{commit: "c", m: m, readable: true}
	}
	plain := res("octo", false)
	cases := []struct {
		name     string
		links    []chainLink
		complete bool
		want     bool
	}{
		{"recorded by octo's reservation", []chainLink{res("octo", true, w11), res("octo", false, w11), plain}, true, true},
		{"first seen on a tombstone", []chainLink{res("mal", true, w11), plain}, true, false},
		{"first seen under another holder", []chainLink{res("octo", true, w11), res("mal", false, w11), plain}, true, false},
		{"first record past the bound", []chainLink{res("octo", false, w11), res("octo", false, w11)}, false, false},
		{"beneath an unreadable commit", []chainLink{res("octo", false, w11), {commit: "u"}}, true, false},
		{"the root commit records it", []chainLink{res("octo", false, w11)}, true, true},
	}
	for _, c := range cases {
		if got := vouched(c.links, c.complete, w11); got != c.want {
			t.Errorf("%s: vouched = %v; want %v", c.name, got, c.want)
		}
	}
	// An older binary's renewal dropped both: both lost.
	links := []chainLink{res("octo", false), res("octo", false, w11, w12), plain}
	if got := numbersOf(lostWithdrawals(links, true)); !slices.Equal(got, []int{11, 12}) {
		t.Fatalf("lost = %v; want #11 and #12", got)
	}
	// #11 re-withdrawn at a new head is carried, not lost; #12 dropped is.
	links = []chainLink{res("octo", false, w11b), res("octo", false, w11b), res("octo", false, w11, w12), plain}
	if got := numbersOf(lostWithdrawals(links, true)); !slices.Equal(got, []int{12}) {
		t.Fatalf("lost = %v; want #12 alone", got)
	}
	// Settled on the tombstone above: nothing lost.
	settled := w11
	settled.State = WithdrawnRearmed
	links = []chainLink{res("octo", true, settled), res("octo", false, w11), plain}
	if got := lostWithdrawals(links, true); len(got) != 0 {
		t.Fatalf("lost = %+v; want none", got)
	}
}
