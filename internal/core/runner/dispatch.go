package runner

// dispatch.go is the one place the fallback is decided (spc-2609221533057881
// scope 4). A role routed to a runner runs through it; when the runner is
// absent, refuses, fails, answers unparsably, or its answer fails the
// contract's validator, the role goes to the host session, or with no host
// session to the host the operator configured, and exactly one receipt is
// written for the event. Because the branch and the receipt writer are one,
// the count the run's summary reports is every fallback there was. A runner
// that reports a rate limit is the one failure not fallen back on: it reaches
// the caller as itself, with no receipt.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// maxDetail bounds a validator's refusal as a receipt carries it.
const maxDetail = 300

// Dispatcher runs roles by their routes.
type Dispatcher struct {
	// Config is the runner configuration read at the lane's start.
	Config *Config
	// HostSession is true when a host session drives the loop and can take a
	// role handed back to it.
	HostSession bool
	// Attended is true when the person is at the terminal with no host session
	// (the plain-Terminal interviews, spc-2610030911534855): a role routed to a
	// runner runs there although no runner.fallback_host is configured, and a
	// runner that does not answer stops the run, its fallback receipt naming
	// none as the route that ran, rather than leaving the role nowhere to land.
	// A role on the host still needs a host session or a configured host.
	Attended bool
	// Validate is the contract's validator: the same check the host sub-agent's
	// answer passes. It is handed the runner that ran the role, so a caller
	// whose validator also records the answer (the loop's receipt verifier)
	// can name the route that ran.
	Validate func(runner string, req Request, ans Answer) error
	// Transcripts is where every runner's transcript lands.
	Transcripts TranscriptStore
	// Record appends one fallback receipt to the run's state.
	Record func(FallbackReceipt) error
	// Prepare, when set, readies req's files for the runner about to run it,
	// before each runner starts: the routed one and a fallback host alike. A
	// *Failure it returns is that runner's failure, recorded and fallen back
	// on as a launch's would be, with nothing launched; any other error ends
	// the dispatch.
	Prepare func(runner string, req Request) error
	// Now is the clock the receipts are stamped with; time.Now when nil.
	Now func() time.Time
}

// Outcome is one dispatch's result.
type Outcome struct {
	// Handoff is true when the host session runs the role, exactly as it does
	// with no runner configured: the caller hands the host the brief and
	// awaits its receipt.
	Handoff bool
	// Answer is the runner's parsed answer when a runner ran the role.
	Answer *Answer
	// Fallback is the receipt this dispatch recorded, nil when none.
	Fallback *FallbackReceipt
	// Receipt is the role's receipt block.
	Receipt RoleReceipt
}

// Dispatch runs req by its role's route.
func (d *Dispatcher) Dispatch(ctx context.Context, req Request) (Outcome, error) {
	if d.Config == nil || d.Validate == nil || d.Transcripts == nil || d.Record == nil {
		return Outcome{}, errors.New("runner: a dispatch needs the configuration, the contract's validator, " +
			"the transcript store and the run's receipt writer")
	}
	if err := req.check(); err != nil {
		return Outcome{}, err
	}
	route := d.Config.RouteFor(req.Role)
	landing := Host
	if !d.HostSession {
		landing = d.Config.FallbackHost()
		if landing == "" && !(d.Attended && route.Runner != Host) {
			return Outcome{}, fmt.Errorf("runner: there is no host session and no %s.%s is configured, so a role "+
				"has nowhere to land; set it in the machine's config to the runner that stands in for the host",
				runnerKey, fallbackKey)
		}
	}
	out := Outcome{Receipt: newRoleReceipt(req, route.Runner)}

	if route.Runner == Host {
		if d.HostSession {
			out.Handoff = true
			out.Receipt.Route.Ran = Host
			return out, nil
		}
		ans, err := d.runOn(ctx, landing, req)
		if err != nil {
			return Outcome{}, fmt.Errorf("runner: the configured host %s did not run %s: %w", landing, req.Role, err)
		}
		return out.ran(landing, ans), nil
	}

	ans, err := d.runOn(ctx, route.Runner, req)
	if err == nil {
		return out.ran(route.Runner, ans), nil
	}
	var fl *Failure
	if !errors.As(err, &fl) {
		return Outcome{}, err
	}
	if fl.Reason == ReasonRateLimited {
		// Not a fallback: every route spends the budget the run's window
		// paces, so the role is handed to no one and the caller ends the
		// window (itd-2609201925079472 criterion 8). No receipt is written,
		// since nothing fell back.
		return Outcome{}, err
	}
	fb := FallbackReceipt{At: d.now(), Role: req.Role, Asked: route.Runner, Reason: fl.Reason, Detail: fl.Detail, Ran: landing}
	if landing == route.Runner || landing == "" {
		fb.Ran = none
	}
	if rerr := d.Record(fb); rerr != nil {
		return Outcome{}, fmt.Errorf("runner: the fallback receipt for %s was not recorded: %w", req.Role, rerr)
	}
	out.Fallback = &fb
	if landing == "" {
		return Outcome{}, fmt.Errorf("runner: %s %s for %s, and with no host session and no %s.%s configured nothing else runs it: %s",
			route.Runner, fl.Reason, req.Role, runnerKey, fallbackKey, fl.Detail)
	}
	if fb.Ran == none {
		return Outcome{}, fmt.Errorf("runner: %s %s for %s and it is the configured host, so nothing else can run it: %s",
			route.Runner, fl.Reason, req.Role, fl.Detail)
	}
	if d.HostSession {
		out.Handoff = true
		out.Receipt.Route.Ran = Host
		return out, nil
	}
	ans, err = d.runOn(ctx, landing, req)
	if err != nil {
		return Outcome{}, fmt.Errorf("runner: %s %s for %s, and the configured host %s did not run it either: %w",
			route.Runner, fl.Reason, req.Role, landing, err)
	}
	return out.ran(landing, ans), nil
}

// none is the route a receipt names when nothing could run the role.
const none = "none"

func (o Outcome) ran(route string, ans Answer) Outcome {
	o.Answer = &ans
	o.Receipt.Route.Ran = route
	o.Receipt.Route.Model = ans.Model
	return o
}

func (d *Dispatcher) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now().UTC()
}

// runOn runs req on the named runner, stores its transcript and validates its
// answer. A *Failure is a reason to fall back; any other error is not.
func (d *Dispatcher) runOn(ctx context.Context, name string, req Request) (Answer, error) {
	rc, ok := d.Config.Runner(name)
	if !ok {
		return Answer{}, fail(name, ReasonAbsent, "it is not enabled on this machine (%s.%s in the machine's config)", runnerKey, name)
	}
	if rc.Model != "" {
		// Admitted when the configuration was read; consulted again so a
		// runner never launches on a stale answer.
		if err := d.Config.admitModel(rc.Model); err != nil {
			return Answer{}, fmt.Errorf("runner: %s.%s.%s is %q, %w", runnerKey, name, modelKey, rc.Model, err)
		}
	}
	if d.Prepare != nil {
		if err := d.Prepare(name, req); err != nil {
			return Answer{}, err
		}
	}
	ans, transcript, err := d.Config.adapter(rc).Run(ctx, req)
	if len(transcript) > 0 {
		if serr := d.Transcripts.Store(name, req, ans, transcript); serr != nil {
			return Answer{}, fmt.Errorf("runner: the %s transcript for %s was not stored: %w", name, req.Role, serr)
		}
	}
	if err != nil {
		return Answer{}, err
	}
	if verr := d.Validate(name, req, ans); verr != nil {
		detail := termsafe.Sanitize(verr.Error())
		if len(detail) > maxDetail {
			detail = strings.ToValidUTF8(detail[:maxDetail], "") + "..."
		}
		return Answer{}, fail(name, ReasonInvalid, "the contract's validator refused its answer: %s", detail)
	}
	return ans, nil
}
