package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// intent_audit_conditions_test.go is spc-59's wiring proof: the scope-condition
// disposition surface is reachable through the front door
// (`abcd intent audit ingest --verdict-json`) and the per-disposition split it
// returns is rendered, not merely computed.

const auditConditionID = "cond-2608300000000009"

// conditionedIntent is a shipped intent carrying one identified scope condition.
const conditionedIntent = "---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n---\n" +
	"# alpha\n\n## Scope Conditions\n\n" +
	"- holds below 10k records <!-- cond: " + auditConditionID + " -->\n\n" +
	"## Acceptance Criteria\n\n- the ship move parks an OWED stub\n\n## Audit Notes\n"

// conditionedVerdict is a schema-valid verdict disposing the one condition.
func conditionedVerdict(receiptID string) string {
	return `{
  "_type": "abcd/intent-fidelity-verdict/v1",
  "receipt_id": "` + receiptID + `",
  "verifier": {"id": "intent-auditor", "version": "test"},
  "policy": {"rubric_hash": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "prompt_hash": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
  "input_attestations": [],
  "criteria": [
    {"criterion_id": "ac-1", "verdict": "MET", "rationale": "the stub is parked",
     "evidence": [{"ref": "internal/core/intent/audit.go:230", "quote": "owedBlock"}]}
  ],
  "acceptance_rollup": {"MET": 1, "MET_WITH_CONCERNS": 0, "NOT_MET": 0, "INCONCLUSIVE": 0},
  "gap_audit": {"honoured": [], "diverged": [], "missing": []},
  "scope_conditions": [
    {"condition_id": "` + auditConditionID + `", "disposition": "narrowed",
     "rationale": "the delivered index is bounded tighter than the design assumed",
     "narrowing": "holds below 2k records, not 10k",
     "evidence": [{"ref": "internal/core/intent/audit.go:400", "quote": "validateVerdict"}]}
  ]
}`
}

// conditionedRepo stages a shipped intent carrying one identified scope
// condition, chdirs into it, emits its audit receipt through the CLI, and
// returns the repo root with the staged verdict's path.
func conditionedRepo(t *testing.T) (root, verdictPath string) {
	t.Helper()
	root = intentTestRepo(t)
	shipped := filepath.Join(root, ".abcd", "development", "intents", "shipped")
	if err := os.MkdirAll(shipped, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shipped, "itd-10-alpha.md"), []byte(conditionedIntent), 0o644); err != nil {
		t.Fatal(err)
	}
	var emitted struct {
		ReceiptID   string `json:"receipt_id"`
		RequestPath string `json:"request_path"`
	}
	if err := json.Unmarshal(runCLI(t, "intent", "audit", "itd-10", "--json"), &emitted); err != nil {
		t.Fatalf("intent audit output not JSON: %v", err)
	}
	if emitted.ReceiptID == "" {
		t.Fatal("intent audit emitted no receipt id")
	}
	payload := echoRequestPolicy(t, root, emitted.RequestPath, conditionedVerdict(emitted.ReceiptID))
	return root, writeVerdict(t, payload)
}

// echoRequestPolicy takes the two policy hashes out of the request the front door
// just emitted and puts them in the verdict, which is the whole loop
// iss-2609100505140261 was missing: the host issues the provenance, the auditor
// echoes it, and the ingest recomputes and checks it. Reading them from the
// request is how a real auditor gets them, so this doubles as the wiring proof
// that `intent audit` states them at all.
func echoRequestPolicy(t *testing.T, root, requestRel, payload string) string {
	t.Helper()
	if requestRel == "" {
		t.Fatal("intent audit reported no request path, so the auditor has no provenance to echo")
	}
	rb, err := os.ReadFile(filepath.Join(root, requestRel))
	if err != nil {
		t.Fatalf("reading the emitted request %s: %v", requestRel, err)
	}
	var n int
	for _, ln := range strings.Split(string(rb), "\n") {
		for _, f := range []struct{ prefix, placeholder string }{
			{"- rubric_hash: ", "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
			{"- prompt_hash: ", "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		} {
			if v, ok := strings.CutPrefix(ln, f.prefix); ok {
				payload = strings.Replace(payload, f.placeholder, strings.TrimSpace(v), 1)
				n++
			}
		}
	}
	if n != 2 {
		t.Fatalf("the emitted request states %d of the 2 policy hashes an auditor must echo:\n%s", n, rb)
	}
	return payload
}

// TestIntentAuditIngestReportsTheDispositionSplit is the machine surface: the
// ingest result carries the per-disposition split, and the record carries the
// keyed disposition with its stated narrowing.
func TestIntentAuditIngestReportsTheDispositionSplit(t *testing.T) {
	root, vp := conditionedRepo(t)

	var res struct {
		Status     string `json:"status"`
		Conditions int    `json:"conditions"`
		Narrowed   int    `json:"narrowed"`
		Survived   int    `json:"survived"`
	}
	if err := json.Unmarshal(runCLI(t, "intent", "audit", "ingest", "--verdict-json", vp, "--json"), &res); err != nil {
		t.Fatalf("ingest output not JSON: %v", err)
	}
	if res.Status != "ingested" || res.Conditions != 1 || res.Narrowed != 1 || res.Survived != 0 {
		t.Fatalf("ingest result = %+v, want one narrowed condition ingested", res)
	}

	body, err := os.ReadFile(filepath.Join(root, ".abcd", "development", "intents", "shipped", "itd-10-alpha.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), auditConditionID+" — narrowed") ||
		!strings.Contains(string(body), "narrowing: holds below 2k records, not 10k") {
		t.Fatalf("the record must carry the keyed disposition and its stated narrowing:\n%s", body)
	}
}

// TestIntentAuditIngestRendersTheDispositionSplit is the human surface: the
// split is reported in the text render too, not only in --json.
func TestIntentAuditIngestRendersTheDispositionSplit(t *testing.T) {
	_, vp := conditionedRepo(t)
	text := string(runCLI(t, "intent", "audit", "ingest", "--verdict-json", vp))
	if !strings.Contains(text, "scope conditions 1: survived 0 · narrowed 1 · falsified 0 · untested 0") {
		t.Fatalf("the human render must report the disposition split:\n%s", text)
	}
}

// TestIntentAuditReingestReportsTheReplacement is the front door for a re-ingest
// for the same receipt: a payload that renders differently replaces the ingested
// verdict and says so, and the identical payload again is a noop.
func TestIntentAuditReingestReportsTheReplacement(t *testing.T) {
	root, vp := conditionedRepo(t)
	runCLI(t, "intent", "audit", "ingest", "--verdict-json", vp)
	raw, err := os.ReadFile(vp)
	if err != nil {
		t.Fatal(err)
	}
	changed := writeVerdict(t, strings.Replace(string(raw), "bounded tighter than the design assumed",
		"bounded tighter than the design assumed, weighed again", 1))
	text := string(runCLI(t, "intent", "audit", "ingest", "--verdict-json", changed))
	if !strings.Contains(text, "— ingested") || !strings.Contains(text, "replaced the verdict already ingested") {
		t.Fatalf("re-ingest render does not report the replacement:\n%s", text)
	}
	body, err := os.ReadFile(filepath.Join(root, ".abcd", "development", "intents", "shipped", "itd-10-alpha.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(body), "abcd-review: INGESTED") != 1 || !strings.Contains(string(body), "weighed again") {
		t.Fatalf("the record must carry the one replaced verdict:\n%s", body)
	}
	if text := string(runCLI(t, "intent", "audit", "ingest", "--verdict-json", changed)); !strings.Contains(text, "— noop") {
		t.Fatalf("an identical re-ingest must be a noop:\n%s", text)
	}
}
