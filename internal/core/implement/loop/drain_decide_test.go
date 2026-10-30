package loop

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
	withDrainRouterWindow(t, mode, 0, adapter, got)
}

// withDrainRouterWindow is withDrainRouter with the adapter's window in
// tokens; the default window when zero.
func withDrainRouterWindow(t *testing.T, mode string, window int, adapter *drainFakeJudge, got *[]lab.Pair) {
	t.Helper()
	routes := map[decision.Type]oracle.Target{}
	if mode != oracle.JudgementHost {
		routes[decision.EligibilityResidual] = oracle.Target{Provider: "openrouter", Model: "typesafe/jev-1.13", Mode: mode}
	}
	prev := loadDrainRouter
	loadDrainRouter = func(string, Options) (*decision.Router, error) {
		r := decision.NewForTest(routes, func(oracle.Target) decision.Judge { return adapter }, func(p lab.Pair) (string, string, error) {
			*got = append(*got, p)
			return "lab-261029000000-abcdef0", abcdhome.Display("lab/x/lab-261029000000-abcdef0/pairs.jsonl"), nil
		})
		r.Window = window
		return r, nil
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

// The drain's real eligibility question through the log-probability
// candidate, against a fake OpenAI-compatible endpoint: the question's own
// words name the answers (user-visible, trust-rule, no), and the candidate's
// prompt maps each to the letter it scores and says the letter wins. A letter
// is taken as the kind it labels; the word itself, answered against that
// instruction, is no answer, so the host's request stands with a receipt.
func TestTheDrainsQuestionThroughTheLogprobCandidate(t *testing.T) {
	plan, root, judging := drainDecidePlan(t)
	var (
		mu    sync.Mutex
		reply string
		sent  []byte
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			http.NotFound(w, r)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		mu.Lock()
		sent = raw
		body := reply
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	answer := func(content, logprobs string) string {
		return `{"model":"small/model","choices":[{"index":0,"finish_reason":"length","message":{"role":"assistant","content":` +
			strconv.Quote(content) + `},"logprobs":` + logprobs + `}]}`
	}
	home := t.TempDir()
	cfg := abcdhome.Path(home, "config.json")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte(`{"oracle":{"api":{"lp":{"base_url":"`+srv.URL+`/v1","models":["small/model"],`+
		`"logprob_models":["small/model"]}},"judgements":{"eligibility-residual":"lp/small/model"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	o := Options{Roots: &layered.Roots{Repo: root, Home: home}}
	now := time.Date(2026, 10, 30, 9, 0, 0, 0, time.UTC)

	mu.Lock()
	reply = answer("C", `{"content":[{"token":"C","logprob":-0.1,"top_logprobs":[`+
		`{"token":"C","logprob":-0.1},{"token":"B","logprob":-2.5},{"token":"no","logprob":-3.0}]}]}`)
	mu.Unlock()
	taken, fb, err := decideDrainJudgement(context.Background(), root, o, judging, *plan, now)
	if err != nil || fb != nil || taken == nil {
		t.Fatalf("taken %+v, fallback %+v, err %v", taken, fb, err)
	}
	if taken.Answer != JudgementYes || taken.Kind != HandBackTrustRule || !strings.Contains(taken.Reason, "probability 0.92") {
		t.Fatalf("C labels trust-rule, at its probability over the letters: %+v", taken)
	}
	mu.Lock()
	var req struct {
		Messages []struct {
			Content string `json:"content"`
		} `json:"messages"`
		MaxTokens   json.RawMessage `json:"max_tokens"`
		Logprobs    json.RawMessage `json:"logprobs"`
		TopLogprobs json.RawMessage `json:"top_logprobs"`
	}
	err = json.Unmarshal(sent, &req)
	mu.Unlock()
	if err != nil {
		t.Fatalf("the request: %v", err)
	}
	var prompt strings.Builder
	for _, m := range req.Messages {
		prompt.WriteString(m.Content + "\n")
	}
	p := prompt.String()
	for _, want := range []string{
		drainQuestion().Ask,
		"A. " + JudgementNo + "\n", "B. " + HandBackUserVisible + "\n", "C. " + HandBackTrustRule + "\n",
		"reply with that answer's letter, never the word",
		"Reply with one letter: A, B or C.",
	} {
		if !strings.Contains(p, want) {
			t.Errorf("the prompt lacks %q:\n%s", want, p)
		}
	}
	if !strings.HasSuffix(strings.TrimSpace(p), "Reply with one letter: A, B or C.") {
		t.Errorf("the letter instruction is the prompt's last word:\n%s", p)
	}
	if string(req.MaxTokens) != "1" || string(req.Logprobs) != "true" || string(req.TopLogprobs) != "20" {
		t.Errorf("one token and its alternatives: max_tokens %s logprobs %s top_logprobs %s", req.MaxTokens, req.Logprobs, req.TopLogprobs)
	}

	// The question's own word for the answer, against the instruction, is no
	// answer, however a letter ranks below it.
	mu.Lock()
	reply = answer("no", `{"content":[{"token":"no","logprob":-0.01,"top_logprobs":[`+
		`{"token":"no","logprob":-0.01},{"token":"C","logprob":-7.0}]}]}`)
	mu.Unlock()
	taken, fb, err = decideDrainJudgement(context.Background(), root, o, judging, *plan, now)
	if err != nil || taken != nil || fb == nil || !strings.Contains(fb.Detail, `answered "no"`) ||
		!strings.Contains(fb.Detail, "the host's request stands") {
		t.Fatalf("an off-label answer leaves the host's request standing: taken %+v, fallback %+v, err %v", taken, fb, err)
	}
}
