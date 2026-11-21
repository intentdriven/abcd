package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPhaseNegotiatorDraftNamesThePlanningInterview holds itd-41 to the
// surfaces register (iss-2610262039128746). The draft called the grill a
// skill and cited `/abcd:intent grill` as a live surface, while the register
// says the grill's design target was an `/abcd:intent` sub-verb that never
// shipped, superseded by the planning interview (itd-94,
// `abcd intent interview`).
func TestPhaseNegotiatorDraftNamesThePlanningInterview(t *testing.T) {
	path := filepath.Join(repoRootFromPackage, ".abcd", "development", "intents", "drafts", "itd-41-phase-negotiator.md")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		t.Skip("itd-41 has left drafts/; the remedy is moot once the draft is superseded")
	}
	if err != nil {
		t.Fatalf("read itd-41: %v", err)
	}
	text := string(raw)
	for _, banned := range []string{"grill skill", "/abcd:intent grill", "alongside `grill`", "grill reports"} {
		if strings.Contains(text, banned) {
			t.Errorf("itd-41 still says %q; the grill never shipped, and the planning interview superseded it", banned)
		}
	}
	for _, want := range []string{"itd-94", "`abcd intent interview`"} {
		if !strings.Contains(text, want) {
			t.Errorf("itd-41 does not name %q, the planning interview that superseded the grill", want)
		}
	}
}
