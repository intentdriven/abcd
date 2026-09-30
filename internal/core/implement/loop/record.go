package loop

// record.go is the run record and the run's transcripts (spec piece 10;
// criterion 10). The state file accumulates the record as stages complete; at
// the end it is read back as one document naming every lane, the receipts the
// loop verified with the model each runner reported, every verdict the loop
// recorded, the landing, and the transcripts captured into the history store.
// The transcripts are captured by path, one history capture per path, once the
// run is complete: the host names the paths, since only the host knows where
// its sessions' transcripts are.

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/intentdriven/abcd/internal/core/runner"
)

// StageTranscript is the run record's stage for a captured transcript, and
// StageRecord the refusal stage of the record's own verb.
const (
	StageTranscript = "transcript"
	StageRecord     = "record"
)

// RunRecord is a run's record as it is read at the end.
type RunRecord struct {
	RunID     string    `json:"run_id"`
	Key       string    `json:"key"`
	Intent    string    `json:"intent"`
	Spec      string    `json:"spec"`
	Driver    Driver    `json:"driver"`
	Complete  bool      `json:"complete"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// Pace is the pace the run ran on, nil for a run started before the loop
	// paced a run.
	Pace        *Pace         `json:"pace"`
	Lanes       []RecordLane  `json:"lanes"`
	Pending     []PendingStep `json:"pending"`
	Transcripts []Transcript  `json:"transcripts"`
	// Fallbacks are the run's fallback receipts, and FallbackCounts their
	// count per runner asked for and per role (itd-2609201916056194
	// criterion 4).
	Fallbacks      []runner.FallbackReceipt `json:"fallbacks"`
	FallbackCounts runner.Counts            `json:"fallback_counts"`
	Record         []Entry                  `json:"record"`
}

// RecordLane is one lane of the run record.
type RecordLane struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	SpecStep  int    `json:"spec_step"`
	StepTitle string `json:"step_title"`
	Stage     Stage  `json:"stage"`
	Branch    string `json:"branch,omitempty"`
	BaseSHA   string `json:"base_sha,omitempty"`
	HeadSHA   string `json:"head_sha,omitempty"`
	// Receipts are the implementers' receipts the loop verified, each with the
	// model its runner reported.
	Receipts []ReceiptRecord `json:"receipts"`
	// Verdicts are every verdict the loop recorded, round by round.
	Verdicts []RecordVerdict `json:"verdicts"`
	// Resolves are the captures the lane fixed.
	Resolves []string `json:"resolves"`
	// PR is the lane's pull request, and Landing what its landing did.
	PR       int       `json:"pr,omitempty"`
	Landing  *Landing  `json:"landing,omitempty"`
	HandBack *HandBack `json:"hand_back,omitempty"`
}

// RecordVerdict is one verdict the loop recorded from a validator's return.
type RecordVerdict struct {
	Round   int    `json:"round"`
	HeadSHA string `json:"head_sha"`
	Role    string `json:"role"`
	Verdict string `json:"verdict"`
	Pass    bool   `json:"pass"`
	// Return is the validator's return the verdict was parsed from.
	Return string `json:"return"`
	// Route is the runner route that ran the validator; absent when the host
	// ran it.
	Route *runner.RouteRecord `json:"route,omitempty"`
}

// recordOf renders a run's state as its record.
func recordOf(st State) RunRecord {
	rec := RunRecord{RunID: st.RunID, Key: st.Key, Intent: st.Intent, Spec: st.Spec, Driver: st.Driver,
		Complete: st.Complete(), CreatedAt: st.CreatedAt, UpdatedAt: st.UpdatedAt, Pace: st.Pace,
		Lanes: []RecordLane{}, Pending: st.Pending, Transcripts: st.Transcripts, Record: st.Record,
		Fallbacks: st.Fallbacks, FallbackCounts: runner.Tally(st.Fallbacks)}
	if rec.Fallbacks == nil {
		rec.Fallbacks = []runner.FallbackReceipt{}
	}
	if rec.Pending == nil {
		rec.Pending = []PendingStep{}
	}
	if rec.Transcripts == nil {
		rec.Transcripts = []Transcript{}
	}
	if rec.Record == nil {
		rec.Record = []Entry{}
	}
	for _, l := range st.Lanes {
		rl := RecordLane{ID: l.ID, Key: l.Key, SpecStep: l.SpecStep, StepTitle: l.StepTitle, Stage: l.Stage,
			Branch: l.Branch, BaseSHA: l.BaseSHA, HeadSHA: l.HeadSHA, Receipts: l.Receipts, Verdicts: []RecordVerdict{},
			Resolves: []string{}, PR: l.PR, Landing: l.Landing, HandBack: l.HandBack}
		if rl.Receipts == nil {
			rl.Receipts = []ReceiptRecord{}
		}
		for _, r := range l.Validation {
			for _, v := range r.Validators {
				if v.Verdict == "" {
					continue
				}
				rl.Verdicts = append(rl.Verdicts, RecordVerdict{Round: r.Round, HeadSHA: r.HeadSHA, Role: v.Role,
					Verdict: v.Verdict, Pass: v.Pass, Return: v.Return, Route: v.Route})
			}
		}
		for _, r := range l.Resolves {
			rl.Resolves = append(rl.Resolves, r.Issue)
		}
		rec.Lanes = append(rec.Lanes, rl)
	}
	return rec
}

// ReadRecord reads a run's record.
func ReadRecord(repoRoot, runID string) (RunRecord, error) {
	st, err := ReadState(repoRoot, runID)
	if err != nil {
		return RunRecord{}, err
	}
	return recordOf(st), nil
}

// LatestRun names the run a record read addresses when none is named: the one
// run in progress when there is one, else the most recently started run in
// this checkout. Several runs in progress are refused, naming them.
func LatestRun(repoRoot string) (string, error) {
	runs, err := Runs(repoRoot)
	if err != nil {
		return "", err
	}
	if len(runs) == 0 {
		return "", refuse(StageRecord, "", "", "no run in this checkout", "start one with `abcd build <itd-N>`")
	}
	var live []string
	for _, st := range runs {
		if !st.Complete() {
			live = append(live, st.RunID)
		}
	}
	switch len(live) {
	case 0:
		return runs[len(runs)-1].RunID, nil
	case 1:
		return live[0], nil
	}
	return "", refuse(StageRecord, "", "", fmt.Sprintf("%d runs are in progress: %v", len(live), live), "name one with --run")
}

// TranscriptCapturer captures one transcript by path into the history store,
// as `abcd history capture <path>` does, and reports what it stored.
type TranscriptCapturer func(path string) (Transcript, error)

// maxTranscriptPaths caps the transcripts one call captures.
const maxTranscriptPaths = 256

// CaptureTranscripts captures each path into the history store through
// capture, one capture per path, and records each in the run's state. It is
// refused on a run that is not complete: the record's transcripts are the
// run's, captured at its end. A capture that fails stops the call: the ones
// before it are recorded, and the refusal names the path that failed, so the
// call can be made again with the rest. A path captured before is captured
// again (the store's capture is idempotent) and recorded once.
func CaptureTranscripts(repoRoot, runID string, paths []string, capture TranscriptCapturer, o Options) (RunRecord, error) {
	switch {
	case len(paths) == 0:
		return RunRecord{}, refuse(StageRecord, "", "", "no transcript path was named", "name each transcript with --transcript <path>")
	case len(paths) > maxTranscriptPaths:
		return RunRecord{}, refuse(StageRecord, "", "", fmt.Sprintf("%d transcript paths is more than one call captures (%d)", len(paths), maxTranscriptPaths),
			"capture them in several calls")
	case capture == nil:
		return RunRecord{}, errors.New("no transcript capturer")
	}
	var rec RunRecord
	var failed error
	err := mutate(repoRoot, runID, func(_ *os.Root, st *State) (bool, error) {
		if !st.Complete() {
			return false, refuse(StageRecord, "", "", st.RunID+" is not complete, and its transcripts are captured at its end",
				"finish the run with `abcd implement step`, then capture its transcripts")
		}
		now := o.now()
		changed := false
		for _, p := range paths {
			t, err := capture(p)
			if err != nil {
				failed = refuse(StageRecord, "", "", fmt.Sprintf("the history capture of a transcript failed: %v", err),
					"settle what the capture names, then capture that transcript and the ones after it again")
				break
			}
			t.At = now
			if i := slices.IndexFunc(st.Transcripts, func(o Transcript) bool { return o.Path == t.Path }); i >= 0 {
				st.Transcripts[i] = t
			} else {
				st.Transcripts = append(st.Transcripts, t)
			}
			how := "stored"
			if !t.Wrote {
				how = "already stored"
			}
			st.Record = append(st.Record, Entry{At: now, Stage: StageTranscript, Note: fmt.Sprintf("captured %s into the history store as session %s (%s)", t.Path, t.Session, how)})
			changed = true
		}
		if changed {
			st.UpdatedAt = now
		}
		rec = recordOf(*st)
		return changed, nil
	})
	if err != nil {
		return RunRecord{}, err
	}
	return rec, failed
}
