package loop

// drain_decide.go is the decision adapter at the drain's judgement
// (itd-2609221009495079, spc-2609221011151661 scope 2). The judgement a drain
// asks before it opens an issue's lane is a closed-option question over one
// record — no, or yes and which kind of hand-back — so it is the judgement
// type eligibility-residual: the residual of the drain's eligibility once the
// field rule has passed, and it is routed through core/decide/judge, as every
// typed judgement is.
//
//   - Configured for shadow, the host (or the routed runner) decides exactly
//     as it does today; once its answer is taken, the adapter is asked the
//     same question over the same record and the pair is recorded in the
//     newest lab home (shadowDrainJudgement). Shadow never changes the
//     drain's answer or stops the drain. Its one call is made after the
//     answer is taken, inside the move and synchronously, so a slow provider
//     holds the move up to drainShadowTimeout before its pair is recorded as
//     failed. A drain already stopped asks nothing and keeps no pair.
//   - Turned on, the adapter is asked once the host's request is written
//     (decideDrainJudgement, through adapterJudgement), before any runner the
//     drain's judge is routed to: its answer is taken as a routed judge's is,
//     under the same disposition rules (judgeAnswer), and the judgement's
//     Route names the adapter and the model that answered. An adapter that
//     cannot answer leaves the host's request standing, with a fallback
//     receipt kept beside the routed judge's, and the request is marked as
//     asked of the adapter (DrainJudging.AdapterAsked), so the polls that
//     follow ask it nothing again; a routed runner is marked apart, only once
//     it was started.
//
// Both functions read the configuration themselves (loadDrainRouter), so the
// drain's call sites pass only what they already hold. On the host, the
// default, neither asks anything, and the drain moves as it did before the
// adapter existed.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	decision "github.com/intentdriven/abcd/internal/core/decide/judge"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// drainJudgementType is the drain's judgement in the decision adapter's
// registry.
const drainJudgementType = decision.EligibilityResidual

// drainAdapterRoute names the decision adapter in a judgement's Route and a
// fallback receipt's Asked, beside its <provider>/<model>.
const drainAdapterRoute = "decision-adapter"

// drainAdapterTimeout bounds one adapter call: a typed answer over one record
// takes about a second, so this is for a provider that hangs.
const drainAdapterTimeout = 2 * time.Minute

// drainShadowTimeout bounds the adapter's call in shadow, where its answer
// decides nothing. The call is synchronous, made inside the move once the
// answer is taken, so a provider this slow holds the move up to this long and
// then costs only its pair, recorded as failed.
const drainShadowTimeout = 20 * time.Second

// loadDrainRouter reads the decision adapter's configuration for the checkout
// at repoRoot. Tests replace it.
var loadDrainRouter = func(repoRoot string, o Options) (*decision.Router, error) {
	return decision.Load(o.roots(repoRoot), repoRoot)
}

// drainQuestion is the drain's one question as a typed judgement: no, or the
// kind of hand-back a yes names.
func drainQuestion() decision.Question {
	return decision.Question{Type: drainJudgementType, Kind: decision.KindChoose,
		Ask: "Does this remedy, carried out as written, change what a user sees, or a trust boundary? " +
			"What a user sees is anything a person reads or relies on: a command's output or exit code, a flag, " +
			"a message, a page, a prompt, or what a command does when a person runs it. A trust boundary is what is " +
			"allowed or refused, read, written, run or sent, whose input is trusted, or a rule about safety, secrets " +
			"or permissions. Answer " + HandBackUserVisible + " when it changes what a user sees, " + HandBackTrustRule +
			" when it changes a trust boundary (or both, or you cannot tell), and no only when it changes neither.",
		Options: []string{JudgementNo, HandBackUserVisible, HandBackTrustRule}}
}

// drainState is the state the judgement is made over: the issue as the
// host's request shows it, without the request's instructions about where to
// write the answer.
func drainState(iss capture.Issue) decision.State {
	var b strings.Builder
	fmt.Fprintf(&b, "Issue %s\nseverity: %s\ncategory: %s\n\nRemedy:\n%s\n\nRecord body:\n%s\n",
		iss.ID, termsafe.Sanitize(string(iss.Severity)), termsafe.Sanitize(string(iss.Category)),
		termsafe.Sanitize(iss.Remedy), termsafe.SanitizeBlock(strings.TrimSpace(iss.Body)))
	return decision.State{Text: b.String()}
}

// drainValue is a taken judgement as the question's choice: no, or the kind.
func drainValue(j DrainJudgement) string {
	if j.Answer == JudgementNo {
		return JudgementNo
	}
	return j.Kind
}

// shadowDrainJudgement asks the adapter beside a judgement the drain has
// taken, j, when eligibility-residual is configured for shadow, and records
// the pair. It returns the receipt, nil when the type is not in shadow, the
// issue has left the open ledger, or the configuration cannot be read (the
// drain has its answer; shadow never stops it). The drain's take point calls
// it once j is applied and recorded, so an answer the move could not take is
// never paired. A drain already stopped asks nothing and keeps no pair, and
// one stopped during the call keeps none either (the router's rule).
func shadowDrainJudgement(ctx context.Context, repoRoot string, o Options, j DrainJudgement, plan capture.DrainPlan) *decision.Receipt {
	iss, ok := plan.OpenIssue(j.Issue)
	if !ok || drainValue(j) == "" || ctxOr(ctx).Err() != nil {
		return nil
	}
	r, err := loadDrainRouter(repoRoot, o)
	if err != nil || r.Mode(drainJudgementType) != oracle.JudgementShadow {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctxOr(ctx), drainShadowTimeout)
	defer cancel()
	_, rc, err := r.Decide(ctx, drainState(iss), drainQuestion(), decision.Answered(drainValue(j)))
	if err != nil {
		return nil
	}
	return &rc
}

// decideDrainJudgement asks the adapter for the judgement the drain awaits,
// judging, when eligibility-residual is turned on. It returns the judgement
// the adapter decided, taken as a routed judge's answer is (judgeAnswer), with
// its Route naming the adapter; or nil and the fallback receipt when the
// adapter could not answer, the host's request then standing; or nil, nil when
// the type is not turned on. A configuration the oracle reader refuses (a
// model off its provider's list) is an error, since a route the person set is
// never dropped silently; nothing was called.
func decideDrainJudgement(ctx context.Context, repoRoot string, o Options, judging DrainJudging, plan capture.DrainPlan,
	now time.Time) (*DrainJudgement, *runner.FallbackReceipt, error) {
	iss, ok := plan.OpenIssue(judging.Issue)
	if !ok {
		return nil, nil, nil
	}
	r, err := loadDrainRouter(repoRoot, o)
	if err != nil {
		return nil, nil, fmt.Errorf("the decision adapter's configuration: %w", err)
	}
	if r.Mode(drainJudgementType) != oracle.JudgementOn {
		return nil, nil, nil
	}
	ctx, cancel := context.WithTimeout(ctxOr(ctx), drainAdapterTimeout)
	defer cancel()
	a, rc, err := r.Decide(ctx, drainState(iss), drainQuestion(), decision.Pending())
	asked := drainAdapterRoute + " " + rc.Target
	if errors.Is(err, decision.ErrHostPending) {
		return nil, &runner.FallbackReceipt{At: now, Role: RoleDrainJudge, Asked: asked, Reason: runner.ReasonFailed,
			Detail: strings.TrimSuffix(rc.Fallback, "the host decided") + "the host's request stands", Ran: runner.Host}, nil
	}
	if err != nil {
		return nil, nil, err
	}
	ans := JudgementAnswer{SchemaVersion: DrainJudgementSchemaVersion, Issue: judging.Issue, RemedySHA256: judging.RemedySHA256,
		Answer: JudgementYes, Kind: a.Value,
		Reason: fmt.Sprintf("the decision adapter %s answered %s at probability %.2f", rc.Target, a.Value, a.Probability)}
	if a.Value == JudgementNo {
		ans.Answer, ans.Kind = JudgementNo, ""
	}
	taken, err := judgeAnswer(ans, judging, plan, now)
	if err != nil {
		return nil, nil, err
	}
	taken.Route = &runner.RouteRecord{Asked: asked, Ran: asked, Model: termsafe.Sanitize(rc.ModelReported)}
	return &taken, nil, nil
}

// adapterJudgement is the drain's move over the request it awaits, judging,
// when eligibility-residual is turned on: the adapter's answer, taken as a
// routed judge's is, or nil with the clause the move's next step adds and the
// fallback receipt kept in st, the host's request then standing. asked
// reports whether the adapter was asked and did not answer, so the request is
// marked as asked of it (DrainJudging.AdapterAsked) and the polls that follow
// while the host's request stands ask it nothing again. Not turned on, it asks
// nothing and returns nil, "", false. A drain stopped while the adapter was
// asked is no fallback: the move refuses, and the next one asks again.
func adapterJudgement(ctx context.Context, repoRoot string, o Options, judging DrainJudging, plan capture.DrainPlan,
	st *DrainState, now time.Time) (taken *DrainJudgement, clause string, asked bool, err error) {
	ctx = ctxOr(ctx)
	taken, fb, err := decideDrainJudgement(ctx, repoRoot, o, judging, plan, now)
	if err != nil {
		return nil, "", false, err
	}
	if fb == nil {
		return taken, "", false, nil
	}
	if ctx.Err() != nil {
		return nil, "", false, refuse(StageDrain, "", "", "the drain was stopped while the decision adapter judged "+judging.Issue+
			"; the move took no answer", "run `abcd drain` again, which asks the decision adapter again")
	}
	keepJudgeFallback(st, *fb)
	return nil, "the decision adapter did not judge " + judging.Issue + ", recorded as a fallback: " + fb.Detail, true, nil
}

// adapterAskedClause is the move's clause for a request the decision adapter
// was asked in an earlier move and did not answer (DrainJudging.AdapterAsked):
// it is not asked again.
func adapterAskedClause(issue string) string {
	return "the decision adapter was already asked over " + issue + " for this request and did not answer, so it is not asked again and the request is the host's"
}

// drainRouterDiagnostics are the decision adapter's configuration's non-fatal
// reports for the checkout at repoRoot (a route naming a type outside the
// registry, a route the oracle reader skipped), each as a clause the move's
// next step names, so a misspelt route is never silently on the host. A
// configuration that cannot be read names nothing here: the judgement that
// reads it refuses with why.
func drainRouterDiagnostics(repoRoot string, o Options) []string {
	r, err := loadDrainRouter(repoRoot, o)
	if err != nil || r == nil {
		return nil
	}
	out := make([]string, 0, len(r.Diagnostics))
	for _, d := range r.Diagnostics {
		out = append(out, termsafe.Sanitize(d))
	}
	return out
}

// ctxOr is ctx, or the background context when it is nil.
func ctxOr(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
