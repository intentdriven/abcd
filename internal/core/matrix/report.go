package matrix

// report.go is the run's report (spc-2610161949559050 step 8; decisions 16,
// 26 and 28): one row set per task kind, read from the run's own records in
// its lab, never from anything a copy or a judge wrote straight.
//
//   - a code task: each builder cell's pass@1 on its hidden tests with its
//     interval, its pass^k, what the reviewers caught of its copies, its cost
//     per passing build stated by source with what it is comparable with, and
//     its model time apart from the time spent waiting on gates; and each
//     pair of builders compared, ranked only at the repeats its power
//     estimate asks for, otherwise stating the smallest difference it could
//     have detected;
//   - a written doc: each model's rating from all the judges' rankings with
//     its interval and whether it is separable, how often its copies were
//     good enough, the cost per good-enough copy, the judges' agreement and
//     each copy's length; with fewer than three judges, each judge's ranking
//     and the good-enough counts only;
//   - a check: the table the person's locked verdicts unlocked.
//
// Every report lists the run's limitations (what it could not pin, examine or
// detect at its repeat count) and the improvements it suggests, so the next
// run starts from them; a re-run's report shows, field by field, where it
// drifted from the run it repeats (rerun.go).

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/core/oracle"
)

// ReportSchema is the report's schema version.
const ReportSchema = 1

// Report is a run's report.
type Report struct {
	SchemaVersion int           `json:"schema_version"`
	Run           string        `json:"run"`
	Lab           string        `json:"lab"`
	FrozenCommit  string        `json:"frozen_commit"`
	Trials        int           `json:"trials"`
	Code          []CodeReport  `json:"code"`
	Writing       []WritingRow  `json:"writing"`
	Check         []CheckReport `json:"check"`
	// Drift is where a re-run drifted from the run it repeats, nil for a run
	// that repeats none.
	Drift *Drift `json:"drift"`
	// Unreturned is every copy the run handed off, or its pilot asked for
	// once it began handing copies off, that holds no trial, by task, cell and
	// where it stands. None of them is in any rate.
	Unreturned   []UnreturnedCopies `json:"unreturned"`
	Limitations  []string           `json:"limitations"`
	Improvements []string           `json:"improvements"`
}

// CodeReport is a code task's rows.
type CodeReport struct {
	Task        string          `json:"task"`
	Intent      string          `json:"intent"`
	Cells       []CodeCell      `json:"cells"`
	Comparisons []Comparison    `json:"comparisons"`
	Reviewers   []ReviewerScore `json:"reviewers"`
}

// CodeCell is one builder cell's row of a code task.
type CodeCell struct {
	Cell   string `json:"cell"`
	Model  string `json:"model"`
	Source Source `json:"source"`
	Runner string `json:"runner"`
	// PassAt1 counts the passes and the failures; a copy never started, or
	// one set apart by the compliance check, is in neither.
	PassAt1 PassRate `json:"pass_at_1"`
	// HostReported counts the attempts in PassAt1 whose hidden tests' result
	// is the host's report, which abcd neither placed nor ran.
	HostReported int       `json:"host_reported"`
	PassHatK     []PassHat `json:"pass_hat_k"`
	NotStarted   int       `json:"not_started"`
	Excluded     int       `json:"excluded"`
	Unexamined   int       `json:"unexamined"`
	// NotHandedBack, AwaitingRunner, Launched, Refused and NeverHandedOff
	// count the copies with no trial, by where they stand; none is in any
	// rate. Launched counts a runner cell's copies whose launch began and
	// wrote no trial: at work, stopped, or made and awaiting their hidden
	// tests' result.
	NotHandedBack  int         `json:"not_handed_back"`
	AwaitingRunner int         `json:"awaiting_runner"`
	Launched       int         `json:"launched"`
	Refused        int         `json:"refused"`
	NeverHandedOff int         `json:"never_handed_off"`
	Reviews        CellReviews `json:"reviews"`
	Costs          []CellCost  `json:"costs"`
	// ModelTimeMS and GateWaitMS are each the mean over the attempts: the
	// time the models spent, and apart from it the time spent waiting on
	// gates.
	ModelTimeMS int64 `json:"model_time_ms"`
	GateWaitMS  int64 `json:"gate_wait_ms"`
}

// UnreturnedState is where a copy with no trial stands.
type UnreturnedState string

const (
	// UnreturnedNotHandedBack: handed to the host and never handed back.
	UnreturnedNotHandedBack UnreturnedState = "not_handed_back"
	// UnreturnedAwaitingRunner: a runner cell's copy, waiting for its launch.
	UnreturnedAwaitingRunner UnreturnedState = "awaiting_runner"
	// UnreturnedLaunching: a runner cell's copy whose launch began and has
	// not ended, or was stopped before it could say so.
	UnreturnedLaunching UnreturnedState = "launching"
	// UnreturnedInterrupted: a runner cell's copy whose launch was stopped
	// before the copy was made; it is not launched again.
	UnreturnedInterrupted UnreturnedState = "interrupted"
	// UnreturnedAwaitingScore: a code copy its runner made, waiting for its
	// hidden tests' result.
	UnreturnedAwaitingScore UnreturnedState = "awaiting_score"
	// UnreturnedRefused: its cell's clone was refused before the copy started.
	UnreturnedRefused UnreturnedState = "refused"
	// UnreturnedNeverHandedOff: a pilot copy the run's start never reached.
	UnreturnedNeverHandedOff UnreturnedState = "never_handed_off"
)

// UnreturnedCopies is the copies of one task on one builder cell that stand
// alike and hold no trial: their repeats, and for a refused cell, why.
type UnreturnedCopies struct {
	Task    string          `json:"task"`
	Cell    string          `json:"cell"`
	State   UnreturnedState `json:"state"`
	Repeats []int           `json:"repeats"`
	Reason  string          `json:"reason,omitempty"`
}

// CellReviews is what the reviewers caught of one builder's copies: a flag on
// a copy its hidden tests failed is a confirmed finding, and a flag on one
// they passed is unconfirmed.
type CellReviews struct {
	Confirmed   int `json:"confirmed"`
	Unconfirmed int `json:"unconfirmed"`
}

// CellCost is a cell's spend in one unit: the total, the cost per passing (or
// good-enough) copy, nil where none passed or nothing metered was spent, and
// what the figure is comparable with.
type CellCost struct {
	Unit       string   `json:"unit"`
	Total      float64  `json:"total"`
	PerPass    *float64 `json:"per_pass"`
	Comparable string   `json:"comparable"`
}

// Comparison is two builder cells' pass rates set against each other.
type Comparison struct {
	A string `json:"a"`
	B string `json:"b"`
	// Kind is "model" within one source and "model and harness" across
	// sources, since model and harness travel together by source (decision
	// 10).
	Kind       string   `json:"kind"`
	Difference *float64 `json:"difference"`
	RepeatsA   int      `json:"repeats_a"`
	RepeatsB   int      `json:"repeats_b"`
	// MinDetectable is the smallest difference the repeats could have
	// detected; Needed the repeats each cell needs to detect the declared
	// smallest difference, nought where none was declared.
	MinDetectable float64 `json:"min_detectable"`
	Needed        int     `json:"needed"`
	Ranked        bool    `json:"ranked"`
	// HostReported is whether either cell's pass rate rests on hidden
	// tests' results the host reported; its Note then says so, so no
	// ranking on them is shown unqualified.
	HostReported bool `json:"host_reported"`
	// Ahead is the cell whose pass rate is ahead where the comparison is
	// ranked and the difference is past chance; "" otherwise.
	Ahead string `json:"ahead"`
	Note  string `json:"note"`
}

// WritingRow is a written doc's rows.
type WritingRow struct {
	Task            string `json:"task"`
	Judged          bool   `json:"judged"`
	Members         int    `json:"members"`
	Guests          int    `json:"guests"`
	HostSelfBlinded int    `json:"host_self_blinded"`
	// Floor says why the members' rankings are not pooled, "" where they are.
	Floor      string                    `json:"floor,omitempty"`
	Ratings    []BTRating                `json:"ratings"`
	Agreement  *WritingAgreement         `json:"agreement"`
	GoodEnough []GoodEnoughRow           `json:"good_enough"`
	Copies     []CopyLength              `json:"copies"`
	Rankings   []JudgeRanking            `json:"rankings,omitempty"`
	Apart      map[string][]JudgeRanking `json:"apart,omitempty"`
}

// WritingAgreement is the members' agreement on the good-enough marks and on
// the order of the copies.
type WritingAgreement struct {
	GoodEnough Agreement `json:"good_enough"`
	Order      Agreement `json:"order"`
}

// GoodEnoughRow is one model's good-enough marks among the members, its
// copies the members found good enough (more than half of their marks), and
// its cost per good-enough copy.
type GoodEnoughRow struct {
	Cell        string     `json:"cell"`
	Good        int        `json:"good"`
	Of          int        `json:"of"`
	GoodCopies  int        `json:"good_copies"`
	Costs       []CellCost `json:"costs"`
	ModelTimeMS int64      `json:"model_time_ms"`
}

// CopyLength is one copy's length beside its place: the mean of the places
// the members ranked it at, counted from one, since length sways people more
// than any other style signal (decision 21).
type CopyLength struct {
	Key       TrialKey `json:"key"`
	Words     int      `json:"words"`
	MeanPlace float64  `json:"mean_place"`
}

// CheckReport is a check task's rows: its second phase, nil until the
// person's verdicts are locked and the table recorded.
type CheckReport struct {
	Task  string      `json:"task"`
	Table *CheckTable `json:"table"`
}

// reviewRecord is a code task's reviews as the run records them: the copies
// the reviewers were asked about, under their labels, and every verdict.
type reviewRecord struct {
	SchemaVersion int             `json:"schema_version"`
	Run           string          `json:"run"`
	Task          string          `json:"task"`
	Copies        []ReviewedCopy  `json:"copies"`
	Verdicts      []ReviewVerdict `json:"verdicts"`
}

// reviewRel is task's reviews in its run's directory.
func reviewRel(run, task string) string { return run + "/review/" + task + ".json" }

// recordReviews records task's reviews beside the run's record, once, after
// scoring them, so a set of verdicts the scoring refuses is never recorded,
// and passed through the secret screen, since an author guess is a model's
// text.
func recordReviews(repoRoot, labID, run, task string, copies []ReviewedCopy, verdicts []ReviewVerdict) error {
	if !runIDRe.MatchString(run) || !validID(task) {
		return fmt.Errorf("%q and %q are not a run and a task", clip(run), clip(task))
	}
	proc, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return err
	}
	defer proc.Close()
	rec, _, err := readRun(proc, run)
	if err != nil {
		return err
	}
	for _, c := range copies {
		if c.Key.Task != task {
			return fmt.Errorf("the reviewed copy %s is not of the task %s", c.Key, task)
		}
	}
	if _, err := scoreReviewers(copies, rec.Cells, verdicts); err != nil {
		return fmt.Errorf("the reviews are not recorded: %w", err)
	}
	data, err := encodeCheck(reviewRecord{SchemaVersion: ReportSchema, Run: run, Task: task, Copies: copies, Verdicts: verdicts})
	if err != nil {
		return err
	}
	rel := reviewRel(run, task)
	if err := screen(repoRoot, []screened{{lab.StateFile{Rel: rel, Data: data}, data}}); err != nil {
		return err
	}
	return proc.WriteState(rel, data)
}

// writeTrial writes one copy's trial record beside its run's record, once,
// validated and passed through the secret screen first, as the run record is
// (spec step 2): a trial's reason and score detail carry what a runner said.
func writeTrial(repoRoot, labID string, t TrialRecord) error {
	if err := t.validate(); err != nil {
		return fmt.Errorf("the trial is not written: %w", err)
	}
	proc, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return err
	}
	defer proc.Close()
	rec, _, err := readRun(proc, t.Run)
	if err != nil {
		return err
	}
	if t.FrozenCommit != rec.FrozenCommit {
		return fmt.Errorf("the trial %s starts from %s, not its run's frozen commit %s", t.Key, t.FrozenCommit, rec.FrozenCommit)
	}
	data, err := screenTrial(repoRoot, t)
	if err != nil {
		return err
	}
	return proc.WriteState(t.Run+"/"+t.Key.rel(), data)
}

// screenTrial is trial t as written, passed through the secret screen: the
// network screen reads it as a run's record is read, with its harness version
// emptied only when it is a dotted number and nothing else (asVersion), so
// 2.10.3.4 is not read as an address, and any other value read as written.
func screenTrial(repoRoot string, t TrialRecord) ([]byte, error) {
	data, network, err := encodeTrial(t)
	if err != nil {
		return nil, err
	}
	rel := t.Run + "/" + t.Key.rel()
	if err := screen(repoRoot, []screened{{lab.StateFile{Rel: rel, Data: data}, network}}); err != nil {
		return nil, err
	}
	return data, nil
}

// encodeTrial is trial t as written, and as the network screen reads it.
func encodeTrial(t TrialRecord) (data, network []byte, err error) {
	if data, err = encodeCheck(t); err != nil {
		return nil, nil, err
	}
	if network, err = encodeCheck(t.withoutVersions()); err != nil {
		return nil, nil, err
	}
	return data, network, nil
}

// withoutVersions is the trial as the network screen reads it, as
// withoutVersionsAndDigests is a run's record: its harness version withheld
// only when it is a dotted number and nothing else (asVersion), and nothing
// else changed, so it encodes to the trial's own lines and any other harness
// version a host reports is screened as written.
func (t TrialRecord) withoutVersions() TrialRecord {
	t.HarnessVersion = asVersion(t.HarnessVersion)
	return t
}

// readRun is run's record and declaration as its lab holds them, the two
// checked against each other.
func readRun(proc stateReader, run string) (RunRecord, Declaration, error) {
	raw, err := proc.ReadState(run + "/run.json")
	if err != nil {
		return RunRecord{}, Declaration{}, fmt.Errorf("the run %s is not recorded in this lab: %w", run, err)
	}
	var rec RunRecord
	if err := jsonstrict.Decode(raw, &rec); err != nil || rec.RunID != run {
		return RunRecord{}, Declaration{}, fmt.Errorf("the run %s's record does not read as its own", run)
	}
	decl, err := proc.ReadState(run + "/declaration.json")
	if err != nil || contentOf("declaration.json", decl) != rec.Declaration {
		return RunRecord{}, Declaration{}, fmt.Errorf("the run %s's declaration does not read as the one its record keeps", run)
	}
	d, err := parseDeclaration(decl)
	if err != nil {
		return RunRecord{}, Declaration{}, err
	}
	return rec, d, nil
}

// stateReader reads a run's records: its lab's procedure, or a run's folder
// carried from elsewhere (rerun.go).
type stateReader interface {
	ReadState(rel string) ([]byte, error)
	ListState(rel string) ([]string, error)
}

// readTrials is every trial record under dir (a run's trials/ folder) in the
// lab, each validated and its run's, in key order. A folder never written
// holds none.
func readTrials(proc stateReader, run, dir string) ([]TrialRecord, error) {
	list := func(rel string) ([]string, error) {
		names, err := proc.ListState(rel)
		if len(names) > MaxRepeats+maxTasks+maxCells {
			return nil, fmt.Errorf("%s holds %d entries, more than a run writes", rel, len(names))
		}
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return names, err
	}
	tasks, err := list(dir)
	if err != nil {
		return nil, err
	}
	var out []TrialRecord
	for _, task := range tasks {
		cells, err := list(dir + "/" + task)
		if err != nil {
			return nil, err
		}
		for _, cell := range cells {
			files, err := list(dir + "/" + task + "/" + cell)
			if err != nil {
				return nil, err
			}
			for _, f := range files {
				raw, err := proc.ReadState(dir + "/" + task + "/" + cell + "/" + f)
				if err != nil {
					return nil, err
				}
				var t TrialRecord
				if err := jsonstrict.Decode(raw, &t); err != nil {
					return nil, fmt.Errorf("the trial %s/%s/%s does not parse: %w", task, cell, f, err)
				}
				if err := t.validate(); err != nil {
					return nil, err
				}
				if t.Run != run || t.Key.Task != task || t.Key.Cell != cell || strconv.Itoa(t.Key.Repeat)+".json" != f {
					return nil, fmt.Errorf("the trial at %s/%s/%s is another copy's", task, cell, f)
				}
				out = append(out, t)
			}
		}
	}
	sortTrials(out)
	return out, nil
}

func sortTrials(ts []TrialRecord) {
	sort.Slice(ts, func(i, j int) bool {
		a, b := ts[i].Key, ts[j].Key
		if a.Task != b.Task {
			return a.Task < b.Task
		}
		if a.Cell != b.Cell {
			return a.Cell < b.Cell
		}
		return a.Repeat < b.Repeat
	})
}

// BuildReport builds run's report from its records in the lab labID of the
// repository at repoRoot. It reads and writes nothing else.
func BuildReport(repoRoot, labID, run string) (Report, error) {
	if !runIDRe.MatchString(run) {
		return Report{}, fmt.Errorf("%q is not a run id (mx-<yymmddHHMMSS>-<4 hex>)", clip(run))
	}
	proc, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return Report{}, err
	}
	defer proc.Close()
	rec, d, err := readRun(proc, run)
	if err != nil {
		return Report{}, err
	}
	trials, err := readTrials(proc, run, run+"/trials")
	if err != nil {
		return Report{}, err
	}
	unreturned, err := accountCopies(proc, rec, d, trials)
	if err != nil {
		return Report{}, err
	}
	b := reportBuilder{rec: rec, decl: d, trials: trials, unreturned: unreturned}
	rep := Report{SchemaVersion: ReportSchema, Run: run, Lab: rec.Lab, FrozenCommit: rec.FrozenCommit, Trials: len(trials),
		Code: []CodeReport{}, Writing: []WritingRow{}, Check: []CheckReport{}, Unreturned: unreturned}
	if rep.Unreturned == nil {
		rep.Unreturned = []UnreturnedCopies{}
	}
	for _, task := range d.Tasks {
		switch task.Kind {
		case KindCode:
			var rr *reviewRecord
			if raw, err := proc.ReadState(reviewRel(run, task.ID)); err == nil {
				var r reviewRecord
				if err := jsonstrict.Decode(raw, &r); err != nil || r.Run != run || r.Task != task.ID {
					return Report{}, fmt.Errorf("the reviews of %s do not read as its own", task.ID)
				}
				rr = &r
			} else if !errors.Is(err, fs.ErrNotExist) {
				return Report{}, err
			}
			cr, err := b.code(task, rr)
			if err != nil {
				return Report{}, err
			}
			rep.Code = append(rep.Code, cr)
		case KindWriting:
			wr, err := b.writing(proc, task)
			if err != nil {
				return Report{}, err
			}
			rep.Writing = append(rep.Writing, wr)
		case KindCheck:
			cr := CheckReport{Task: task.ID}
			if raw, err := proc.ReadState(checkRel(run, task.ID, "table.json")); err == nil {
				var t CheckTable
				if err := jsonstrict.Decode(raw, &t); err != nil || t.Run != run || t.Task != task.ID {
					return Report{}, fmt.Errorf("the check table of %s does not read as its own", task.ID)
				}
				cr.Table = &t
			} else if !errors.Is(err, fs.ErrNotExist) {
				return Report{}, err
			}
			if cr.Table == nil {
				b.limit("the check %s is not judged yet: no locked verdicts have unlocked its comparison of findings, so its precision, unique valid findings and recall are not reported", task.ID)
			} else if cr.Table.Matcher == nil {
				b.limit("the matcher's agreement with a human-labelled sample was not measured for %s, so the rows it joined rest on its word", task.ID)
				b.improve("label a sample of %s's matched findings by hand, so the matcher's agreement is measured", task.ID)
			}
			rep.Check = append(rep.Check, cr)
		}
	}
	if drift, err := rerunDrift(proc, rec, trials); err != nil {
		return Report{}, err
	} else if drift != nil {
		rep.Drift = drift
	}
	b.unreturnedLimits()
	b.runLimits()
	rep.Limitations, rep.Improvements = b.limits, b.improves
	return rep, nil
}

// accountCopies is every copy of run that holds no trial: each the run handed
// off, and, once it began handing copies off, each pilot copy its start never
// reached. A run no launcher started (one whose trials were written another
// way) holds no hand-off and is accounted by its trials alone.
func accountCopies(proc stateReader, rec RunRecord, d Declaration, trials []TrialRecord) ([]UnreturnedCopies, error) {
	handed, err := readHandOffs(proc, rec.RunID)
	if err != nil || len(handed) == 0 {
		return nil, err
	}
	tried := map[TrialKey]bool{}
	for _, t := range trials {
		tried[t.Key] = true
	}
	byKey := map[TrialKey]HandOff{}
	for _, h := range handed {
		byKey[h.Key] = h
	}
	var out []UnreturnedCopies
	for _, task := range d.Tasks {
		for _, cell := range d.Builders {
			repeats := map[int]bool{}
			for r := 1; r <= d.Repeats.Pilot; r++ {
				repeats[r] = true
			}
			for k := range byKey {
				if k.Task == task.ID && k.Cell == cell.ID {
					repeats[k.Repeat] = true
				}
			}
			rs := make([]int, 0, len(repeats))
			for r := range repeats {
				rs = append(rs, r)
			}
			slices.Sort(rs)
			groups := map[UnreturnedState]*UnreturnedCopies{}
			for _, r := range rs {
				k := TrialKey{Task: task.ID, Cell: cell.ID, Repeat: r}
				if tried[k] {
					continue
				}
				state, reason := UnreturnedNeverHandedOff, ""
				if h, ok := byKey[k]; ok {
					switch h.State {
					case HandOffWaiting:
						state = UnreturnedNotHandedBack
					case HandOffAwaitingRunner:
						state = UnreturnedAwaitingRunner
					case HandOffLaunching:
						state = UnreturnedLaunching
					case HandOffInterrupted:
						state, reason = UnreturnedInterrupted, h.Reason
					case HandOffLaunched:
						state = UnreturnedAwaitingScore
					default:
						state, reason = UnreturnedRefused, h.Reason
					}
				}
				g := groups[state]
				if g == nil {
					g = &UnreturnedCopies{Task: task.ID, Cell: cell.ID, State: state, Reason: reason}
					groups[state] = g
				}
				g.Repeats = append(g.Repeats, r)
			}
			for _, s := range []UnreturnedState{UnreturnedNotHandedBack, UnreturnedAwaitingRunner, UnreturnedLaunching, UnreturnedAwaitingScore,
				UnreturnedInterrupted, UnreturnedRefused, UnreturnedNeverHandedOff} {
				if g := groups[s]; g != nil {
					out = append(out, *g)
				}
			}
		}
	}
	return out, nil
}

// unreturnedInto counts task's copies on row's cell that hold no trial into it.
func (b *reportBuilder) unreturnedInto(task string, row *CodeCell) {
	for _, u := range b.unreturned {
		if u.Task != task || u.Cell != row.Cell {
			continue
		}
		switch u.State {
		case UnreturnedNotHandedBack:
			row.NotHandedBack += len(u.Repeats)
		case UnreturnedAwaitingRunner:
			row.AwaitingRunner += len(u.Repeats)
		case UnreturnedLaunching, UnreturnedInterrupted, UnreturnedAwaitingScore:
			row.Launched += len(u.Repeats)
		case UnreturnedRefused:
			row.Refused += len(u.Repeats)
		case UnreturnedNeverHandedOff:
			row.NeverHandedOff += len(u.Repeats)
		}
	}
}

// unreturnedLimits names every copy that holds no trial, by task and cell.
func (b *reportBuilder) unreturnedLimits() {
	for _, u := range b.unreturned {
		n, which := len(u.Repeats), repeatList(u.Repeats)
		switch u.State {
		case UnreturnedNotHandedBack:
			b.limit("%d copies of %s on %s (%s) were handed to the host and never handed back; they are in no rate, so the cell's figures stand on the copies returned alone", n, u.Task, u.Cell, which)
			b.improve("hand back every copy of %s on %s, a failed one with its reason, so the cell's figures count every copy the run made", u.Task, u.Cell)
		case UnreturnedAwaitingRunner:
			b.limit("%d copies of %s on %s (%s) wait for the cell's runner, which their launch starts; nothing stands in for it, so they are in no rate, review or judging", n, u.Task, u.Cell, which)
			b.improve("launch the copies of %s on %s with abcd lab matrix launch, so the cell's figures count every copy the run made", u.Task, u.Cell)
		case UnreturnedLaunching:
			b.limit("%d copies of %s on %s (%s) were launched and wrote no trial: their runner is at work, or their launch was stopped before it said so; they are in no rate", n, u.Task, u.Cell, which)
		case UnreturnedInterrupted:
			b.limit("%d copies of %s on %s (%s) were launched and stopped before they were made (%s); that is the run's error, not the copies' score, so they are in no rate", n, u.Task, u.Cell, which, u.Reason)
		case UnreturnedAwaitingScore:
			b.limit("%d copies of %s on %s (%s) were made by the cell's runner and wait for their hidden tests' result; they are in no rate", n, u.Task, u.Cell, which)
			b.improve("hand back the hidden tests' result of each copy of %s on %s that its runner made, so the cell's figures count it", u.Task, u.Cell)
		case UnreturnedRefused:
			b.limit("%d copies of %s on %s (%s) were refused before they started (%s); they are in no rate", n, u.Task, u.Cell, which, u.Reason)
		case UnreturnedNeverHandedOff:
			b.limit("%d copies of %s on %s (%s) were never handed off: the run's start stopped before them; they are in no rate", n, u.Task, u.Cell, which)
		}
	}
}

// repeatList names repeats: "repeat 4", "repeats 1, 2 and 3".
func repeatList(rs []int) string {
	if len(rs) == 1 {
		return "repeat " + strconv.Itoa(rs[0])
	}
	s := make([]string, len(rs))
	for i, r := range rs {
		s[i] = strconv.Itoa(r)
	}
	return "repeats " + strings.Join(s[:len(s)-1], ", ") + " and " + s[len(s)-1]
}

// reportBuilder gathers a report's rows and, as it goes, its limitations and
// improvements, each said once.
type reportBuilder struct {
	rec        RunRecord
	decl       Declaration
	trials     []TrialRecord
	unreturned []UnreturnedCopies
	limits     []string
	improves   []string
}

func (b *reportBuilder) limit(format string, args ...any) {
	if s := fmt.Sprintf(format, args...); !slices.Contains(b.limits, s) {
		b.limits = append(b.limits, s)
	}
}

func (b *reportBuilder) improve(format string, args ...any) {
	if s := fmt.Sprintf(format, args...); !slices.Contains(b.improves, s) {
		b.improves = append(b.improves, s)
	}
}

// pin is cell's pin, the zero pin where the run pins none.
func (b *reportBuilder) pin(cell string) CellPin {
	for _, p := range b.rec.Cells {
		if p.Cell == cell {
			return p
		}
	}
	return CellPin{Cell: cell}
}

// of is task's trials of cell.
func (b *reportBuilder) of(task, cell string) []TrialRecord {
	var out []TrialRecord
	for _, t := range b.trials {
		if t.Key.Task == task && t.Key.Cell == cell {
			out = append(out, t)
		}
	}
	return out
}

// seed is the draw a task's statistics are seeded by: the run's and the
// task's ids, so a report read twice says the same thing.
func (b *reportBuilder) seed(task string) [32]byte {
	return sha256.Sum256([]byte("abcd matrix report\x00" + b.rec.RunID + "\x00" + task))
}

// code builds a code task's rows.
func (b *reportBuilder) code(task Task, rr *reviewRecord) (CodeReport, error) {
	cr := CodeReport{Task: task.ID, Intent: task.Intent, Cells: []CodeCell{}, Comparisons: []Comparison{}, Reviewers: []ReviewerScore{}}
	reviews := map[string]*CellReviews{}
	if rr != nil {
		scores, err := scoreReviewers(rr.Copies, b.rec.Cells, rr.Verdicts)
		if err != nil {
			return CodeReport{}, fmt.Errorf("the reviews of %s: %w", task.ID, err)
		}
		cr.Reviewers = scores
		truth := map[string]ReviewedCopy{}
		for _, c := range rr.Copies {
			truth[c.Label] = c
		}
		for _, v := range rr.Verdicts {
			c, ok := truth[v.Label]
			if !ok || !v.Defective {
				continue
			}
			r := reviews[c.Key.Cell]
			if r == nil {
				r = &CellReviews{}
				reviews[c.Key.Cell] = r
			}
			switch c.Truth {
			case TruthFailing:
				r.Confirmed++
			case TruthPassing:
				r.Unconfirmed++
			}
		}
	} else {
		b.limit("no reviewer verdicts are recorded for %s, so nothing is reported of what the reviewers caught", task.ID)
	}
	for _, d := range b.decl.Builders {
		pin := b.pin(d.ID)
		ts := b.of(task.ID, d.ID)
		row := CodeCell{Cell: d.ID, Model: pin.Model, Source: pin.Source, Runner: pin.Runner}
		passed, counted := 0, 0
		var model, gates int64
		var hostReported []int
		for _, t := range ts {
			switch {
			case !t.Attempted():
				row.NotStarted++
			case t.Status == StatusUnexamined:
				row.Unexamined++
			case t.Apart():
				row.Excluded++
			case t.Passed():
				passed++
				counted++
			case t.Failed():
				counted++
			}
			if (t.Passed() || t.Failed()) && t.HostReportedScore() {
				hostReported = append(hostReported, t.Key.Repeat)
			}
			if t.Attempted() {
				model += t.ModelTimeMS
				gates += t.GateWaitMS
			}
		}
		row.PassAt1 = passAt1(passed, counted)
		row.PassHatK = passHatK(passed, counted)
		if attempts := len(ts) - row.NotStarted; attempts > 0 {
			row.ModelTimeMS, row.GateWaitMS = model/int64(attempts), gates/int64(attempts)
		}
		if r := reviews[d.ID]; r != nil {
			row.Reviews = *r
		}
		row.Costs = costs(pin.Source, ts, passed)
		row.HostReported = len(hostReported)
		if len(hostReported) > 0 {
			b.limit("the hidden tests' results of %s on %s (%s) are the host's report: abcd neither placed nor ran those tests, so the cell's pass rate, and any comparison on it, rests on the host's word", task.ID, d.ID, repeatList(hostReported))
			b.improve("run %s's hidden tests through abcd, so %s's pass rate is measured rather than reported", task.ID, d.ID)
		}
		b.unreturnedInto(task.ID, &row)
		b.cellLimits(task.ID, row)
		cr.Cells = append(cr.Cells, row)
	}
	if len(b.trials) == 0 {
		b.limit("no copy of the run is recorded yet, so no cell's row has anything under it")
	}
	for i := range cr.Cells {
		for j := i + 1; j < len(cr.Cells); j++ {
			cr.Comparisons = append(cr.Comparisons, b.compare(task.ID, cr.Cells[i], cr.Cells[j]))
		}
	}
	return cr, nil
}

// cellLimits notes what a cell's row could not stand on.
func (b *reportBuilder) cellLimits(task string, row CodeCell) {
	if row.NotStarted > 0 {
		b.limit("%d copies of %s on %s never started (its provider was down before they began); they are in no rate", row.NotStarted, task, row.Cell)
	}
	if row.Unexamined > 0 {
		b.limit("the compliance check could not examine %d copies of %s on %s; they are reported apart, never ranked as compliant", row.Unexamined, task, row.Cell)
		b.improve("keep each copy's transcript where the compliance check reads it, so no copy of %s on %s is left unexamined", task, row.Cell)
	}
	if row.Excluded > 0 {
		b.limit("%d copies of %s on %s broke a rule of the run and are excluded, named in their trial records", row.Excluded, task, row.Cell)
	}
}

// costs is a cell's spend by unit over its copies, and the cost per pass, with
// what each is comparable with by the cell's source. A cell that recorded no
// cost has one row saying why.
func costs(src Source, ts []TrialRecord, passes int) []CellCost {
	totals := map[string]float64{}
	for _, t := range ts {
		if t.Cost != nil {
			totals[t.Cost.Unit] += t.Cost.Amount
		}
	}
	note := comparable(src)
	if len(totals) == 0 {
		return []CellCost{{Comparable: note}}
	}
	units := make([]string, 0, len(totals))
	for u := range totals {
		units = append(units, u)
	}
	slices.Sort(units)
	out := make([]CellCost, 0, len(units))
	for _, u := range units {
		c := CellCost{Unit: u, Total: totals[u], Comparable: note}
		if passes > 0 {
			per := totals[u] / float64(passes)
			c.PerPass = &per
		}
		out = append(out, c)
	}
	return out
}

// comparable is what a cost from src is comparable with.
func comparable(src Source) string {
	switch src {
	case SourceHost:
		return "a host cell runs on the host's own plan and spends nothing metered a copy at a time, so its cost is not comparable with a metered cell's"
	case SourceLocal:
		return "a local cell spends nothing metered; the machine's time and power are not counted, so its cost is not comparable with a metered cell's"
	default:
		return "metered spend as the route reported it; comparable with another metered cell's in the same unit"
	}
}

// compare sets two builder cells' pass rates against each other, ranked only
// at the repeats the power estimate asks for.
func (b *reportBuilder) compare(task string, a, c CodeCell) Comparison {
	cmp := b.comparison(task, a, c)
	if a.HostReported > 0 || c.HostReported > 0 {
		cmp.HostReported = true
		cmp.Note += "; its pass rates rest on hidden tests' results that are the host's report, which abcd neither placed nor ran"
	}
	return cmp
}

// comparison is compare's figures and note, before any is qualified.
func (b *reportBuilder) comparison(task string, a, c CodeCell) Comparison {
	cmp := Comparison{A: a.Cell, B: c.Cell, Kind: "model", RepeatsA: a.PassAt1.Attempts, RepeatsB: c.PassAt1.Attempts}
	if a.Source != c.Source {
		cmp.Kind = "model and harness"
		b.limit("%s and %s come from different sources, so their comparison sets model and harness together, never the model alone (decision 10)", a.Cell, c.Cell)
		b.improve("add an anchor cell that runs one model through both harnesses, so %s against %s separates the model from the harness", a.Cell, c.Cell)
	}
	na, nb := a.PassAt1.Attempts, c.PassAt1.Attempts
	if na == 0 || nb == 0 {
		cmp.MinDetectable = 1
		cmp.Note = "one of the two has no attempt that counts, so the comparison could detect no difference and is not ranked"
		return cmp
	}
	diff := *a.PassAt1.Rate - *c.PassAt1.Rate
	cmp.Difference = &diff
	pooled := float64(a.PassAt1.Passed+c.PassAt1.Passed) / float64(na+nb)
	cmp.MinDetectable = minDetectable(na, nb, pooled)
	delta := b.decl.Repeats.MinDifference
	if delta == nil {
		cmp.Note = fmt.Sprintf("no smallest difference that would change a choice was declared, so the repeats were not sized; at %d and %d repeats the smallest difference it could have detected is %.2f, so it is not ranked",
			na, nb, cmp.MinDetectable)
		b.limit("%s against %s on %s is not ranked: the run declared no smallest difference, and the smallest difference its repeats could have detected is %.2f",
			a.Cell, c.Cell, task, cmp.MinDetectable)
		b.improve("declare repeats.min_difference, the smallest difference in pass rate that would change a routing choice, so the next run sizes its repeats by a power estimate")
		return cmp
	}
	cmp.Needed = repeatsNeeded(*delta, pooled)
	if na < cmp.Needed || nb < cmp.Needed {
		cmp.Note = fmt.Sprintf("it rests on %d and %d repeats, fewer than the %d each its power estimate asks for to detect a difference of %.2f; the smallest difference it could have detected is %.2f, so it is not ranked",
			na, nb, cmp.Needed, *delta, cmp.MinDetectable)
		b.limit("%s against %s on %s is under its power estimate: the smallest difference it could have detected is %.2f, past the %.2f declared", a.Cell, c.Cell, task, cmp.MinDetectable, *delta)
		b.improve("repeat %s and %s to %d attempts each, so their comparison can detect a difference of %.2f", a.Cell, c.Cell, cmp.Needed, *delta)
		return cmp
	}
	cmp.Ranked = true
	if z := math.Abs(diff) / math.Sqrt(spread(pooled)*(1/float64(na)+1/float64(nb))); z >= zAlpha {
		cmp.Ahead = a.Cell
		if diff < 0 {
			cmp.Ahead = c.Cell
		}
		cmp.Note = fmt.Sprintf("%s is ahead by %.2f, past chance at its repeats", cmp.Ahead, math.Abs(diff))
	} else {
		cmp.Note = fmt.Sprintf("the difference of %.2f is within chance at its repeats: the two are not separable", math.Abs(diff))
	}
	return cmp
}

// writing builds a written doc's rows from the session's recorded verdicts and
// the key the run drew them against.
func (b *reportBuilder) writing(proc *lab.Procedure, task Task) (WritingRow, error) {
	row := WritingRow{Task: task.ID, GoodEnough: []GoodEnoughRow{}, Copies: []CopyLength{}}
	b.limit("people judging %s were not asked to guess who wrote a copy (decision 32), so how far they were kept blind is unmeasured", task.ID)
	b.limit("a writing task judged by a few people supports choosing a copy, or a model for this kind of page; a claim about models in general needs more pages and judges than one run")
	raw, err := proc.ReadState(judgingRel(b.rec.RunID, task.ID, "verdicts.json"))
	if errors.Is(err, fs.ErrNotExist) {
		b.limit("the writing task %s is not judged yet: no verdicts are recorded", task.ID)
		return row, nil
	}
	if err != nil {
		return WritingRow{}, err
	}
	var res JudgingResult
	if err := jsonstrict.Decode(raw, &res); err != nil || res.Run != b.rec.RunID || res.Task != task.ID {
		return WritingRow{}, fmt.Errorf("the verdicts of %s do not read as its own", task.ID)
	}
	row.Judged = true
	row.Members, row.Guests, row.HostSelfBlinded = len(res.Members), len(res.Guests), len(res.HostSelfBlinded)
	if row.Guests > 0 || row.HostSelfBlinded > 0 {
		row.Apart = map[string][]JudgeRanking{}
		if row.Guests > 0 {
			row.Apart["guests"] = res.Guests
		}
		if row.HostSelfBlinded > 0 {
			row.Apart["host_self_blinded"] = res.HostSelfBlinded
			b.limit("the host also judged %s and holds the key, so the host's rankings are self-blinded and reported apart", task.ID)
		}
	}
	good := map[string]*GoodEnoughRow{}
	copyMarks := map[TrialKey][2]int{}
	places := map[TrialKey][]int{}
	words := map[TrialKey]int{}
	for _, d := range b.decl.Builders {
		good[d.ID] = &GoodEnoughRow{Cell: d.ID}
	}
	for _, r := range res.Members {
		for i, c := range r.Ranked {
			g := good[c.Key.Cell]
			if g == nil {
				g = &GoodEnoughRow{Cell: c.Key.Cell}
				good[c.Key.Cell] = g
			}
			g.Of++
			m := copyMarks[c.Key]
			m[1]++
			if c.GoodEnough {
				g.Good++
				m[0]++
			}
			copyMarks[c.Key] = m
			places[c.Key] = append(places[c.Key], i+1)
			words[c.Key] = c.Words
		}
	}
	for k, m := range copyMarks {
		if 2*m[0] > m[1] {
			good[k.Cell].GoodCopies++
		}
	}
	cells := make([]string, 0, len(good))
	for c := range good {
		cells = append(cells, c)
	}
	slices.Sort(cells)
	for _, c := range cells {
		g := good[c]
		ts := b.of(task.ID, c)
		g.Costs = costs(b.pin(c).Source, ts, g.GoodCopies)
		var model int64
		n := 0
		for _, t := range ts {
			if t.Attempted() {
				model += t.ModelTimeMS
				n++
			}
		}
		if n > 0 {
			g.ModelTimeMS = model / int64(n)
		}
		row.GoodEnough = append(row.GoodEnough, *g)
	}
	keys := make([]TrialKey, 0, len(places))
	for k := range places {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })
	for _, k := range keys {
		var sum int
		for _, p := range places[k] {
			sum += p
		}
		row.Copies = append(row.Copies, CopyLength{Key: k, Words: words[k], MeanPlace: float64(sum) / float64(len(places[k]))})
	}
	if row.Members < judgeFloor {
		row.Floor = fmt.Sprintf("fewer than three judges (%d): each judge's ranking and the good-enough counts only, never a pooled rating or an agreement", row.Members)
		row.Rankings = res.Members
		b.limit("%s was judged by fewer than three judges (%d), so it gives each judge's ranking and the good-enough counts only", task.ID, row.Members)
		b.improve("invite at least three judges to %s, so its rankings can be pooled into ratings and its agreement estimated", task.ID)
		return row, nil
	}
	seed := b.seed(task.ID)
	row.Ratings = rateModels(res.Members, seed)
	ge, order := agreement(res.Members, seed)
	row.Agreement = &WritingAgreement{GoodEnough: ge, Order: order}
	for _, a := range []Agreement{ge, order} {
		if a.Alpha == nil {
			b.limit("the judges' agreement on %s is not estimable: %s", task.ID, a.NotEstimable)
			b.improve("overlap the judges' subsets of %s, so copies are judged twice or more and agreement can be estimated", task.ID)
		}
	}
	for _, r := range row.Ratings {
		if !r.Separable && len(row.Ratings) > 1 {
			b.limit("on %s the model %s cannot be separated from any other at this run's judges and copies", task.ID, r.Model)
			b.improve("add judges or copies to %s, so %s can be separated from the others", task.ID, r.Model)
		}
	}
	return row, nil
}

// runLimits notes what the run as a whole could not pin or examine.
func (b *reportBuilder) runLimits() {
	for _, u := range b.rec.Unpinned {
		b.limit("%s could not be pinned (%s)", u.Field, u.Reason)
		b.improve("pin %s in the next run, or say why it cannot be", u.Field)
	}
	host, examined, noCalls, launched := false, false, false, false
	for _, t := range b.trials {
		if t.Source == SourceHost {
			host = true
		} else if len(t.Roles) > 0 {
			launched = true
		}
		if slices.Contains(t.Unpinned, oracle.Unpinned{Field: TrialCalls, Reason: oracle.NotReported}) {
			noCalls = true
		}
		if t.Compliance != nil {
			examined = true
		}
	}
	for _, p := range b.rec.Cells {
		if p.Source == SourceHost {
			host = true
		}
	}
	if host {
		b.limit("copies made in the host's harness ran where abcd neither launches nor confines them, so their isolation is asserted, not proven")
	}
	if launched {
		b.limit("copies launched on a runner rest on abcd's runner launch, which is tested against a fake runner and has no live proof yet, so a harness's real events may read otherwise than its tests assume")
		b.improve("prove the runner launch live on one cheap cell before a run's figures are relied on")
	}
	if noCalls {
		b.limit("copies made in the host's harness carry no call receipts: the model served, the harness version and the times of each are the host's word, marked self-reported in its trial, and its tokens and cost are not reported")
	}
	if examined {
		b.limit("no runner reports what a copy reached on the network, so the run's network policy is asserted for every copy, not examined")
		b.limit("whether a copy read only its own clone rests on a heuristic scan of its transcript, not a barrier: a read the transcript does not record is not seen")
	}
	b.limit("a model run is re-runnable, not reproducible: a re-run starts from this record and reports where it drifted")
}
