package lint

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestSafetyGateIntentNamesItsInvocationPointsPlainly holds itd-62 off the
// word "tier" for where its gate is invoked (iss-2610262033027180). The brief
// spends "three-tier" on the `.abcd/` record layout (04-surfaces/01-ahoy.md),
// and itd-62 once called CI, pre-commit and runtime hooks "the same three-tier
// model abcd uses elsewhere", a precedent that points a reader at the record
// layout. The intent names the three invocation points plainly instead.
func TestSafetyGateIntentNamesItsInvocationPointsPlainly(t *testing.T) {
	const intent = ".abcd/development/intents/drafts/itd-62-pluggable-safety-gate.md"
	raw, err := os.ReadFile(filepath.Join(repoRootFromPackage, filepath.FromSlash(intent)))
	if err != nil {
		t.Fatalf("read %s: %v", intent, err)
	}
	body := strings.Join(strings.Fields(string(raw)), " ")

	tier := regexp.MustCompile(`(?i)\bthree[- ]tiers?\b`)
	if loc := tier.FindStringIndex(body); loc != nil {
		from, to := max(0, loc[0]-60), min(len(body), loc[1]+60)
		t.Errorf("%s calls its invocation points tiers (%q); the brief's three tiers are the .abcd/ record layout", intent, body[from:to])
	}
	if !strings.Contains(body, "CI, pre-commit, and runtime hooks") {
		t.Errorf("%s no longer names the gate's three invocation points (CI, pre-commit, and runtime hooks)", intent)
	}
}
