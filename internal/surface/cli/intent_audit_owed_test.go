package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// intent_audit_owed_test.go is ruling DQ1c's wiring proof: through the front
// door, an after-merge audit that comes back undecided leaves the intent
// shipped and flagged, says which issue carries the owed check, and `intent
// audit <itd-N>` rewrites the request for the re-run the flag asks for.
func TestIntentAuditIngestFlagsAnUndecidedAuditAndCapturesItsCheck(t *testing.T) {
	root, vp := conditionedRepo(t)
	raw, err := os.ReadFile(vp)
	if err != nil {
		t.Fatal(err)
	}
	undecided := writeVerdict(t, strings.NewReplacer(
		`"verdict": "MET"`, `"verdict": "INCONCLUSIVE"`,
		`{"MET": 1, "MET_WITH_CONCERNS": 0, "NOT_MET": 0, "INCONCLUSIVE": 0}`,
		`{"MET": 0, "MET_WITH_CONCERNS": 0, "NOT_MET": 0, "INCONCLUSIVE": 1}`).Replace(string(raw)))
	text := string(runCLI(t, "intent", "audit", "ingest", "--verdict-json", undecided))
	m := regexp.MustCompile(`captured (iss-[0-9]+) to carry the check`).FindStringSubmatch(text)
	if !strings.Contains(text, "audit owed: ac-1 INCONCLUSIVE") || m == nil {
		t.Fatalf("the ingest must name the owed criteria and the issue it captured:\n%s", text)
	}
	body, err := os.ReadFile(filepath.Join(root, ".abcd", "development", "intents", "shipped", "itd-10-alpha.md"))
	if err != nil {
		t.Fatalf("the intent must stay shipped: %v", err)
	}
	if !strings.Contains(string(body), "Remedy: re-run the audit. Carried by "+m[1]+".") {
		t.Fatalf("the Audit Notes must carry the flag naming %s:\n%s", m[1], body)
	}
	if text := string(runCLI(t, "intent", "audit", "itd-10")); !strings.Contains(text, "check_owed") ||
		!strings.Contains(text, "request rewritten for the re-run") {
		t.Fatalf("a flagged receipt's re-emit must rewrite the request for the re-run:\n%s", text)
	}
}
