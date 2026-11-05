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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/runner"
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
	// TerminalRefused: a move refused in a way the run cannot go past.
	TerminalRefused = "refused"
	// TerminalHalted: a stop set from outside this terminal holds
	// (itd-2611041549471712). What the run holds is that stop's to let go.
	TerminalHalted = "halted"
	// TerminalLimited: the account's rate limit outlasts what a drive waits.
	TerminalLimited = "rate-limited"
	// TerminalWaited: nothing moved for as long as a drive waits.
	TerminalWaited = "waited"
	// TerminalUntil: the bound the person set with --until has passed.
	TerminalUntil = "until"
	// TerminalKilled: the context ended, a second signal: the agent's process
	// group was killed, and what it held lapses on its own.
	TerminalKilled = "killed"
)

// terminalHaltStage is the stage a move or a step refuses at while a stop set
// from outside this terminal holds.
const terminalHaltStage = "stop"

// TerminalResumeCommand resumes a run that left a lane.
const TerminalResumeCommand = "abcd drain --drive"

// StageDriveReleased is the run record's stage for a reservation a drive let
// go as it ended, which the drive that takes the lane up reserves again.
const StageDriveReleased = "drive_released"

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
	// PreflightAllowed says whether a lane's preflight may run, given its
	// worktree and base; a *PreflightRefused hands the lane back. Nil allows all.
	PreflightAllowed func(worktree, base string) error
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
	// Until is the bound the person set (--until): the run ends at this time,
	// and no wait ends it before then. Zero keeps the eight-hour limit on each
	// wait (adr-2610222222326104's amendment of 2026-11-05).
	Until time.Time
	// Probe asks whether a limit that named no reset has lifted, for role, with
	// the cheapest launch that can tell. It reports true when it has; otherwise
	// the failure says what came back. Nil leaves the lane's next launch to find
	// out.
	Probe func(ctx context.Context, role, model string) (bool, *runner.Failure)
}

// errRunStateChanged is a run whose state changed while nothing of the driver's
// was writing it.
var errRunStateChanged = errors.New("the run's state changed while an agent was in its call, and only the driver writes it")

// HeldBackRel is the file that keeps what a drive held back.
const HeldBackRel = ".abcd/.work.local/logs/drive-held-back.log"

// JudgeKind is what became of a routed judgement.
type JudgeKind int

// The ends a routed judgement comes to that give no answer.
const (
	// JudgeNoOutcome: no routed judge was asked.
	JudgeNoOutcome JudgeKind = iota
	// JudgeNoAnswer: its runner ran it and it answered nothing usable.
	JudgeNoAnswer
	// JudgeRunnerFailed: its runner did not run it.
	JudgeRunnerFailed
	// JudgeLimited: the account's rate limit stopped it.
	JudgeLimited
)

// JudgeOutcome is what became of a routed judgement.
type JudgeOutcome struct {
	Kind JudgeKind
	// ResetsAt is when a limit resets, zero when none was reported.
	ResetsAt time.Time
}

// PreflightRefused is why a lane's preflight is not run.
type PreflightRefused struct{ Reason string }

func (e *PreflightRefused) Error() string { return e.Reason }

// TerminalResult is how one terminal run ended.
type TerminalResult struct {
	// Ended is why: the drain's Stopped, TerminalStopped or TerminalOutage.
	Ended string `json:"ended"`
	// Run, Issue and Lane name the lane left on its branch, when one is.
	Run   string `json:"run,omitempty"`
	Issue string `json:"issue,omitempty"`
	Lane  string `json:"lane,omitempty"`
	// Resume is the command that takes the lane up, when one is left.
	Resume string `json:"resume,omitempty"`
	// Lapses is when the lane's reservation lapses, when a kill left it held.
	Lapses *time.Time `json:"lapses,omitempty"`
	// Last is the last move's result.
	Last DrainResult `json:"last"`
	// HeldBack names each issue the run held back.
	HeldBack []string `json:"held_back,omitempty"`
}

// terminal is one run in progress.
type terminal struct {
	repoRoot string
	o        TerminalOptions
	outage   terminalOutageClock
	// limit is the outage's schedule run for a rate limit.
	limit terminalOutageClock
	last  DrainResult
	// idleSince is when the run last stopped moving, zero while it moves.
	idleSince time.Time
	// ranClean names the lane and head a preflight last exited 0 on.
	ranClean string
	// ctx is the run's, which says whether a failed launch was the run ending.
	ctx context.Context
	// evidence keeps what the run's first limit showed (limit_evidence.go).
	evidence *driveEvidence
	// lastWhy and lastSaid are the wait a line last named, and when.
	lastWhy  string
	lastSaid time.Time
	// waited adds up the waits since whose holds were last read, short ones
	// included and a machine asleep too, towards a check on the lane's claim.
	waited time.Duration
	// limitModel is the model of the launch that last met a limit.
	limitModel string
	// reached counts, by role, launches of the lane in hand that reached the
	// model and failed, in a row.
	reached map[string]int
	// call is the agent's call or preflight in flight, what its lines say.
	call terminalCall
	// unanswered counts, by issue, the judgements a runner ran and that gave no
	// answer; heldBack names each issue the run held back for that.
	unanswered map[string]int
	heldBack   []string
	// judge is what the last move's routed judgement came to.
	judge JudgeOutcome
	// minted names each head a preflight this run ran exited 0 on, and bases each
	// lane's base as the run first read it.
	minted map[string]bool
	bases  map[string]string
}

// terminalCall is what is in flight, as the line that says so reads it.
type terminalCall struct {
	mu          sync.Mutex
	what        string
	since, said time.Time
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
	return newTerminal(repoRoot, o).run(ctx)
}

// newTerminal is one run, not yet started.
func newTerminal(repoRoot string, o TerminalOptions) *terminal {
	t := &terminal{repoRoot: repoRoot, o: o, minted: map[string]bool{}, bases: map[string]string{}}
	t.o.Drain.JudgeOutcome = &t.judge
	t.o.Options.RunStateWatch = true
	t.o.Options.ReceiptOwn = func(sha string) bool { return t.minted[sha] }
	t.evidence = newDriveEvidence(repoRoot, func(line string) { t.say("%s", line) })
	t.o.Options.RunnerFailed = t.noteFailure
	t.o.Options.RunnerStarting = func(role string) { t.startCall("the " + role) }
	if t.bounded() {
		t.o.Drain.KeepWaiting = true
		t.outage.noLimit, t.limit.noLimit = true, true
	}
	return t
}

// bounded reports whether the person set the run's bound.
func (t *terminal) bounded() bool { return !t.o.Until.IsZero() }

// pastBound reports whether the person's bound has passed, saying so.
func (t *terminal) pastBound() bool {
	if !t.bounded() || t.now().Before(t.o.Until) {
		return false
	}
	t.say("stopping: --until named %s, and that has passed", t.o.Until.UTC().Format(time.RFC3339))
	return true
}

// noteFailure hears of each launch that failed. A limit the judge met is the move's
// to answer, and the first of each kind leaves its evidence.
func (t *terminal) noteFailure(role string, fl *runner.Failure) {
	if t.ctx != nil && t.ctx.Err() != nil {
		return
	}
	entry := evidenceEntry{Event: "failure", Role: role, Runner: fl.Runner, Response: fl.Detail, Seen: fl.Seen}
	if fl.Reason != runner.ReasonRateLimited {
		switch {
		case role == RoleDrainJudge:
		case t.bounded() && fl.ReachedModel():
			if t.reached == nil {
				t.reached = map[string]int{}
			}
			t.reached[role]++
		}
		if fl.Seen != nil {
			t.evidence.begin("suspect", entry, t.now())
		}
		return
	}
	t.limitModel = fl.Model
	entry.Event = "limit"
	if !fl.ResetsAt.IsZero() {
		entry.ResetRead = fl.ResetsAt.UTC().Format(time.RFC3339)
	}
	t.evidence.begin("limit", entry, t.now())
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

// haltedBy reports whether err is a stop set from outside this terminal, saying
// what holds and what lifts it. Sleeping through it, as through another
// contention, would keep the drive alive for ever under a stop.
func (t *terminal) haltedBy(err error) bool {
	r, ok := AsRefusal(err)
	if !ok || r.Stage != terminalHaltStage {
		return false
	}
	t.say("stopping: %s; %s", r.Reason, r.Remedy)
	return true
}

// sleep waits d, saying why, in stretches no longer than half a lease, renewing
// what the run's lanes hold between them, so a long wait keeps their claims. A
// stop, a kill or the person's bound ends it. The wait is measured twice, by the
// time slept and by the wall clock, and whichever ends it first stands: a machine
// that slept through part of it wakes on time, and a clock set back does not
// stretch it. A clock set forward ends it early, which the next launch answers.
func (t *terminal) sleep(ctx context.Context, d time.Duration, why string) {
	now := t.now()
	if t.bounded() {
		d = min(d, max(t.o.Until.Sub(now), 0))
	}
	end := now.Add(d)
	asked := d
	defer func() { t.waited += max(asked-d, t.now().Sub(now)) }()
	t.sayWait(d, why)
	t.renewHeld()
	for d > 0 {
		step := min(d, terminalRenewEvery, max(end.Sub(t.now()), time.Second))
		t.o.Sleep(ctx, step)
		d -= step
		if ctx.Err() != nil || t.stopped() {
			return
		}
		t.renewHeld()
		if !t.now().Before(end) {
			break
		}
		t.sayWait(d, why)
	}
}

// sayWait prints a wait's line when its reason changed and every so often while it
// stands, so a wait of days shows it is alive and a thirty-second poll is no flood.
func (t *terminal) sayWait(d time.Duration, why string) {
	now := t.now()
	if why == t.lastWhy && now.Sub(t.lastSaid) < terminalSayEvery {
		return
	}
	t.lastWhy, t.lastSaid = why, now
	t.say("waiting %s: %s", d.Round(time.Second), why)
}

// renewHeld renews each due reservation of a run in progress, but one a drive
// let go, and of a lane parked at its pull request, and says what failed.
func (t *terminal) renewHeld() {
	now := t.now()
	for _, r := range renewRunsAt(t.repoRoot, "", func(st State) bool { return !letGoByDrive(st) }, now) {
		if !r.Renewed {
			t.say("%s: %s", r.Key, r.Note)
		}
	}
	for _, note := range renewParkedDue(t.repoRoot, now) {
		t.say("%s", note)
	}
}

// idle waits d on something someone else must do, reporting true when nothing
// has moved for as long as a drive waits (adr-2610222222326104 condition 3): the
// run then ends, saying so. Under the person's bound the wait goes on, looking
// less often the longer it lasts, up to its cap.
func (t *terminal) idle(ctx context.Context, d time.Duration, why string) bool {
	now := t.now()
	if t.idleSince.IsZero() {
		t.idleSince = now
	}
	if t.bounded() {
		t.sleep(ctx, max(d, min(now.Sub(t.idleSince)/4, terminalIdleCap)), why)
		return false
	}
	if now.Sub(t.idleSince) >= terminalOutageLimit {
		t.say("stopping: nothing has moved for eight hours, as long as a drive waits: %s", why)
		return true
	}
	t.sleep(ctx, d, why)
	return false
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
	t.ctx = ctx
	for {
		if ctx.Err() != nil {
			return t.ended(TerminalKilled), nil
		}
		if t.stopped() {
			return t.ended(TerminalStopped), nil
		}
		if t.pastBound() {
			return t.ended(TerminalUntil), nil
		}
		if !t.wallHeld() {
			return t.ended(TerminalWall), nil
		}
		t.judge = JudgeOutcome{}
		res, err := Drain(t.repoRoot, t.o.Options, t.o.Drain)
		t.endCall()
		if err != nil {
			if ctx.Err() != nil {
				return t.ended(TerminalKilled), nil
			}
			// The terminal's interrupt reaches the git a move was running.
			if t.stopped() {
				return t.ended(TerminalStopped), nil
			}
			if t.haltedBy(err) {
				return t.ended(TerminalHalted), nil
			}
			// A push or fetch origin refused while reserving may pass; under a
			// bound that is waited out, as a peer's hold is.
			if r, ok := AsRefusal(err); ok && (r.Contention || (t.bounded() && r.originUnread)) {
				if t.idle(ctx, TerminalWaitFloor, r.Reason) {
					return t.ended(TerminalWaited), nil
				}
				continue
			}
			return t.ended(TerminalRefused), err
		}
		t.last = res
		// What was given once is taken once: the pace, the skips.
		t.o.Options.Pace, t.o.Options.SubAgents, t.o.Options.FixRounds = nil, nil, nil
		t.o.Drain.Skip, t.o.Drain.SkipReason, t.o.Drain.Unskip = nil, "", nil
		t.sayMove(res)
		if res.Stopped != "" {
			return t.ended(res.Stopped), nil
		}
		if res.Judged != nil {
			t.moved()
		}
		if res.Judged != nil || len(res.Routed) > 0 || len(res.Finished) > 0 || res.Start != nil {
			t.idleSince = time.Time{}
		}
		// A judge that met a limit is an agent like any other: none starts,
		// this one included, before the limit lets it.
		if t.judge.Kind == JudgeLimited && res.Judging != nil {
			if ended := t.limited(ctx, RoleDrainJudge, t.judge.ResetsAt); ended != "" {
				return t.ended(ended), nil
			}
			continue
		}
		if d, ok := moveWait(res, t.now()); ok {
			if res.WaitingUntil == nil {
				t.sleep(ctx, d, res.Next)
			} else if t.idle(ctx, d, res.Next) {
				return t.ended(TerminalWaited), nil
			}
			continue
		}
		if res.Judging != nil {
			if t.judge.Kind == JudgeNoAnswer && t.holdBack(res.Judging.Issue) {
				continue
			}
			if t.failed(ctx, true, "the judge did not judge "+res.Judging.Issue) {
				return t.ended(TerminalOutage), nil
			}
			continue
		}
		if res.Lane == nil {
			if t.idle(ctx, TerminalWaitFloor, res.Next) {
				return t.ended(TerminalWaited), nil
			}
			continue
		}
		ended, err := t.driveLane(ctx, res.Lane.RunID)
		if err != nil {
			return t.ended(TerminalRefused), err
		}
		if ended != "" {
			return t.ended(ended), nil
		}
	}
}

// holdBack counts one judgement of issue a runner ran that gave no answer, and at
// the limit has the next move hold it back so the run goes on to other work,
// releasing its claim and keeping the fact, since the skip list ends with the
// drain.
func (t *terminal) holdBack(issue string) bool {
	if t.unanswered == nil {
		t.unanswered = map[string]int{}
	}
	t.unanswered[issue]++
	if t.unanswered[issue] < terminalJudgeAsks {
		return false
	}
	delete(t.unanswered, issue)
	t.say("holding %s back: its judge did not answer %d times", issue, terminalJudgeAsks)
	t.o.Drain.Skip, t.o.Drain.SkipReason = []string{issue}, "its judge did not answer"
	t.heldBack = append(t.heldBack, issue)
	clause, _ := releaseReservation(t.repoRoot, issue, "", t.now())
	t.say("%s: %s", issue, clause)
	t.keepHeldBack(issue)
	return true
}

// keepHeldBack appends the fact to a file of its own.
func (t *terminal) keepHeldBack(issue string) {
	line := t.now().UTC().Format(time.RFC3339) + " " + issue + " its judge did not answer\n"
	if err := fsutil.EnsureRealDirAll(t.repoRoot, filepath.ToSlash(filepath.Dir(HeldBackRel)), 0o700); err == nil {
		f, err := os.OpenFile(filepath.Join(t.repoRoot, filepath.FromSlash(HeldBackRel)), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err == nil {
			_, _ = f.WriteString(line)
			_ = f.Close()
			return
		}
	}
	t.say("%s was held back, and that could not be kept in %s", issue, HeldBackRel)
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
	t.reached = nil
	if done, ended := t.takeUp(ctx, runID); done {
		return ended, nil
	}
	for {
		if ctx.Err() != nil {
			return TerminalKilled, nil
		}
		if t.stopped() {
			return TerminalStopped, nil
		}
		if t.pastBound() {
			return TerminalUntil, nil
		}
		if !t.wallHeld() {
			return TerminalWall, nil
		}
		if t.peerTook(runID) {
			return "", nil
		}
		run, err := ReadState(t.repoRoot, runID)
		if err != nil {
			return "", err
		}
		if outcome, _, _ := laneOutcome(run); outcome != DrainLaneInProgress {
			return "", nil
		}
		if i := run.current(); i >= 0 {
			lane := run.Lanes[i]
			key := runID + " " + lane.ID
			if base, seen := t.bases[key]; lane.BaseSHA != "" {
				switch {
				case !seen:
					t.bases[key] = lane.BaseSHA
				case base != lane.BaseSHA:
					t.say("stopping: %s", errRunStateChanged.Error())
					return TerminalWall, nil
				}
			}
			if lane.Stage == StageLand && t.o.PreflightAllowed != nil {
				var refused *PreflightRefused
				if perr := t.o.PreflightAllowed(lane.Worktree, t.bases[key]); errors.As(perr, &refused) {
					_, ended, err := t.laneRefused(ctx, runID, lane.ID, refused.Reason, refused.Reason)
					return ended, err
				} else if perr != nil {
					return "", perr
				}
			}
		}
		var res StepResult
		t.renewing(func() {
			res, err = Drive(ctx, t.repoRoot, runID, t.o.Stages, t.o.Options, t.o.Runners)
			if err == nil {
				res, err = DriveReview(ctx, t.repoRoot, runID, res, t.o.Stages, t.o.Options, t.o.Review)
			}
		})
		if ctx.Err() != nil {
			return TerminalKilled, nil
		}
		if errors.Is(err, errRunStateChanged) {
			t.say("stopping: %s", err.Error())
			return TerminalWall, nil
		}
		if t.poisoned(runID) {
			return "", nil
		}
		if err != nil {
			// The terminal's interrupt reaches the git a step was running.
			if t.stopped() {
				return TerminalStopped, nil
			}
			if t.haltedBy(err) {
				return TerminalHalted, nil
			}
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
		case res.RateLimit != nil:
			if ended := t.limited(ctx, res.RateLimit.Role, res.RateLimit.resetsAt); ended != "" {
				return ended, nil
			}
		case res.NextEligibleAt != nil:
			t.sleep(ctx, max(res.NextEligibleAt.Sub(t.now()), TerminalWaitFloor), res.Next)
		case res.Route != nil || res.PerformedStage != "":
			t.idleSince = time.Time{}
			t.moved()
			t.sayStep(runID, res)
		default:
			// No agent ran and no stage moved. Every wait has a deadline
			// (adr-2610222222326104), so this one runs on the outage's.
			if t.failed(ctx, false, fmt.Sprintf("%s did not move: %s", runID, res.Next)) {
				return TerminalOutage, nil
			}
		}
	}
}

// moved ends the counts a wait keeps: the outage's and the limit's. It is the moment
// a limit's evidence says the run resumed.
func (t *terminal) moved() {
	t.outage.succeed()
	t.limit.succeed()
	t.reached = nil
	t.evidence.resumed(t.now())
}

// poisoned hands a lane back when one role's launches have reached the model and
// failed too many times in a row, under a bound, where nothing else would end it:
// the fault is then the lane's, and the run goes on to other issues. Launches that
// never got that far say nothing of the lane and stay on the outage's wait.
func (t *terminal) poisoned(runID string) bool {
	for role, n := range t.reached {
		if n < terminalPoisonLaunches {
			continue
		}
		run, err := ReadState(t.repoRoot, runID)
		if err != nil {
			return false
		}
		r := refuse(StageRunner, "", "", fmt.Sprintf("the %s reached the model and failed %d times in a row", role, n), "see the reason")
		r.final = "launches of the same work failed the same way, so the lane goes back and the run goes on"
		t.stayAsItIs(runID, run.Key, r)
		t.reached = nil
		return true
	}
	return false
}

// limited waits out the account's rate limit, role's agent having met it, and says
// why the run ends instead, "" to go on. No agent starts during the wait, the
// judge's included. A reset the harness named is waited for, however far off under
// the person's bound; beyond what a drive waits with no bound, the run ends now,
// saying when it is. With none named the wait runs on the outage's schedule, capped
// under a bound.
//
// Under a bound a probe asks too, so the lane is not disturbed by asking: at each
// turn of that schedule, and at a low rate during a named wait, since a reset read
// from a stream can be wrong and a limit that lets a probe through is over. A
// probe that names a reset turns the wait into that one, and one rejected naming
// none leaves the wait as it was. Probes with no answer say nothing of the limit: a
// named wait goes on, and with none named a few in a row leave it to the lane's own
// launch, which also serves a harness whose probe can never answer.
func (t *terminal) limited(ctx context.Context, role string, at time.Time) string {
	misses, firstLook := 0, true
	model := t.limitModel
	probing := t.o.Probe != nil && t.bounded()
	for {
		now := t.now()
		d, up := t.limit.fail(now, true)
		if at.Sub(now) > terminalResetSane {
			t.say("the rate limit names a reset at %s, too far off to be one; it is treated as naming none", at.UTC().Format(time.RFC3339))
			at = time.Time{}
		}
		if at.After(now) {
			if !t.bounded() && at.Sub(t.limit.since) > terminalOutageLimit {
				t.say("stopping: the account's rate limit resets at %s, later than a drive waits; run %s then", at.UTC().Format(time.RFC3339), TerminalResumeCommand)
				return TerminalLimited
			}
			left, look := at.Sub(now), terminalNamedProbeEvery
			if firstLook && left > terminalFarReset {
				look = terminalFarResetFirstLook
			}
			firstLook = false
			why := fmt.Sprintf("the account's rate limit resets at %s", at.UTC().Format(time.RFC3339))
			if !probing || left <= look {
				t.evidence.add(evidenceEntry{Event: "decided", Decided: "wait until the reset named", NextLook: at.UTC().Format(time.RFC3339)}, now)
				t.sleep(ctx, max(left, TerminalWaitFloor), why)
				return ""
			}
			t.evidence.add(evidenceEntry{Event: "decided", Decided: "wait for the reset named, asking at a low rate", NextLook: now.Add(look).UTC().Format(time.RFC3339)}, now)
			t.sleep(ctx, look, why+"; it is asked again after this wait")
		} else {
			if up {
				t.say("stopping: the account has met its rate limit for eight hours and nothing says when it resets; run %s later", TerminalResumeCommand)
				return TerminalLimited
			}
			if t.bounded() {
				d = min(d, terminalProbeCap)
			}
			t.evidence.add(evidenceEntry{Event: "decided", Decided: "probe, since nothing says when it resets", NextLook: now.Add(d).UTC().Format(time.RFC3339)}, now)
			t.sleep(ctx, d, "the account has met its rate limit, and nothing says when it resets; it is asked again after this wait")
			if !probing {
				return ""
			}
		}
		if ctx.Err() != nil || t.stopped() || !t.now().Before(t.o.Until) {
			return ""
		}
		if !t.wallHeld() {
			return TerminalWall
		}
		renewed, fl := t.o.Probe(ctx, role, model)
		entry := evidenceEntry{Event: "probe", Answer: "no answer"}
		if fl != nil {
			entry.Seen = fl.Seen
		}
		switch {
		case ctx.Err() != nil:
			return ""
		case renewed:
			entry.Answer = "renewed"
			t.evidence.add(entry, t.now())
			t.say("the account's rate limit has lifted: a probe was answered")
			return ""
		case fl != nil && fl.Reason == runner.ReasonRateLimited:
			entry.Answer = "limited"
			misses = 0
			if !fl.ResetsAt.IsZero() {
				at = fl.ResetsAt
				entry.ResetRead = at.UTC().Format(time.RFC3339)
			}
			t.evidence.add(entry, t.now())
		default:
			t.evidence.add(entry, t.now())
			if !at.After(t.now()) {
				if misses++; misses >= terminalProbeMisses {
					t.say("the probe got no answer %d times running; the lane's own agent finds out instead", misses)
					return ""
				}
			}
		}
	}
}

// WaitOutProof waits out a limit the drive's live proof met, on the schedule any
// agent's limit waits on, proving again each time it lets the run go on. It returns
// why the wait ended the run, "" when prove has run clean or failed for itself, and
// then what prove said. Nothing has started: what it may renew is what an earlier
// run left held.
func WaitOutProof(ctx context.Context, repoRoot string, o TerminalOptions, first *runner.LimitError, prove func(context.Context) error) (string, error) {
	if o.Sleep == nil {
		o.Sleep = func(ctx context.Context, d time.Duration) { sleepOr(ctx, d, o.Stop) }
	}
	t := newTerminal(repoRoot, o)
	t.ctx = ctx
	limit := first
	for {
		entry := evidenceEntry{Event: "limit", Role: "live proof", Seen: limit.Seen}
		if !limit.ResetsAt.IsZero() {
			entry.ResetRead = limit.ResetsAt.UTC().Format(time.RFC3339)
		}
		t.evidence.begin("limit", entry, t.now())
		if ended := t.limited(ctx, RoleImplementer, limit.ResetsAt); ended != "" {
			return ended, nil
		}
		switch {
		case ctx.Err() != nil:
			return TerminalKilled, nil
		case t.stopped():
			return TerminalStopped, nil
		case t.pastBound():
			return TerminalUntil, nil
		}
		err := prove(ctx)
		if !errors.As(err, &limit) {
			return "", err
		}
	}
}

// peerTook reports whether the lane's claim has gone from under it, after waits that
// add up to long enough for that: a peer who took the issue keeps it, and one
// released while the lane waited says what it was held for has changed, so either
// way the lane stops spending and goes back as stuck. What the run renewed through
// the wait it still holds, and a lapse with no peer is revived as it wakes. A check
// that could not be made stays owed.
func (t *terminal) peerTook(runID string) bool {
	if t.waited < terminalRenewEvery {
		return false
	}
	run, err := ReadState(t.repoRoot, runID)
	if err != nil {
		return false
	}
	if run.ReservedAt == nil || letGoByDrive(run) {
		t.waited = 0
		return false
	}
	now := t.now()
	if _, err := claim.Fetch(t.repoRoot, claimOptions(now)); err != nil {
		return false
	}
	snap, err := claim.View(t.repoRoot, now)
	if err != nil {
		return false
	}
	t.waited = 0
	own, _, lost := renewalDue(run, snap, now)
	if lost == "" && own != nil && own.State == claim.StateReleased {
		lost = "the reservation on " + run.Key + " was released while this lane waited"
	}
	if lost == "" {
		return false
	}
	r := refuse(StageReservation, "", "", lost, "see the reason")
	r.final = "the issue is no longer this lane's to work on"
	t.stayAsItIs(runID, run.Key, r)
	return true
}

// renewing runs work that can outlast the lease, an agent's call or a preflight,
// renewing what the run's lanes hold while it does. The renewals have stopped when
// it returns.
func (t *terminal) renewing(work func()) {
	stop, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		tick := time.NewTicker(terminalRenewTick)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				t.renewHeld()
				t.sayCallStill()
			case <-stop:
				return
			}
		}
	}()
	work()
	close(stop)
	<-done
	t.endCall()
}

// startCall says something is starting that may take a long while, which from
// outside looks just like a hang unless it says so.
func (t *terminal) startCall(what string) {
	t.beginCall(what)
	t.say("starting %s", what)
}

// beginCall notes what is in flight, with no line of its own.
func (t *terminal) beginCall(what string) {
	now := t.now()
	t.call.mu.Lock()
	defer t.call.mu.Unlock()
	t.call.what, t.call.since, t.call.said = what, now, now
}

// sayCallStill says what is in flight is still going, every so often.
func (t *terminal) sayCallStill() {
	now := t.now()
	t.call.mu.Lock()
	what, since := t.call.what, t.call.since
	due := what != "" && now.Sub(t.call.said) >= terminalCallSayEvery
	if due {
		t.call.said = now
	}
	t.call.mu.Unlock()
	if due {
		t.say("%s is still running, %s so far", what, now.Sub(since).Round(time.Minute))
	}
}

// endCall notes nothing is in flight.
func (t *terminal) endCall() {
	t.call.mu.Lock()
	defer t.call.mu.Unlock()
	t.call.what = ""
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
	if r.checkWait.IsZero() {
		if t.idle(ctx, TerminalWaitFloor, r.Reason) {
			return false, TerminalWaited, nil
		}
		return false, "", nil
	}
	run, rerr := ReadState(t.repoRoot, runID)
	if rerr != nil {
		return false, "", rerr
	}
	i := run.current()
	if i < 0 {
		return false, "", nil
	}
	lane := run.Lanes[i]
	head := runID + " " + lane.ID + " " + lane.HeadSHA
	if t.ranClean == head {
		// It exited 0 and the receipt is still not there, which running it again
		// would not change: the lane goes to the stuck check.
		return t.laneRefused(ctx, runID, lane.ID, "its preflight exits 0 and leaves no receipt naming the lane's head", "")
	}
	if !t.wallHeld() {
		return false, TerminalWall, nil
	}
	t.say("%s %s: running the preflight", runID, lane.ID)
	switch perr := t.preflight(ctx, lane.Worktree); {
	case perr == nil:
		t.ranClean = head
		t.minted[lane.HeadSHA] = true
		return true, "", nil
	case ctx.Err() != nil || t.stopped():
		return false, "", nil
	default:
		return t.laneRefused(ctx, runID, lane.ID, "its preflight failed: "+perr.Error(), "")
	}
}

// preflight runs a lane's preflight on a context a stop also ends, so one queued
// at the load gate or running does not outlast it.
func (t *terminal) preflight(ctx context.Context, worktree string) error {
	pctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		select {
		case <-t.o.Stop:
			cancel()
		case <-pctx.Done():
		}
	}()
	var err error
	t.beginCall("the preflight")
	t.renewing(func() { err = t.o.Preflight(pctx, worktree) })
	return err
}

// laneRefused logs a refusal on the lane for the next move's stuck check, final
// saying why it cannot change, and waits at the floor.
func (t *terminal) laneRefused(ctx context.Context, runID, laneID, reason, final string) (bool, string, error) {
	why := refuse(string(StageLand), "", laneID, reason, "see the reason")
	why.final = final
	now := t.now()
	if err := mutate(t.repoRoot, runID, func(_ *os.Root, st *State) (bool, error) {
		return noteRefusal(st, why, now), nil
	}); err != nil {
		return false, "", err
	}
	t.sleep(ctx, TerminalWaitFloor, reason)
	return false, "", nil
}

// takeUp reserves again a lane a drive let go as it ended. It reports true when
// the lane cannot go on here yet, with why the run ends, if it does. Someone
// else holding its issue now is recorded as a refusal that cannot change, so the
// next move hands the lane back as stuck and moves on. Anything else may pass, a
// failed push or the person's own cap among them, so the lane waits on the
// outage's deadline with nothing logged against it.
func (t *terminal) takeUp(ctx context.Context, runID string) (bool, string) {
	run, err := ReadState(t.repoRoot, runID)
	if err != nil || !letGoByDrive(run) {
		return false, ""
	}
	err = reserveResumed(t.repoRoot, &StartResult{RunID: runID}, t.o.Options)
	if err == nil {
		t.say("took up %s, run %s: its reservation is held again", run.Key, runID)
		return false, ""
	}
	r, ok := AsRefusal(err)
	if !ok {
		r = refuse(StageReservation, "", "", err.Error(), "see the reason")
	}
	if r.final == "" && r.Contention {
		r.final = "another drive took the issue up after this lane stopped"
	}
	if r.final == "" {
		if t.failed(ctx, false, fmt.Sprintf("%s, run %s, is not taken up yet: %s", run.Key, runID, r.Reason)) {
			return true, TerminalOutage
		}
		return true, ""
	}
	r.Contention = false
	t.stayAsItIs(runID, run.Key, r)
	return true, ""
}

// stayAsItIs records a refusal that cannot change on the run's lane, so the next
// move hands it back as stuck.
func (t *terminal) stayAsItIs(runID, key string, r *Refusal) {
	t.say("%s, run %s, stays as it is: %s", key, runID, r.Reason)
	now := t.now()
	_ = mutate(t.repoRoot, runID, func(_ *os.Root, st *State) (bool, error) {
		i := st.current()
		if i < 0 {
			return false, nil
		}
		r.Lane = st.Lanes[i].ID
		return noteRefusal(st, r, now), nil
	})
}

// letGoByDrive reports whether the run's latest reservation entry is a drive
// letting it go.
func letGoByDrive(run State) bool {
	for i := len(run.Record) - 1; i >= 0; i-- {
		switch run.Record[i].Stage {
		case StageDriveReleased:
			return true
		case StageReservation, StageReservationRenewed:
			return false
		}
	}
	return false
}

// letGo releases what the run holds and no lane is working on: each lane in
// progress, left on its branch, and a judgement's hold no lane opened on. A
// lane parked at its pull request keeps its own.
func (t *terminal) letGo() {
	now := t.now()
	err := fsutil.WithFileLock(filepath.Join(t.repoRoot, filepath.FromSlash(drainLockRel)), lockTimeout, func() error {
		// A drain that ended on an outage can have a lane still in progress, so
		// its lanes are read whether it is live or not.
		st, live, err := readDrain(t.repoRoot)
		if err != nil {
			return err
		}
		for _, l := range st.Lanes {
			if l.Outcome != DrainLaneInProgress {
				continue
			}
			run, err := ReadState(t.repoRoot, l.RunID)
			if err != nil || run.ReservedAt == nil || letGoByDrive(run) {
				continue
			}
			clause, _ := releaseReservation(t.repoRoot, run.Key, "", now)
			t.say("%s: %s", run.Key, clause)
			_ = mutate(t.repoRoot, l.RunID, func(_ *os.Root, st *State) (bool, error) {
				st.Record = append(st.Record, Entry{At: now, Stage: StageDriveReleased, Note: clause})
				st.UpdatedAt = now
				return true, nil
			})
		}
		if j := st.Judging; j != nil && live {
			if j.Hold != nil {
				clause, _ := releaseReservation(t.repoRoot, j.Issue, "", now)
				t.say("%s: %s", j.Issue, clause)
			}
			st.Judging = nil
			st.UpdatedAt = now
			return writeDrain(t.repoRoot, st)
		}
		return nil
	})
	if err != nil {
		t.say("what this run holds was not released, and lapses with its lease: %s", err.Error())
	}
}

// ended is the run's result as it ends: unless a kill or a stop from elsewhere
// ended it, what no lane is working on is released, so a peer may take it up; the
// lane left in progress is named, with the command that resumes it, or, after a
// kill, when its reservation lapses. A stop from elsewhere names its own remedy.
func (t *terminal) ended(why string) TerminalResult {
	if why != TerminalKilled && why != TerminalHalted {
		t.letGo()
	}
	res := TerminalResult{Ended: why, Last: t.last, HeldBack: t.heldBack}
	st, _, err := readDrain(t.repoRoot)
	if err != nil {
		return res
	}
	for _, l := range st.Lanes {
		if l.Outcome != DrainLaneInProgress {
			continue
		}
		res.Run, res.Issue = l.RunID, l.Issue
		if why != TerminalHalted {
			res.Resume = TerminalResumeCommand
		}
		if run, err := ReadState(t.repoRoot, l.RunID); err == nil {
			if i := run.current(); i >= 0 {
				res.Lane = run.Lanes[i].ID
			}
		}
		if why == TerminalKilled {
			if snap, err := claim.View(t.repoRoot, t.now()); err == nil {
				if r, ok := snap.Lookup(l.Issue); ok && r.State == claim.StateLive {
					at := r.EffectiveExpiry
					res.Lapses = &at
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
