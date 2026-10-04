package loop

// loop.go is the step interface (spec piece 2; decision 5's default driver):
// Start creates a run after the checks, advance performs the lane's next stage
// and exits, Receipt verifies what an agent stage waited on and advances, and
// Status reads. Each takes the run tier's lock, reads the state first and
// writes it last; a stage that fails leaves the state exactly as it was.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/core/statusblock"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// Options are the seams tests set; the zero value is production.
type Options struct {
	// Now is the clock; nil means time.Now.
	Now func() time.Time
	// Minter mints the run id; the zero value is production.
	Minter recordid.Minter
	// Session is the joined session of the shared run state
	// (~/.abcd.noindex/runs/<root-sha>/) a new run is started for. When set, Start
	// claims the intent for it there, so a build of the same intent from any
	// other checkout of the repository sees the run before its lane has moved
	// or claimed anything (iss-2609252050506863). Empty, the run holds no
	// claim and is invisible to another checkout until its lane shows.
	Session string
	// Pace, SubAgents and FixRounds are the --pace, --sub-agents and
	// --fix-rounds flags as typed; nil when the flag was not given. They set a
	// new run's pace over every configured layer.
	Pace, SubAgents, FixRounds *string
	// Roots are where the pace's configuration layers are read; nil reads
	// them at layered.RootsFor(repoRoot).
	Roots *layered.Roots
}

// StageClaim is the refusal stage of a start whose shared-run claim is refused
// for a reason of the caller's own (a session that has not joined, a bound).
const StageClaim = "claim"

// roots are where the pace's configuration layers are read.
func (o Options) roots(repoRoot string) layered.Roots {
	if o.Roots != nil {
		return *o.Roots
	}
	r, _ := layered.RootsFor(repoRoot)
	return r
}

func (o Options) now() time.Time {
	if o.Now != nil {
		return o.Now().UTC().Truncate(time.Second)
	}
	return time.Now().UTC().Truncate(time.Second)
}

// Context is what a stage's body is handed: where the checkout and the run
// live, the run as it stood before the stage, and the clock.
type Context struct {
	RepoRoot string
	// RunDir is the run's directory, relative to RepoRoot.
	RunDir string
	State  State
	Now    time.Time
	// Await is the await a verifier is handed the receipt of; nil for a body.
	Await *Await
}

// Outcome is what a stage's body returns. A body that hands its work to an agent
// returns Await: the lane then waits on the receipt it names and the stage
// completes only when Receipt verifies it.
type Outcome struct {
	Await *Await
	// HandBack stops the lane: it is handed back to the person with what the
	// stage found, and the loop starts nothing further for it (itd-50,
	// criterion 2). Set only with no Await.
	HandBack *HandBack
	// Stay records a step of a stage that takes several invocations (the
	// landing): the lane's changes are written and the record gets the note,
	// and the lane stays at the stage for the next invocation's step. Set only
	// with no Await and no HandBack.
	Stay bool
	// Goto sends the lane back to an earlier stage (the landing's sync sends
	// it to a fresh round); set only with no Await, HandBack or Stay.
	Goto Stage
	// Note is the run record's line for the stage.
	Note string
}

// Handler performs one stage for a lane, writing what it made into lane. It
// must be idempotent: a process killed after the body's effect and before the
// state write runs the body again on the next call, so a body finds what it
// made last time rather than making it twice. An error leaves the state as it
// was; a *Refusal error is passed to the caller as the refusal.
type Handler func(c Context, lane *Lane) (Outcome, error)

// Verifier checks the receipt an agent stage waited on. An error refuses the
// receipt and the lane stays where it was.
type Verifier func(c Context, lane *Lane, receipt string) error

// StageDef is one stage of the lane sequence: its name, the spec piece that
// delivers its body, and the body — nil when this build does not carry it.
//
// A stage that hands the lane to several agents in turn (the validators, and the
// fresh implementer their findings go to) sets Repeats: a verified receipt then
// returns the lane to the stage's body rather than completing the stage, and the
// body decides, on the next step, whom it hands the lane to next or that the
// stage is complete.
type StageDef struct {
	Name    Stage
	Piece   int
	Run     Handler
	Verify  Verifier
	Repeats bool
}

// Stages is the lane sequence with its bodies.
type Stages []StageDef

func (s Stages) lookup(name Stage) (StageDef, bool) {
	for _, d := range s {
		if d.Name == name {
			return d, true
		}
	}
	return StageDef{}, false
}

// Sequence is the lane's stages in the order the loop performs them: make the
// lane's worktree, render its brief, hand it to an implementer and take the
// receipt, run the validators, land it. A lane past its last stage is
// StageDone.
var Sequence = []Stage{StageWorktree, StageBrief, StageImplement, StageValidate, StageLand}

// after returns the stage that follows name in Sequence.
func after(name Stage) Stage {
	for i, n := range Sequence {
		if n == name && i+1 < len(Sequence) {
			return Sequence[i+1]
		}
	}
	return StageDone
}

// DefaultStages is the lane sequence this build carries, each stage with the
// spec piece that delivers its body: the worktree (lane.go), the brief
// (brief.go), the implement stage with its receipt's verifier (receipt.go) and
// the validators with theirs (validate.go), and the landing (land.go).
func DefaultStages() Stages {
	return Stages{
		{Name: StageWorktree, Piece: 6, Run: worktreeStage},
		{Name: StageBrief, Piece: 5, Run: briefStage},
		{Name: StageImplement, Piece: 7, Run: implementStage, Verify: verifyReceipt},
		{Name: StageValidate, Piece: 8, Run: validateStage, Verify: verifyValidation, Repeats: true},
		{Name: StageLand, Piece: 9, Run: landStage},
	}
}

// StartResult is what Start returns.
type StartResult struct {
	RunID string `json:"run_id"`
	// State is the state file, relative to the checkout root.
	State string `json:"state"`
	// Resumed is true when a live run for the key already existed: the call
	// created nothing and names that run.
	Resumed bool          `json:"resumed"`
	Lane    Lane          `json:"lane"`
	Pending []PendingStep `json:"pending"`
	// Checks are the pre-start checks' rows when the call created the run; a
	// resumed start runs none, and carries none.
	Checks []CheckRow `json:"checks"`
	// Claim is the shared-run claim a new run took for Options.Session; nil when
	// no session was named or the start resumed a run, and then null in the
	// JSON, never absent, so the payload says the run holds no claim.
	Claim *implement.ClaimResult `json:"claim"`
	// Pace is the run's pace, each number with the layer that supplied it.
	// Null for a run started before the loop paced a run.
	Pace *Pace `json:"pace"`
	// HandBack is set when the run's lane stands handed back to the person.
	HandBack *HandBack `json:"hand_back,omitempty"`
	Next     string    `json:"next"`
}

// StepResult is what advance and Receipt return.
type StepResult struct {
	RunID string `json:"run_id"`
	Lane  string `json:"lane,omitempty"`
	// PerformedStage is the stage this call completed; empty when it completed none
	// (the lane awaits a receipt, or the run is complete).
	PerformedStage Stage `json:"performed_stage,omitempty"`
	// Stage is the lane's next stage after the call.
	Stage Stage `json:"stage,omitempty"`
	// Awaiting is what the lane waits on, when it waits on an agent.
	Awaiting *Await `json:"awaiting,omitempty"`
	Complete bool   `json:"complete"`
	// NextEligibleAt is set when this call closed the run's window: nothing
	// was performed, and no stage is taken before this time.
	NextEligibleAt *time.Time `json:"next_eligible_at,omitempty"`
	// HandBack is set when the lane was handed back to the person: this call
	// stopped it, or it stood stopped when the run was started again.
	HandBack *HandBack `json:"hand_back,omitempty"`
	// Slots is the agents the run has out after the call, and Ceiling the most
	// it may have (the run's pace.sub_agents); CeilingReached is set when this
	// call found every slot taken and handed nothing out.
	Slots          int  `json:"slots"`
	Ceiling        int  `json:"ceiling"`
	CeilingReached bool `json:"ceiling_reached,omitempty"`
	// Alive is every lane of the run with anything left, with its stage and
	// each agent it awaits, the held lanes included.
	Alive []AliveLane `json:"alive,omitempty"`
	// Blocked are the landings that waited on the forge's merge this call
	// while another lane moved: each holds only its own lane. Any other refused
	// stage is the call's answer, and no lane moves.
	Blocked []Refusal `json:"blocked,omitempty"`
	// Route is the route that ran the agent when the process driver started
	// it through a runner (Drive); absent when the host is to run it.
	Route *runner.RouteRecord `json:"route,omitempty"`
	// Fallback is the fallback receipt this call recorded when the runner a
	// role is routed to did not run it and the host is handed the role.
	Fallback *runner.FallbackReceipt `json:"fallback,omitempty"`
	Next     string                  `json:"next"`
	// handed is true when this call handed the lane to an agent, false when
	// it re-told an await an earlier call began: only the call that hands the
	// work out may start a runner for it.
	handed bool
}

// Start resumes the live run for key, or runs the checks and, when every one
// passes, creates the run: the state file with one lane at the sequence's first
// stage, the spec's other unlanded steps pending, and the record's first line. A
// refused check writes nothing.
//
// A live run for the same key in this checkout is looked up first, under the
// lock, and resumed — named, not duplicated, and not re-judged. The checks
// judged the record when the run was created; since then the run's own lanes
// change the tree they read (a lane's worktree moves the intent to shipped/, a
// lane claims it), so judging again would refuse the run as its own peer. So
// starting again after a kill loses nothing and repeats nothing, and the checks
// run only when a run is created. Only the key's shape is checked before the
// lookup, so a path is never built from a key that is not an intent id.
//
// With o.Session named, a new run also claims its intent in the shared run
// state for that session, with the run id as the lane and the longest lease a
// claim takes: the peers check then counts that session's own live claim on the
// intent as its own, and a build from another checkout counts it as a peer's. A
// session that has not joined is refused before anything is created; a claim
// refused under the lock (a racing holder, a second session's bound) leaves no
// run behind; and a run whose state cannot be written releases the claim it
// took.
func Start(repoRoot, key string, o Options) (StartResult, error) { return start(repoRoot, key, o, nil) }

// start is Start, carrying the pick `abcd build next` made when pick is not
// nil: the run's state records it, with the entry its first lane commits
// composed from the run id minted here. A picked start never resumes: a live
// run for the key is refused, since the pick is a new run's reason.
func start(repoRoot, key string, o Options, pick *RunPick) (StartResult, error) {
	if err := tierPresent(repoRoot); err != nil {
		return StartResult{}, err
	}
	if row, ok := keyCheck(key); !ok {
		return StartResult{}, CheckResult{Key: key, Checks: []CheckRow{row}}.refusal()
	}
	flags, err := parsePaceFlags(o.Pace, o.SubAgents, o.FixRounds)
	if err != nil {
		return StartResult{}, err
	}
	if res, ok, err := resumeLive(repoRoot, key); err != nil || ok {
		if err == nil && pick != nil {
			err = pickedLive(res)
		}
		if err == nil && flags.set {
			err = resumeWithFlags(res, flags)
		}
		if err != nil {
			return StartResult{}, err
		}
		return res, nil
	}
	pace, err := resolvePace(o.roots(repoRoot), flags)
	if err != nil {
		return StartResult{}, err
	}
	var shared *implement.Run
	if o.Session != "" {
		run, err := sharedRunFor(repoRoot, o.Session)
		if err != nil {
			return StartResult{}, err
		}
		shared = run
	}
	chk, err := check(repoRoot, key, o.Session, nil)
	if err != nil {
		return StartResult{}, err
	}
	if !chk.OK {
		return StartResult{}, chk.refusal()
	}
	if err := fsutil.EnsureRealDirAll(repoRoot, RunRelDir, dirPerm); err != nil {
		return StartResult{}, fmt.Errorf("creating %s: %w", RunRelDir, err)
	}
	var res StartResult
	err = withLock(repoRoot, func(root *os.Root) error {
		runs, err := readRuns(root)
		if err != nil {
			return err
		}
		// A start that raced this one past the lookup above created the run
		// while the checks ran: it is resumed, not duplicated.
		if st, ok := liveRun(runs, chk.Key); ok {
			res = startResult(st, nil, true)
			if pick != nil {
				return pickedLive(res)
			}
			if flags.set {
				return resumeWithFlags(res, flags)
			}
			return nil
		}
		id, err := freeRunID(root, o.Minter)
		if err != nil {
			return err
		}
		if err := fsutil.EnsureRealDirAll(repoRoot, runRel(id), dirPerm); err != nil {
			return fmt.Errorf("creating %s: %w", runRel(id), err)
		}
		var claim *implement.ClaimResult
		if shared != nil {
			c, err := shared.Claim(implement.ClaimRequest{Session: o.Session, Record: chk.record(), Lane: id, Lease: implement.MaxLease})
			if err != nil {
				_ = root.Remove(runRel(id))
				return claimRefusal(o.Session, chk.record(), err)
			}
			claim = &c
		}
		now := o.now()
		st := State{
			SchemaVersion: SchemaVersion,
			RunID:         id,
			Key:           chk.Key,
			Intent:        chk.Intent,
			Spec:          chk.Spec,
			Driver:        DriverHost,
			CreatedAt:     now,
			UpdatedAt:     now,
			Lanes:         []Lane{},
			Pending:       append([]PendingStep(nil), chk.steps...),
			Record:        []Entry{},
			// The run's first window opens at its start.
			WindowStartedAt: &now,
			Pace:            &pace,
		}
		openNextLane(&st)
		st.Record = append(st.Record, Entry{At: now, Lane: st.Lanes[0].ID, Stage: "start",
			Note: "checks passed; " + st.Lanes[0].ID + " opened for " + laneWork(st, st.Lanes[0])})
		st.Record = append(st.Record, Entry{At: now, Stage: StagePace,
			Note: "pace " + pace.String() + "; the first window opens now"})
		if pick != nil {
			rp := *pick
			rp.Lane = st.Lanes[0].ID
			rp.Entry = intent.PickEntryText(id, now, rp.Pick)
			if rp.Excluded == nil {
				rp.Excluded = []Excluded{}
			}
			st.Pick = &rp
			st.Record = append(st.Record, Entry{At: now, Lane: rp.Lane, Stage: StagePick,
				Note: pickNote(rp)})
		}
		if err := writeState(root, st); err != nil {
			if claim != nil && !claim.Renewed {
				_, _ = shared.Release(o.Session, chk.record())
			}
			return err
		}
		res = startResult(st, chk.Checks, false)
		res.Claim = claim
		return nil
	})
	return res, err
}

// pickedLive refuses a pick whose intent already has a run in progress: the
// pick starts a run, and that run is resumed with `abcd implement step`.
func pickedLive(res StartResult) error {
	return contend(StagePick, "", "", res.RunID+" is already in progress for the intent the pick chose",
		"resume it with `abcd implement step`, or run `abcd build next` again to pick among the rest")
}

// pickNote is the run record's line for the pick: the chosen intent, its
// score table and the runner-up.
func pickNote(p RunPick) string {
	var rows []string
	for _, c := range p.Pick.Candidates {
		rows = append(rows, fmt.Sprintf("%s %d", c.ID, c.Score.Total))
	}
	note := fmt.Sprintf("picked %s from %d candidate(s) (%s), %d excluded; its entry is %s's first commit",
		p.Pick.Chosen.ID, len(p.Pick.Candidates), strings.Join(rows, ", "), len(p.Excluded), p.Lane)
	if p.Pick.TieBrokenByAge {
		note += "; the tie was broken by age"
	}
	return note
}

// sharedRunFor opens the shared run state for session, refusing at the claim
// stage a checkout with no root commit, a session that is not a name, and a run
// the session has not joined, before anything is created.
func sharedRunFor(repoRoot, session string) (*implement.Run, error) {
	sha := gitutil.RootCommit(repoRoot)
	run, err := implement.OpenJoined(sha, session)
	if err == nil {
		_, err = run.Joined(session)
	}
	if err != nil {
		if errors.Is(err, implement.ErrRefused) {
			return nil, refuse(StageClaim, "", "", err.Error(),
				"join the shared run first: `abcd implement join --session <id> --role first|second`")
		}
		return nil, err
	}
	return run, nil
}

// claimRefusal maps a refused shared-run claim onto the loop's refusal: a
// record another session holds is the peers check's contention, anything else
// the claim stage's refusal.
func claimRefusal(session, record string, err error) error {
	switch {
	case errors.Is(err, implement.ErrContention):
		return contend("check", CheckPeers, "", err.Error(),
			"take other work, or coordinate with the peer; `abcd implement` shows what each session holds")
	case errors.Is(err, implement.ErrRefused):
		return refuse(StageClaim, "", "", fmt.Sprintf("session %s cannot claim %s: %v", session, record, err),
			"start the build without --session, or from a session whose bounds allow the lane")
	}
	return err
}

// resumeLive returns the live run for key, under the lock, when this checkout
// has one. A checkout with no run directory has none, and the lookup creates
// nothing; a run directory that is not a real directory is left to the create
// path, which refuses it.
func resumeLive(repoRoot, key string) (StartResult, bool, error) {
	if !fsutil.IsRealDir(filepath.Join(repoRoot, filepath.FromSlash(RunRelDir))) {
		return StartResult{}, false, nil
	}
	var res StartResult
	found := false
	err := withLock(repoRoot, func(root *os.Root) error {
		runs, err := readRuns(root)
		if err != nil {
			return err
		}
		if st, ok := liveRun(runs, key); ok {
			res, found = startResult(st, nil, true), true
		}
		return nil
	})
	return res, found, err
}

// resumeWithFlags holds a resumed start's --pace and --sub-agents to the pace
// the run started on: the pace is set when a run starts, so a flag naming
// other numbers is refused rather than silently ignored, and one naming the
// same numbers resumes.
func resumeWithFlags(res StartResult, f paceFlags) error {
	var want Pace
	if res.Pace != nil {
		want = *res.Pace
	}
	got := want
	if f.work != nil {
		got.WorkMinutes.Value, got.PauseMinutes.Value = *f.work, *f.pause
	}
	if f.subs != nil {
		got.SubAgents.Value = *f.subs
	}
	if f.fix != nil {
		// A run started before the pace carried a cap runs on the bundled one.
		if want.FixRounds.Layer == "" {
			want.FixRounds.Value = BundledFixRounds
		}
		got.FixRounds.Value = *f.fix
	}
	if res.Pace != nil && got.same(want) {
		return nil
	}
	running := "no pace (it started before the loop paced a run)"
	if res.Pace != nil {
		running = "pace " + res.Pace.String()
	}
	typed := strings.Join(strings.Fields(f.paceOrigin+" "+f.subsOrigin+" "+f.fixOrigin), " ")
	return refuse(StagePace, "", "", fmt.Sprintf("%s is in progress on %s; %s names another, and a pace is set when a run starts",
		res.RunID, running, typed),
		"resume without --pace, --sub-agents and --fix-rounds; the run keeps the pace it started on")
}

// liveRun returns the run for key that is not complete.
func liveRun(runs []State, key string) (State, bool) {
	for _, st := range runs {
		if recordid.SameID(st.Key, key) && !st.Complete() {
			return st, true
		}
	}
	return State{}, false
}

// startResult reports a run as Start returns it. checks are the rows the call
// ran: a resumed start runs none.
func startResult(st State, checks []CheckRow, resumed bool) StartResult {
	if checks == nil {
		checks = []CheckRow{}
	}
	res := StartResult{RunID: st.RunID, State: StateRelPath(st.RunID), Resumed: resumed,
		Pending: st.Pending, Checks: checks, Pace: st.Pace}
	if i := st.current(); i >= 0 {
		res.Lane = st.Lanes[i]
		res.Next = nextMove(st, st.Lanes[i])
	}
	res.HandBack = res.Lane.HandBack
	if res.Pending == nil {
		res.Pending = []PendingStep{}
	}
	return res
}

// laneWork names what a lane builds, for the run record: a spec step, or the
// issue an issue-keyed run fixes.
func laneWork(st State, l Lane) string {
	if st.Issue() != "" {
		return fmt.Sprintf("%s (%s)", st.Issue(), l.StepTitle)
	}
	return fmt.Sprintf("step %d of %s (%s)", l.SpecStep, st.Spec, l.StepTitle)
}

// openNextLane opens a lane for the first pending spec step. It is state-only:
// the lane's first stage is what makes anything.
func openNextLane(st *State) {
	if len(st.Pending) > 0 {
		openLane(st, 0)
	}
}

// openLane opens a lane for the pending step at index k, state-only.
func openLane(st *State, k int) {
	p := st.Pending[k]
	st.Pending = slices.Delete(slices.Clone(st.Pending), k, k+1)
	st.Lanes = append(st.Lanes, Lane{
		ID:        fmt.Sprintf("lane-%d", len(st.Lanes)+1),
		Key:       st.Key,
		SpecStep:  p.Number,
		StepTitle: p.Title,
		Stage:     Sequence[0],
	})
}

// advance performs one move of the run and returns (ruling DR6,
// spc-2609202134341288): a stage the binary performs itself on any lane first,
// which takes no slot, then, while the run's ceiling has a slot free, the first
// piece of work that needs an agent, in the order schedule.go gives. A call
// that finds the ceiling reached hands out nothing, names every lane alive and
// what it awaits, and writes the held work into the run's `waiting`. A run that
// is complete says so; a run paused by its window clock is refused until
// next_eligible_at. A stage whose body this build does not carry is refused
// naming the piece that delivers it. The state is written only after a body
// succeeds, and then once.
func advance(repoRoot, runID string, steps Stages, o Options) (StepResult, error) {
	var res StepResult
	var wait error
	err := mutate(repoRoot, runID, func(root *os.Root, st *State) (bool, error) {
		now := o.now()
		if st.NextEligibleAt != nil && now.Before(*st.NextEligibleAt) {
			return false, contend("pause", "", "", "the run is paused until "+st.NextEligibleAt.UTC().Format(time.RFC3339),
				"run `abcd implement step` again at or after that time")
		}
		if st.Complete() {
			res = StepResult{RunID: st.RunID, Complete: true, Next: "nothing: every lane of " + st.RunID + " is done"}
			return false, nil
		}
		// The window clock (itd-2609201925079472) is the run's, one for every
		// lane: a pause that has ended opens the next window; a window that has
		// elapsed closes here, and the call starts nothing on any lane.
		changed := false
		if st.NextEligibleAt != nil {
			openWindow(st, now)
			changed = true
		}
		if until, ok := windowElapsed(*st, now); ok {
			closeWindow(st, now, until)
			res = idleResult(*st)
			res.NextEligibleAt = &until
			res.Next = pausedMove(*st, until)
			return true, nil
		}
		r, moved, err := move(repoRoot, st, steps, now)
		if err != nil {
			if moved {
				// The wait, or a later lane's refusal, is the call's answer,
				// and the time a lane began waiting for its full check is
				// written with it (ruling DR6d-2).
				wait = err
				return true, nil
			}
			return false, err
		}
		res = r
		return changed || moved, nil
	})
	if err == nil && wait != nil {
		return StepResult{}, wait
	}
	return res, err
}

// windowElapsed reports whether a paced run's window has run its working
// minutes by now, and the next_eligible_at a pause starting now ends at. The
// pause runs from the moment the loop closes the window, not from the window's
// nominal end, so an invocation that comes late never shortens it.
func windowElapsed(st State, now time.Time) (time.Time, bool) {
	if st.Pace == nil || st.WindowStartedAt == nil {
		return time.Time{}, false
	}
	end := st.WindowStartedAt.Add(time.Duration(st.Pace.WorkMinutes.Value) * time.Minute)
	if now.Before(end) {
		return time.Time{}, false
	}
	return now.Add(time.Duration(st.Pace.PauseMinutes.Value) * time.Minute), true
}

// closeWindow ends the run's window: next_eligible_at is written and the
// record names the pause.
func closeWindow(st *State, now, until time.Time) {
	st.NextEligibleAt = &until
	st.UpdatedAt = now
	st.Record = append(st.Record, Entry{At: now, Stage: "pause",
		Note: fmt.Sprintf("the %d-minute window opened at %s has elapsed; no stage is taken before %s (a %d-minute pause)",
			st.Pace.WorkMinutes.Value, st.WindowStartedAt.UTC().Format(time.RFC3339), until.UTC().Format(time.RFC3339), st.Pace.PauseMinutes.Value)})
}

// openWindow opens the run's next window once its pause has ended.
func openWindow(st *State, now time.Time) {
	st.NextEligibleAt = nil
	st.WindowStartedAt = &now
	st.UpdatedAt = now
	note := "the pause has ended; a window opens"
	if st.Pace != nil {
		note = fmt.Sprintf("the pause has ended; a %d-minute window opens", st.Pace.WorkMinutes.Value)
	}
	st.Record = append(st.Record, Entry{At: now, Stage: "window", Note: note})
}

// pausedMove is the next move of a run whose window this call closed.
func pausedMove(st State, until time.Time) string {
	at := until.UTC().Format(time.RFC3339)
	if out := st.allAwaits(); len(out) > 0 {
		parts := make([]string, 0, len(out))
		for _, a := range out {
			parts = append(parts, fmt.Sprintf("the %s of %s with `abcd implement receipt %s`", a.Await.Role, a.Lane, a.Await.Receipt))
		}
		return fmt.Sprintf("nothing new before %s: the run's window has elapsed. The agents already started may still hand back their receipts (%s); run `abcd implement step` at or after %s",
			at, strings.Join(parts, "; "), at)
	}
	return fmt.Sprintf("nothing before %s: the run's window has elapsed; run `abcd implement step` at or after %s", at, at)
}

// Receipt hands back the receipt an agent stage waited on. The path is looked
// up among every outstanding await of the run, not only one lane's, and the
// lane that await belongs to is advanced (ruling DR6): a verified receipt frees
// its slot, which the next `implement step` fills. It is refused when no await
// names the path, when this build carries no verifier for the stage, and when
// the verifier refuses it; in every refusal the run stays where it was and the
// agent's slot stays taken.
func Receipt(repoRoot, runID, receiptPath string, steps Stages, o Options) (StepResult, error) {
	return receipt(repoRoot, runID, receiptPath, steps, o, nil)
}

// receipt is Receipt. A route that is not nil is the runner that ran the
// agent (Drive): it is stamped on what the verifier recorded from the receipt,
// and the record names it. The host's receipt carries none.
func receipt(repoRoot, runID, receipt string, steps Stages, o Options, route *runner.RouteRecord) (StepResult, error) {
	var res StepResult
	err := mutate(repoRoot, runID, func(root *os.Root, st *State) (bool, error) {
		now := o.now()
		i, k := st.findAwait(repoRoot, receipt)
		if i < 0 {
			return false, unknownReceipt(*st)
		}
		lane := st.Lanes[i]
		await := lane.Awaits[k]
		def, ok := steps.lookup(lane.Stage)
		if !ok || def.Verify == nil {
			return false, refusef("receipt", lane.ID, "use an abcd that carries the verifier; the lane still awaits the receipt",
				"the %s stage's receipt verifier is not built in this abcd (piece %d of %s delivers it)", lane.Stage, def.Piece, specOf(*st))
		}
		c := Context{RepoRoot: repoRoot, RunDir: runRel(st.RunID), State: *st, Now: now, Await: &await}
		if err := def.Verify(c, &lane, await.Receipt); err != nil {
			if _, ok := AsRefusal(err); ok {
				return false, err
			}
			return false, refuse("receipt", "", lane.ID, err.Error(), "correct what the reason names, then hand the receipt back")
		}
		if route != nil {
			stampRoute(&lane, await.Receipt, route)
			st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Stage: StageRunner, Note: routeNote(await.Role, *route)})
		}
		if lane.HandBack != nil {
			// The lane's own receipt handed the work back: the verifier has
			// discarded it, and the lane ends here, before the validators.
			handBackLane(st, &lane, *lane.HandBack, "", now)
			st.Lanes[i] = lane
			st.UpdatedAt = now
			res = laneResult(*st, lane, "", nil)
			res.HandBack = lane.HandBack
			return true, nil
		}
		performed := lane.Stage
		verified := await.Receipt
		lane.Awaits = slices.Delete(slices.Clone(lane.Awaits), k, k+1)
		if len(lane.Awaits) == 0 {
			lane.Awaits = nil
		}
		note := "the " + string(performed) + " stage's receipt verified at " + verified
		if def.Repeats {
			// The stage hands the lane to its next agent, or completes, on a
			// later step; the lane's own receipt stays the implementer's.
			if n := validationNote(lane, verified); n != "" {
				note += "; " + n
			}
			st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Stage: "receipt", Note: note})
			st.Lanes[i] = lane
			st.UpdatedAt = now
			res = laneResult(*st, lane, "", nil)
			return true, nil
		}
		lane.Receipt = verified
		st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Stage: "receipt", Note: note})
		lane.Stage = after(lane.Stage)
		st.Lanes[i] = lane
		st.UpdatedAt = now
		res = laneResult(*st, lane, performed, nil)
		return true, nil
	})
	return res, err
}

// laneResult reports where a lane stands after a call: the await this call
// handed out when it handed one out, or the lane's first.
func laneResult(st State, lane Lane, performed Stage, handed *Await) StepResult {
	res := StepResult{RunID: st.RunID, Lane: lane.ID, PerformedStage: performed, Stage: lane.Stage, Awaiting: handed,
		Slots: st.slotsInUse(), Ceiling: st.ceiling(), Alive: st.alive(), handed: handed != nil}
	if res.Awaiting == nil {
		res.Awaiting = lane.awaiting()
	}
	if st.Complete() {
		res.Complete = true
		res.Next = "nothing: every lane of " + st.RunID + " is done"
		return res
	}
	switch {
	case handed != nil:
		res.Next = handedMove(*handed)
	case lane.HandBack != nil:
		res.Next = handBackMove(st, lane)
	default:
		res.Next = nextMove(st, lane)
	}
	return res
}

// handedMove is the one sentence a caller is told when a call handed work to
// an agent.
func handedMove(a Await) string {
	return fmt.Sprintf("start a fresh %s agent with the brief %s; when it has written its receipt, run `abcd implement receipt %s`",
		a.Role, a.Brief, a.Receipt)
}

// nextMove is the one sentence a caller is told to do next.
func nextMove(st State, lane Lane) string {
	if lane.HandBack != nil {
		return handBackMove(st, lane)
	}
	if a := lane.awaiting(); a != nil {
		return handedMove(*a)
	}
	if lane.Stage == StageHeld {
		return heldMove(lane)
	}
	return fmt.Sprintf("run `abcd implement step` to take %s's %s stage", lane.ID, lane.Stage)
}

// specOf names the spec a run builds against, for a refusal.
func specOf(st State) string {
	if st.Spec != "" {
		return st.Spec
	}
	return "the spec"
}

// samePath reports whether two paths name the same file, a relative one read
// against the checkout root. Each is compared by its real path, so a checkout
// reached through a symlinked spelling (the caller's working directory) and
// the resolved one git names as the root are the same place
// (iss-2609261534097255).
func samePath(repoRoot, a, b string) bool {
	abs := func(p string) string {
		if !filepath.IsAbs(p) {
			p = filepath.Join(repoRoot, filepath.FromSlash(p))
		}
		return fsutil.RealExistingPath(filepath.Clean(p))
	}
	return a != "" && abs(a) == abs(b)
}

// Runs lists this checkout's runs, oldest first. An absent tier or run
// directory holds none. A state file that cannot be read fails the listing,
// naming it: a run the loop cannot read is not one it may skip past.
func Runs(repoRoot string) ([]State, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("opening the checkout to read the run state: %w", err)
	}
	defer root.Close()
	return readRuns(root)
}

func readRuns(root *os.Root) ([]State, error) {
	ids, err := runIDs(root)
	if err != nil {
		return nil, err
	}
	out := []State{}
	for _, id := range ids {
		st, err := readStateIn(root, id)
		if err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, nil
}

// Resolve names the run a call addresses. An explicit id is checked for shape
// and presence; no id addresses the one run in this checkout that is not
// complete, and is refused naming them when there are several, or none.
func Resolve(repoRoot, runID string) (string, error) {
	if runID != "" {
		if _, err := ReadState(repoRoot, runID); err != nil {
			return "", err
		}
		return runID, nil
	}
	runs, err := Runs(repoRoot)
	if err != nil {
		return "", err
	}
	var live []string
	for _, st := range runs {
		if !st.Complete() {
			live = append(live, st.RunID+" ("+st.Key+")")
		}
	}
	switch len(live) {
	case 0:
		return "", refuse("state", "", "", "no run in this checkout is in progress", "start one with `abcd build <itd-N>`")
	case 1:
		for _, st := range runs {
			if !st.Complete() {
				return st.RunID, nil
			}
		}
	}
	return "", refuse("state", "", "", fmt.Sprintf("%d runs are in progress: %v", len(live), live), "name one with --run")
}

// tierPresent refuses a checkout without the local tier, and never creates it:
// only a repository abcd manages has one. A symlink standing in for it is
// refused by the same test.
func tierPresent(repoRoot string) error {
	fi, err := os.Lstat(filepath.Join(repoRoot, filepath.FromSlash(TierRelDir)))
	if err != nil || !fi.IsDir() {
		return refuse("state", "", "", TierRelDir+"/ is not a directory in this checkout, so there is nowhere for a run to live",
			"run abcd in a repository it manages (`abcd ahoy` sets one up); the tier is never created on the way to a write")
	}
	return nil
}

// withLock runs fn holding the run tier's advisory lock, with the checkout
// opened as an os.Root for fn's reads and writes.
func withLock(repoRoot string, fn func(root *os.Root) error) error {
	lock := filepath.Join(repoRoot, filepath.FromSlash(RunRelDir), lockFileName)
	err := fsutil.WithFileLock(lock, lockTimeout, func() error {
		root, err := os.OpenRoot(repoRoot)
		if err != nil {
			return fmt.Errorf("opening the checkout: %w", err)
		}
		defer root.Close()
		return fn(root)
	})
	if errors.Is(err, fsutil.ErrLockContention) {
		return contend("state", "", "", "another invocation is changing this checkout's runs", "back off and retry")
	}
	return err
}

// mutate is every in-run mutation's frame: resolve nothing, check the tier,
// take the lock, read the state, let fn change it, and write it only when fn
// says it changed.
func mutate(repoRoot, runID string, fn func(root *os.Root, st *State) (bool, error)) error {
	if err := tierPresent(repoRoot); err != nil {
		return err
	}
	if !ValidRunID(runID) {
		_, err := ReadState(repoRoot, runID)
		return err
	}
	if !fsutil.IsRealDir(filepath.Join(repoRoot, filepath.FromSlash(runRel(runID)))) {
		return refuse("state", "", "", "no run "+runID+" in this checkout",
			"name a run `abcd implement status` lists, or start one with `abcd build <itd-N>`")
	}
	return withLock(repoRoot, func(root *os.Root) error {
		st, err := readStateIn(root, runID)
		if err != nil {
			return err
		}
		changed, err := fn(root, &st)
		if err != nil || !changed {
			return err
		}
		return writeState(root, st)
	})
}

// runIDDraws bounds the redraws freeRunID makes before it refuses.
const runIDDraws = 8

// freeRunID mints a run id whose directory does not exist yet. The mint reads
// no maximum (adr-45), so two starts in one second can draw one suffix; the
// coincidence is absorbed here by redrawing.
func freeRunID(root *os.Root, m recordid.Minter) (string, error) {
	var last string
	for range runIDDraws {
		id, err := m.Mint(RunIDFamily)
		if err != nil {
			return "", err
		}
		if _, err := root.Lstat(runRel(id)); errors.Is(err, os.ErrNotExist) {
			return id, nil
		} else if err != nil {
			return "", fmt.Errorf("checking %s: %w", runRel(id), err)
		}
		last = id
	}
	return "", fmt.Errorf("%d draws in a row named a run directory that already exists (last %s)", runIDDraws, last)
}

// StatusLanes is the state file read the status block's Now takes
// (itd-2609212103568351): one entry per run in progress, naming its intent and
// every lane of it alive (ruling DR6) — each lane's next stage and the roles it
// waits on, a held lane included — or, while no lane is alive and a spec step
// still waits, the run with its stage "pending". A complete run is not in a
// lane. An absent tier or run directory holds none. It is a
// statusblock.LaneReader.
func StatusLanes(repoRoot string) ([]statusblock.Started, error) {
	runs, err := Runs(repoRoot)
	if err != nil {
		return nil, err
	}
	out := []statusblock.Started{}
	for _, st := range runs {
		if st.Complete() {
			continue
		}
		id := st.Intent
		if id == "" {
			id = st.Key
		}
		// A lane's branch is its own record's (build/<run>-<lane>), set once its
		// worktree stage cuts it; the board marks the lane in flight from it.
		branches := map[string]string{}
		for _, l := range st.Lanes {
			branches[l.ID] = l.Branch
		}
		var lanes []statusblock.Lane
		for _, a := range st.alive() {
			lane := statusblock.Lane{Run: st.RunID, Lane: a.Lane, Stage: string(a.Stage), Waiting: a.Waiting, Branch: branches[a.Lane]}
			roles := make([]string, 0, len(a.Awaits))
			for _, aw := range a.Awaits {
				roles = append(roles, aw.Role)
			}
			lane.Awaiting = strings.Join(roles, ", ")
			lanes = append(lanes, lane)
		}
		if len(lanes) == 0 {
			lanes = []statusblock.Lane{{Run: st.RunID, Stage: "pending"}}
		}
		out = append(out, statusblock.Started{Intent: id, Lanes: lanes})
	}
	return out, nil
}

// StatusPeers is the peers read the status block's head takes (ruling CC1 of
// 2026-09-29): build next's own peers check, read once for the block and
// judged per intent, so the board's "next up" passes over exactly the intents
// another checkout holds that the pick passes over. It judges on behalf of no
// session, so every live claim is a peer's. A peer the listing cannot read
// fails closed on each record, as it does for the pick. It is a
// statusblock.PeerReader.
func StatusPeers(repoRoot string) (statusblock.HeldBy, error) {
	snap, err := readPeers(repoRoot)
	if err != nil {
		return nil, err
	}
	return func(r intent.ReadyResult) string {
		if row := peersCheck(r.IntentID, r.Bucket, "", snap); !row.OK {
			return row.Detail
		}
		return ""
	}, nil
}
