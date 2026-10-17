package lint

import (
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core/recordid"
)

// TestRecordFamilyKeyReadsTheDecisionStore closes the gap step 1 left
// (itd-2609151138388536): the committed record_family_key rule reads its stores
// from its own record_stores map, and that map must carry the dcn store the
// record_schema rule already lists, or a decision record's frontmatter keys are
// never read by the rule at all.
func TestRecordFamilyKeyReadsTheDecisionStore(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("..", "..", "..", ".abcd", "record-lint.json"))
	if err != nil {
		t.Fatalf("the committed record-lint.json does not load: %v", err)
	}
	rule, ok := cfg.Rules[ruleRecordFamilyKey]
	if !ok {
		t.Fatalf("the committed config carries no %s rule", ruleRecordFamilyKey)
	}
	if got := rule.RecordStores["dcn"]; got != recordid.DecisionsRelDir {
		t.Errorf("%s's record_stores maps dcn to %q, want %q", ruleRecordFamilyKey, got, recordid.DecisionsRelDir)
	}
}
