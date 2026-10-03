package ahoy

import (
	"encoding/json"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/question"
)

// TestQuestionEmDashRuleIsTheDocsLintToken holds the question check's rule 12
// (no em dash in a list item, spc-2610030944505997) to the docs-lint token it
// names, punctuation/em-dash-in-list-item, as the seed every install writes
// defines it. The two live in different packages, so this test is what stops
// one drifting from the other silently.
func TestQuestionEmDashRuleIsTheDocsLintToken(t *testing.T) {
	var cfg struct {
		BannedTokens []lint.BannedToken `json:"banned_tokens"`
	}
	if err := json.Unmarshal([]byte(publicFamilySeed), &cfg); err != nil {
		t.Fatalf("the docs-lint seed is not JSON: %v", err)
	}
	for _, tok := range cfg.BannedTokens {
		if tok.ID != emDashTokenID {
			continue
		}
		if tok.Pattern != question.EmDashListItemPattern {
			t.Errorf("the question check's em-dash pattern %q differs from the docs-lint token's %q",
				question.EmDashListItemPattern, tok.Pattern)
		}
		return
	}
	t.Fatalf("the docs-lint seed carries no %s token", emDashTokenID)
}
