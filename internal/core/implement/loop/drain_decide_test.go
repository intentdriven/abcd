package loop

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/core/capture"
	decision "github.com/intentdriven/abcd/internal/core/decide/judge"
	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// The drain's judgement through the decision adapter (itd-2609221009495079):
// eligibility-residual in shadow records a pair beside the host's answer and
// changes nothing; turned on, the adapter's answer is taken as a routed
// judge's is, and one it cannot give leaves the host's request standing with
// a fallback receipt.

type drainFakeJudge struct {
	value string
	p     float64
	err   error
	calls int
}

func (f *drainFakeJudge) Judge(context.Context, decision.State, decision.Question) (decision.Answer, error) {
	f.calls++
	if f.err != nil {
		return decision.Answer{}, f.err
	}
	return decision.Answer{Value: f.value, Probability: f.p, Model: "typesafe/jev-1.13-0901"}, nil
}

// withDrainRouter makes the drain judge with eligibility-residual in mode,
// adapter answering, and pairs kept in got.
func withDrainRouter(t *testing.T, mode string, adapter *drainFakeJudge, got *[]lab.Pair) {
	t.Helper()
	routes := map[decision.Type]oracle.Target{}
	if mode != oracle.JudgementHost {
		routes[decision.EligibilityResidual] = oracle.Target{Provider: "openrouter", Model: "typesafe/jev-1.13", Mode: mode}
	}
	prev := loadDrainRouter
	loadDrainRouter = func(string, Options) (*decision.Router, error) {
		return decision.New(routes, func(oracle.Target) decision.Judge { return adapter }, func(p lab.Pair) (string, string, error) {
			*got = append(*got, p)
			return "lab-261029000000-abcdef0", "~/.abcd.noindex/lab/x/lab-261029000000-abcdef0/pairs.jsonl", nil
		}), nil
	}
	t.Cleanup(func() { loadDrainRouter = prev })
}

func drainDecidePlan(t *testing.T) (*capture.DrainPlan, string, DrainJudging) {
	t.Helper()
	repo := drainRepo(t)
	plan, err := capture.PlanDrain(capture.DrainPlanRequest{RepoRoot: repo.Root()})
	if err != nil {
		t.Fatal(err)
	}
	iss, ok := plan.OpenIssue(eligibleIssue)
	if !ok {
		t.Fatalf("%s is not open in the plan", eligibleIssue)
	}
	return &plan, repo.Root(), DrainJudging{Issue: eligibleIssue, RemedySHA256: remedyDigest(iss.Remedy)}
}

func TestShadowDrainJudgementRecordsThePairBesideTheHostsAnswer(t *testing.T) {
	plan, root, _ := drainDecidePlan(t)
	adapter, got := &drainFakeJudge{value: HandBackTrustRule, p: 0.8}, []lab.Pair{}
	withDrainRouter(t, oracle.JudgementShadow, adapter, &got)
	j := DrainJudgement{Issue: eligibleIssue, Answer: JudgementNo, Reason: "a renderer guard", Applied: true}
	rc := shadowDrainJudgement(context.Background(), root, Options{}, j, *plan)
	if rc == nil || rc.Route != decision.RouteHost || rc.Type != decision.EligibilityResidual || !strings.HasSuffix(rc.Pair, "pairs.jsonl") {
		t.Fatalf("receipt = %+v", rc)
	}
	if len(got) != 1 || got[0].Host != JudgementNo || got[0].Adapter != HandBackTrustRule || *got[0].Probability != 0.8 ||
		got[0].Type != string(decision.EligibilityResidual) {
		t.Fatalf("pairs = %+v", got)
	}
	// A yes is paired as its kind.
	j = DrainJudgement{Issue: eligibleIssue, Answer: JudgementYes, Kind: HandBackUserVisible, Reason: "a flag"}
	shadowDrainJudgement(context.Background(), root, Options{}, j, *plan)
	if len(got) != 2 || got[1].Host != HandBackUserVisible {
		t.Fatalf("pairs = %+v", got)
	}
}

func TestShadowDrainJudgementDoesNothingOffShadow(t *testing.T) {
	plan, root, _ := drainDecidePlan(t)
	for _, mode := range []string{oracle.JudgementHost, oracle.JudgementOn} {
		adapter, got := &drainFakeJudge{value: JudgementNo, p: 0.9}, []lab.Pair{}
		withDrainRouter(t, mode, adapter, &got)
		if rc := shadowDrainJudgement(context.Background(), root, Options{}, DrainJudgement{Issue: eligibleIssue, Answer: JudgementNo}, *plan); rc != nil || adapter.calls != 0 || len(got) != 0 {
			t.Fatalf("%s: receipt %+v, adapter calls %d, pairs %d", mode, rc, adapter.calls, len(got))
		}
	}
}

func TestDecideDrainJudgementTakesTheAdaptersAnswerWhenTurnedOn(t *testing.T) {
	plan, root, judging := drainDecidePlan(t)
	adapter, got := &drainFakeJudge{value: HandBackUserVisible, p: 0.91}, []lab.Pair{}
	withDrainRouter(t, oracle.JudgementOn, adapter, &got)
	now := time.Date(2026, 10, 29, 6, 0, 0, 0, time.UTC)
	taken, fb, err := decideDrainJudgement(context.Background(), root, Options{}, judging, *plan, now)
	if err != nil || fb != nil || taken == nil {
		t.Fatalf("taken %+v, fallback %+v, err %v", taken, fb, err)
	}
	if taken.Answer != JudgementYes || taken.Kind != HandBackUserVisible || !taken.Applied || !taken.At.Equal(now) ||
		!strings.Contains(taken.Reason, "probability 0.91") {
		t.Fatalf("taken = %+v", taken)
	}
	if r := taken.Route; r == nil || r.Asked != "decision-adapter openrouter/typesafe/jev-1.13" || r.Ran != r.Asked || r.Model != "typesafe/jev-1.13-0901" {
		t.Fatalf("route = %+v: the run record names the route", taken.Route)
	}
	// A no is taken as a no, and opens the lane.
	adapter.value = JudgementNo
	taken, _, _ = decideDrainJudgement(context.Background(), root, Options{}, judging, *plan, now)
	if taken == nil || taken.Answer != JudgementNo || taken.Kind != "" || !taken.Applied {
		t.Fatalf("no = %+v", taken)
	}
}

func TestDecideDrainJudgementFallsBackToTheHostWithAReceipt(t *testing.T) {
	plan, root, judging := drainDecidePlan(t)
	adapter, got := &drainFakeJudge{err: openaiapi.ErrUnreachable}, []lab.Pair{}
	withDrainRouter(t, oracle.JudgementOn, adapter, &got)
	taken, fb, err := decideDrainJudgement(context.Background(), root, Options{}, judging, *plan, time.Now())
	if err != nil || taken != nil || fb == nil {
		t.Fatalf("taken %+v, fallback %+v, err %v", taken, fb, err)
	}
	if fb.Role != RoleDrainJudge || fb.Ran != runner.Host || !strings.Contains(fb.Asked, "openrouter/typesafe/jev-1.13") ||
		!strings.Contains(fb.Detail, "the host's request stands") {
		t.Fatalf("fallback = %+v", fb)
	}
}

func TestDecideDrainJudgementAsksNothingUnlessTurnedOn(t *testing.T) {
	plan, root, judging := drainDecidePlan(t)
	for _, mode := range []string{oracle.JudgementHost, oracle.JudgementShadow} {
		adapter, got := &drainFakeJudge{value: JudgementNo, p: 0.9}, []lab.Pair{}
		withDrainRouter(t, mode, adapter, &got)
		taken, fb, err := decideDrainJudgement(context.Background(), root, Options{}, judging, *plan, time.Now())
		if taken != nil || fb != nil || err != nil || adapter.calls != 0 {
			t.Fatalf("%s: taken %+v fallback %+v err %v calls %d", mode, taken, fb, err, adapter.calls)
		}
	}
}

// The real configuration: a model off its provider's list refuses the
// judgement before any call, naming the list.
func TestDecideDrainJudgementRefusesAnUnlistedModelBeforeAnyCall(t *testing.T) {
	plan, root, judging := drainDecidePlan(t)
	home := t.TempDir()
	cfg := abcdhome.Path(home, "config.json")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte(`{"oracle":{"api":{"openrouter":{"base_url":"https://openrouter.ai/api/v1","key":"openrouter",
		"models":["typesafe/jev-1.13"]}},"judgements":{"eligibility-residual":"openrouter/mistral/small"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	o := Options{Roots: &layered.Roots{Repo: root, Home: home}}
	_, _, err := decideDrainJudgement(context.Background(), root, o, judging, *plan, time.Now())
	if err == nil || !strings.Contains(err.Error(), "not on openrouter's list") {
		t.Fatalf("err = %v, want the allowlist refusal", err)
	}
}
