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
