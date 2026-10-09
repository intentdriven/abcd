// Package runner runs a delegated role through a command-line harness the
// operator named, with the same brief, inputs and output contract the host's
// own sub-agent gets (itd-2609201916056194, spc-2609221533057881).
//
// A role's route is roles.<role>.runner in the layered configuration
// (config.go): host, the default, or a runner this machine enabled under
// runner.<name>. The shipped runners are the claude CLI in print mode with the
// bare flag (claude.go) and opencode's run mode (opencode.go). Each is started
// as a process (proc.go): the argv is a vector and never a shell line, the
// binary is resolved on PATH and refused inside the repository, the
// environment is the parent's scrubbed of every git repository-selection and
// config-injection variable, stdin is the null device, output is bounded, and
// a run past its time is killed with the process group it leads.
//
// The fallback is decided in one place (dispatch.go): an absent, refusing,
// failing or unparsable runner, or an answer the contract's validator refuses,
// hands the role to the host session, or with no host session to the host the
// operator configured (runner.fallback_host), and writes one receipt naming
// the role, the runner asked for, the reason and the route that ran. Tally
// counts those receipts per runner and per role for the run's summary.
//
// A harness whose provider refused the run at a rate or usage limit fails
// with ReasonRateLimited; a dispatcher that pauses on it (the loop's process
// driver) hands the response back instead of falling back, so the loop can
// end the run's window early (itd-2609201925079472 criterion 8). A route
// reports its remaining quota through QuotaFor (quota.go) for the budget a run
// checks before it starts (criterion 7).
//
// The package never prints. It starts processes only through Dispatch and the
// adapters' Run, and never logs a harness in: a harness's credential is its
// own, read from its own environment or configuration, and abcd never puts one
// on a command line, in an error or in a receipt.
package runner

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// The routes a role may name. Host is the default.
const (
	Host     = "host"
	Claude   = "claude"
	OpenCode = "opencode"
)

// runnerNames are the runners this build ships, in the order a refusal names
// them.
var runnerNames = []string{Claude, OpenCode}

// DefaultTimeout bounds one run when the request names no timeout.
const DefaultTimeout = 60 * time.Minute

// Request is what a role is handed: the same the host sub-agent is handed.
type Request struct {
	// Role is the agent the run plays: an agent in the roster, or the
	// implementer.
	Role string
	// Brief is the absolute path of the brief the role follows; it states the
	// task and the output contract.
	Brief string
	// Receipt is the absolute path the contract says the answer is written to.
	Receipt string
	// Dir is the absolute path of the working tree the role runs in.
	Dir string
	// Checkout is the absolute path of the checkout the run belongs to when
	// Dir is a worktree of it (a lane's worktree lives outside it); a program
	// inside it is repository content and is never run, as one inside Dir is
	// not. Empty when Dir is the checkout itself.
	Checkout string
	// Tools are the tools the role's contract grants, granted without a
	// prompt; none granted when empty.
	Tools []string
	// SessionID names the run's transcript record when the harness reports no
	// session of its own.
	SessionID string
	// Timeout bounds the run; DefaultTimeout when zero.
	Timeout time.Duration
}

// Answer is the one answer shape every adapter parses its harness's events
// into. The contract's own answer is the receipt file; this is what the
// harness said about the run.
type Answer struct {
	// Text is the harness's final message.
	Text string
	// Model is the model the harness reports, or the one it was asked for when
	// it reports none.
	Model string
	// SessionID is the harness's own session id, "" when it reports none.
	SessionID string
}

// Runner is one route a role can run through.
type Runner interface {
	// Name is the route's name, as roles.<role>.runner spells it.
	Name() string
	// Run runs the role and returns the parsed answer and the transcript, the
	// harness's raw event stream. A failure is a *Failure; the transcript is
	// returned whenever the harness produced one, failure or not.
	Run(ctx context.Context, req Request) (Answer, []byte, error)
}

// Reason is why a runner did not answer: the kinds the fallback records.
type Reason string

const (
	// ReasonAbsent: the runner is not enabled on this machine, or its binary
	// is not on PATH or is refused.
	ReasonAbsent Reason = "absent"
	// ReasonRefused: the harness ran and reported that it refused or erred.
	ReasonRefused Reason = "refused"
	// ReasonFailed: the harness could not start, exited non-zero, ran out of
	// time or overran its output bound.
	ReasonFailed Reason = "failed"
	// ReasonUnparsable: the harness's output is not its structured events.
	ReasonUnparsable Reason = "unparsable"
	// ReasonInvalid: the answer failed the contract's validator.
	ReasonInvalid Reason = "invalid"
	// ReasonRateLimited: the harness reported that its provider refused the
	// run at a rate or usage limit (itd-2609201925079472 criterion 8). A
	// dispatcher that pauses on it (PauseOnRateLimit) hands it back rather
	// than falling back, since every route of a run spends the same budget.
	ReasonRateLimited Reason = "rate-limited"
)

// Failure is a runner's failure. Detail is written by abcd, never copied from
// the harness's output, so it cannot carry anything the harness printed, a
// credential it echoed included.
type Failure struct {
	Runner string
	Reason Reason
	Detail string
	// cause is the launch stage that failed (ErrNotOnPath, ErrRefused,
	// ErrNotStarted), nil for every other failure; errors.Is reads it.
	cause error
}

func (f *Failure) Error() string {
	return fmt.Sprintf("runner %s %s: %s", f.Runner, f.Reason, f.Detail)
}

// Unwrap is the launch stage that failed, nil when the failure is not one.
func (f *Failure) Unwrap() error { return f.cause }

func fail(runner string, reason Reason, format string, args ...any) *Failure {
	return &Failure{Runner: runner, Reason: reason, Detail: fmt.Sprintf(format, args...)}
}

// failAt is fail for a launch stage: the same failure, carrying cause.
func failAt(cause error, runner string, reason Reason, format string, args ...any) *Failure {
	f := fail(runner, reason, format, args...)
	f.cause = cause
	return f
}

var (
	roleRe    = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
	sessionRe = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)
	// modelRe is a model id as a harness reports it (claude-opus-4-6,
	// provider/model, a bracketed context suffix): bounded and plain, as a
	// session id is, since it is written into the run's state and records.
	modelRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}(\[[A-Za-z0-9._-]{1,16}\])?$`)
	// toolRe is one tool as a contract names it (Read, Bash(git log:*)). A
	// comma or a line break would split the one flag the tools travel in.
	toolRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(\([^,\r\n()]{1,120}\))?$`)
)

// check refuses a request no runner may be started with, before any launch.
func (r Request) check() error {
	if !roleRe.MatchString(r.Role) {
		return fmt.Errorf("runner: role %q is not a plain lower-case name", termsafe.Sanitize(r.Role))
	}
	if r.Checkout != "" && (!filepath.IsAbs(r.Checkout) || filepath.Clean(r.Checkout) != r.Checkout) {
		return fmt.Errorf("runner: the checkout %q is not a clean absolute path", termsafe.Sanitize(r.Checkout))
	}
	for _, p := range []struct{ name, path string }{{"brief", r.Brief}, {"receipt", r.Receipt}, {"directory", r.Dir}} {
		if !filepath.IsAbs(p.path) || filepath.Clean(p.path) != p.path {
			return fmt.Errorf("runner: the %s %q is not a clean absolute path", p.name, termsafe.Sanitize(p.path))
		}
	}
	for _, t := range r.Tools {
		if !toolRe.MatchString(t) {
			return fmt.Errorf("runner: tool %q is not a tool name as a contract spells it", termsafe.Sanitize(t))
		}
	}
	if !sessionRe.MatchString(r.SessionID) {
		return errors.New("runner: the request names no session id for its transcript")
	}
	if r.Timeout < 0 {
		return errors.New("runner: a negative timeout")
	}
	return nil
}

func (r Request) timeout() time.Duration {
	if r.Timeout == 0 {
		return DefaultTimeout
	}
	return r.Timeout
}

// prompt is the one prompt every route hands a role: the role, the brief and
// the receipt path, which are what the host sub-agent is handed. Everything
// else the role needs is in the brief.
func prompt(r Request) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are the %s agent, started by abcd.\n", r.Role)
	fmt.Fprintf(&b, "Brief: %s\n", r.Brief)
	fmt.Fprintf(&b, "Receipt: %s\n", r.Receipt)
	b.WriteString("Read the brief and follow it exactly: it states your task and the output contract. " +
		"Write your receipt to the Receipt path, in the shape the brief names.\n")
	return b.String()
}
