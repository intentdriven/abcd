package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lifeboat"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/spf13/cobra"
)

// routeWorld isolates a route test: a fresh home (with the machine routing
// table it is given, "" for none) and a fresh checkout as the working
// directory (with the repository table it is given), so no developer's own
// ~/.abcd/oracle-routing.json reaches a verb under test.
func routeWorld(t *testing.T, repoTable, machineTable string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir()
	gitInitAt(t, repo)
	t.Chdir(repo)
	put := func(p, body string) {
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if repoTable != "" {
		put(filepath.Join(repo, ".abcd", "config", "oracle-routing.json"), repoTable)
	}
	if machineTable != "" {
		put(filepath.Join(home, ".abcd", "oracle-routing.json"), machineTable)
	}
	return repo
}

// member decodes one top-level member of a verb's --json output.
func member(t *testing.T, out []byte, key string, into any) {
	t.Helper()
	var env map[string]json.RawMessage
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("not a JSON object: %v\n%s", err, out)
	}
	raw, ok := env[key]
	if !ok {
		t.Fatalf("the output carries no %q member:\n%s", key, out)
	}
	if err := json.Unmarshal(raw, into); err != nil {
		t.Fatalf("%s: %v", key, err)
	}
}

// TestEveryDelegatingVerbCarriesRoute holds the command tree to the delegating
// verbs (spc-2609180535002478 step 3): the flag is on each of them and on
// nothing else.
func TestEveryDelegatingVerbCarriesRoute(t *testing.T) {
	want := []string{
		"abcd disembark graveyard",
		"abcd disembark press-release",
		"abcd disembark principles",
		"abcd disembark review",
		"abcd intent audit",
		"abcd intent audit ingest",
		"abcd launch ship",
		"abcd reading ingest",
	}
	var got []string
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if c.Flags().Lookup(routeFlagName) != nil {
			got = append(got, c.CommandPath())
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(NewRootCommand())
	sort.Strings(got)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("--route is on:\n  %s\nwant:\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
}

func principlesPayload(t *testing.T) string {
	return synthPayloadFile(t, `{"schema_version":2,"mode":"delegated","prompt_version":"0.2.0",`+
		`"principles":[{"id":"prn-cascade","principle":"the cascade is fixed","confidence":"high",`+
		`"claim_type":"causal","reference":"adr-12","comparison":null,"evidence":["adr-12"]}]}`)
}

// TestRouteRefusalsExitTwoBeforeAnyStep is AC 7's refusal half at the surface:
// an agent this invocation does not dispatch, a tier outside the vocabulary, a
// connection not configured on this machine, a malformed routing table and a
// --route on a deterministic mode each exit 2, and nothing is written.
func TestRouteRefusalsExitTwoBeforeAnyStep(t *testing.T) {
	routeWorld(t, "", "")
	cases := []struct {
		name   string
		args   []string
		want   string
		broken string
	}{
		{"agent not dispatched", []string{"--route", "scribe=economy"}, `this invocation dispatches principle-distiller, not "scribe"`, ""},
		{"tier outside the vocabulary", []string{"--route", "principle-distiller=cheap"}, `tier "cheap" is not one of`, ""},
		{"connection not configured", []string{"--route", "principle-distiller=local@lab"}, `connection "lab" is not configured`, ""},
		{"malformed routing table", nil, "schema_version", `{"schema_version":2,"agents":{}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			routeWorld(t, tc.broken, "")
			dir := buildSynthLifeboat(t, synthOpts{})
			args := append([]string{"disembark", "principles", dir, "--principles-json", principlesPayload(t)}, tc.args...)
			stdout, stderr, err := runCLISplit(t, args...)
			if code := exitCodeOf(err); code != 2 {
				t.Fatalf("exit %d, want 2 (err %v)\nstdout %s\nstderr %s", code, err, stdout, stderr)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("refusal %q does not name %q", err, tc.want)
			}
			if _, statErr := os.Stat(filepath.Join(dir, "principles.json")); statErr == nil {
				t.Fatal("principles.json was written by a refused invocation")
			}
		})
	}

	t.Run("deterministic mode dispatches no agent", func(t *testing.T) {
		routeWorld(t, "", "")
		dir := buildSynthLifeboat(t, synthOpts{})
		_, _, err := runCLISplit(t, "disembark", "principles", dir, "--route", "principle-distiller=economy")
		if exitCodeOf(err) != 2 || !strings.Contains(err.Error(), "dispatches none") {
			t.Fatalf("err %v", err)
		}
		if _, statErr := os.Stat(filepath.Join(dir, "principles.json")); statErr == nil {
			t.Fatal("principles.json was written by a refused invocation")
		}
	})
}

// TestNothingAcceptedReceiptIsHostDecidesOnTheHarness is AC 1 at an ingest:
// no table and no flag, so the receipt records host-decides on the harness, no
// fallback, and nothing is printed on stderr.
func TestNothingAcceptedReceiptIsHostDecidesOnTheHarness(t *testing.T) {
	routeWorld(t, "", "")
	dir := buildSynthLifeboat(t, synthOpts{})
	stdout, stderr, err := runCLISplit(t, "disembark", "principles", dir, "--principles-json", principlesPayload(t), "--json")
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr carries %q with nothing accepted", stderr)
	}
	var rc oracle.ReceiptRoute
	member(t, []byte(stdout), "route", &rc)
	want := oracle.ReceiptRoute{Agent: "principle-distiller", TierAsked: oracle.HostDecides,
		ConnectionUsed: oracle.Harness, SettingsSent: oracle.Settings{}}
	if !receiptEqual(rc, want) {
		t.Fatalf("route = %+v, want %+v", rc, want)
	}
	var res lifeboat.PrinciplesResult
	if err := json.Unmarshal([]byte(stdout), &res); err != nil || res.Written != 1 {
		t.Fatalf("the verb's own result is not intact: %+v %v", res, err)
	}
}

func receiptEqual(a, b oracle.ReceiptRoute) bool {
	ea, _ := json.Marshal(a)
	eb, _ := json.Marshal(b)
	return string(ea) == string(eb)
}

// TestDisembarkIngestsCarryTheReceipt is step 4's golden receipt for each of
// the four disembark ingests, with an accepted repository row that no
// provider serves: the stderr line names the fallback before the verb writes
// (AC 4), and the receipt carries the tier asked, the connections tried and
// used, the reason, and the override verbatim (AC 4, 5, 7).
func TestDisembarkIngestsCarryTheReceipt(t *testing.T) {
	type verb struct {
		agent string
		args  func(t *testing.T) []string
	}
	verbs := []verb{
		{"principle-distiller", func(t *testing.T) []string {
			return []string{"disembark", "principles", buildSynthLifeboat(t, synthOpts{}), "--principles-json", principlesPayload(t)}
		}},
		{"press-release-composer", func(t *testing.T) []string {
			return []string{"disembark", "press-release", buildSynthLifeboat(t, synthOpts{briefPR: true}), "--press-release-json",
				synthPayloadFile(t, `{"schema_version":1,"mode":"delegated","prompt_version":"0.1.0",`+
					`"headline":"H","body":"B","evidence":["brief/01-product/01-press-release.md"]}`)}
		}},
		{"lifeboat-reviewer", func(t *testing.T) []string {
			return []string{"disembark", "review",
				buildSynthLifeboat(t, synthOpts{sourceName: "src", coverage: &lifeboat.Summary{Grounded: 7, Blank: 3}}),
				realSrcDir(t, "src"), "--review-json",
				synthPayloadFile(t, `{"schema_version":1,"mode":"delegated","prompt_version":"0.1.0","verdict":"SHIP","findings":[]}`)}
		}},
		{"graveyard-interpreter", func(t *testing.T) []string {
			dir := buildGraveyardLifeboat(t)
			return []string{"disembark", "graveyard", dir, "--lessons-json", lessonsPayloadFile(t, t.TempDir(),
				lifeboat.Lesson{ID: "les-engine-v1", Lesson: "engine retired", Confidence: lifeboat.ConfidenceHigh,
					Evidence: []string{"rev-9f3a1c2d4e5b"}})}
		}},
	}
	for _, v := range verbs {
		t.Run(v.agent, func(t *testing.T) {
			routeWorld(t, `{"schema_version":1,"agents":{"`+v.agent+`":{"tier":"local"}}}`, "")
			args := append(v.args(t), "--json")

			stdout, stderr, err := runCLISplit(t, args...)
			if err != nil {
				t.Fatalf("%v\n%s", err, stderr)
			}
			if n := strings.Count(stderr, "\n"); n != 1 || !strings.Contains(stderr, "serves tier local") {
				t.Fatalf("want one fallback line on stderr, got %q", stderr)
			}
			var rc oracle.ReceiptRoute
			member(t, []byte(stdout), "route", &rc)
			want := oracle.ReceiptRoute{Agent: v.agent, TierAsked: oracle.Local, ConnectionUsed: oracle.Harness,
				FallbackReason: rc.FallbackReason, SettingsSent: oracle.Settings{}}
			if rc.FallbackReason == "" || !receiptEqual(rc, want) {
				t.Fatalf("route = %+v", rc)
			}

			override := v.agent + "=frontier?seed=7"
			stdout, stderr, err = runCLISplit(t, append(v.args(t), "--json", "--route", override)...)
			if err != nil {
				t.Fatalf("%v\n%s", err, stderr)
			}
			member(t, []byte(stdout), "route", &rc)
			if rc.TierAsked != oracle.Frontier || rc.Override != override {
				t.Fatalf("with --route the receipt is %+v", rc)
			}

			text, _, err := runCLISplit(t, v.args(t)...)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(text, "route: "+v.agent+" asked tier local, used harness, model reported not reported") {
				t.Fatalf("the human rendering carries no route line:\n%s", text)
			}
		})
	}
}

// TestWithMemberRefusesADuplicateKey: the joined member is added after the
// verb's own, so a result that already carries the key would yield an object
// with the key twice, which most readers resolve last-wins, silently. The join
// refuses instead, and joins cleanly where the key is absent.
func TestWithMemberRefusesADuplicateKey(t *testing.T) {
	type result struct {
		Route  string `json:"route"`
		Status string `json:"status"`
	}
	if _, err := json.Marshal(withMember{v: result{Route: "own", Status: "ok"}, key: "route", val: 1}); err == nil ||
		!strings.Contains(err.Error(), `already carries a "route" member`) {
		t.Fatalf("a duplicate route member was joined: err %v", err)
	}
	out, err := json.Marshal(withMember{v: result{Route: "own", Status: "ok"}, key: "routing", val: 1})
	if err != nil || string(out) != `{"route":"own","status":"ok","routing":1}` {
		t.Fatalf("out %s err %v", out, err)
	}
	if out, err := json.Marshal(withMember{v: struct{}{}, key: "route", val: 1}); err != nil || string(out) != `{"route":1}` {
		t.Fatalf("an empty base: out %s err %v", out, err)
	}
}
