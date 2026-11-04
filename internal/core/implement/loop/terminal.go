package loop

// terminal.go is the terminal loop (spc-2610230813348813 step 2;
// adr-2610222222326104): `abcd drain --drive`'s core, the moves a host would
// make between the drain's steps made here instead. Each iteration is one move
// of the drain, under the drain's lock for that move alone; a lane in progress
// is driven in-process as `abcd implement step` drives it, through Drive and
// DriveReview, its receipts recorded as they record them; a landing that waits
// for its receipt has its preflight run; a wait a move names is slept out with
// the lock free. The runners are whatever Runners holds, so the walled launch
// is one value the front door hands in.

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// Why a terminal run ended that the drain's own Stopped does not say.
const (
	// TerminalStopped: a stop was asked for.
	TerminalStopped = "stop"
	// TerminalOutage: failures ran through the outage schedule to its limit.
	TerminalOutage = "runner-outage"
	// TerminalWall: a wall checked before a stage no longer held.
	TerminalWall = "wall"
)

// TerminalOptions are one terminal run's inputs.
type TerminalOptions struct {
	// Options are the loop's seams and configuration, Runners among them.
	Options Options
	// Drain are the drain's own flags; Context is set to the run's.
	Drain DrainOptions
	// Runners starts each role.
	Runners Runners
	// Stages are the lane's stages; DefaultStages when nil.
	Stages Stages
	// Review is the reviewer route, as `implement step` hands it.
	Review *ReviewRoute
	// Preflight runs the repository's preflight in a lane's worktree, behind
	// the load gate.
	Preflight func(ctx context.Context, worktree string) error
	// Stop is closed when a stop is asked for. The context stays live, so a
	// call in flight finishes.
	Stop <-chan struct{}
	// BeforeStage is called before each move and each step, so before any
	// stage starts; an error ends the run, TerminalWall. Nil checks nothing.
	BeforeStage func() error
	// Say prints one line, already plain; nil says nothing.
	Say func(string)
	// Sleep waits; nil sleeps until the time passes or the stop or the
	// context ends it.
	Sleep func(ctx context.Context, d time.Duration)
}

// TerminalResult is how one terminal run ended.
type TerminalResult struct {
	// Ended is why: the drain's Stopped, TerminalStopped or TerminalOutage.
	Ended string `json:"ended"`
	// Run, Issue and Lane name the lane left on its branch, when one is.
	Run   string `json:"run,omitempty"`
	Issue string `json:"issue,omitempty"`
	Lane  string `json:"lane,omitempty"`
	// Last is the last move's result.
	Last DrainResult `json:"last"`
}

// terminal is one run in progress.
type terminal struct {
	repoRoot string
	o        TerminalOptions
	outage   terminalOutageClock
	last     DrainResult
}

// RunTerminal drives the drain from here until its cap, an empty ledger, the
// outage limit or a stop. An error is a refusal the run cannot go past.
func RunTerminal(ctx context.Context, repoRoot string, o TerminalOptions) (TerminalResult, error) {
	if o.Stages == nil {
		o.Stages = DefaultStages()
	}
	if o.Sleep == nil {
		o.Sleep = func(ctx context.Context, d time.Duration) { sleepOr(ctx, d, o.Stop) }
	}
	o.Drain.Context = ctx
	o.Drain.AskJudgeAgain = true
	if o.Drain.Transcripts == nil {
		o.Drain.Transcripts = o.Runners.Transcripts
	}
	t := &terminal{repoRoot: repoRoot, o: o}
	return t.run(ctx)
}

// say prints one line.
func (t *terminal) say(format string, args ...any) {
	if t.o.Say != nil {
		t.o.Say(termsafe.Sanitize(fsutil.RedactHome(fmt.Sprintf(format, args...))))
	}
}

func (t *terminal) stopped() bool {
	select {
	case <-t.o.Stop:
		return true
	default:
		return false
	}
}

func (t *terminal) now() time.Time { return t.o.Options.now() }

// wallHeld runs the check before a stage, saying what stopped the run.
func (t *terminal) wallHeld() bool {
	if t.o.BeforeStage == nil {
		return true
	}
	err := t.o.BeforeStage()
	if err != nil {
		t.say("stopping: %s", err.Error())
	}
	return err == nil
}

// sleep waits d, saying why.
func (t *terminal) sleep(ctx context.Context, d time.Duration, why string) {
	t.say("waiting %s: %s", d.Round(time.Second), why)
	t.o.Sleep(ctx, d)
}

// failed counts one failure and waits it out; it reports whether the outage
// has run its limit.
func (t *terminal) failed(ctx context.Context, judge bool, what string) bool {
	d, up := t.outage.fail(t.now(), judge)
	if up {
		t.say("outage: %s; eight hours of failures, so the run ends and nothing is handed back", what)
		return true
	}
	why := what + "; trying again"
	if t.outage.inOutage() {
		why = "outage: " + what + "; nothing is handed back"
	}
	t.sleep(ctx, d, why)
	return false
}

// run is the loop.
func (t *terminal) run(ctx context.Context) (TerminalResult, error) {
	for {
		if t.stopped() {
			return t.ended(TerminalStopped), nil
		}
		if !t.wallHeld() {
			return t.ended(TerminalWall), nil
		}
		res, err := Drain(t.repoRoot, t.o.Options, t.o.Drain)
		if err != nil {
			if r, ok := AsRefusal(err); ok && r.Contention && ctx.Err() == nil {
				t.sleep(ctx, TerminalWaitFloor, r.Reason)
				continue
			}
			return t.ended(TerminalStopped), err
		}
		t.last = res
		// What was given once is taken once: the pace, the skips.
		t.o.Options.Pace, t.o.Options.SubAgents, t.o.Options.FixRounds = nil, nil, nil
		t.o.Drain.Skip, t.o.Drain.SkipReason, t.o.Drain.Unskip = nil, "", nil
		t.sayMove(res)
		if res.Stopped != "" {
			return t.ended(res.Stopped), nil
		}
		if d, ok := moveWait(res, t.now()); ok {
			t.sleep(ctx, d, res.Next)
			continue
		}
		if res.Judged != nil {
			t.outage.succeed()
		}
		if res.Judging != nil {
			if t.failed(ctx, true, "the judge did not judge "+res.Judging.Issue) {
				return t.ended(TerminalOutage), nil
			}
			continue
		}
		if res.Lane == nil {
			t.sleep(ctx, TerminalWaitFloor, res.Next)
			continue
		}
		ended, err := t.driveLane(ctx, res.Lane.RunID)
		if err != nil {
			return t.ended(TerminalStopped), err
		}
		if ended != "" {
			return t.ended(ended), nil
		}
	}
}

// sayMove prints what a move did, one line for each thing.
func (t *terminal) sayMove(res DrainResult) {
	if j := res.Judged; j != nil {
		t.say("judged %s: %s", j.Issue, JudgementSummary(*j))
	}
	for _, r := range res.Routed {
		t.say("handed back: %s", DrainSummaryLine(r))
	}
	for _, f := range res.Finished {
		t.say("finished %s", f.Issue)
	}
	if res.Start != nil && res.Lane != nil {
		t.say("opened the lane of %s: run %s", res.Lane.Issue, res.Lane.RunID)
	}
}

// driveLane steps one lane until it has landed or been handed back, or the
// next move has something to say about it. It returns why the run ends, "" to
// go on to the next move.
func (t *terminal) driveLane(ctx context.Context, runID string) (string, error) {
	for {
		if t.stopped() {
			return TerminalStopped, nil
		}
		if !t.wallHeld() {
			return TerminalWall, nil
		}
		run, err := ReadState(t.repoRoot, runID)
		if err != nil {
			return "", err
		}
		if outcome, _, _ := laneOutcome(run); outcome != DrainLaneInProgress {
			return "", nil
		}
		res, err := Drive(ctx, t.repoRoot, runID, t.o.Stages, t.o.Options, t.o.Runners)
		if err == nil {
			res, err = DriveReview(ctx, t.repoRoot, runID, res, t.o.Stages, t.o.Options, t.o.Review)
		}
		if err != nil {
			again, ended, err := t.stepRefused(ctx, runID, err)
			if err != nil || ended != "" || !again {
				return ended, err
			}
			continue
		}
		switch {
		case res.Paused != nil:
			if t.failed(ctx, false, fmt.Sprintf("the %s of %s did not run: %s", res.Paused.Role, runID, pauseWhy(res.Paused))) {
				return TerminalOutage, nil
			}
		case res.Retried != nil && res.Retried.Paused != nil:
			if t.failed(ctx, false, fmt.Sprintf("the %s of %s did not run: %s", res.Retried.Role, runID, res.Retried.Paused.Why())) {
				return TerminalOutage, nil
			}
		case res.NextEligibleAt != nil:
			t.sleep(ctx, max(res.NextEligibleAt.Sub(t.now()), TerminalWaitFloor), res.Next)
		case res.Route != nil || res.PerformedStage != "":
			t.outage.succeed()
			t.sayStep(runID, res)
		default:
			t.sleep(ctx, TerminalWaitFloor, res.Next)
		}
	}
}

// pauseWhy is a paused await's reason.
func pauseWhy(a *Await) string {
	if a.Paused == nil {
		return "no reason recorded"
	}
	return a.Paused.Why()
}

// sayStep prints what one step did.
func (t *terminal) sayStep(runID string, res StepResult) {
	if res.Route != nil {
		t.say("%s %s: an agent ran on %s", runID, res.Lane, res.Route.Ran)
	}
	if res.PerformedStage != "" {
		t.say("%s %s: %s done", runID, res.Lane, res.PerformedStage)
	}
}

// stepRefused answers a refused step: a landing waiting for its receipt has
// its preflight run and is stepped again (again); another step's hold is
// waited out; any other refusal goes back to the next move, which reads the
// count the step logged and hands back a lane that is stuck.
func (t *terminal) stepRefused(ctx context.Context, runID string, err error) (again bool, ended string, _ error) {
	r, ok := AsRefusal(err)
	if !ok || ctx.Err() != nil {
		return false, "", err
	}
	if !r.checkWait.IsZero() {
		run, rerr := ReadState(t.repoRoot, runID)
		if rerr != nil {
			return false, "", rerr
		}
		i := run.current()
		if i < 0 {
			return false, "", nil
		}
		t.say("%s %s: running the preflight", runID, run.Lanes[i].ID)
		if perr := t.o.Preflight(ctx, run.Lanes[i].Worktree); perr != nil {
			if t.failed(ctx, false, "the preflight of "+runID+" did not mint its receipt: "+perr.Error()) {
				return false, TerminalOutage, nil
			}
			return true, "", nil
		}
		t.outage.succeed()
		return true, "", nil
	}
	t.sleep(ctx, TerminalWaitFloor, r.Reason)
	return false, "", nil
}

// ended is the run's result as it ends: the lane left in progress, if any.
func (t *terminal) ended(why string) TerminalResult {
	res := TerminalResult{Ended: why, Last: t.last}
	if st, live, err := readDrain(t.repoRoot); err == nil && live {
		for _, l := range st.Lanes {
			if l.Outcome == DrainLaneInProgress {
				res.Run, res.Issue = l.RunID, l.Issue
				if run, err := ReadState(t.repoRoot, l.RunID); err == nil {
					if i := run.current(); i >= 0 {
						res.Lane = run.Lanes[i].ID
					}
				}
			}
		}
	}
	return res
}

// DrainProgress is the drain in progress in this checkout, as a drive reads it
// before its yes.
type DrainProgress struct {
	Live  bool
	Max   int
	Lanes int
}

// ReadDrainProgress reads the drain in progress, if any.
func ReadDrainProgress(repoRoot string) (DrainProgress, error) {
	st, live, err := readDrain(repoRoot)
	if err != nil {
		return DrainProgress{}, err
	}
	return DrainProgress{Live: live, Max: st.Max, Lanes: len(st.Lanes)}, nil
}

// CapDrainInProgress gives a drain in progress that has no cap one: lanes more
// than it has opened. A drive is always bounded (adr-2610222222326104), and a
// cap is otherwise set only when a drain starts.
func CapDrainInProgress(repoRoot string, lanes int) error {
	return fsutil.WithFileLock(filepath.Join(repoRoot, filepath.FromSlash(drainLockRel)), lockTimeout, func() error {
		st, live, err := readDrain(repoRoot)
		if err != nil || !live || st.Max != 0 {
			return err
		}
		st.Max = len(st.Lanes) + lanes
		return writeDrain(repoRoot, st)
	})
}

// EstimateRuns is the agent runs a drive of lanes lanes may start, by the
// budget check's estimate.
func EstimateRuns(o Options, repoRoot string, lanes int) (int, error) {
	review, _, err := resolveReviewSet(o.roots(repoRoot), o)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, s := range estimateShares(o.Runners, lanes, false, &review) {
		total += s.runs
	}
	return total, nil
}
