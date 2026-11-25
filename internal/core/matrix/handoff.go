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
// under one overall deadline (checkCompliance), records a check copy's
// findings (recordCheckCopy), writes its trial record through the secret
// screen (writeTrial), and only then deletes a check copy's clone.
//
// A copy on a runner cell is handed off and waits for its launch (launch.go,
// step 11): `lab matrix launch` starts the cell's runner in the copy's clone,
// and nothing stands in for it, since an answer from any other model would
// measure the wrong one (decision 4). So the host is never taken at its word
// for a runner cell's copy, and the report counts it as awaiting its runner,
// in no rate, until its launch makes it. A launched code copy waits for its
// hidden tests' result, which the host hands back alone (scoreLaunched) until
// abcd runs them itself (step 10).
//
// Every copy handed off is accounted for (readHandOffs): the report shows each
// one that holds no trial, and a review or a judging, drawn once, waits while
// any copy of its task is handed to the host and not back (awaitHost).

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/core/oracle"
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
	// HandOffLaunching: a runner cell's copy whose launch has begun and not
	// ended: its runner is at work, or the launch stopped before it could say
	// so.
	HandOffLaunching HandOffState = "launching"
	// HandOffLaunched: a runner cell's copy its launch ended: its trial is
	// written, or, for a code copy its runner made, kept in the hand-off
	// (HandOff.Trial) until its hidden tests' result is handed back.
	HandOffLaunched HandOffState = "launched"
	// HandOffInterrupted: a runner cell's copy whose launch was stopped
	// before the copy was made. That is the run's error, never the copy's
	// score, and the copy is not launched again: its clone holds whatever the
	// runner left.
	HandOffInterrupted HandOffState = "interrupted"
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
	// Trial is a launched code copy's trial as its launch made it, its
	// compliance checked, awaiting only its hidden tests' result; nil for
	// every other copy.
	Trial *TrialRecord `json:"trial,omitempty"`
}

// handOffRel is copy key's hand-off in its run's directory.
func handOffRel(run string, key TrialKey) string {
	return run + "/copies/" + key.Task + "/" + key.Cell + "/" + strconv.Itoa(key.Repeat) + ".json"
}

// The stages of a copy's launch, each recorded once beside its hand-off: the
// lab's records are written once and never replaced, so a launch never writes
// over the hand-off it starts from.
const (
	launchBegun = "launching"
	launchEnded = "ended"
)

// launchRel is the record of copy key's launch at stage in run's state.
func launchRel(run string, key TrialKey, stage string) string {
	return run + "/launches/" + key.Task + "/" + key.Cell + "/" + strconv.Itoa(key.Repeat) + "." + stage + ".json"
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
		State: HandOffWaiting, Clone: clone, Brief: copyBrief(task)}
	if pin.Source != SourceHost {
		h.State = HandOffAwaitingRunner
		h.Reason = "the copy waits for its launch (abcd lab matrix launch), which starts the cell's runner in its clone; nothing stands in for the runner"
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

// copyBrief is what a copy of task is asked: the work and where to do it, and
// nothing of the run, the task's id, the cell or any model. Where it works is
// named as the directory it starts in, its clone, and never by the clone's
// path, which carries the account's home directory; the hand-off records the
// path for whoever starts the copy there.
func copyBrief(task Task) string {
	stay := " Work only in the current directory, this repository's clone, and read nothing outside it."
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
	if h.State != HandOffAwaitingRunner {
		return h, nil
	}
	return launchOverlay(proc, run, h), nil
}

// launchOverlay is runner cell's copy h as its launch left it: the stage it
// ended at, else the one it began. A launch that began and recorded no end
// that reads (none, or one torn by a kill mid-write) is still at work while a
// launch of the run holds the run's lock, and otherwise is gone: the copy
// reads as interrupted, saying why, and the rest of the run reads as it is.
// The end is read once more after the lock is found free, since a launch
// writes its last end record and then lets go of the lock: a reader between
// the two reads the end it wrote, never an interrupted copy.
func launchOverlay(proc stateReader, run string, h HandOff) HandOff {
	ended, endTorn := readLaunchStage(proc, run, h, launchEnded)
	if ended != nil {
		return *ended
	}
	begun, begunTorn := readLaunchStage(proc, run, h, launchBegun)
	if begun == nil && !endTorn && !begunTorn {
		return h
	}
	at := h
	if begun != nil {
		at = *begun
	}
	if launchHeld(proc, run) {
		at.State = HandOffLaunching
		if begun == nil {
			at.Reason = "its launch began and is still at work"
		}
		return at
	}
	if ended, endTorn = readLaunchStage(proc, run, h, launchEnded); ended != nil {
		return *ended
	}
	var torn []string
	if endTorn {
		torn = append(torn, launchRel(run, h.Key, launchEnded))
	}
	if begunTorn {
		torn = append(torn, launchRel(run, h.Key, launchBegun))
	}
	at.State = HandOffInterrupted
	if len(torn) > 0 {
		at.Reason = "its launch record " + strings.Join(torn, " and ") + " does not read (written in part when its launch was cut off), " +
			"so the copy is read as interrupted and is not launched again"
	} else {
		at.Reason = "its launch began and left no record of how it ended (the launch was killed, or its machine lost), " +
			"so the copy is read as interrupted and is not launched again"
	}
	return at
}

// readLaunchStage is copy h's launch record at stage, nil where it is not
// written, and torn where it is written and does not read as its own.
func readLaunchStage(proc stateReader, run string, h HandOff, stage string) (*HandOff, bool) {
	raw, err := proc.ReadState(launchRel(run, h.Key, stage))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false
	}
	var l HandOff
	if err == nil {
		err = jsonstrict.Decode(raw, &l)
	}
	if err != nil || l.SchemaVersion != HandOffSchema || l.Run != run || l.Key != h.Key ||
		l.Kind != h.Kind || l.Source != h.Source || launchStage(l.State) != stage {
		return nil, true
	}
	return &l, false
}

// launchHeld reports whether a launch of run holds the run's launch lock now.
// A run read from anything but its lab (a carried run's folder) has no live
// launch on this machine; one whose lock cannot be judged is held to be at
// work, which keeps its copies where they are.
func launchHeld(proc stateReader, run string) bool {
	e, ok := proc.(interface{ Entry() lab.Entry })
	if !ok {
		return false
	}
	home, err := os.UserHomeDir()
	if err != nil || !filepath.IsAbs(home) {
		return true
	}
	return runLockHeld(launchLockPath(handoffStore(home, e.Entry().RootSHA), run))
}

// launchStage is the stage of a launch whose copy is left at state, "" for a
// state no launch leaves.
func launchStage(state HandOffState) string {
	switch state {
	case HandOffLaunching:
		return launchBegun
	case HandOffLaunched, HandOffInterrupted, HandOffRefused:
		return launchEnded
	}
	return ""
}

// readHandOffs is every copy hand-off of run, each read as its own, in key
// order. A run no launcher started holds none.
func readHandOffs(proc stateReader, run string) ([]HandOff, error) {
	dir := run + "/copies"
	list := func(rel string) ([]string, error) {
		names, err := proc.ListState(rel)
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		if len(names) > MaxRepeats+maxTasks+maxCells {
			return nil, fmt.Errorf("%s holds %d entries, more than a run writes", rel, len(names))
		}
		return names, err
	}
	tasks, err := list(dir)
	if err != nil {
		return nil, err
	}
	var out []HandOff
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
				n, ok := strings.CutSuffix(f, ".json")
				r, err := strconv.Atoi(n)
				key := TrialKey{Task: task, Cell: cell, Repeat: r}
				if !ok || err != nil || strconv.Itoa(r) != n || key.validate() != nil {
					return nil, fmt.Errorf("the hand-off at %s/%s/%s is not a copy's", clip(task), clip(cell), clip(f))
				}
				h, err := readHandOff(proc, run, key)
				if err != nil {
					return nil, err
				}
				out = append(out, h)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key.String() < out[j].Key.String() })
	return out, nil
}

// awaitHost refuses a draw of task's what (its review, its judging) while a
// copy of task is handed to the host and not handed back, or launched and not
// yet written as its trial: a draw is made once, so a copy that came back
// after it could never join it. A runner cell's copy not yet launched, an
// interrupted launch's, and a refused cell's, none of which comes back by
// itself, do not hold a draw; the report names each.
func awaitHost(proc stateReader, run, task, what string) error {
	handed, err := readHandOffs(proc, run)
	if err != nil {
		return err
	}
	var out []string
	for _, h := range handed {
		if h.Key.Task != task || h.State != HandOffWaiting && h.State != HandOffLaunching && h.State != HandOffLaunched {
			continue
		}
		if _, err := proc.ReadState(run + "/" + h.Key.rel()); err == nil {
			continue
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		out = append(out, h.Key.String())
	}
	if len(out) == 0 {
		return nil
	}
	return fmt.Errorf("%d copies of %s are handed to the host and not handed back, or launched and not yet done (%s); %s is drawn once, from every copy, so it waits for them: hand each back, a failed one with its reason",
		len(out), task, strings.Join(out, ", "), what)
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

// hostTrialUnpinned is what a trial built from the host's receipt rc could
// not pin. The host hands back no call receipts, so every field of a call's
// served side is not reported but the model the receipt names, which is the
// host's own word, as are the harness version and the times it gives, and, for
// a scored copy, its hidden tests' result: the host ran them, not abcd.
func hostTrialUnpinned(rc CopyReceipt, scored bool) []oracle.Unpinned {
	var un oracle.Unpin
	reported := func(have bool) oracle.UnpinnedReason {
		if have {
			return oracle.SelfReported
		}
		return oracle.NotReported
	}
	un.Add(oracle.AskedHarnessVersion, reported(rc.HarnessVersion != ""))
	for _, f := range oracle.ServedFields {
		un.Add(f, reported(f == oracle.ServedModel && rc.ModelServed != ""))
	}
	un.Add(TrialCalls, oracle.NotReported)
	if scored {
		un.Add(TrialScore, oracle.SelfReported)
	}
	un.Add(TrialTimes, reported(!rc.StartedAt.IsZero() || !rc.FinishedAt.IsZero() || rc.ModelTimeMS != 0 || rc.GateWaitMS != 0))
	return un.List()
}

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
// back (short of a check copy whose clone outlived its trial, which a hand-back
// again deletes, returning the trial written: leftoverClone), a receipt that
// does not validate, a made code copy without its hidden
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
	var written []byte
	if err == nil {
		h, err = readHandOff(proc, run, key)
	}
	if err == nil {
		alias, err = readAlias(proc, run, key)
	}
	if err == nil {
		if raw, terr := proc.ReadState(run + "/" + key.rel()); terr == nil {
			written = raw
		} else if !errors.Is(terr, fs.ErrNotExist) {
			err = terr
		}
	}
	proc.Close()
	if err != nil {
		return TrialRecord{}, err
	}
	if written != nil {
		return leftoverClone(home, rec, h, alias, key, written)
	}
	switch h.State {
	case HandOffWaiting:
	case HandOffAwaitingRunner, HandOffLaunching, HandOffInterrupted:
		return TrialRecord{}, fmt.Errorf("copy %s is its cell's runner's to make, with abcd lab matrix launch; the host never makes it, since its answer would measure the wrong model", key)
	case HandOffLaunched:
		return scoreLaunched(repoRoot, labID, h, rc, back)
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
	scored := false
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
		scored = true
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
		Frozen: rec.FrozenCommit, Source: pin.Source, Transcript: back.Transcript, State: labStore(home, rec.RootSHA),
		Handoff: handoffStore(home, rec.RootSHA)})
	if err != nil {
		return TrialRecord{}, err
	}
	t = c.apply(t)
	t.Unpinned = hostTrialUnpinned(rc, scored)
	if err := t.validate(); err != nil {
		return TrialRecord{}, fmt.Errorf("the trial is not written: %w", err)
	}
	switch task.Kind {
	case KindWriting:
		// The trial is screened before the copy's text is kept, so a
		// receipt the screen refuses keeps nothing and the copy can be
		// handed back again with any document. A failed copy has no text.
		if _, err := screenTrial(repoRoot, t); err != nil {
			return TrialRecord{}, err
		}
		if rc.Outcome == OutcomeFailed {
			break
		}
		// Where the copy ran is withheld before its text is kept, so the
		// kept text never carries a place on this machine; the judging's
		// scrub withholds everything else that could sign it.
		text, _ = newRedactor(reviewInputs{RepoRoot: repoRoot, Store: store, Dir: cloneDir(store, run, alias), Run: run}).redact(text)
		if err := keepWriting(repoRoot, labID, run, key, text); err != nil {
			return TrialRecord{}, err
		}
		// The text is kept from here on, so a trial that then fails to
		// write says so rather than that nothing was written.
		if err := writeTrial(repoRoot, labID, t); err != nil {
			return t, afterWrite(fmt.Errorf("copy %s's text is kept, and its trial is not written: %w", key, err))
		}
		return t, nil
	case KindCheck:
		// The trial is screened before anything of the copy is recorded, so
		// a receipt the screen refuses leaves the copy as it was, its clone
		// in place, to be handed back again.
		if _, err := screenTrial(repoRoot, t); err != nil {
			return TrialRecord{}, err
		}
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
		cc, clone, err := recordCheckCopy(labID, in)
		if err != nil {
			return TrialRecord{}, err
		}
		if !cc.Answered && t.Status == StatusPassed {
			t.Status, t.Reason = StatusFailed, cc.Reason
		}
		// The clone goes only once the trial is written: a write the screen
		// refuses keeps it, and a retry finds the same findings recorded and
		// goes on; a write a concurrent hand-back of the same copy won leaves
		// the clone to that one (lostTrialWrite), and a deletion that fails
		// is tried again by handing the copy back again (leftoverClone).
		if err := writeTrial(repoRoot, labID, t); err != nil {
			return TrialRecord{}, lostTrialWrite(repoRoot, labID, run, key, err)
		}
		if err := os.RemoveAll(clone); err != nil {
			return TrialRecord{}, afterWrite(fmt.Errorf("copy %s's trial is written, and its clone at %s could not be deleted: %w; hand the copy back again to delete it", key, clone, err))
		}
		return t, nil
	}
	return t, writeTrial(repoRoot, labID, t)
}

// leftoverClone answers a hand-back of copy key whose trial is already
// written. A check copy whose clone outlived its trial, because deleting it
// failed, has the clone deleted now and its trial, as written, returned, so
// the hand-back can be tried again until the clone is gone; its receipt is not
// read again and no second trial is written. Anything else is refused: a
// copy's trial is written once.
func leftoverClone(home string, rec RunRecord, h HandOff, alias string, key TrialKey, written []byte) (TrialRecord, error) {
	refused := fmt.Errorf("copy %s is already handed back; a copy's trial is written once", key)
	if h.Kind != KindCheck || alias == "" {
		return TrialRecord{}, refused
	}
	clone := cloneDir(cloneStore(home, rec.RootSHA), rec.RunID, alias)
	if _, err := os.Lstat(clone); errors.Is(err, fs.ErrNotExist) {
		return TrialRecord{}, refused
	}
	var t TrialRecord
	if err := jsonstrict.Decode(written, &t); err != nil || t.Key != key || t.Run != rec.RunID {
		return TrialRecord{}, fmt.Errorf("copy %s's trial does not read as its own, so its clone at %s is kept", key, clone)
	}
	if err := os.RemoveAll(clone); err != nil {
		return TrialRecord{}, afterWrite(fmt.Errorf("copy %s's trial is written, and its clone at %s could not be deleted: %w; hand the copy back again to delete it", key, clone, err))
	}
	return t, nil
}

// lostTrialWrite is the refusal of a check copy's hand-back whose trial was
// not written. Where a trial of the copy is written even so, another
// hand-back of it wrote it first, and that one deletes the clone, so the
// refusal says so rather than that the clone is kept, and promises no retry:
// a hand-back once the clone is gone is refused (leftoverClone). Otherwise
// the clone is kept for the copy to be handed back again. Either way the
// copy's findings are recorded already (recordCheckCopy), so the refusal is
// marked as raised after a write.
func lostTrialWrite(repoRoot, labID, run string, key TrialKey, err error) error {
	if proc, perr := lab.OpenProcedure(repoRoot, labID); perr == nil {
		_, terr := proc.ReadState(run + "/" + key.rel())
		proc.Close()
		if terr == nil {
			return afterWrite(fmt.Errorf("copy %s's findings are recorded, and it was handed back by another hand-back, whose trial is the one written: %w; that hand-back deletes the copy's clone, so there is nothing more to hand back", key, err))
		}
	}
	return afterWrite(fmt.Errorf("copy %s's findings are recorded, and it keeps its clone: %w", key, err))
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
