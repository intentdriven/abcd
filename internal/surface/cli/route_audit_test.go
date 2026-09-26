package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/oracle"
)

// acceptRepoRow writes an accepted repository routing table into the working
// checkout, one row.
func acceptRepoRow(t *testing.T, root, agent, tier string) {
	t.Helper()
	p := filepath.Join(root, ".abcd", "config", "oracle-routing.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	body := `{"schema_version":1,"agents":{"` + agent + `":{"tier":"` + tier + `"}}}`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestIntentAuditRequestCarriesTheRoutingSection is step 4's golden request
// block for `intent audit`: the emit's result carries the routing member, the
// request document the host reads carries the routing section (outside the
// hashed prompt, so the verdict's prompt_hash is unchanged), and a fallback is
// announced on stderr before the step runs.
func TestIntentAuditRequestCarriesTheRoutingSection(t *testing.T) {
	root := intentTestRepo(t)
	writeRepoFile(t, root, ".abcd/development/intents/shipped/itd-10-alpha.md", conditionedIntent)

	// Nothing accepted: host-decides at the contract ceiling, on the harness.
	stdout, stderr, err := runCLISplit(t, "intent", "audit", "itd-10", "--json")
	if err != nil || stderr != "" {
		t.Fatalf("err %v stderr %q", err, stderr)
	}
	var rr oracle.RequestRouting
	member(t, []byte(stdout), "routing", &rr)
	if rr != (oracle.RequestRouting{Agent: "intent-auditor", Tier: oracle.HostDecides, FanOut: 1, Source: "none", Origin: "none", Connection: "harness"}) {
		t.Fatalf("routing = %+v", rr)
	}
	var emitted struct {
		RequestPath string `json:"request_path"`
	}
	member(t, []byte(`{"x":`+stdout+`}`), "x", &emitted)
	doc, err := os.ReadFile(filepath.Join(root, emitted.RequestPath))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(doc), "## Routing\n\nrouting:\n  agent: intent-auditor\n  tier: host-decides\n  fan_out: 1\n") {
		t.Fatalf("the request document carries no routing section:\n%s", doc)
	}

	// An accepted row no provider serves: the fallback line, then the request.
	acceptRepoRow(t, root, "intent-auditor", "frontier")
	stdout, stderr, err = runCLISplit(t, "intent", "audit", "itd-10", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr, "serves tier frontier") || strings.Count(stderr, "\n") != 1 {
		t.Fatalf("stderr %q", stderr)
	}
	member(t, []byte(stdout), "routing", &rr)
	if rr.Tier != oracle.Frontier || rr.Source != "repo" || rr.Fallback == "" {
		t.Fatalf("routing = %+v", rr)
	}
	doc, _ = os.ReadFile(filepath.Join(root, emitted.RequestPath))
	if !strings.Contains(string(doc), "  tier: frontier\n") || !strings.Contains(string(doc), "  fallback: ") {
		t.Fatalf("the rewritten request carries the old routing:\n%s", doc)
	}

	text, _, err := runCLISplit(t, "intent", "audit", "itd-10", "--route", "intent-auditor=economy")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "routing: intent-auditor at tier economy, fan-out 1, decided by flag (intent-auditor=economy), via harness, override intent-auditor=economy") {
		t.Fatalf("the human rendering carries no routing line:\n%s", text)
	}
}

// TestIntentAuditIngestCarriesTheReceipt is step 4's golden receipt for
// `intent audit ingest`, with the override verbatim (AC 7).
func TestIntentAuditIngestCarriesTheReceipt(t *testing.T) {
	root, vp := conditionedRepo(t)
	acceptRepoRow(t, root, "intent-auditor", "frontier")
	stdout, stderr, err := runCLISplit(t, "intent", "audit", "ingest", "--verdict-json", vp, "--json",
		"--route", "intent-auditor=economy")
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	var rc oracle.ReceiptRoute
	member(t, []byte(stdout), "route", &rc)
	want := oracle.ReceiptRoute{Agent: "intent-auditor", TierAsked: oracle.Economy, ConnectionUsed: oracle.Harness,
		FallbackReason: rc.FallbackReason, Override: "intent-auditor=economy", SettingsSent: oracle.Settings{}}
	if rc.FallbackReason == "" || !receiptEqual(rc, want) {
		t.Fatalf("route = %+v", rc)
	}
	if !strings.Contains(stderr, "serves tier economy") {
		t.Fatalf("stderr %q", stderr)
	}
}

// TestIntentAuditIssueDriftRefusesRoute: the drift check dispatches no agent.
func TestIntentAuditIssueDriftRefusesRoute(t *testing.T) {
	intentTestRepo(t)
	_, _, err := runCLISplit(t, "intent", "audit", "--issue-drift", "--route", "intent-auditor=economy")
	if exitCodeOf(err) != 2 || !strings.Contains(err.Error(), "dispatches none") {
		t.Fatalf("err %v", err)
	}
}

// specCloseWorld is a checkout holding one planned intent and the one open
// spec that realises it, so `spec close spc-1` ships the intent and emits its
// fidelity-review request.
func specCloseWorld(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	repo := t.TempDir()
	gitInitAt(t, repo)
	t.Chdir(repo)
	writeRepoFile(t, repo, cliPlanned+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\nimpact: fix\n---\n# alpha\n\n## Acceptance Criteria\n\n- ok\n")
	writeRepoFile(t, repo, cliSpecsOpen+"/spc-1-alpha.md",
		"---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# alpha\n")
	return repo
}

// closeRequest runs `spec close spc-1 --json` and returns the request document
// the close emitted, with the close's stderr.
func closeRequest(t *testing.T, repo string) (doc, stderr string) {
	t.Helper()
	stdout, stderr, err := runCLISplit(t, "spec", "close", "spc-1", "--json")
	if err != nil {
		t.Fatalf("%v\nstderr %s", err, stderr)
	}
	var res struct {
		ReceiptID string `json:"receipt_id"`
		To        string `json:"to"`
	}
	member(t, []byte(`{"x":`+stdout+`}`), "x", &res)
	if res.To != "shipped" || res.ReceiptID == "" {
		t.Fatalf("the close shipped nothing: %s", stdout)
	}
	raw, err := os.ReadFile(filepath.Join(repo, ".abcd", ".work.local", "reviews", res.ReceiptID+".request.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw), stderr
}

// TestSpecCloseRequestCarriesTheRoutingSection: the close that ships an intent
// emits the auditor's request, and that request carries the same `## Routing`
// section `intent audit <itd-N>` writes, so a host that reads the close's
// request directly runs the auditor at the resolved tier.
func TestSpecCloseRequestCarriesTheRoutingSection(t *testing.T) {
	repo := specCloseWorld(t)
	acceptRepoRow(t, repo, "intent-auditor", "economy")
	doc, stderr := closeRequest(t, repo)
	if !strings.Contains(doc, "## Routing\n\nrouting:\n  agent: intent-auditor\n  tier: economy\n") {
		t.Fatalf("the close's request carries no routing section:\n%s", doc)
	}
	if !strings.Contains(stderr, "serves tier economy") {
		t.Fatalf("the fallback is not announced: stderr %q", stderr)
	}
}

// TestSpecCloseUnreadableRoutingTableStillCloses: the close is a record move and
// its emit is report-only, so a routing table that cannot be read leaves the
// request without a routing section and says so on stderr, naming the re-emit,
// rather than refusing a close whose intent has already shipped.
func TestSpecCloseUnreadableRoutingTableStillCloses(t *testing.T) {
	repo := specCloseWorld(t)
	writeRepoFile(t, repo, ".abcd/config/oracle-routing.json", `{"schema_version":2,"agents":{}}`)
	doc, stderr := closeRequest(t, repo)
	if strings.Contains(doc, "## Routing") {
		t.Fatalf("a routing section was written from a table that cannot be read:\n%s", doc)
	}
	if !strings.Contains(stderr, "schema_version") || !strings.Contains(stderr, "abcd intent audit itd-10") {
		t.Fatalf("stderr %q", stderr)
	}
}
