package lint

import (
	"strings"
	"testing"
)

// An enabled rule whose own input path is blank checked nothing and returned
// clean, before any of its fail-closed guards ran: gate_lockstep with a blank
// runbook or workflow (iss-336), and its sibling surface_coverage with a blank
// registry. Both are refused when the config loads, naming the key.
func TestLoadConfigRefusesAnArmedRuleWithABlankInputPath(t *testing.T) {
	for name, tc := range map[string]struct{ body, key string }{
		"gate_lockstep without a workflow":    {`{"roots":["rec"],"rules":{"gate_lockstep":{"enabled":true,"severity":"blocker","runbook":"r.md"}}}`, "workflow"},
		"gate_lockstep without a runbook":     {`{"roots":["rec"],"rules":{"gate_lockstep":{"enabled":true,"severity":"blocker","workflow":"w.yml"}}}`, "runbook"},
		"surface_coverage without a registry": {`{"roots":["rec"],"rules":{"surface_coverage":{"enabled":true,"severity":"blocker"}}}`, "registry"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := LoadConfig(writeConfig(t, tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.key) {
				t.Fatalf("an armed rule with a blank %s loaded: %v", tc.key, err)
			}
		})
	}
	// Disabled, the blank is not a refusal: the rule runs nothing on purpose.
	if _, err := LoadConfig(writeConfig(t, `{"roots":["rec"],"rules":{"gate_lockstep":{"enabled":false}}}`)); err != nil {
		t.Fatalf("a disabled rule with blank paths must load: %v", err)
	}
}
