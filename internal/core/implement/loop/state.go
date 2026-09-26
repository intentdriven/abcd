// Package loop is the implement loop's engine (itd-2609201916151817,
// spc-2609202134338445): the state file a run lives in, the checks that decide
// whether a run may start, and the step interface a driving host calls. `abcd
// build <itd-N>` starts a run; `abcd implement step` performs the next step and
// exits; `abcd implement receipt <path>` hands back what an agent step waited
// on; `abcd implement status` renders the state. Nothing here holds a run in
// memory between invocations: every call reads the state file first, does at
// most one step, writes the state file last and returns (decisions 1 and 2), so
// a killed process loses only the step it was in, which the next call repeats.
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
// The lane's steps are a sequence (Sequence), and each step's body is a
// Handler the piece of the spec that delivers it registers in DefaultSteps: the
// worktree (piece 6, lane.go), the brief (piece 5, brief.go), the implement
// step and its receipt's verifier (piece 7, receipt.go), the validators (piece
// 8) and the landing (piece 9). A step whose body this build does not carry is
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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"slices"
	"time"

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

// SchemaVersion is the state file's shape. A file carrying any other version is
// refused rather than read as this one: a field a later build added and this
// one would drop on its next write is a run silently losing state.
const SchemaVersion = 1

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
// step; a file past this is not one the loop wrote.
const maxStateBytes = 4 << 20

// lockTimeout bounds how long a mutation waits for another invocation's
// mutation to finish before it reports contention.
const lockTimeout = 3 * time.Second

// runIDRe is the shape of a run id, checked before any path is built from one.
var runIDRe = regexp.MustCompile(`^run-[0-9]{16}$`)

// ValidRunID reports whether id is a run id this package mints.
func ValidRunID(id string) bool { return runIDRe.MatchString(id) }

// StepName is one step of a lane.
type StepName string

// The lane's steps, in the order Sequence performs them. StepDone is the state
// of a lane with nothing left to do, never a step with a body.
const (
	StepWorktree  StepName = "worktree"
	StepBrief     StepName = "brief"
	StepImplement StepName = "implement"
	StepValidate  StepName = "validate"
	StepLand      StepName = "land"
	StepDone      StepName = "done"
)

// Driver names what drives the loop. The host session is decision 5's default;
// the process driver is piece 3's, opt-in by configuration.
type Driver string

// DriverHost is a host session calling step and receipt itself.
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
	// before it, a step is refused as a pause and nothing moves (decision 2).
	WindowStartedAt *time.Time `json:"window_started_at,omitempty"`
	NextEligibleAt  *time.Time `json:"next_eligible_at,omitempty"`
	// Lanes are the lanes opened so far, one at a time, in order. A lane lands
	// one step of the spec's `## Steps` (the whole spec when it lists none).
	Lanes []Lane `json:"lanes"`
	// Pending are the spec's unlanded steps no lane has been opened for yet.
	Pending []PendingStep `json:"pending"`
	// Record is the run record, accumulated as steps complete.
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
	// Step is the next step the loop performs for this lane; StepDone when the
	// lane has nothing left.
	Step StepName `json:"step"`
	// Awaiting is set while the lane waits on an agent: the step handed its
	// work out and advances only on the receipt it names (criterion 8).
	Awaiting *Await `json:"awaiting,omitempty"`
	// The lane's footprint, filled by the steps that make it.
	Branch   string `json:"branch,omitempty"`
	BaseSHA  string `json:"base_sha,omitempty"`
	HeadSHA  string `json:"head_sha,omitempty"`
	Worktree string `json:"worktree,omitempty"`
	Brief    string `json:"brief,omitempty"`
	Receipt  string `json:"receipt,omitempty"`
	PR       int    `json:"pr,omitempty"`
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
	// Step is the step the entry records: a lane step, "start" or "receipt".
	Step string `json:"step"`
	Note string `json:"note,omitempty"`
}

// Complete reports whether the run has nothing left: every lane is done and no
// spec step is waiting for one.
func (s State) Complete() bool {
	if len(s.Pending) > 0 {
		return false
	}
	for _, l := range s.Lanes {
		if l.Step != StepDone {
			return false
		}
	}
	return true
}

// current returns the index of the lane the loop works on — the first lane not
// done — or -1 when every opened lane is done.
func (s State) current() int {
	for i, l := range s.Lanes {
		if l.Step != StepDone {
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
	var st State
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&st); err != nil {
		return State{}, refuse("state", "", "", fmt.Sprintf("%s does not parse as a run state: %v", rel, err),
			"the loop is the file's only writer; restore it or remove the run directory "+runRel(runID))
	}
	if st.SchemaVersion != SchemaVersion {
		return State{}, refuse("state", "", "", fmt.Sprintf("%s is schema version %d; this abcd reads version %d", rel, st.SchemaVersion, SchemaVersion),
			"run the abcd that wrote it")
	}
	if st.RunID != runID {
		return State{}, refuse("state", "", "", fmt.Sprintf("%s names run %q, not the run it is stored under", rel, st.RunID),
			"the loop is the file's only writer; restore it or remove the run directory "+runRel(runID))
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
