package loop

// loop.go is the step interface (spec piece 2; decision 5's default driver):
// Start creates a run after the checks, Advance performs the next step and
// exits, Receipt verifies what an agent step waited on and advances, and
// Status reads. Each takes the run tier's lock, reads the state first and
// writes it last; a step that fails leaves the state exactly as it was.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// Options are the seams tests set; the zero value is production.
type Options struct {
	// Now is the clock; nil means time.Now.
	Now func() time.Time
	// Minter mints the run id; the zero value is production.
	Minter recordid.Minter
}

func (o Options) now() time.Time {
	if o.Now != nil {
		return o.Now().UTC().Truncate(time.Second)
	}
	return time.Now().UTC().Truncate(time.Second)
}

// Context is what a step's body is handed: where the checkout and the run
// live, the run as it stood before the step, and the clock.
type Context struct {
	RepoRoot string
	// RunDir is the run's directory, relative to RepoRoot.
	RunDir string
	State  State
	Now    time.Time
}

// Outcome is what a step's body returns. A body that hands its work to an agent
// returns Await: the lane then waits on the receipt it names and the step
// completes only when Receipt verifies it.
type Outcome struct {
	Await *Await
	// Note is the run record's line for the step.
	Note string
}

// Handler performs one step for a lane, writing what it made into lane. It
// must be idempotent: a process killed after the body's effect and before the
// state write runs the body again on the next call, so a body finds what it
// made last time rather than making it twice. An error leaves the state as it
// was; a *Refusal error is passed to the caller as the refusal.
type Handler func(c Context, lane *Lane) (Outcome, error)

// Verifier checks the receipt an agent step waited on. An error refuses the
// receipt and the lane stays where it was.
type Verifier func(c Context, lane *Lane, receipt string) error

// StepDef is one step of the lane sequence: its name, the spec piece that
// delivers its body, and the body — nil when this build does not carry it.
type StepDef struct {
	Name   StepName
	Piece  int
	Run    Handler
	Verify Verifier
}

// Steps is the lane sequence with its bodies.
type Steps []StepDef

func (s Steps) lookup(name StepName) (StepDef, bool) {
	for _, d := range s {
		if d.Name == name {
			return d, true
		}
	}
	return StepDef{}, false
}

// Sequence is the lane's steps in the order the loop performs them: make the
// lane's worktree, render its brief, hand it to an implementer and take the
// receipt, run the validators, land it. A lane past its last step is
// StepDone.
var Sequence = []StepName{StepWorktree, StepBrief, StepImplement, StepValidate, StepLand}

// after returns the step that follows name in Sequence.
func after(name StepName) StepName {
	for i, n := range Sequence {
		if n == name && i+1 < len(Sequence) {
			return Sequence[i+1]
		}
	}
	return StepDone
}

// DefaultSteps is the lane sequence this build carries, each step with the
// spec piece that delivers its body. No body is built yet: the worktree and
// the brief, the receipt and the validators, and the landing are the next
// lanes of spc-2609202134338445, and each registers its body here.
func DefaultSteps() Steps {
	return Steps{
		{Name: StepWorktree, Piece: 6},
		{Name: StepBrief, Piece: 5},
		{Name: StepImplement, Piece: 7},
		{Name: StepValidate, Piece: 8},
		{Name: StepLand, Piece: 9},
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
	Checks  []CheckRow    `json:"checks"`
	Next    string        `json:"next"`
}

// StepResult is what Advance and Receipt return.
type StepResult struct {
	RunID string `json:"run_id"`
	Lane  string `json:"lane,omitempty"`
	// Performed is the step this call completed; empty when it completed none
	// (the lane awaits a receipt, or the run is complete).
	Performed StepName `json:"performed,omitempty"`
	// Step is the lane's next step after the call.
	Step StepName `json:"step,omitempty"`
	// Awaiting is what the lane waits on, when it waits on an agent.
	Awaiting *Await `json:"awaiting,omitempty"`
	Complete bool   `json:"complete"`
	Next     string `json:"next"`
}

// Start runs the checks for key and, when every one passes, creates the run:
// the state file with one lane at the sequence's first step, the spec's other
// unlanded steps pending, and the record's first line. A refused check writes
// nothing. A live run for the same key in this checkout is resumed — named, not
// duplicated — so starting again after a kill loses nothing and repeats nothing.
func Start(repoRoot, key string, o Options) (StartResult, error) {
	if err := tierPresent(repoRoot); err != nil {
		return StartResult{}, err
	}
	chk, err := Check(repoRoot, key)
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
		for _, st := range runs {
			if recordid.SameID(st.Key, chk.Key) && !st.Complete() {
				res = startResult(st, chk, true)
				return nil
			}
		}
		id, err := freeRunID(root, o.Minter)
		if err != nil {
			return err
		}
		if err := fsutil.EnsureRealDirAll(repoRoot, runRel(id), dirPerm); err != nil {
			return fmt.Errorf("creating %s: %w", runRel(id), err)
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
		}
		openNextLane(&st)
		st.Record = append(st.Record, Entry{At: now, Lane: st.Lanes[0].ID, Step: "start",
			Note: fmt.Sprintf("checks passed; %s opened for step %d of %s (%s)", st.Lanes[0].ID, st.Lanes[0].SpecStep, st.Spec, st.Lanes[0].StepTitle)})
		if err := writeState(root, st); err != nil {
			return err
		}
		res = startResult(st, chk, false)
		return nil
	})
	return res, err
}

func startResult(st State, chk CheckResult, resumed bool) StartResult {
	res := StartResult{RunID: st.RunID, State: StateRelPath(st.RunID), Resumed: resumed,
		Pending: st.Pending, Checks: chk.Checks}
	if i := st.current(); i >= 0 {
		res.Lane = st.Lanes[i]
		res.Next = nextMove(st, st.Lanes[i])
	}
	if res.Pending == nil {
		res.Pending = []PendingStep{}
	}
	return res
}

// openNextLane opens a lane for the first pending spec step. It is state-only:
// the lane's first step is what makes anything.
func openNextLane(st *State) {
	if len(st.Pending) == 0 {
		return
	}
	p := st.Pending[0]
	st.Pending = st.Pending[1:]
	st.Lanes = append(st.Lanes, Lane{
		ID:        fmt.Sprintf("lane-%d", len(st.Lanes)+1),
		Key:       st.Key,
		SpecStep:  p.Number,
		StepTitle: p.Title,
		Step:      Sequence[0],
	})
}

// Advance performs the next step of the run's current lane and returns. A lane
// that awaits a receipt performs nothing and re-tells what it awaits; a run
// that is complete says so; a run paused by its window clock is refused until
// next_eligible_at. A step whose body this build does not carry is refused
// naming the piece that delivers it. The state is written only after a body
// succeeds, and then once.
func Advance(repoRoot, runID string, steps Steps, o Options) (StepResult, error) {
	var res StepResult
	err := mutate(repoRoot, runID, func(root *os.Root, st *State) (bool, error) {
		now := o.now()
		if st.NextEligibleAt != nil && now.Before(*st.NextEligibleAt) {
			return false, contend("pause", "", "", "the run is paused until "+st.NextEligibleAt.UTC().Format(time.RFC3339),
				"run the step again at or after that time")
		}
		i := st.current()
		if i < 0 {
			res = StepResult{RunID: st.RunID, Complete: true, Next: "nothing: every lane of " + st.RunID + " is done"}
			return false, nil
		}
		lane := st.Lanes[i]
		if lane.Awaiting != nil {
			res = laneResult(*st, lane, "")
			return false, nil
		}
		def, ok := steps.lookup(lane.Step)
		if !ok || def.Run == nil {
			piece := ""
			if ok {
				piece = fmt.Sprintf(" (piece %d of %s delivers it)", def.Piece, specOf(*st))
			}
			return false, refusef(string(lane.Step), lane.ID,
				"use an abcd that carries the step; the run is unchanged and resumes here",
				"the %s step is not built in this abcd%s", lane.Step, piece)
		}
		c := Context{RepoRoot: repoRoot, RunDir: runRel(st.RunID), State: *st, Now: now}
		out, err := def.Run(c, &lane)
		if err != nil {
			return false, err
		}
		performed := StepName("")
		if out.Await != nil {
			if out.Await.Since.IsZero() {
				out.Await.Since = now
			}
			lane.Awaiting = out.Await
			note := out.Note
			if note == "" {
				note = "awaiting the " + out.Await.Role + "'s receipt at " + out.Await.Receipt
			}
			st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Step: string(lane.Step), Note: note})
		} else {
			performed = lane.Step
			st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Step: string(lane.Step), Note: out.Note})
			lane.Step = after(lane.Step)
		}
		st.Lanes[i] = lane
		if lane.Step == StepDone {
			openNextLane(st)
		}
		st.UpdatedAt = now
		res = laneResult(*st, lane, performed)
		return true, nil
	})
	return res, err
}

// Receipt hands back the receipt an agent step waited on. It is refused when no
// lane awaits one, when the path is not the one the step named, when this build
// carries no verifier for the step, and when the verifier refuses it; in every
// refusal the lane stays where it was. A verified receipt completes the step.
func Receipt(repoRoot, runID, receipt string, steps Steps, o Options) (StepResult, error) {
	var res StepResult
	err := mutate(repoRoot, runID, func(root *os.Root, st *State) (bool, error) {
		now := o.now()
		i := st.current()
		if i < 0 || st.Lanes[i].Awaiting == nil {
			return false, refuse("receipt", "", "", "no lane of "+st.RunID+" awaits a receipt",
				"run `abcd implement step`; it names the receipt when a step hands work to an agent")
		}
		lane := st.Lanes[i]
		if !samePath(repoRoot, receipt, lane.Awaiting.Receipt) {
			return false, refuse("receipt", "", lane.ID, "the "+string(lane.Step)+" step awaits its receipt at "+lane.Awaiting.Receipt+", not at the path given",
				"hand back `abcd implement receipt "+lane.Awaiting.Receipt+"`")
		}
		def, ok := steps.lookup(lane.Step)
		if !ok || def.Verify == nil {
			return false, refusef("receipt", lane.ID, "use an abcd that carries the verifier; the lane still awaits the receipt",
				"the %s step's receipt verifier is not built in this abcd (piece %d of %s delivers it)", lane.Step, def.Piece, specOf(*st))
		}
		c := Context{RepoRoot: repoRoot, RunDir: runRel(st.RunID), State: *st, Now: now}
		if err := def.Verify(c, &lane, lane.Awaiting.Receipt); err != nil {
			if _, ok := AsRefusal(err); ok {
				return false, err
			}
			return false, refuse("receipt", "", lane.ID, err.Error(), "correct what the reason names, then hand the receipt back")
		}
		performed := lane.Step
		lane.Receipt = lane.Awaiting.Receipt
		lane.Awaiting = nil
		st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Step: "receipt",
			Note: "the " + string(performed) + " step's receipt verified at " + lane.Receipt})
		lane.Step = after(lane.Step)
		st.Lanes[i] = lane
		if lane.Step == StepDone {
			openNextLane(st)
		}
		st.UpdatedAt = now
		res = laneResult(*st, lane, performed)
		return true, nil
	})
	return res, err
}

// laneResult reports where a lane stands after a call.
func laneResult(st State, lane Lane, performed StepName) StepResult {
	res := StepResult{RunID: st.RunID, Lane: lane.ID, Performed: performed, Step: lane.Step, Awaiting: lane.Awaiting}
	if st.Complete() {
		res.Complete = true
		res.Next = "nothing: every lane of " + st.RunID + " is done"
		return res
	}
	if i := st.current(); i >= 0 {
		res.Next = nextMove(st, st.Lanes[i])
	}
	return res
}

// nextMove is the one sentence a caller is told to do next.
func nextMove(st State, lane Lane) string {
	if lane.Awaiting != nil {
		return fmt.Sprintf("start a fresh %s agent with the brief %s; when it has written its receipt, run `abcd implement receipt %s`",
			lane.Awaiting.Role, lane.Awaiting.Brief, lane.Awaiting.Receipt)
	}
	return fmt.Sprintf("run `abcd implement step` to take %s's %s step", lane.ID, lane.Step)
}

// specOf names the spec a run builds against, for a refusal.
func specOf(st State) string {
	if st.Spec != "" {
		return st.Spec
	}
	return "the spec"
}

// samePath reports whether two paths name the same file, a relative one read
// against the checkout root.
func samePath(repoRoot, a, b string) bool {
	abs := func(p string) string {
		if !filepath.IsAbs(p) {
			p = filepath.Join(repoRoot, filepath.FromSlash(p))
		}
		return filepath.Clean(p)
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
