package matrix

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/intentdriven/abcd/internal/core/oracle"
)

// planAPI is this machine's provider configuration for the plan tests: an
// OpenRouter block, a block on this machine, and a paid provider whose base
// URL is paidURL.
func planAPI(t *testing.T, paidURL string) *oracle.APIConfig {
	t.Helper()
	machine := `{"oracle":{"api":{` +
		`"openrouter":{"base_url":"https://openrouter.ai/api/v1","key":"openrouter","models":["vendor/model","vendor/unpinned"],` +
		`"pins":{"vendor/model":{"order":["deepinfra"],"quantizations":["fp8"],"canonical_slug":"vendor/model-20240723"}}},` +
		`"local":{"base_url":"http://127.0.0.1:11535/v1","models":["qwen","bare"],"pins":{"qwen":{"runtime":"ollama","runtime_version":"0.14.2",` +
		`"weights_sha256":"sha256:06C1097EFCE0431C2045FE7B2E5108366E43BEE1B4603A7ADED8F21689E90BCA","quantisation":"Q4_K_M","context_length":32768,"kv_cache_type":"f16"}}},` +
		`"paid":{"base_url":"` + paidURL + `","key":"paid","models":["big"]}}}}`
	api, err := oracle.LoadAPI(capRoots(t, "", machine))
	if err != nil {
		t.Fatal(err)
	}
	return api
}

// planDecl is a declaration with the given builders and reviewers, one task
// of each kind, a pilot of 3 and a most of 10 repeats.
func planDecl(builders, reviewers, extra string) []byte {
	return []byte(`{
  "schema_version": 1,
  "name": "routing check",
  "tasks": [
    {"id": "feature-a", "kind": "code", "intent": "itd-2610161733316518"},
    {"id": "doc-page", "kind": "writing", "source": ["docs/guide.md"], "criteria": "accurate against the source first"},
    {"id": "fidelity", "kind": "check", "material": ["docs/reference"]}
  ],
  "builders": [` + builders + `],
  "reviewers": [` + reviewers + `],
  "repeats": {"pilot": 3, "max": 10},` + extra + `
  "concurrency": 2,
  "judging": {"guests_may_judge": false}
}`)
}

func anyRoute(oracle.FlagRoute) error { return nil }

func plan(t *testing.T, decl []byte, c ResolvedCap) (Plan, error) {
	t.Helper()
	return buildPlan(PlanInputs{Lab: "lab-1", FrozenCommit: strings.Repeat("a", 40), Declaration: decl,
		API: planAPI(t, "https://api.example.com/v1"), CheckRoute: anyRoute, Cap: c, Cores: 16})
}

var usd25 = ResolvedCap{Cap: &Money{Amount: 25, Unit: "usd"}, Origin: CapFromRun}

func TestThePlanCountsPilotAndMaximumCopiesPerCell(t *testing.T) {
	p, err := plan(t, planDecl(`{"id":"host-big","route":"host"},{"id":"or-cheap","route":"builder=runner:opencode@openrouter/vendor/model"}`,
		`{"id":"local-rev","route":"reviewer=runner:opencode@local/qwen"}`, `"load_cap": 4,`), usd25)
	if err != nil {
		t.Fatal(err)
	}
	// Three cells, three tasks each: a pilot of 3 and a most of 10 repeats.
	if p.Copies.PilotTotal != 27 || p.Copies.MaxTotal != 90 || p.Copies.MaxByCell["or-cheap"] != 30 {
		t.Fatalf("copies = %+v; want pilot 27, most 90, 30 for each cell", p.Copies)
	}
}

// The claude runner is bare on its API key, not the host's harness: a cell on
// it names its model like any runner's, and is never planned as the host.
func TestAClaudeRunnerCellIsNotTheHost(t *testing.T) {
	_, err := plan(t, planDecl(`{"id":"host-big","route":"host"},{"id":"claude-own","route":"builder=runner:claude"}`, "", ""), ResolvedCap{})
	if err == nil || !strings.Contains(err.Error(), "claude-own") || !strings.Contains(err.Error(), "no model") {
		t.Fatalf("buildPlan = %v; want the claude cell refused for naming no model", err)
	}
}

// A figure of money past the bound is refused, so a plan's sums stay finite
// and every plan has a digest of its own.
func TestAnAmountPastTheBoundIsRefused(t *testing.T) {
	_, err := plan(t, planDecl(`{"id":"host-big","route":"host"},{"id":"paid-big","route":"builder=runner:opencode@paid/big","cost_per_copy":{"amount":1e308,"unit":"usd"}}`, "", ""), usd25)
	if err == nil || !strings.Contains(err.Error(), "at most") {
		t.Fatalf("buildPlan = %v; want the amount refused past the bound", err)
	}
}

// A plan that cannot be encoded has no digest, so it cannot be consented to.
func TestAPlanThatCannotBeEncodedHasNoDigest(t *testing.T) {
	p, err := plan(t, planDecl(`{"id":"host-big","route":"host"}`, "", ""), ResolvedCap{})
	if err != nil {
		t.Fatal(err)
	}
	p.Cost = append(p.Cost, SourceCost{Source: SourceProvider, Metered: true, Expected: &Money{Amount: math.Inf(1), Unit: "usd"}})
	if d, err := p.digest(); err == nil || d != "" {
		t.Fatalf("digest = %q, %v; want no digest for a plan that does not encode", d, err)
	}
}

func TestThePlanNamesEachCellsSource(t *testing.T) {
	p, err := plan(t, planDecl(`{"id":"host-big","route":"host"},{"id":"or-cheap","route":"builder=runner:opencode@openrouter/vendor/model"},{"id":"paid-big","route":"builder=runner:opencode@paid/big"}`,
		`{"id":"local-rev","route":"reviewer=runner:opencode@local/qwen"}`, `"load_cap": 4,`), usd25)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]Source{"host-big": SourceHost, "or-cheap": SourceOpenRouter, "paid-big": SourceProvider, "local-rev": SourceLocal}
	for _, c := range p.Cells {
		if c.Source != want[c.Cell] {
			t.Errorf("cell %s source %q, want %q", c.Cell, c.Source, want[c.Cell])
		}
	}
	if len(p.Cells) != len(want) {
		t.Fatalf("cells = %+v", p.Cells)
	}
	for _, c := range p.Cells {
		switch c.Source {
		case SourceOpenRouter:
			if c.ProviderPin == nil || c.ProviderPin.Order[0] != "deepinfra" || c.LocalPin != nil {
				t.Errorf("OpenRouter cell %s pins = %+v, %+v", c.Cell, c.ProviderPin, c.LocalPin)
			}
		case SourceLocal:
			if c.LocalPin == nil || c.LocalPin.Runtime != "ollama" || c.ProviderPin != nil {
				t.Errorf("local cell %s pins = %+v, %+v", c.Cell, c.ProviderPin, c.LocalPin)
			}
		}
	}
}

func TestThePlanShowsExpectedCostBySourceAgainstTheCap(t *testing.T) {
	p, err := plan(t, planDecl(`{"id":"host-big","route":"host"},{"id":"or-cheap","route":"builder=runner:opencode@openrouter/vendor/model","cost_per_copy":{"amount":0.05,"unit":"usd"}}`,
		``, ``), usd25)
	if err != nil {
		t.Fatal(err)
	}
	if p.SpendCap == nil || p.SpendCap.Amount != 25 || p.SpendCapOrigin != CapFromRun {
		t.Fatalf("cap = %+v from %q", p.SpendCap, p.SpendCapOrigin)
	}
	got := map[Source]SourceCost{}
	for _, c := range p.Cost {
		got[c.Source] = c
	}
	// or-cheap makes 30 copies at 0.05 usd each.
	if c := got[SourceOpenRouter]; !c.Metered || c.Expected == nil || c.Expected.Amount != 1.5 {
		t.Fatalf("openrouter cost = %+v; want 1.5 usd expected", c)
	}
	if c := got[SourceHost]; c.Metered || c.Expected != nil {
		t.Fatalf("host cost = %+v; want unmetered", c)
	}
	// A metered cell with no declared price is not estimated, never fetched.
	p, err = plan(t, planDecl(`{"id":"or-cheap","route":"builder=runner:opencode@openrouter/vendor/model"}`, ``, ``), usd25)
	if err != nil {
		t.Fatal(err)
	}
	if c := p.Cost[0]; c.Expected != nil || c.Note != "not estimated; bounded by your cap" {
		t.Fatalf("cost = %+v; want not estimated, bounded by the cap", c)
	}
}

func TestAMeteredRunWithoutANamedCapIsRefused(t *testing.T) {
	for _, route := range []string{"builder=runner:opencode@openrouter/vendor/model", "builder=runner:opencode@paid/big"} {
		_, err := plan(t, planDecl(`{"id":"cell","route":"`+route+`"}`, ``, ``), ResolvedCap{})
		if err == nil || !strings.Contains(err.Error(), "no spending cap") {
			t.Fatalf("%s: err = %v; want a refusal for the missing cap", route, err)
		}
	}
}

func TestALocalOrHostOnlyRunNeedsNoCap(t *testing.T) {
	p, err := plan(t, planDecl(`{"id":"host-big","route":"host"}`, `{"id":"local-rev","route":"reviewer=runner:opencode@local/qwen"}`, `"load_cap": 4,`), ResolvedCap{})
	if err != nil || p.SpendCap != nil {
		t.Fatalf("plan = %+v, %v; want a plan with no cap", p.SpendCap, err)
	}
}

func TestThePlanShowsTheLoadCapTheCoresTimeAndJudges(t *testing.T) {
	p, err := plan(t, planDecl(`{"id":"host-big","route":"host","minutes_per_copy":6}`,
		`{"id":"local-rev","route":"reviewer=runner:opencode@local/qwen","minutes_per_copy":2}`, `"load_cap": 4,`), ResolvedCap{})
	if err != nil {
		t.Fatal(err)
	}
	if p.LoadCap == nil || *p.LoadCap != 4 || p.Cores != 16 || p.Concurrency != 2 {
		t.Fatalf("load cap %v of %d cores at concurrency %d", p.LoadCap, p.Cores, p.Concurrency)
	}
	// (6 + 2) minutes x 30 copies each, over a concurrency of 2.
	if p.Time.BoundMinutes == nil || *p.Time.BoundMinutes != 120 {
		t.Fatalf("time = %+v; want a 120-minute bound", p.Time)
	}
	kinds := map[TaskKind]JudgeNeed{}
	for _, j := range p.Judges {
		kinds[j.Kind] = j
	}
	if kinds[KindCode].Count != 1 || kinds[KindWriting].Count < 3 || kinds[KindCheck].Count != 1 {
		t.Fatalf("judges = %+v", p.Judges)
	}
	// Without a declared time on every cell, the time waits for the pilot.
	p, err = plan(t, planDecl(`{"id":"host-big","route":"host"}`, ``, ``), ResolvedCap{})
	if err != nil || p.Time.BoundMinutes != nil || p.Time.Note != "known after the pilot" {
		t.Fatalf("time = %+v, %v; want known after the pilot", p.Time, err)
	}
}

func TestAWritingTaskNeedsEnoughJudgesThatEveryPairOfModelsMeets(t *testing.T) {
	var b []string
	for i := range 12 {
		b = append(b, `{"id":"h`+string(rune('a'+i))+`","route":"host"}`)
	}
	p, err := plan(t, planDecl(strings.Join(b, ","), ``, ``), ResolvedCap{})
	if err != nil {
		t.Fatal(err)
	}
	for _, j := range p.Judges {
		if j.Kind == KindWriting && (j.Count < 3 || !strings.Contains(j.Note, "at most 8")) {
			t.Fatalf("writing judges = %+v; want at least 3, each ranking at most 8", j)
		}
	}
}

func TestALoadCapAtOrAboveTheCoreCountIsRefused(t *testing.T) {
	for _, extra := range []string{`"load_cap": 16,`, `"load_cap": 20,`, ``} {
		_, err := plan(t, planDecl(`{"id":"host-big","route":"host"}`, `{"id":"local-rev","route":"reviewer=runner:opencode@local/qwen"}`, extra), ResolvedCap{})
		if err == nil || !strings.Contains(err.Error(), "load_cap") {
			t.Fatalf("%q: err = %v; want the load cap refused", extra, err)
		}
	}
}

func TestACellRouteTheRunnerConfigRefusesIsRefused(t *testing.T) {
	decl := planDecl(`{"id":"or-cheap","route":"builder=runner:opencode@openrouter/vendor/model"}`, ``, ``)
	_, err := buildPlan(PlanInputs{Lab: "lab-1", FrozenCommit: strings.Repeat("a", 40), Declaration: decl,
		API: planAPI(t, "https://api.example.com/v1"), Cap: usd25, Cores: 16,
		CheckRoute: func(oracle.FlagRoute) error { return errors.New("runner opencode is not enabled on this machine") }})
	if err == nil || !strings.Contains(err.Error(), "not enabled") || !strings.Contains(err.Error(), "or-cheap") {
		t.Fatalf("err = %v; want the runner config's refusal, naming the cell", err)
	}
	for _, c := range []struct{ route, want string }{
		{"builder=runner:opencode", "names its model"},
		{"builder=runner:opencode@openrouter/unlisted", "not on openrouter's list"},
		{"reviewer=runner:opencode@local/qwen", "reviewer"},
		{"builder=strong", "runner"},
		{"builder=runner:opencode@openrouter/vendor/unpinned", "no provider pin"},
		{"builder=runner:opencode@local/bare", "no pin"},
	} {
		if _, err := plan(t, planDecl(`{"id":"cell","route":"`+c.route+`"}`, ``, ``), usd25); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v; want a refusal mentioning %q", c.route, err, c.want)
		}
	}
}

// TestARefusedCellRouteIsNamedAsDeclared: the route parser names a typed
// --route by its position and a part it does not know by its length; a cell's
// route is the declaration's own text, so its refusal quotes it as written.
func TestARefusedCellRouteIsNamedAsDeclared(t *testing.T) {
	for _, route := range []string{"builder=strong-tier", "builder=runner:opencode@notamodel"} {
		_, err := plan(t, planDecl(`{"id":"cell","route":"`+route+`"}`, ``, ``), usd25)
		if err == nil || !strings.Contains(err.Error(), `the declared route "`+route+`"`) {
			t.Errorf("%s: err = %v; want the refusal to name the declared route as written", route, err)
		}
	}
	decl := planDecl(`{"id":"or-cheap","route":"builder=runner:opencode@openrouter/vendor/model"}`, ``, ``)
	_, err := buildPlan(PlanInputs{Lab: "lab-1", FrozenCommit: strings.Repeat("a", 40), Declaration: decl,
		API: planAPI(t, "https://api.example.com/v1"), Cap: usd25, Cores: 16,
		CheckRoute: func(oracle.FlagRoute) error { return errors.New("runner opencode is not enabled on this machine") }})
	if err == nil || !strings.Contains(err.Error(), `the declared route "builder=runner:opencode@openrouter/vendor/model"`) {
		t.Fatalf("err = %v; want the runner config's refusal to name the declared route", err)
	}
}

func TestBuildingThePlanMakesNoNetworkCall(t *testing.T) {
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }))
	defer srv.Close()
	// A pinned block served by the counting server: building the plan reads
	// its configuration and must never ask the server anything (decision 12).
	machine := `{"oracle":{"api":{"probe":{"base_url":"` + srv.URL + `/v1","models":["qwen"],"pins":{"qwen":{"runtime":"ollama","runtime_version":"0.14.2",` +
		`"weights_sha256":"sha256:06C1097EFCE0431C2045FE7B2E5108366E43BEE1B4603A7ADED8F21689E90BCA","quantisation":"Q4_K_M","context_length":32768,"kv_cache_type":"f16"}}}}}}`
	api, err := oracle.LoadAPI(capRoots(t, "", machine))
	if err != nil {
		t.Fatal(err)
	}
	decl := planDecl(`{"id":"probe-cell","route":"builder=runner:opencode@probe/qwen"}`, ``, `"load_cap": 4,`)
	if _, err := buildPlan(PlanInputs{Lab: "lab-1", FrozenCommit: strings.Repeat("a", 40), Declaration: decl,
		API: api, CheckRoute: anyRoute, Cap: ResolvedCap{}, Cores: 16}); err != nil {
		t.Fatal(err)
	}
	if n := calls.Load(); n != 0 {
		t.Fatalf("building the plan made %d network calls; want none before consent", n)
	}
}

func TestThePlanDigestChangesWithWhatTheRunSpends(t *testing.T) {
	decl := planDecl(`{"id":"or-cheap","route":"builder=runner:opencode@openrouter/vendor/model"}`, ``, ``)
	a, err := plan(t, decl, usd25)
	if err != nil {
		t.Fatal(err)
	}
	b, err := plan(t, decl, ResolvedCap{Cap: &Money{Amount: 26, Unit: "usd"}, Origin: CapFromRun})
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Digest) != 64 || a.Digest == b.Digest {
		t.Fatalf("digests %q and %q: want two different sha256 values for two caps", a.Digest, b.Digest)
	}
	if again, _ := plan(t, decl, usd25); again.Digest != a.Digest {
		t.Fatal("the same plan built twice has two digests")
	}
}

// A matcher and a proposer are planned as cells of their own, outside the
// copies: each answers once a check task, so its cost is reckoned on that, and
// the check's judges say who proposes a verdict and what joins the findings no
// exact key joins. Adding either changes the plan's digest (spec step 9).
func TestAMatcherAndAProposerArePlannedAsTheChecksJudges(t *testing.T) {
	builders := `{"id":"host-big","route":"host"}`
	bare, err := plan(t, planDecl(builders, ``, ``), ResolvedCap{})
	if err != nil {
		t.Fatal(err)
	}
	check := func(p Plan) JudgeNeed {
		for _, j := range p.Judges {
			if j.Kind == KindCheck {
				return j
			}
		}
		t.Fatalf("judges = %+v; want the check's", p.Judges)
		return JudgeNeed{}
	}
	// With no matcher, the leftovers are stated as unmatched rows, and with
	// no proposer no verdict is proposed.
	if j := check(bare); j.Count != 1 || !strings.Contains(j.Who, "no verdict proposed") || !strings.Contains(j.Who, "stay unmatched") ||
		!strings.Contains(j.Note, "not estimable") {
		t.Fatalf("the check's judges with none declared = %+v", j)
	}
	matched, err := plan(t, planDecl(builders, ``, `
  "matcher": {"id": "match-or", "route": "matcher=runner:opencode@openrouter/vendor/model", "cost_per_copy": {"amount": 0.25, "unit": "usd"}},`), usd25)
	if err != nil {
		t.Fatal(err)
	}
	// A proposer on the host builder's own route would judge its own copies.
	if _, err := plan(t, planDecl(builders, ``, `
  "proposer": {"id": "propose-host", "route": "host"},`), ResolvedCap{}); err == nil || !strings.Contains(err.Error(), "builder cell host-big") {
		t.Fatalf("a proposer on the builder's route = %v; want it refused", err)
	}
	proposed, err := plan(t, planDecl(builders, ``, `
  "matcher": {"id": "match-or", "route": "matcher=runner:opencode@openrouter/vendor/model", "cost_per_copy": {"amount": 0.25, "unit": "usd"}},
  "proposer": {"id": "propose-local", "route": "proposer=runner:opencode@local/qwen"},
  "load_cap": 4,`), usd25)
	if err != nil {
		t.Fatal(err)
	}
	if j := check(matched); j.Count != 1 || !strings.Contains(j.Who, "the matcher cell match-or") || strings.Contains(j.Note, "not estimable") {
		t.Fatalf("the check's judges with a matcher = %+v", j)
	}
	if j := check(proposed); j.Count != 2 || !strings.Contains(j.Who, "the proposer cell propose-local") {
		t.Fatalf("the check's judges with a proposer = %+v", j)
	}
	// A judge on a runner route waits for a launcher, and the plan says so
	// before consent, naming each such judge.
	for _, c := range []struct {
		j    JudgeNeed
		want []string
	}{{check(matched), []string{"match-or is a runner cell"}}, {check(proposed), []string{"match-or is a runner cell", "propose-local is a runner cell"}}} {
		for _, w := range c.want {
			if !strings.Contains(c.j.Note, w) || !strings.Contains(c.j.Note, "waits for a launcher") {
				t.Errorf("the judges note = %q; want it to say %s and waits for a launcher", c.j.Note, w)
			}
		}
	}
	if j := check(bare); strings.Contains(j.Note, "launcher") {
		t.Errorf("the judges note with no judge declared = %q; want no wait named", j.Note)
	}
	roles := map[string]Role{}
	for _, c := range proposed.Cells {
		roles[c.Cell] = c.Role
	}
	if roles["match-or"] != RoleMatcher || roles["propose-local"] != RoleProposer {
		t.Fatalf("cells = %+v; want the matcher and the proposer planned", proposed.Cells)
	}
	// The judges make no copy: the copies are the host builder's alone, and
	// the matcher's cost is its one answer on the one check task.
	if proposed.Copies.MaxTotal != 30 || len(proposed.Copies.MaxByCell) != 1 {
		t.Fatalf("copies = %+v; want the builder's 30 alone", proposed.Copies)
	}
	for _, c := range matched.Cost {
		if c.Source == SourceOpenRouter && (c.Expected == nil || c.Expected.Amount != 0.25) {
			t.Fatalf("the matcher's cost = %+v; want 0.25 usd, one answer on one check", c)
		}
	}
	if bare.Digest == matched.Digest || matched.Digest == proposed.Digest {
		t.Fatalf("digests %s, %s, %s: want each judge added to change the plan's", bare.Digest, matched.Digest, proposed.Digest)
	}
}

// The judges note asks the person to check the judges' routes wherever a
// judge is declared, on the check task's judges and once, and says nothing of
// it where no judge is: a plan with neither cell carries the sentence nowhere
// (spec step 9).
func TestThePlanAsksToCheckTheJudgesRoutesOnlyWhereAJudgeIsDeclared(t *testing.T) {
	const sentence = "a judge is held apart from the builders by its model's name read with its provider, carrier, variant tag and date set aside, so check that the judges' routes above name other models than the builders'"
	builders := `{"id":"host-big","route":"host"}`
	matcher := `
  "matcher": {"id": "match-or", "route": "matcher=runner:opencode@openrouter/vendor/model", "cost_per_copy": {"amount": 0.25, "unit": "usd"}},`
	proposer := `
  "proposer": {"id": "propose-local", "route": "proposer=runner:opencode@local/qwen"},
  "load_cap": 4,`
	for _, c := range []struct {
		name, extra string
		cap         ResolvedCap
		want        int
	}{
		{"no judge", ``, ResolvedCap{}, 0},
		{"a matcher", matcher, usd25, 1},
		{"a proposer", proposer, ResolvedCap{}, 1},
		{"a matcher and a proposer", matcher + proposer, usd25, 1},
	} {
		p, err := plan(t, planDecl(builders, ``, c.extra), c.cap)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		var all bytes.Buffer
		enc := json.NewEncoder(&all)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(p); err != nil {
			t.Fatal(err)
		}
		if n := strings.Count(all.String(), sentence); n != c.want {
			t.Errorf("%s: the plan carries the judges sentence %d times; want %d", c.name, n, c.want)
		}
		for _, j := range p.Judges {
			if n, want := strings.Count(j.Note, sentence), map[bool]int{true: c.want, false: 0}[j.Kind == KindCheck]; n != want {
				t.Errorf("%s: the %s task's judges note carries the sentence %d times; want %d: %q", c.name, j.Kind, n, want, j.Note)
			}
		}
	}
}

// A code task's hidden tests are shown on the plan with the line the person
// consents to: abcd will run each copy's code with that command, in an export
// of the copy, never in its clone (spec step 9; decision 9).
func TestAPlanCarryingHiddenTestsShowsTheConsentLine(t *testing.T) {
	builders := `{"id":"host-big","route":"host"}`
	bare, err := plan(t, planDecl(builders, ``, ``), ResolvedCap{})
	if err != nil {
		t.Fatal(err)
	}
	if bare.HiddenTests != nil {
		t.Fatalf("hidden tests = %+v; want none where no task names any", bare.HiddenTests)
	}
	decl := strings.Replace(string(planDecl(builders, ``, ``)), `"intent": "itd-2610161733316518"}`,
		`"intent": "itd-2610161733316518", "hidden_tests": "/work/hidden/feature-a", "test_command": ["go", "test", "-run", "Test Hidden", "One\u00a0Two", "g\u043e", "a`+"`"+`b", "./..."]}`, 1)
	p, err := plan(t, []byte(decl), ResolvedCap{})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.HiddenTests) != 1 {
		t.Fatalf("hidden tests = %+v; want the code task's", p.HiddenTests)
	}
	h := p.HiddenTests[0]
	// Every word holding a space, a Unicode one among them, is quoted, and so
	// is the folder; the line says abcd runs them, unsandboxed, and that the
	// result is that run's, never the host's report.
	// A letter outside ASCII is shown as its escape, and a backtick never
	// ends the command's fence.
	want := "abcd will run each copy's code of feature-a with `go test -run \"Test Hidden\" \"One\\u00a0Two\" \"g\\u043e\" \"a\\x60b\" ./...`, in an export of the copy with the hidden tests from \"/work/hidden/feature-a\" added, never in its clone, " +
		"as this account and not sandboxed; a code copy's result is that run's exit status, never the host's report"
	if h.Task != "feature-a" || h.Folder != "/work/hidden/feature-a" || len(h.Command) != 8 || h.Consent != want {
		t.Fatalf("hidden tests = %+v\nwant consent %q", h, want)
	}
	if p.Digest == bare.Digest {
		t.Fatal("naming hidden tests did not change the plan's digest")
	}
}
