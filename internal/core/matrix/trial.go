package matrix

// trial.go is one copy's record (decision 4, the trial interface of the SOTA
// note's path 2): from a frozen commit, a task and a cell to what came back.
// It carries the runners' role receipts, the routes' receipts and the fallback
// receipts as those packages write them (itd-166's asked and served sides,
// decision 14), never a parallel shape, beside the copy's own outcome.

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TrialSchema is the trial record's schema version.
const TrialSchema = 1

// TrialKey is what a copy is a copy of: the task (for a code task, the task
// that builds its intent), the cell and the repeat, counted from one.
type TrialKey struct {
	Task   string `json:"task"`
	Cell   string `json:"cell"`
	Repeat int    `json:"repeat"`
}

// String is the key as one token.
func (k TrialKey) String() string { return k.Task + "/" + k.Cell + "/" + strconv.Itoa(k.Repeat) }

// rel is the trial record's path in its run's directory.
func (k TrialKey) rel() string {
	return "trials/" + k.Task + "/" + k.Cell + "/" + strconv.Itoa(k.Repeat) + ".json"
}

func (k TrialKey) validate() error {
	if !validID(k.Task) || !validID(k.Cell) || k.Repeat < 1 || k.Repeat > MaxRepeats {
		return fmt.Errorf("%q is not a trial key (task and cell ids, a repeat from 1 to %d)", clip(k.String()), MaxRepeats)
	}
	return nil
}

// TrialStatus is how a copy came out.
type TrialStatus string

const (
	// StatusPassed: the copy passed its judging (a code copy, its hidden
	// tests).
	StatusPassed TrialStatus = "passed"
	// StatusFailed: the copy was attempted and did not pass, including a
	// runner that failed, refused or answered unparsably inside its cell.
	StatusFailed TrialStatus = "failed"
	// StatusNotStarted: the copy never started (a provider down before it
	// began, decision 6); retried on the same pin and never scored.
	StatusNotStarted TrialStatus = "not_started"
	// StatusExcluded: the copy broke a rule (it carries a fallback receipt, or
	// the compliance check excluded it); named, and counted a failure.
	StatusExcluded TrialStatus = "excluded"
	// StatusUnexamined: the compliance check could not examine the copy;
	// reported apart, never ranked as compliant.
	StatusUnexamined TrialStatus = "unexamined"
)

func (s TrialStatus) valid() bool {
	switch s {
	case StatusPassed, StatusFailed, StatusNotStarted, StatusExcluded, StatusUnexamined:
		return true
	}
	return false
}

// Score is a copy's judged result: for a code copy, the hidden tests it passed
// of those run.
type Score struct {
	Passed int    `json:"passed"`
	Total  int    `json:"total"`
	Detail string `json:"detail,omitempty"`
}

// TrialRecord is one copy's record.
type TrialRecord struct {
	SchemaVersion int      `json:"schema_version"`
	Run           string   `json:"run"`
	Key           TrialKey `json:"key"`
	Kind          TaskKind `json:"kind"`
	// Intent is a code task's intent; empty for any other kind.
	Intent       string `json:"intent,omitempty"`
	FrozenCommit string `json:"frozen_commit"`
	// The cell as it ran: its source, the runner (or host) whose harness made
	// the copy and that harness's version, the model asked for and the one
	// that served, the provider that served it, and the settings sent.
	Source         Source          `json:"source"`
	Runner         string          `json:"runner"`
	HarnessVersion string          `json:"harness_version"`
	ModelAsked     string          `json:"model_asked"`
	ModelServed    string          `json:"model_served"`
	ServedProvider string          `json:"served_provider"`
	Settings       oracle.Settings `json:"settings"`
	Status         TrialStatus     `json:"status"`
	// Reason is abcd's own account of the status; it never carries a
	// harness's output.
	Reason string `json:"reason"`
	Score  *Score `json:"score"`
	// The receipts of every role the copy ran and every routed call, both
	// sides each (itd-166), and every fallback (which excludes the copy).
	Roles     []runner.RoleReceipt     `json:"roles"`
	Calls     []oracle.ReceiptRoute    `json:"calls"`
	Fallbacks []runner.FallbackReceipt `json:"fallbacks"`
	// Usage and Cost are the copy's totals; a count or cost not reported is
	// listed in Unpinned.
	Usage      oracle.Usage `json:"usage"`
	Cost       *oracle.Cost `json:"cost"`
	StartedAt  time.Time    `json:"started_at,omitzero"`
	FinishedAt time.Time    `json:"finished_at,omitzero"`
	// ModelTimeMS is the time the models spent, excluding GateWaitMS, the
	// time spent waiting on gates (the code report's criterion).
	ModelTimeMS int64             `json:"model_time_ms"`
	GateWaitMS  int64             `json:"gate_wait_ms"`
	Unpinned    []oracle.Unpinned `json:"unpinned"`
}

// Attempted reports whether the copy counts as an attempt of its cell: every
// copy but one that never started.
func (t TrialRecord) Attempted() bool { return t.Status != StatusNotStarted }

// Passed reports whether the copy counts as its cell's pass.
func (t TrialRecord) Passed() bool { return t.Status == StatusPassed }

// Failed reports whether the copy counts as its cell's failure: a failed
// attempt, or an excluded one (a fallback is never the model's pass).
func (t TrialRecord) Failed() bool { return t.Status == StatusFailed || t.Status == StatusExcluded }

func (t TrialRecord) validate() error {
	if t.SchemaVersion != TrialSchema {
		return fmt.Errorf("trial schema_version %d; this abcd reads %d", t.SchemaVersion, TrialSchema)
	}
	if !runIDRe.MatchString(t.Run) {
		return fmt.Errorf("%q is not a run id", clip(t.Run))
	}
	if err := t.Key.validate(); err != nil {
		return err
	}
	switch t.Kind {
	case KindCode:
		if !intentRe.MatchString(t.Intent) {
			return fmt.Errorf("the code trial %s names no intent", t.Key)
		}
	case KindWriting, KindCheck:
		if t.Intent != "" {
			return fmt.Errorf("the %s trial %s names an intent", t.Kind, t.Key)
		}
	default:
		return fmt.Errorf("the trial %s has kind %q", t.Key, clip(string(t.Kind)))
	}
	if !gitutil.IsFullSHA(t.FrozenCommit) {
		return fmt.Errorf("the trial %s names no frozen commit", t.Key)
	}
	if !t.Source.valid() || text("runner", t.Runner) != nil {
		return fmt.Errorf("the trial %s names no valid source and runner", t.Key)
	}
	if !t.Status.valid() {
		return fmt.Errorf("the trial %s has status %q", t.Key, clip(string(t.Status)))
	}
	if t.Status == StatusNotStarted && (t.Score != nil || t.Usage != (oracle.Usage{}) || t.Cost != nil) {
		return fmt.Errorf("the trial %s never started, so it carries no score, tokens or cost", t.Key)
	}
	if len(t.Fallbacks) > 0 && t.Status != StatusExcluded {
		return fmt.Errorf("the trial %s carries a fallback receipt, so it is excluded, never %s", t.Key, t.Status)
	}
	if s := t.Score; s != nil && (s.Total < 0 || s.Passed < 0 || s.Passed > s.Total) {
		return fmt.Errorf("the trial %s's score %d of %d is not a score", t.Key, s.Passed, s.Total)
	}
	if t.ModelTimeMS < 0 || t.GateWaitMS < 0 {
		return errors.New("a trial's times are not negative")
	}
	if !t.FinishedAt.IsZero() && t.FinishedAt.Before(t.StartedAt) {
		return fmt.Errorf("the trial %s finished before it started", t.Key)
	}
	return nil
}
