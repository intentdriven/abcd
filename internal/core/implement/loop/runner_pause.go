package loop

// runner_pause.go is what a step does when a runner fails to start a role
// (iss-2611012047534449, following dcn-2611012113432054): the role waits for
// its runner instead of being handed to the host session that started the
// build, so nothing is billed to that session's subscription.
//
//   - The lane pauses and says why: the await stays out, marked `paused` with
//     the runner, the reason in abcd's words and the first line the runner
//     wrote to stderr (redacted, one line, bounded), so "Not logged in" or a
//     spending limit reads as the cause rather than "exited with status 1".
//     A step's result names the paused role (`paused`) and never as an
//     `awaiting` the host would start.
//   - Each later step that hands no agent out retries one held role once on
//     its runner, the one waiting longest. A held role is a paused one, or one
//     already with the host, from an earlier fallback or a step before its
//     route existed, whose receipt nothing has written: so lanes parked before
//     this change move too. A role whose receipt is written is the host's to
//     hand back, and is never retried.
//   - `--route` for a held role retries it at once on that route, and a
//     restart while a route is in force starts the fresh agent on its runner
//     rather than re-telling it to the host.
//   - Every retry is named in the run record (stage `retry`), and every
//     pause (stage `paused`), with one fallback receipt per failed start,
//     whose route that ran is none.
//
// Whoever starts a runner for an await first claims it under the run's lock
// (`running`, the runner and the session, until the run's time bound has
// passed), and the claim is dropped when the dispatch ends: so a second step,
// in this checkout or a peer's, never starts a second runner for a role one is
// already running, and a claim whose step was killed lapses with its bound.
// The bound a claim outlasts is the longest the run can last: its own, and the
// most its clock stands still while it waits at the load gate
// (runner.LongestRun, iss-2611021751406429).
//
// A role killed is not always a failed start (iss-2611021751406429): an
// implementer killed after it committed its work on the lane's branch and
// wrote its report has done what the lane waits on but its receipt. So that
// lane is kept at its stage (stage `killed`): the await stays out, neither
// paused nor counted as a fallback, for the host or the next step's runner to
// write the receipt from the branch and the report. Whatever the kill, what
// the killed launch spent, as its transcript shows it, is kept: on the
// fallback receipt of a failed start, and in the record's line of a kept one.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// StageRetry is the record's stage for a held role retried on its runner, and
// StagePaused for a role its runner failed to start.
const (
	StageRetry  = "retry"
	StagePaused = "paused"
)

// StageUnpaused is the record's stage for a paused role the person has since
// routed to the host: the pause is cleared and the host's agent takes it.
const StageUnpaused = "unpaused"

// ranNone is the route a fallback receipt names when nothing ran the role: a
// paused role is handed to no one.
const ranNone = "none"

// claimGrace is how long past the run's longest (runner.LongestRun) a claim
// stands, so a runner killed at its bound has had its dispatch end before the
// claim lapses.
const claimGrace = 5 * time.Minute

// maxStderrLine bounds the stderr line a pause carries.
const maxStderrLine = 200

// stderrMarker is the line a runner's transcript puts before the harness's
// stderr (internal/core/runner's procResult.transcript). The tap reads the
// first line after it; TestARunnerThatFailsToStartPausesTheLaneAndSaysWhy
// holds the two to each other through a real launch.
const stderrMarker = "== abcd runner: the harness's stderr =="

// RunnerPause is why a role waits for its runner: the runner that failed to
// start it, the reason in abcd's words, the first line the runner wrote to
// stderr, when the pause began and when the latest start failed, and how many
// starts have failed.
type RunnerPause struct {
	Runner string        `json:"runner"`
	Reason runner.Reason `json:"reason"`
	Detail string        `json:"detail"`
	// Stderr is the first line the runner wrote to stderr, redacted, one line
	// and bounded; "" when it wrote none.
	Stderr string    `json:"stderr,omitempty"`
	Since  time.Time `json:"since"`
	At     time.Time `json:"at"`
	Tries  int       `json:"tries"`
}

// Why is the pause's reason as one clause.
func (p RunnerPause) Why() string {
	s := string(p.Reason) + ": " + p.Detail
	if p.Stderr != "" {
		s += "; its stderr began: " + p.Stderr
	}
	return s
}

// RunnerClaim is a runner a step started for an await: the runner, the
// session it was started under, and when the claim lapses.
type RunnerClaim struct {
	Runner  string    `json:"runner"`
	Session string    `json:"session"`
	Until   time.Time `json:"until"`
}

// Retry is a held role a step retried on its runner: the lane, the role, the
// runner, why it was held, and, when the runner failed to start it again, the
// pause it stays in.
type Retry struct {
	Lane   string       `json:"lane"`
	Role   string       `json:"role"`
	Runner string       `json:"runner"`
	Why    string       `json:"why"`
	Paused *RunnerPause `json:"paused,omitempty"`
}

// errClaimed is a dispatch that found its await gone or another runner on it.
var errClaimed = errors.New("the await is gone or another step's runner holds it")

// held is one await a retry may start a runner for, and why it is held.
type held struct {
	lane  string
	await Await
	why   string
	last  time.Time
}

// heldRole is the held await waiting longest whose role cfg routes to a
// runner, among the roles in only when only is not nil, and whether there is
// one. An auto-plan run's reviews stay on the host (autoplan.go), so it holds
// none.
func heldRole(repoRoot string, st State, cfg *runner.Config, now time.Time, only map[string]string) (held, bool) {
	var best held
	found := false
	if cfg == nil || st.autoPlanning() {
		return best, false
	}
	for _, i := range st.laneOrder() {
		l := st.Lanes[i]
		if l.Stage == StageDone || l.Stage == StageDiscarded || l.HandBack != nil || l.Worktree == "" {
			continue
		}
		for _, a := range l.Awaits {
			if only != nil && only[a.Role] == "" {
				continue
			}
			if toolsFor(a.Role) == nil || cfg.RouteFor(a.Role).Runner == runner.Host {
				continue
			}
			// A rate limit's kept await is a restart's to re-tell, and a
			// reviewer route's placed answer is the route's.
			if a.Kept != "" || a.Answered != nil {
				continue
			}
			if a.Running != nil && now.Before(a.Running.Until) {
				continue
			}
			h := held{lane: l.ID, await: a, last: a.Since}
			switch {
			case a.Paused != nil:
				h.why = fmt.Sprintf("paused since %s (%s)", a.Paused.Since.UTC().Format(time.RFC3339), a.Paused.Why())
				h.last = a.Paused.At
			case received(repoRoot, a.Receipt):
				// The host's agent wrote it: `implement receipt` hands it back.
				continue
			case a.Running != nil:
				h.why = fmt.Sprintf("the %s runner an earlier step started for it lapsed at %s, nothing received",
					a.Running.Runner, a.Running.Until.UTC().Format(time.RFC3339))
			default:
				h.why = fmt.Sprintf("with the host since %s, nothing received", a.Since.UTC().Format(time.RFC3339))
			}
			if !found || h.last.Before(best.last) {
				best, found = h, true
			}
		}
	}
	return best, found
}

// releaseToHost is the move a step that handed no agent out makes for a role
// paused for its runner whose route the person has since moved to the host
// (their machine configuration dropping the runner, or a repository layer
// keeping the role on the host): the pause waited for the runner, never for
// the person's own choice. The paused role waiting longest is released in one
// write under the lock: its pause is cleared, it is with the host from now,
// and the record names the person's route, which is not a fallback, so no
// fallback receipt is written. The step's answer names it for the host to
// start, as any await with the host is re-told. It reports false, with res as
// it is, when no paused role is routed to the host or the run cannot hand an
// agent out now. res is the step's own result: when the step also performed a
// stage abcd runs itself, that stays its answer and the release is added to
// its next move.
func releaseToHost(repoRoot, runID string, o Options, rs Runners, res StepResult) (StepResult, bool, error) {
	if rs.Config == nil || res.Complete || res.NextEligibleAt != nil || res.Outage != nil || res.RateLimit != nil || res.HandBack != nil {
		return res, false, nil
	}
	var out StepResult
	released := false
	err := mutate(repoRoot, runID, func(_ *os.Root, st *State) (bool, error) {
		if st.autoPlanning() {
			return false, nil
		}
		now := o.now()
		bi, bk := -1, -1
		for _, i := range st.laneOrder() {
			l := st.Lanes[i]
			if l.Stage == StageDone || l.Stage == StageDiscarded || l.HandBack != nil {
				continue
			}
			for k, a := range l.Awaits {
				if a.Paused == nil || rs.Config.RouteFor(a.Role).Runner != runner.Host {
					continue
				}
				if a.Running != nil && now.Before(a.Running.Until) {
					// A step's runner holds it: its dispatch ends the claim.
					continue
				}
				if bi < 0 || a.Paused.At.Before(st.Lanes[bi].Awaits[bk].Paused.At) {
					bi, bk = i, k
				}
			}
		}
		if bi < 0 {
			return false, nil
		}
		lane := st.Lanes[bi]
		a := lane.Awaits[bk]
		p := *a.Paused
		a.Since, a.Paused, a.Running = now, nil, nil
		lane.Awaits = slices.Clone(lane.Awaits)
		lane.Awaits[bk] = a
		st.Lanes[bi] = lane
		st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Stage: StageUnpaused,
			Note: fmt.Sprintf("the %s, paused since %s for the %s runner (%s), is routed to the host now: the person's route, not a fallback, so the pause is cleared and the host's agent takes it",
				a.Role, p.Since.UTC().Format(time.RFC3339), p.Runner, p.Why())})
		st.UpdatedAt = now
		out = laneResult(*st, lane, "", nil)
		out.Paused, out.Awaiting = nil, &a
		out.Tier = roundTier(lane, a)
		out.Next = fmt.Sprintf("%s's %s, paused for the %s runner, is routed to the host now, so it is yours: %s",
			lane.ID, a.Role, p.Runner, handedMove(a, out.Tier))
		released = true
		return true, nil
	})
	if err != nil {
		return res, true, err
	}
	if !released {
		return res, false, nil
	}
	if res.PerformedStage == "" {
		return out, true, nil
	}
	res.Next += "; and " + out.Next
	return res, true, nil
}

// received reports whether anything stands at an await's receipt path. A path
// that cannot be examined counts as received: no runner is started over it.
func received(repoRoot, receipt string) bool {
	_, err := os.Lstat(absIn(repoRoot, receipt))
	return !errors.Is(err, os.ErrNotExist)
}

// retryHeld is the retry a step that handed no agent out makes: the held role
// waiting longest, once, on its runner. res is the step's own result; it is
// returned as it is when nothing is held or the run cannot start an agent now.
func retryHeld(ctx context.Context, repoRoot, runID string, steps Stages, o Options, rs Runners, res StepResult) (StepResult, error) {
	if res.Complete || res.NextEligibleAt != nil || res.Outage != nil || res.RateLimit != nil || res.HandBack != nil {
		return res, nil
	}
	st, err := ReadState(repoRoot, runID)
	if err != nil {
		return res, err
	}
	h, ok := heldRole(repoRoot, st, rs.Config, o.now(), nil)
	if !ok {
		return res, nil
	}
	r, err := dispatchAwait(ctx, repoRoot, runID, steps, o, rs, h.lane, h.await, h.why, res)
	if errors.Is(err, errClaimed) {
		return res, nil
	}
	if err != nil {
		return r, err
	}
	if res.PerformedStage == "" || r.RateLimit != nil {
		// A rate limit ended the run's window: that is the step's answer.
		return r, nil
	}
	// The step also performed a stage abcd runs itself: that stays its
	// answer, and the retry is named beside it.
	res.Retried = r.Retried
	res.Next += "; " + retrySentence(*r.Retried, r.Route != nil)
	return res, nil
}

// retrySentence says what a retry came to.
func retrySentence(r Retry, ran bool) string {
	if ran {
		return fmt.Sprintf("this step also retried %s's %s on the %s runner, which ran it", r.Lane, r.Role, r.Runner)
	}
	if r.Paused != nil {
		return fmt.Sprintf("this step also retried %s's %s on the %s runner, which did not start it (%s), so it stays paused", r.Lane, r.Role, r.Runner, r.Paused.Why())
	}
	return fmt.Sprintf("this step also retried %s's %s on the %s runner", r.Lane, r.Role, r.Runner)
}

// retryRouted is a step's --route for a held role: the role is retried at
// once on the route, and the step makes no other move. It reports false when
// no held role is one a --route names, and the step proceeds as any other.
func retryRouted(ctx context.Context, repoRoot, runID string, steps Stages, o Options, rs Runners, overrides map[string]string) (StepResult, bool, error) {
	st, err := ReadState(repoRoot, runID)
	if err != nil {
		return StepResult{}, true, err
	}
	now := o.now()
	h, ok := heldRole(repoRoot, st, rs.Config, now, overrides)
	if !ok {
		return StepResult{}, false, nil
	}
	for role, text := range overrides {
		if role != h.await.Role {
			return StepResult{}, true, refuse(StageRoute, "", h.lane,
				fmt.Sprintf("--route %s routes the %s, and this step retries %s's held %s at once", text, role, h.lane, h.await.Role),
				"step again without that --route; nothing was written")
		}
	}
	if st.NextEligibleAt != nil && now.Before(*st.NextEligibleAt) {
		return StepResult{}, true, contend("pause", "", "", "the run is paused until "+st.NextEligibleAt.UTC().Format(time.RFC3339),
			"run `abcd implement step` again at or after that time")
	}
	if err := configTripwire(repoRoot, runID); err != nil {
		return StepResult{}, true, err
	}
	if err := outageOpen(repoRoot, h.lane); err != nil {
		if r, ok := AsRefusal(err); ok {
			r.Stage = StageRetry
			r.Remedy = strings.Replace(r.Remedy, "then restart the lane", "then step again with the --route", 1)
		}
		return StepResult{}, true, err
	}
	why := h.why + ", under --route " + overrides[h.await.Role]
	r, err := dispatchAwait(ctx, repoRoot, runID, steps, o, rs, h.lane, h.await, why, StepResult{})
	if errors.Is(err, errClaimed) {
		return StepResult{}, true, contend(StageRetry, "", h.lane, "another step's runner took "+h.lane+"'s "+h.await.Role+" first",
			"run `abcd implement step` once it has finished; nothing was written")
	}
	return r, true, err
}

// dispatchAwait starts the runner aw's role is routed to for lane laneID's
// await aw, and hands its receipt back through Receipt. why is "" for an
// await this step handed out, and otherwise why the held role is retried,
// which the record names. The await is claimed first, under the lock; a
// runner that does not start the role pauses it (pauseAwait), and every other
// end drops the claim. on is returned with any refusal, as the step's answer.
func dispatchAwait(ctx context.Context, repoRoot, runID string, steps Stages, o Options, rs Runners, laneID string, aw Await, why string, on StepResult) (StepResult, error) {
	route := rs.Config.RouteFor(aw.Role)
	timeout := rs.Timeout
	if timeout <= 0 {
		timeout = rs.Config.TimeoutFor(aw.Role)
	}
	session := fmt.Sprintf("%s-%s-%s-%d", runID, laneID, aw.Role, o.now().Unix())
	var worktree, base string
	err := mutate(repoRoot, runID, func(_ *os.Root, st *State) (bool, error) {
		now := o.now()
		i, k := st.findAwait(repoRoot, aw.Receipt)
		if i < 0 || st.Lanes[i].ID != laneID {
			return false, errClaimed
		}
		lane := st.Lanes[i]
		a := lane.Awaits[k]
		if a.Running != nil && now.Before(a.Running.Until) {
			return false, errClaimed
		}
		a.Running = &RunnerClaim{Runner: route.Runner, Session: session, Until: now.Add(runner.LongestRun(timeout) + claimGrace)}
		lane.Awaits = slices.Clone(lane.Awaits)
		lane.Awaits[k] = a
		st.Lanes[i] = lane
		worktree, base = lane.Worktree, lane.BaseSHA
		if why != "" {
			st.Record = append(st.Record, Entry{At: now, Lane: laneID, Stage: StageRetry,
				Note: fmt.Sprintf("retried %s's %s on the %s runner: %s", laneID, aw.Role, route.Runner, why)})
		}
		st.UpdatedAt = now
		return true, nil
	})
	if err != nil {
		return on, err
	}
	release := func() error { return releaseClaim(repoRoot, runID, aw.Receipt, session) }

	req := runner.Request{
		Role:     aw.Role,
		Brief:    absIn(repoRoot, aw.Brief),
		Receipt:  absIn(repoRoot, aw.Receipt),
		Dir:      worktree,
		Checkout: filepath.Clean(repoRoot),
		Tools:    toolsFor(aw.Role),
		// The approver of a shell launch judges against the registry the
		// lane was cut from, never one the lane has since changed.
		Base: base,
		// A review set's reviewer runs at its class's model, over the
		// runner's own (reviewers.go); the runner admits it against the
		// provider's allowlist whoever stored it. "" keeps the route's.
		Model:     aw.Model,
		SessionID: session,
		Timeout:   timeout,
	}
	var done *StepResult
	var fell *runner.FallbackReceipt
	var tap *stderrTap
	var transcripts runner.TranscriptStore
	if rs.Transcripts != nil {
		tap = &stderrTap{store: rs.Transcripts}
		transcripts = tap
	}
	d := &runner.Dispatcher{
		Config:      rs.Config,
		HostSession: true,
		Transcripts: transcripts,
		Validate: func(ran string, _ runner.Request, ans runner.Answer) error {
			rec := runner.RecordOf(route.Runner, ran, ans)
			rec.Override = route.Override()
			r, err := receipt(repoRoot, runID, aw.Receipt, steps, o, &rec)
			if err != nil {
				return err
			}
			done = &r
			return nil
		},
		// The receipt is written with the pause, in one write, below: the
		// host session is never handed the role.
		Record: func(fb runner.FallbackReceipt) error { fell = &fb; return nil },
		Now:    o.Now,
	}
	out, err := d.Dispatch(ctx, req)
	retry := func(r StepResult, p *RunnerPause) StepResult {
		if why != "" {
			r.Retried = &Retry{Lane: laneID, Role: aw.Role, Runner: route.Runner, Why: why, Paused: p}
		}
		return r
	}
	if err != nil {
		if rerr := release(); rerr != nil {
			return on, errors.Join(err, rerr)
		}
		var fl *runner.Failure
		if errors.As(err, &fl) && fl.Reason == runner.ReasonRateLimited {
			// Not a pause and not a refusal: the run's window ends early
			// and every lane in flight is checkpointed.
			r, rerr := rateLimitWindow(repoRoot, runID, laneID, aw, fl, o)
			return retry(r, nil), rerr
		}
		return on, refuse(StageRunner, "", laneID, err.Error(),
			"the lane still awaits its receipt: correct what the reason names and step again")
	}
	if out.Handoff {
		fb := runner.FallbackReceipt{Role: aw.Role, Asked: route.Runner, Reason: runner.ReasonFailed, Detail: "the runner did not run it"}
		if fell != nil {
			fb = *fell
		}
		if out.Failure != nil && errors.Is(out.Failure, runner.ErrKilled) && aw.Role == RoleImplementer {
			if n, ok := workStands(repoRoot, worktree, base, aw); ok {
				r, err := keepKilled(repoRoot, runID, laneID, aw, session, fb, n, o)
				return retry(r, nil), err
			}
		}
		stderr := ""
		if tap != nil {
			stderr = redactStderr(repoRoot, tap.first)
		}
		r, p, err := pauseAwait(repoRoot, runID, laneID, aw, session, fb, stderr, o)
		return retry(r, p), err
	}
	if done == nil {
		if rerr := release(); rerr != nil {
			return on, rerr
		}
		return on, refuse(StageRunner, "", laneID, "the runner ran the "+aw.Role+" but no receipt was handed back",
			"the lane still awaits its receipt; step again")
	}
	r := *done
	r.Route = &out.Receipt.Route
	return retry(r, nil), nil
}

// pauseAwait records that the runner did not start the role: the await stays
// out, paused with the reason and the runner's first stderr line, its claim
// dropped; one fallback receipt naming none as the route that ran; and the
// record's line. It returns the lane's result, which names the paused role.
func pauseAwait(repoRoot, runID, laneID string, aw Await, session string, fb runner.FallbackReceipt, stderr string, o Options) (StepResult, *RunnerPause, error) {
	var res StepResult
	var paused *RunnerPause
	err := mutate(repoRoot, runID, func(_ *os.Root, st *State) (bool, error) {
		now := o.now()
		fb.At, fb.Ran = now, ranNone
		st.Fallbacks = append(st.Fallbacks, fb)
		p := RunnerPause{Runner: fb.Asked, Reason: fb.Reason, Detail: fsutil.RedactHome(fb.Detail), Stderr: stderr, Since: now, At: now, Tries: 1}
		i, k := st.findAwait(repoRoot, aw.Receipt)
		var lane Lane
		if i >= 0 && st.Lanes[i].ID == laneID {
			lane = st.Lanes[i]
			a := lane.Awaits[k]
			if a.Running != nil && a.Running.Session == session {
				a.Running = nil
			}
			if a.Paused != nil {
				p.Since, p.Tries = a.Paused.Since, a.Paused.Tries+1
			}
			a.Paused = &p
			lane.Awaits = slices.Clone(lane.Awaits)
			lane.Awaits[k] = a
			st.Lanes[i] = lane
		} else if j := slices.IndexFunc(st.Lanes, func(l Lane) bool { return l.ID == laneID }); j >= 0 {
			// The await was handed back while the runner ran: nothing waits.
			lane = st.Lanes[j]
		}
		paused = &p
		st.Record = append(st.Record, Entry{At: now, Lane: laneID, Stage: StagePaused,
			Note: fmt.Sprintf("the %s was routed to %s, which did not start it (%s); the lane is paused rather than handed to the host, and the next step retries it on the runner (try %d)",
				aw.Role, fb.Asked, p.Why(), p.Tries)})
		st.UpdatedAt = now
		res = laneResult(*st, lane, "", nil)
		return true, nil
	})
	return res, paused, err
}

// StageKilled is the record's stage for a role killed after its work stood:
// the lane is kept at its stage for its receipt, not paused.
const StageKilled = "killed"

// workStands reports whether a killed implementer's work stands: commits on
// the lane's branch past its base, which its worktree has checked out, and
// its report written beside the receipt it did not write; and how many
// commits. Anything that cannot be read stands for nothing, and the kill is a
// failed start as any other.
func workStands(repoRoot, worktree, base string, aw Await) (int, bool) {
	if worktree == "" || base == "" {
		return 0, false
	}
	report := filepath.Join(filepath.Dir(absIn(repoRoot, aw.Receipt)), ReportFileName)
	fi, err := os.Lstat(report)
	if err != nil || !fi.Mode().IsRegular() || fi.Size() == 0 {
		return 0, false
	}
	out, err := gitutil.Run(worktree, "rev-list", "--count", base+"..HEAD", "--")
	if err != nil {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// keepKilled records a role its runner killed after its work stood
// (workStands): its claim and any pause are dropped and no fallback receipt
// is written, since this was no failed start; the await stays out for the
// host, or the next step's runner, to write its receipt from the branch and
// the report; and the record's line names the kill, the commits and what the
// launch spent. It returns the lane's result, which names the await.
func keepKilled(repoRoot, runID, laneID string, aw Await, session string, fb runner.FallbackReceipt, commits int, o Options) (StepResult, error) {
	var res StepResult
	err := mutate(repoRoot, runID, func(_ *os.Root, st *State) (bool, error) {
		now := o.now()
		i, k := st.findAwait(repoRoot, aw.Receipt)
		var lane Lane
		if i >= 0 && st.Lanes[i].ID == laneID {
			lane = st.Lanes[i]
			a := lane.Awaits[k]
			if a.Running != nil && a.Running.Session == session {
				a.Running = nil
			}
			a.Paused = nil
			lane.Awaits = slices.Clone(lane.Awaits)
			lane.Awaits[k] = a
			st.Lanes[i] = lane
		} else if j := slices.IndexFunc(st.Lanes, func(l Lane) bool { return l.ID == laneID }); j >= 0 {
			// The await was handed back while the runner ran: nothing waits.
			lane = st.Lanes[j]
		}
		spent := "its transcript showed nothing it spent"
		if fb.Spent != nil {
			spent = "it spent " + fb.Spent.Text(fb.Asked)
		}
		st.Record = append(st.Record, Entry{At: now, Lane: laneID, Stage: StageKilled,
			Note: fmt.Sprintf("the %s on the %s runner was killed (%s) after it committed %d commit(s) past the lane's base and wrote its report; "+
				"that is no failed start, so the lane stays at its stage, neither paused nor counted as a fallback, for the host or the next step's runner to write its receipt; %s",
				aw.Role, fb.Asked, fsutil.RedactHome(fb.Detail), commits, spent)})
		st.UpdatedAt = now
		res = laneResult(*st, lane, "", nil)
		if a := res.Awaiting; a != nil && a.Receipt == aw.Receipt {
			res.Next = fmt.Sprintf("%s's %s was killed by the %s runner after committing %d commit(s) past its base and writing its report, so its work stands and only its receipt is missing; "+
				"the lane stays at its stage. Write that receipt from the branch and the report: %s. Or the next `abcd implement step` retries it on its runner",
				laneID, aw.Role, fb.Asked, commits, handedMove(*a, res.Tier))
		}
		return true, nil
	})
	return res, err
}

// releaseClaim drops the claim session took on the await at receipt, when the
// await still carries it.
func releaseClaim(repoRoot, runID, receipt, session string) error {
	return mutate(repoRoot, runID, func(_ *os.Root, st *State) (bool, error) {
		i, k := st.findAwait(repoRoot, receipt)
		if i < 0 {
			return false, nil
		}
		lane := st.Lanes[i]
		a := lane.Awaits[k]
		if a.Running == nil || a.Running.Session != session {
			return false, nil
		}
		a.Running = nil
		lane.Awaits = slices.Clone(lane.Awaits)
		lane.Awaits[k] = a
		st.Lanes[i] = lane
		return true, nil
	})
}

// pausedRoleMove is the next move of a lane whose role waits for its runner.
func pausedRoleMove(lane Lane, a Await) string {
	p := a.Paused
	return fmt.Sprintf("%s's %s is paused: the %s runner did not start it (%s), and it is not handed to the host. "+
		"The next `abcd implement step` retries it on the runner once, or hands it to you if its route has since moved to the host; `abcd implement step --route %s=runner:<name>` retries it now. "+
		"Do not start an agent for it; tell the user the reason",
		lane.ID, a.Role, p.Runner, p.Why(), a.Role)
}

// stderrTap passes a runner's transcript to the store it wraps, keeping the
// first line the harness wrote to stderr.
type stderrTap struct {
	store runner.TranscriptStore
	first string
}

func (s *stderrTap) Store(name string, req runner.Request, ans runner.Answer, raw []byte) error {
	s.first = firstStderrLine(raw)
	return s.store.Store(name, req, ans, raw)
}

// firstStderrLine is the first line that is not blank after the transcript's
// stderr marker, "" when there is none. The event stream before the marker is
// one JSON object per line, so the first marker line is the runner's own.
func firstStderrLine(raw []byte) string {
	lines := strings.Split(string(raw), "\n")
	at := slices.Index(lines, stderrMarker)
	if at < 0 {
		return ""
	}
	for _, l := range lines[at+1:] {
		if t := strings.TrimSpace(l); t != "" {
			return t
		}
	}
	return ""
}

// redactStderr is a runner's stderr line as a pause keeps it: through the
// canonical refusal redactor (a token or the home directory never kept, the
// line described rather than echoed when the scanner cannot vouch for it),
// one line, at most maxStderrLine bytes.
func redactStderr(repoRoot, line string) string {
	if line == "" {
		return ""
	}
	s := fsutil.RedactHome(scanner.RedactRefusal(repoRoot, line))
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > maxStderrLine {
		cut := maxStderrLine
		for cut > 0 && !utf8.RuneStart(s[cut]) {
			cut--
		}
		s = s[:cut] + "..."
	}
	return s
}
