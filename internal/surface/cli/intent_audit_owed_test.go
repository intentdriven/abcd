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
	if text := string(runCLI(t, "intent", "audit")); !strings.Contains(text, "audit owed (reviewed, flagged; not counted as owed):") ||
		!strings.Contains(text, "itd-10") || !strings.Contains(text, "carried by "+m[1]) {
		t.Fatalf("the owed-review listing must name the flagged receipt and its issue:\n%s", text)
	}
	if text := string(runCLI(t, "intent", "audit", "itd-10")); !strings.Contains(text, "check_owed") ||
		!strings.Contains(text, "request rewritten for the re-run") {
		t.Fatalf("a flagged receipt's re-emit must rewrite the request for the re-run:\n%s", text)
	}
}

// A review a provider ran is ingested through the same path, so an undecided
// answer flags the intent and names the captured issue in the text render too.
func TestAProviderRunAuditThatIsUndecidedNamesTheOwedCheck(t *testing.T) {
	srcRoot := repoRootFromTest(t)
	root := intentTestRepo(t)
	writeRepoFile(t, root, ".abcd/development/intents/shipped/itd-10-alpha.md", conditionedIntent)
	t.Setenv("ABCD_PLUGIN_ROOT", srcRoot)
	p := newChatFake(t, "typesafe/jev-1.13", func(body string) string {
		v := strings.NewReplacer(
			`"verdict": "MET"`, `"verdict": "INCONCLUSIVE"`,
			`{"MET": 1, "MET_WITH_CONCERNS": 0, "NOT_MET": 0, "INCONCLUSIVE": 0}`,
			`{"MET": 0, "MET_WITH_CONCERNS": 0, "NOT_MET": 0, "INCONCLUSIVE": 1}`).
			Replace(conditionedVerdict(regexp.MustCompile(`rcp-[0-9a-f]{12}`).FindString(body)))
		for _, h := range []struct{ key, placeholder string }{
			{"rubric_hash", "sha256:" + strings.Repeat("a", 64)},
			{"prompt_hash", "sha256:" + strings.Repeat("b", 64)},
		} {
			m := regexp.MustCompile(`- ` + h.key + `: (sha256:[0-9a-f]{64})`).FindStringSubmatch(body)
			if m == nil {
				return "{}"
			}
			v = strings.Replace(v, h.placeholder, m[1], 1)
		}
		return v
	})
	pointMachine(t, os.Getenv("HOME"), p.srv.URL, false, "", "intent-auditor")
	stdout, stderr, err := runCLISplit(t, "intent", "audit", "itd-10")
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	if !strings.Contains(stdout, "audit owed: ac-1 INCONCLUSIVE") || !regexp.MustCompile(`captured iss-[0-9]+ to carry the check`).MatchString(stdout) {
		t.Fatalf("the provider-run ingest must name the owed check and its issue:\n%s", stdout)
	}
	if _, err := os.Stat(filepath.Join(root, ".abcd", "development", "intents", "shipped", "itd-10-alpha.md")); err != nil {
		t.Fatalf("the intent must stay shipped: %v", err)
	}
}
