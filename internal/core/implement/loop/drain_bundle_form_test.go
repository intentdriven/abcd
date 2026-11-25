package loop

import (
	"fmt"
	"os"
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
// with and start the lead's run; the run's members are the start's to write
// (loop.go), not this step's.
func recordBundles(t *testing.T) *[]openedBundle {
	t.Helper()
	var got []openedBundle
	was := startBundle
	startBundle = func(repoRoot, lead string, o Options, members []laneMember) (StartResult, error) {
		got = append(got, openedBundle{lead: lead, members: members})
		return startLane(repoRoot, lead, o)
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

// TestABundlesHandBacksAreRoutedByIssue: a member handed back alone is routed
// alone, once, while its lane goes on, and is in the lane's hand no longer; a
// whole-lane hand-back routes every issue the run still carries, the lead
// first, and not again the member routed alone.
func TestABundlesHandBacksAreRoutedByIssue(t *testing.T) {
	repo := drainRepo(t)
	at := time.Now().UTC().Truncate(time.Second)
	alone := HandBack{At: at, Kind: HandBackDesignFinding, Reason: "the guard needs a decision", Home: "the brief"}
	run := State{Key: formLead, Bundle: []Member{
		{Issue: formM1, Footprint: []string{"internal/a/f1.go"}, HandBack: &alone},
		{Issue: formM2, Footprint: []string{"internal/a/f2.go"}},
	}}
	l := DrainLane{Issue: formLead, RunID: "run-2611250000000001", OpenedAt: at, Outcome: DrainLaneInProgress, Bundle: []string{formM1, formM2}}
	st := DrainState{Lanes: []DrainLane{l}}
	blobs := &originBlobs{}

	routes, err := routeAlone(repo.Root(), st, l, run, nil, blobs, at.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 1 || routes[0].Issue != formM1 || routes[0].From != "lane" || routes[0].Kind != HandBackDesignFinding {
		t.Fatalf("the member handed back alone is routed alone: %+v", routes)
	}
	st.HandBacks = routes
	if again, err := routeAlone(repo.Root(), st, l, run, nil, blobs, at.Add(2*time.Minute)); err != nil || len(again) != 0 {
		t.Fatalf("a member routed alone is not routed again: %+v %v", again, err)
	}
	if !st.handedBackAloneIn(l, formM1) || st.handedBackAloneIn(l, formM2) {
		t.Fatal("only the member routed alone is out of the lane's hand")
	}
	if !st.taken(formM2, blobs) || !st.taken(formLead, blobs) {
		t.Fatal("the lane's lead and its member still held are not counted as taken")
	}

	whole := HandBack{At: at, Kind: HandBackDesignFinding, Reason: "the lane needs a decision", Home: "the brief"}
	routes, err = routeLaneHandBack(repo.Root(), st, l, run, whole, nil, blobs, at.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, r := range routes {
		ids = append(ids, r.Issue)
		if r.Reason != whole.Reason {
			t.Fatalf("each issue still held is routed by the lane's hand-back: %+v", r)
		}
	}
	if !slices.Equal(ids, []string{formLead, formM2}) {
		t.Fatalf("a whole-lane hand-back routes every issue still held, the lead first: %v", ids)
	}

	single := State{Key: formLead}
	one := DrainLane{Issue: formLead, RunID: "run-2611250000000002", OpenedAt: at, Outcome: DrainLaneInProgress}
	routes, err = routeLaneHandBack(repo.Root(), DrainState{Lanes: []DrainLane{one}}, one, single, whole, nil, blobs, at.Add(time.Minute))
	if err != nil || len(routes) != 1 || routes[0].Issue != formLead {
		t.Fatalf("a lane of one issue routes its issue alone: %+v %v", routes, err)
	}
}
