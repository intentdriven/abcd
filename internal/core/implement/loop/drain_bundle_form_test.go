package loop

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The drain forms and opens bundles (itd-2611201450052326,
// spc-2611201455051897 step 4): after the lead's no, the drain takes each
// candidate the way it takes an issue, reserving it before its judgement, and
// opens one lane for the lead and the members that answered no, keeping every
// hold across moves until it does.

const (
	formLead = "iss-2611250000000001"
	formM1   = "iss-2611250000000002"
	formM2   = "iss-2611250000000003"
	formM3   = "iss-2611250000000004"
)

var formIssues = []string{formLead, formM1, formM2, formM3}

// bundleDrainRepo is a drain repository on a local bare origin, P's
// checkout, with four bugs of one severity, each found at its own file in
// internal/a: a lead and three candidates.
func bundleDrainRepo(t *testing.T) (*gittest.Repo, string) {
	t.Helper()
	_, w := builtOnce(t, worldKey("bundleDrainRepo"), func(w *fixtureWorld) struct{} {
		repo := briefRepo(t, agentsMarked)
		issueRepoRule(t, repo)
		for i, id := range formIssues {
			file := fmt.Sprintf("internal/a/f%d.go", i)
			repo.Write(file, "package a\n")
			fileIssueAt(t, repo, id, capture.SeverityMinor, capture.Category("bug"), fmt.Sprintf("Guard case %d in the renderer and test it.", i), file)
		}
		repo.Commit("the ledger")
		gittest.InitBare(t, w.bare)
		repo.Git("remote", "add", "origin", w.bare)
		repo.Git("push", "-q", "origin", "main")
		repo.Git("fetch", "-q", "origin")
		asPat(t, repo)
		return struct{}{}
	})
	return w.repo, w.bare
}

// machineConfig writes the person's machine configuration.
func bundleMachineConfig(t *testing.T, body string) {
	t.Helper()
	writeConfigAt(t, abcdhome.Path(os.Getenv("HOME"), "config.json"), body)
}

// openedBundle is what a bundle's lane was opened with.
type openedBundle struct {
	lead    string
	members []laneMember
}

// recordBundles makes the drain's bundle start record what each lane opens
// with and then start it as the drain does, the run's start writing the
// members (loop.go).
func recordBundles(t *testing.T) *[]openedBundle {
	t.Helper()
	var got []openedBundle
	was := startBundle
	startBundle = func(repoRoot, lead string, o Options, members []laneMember) (StartResult, error) {
		got = append(got, openedBundle{lead: lead, members: members})
		return was(repoRoot, lead, o, members)
	}
	t.Cleanup(func() { startBundle = was })
	return &got
}

func mustDrain(t *testing.T, repo *gittest.Repo, o Options, d DrainOptions) DrainResult {
	t.Helper()
	res, err := Drain(repo.Root(), o, d)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func formedState(t *testing.T, repo *gittest.Repo) DrainState {
	t.Helper()
	st, _, err := readDrain(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	return st
}

// plainlyReleasedOver fails unless origin's reservation of id is a release
// over taken that marks no hand-back.
func plainlyReleasedOver(t *testing.T, bare, id, taken string) {
	t.Helper()
	releasedOver(t, bare, id, taken)
	tip := refOf(t, bare, id)
	if body := gitIn(t, bare, "log", "-1", "--format=%B", tip); strings.Contains(body, "Abcd-Handed-Back") {
		t.Fatalf("the release of %s marks a hand-back:\n%s", id, body)
	}
}

const noReason = "the remedy changes neither what a user sees nor a trust boundary"

// orderOf is the fixture's four issues in the drain's order as res planned
// it: the lead, then its candidates. Within a category and severity the order
// is the drain's own hash, so a test reads it rather than assuming one.
func orderOf(t *testing.T, res DrainResult) (lead, m1, m2, m3 string) {
	t.Helper()
	var ids []string
	for _, v := range res.Dispositions {
		if v.Outcome == capture.DrainEligible && slices.Contains(formIssues, v.ID) {
			ids = append(ids, v.ID)
		}
	}
	if len(ids) != 4 {
		t.Fatalf("the drain plans the fixture's four issues: %v", ids)
	}
	return ids[0], ids[1], ids[2], ids[3]
}

// TestABundleFormsOverMovesAndOpensWithTheRest: the lead's no forms a bundle
// and the move asks over the first candidate, reserved before its judgement;
// a yes routes that candidate, which leaves; each no joins; and once no
// candidate is left the lane opens with the lead and the members, each on the
// hold it was judged on, every hold kept until then.
func TestABundleFormsOverMovesAndOpensWithTheRest(t *testing.T) {
	repo, bare := bundleDrainRepo(t)
	bundleMachineConfig(t, `{"drain":{"bundle_max":4}}`)
	opened := recordBundles(t)
	o := Options{}

	first := mustDrain(t, repo, o, DrainOptions{})
	lead, m1, m2, m3 := orderOf(t, first)
	if first.Judging == nil || first.Judging.Issue != lead {
		t.Fatalf("the drain asks over the lead first: %+v", first.Judging)
	}
	leadHold := refOf(t, bare, lead)

	second := judge(t, repo, o, first, JudgementNo, "", noReason)
	if second.Start != nil || second.Judging == nil || second.Judging.Issue != m1 {
		t.Fatalf("the lead's no opens no lane and asks over the first candidate: %+v", second)
	}
	st := formedState(t, repo)
	if st.Forming == nil || st.Forming.Lead != lead || st.Forming.Hold == nil || st.Forming.Hold.Commit != leadHold || len(st.Forming.Members) != 0 {
		t.Fatalf("the drain records the bundle forming on the lead and its hold: %+v", st.Forming)
	}
	heldStill(t, bare, lead, leadHold)
	m1Hold := refOf(t, bare, m1)
	if m1Hold == "" || second.Judging.Hold == nil || second.Judging.Hold.Commit != m1Hold {
		t.Fatalf("the candidate is reserved before its judgement: %q %+v", m1Hold, second.Judging.Hold)
	}
	if runCount(t, repo) != 0 {
		t.Fatal("a run was started while the bundle forms")
	}

	third := judge(t, repo, o, second, JudgementYes, HandBackUserVisible, "the renderer's output changes")
	routeOf(t, third.Routed, m1)
	if r := originReservation(t, bare, claim.RefPrefix+m1); !r.released {
		t.Fatalf("the candidate answered yes is released: %+v", r)
	}
	if third.Judging == nil || third.Judging.Issue != m2 {
		t.Fatalf("the move goes on to the next candidate: %+v", third.Judging)
	}
	heldStill(t, bare, lead, leadHold)

	fourth := judge(t, repo, o, third, JudgementNo, "", noReason)
	m2Hold := refOf(t, bare, m2)
	if fourth.Judging == nil || fourth.Judging.Issue != m3 {
		t.Fatalf("the move goes on to the last candidate: %+v", fourth.Judging)
	}
	st = formedState(t, repo)
	if st.Forming == nil || len(st.Forming.Members) != 1 || st.Forming.Members[0].Issue != m2 || st.Forming.Members[0].Hold.Commit != m2Hold {
		t.Fatalf("the candidate's no joins the bundle with its hold: %+v", st.Forming)
	}
	if !st.taken(m2, &originBlobs{}) {
		t.Fatal("a member of the bundle forming is not counted as taken")
	}

	fifth := judge(t, repo, o, fourth, JudgementNo, "", noReason)
	m3Hold := refOf(t, bare, m3)
	if fifth.Start == nil || fifth.Lane == nil || fifth.Lane.Issue != lead {
		t.Fatalf("with no candidate left the bundle's lane opens: %+v", fifth)
	}
	if !slices.Equal(fifth.Lane.Bundle, []string{m2, m3}) {
		t.Fatalf("the lane names its members: %v", fifth.Lane.Bundle)
	}
	if len(*opened) != 1 || (*opened)[0].lead != lead {
		t.Fatalf("one bundle opened, on the lead: %+v", *opened)
	}
	members := (*opened)[0].members
	if len(members) != 2 || members[0].issue != m2 || members[1].issue != m3 {
		t.Fatalf("the lane opens with the members in the drain's order: %+v", members)
	}
	for _, m := range members {
		if m.held == nil || !strings.Contains(m.held.note, "reserved "+m.issue) || len(m.footprint) != 1 || !strings.HasPrefix(m.footprint[0], "internal/a/") {
			t.Fatalf("each member opens on the hold it was judged on, with its footprint: %+v", m)
		}
	}
	st = formedState(t, repo)
	if st.Forming != nil {
		t.Fatalf("an opened bundle forms no longer: %+v", st.Forming)
	}
	heldStill(t, bare, lead, leadHold)
	heldStill(t, bare, m2, m2Hold)
	heldStill(t, bare, m3, m3Hold)
	for _, id := range []string{lead, m2, m3} {
		if !st.taken(id, &originBlobs{}) {
			t.Fatalf("%s, which the lane carries, is not counted as taken", id)
		}
	}
}

// TestACapOfOneOpensTheLeadsLaneAlone: with the bundled cap the lead's no
// opens its lane at once, forming nothing.
func TestACapOfOneOpensTheLeadsLaneAlone(t *testing.T) {
	repo, _ := bundleDrainRepo(t)
	opened := recordBundles(t)
	res := drainToLane(t, repo, Options{}, DrainOptions{})
	lead, _, _, _ := orderOf(t, res)
	if res.Start == nil || res.Lane == nil || res.Lane.Issue != lead || len(res.Lane.Bundle) != 0 {
		t.Fatalf("a cap of 1 opens the lead's lane alone: %+v", res)
	}
	if st := formedState(t, repo); st.Forming != nil {
		t.Fatalf("a cap of 1 forms nothing: %+v", st.Forming)
	}
	if len(*opened) != 0 {
		t.Fatalf("no bundle opened: %+v", *opened)
	}
}

// formingOnLead is the fixture moved to a bundle forming on the lead with the
// first candidate's judgement awaited, under o, and the lead's hold.
func formingOnLead(t *testing.T, repo *gittest.Repo, bare string, o Options) (DrainResult, string, []string) {
	t.Helper()
	first := mustDrain(t, repo, o, DrainOptions{})
	lead, m1, m2, m3 := orderOf(t, first)
	second := judge(t, repo, o, first, JudgementNo, "", noReason)
	if st := formedState(t, repo); st.Forming == nil || second.Judging == nil {
		t.Fatalf("the bundle forms: %+v %+v", st.Forming, second.Judging)
	}
	return second, refOf(t, bare, lead), []string{lead, m1, m2, m3}
}

// TestAFormingBundleKeepsItsHoldsAndRenewsThem: a move with the candidate's
// judgement still awaited leaves the lead's hold as it stands, and one past
// half its lease renews it, the bundle recording the renewal.
func TestAFormingBundleKeepsItsHoldsAndRenewsThem(t *testing.T) {
	repo, bare := bundleDrainRepo(t)
	bundleMachineConfig(t, `{"drain":{"bundle_max":4}}`)
	c := &clock{t: time.Now().UTC().Truncate(time.Second)}
	o := Options{Now: c.now}
	_, leadHold, ids := formingOnLead(t, repo, bare, o)
	lead, m1 := ids[0], ids[1]

	mustDrain(t, repo, o, DrainOptions{})
	heldStill(t, bare, lead, leadHold)

	r := originReservation(t, bare, claim.RefPrefix+lead)
	c.t = r.claimed.Add(r.expires.Sub(r.claimed) * 3 / 4)
	res := mustDrain(t, repo, o, DrainOptions{})
	if res.Judging == nil || res.Judging.Issue != m1 {
		t.Fatalf("the candidate's judgement is still awaited: %+v", res.Judging)
	}
	renewed := originReservation(t, bare, claim.RefPrefix+lead)
	if renewed.commit == leadHold || renewed.released || !renewed.expires.After(r.expires) {
		t.Fatalf("past half its lease the lead's hold is renewed: %+v, was %+v", renewed, r)
	}
	if st := formedState(t, repo); st.Forming == nil || st.Forming.Hold.Commit != renewed.commit {
		t.Fatalf("the bundle records the renewed hold: %+v", st.Forming)
	}
	mustDrain(t, repo, o, DrainOptions{})
	heldStill(t, bare, lead, renewed.commit)
}

// TestAWindowAndAStopKeepTheFormingBundle: a move in the pause, and a move a
// stop holds, keep the bundle forming and its hold.
func TestAWindowAndAStopKeepTheFormingBundle(t *testing.T) {
	repo, bare := bundleDrainRepo(t)
	bundleMachineConfig(t, `{"drain":{"bundle_max":4}}`)
	_, leadHold, ids := formingOnLead(t, repo, bare, Options{})
	lead := ids[0]

	st := formedState(t, repo)
	later := time.Now().Add(time.Hour)
	st.NextEligibleAt = &later
	if err := writeDrain(repo.Root(), st); err != nil {
		t.Fatal(err)
	}
	if res := mustDrain(t, repo, Options{}, DrainOptions{}); !strings.Contains(res.Next, "nothing before") {
		t.Fatalf("the move waits out the pause: %s", res.Next)
	}
	if st := formedState(t, repo); st.Forming == nil || st.Forming.Lead != lead {
		t.Fatalf("the pause keeps the bundle forming: %+v", st.Forming)
	}
	heldStill(t, bare, lead, leadHold)

	st = formedState(t, repo)
	st.NextEligibleAt = nil
	if err := writeDrain(repo.Root(), st); err != nil {
		t.Fatal(err)
	}
	so, _, _ := stopped(t)
	if _, err := Drain(repo.Root(), so, DrainOptions{}); err == nil {
		t.Fatal("a move under a stop went ahead")
	}
	if st := formedState(t, repo); st.Forming == nil || st.Forming.Lead != lead {
		t.Fatalf("the stop keeps the bundle forming: %+v", st.Forming)
	}
	heldStill(t, bare, lead, leadHold)
}

// TestADrainThatEndsReleasesTheFormingHoldsPlainly: an outage the run gave up
// on ends the drain while a bundle forms; the lead's hold and the member's
// are released with no hand-back marked, and nothing forms any longer.
func TestADrainThatEndsReleasesTheFormingHoldsPlainly(t *testing.T) {
	repo, bare := bundleDrainRepo(t)
	bundleMachineConfig(t, `{"drain":{"bundle_max":4}}`)
	c := &clock{t: time.Now().UTC().Truncate(time.Second)}
	o := Options{Now: c.now}
	second, leadHold, ids := formingOnLead(t, repo, bare, o)
	lead, m1, m2 := ids[0], ids[1], ids[2]
	third := judge(t, repo, o, second, JudgementNo, "", noReason)
	if third.Judging == nil || third.Judging.Issue != m2 {
		t.Fatalf("the first candidate joins and the next is asked: %+v", third.Judging)
	}
	m1Hold := refOf(t, bare, m1)

	giveUp(t, joinShared(t, repo.Root(), c.now), "host-a", func(d time.Duration) { c.t = c.t.Add(d) })
	res := mustDrain(t, repo, o, DrainOptions{})
	if res.Stopped != DrainStoppedOutage || !strings.Contains(res.Next, "ended with the drain") {
		t.Fatalf("the drain ends and says the bundle ended with it: %+v", res)
	}
	plainlyReleasedOver(t, bare, lead, leadHold)
	plainlyReleasedOver(t, bare, m1, m1Hold)
	if st := formedState(t, repo); st.Forming != nil {
		t.Fatalf("an ended drain forms nothing: %+v", st.Forming)
	}
}

// TestABundleOpensAtTheReservationCapWithoutWaiting: with
// reservation.max_live 3 under a cap of 5, a bundle holding three opens as it
// stands rather than waiting, the next candidate not reserved.
func TestABundleOpensAtTheReservationCapWithoutWaiting(t *testing.T) {
	repo, bare := bundleDrainRepo(t)
	bundleMachineConfig(t, `{"drain":{"bundle_max":5},"reservation":{"max_live":3}}`)
	opened := recordBundles(t)
	res := mustDrain(t, repo, Options{}, DrainOptions{})
	_, m1, m2, m3 := orderOf(t, res)
	for range 3 {
		res = judge(t, repo, Options{}, res, JudgementNo, "", noReason)
	}
	if res.Start == nil || res.Lane == nil || !slices.Equal(res.Lane.Bundle, []string{m1, m2}) {
		t.Fatalf("the bundle of three opens at the cap: %+v", res)
	}
	if st := formedState(t, repo); st.Waiting != "" || st.Forming != nil {
		t.Fatalf("the drain does not wait at the cap: %q %+v", st.Waiting, st.Forming)
	}
	if refOf(t, bare, m3) != "" {
		t.Fatal("the candidate past the cap was reserved")
	}
	if len(*opened) != 1 || len((*opened)[0].members) != 2 {
		t.Fatalf("one bundle of three opened: %+v", *opened)
	}
}

// openedBundleLane is the fixture drained, under a cap of 3, to a bundle's
// lane opened on the lead with the first two candidates as members, which
// the run's start writes onto its run; it returns the lead, the members and
// the run.
func openedBundleLane(t *testing.T, repo *gittest.Repo) (lead, m1, m2, runID string) {
	t.Helper()
	bundleMachineConfig(t, `{"drain":{"bundle_max":3}}`)
	res := mustDrain(t, repo, Options{}, DrainOptions{})
	lead, m1, m2, _ = orderOf(t, res)
	for range 3 {
		res = judge(t, repo, Options{}, res, JudgementNo, "", noReason)
	}
	if res.Start == nil || res.Lane == nil || !slices.Equal(res.Lane.Bundle, []string{m1, m2}) {
		t.Fatalf("the bundle's lane opens: %+v", res)
	}
	return lead, m1, m2, res.Lane.RunID
}

// handBackMember hands m back alone on the run's current lane, as a receipt's
// handbacks does.
func handBackMember(t *testing.T, repo *gittest.Repo, runID, m string) {
	t.Helper()
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		hb := HandBack{Kind: HandBackDesignFinding, Reason: "the guard needs a decision", Home: "the brief"}
		handBackAlone(st, st.Lanes[st.current()].ID, []issueHandedBack{{issue: m, hb: hb}}, time.Now())
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
}

// routedFor are the routes in rs of id.
func routedFor(rs []DrainRoute, id string) []DrainRoute {
	var out []DrainRoute
	for _, r := range rs {
		if r.Issue == id {
			out = append(out, r)
		}
	}
	return out
}

// TestABundlesHandBacksAreRoutedByIssue: through the drain's moves, a member
// handed back alone is routed alone, once, while its lane goes on, and is in
// the lane's hand no longer; a whole-lane hand-back then routes every issue
// the run still carries, the lead first, and not again the member routed
// alone.
func TestABundlesHandBacksAreRoutedByIssue(t *testing.T) {
	repo, _ := bundleDrainRepo(t)
	lead, m1, m2, runID := openedBundleLane(t, repo)

	handBackMember(t, repo, runID, m1)
	res := mustDrain(t, repo, Options{}, DrainOptions{})
	if rs := routedFor(res.Routed, m1); len(rs) != 1 || rs[0].From != "lane" || rs[0].Kind != HandBackDesignFinding {
		t.Fatalf("the member handed back alone is routed alone: %+v", res.Routed)
	}
	if res.Lane == nil || res.Lane.RunID != runID {
		t.Fatalf("the lane goes on with the rest: %+v", res.Lane)
	}
	again := mustDrain(t, repo, Options{}, DrainOptions{})
	if rs := routedFor(again.Routed, m1); len(rs) != 0 {
		t.Fatalf("a member routed alone is not routed again: %+v", again.Routed)
	}
	st := formedState(t, repo)
	if !st.taken(lead, &originBlobs{}) || !st.taken(m2, &originBlobs{}) {
		t.Fatal("the lane's lead and its member still held are not counted as taken")
	}
	if !st.handedBackAloneIn(st.Lanes[0], m1) {
		t.Fatal("the member routed alone is still in the lane's hand")
	}

	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		handBackLane(st, &st.Lanes[st.current()], HandBack{Kind: HandBackDesignFinding, Reason: "the lane needs a decision", Home: "the brief"}, "", time.Now())
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	whole := mustDrain(t, repo, Options{}, DrainOptions{})
	var ids []string
	for _, r := range whole.Routed {
		if r.From == "lane" {
			ids = append(ids, r.Issue)
		}
	}
	if !slices.Equal(ids, []string{lead, m2}) {
		t.Fatalf("a whole-lane hand-back routes every issue still held, the lead first: %v", ids)
	}
}

// TestAMemberHandedBackAloneIsRoutedWhenItsLaneMerges: a member handed back
// alone whose lane merges before the drain's next move reads it is routed as
// the lane is marked done, and is in the lane's hand no longer.
func TestAMemberHandedBackAloneIsRoutedWhenItsLaneMerges(t *testing.T) {
	repo, _ := bundleDrainRepo(t)
	lead, m1, _, runID := openedBundleLane(t, repo)
	handBackMember(t, repo, runID, m1)
	was := drainFinish
	drainFinish = func(string, []DrainLane, time.Time, Options, []string) ([]DrainFinished, []DrainParked, error) {
		return []DrainFinished{{Issue: lead, RunID: runID, PR: 7, Done: true}}, nil, nil
	}
	t.Cleanup(func() { drainFinish = was })
	res := mustDrain(t, repo, Options{}, DrainOptions{})
	if rs := routedFor(res.Routed, m1); len(rs) != 1 || rs[0].From != "lane" {
		t.Fatalf("the member handed back alone is routed as its lane is done: %+v", res.Routed)
	}
	st := formedState(t, repo)
	if st.Lanes[0].Outcome != DrainLaneDone || !st.handedBackAloneIn(st.Lanes[0], m1) {
		t.Fatalf("the lane is done and the member out of its hand: %+v", st.Lanes[0])
	}
}

// handRenewedCandidate is the fixture moved to a bundle forming on the lead
// whose first candidate this person renewed by hand from this copy before its
// no was answered: it returns the order, the candidate's tip after the hand
// renewal, and the move that answered.
func handRenewedCandidate(t *testing.T, repo *gittest.Repo, bare string, o Options) ([]string, string, DrainResult) {
	t.Helper()
	second, _, ids := formingOnLead(t, repo, bare, o)
	m1 := ids[1]
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: m1, Footprint: footprintOf(t, bare, m1)}, clockAt(o.now())); err != nil {
		t.Fatalf("renewing %s by hand: %v", m1, err)
	}
	byHand := refOf(t, bare, m1)
	if byHand == second.Judging.Hold.Commit {
		t.Fatal("premise: the hand renewal moved the tip")
	}
	third := judge(t, repo, o, second, JudgementNo, "", noReason)
	if i := slices.IndexFunc(third.Passed, func(e Excluded) bool { return e.ID == m1 }); i < 0 || !strings.Contains(third.Passed[i].Reason, "by hand") {
		t.Fatalf("the move says the candidate is left as this person holds it: %+v", third.Passed)
	}
	if st := formedState(t, repo); st.Forming == nil || st.Forming.has(m1) {
		t.Fatalf("the hand-renewed candidate does not join: %+v", st.Forming)
	}
	heldStill(t, bare, m1, byHand)
	return ids, byHand, third
}

// TestAHandRenewedCandidateIsLeftAsItStandsWhenTheDrainEnds: a candidate
// this person renewed by hand is never the drain's: a move past half the
// bundle's lease renews it not, and a drain that ends does not release it.
func TestAHandRenewedCandidateIsLeftAsItStandsWhenTheDrainEnds(t *testing.T) {
	repo, bare := bundleDrainRepo(t)
	bundleMachineConfig(t, `{"drain":{"bundle_max":4}}`)
	c := &clock{t: time.Now().UTC().Truncate(time.Second)}
	o := Options{Now: c.now}
	ids, byHand, _ := handRenewedCandidate(t, repo, bare, o)
	r := originReservation(t, bare, claim.RefPrefix+ids[0])
	c.t = r.claimed.Add(r.expires.Sub(r.claimed) * 3 / 4)
	mustDrain(t, repo, o, DrainOptions{})
	heldStill(t, bare, ids[1], byHand)
	giveUp(t, joinShared(t, repo.Root(), c.now), "host-a", func(d time.Duration) { c.t = c.t.Add(d) })
	if res := mustDrain(t, repo, o, DrainOptions{}); res.Stopped != DrainStoppedOutage {
		t.Fatalf("the drain ends: %+v", res)
	}
	heldStill(t, bare, ids[1], byHand)
}

// TestAHandRenewedCandidateIsLeftAsItStandsWhenTheStartFails: a bundle whose
// start fails releases the holds it took, and not a candidate this person
// renewed by hand.
func TestAHandRenewedCandidateIsLeftAsItStandsWhenTheStartFails(t *testing.T) {
	repo, bare := bundleDrainRepo(t)
	bundleMachineConfig(t, `{"drain":{"bundle_max":4}}`)
	ids, byHand, third := handRenewedCandidate(t, repo, bare, Options{})
	fourth := judge(t, repo, Options{}, third, JudgementNo, "", noReason)
	if fourth.Judging == nil || fourth.Judging.Issue != ids[3] {
		t.Fatalf("the last candidate is asked: %+v", fourth.Judging)
	}
	was := startBundle
	startBundle = func(string, string, Options, []laneMember) (StartResult, error) {
		return StartResult{}, errors.New("the start failed")
	}
	t.Cleanup(func() { startBundle = was })
	path := writeAnswer(t, repo, fourth.Judging, judgementAnswer(fourth.Judging, JudgementNo, "", noReason))
	if _, err := Drain(repo.Root(), Options{}, DrainOptions{Judgement: path}); err == nil {
		t.Fatal("the bundle's failed start did not refuse the move")
	}
	heldStill(t, bare, ids[1], byHand)
	for _, id := range []string{ids[0], ids[2], ids[3]} {
		if r := originReservation(t, bare, claim.RefPrefix+id); !r.released {
			t.Fatalf("%s's hold, the drain's, is released when the start fails: %+v", id, r)
		}
	}
}

// moveFoundAtOnOrigin rewrites id's found_at on origin's default branch.
func moveFoundAtOnOrigin(t *testing.T, bare, id, to string) {
	t.Helper()
	pushFromOtherCopy(t, bare, "move "+id, func(dir string) {
		matches, _ := filepath.Glob(filepath.Join(dir, ".abcd", "work", "issues", "open", id+"-*.md"))
		if len(matches) != 1 {
			t.Fatalf("origin holds no open record of %s", id)
		}
		data, err := os.ReadFile(matches[0])
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(string(data), "\n")
		moved := false
		for i, l := range lines {
			if strings.HasPrefix(l, "found_at:") {
				lines[i], moved = "found_at: "+to, true
			}
		}
		if !moved {
			t.Fatalf("%s's record names no found_at", id)
		}
		if err := os.WriteFile(matches[0], []byte(strings.Join(lines, "\n")), 0o644); err != nil {
			t.Fatal(err)
		}
	})
}

// TestAMemberWhoseFoundAtMovedLeavesTheBundle: a member whose found_at names
// another path than the footprint its hold reserved, once the bundle is
// full, leaves it before the lane opens, its hold released plainly, and the
// lane opens with the rest.
func TestAMemberWhoseFoundAtMovedLeavesTheBundle(t *testing.T) {
	repo, bare := bundleDrainRepo(t)
	bundleMachineConfig(t, `{"drain":{"bundle_max":3}}`)
	opened := recordBundles(t)
	second, _, ids := formingOnLead(t, repo, bare, Options{})
	third := judge(t, repo, Options{}, second, JudgementNo, "", noReason)
	m1Hold := refOf(t, bare, ids[1])
	if st := formedState(t, repo); st.Forming == nil || !st.Forming.has(ids[1]) {
		t.Fatalf("the first candidate joins: %+v", st.Forming)
	}
	moveFoundAtOnOrigin(t, bare, ids[1], "internal/a/moved.go")
	res := judge(t, repo, Options{}, third, JudgementNo, "", noReason)
	if res.Start == nil || res.Lane == nil || !slices.Equal(res.Lane.Bundle, []string{ids[2]}) {
		t.Fatalf("the lane opens with the members whose footprint stands: %+v", res.Lane)
	}
	if !strings.Contains(res.Next, ids[1]+" left the bundle") {
		t.Fatalf("the move says the member left: %s", res.Next)
	}
	plainlyReleasedOver(t, bare, ids[1], m1Hold)
	if len(*opened) != 1 || len((*opened)[0].members) != 1 || (*opened)[0].members[0].issue != ids[2] {
		t.Fatalf("the bundle opens without the moved member: %+v", *opened)
	}
}

// TestANoWhoseHoldIsGoneIsReservedBeforeItJoins: a candidate answered no in
// a move whose window had closed loses the hold it was judged on; the next
// move reserves it again before it joins, so it never joins unreserved.
func TestANoWhoseHoldIsGoneIsReservedBeforeItJoins(t *testing.T) {
	repo, bare := bundleDrainRepo(t)
	bundleMachineConfig(t, `{"drain":{"bundle_max":4}}`)
	second, _, ids := formingOnLead(t, repo, bare, Options{})
	judgedOn := second.Judging.Hold.Commit
	st := formedState(t, repo)
	long := time.Now().Add(-24 * time.Hour)
	st.WindowStartedAt = &long
	if err := writeDrain(repo.Root(), st); err != nil {
		t.Fatal(err)
	}
	closed := judge(t, repo, Options{}, second, JudgementNo, "", noReason)
	if closed.Start != nil || !strings.Contains(closed.Next, "window has elapsed") {
		t.Fatalf("the answer is taken in a move whose window closed: %s", closed.Next)
	}
	releasedOver(t, bare, ids[1], judgedOn)
	next := mustDrain(t, repo, Options{}, DrainOptions{})
	st = formedState(t, repo)
	if st.Forming == nil || !st.Forming.has(ids[1]) {
		t.Fatalf("the candidate joins on the next move: %+v %s", st.Forming, next.Next)
	}
	i := slices.IndexFunc(st.Forming.Members, func(m DrainFormingMember) bool { return m.Issue == ids[1] })
	h := st.Forming.Members[i].Hold
	if h == nil || h.Commit != refOf(t, bare, ids[1]) {
		t.Fatalf("the candidate joins on a hold taken again: %+v, origin %s", h, refOf(t, bare, ids[1]))
	}
	heldStill(t, bare, ids[1], h.Commit)
}

// TestAFormingBundleWaitsOnASilentOrigin: a move that cannot read origin's
// default branch keeps the bundle forming and its hold.
func TestAFormingBundleWaitsOnASilentOrigin(t *testing.T) {
	repo, bare := bundleDrainRepo(t)
	bundleMachineConfig(t, `{"drain":{"bundle_max":4}}`)
	_, leadHold, ids := formingOnLead(t, repo, bare, Options{})
	gitIn(t, bare, "branch", "-m", "main", "trunk")
	gitIn(t, bare, "symbolic-ref", "HEAD", "refs/heads/trunk")
	res := mustDrain(t, repo, Options{}, DrainOptions{})
	if res.OriginWait == nil {
		t.Fatalf("the move waits on origin: %+v", res)
	}
	if st := formedState(t, repo); st.Forming == nil || st.Forming.Lead != ids[0] {
		t.Fatalf("a silent origin keeps the bundle forming: %+v", st.Forming)
	}
	heldStill(t, bare, ids[0], leadHold)
}

// TestAClosedBundleWhoseLeadWentAloneRoutesTheRunsIssues: a bundle's pull
// request closed unmerged after its lead was handed back alone routes each
// issue the run still holds with the closed hand-back and the lead once,
// alone, and never an issue named as none, whatever the parked lane names.
func TestAClosedBundleWhoseLeadWentAloneRoutesTheRunsIssues(t *testing.T) {
	repo, _ := bundleDrainRepo(t)
	lead, m1, m2, runID := openedBundleLane(t, repo)
	handBackMember(t, repo, runID, lead)
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		handBackLane(st, &st.Lanes[st.current()], HandBack{Kind: HandBackClosed, Reason: "the pull request was closed unmerged", Home: "a person"}, "", time.Now())
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	was := drainFinish
	drainFinish = func(string, []DrainLane, time.Time, Options, []string) ([]DrainFinished, []DrainParked, error) {
		// As readParked names a closed bundle whose lead went alone: no
		// issue, its members in Bundle.
		return nil, []DrainParked{{Issue: "", RunID: runID, PR: 7, State: ParkedClosed, Bundle: []string{m1, m2}}}, nil
	}
	t.Cleanup(func() { drainFinish = was })
	res := mustDrain(t, repo, Options{}, DrainOptions{})
	if rs := routedFor(res.Routed, ""); len(rs) != 0 {
		t.Fatalf("a route names no issue: %+v", rs)
	}
	if rs := routedFor(res.Routed, lead); len(rs) != 1 || rs[0].Kind != HandBackDesignFinding {
		t.Fatalf("the lead is routed once, by its own hand-back: %+v", res.Routed)
	}
	for _, m := range []string{m1, m2} {
		if rs := routedFor(res.Routed, m); len(rs) != 1 || rs[0].Kind != HandBackClosed {
			t.Fatalf("%s is routed by the closed hand-back: %+v", m, res.Routed)
		}
	}
}

// TestABundleStartsEndToEndWithItsMember: through the drain's own moves and
// the run's own start, with no seam in between, a lead and one candidate that
// both answer no form under a cap of 2, and the lane opens on a run that
// carries both: the member written on the run with the hold it was judged
// on, a reservation line naming it, and both holds kept on origin.
func TestABundleStartsEndToEndWithItsMember(t *testing.T) {
	repo, bare := bundleDrainRepo(t)
	bundleMachineConfig(t, `{"drain":{"bundle_max":2}}`)
	o := Options{}
	first := mustDrain(t, repo, o, DrainOptions{})
	lead, m1, _, _ := orderOf(t, first)
	leadHold := refOf(t, bare, lead)
	second := judge(t, repo, o, first, JudgementNo, "", noReason)
	if second.Judging == nil || second.Judging.Issue != m1 {
		t.Fatalf("the lead's no asks over the first candidate: %+v", second)
	}
	m1Hold := refOf(t, bare, m1)
	third := judge(t, repo, o, second, JudgementNo, "", noReason)
	if third.Start == nil || third.Lane == nil || third.Lane.Issue != lead || !slices.Equal(third.Lane.Bundle, []string{m1}) {
		t.Fatalf("at the cap the bundle's lane opens with the lead and its member: %+v", third)
	}
	run, err := ReadState(repo.Root(), third.Lane.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Key != lead || !slices.Equal(run.Issues(), []string{lead, m1}) {
		t.Fatalf("the run carries the lead and the member: key %s, issues %v", run.Key, run.Issues())
	}
	if len(run.Bundle) != 1 || run.Bundle[0].ReservedAt == nil || len(run.Bundle[0].Footprint) != 1 || !strings.HasPrefix(run.Bundle[0].Footprint[0], "internal/a/") {
		t.Fatalf("the member is written with its hold and footprint: %+v", run.Bundle)
	}
	if !slices.ContainsFunc(run.Record, func(e Entry) bool { return e.Stage == StageReservation && e.Issue == m1 }) {
		t.Fatalf("a reservation line names the member: %+v", run.Record)
	}
	if st := formedState(t, repo); st.Forming != nil {
		t.Fatalf("an opened bundle forms no longer: %+v", st.Forming)
	}
	heldStill(t, bare, lead, leadHold)
	heldStill(t, bare, m1, m1Hold)
}

// refuseBundles makes the drain's bundle start refuse each lead refuse
// answers a refusal for, recording each start asked for; a lead it answers
// nil for starts as the drain does.
func refuseBundles(t *testing.T, refuse func(lead string) *Refusal) *[]openedBundle {
	t.Helper()
	var got []openedBundle
	was := startBundle
	startBundle = func(repoRoot, lead string, o Options, members []laneMember) (StartResult, error) {
		got = append(got, openedBundle{lead: lead, members: members})
		if r := refuse(lead); r != nil {
			return StartResult{}, r
		}
		return was(repoRoot, lead, o, members)
	}
	t.Cleanup(func() { startBundle = was })
	return &got
}

// budgetRefusal is the start's budget check refusing: no reservation's.
func budgetRefusal() *Refusal {
	return &Refusal{Stage: "check", Check: CheckBudget, Reason: "the runner's quota is under the run's estimate", Remedy: "wait for the quota to refill"}
}

// peerRefusal is the start's peers check refusing id: another holds it.
func peerRefusal(id string) *Refusal {
	return &Refusal{Stage: "check", Check: CheckPeers, Contention: true, Reason: "a peer session's live claim names " + id, Remedy: "wait for the peer to let it go"}
}

// driveMove is one drain move as a drive makes it: the host's answer to the
// judgement res awaits, if any, under d, and what the move passed over kept
// in d for every move after.
func driveMove(t *testing.T, repo *gittest.Repo, d *DrainOptions, res DrainResult, answer string) DrainResult {
	t.Helper()
	move := *d
	if res.Judging != nil {
		move.Judgement = writeAnswer(t, repo, res.Judging, judgementAnswer(res.Judging, answer, "", noReason))
	}
	out, err := Drain(repo.Root(), Options{}, move)
	if err != nil {
		t.Fatal(err)
	}
	d.passLeads(out)
	return out
}

// TestARefusedBundleGoesOnToTheNextEligibleIssue: with the cap at 2, a
// bundle whose start is refused for a reason that is not its reservation's
// passes its lead over as a single lane's start does: its holds are released,
// and the move goes on in the drain's order, its member back among the
// eligible issues, so a lane opens on the next eligible issue in the same
// move or the next. Watched failing before the move went on: every move formed
// the same bundle on the same lead and was refused again.
func TestARefusedBundleGoesOnToTheNextEligibleIssue(t *testing.T) {
	repo, bare := bundleDrainRepo(t)
	bundleMachineConfig(t, `{"drain":{"bundle_max":2}}`)
	var lead string
	asked := refuseBundles(t, func(l string) *Refusal {
		if l == lead {
			return budgetRefusal()
		}
		return nil
	})
	first := mustDrain(t, repo, Options{}, DrainOptions{})
	var m1 string
	lead, m1, _, _ = orderOf(t, first)
	leadHold := refOf(t, bare, lead)
	second := judge(t, repo, Options{}, first, JudgementNo, "", noReason)
	third := judge(t, repo, Options{}, second, JudgementNo, "", noReason)
	if len(*asked) != 1 || (*asked)[0].lead != lead {
		t.Fatalf("the bundle on the lead was asked to start once: %+v", *asked)
	}
	if !slices.ContainsFunc(third.Passed, func(e Excluded) bool { return e.ID == lead && e.Check == CheckBudget }) {
		t.Fatalf("the refused bundle's lead is passed over with the start's check: %+v", third.Passed)
	}
	plainlyReleasedOver(t, bare, lead, leadHold)
	lane := third
	if lane.Start == nil {
		d := DrainOptions{}
		lane = driveMove(t, repo, &d, third, JudgementNo)
	}
	if lane.Start == nil || lane.Lane == nil || lane.Lane.Issue != m1 {
		t.Fatalf("a lane opens on the next eligible issue in the same move or the next: %+v", lane)
	}
	if slices.Contains(lane.Lane.Bundle, lead) {
		t.Fatalf("the lane carries the lead passed over: %+v", lane.Lane)
	}
}

// TestARefusedLeadIsNotPickedAgainInTheDrive: a drive keeps the lead of a
// bundle whose start was refused passed over for the rest of its run, so no
// bundle it forms after, in that move or a later one, is led by that lead or
// carries it; with every bundle refused, each lead is asked to start once,
// and the drain works through its order to the last issue, which opens alone.
// Watched failing before a drive kept the pass-over: the next move formed a
// bundle on the same lead again.
func TestARefusedLeadIsNotPickedAgainInTheDrive(t *testing.T) {
	repo, _ := bundleDrainRepo(t)
	bundleMachineConfig(t, `{"drain":{"bundle_max":2}}`)
	asked := refuseBundles(t, func(string) *Refusal { return budgetRefusal() })
	d := DrainOptions{}
	res := driveMove(t, repo, &d, DrainResult{}, "")
	lead, _, _, last := orderOf(t, res)
	for i := 0; i < 8 && res.Start == nil; i++ {
		if res.Judging == nil && res.Stopped != "" {
			break
		}
		res = driveMove(t, repo, &d, res, JudgementNo)
	}
	var passed []string
	for _, b := range *asked {
		ids := []string{b.lead}
		for _, m := range b.members {
			ids = append(ids, m.issue)
		}
		for _, id := range ids {
			if slices.Contains(passed, id) {
				t.Fatalf("a bundle the drive formed after %s was passed over carries it: %+v", id, *asked)
			}
		}
		passed = append(passed, b.lead)
	}
	if len(passed) == 0 || passed[0] != lead {
		t.Fatalf("the first bundle asked to start is on the lead: %+v", *asked)
	}
	if res.Start == nil || res.Lane == nil || res.Lane.Issue != last || len(res.Lane.Bundle) != 0 {
		t.Fatalf("the drain works through its order to the last issue, which opens alone: %+v", res)
	}
}

// TestABundleAnotherHoldsLeavesTheDriveWaiting: a bundle whose start is
// refused because another holds its lead counts that lead as held work, as
// the single path's start does: with nothing else eligible, the drive waits
// on it and does not end, every move until the wait ends, and the move after
// the wait asks the lead's start again. Watched failing before the drive kept
// the hold beside its pass-over: the move after the refusal ended the drain as
// empty.
func TestABundleAnotherHoldsLeavesTheDriveWaiting(t *testing.T) {
	repo, _ := bundleDrainRepo(t)
	bundleMachineConfig(t, `{"drain":{"bundle_max":2}}`)
	var lead string
	asked := refuseBundles(t, func(l string) *Refusal {
		if l == lead {
			return peerRefusal(l)
		}
		return budgetRefusal()
	})
	d := DrainOptions{}
	first := driveMove(t, repo, &d, DrainResult{}, "")
	var m2, m3 string
	lead, _, m2, m3 = orderOf(t, first)
	d.Skip, d.SkipReason = []string{m2, m3}, "held back so the bundle's two issues are all that is eligible"
	second := driveMove(t, repo, &d, first, JudgementNo)
	d.Skip, d.SkipReason = nil, ""
	third := driveMove(t, repo, &d, second, JudgementNo)
	leadAsked := func() int {
		n := 0
		for _, b := range *asked {
			if b.lead == lead {
				n++
			}
		}
		return n
	}
	if leadAsked() != 1 {
		t.Fatalf("the bundle on the lead is asked to start once: %+v", *asked)
	}
	for i, res := range []DrainResult{third, driveMove(t, repo, &d, third, "")} {
		if res.Stopped != "" || res.Waiting != DrainWaitHeld {
			t.Fatalf("move %d after the refusal waits on the lead another holds, the drain not ended: stopped %q, waiting %q, next %s", i, res.Stopped, res.Waiting, res.Next)
		}
	}
	if leadAsked() != 1 {
		t.Fatalf("a move inside the wait pushes nothing for the lead: %+v", *asked)
	}
	d.retryHeldLeads()
	again := driveMove(t, repo, &d, DrainResult{}, "")
	if leadAsked() != 2 {
		t.Fatalf("the move after the wait asks the lead's start again: %+v", *asked)
	}
	if again.Stopped != "" || again.Waiting != DrainWaitHeld {
		t.Fatalf("a lead still held keeps the drive waiting: stopped %q, waiting %q", again.Stopped, again.Waiting)
	}
}

// TestALeadWhoseFoundAtMovedFormsAgainOnTheNextMove: a lead whose found_at
// names another path than its hold reserved, once its bundle is full, is not
// opened on that hold, and a drive does not keep it passed over: the next
// move reserves it again on its new path and forms its bundle again. Watched
// failing before the drive told this pass-over from a refused start: the lead
// was kept out for the rest of the run, and the next lane opened on its member.
func TestALeadWhoseFoundAtMovedFormsAgainOnTheNextMove(t *testing.T) {
	repo, bare := bundleDrainRepo(t)
	bundleMachineConfig(t, `{"drain":{"bundle_max":2}}`)
	d := DrainOptions{}
	first := driveMove(t, repo, &d, DrainResult{}, "")
	lead, _, _, _ := orderOf(t, first)
	second := driveMove(t, repo, &d, first, JudgementNo)
	// The new path is a file, as the old one was, so the lead keeps its place
	// in the drain's order.
	pushFromOtherCopy(t, bare, "add the moved file", func(dir string) {
		if err := os.WriteFile(filepath.Join(dir, "internal", "a", "moved.go"), []byte("package a\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	})
	moveFoundAtOnOrigin(t, bare, lead, "internal/a/moved.go")
	third := driveMove(t, repo, &d, second, JudgementNo)
	if third.Start != nil || !slices.ContainsFunc(third.Passed, func(e Excluded) bool { return e.ID == lead && e.Check == CheckReservation }) {
		t.Fatalf("the moved lead's bundle does not open on its hold: %+v", third.Passed)
	}
	next := driveMove(t, repo, &d, third, JudgementNo)
	st := formedState(t, repo)
	reformed := st.Forming != nil && st.Forming.Lead == lead
	opened := next.Lane != nil && next.Lane.Issue == lead
	if !reformed && !opened {
		t.Fatalf("the next move forms the bundle on %s again: forming %+v, lane %+v, passed %+v; %s", lead, st.Forming, next.Lane, next.Passed, next.Next)
	}
	r := originReservation(t, bare, claim.RefPrefix+lead)
	if body := gitIn(t, bare, "log", "-1", "--format=%B", r.commit); r.released || !strings.Contains(body, "internal/a/moved.go") {
		t.Fatalf("the lead is reserved again on its new path: %+v\n%s", r, body)
	}
	if reformed && (st.Forming.Hold == nil || st.Forming.Hold.Commit != r.commit) {
		t.Fatalf("the bundle forms again on the lead's new reservation: %+v, origin %s", st.Forming.Hold, r.commit)
	}
}

// TestAnIssueTheCapStoppedAtIsTakenUpOnceARefusedBundleFreesHolds: a walk
// stopped at the reservation cap opens the bundle forming as it stands; when
// that bundle's start is refused, the holds it frees bring the person under
// the cap, and the same move takes up the issue it stopped at, in the drain's
// order. Watched failing before such an issue was walked again: the move
// passed it and asked over the one after it.
func TestAnIssueTheCapStoppedAtIsTakenUpOnceARefusedBundleFreesHolds(t *testing.T) {
	repo, _ := bundleDrainRepo(t)
	bundleMachineConfig(t, `{"drain":{"bundle_max":4},"reservation":{"max_live":2}}`)
	var lead string
	refuseBundles(t, func(l string) *Refusal {
		if l == lead {
			return budgetRefusal()
		}
		return nil
	})
	first := mustDrain(t, repo, Options{}, DrainOptions{})
	var m2 string
	lead, _, m2, _ = orderOf(t, first)
	second := judge(t, repo, Options{}, first, JudgementNo, "", noReason)
	third := judge(t, repo, Options{}, second, JudgementNo, "", noReason)
	if !slices.ContainsFunc(third.Passed, func(e Excluded) bool { return e.ID == lead && e.Check == CheckBudget }) {
		t.Fatalf("the bundle opened at the cap is refused and its lead passed over: %+v", third.Passed)
	}
	if third.Judging == nil || third.Judging.Issue != m2 {
		t.Fatalf("the move takes up the issue the cap stopped it at: %+v, next %s", third.Judging, third.Next)
	}
}
