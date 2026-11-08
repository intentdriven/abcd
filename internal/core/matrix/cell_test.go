package matrix

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

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

func down() runner.Outcome {
	return runner.Outcome{Failure: &runner.Failure{Runner: "opencode", Reason: runner.ReasonFailed, Detail: "503",
		Seen: &runner.HarnessSeen{APIStatus: "503"}}}
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
	s := &scripted{outs: []runner.Outcome{down(), down(), answered()}}
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

	s = &scripted{outs: []runner.Outcome{down()}}
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
	s := &scripted{outs: []runner.Outcome{down()}}
	_, err := runCopy(context.Background(), openRouterPin, 3, func(context.Context, int) error { return stop }, s.make)
	if !errors.Is(err, stop) || len(s.pins) != 1 {
		t.Fatalf("the wait's end ends the copy as the run's error: %v, %d tries", err, len(s.pins))
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
