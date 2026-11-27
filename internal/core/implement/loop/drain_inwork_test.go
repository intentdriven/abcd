package loop

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/layered"
)

// A drain holds its lanes in work to a limit (itd-2611251809580662 decision
// 2, spc-2611251810508418 step 2): drain.in_work_max, or --in-work, read once
// when a drain begins, from the repository's file at origin's default branch
// and never from a working tree, kept on the drain's state with its layer and
// origin. A lane in progress counts; one whose landing armed its merge, or one
// owing only its withdrawal, does not; a lane of several issues counts once.

// liveOnOrigin reports whether origin holds a live reservation of id, as a
// fresh view of the checkout at root reads it.
func liveOnOrigin(t *testing.T, root, id string) bool {
	t.Helper()
	if _, err := claim.Fetch(root, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	snap, err := claim.View(root, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	r, ok := snap.Lookup(id)
	return ok && r.State == claim.StateLive
}

// TestADrainOpensNothingBesideItsOneLaneAtImplement: with the bundled limit
// of one, a move over a lane at implement reserves nothing and opens nothing,
// and names the lane with both counts.
func TestADrainOpensNothingBesideItsOneLaneAtImplement(t *testing.T) {
	f := drainLandFixture(t)
	first := f.firstMove(t)
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	stepTo(t, f.repo, first.Start.RunID, f.stages, StageImplement)
	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Start != nil || res.Judging != nil {
		t.Fatalf("a lane at implement holds the drain's one lane in work: %+v", res)
	}
	if liveOnOrigin(t, f.repo.Root(), eligibleIssue) {
		t.Fatalf("the move reserved the next issue beside a lane in work")
	}
	if res.InWork != (LanesInWork{Count: 1, Limit: 1, Layer: "bundled", Origin: "bundled"}) {
		t.Fatalf("the move counts its one lane in work against the bundled one: %+v", res.InWork)
	}
	for _, want := range []string{"drive " + secondIssue + "'s lane", first.Start.RunID, "1 of 1 lane in work"} {
		if !strings.Contains(res.Next, want) {
			t.Fatalf("next names %q: %s", want, res.Next)
		}
	}
}

// TestALaneWhoseMergeIsArmedLeavesTheCount: once the lane's landing has armed
// its merge, the next move reserves the next issue and asks its judgement,
// while the first lane's reservation still stands.
func TestALaneWhoseMergeIsArmedLeavesTheCount(t *testing.T) {
	f := drainLandFixture(t)
	first := f.firstMove(t)
	f.runID, f.issue = first.Start.RunID, secondIssue
	f.armIssueLaneOnce(t)
	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Judging == nil || res.Judging.Issue != eligibleIssue {
		t.Fatalf("with the lane's merge armed the move takes up the next issue: %+v", res)
	}
	if res.InWork.Count != 0 {
		t.Fatalf("a lane whose merge is armed is not in work: %+v", res.InWork)
	}
	if !liveOnOrigin(t, f.repo.Root(), eligibleIssue) || !liveOnOrigin(t, f.repo.Root(), secondIssue) {
		t.Fatalf("the next issue is reserved while the first lane's reservation still stands")
	}
}

// TestALimitOfTwoOpensASecondLaneBesideOneInWork: with the repository's
// drain.in_work_max 2 at origin's default branch, a second lane opens beside
// one in work, the move naming both, and a third waits for one of them.
func TestALimitOfTwoOpensASecondLaneBesideOneInWork(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	ceilingOnOrigin(t, repo, `{"drain": {"in_work_max": 2}}`)
	first := drainToLane(t, repo, Options{}, DrainOptions{})
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	if first.InWork.Limit != 2 || first.InWork.Layer != "repo" || !strings.Contains(first.InWork.Origin, "origin/main") {
		t.Fatalf("the drain holds the repository's limit with where it came from: %+v", first.InWork)
	}
	second := drainToLane(t, repo, Options{}, DrainOptions{})
	if second.Start == nil || second.Start.RunID == first.Start.RunID {
		t.Fatalf("a second lane opens beside the one in work: %+v", second)
	}
	for _, want := range []string{second.Start.RunID, first.Start.RunID, "1 of 2 lanes in work"} {
		if !strings.Contains(second.Next, want) {
			t.Fatalf("the move's next names %q: %s", want, second.Next)
		}
	}
	if second.InWork.Count != 2 {
		t.Fatalf("the drain leaves two lanes in work: %+v", second.InWork)
	}
	third, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if third.Start != nil || third.Judging != nil || third.Complete {
		t.Fatalf("at the limit the move opens, asks and ends nothing: %+v", third)
	}
	for _, want := range []string{first.Start.RunID, second.Start.RunID, "2 of 2 lanes in work"} {
		if !strings.Contains(third.Next, want) {
			t.Fatalf("at the limit next names %q: %s", want, third.Next)
		}
	}
}

// TestALaneOwingOnlyItsWithdrawalIsNotCounted: a lane handed back with its
// pull request still armed is in progress only for the withdrawal it owes,
// and is not among the lanes in work; it is still settled before another lane
// opens: the next move, with no lane in work and the limit of one free,
// reserves, asks and opens nothing.
func TestALaneOwingOnlyItsWithdrawalIsNotCounted(t *testing.T) {
	f := drainLandFixture(t)
	first := f.firstMove(t)
	root := f.repo.Root()
	if iw, err := DrainInWork(root); err != nil || iw == nil || iw.Count != 1 {
		t.Fatalf("premise: the lane is in work: %+v %v", iw, err)
	}
	if err := mutate(root, first.Start.RunID, func(_ *os.Root, st *State) (bool, error) {
		i := st.current()
		st.Lanes[i].Stage, st.Lanes[i].PR = StageHandedBack, 7
		st.Lanes[i].Landing = &Landing{Merge: "queue", Armed: true}
		st.Lanes[i].HandBack = &HandBack{Kind: HandBackStuck}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	if run, err := ReadState(root, first.Start.RunID); err != nil || owedArming(run) != 7 {
		t.Fatalf("premise: the lane owes its arming's withdrawal: %v", err)
	}
	iw, err := DrainInWork(root)
	if err != nil || iw == nil || iw.Count != 0 || iw.Limit != 1 {
		t.Fatalf("a lane owing only its withdrawal is not in work: %+v %v", iw, err)
	}
	runs := runCount(t, f.repo)
	res, err := Drain(root, Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Start != nil || res.Judging != nil || runCount(t, f.repo) != runs || liveOnOrigin(t, root, eligibleIssue) {
		t.Fatalf("a lane owing its withdrawal holds the next lane back, though it is not counted: %+v", res)
	}
	if res.InWork.Count != 0 || !strings.Contains(res.Next, first.Start.RunID) {
		t.Fatalf("the move names the owed lane and counts none in work: %+v %s", res.InWork, res.Next)
	}
}

// TestALaneOfSeveralIssuesCountsOnce: a bundle's lane is one lane in work,
// whatever its members; a lane at its pull request or done is none.
func TestALaneOfSeveralIssuesCountsOnce(t *testing.T) {
	two := 2
	st := DrainState{InWorkMax: &PaceValue{Value: two, Layer: "flag", Origin: "--in-work 2"}, Lanes: []DrainLane{
		{Issue: "iss-2610010000000001", RunID: "run-unread-1", Outcome: DrainLaneInProgress, Bundle: []string{"iss-2610010000000002", "iss-2610010000000003"}},
		{Issue: "iss-2610010000000004", RunID: "run-unread-2", Outcome: DrainLanePullRequest},
		{Issue: "iss-2610010000000005", RunID: "run-unread-3", Outcome: DrainLaneDone},
	}}
	got := st.lanesInWork(t.TempDir(), DrainOptions{})
	if got != (LanesInWork{Count: 1, Limit: 2, Layer: "flag", Origin: "--in-work 2"}) {
		t.Fatalf("a bundle's lane counts once, and a lane at its pull request none: %+v", got)
	}
}

// TestTheInWorkLimitIsTheFlagsThenTheRepositorysThenOne: --in-work wins over
// the repository's figure at origin's default branch, which wins over the
// bundled one; a working tree's file and a branch that is not origin's
// default change nothing.
func TestTheInWorkLimitIsTheFlagsThenTheRepositorysThenOne(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	root := repo.Root()
	roots := layered.Roots{Repo: root, Home: t.TempDir()}
	if v, _, err := resolveInWorkMax(root, roots, 0); err != nil || v != bundledInWorkMax() {
		t.Fatalf("with nothing set the limit is the bundled one: %+v %v", v, err)
	}
	repo.Write(".abcd/config.json", `{"drain": {"in_work_max": 9}}`)
	ceilingOnBranch(t, repo, "develop", `{"drain": {"in_work_max": 8}}`)
	if v, _, err := resolveInWorkMax(root, roots, 0); err != nil || v != bundledInWorkMax() {
		t.Fatalf("a working tree's file and another branch on origin change nothing: %+v %v", v, err)
	}
	ceilingOnOrigin(t, repo, `{"drain": {"in_work_max": 3}}`)
	v, _, err := resolveInWorkMax(root, roots, 0)
	if err != nil || v.Value != 3 || v.Layer != "repo" || v.Origin != ".abcd/config.json at origin/main" {
		t.Fatalf("the repository's figure at origin's default branch sets the limit: %+v %v", v, err)
	}
	v, _, err = resolveInWorkMax(root, roots, 5)
	if err != nil || v != (PaceValue{Value: 5, Layer: "flag", Origin: "--in-work 5"}) {
		t.Fatalf("--in-work wins over the repository's figure: %+v %v", v, err)
	}
}

// TestAnInWorkLimitOutOfRangeIsRefusedNamingItsOrigin: 0, 11 and a string in
// the repository's file at origin's default branch, and --in-work 11, are
// refused naming where they came from.
func TestAnInWorkLimitOutOfRangeIsRefusedNamingItsOrigin(t *testing.T) {
	for _, body := range []string{`{"drain": {"in_work_max": 0}}`, `{"drain": {"in_work_max": 11}}`, `{"drain": {"in_work_max": "two"}}`} {
		t.Run(body, func(t *testing.T) {
			repo, _ := reservingDrainRepo(t, "commands/drain.md")
			ceilingOnOrigin(t, repo, body)
			_, _, err := resolveInWorkMax(repo.Root(), layered.Roots{Repo: repo.Root(), Home: t.TempDir()}, 0)
			rf := mustRefusal(t, err)
			if !strings.Contains(rf.Reason, ".abcd/config.json at origin/main") || !strings.Contains(rf.Reason, "drain.in_work_max") {
				t.Fatalf("the refusal names the file at origin's default branch: %q", rf.Reason)
			}
		})
	}
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	_, err := Drain(repo.Root(), Options{}, DrainOptions{InWork: 11})
	if rf := mustRefusal(t, err); !strings.Contains(rf.Reason, "--in-work 11") {
		t.Fatalf("--in-work 11 is refused naming the flag: %q", rf.Reason)
	}
}

// TestADrainInWorkMaxInTheMachinesFileIsRefused: the machine's file does not
// hold the limit, and one there is refused naming the file, never passed over.
func TestADrainInWorkMaxInTheMachinesFileIsRefused(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	home := t.TempDir()
	writeConfigAt(t, abcdhome.Path(home, "config.json"), `{"drain": {"in_work_max": 1}}`)
	_, err := Drain(repo.Root(), Options{Roots: &layered.Roots{Repo: repo.Root(), Home: home}}, DrainOptions{})
	rf := mustRefusal(t, err)
	if !strings.Contains(rf.Reason, "config.json sets drain.in_work_max") || !strings.Contains(rf.Reason, "does not hold") {
		t.Fatalf("the machine's drain.in_work_max is refused naming the file: %q", rf.Reason)
	}
	if _, live, err := readDrain(repo.Root()); err != nil || live {
		t.Fatalf("no drain began: %v %v", live, err)
	}
}

// TestBothDrainResolversClaimBothKeys: a file setting drain.in_work_max is
// not refused by the bundle cap's resolver, and one setting drain.bundle_max
// is not refused by the lanes-in-work limit's.
func TestBothDrainResolversClaimBothKeys(t *testing.T) {
	if _, _, err := resolveBundleMax(bundleRoots(t, `{"drain":{"in_work_max":2}}`, `{"drain":{"bundle_max":2}}`)); err != nil {
		t.Fatalf("the bundle cap's resolver refuses drain.in_work_max: %v", err)
	}
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	ceilingOnOrigin(t, repo, `{"drain": {"bundle_max": 1, "in_work_max": 2}}`)
	home := t.TempDir()
	writeConfigAt(t, abcdhome.Path(home, "config.json"), `{"drain": {"bundle_max": 3}}`)
	v, _, err := resolveInWorkMax(repo.Root(), layered.Roots{Repo: repo.Root(), Home: home}, 0)
	if err != nil || v.Value != 2 {
		t.Fatalf("the lanes-in-work limit's resolver refuses drain.bundle_max: %+v %v", v, err)
	}
}

// TestTheInWorkLimitIsKeptForTheDrainsLife: the drain keeps the limit it
// began with whatever origin's file says by then, and --in-work naming
// another while it runs is refused.
func TestTheInWorkLimitIsKeptForTheDrainsLife(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	ceilingOnOrigin(t, repo, `{"drain": {"in_work_max": 2}}`)
	first := drainToLane(t, repo, Options{}, DrainOptions{})
	if first.Start == nil {
		t.Fatalf("the drain opens a lane: %+v", first)
	}
	ceilingOnOrigin(t, repo, `{"drain": {"in_work_max": 4}}`)
	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.InWork.Limit != 2 {
		t.Fatalf("a running drain keeps its limit whatever the file says: %+v", res.InWork)
	}
	st, _, err := readDrain(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if st.InWorkMax == nil || st.InWorkMax.Value != 2 || st.InWorkMax.Layer != "repo" {
		t.Fatalf("the drain's state keeps the limit with its layer: %+v", st.InWorkMax)
	}
	_, err = Drain(repo.Root(), Options{}, DrainOptions{InWork: 3})
	if rf := mustRefusal(t, err); !strings.Contains(rf.Reason, "--in-work 3 names another") {
		t.Fatalf("--in-work naming another limit while the drain runs is refused: %q", rf.Reason)
	}
	if _, err := Drain(repo.Root(), Options{}, DrainOptions{InWork: 2}); err != nil {
		t.Fatalf("--in-work naming the drain's own limit is taken: %v", err)
	}
}

// TestAnOriginThatCannotBeAskedLeavesOneLaneInWork: where origin cannot be
// asked which branch is its default, no repository figure is read, the drain
// holds the bundled one, the lowest the setting can be, and its next step
// says why.
func TestAnOriginThatCannotBeAskedLeavesOneLaneInWork(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	ceilingOnOrigin(t, repo, `{"drain": {"in_work_max": 4}}`)
	unanswerable := func(string, string) (claim.OriginDefault, error) {
		return claim.OriginDefault{}, os.ErrDeadlineExceeded
	}
	v, note, err := resolveInWorkMaxAsking(repo.Root(), layered.Roots{Repo: repo.Root(), Home: t.TempDir()}, 0, unanswerable)
	if err != nil || v != bundledInWorkMax() || !strings.Contains(note, "could not be asked") {
		t.Fatalf("an origin that cannot be asked leaves the bundled one, said: %+v %q %v", v, note, err)
	}
}

// TestADriveHoldsOneLaneInWorkWhateverTheRepositorysLimit: under a drive
// (driveEnds, which only a terminal drive sets), a repository's
// drain.in_work_max 3 at origin's default branch opens no second lane beside
// the one the drive drives: the drive returns to its lane in progress after a
// refused step or a contended driver, and a lane opened beside it would sit
// reserved with no one driving it. The drain records the drive's limit as its
// own, with layer drive, and names it in the next step.
func TestADriveHoldsOneLaneInWorkWhateverTheRepositorysLimit(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	ceilingOnOrigin(t, repo, `{"drain": {"in_work_max": 3}}`)
	drive := DrainOptions{driveEnds: true}
	first := drainToLane(t, repo, Options{}, drive)
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drive's drain opens the documentation issue's lane: %+v", first)
	}
	if first.InWork.Limit != 1 || first.InWork.Layer != "drive" {
		t.Fatalf("a drive's drain holds one lane in work, from the drive: %+v", first.InWork)
	}
	second, err := Drain(repo.Root(), Options{}, drive)
	if err != nil {
		t.Fatal(err)
	}
	if second.Start != nil || second.Judging != nil || runCount(t, repo) != 1 {
		t.Fatalf("under a drive the move opens and asks nothing beside its lane in work: %+v", second)
	}
	for _, want := range []string{first.Start.RunID, "1 of 1 lane in work", "a drive drives one lane at a time"} {
		if !strings.Contains(second.Next, want) {
			t.Fatalf("next names %q: %s", want, second.Next)
		}
	}
	st, live, err := readDrain(repo.Root())
	if err != nil || !live || st.InWorkMax == nil || st.InWorkMax.Layer != "drive" || st.InWorkMax.Value != 1 {
		t.Fatalf("the drain's state records the drive's limit: %+v %v %v", st.InWorkMax, live, err)
	}
}

// TestADriveTakingUpADrainOfThreeHoldsOne: a drain begun by hand with
// --in-work 3 that a drive takes up holds one lane in work from the drive's
// first move, its state recording the drive's limit and the move saying the
// limit it narrowed.
func TestADriveTakingUpADrainOfThreeHoldsOne(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	first := drainToLane(t, repo, Options{}, DrainOptions{InWork: 3})
	if first.Start == nil || first.InWork.Limit != 3 {
		t.Fatalf("premise: a drain by hand holds the flag's three: %+v", first)
	}
	res, err := Drain(repo.Root(), Options{}, DrainOptions{driveEnds: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Start != nil || res.Judging != nil || runCount(t, repo) != 1 {
		t.Fatalf("the drive's first move opens and asks nothing beside the lane in work: %+v", res)
	}
	if res.InWork != (LanesInWork{Count: 1, Limit: 1, Layer: "drive", Origin: driveInWorkOrigin}) {
		t.Fatalf("the drive's move counts against the drive's one: %+v", res.InWork)
	}
	if !strings.Contains(res.Next, "--in-work 3") || !strings.Contains(res.Next, "a drive drives one lane at a time") {
		t.Fatalf("next says the drive narrowed the limit it started with: %s", res.Next)
	}
	if iw, err := DrainInWork(repo.Root()); err != nil || iw == nil || iw.Limit != 1 || iw.Layer != "drive" {
		t.Fatalf("abcd peers reads the drive's limit from the state: %+v %v", iw, err)
	}
}
