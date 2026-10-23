package matrix

import (
	"errors"
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

func TestThePlanNamesEachCellsSource(t *testing.T) {
	p, err := plan(t, planDecl(`{"id":"host-big","route":"host"},{"id":"claude-own","route":"builder=runner:claude"},{"id":"or-cheap","route":"builder=runner:opencode@openrouter/vendor/model"},{"id":"paid-big","route":"builder=runner:opencode@paid/big"}`,
		`{"id":"local-rev","route":"reviewer=runner:opencode@local/qwen"}`, `"load_cap": 4,`), usd25)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]Source{"host-big": SourceHost, "claude-own": SourceHost, "or-cheap": SourceOpenRouter, "paid-big": SourceProvider, "local-rev": SourceLocal}
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
	if kinds[KindCode].Count != 1 || kinds[KindWriting].Count < 3 || kinds[KindCheck].Count != 2 {
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
