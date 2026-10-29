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
//     newest lab home (shadowDrainJudgement). Shadow never changes, delays or
//     stops the drain.
//   - Turned on, the adapter is asked once the host's request is written
//     (decideDrainJudgement): its answer is taken as a routed judge's is,
//     under the same disposition rules (judgeAnswer), and the judgement's
//     Route names the adapter and the model that answered. An adapter that
//     cannot answer leaves the host's request standing, with a fallback
//     receipt kept beside the routed judge's.
//
// Both functions read the configuration themselves (loadDrainRouter), so the
// drain's call sites pass only what they already hold.

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
// it before applying j.
func shadowDrainJudgement(ctx context.Context, repoRoot string, o Options, j DrainJudgement, plan capture.DrainPlan) *decision.Receipt {
	iss, ok := plan.OpenIssue(j.Issue)
	if !ok || drainValue(j) == "" {
		return nil
	}
	r, err := loadDrainRouter(repoRoot, o)
	if err != nil || r.Mode(drainJudgementType) != oracle.JudgementShadow {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctxOr(ctx), drainAdapterTimeout)
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
			Detail: rc.Fallback + "; the host's request stands", Ran: runner.Host}, nil
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

// ctxOr is ctx, or the background context when it is nil.
func ctxOr(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
