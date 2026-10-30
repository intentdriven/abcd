package judge

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/core/credential"
	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The decision adapter (itd-2609221009495079, spc-2609221011151661): one
// interface the host and the adapter both implement, and one router with three
// modes per judgement type. The adapter here is a fake answering through the
// same interface; TestTheAPIAdapterParsesTheTypedAnswer drives the real
// Adapter over a fake oracle caller.

var target = oracle.Target{Provider: "openrouter", Model: "typesafe/jev-1.13", Origin: abcdhome.Display("config.json")}

func dupQuestion() Question {
	return Question{Type: DuplicateMatch, Kind: KindChoose, Ask: "Which record does this capture duplicate?",
		Options: []string{"none", "iss-1", "iss-2"}}
}

// fake is a Judge that answers value at p, or fails with err, and counts calls.
type fake struct {
	value string
	p     float64
	err   error
	calls int
	seen  []string
}

func (f *fake) Judge(_ context.Context, s State, q Question) (Answer, error) {
	f.calls++
	f.seen = append(f.seen, render(s, q))
	if f.err != nil {
		return Answer{}, f.err
	}
	return Answer{Value: f.value, Probability: f.p, Model: "typesafe/jev-1.13-0901"}, nil
}

// pairs is a recorder that keeps what it is given.
type pairs struct {
	got []lab.Pair
	err error
}

func (p *pairs) record(pr lab.Pair) (string, string, error) {
	if p.err != nil {
		return "", "", p.err
	}
	p.got = append(p.got, pr)
	return "lab-261029000000-abcdef0", abcdhome.Display("lab/x/lab-261029000000-abcdef0/pairs.jsonl"), nil
}

func router(mode string, adapter *fake, rec *pairs) *Router {
	routes := map[Type]oracle.Target{}
	if mode != oracle.JudgementHost {
		t := target
		t.Mode = mode
		routes[DuplicateMatch] = t
	}
	return newRouter(routes, func(oracle.Target) Judge { return adapter }, rec.record)
}

// Criterion 1: shadow — the host decides, the adapter answers beside it, and
// the pair with its probability and the state's hash is recorded.
func TestShadowTheHostDecidesAndThePairIsRecorded(t *testing.T) {
	adapter, rec := &fake{value: "iss-2", p: 0.81}, &pairs{}
	host := Answered("iss-1")
	s := State{Text: "capture: the parser drops a trailing comma"}
	a, rc, err := router(oracle.JudgementShadow, adapter, rec).Decide(context.Background(), s, dupQuestion(), host)
	if err != nil {
		t.Fatal(err)
	}
	if a.Value != "iss-1" || a.Probability != 1 || rc.Route != RouteHost || rc.Mode != oracle.JudgementShadow {
		t.Fatalf("answer %+v receipt %+v: the host decides in shadow", a, rc)
	}
	if adapter.calls != 1 || len(rec.got) != 1 {
		t.Fatalf("adapter calls %d, pairs %d; want one each", adapter.calls, len(rec.got))
	}
	p := rec.got[0]
	if p.Type != string(DuplicateMatch) || p.Kind != string(KindChoose) || p.Host != "iss-1" || p.Adapter != "iss-2" ||
		p.Probability == nil || *p.Probability != 0.81 || p.StateSHA256 != s.SHA256() || p.Target != "openrouter/typesafe/jev-1.13" ||
		p.ModelReported != "typesafe/jev-1.13-0901" || p.Skipped != "" {
		t.Fatalf("pair = %+v", p)
	}
	if !strings.Contains(rc.Pair, "pairs.jsonl") {
		t.Fatalf("receipt pair = %q", rc.Pair)
	}
}

// Criterion 1, end to end: the pair lands in pairs.jsonl under the newest lab
// home in the machine-scoped store, and nothing is written into the
// repository.
func TestShadowWritesThePairUnderTheLabHomeAndNothingInTheRepository(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	r := gittest.NewRepo(t)
	r.Write("README.md", "root\n")
	r.Commit("root")
	t.Cleanup(lab.SetAbcdRootCommitForTest(strings.Repeat("a", 40)))
	m, err := lab.Mint(r.Root(), "does the decision adapter agree with the host?", "")
	if err != nil {
		t.Fatal(err)
	}
	before := r.Git("status", "--porcelain", "--ignored")
	t2 := target
	t2.Mode = oracle.JudgementShadow
	rt := newRouter(map[Type]oracle.Target{DuplicateMatch: t2}, func(oracle.Target) Judge { return &fake{value: "none", p: 0.7} },
		func(p lab.Pair) (string, string, error) { return lab.RecordPair(r.Root(), p) })
	if _, _, err := rt.Decide(context.Background(), State{Text: "s"}, dupQuestion(), Answered("none")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(abcdhome.Path(home, "lab", m.RootSHA, m.ID, "pairs.jsonl"))
	if err != nil {
		t.Fatalf("pairs.jsonl: %v", err)
	}
	var p lab.Pair
	if err := json.Unmarshal(data, &p); err != nil || p.Host != "none" || p.Adapter != "none" {
		t.Fatalf("pair = %s (%v)", data, err)
	}
	if after := r.Git("status", "--porcelain", "--ignored"); after != before {
		t.Fatalf("the repository changed:\n%s", after)
	}
}

// Criterion 5: a state larger than the window is recorded as skipped with its
// size, the adapter is never called, and the host decides alone.
func TestShadowSkipsAStateLargerThanTheWindow(t *testing.T) {
	adapter, rec := &fake{value: "iss-2", p: 0.9}, &pairs{}
	rt := router(oracle.JudgementShadow, adapter, rec)
	rt.Window = 100
	s := State{Text: strings.Repeat("word ", 400)}
	a, rc, err := rt.Decide(context.Background(), s, dupQuestion(), Answered("iss-1"))
	if err != nil || a.Value != "iss-1" || rc.Route != RouteHost {
		t.Fatalf("answer %+v, receipt %+v, err %v", a, rc, err)
	}
	if adapter.calls != 0 {
		t.Fatal("the adapter was called on a state larger than its window")
	}
	if len(rec.got) != 1 || rec.got[0].Skipped != lab.SkippedWindow || rec.got[0].StateTokens <= 100 ||
		!strings.Contains(rec.got[0].Detail, "window of 100") || rec.got[0].Adapter != "" {
		t.Fatalf("pairs = %+v", rec.got)
	}
	if rc.PairSkipped != lab.SkippedWindow || rc.StateTokens != rec.got[0].StateTokens {
		t.Fatalf("receipt = %+v", rc)
	}
}

// A shadow adapter that fails changes nothing for the host's decision; the
// pair is recorded as skipped, naming why.
func TestShadowRecordsAFailedAdapterAndTheHostStillDecides(t *testing.T) {
	adapter, rec := &fake{err: errors.New("connection refused")}, &pairs{}
	a, _, err := router(oracle.JudgementShadow, adapter, rec).Decide(context.Background(), State{Text: "s"}, dupQuestion(), Answered("iss-2"))
	if err != nil || a.Value != "iss-2" {
		t.Fatalf("answer %+v, err %v", a, err)
	}
	if len(rec.got) != 1 || rec.got[0].Skipped != lab.SkippedFailed || !strings.Contains(rec.got[0].Detail, "connection refused") {
		t.Fatalf("pairs = %+v", rec.got)
	}
}

// A caller stopped before or while the adapter is asked in shadow keeps no
// pair: the adapter's silence is the stop's, not the adapter's, and the host
// still decides.
func TestShadowKeepsNoPairForAStoppedCaller(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	adapter, rec := &fake{value: "iss-1", p: 0.9}, &pairs{}
	a, rc, err := router(oracle.JudgementShadow, adapter, rec).Decide(ctx, State{Text: "s"}, dupQuestion(), Answered("iss-2"))
	if err != nil || a.Value != "iss-2" || adapter.calls != 0 || len(rec.got) != 0 || rc.Pair != "" || !strings.Contains(rc.Note, "stopped") {
		t.Fatalf("an already stopped caller: answer %+v, receipt %+v, err %v, calls %d, pairs %+v", a, rc, err, adapter.calls, rec.got)
	}
	// Stopped while the adapter answers: its failure is the stop's.
	ctx, cancel = context.WithCancel(context.Background())
	stopping := &fake{err: context.Canceled}
	stop := func(oracle.Target) Judge { cancel(); return stopping }
	r := newRouter(map[Type]oracle.Target{DuplicateMatch: {Provider: "openrouter", Model: "typesafe/jev-1.13", Mode: oracle.JudgementShadow}}, stop, rec.record)
	if _, rc, err := r.Decide(ctx, State{Text: "s"}, dupQuestion(), Answered("iss-2")); err != nil || len(rec.got) != 0 || !strings.Contains(rc.Note, "stopped") {
		t.Fatalf("a caller stopped mid-call: receipt %+v, err %v, pairs %+v", rc, err, rec.got)
	}
	// A deadline the caller set for the adapter alone is the adapter's
	// slowness, and its pair is kept as failed.
	slow, cancelSlow := context.WithTimeout(context.Background(), 0)
	defer cancelSlow()
	if _, _, err := router(oracle.JudgementShadow, &fake{err: context.DeadlineExceeded}, rec).Decide(slow, State{Text: "s"}, dupQuestion(), Answered("iss-2")); err != nil ||
		len(rec.got) != 1 || rec.got[0].Skipped != lab.SkippedFailed {
		t.Fatalf("a slow adapter's pair is kept as failed: err %v, pairs %+v", err, rec.got)
	}
}

// No lab minted: the host decides, and the receipt says the pair was not kept.
func TestShadowWithNoLabStillDecidesAndSaysSo(t *testing.T) {
	rec := &pairs{err: lab.ErrNoLab}
	a, rc, err := router(oracle.JudgementShadow, &fake{value: "none", p: 0.5}, rec).Decide(context.Background(), State{Text: "s"}, dupQuestion(), Answered("none"))
	if err != nil || a.Value != "none" || !strings.Contains(rc.Note, "no lab minted") {
		t.Fatalf("answer %+v, receipt %+v, err %v", a, rc, err)
	}
}

// Criterion 3: turned on, the adapter decides and the receipt names the route.
func TestOnTheAdapterDecidesAndTheReceiptNamesTheRoute(t *testing.T) {
	adapter, rec := &fake{value: "iss-2", p: 0.93}, &pairs{}
	host := &fake{value: "iss-1", p: 1}
	a, rc, err := router(oracle.JudgementOn, adapter, rec).Decide(context.Background(), State{Text: "s"}, dupQuestion(), host)
	if err != nil {
		t.Fatal(err)
	}
	if a.Value != "iss-2" || a.Probability != 0.93 || host.calls != 0 {
		t.Fatalf("answer %+v, host calls %d: the adapter decides", a, host.calls)
	}
	if rc.Route != RouteAdapter || rc.Target != "openrouter/typesafe/jev-1.13" || rc.ModelReported != "typesafe/jev-1.13-0901" ||
		rc.Fallback != "" || len(rec.got) != 0 {
		t.Fatalf("receipt %+v, pairs %d", rc, len(rec.got))
	}
}

// Criterion 3: an unreachable adapter falls back to the host with a receipt
// saying so.
func TestOnAnUnreachableAdapterFallsBackToTheHostWithAReceipt(t *testing.T) {
	unreachable := &fake{err: openaiapi.ErrUnreachable}
	host := &fake{value: "iss-1", p: 1}
	a, rc, err := router(oracle.JudgementOn, unreachable, &pairs{}).Decide(context.Background(), State{Text: "s"}, dupQuestion(), host)
	if err != nil || a.Value != "iss-1" || host.calls != 1 {
		t.Fatalf("answer %+v, host calls %d, err %v", a, host.calls, err)
	}
	if rc.Route != RouteHost || !strings.Contains(rc.Fallback, "openrouter/typesafe/jev-1.13") || !strings.Contains(rc.Fallback, "the host decided") {
		t.Fatalf("receipt = %+v", rc)
	}
	// A host asked out of band (the drain's request file) is pending: the
	// fallback is still in the receipt.
	_, rc, err = router(oracle.JudgementOn, unreachable, &pairs{}).Decide(context.Background(), State{Text: "s"}, dupQuestion(), Pending())
	if !errors.Is(err, ErrHostPending) || rc.Fallback == "" {
		t.Fatalf("pending host: receipt %+v, err %v", rc, err)
	}
	// A state over the window is a fallback too, never a call.
	rt := router(oracle.JudgementOn, unreachable, &pairs{})
	rt.Window = 10
	unreachable.calls = 0
	_, rc, _ = rt.Decide(context.Background(), State{Text: strings.Repeat("x", 400)}, dupQuestion(), host)
	if unreachable.calls != 0 || !strings.Contains(rc.Fallback, "window") {
		t.Fatalf("over-window on: calls %d, receipt %+v", unreachable.calls, rc)
	}
}

// Criterion 6: one interface, so a type is switched by configuration alone.
// The same caller, the same arguments: host, shadow, on.
func TestOneInterfaceSwitchesATypeWithoutTheCallerChanging(t *testing.T) {
	caller := func(rt *Router) (Answer, Receipt) {
		a, rc, err := rt.Decide(context.Background(), State{Text: "s"}, dupQuestion(), Answered("iss-1"))
		if err != nil {
			t.Fatal(err)
		}
		return a, rc
	}
	for _, c := range []struct {
		mode, value, route string
		pairs              int
	}{
		{oracle.JudgementHost, "iss-1", RouteHost, 0},
		{oracle.JudgementShadow, "iss-1", RouteHost, 1},
		{oracle.JudgementOn, "iss-2", RouteAdapter, 0},
	} {
		adapter, rec := &fake{value: "iss-2", p: 0.9}, &pairs{}
		a, rc := caller(router(c.mode, adapter, rec))
		if a.Value != c.value || rc.Route != c.route || len(rec.got) != c.pairs || rc.Mode != c.mode {
			t.Errorf("%s: answer %+v receipt %+v pairs %d", c.mode, a, rc, len(rec.got))
		}
		if c.mode == oracle.JudgementHost && adapter.calls != 0 {
			t.Errorf("host: the adapter was called")
		}
	}
	// Both implementations satisfy the one interface.
	var _ Judge = Host{}
	var _ Judge = Adapter{}
}

// The registry is closed: a type outside it, or a question that does not hold
// together, is refused before anything is asked.
func TestTheRegistryIsClosedAndAQuestionIsChecked(t *testing.T) {
	rt := router(oracle.JudgementOn, &fake{value: "x"}, &pairs{})
	host := &fake{value: "none"}
	if _, _, err := rt.Decide(context.Background(), State{Text: "s"}, Question{Type: "lesson-polish", Kind: KindChoose, Ask: "?", Options: []string{"a"}}, host); err == nil {
		t.Fatal("a type outside the registry was asked")
	}
	if _, _, err := rt.Decide(context.Background(), State{Text: "s"}, Question{Type: DuplicateMatch, Kind: KindChoose, Ask: "?"}, host); err == nil {
		t.Fatal("a choose question with no options was asked")
	}
	if host.calls != 0 {
		t.Fatal("the host was asked a refused question")
	}
	for _, ty := range types() {
		if !known(string(ty)) {
			t.Errorf("%s is listed but not known", ty)
		}
	}
	// Embark's lesson ranking is a judgement routed through the adapter in
	// place of its term-overlap ordering (iss-2610262033055327).
	if !known(string(EmbarkLessonRanking)) || EmbarkLessonRanking != "embark-lesson-ranking" {
		t.Error("embark's lesson ranking is not in the registry")
	}
}

// An adapter's answer outside the question's options is not an answer: turned
// on, the host decides instead, with a receipt.
func TestOnAnAnswerOutsideTheOptionsFallsBack(t *testing.T) {
	host := &fake{value: "none"}
	a, rc, err := router(oracle.JudgementOn, &fake{value: "iss-9", p: 0.99}, &pairs{}).Decide(context.Background(), State{Text: "s"}, dupQuestion(), host)
	if err != nil || a.Value != "none" || !strings.Contains(rc.Fallback, "not one of") {
		t.Fatalf("answer %+v receipt %+v err %v", a, rc, err)
	}
}

func TestTheHostParsesItsAnswerAtProbabilityOne(t *testing.T) {
	for _, c := range []struct {
		q    Question
		text string
		want string
		bad  bool
	}{
		{dupQuestion(), " ISS-2 \n", "iss-2", false},
		{dupQuestion(), "iss-7", "", true},
		{Question{Type: RemedyRead, Kind: KindYesNo, Ask: "Does it change what a user sees?"}, "Yes", "yes", false},
		{Question{Type: LedgerSeverity, Kind: KindScore, Ask: "How severe?", Min: 1, Max: 5}, "4", "4", false},
		{Question{Type: LedgerSeverity, Kind: KindScore, Ask: "How severe?", Min: 1, Max: 5}, "9", "", true},
	} {
		a, err := Answered(c.text).Judge(context.Background(), State{Text: "s"}, c.q)
		if c.bad {
			if err == nil {
				t.Errorf("%q: parsed as %+v, want a refusal", c.text, a)
			}
			continue
		}
		if err != nil || a.Value != c.want || a.Probability != 1 {
			t.Errorf("%q: %+v, %v", c.text, a, err)
		}
	}
	// The host is asked the prompt the adapter is asked: like for like.
	var asked string
	h := Host{Ask: func(_ context.Context, prompt string) (string, error) { asked = prompt; return "none", nil }}
	if _, err := h.Judge(context.Background(), State{Text: "the state"}, dupQuestion()); err != nil {
		t.Fatal(err)
	}
	if asked != render(State{Text: "the state"}, dupQuestion()) || !strings.Contains(asked, "the state") || !strings.Contains(asked, "iss-2") {
		t.Fatalf("host prompt = %q", asked)
	}
}

// caller is a fake oracle call: it checks the request and answers content.
type caller struct {
	content string
	req     oracle.CallRequest
}

func (c *caller) Call(_ context.Context, _ credential.Source, req oracle.CallRequest, _ ...openaiapi.Option) ([]byte, oracle.CallRecord, error) {
	c.req = req
	if req.Contract != nil {
		if err := req.Contract([]byte(c.content)); err != nil {
			return nil, oracle.CallRecord{}, err
		}
	}
	return []byte(c.content), oracle.CallRecord{Provider: req.Target.Provider, ModelAsked: req.Target.Model, ModelReported: "typesafe/jev-1.13-0901"}, nil
}

func TestTheAPIAdapterParsesTheTypedAnswer(t *testing.T) {
	c := &caller{content: "```json\n{\"value\": \"iss-2\", \"probability\": 0.87}\n```"}
	a, err := Adapter{Calls: c, Target: target}.Judge(context.Background(), State{Text: "s"}, dupQuestion())
	if err != nil || a.Value != "iss-2" || a.Probability != 0.87 || a.Model != "typesafe/jev-1.13-0901" {
		t.Fatalf("answer %+v, err %v", a, err)
	}
	if c.req.Target != target || !strings.Contains(c.req.Brief.Input, "iss-1") || !strings.Contains(c.req.Brief.Instructions, "probability") {
		t.Fatalf("request = %+v", c.req)
	}
	for _, bad := range []string{`{"value":"iss-2","probability":1.5}`, `{"value":"iss-9","probability":0.5}`, `not json`, `{"value":"iss-2"}`} {
		c.content = bad
		if a, err := (Adapter{Calls: c, Target: target}).Judge(context.Background(), State{Text: "s"}, dupQuestion()); err == nil {
			t.Errorf("%s: parsed as %+v", bad, a)
		}
	}
}

// Criterion 4: a configured model not on its provider's list is refused when
// the configuration is read, before any call; a listed one loads with its mode.
func TestLoadRefusesAnUnlistedModelBeforeAnyCall(t *testing.T) {
	roots := layered.Roots{Repo: t.TempDir(), Home: t.TempDir()}
	write := func(body string) {
		p := abcdhome.Path(roots.Home, "config.json")
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	block := `"api":{"openrouter":{"base_url":"https://openrouter.ai/api/v1","key":"openrouter","models":["typesafe/jev-1.13"]}}`
	write(`{"oracle":{` + block + `,"judgements":{"duplicate-match":"shadow:openrouter/mistral/small"}}}`)
	if _, err := Load(roots, roots.Repo); err == nil || !strings.Contains(err.Error(), "refused before any call") {
		t.Fatalf("Load = %v, want the allowlist refusal", err)
	}
	write(`{"oracle":{` + block + `,"judgements":{"duplicate-match":"shadow:openrouter/typesafe/jev-1.13",
		"remedy-read":"openrouter/typesafe/jev-1.13","lesson-polish":"openrouter/typesafe/jev-1.13"}}}`)
	rt, err := Load(roots, roots.Repo)
	if err != nil {
		t.Fatal(err)
	}
	if rt.Mode(DuplicateMatch) != oracle.JudgementShadow || rt.Mode(RemedyRead) != oracle.JudgementOn || rt.Mode(MemoryCheck) != oracle.JudgementHost {
		t.Fatalf("modes: %s %s %s", rt.Mode(DuplicateMatch), rt.Mode(RemedyRead), rt.Mode(MemoryCheck))
	}
	if len(rt.Diagnostics) != 3 || !strings.Contains(strings.Join(rt.Diagnostics, "\n"), "lesson-polish") {
		t.Fatalf("diagnostics = %q, want the unknown type named", rt.Diagnostics)
	}
}

// A route to a type in the registry that no step asks through the router yet
// is kept, and said: the host decides it alone and nothing runs beside it, so
// a shadow route there records no pair. A route to a type a step asks is not
// remarked on.
func TestLoadSaysARouteToATypeNoStepAsksYetRunsNothing(t *testing.T) {
	roots := layered.Roots{Repo: t.TempDir(), Home: t.TempDir()}
	p := abcdhome.Path(roots.Home, "config.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(`{"oracle":{"api":{"openrouter":{"base_url":"https://openrouter.ai/api/v1","key":"openrouter",
		"models":["typesafe/jev-1.13"]}},"judgements":{"duplicate-match":"shadow:openrouter/typesafe/jev-1.13",
		"eligibility-residual":"shadow:openrouter/typesafe/jev-1.13","embark-lesson-ranking":"shadow:openrouter/typesafe/jev-1.13"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	rt, err := Load(roots, roots.Repo)
	if err != nil {
		t.Fatal(err)
	}
	if rt.Mode(DuplicateMatch) != oracle.JudgementShadow {
		t.Fatalf("the route is kept: %s", rt.Mode(DuplicateMatch))
	}
	if len(rt.Diagnostics) != 1 || !strings.Contains(rt.Diagnostics[0], "oracle.judgements.duplicate-match") ||
		!strings.Contains(rt.Diagnostics[0], "no step asks") || !strings.Contains(rt.Diagnostics[0], "no pair") {
		t.Fatalf("diagnostics = %q, want the one route that runs nothing named", rt.Diagnostics)
	}
}
