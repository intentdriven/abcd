package matrix

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// Making one copy inside its cell (spc-2610161949559050 step 3): the cell is
// the route under test, so a runner's failure is the cell's failure, an
// outage before the copy began is retried on the same pin and never scored,
// and a clone bound for a provider passes the launch bundle's scan first.

var openRouterPin = CellPin{
	Cell: "cheap", Role: RoleBuilder, Source: SourceOpenRouter, Route: "opencode:openrouter/qwen/qwen3-coder",
	Runner: "opencode", Model: "openrouter/qwen/qwen3-coder",
	ProviderPin: &ProviderPin{Order: []string{"chutes"}, DataCollection: "deny"},
}

// scripted is a cell's dispatch that answers from a script, one outcome per
// call, recording the pin each call was made on.
type scripted struct {
	outs []runner.Outcome
	pins []CellPin
}

func (s *scripted) make(_ context.Context, pin CellPin) (runner.Outcome, error) {
	s.pins = append(s.pins, pin)
	out := s.outs[0]
	if len(s.outs) > 1 {
		s.outs = s.outs[1:]
	}
	return out, nil
}

// openCodeDown is opencode 1.18.31's run mode meeting a provider that is down,
// as its source writes it (cli/cmd/run.ts emits a session.error as an "error"
// event carrying NamedError.toObject(), whose APIError data names the HTTP
// status as statusCode; session/retry.ts has already retried the 5xx): a
// step started, the error event, and a non-zero exit.
const openCodeDown = `{"type":"step_start","timestamp":1,"sessionID":"ses_down","part":{"type":"step-start"}}
{"type":"error","timestamp":2,"sessionID":"ses_down","error":{"name":"APIError","data":{"message":"Service Unavailable","statusCode":503,"isRetryable":true}}}`

// realCell is a copy's dispatch through the runners' real cell mode: the
// builder role routed to opencode on the person's own layer, and on PATH a
// fake opencode that answers every launch with stream and exits with exit,
// counting its launches in the file launches names.
type realCell struct {
	d        *runner.Dispatcher
	req      runner.Request
	launches string
}

func newRealCell(t *testing.T, stream string, exit int) *realCell {
	t.Helper()
	bin, lane, dir := t.TempDir(), t.TempDir(), t.TempDir()
	launches := filepath.Join(lane, "launches")
	quoted := "'" + strings.ReplaceAll(stream, "'", `'\''`) + "'"
	script := fmt.Sprintf("#!/bin/sh\necho launched >> '%s'\nprintf '%%s\\n' %s\nexit %d\n", launches, quoted, exit)
	if err := os.WriteFile(filepath.Join(bin, "opencode"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	for _, v := range []string{"OPENCODE_CONFIG", "OPENCODE_CONFIG_CONTENT", "OPENCODE_CONFIG_DIR"} {
		t.Setenv(v, "")
	}
	r := layered.Roots{Repo: t.TempDir(), Home: t.TempDir()}
	machine := abcdhome.Path(r.Home, "config.json")
	if err := os.MkdirAll(filepath.Dir(machine), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(machine, []byte(`{"roles":{"implementer":{"runner":"opencode"}},"runner":{"opencode":{}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := runner.Load(r)
	if err != nil {
		t.Fatal(err)
	}
	return &realCell{
		d: &runner.Dispatcher{Config: c, HostSession: true, Cell: true, Transcripts: discard{},
			Validate: func(string, runner.Request, runner.Answer) error { return nil },
			Record:   func(runner.FallbackReceipt) error { t.Fatal("a cell's dispatch records no fallback"); return nil }},
		req: runner.Request{Role: "implementer", Brief: filepath.Join(lane, "brief.md"), Receipt: filepath.Join(lane, "receipt.json"),
			Dir: dir, SessionID: "mx-cell-implementer", Timeout: 20 * time.Second},
		launches: launches,
	}
}

func (c *realCell) make(ctx context.Context, _ CellPin) (runner.Outcome, error) {
	return c.d.Dispatch(ctx, c.req)
}

// count is how many times the fake opencode was launched.
func (c *realCell) count(t *testing.T) int {
	t.Helper()
	raw, err := os.ReadFile(c.launches)
	if errors.Is(err, os.ErrNotExist) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Count(string(raw), "launched")
}

// discard is a transcript store that keeps nothing.
type discard struct{}

func (discard) Store(string, runner.Request, runner.Answer, []byte) error { return nil }

// down is the outcome a real cell dispatch returns when the cell's provider is
// down: the opencode adapter's own failure, not one built by hand.
func down(t *testing.T) runner.Outcome {
	t.Helper()
	out, err := newRealCell(t, openCodeDown, 1).make(context.Background(), openRouterPin)
	if err != nil {
		t.Fatal(err)
	}
	if out.Failure == nil {
		t.Fatalf("the fake provider outage failed nothing: %+v", out)
	}
	return out
}

// TestAnOpenRouterOutageThroughOpenCodeIsNotStarted: a copy whose cell is
// dispatched for real, through the opencode adapter in the runners' cell mode,
// to a provider that stays down, is tried on its pin each time and left not
// started, never scored as the model's failure.
func TestAnOpenRouterOutageThroughOpenCodeIsNotStarted(t *testing.T) {
	c := newRealCell(t, openCodeDown, 1)
	res, err := runCopy(context.Background(), openRouterPin, 3, noWait, c.make)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusNotStarted || res.NotStarted != 3 || c.count(t) != 3 {
		t.Fatalf("the outage is retried on the pin and the copy not started: %+v, %d launches", res, c.count(t))
	}
	if f := res.Outcome.Failure; f == nil || f.Runner != "opencode" || !f.ProviderDown() {
		t.Fatalf("the adapter's own failure reads as the provider down: %+v", res.Outcome.Failure)
	}
	if rec := (TrialRecord{Status: res.Status}); rec.Attempted() || rec.Failed() || rec.Passed() {
		t.Fatal("a copy that never started is never scored")
	}
}

// openCodeThrottled is the same run mode meeting a provider that throttles
// the call (OpenRouter answering 429) before any step finished: the error
// event naming the status, and a non-zero exit.
const openCodeThrottled = `{"type":"step_start","timestamp":1,"sessionID":"ses_slow","part":{"type":"step-start"}}
{"type":"error","timestamp":2,"sessionID":"ses_slow","error":{"name":"APIError","data":{"message":"Rate limit exceeded","statusCode":429,"isRetryable":true}}}`

// TestAnOpenRouterThrottleThroughOpenCodeIsNotStarted: a copy whose provider
// throttles every try, with nothing spent, is retried on its pin within the
// try bound and left not started, never scored as the model's failure: the
// throttling is the window's, not the model's.
func TestAnOpenRouterThrottleThroughOpenCodeIsNotStarted(t *testing.T) {
	c := newRealCell(t, openCodeThrottled, 1)
	res, err := runCopy(context.Background(), openRouterPin, 3, noWait, c.make)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusNotStarted || res.NotStarted != 3 || c.count(t) != 3 {
		t.Fatalf("the throttling is retried on the pin and the copy not started: %+v, %d launches", res, c.count(t))
	}
	if f := res.Outcome.Failure; f == nil || f.Seen == nil || f.Seen.APIStatus != "429" || !f.ProviderDown() {
		t.Fatalf("the adapter's own 429 reads as the provider down: %+v", res.Outcome.Failure)
	}
	if rec := (TrialRecord{Status: res.Status}); rec.Attempted() || rec.Failed() || rec.Passed() {
		t.Fatal("a copy that never started is never scored")
	}
}

func answered() runner.Outcome {
	return runner.Outcome{Answer: &runner.Answer{}, Receipt: runner.RoleReceipt{Role: "implementer"}}
}

func noWait(context.Context, int) error { return nil }

// TestARunnerFailureInACellIsTheCellsFailure: a runner that failed, refused or
// answered unparsably makes the copy a failed attempt of its cell, named with
// the runner's reason, after one try.
func TestARunnerFailureInACellIsTheCellsFailure(t *testing.T) {
	for _, reason := range []runner.Reason{runner.ReasonFailed, runner.ReasonRefused, runner.ReasonUnparsable, runner.ReasonAbsent} {
		s := &scripted{outs: []runner.Outcome{{Failure: &runner.Failure{Runner: "opencode", Reason: reason, Detail: "it said no"}}}}
		res, err := runCopy(context.Background(), openRouterPin, 3, noWait, s.make)
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != StatusFailed || !strings.Contains(res.Reason, string(reason)) || len(s.pins) != 1 {
			t.Fatalf("%s: the copy is its cell's failure after one try: %+v, %d tries", reason, res, len(s.pins))
		}
		if res.Made() {
			t.Fatalf("%s: a failed copy is not made", reason)
		}
	}
}

// TestACopyThatFellBackIsExcludedAsTheCellsFailure: a dispatch that carries a
// fallback receipt never counts as the model's work: the copy is excluded with
// the receipt kept, which the trial record counts as the cell's failure.
func TestACopyThatFellBackIsExcludedAsTheCellsFailure(t *testing.T) {
	fb := runner.FallbackReceipt{Role: "implementer", Asked: "opencode", Reason: runner.ReasonFailed, Ran: runner.Host}
	s := &scripted{outs: []runner.Outcome{{Handoff: true, Fallback: &fb}}}
	res, err := runCopy(context.Background(), openRouterPin, 3, noWait, s.make)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusExcluded || len(res.Fallbacks) != 1 || res.Made() {
		t.Fatalf("a fallen-back copy is excluded with its receipt: %+v", res)
	}
	if !(TrialRecord{Status: res.Status, Fallbacks: res.Fallbacks}).Failed() {
		t.Fatal("the trial record counts it as the cell's failure")
	}
}

// TestAnOutageIsRetriedOnTheSamePinAndNotScored: a provider down before the
// copy began is retried, each time on the identical pin, and the attempts are
// counted as not started; a copy whose every try met the outage is not
// started, which is never scored, and one that then answers is made.
func TestAnOutageIsRetriedOnTheSamePinAndNotScored(t *testing.T) {
	s := &scripted{outs: []runner.Outcome{down(t), down(t), answered()}}
	res, err := runCopy(context.Background(), openRouterPin, 3, noWait, s.make)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Made() || res.NotStarted != 2 || len(s.pins) != 3 {
		t.Fatalf("the copy is made on the third try, two not started: %+v, %d tries", res, len(s.pins))
	}
	for i, p := range s.pins {
		if !reflect.DeepEqual(p, openRouterPin) {
			t.Fatalf("try %d ran on another pin: %+v", i+1, p)
		}
	}

	s = &scripted{outs: []runner.Outcome{down(t)}}
	res, err = runCopy(context.Background(), openRouterPin, 3, noWait, s.make)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusNotStarted || res.NotStarted != 3 || len(s.pins) != 3 {
		t.Fatalf("a copy the outage outlasted is not started after three tries: %+v, %d tries", res, len(s.pins))
	}
	if rec := (TrialRecord{Status: res.Status}); rec.Attempted() || rec.Failed() || rec.Passed() {
		t.Fatal("a copy that never started is never scored")
	}
}

// TestAnOutageWaitIsBoundByTheRun: the wait between tries ends with the run,
// and its end is the run's error, not a score.
func TestAnOutageWaitIsBoundByTheRun(t *testing.T) {
	stop := errors.New("the run was stopped")
	s := &scripted{outs: []runner.Outcome{down(t)}}
	_, err := runCopy(context.Background(), openRouterPin, 3, func(context.Context, int) error { return stop }, s.make)
	if !errors.Is(err, stop) || len(s.pins) != 1 {
		t.Fatalf("the wait's end ends the copy as the run's error: %v, %d tries", err, len(s.pins))
	}
}

// TestACopyStoppedMidLaunchIsTheRunsNotTheCopys: a run stopped while its copy's
// runner is up ends the copy with the run's error, unscored: the runner kills
// the launch because the run ended, which says nothing of the model.
func TestACopyStoppedMidLaunchIsTheRunsNotTheCopys(t *testing.T) {
	c := newRealCell(t, "", 0)
	// The fake opencode stays up until it is killed, as a copy at work does.
	script := fmt.Sprintf("#!/bin/sh\necho launched >> '%s'\nexec /bin/sleep 30\n", c.launches)
	if err := os.WriteFile(filepath.Join(os.Getenv("PATH"), "opencode"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
			if c.count(t) > 0 {
				cancel()
				return
			}
		}
	}()
	res, err := runCopy(ctx, openRouterPin, 3, noWait, c.make)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("the stop ends the copy as the run's error: %v, %+v", err, res)
	}
	if res.Status != "" || res.NotStarted != 0 {
		t.Fatalf("a stopped copy carries no score: %+v", res)
	}
	if n := c.count(t); n != 1 {
		t.Fatalf("a stopped copy is not tried again: %d launches", n)
	}
}

// TestADispatchErrorIsTheRunsNotTheCopys: an error from the dispatch itself (a
// rate limit, a route the configuration refuses) is not scored.
func TestADispatchErrorIsTheRunsNotTheCopys(t *testing.T) {
	limit := errors.New("rate-limited")
	_, err := runCopy(context.Background(), openRouterPin, 3, noWait, func(context.Context, CellPin) (runner.Outcome, error) {
		return runner.Outcome{}, limit
	})
	if !errors.Is(err, limit) {
		t.Fatalf("the dispatch's error reaches the run: %v", err)
	}
}

// TestAProviderBoundCloneWithASecretIsRefused: a clone bound for OpenRouter or
// another provider passes the launch bundle's scan or its cell is refused,
// naming the scan; a local or host cell's clone is not sent anywhere and is
// not scanned.
func TestAProviderBoundCloneWithASecretIsRefused(t *testing.T) {
	scanRoot, clone := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(clone, "config.txt"), []byte("token = ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ab\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, src := range []Source{SourceOpenRouter, SourceProvider} {
		err := admitClone(scanRoot, clone, CellPin{Cell: "cheap", Source: src})
		if err == nil || !strings.Contains(err.Error(), "cheap") || !strings.Contains(err.Error(), "scan") {
			t.Fatalf("%s: a clone with a secret-shaped file refuses its cell: %v", src, err)
		}
		if strings.Contains(err.Error(), "ghp_") {
			t.Fatalf("%s: the refusal never carries the value: %v", src, err)
		}
	}
	for _, src := range []Source{SourceLocal, SourceHost} {
		if err := admitClone(scanRoot, clone, CellPin{Cell: "here", Source: src}); err != nil {
			t.Fatalf("%s: a clone that is not sent is not scanned: %v", src, err)
		}
	}
	// Only a source known to keep the clone on this machine skips the scan: an
	// empty or unknown one is scanned as a provider's would be.
	for _, src := range []Source{"", "elsewhere"} {
		if err := admitClone(scanRoot, clone, CellPin{Cell: "odd", Source: src}); err == nil || !strings.Contains(err.Error(), "scan") {
			t.Fatalf("source %q: a clone whose source is not known to stay here is scanned and refused: %v", src, err)
		}
	}
	clean := t.TempDir()
	if err := os.WriteFile(filepath.Join(clean, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := admitClone(scanRoot, clean, openRouterPin); err != nil {
		t.Fatalf("a clean clone passes: %v", err)
	}
}
