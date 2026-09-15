package lint

import (
	"path/filepath"
	"testing"
)

// The runbook, the receipt example and the iss-35 plan all say a receipt's
// judgeModel is a pinned snapshot, never a floating alias — a release verdict
// must name the judge that produced it, so the pass can be re-run against the
// same judge. The gate refused only a blank value (iss-2609012020521049), so a
// bare family name or a rolling alias passed as if pinned, and the v0.7.0
// receipts had to disclose in prose that the rule had no code behind it.
//
// Pinned means the id carries a version or date component. That is the rule
// the documented example (claude-opus-4-8) and every committed receipt already
// satisfy; what it refuses is the id that resolves to whatever the vendor
// serves today.
func TestReceiptGateRefusesFloatingJudgeModel(t *testing.T) {
	const sha = "0123456789abcdef0123456789abcdef01234567"
	const gate = "iss35-brief-surface-crosscheck"
	reviews := filepath.Join(".abcd", "work", "reviews")

	cases := []struct {
		model string
		want  int
	}{
		{"claude-opus-4-8", 0},      // the documented example: family plus version
		{"Claude Fable 5", 0},       // a committed receipt's spelling
		{"claude-opus-5", 0},        // a committed receipt's spelling
		{"gpt-5-2026-08-01", 0},     // a dated snapshot
		{"opus", 1},                 // bare family alias
		{"claude-opus", 1},          // family with no version
		{"claude-opus-latest", 1},   // rolling alias, even though it names a family
		{"claude-opus-4-latest", 1}, // rolling alias with a version fragment
	}
	for _, c := range cases {
		t.Run(c.model, func(t *testing.T) {
			root := t.TempDir()
			// Pre-manifest era: the judge-model rule predates the manifest and
			// applies in both eras, so the simplest receipt exercises it.
			body := `{
  "subject": {"digest": {"gitCommit": "` + sha + `"}},
  "verificationResult": "PROMOTE",
  "judgeModel": "` + c.model + `",
  "policy": {"detector": "` + gate + `"},
  "failing": []
}`
			writeFile(t, root, filepath.Join(reviews, sha, gate+".json"), body)
			cfg := RuleConfig{Enabled: true, Severity: severityBlocker, ReceiptsDir: reviews, Commit: sha, RequiredGates: []string{gate}}
			if n := countRule(runReceiptGate(t, root, cfg), "receipt_gate"); n != c.want {
				t.Fatalf("judgeModel %q: want %d receipt_gate finding(s), got %d", c.model, c.want, n)
			}
		})
	}
}
