package loop

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The build loop's trial mode (itd-2610161733316518 decision 4,
// spc-2610161949559050 step 3): a model-matrix copy runs the loop's stages
// taking no reservation and landing nothing, so copies of one intent never
// collide on its footprint and none of them lands.

// TestATrialRunTakesNoReservation: a run started in trial mode on a checkout
// with an origin pushes no reservation, records none in its state, and says
// in its record that it is a trial; resumed, it reserves nothing either.
func TestATrialRunTakesNoReservation(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), footprintedSpec)
	bare := onOrigin(t, repo)

	frozen := repo.Git("rev-parse", "HEAD")
	res, err := Start(repo.Root(), "itd-10", Options{Trial: true, Frozen: frozen})
	if err != nil {
		t.Fatal(err)
	}
	if tip := originReservation(t, bare, reservedRef); tip.commit != "" {
		t.Fatalf("a trial run pushes no reservation: %+v", tip)
	}
	st, err := ReadState(repo.Root(), res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Trial || st.ReservedAt != nil {
		t.Fatalf("the state is a trial's and records no reservation: trial %v, reserved %v", st.Trial, st.ReservedAt)
	}
	if res.Reservation != "" {
		t.Fatalf("the start names no reservation: %q", res.Reservation)
	}
	if note := recordNote(t, repo, res.RunID, StageTrial); !strings.Contains(note, "takes no reservation and lands nothing") {
		t.Fatalf("the record says the run is a trial: %q", note)
	}

	again, err := Start(repo.Root(), "itd-10", Options{Trial: true, Frozen: frozen})
	if err != nil || !again.Resumed || again.RunID != res.RunID {
		t.Fatalf("starting again resumes the trial: %+v %v", again, err)
	}
	if tip := originReservation(t, bare, reservedRef); tip.commit != "" {
		t.Fatalf("a resumed trial reserves nothing either: %+v", tip)
	}
}

// TestATrialRunIsNeverResumedAsABuildNorABuildAsATrial: a trial and a build of
// one intent are different runs, so neither start resumes the other's.
func TestATrialRunIsNeverResumedAsABuildNorABuildAsATrial(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), footprintedSpec)
	trial, err := Start(repo.Root(), "itd-10", Options{Trial: true, Frozen: repo.Git("rev-parse", "HEAD")})
	if err != nil {
		t.Fatal(err)
	}
	build, err := Start(repo.Root(), "itd-10", Options{})
	if err == nil && build.Resumed && build.RunID == trial.RunID {
		t.Fatalf("a build resumed the trial %s", trial.RunID)
	}
}

// TestATrialLaneLandsNothing: the landing stage of a trial lane pushes,
// merges and records nothing to land, and completes with a line saying so,
// whatever the lane holds.
func TestATrialLaneLandsNothing(t *testing.T) {
	lane := &Lane{ID: "lane-1", Branch: "main", Stage: StageLand}
	out, err := landStage(Context{State: State{Trial: true}}, lane)
	if err != nil {
		t.Fatal(err)
	}
	if out.Await != nil || out.HandBack != nil || out.Stay || out.Goto != "" {
		t.Fatalf("a trial's landing completes at once: %+v", out)
	}
	if !strings.Contains(out.Note, "lands nothing") {
		t.Fatalf("the landing says it landed nothing: %q", out.Note)
	}
	if lane.Landing != nil {
		t.Fatalf("a trial's lane starts no landing: %+v", lane.Landing)
	}
}

// asTrial makes the run a trial's, as a trial start writes it, for a test
// that drives a lane to its implementer first.
func asTrial(t *testing.T, root, id string) {
	t.Helper()
	if err := mutate(root, id, func(_ *os.Root, st *State) (bool, error) {
		st.Trial = true
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
}

// TestATrialsRunnerFailureIsTheCellsFailure: inside a trial, the routed
// runner's failure ends the lane as the cell's failure. Nothing falls back,
// so no fallback receipt is written; the role is handed to no one, the host
// included; and no later step retries it, on its runner or on the host.
func TestATrialsRunnerFailureIsTheCellsFailure(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	asTrial(t, root, id)
	env := newDriveEnv(t, "notloggedin", "claude")
	routed := Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}}
	res, err := Drive(context.Background(), root, id, steps, o, routed)
	if err != nil {
		t.Fatal(err)
	}
	if env.launches(t, "claude") != 1 {
		t.Fatalf("the cell's runner is started once: %d", env.launches(t, "claude"))
	}
	if res.Awaiting != nil || res.Paused != nil {
		t.Fatalf("the cell's role is handed to no one and not paused for a retry: awaiting %+v paused %+v", res.Awaiting, res.Paused)
	}
	if res.HandBack == nil || res.HandBack.Verdict != VerdictCellFailed || !strings.Contains(res.HandBack.Reason, "Not logged in") {
		t.Fatalf("the lane ends as the cell's failure, naming the runner's: %+v", res.HandBack)
	}
	st, err := ReadState(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Fallbacks) != 0 {
		t.Fatalf("nothing fell back, so no fallback receipt is written: %+v", st.Fallbacks)
	}
	if !recordSays(t, root, id, string(StageHandedBack), "cell", "Not logged in") {
		t.Fatal("the run record names the cell's failure")
	}
	for name, rs := range map[string]Runners{
		"its runner": routed,
		"the host":   {Config: runnerConfig(t, "", ""), Transcripts: &memTranscripts{}},
	} {
		res, _ := Drive(context.Background(), root, id, steps, o, rs)
		if env.launches(t, "claude") != 1 || res.Awaiting != nil {
			t.Fatalf("no later step retries the cell's role on %s: %d launches, awaiting %+v", name, env.launches(t, "claude"), res.Awaiting)
		}
	}
}

// TestACellFailedTrialLaneLeavesThePickUnfalsified: a lane handed back as the
// cell's failure says nothing of whether the pick was achievable, so a run
// with a pick whose trial lane is handed back as cell-failed neither records
// the pick as falsified nor reports it so: the pick's falsifier is the
// fix-round bound, which a cell's failure never met.
func TestACellFailedTrialLaneLeavesThePickUnfalsified(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	if err := mutate(root, id, func(_ *os.Root, st *State) (bool, error) {
		st.Trial = true
		st.Pick = &RunPick{Lane: "lane-1", Entry: "picked by run " + id}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	newDriveEnv(t, "notloggedin", "claude")
	res, err := Drive(context.Background(), root, id, steps, o, Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatal(err)
	}
	if res.HandBack == nil || res.HandBack.Verdict != VerdictCellFailed {
		t.Fatalf("the lane ends as the cell's failure: %+v", res.HandBack)
	}
	st, err := ReadState(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if lane, hb, ok := pickFalsifiedBy(st); ok {
		t.Fatalf("a cell-failed lane falsified the pick: %s handed back as %s", lane, hb.Verdict)
	}
	if p := recordPick(st); p == nil || p.Falsified || p.FalsifiedBy != "" {
		t.Fatalf("the run record keeps the pick unfalsified: %+v", p)
	}
	for _, e := range st.Record {
		if e.Stage == "pick" {
			t.Fatalf("the run record names no pick line for a cell's failure: %+v", e)
		}
	}
}

// TestATrialsProviderOutageIsRetriedOnItsRoute: a provider down before the
// copy began pauses the trial's role for its own runner with no fallback
// receipt, and a role so paused is never released to the host, even once the
// person's route moves the role there.
func TestATrialsProviderOutageIsRetriedOnItsRoute(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	asTrial(t, root, id)
	newDriveEnv(t, "apicrash", "claude")
	res, err := Drive(context.Background(), root, id, steps, o, Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Paused == nil || res.HandBack != nil {
		t.Fatalf("the outage pauses the role for its runner: paused %+v hand-back %+v", res.Paused, res.HandBack)
	}
	st, err := ReadState(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Fallbacks) != 0 {
		t.Fatalf("an outage in a cell writes no fallback receipt: %+v", st.Fallbacks)
	}
	res, err = Drive(context.Background(), root, id, steps, o, Runners{Config: runnerConfig(t, "", ""), Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Awaiting != nil {
		t.Fatalf("a trial's paused role is never released to the host: %+v", res.Awaiting)
	}
	if recordSays(t, root, id, StageUnpaused) {
		t.Fatal("the record releases no trial role to the host")
	}
}

// TestATrialStoppedMidLaunchIsNotTheCellsFailure: a launch the runner kills
// because the run's own context ended (Ctrl-C, SIGTERM) says nothing of the
// cell. The lane is not handed back as the cell's failure: the role is paused
// for its own runner, as a build's is, with no fallback receipt.
func TestATrialStoppedMidLaunchIsNotTheCellsFailure(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	asTrial(t, root, id)
	env := newDriveEnv(t, "hang", "claude")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		// The person stops the run once the cell's runner is up.
		for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
			if _, err := os.Stat(filepath.Join(env.log, "claude.pid")); err == nil {
				cancel()
				return
			}
		}
	}()
	res, err := Drive(ctx, root, id, steps, o, Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}})
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if ctx.Err() == nil {
		t.Fatal("the run was never stopped: the runner did not start")
	}
	if res.HandBack != nil {
		t.Fatalf("a run's own stop is never the cell's failure: %+v", res.HandBack)
	}
	st, rerr := ReadState(root, id)
	if rerr != nil {
		t.Fatal(rerr)
	}
	if hb := st.Lanes[0].HandBack; hb != nil {
		t.Fatalf("the lane is not handed back: %+v", hb)
	}
	if recordSays(t, root, id, string(StageHandedBack), "cell") {
		t.Fatal("the record names no cell failure")
	}
	if len(st.Fallbacks) != 0 {
		t.Fatalf("nothing fell back, so no fallback receipt is written: %+v", st.Fallbacks)
	}
	if a := st.Lanes[0].Awaits; len(a) == 0 || a[0].Paused == nil {
		t.Fatalf("the role is paused for its own runner, as a build's is: %+v", a)
	}
}

// trialClone is a copy's clone of src's head, as the model matrix makes one:
// a fresh repository holding that commit alone on a detached HEAD, with no
// branch, tag or remote, given the empty local tier the run lives in.
func trialClone(t *testing.T, src *gittest.Repo) (*gittest.Repo, string) {
	t.Helper()
	frozen := src.Git("rev-parse", "HEAD")
	c := gittest.NewRepo(t)
	c.Git("fetch", "-q", "--no-tags", "--depth=1", src.Root(), frozen)
	c.Git("-c", "advice.detachedHead=false", "checkout", "-q", "--detach", frozen)
	if err := os.MkdirAll(filepath.Join(c.Root(), filepath.FromSlash(TierRelDir)), 0o700); err != nil {
		t.Fatal(err)
	}
	return c, frozen
}

// TestATrialLaneIsCutFromTheFrozenCommit: a trial names its frozen commit and
// runs in its copy's clone, which has no default branch; its lane is cut from
// that commit, and the run records it.
func TestATrialLaneIsCutFromTheFrozenCommit(t *testing.T) {
	src := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	clone, frozen := trialClone(t, src)
	res, err := Start(clone.Root(), "itd-10", Options{Trial: true, Frozen: frozen})
	if err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(clone.Root(), res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Frozen != frozen {
		t.Fatalf("the trial records its frozen commit: %q, want %s", st.Frozen, frozen)
	}
	lane := st.Lanes[0]
	if _, err := worktreeStage(Context{RepoRoot: clone.Root(), RunDir: runRel(st.RunID), State: st}, &lane); err != nil {
		t.Fatalf("a trial's lane is cut in its clone, which has no default branch: %v", err)
	}
	if lane.BaseSHA != frozen {
		t.Fatalf("the lane is cut from the frozen commit: %s, want %s", lane.BaseSHA, frozen)
	}
}

// TestATrialStartNamesItsFrozenCommit: a trial is refused without a frozen
// commit, with one that is not the checkout's head, or over a spec that
// leaves more than one step to build; only a trial names a frozen commit.
// Nothing is written.
func TestATrialStartNamesItsFrozenCommit(t *testing.T) {
	src := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	head := src.Git("rev-parse", "HEAD")
	other := strings.Repeat("a", len(head))
	for name, o := range map[string]Options{
		"no frozen commit":            {Trial: true},
		"a short one":                 {Trial: true, Frozen: head[:12]},
		"one that is not the head":    {Trial: true, Frozen: other},
		"a frozen commit for a build": {Frozen: head},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Start(src.Root(), "itd-10", o)
			mustRefusal(t, err)
			runTierAbsent(t, src.Root())
		})
	}
	t.Run("a spec of several steps", func(t *testing.T) {
		repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps("1. First\n2. Second\n"))
		_, err := Start(repo.Root(), "itd-10", Options{Trial: true, Frozen: repo.Git("rev-parse", "HEAD")})
		if r := mustRefusal(t, err); !strings.Contains(r.Reason, "one step") {
			t.Fatalf("the refusal says a trial builds one step: %+v", r)
		}
		runTierAbsent(t, repo.Root())
	})
}
