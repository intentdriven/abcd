package oracle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/layered"
)

// ---------------------------------------------------------------------------
// fixtures
// ---------------------------------------------------------------------------

type fx struct {
	t     *testing.T
	roots layered.Roots
}

func newFx(t *testing.T) *fx {
	t.Helper()
	base := t.TempDir()
	r := layered.Roots{Repo: filepath.Join(base, "repo"), Home: filepath.Join(base, "home")}
	for _, d := range []string{r.Repo, r.Home} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	return &fx{t: t, roots: r}
}

func (f *fx) put(abs, body string) {
	f.t.Helper()
	if err := os.MkdirAll(filepath.Dir(abs), 0o700); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o600); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fx) repo(agents string) {
	f.put(filepath.Join(f.roots.Repo, ".abcd", "config", "oracle-routing.json"),
		`{"schema_version":1,"agents":`+agents+`}`)
}

func (f *fx) machine(agents string) {
	f.put(filepath.Join(f.roots.Home, ".abcd", "oracle-routing.json"),
		`{"schema_version":1,"agents":`+agents+`}`)
}

func (f *fx) load() *Layered {
	f.t.Helper()
	l, err := Load(f.roots)
	if err != nil {
		f.t.Fatalf("Load: %v", err)
	}
	return l
}

// spy is a Connections that records whether it was consulted and serves the
// tiers it is given.
type spy struct {
	serves map[Tier]Connection
	named  map[string]Connection
	calls  int
}

func (s *spy) Serves(t Tier) (Connection, bool) {
	s.calls++
	c, ok := s.serves[t]
	return c, ok
}

func (s *spy) Named(name string) (Connection, bool) {
	s.calls++
	c, ok := s.named[name]
	return c, ok
}

func raw(v string) json.RawMessage { return json.RawMessage(v) }

// ---------------------------------------------------------------------------
// tier vocabulary
// ---------------------------------------------------------------------------

func TestTierVocabularyIsClosed(t *testing.T) {
	want := []Tier{Local, Economy, Frontier, HostDecides}
	if !reflect.DeepEqual(Tiers(), want) {
		t.Fatalf("Tiers() = %v, want %v", Tiers(), want)
	}
	for _, tr := range want {
		got, err := ParseTier(string(tr))
		if err != nil || got != tr {
			t.Fatalf("ParseTier(%q) = %q, %v", tr, got, err)
		}
	}
	for _, bad := range []string{"", "Frontier", "cheap", "host_decides", "opus"} {
		_, err := ParseTier(bad)
		if err == nil {
			t.Fatalf("ParseTier(%q) admitted", bad)
		}
		for _, tr := range want {
			if !strings.Contains(err.Error(), string(tr)) {
				t.Fatalf("refusal %q does not name the vocabulary member %q", err, tr)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// the bundled proposal
// ---------------------------------------------------------------------------

// rosterFromTree reads the shipped agent roster: every agents/*.md prompt other
// than the tree's own README and CHANGELOG, keyed on its frontmatter name.
func rosterFromTree(t *testing.T) []string {
	t.Helper()
	dir := filepath.Join("..", "..", "..", "agents")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading the agent roster: %v", err)
	}
	var names []string
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".md") || n == "README.md" || n == "CHANGELOG.md" {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			t.Fatal(err)
		}
		fields := frontmatter.Fields(strings.Split(string(body), "\n"))
		name := frontmatter.Unquote(fields["name"].Value)
		if name != strings.TrimSuffix(n, ".md") {
			t.Fatalf("%s declares name %q; the roster is keyed on the name and the file stem alike", n, name)
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// TestProposalNamesEveryAgentInTheRosterAndNoOther answers the intent's second
// open question at build time: an agent added without a row, or a row for an
// agent since removed, fails here rather than surfacing at run time.
func TestProposalNamesEveryAgentInTheRosterAndNoOther(t *testing.T) {
	roster := rosterFromTree(t)
	if !reflect.DeepEqual(Roster(), roster) {
		t.Fatalf("proposal roster %v\nagents/ roster  %v", Roster(), roster)
	}
}

// TestProposalSplitIsTheSpecs pins the initial rows the spec names.
func TestProposalSplitIsTheSpecs(t *testing.T) {
	frontier := []string{"intent-auditor", "lifeboat-reviewer", "release-changelog-composer", "ruthless-reviewer", "security-reviewer"}
	p := Proposal()
	for _, a := range Roster() {
		want := Economy
		for _, f := range frontier {
			if a == f {
				want = Frontier
			}
		}
		if p[a].Tier != want {
			t.Errorf("%s: proposed %q, want %q", a, p[a].Tier, want)
		}
		c, ok := Ceiling(a)
		if !ok || c < 1 || p[a].FanOut != c {
			t.Errorf("%s: fan-out %d, ceiling %d (%v): the proposal takes the contract ceiling", a, p[a].FanOut, c, ok)
		}
	}
	// Proposal hands out a copy: a caller mutating it changes nothing.
	p["scribe"] = Row{Tier: Local}
	if Proposal()["scribe"].Tier != Economy {
		t.Fatal("Proposal returned the package's own map")
	}
}

// ---------------------------------------------------------------------------
// store readers
// ---------------------------------------------------------------------------

// TestOrphanRowIsReportedAndTheRestApply is AC 9.
func TestOrphanRowIsReportedAndTheRestApply(t *testing.T) {
	f := newFx(t)
	f.repo(`{"ghost-agent":{"tier":"local"},"scribe":{"tier":"frontier"}}`)
	l := f.load()
	var named bool
	for _, d := range l.Diagnostics {
		if strings.Contains(d, `"ghost-agent"`) && strings.Contains(d, ".abcd/config/oracle-routing.json") {
			named = true
		}
	}
	if !named {
		t.Fatalf("diagnostics %q do not report the orphan row by name and file", l.Diagnostics)
	}
	r, err := Resolve("scribe", l, NoConnections{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Row.Tier != Frontier || r.Source != layered.Repo {
		t.Fatalf("the remaining row did not apply: %+v", r)
	}
}

// TestFanOutAboveTheCeilingIsClampedAndReported: a row may tighten the
// contract's ceiling, never raise it.
func TestFanOutAboveTheCeilingIsClampedAndReported(t *testing.T) {
	f := newFx(t)
	f.machine(`{"scribe":{"tier":"economy","fan_out":50}}`)
	l := f.load()
	ceiling, _ := Ceiling("scribe")
	r, err := Resolve("scribe", l, NoConnections{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Row.FanOut != ceiling {
		t.Fatalf("fan-out %d, want the ceiling %d", r.Row.FanOut, ceiling)
	}
	if len(l.Diagnostics) != 1 || !strings.Contains(l.Diagnostics[0], "fan_out 50") {
		t.Fatalf("diagnostics = %q, want one clamp report", l.Diagnostics)
	}
}

func TestMalformedRowsRefuseNamingTheFile(t *testing.T) {
	cases := []struct{ name, agents, want string }{
		{"tier outside the enum", `{"scribe":{"tier":"cheap"}}`, "host-decides"},
		{"tier absent", `{"scribe":{"fan_out":1}}`, "tier"},
		{"unknown row field", `{"scribe":{"tier":"local","model":"x"}}`, "model"},
		{"negative fan-out", `{"scribe":{"tier":"local","fan_out":-1}}`, "fan_out"},
		{"nested setting", `{"scribe":{"tier":"local","settings":{"temperature":{"a":1}}}}`, "temperature"},
		{"bad setting key", `{"scribe":{"tier":"local","settings":{"Temp Erature":1}}}`, "Temp Erature"},
		{"agents not an object", `[]`, "agents"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFx(t)
			f.repo(tc.agents)
			_, err := Load(f.roots)
			if err == nil {
				t.Fatal("Load admitted a malformed row")
			}
			for _, want := range []string{tc.want, ".abcd/config/oracle-routing.json"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("error %q does not name %q", err, want)
				}
			}
		})
	}
	t.Run("unknown top-level key", func(t *testing.T) {
		f := newFx(t)
		f.put(filepath.Join(f.roots.Home, ".abcd", "oracle-routing.json"), `{"schema_version":1,"agents":{},"backend":"x"}`)
		if _, err := Load(f.roots); err == nil || !strings.Contains(err.Error(), "backend") {
			t.Fatalf("err = %v, want a refusal naming backend", err)
		}
	})
}

// ---------------------------------------------------------------------------
// resolution
// ---------------------------------------------------------------------------

// TestNothingAcceptedRunsThroughTheHarnessAtHostDecides is AC 1: no table on
// the machine or in the repository applies nothing — not even the bundled
// proposal — and no connection is consulted.
func TestNothingAcceptedRunsThroughTheHarnessAtHostDecides(t *testing.T) {
	f := newFx(t)
	l := f.load()
	s := &spy{serves: map[Tier]Connection{Frontier: {Name: "cloud"}, HostDecides: {Name: "cloud"}}}
	if len(Roster()) == 0 {
		t.Fatal("the roster is empty, so nothing below would be checked")
	}
	for _, agent := range Roster() {
		r, err := Resolve(agent, l, s)
		if err != nil {
			t.Fatal(err)
		}
		ceiling, _ := Ceiling(agent)
		if r.Source != layered.None || r.Row.Tier != HostDecides || r.Row.FanOut != ceiling ||
			r.ConnectionUsed != Harness || r.ConnectionTried != "" || r.Fallback != "" || r.SettingsSent != nil {
			t.Fatalf("%s: %+v", agent, r)
		}
	}
	if s.calls != 0 {
		t.Fatalf("a connection was consulted %d times with nothing accepted", s.calls)
	}
}

// TestBundledAppliesOnceATableIsAccepted: an accepted table that lacks a row
// for an agent leaves that agent on the bundled proposal.
func TestBundledAppliesOnceATableIsAccepted(t *testing.T) {
	f := newFx(t)
	f.machine(`{"scribe":{"tier":"local"}}`)
	l := f.load()
	r, err := Resolve("intent-auditor", l, NoConnections{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Source != layered.Bundled || r.Row.Tier != Frontier || r.Origin != "bundled" {
		t.Fatalf("%+v", r)
	}
}

// TestPrecedenceAllFourLayersDisagree is AC 6's resolution half and AC 7's
// layer zero: flag over repo over machine over bundled, each step down taken
// only when the layer above holds no row.
func TestPrecedenceAllFourLayersDisagree(t *testing.T) {
	f := newFx(t)
	f.repo(`{"scribe":{"tier":"frontier"}}`)
	f.machine(`{"scribe":{"tier":"local"},"sota-researcher":{"tier":"frontier"}}`)
	l := f.load()

	check := func(agent string, tier Tier, src layered.Layer) {
		t.Helper()
		r, err := Resolve(agent, l, NoConnections{})
		if err != nil {
			t.Fatal(err)
		}
		if r.Row.Tier != tier || r.Source != src {
			t.Fatalf("%s: tier %q from %s, want %q from %s", agent, r.Row.Tier, r.Source, tier, src)
		}
	}
	check("scribe", Frontier, layered.Repo)
	check("sota-researcher", Frontier, layered.Machine)
	check("docs-currency-reviewer", Economy, layered.Bundled)

	routes, err := ParseRoutes([]string{"scribe=host-decides"}, []string{"scribe"}, NoConnections{})
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Apply(routes); err != nil {
		t.Fatal(err)
	}
	check("scribe", HostDecides, layered.Flag)

	rows, err := l.Rows("scribe")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, lr := range rows {
		got = append(got, lr.Layer.String()+"="+string(lr.Row.Tier))
	}
	want := []string{"flag=host-decides", "repo=frontier", "machine=local", "bundled=economy"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Rows = %v, want %v (winner first)", got, want)
	}
}

// TestAProviderThatServesTheTierTakesTheStep is AC 3.
func TestAProviderThatServesTheTierTakesTheStep(t *testing.T) {
	f := newFx(t)
	f.machine(`{"scribe":{"tier":"local"}}`)
	l := f.load()
	s := &spy{serves: map[Tier]Connection{Local: {Name: "ollama-desk"}}}
	r, err := Resolve("scribe", l, s)
	if err != nil {
		t.Fatal(err)
	}
	if r.ConnectionUsed != "ollama-desk" || r.ConnectionTried != "ollama-desk" || r.Fallback != "" {
		t.Fatalf("%+v", r)
	}
}

// TestNoProviderFallsBackToTheHarnessWithAReason is AC 4's resolution half.
func TestNoProviderFallsBackToTheHarnessWithAReason(t *testing.T) {
	f := newFx(t)
	f.machine(`{"scribe":{"tier":"local","fan_out":1}}`)
	l := f.load()
	r, err := Resolve("scribe", l, NoConnections{})
	if err != nil {
		t.Fatal(err)
	}
	if r.ConnectionUsed != Harness || r.ConnectionTried != "" || r.Row.Tier != Local || r.Row.FanOut != 1 {
		t.Fatalf("%+v", r)
	}
	if !strings.Contains(r.Fallback, "local") || !strings.Contains(r.Fallback, "harness") {
		t.Fatalf("fallback reason %q does not name the tier and the harness", r.Fallback)
	}
	// host-decides is the harness by definition, not a fallback.
	f2 := newFx(t)
	f2.machine(`{"scribe":{"tier":"host-decides"}}`)
	r2, err := Resolve("scribe", f2.load(), NoConnections{})
	if err != nil || r2.Fallback != "" || r2.ConnectionUsed != Harness {
		t.Fatalf("%+v, %v", r2, err)
	}
}

// TestSettingsMergeConnectionThenRowThenFlag is AC 8's merge.
func TestSettingsMergeConnectionThenRowThenFlag(t *testing.T) {
	f := newFx(t)
	f.repo(`{"scribe":{"tier":"economy","settings":{"temperature":0.2,"top_p":0.9}}}`)
	l := f.load()
	conn := Connection{Name: "openrouter", Defaults: Settings{"temperature": raw("1"), "seed": raw("7"), "top_p": raw("1")}}
	s := &spy{serves: map[Tier]Connection{Economy: conn}, named: map[string]Connection{"openrouter": conn}}
	routes, err := ParseRoutes([]string{"scribe=economy?seed=42"}, []string{"scribe"}, s)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Apply(routes); err != nil {
		t.Fatal(err)
	}
	r, err := Resolve("scribe", l, s)
	if err != nil {
		t.Fatal(err)
	}
	want := Settings{"temperature": raw("0.2"), "top_p": raw("0.9"), "seed": raw("42")}
	if !reflect.DeepEqual(r.SettingsSent, want) {
		t.Fatalf("settings sent %s, want %s", dump(r.SettingsSent), dump(want))
	}
	if r.Override != "scribe=economy?seed=42" {
		t.Fatalf("override %q, want the flag verbatim", r.Override)
	}
}

func dump(s Settings) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// ---------------------------------------------------------------------------
// the invocation override
// ---------------------------------------------------------------------------

// TestRouteFlagForms is AC 7: the flag's row wins, a second --route for
// another agent applies alongside it, and the connection named is used.
func TestRouteFlagForms(t *testing.T) {
	f := newFx(t)
	f.repo(`{"scribe":{"tier":"economy"},"intent-auditor":{"tier":"economy"}}`)
	l := f.load()
	conn := Connection{Name: "desk"}
	s := &spy{named: map[string]Connection{"desk": conn}}
	routes, err := ParseRoutes(
		[]string{"scribe=local@desk", "intent-auditor=frontier"},
		[]string{"scribe", "intent-auditor"}, s)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Apply(routes); err != nil {
		t.Fatal(err)
	}
	r1, err := Resolve("scribe", l, s)
	if err != nil {
		t.Fatal(err)
	}
	if r1.Row.Tier != Local || r1.ConnectionUsed != "desk" || r1.Source != layered.Flag || r1.Override != "scribe=local@desk" {
		t.Fatalf("%+v", r1)
	}
	r2, err := Resolve("intent-auditor", l, s)
	if err != nil {
		t.Fatal(err)
	}
	if r2.Row.Tier != Frontier || r2.Source != layered.Flag || r2.Override != "intent-auditor=frontier" {
		t.Fatalf("%+v", r2)
	}
}

func TestRouteFlagRefusals(t *testing.T) {
	s := &spy{named: map[string]Connection{"desk": {Name: "desk"}}}
	dispatched := []string{"scribe", "intent-auditor"}
	cases := []struct{ name, route, want string }{
		{"agent the invocation does not dispatch", "security-reviewer=frontier", `this invocation dispatches scribe, intent-auditor, not "security-reviewer"`},
		{"agent not in the roster", "ghost=frontier", `this invocation dispatches scribe, intent-auditor, not "ghost"`},
		{"tier outside the enum", "scribe=cheap", "host-decides"},
		{"connection not configured", "scribe=local@elsewhere", "elsewhere"},
		{"no tier", "scribe", "<agent>=<tier>"},
		{"empty connection", "scribe=local@", "connection"},
		{"bad setting", "scribe=local?temperature", "k=v"},
		{"bad setting key", "scribe=local?Temp=1", "Temp"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseRoutes([]string{tc.route}, dispatched, s)
			if err == nil {
				t.Fatalf("--route %s admitted", tc.route)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not name %q", err, tc.want)
			}
		})
	}
	t.Run("the same agent twice", func(t *testing.T) {
		_, err := ParseRoutes([]string{"scribe=local", "scribe=frontier"}, dispatched, s)
		if err == nil || !strings.Contains(err.Error(), "more than once") {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestRouteSettingValuesKeepTheirJSONType(t *testing.T) {
	routes, err := ParseRoutes([]string{"scribe=economy?seed=42,temperature=0.5,stream=true,stop=END"},
		[]string{"scribe"}, NoConnections{})
	if err != nil {
		t.Fatal(err)
	}
	want := Settings{"seed": raw("42"), "temperature": raw("0.5"), "stream": raw("true"), "stop": raw(`"END"`)}
	if !reflect.DeepEqual(routes[0].Row.Settings, want) {
		t.Fatalf("settings %s, want %s", dump(routes[0].Row.Settings), dump(want))
	}
}

// TestBoardMarksTheWinnerPerAgent is the board's core half of AC 6: one row
// per agent in the roster, every layer that holds a row listed highest first,
// and the winner named. Nothing accepted gives no rows.
func TestBoardMarksTheWinnerPerAgent(t *testing.T) {
	empty := newFx(t).load()
	if rows, err := empty.Board(); err != nil || rows != nil {
		t.Fatalf("nothing accepted: rows %v, %v", rows, err)
	}
	f := newFx(t)
	f.repo(`{"scribe":{"tier":"frontier"}}`)
	f.machine(`{"scribe":{"tier":"local"}}`)
	rows, err := f.load().Board()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != len(Roster()) {
		t.Fatalf("%d rows, want one per agent (%d)", len(rows), len(Roster()))
	}
	for _, r := range rows {
		switch r.Agent {
		case "scribe":
			if r.Winner != "repo" || len(r.Layers) != 3 || r.Layers[1].Tier != Local || r.Layers[2].Layer != "bundled" {
				t.Fatalf("scribe = %+v", r)
			}
		case "intent-auditor":
			if r.Winner != "bundled" || len(r.Layers) != 1 || r.Layers[0].Tier != Frontier {
				t.Fatalf("intent-auditor = %+v", r)
			}
		}
	}
}
