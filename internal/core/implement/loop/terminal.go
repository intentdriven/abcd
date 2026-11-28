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
	"strings"
	"sync"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/recordid"
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
	// group was killed, and what it held lapses on its own unless a load-gate
	// waiter in this copy renews it (gateWaiterRenews).
	TerminalKilled = "killed"
	// TerminalPassedLanes: the only lanes the drain has left in progress are ones the
	// drive passed over, since it may not save them (PassedLane): the drain stays open,
	// and a drive that may save them takes them up.
	TerminalPassedLanes = "passed-lanes"
)

// PassedLane is a lane a drive passed over: its run's file states another schema
// version, and the judge of a whole save (mayWholeSave) refused this drive the save
// taking it up or taking it over needs. Reason is that judge's reason and remedy. From
// the pass to the drive's end, whatever the reason, the drive writes nothing for the
// lane: no save, claim, reservation pushed or renewed, or line in its record, and no
// agent runs for it.
type PassedLane struct {
	Issue  string `json:"issue"`
	Run    string `json:"run"`
	Reason string `json:"reason"`
}

// SetAsideLane is a lane a drive set aside (itd-2611141610197753 step 3; decisions 3, 4,
// 5, 8 and 9): it waited the stuck bound's minutes (pace.stuck_minutes) on something this
// drive cannot end, which the stuck bounds never count: a run whose driver it could not
// claim at the lane's opening or take over, a take-up of the lane that was refused, or
// the landing's issue record gone from the checkout. From then the drive writes nothing
// for it, as for a passed lane (DrainOptions.PassedLanes), and its reservation is left to
// lapse at its lease (claim.LeaveToLapse): neither released nor renewed. Once no other
// lane is in progress the drive tries it again, and then once an hour, until its end.
type SetAsideLane struct {
	Issue  string `json:"issue"`
	Run    string `json:"run"`
	Reason string `json:"reason"`
	// Since is when the lane began to wait, and At when it was set aside.
	Since time.Time `json:"since"`
	At    time.Time `json:"at"`
	// LeftToLapse is true when the drive left the lane's reservation to lapse at its lease
	// (leaveToLapse). A lane a live driver holds is not: that driver is still working it,
	// so its own renewals keep the hold (dcn-2611270753395749).
	LeftToLapse bool `json:"left_to_lapse,omitempty"`
	// Holds names each issue of the lane, and whether its hold was left to lapse, where
	// only some of them could be (leaveToLapse; LeftToLapse is false then): a renewal of
	// the run keeps every hold of it while any is not left (leftToLapse), so the drive's end
	// names them issue by issue rather than say the lane's reservation is left to lapse
	// (dcn-2611271952390075).
	Holds []SetAsideHold `json:"holds,omitempty"`
	// site is where it waited (blockClaim and its kin), which its next try compares.
	site string
}

// SetAsideHold is one issue of a set-aside lane only some of whose holds were left to lapse,
// and whether its own was.
type SetAsideHold struct {
	Issue       string `json:"issue"`
	LeftToLapse bool   `json:"left_to_lapse"`
}

// SplitHolds is the issues of holds whose own hold was left to lapse, and those whose was not.
func SplitHolds(holds []SetAsideHold) (left, kept []string) {
	for _, h := range holds {
		if h.LeftToLapse {
			left = append(left, h.Issue)
		} else {
			kept = append(kept, h.Issue)
		}
	}
	return left, kept
}

// heldByLiveDriver reports whether site is a block on a live driver's claim of the run, the
// claim at a lane's opening or at a take-over: a session is working the run, so the drive
// that sets the lane aside leaves its renewals to that session.
func heldByLiveDriver(site string) bool {
	return site == blockClaim || site == blockTakeOver
}

// Where a lane waits on what a drive cannot end, which sets it aside at the stuck bound's
// minutes (terminal.blocked).
const (
	// blockClaim: the claim of the run's driver at the lane's opening (driveLane).
	blockClaim = "claim"
	// blockTakeOver: the claim of a halted run's driver as it is taken over (takeOver).
	blockTakeOver = "take-over"
	// blockTakeUp: the reservation of a let-go lane taken up again (tookUp).
	blockTakeUp = "take-up"
	// blockRecord: the landing's issue record, in no status folder (Refusal.recordAbsent).
	blockRecord = "record"
)

// terminalSetAsideRetry is how often a drive whose every lane left is set aside tries them
// again (itd-2611141610197753 decision 9): a look costs seconds, and a lane that stays
// blocked is set aside again at once.
const terminalSetAsideRetry = time.Hour

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
	// Preflight runs the repository's preflight behind the load gate in
	// checkout, the drive's own checkout of a lane's head, whose full commit
	// id is head (preflightcheckout.go), never the lane's worktree.
	Preflight func(ctx context.Context, checkout, head string) error
	// PreflightAllowed says whether a lane's preflight may run, given its
	// worktree, base and head: the head its landing lands, the judged head or
	// the landing's records commit on it, never what the worktree has checked
	// out. A *PreflightRefused hands the lane back. Nil allows all.
	PreflightAllowed func(worktree, base, head string) error
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
	// State is handed the drive's state each time the drive writes it (drivestate.go),
	// as it is stored, and one call at a time: at each look of a wait, at each renewal
	// tick of an agent's call or a preflight, at each movement and at the end. Nil is
	// handed nothing; the file is written either way.
	State func(DriveState)
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
	// Lapses is when the lane's reservation lapses, when a kill left it held or a signal's
	// stop could not be applied to its run.
	Lapses *time.Time `json:"lapses,omitempty"`
	// LeftToLapse says the lane's reservation was left to lapse at Lapses, the stop
	// having not been applied to its run: no renewal that reads no driver renews it
	// (claim.LeaveToLapse), so only a take-up works it again.
	LeftToLapse bool `json:"left_to_lapse,omitempty"`
	// Last is the last move's result.
	Last DrainResult `json:"last"`
	// HeldBack names each issue the run held back.
	HeldBack []string `json:"held_back,omitempty"`
	// PassedOver names each issue the run did not open, with why.
	PassedOver []Excluded `json:"passed_over,omitempty"`
	// PassedLanes names each lane the run passed over, with why.
	PassedLanes []PassedLane `json:"passed_lanes,omitempty"`
	// SetAside names each lane still set aside as the run ended, with how long it
	// waited and why (SetAsideLane).
	SetAside []SetAsideLane `json:"set_aside,omitempty"`
	// Why is what stopped the run when a check did.
	Why string `json:"why,omitempty"`
	// Swept is the line of the sweep the drive ran on its own at its end
	// (autosweep.go), "" when it ran none or it said nothing.
	Swept string `json:"swept,omitempty"`
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
	// unmarked names the runs whose profile markProfile said it could not mark, so it says
	// so once a drive.
	unmarked []string
	// idleSince is when the run last stopped moving, zero while it moves.
	idleSince time.Time
	// ranClean names the lane and head a preflight last exited 0 on, and
	// ranCleanHeld why that landing was held once it had, "" when it was not
	// (ownPreflight): a later step on the same head is refused again with it,
	// and the preflight is not run again.
	ranClean     string
	ranCleanHeld string
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
	// notedParked is what each parked run's watch last said (sayParked), so it
	// is said once.
	notedParked map[string]string
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
	// passedLanes are the lanes passed over because the drive may not save them, each
	// said once, and given to every move after (DrainOptions.PassedLanes).
	passedLanes []PassedLane
	// why is what stopped the run, when a check did, and wallEnded the reason a wait
	// inside that check ended it for, when one did.
	why       string
	wallEnded string
	// judge is what the last move's routed judgement came to.
	judge JudgeOutcome
	// minted names each head a preflight this run ran exited 0 on in the
	// drive's own checkout of that head, which held the head's content before
	// and after (ownPreflight); it is the drive's memory, never a file. bases each
	// lane's base as the run first read it.
	minted map[string]bool
	bases  map[string]string
	// stepped are the runs the start took and steps that no drain lists (takeEarlier),
	// which a signal halts and the drive's end lets go as it does the drain's lanes.
	stepped []string
	// unstopped are the runs a signal's stop could not be applied to (haltOnSignal): no
	// mark stands on them, so the drive's end keeps their reservations, which lapse with
	// their leases, rather than leave a run unmarked and unreserved (criterion 10).
	unstopped []string
	// stuckGiven is the run whose lane the drive last gave back to the next move as
	// stuck (stuckBack), "" once a lane it steps is not stuck.
	stuckGiven string
	// blocks are, by run, where and since when a lane has waited on what this drive cannot
	// end (blocked), until a stage moves or it is set aside.
	blocks map[string]laneBlock
	// setAside are the lanes set aside and not yet tried again, in the order set aside;
	// retried are those given back to the moves for one look (retrySetAside), by run; and
	// nextRetry is when the lanes set aside are next tried again.
	setAside  []SetAsideLane
	retried   map[string]SetAsideLane
	nextRetry time.Time
	// lapsing are the runs every hold of which this drive left to lapse (leaveToLapse), by
	// run.
	lapsing map[string]bool
	// The drive's state (drivestate.go). stillSince is when anything last moved, set by the
	// first wait after a movement, whatever the wait, and cleared by the next movement
	// (moving), so an outage and a limit measure from their first failure and waits whose
	// reasons alternate keep one start. wait is what the next sleep waits on, which its
	// caller names (waitFor). streak is the lane hand-backs in a row, and prsSeen the
	// pull requests the last move counted, so a lane that reaches one clears it. onLane is
	// the lane a call works on, for a working state.
	stillSince time.Time
	wait       driveWait
	streak     handBackStreak
	prsSeen    int
	onLane     string
	// stateMu serialises every write of the drive's state and every State call: the loop
	// writes from its own goroutine, and a renewal tick (renewing) from another while an
	// agent is in its call. state is the value last written, which a tick writes again;
	// stateErr the last write's error, said once; proc the drive's own process.
	stateMu  sync.Mutex
	state    DriveState
	stateErr string
	proc     *DriveProcess
}

// laneBlock is where a lane waits on what the drive cannot end, and since when.
type laneBlock struct {
	site  string
	since time.Time
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
	o.Drain.driveEnds = true
	if o.Drain.Transcripts == nil {
		o.Drain.Transcripts = o.Runners.Transcripts
	}
	// Every push the reservation code makes while the drive runs passes
	// --no-follow-tags (claim.DrivePushes), as the landing push does
	// (landPushArgs), so no tag a lane made is sent with either.
	defer claim.DrivePushes()()
	return newTerminal(repoRoot, o).run(ctx)
}

// newTerminal is one run, not yet started.
func newTerminal(repoRoot string, o TerminalOptions) *terminal {
	t := &terminal{repoRoot: repoRoot, o: o, minted: map[string]bool{}, bases: map[string]string{}}
	if t.o.Sleep == nil {
		t.o.Sleep = func(ctx context.Context, d time.Duration) { sleepOr(ctx, d, o.Stop, t.stopHolds) }
	}
	t.o.Options.driverIsSelf = true
	// A person's stop ends every sweep the drive runs on its own, a lane's
	// landing's and a drain move's, at once (autosweep.go).
	t.o.Options.sweepStop = o.Stop
	t.o.Drain.JudgeOutcome = &t.judge
	t.o.Options.RunStateWatch = true
	t.o.Options.ReceiptOwn = func(sha string) bool { return t.minted[sha] }
	t.evidence = newDriveEvidence(repoRoot, func(line string) { t.say("%s", line) })
	t.o.Options.RunnerFailed = t.noteFailure
	t.o.Options.RunnerStarting = func(role string) {
		t.startedAgent = true
		t.startCall("the " + role)
		// A judge works on the drain's judgement, never on the lane the drive last drove,
		// which may be done; every other role is a lane's.
		on := t.onLane
		if strings.HasPrefix(role, RoleDrainJudge) {
			on = driveOnJudge
		}
		t.showWorking(on)
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
			if l.Outcome != DrainLaneInProgress || t.passedLane(l.RunID) {
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
// A run the stop could not be applied to, such as one whose file states another version
// and that the drive could not claim, carries no mark, so it is remembered (unstopped) and
// the drive's end keeps its reservation: the lane stays reserved until its lease lapses,
// and the next start reads it as reserved (itd-2611141610197753 criterion 10).
func (t *terminal) haltOnSignal() {
	st, _, err := readDrain(t.repoRoot)
	if err != nil {
		t.say("the lanes this run holds could not be read to halt them: %s", err.Error())
		return
	}
	now := t.now()
	for _, id := range t.heldInProgress(st) {
		if driveStopped(id) {
			continue
		}
		s := runstop.Stop{SetBy: "a signal to the drive", SetAt: now}
		rec, err := haltRun(t.repoRoot, id, s, t.o.Stages, t.o.Options)
		switch {
		case err != nil:
			t.say("%s: the stop could not be applied, so its reservation is kept: %s", id, err.Error())
			t.unstopped = append(t.unstopped, id)
		case rec != nil && rec.HaltedAt == nil:
			t.say("%s: an agent is still in its call, so the next drive settles what it left", id)
		case rec != nil:
			t.say("%s: halted, and what its lanes left is saved", id)
		}
	}
}

// sleep waits d, saying why, in stretches no longer than half a lease, renewing
// what the run's lanes still hold between them, so a long wait keeps those claims. A
// stop, a kill or the person's bound ends it. The wait is measured twice, by the
// time slept and by the wall clock, and whichever ends it first stands: a machine
// that slept through part of it wakes on time, and a clock set back does not
// stretch it. A clock set forward ends it early, which the next launch answers.
//
// The drive's state is written as the wait starts and at each look after a stretch,
// from the wait its caller named (waitFor), measured from when anything last moved
// (stillSince), and due again at the stretch's end.
func (t *terminal) sleep(ctx context.Context, d time.Duration, why string) {
	now := t.now()
	if t.bounded() {
		d = min(d, max(t.o.Until.Sub(now), 0))
	}
	end := now.Add(d)
	asked := d
	defer func() { t.waited += max(asked-d, t.now().Sub(now)) }()
	w := t.wait
	t.wait = driveWait{}
	t.still(now)
	t.sayWait(d, why)
	t.renewHeld()
	t.showWait(w, min(d, terminalRenewEvery))
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
		t.showWait(w, min(d, terminalRenewEvery))
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
// what failed. A lane this drive passed over is renewed by neither, whatever its file
// says (DrainOptions.PassedLanes): a drive's waits leave a passed lane's reservation and
// record alone.
func (t *terminal) renewHeld() {
	now := t.now()
	for _, r := range renewRunsAt(t.repoRoot, "", func(st State) bool { return !letGoByDrive(st) && !t.passedLane(st.RunID) }, now) {
		if !r.Renewed {
			t.say("%s: %s", r.Key, r.Note)
		}
	}
	for _, note := range renewParkedDue(t.repoRoot, now, t.o.Drain.PassedLanes) {
		t.say("%s", note)
	}
}

// idle waits d on something someone else must do, reporting true when nothing
// has moved for as long as a drive waits (adr-2610222222326104 condition 3): the
// run then ends, saying so. Under the person's bound the wait goes on, looking
// less often the longer it lasts, up to its cap.
func (t *terminal) idle(ctx context.Context, d time.Duration, why string) bool {
	now := t.now()
	t.still(now)
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
//
// Its start is the drive's (stillSince), set at the first failure after a movement, and
// its next try is no named end, so an outage reads stalled from the stuck bound's minutes
// on.
func (t *terminal) failed(ctx context.Context, judge bool, what string) bool {
	d, up := t.outage.fail(t.now(), judge)
	if up {
		t.say("outage: %s; eight hours of failures, so the run ends and nothing is handed back", what)
		return true
	}
	why := what + "; trying again"
	on := "a failure, tried again"
	if t.outage.inOutage() {
		why = "outage: " + what + "; nothing is handed back"
		on = "an outage"
	}
	if t.wait.On == "" {
		t.waitFor(on, time.Time{}, driveAct{})
	}
	t.sleep(ctx, d, why)
	return false
}

// What a wait of the drive is on, as its state names it (drivestate.go).
const (
	driveOnHeld     = "what another holds"
	driveOnOrigin   = "origin"
	driveOnPace     = "the pace window"
	driveOnHeldWork = "the drain's cap or work others hold"
	driveOnNextMove = "the drain's next move"
	driveOnLimit    = "the rate limit"
	driveOnProof    = "the live check"
	driveOnStart    = "the drive's start"
	driveOnJudge    = "the drain's judgement"
)

// driveStateMargin is how far past the end of a stretch or a tick the drive's next write
// may come: a write is due at the stretch's end, and the margin covers the look and
// renewal that come before it.
const driveStateMargin = 2 * time.Minute

// waitFor names what the next sleep waits on, for the drive's state: on what, its named
// end (zero for none: decisions 6 and 9 name the only three) and its act.
func (t *terminal) waitFor(on string, until time.Time, act driveAct) {
	t.wait = driveWait{On: on, Until: until, Act: act}
}

// moveWaitOn is what a move's wait is on (moveWait) and its named end: the pace window
// ends at its next eligible time; a look at origin, at the cap or at work others hold
// names no end.
func moveWaitOn(res DrainResult) (string, time.Time, driveAct) {
	switch {
	case res.NextEligibleAt != nil:
		return driveOnPace, *res.NextEligibleAt, driveAct{}
	case res.OriginWait != nil:
		return driveOnOrigin, time.Time{}, driveAct{}
	}
	return driveOnHeldWork, time.Time{}, driveAct{}
}

// moveOn is what a drive works on after a move that moved: the lane it opened or goes
// on with, else the move after.
func moveOn(res DrainResult) string {
	if res.Lane != nil {
		return res.Lane.Issue
	}
	return driveOnNextMove
}

// laneKey is the issue runID's lane works, runID itself where its state cannot be read.
func (t *terminal) laneKey(runID string) string {
	if run, err := ReadState(t.repoRoot, runID); err == nil && run.Key != "" {
		return run.Key
	}
	return runID
}

// still notes the drive has stopped moving: the first wait after a movement sets when.
func (t *terminal) still(now time.Time) {
	if t.stillSince.IsZero() {
		t.stillSince = now
	}
}

// moving notes a movement: the drive works on on, and its next wait is measured from its
// own start.
func (t *terminal) moving(on string) {
	t.stillSince = time.Time{}
	t.showWorking(on)
}

// stateBounds are the stuck bound's minutes and count (pace.stuck_minutes and
// pace.stuck_refusals through the pace's layers), which the stall rules read; the bundled
// ones where they cannot be read, as the set-aside's bound reads them.
func (t *terminal) stateBounds() (time.Duration, int) {
	b, err := resolveStuckBounds(t.o.Options.roots(t.repoRoot))
	if err != nil {
		return BundledStuckMinutes * time.Minute, BundledStuckRefusals
	}
	return time.Duration(b.minutes.Value) * time.Minute, b.refusals.Value
}

// showWait writes the state of the wait w, measured from when anything last moved and
// due again after next: waiting or stalled by the time rule, then the hand-back streak.
func (t *terminal) showWait(w driveWait, next time.Duration) {
	now := t.now()
	w.Start = t.stillSince
	minutes, refusals := t.stateBounds()
	t.writeState(t.streak.over(waitState(w, minutes, now), refusals), now, next)
}

// showWorking writes a working state on on, since the work on it began: a working state on
// the same thing keeps its since, so an agent's call after another on one stage reads as
// one stretch of work.
func (t *terminal) showWorking(on string) {
	now := t.now()
	since := now
	t.stateMu.Lock()
	prev := t.state
	t.stateMu.Unlock()
	if prev.State == DriveWorking && prev.On == driveText(on) {
		since = prev.Since
	}
	_, refusals := t.stateBounds()
	t.writeState(t.streak.over(DriveState{State: DriveWorking, Since: since, On: on}, refusals), now, terminalRenewTick)
}

// showEnded writes the state of a drive that ended as res says: stopped, on its end, and
// with the command that goes on as the act where there is one.
func (t *terminal) showEnded(res TerminalResult) {
	st := DriveState{State: DriveStopped, Since: t.now(), On: "its end: " + res.Ended}
	if res.Resume != "" {
		st.Do = "run `" + res.Resume + "` to go on"
	}
	t.writeState(st, t.now(), terminalRenewEvery)
}

// lanes counts the drive's lanes from the last move and the lanes set aside.
func (t *terminal) lanes() DriveLanes {
	c := t.last.Counts
	aside := len(t.setAside)
	return DriveLanes{Done: c.PullRequests, InFlight: max(c.InProgress-aside, 0), SetAside: aside, HandedBack: c.LanesHandedBack}
}

// process is the drive's own process as a run's driver record names it (selfDriver),
// read once; its id alone where the process table cannot be read. Called under stateMu.
func (t *terminal) process() DriveProcess {
	if t.proc != nil {
		return *t.proc
	}
	p := DriveProcess{PID: os.Getpid()}
	if tab, err := readProcs(); err == nil {
		if h, _, err := selfDriver(tab); err == nil {
			p = DriveProcess{PID: h.PID, StartUnix: h.StartUnix, SlackMS: h.Slack.Milliseconds()}
		}
	}
	t.proc = &p
	return p
}

// writeState writes st, with the drive's lane counts and process, at now, due again after
// next and its margin, under stateMu (storeState).
func (t *terminal) writeState(st DriveState, now time.Time, next time.Duration) {
	st.Lanes = t.lanes()
	t.stateMu.Lock()
	defer t.stateMu.Unlock()
	st.Process = t.process()
	t.state = st
	t.storeState(now, next)
}

// rewriteState writes the state last written again, as it stands, due again after next:
// a renewal tick's, while an agent is in its call or a preflight runs, from the tick's own
// goroutine. Nothing written yet is nothing to write again.
func (t *terminal) rewriteState(next time.Duration) {
	t.stateMu.Lock()
	defer t.stateMu.Unlock()
	if t.state.State == "" {
		return
	}
	t.storeState(t.now(), next)
}

// storeState writes t.state to DriveStateRel (writeDriveState) and hands the value as
// stored to State. A write that fails is said once until one succeeds, and the value is
// handed on still, so the terminal shows the line the file could not hold. The caller
// holds stateMu.
func (t *terminal) storeState(now time.Time, next time.Duration) {
	every := next + driveStateMargin
	stored, err := writeDriveState(t.repoRoot, t.state, now, every)
	if err != nil {
		stored = storedDriveState(t.state, now, every)
		if msg := termsafe.Sanitize(fsutil.RedactHome(err.Error())); msg != t.stateErr {
			t.stateErr = msg
			t.say("the drive's state could not be written to %s: %s", DriveStateRel, msg)
		}
	} else {
		t.stateErr = ""
	}
	if t.o.State != nil {
		t.o.State(stored)
	}
}

// noteHandBacks counts the lane hand-backs res routed in the hand-back streak, keyed by
// the lane's last logged refusal; each lane, a bundle's by its run, counts once in a
// streak however many of its issues it routes and over however many moves
// (handBackStreak.noteLane); a lane that reached its pull request since the last move
// clears the streak first.
func (t *terminal) noteHandBacks(res DrainResult) {
	if res.Counts.PullRequests > t.prsSeen {
		t.streak.landed()
	}
	t.prsSeen = res.Counts.PullRequests
	var st *DrainState
	for _, r := range res.Routed {
		if r.From != "lane" {
			continue
		}
		if st == nil {
			st = &DrainState{}
			if read, _, err := readDrain(t.repoRoot); err == nil {
				st = &read
			}
		}
		lane := r.Issue
		var refused *LaneRefused
		if l, ok := laneOfIssue(*st, r.Issue); ok {
			lane = l.RunID
			refused = lastRefused(t.repoRoot, l.RunID)
		}
		at := t.now()
		if r.At != nil {
			at = *r.At
		}
		t.streak.noteLane(r, lane, refused, at)
	}
}

// laneOfIssue is the lane of st, the last opened, that carries id, as its lead or a member.
func laneOfIssue(st DrainState, id string) (DrainLane, bool) {
	for i := len(st.Lanes) - 1; i >= 0; i-- {
		l := st.Lanes[i]
		if recordid.SameID(l.Issue, id) || slices.ContainsFunc(l.Bundle, func(m string) bool { return recordid.SameID(m, id) }) {
			return l, true
		}
	}
	return DrainLane{}, false
}

// lastRefused is the last refusal runID's lanes logged, nil for none or a state that
// cannot be read: a lane handed back as stuck keeps its run of refusals.
func lastRefused(repoRoot, runID string) *LaneRefused {
	run, err := ReadState(repoRoot, runID)
	if err != nil {
		return nil
	}
	for i := len(run.Lanes) - 1; i >= 0; i-- {
		if r := run.Lanes[i].Refused; r != nil {
			c := *r
			return &c
		}
	}
	return nil
}

// run is the loop.
func (t *terminal) run(ctx context.Context) (TerminalResult, error) {
	t.ctx = ctx
	// The state names this process from the start, over what an earlier drive left.
	t.showWorking(driveOnStart)
	t.joinSharedRun()
	// The settings the drive holds, the hooks directories its git runs and
	// info/attributes are read once, before the drive's first move, and held
	// here for the lanes it opens (remoteid.go, with what it cannot hold): a
	// lane's agent can rewrite the checkout's settings and hooks, and each
	// lane's run records its own afresh. A restarted drive reads them
	// afresh, and so compares an in-tree hooks copy with the tip origin
	// reports at that moment (proveCheckoutHooks). The drive refuses to start
	// when a read the hold depends on gives no answer it can take, the hooks
	// tree is past its bound (readRemoteIdentity), or the comparison finds a
	// difference. The ask of origin is a lane's ask (startHold): it waits on
	// the run's shared outage and reports a lost network to it, and its wait
	// is the drive's own, ended by the bounds every pass of the loop below
	// obeys (boundEnded) and by nothing moving for as long as a drive waits
	// (idle), then asked again.
	var id *remoteIdentity
	for {
		var err error
		if id, err = t.startHold(); err == nil {
			break
		}
		r, ok := AsRefusal(err)
		if !ok || !r.Contention {
			t.leaveSharedRun()
			t.showEnded(TerminalResult{Ended: TerminalRefused})
			return TerminalResult{}, startRefusal(t.repoRoot, err)
		}
		if ended := t.boundEnded(ctx); ended != "" {
			return t.ended(ended), nil
		}
		t.waitFor(driveOnHeld, time.Time{}, driveAct{})
		if t.idle(ctx, TerminalWaitFloor, r.Reason) {
			return t.ended(TerminalWaited), nil
		}
		if ended := t.boundEnded(ctx); ended != "" {
			return t.ended(ended), nil
		}
	}
	t.o.Options.remote = id
	// From here every git command of the drive that runs the repository's
	// hooks, merge drivers or content filters, or reaches the network, first
	// compares what the drive holds (heldBeforeGitIn: heldBeforeGit, reading
	// the own settings of the lane worktrees that git runs in or names): each
	// runner gitutil has for such a command asks the check set here, given
	// the directory the command runs in and its arguments, before it builds
	// the command (gitutil.SetBeforeHookedGit names them), so one made inside
	// a step (the lane's add, commit, merge and reset, the landing push) or
	// outside one (a move's reservation, a renewal while an agent runs, the
	// network probe) never runs under a setting or hook changed since the
	// drive started; it fails instead, naming the difference. One drive runs
	// in a process: a check already set refuses the start.
	unset, err := gitutil.SetBeforeHookedGitAt(func(dir string, args []string) error { return heldBeforeGitIn(t.repoRoot, id, dir, args) })
	if err != nil {
		t.leaveSharedRun()
		t.showEnded(TerminalResult{Ended: TerminalRefused})
		return TerminalResult{}, refuse(StageDrain, "", "", "the drive cannot start: "+err.Error(),
			"run one drive at a time in a process; nothing was made")
	}
	defer unset()
	if t.o.Resumed {
		t.reportHeldBack()
	}
	if ended, err := t.takeEarlier(ctx); err != nil {
		return t.ended(TerminalRefused), err
	} else if ended != "" {
		return t.ended(ended), nil
	}
	for {
		if ended := t.boundEnded(ctx); ended != "" {
			return t.ended(ended), nil
		}
		// Each pass compares what the drive holds (heldBeforeGit): a
		// difference ends the drive, which reads what it holds only as it
		// starts, rather than have each git command of the pass refused.
		if err := heldBeforeGit(t.repoRoot, t.o.Options.remote); err != nil {
			return t.ended(TerminalRefused), err
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
		var res DrainResult
		t.ticking(func() { res, err = Drain(t.repoRoot, t.o.Options, t.o.Drain) })
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
				on := driveOnHeld
				if r.originUnread {
					on = driveOnOrigin
				}
				t.waitFor(on, time.Time{}, driveAct{})
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
		// A bundle's lead whose start was refused is passed over for the
		// rest of the drive (DrainOptions.PassedLeads).
		for _, p := range t.o.Drain.passLeads(res) {
			if p.Contention {
				t.say("passed over %s, a bundle's lead another holds, until the drive's wait on it ends: %s", p.ID, p.Reason)
				continue
			}
			t.say("passed over %s for the rest of this drive, as a bundle's lead whose start was refused: %s", p.ID, p.Reason)
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
		// Only lanes passed over are left. A lane passed over for its file's version is a
		// later drive's, so the drive ends; one set aside is this drive's to try again, so
		// it waits and tries again, ending at its bound or when nothing moves for as long
		// as a drive waits (awaitSetAside; itd-2611141610197753 decision 9).
		if len(res.passedOnly) > 0 {
			if !t.setAsideAmong(res.passedOnly) {
				return t.ended(TerminalPassedLanes), nil
			}
			if ended := t.awaitSetAside(ctx); ended != "" {
				return t.ended(ended), nil
			}
			continue
		}
		if res.Judged != nil {
			t.moved()
		}
		if drainMoved(res) {
			t.idleSince = time.Time{}
			t.moving(moveOn(res))
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
			// A wait on held work, or at the cap, has no lane in progress beside the lanes
			// set aside, so they are tried again when it is time, rather than wait first.
			if res.WaitingUntil != nil && t.retryDue() {
				if ended := t.retrySetAside(ctx); ended != "" {
					return t.ended(ended), nil
				}
				continue
			}
			t.waitFor(moveWaitOn(res))
			if res.WaitingUntil == nil {
				t.sleep(ctx, d, res.Next)
			} else if t.idle(ctx, d, res.Next) {
				return t.ended(TerminalWaited), nil
			} else {
				// The wait on held work is over: a bundle's lead another held
				// is looked at again (DrainOptions.PassedLeads).
				t.o.Drain.retryHeldLeads()
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
			t.waitFor(driveOnNextMove, time.Time{}, driveAct{})
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
		// A lane the take-up in driveLane has just passed over is not in hand, so a new
		// release does not wait on it.
		t.inHand = !t.passedLane(res.Lane.RunID) && t.laneInProgress(res.Lane.RunID)
	}
}

// boundEnded is how the drive ends at the top of each pass, "" when it goes
// on: killed, stopped from the terminal, halted by a stop set from outside,
// past the person's bound, or held by the check before a stage. The start's
// wait on origin obeys it too.
func (t *terminal) boundEnded(ctx context.Context) string {
	switch {
	case ctx.Err() != nil:
		return TerminalKilled
	case t.stopped():
		return TerminalStopped
	case t.stopHolds():
		return TerminalHalted
	case t.pastBound():
		return TerminalUntil
	case !t.wallHeld():
		return TerminalWall
	}
	return ""
}

// startHold is one try at what the drive holds as it starts
// (readRemoteIdentity) and the start's comparison with origin's tip
// (proveCheckoutHooks), the ask of origin made as a lane's is: through the
// run's shared outage (outageFor, which runs the shared network probe when
// it is due), so while the outage has the network down the ask is not made,
// and a lost network is reported to it.
func (t *terminal) startHold() (*remoteIdentity, error) {
	id, err := readRemoteIdentity(t.repoRoot)
	if err != nil {
		return nil, err
	}
	g, err := outageFor(t.repoRoot, "", t.o.Options)
	if err != nil {
		return nil, err
	}
	err = proveCheckoutHooks(t.repoRoot, id, g)
	g.record()
	if note := g.recordNote(); note != "" {
		if r, ok := AsRefusal(err); ok {
			r.Remedy += "; " + note
		}
	}
	if err != nil {
		return nil, err
	}
	return id, nil
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
			t.say("finished %s", laneIssues(f.Issue, f.RunID, f.Bundle))
		case f.Note != "" && t.notedFinish[f.RunID] != f.Note:
			if t.notedFinish == nil {
				t.notedFinish = map[string]string{}
			}
			t.notedFinish[f.RunID] = f.Note
			t.say("%s: %s", f.Issue, f.Note)
		}
	}
	t.sayParked(res.Parked)
	if res.Start != nil && res.Lane != nil {
		t.say("opened the lane of %s: run %s", res.Lane.Issue, res.Lane.RunID)
	}
	t.noteHandBacks(res)
}

// sayParked says two notes of a move's parked lanes, and no other: the line of
// an act the watch made on a lane (an arming made again, a queuing, a toggle),
// and the report of a lane read dropped, which names the failed check and the
// commands that re-run it and arm it again. Each is said once while it stands,
// kept per run as a finished lane's is. A parked lane's ordinary reading
// (waiting, conflicting, syncing, unread) is the listing's and not an event:
// its note changes with the queue's every move.
func (t *terminal) sayParked(parked []DrainParked) {
	for _, p := range parked {
		if p.Note == "" || (!p.acted && p.State != ParkedDropped) || t.notedParked[p.RunID] == p.Note {
			continue
		}
		if t.notedParked == nil {
			t.notedParked = map[string]string{}
		}
		t.notedParked[p.RunID] = p.Note
		t.say("%s: %s", laneIssues(p.Issue, p.RunID, p.Bundle), p.Note)
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
		// A driver this drive may not take holds the run: past the stuck bound's minutes
		// the lane is set aside and the next opens (blocked).
		if t.blocked(runID, blockClaim, r.Reason) {
			return "", nil
		}
		// The claim's remedy under a drive is written for the drive's own terminal: what
		// keeps the drive from the run, and how a person there ends it (claimRun).
		t.waitFor(t.laneKey(runID)+", which another driver holds", time.Time{}, driveAct{Words: r.Remedy, Person: r.personAct})
		if t.idle(ctx, TerminalWaitFloor, r.Reason) {
			return TerminalWaited, nil
		}
		return "", nil
	}
	t.cleared(runID, blockClaim)
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
		if back, ended := t.stuckBack(ctx, runID, run); back {
			return ended, nil
		}
		t.onLane = run.Key
		if i := run.current(); i >= 0 {
			lane := run.Lanes[i]
			t.onLane = run.Key + " at " + string(lane.Stage)
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
				if perr := t.o.PreflightAllowed(lane.Worktree, diffBase(lane), lane.HeadSHA); errors.As(perr, &refused) {
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
			t.waitFor(driveOnPace, *res.NextEligibleAt, driveAct{})
			t.sleep(ctx, max(res.NextEligibleAt.Sub(t.now()), TerminalWaitFloor), res.Next)
		case res.Route != nil || res.PerformedStage != "" || res.stepNote != "":
			t.idleSince = time.Time{}
			t.moved()
			t.sayStep(runID, res)
			t.moving(t.onLane)
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

// stuckBack reports whether the lane in progress of run is stuck by the drain's
// stuck bounds (stuck.go), so the drive gives it back to the next move, which hands
// it back and takes up the next issue, rather than wait on it here; and why the run
// ends, "" to go on. A drive holding the lane was the one place the drain's stuck
// check did not reach (iss-2611071242184002). Bounds that cannot be read give it
// back too: the move reads them again and refuses in its own frame. The same run
// given back again with no other lane stepped between, the move having not handed
// it back, first waits at the floor, as an idle drive does, so the drive and the
// move never turn between them with no wait.
func (t *terminal) stuckBack(ctx context.Context, runID string, run State) (bool, string) {
	why := ""
	b, err := resolveStuckBounds(t.o.Options.roots(t.repoRoot))
	switch {
	case err != nil:
		why = refusalReason(err)
	default:
		why = stuckReason(run, b)
	}
	if why == "" {
		t.stuckGiven = ""
		return false, ""
	}
	if t.stuckGiven == runID {
		t.waitFor(run.Key+", stuck and not yet handed back", time.Time{}, driveAct{})
		if t.idle(ctx, TerminalWaitFloor, runID+" is stuck and the last move did not hand it back: "+why) {
			return true, TerminalWaited
		}
		return true, ""
	}
	t.stuckGiven = runID
	t.say("%s is stuck, so the next move hands it back: %s", runID, why)
	return true, ""
}

// moved ends the counts a wait keeps: the outage's and the limit's. It is the moment
// a limit's evidence says the run resumed.
func (t *terminal) moved() {
	// A stage done ends the lane's wait, and a lane tried again that moves gets the whole
	// bound if it is blocked again (blocked).
	if t.laneInHand != "" {
		delete(t.blocks, t.laneInHand)
		delete(t.retried, t.laneInHand)
	}
	t.outage.succeed()
	t.limit.succeed()
	t.stillSince = time.Time{}
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
			// The reset the provider names is the wait's end, whether the drive sleeps to it
			// or asks before it.
			t.waitFor(driveOnLimit, at, driveAct{})
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
			t.waitFor(driveOnLimit, time.Time{}, driveAct{})
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
				t.waitFor(driveOnProof, time.Time{}, driveAct{})
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
	// A run with members is read while it holds any issue, the lead's
	// reserved_at set or not (holdsAny).
	if !run.holdsAny() || letGoByDrive(run) {
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
	// A run with members reads every issue it holds, the lead's first; the
	// first one gone stops the lane, its reason naming that issue.
	lost := ""
	for _, id := range run.reservedIssues() {
		own, _, l := issueRenewalDue(id, snap, now)
		if l == "" && own != nil && own.State == claim.StateReleased {
			l = "the reservation on " + id + " was released while this lane waited"
		}
		if lost = l; lost != "" {
			break
		}
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
				t.rewriteState(terminalRenewTick)
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

// ticking runs work, a drain's move, writing the drive's state again at each renewal tick
// while it runs, as renewing does for an agent's call: a judge's call inside the move can
// outlast the value's next-write time. It renews nothing, which the move does for itself.
// The ticks have stopped when it returns.
func (t *terminal) ticking(work func()) {
	stop, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		tick := time.NewTicker(terminalRenewTick)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				t.rewriteState(terminalRenewTick)
			case <-stop:
				return
			}
		}
	}()
	work()
	close(stop)
	<-done
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
		// The landing's issue record gone from the checkout names no lane, so the stuck
		// bounds never count it: the lane is set aside at their minutes (blocked). Every
		// other refusal that names no lane is the machine's, and is waited on as before.
		if r.recordAbsent && t.blocked(runID, blockRecord, r.Reason) {
			return false, "", nil
		}
		t.waitFor(t.onLane, time.Time{}, driveAct{})
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
		// It exited 0 on this head, and the landing was held after it or the
		// receipt is still not there, which running it again would not change:
		// the lane goes to the stuck check.
		why := t.ranCleanHeld
		if why == "" {
			why = "its preflight exits 0 and leaves no receipt naming the lane's head"
		}
		return t.laneRefused(ctx, runID, lane.ID, why, "")
	}
	if !t.wallHeld() {
		return false, TerminalWall, nil
	}
	// The preflight runs in the drive's own checkout of the lane's stored head,
	// never in the lane's worktree (preflightcheckout.go), and a receipt is the
	// drive's own only for a head it saw pass there.
	held, passed, perr := t.ownPreflight(ctx, runID, lane)
	switch {
	case ctx.Err() != nil || t.stopped() || t.stopHolds():
		return false, "", nil
	case held != "":
		if passed {
			t.ranClean, t.ranCleanHeld = head, held
		}
		return t.laneRefused(ctx, runID, lane.ID, held, "")
	case perr == nil:
		t.ranClean, t.ranCleanHeld = head, ""
		t.minted[lane.HeadSHA] = true
		return true, "", nil
	default:
		return t.laneRefused(ctx, runID, lane.ID, "its preflight failed: "+perr.Error(), "")
	}
}

// preflight runs a lane's preflight on a context a stop also ends, this terminal's or one
// set from outside it, so one queued at the load gate or running does not outlast it.
func (t *terminal) preflight(ctx context.Context, checkout, head string) error {
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
	t.showWorking(t.onLane)
	t.renewing(func() { err = t.o.Preflight(pctx, checkout, head) })
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
	if err := saveRun(t.repoRoot, runID, t.o.Options, func(_ *os.Root, st *State) (bool, error) {
		return noteRefusal(st, why, now), nil
	}); err != nil {
		return false, "", err
	}
	t.waitFor(t.laneKey(runID)+" at land", time.Time{}, driveAct{})
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
	// Nothing reserves while a stop's mark stands. A stop that holds ends the drive, as
	// it ends every wait. Otherwise the run is taken over where this drive may take it
	// (mayTake), a person's lifted stop as a signal's (itd-2611141610197753 decision 1):
	// a mark not yet settled takes the check first (settleMark), and the take-over
	// reserves the run again and clears the mark in that order. A run the drive may not
	// take is named and passed over, and the drain goes on to other work.
	if c, err := runControl(t.repoRoot, runID); err == nil && c.Stop != nil {
		if t.stopHolds() {
			_, checked := stopCheck(t.repoRoot, runID, t.o.Stages, t.o.Options)
			if t.haltedBy(checked) {
				return true, TerminalHalted
			}
			return false, ""
		}
		if why := t.mayTake(run, c); why != "" {
			return t.passOverHalted(run, why)
		}
		why, gone := t.settleMark(runID, "this drive takes the run over once it has settled")
		switch {
		case why != "":
			t.waitFor(run.Key+"'s stop, being settled", time.Time{}, driveAct{})
			if t.idle(ctx, TerminalWaitFloor, why) {
				return true, TerminalWaited
			}
			return true, ""
		case !gone:
			return t.takeOver(ctx, run)
		}
		if run, err = ReadState(t.repoRoot, runID); err != nil {
			return false, ""
		}
	}
	// A hold a drive left to lapse, a lane it set aside or the kept hold of a stop that
	// could not be applied, is worked again from here (resumeKept); so is a lane set aside
	// and now tried again whose tip is no longer the one left to lapse, a peer's push
	// among them, so it is reserved again or dropped before it is stepped.
	if _, again := t.retried[run.RunID]; again || t.keptToLapse(run) {
		return t.resumeKept(ctx, run)
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
	err := saveRun(t.repoRoot, runID, t.o.Options, func(_ *os.Root, st *State) (bool, error) {
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

// noteRecordReleasedFor records, as a drive or a stop lets the issue id of a run go (the
// lead or, in a run with members, a member, its lines naming id), the blob of its record
// on origin's default branch as last fetched, and, for a release that did not reach
// origin, the tip it left. A record it cannot read leaves no note, and with no note
// reserving again compares nothing.
func noteRecordReleasedFor(repoRoot string, st *State, id string, failed bool, now time.Time) {
	if blob, _, _ := claim.RecordOnDefault(repoRoot, id); blob != "" {
		st.Record = append(st.Record, Entry{At: now, Stage: StageReleasedOver, Note: blob, Issue: st.issueOn(id)})
	}
	if !failed {
		return
	}
	snap, err := claim.View(repoRoot, now)
	if err != nil {
		return
	}
	if r, ok := snap.Lookup(id); ok && r.Copy == claim.CopyThis && (r.State == claim.StateLive || r.State == claim.StateExpired) {
		st.Record = append(st.Record, Entry{At: now, Stage: StageDriveReleaseFailed, Note: r.Commit, Issue: st.issueOn(id)})
	}
}

// failedReleaseTip is the tip the run's latest release left when it did not reach origin,
// "" when that release did, a later one reached origin, or the run has reserved since.
func failedReleaseTip(run State) string {
	return failedReleaseTipFor(run, run.Key)
}

// failedReleaseTipFor is failedReleaseTip for the issue id: in a run with members, read
// from the lines for id alone (lineIssue).
func failedReleaseTipFor(run State, id string) string {
	for i := len(run.Record) - 1; i >= 0; i-- {
		if run.hasMembers() && !recordid.SameID(run.lineIssue(run.Record[i]), id) {
			continue
		}
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
	return releasedOverBlobFor(run, run.Key)
}

// releasedOverBlobFor is releasedOverBlob for the issue id: in a run with members, read
// from the lines for id alone (lineIssue).
func releasedOverBlobFor(run State, id string) string {
	for i := len(run.Record) - 1; i >= 0; i-- {
		if run.hasMembers() && !recordid.SameID(run.lineIssue(run.Record[i]), id) {
			continue
		}
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
// In a run with members, issue names the issue moved or unread is about, the first whose
// record moved or, failing that, the first that could not be compared, and byIssue holds
// what each issue's comparison came to.
type recordSince struct {
	moved, unread string
	issue         string
	byIssue       map[string]recordSince
}

// of is what the comparison of id's record came to, in a run with members.
func (r recordSince) of(id string) (moved, unread string) {
	x := r.byIssue[id]
	return x.moved, x.unread
}

// recordMovedOn compares the issue's record on origin's default branch, freshly fetched, with
// what the run noted as it let the issue go. The note proves the record stood then, so a
// branch that can be read and holds no file for it is a change. A finished record is the
// reservation's to refuse, which says so.
//
// A run with members compares every issue of Issues() the same way, origin fetched once,
// before anything is pushed.
func recordMovedOn(repoRoot string, run State, now time.Time) recordSince {
	if !run.hasMembers() {
		noted := releasedOverBlob(run)
		if noted == "" {
			return recordSince{}
		}
		if _, err := recordFetch(repoRoot, claimOptions(now)); err != nil {
			return recordSince{unread: notComparedWhy(run.Key, claim.Remote+" could not be fetched")}
		}
		return recordComparedOn(repoRoot, run.Key, noted)
	}
	out := recordSince{byIssue: map[string]recordSince{}}
	fetched, unfetched := false, false
	for _, id := range run.Issues() {
		noted := releasedOverBlobFor(run, id)
		if noted == "" {
			continue
		}
		if !fetched && !unfetched {
			if _, err := recordFetch(repoRoot, claimOptions(now)); err != nil {
				unfetched = true
			} else {
				fetched = true
			}
		}
		r := recordSince{unread: notComparedWhy(id, claim.Remote+" could not be fetched")}
		if fetched {
			r = recordComparedOn(repoRoot, id, noted)
		}
		out.byIssue[id] = r
	}
	for _, id := range run.Issues() {
		if r := out.byIssue[id]; r.moved != "" && out.moved == "" {
			out.moved, out.issue = r.moved, id
		}
	}
	for _, id := range run.Issues() {
		if r := out.byIssue[id]; r.unread != "" && out.moved == "" && out.unread == "" {
			out.unread, out.issue = r.unread, id
		}
	}
	return out
}

// notComparedWhy is the unread clause for the record of id that could not be compared.
func notComparedWhy(id, why string) string {
	return fmt.Sprintf("%s's record could not be compared with what stood when the issue was let go: %s", id, why)
}

// recordComparedOn compares id's record on origin's default branch, just fetched, with the
// blob noted when the issue was let go.
func recordComparedOn(repoRoot, id, noted string) recordSince {
	blob, files, readable := claim.RecordOnDefault(repoRoot, id)
	switch {
	case !readable:
		return recordSince{unread: notComparedWhy(id, claim.Remote+"'s default branch cannot be read")}
	case claim.Finished(repoRoot, id):
		return recordSince{}
	case files == 0:
		return recordSince{moved: fmt.Sprintf("%s's record is gone from %s's default branch, where it stood when the issue was let go", id, claim.Remote)}
	case files > 1:
		return recordSince{unread: notComparedWhy(id, claim.Remote+"'s default branch holds it in more than one file")}
	case blob == noted:
		return recordSince{}
	}
	return recordSince{moved: fmt.Sprintf("%s's record on %s's default branch changed while the issue was let go", id, claim.Remote)}
}

// dropMovedRun ends the lane of a run whose record changed while it was let go, and returns
// the reason it goes back for. Only a release of the run's own that did not reach origin is
// tried again, and only over the tip it left, so a reservation anyone made since stands,
// and the reason says which.
//
// In a run with members each issue's own let-go release that did not reach origin is
// tried again, read by the issue on its lines, and only that issue's: one whose release
// reached origin is left as it stands.
func dropMovedRun(repoRoot string, run State, why string, now time.Time, o Options) (string, error) {
	issues := []string{run.Key}
	if run.hasMembers() {
		issues = run.Issues()
	}
	for _, id := range issues {
		var err error
		if why, err = retryLetGoRelease(repoRoot, run, id, why, now); err != nil {
			return why, err
		}
	}
	return why, dropRun(repoRoot, run.RunID, why, recordMovedOnWith, now, o)
}

// retryLetGoRelease is dropMovedRun's retry of id's own let-go release, its reason why
// with what came of it.
func retryLetGoRelease(repoRoot string, run State, id, why string, now time.Time) (string, error) {
	if tip := failedReleaseTipFor(run, id); tip != "" {
		clause, failed := releaseReservationAt(repoRoot, id, tip, "", now)
		why += "; " + clause
		if failed {
			tail, _, _ := lapseTail(repoRoot, run.forIssue(id), now)
			why += tail
		} else if err := mutateReservation(repoRoot, run.RunID, func(st *State) (bool, error) {
			st.Record = append(st.Record, Entry{At: now, Stage: StageReservationReleased, Note: clause, Issue: st.issueOn(id)})
			st.UpdatedAt = now
			return true, nil
		}); err != nil {
			return why, err
		}
	} else if snap, err := claim.View(repoRoot, now); err == nil {
		if r, ok := snap.Lookup(id); ok && r.Copy == claim.CopyThis && r.State == claim.StateLive {
			if run.hasMembers() {
				why += "; the reservation made on " + id + " from this copy since it was let go stands"
			} else {
				why += "; the reservation made on it from this copy since it was let go stands"
			}
		}
	}
	return why, nil
}

// passOverHalted leaves a marked run this drive may not take as it stands, names it with
// why, and has the drain go on to other work, as takeUp reports. The lane stays the
// drain's, in progress in its state: what leaves it here passes by itself or by a person's
// act (a driver that ends, a table read again, a stop read clearly), so it is passed over
// in the drive's memory alone (DrainOptions.PassedLanes), and the next start judges it
// again as it comes to hand.
func (t *terminal) passOverHalted(run State, why string) (bool, string) {
	if !t.passedLane(run.RunID) {
		t.o.Drain.PassedLanes = append(t.o.Drain.PassedLanes, run.RunID)
	}
	t.notePassedOver(Excluded{ID: run.Key, Check: CheckNeedsPerson, Reason: why})
	return true, ""
}

// takeEarlier takes over, before the drive's first move, every issue run in this checkout
// that a stop's mark stands on and that this drive may take (itd-2611141610197753
// decisions 1 and 2): a signal's mark, settled or not, and a person's stop once it is
// lifted, on a run parked at its pull request or on one no drain lists. It also takes up a
// run no drain lists that a drive's end let go with no mark, at its bound or for a limit,
// or whose hold a drive's end kept and left to lapse, its stop not applied (letGoUnlisted),
// by the same judgement, reserving it again as it is stepped where its hold has lapsed.
//
// A run the drain lists short of its pull request is the drain's, and is left to come to
// hand, where takeUp takes it over by the same rules: one an earlier abcd wrote halted is
// put back in progress for that (readmitHalted), never stepped here outside its drain. A
// stop that holds over a run is named here whatever lists it, since the drive then ends
// before any lane comes to hand. A drain state that cannot be read takes nothing: which
// runs are the drain's cannot be told, and the first move refuses on the same read.
//
// Each run is judged before anything is changed (mayTake): one a stop still holds over,
// one whose live recorded driver is neither gone, this process nor the process this
// drive's shell runs under (startedUnder), one that names no driver with no sign its
// session ended, and one whose driver cannot be judged because the process table cannot be
// read are left as they stand and named, with who holds them and the act that frees them.
// The take-over's own claim (TakeRun) lets an idle driver that claimed before the halt
// give way without asking any of that, which is why the judgement is made here first. A
// run that builds an intent is left to the person: a drive never starts agents on a build
// it did not open.
//
// A run taken that is parked at its pull request is read and finished by the moves, as
// any parked run of the checkout is (finishMerged). One no drain lists is stepped here,
// before the first move, as a plain run (driveLane): it counts against no cap, a hand-back
// stays on it, and a signal or the drive's end halts or lets it go as one of the drive's
// own lanes (stepped). Between runs an interrupt, a kill and the person's bound end the
// walk. It returns why the drive ends, "" to go on.
func (t *terminal) takeEarlier(ctx context.Context) (string, error) {
	st, _, err := readDrain(t.repoRoot)
	if err != nil {
		t.say("the drain's state could not be read, so nothing an interrupt left is taken up at the start: %s", err.Error())
		return "", nil
	}
	runs, err := Runs(t.repoRoot)
	if err != nil {
		t.say("the runs in this checkout could not be read to take up what an interrupt left: %s", err.Error())
		return "", nil
	}
	listed := func(runID string) (string, bool) {
		for _, l := range st.Lanes {
			if l.RunID == runID {
				return l.Outcome, true
			}
		}
		return "", false
	}
	for _, run := range runs {
		switch {
		case ctx.Err() != nil:
			return TerminalKilled, nil
		case t.stopped():
			return TerminalStopped, nil
		case t.pastBound():
			return TerminalUntil, nil
		}
		if run.Complete() || driveStopped(run.RunID) || t.passedLane(run.RunID) {
			continue
		}
		c, err := runControl(t.repoRoot, run.RunID)
		if err != nil {
			t.notePassedOver(Excluded{ID: run.Key, Check: CheckNeedsPerson, Reason: waitsForItsTakeOver(t.repoRoot, run.RunID, true)})
			continue
		}
		if c.Stop == nil {
			// A run no drain lists that a drive's end let go, or kept and left to
			// lapse, carries no mark, and no drain hands it out again, so the start
			// takes it up here by the same rules (mayTake): driveLane reserves it
			// again before it steps where it holds nothing (takeUp, resumeKept).
			if !t.letGoUnlisted(run, listed) {
				continue
			}
			if why := t.mayTake(run, c); why != "" {
				t.notePassedOver(Excluded{ID: run.Key, Check: CheckNeedsPerson, Reason: why})
				continue
			}
			t.stepped = append(t.stepped, run.RunID)
			ended, err := t.driveLane(ctx, run.RunID)
			if err != nil || ended != "" {
				return ended, err
			}
			t.letGoTakenUp(run.RunID)
			continue
		}
		if why := t.stopOver(run.RunID, c); why != "" {
			t.notePassedOver(Excluded{ID: run.Key, Check: CheckNeedsPerson, Reason: why})
			continue
		}
		parked := parkedAtLand(run) >= 0
		outcome, ours := listed(run.RunID)
		if ours && !parked {
			if outcome == DrainLaneHalted {
				t.readmitHalted(run)
			}
			continue
		}
		if run.Issue() == "" {
			id := termsafe.Sanitize(run.RunID)
			t.notePassedOver(Excluded{ID: run.Key, Check: CheckNeedsPerson,
				Reason: fmt.Sprintf("run %s builds an intent, which a drive does not take up: `abcd implement resume --take %s` takes it over", id, id)})
			continue
		}
		if why := t.mayTake(run, c); why != "" {
			t.notePassedOver(Excluded{ID: run.Key, Check: CheckNeedsPerson, Reason: why})
			continue
		}
		why, gone := t.settleMark(run.RunID, "a drive's next start takes it over once it has settled")
		switch {
		case why != "":
			t.notePassedOver(Excluded{ID: run.Key, Check: CheckNeedsPerson, Reason: why})
			continue
		case gone:
			t.say("the stop marking %s, run %s, lifted before the run halted, so it goes on", run.Key, run.RunID)
		default:
			res, err := TakeRun(t.repoRoot, run.RunID, t.takeUpOptions())
			if err != nil {
				if t.haltedBy(err) {
					return TerminalHalted, nil
				}
				t.notePassedOver(Excluded{ID: run.Key, Check: CheckNeedsPerson,
					Reason: termsafe.Sanitize(fsutil.RedactHome(fmt.Sprintf("run %s was not taken over at the drive's start: %s", run.RunID, refusalReason(err))))})
				continue
			}
			t.say("took over %s, run %s: its reservation is held again", run.Key, run.RunID)
			for _, line := range res.Left {
				t.say("%s", line)
			}
		}
		if parked || ours {
			continue
		}
		t.stepped = append(t.stepped, run.RunID)
		ended, err := t.driveLane(ctx, run.RunID)
		if err != nil || ended != "" {
			return ended, err
		}
		// A run taken over from its mark comes back still in progress as one taken up does,
		// and no move steps it either (iss-2611270555396370).
		t.letGoTakenUp(run.RunID)
	}
	return "", nil
}

// letGoUnlisted reports whether run is an issue's run in progress, short of its pull
// request, that a drive's end or a limit let go (letGoByDrive), or whose hold a drive left
// to lapse (keptToLapse: the kept hold of a stop that could not be applied), and that no
// lane of the drain lists: the drain's own lanes are taken up as they come to hand
// (takeUp), a parked run keeps its holds, an intent's build is the person's, and a run
// whose lane was handed back or dropped since (a take-up that backed off) has nothing to
// take up. A run reserved with no such sign, one a kill left, is not one a start takes.
func (t *terminal) letGoUnlisted(run State, listed func(string) (string, bool)) bool {
	if run.Issue() == "" || parkedAtLand(run) >= 0 || !(letGoByDrive(run) || t.keptToLapse(run)) {
		return false
	}
	if outcome, _, _ := laneOutcome(run); outcome != DrainLaneInProgress {
		return false
	}
	_, ours := listed(run.RunID)
	return !ours
}

// readmitHalted puts the drain's lane of run, which an earlier abcd wrote halted when it
// passed over a person's stop, back in progress, so the drain hands it out again and
// takeUp takes it over by the drive's rules, within the drain's own bounds. A write that
// fails is said, and the lane stays as it was.
func (t *terminal) readmitHalted(run State) {
	err := fsutil.WithFileLock(filepath.Join(t.repoRoot, filepath.FromSlash(drainLockRel)), lockTimeout, func() error {
		st, _, err := readDrain(t.repoRoot)
		if err != nil {
			return err
		}
		for i := range st.Lanes {
			if st.Lanes[i].RunID == run.RunID && st.Lanes[i].Outcome == DrainLaneHalted {
				st.Lanes[i].Outcome = DrainLaneInProgress
				st.UpdatedAt = t.now()
				return writeDrain(t.repoRoot, st)
			}
		}
		return nil
	})
	if err != nil {
		t.say("%s, run %s, which a stop halted, could not be put back among the drain's lanes: %s", run.Key, run.RunID, err.Error())
	}
}

// heldInProgress are the runs whose lanes this drive holds in progress, to halt on a signal
// or let go at its end: the drain's lanes in progress and the runs it steps that no drain
// lists, each once, leaving out what the drive passed over.
func (t *terminal) heldInProgress(st DrainState) []string {
	var ids []string
	for _, l := range st.Lanes {
		if l.Outcome == DrainLaneInProgress && !t.passedLane(l.RunID) {
			ids = append(ids, l.RunID)
		}
	}
	for _, id := range t.stepped {
		if slices.Contains(ids, id) || t.passedLane(id) {
			continue
		}
		if run, err := ReadState(t.repoRoot, id); err == nil {
			if outcome, _, _ := laneOutcome(run); outcome == DrainLaneInProgress {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// stopOver says why a stop's read leaves run c marks as it stands, "" where the read saw
// clearly and found no stop: a stop that holds over the run, which the person lifts, or a
// read that cannot be trusted, which is neither a stop nor its lift.
func (t *terminal) stopOver(runID string, c control) string {
	id := termsafe.Sanitize(runID)
	r := t.o.Options.stopRead(c.Shared)
	switch {
	case r.Stop != nil:
		return fmt.Sprintf("a stop set by %s at %s holds over run %s, which is left as it stands; once the person lifts the stop, a drive's start takes it up",
			termsafe.Sanitize(r.Stop.SetBy), r.Stop.SetAt.UTC().Format(time.RFC3339), id)
	case r.Held != "" || len(r.Refused) > 0:
		why := r.Held
		if why == "" {
			why = strings.Join(r.Refused, "; ")
		}
		return termsafe.Sanitize(fsutil.RedactHome(fmt.Sprintf("whether a stop holds over run %s cannot be told, so it is left as it stands: %s; once the read is clear, a drive's next start takes it up", id, why)))
	}
	return ""
}

// mayTake judges the driver run's control record c names before this drive takes the run
// over (itd-2611141610197753's third criterion), "" where it may: the driver is gone, it is
// this process, or it is the process this drive's shell runs under with nothing but shells
// beneath it (startedUnder). A run that names no driver is taken only on a sign its session
// ended: its mark has settled, or the reservation of its issue has lapsed. A run a drive's
// end let go with no mark is judged the same way, so one that names no driver is taken
// only where its release did not reach origin and that tip has lapsed: a release that
// reached origin reads as released, never as lapsed, so no start takes such a run, and
// the step that claims it (`abcd implement step --run`) is the way on. Otherwise it
// says who holds the run and the act that frees it: a live driver in another terminal or a
// host session keeps it, and a process table that cannot be read leaves it, since whether
// its driver lives cannot be told and a take-over resets the run's worktrees.
func (t *terminal) mayTake(run State, c control) string {
	if why := t.stopOver(run.RunID, c); why != "" {
		return why
	}
	id := termsafe.Sanitize(run.RunID)
	if c.Driver == nil {
		if (c.Stop != nil && c.Stop.HaltedAt != nil) || t.lapsed(run) {
			return ""
		}
		if c.Stop == nil {
			return fmt.Sprintf("run %s names no driver and carries no mark, so whether the session that let it go has ended cannot be told, and it is left as it stands; `abcd implement step --run %s` claims it and steps it on", id, id)
		}
		return fmt.Sprintf("run %s names no driver and its mark has not settled, so whether the session that drove it has ended cannot be told, and it is left as it stands; once its reservation lapses, a drive's next start takes it up, or the person takes it over with `abcd implement resume --take %s`", id, id)
	}
	d := *c.Driver
	tab, err := readProcs()
	if err != nil {
		return fmt.Sprintf("the process table cannot be read, so whether %s still drives run %s cannot be told, and the run is left as it stands; a drive's next start looks again",
			driverName(d), id)
	}
	if !tab.Alive(d.host()) {
		return ""
	}
	if h, _, err := selfDriver(tab); err == nil && h.PID == d.PID {
		return ""
	}
	took, kept := t.o.Options.startedUnder(tab, d)
	if took {
		return ""
	}
	why := fmt.Sprintf("run %s is held by %s, which has been alive since it took the run at %s, so this drive leaves it as it stands",
		id, driverName(d), d.At.UTC().Format(time.RFC3339))
	if kept.why != "" {
		why += "; " + kept.why
	}
	// The claim's act is worded for a drive that looks again; this one is passed over until
	// the next start, which is what looks again.
	act := "once that driver is gone, a drive's next start takes the run up"
	if kept.remedy != "" {
		act = strings.NewReplacer("this drive's next look", "a drive's next start", "this drive reads them again at its next look", "a drive's next start reads them again").Replace(kept.remedy)
	}
	return termsafe.Sanitize(fsutil.RedactHome(why + "; " + act))
}

// lapsed reports whether the reservation of run's issue has lapsed on its lease, as origin
// reads it now: a sign that whoever held it has ended. The reservations are fetched first,
// as peerTook fetches them, since this checkout's last fetch can miss a renewal made since;
// a fetch that fails, or a hold that cannot be read, is no such sign.
func (t *terminal) lapsed(run State) bool {
	issue := run.Issue()
	if issue == "" {
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
	r, ok := snap.Lookup(issue)
	return ok && r.State == claim.StateExpired
}

// settleMark has a mark on the run that has not settled take the check every move makes,
// so the run settles where it may: why it could not, "" where it did, and gone where the
// mark went with the check, a person's stop lifted before the run halted. The check saves
// the run whole, which a file stating another version allows its driver alone, so the
// drive claims the run first, over the driver mayTake has judged it may replace. A mark
// that stays unsettled is said as the check said it, with then, what takes it up after.
func (t *terminal) settleMark(runID, then string) (why string, gone bool) {
	id := termsafe.Sanitize(runID)
	c, err := runControl(t.repoRoot, runID)
	switch {
	case err != nil:
		return waitsForItsTakeOver(t.repoRoot, runID, true), false
	case c.Stop == nil:
		return "", true
	case c.Stop.HaltedAt != nil:
		return "", false
	}
	if err := claimDriver(t.repoRoot, runID, t.takeUpOptions(), false); err != nil {
		return termsafe.Sanitize(fsutil.RedactHome(fmt.Sprintf("run %s could not be claimed to settle its mark: %s", id, refusalReason(err)))), false
	}
	_, checked := stopCheck(t.repoRoot, runID, t.o.Stages, t.o.Options)
	c, err = runControl(t.repoRoot, runID)
	switch {
	case err != nil:
		return waitsForItsTakeOver(t.repoRoot, runID, true), false
	case c.Stop == nil:
		return "", true
	case c.Stop.HaltedAt != nil:
		return "", false
	}
	said := fmt.Sprintf("the mark on run %s has not settled", id)
	if checked != nil {
		said += ": " + refusalReason(checked)
	}
	return termsafe.Sanitize(fsutil.RedactHome(said + "; " + then)), false
}

// passLane passes over the lane of run, whose file states another version this drive may
// not save (r, the judge's refusal): every move after leaves it (DrainOptions.PassedLanes),
// and so do the drive's waits, its halts and its let-go at the end; the line saying what
// was left and why, the judge's reason and remedy, is said once.
func (t *terminal) passLane(run State, r *Refusal) {
	if t.passedLane(run.RunID) {
		return
	}
	p := PassedLane{Issue: termsafe.Sanitize(run.Key), Run: run.RunID,
		Reason: termsafe.Sanitize(fsutil.RedactHome(r.Reason + "; " + r.Remedy))}
	t.passedLanes = append(t.passedLanes, p)
	t.o.Drain.PassedLanes = append(t.o.Drain.PassedLanes, run.RunID)
	t.say("passed over %s's lane, run %s, and going on with the drain's other work: %s", p.Issue, p.Run, p.Reason)
}

// passedLane reports whether the drive passed the lane of runID over.
func (t *terminal) passedLane(runID string) bool {
	return t.o.Drain.passedLane(runID)
}

// blocked counts how long the lane of runID has waited at site on something this drive
// cannot end (why), and at the stuck bound's minutes sets it aside (setLaneAside),
// reporting true: the caller then gives the lane back to the next move, which goes on to
// other work, rather than wait (itd-2611141610197753 decisions 3 and 8). The wait is
// counted from the first block at site with no stage done since (moved) and the block not
// cleared since (cleared); a block at another site starts it again. A lane tried again (retrySetAside) that is blocked at the
// same site, with no stage done since, is set aside again at once, so a lane that stays
// blocked costs one look an hour and never spins the drive (decision 5).
func (t *terminal) blocked(runID, site, why string) bool {
	now := t.now()
	why = termsafe.Sanitize(fsutil.RedactHome(why))
	if s, ok := t.retried[runID]; ok {
		delete(t.retried, runID)
		if s.site == site {
			s.Reason = why
			t.setLaneAside(s, true)
			return true
		}
	}
	if t.blocks == nil {
		t.blocks = map[string]laneBlock{}
	}
	b, ok := t.blocks[runID]
	if !ok || b.site != site {
		b = laneBlock{site: site, since: now}
		t.blocks[runID] = b
	}
	if now.Sub(b.since) < t.setAsideBound() {
		return false
	}
	delete(t.blocks, runID)
	key := runID
	if run, err := ReadState(t.repoRoot, runID); err == nil {
		key = run.Key
	}
	t.setLaneAside(SetAsideLane{Issue: termsafe.Sanitize(key), Run: runID, Reason: why, Since: b.since, At: now, site: site}, false)
	return true
}

// cleared ends the wait of runID's lane at site, the block there having cleared, so a later
// block, there or elsewhere, counts its own wait from its own start.
func (t *terminal) cleared(runID, site string) {
	if b, ok := t.blocks[runID]; ok && b.site == site {
		delete(t.blocks, runID)
	}
}

// setAsideBound is the stuck bound's minutes (pace.stuck_minutes through the pace's
// layers), how long a lane waits on what the drive cannot end before it is set aside. A
// bound that cannot be read is the bundled one: the move reads it again and refuses in
// its own frame.
func (t *terminal) setAsideBound() time.Duration {
	b, err := resolveStuckBounds(t.o.Options.roots(t.repoRoot))
	if err != nil {
		return BundledStuckMinutes * time.Minute
	}
	return time.Duration(b.minutes.Value) * time.Minute
}

// setLaneAside sets the lane s names aside: every move after leaves it, as a passed lane
// (DrainOptions.PassedLanes), so nothing is saved, claimed, pushed or renewed for it and
// no agent runs for it, and its reservation is left to lapse at its lease (leaveToLapse).
// The first set-aside of a wait is movement of the drive, which goes on to other work; a
// lane set aside again at once (again) is not.
func (t *terminal) setLaneAside(s SetAsideLane, again bool) {
	if !t.passedLane(s.Run) {
		t.o.Drain.PassedLanes = append(t.o.Drain.PassedLanes, s.Run)
	}
	// A lane a live driver holds keeps its renewals, which are that driver's: only a hold
	// no one is working is left to lapse, and a note an earlier set-aside left on it is
	// taken off (keepRenewing).
	if heldByLiveDriver(s.site) {
		if run, err := ReadState(t.repoRoot, s.Run); err == nil {
			t.keepRenewing(run)
		}
		s.LeftToLapse, s.Holds = false, nil
	} else {
		s.LeftToLapse, s.Holds = t.leaveToLapse(s.Run)
	}
	t.setAside = append(slices.DeleteFunc(t.setAside, func(o SetAsideLane) bool { return o.Run == s.Run }), s)
	t.syncSetAside()
	if again {
		t.say("set aside %s's lane, run %s, again: it is blocked where it was, with no stage done since: %s; the drive tries it again at %s",
			s.Issue, s.Run, s.Reason, stamp(t.nextRetry))
		return
	}
	t.idleSince = time.Time{}
	t.moving(driveOnNextMove)
	hold := "its reservation is left to lapse"
	switch left, kept := SplitHolds(s.Holds); {
	case len(s.Holds) > 0:
		hold = fmt.Sprintf("the reservation of %s is left to lapse and that of %s could not be, so a renewal of the run keeps each", strings.Join(left, ", "), strings.Join(kept, ", "))
	case !s.LeftToLapse:
		hold = "its reservation is left to the driver that holds the run, whose renewals keep it"
	}
	t.say("set aside %s's lane, run %s, after waiting %s, and going on with the drain's other work: %s; %s, and the drive tries the lane again once no other lane is in progress",
		s.Issue, s.Run, s.At.Sub(s.Since).Round(time.Second), s.Reason, hold)
}

// setAsideAmong reports whether a lane of lanes is one the drive set aside.
func (t *terminal) setAsideAmong(lanes []DrainLane) bool {
	return slices.ContainsFunc(lanes, func(l DrainLane) bool {
		return slices.ContainsFunc(t.setAside, func(s SetAsideLane) bool { return s.Run == l.RunID })
	})
}

// retryDue reports whether lanes are set aside and it is time to try them again: at once
// the first time no other lane is in progress, and then once an hour.
func (t *terminal) retryDue() bool {
	return len(t.setAside) > 0 && !t.now().Before(t.nextRetry)
}

// retrySetAside gives every lane set aside back to the moves for one look each
// (itd-2611141610197753 decision 5): it leaves the passed lanes, and the next move reaches
// it. A lane whose issue a peer holds now is dropped here, through the take-up's own
// refusal (tookUp), which pushes nothing, since no move would step it (T6a). A lane whose
// hold lapsed while it was set aside is reserved again only once its block clears, as its
// take-up finds the hold left to lapse (resumeKept), so a lane still blocked pushes
// nothing an hour. The next try is an hour off. It returns why the drive ends, "" to go
// on.
func (t *terminal) retrySetAside(ctx context.Context) string {
	if t.retried == nil {
		t.retried = map[string]SetAsideLane{}
	}
	names := make([]string, 0, len(t.setAside))
	for _, s := range t.setAside {
		t.o.Drain.PassedLanes = slices.DeleteFunc(t.o.Drain.PassedLanes, func(id string) bool { return id == s.Run })
		t.retried[s.Run] = s
		names = append(names, fmt.Sprintf("%s's lane, run %s", s.Issue, s.Run))
	}
	tried := t.setAside
	t.setAside = nil
	t.syncSetAside()
	t.nextRetry = t.now().Add(terminalSetAsideRetry)
	t.say("trying again the lanes set aside, one look each: %s", strings.Join(names, "; "))
	for _, s := range tried {
		run, err := ReadState(t.repoRoot, s.Run)
		if err != nil {
			continue
		}
		if outcome, _, _ := laneOutcome(run); outcome != DrainLaneInProgress || !t.peerHolds(run) {
			continue
		}
		err = reserveResumed(t.repoRoot, &StartResult{RunID: run.RunID}, t.takeUpOptions())
		if _, ended := t.tookUp(ctx, run, err); ended != "" {
			return ended
		}
	}
	return ""
}

// peerHolds reports whether another copy holds a live reservation on an issue of run.
func (t *terminal) peerHolds(run State) bool {
	snap, err := claim.View(t.repoRoot, t.now())
	if err != nil {
		return false
	}
	return slices.ContainsFunc(run.reservedIssues(), func(id string) bool {
		r, ok := snap.Lookup(id)
		return ok && r.Copy != claim.CopyThis && r.State == claim.StateLive
	})
}

// foldRetried sets aside again, quietly, every lane given back for its look (retrySetAside)
// that the drive has not reached since, as the drive ends: the lane in hand is the drive's
// work and is let go as any lane is, and a lane done, handed back or dropped is no longer
// in progress. A lane whose hold was left to lapse is left to lapse again, since the look
// that would take it up did not come.
func (t *terminal) foldRetried() {
	tried := make([]SetAsideLane, 0, len(t.retried))
	for runID, s := range t.retried {
		if runID != t.laneInHand {
			tried = append(tried, s)
		}
	}
	slices.SortFunc(tried, func(a, b SetAsideLane) int { return a.At.Compare(b.At) })
	for _, s := range tried {
		delete(t.retried, s.Run)
		run, err := ReadState(t.repoRoot, s.Run)
		if err != nil {
			continue
		}
		if outcome, _, _ := laneOutcome(run); outcome != DrainLaneInProgress {
			continue
		}
		if s.LeftToLapse || len(s.Holds) > 0 {
			s.LeftToLapse, s.Holds = t.leaveToLapse(s.Run)
		}
		if !t.passedLane(s.Run) {
			t.o.Drain.PassedLanes = append(t.o.Drain.PassedLanes, s.Run)
		}
		t.setAside = append(t.setAside, s)
	}
	t.syncSetAside()
}

// syncSetAside tells the moves which lanes passed over are set aside (DrainOptions.SetAside),
// so a move names them as set aside, not as lanes the drive may not save.
func (t *terminal) syncSetAside() {
	ids := make([]string, 0, len(t.setAside))
	for _, s := range t.setAside {
		ids = append(ids, s.Run)
	}
	t.o.Drain.SetAside = ids
}

// awaitSetAside answers a move whose only lanes left in progress are passed over, one or
// more of them set aside (itd-2611141610197753 decision 9): it tries them again when it is
// time (retryDue), and otherwise waits until then, saying so. Under the person's bound the
// wait goes on to the bound; with none it ends the drive once nothing has moved for as
// long as a drive waits (idle). A try that sets a lane aside again is no movement. It
// returns why the drive ends, "" to go on.
func (t *terminal) awaitSetAside(ctx context.Context) string {
	if t.retryDue() {
		return t.retrySetAside(ctx)
	}
	names := make([]string, 0, len(t.setAside))
	for _, s := range t.setAside {
		names = append(names, s.Issue)
	}
	now := t.now()
	d := max(t.nextRetry.Sub(now), TerminalWaitFloor)
	why := fmt.Sprintf("every lane left in progress is set aside (%s), and each is tried again at %s", strings.Join(names, ", "), stamp(t.nextRetry))
	// The hourly retry is the wait's named end (decision 9), so the line reads waiting
	// however long every lane stays set aside.
	t.waitFor(fmt.Sprintf("%d %s set aside", len(names), plural(len(names), "lane", "lanes")), t.nextRetry, driveAct{})
	if !t.bounded() {
		if t.idle(ctx, d, why) {
			return TerminalWaited
		}
		return ""
	}
	if t.idleSince.IsZero() {
		t.idleSince = now
	}
	t.sleep(ctx, d, why)
	return ""
}

// leaveToLapse leaves every hold of runID this copy has, live or lapsed, to lapse at its
// tip (claim.LeaveToLapse): the drive keeps the hold of a lane it is not working, so no
// peer takes the issue while that is unsure, and no renewal that reads no driver keeps it
// past its lease, an `abcd implement status` call, a load-gate waiter, a drive's waits and
// a drain's move among them (leftToLapse). A note that cannot be written is said. It
// reports whether every hold of the run was left to lapse, which is what keeps the run
// from those renewals; where only some were, it names each issue with whether its own
// was, since a renewal of the run then keeps every hold of it (dcn-2611271952390075).
func (t *terminal) leaveToLapse(runID string) (bool, []SetAsideHold) {
	run, err := ReadState(t.repoRoot, runID)
	if err != nil {
		return false, nil
	}
	ids := run.reservedIssues()
	if len(ids) == 0 {
		return false, nil
	}
	snap, err := claim.View(t.repoRoot, t.now())
	if err != nil {
		t.say("%s: its reservation could not be read to leave it to lapse, so a renewal in this copy may keep it: %s", termsafe.Sanitize(run.Key), err.Error())
		return false, nil
	}
	holds := make([]SetAsideHold, 0, len(ids))
	left := 0
	for _, id := range ids {
		h := SetAsideHold{Issue: termsafe.Sanitize(id)}
		if r, ok := snap.Lookup(id); ok && r.Copy == claim.CopyThis && (r.State == claim.StateLive || r.State == claim.StateExpired) {
			if err := claim.LeaveToLapse(t.repoRoot, id, r.Commit); err != nil {
				t.say("%s: its reservation could not be left to lapse, so a renewal in this copy may keep it: %s", termsafe.Sanitize(id), termsafe.Sanitize(fsutil.RedactHome(err.Error())))
			} else {
				h.LeftToLapse = true
				left++
			}
		}
		holds = append(holds, h)
	}
	switch {
	case left == 0:
		return false, nil
	case left < len(ids):
		return false, holds
	}
	if t.lapsing == nil {
		t.lapsing = map[string]bool{}
	}
	t.lapsing[runID] = true
	return true, nil
}

// keptToLapse reports whether a hold of run is one a drive left to lapse at its tip
// (leaveToLapse): a lane it set aside, or the kept hold of a stop that could not be
// applied. A view that cannot be read is no such sign.
func (t *terminal) keptToLapse(run State) bool {
	ids := run.reservedIssues()
	if len(ids) == 0 {
		return false
	}
	snap, err := claim.View(t.repoRoot, t.now())
	if err != nil {
		return false
	}
	return slices.ContainsFunc(ids, func(id string) bool {
		r, ok := snap.Lookup(id)
		return ok && r.Copy == claim.CopyThis && claim.LeftToLapse(t.repoRoot, id, r.Commit)
	})
}

// resumeKept takes up a lane whose hold a drive left to lapse, as takeUp reports: the
// note is taken off, so the hold is renewed again as the lane is worked; where any hold
// has lapsed meanwhile, or the view cannot tell, the run is reserved again first, as a
// let-go lane is (reserveResumed), before any step, so a lane is never stepped holding
// nothing, and a peer that holds the issue now drops the lane (tookUp;
// itd-2611141610197753 T6a).
func (t *terminal) resumeKept(ctx context.Context, run State) (bool, string) {
	ids := run.reservedIssues()
	t.keepRenewing(run)
	live := false
	if snap, err := claim.View(t.repoRoot, t.now()); err == nil {
		live = !slices.ContainsFunc(ids, func(id string) bool {
			r, ok := snap.Lookup(id)
			return !ok || r.Copy != claim.CopyThis || r.State != claim.StateLive
		})
	}
	if live {
		return false, ""
	}
	err := reserveResumed(t.repoRoot, &StartResult{RunID: run.RunID}, t.takeUpOptions())
	return t.tookUp(ctx, run, err)
}

// keepRenewing takes off every note that left a hold of run to lapse (claim.KeepRenewing),
// so a renewal in this copy keeps the hold again, and the run is no longer counted as left
// to lapse (lapsing) unless a note could not be taken off, which is said.
func (t *terminal) keepRenewing(run State) {
	kept := true
	for _, id := range run.reservedIssues() {
		if err := claim.KeepRenewing(t.repoRoot, id); err != nil {
			kept = false
			t.say("%s: what left its reservation to lapse could not be taken off: %s", termsafe.Sanitize(id), err.Error())
		}
	}
	if kept {
		delete(t.lapsing, run.RunID)
	}
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
		t.cleared(run.RunID, blockTakeOver)
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
		switch {
		// The refusal of a whole save is at the driver's stage, and the lane is passed
		// over as the take-up passes it (tookUp), not waited on as a driver is.
		case r.leftAtVersion:
			return t.tookUp(ctx, run, err)
		case r.Stage == StageResume:
			t.stoppedBy(fmt.Errorf("%s; %s", r.Reason, r.Remedy))
			return true, TerminalWall
		case r.Stage == StageDriver:
			if t.blocked(run.RunID, blockTakeOver, r.Reason) {
				return true, ""
			}
			t.waitFor(run.Key+", which another driver holds", time.Time{}, driveAct{Words: r.Remedy, Person: r.personAct})
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
		t.cleared(runID, blockTakeUp)
		t.say("took up %s, run %s: its reservation is held again", run.Key, runID)
		return false, ""
	}
	r, ok := AsRefusal(err)
	if !ok {
		r = refuse(StageReservation, "", "", err.Error(), "see the reason")
	}
	// The run's file states another version and this drive may not save it: no other
	// drive took the issue up, and nothing the drive saves of the lane would be kept, so
	// the drive passes the lane over, once, saying what was left and why, and goes on
	// with its other work (passLane).
	if r.leftAtVersion {
		t.passLane(run, r)
		return true, ""
	}
	if r.recordMoved {
		reason, derr := dropMovedRun(t.repoRoot, run, r.Reason, t.now(), t.o.Options)
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
		if t.blocked(runID, blockTakeUp, r.Reason) {
			return true, ""
		}
		t.waitFor(run.Key+"'s run, which another invocation holds", time.Time{}, driveAct{})
		if t.idle(ctx, TerminalWaitFloor, r.Reason) {
			return true, TerminalWaited
		}
		return true, ""
	}
	if r.final == "" && r.Contention {
		r.final = "another drive took the issue up after this lane stopped"
	}
	if r.final == "" {
		if t.blocked(runID, blockTakeUp, r.Reason) {
			return true, ""
		}
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
	_ = saveRun(t.repoRoot, runID, t.o.Options, func(_ *os.Root, st *State) (bool, error) {
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
	err := saveRun(t.repoRoot, runID, t.o.Options, func(root *os.Root, st *State) (bool, error) {
		if !st.holdsAny() || letGoByDrive(*st) {
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
		// while it was let go. A run with members lets every issue it holds go, each
		// plainly, each line naming its issue.
		for _, id := range st.reservedIssues() {
			c, f := releaseReservation(t.repoRoot, id, "", now)
			noteRecordReleasedFor(t.repoRoot, st, id, f, now)
			st.Record = append(st.Record, Entry{At: now, Stage: StageDriveReleased, Note: c, Issue: st.issueOn(id)})
			clause, failed = joinNext(clause, c), failed || f
		}
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
	t.sayDropped(run, r.Reason, dropRun(t.repoRoot, run.RunID, r.Reason, t.issueIsWith(run.Key), t.now(), t.o.Options))
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
func dropRun(repoRoot, runID, why, whose string, now time.Time, o Options) error {
	reason := termsafe.Sanitize(fsutil.RedactHome(why))
	return saveRun(repoRoot, runID, o, func(_ *os.Root, st *State) (bool, error) {
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
// progress, left on its branch, a judgement's hold no lane opened on, and
// every hold of a bundle the drain is forming, which it clears. Each goes
// plainly, publishing no hand-back. A lane parked at its pull request keeps
// its own, and so does a lane a killed move opened and never recorded,
// which the let-go records; the drain state is written before any of the
// drain's own holds is released.
//
// A release is a push, which a drive makes only under what it holds
// (heldBeforeGit), so it compares first, before it takes the drain's lock,
// whose wait other sessions keep short (lockTimeout): on a difference, a
// read that gives no answer, or a comparison that does not finish, no
// release is tried, and each reservation left is named with when its lease
// lapses and how the person releases it (leftHeld). Each push inside the
// lock asks the comparison again, as every networked git of a drive does.
func (t *terminal) letGo() {
	now := t.now()
	if err := heldBeforeGit(t.repoRoot, t.o.Options.remote); err != nil {
		t.leftHeld(err, now)
		return
	}
	err := fsutil.WithFileLock(filepath.Join(t.repoRoot, filepath.FromSlash(drainLockRel)), lockTimeout, func() error {
		// A drain that ended on an outage can have a lane still in progress, so
		// its lanes are read whether it is live or not.
		st, live, err := readDrain(t.repoRoot)
		if err != nil {
			return err
		}
		for _, runID := range t.heldInProgress(st) {
			run, err := ReadState(t.repoRoot, runID)
			if err != nil || !run.holdsAny() || letGoByDrive(run) {
				continue
			}
			// A stop that could not settle the lane holds its claims with its work.
			if c, err := runControl(t.repoRoot, runID); err == nil && c.Stop != nil {
				continue
			}
			// So does a run the signal's stop could not be applied to, which no mark
			// stands on: its reservations are left to lapse with their leases.
			if t.unstoppedRun(runID) {
				t.sayKept(run, drainLists(st, runID), t.keepUnstopped(runID), now)
				continue
			}
			t.releaseRun(run, now)
		}
		if !live || (st.Judging == nil && st.Forming == nil) {
			return nil
		}
		// A hold a run of this checkout was opened on is the run's, though a
		// move killed before it recorded the lane left it here as the
		// drain's (openedUnrecorded): the lane is recorded and its holds are
		// kept. The state is written before any release is pushed, so a
		// let-go that dies part way records no hold it was letting go.
		opened, carried, err := openedUnrecorded(t.repoRoot, st)
		if err != nil {
			return err
		}
		var free []formingEntry
		if j := st.Judging; j != nil && j.Hold != nil && !carried[j.Issue] {
			free = append(free, formingEntry{issue: j.Issue, hold: j.Hold})
		}
		// A bundle forming lets its lead and every member go, each plainly
		// and over the hold the drain took on it, as a judgement's hold is.
		for _, h := range formingHolds(st.Forming) {
			if !carried[h.issue] {
				free = append(free, h)
			}
		}
		st.Lanes = append(st.Lanes, opened...)
		st.Judging, st.Forming = nil, nil
		st.UpdatedAt = now
		if err := writeDrain(t.repoRoot, st); err != nil {
			return err
		}
		for _, l := range opened {
			t.say("%s", openedLine(l))
		}
		for _, h := range free {
			clause, _ := releaseReservationAt(t.repoRoot, h.issue, h.hold.Commit, "", now)
			t.say("%s: %s", h.issue, clause)
		}
		return nil
	})
	if err != nil {
		t.say("what this run holds was not released, and lapses with its lease "+unlessRenewedEither+": %s", err.Error())
	}
}

// releaseRun lets every issue run holds go, the lead's and each member's, each plainly, and
// records the let-go on the run, so a later start takes it up (letGoUnlisted). Lines go in
// the run's record alone: its file keeps the version it states.
func (t *terminal) releaseRun(run State, now time.Time) {
	type letGone struct {
		id, clause string
		failed     bool
	}
	var gone []letGone
	for _, id := range run.reservedIssues() {
		clause, failed := releaseReservation(t.repoRoot, id, "", now)
		t.say("%s: %s", id, clause)
		gone = append(gone, letGone{id: id, clause: clause, failed: failed})
	}
	_ = mutateReservation(t.repoRoot, run.RunID, func(st *State) (bool, error) {
		for _, g := range gone {
			noteRecordReleasedFor(t.repoRoot, st, g.id, g.failed, now)
			st.Record = append(st.Record, Entry{At: now, Stage: StageDriveReleased, Note: g.clause, Issue: st.issueOn(g.id)})
		}
		st.UpdatedAt = now
		return true, nil
	})
}

// letGoTakenUp lets go a run no drain lists that the start took up (takeEarlier) and that
// came back from driveLane still in progress: refused, unable to move, or given back for
// any other reason (iss-2611270555396370). No move of this drive reaches a run no drain
// lists, so nothing steps it again, and a hold kept for it would be renewed by the drive's
// waits beside the lane the drain's first move opens, which counts no lane in work: two
// reserved runs, one stepped. It is let go as the drive's end lets it go (releaseRun),
// which a later start takes up again, and the drain opens no lane on its issue while the
// run stands (the move passes over an issue this checkout has a run for). A run a live
// driver elsewhere holds is that driver's work, and a run set aside keeps its hold as any
// lane set aside does; neither is let go. A release is a push, so it waits on the drive's
// comparison (heldBeforeGit), and a comparison that stops it keeps the run for the drive's
// end, which lets it go or names it.
func (t *terminal) letGoTakenUp(runID string) {
	run, err := ReadState(t.repoRoot, runID)
	if err != nil || !run.holdsAny() || letGoByDrive(run) || t.passedLane(runID) {
		return
	}
	if outcome, _, _ := laneOutcome(run); outcome != DrainLaneInProgress || t.drivenElsewhere(runID) {
		return
	}
	if err := heldBeforeGit(t.repoRoot, t.o.Options.remote); err != nil {
		return
	}
	t.say("%s, run %s, which no drain lists, is still in progress and no move of this drive steps it, so its reservation is let go; a later start takes it up",
		termsafe.Sanitize(run.Key), termsafe.Sanitize(runID))
	t.releaseRun(run, t.now())
}

// leftHeld says, for a let-go the drive's comparison stopped (why), that
// nothing was released, and names each reservation letGo would have
// released, with when its lease lapses and how the person releases it once
// what the drive holds is as it should be. Nothing is pushed: the one write is
// the local note that leaves a kept hold to lapse (leaveToLapse).
func (t *terminal) leftHeld(why error, now time.Time) {
	reason := why.Error()
	if r, ok := AsRefusal(why); ok {
		reason = r.Reason
	}
	t.say("nothing this drive holds was released: releasing is a push, and the drive runs none while %s", termsafe.Sanitize(fsutil.RedactHome(reason)))
	st, live, err := readDrain(t.repoRoot)
	if err != nil {
		t.say("which reservations this drive holds could not be read, and each lapses with its lease "+unlessRenewedEither+": %s", termsafe.Sanitize(fsutil.RedactHome(err.Error())))
		return
	}
	type left struct {
		id, runID string
		// renewers is the clause of a lane's run still in progress (runRenewers), and
		// empty for a judgement's or a forming bundle's hold.
		renewers          string
		forming, unlisted bool
	}
	var ids []left
	snap, verr := claim.View(t.repoRoot, now)
	for _, runID := range t.heldInProgress(st) {
		run, err := ReadState(t.repoRoot, runID)
		if err != nil || !run.holdsAny() || letGoByDrive(run) {
			continue
		}
		if c, err := runControl(t.repoRoot, runID); err == nil && c.Stop != nil {
			continue
		}
		// A run the signal's stop could not be applied to keeps its holds on purpose, so
		// it is named as kept, not as a release to make by hand, and left to lapse.
		if t.unstoppedRun(runID) {
			t.sayKept(run, drainLists(st, runID), t.keepUnstopped(runID), now)
			continue
		}
		for _, id := range run.reservedIssues() {
			ids = append(ids, left{id: id, runID: runID, renewers: keptRenewers(t.repoRoot, run, snap, verr == nil), unlisted: !drainLists(st, runID)})
		}
	}
	if j := st.Judging; j != nil && live && j.Hold != nil {
		ids = append(ids, left{id: j.Issue})
	}
	if live {
		for _, h := range formingHolds(st.Forming) {
			ids = append(ids, left{id: h.issue, forming: true})
		}
	}
	for _, l := range ids {
		var r claim.Reservation
		read := verr == nil
		if read {
			r, _ = snap.Lookup(l.id)
		}
		// A run no drain lists is not taken up by a start while it is reserved, so its way
		// on is the step that claims it.
		again := "start the drive again, which takes the run up"
		if l.unlisted {
			again = "step the run on with `abcd implement step --run " + termsafe.Sanitize(l.runID) + "`"
		}
		t.say("%s: its reservation was not released and %s; once what the drive holds is as it should be, release it with `abcd reserve %s --release`, or %s",
			termsafe.Sanitize(l.id), lapse(r, read, l.renewers, l.forming), termsafe.Sanitize(l.id), again)
	}
}

// keepUnstopped keeps the holds of runID, a run the signal's stop could not be applied to,
// and reports whether it left them to lapse: a run a live driver other than this drive
// holds is being worked, so its holds keep that driver's renewals and no note is written;
// any other run is no one's work once the drive ends, and its holds are left to lapse
// (leaveToLapse). A process table that cannot be read is read as a live driver, since a
// hold lost under a working session costs more than one kept a lease too long.
func (t *terminal) keepUnstopped(runID string) bool {
	if t.drivenElsewhere(runID) {
		return false
	}
	all, _ := t.leaveToLapse(runID)
	return all
}

// drivenElsewhere reports whether the control of runID names a driver other than this
// drive that is alive, or whose life cannot be read.
func (t *terminal) drivenElsewhere(runID string) bool {
	c, err := runControl(t.repoRoot, runID)
	if err != nil || c.Driver == nil {
		return false
	}
	tab, err := readProcs()
	if err != nil {
		return true
	}
	if here, _, herr := t.o.Options.driverHere(tab); herr == nil && here.PID == c.Driver.PID {
		return false
	}
	return tab.Alive(c.Driver.host())
}

// unstoppedRun reports whether a signal's stop could not be applied to runID (haltOnSignal).
func (t *terminal) unstoppedRun(runID string) bool {
	return slices.Contains(t.unstopped, runID)
}

// sayKept names each issue run holds, the lead's and every member's, as kept for the stop
// that could not be applied to it, with when its lease lapses, what frees the run, and the
// release a person makes by hand. The hold is left to lapse (leaveToLapse, lapsing), so no
// renewal that reads no driver keeps it (UnlessKeptTakenUp); one a live driver elsewhere
// holds keeps that driver's renewals (keepUnstopped, runRenewers). A run the drain lists (listed) is
// read as reserved by the next start until then; one no drain lists is taken up by a
// drive's next start once its driver is gone (letGoUnlisted), and the step that claims it
// is named too. It reads only.
func (t *terminal) sayKept(run State, listed, lapsing bool, now time.Time) {
	snap, verr := claim.View(t.repoRoot, now)
	runID := termsafe.Sanitize(run.RunID)
	for _, id := range run.reservedIssues() {
		var r claim.Reservation
		read := verr == nil
		if read {
			r, _ = snap.Lookup(id)
		}
		until := "its lease lapses"
		if read && r.Copy == claim.CopyThis && r.State == claim.StateLive {
			until = stamp(r.EffectiveExpiry)
		}
		on := "the next start reads the run as reserved until " + until
		if !listed {
			on = "no drain lists the run, so a drive's next start takes it up once its driver is gone, or `abcd implement step --run " + runID + "` claims it and steps it on"
		}
		key := termsafe.Sanitize(id)
		unless := UnlessKeptTakenUp
		if !lapsing {
			unless = keptRenewers(t.repoRoot, run, snap, verr == nil)
		}
		t.say("%s: its reservation is kept, since the stop could not be applied to run %s, and %s; %s; `abcd reserve %s --release` releases it by hand",
			key, runID, lapsesUnless(r, read, unless), on, key)
	}
}

// keptRenewers is the clause a lapse says of run, a lane's run still in progress, where a
// view of the reservations was read (read): a hold a drive left to lapse, an earlier
// drive's note included, that no take-up has taken off (takeUp), is renewed by no `abcd
// drain` move, `abcd implement status` call or load-gate waiter (leftToLapse,
// claim.RenewDue), so it takes UnlessKeptTakenUp; any other takes the renewers of where it
// stands (runRenewers).
func keptRenewers(root string, run State, view claim.Snapshot, read bool) string {
	if read && leftToLapse(root, run, view) {
		return UnlessKeptTakenUp
	}
	return runRenewers(run)
}

// drainLists reports whether a lane of st names runID.
func drainLists(st DrainState, runID string) bool {
	return slices.ContainsFunc(st.Lanes, func(l DrainLane) bool { return l.RunID == runID })
}

// formingHolds is each hold of the bundle f forms, the lead's first and then
// each member's in the order judged (DrainForming.entries); an issue recorded
// with no hold has none to release. Nil f forms none.
func formingHolds(f *DrainForming) []formingEntry {
	var out []formingEntry
	for _, e := range f.entries() {
		if e.hold != nil {
			out = append(out, e)
		}
	}
	return out
}

// unlessFormingRenewed is the clause a lapse says of a live reservation this
// copy's last push takes and a forming bundle holds: a drain's move renews
// it at half its lease (renewInProgress), as a load-gate waiter does.
const unlessFormingRenewed = "unless an `abcd drain` move in this copy, or a load-gate waiter in this copy, renews it"

// lapse is heldLapse, with a forming bundle's hold (forming) named with the
// renewers it has.
func lapse(r claim.Reservation, read bool, renewers string, forming bool) string {
	if forming {
		return lapsesUnless(r, read, unlessFormingRenewed)
	}
	return heldLapse(r, read, renewers)
}

// heldLapse says when a reservation leftHeld names lapses. Only a live
// reservation this copy's last push takes can be renewed here, so only its
// lapse takes a clause; read is false when the view could not be read. A
// lane's run still in progress takes renewers, the clause runRenewers reads
// from where the run stands: a step or `abcd drain` run in this copy and a
// load-gate waiter renew it, and an `abcd implement status` call does too
// only for a lane short of its pull request. A judgement's hold (renewers
// empty) is renewed by the waiter alone (claim.RenewDue).
func heldLapse(r claim.Reservation, read bool, renewers string) string {
	if renewers != "" {
		return lapsesUnless(r, read, renewers)
	}
	return lapsesUnless(r, read, unlessWaiterRenews)
}

// lapsesUnless says when a reservation leftHeld names lapses, with unless,
// the clause naming what renews it, for a live reservation this copy's last
// push takes, the only one that can be renewed here.
func lapsesUnless(r claim.Reservation, read bool, unless string) string {
	if !read || r.Copy != claim.CopyThis || r.State != claim.StateLive {
		return "lapses with its lease"
	}
	return "lapses at " + stamp(r.EffectiveExpiry) + " " + unless
}

// ended is the run's result as it ends: unless a kill or a stop from elsewhere
// ended it, what no lane is working on is released, so a peer may take it up; the
// lane left in progress is named, with the command that resumes it, and, after a
// kill or a signal whose stop could not be applied to its run (unstopped), when its
// reservation lapses. A stop from elsewhere names its own remedy.
func (t *terminal) ended(why string) (res TerminalResult) {
	defer t.leaveSharedRun()
	defer func() { t.showEnded(res) }()
	if why == TerminalWall && t.wallEnded != "" {
		why = t.wallEnded
	}
	// A lane given back for its look and not yet reached is still set aside: it is named so,
	// and its hold is kept, as it was, rather than let go as a lane this drive worked.
	t.foldRetried()
	switch why {
	case TerminalStopped:
		t.haltOnSignal()
	case TerminalHalted:
		t.settleHalted()
	}
	if why != TerminalKilled && why != TerminalHalted {
		t.letGo()
	}
	res = TerminalResult{Ended: why, Last: t.last, HeldBack: t.heldBack, PassedOver: t.passedOver, PassedLanes: t.passedLanes, SetAside: t.setAside}
	// A drive that ends, other than killed, halted on a check or stopped by
	// a person, sweeps on its own once, after it has let its lanes go
	// (autosweep.go); the line is the drive's end's to print (Swept). A
	// stop that comes while it sweeps ends it between removals.
	if why != TerminalKilled && why != TerminalHalted && why != TerminalStopped {
		res.Swept = autoSweep(t.ctx, t.o.Stop, t.repoRoot)
	}
	if why == TerminalWall || why == TerminalHalted {
		res.Why = t.why
	}
	// The lanes passed over are a later drive's to take up, so the command that starts one
	// is named, as it is for a lane left on its branch.
	if why == TerminalPassedLanes || len(t.setAside) > 0 {
		res.Resume = TerminalResumeCommand
	}
	st, _, err := readDrain(t.repoRoot)
	if err != nil {
		return res
	}
	for _, l := range st.Lanes {
		// A lane passed over is named among PassedLanes, not as the lane left.
		if l.Outcome != DrainLaneInProgress || t.passedLane(l.RunID) {
			continue
		}
		res.Run, res.Issue, res.Lapses, res.LeftToLapse = l.RunID, l.Issue, nil, false
		run, rerr := ReadState(t.repoRoot, l.RunID)
		// A later drive's start takes up a parked lane only where a stop's mark stands on
		// it (takeEarlier), so the command is named for that one alone; the moves of any
		// drain read the others from their pull requests.
		parked := rerr == nil && parkedAtLand(run) >= 0
		note, marked := "", false
		if parked {
			note = waitsForItsTakeOver(t.repoRoot, l.RunID, true)
			c, cerr := runControl(t.repoRoot, l.RunID)
			marked = cerr == nil && c.Stop != nil
		}
		// Nor is any named after a hold to a run's profile, whose reason names the ways on.
		if why != TerminalHalted && !driveStopped(l.RunID) && (!parked || marked) && !t.heldToProfile {
			res.Resume = TerminalResumeCommand
		}
		if note != "" {
			t.say("%s: %s", l.Issue, note)
		}
		if rerr == nil {
			if i := run.current(); i >= 0 {
				res.Lane = run.Lanes[i].ID
			}
		}
		if why == TerminalKilled || t.unstoppedRun(l.RunID) {
			if snap, err := claim.View(t.repoRoot, t.now()); err == nil {
				if r, ok := snap.Lookup(l.Issue); ok && r.Copy == claim.CopyThis && r.State == claim.StateLive {
					// A hold an earlier drive left to lapse that no take-up has taken off is
					// left to lapse as well as one this drive left (keptRenewers).
					at := r.EffectiveExpiry
					res.Lapses, res.LeftToLapse = &at, t.lapsing[l.RunID] || (rerr == nil && leftToLapse(t.repoRoot, run, snap))
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
