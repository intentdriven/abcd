package oracle

import (
	"strings"
	"testing"
)

// TestAnExactDenylistEntryMatchesTheModelWithABracketedSuffix: a bracketed
// suffix on the model's last segment ([1m]) is a spelling of the same model,
// as a :variant is, so an exact entry refuses it, and a vendor entry still
// refuses it through the route.
func TestAnExactDenylistEntryMatchesTheModelWithABracketedSuffix(t *testing.T) {
	for _, tc := range []struct{ entry, model string }{
		{"claude-opus-5-5", "claude-opus-5-5[1m]"},
		{"claude-opus-5-5", "CLAUDE-OPUS-5-5[1M]"},
		{"anthropic/claude-opus-5-5", "anthropic/claude-opus-5-5[1m]"},
		{"anthropic/*", "anthropic/claude-opus-5-5[1m]"},
		{"vendor/m", "vendor/m:free[1m]"},
	} {
		if _, denied := Denied([]DenyEntry{{Pattern: tc.entry}}, tc.model); !denied {
			t.Errorf("%q does not refuse %q", tc.entry, tc.model)
		}
	}
	if _, denied := Denied([]DenyEntry{{Pattern: "claude-opus-5-5"}}, "claude-opus-5-55[1m]"); denied {
		t.Error("an entry refuses a different model")
	}
}

// TestTheAllowlistStaysExactForABracketedSuffix: the provider's list admits a
// model only as it is written there; the suffix the denylist reads past never
// widens what a list admits.
func TestTheAllowlistStaysExactForABracketedSuffix(t *testing.T) {
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{"anthropic":{"base_url":"https://api.example.invalid/v1","models":["claude-haiku-4-5"]}}}}`)
	c := f.loadAPI()
	if err := c.Admit("anthropic", "claude-haiku-4-5"); err != nil {
		t.Fatalf("the listed model is refused: %v", err)
	}
	if err := c.Admit("anthropic", "claude-haiku-4-5[1m]"); err == nil || !strings.Contains(err.Error(), "not on anthropic's list") {
		t.Fatalf("a suffixed spelling the list does not carry is admitted: %v", err)
	}
}
