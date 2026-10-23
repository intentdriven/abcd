package matrix

// trial.go is one copy's record (decision 4, the trial interface of the SOTA
// note's path 2): from a frozen commit, a task and a cell to what came back.
// It carries the runners' role receipts, the routes' receipts and the fallback
// receipts as those packages write them (itd-166's asked and served sides,
// decision 14), never a parallel shape, beside the copy's own outcome.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// TrialSchema is the trial record's schema version.
const TrialSchema = 1

// A trial record is read at a trust boundary (an imported Harbor or Inspect
// document becomes one), so what it counts and says is bounded: a token count
// to maxTokens (about a trillion, past any copy's spend, and small enough that
// any sum a mapping takes cannot overflow), a time to maxMS milliseconds, a
// score to maxTests tests, abcd's account of a status to maxReasonBytes, and
// each list of receipts to maxReceipts.
const (
	maxTokens      = 1 << 40
	maxMS          = 1 << 40
	maxTests       = 1 << 20
	maxReasonBytes = 1 << 10
	maxReceipts    = 256
)

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
	// StatusExcluded: the copy broke a rule and is named. One that carries a
	// fallback receipt counts as its cell's failure, never the model's pass
	// (the acceptance criterion on fallback copies); one the compliance check
	// excluded broke a rule of the run rather than failing the task, so it is
	// counted apart, neither a pass nor a failure (rule-breaking runs are
	// rejected before ranking; dcn-2610230216139961).
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
	// Reason is abcd's own account of the status: one line of plain text of
	// at most maxReasonBytes.
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
// attempt, or one excluded for carrying a fallback receipt (a fallback is never
// the model's pass). A copy the compliance check excluded is not a failure.
func (t TrialRecord) Failed() bool {
	return t.Status == StatusFailed || t.Status == StatusExcluded && len(t.Fallbacks) > 0
}

// Apart reports whether the copy is reported apart from the cell's ranking,
// neither a pass nor a failure: one the compliance check excluded, or one it
// could not examine.
func (t TrialRecord) Apart() bool {
	return t.Status == StatusUnexamined || t.Status == StatusExcluded && len(t.Fallbacks) == 0
}

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
	if s := t.Score; s != nil {
		if s.Total < 0 || s.Passed < 0 || s.Passed > s.Total || s.Total > maxTests {
			return fmt.Errorf("the trial %s's score %d of %d is not a score of at most %d tests", t.Key, s.Passed, s.Total, maxTests)
		}
		if err := line("score.detail", s.Detail, maxReasonBytes); err != nil {
			return fmt.Errorf("the trial %s: %w", t.Key, err)
		}
	}
	if t.ModelTimeMS < 0 || t.GateWaitMS < 0 || t.ModelTimeMS > maxMS || t.GateWaitMS > maxMS {
		return fmt.Errorf("the trial %s's times are from 0 to %d ms", t.Key, int64(maxMS))
	}
	if !t.FinishedAt.IsZero() && t.FinishedAt.Before(t.StartedAt) {
		return fmt.Errorf("the trial %s finished before it started", t.Key)
	}
	for field, v := range map[string]string{"harness_version": t.HarnessVersion, "model_asked": t.ModelAsked,
		"model_served": t.ModelServed, "served_provider": t.ServedProvider} {
		if err := optional(field, v); err != nil {
			return fmt.Errorf("the trial %s: %w", t.Key, err)
		}
	}
	if err := line("reason", t.Reason, maxReasonBytes); err != nil {
		return fmt.Errorf("the trial %s: %w", t.Key, err)
	}
	if err := checkUsage(t.Usage); err != nil {
		return fmt.Errorf("the trial %s: %w", t.Key, err)
	}
	if err := checkCost(t.Cost); err != nil {
		return fmt.Errorf("the trial %s: %w", t.Key, err)
	}
	if err := settingsBounded("settings", t.Settings); err != nil {
		return fmt.Errorf("the trial %s: %w", t.Key, err)
	}
	for field, v := range map[string]any{"roles": t.Roles, "calls": t.Calls, "fallbacks": t.Fallbacks, "unpinned": t.Unpinned} {
		if err := plain(field, reflect.ValueOf(v)); err != nil {
			return fmt.Errorf("the trial %s: %w", t.Key, err)
		}
	}
	return nil
}

// checkUsage holds every token count to zero or more and at most maxTokens.
func checkUsage(u oracle.Usage) error {
	for field, n := range map[string]int{"prompt_tokens": u.PromptTokens, "completion_tokens": u.CompletionTokens,
		"reasoning_tokens": u.ReasoningTokens, "cache_read_tokens": u.CacheReadTokens, "cache_write_tokens": u.CacheWriteTokens} {
		if !countOK(int64(n)) {
			return fmt.Errorf("usage.%s is %d; a token count is from 0 to %d", field, n, int64(maxTokens))
		}
	}
	return nil
}

func countOK(n int64) bool { return n >= 0 && n <= maxTokens }

// checkCost holds a cost to a readable figure (oracle.ReadableCost) in a unit
// a route reports.
func checkCost(c *oracle.Cost) error {
	if c == nil {
		return nil
	}
	if c.Unit != oracle.CostUSD && c.Unit != oracle.CostCredits {
		return fmt.Errorf("cost.unit %q is not %s or %s", clip(c.Unit), oracle.CostUSD, oracle.CostCredits)
	}
	if !oracle.ReadableCost(c.Amount) {
		return fmt.Errorf("cost.amount %v is not a readable cost", c.Amount)
	}
	return nil
}

// line holds a prose field a record may leave empty to one line of plain
// text: valid UTF-8, at most limit bytes, and nothing termsafe would mask (a
// newline, a control, bidirectional or zero-width character).
func line(field, s string, limit int) error {
	if len(s) > limit || !utf8.ValidString(s) || termsafe.Sanitize(s) != s {
		return fmt.Errorf("%s is longer than %d bytes or is not one line of plain text", field, limit)
	}
	return nil
}

// settingsBounded holds settings as a record carries them to the oracle's own
// check (oracle.CheckSettings: at most oracle.MaxSettings, each a
// parameter-shaped name and a bounded, plain JSON scalar), and each value to
// its bytes as sent, with no space around it, so the record carries a setting
// exactly as the oracle would.
func settingsBounded(field string, s oracle.Settings) error {
	checked, err := oracle.CheckSettings(s)
	if err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}
	for k, v := range s {
		if !bytes.Equal(checked[k], v) {
			return fmt.Errorf("%s: the setting %s carries space around its value", field, k)
		}
	}
	return nil
}

var (
	timeType     = reflect.TypeOf(time.Time{})
	rawType      = reflect.TypeOf(json.RawMessage(nil))
	settingsType = reflect.TypeOf(oracle.Settings(nil))
)

// plain walks a receipt the trial carries as its package wrote it, so that no
// field of it, now or added later, escapes the bounds: every string is one
// line of plain text of at most oracle.MaxRouteBytes (the longest a receipt
// carries, a --route verbatim), every count is from 0 to maxTokens, every
// figure is a readable cost, every list holds at most maxReceipts entries,
// and every settings map is bounded as settingsBounded holds it.
func plain(field string, v reflect.Value) error {
	switch {
	case v.Type() == timeType:
		return nil
	case v.Type() == settingsType:
		return settingsBounded(field, v.Interface().(oracle.Settings))
	case v.Type() == rawType:
		if v.Len() > oracle.MaxSettingBytes || !json.Valid(v.Bytes()) {
			return fmt.Errorf("%s is not JSON of at most %d bytes", field, oracle.MaxSettingBytes)
		}
		return nil
	}
	switch v.Kind() {
	case reflect.String:
		return line(field, v.String(), oracle.MaxRouteBytes)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if !countOK(v.Int()) {
			return fmt.Errorf("%s is %d; a count is from 0 to %d", field, v.Int(), int64(maxTokens))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if v.Uint() > maxTokens {
			return fmt.Errorf("%s is %d; a count is at most %d", field, v.Uint(), int64(maxTokens))
		}
	case reflect.Float32, reflect.Float64:
		if f := v.Float(); math.IsNaN(f) || !oracle.ReadableCost(f) {
			return fmt.Errorf("%s is %v; a figure is zero or more and finite", field, f)
		}
	case reflect.Bool:
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			return plain(field, v.Elem())
		}
	case reflect.Slice, reflect.Array:
		if v.Len() > maxReceipts {
			return fmt.Errorf("%s holds %d entries; at most %d", field, v.Len(), maxReceipts)
		}
		for i := range v.Len() {
			if err := plain(fmt.Sprintf("%s[%d]", field, i), v.Index(i)); err != nil {
				return err
			}
		}
	case reflect.Map:
		if v.Len() > maxReceipts {
			return fmt.Errorf("%s holds %d entries; at most %d", field, v.Len(), maxReceipts)
		}
		for it := v.MapRange(); it.Next(); {
			k := fmt.Sprintf("%s[%q]", field, clip(fmt.Sprint(it.Key())))
			if err := plain(k, it.Key()); err != nil {
				return err
			}
			if err := plain(k, it.Value()); err != nil {
				return err
			}
		}
	case reflect.Struct:
		for i := range v.NumField() {
			sf := v.Type().Field(i)
			if !sf.IsExported() {
				continue
			}
			if err := plain(field+"."+sf.Name, v.Field(i)); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("%s is a %s, which a trial record does not carry", field, v.Kind())
	}
	return nil
}
