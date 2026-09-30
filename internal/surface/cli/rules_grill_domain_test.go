package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGrillDomainCarriesQuestionVisibilityAndAddresseeRules pins this
// repository's GRILL domain to the two rules the product thinker's
// 2026-09-29 captures asked for: the example that makes a question answerable
// sits in the question and in each option's preview (iss-2609291925134691),
// and the addressee is classified and the mode label set before every question
// (iss-2609291925149138). The count is pinned too, so the ten rules the domain
// already carried are added to, never rewritten away: a change to the count is
// a deliberate edit to this pin.
func TestGrillDomainCarriesQuestionVisibilityAndAddresseeRules(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(testRepoRoot(), ".abcd", "rules.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rules struct {
		Domains map[string]struct {
			Rules []string `json:"rules"`
		} `json:"domains"`
	}
	if err := json.Unmarshal(data, &rules); err != nil {
		t.Fatalf("parse .abcd/rules.json: %v", err)
	}
	dom, ok := rules.Domains["GRILL"]
	if !ok {
		t.Fatal(".abcd/rules.json declares no GRILL domain")
	}
	const wantCount = 12
	if len(dom.Rules) != wantCount {
		t.Errorf("GRILL carries %d rules, want %d (ten standing rules plus the two from the 2026-09-29 captures)", len(dom.Rules), wantCount)
	}
	for _, want := range []struct {
		issue  string
		phrase []string
	}{
		{"iss-2609291925134691", []string{"IN the question text", "each option's preview", "invisible while the question shows"}},
		{"iss-2609291925149138", []string{"addressee", "FIRST", "mode verb", "mixed interview"}},
	} {
		found := false
		for _, r := range dom.Rules {
			all := true
			for _, p := range want.phrase {
				if !strings.Contains(r, p) {
					all = false
					break
				}
			}
			if all {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GRILL has no rule carrying %q (%s)", want.phrase, want.issue)
		}
	}
}
