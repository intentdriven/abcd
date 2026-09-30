// Package loop is the implement loop's engine (itd-2609201916151817,
// spc-2609202134338445): the state file a run lives in, the checks that decide
// whether a run may start, and the step interface a driving host calls. `abcd
// build <itd-N>` starts a run; `abcd implement step` performs the next stage of
// the current lane and exits; `abcd implement receipt <path>` hands back what an
// agent stage waited on; `abcd implement status` renders the state. Nothing here holds a run in
// memory between invocations: every call reads the state file first, does at
// most one stage, writes the state file last and returns (decisions 1 and 2), so
// a killed process loses only the stage it was in, which the next call repeats.
//
// The state lives in the checkout's local tier:
//
//	.abcd/.work.local/run/<run-id>/state.json   the run: its key, lanes, clock and record
//	.abcd/.work.local/run/.lock                 the advisory lock every mutation takes
//
// The tier is never created here: only a repository abcd manages has it, which
// is what keeps a run managed-only without a check of its own to drift from
// (internal/core/mode draws the same line). The run directories beneath it are
// created one level at a time and proved real (fsutil.EnsureRealDirAll), and the
// state is written through the package's one atomic writer inside an os.Root, so
// a symlinked component is refused rather than followed.
//
// The lane's stages are a sequence (Sequence), and each stage's body is a
// Handler the piece of the spec that delivers it registers in DefaultStages: the
// worktree (piece 6, lane.go), the brief (piece 5, brief.go), the implement
// stage and its receipt's verifier (piece 7, receipt.go), the validators (piece
// 8) and the landing (piece 9). A stage whose body this build does not carry is
// refused by name, with the piece that delivers it, and the run is left
// unchanged. A lane's files live in its own directory of the run:
//
//	.abcd/.work.local/run/<run-id>/<lane-id>/brief.md      the brief the loop renders
//	.abcd/.work.local/run/<run-id>/<lane-id>/receipt.json  the implementer's receipt
//
// and its worktree in the machine-scoped store,
// ~/.abcd/worktrees/<root-sha>/<run-id>-<lane-id>. The
// process driver (piece 3, waiting on the runner intent itd-2609201916056194)
// is the same loop called by a process instead of a host: it starts the agent an
// Await names through the runner and then calls Receipt, so it needs no seam
// beyond the two this package exports.
//
// Core never writes to stdout; the CLI front door formats what these functions
// return.
package loop

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"slices"
	"time"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// TierRelDir is the local-ephemeral tier the run state lives in. It is never
// created here.
const TierRelDir = ".abcd/.work.local"

// RunRelDir is the directory every run of this checkout lives under.
const RunRelDir = TierRelDir + "/run"

// StateFileName is the name of a run's state file inside its directory.
const StateFileName = "state.json"

// lockFileName is the advisory lock every mutation of any run takes.
const lockFileName = ".lock"

// SchemaVersion is the state file's shape. A file carrying a version this
// build does not know is refused rather than read as this one: a field a later
// build added and this one would drop on its next write is a run silently
// losing state.
//
// Version 2 added the run's pace (itd-2609201925079472). Version 1 is its
// strict subset, so a version-1 file is read as a run started before the loop
// paced a run: it carries no pace, runs unpaced, and is written back at version
// 2 by its next mutation. A version-1 file carrying a pace is not one version 1
// wrote, and is refused.
//
// Version 3 added the pick `abcd build next` made (itd-2609211116005482): the
// run's `pick` and a lane's `pick_sha`. Versions 1 and 2 are its strict
// subsets, read as runs no pick started and written back at version 3; one of
// them carrying a pick is not one its version wrote, and is refused.
//
// Version 4 renamed the lane's stage (BU1, iss-2609291313276243): a lane's and
// a record line's `step` became `stage`, so the word "step" names only the
// spec's steps. Versions 1 to 3 wrote `step`, and are migrated on read: the
// read carries each `step` over to `stage` and writes nothing, and the run's
// next mutation writes the file back at version 4, as it does for the versions
// before. One of them that already says `stage` is not one its version wrote,
// and is refused.
//
// Version 5 added the validate stage's record (spc-2609202134338445 piece 8): a
// lane's `validation`, its rounds and the verdicts the loop recorded. Version 4
// is its strict subset, read as a run no validator has judged yet and written
// back at version 5; a version-4 file carrying a validation is not one version
// 4 wrote, and is refused.
const SchemaVersion = 5

// schemaVersionUnvalidated is the version before the validate stage's record:
// read, never written.
const schemaVersionUnvalidated = 4

// schemaVersionUnpaced is the version before the pace: read, never written.
const schemaVersionUnpaced = 1

// schemaVersionUnpicked is the version before the pick: read, never written.
const schemaVersionUnpicked = 2

// schemaVersionStepNamed is the last version that named the lane's stage
// `step`: read and migrated, never written.
const schemaVersionStepNamed = 3

// RunIDFamily is the run id's prefix; the id is minted through the record-id
// seam (adr-45), so two checkouts starting runs in one second draw distinct ids.
const RunIDFamily = "run"

// dirPerm and filePerm are the modes the run state is created with: the run is
// the caller's own business.
const (
	dirPerm  fs.FileMode = 0o700
	filePerm fs.FileMode = 0o600
)

// maxStateBytes caps a state file read. A run's record grows by one entry per
// stage; a file past this is not one the loop wrote.
const maxStateBytes = 4 << 20

// lockTimeout bounds how long a mutation waits for another invocation's
// mutation to finish before it reports contention.
const lockTimeout = 3 * time.Second

// runIDRe is the shape of a run id, checked before any path is built from one.
var runIDRe = regexp.MustCompile(`^run-[0-9]{16}$`)

// ValidRunID reports whether id is a run id this package mints.
func ValidRunID(id string) bool { return runIDRe.MatchString(id) }

// Stage is one stage of a lane: the loop performs a lane's stages in order, and
// the lane as a whole lands one step of the spec (a spec's steps keep that
// word; BU1, iss-2609291313276243).
type Stage string

// The lane's stages, in the order Sequence performs them. StageDone is the state
// of a lane with nothing left to do, never a stage with a body.
const (
	StageWorktree  Stage = "worktree"
	StageBrief     Stage = "brief"
	StageImplement Stage = "implement"
	StageValidate  Stage = "validate"
	StageLand      Stage = "land"
	StageDone      Stage = "done"
)

// Driver names what drives the loop. The host session is decision 5's default;
// the process driver is piece 3's, opt-in by configuration.
type Driver string

// DriverHost is a host session calling `implement step` and `implement receipt` itself.
const DriverHost Driver = "host"

// State is one run: everything the loop needs between two invocations.
type State struct {
	SchemaVersion int `json:"schema_version"`
	// RunID names the run and its directory.
	RunID string `json:"run_id"`
	// Key is the record the run was started for: an intent id (an issue id is
	// decision 10's, which a later piece admits).
	Key string `json:"key"`
	// Intent and Spec are the intent the run delivers and the open spec it
	// builds against, as the readiness gate judged them.
	Intent string `json:"intent"`
	Spec   string `json:"spec"`
	// Driver is what drives the loop (DriverHost unless configured).
	Driver    Driver    `json:"driver"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// WindowStartedAt and NextEligibleAt are the window clock the pacing intent
	// (itd-2609201925079472) writes and reads. The loop honours NextEligibleAt:
	// before it, a stage is refused as a pause and nothing moves (decision 2).
	WindowStartedAt *time.Time `json:"window_started_at,omitempty"`
	NextEligibleAt  *time.Time `json:"next_eligible_at,omitempty"`
	// Pace is the pace the run started on, each number with the layer that
	// supplied it (itd-2609201925079472). Nil in a run a version-1 state file
	// holds: it started before the loop paced a run, and runs unpaced.
	Pace *Pace `json:"pace,omitempty"`
	// Pick is the pick `abcd build next` made to start the run: the
	// candidates, their scores and the grounds entry its first lane commits.
	// Nil for a run `abcd build <itd-N>` started.
	Pick *RunPick `json:"pick,omitempty"`
	// Lanes are the lanes opened so far, one at a time, in order. A lane lands
	// one step of the spec's `## Steps` (the whole spec when it lists none).
	Lanes []Lane `json:"lanes"`
	// Pending are the spec's unlanded steps no lane has been opened for yet.
	Pending []PendingStep `json:"pending"`
	// Record is the run record, accumulated as stages complete.
	Record []Entry `json:"record"`
}

// PendingStep is a spec step the run will open a lane for.
type PendingStep struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
}

// Lane is one lane: one spec step, one branch, one pull request.
type Lane struct {
	// ID is the lane's name inside the run: lane-1, lane-2, ….
	ID string `json:"id"`
	// Key is the record the lane delivers (itd-N; iss-N once decision 10 lands).
	Key string `json:"key"`
	// SpecStep is the number of the spec step the lane lands, and StepTitle its
	// title. An unstepped spec is one implicit step, number 1.
	SpecStep  int    `json:"spec_step"`
	StepTitle string `json:"step_title"`
	// Stage is the next stage the loop performs for this lane; StageDone when the
	// lane has nothing left.
	Stage Stage `json:"stage"`
	// Awaiting is set while the lane waits on an agent: the stage handed its
	// work out and advances only on the receipt it names (criterion 8).
	Awaiting *Await `json:"awaiting,omitempty"`
	// The lane's footprint, filled by the stages that make it.
	Branch   string `json:"branch,omitempty"`
	BaseSHA  string `json:"base_sha,omitempty"`
	HeadSHA  string `json:"head_sha,omitempty"`
	Worktree string `json:"worktree,omitempty"`
	Brief    string `json:"brief,omitempty"`
	Receipt  string `json:"receipt,omitempty"`
	PR       int    `json:"pr,omitempty"`
	// PickSHA is the record-only commit at the branch base carrying the run's
	// pick entry, on the lane a picked run commits it on; the receipt verifier
	// counts the implementer's commits from after it.
	PickSHA string `json:"pick_sha,omitempty"`
	// Validation is the validate stage's record (piece 8): one round per head
	// the validators judged, the last the current one. Only the loop writes a
	// verdict into it, parsed from the validator's own return (itd-58).
	Validation []ValidationRound `json:"validation,omitempty"`
}

// ValidationRound is one round of the validate stage: the validators it hands
// the lane's head to, one at a time, and the fresh implementer it hands their
// findings to when one of them did not pass.
type ValidationRound struct {
	Round int `json:"round"`
	// HeadSHA is the lane's head the round's validators judge.
	HeadSHA    string         `json:"head_sha"`
	Validators []ValidatorRun `json:"validators"`
	// Fix is the verified receipt of the fresh implementer the round's
	// findings were handed to; once set, the next step opens the next round.
	Fix string `json:"fix,omitempty"`
}

// ValidatorRun is one validator of a round: the fresh agent handed the lane,
// its brief, the return it writes, and the verdict the loop parsed from that
// return.
type ValidatorRun struct {
	Role   string `json:"role"`
	Brief  string `json:"brief"`
	Return string `json:"return"`
	// Verdict is the verdict the loop parsed from the return; empty until the
	// return is handed back. Pass is whether it lets the lane advance.
	Verdict string `json:"verdict,omitempty"`
	Pass    bool   `json:"pass"`
	// Audit is the fidelity audit's request and reading, on the
	// intent-auditor's run.
	Audit *AuditRun `json:"audit,omitempty"`
}

// AuditRun is the fidelity audit the lane that closes the spec takes, once, over
// the whole delivery (ruling AI): the receipt the close parks for the same
// record, the request, the range it reads, and what the loop read from the
// verdict. The verdict itself is the return the run names; the close consumes
// it (piece 9).
type AuditRun struct {
	ReceiptID string `json:"receipt_id"`
	Request   string `json:"request"`
	// BaseSHA..HeadSHA is the whole delivery: from the base of the run's first
	// lane to this lane's head.
	BaseSHA string `json:"base_sha"`
	HeadSHA string `json:"head_sha"`
	// Worst, NotMet and Inconclusive are read from the verdict.
	Worst        string   `json:"worst,omitempty"`
	NotMet       []string `json:"not_met,omitempty"`
	Inconclusive []string `json:"inconclusive,omitempty"`
}

// Await is what a lane waits on: the agent a host must start, the brief it is
// handed and the receipt it writes.
type Await struct {
	// Role is the agent the host starts: an implementer, or a validator that
	// did not implement.
	Role string `json:"role"`
	// Brief is the file the agent is handed.
	Brief string `json:"brief"`
	// Receipt is where the agent writes its receipt; `implement receipt` is
	// called with this path.
	Receipt string    `json:"receipt"`
	Since   time.Time `json:"since"`
}

// Entry is one line of the run record.
type Entry struct {
	At   time.Time `json:"at"`
	Lane string    `json:"lane,omitempty"`
	// Stage is the stage the entry records: a lane stage, "start", "open" (a
	// later lane opened for its spec step) or "receipt".
	Stage string `json:"stage"`
	Note  string `json:"note,omitempty"`
}

// Complete reports whether the run has nothing left: every lane is done and no
// spec step is waiting for one.
func (s State) Complete() bool {
	if len(s.Pending) > 0 {
		return false
	}
	for _, l := range s.Lanes {
		if l.Stage != StageDone {
			return false
		}
	}
	return true
}

// picked reports whether the state carries anything only a pick writes.
func (s State) picked() bool {
	if s.Pick != nil {
		return true
	}
	for _, l := range s.Lanes {
		if l.PickSHA != "" {
			return true
		}
	}
	return false
}

// validated reports whether the state carries anything only the validate
// stage writes.
func (s State) validated() bool {
	for _, l := range s.Lanes {
		if len(l.Validation) > 0 {
			return true
		}
	}
	return false
}

// current returns the index of the lane the loop works on — the first lane not
// done — or -1 when every opened lane is done.
func (s State) current() int {
	for i, l := range s.Lanes {
		if l.Stage != StageDone {
			return i
		}
	}
	return -1
}

// runRel is a run's directory, relative to the checkout root.
func runRel(runID string) string { return RunRelDir + "/" + runID }

// StateRelPath is a run's state file, relative to the checkout root.
func StateRelPath(runID string) string { return runRel(runID) + "/" + StateFileName }

// ReadState is the state file's one reader. It refuses a run id of the wrong
// shape before building a path from it, reads through the checkout's os.Root
// (so a symlinked component cannot walk the read out of the checkout), decodes
// strictly — an unknown field is a file some other writer produced — and holds
// the file to the version and the id it is stored under.
func ReadState(repoRoot, runID string) (State, error) {
	if !ValidRunID(runID) {
		return State{}, refuse("state", "", "", fmt.Sprintf("%q is not a run id (run-<16 digits>)", runID),
			"name a run `abcd implement status` lists")
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return State{}, fmt.Errorf("opening the checkout to read the run state: %w", err)
	}
	defer root.Close()
	return readStateIn(root, runID)
}

func readStateIn(root *os.Root, runID string) (State, error) {
	rel := StateRelPath(runID)
	data, err := fsutil.ReadGuardedInRoot(root, rel, maxStateBytes)
	if errors.Is(err, fs.ErrNotExist) {
		return State{}, refuse("state", "", "", "no run "+runID+" in this checkout",
			"name a run `abcd implement status` lists, or start one with `abcd build <itd-N>`")
	}
	if err != nil {
		// A symlinked file or run directory, or one the filesystem will not
		// hand over, fails closed in the refusal shape: it is not a file the
		// loop wrote.
		return State{}, refuse("state", "", "", fmt.Sprintf("%s cannot be read as the run's state: %v", rel, err),
			"the loop writes a regular file in a real directory; restore that, or remove the run directory "+runRel(runID))
	}
	// One strict decode, the lane receipt's: a repeated key, a field State
	// does not name and a second document are each refused (iss-2609281204381700).
	// A file of a version that named the lane's stage `step` is decoded as
	// strictly in its own shape and migrated; the version is peeked first, and
	// the peek decides only which shape the strict decode holds the file to.
	st, err := decodeState(data)
	if err != nil {
		var sr *Refusal
		if errors.As(err, &sr) {
			sr.Reason = rel + " " + sr.Reason
			sr.Remedy = "the loop is the file's only writer; restore it or remove the run directory " + runRel(runID)
			return State{}, sr
		}
		return State{}, refuse("state", "", "", fmt.Sprintf("%s does not parse as a run state: %v", rel, err),
			"the loop is the file's only writer; restore it or remove the run directory "+runRel(runID))
	}
	switch {
	case st.SchemaVersion == schemaVersionUnpaced && st.Pace != nil:
		return State{}, refuse("state", "", "", fmt.Sprintf("%s is schema version %d but carries a pace, which version %d never wrote", rel, st.SchemaVersion, schemaVersionUnpaced),
			"the loop is the file's only writer; restore it or remove the run directory "+runRel(runID))
	case (st.SchemaVersion == schemaVersionUnpaced || st.SchemaVersion == schemaVersionUnpicked) && st.picked():
		return State{}, refuse("state", "", "", fmt.Sprintf("%s is schema version %d but carries a pick, which version %d never wrote", rel, st.SchemaVersion, st.SchemaVersion),
			"the loop is the file's only writer; restore it or remove the run directory "+runRel(runID))
	case st.SchemaVersion <= schemaVersionUnvalidated && st.validated():
		return State{}, refuse("state", "", "", fmt.Sprintf("%s is schema version %d but carries a validation, which version %d never wrote", rel, st.SchemaVersion, st.SchemaVersion),
			"the loop is the file's only writer; restore it or remove the run directory "+runRel(runID))
	case st.SchemaVersion >= schemaVersionUnpaced && st.SchemaVersion <= schemaVersionUnvalidated:
		// Read as the current version, its stages already carried over by
		// decodeState when it named them `step`; the next write carries it, and
		// this read writes nothing.
		st.SchemaVersion = SchemaVersion
	case st.SchemaVersion != SchemaVersion:
		return State{}, refuse("state", "", "", fmt.Sprintf("%s is schema version %d; this abcd reads versions %d to %d", rel, st.SchemaVersion, schemaVersionUnpaced, SchemaVersion),
			"run the abcd that wrote it")
	}
	if st.RunID != runID {
		return State{}, refuse("state", "", "", fmt.Sprintf("%s names run %q, not the run it is stored under", rel, st.RunID),
			"the loop is the file's only writer; restore it or remove the run directory "+runRel(runID))
	}
	return st, nil
}

// stateStepNamed is a state file of versions 1 to 3, which named the lane's
// stage `step`: State with its lanes and its record in that shape. The outer
// fields shadow the embedded ones of the same name, so a strict decode into it
// refuses a field none of those versions wrote, exactly as one into State does.
type stateStepNamed struct {
	State
	Lanes  []laneStepNamed  `json:"lanes"`
	Record []entryStepNamed `json:"record"`
}

// laneStepNamed is a lane in a file of versions 1 to 3.
type laneStepNamed struct {
	Lane
	Step Stage `json:"step"`
}

// entryStepNamed is a record line in a file of versions 1 to 3.
type entryStepNamed struct {
	Entry
	Step string `json:"step"`
}

// decodeState decodes a state file strictly in the shape its version wrote:
// the current shape, or, for versions 1 to 3, the shape that named the lane's
// stage `step`, carried over to `stage` here. A file of one of those versions
// that already says `stage` is not one its version wrote, and is refused. The
// version the file states is returned as it was; the caller holds it to what
// the version could have written.
func decodeState(data []byte) (State, error) {
	var peek struct {
		SchemaVersion int `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &peek); err != nil || peek.SchemaVersion < schemaVersionUnpaced || peek.SchemaVersion > schemaVersionStepNamed {
		var st State
		err := jsonstrict.Decode(data, &st)
		return st, err
	}
	var old stateStepNamed
	if err := jsonstrict.Decode(data, &old); err != nil {
		return State{}, err
	}
	st := old.State
	neverWrote := func(what string) error {
		return refuse("state", "", "", fmt.Sprintf("is schema version %d but %s says `stage`, which version %d never wrote", old.SchemaVersion, what, old.SchemaVersion), "")
	}
	st.Lanes = nil
	if old.Lanes != nil {
		st.Lanes = make([]Lane, 0, len(old.Lanes))
	}
	for _, l := range old.Lanes {
		if l.Lane.Stage != "" {
			return State{}, neverWrote("lane " + l.ID)
		}
		lane := l.Lane
		lane.Stage = l.Step
		st.Lanes = append(st.Lanes, lane)
	}
	st.Record = nil
	if old.Record != nil {
		st.Record = make([]Entry, 0, len(old.Record))
	}
	for i, e := range old.Record {
		if e.Entry.Stage != "" {
			return State{}, neverWrote(fmt.Sprintf("record line %d", i+1))
		}
		entry := e.Entry
		entry.Stage = e.Step
		st.Record = append(st.Record, entry)
	}
	return st, nil
}

// writeState is the state file's one writer: a whole-file atomic replacement
// inside the checkout's os.Root, so a reader sees the old state or the new one,
// never half of either. The caller holds the lock.
func writeState(root *os.Root, st State) error {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	rel := StateRelPath(st.RunID)
	if err := fsutil.WriteFileAtomicInRoot(root, rel, data, filePerm); err != nil {
		return fmt.Errorf("writing %s: %w", rel, err)
	}
	return nil
}

// runIDs lists the run directories under the run tier, in name order (which is
// mint order). An absent tier holds none.
func runIDs(root *os.Root) ([]string, error) {
	f, err := root.Open(RunRelDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, refuse("state", "", "", fmt.Sprintf("%s cannot be listed: %v", RunRelDir, err),
			"the loop creates it as a real directory; restore that, or remove it")
	}
	defer f.Close()
	names, err := f.Readdirnames(-1)
	if err != nil {
		return nil, fmt.Errorf("listing %s: %w", RunRelDir, err)
	}
	var ids []string
	for _, n := range names {
		if ValidRunID(n) {
			ids = append(ids, n)
		}
	}
	slices.Sort(ids)
	return ids, nil
}
