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
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/gitutil"
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

// DefaultTimeout bounds one run when the request names no timeout and the
// person set no roles.<role>.timeout (Config.TimeoutFor).
const DefaultTimeout = 60 * time.Minute

// DefaultImplementerTimeout bounds an implementer's run when the person set no
// roles.implementer.timeout. An implementer builds, commits, writes its report
// and runs its scoped definition of done, which waits at the load gate behind
// other preflights on a shared machine; on 2026-11-01 three routed
// implementers were killed at the hour with their work committed and their
// receipts unwritten (iss-2611012202511008).
const DefaultImplementerTimeout = 3 * time.Hour

// MinTimeout and MaxTimeout bound a roles.<role>.timeout the person sets.
const (
	MinTimeout = 5 * time.Minute
	MaxTimeout = 12 * time.Hour
)

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
	// Base is the full object name of the commit the run's working tree was
	// cut from: a lane's base, or the HEAD an interview launched on. A launch
	// that grants the shell needs it, since the approver judges every
	// command against the hazard registry at that commit.
	Base string
	// Declared are the paths the work declares it changes, relative to Dir.
	Declared []string
	// Denied are the tools the role is refused outright, whatever else would
	// grant them: a harness that keeps a deny list takes them there, ahead of
	// every grant.
	Denied []string
	// Model is the <provider>/<model> route this request runs at, over the
	// model the runner's own route names, so one runner serves two reviewers
	// at two classes (itd-2610240922461281 decision 4); "" keeps the route's.
	// It is admitted against the provider's allowlist when the request is
	// dispatched, as a configured route is when it is read, never trusted
	// because a caller stored it.
	Model string
	// SessionID names the run's transcript record when the harness reports no
	// session of its own.
	SessionID string
	// Timeout bounds the run; DefaultTimeout when zero.
	Timeout time.Duration
	// Work is the piece of work the launch is for, as the ledger lists it.
	Work LaunchWork
	// Key is the configured name of the key abcd hands this launch, never its
	// value, which the ledger's route reads; "" when it hands none.
	Key string
}

// Answer is the one answer shape every adapter parses its harness's events
// into. The contract's own answer is the receipt file; this is what the
// harness said about the run.
type Answer struct {
	// Text is the harness's final message.
	Text string
	// Model is the model the harness reports, "" when it reports none: the
	// model the runner asked for is Sides.ModelAsked, and never stands in for
	// the one that answered (itd-166, iss-2610161709506207).
	Model string
	// SessionID is the harness's own session id, "" when it reports none.
	SessionID string
	// Sides is both sides of the run: what the runner asked for, and the
	// token counts, cost and served models the harness's events report.
	Sides
	// counted is which token counts the harness's events carried, read
	// with them, for the record's list of what it could not pin.
	counted counted
	// lost is which summed counts stopped being readable (a sum that would
	// overflow), and costLost whether the summed cost did (a sum that is no
	// longer finite): each stays unread for the rest of the run.
	lost     counted
	costLost bool
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
	// ReasonRateLimited: the harness reported a rate-limit response and did
	// not finish. It is never fallen back on (dispatch.go): every route spends
	// the budget the run's window paces, so the caller ends the window
	// (itd-2609201925079472 criterion 8).
	ReasonRateLimited Reason = "rate-limited"
)

// Quota is what a runner reports of its budget before a run starts
// (itd-2609201925079472 criterion 7): the agent runs it can still start in its
// current window. A runner whose harness counts its budget in another unit
// converts it, or reports none.
type Quota struct {
	Remaining int `json:"remaining"`
}

// QuotaReporter is a Runner that reports its remaining quota. Neither shipped
// runner is one: the claude CLI and opencode expose no remaining quota a
// launch can read before it starts, so a run routed to either names it and
// skips the budget check out loud.
type QuotaReporter interface {
	Quota(ctx context.Context) (Quota, error)
}

// quotaOf asks r for its remaining quota. reported is false when r reports
// none; an error is r's failure to report the one it keeps. The deadline is
// the caller's, held here rather than trusted to the reporter: when ctx ends
// first, quotaOf returns ctx's error and the reporter's answer, whenever it
// comes, is discarded.
func quotaOf(ctx context.Context, r Runner) (q Quota, reported bool, err error) {
	qr, ok := r.(QuotaReporter)
	if !ok {
		return Quota{}, false, nil
	}
	type answer struct {
		q   Quota
		err error
	}
	got := make(chan answer, 1)
	go func() {
		q, err := qr.Quota(ctx)
		got <- answer{q, err}
	}()
	select {
	case a := <-got:
		return a.q, true, a.err
	case <-ctx.Done():
		return Quota{}, true, ctx.Err()
	}
}

// Failure is a runner's failure. Detail is written by abcd, never copied from
// the harness's output, so it cannot carry anything the harness printed, a
// credential it echoed included.
type Failure struct {
	Runner string
	Reason Reason
	Detail string
	// Spent is what a killed launch spent, read from its transcript
	// (ErrKilled, Spend); nil for every other failure.
	Spent *Spend
	// ResetsAt is when a rate limit resets, as the harness reported it; zero
	// when it reported none or the failure is no limit.
	ResetsAt time.Time
	// Seen is what the harness showed of its failure, for a run's evidence
	// file; nil when no launch ran. It stays out of Detail, which is abcd's.
	Seen *HarnessSeen
	// Model is the model route the failed launch asked for, "" for the
	// harness's own default.
	Model string
	// cause is the launch stage that failed (ErrNotOnPath, ErrRefused,
	// ErrNotStarted), nil for every other failure; errors.Is reads it.
	cause error
}

// HarnessSeen is what a launch that failed showed, in the terms a person needs
// to learn how a harness behaves at a limit: nothing here is read to decide
// anything. The strings are the harness's own and not yet redacted, so whoever
// keeps them passes them through the scanner first.
type HarnessSeen struct {
	// ExitCode is the process's status, -1 when it was killed or did not exit.
	ExitCode int
	// Shape is the end of its event stream, one line per event, with every
	// identifier dropped and free text reduced to its length.
	Shape []string
	// Stderr is the first line it wrote to stderr.
	Stderr string
	// SilentFor is how long it had written nothing when it ended.
	SilentFor time.Duration
	// Spent says its stream shows tokens or a cost above zero.
	Spent bool
	// APIStatus is the status a failed result names, "" for none.
	APIStatus string
	// Retries counts the retries of the model service its stream reports, and
	// Retrying says the stream ended on one.
	Retries  int
	Retrying bool
}

// Blame is what a failed launch says of the work it was launched for.
type Blame int

// What a failed launch says.
const (
	// BlameNone: nothing: it shows an outage, or no launch ran.
	BlameNone Blame = iota
	// BlameAsk: it does not say; a probe answering shows the account is alive,
	// which leaves the work.
	BlameAsk
	// BlameWork: it shows the model was reached, and the work is what failed.
	BlameWork
)

// NoVerdictError is the live proof's launch ending with no verdict on any check:
// an outage, say, or a limit waited out inside it. It reads as the launch's own
// failure, wrapped.
type NoVerdictError struct {
	// Detail is abcd's own words for how it ended.
	Detail string
	// Seen is what the launch showed.
	Seen *HarnessSeen
}

func (e *NoVerdictError) Error() string { return e.wall().Error() }

// Unwrap reads as the failed launch to whoever does not ask for this type.
func (e *NoVerdictError) Unwrap() error { return e.wall() }

func (e *NoVerdictError) wall() *WallError { return &WallError{WallLaunch, e.Detail} }

// LimitError is a limit a launch met before it could show anything else: a
// limit, never the thing it was launched to check.
type LimitError struct {
	// ResetsAt is the reset the launch named, zero when it named none.
	ResetsAt time.Time
	// Seen is what the launch showed.
	Seen *HarnessSeen
}

func (e *LimitError) Error() string { return "the account has met its rate limit" }

// Blame is what this launch says of the work it was launched for. Only measured
// spend or an answer the contract refused shows the work is what failed. An init
// event shows nothing: it comes before any contact with the model. A launch that
// was retrying the model service, or a result naming a status that is the account's
// or the service's, says nothing of the work, whatever came before. Anything else
// does not say, so the caller may ask a probe.
func (f *Failure) Blame() Blame {
	if f.Reason == ReasonInvalid {
		return BlameWork
	}
	s := f.Seen
	switch {
	case s == nil, serviceStatus(s.APIStatus), f.Outage():
		return BlameNone
	case s.Spent:
		return BlameWork
	}
	return BlameAsk
}

// Outage reports whether this launch was retrying the model service when it ended,
// or ran out of retries having spent nothing.
func (f *Failure) Outage() bool {
	s := f.Seen
	return s != nil && (s.Retrying || (s.Retries > 0 && !s.Spent))
}

// serviceStatus reports whether an API status is the account's or the service's,
// never the work's: 401, 403, 429 and any 5xx.
func serviceStatus(status string) bool {
	return status == "401" || status == "403" || status == "429" || (len(status) == 3 && status[0] == '5')
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
	// toolRe is one tool as a contract names it (Read, Bash(git log:*),
	// Read(//abs/dir/**)). A comma or a line break would split the one flag
	// the tools travel in. The specifier is bounded to hold an absolute path
	// (Go's regexp bounds a repeat at 1000).
	toolRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(\([^,\r\n()]{1,1000}\))?$`)
)

// Check refuses a request no runner may be started with, as every launch
// does before it starts anything: a caller that must know whether a runner
// was started asks it first.
func (r Request) Check() error { return r.check() }

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
	for _, t := range slices.Concat(r.Tools, r.Denied) {
		if !toolRe.MatchString(t) {
			return fmt.Errorf("runner: tool %q is not a tool name as a contract spells it", termsafe.Sanitize(t))
		}
	}
	if r.Base != "" && !gitutil.IsFullSHA(r.Base) {
		return fmt.Errorf("runner: the base %q is not a commit's full object name", termsafe.Sanitize(r.Base))
	}
	if r.Model != "" {
		provider, model, ok := strings.Cut(r.Model, "/")
		if !ok || provider == "" || model == "" || !modelRe.MatchString(r.Model) {
			return fmt.Errorf("runner: the request's model %q is not a <provider>/<model> route of at most 128 plain characters",
				termsafe.Sanitize(r.Model))
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

// modelOr is the model route the request runs at: its own when it names one,
// else route, the model the runner's configuration names ("" for the
// harness's own default).
func (r Request) modelOr(route string) string {
	if r.Model != "" {
		return r.Model
	}
	return route
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
	fmt.Fprintf(&b, "Time: this run is stopped %s after it starts, receipt written or not, "+
		"not counting up to as long again that it waits queued at abcd's load gate; "+
		"leave time to write your receipt.\n", r.timeout())
	return b.String()
}
