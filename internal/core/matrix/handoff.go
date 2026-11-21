package matrix

// handoff.go starts a consented run's copies and takes each one back
// (spc-2610161949559050 step 8; the Starter consent.go names). Model work is
// delegated to the host by default, as the build loop hands its agent stages
// to the host: the run makes every pilot copy's clone under a random alias
// (makeCopyClone), passes a provider-bound clone through the launch bundle's
// scan (admitClone), snapshots what the release gates read before a check
// copy starts, and records each copy's hand-off in the run's state, with the
// blind brief the copy is made from. The host makes a host cell's copy in its
// clone and hands it back (CollectCopy): the run checks it for compliance
// under one overall deadline (checkCompliance), reads a check copy's findings
// before its clone is deleted (collectCheckCopy), and writes its trial record
// through the secret screen (writeTrial).
//
// A copy on a runner cell is handed off and waits for a launcher: nothing in
// this step starts a runner in a copy's clone, and nothing stands in for one,
// since an answer from any other model would measure the wrong one (decision
// 4). So the host is never taken at its word for a runner cell's copy, and
// the copy stays absent from the report until a launcher makes it.
// dcn-2611211433133728 records what is left.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// HandOffSchema is a copy hand-off's schema version.
const HandOffSchema = 1

// HandOffState is where a copy stands between its clone and its trial.
type HandOffState string

const (
	// HandOffWaiting: a host cell's copy, its clone made; the host makes the
	// copy there and hands it back.
	HandOffWaiting HandOffState = "handed_off"
	// HandOffAwaitingRunner: a runner cell's copy, its clone made and, for a
	// provider, scanned; nothing launches its runner yet, and the host never
	// stands in for it.
	HandOffAwaitingRunner HandOffState = "awaiting_runner"
	// HandOffRefused: the cell's clone failed the launch bundle's secret scan,
	// so the copy is never sent to its provider; its clone is removed.
	HandOffRefused HandOffState = "refused"
)

// HandOff is one copy's hand-off as the run's state records it. It is the
// host's to read and never the copy's: it names the copy's key. Brief is what
// the copy is asked, and it names no task id, cell, route or model.
type HandOff struct {
	SchemaVersion int          `json:"schema_version"`
	Run           string       `json:"run"`
	Key           TrialKey     `json:"key"`
	Kind          TaskKind     `json:"kind"`
	Source        Source       `json:"source"`
	State         HandOffState `json:"state"`
	// Clone is the copy's clone, an absolute path; empty for a refused cell.
	Clone  string `json:"clone,omitempty"`
	Brief  string `json:"brief"`
	Reason string `json:"reason,omitempty"`
	// Gates is what the release gates read, snapshotted before a check copy
	// starts, so its collection can prove them unchanged. It is always
	// written: a snapshot of a repository with nothing under the gates' roots
	// is {} and still a snapshot, and only a code or writing copy's is null.
	Gates gateSnapshot `json:"gates"`
}

// handOffRel is copy key's hand-off in its run's directory.
func handOffRel(run string, key TrialKey) string {
	return run + "/copies/" + key.Task + "/" + key.Cell + "/" + strconv.Itoa(key.Repeat) + ".json"
}

// writingRel is a writing copy's text in its run's directory.
func writingRel(run string, key TrialKey) string {
	return run + "/writing/" + key.Task + "/" + key.Cell + "/" + strconv.Itoa(key.Repeat) + ".md"
}

// Launcher is the Starter a consented run is started with: it hands off every
// pilot copy, each in its own clone, from the run's checkout at RepoRoot into
// the clone store under the account's home at Home.
type Launcher struct {
	RepoRoot, Home string
	// Handed is every hand-off Start wrote, in the order it wrote them.
	Handed []HandOff
}

// Start hands off every pilot copy of the run rec records: each task, each
// builder cell, repeats one to the pilot. A clone that cannot be made, or a
// hand-off that cannot be written, is the run's error; a cell its clone's
// scan refuses is recorded refused and the run goes on.
func (l *Launcher) Start(_ Plan, rec RunRecord) error {
	proc, err := lab.OpenProcedure(l.RepoRoot, rec.Lab)
	if err != nil {
		return err
	}
	_, d, err := readRun(proc, rec.RunID)
	proc.Close()
	if err != nil {
		return err
	}
	store := cloneStore(l.Home, rec.RootSHA)
	for _, task := range d.Tasks {
		for _, pin := range rec.Cells {
			if pin.Role != RoleBuilder {
				continue
			}
			for r := 1; r <= d.Repeats.Pilot; r++ {
				h, err := l.handOff(rec, store, task, pin, TrialKey{Task: task.ID, Cell: pin.Cell, Repeat: r})
				if err != nil {
					return fmt.Errorf("copy %s/%s/%d is not handed off: %w", task.ID, pin.Cell, r, err)
				}
				l.Handed = append(l.Handed, h)
			}
		}
	}
	return nil
}

// handOff makes one copy's clone and records its hand-off.
func (l *Launcher) handOff(rec RunRecord, store string, task Task, pin CellPin, key TrialKey) (HandOff, error) {
	clone, err := makeCopyClone(rec.Lab, l.RepoRoot, store, rec.RunID, rec.FrozenCommit, key)
	if err != nil {
		return HandOff{}, err
	}
	h := HandOff{SchemaVersion: HandOffSchema, Run: rec.RunID, Key: key, Kind: task.Kind, Source: pin.Source,
		State: HandOffWaiting, Clone: clone, Brief: copyBrief(task, clone)}
	if pin.Source != SourceHost {
		h.State = HandOffAwaitingRunner
		h.Reason = "abcd does not launch a runner in a copy's clone yet, and nothing stands in for the cell's runner, so the copy waits for a launcher"
	}
	if err := admitClone(l.RepoRoot, clone, pin); err != nil {
		if rerr := os.RemoveAll(clone); rerr != nil {
			return HandOff{}, errors.Join(err, rerr)
		}
		h.State, h.Clone, h.Reason = HandOffRefused, "", termsafe.SanitizeLine(err.Error())
	}
	if task.Kind == KindCheck && h.State != HandOffRefused {
		if h.Gates, err = snapshotGates(l.RepoRoot); err != nil {
			return HandOff{}, err
		}
	}
	data, err := encodeCheck(h)
	if err != nil {
		return HandOff{}, err
	}
	proc, err := lab.OpenProcedure(l.RepoRoot, rec.Lab)
	if err != nil {
		return HandOff{}, err
	}
	defer proc.Close()
	return h, proc.WriteState(handOffRel(rec.RunID, key), data)
}

// copyBrief is what a copy of task is asked, in its clone at clone: the work
// and where to do it, and nothing of the run, the task's id, the cell or any
// model.
func copyBrief(task Task, clone string) string {
	stay := " Work only in " + clone + ", and read nothing outside it."
	switch task.Kind {
	case KindCode:
		return "Build the change " + task.Intent + " asks for, as its records in this repository set it out." + stay +
			" Change no test the repository holds; tests of your own are welcome."
	case KindWriting:
		return "Write the document judged on these criteria: " + task.Criteria + " Its sources are " +
			strings.Join(task.Source, ", ") + " in this repository." + stay + " Hand back the document's text."
	default:
		return "Run the check over " + strings.Join(task.Material, ", ") + " in report mode, applying nothing it proposes." + stay +
			" Hand back its findings as the check writes them."
	}
}

// ReadHandOff is copy key's hand-off in run, from the lab labID.
func ReadHandOff(repoRoot, labID, run string, key TrialKey) (HandOff, error) {
	if !runIDRe.MatchString(run) {
		return HandOff{}, fmt.Errorf("%q is not a run id", clip(run))
	}
	if err := key.validate(); err != nil {
		return HandOff{}, err
	}
	proc, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return HandOff{}, err
	}
	defer proc.Close()
	return readHandOff(proc, run, key)
}

func readHandOff(proc stateReader, run string, key TrialKey) (HandOff, error) {
	raw, err := proc.ReadState(handOffRel(run, key))
	if err != nil {
		return HandOff{}, fmt.Errorf("the run %s handed off no copy %s: %w", run, key, err)
	}
	var h HandOff
	if err := jsonstrict.Decode(raw, &h); err != nil || h.SchemaVersion != HandOffSchema || h.Run != run || h.Key != key {
		return HandOff{}, fmt.Errorf("copy %s's hand-off does not read as its own", key)
	}
	return h, nil
}

// ParseTrialKey reads a copy's key from its one-token form, task/cell/repeat.
func ParseTrialKey(s string) (TrialKey, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 3 {
		return TrialKey{}, fmt.Errorf("%q is not a copy's key (task/cell/repeat)", clip(s))
	}
	r, err := strconv.Atoi(parts[2])
	k := TrialKey{Task: parts[0], Cell: parts[1], Repeat: r}
	if err != nil || strconv.Itoa(r) != parts[2] {
		return TrialKey{}, fmt.Errorf("%q is not a copy's key (task/cell/repeat)", clip(s))
	}
	return k, k.validate()
}

// CopyReceipt is the host's account of a copy it made, read as untrusted
// input: what came of it, why where it failed, its harness and served model
// where the host knows them, its times, and, for a code copy, the result of
// its hidden tests as the host ran them.
type CopyReceipt struct {
	SchemaVersion  int       `json:"schema_version"`
	Outcome        string    `json:"outcome"`
	Reason         string    `json:"reason,omitempty"`
	HarnessVersion string    `json:"harness_version,omitempty"`
	ModelServed    string    `json:"model_served,omitempty"`
	StartedAt      time.Time `json:"started_at,omitzero"`
	FinishedAt     time.Time `json:"finished_at,omitzero"`
	ModelTimeMS    int64     `json:"model_time_ms,omitempty"`
	GateWaitMS     int64     `json:"gate_wait_ms,omitempty"`
	Score          *Score    `json:"score,omitempty"`
}

// The outcomes a copy receipt names.
const (
	OutcomeMade   = "made"
	OutcomeFailed = "failed"
)

// maxReceiptBytes bounds a copy receipt.
const maxReceiptBytes = 64 << 10

// hostScoreDetail is a code copy's score detail: the host, not abcd, ran the
// hidden tests.
const hostScoreDetail = "the hidden tests' result as the host session ran and reported it; abcd neither placed nor ran them"

// CopyBack is one copy handed back: the host's receipt, the copy's transcript
// as its harness wrote it (nil where none), and, for a writing or a check
// copy, its answer and, for a check, the form it is in.
type CopyBack struct {
	Receipt    []byte
	Transcript []byte
	Answer     []byte
	Form       CheckForm
}

// CollectCopy takes copy key of run back from the host and writes its trial
// record: the copy is checked for compliance under one overall deadline first,
// reading its clone as data and its transcript for reads outside it, the run's
// state among them (labStore under home); a check copy's findings are read
// into the run's record before its clone is deleted; a writing copy's text is
// kept for its judging. It refuses, writing nothing: a copy the run did not
// hand to the host (a runner cell's, or a refused cell's), one already taken
// back, a receipt that does not validate, a made code copy without its hidden
// tests' result, and a made writing or check copy without its answer.
func CollectCopy(repoRoot, home, labID, run string, key TrialKey, back CopyBack) (TrialRecord, error) {
	if !runIDRe.MatchString(run) {
		return TrialRecord{}, fmt.Errorf("%q is not a run id", clip(run))
	}
	if err := key.validate(); err != nil {
		return TrialRecord{}, err
	}
	rc, err := parseCopyReceipt(back.Receipt)
	if err != nil {
		return TrialRecord{}, err
	}
	proc, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return TrialRecord{}, err
	}
	rec, d, err := readRun(proc, run)
	var h HandOff
	var alias string
	if err == nil {
		h, err = readHandOff(proc, run, key)
	}
	if err == nil {
		alias, err = readAlias(proc, run, key)
	}
	if err == nil {
		if _, terr := proc.ReadState(run + "/" + key.rel()); terr == nil {
			err = fmt.Errorf("copy %s is already handed back; a copy's trial is written once", key)
		} else if !errors.Is(terr, fs.ErrNotExist) {
			err = terr
		}
	}
	proc.Close()
	if err != nil {
		return TrialRecord{}, err
	}
	switch h.State {
	case HandOffWaiting:
	case HandOffAwaitingRunner:
		return TrialRecord{}, fmt.Errorf("copy %s is its cell's runner's to make, and abcd does not launch it yet; the host never makes it, since its answer would measure the wrong model", key)
	default:
		return TrialRecord{}, fmt.Errorf("copy %s's cell was refused (%s), so there is no copy to hand back", key, h.Reason)
	}
	task, pin, err := taskAndPin(rec, d, key)
	if err != nil {
		return TrialRecord{}, err
	}
	t := TrialRecord{SchemaVersion: TrialSchema, Run: run, Key: key, Kind: task.Kind, FrozenCommit: rec.FrozenCommit,
		Source: pin.Source, Runner: pin.Runner, HarnessVersion: rc.HarnessVersion, ModelAsked: pin.Model,
		ModelServed: termsafe.SanitizeLine(rc.ModelServed), Settings: pin.Settings, StartedAt: rc.StartedAt, FinishedAt: rc.FinishedAt,
		ModelTimeMS: rc.ModelTimeMS, GateWaitMS: rc.GateWaitMS}
	if task.Kind == KindCode {
		t.Intent = task.Intent
	}
	if t.Runner == "" {
		t.Runner = "host"
	}
	if task.Kind == KindCheck && back.Form != CheckDocFidelity && back.Form != CheckConsistency {
		return TrialRecord{}, fmt.Errorf("copy %s is a check copy, handed back with its answer's form (doc-fidelity or consistency)", key)
	}
	var text string
	switch {
	case rc.Outcome == OutcomeFailed:
		if t.FinishedAt.IsZero() {
			t.FinishedAt = time.Now().UTC()
		}
		t.Status, t.Reason = StatusFailed, clipTo("the host reports the copy failed: "+termsafe.SanitizeLine(rc.Reason), maxReasonBytes)
	case task.Kind == KindCode:
		s := rc.Score
		if s == nil || s.Total < 1 {
			return TrialRecord{}, fmt.Errorf("copy %s is a made code copy, handed back with its hidden tests' result (a score of at least one test); it carries none", key)
		}
		t.Score = &Score{Passed: s.Passed, Total: s.Total, Detail: hostScoreDetail}
		t.Status, t.Reason = StatusFailed, fmt.Sprintf("its hidden tests: %d of %d passed", s.Passed, s.Total)
		if s.Passed == s.Total {
			t.Status = StatusPassed
		}
	case task.Kind == KindWriting:
		text = string(back.Answer)
		if err := readable("copy "+key.String()+"'s document", text); err != nil {
			return TrialRecord{}, err
		}
		t.Status, t.Reason = StatusPassed, "the copy was made; people judge it in the run's judging session, never by its trial"
	default:
		if len(back.Answer) == 0 {
			return TrialRecord{}, fmt.Errorf("copy %s is a made check copy, handed back with its answer", key)
		}
		t.Status, t.Reason = StatusPassed, "the copy was made; its findings are judged in the run's two-phase comparison, never by its trial"
	}
	store := cloneStore(home, rec.RootSHA)
	c, err := checkCompliance(complianceInputs{RepoRoot: repoRoot, Store: store, Run: run, Key: key, Alias: alias,
		Frozen: rec.FrozenCommit, Source: pin.Source, Transcript: back.Transcript, State: labStore(home, rec.RootSHA)})
	if err != nil {
		return TrialRecord{}, err
	}
	t = c.apply(t)
	if err := t.validate(); err != nil {
		return TrialRecord{}, fmt.Errorf("the trial is not written: %w", err)
	}
	switch task.Kind {
	case KindWriting:
		// Where the copy ran is withheld before its text is kept, so the
		// kept text never carries a place on this machine; the judging's
		// scrub withholds everything else that could sign it.
		text, _ = newRedactor(reviewInputs{RepoRoot: repoRoot, Store: store, Dir: cloneDir(store, run, alias), Run: run}).redact(text)
		if err := keepWriting(repoRoot, labID, run, key, text); err != nil {
			return TrialRecord{}, err
		}
	case KindCheck:
		in := checkCopyInputs{RepoRoot: repoRoot, Store: store, Run: run, Frozen: rec.FrozenCommit, Key: key, Form: back.Form,
			Answer: back.Answer, Gates: h.Gates}
		if rc.Outcome == OutcomeFailed {
			in.Failure, in.Answer = clipTo(termsafe.SanitizeLine(rc.Reason), maxReasonBytes), nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), complianceDeadline)
		in.Material, err = readFrozenMaterial(ctx, repoRoot, rec.FrozenCommit, task.Material)
		cancel()
		if err != nil {
			return TrialRecord{}, fmt.Errorf("copy %s keeps its clone: its frozen material could not be read: %w", key, err)
		}
		cc, err := collectCheckCopy(labID, in)
		if err != nil {
			return TrialRecord{}, err
		}
		if !cc.Answered && t.Status == StatusPassed {
			t.Status, t.Reason = StatusFailed, cc.Reason
		}
	}
	return t, writeTrial(repoRoot, labID, t)
}

// parseCopyReceipt reads a copy receipt strictly and holds it to its bounds.
func parseCopyReceipt(data []byte) (CopyReceipt, error) {
	if len(data) > maxReceiptBytes {
		return CopyReceipt{}, fmt.Errorf("the copy's receipt is %d bytes; a receipt is at most %d", len(data), maxReceiptBytes)
	}
	var rc CopyReceipt
	if err := jsonstrict.Decode(data, &rc); err != nil {
		return CopyReceipt{}, fmt.Errorf("the copy's receipt does not parse: %w", err)
	}
	switch {
	case rc.SchemaVersion != HandOffSchema:
		return CopyReceipt{}, fmt.Errorf("the copy's receipt has schema_version %d; this abcd reads %d", rc.SchemaVersion, HandOffSchema)
	case rc.Outcome != OutcomeMade && rc.Outcome != OutcomeFailed:
		return CopyReceipt{}, fmt.Errorf("the copy's receipt names the outcome %q; a copy is made or failed", clip(rc.Outcome))
	case rc.Outcome == OutcomeFailed && strings.TrimSpace(rc.Reason) == "":
		return CopyReceipt{}, errors.New("the copy's receipt says it failed and names no reason")
	case rc.Outcome == OutcomeMade && rc.Reason != "":
		return CopyReceipt{}, errors.New("the copy's receipt says it was made and gives a failure's reason")
	}
	for field, v := range map[string]string{"reason": rc.Reason, "harness_version": rc.HarnessVersion, "model_served": rc.ModelServed} {
		if err := line(field, v, maxReasonBytes); err != nil {
			return CopyReceipt{}, fmt.Errorf("the copy's receipt: %w", err)
		}
	}
	return rc, nil
}

// taskAndPin is copy key's task and builder cell, as run rec declares them.
func taskAndPin(rec RunRecord, d Declaration, key TrialKey) (Task, CellPin, error) {
	var task *Task
	for i := range d.Tasks {
		if d.Tasks[i].ID == key.Task {
			task = &d.Tasks[i]
		}
	}
	if task == nil {
		return Task{}, CellPin{}, fmt.Errorf("the run declares no task %s", key.Task)
	}
	for _, p := range rec.Cells {
		if p.Cell == key.Cell && p.Role == RoleBuilder {
			return *task, p, nil
		}
	}
	return Task{}, CellPin{}, fmt.Errorf("the run pins no builder cell %s", key.Cell)
}

// keepWriting keeps a writing copy's text for its judging, where it ran
// already withheld, passed through the secret screen first, since it is a
// model's words.
func keepWriting(repoRoot, labID, run string, key TrialKey, text string) error {
	rel := writingRel(run, key)
	data := []byte(text)
	if err := screen(repoRoot, []screened{{lab.StateFile{Rel: rel, Data: data}, data}}); err != nil {
		return err
	}
	proc, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return err
	}
	defer proc.Close()
	// A text kept by a hand-back whose trial then failed to write is the
	// same text, so a retry goes on past it.
	if kept, err := proc.ReadState(rel); err == nil && bytes.Equal(kept, data) {
		return nil
	}
	return proc.WriteState(rel, data)
}
