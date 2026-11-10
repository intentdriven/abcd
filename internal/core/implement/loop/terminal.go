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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sync"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/core/runstop"
	"github.com/intentdriven/abcd/internal/core/sessionname"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
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
	// TerminalHalted: a stop set from outside this terminal holds, or one a run
	// keeps stands against it (itd-2611041549471712). What the run holds is that
	// stop's to let go.
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

// TerminalResumeCommand resumes a run that left a lane.
const TerminalResumeCommand = "abcd drain --drive"

// StageDriveReleased is the run record's stage for a reservation a drive let
// go as it ended, which the drive that takes the lane up reserves again.
const StageDriveReleased = "drive_released"

// StageReleasedOver is the run record's stage for the record a drive's release let go
// of: its note is that record's blob on origin's default branch and nothing else, so
// the take-up can tell what the issue asks now from what the lane was working to.
const StageReleasedOver = "drive_released_over"

// StageDriveReleaseFailed is the run record's stage for a release that did not reach
// origin: its note is the tip it left and nothing else.
const StageDriveReleaseFailed = "drive_release_failed"

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
	// Resumed says this process goes on with a drive another began, so what that one
	// held back is this one's to report at its end.
	Resumed bool
	// BoundaryCeiling is the longest AtBoundary waits for what the run has in hand; past it
	// the call runs regardless. Zero sets none.
	BoundaryCeiling time.Duration
	// Moved is called after each move the drain makes, before anything else the run
	// does with it. Nil does nothing.
	Moved func()
	// AtBoundary is called once a move has been made and before the next, between stages
	// and with the checks above just passed: a lane has ended, or a wait has ended before
	// the next opens. A refused step comes back to the run with its lane still in progress
	// and reserved, and a judgement holds its issue, so the call stays due until the next
	// move has settled each, for BoundaryCeiling at most; past that it runs beside them,
	// and what the run holds is renewed while it does. It always waits for the next move to
	// take a skip the run gave. It is never called during a stage. It reports whether it
	// took time, so that the run goes round the stop, the bound and the checks again before
	// anything starts. Nil does nothing.
	AtBoundary func(ctx context.Context) bool
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
var errRunStateChanged = errors.New("the run's state changed while an agent was in its call")

// errRunStateUnread is a run whose state still could not be read after the retries that
// absorb a passing failure. A changed state can hide behind it, so it stops the drive as
// a change does.
var errRunStateUnread = errors.New("the run's state could not be read after an agent's call")

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
	// PassedOver names each issue the run did not open, with why.
	PassedOver []Excluded `json:"passed_over,omitempty"`
	// Why is what stopped the run when a check did.
	Why string `json:"why,omitempty"`
}

// terminal is one run in progress.
type terminal struct {
	repoRoot string
	o        TerminalOptions
	// boundaryDue is a move made since the last boundary: the first move of a run
	// has none before it, since its start has just proved what it runs.
	boundaryDue bool
	outage      terminalOutageClock
	// limit is the outage's schedule run for a rate limit.
	limit terminalOutageClock
	last  DrainResult
	// heldToProfile is set when the drive ended on a run's profile (holdToProfile).
	heldToProfile bool
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
	// reached counts, by role, launches of the lane in hand in a row whose failure
	// was the work's: an answer refused, or a probe showing the account alive.
	reached map[string]int
	// blames are what the launches of this step or move said, not yet settled.
	blames []launchBlame
	// notedFinish is what each run left unfinished last said, so it is said once.
	notedFinish map[string]string
	// laneInHand is the run the lane counts below are of.
	laneInHand string
	// startedAgent says this step started an agent, which is movement of a kind.
	startedAgent bool
	// spentUnanswered counts, by role, launches in a row that spent with no probe
	// answering, a ceiling on what a probe that never answers would let run.
	spentUnanswered map[string]int
	// netKey and netSince are the act last read as a lost network with origin
	// answering, and when that was first read.
	netKey   string
	netSince time.Time
	// gaveUpLook is when a record another session gave up on is next looked at.
	gaveUpLook time.Time
	// session is the drive's own in the shared run, under a bound, and shared that
	// run: an open outage record needs a session to be probed for.
	session string
	shared  *implement.Run
	// stalled counts the lane in hand's steps in a row that started no agent and
	// moved no stage.
	stalled int
	// call is the agent's call or preflight in flight, what its lines say.
	call terminalCall
	// unanswered counts, by issue, the judgements a runner ran and that gave no
	// answer; heldBack names each issue the run held back for that.
	unanswered map[string]int
	heldBack   []string
	// inHand says the last move left the run holding something the boundary must not
	// outwait: a lane still in progress, or a judgement.
	inHand bool
	// heldSince is when a due boundary began to wait for that, and holdToSay that the next
	// wait's line has yet to say so.
	heldSince time.Time
	holdToSay bool
	// passedOver are the issues said to need a person, each once.
	passedOver []Excluded
	// why is what stopped the run, when a check did, and wallEnded the reason a wait
	// inside that check ended it for, when one did.
	why       string
	wallEnded string
	// judge is what the last move's routed judgement came to.
	judge JudgeOutcome
	// minted names each head a preflight this run ran exited 0 on, and bases each
	// lane's base as the run first read it.
	minted map[string]bool
	bases  map[string]string
}

// launchBlame is what one failed launch said of its work.
type launchBlame struct {
	role, model string
	blame       runner.Blame
	spent       bool
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
	if t.o.Sleep == nil {
		t.o.Sleep = func(ctx context.Context, d time.Duration) { sleepOr(ctx, d, o.Stop, t.stopHolds) }
	}
	t.o.Options.driverIsSelf = true
	t.o.Drain.JudgeOutcome = &t.judge
	t.o.Options.RunStateWatch = true
	t.o.Options.ReceiptOwn = func(sha string) bool { return t.minted[sha] }
	t.evidence = newDriveEvidence(repoRoot, func(line string) { t.say("%s", line) })
	t.o.Options.RunnerFailed = t.noteFailure
	t.o.Options.RunnerStarting = func(role string) {
		t.startedAgent = true
		t.startCall("the " + role)
	}
	t.waited = terminalRenewEvery
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
		if fl.Outage() {
			t.say("the %s's launch was retrying the model service when it ended, which reads as an outage", role)
		}
		if t.bounded() {
			t.blames = append(t.blames, launchBlame{role: role, model: fl.Model, blame: fl.Blame(), spent: fl.Seen != nil && fl.Seen.Spent})
		}
		if fl.Seen != nil {
			t.evidence.begin("suspect-"+failureKind(fl), entry, t.now())
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

// failureKind names what sort of failure a launch's was, for its evidence file: the
// first of each is kept, so an outage early on hides no later expired login.
func failureKind(fl *runner.Failure) string {
	switch {
	case fl.Seen != nil && fl.Seen.APIStatus != "" && statusWord.MatchString(fl.Seen.APIStatus):
		return "api-" + fl.Seen.APIStatus
	case fl.Outage():
		return "outage"
	case errors.Is(fl, runner.ErrKilled):
		return "killed"
	}
	return string(fl.Reason)
}

// statusWord is an API status plain enough to name a file by.
var statusWord = regexp.MustCompile(`^[0-9]{3}$`)

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

// WallEnded is a BeforeStage's error when a wait inside the check ended the run for a
// reason of its own, which the run ends on: a stop, a kill, or a limit it could not outwait.
type WallEnded struct{ Ended string }

func (e *WallEnded) Error() string { return "the check before a stage ended the run: " + e.Ended }

// wallHeld runs the check before a stage, saying what stopped the run.
func (t *terminal) wallHeld() bool {
	if t.o.BeforeStage == nil {
		return true
	}
	err := t.o.BeforeStage()
	var ended *WallEnded
	if errors.As(err, &ended) {
		t.wallEnded = ended.Ended
		return false
	}
	if err != nil {
		t.stoppedBy(err)
	}
	return err == nil
}

// stoppedBy says what stopped the run and keeps it for the end.
func (t *terminal) stoppedBy(err error) {
	t.why = err.Error()
	t.say("stopping: %s", t.why)
}

// stoppedOnChange stops the run over a state changed or unreadable, keeping that where a
// later drive and a later step read it, so neither goes on from what it now says.
func (t *terminal) stoppedOnChange(runID string, cause error) {
	if err := recordDriveStop(runID); err != nil {
		t.say("%s: the stop could not be kept, so do not start another drive on it: %s", runID, err.Error())
	}
	t.stoppedBy(fmt.Errorf("%w; nothing goes on from %s. %s", cause, runID, driveStopRemedy(runID)))
}

// haltedBy reports whether err is a stop set from outside this terminal, saying
// what holds and what lifts it, and keeping that for the end. Sleeping through it, as
// through another contention, would keep the drive alive for ever under a stop.
func (t *terminal) haltedBy(err error) bool {
	r, ok := AsRefusal(err)
	if !ok || r.Stage != StageStop {
		return false
	}
	t.why = termsafe.Sanitize(fsutil.RedactHome(r.Reason + "; " + r.Remedy))
	t.say("stopping: %s", t.why)
	return true
}

// stopHolds reports whether a stop set from outside this terminal holds, or a read that
// cannot be trusted holds the run, for the lane in hand. Every wait asks, so a stop is not
// waited through; a read refused is no stop.
func (t *terminal) stopHolds() bool {
	var pin *runstop.Pin
	if t.laneInHand != "" {
		if c, err := runControl(t.repoRoot, t.laneInHand); err == nil {
			pin = c.Shared
		}
	}
	r := t.o.Options.stopRead(pin)
	return r.Stop != nil || r.Held != ""
}

// settleHalted has each lane in progress take the stop that ended the run, as a step
// would have, and keeps what holds for the end when no step has said.
func (t *terminal) settleHalted() {
	st, _, err := readDrain(t.repoRoot)
	if err == nil {
		for _, l := range st.Lanes {
			if l.Outcome != DrainLaneInProgress {
				continue
			}
			if _, err := stopCheck(t.repoRoot, l.RunID, t.o.Stages, t.o.Options); t.why == "" {
				t.haltedBy(err)
			}
		}
	}
	if t.why != "" {
		return
	}
	r := t.o.Options.stopRead(nil)
	switch {
	case r.Stop != nil:
		t.haltedBy(contend(StageStop, "", "", fmt.Sprintf("a stop set by %s at %s holds", termsafe.Sanitize(r.Stop.SetBy), r.Stop.SetAt.UTC().Format(time.RFC3339)),
			"the person's own `abcd implement stop --lift` lifts it"))
	case r.Held != "":
		t.haltedBy(heldRefusal(r.Held))
	}
}

// haltOnSignal has each lane in progress take the stop this terminal was asked for, in a
// stop's order: the call in hand has finished by now, so what the lane left is saved aside
// and its reservation released. A run a drive stopped on is left as it stands (stop.go).
func (t *terminal) haltOnSignal() {
	st, _, err := readDrain(t.repoRoot)
	if err != nil {
		t.say("the lanes this run holds could not be read to halt them: %s", err.Error())
		return
	}
	now := t.now()
	for _, l := range st.Lanes {
		if l.Outcome != DrainLaneInProgress || driveStopped(l.RunID) {
			continue
		}
		s := runstop.Stop{SetBy: "a signal to the drive", SetAt: now}
		rec, err := haltRun(t.repoRoot, l.RunID, s, t.o.Stages, t.o.Options)
		switch {
		case err != nil:
			t.say("%s: the stop could not be applied: %s", l.RunID, err.Error())
		case rec != nil && rec.HaltedAt == nil:
			t.say("%s: an agent is still in its call, so the next drive settles what it left", l.RunID)
		case rec != nil:
			t.say("%s: halted, and what its lanes left is saved", l.RunID)
		}
	}
}

// sleep waits d, saying why, in stretches no longer than half a lease, renewing
// what the run's lanes still hold between them, so a long wait keeps those claims. A
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
		if t.stopHolds() {
			return
		}
		step := min(d, terminalRenewEvery, max(end.Sub(t.now()), time.Second))
		t.o.Sleep(ctx, step)
		d -= step
		if ctx.Err() != nil || t.stopped() || t.stopHolds() {
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
	if !t.holdToSay && why == t.lastWhy && now.Sub(t.lastSaid) < terminalSayEvery {
		return
	}
	t.lastWhy, t.lastSaid = why, now
	held := ""
	if t.holdToSay {
		t.holdToSay = false
		held = fmt.Sprintf("; a new release, if there is one, waits for this, %s at most", t.o.BoundaryCeiling.Round(time.Second))
	}
	t.say("waiting %s: %s%s", d.Round(time.Second), why, held)
}

// renewHeld renews each due reservation of a run in progress, but one a drive
// let go, as it ended or for a limit, and of a lane parked at its pull request, and says
// what failed.
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
	t.joinSharedRun()
	if t.o.Resumed {
		t.reportHeldBack()
	}
	for {
		if ctx.Err() != nil {
			return t.ended(TerminalKilled), nil
		}
		if t.stopped() {
			return t.ended(TerminalStopped), nil
		}
		if t.stopHolds() {
			return t.ended(TerminalHalted), nil
		}
		if t.pastBound() {
			return t.ended(TerminalUntil), nil
		}
		if !t.wallHeld() {
			return t.ended(TerminalWall), nil
		}
		if t.boundaryDue && t.o.AtBoundary != nil && !t.boundaryHeld() {
			t.boundaryDue = false
			t.heldSince = time.Time{}
			took := false
			if t.inHand {
				t.renewing(func() { took = t.o.AtBoundary(ctx) })
			} else {
				took = t.o.AtBoundary(ctx)
			}
			if took {
				continue
			}
		}
		t.settleOutage(ctx)
		t.judge = JudgeOutcome{}
		t.blames = nil
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
		if t.o.Moved != nil {
			t.o.Moved()
		}
		t.boundaryDue = true
		t.inHand = res.Judging != nil
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
		if drainMoved(res) {
			t.idleSince = time.Time{}
		}
		// A judge that met a limit is an agent like any other: none starts,
		// this one included, before the limit lets it.
		if t.judge.Kind == JudgeLimited && res.Judging != nil {
			if ended := t.limited(ctx, "", RoleDrainJudge, t.judge.ResetsAt); ended != "" {
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
			// The count is the engine's, of judgements a runner ran that gave no
			// answer. A bound adds this: a launch that failed counts too if a
			// probe shows the account alive, so a judge at its deadline having
			// spent is the issue's fault and an account-wide failure is not.
			counts := t.judge.Kind == JudgeNoAnswer
			if t.judge.Kind == JudgeRunnerFailed && t.settleBlames(ctx) > 0 {
				counts = true
			}
			if t.spentUnanswered[RoleDrainJudge] >= terminalSpentUnanswered {
				delete(t.spentUnanswered, RoleDrainJudge)
				if t.unanswered == nil {
					t.unanswered = map[string]int{}
				}
				t.unanswered[res.Judging.Issue] = terminalJudgeAsks - 1
				counts = true
			}
			if counts && t.holdBack(res.Judging.Issue) {
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
		t.inHand = t.laneInProgress(res.Lane.RunID)
	}
}

// boundaryHeld reports whether a due boundary waits: for a skip the run gave and the next
// move has yet to take, which a process that moved first would drop; and for what the run
// has in hand, up to the ceiling.
func (t *terminal) boundaryHeld() bool {
	if len(t.o.Drain.Skip) > 0 {
		return true
	}
	if !t.inHand {
		t.heldSince = time.Time{}
		return false
	}
	if t.o.BoundaryCeiling <= 0 {
		return true
	}
	now := t.now()
	if t.heldSince.IsZero() {
		t.heldSince, t.holdToSay = now, true
	}
	return now.Sub(t.heldSince) < t.o.BoundaryCeiling
}

// laneInProgress reports whether the lane a step came back from is still going. A state
// that cannot be read counts as going: the next move says what is wrong with it.
func (t *terminal) laneInProgress(runID string) bool {
	run, err := ReadState(t.repoRoot, runID)
	if err != nil {
		return true
	}
	outcome, _, _ := laneOutcome(run)
	return outcome == DrainLaneInProgress
}

// reportHeldBack takes on what the drive this process goes on from held back, which the
// drain's skip list still holds.
func (t *terminal) reportHeldBack() {
	st, _, err := readDrain(t.repoRoot)
	if err != nil {
		return
	}
	for _, s := range st.Skips {
		if s.Reason == heldBackReason {
			t.heldBack = append(t.heldBack, s.Target)
		}
	}
}

// heldBackReason is the reason a run holds an issue back under.
const heldBackReason = "its judge did not answer"

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
	t.o.Drain.Skip, t.o.Drain.SkipReason = []string{issue}, heldBackReason
	t.heldBack = append(t.heldBack, issue)
	t.keepHeldBack(issue)
	return true
}

// heldBackRoot is the main checkout of repoRoot, so what a drive held back
// outlives a linked worktree's removal. A repository whose git directory lies
// elsewhere keeps it in repoRoot.
func heldBackRoot(repoRoot string) string {
	common, err := gitutil.RevParseAbsPath(repoRoot, "--git-common-dir")
	if err != nil || filepath.Base(common) != ".git" {
		return repoRoot
	}
	return filepath.Dir(common)
}

// heldBackFile is where a checkout's held-back issues are kept.
func heldBackFile(repoRoot string) string {
	return filepath.Join(heldBackRoot(repoRoot), filepath.FromSlash(HeldBackRel))
}

// keepHeldBack appends the fact to a file of its own.
func (t *terminal) keepHeldBack(issue string) {
	line := t.now().UTC().Format(time.RFC3339) + " " + issue + " " + heldBackReason + "\n"
	if err := fsutil.EnsureRealDirAll(heldBackRoot(t.repoRoot), filepath.ToSlash(filepath.Dir(HeldBackRel)), 0o700); err == nil {
		f, err := os.OpenFile(heldBackFile(t.repoRoot), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
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
	for _, p := range res.Passed {
		if p.Check == CheckNeedsPerson {
			t.notePassedOver(p)
		}
	}
	for _, f := range res.Finished {
		switch {
		case f.Done:
			t.say("finished %s", f.Issue)
		case f.Note != "" && t.notedFinish[f.RunID] != f.Note:
			if t.notedFinish == nil {
				t.notedFinish = map[string]string{}
			}
			t.notedFinish[f.RunID] = f.Note
			t.say("%s: %s", f.Issue, f.Note)
		}
	}
	if res.Start != nil && res.Lane != nil {
		t.say("opened the lane of %s: run %s", res.Lane.Issue, res.Lane.RunID)
	}
}

// drainMoved reports whether a drain's move moved anything: a lane it read and did not finish
// did not.
func drainMoved(res DrainResult) bool {
	return res.Judged != nil || len(res.Routed) > 0 || res.Start != nil ||
		slices.ContainsFunc(res.Finished, func(f DrainFinished) bool { return f.Done })
}

// driveLane steps one lane until it has landed or been handed back, or the
// next move has something to say about it. It returns why the run ends, "" to
// go on to the next move.
func (t *terminal) driveLane(ctx context.Context, runID string) (string, error) {
	// A marked run stops the drive under any bound, before any count or wait.
	if err := driveStopRefusal(runID); err != nil {
		t.stoppedBy(err)
		return TerminalWall, nil
	}
	if ended, err := t.holdToProfile(runID); ended != "" || err != nil {
		return ended, err
	}
	// A refused step goes back through the next move to this same lane, so its
	// counts last until another lane is in hand.
	if runID != t.laneInHand {
		t.laneInHand = runID
		t.reached, t.stalled, t.spentUnanswered = nil, 0, nil
		t.netKey, t.netSince = "", time.Time{}
	}
	if err := ClaimDriver(t.repoRoot, runID, t.o.Options); err != nil {
		r, ok := AsRefusal(err)
		if !ok || !r.Contention {
			return "", err
		}
		if t.idle(ctx, TerminalWaitFloor, r.Reason) {
			return TerminalWaited, nil
		}
		return "", nil
	}
	if done, ended := t.takeUp(ctx, runID); done {
		return ended, nil
	}
	if err := t.markProfile(runID); err != nil {
		return "", err
	}
	for {
		if ctx.Err() != nil {
			return TerminalKilled, nil
		}
		if t.stopped() {
			return TerminalStopped, nil
		}
		if t.stopHolds() {
			return TerminalHalted, nil
		}
		if t.pastBound() {
			return TerminalUntil, nil
		}
		if !t.wallHeld() {
			return TerminalWall, nil
		}
		// Every wait comes back through here, so a bound's waits do not outlast a stop
		// another drive keeps against the run.
		if err := driveStopRefusal(runID); err != nil {
			t.stoppedBy(err)
			return TerminalWall, nil
		}
		t.settleOutage(ctx)
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
					t.stoppedOnChange(runID, errRunStateChanged)
					return TerminalWall, nil
				}
			}
			if lane.Stage == StageLand && t.o.PreflightAllowed != nil {
				var refused *PreflightRefused
				if perr := t.o.PreflightAllowed(lane.Worktree, diffBase(lane)); errors.As(perr, &refused) {
					_, ended, err := t.laneRefused(ctx, runID, lane.ID, refused.Reason, refused.Reason)
					return ended, err
				} else if perr != nil {
					return "", perr
				}
			}
		}
		var res StepResult
		t.startedAgent = false
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
			t.stoppedOnChange(runID, errRunStateChanged)
			return TerminalWall, nil
		}
		if errors.Is(err, errRunStateUnread) {
			t.stoppedOnChange(runID, err)
			return TerminalWall, nil
		}
		if t.poisoned(ctx, runID) {
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
			if ended := t.limited(ctx, runID, res.RateLimit.Role, res.RateLimit.resetsAt); ended != "" {
				return ended, nil
			}
			// A lane let go for the limit goes back through the next move, which takes
			// it up or drops it as any lane a drive let go.
			if after, err := ReadState(t.repoRoot, runID); err == nil && letGoByDrive(after) {
				return "", nil
			}
		case res.NextEligibleAt != nil:
			t.sleep(ctx, max(res.NextEligibleAt.Sub(t.now()), TerminalWaitFloor), res.Next)
		case res.Route != nil || res.PerformedStage != "" || res.stepNote != "":
			t.idleSince = time.Time{}
			t.moved()
			t.sayStep(runID, res)
		default:
			// No agent ran and no stage moved. Every wait has a deadline
			// (adr-2610222222326104), so this one runs on the outage's. Under a
			// bound that schedule never ends, so a lane that keeps doing this goes
			// back, rather than hold every other issue until the bound.
			if t.cannotMove(runID, res) {
				return "", nil
			}
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
	t.reached, t.stalled, t.spentUnanswered = nil, 0, nil
	t.netKey, t.netSince = "", time.Time{}
	t.evidence.resumed(t.now())
}

// cannotMove counts one step that started no agent and moved no stage, under a bound,
// and hands the lane back at the limit, its worktree and branch left in place. A step
// that started an agent is no stall, and ends the count: a silent launch killed at its
// deadline moves no stage, and finished work is not sent back for that. A wait the
// shared outage holds an agent of this lane's to, or a claim on one of its agents that
// has yet to lapse, a killed run's say, is no fault of the lane's and is not counted.
func (t *terminal) cannotMove(runID string, res StepResult) bool {
	if !t.bounded() {
		return false
	}
	if t.startedAgent {
		t.stalled = 0
		return false
	}
	run, err := ReadState(t.repoRoot, runID)
	if err != nil || outageHoldsLane(res, run) {
		return false
	}
	now := t.now()
	for _, l := range run.Lanes {
		for _, a := range l.Awaits {
			if a.Running != nil && now.Before(a.Running.Until) {
				return false
			}
		}
	}
	if t.stalled++; t.stalled < terminalNoMoveLimit {
		return false
	}
	t.stalled = 0
	r := refuse(StageRunner, "", "", fmt.Sprintf("%d steps in a row started no agent and moved no stage: %s", terminalNoMoveLimit, res.Next), "see the reason")
	r.final = "the lane cannot move, so it goes back and the run goes on"
	t.stayAsItIs(runID, run.Key, r)
	return true
}

// outageHoldsLane reports whether the outage a step reports holds an agent the lane
// wants: one kept for a person, or already answered, wants none.
func outageHoldsLane(res StepResult, run State) bool {
	if res.Outage == nil || len(res.Outage.Down) == 0 {
		return false
	}
	for _, l := range run.Lanes {
		for _, a := range l.Awaits {
			if a.Kept == "" && a.Answered == nil {
				return true
			}
		}
	}
	return false
}

// joinSharedRun joins the shared run under a session of the drive's own, under a
// bound: an open outage record is probed only for a session, so without one it would
// hold every agent until the bound. The session is used for the outage alone, so the
// drive's runs claim nothing there. A drive that cannot join says so and goes on.
func (t *terminal) joinSharedRun() {
	if !t.bounded() {
		return
	}
	sha, err := gitutil.RootCommit(t.repoRoot)
	if err != nil || !gitutil.IsFullSHA(sha) {
		return
	}
	run, err := implement.Open(sha)
	if err != nil {
		t.say("the shared run could not be opened, so no open outage will be probed: %s", err.Error())
		return
	}
	run.Now = t.o.Options.Now
	sum := sha256.Sum256([]byte(realPath(t.repoRoot)))
	id := "drive-" + hex.EncodeToString(sum[:6])
	if _, err := run.Join(id, implement.RoleSecond, "", "a drive's outage probes", 0); err != nil {
		t.say("the shared run could not be joined, so no open outage will be probed: %s", err.Error())
		return
	}
	t.shared, t.session = run, id
	t.o.Options.OutageSession = id
}

// leaveSharedRun leaves what joinSharedRun joined.
func (t *terminal) leaveSharedRun() {
	if t.shared == nil {
		return
	}
	_, _ = t.shared.Leave(t.session, "the drive ended")
	t.shared, t.session = nil, ""
	t.o.Options.OutageSession = ""
}

// settleOutage acts for the person a shared outage would otherwise wait on, under a
// bound. With the model down in an open record, a probe answering is the service's
// canary: that outage holds every agent, so no launch would ever find out it is back.
// The probe runs inside the outage's own probe, so only once its lease is taken and it
// is due, and a result that cannot be written costs nothing. The network's own probe
// runs from the step that finds it due (outageFor), and here too when both are down. A
// record another session gave up on can no longer be probed, so it is looked at on a
// schedule of the run's own and cleared when every service down answers.
func (t *terminal) settleOutage(ctx context.Context) {
	if t.shared == nil {
		return
	}
	cur, err := t.shared.Outage().Current()
	if err != nil || cur == nil {
		return
	}
	if cur.Status == implement.OutageGaveUp {
		t.settleGaveUp(ctx, cur)
		return
	}
	if t.o.Probe == nil || cur.Status != implement.OutageOpen ||
		!slices.Contains(cur.Down, implement.ServiceModel) || t.now().Before(cur.NextProbeAt) || !t.wallHeld() {
		return
	}
	_, _ = keepProbing(t.shared.Outage()).ProbeIfDue(t.session, implement.Prober{
		Network: t.networkProbe(),
		Model: func() (bool, string) {
			t.say("the shared outage has the model service down, so a probe asks whether it is back")
			answered, _ := t.o.Probe(ctx, RoleImplementer, "")
			return answered, "a short launch asked it"
		},
	})
}

// settleGaveUp clears a record another session gave up on, when the probes of every
// service it has down answer: the network's first, since it costs nothing, then a short
// launch's for the model's.
func (t *terminal) settleGaveUp(ctx context.Context, cur *implement.Outage) {
	now := t.now()
	if now.Before(t.gaveUpLook) || !t.wallHeld() {
		return
	}
	t.gaveUpLook = now.Add(terminalGaveUpEvery)
	if slices.Contains(cur.Down, implement.ServiceNetwork) {
		if ok, _ := t.networkProbe()(); !ok {
			return
		}
	}
	if slices.Contains(cur.Down, implement.ServiceModel) {
		if t.o.Probe == nil {
			return
		}
		if answered, _ := t.o.Probe(ctx, RoleImplementer, ""); !answered || ctx.Err() != nil {
			return
		}
	}
	if _, err := t.shared.Outage().Clear(t.session, "a drive under a bound cleared it: the probes of every service down answered"); err != nil {
		t.say("the outage another session gave up on could not be cleared: %s", err.Error())
		return
	}
	t.say("the outage another session gave up on is cleared: the probes of every service down answered")
}

// networkProbe is the probe that proves the network back.
func (t *terminal) networkProbe() func() (bool, string) {
	if t.o.Options.NetworkProbe != nil {
		return t.o.Options.NetworkProbe
	}
	return implement.RemoteProbe(t.repoRoot)
}

// netRepeats answers a failure read as a lost network, under a bound. One act that
// keeps failing that way for terminalNetGoesBack while origin answers is no lost
// network, whatever its words say, and waiting would hold the lane until the bound, so
// it goes back as stuck. The act is abcd's own wording, git push or gh pr create, since
// what a command printed can change each time. Origin not answering, another act or a
// step that worked restarts the clock. It reports whether the lane went back.
func (t *terminal) netRepeats(runID string, r *Refusal) bool {
	if !t.bounded() || r.outage == nil || r.outage.service != implement.ServiceNetwork {
		return false
	}
	if ok, _ := t.networkProbe()(); !ok {
		t.netKey, t.netSince = "", time.Time{}
		return false
	}
	now := t.now()
	if r.outage.what != t.netKey {
		t.netKey, t.netSince = r.outage.what, now
	}
	if now.Sub(t.netSince) < terminalNetGoesBack {
		return false
	}
	t.netKey, t.netSince = "", time.Time{}
	run, err := ReadState(t.repoRoot, runID)
	if err != nil {
		return false
	}
	why := refuse(string(StageLand), "", "", r.outage.what+" kept failing for "+terminalNetGoesBack.String()+" with origin answering", "see the reason")
	why.final = "it kept failing while origin answered, so it reads as no lost network"
	t.stayAsItIs(runID, run.Key, why)
	return true
}

// settleBlames counts, by role, what the launches since the last settling said of
// their work, asking a probe of those that do not say: one short launch per such
// failure, after the step and with no lock held. A probe answered shows the account
// is alive, which leaves the work; rejected or unanswered, it counts nothing. It
// returns how many counted.
func (t *terminal) settleBlames(ctx context.Context) int {
	counted := 0
	for _, b := range t.blames {
		count := b.blame == runner.BlameWork
		if b.blame == runner.BlameAsk && t.o.Probe != nil && t.bounded() && ctx.Err() == nil && !t.stopped() && !t.stopHolds() && t.wallHeld() {
			t.say("nothing shows what failed the %s, so a probe asks whether the account is alive", b.role)
			renewed, fl := t.o.Probe(ctx, b.role, b.model)
			entry := evidenceEntry{Event: "probe", Role: b.role, Answer: "no answer"}
			if fl != nil {
				entry.Seen = fl.Seen
			}
			if renewed {
				entry.Answer = "answered"
			}
			t.evidence.add(entry, t.now())
			count = renewed
		}
		if b.blame == runner.BlameAsk && b.spent && !count {
			if t.spentUnanswered == nil {
				t.spentUnanswered = map[string]int{}
			}
			t.spentUnanswered[b.role]++
		} else if count {
			delete(t.spentUnanswered, b.role)
		}
		if !count {
			continue
		}
		counted++
		if t.reached == nil {
			t.reached = map[string]int{}
		}
		t.reached[b.role]++
	}
	t.blames = nil
	return counted
}

// poisoned hands a lane back when one role's launches have failed in a row in a way
// that is the work's, under a bound, where nothing else would end it: the run then
// goes on to other issues. So does a run of launches that spent with no probe
// answering; a launch started with the service unreachable was seen to spend nothing on
// 2.1.318, so an outage of that kind does not raise it. A launch that says nothing of
// the work stays on the outage's wait.
func (t *terminal) poisoned(ctx context.Context, runID string) bool {
	t.settleBlames(ctx)
	for role, n := range t.spentUnanswered {
		if n < terminalSpentUnanswered {
			continue
		}
		run, err := ReadState(t.repoRoot, runID)
		if err != nil {
			return false
		}
		r := refuse(StageRunner, "", "", fmt.Sprintf("the %s spent and failed %d times in a row, with no probe answered", role, n), "see the reason")
		r.final = "launches of the same work kept spending and failing, so the lane goes back and the run goes on"
		t.stayAsItIs(runID, run.Key, r)
		t.spentUnanswered, t.reached = nil, nil
		return true
	}
	for role, n := range t.reached {
		if n < terminalPoisonLaunches {
			continue
		}
		run, err := ReadState(t.repoRoot, runID)
		if err != nil {
			return false
		}
		r := refuse(StageRunner, "", "", fmt.Sprintf("the %s failed %d times in a row with the account alive", role, n), "see the reason")
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
// runID is the lane the limit interrupted, "" for a judge's or a live check's. A wait
// that lasts lets its issue go (letGoForLimit): a reset named terminalReleaseAfter or
// more away at once, and any limit once it has lasted that long. With none named that
// takes a probe to end it: the lane's own launch, which is all that asks with no probe,
// needs the reservation for each asking, and with no bound that wait has its limit.
//
// Under a bound a probe asks too, so the lane is not disturbed by asking: at each
// turn of that schedule, and at a low rate during a named wait, since a reset read
// from a stream can be wrong and a limit that lets a probe through is read as over. A
// probe that names a reset turns the wait into that one, and one rejected naming
// none leaves the wait as it was. Probes with no answer say nothing of the limit: a
// named wait goes on, and with none named a few in a row leave it to the lane's own
// launch, which also serves a harness whose probe can never answer.
func (t *terminal) limited(ctx context.Context, runID, role string, at time.Time) string {
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
				t.say("stopping: the subscription's rate limit resets at %s, later than a drive waits; run %s then", at.UTC().Format(time.RFC3339), t.resumeCommand())
				return TerminalLimited
			}
			left, look := at.Sub(now), terminalNamedProbeEvery
			if left >= terminalReleaseAfter || now.Sub(t.limit.since) >= terminalReleaseAfter {
				t.letGoForLimit(runID)
			}
			if firstLook && left > terminalFarReset {
				look = terminalFarResetFirstLook
			}
			firstLook = false
			why := fmt.Sprintf("the subscription's rate limit resets at %s", at.UTC().Format(time.RFC3339))
			if !probing || left <= look {
				t.evidence.add(evidenceEntry{Event: "decided", Decided: "wait until the reset named", NextLook: at.UTC().Format(time.RFC3339)}, now)
				t.sleep(ctx, max(left, TerminalWaitFloor), why)
				return ""
			}
			t.evidence.add(evidenceEntry{Event: "decided", Decided: "wait for the reset named, asking at a low rate", NextLook: now.Add(look).UTC().Format(time.RFC3339)}, now)
			t.sleep(ctx, look, why+"; it is asked again after this wait")
		} else {
			if up {
				t.say("stopping: the subscription has met its rate limit for eight hours and nothing says when it resets; run %s later", t.resumeCommand())
				return TerminalLimited
			}
			if t.bounded() {
				d = min(d, terminalProbeCap)
			}
			if probing && now.Sub(t.limit.since) >= terminalReleaseAfter {
				t.letGoForLimit(runID)
			}
			t.evidence.add(evidenceEntry{Event: "decided", Decided: "probe, since nothing says when it resets", NextLook: now.Add(d).UTC().Format(time.RFC3339)}, now)
			t.sleep(ctx, d, "the subscription has met its rate limit, and nothing says when it resets; it is asked again after this wait")
			if !probing {
				return ""
			}
		}
		if ctx.Err() != nil || t.stopped() || t.stopHolds() || !t.now().Before(t.o.Until) {
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
			t.say("the subscription's rate limit has lifted: a probe was answered")
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

// WaitOutProof waits out what stopped the drive's live proof short of a verdict: a
// limit, on the schedule any agent's limit waits on, or a launch that came to none,
// on the outage's unless it named its own wait. Each time it may go on it proves again. It returns why the wait
// ended the run, "" when prove has run clean or come to a verdict, and then what
// prove said. Nothing has started: what it may renew is what an earlier run left held.
func WaitOutProof(ctx context.Context, repoRoot string, o TerminalOptions, first error, prove func(context.Context) error) (string, error) {
	t := newTerminal(repoRoot, o)
	t.ctx = ctx
	err := first
	for {
		var limit *runner.LimitError
		var none *runner.NoVerdictError
		switch {
		case errors.As(err, &limit):
			entry := evidenceEntry{Event: "limit", Role: "live proof", Seen: limit.Seen}
			if !limit.ResetsAt.IsZero() {
				entry.ResetRead = limit.ResetsAt.UTC().Format(time.RFC3339)
			}
			t.evidence.begin("limit", entry, t.now())
			if ended := t.limited(ctx, "", RoleImplementer, limit.ResetsAt); ended != "" {
				return ended, nil
			}
		case errors.As(err, &none):
			t.evidence.begin("suspect-live-proof", evidenceEntry{Event: "failure", Role: "live proof", Response: none.Detail, Seen: none.Seen}, t.now())
			if none.RetryAfter > 0 {
				t.sleep(ctx, none.RetryAfter, none.Detail)
			} else if t.failed(ctx, false, "the live check came to no verdict: "+none.Detail) {
				return TerminalOutage, nil
			}
		default:
			return "", err
		}
		switch {
		case ctx.Err() != nil:
			return TerminalKilled, nil
		case t.stopped():
			return TerminalStopped, nil
		case t.stopHolds():
			return TerminalHalted, nil
		case t.pastBound():
			return TerminalUntil, nil
		case !t.wallHeld():
			return TerminalWall, nil
		}
		err = prove(ctx)
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
	if res.stepNote != "" {
		t.say("%s %s: %s", runID, res.Lane, res.stepNote)
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
		if t.netRepeats(runID, r) {
			return false, "", nil
		}
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
	case ctx.Err() != nil || t.stopped() || t.stopHolds():
		return false, "", nil
	default:
		return t.laneRefused(ctx, runID, lane.ID, "its preflight failed: "+perr.Error(), "")
	}
}

// preflight runs a lane's preflight on a context a stop also ends, this terminal's or one
// set from outside it, so one queued at the load gate or running does not outlast it.
func (t *terminal) preflight(ctx context.Context, worktree string) error {
	pctx, cancel := context.WithCancel(ctx)
	defer cancel()
	looked := make(chan struct{})
	go func() {
		defer close(looked)
		look := time.NewTicker(wallLookEvery)
		defer look.Stop()
		for {
			select {
			case <-t.o.Stop:
				cancel()
				return
			case <-pctx.Done():
				return
			case <-look.C:
				if t.stopHolds() {
					cancel()
					return
				}
			}
		}
	}()
	var err error
	t.beginCall("the preflight")
	t.renewing(func() { err = t.o.Preflight(pctx, worktree) })
	cancel()
	<-looked
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

// takeUp reserves again a lane a drive let go, as it ended or for a limit. It reports
// true when the lane cannot go on here yet, with why the run ends, if it does. Someone
// else holding its issue now, or the issue finished on the default branch, drops the
// lane, and the next move goes on. Anything else may pass, a failed push or the
// person's own cap among them, so the lane waits on the outage's deadline with
// nothing logged against it.
func (t *terminal) takeUp(ctx context.Context, runID string) (bool, string) {
	run, err := ReadState(t.repoRoot, runID)
	if err != nil {
		return false, ""
	}
	// Nothing reserves while a stop's mark stands. The run takes what it can of the stop.
	// One a person's stop halted waits for an explicit take-over, after the lift too. One
	// this drive's own signal halted is its own terminal's person's to go on with, so it
	// is taken over here, which reserves it again and clears the mark in that order.
	if c, err := runControl(t.repoRoot, runID); err == nil && c.Stop != nil {
		_, checked := stopCheck(t.repoRoot, runID, t.o.Stages, t.o.Options)
		c, err = runControl(t.repoRoot, runID)
		switch {
		case err != nil:
			return false, ""
		case c.Stop != nil && c.Stop.HaltedAt != nil && c.Stop.FromFile:
			return t.passOverHalted(run)
		case c.Stop != nil && c.Stop.HaltedAt != nil:
			return t.takeOver(ctx, run)
		case c.Stop != nil:
			if t.haltedBy(checked) {
				return true, TerminalHalted
			}
			return false, ""
		}
		if run, err = ReadState(t.repoRoot, runID); err != nil {
			return false, ""
		}
	}
	if !letGoByDrive(run) {
		// A release that never reached origin reads as held once a session's renewal has
		// logged itself, so what it marked is told again here. A reservation made tells it
		// again with the reservation (reserveResumed).
		t.retellLimit(runID)
		return false, ""
	}
	err = reserveResumed(t.repoRoot, &StartResult{RunID: runID}, t.takeUpOptions())
	return t.tookUp(ctx, run, err)
}

// takeUpOptions is what a drive reserves its own lanes again with: a drain's act, so a
// hand-back standing on an issue backs it off as it backs off any drain. A person
// resuming the same run shares the reservation and not the mark.
func (t *terminal) takeUpOptions() Options {
	o := t.o.Options
	o.drain = true
	return o
}

// retellLimit tells again what a limit's release marked on a run that holds its
// reservation after all. A failure is said, and the next move tries again.
func (t *terminal) retellLimit(runID string) {
	now := t.now()
	err := mutate(t.repoRoot, runID, func(_ *os.Root, st *State) (bool, error) {
		told := retellKept(st, limitKept, now)
		if len(told) > 0 {
			st.UpdatedAt = now
		}
		return len(told) > 0, nil
	})
	if err != nil {
		t.say("%s: what waited on the limit could not be told again: %s", runID, err.Error())
	}
}

// noteRecordReleased records, as a drive or a stop lets the issue go, the blob of its record
// on origin's default branch as last fetched, and, for a release that did not reach
// origin, the tip it left. A record it cannot read leaves no note, and with no note
// reserving again compares nothing.
func noteRecordReleased(repoRoot string, st *State, failed bool, now time.Time) {
	if blob, _, _ := claim.RecordOnDefault(repoRoot, st.Key); blob != "" {
		st.Record = append(st.Record, Entry{At: now, Stage: StageReleasedOver, Note: blob})
	}
	if !failed {
		return
	}
	snap, err := claim.View(repoRoot, now)
	if err != nil {
		return
	}
	if r, ok := snap.Lookup(st.Key); ok && r.Copy == claim.CopyThis && (r.State == claim.StateLive || r.State == claim.StateExpired) {
		st.Record = append(st.Record, Entry{At: now, Stage: StageDriveReleaseFailed, Note: r.Commit})
	}
}

// failedReleaseTip is the tip the run's latest release left when it did not reach origin,
// "" when that release did, a later one reached origin, or the run has reserved since.
func failedReleaseTip(run State) string {
	for i := len(run.Record) - 1; i >= 0; i-- {
		switch run.Record[i].Stage {
		case StageDriveReleaseFailed:
			return run.Record[i].Note
		case StageReleasedOver, StageDriveReleased, StageReservationReleaseFailed:
			continue
		case StageReservationReleased, StageReservation, StageReservationRenewed:
			return ""
		}
	}
	return ""
}

// releasedOverBlob is the blob the run's latest release noted, "" when it noted none or
// the run has reserved since.
func releasedOverBlob(run State) string {
	for i := len(run.Record) - 1; i >= 0; i-- {
		switch run.Record[i].Stage {
		case StageReleasedOver:
			return run.Record[i].Note
		case StageReservation, StageReservationRenewed:
			return ""
		}
	}
	return ""
}

// recordFetch is claim.Fetch, a seam a test replaces to make the comparison's fetch fail.
var recordFetch = claim.Fetch

// recordMovedOnWith is who an issue is with whose record changed while it was let go.
const recordMovedOnWith = "whoever changed its record"

// recordSince is what comparing the issue's record with the run's note came to: moved says
// how it changed, unread why it could not be compared, and neither means it stands as noted.
type recordSince struct{ moved, unread string }

// recordMovedOn compares the issue's record on origin's default branch, freshly fetched, with
// what the run noted as it let the issue go. The note proves the record stood then, so a
// branch that can be read and holds no file for it is a change. A finished record is the
// reservation's to refuse, which says so.
func recordMovedOn(repoRoot string, run State, now time.Time) recordSince {
	noted := releasedOverBlob(run)
	if noted == "" {
		return recordSince{}
	}
	notCompared := func(why string) recordSince {
		return recordSince{unread: fmt.Sprintf("%s's record could not be compared with what stood when the issue was let go: %s", run.Key, why)}
	}
	if _, err := recordFetch(repoRoot, claimOptions(now)); err != nil {
		return notCompared(claim.Remote + " could not be fetched")
	}
	blob, files, readable := claim.RecordOnDefault(repoRoot, run.Key)
	switch {
	case !readable:
		return notCompared(claim.Remote + "'s default branch cannot be read")
	case claim.Finished(repoRoot, run.Key):
		return recordSince{}
	case files == 0:
		return recordSince{moved: fmt.Sprintf("%s's record is gone from %s's default branch, where it stood when the issue was let go", run.Key, claim.Remote)}
	case files > 1:
		return notCompared(claim.Remote + "'s default branch holds it in more than one file")
	case blob == noted:
		return recordSince{}
	}
	return recordSince{moved: fmt.Sprintf("%s's record on %s's default branch changed while the issue was let go", run.Key, claim.Remote)}
}

// dropMovedRun ends the lane of a run whose record changed while it was let go, and returns
// the reason it goes back for. Only a release of the run's own that did not reach origin is
// tried again, and only over the tip it left, so a reservation anyone made since stands,
// and the reason says which.
func dropMovedRun(repoRoot string, run State, why string, now time.Time) (string, error) {
	if tip := failedReleaseTip(run); tip != "" {
		clause, failed := releaseReservationAt(repoRoot, run.Key, tip, "", now)
		why += "; " + clause
		if failed {
			why += ", so it lapses with its lease"
		} else if err := mutate(repoRoot, run.RunID, func(_ *os.Root, st *State) (bool, error) {
			st.Record = append(st.Record, Entry{At: now, Stage: StageReservationReleased, Note: clause})
			st.UpdatedAt = now
			return true, nil
		}); err != nil {
			return why, err
		}
	} else if snap, err := claim.View(repoRoot, now); err == nil {
		if r, ok := snap.Lookup(run.Key); ok && r.Copy == claim.CopyThis && r.State == claim.StateLive {
			why += "; the reservation made on it from this copy since it was let go stands"
		}
	}
	return why, dropRun(repoRoot, run.RunID, why, recordMovedOnWith, now)
}

// passOverHalted leaves a run a person's stop halted to its explicit take-over, names it and
// has the drain go on to other work, as takeUp reports.
func (t *terminal) passOverHalted(run State) (bool, string) {
	err := fsutil.WithFileLock(filepath.Join(t.repoRoot, filepath.FromSlash(drainLockRel)), lockTimeout, func() error {
		st, _, err := readDrain(t.repoRoot)
		if err != nil {
			return err
		}
		for i := range st.Lanes {
			if st.Lanes[i].RunID == run.RunID && st.Lanes[i].Outcome == DrainLaneInProgress {
				st.Lanes[i].Outcome = DrainLaneHalted
				st.UpdatedAt = t.now()
				return writeDrain(t.repoRoot, st)
			}
		}
		return nil
	})
	if err != nil {
		t.stoppedBy(fmt.Errorf("%s could not be passed over: %w", run.RunID, err))
		return true, TerminalWall
	}
	t.notePassedOver(Excluded{ID: run.Key, Check: CheckNeedsPerson, Reason: waitsForTakeOver(run.RunID)})
	return true, ""
}

// notePassedOver says an issue needs a person, once.
func (t *terminal) notePassedOver(p Excluded) {
	if slices.ContainsFunc(t.passedOver, func(e Excluded) bool { return e.ID == p.ID }) {
		return
	}
	t.passedOver = append(t.passedOver, p)
	t.say("passed over %s: %s", p.ID, p.Reason)
}

// takeOver takes a run a stop halted over, as takeUp reports: a stop that still holds ends
// the run, as does a take-over that says not to step; a session at the same take-over is
// waited for; and a reservation it could not take is answered as any other is.
func (t *terminal) takeOver(ctx context.Context, run State) (bool, string) {
	res, err := TakeRun(t.repoRoot, run.RunID, t.takeUpOptions())
	if err == nil {
		t.say("took over %s, run %s: its reservation is held again", run.Key, run.RunID)
		for _, line := range res.Left {
			t.say("%s", line)
		}
		return false, ""
	}
	if t.haltedBy(err) {
		return true, TerminalHalted
	}
	if r, ok := AsRefusal(err); ok {
		switch r.Stage {
		case StageResume:
			t.stoppedBy(fmt.Errorf("%s; %s", r.Reason, r.Remedy))
			return true, TerminalWall
		case StageDriver:
			if t.idle(ctx, TerminalWaitFloor, r.Reason) {
				return true, TerminalWaited
			}
			return true, ""
		}
	}
	return t.tookUp(ctx, run, err)
}

// tookUp says what became of reserving the lane of run again.
func (t *terminal) tookUp(ctx context.Context, run State, err error) (bool, string) {
	runID := run.RunID
	if err == nil {
		t.say("took up %s, run %s: its reservation is held again", run.Key, runID)
		return false, ""
	}
	r, ok := AsRefusal(err)
	if !ok {
		r = refuse(StageReservation, "", "", err.Error(), "see the reason")
	}
	if r.recordMoved {
		reason, derr := dropMovedRun(t.repoRoot, run, r.Reason, t.now())
		t.sayDropped(run, reason, derr)
		return true, ""
	}
	if r.Contention && r.Stage == StageReservation {
		t.dropLane(run, r)
		return true, ""
	}
	// Another invocation holding the run lock passes, so it is waited out like a driver's
	// and is no answer about the issue.
	if r.Contention && r.Stage == "state" {
		if t.idle(ctx, TerminalWaitFloor, r.Reason) {
			return true, TerminalWaited
		}
		return true, ""
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

// limitKept is the reason a release for a limit marks a role out on the lane with, so
// a stop reads no agent there and taking the lane up tells it again (retellKept).
const limitKept = "let go for a long limit"

// letGoForLimit lets the issue of the lane in hand go for a limit that lasts: its work
// saved as a stop saves it, then the reservation released, so no peer is kept off a
// wait no one is working through. Only a lane a limit interrupted is let go. One at its
// pull request keeps what it holds, and so does one with an agent in a call or work no
// save could take, which are said and tried again at the wait's next turn; a stop's mark
// leaves it to the stop. The run stays the driver's, and no person's: it is marked as a
// drive's letting go, which the next take-up answers (takeUp). A release that does not
// reach origin leaves the tip standing: neither the wait nor a move renews what it let go,
// so the tip lapses with its lease unless the take-up or a session waiting in this checkout
// (RenewRunsIn) takes it up first, and either way the lane is told again.
func (t *terminal) letGoForLimit(runID string) {
	if runID == "" || t.stopped() || t.stopHolds() || driveStopped(runID) {
		return
	}
	now := t.now()
	key, held, clause, failed := "", "", "", false
	err := mutate(t.repoRoot, runID, func(root *os.Root, st *State) (bool, error) {
		if st.ReservedAt == nil || letGoByDrive(*st) {
			return false, nil
		}
		if outcome, _, _ := laneOutcome(*st); outcome != DrainLaneInProgress {
			return false, nil
		}
		c, err := readControl(root, runID)
		if err != nil || c.Stop != nil {
			return false, err
		}
		key = st.Key
		if tripped := configTripwireIn(t.repoRoot, root, runID); tripped != nil {
			held = "nothing was saved aside: " + tripped.Error()
			return false, nil
		}
		if why := limitKeepsHold(*st, c, now); why != "" {
			held = why
			return false, nil
		}
		for i := range st.Lanes {
			lane := st.Lanes[i]
			if stopFinished(lane) {
				continue
			}
			out := StopLeft{Lane: lane.ID, Stage: lane.Stage}
			if !saveLaneForStop(t.repoRoot, root, st, i, limitKept, &out, now) {
				held = out.Note
			}
			if out.Aside != "" {
				st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Stage: StageCheckpoint,
					Note: fmt.Sprintf("%d uncommitted file(s) saved aside at %s", out.Files, out.Aside)})
			}
		}
		if held != "" {
			// What was marked goes on as it was, since the lane does.
			retellKept(st, limitKept, now)
			st.UpdatedAt = now
			return true, nil
		}
		// Noted after the release's fetch, so a change the lane never fetched is no change
		// while it was let go.
		clause, failed = releaseReservation(t.repoRoot, st.Key, "", now)
		noteRecordReleased(t.repoRoot, st, failed, now)
		st.Record = append(st.Record, Entry{At: now, Stage: StageDriveReleased, Note: clause})
		st.UpdatedAt = now
		return true, nil
	})
	switch {
	case err != nil:
		t.say("%s: the reservation stays held, since the run could not be saved: %s", runID, err.Error())
	case held != "":
		t.say("%s: the reservation stays held for now: %s", key, held)
	case failed:
		t.say("%s: the limit lasts, and the release did not reach %s, so the reservation stands until its lease lapses or a renewal takes it up: %s", key, claim.Remote, clause)
	case clause != "":
		t.say("%s: the limit lasts, so the issue is let go until it lifts: %s", key, clause)
	}
}

// limitKeepsHold says why a limit does not let st's lanes go, "" when it does: a lane
// with a pull request open waits on that and not the limit, and an agent still in its
// call, which a limit met by another role says nothing of, works on what it holds. A role
// out with no running claim has no agent, since a drive starts every agent it awaits.
func limitKeepsHold(st State, c control, now time.Time) string {
	var seen *sessionname.Table
	if tab, err := readProcs(); err == nil {
		seen = &tab
	}
	for _, lane := range st.Lanes {
		if stopFinished(lane) {
			continue
		}
		if lane.PR > 0 {
			return fmt.Sprintf("%s has pull request #%d open", lane.ID, lane.PR)
		}
		for _, a := range lane.Awaits {
			if a.Running == nil {
				continue
			}
			if s := agentState(a, c, seen, now); s == AgentAlive || s == AgentUnknown {
				return fmt.Sprintf("%s's %s is still in its call", lane.ID, a.Role)
			}
		}
	}
	return ""
}

// dropLane ends a lane whose reservation backed off, naming who the issue is with now.
func (t *terminal) dropLane(run State, r *Refusal) {
	t.sayDropped(run, r.Reason, dropRun(t.repoRoot, run.RunID, r.Reason, t.issueIsWith(run.Key), t.now()))
}

// sayDropped says what became of dropping run's lane for reason.
func (t *terminal) sayDropped(run State, reason string, err error) {
	if err != nil {
		t.say("%s, run %s, could not be dropped, and stays as it is: %s", run.Key, run.RunID, err.Error())
		return
	}
	t.say("%s, run %s, is dropped: %s", run.Key, run.RunID, reason)
}

// issueIsWith reads, after a back-off, who the issue is with, in words true of the cause.
func (t *terminal) issueIsWith(key string) string {
	now := t.now()
	if claim.Finished(t.repoRoot, key) {
		return "no one: the issue has finished on the default branch"
	}
	snap, err := claim.View(t.repoRoot, now)
	if err != nil {
		return "no one this drive can name, since the reservations cannot be read"
	}
	if snap.HandBack(key, now) != "" {
		return "a person: a hand-back stands on the issue until its record changes"
	}
	r, ok := snap.Lookup(key)
	switch {
	case ok && r.State == claim.StateUnreadable:
		return "no one this drive can name, since the tip of its reservation cannot be read"
	case ok && r.Copy != claim.CopyThis && (r.State == claim.StateLive || r.State == claim.StateExpired):
		if claim.ReaderSelf(t.repoRoot, claimOptions(now)).Owns(r.Holder) {
			return "your other copy, which holds it"
		}
		return "the peer who holds it"
	}
	return "the holder of the reservation the refusal names"
}

// dropRun ends a run's lane whose issue stopped being its to work on while it held nothing:
// the reservation backed off, or the record changed. It goes back holding nothing, so no
// release is pushed and no hand-back is published over an issue that is no person's
// question, and the next move routes it and takes the next issue. What the lane left stays
// where the hand-back says.
func dropRun(repoRoot, runID, why, whose string, now time.Time) error {
	reason := termsafe.Sanitize(fsutil.RedactHome(why))
	return mutate(repoRoot, runID, func(_ *os.Root, st *State) (bool, error) {
		i := st.current()
		if i < 0 || st.Lanes[i].HandBack != nil {
			return false, nil
		}
		lane := st.Lanes[i]
		home := fmt.Sprintf("%s; the run %s keeps its worktree and branch %s", whose, st.RunID, lane.Branch)
		if rel, err := laneRel(st.RunID, lane.ID, stageRestart); err == nil {
			home += ", and what the lane left under " + rel + "/" + AsideDirName
		}
		home += ", until `abcd implement step --run " + st.RunID + " --discard " + lane.ID + "` discards them"
		handBackLane(st, &lane, HandBack{Kind: HandBackStuck, Reason: reason, Home: home}, "", now)
		st.Lanes[i] = lane
		st.ReservedAt = nil
		st.UpdatedAt = now
		return true, nil
	})
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
			// A stop that could not settle the lane holds its claims with its work.
			if c, err := runControl(t.repoRoot, l.RunID); err == nil && c.Stop != nil {
				continue
			}
			clause, failed := releaseReservation(t.repoRoot, run.Key, "", now)
			t.say("%s: %s", run.Key, clause)
			_ = mutate(t.repoRoot, l.RunID, func(_ *os.Root, st *State) (bool, error) {
				noteRecordReleased(t.repoRoot, st, failed, now)
				st.Record = append(st.Record, Entry{At: now, Stage: StageDriveReleased, Note: clause})
				st.UpdatedAt = now
				return true, nil
			})
		}
		if j := st.Judging; j != nil && live {
			if j.Hold != nil {
				clause, _ := releaseReservationAt(t.repoRoot, j.Issue, j.Hold.Commit, "", now)
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
	defer t.leaveSharedRun()
	if why == TerminalWall && t.wallEnded != "" {
		why = t.wallEnded
	}
	switch why {
	case TerminalStopped:
		t.haltOnSignal()
	case TerminalHalted:
		t.settleHalted()
	}
	if why != TerminalKilled && why != TerminalHalted {
		t.letGo()
	}
	res := TerminalResult{Ended: why, Last: t.last, HeldBack: t.heldBack, PassedOver: t.passedOver}
	if why == TerminalWall || why == TerminalHalted {
		res.Why = t.why
	}
	st, _, err := readDrain(t.repoRoot)
	if err != nil {
		return res
	}
	for _, l := range st.Lanes {
		if l.Outcome != DrainLaneInProgress {
			continue
		}
		res.Run, res.Issue = l.RunID, l.Issue
		run, rerr := ReadState(t.repoRoot, l.RunID)
		// A later drive takes up no parked lane, so the command that would is not named.
		parked := rerr == nil && parkedAtLand(run) >= 0
		// Nor is any after a hold to a run's profile, whose reason names the ways on.
		if why != TerminalHalted && !driveStopped(l.RunID) && !parked && !t.heldToProfile {
			res.Resume = TerminalResumeCommand
		}
		if parked {
			if note := waitsForItsTakeOver(t.repoRoot, l.RunID); note != "" {
				t.say("%s: %s", l.Issue, note)
			}
		}
		if rerr == nil {
			if i := run.current(); i >= 0 {
				res.Lane = run.Lanes[i].ID
			}
		}
		if why == TerminalKilled {
			if snap, err := claim.View(t.repoRoot, t.now()); err == nil {
				if r, ok := snap.Lookup(l.Issue); ok && r.Copy == claim.CopyThis && r.State == claim.StateLive {
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

// ReviewSetFor is the review set a run started now would run on, as a drive's
// start names it before it asks (itd-2611101419006280 decision 8).
func ReviewSetFor(o Options, repoRoot string) (ReviewSet, error) {
	review, _, err := resolveReviewSet(o.roots(repoRoot), o)
	return review, err
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
