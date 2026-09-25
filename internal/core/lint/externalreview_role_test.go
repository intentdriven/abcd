package lint_test

import (
	"strings"
	"testing"
)

// TestExternalReviewRoleLookupFailsLoud is iss-302. role_of folded every
// failed collaborator-permission call into the role "none", so an API error —
// a 403 for a token that cannot read the endpoint, a rate limit — was
// indistinguishable from a real answer: an invited author read as external, a
// real approval went uncounted, and the red check named the wrong cause. A
// failed lookup now fails the check and says so.
func TestExternalReviewRoleLookupFailsLoud(t *testing.T) {
	t.Run("the author's role cannot be read", func(t *testing.T) {
		out, code := runExternalReview(t, [][]review{{}}, collaborators, []string{"outsider"})
		if code == 0 {
			t.Fatalf("the gate passed although the author's role could not be read:\n%s", out)
		}
		if strings.Contains(out, "author is external") {
			t.Errorf("an API failure was reported as an external author:\n%s", out)
		}
		if !strings.Contains(out, "could not read the repository role of 'outsider'") {
			t.Errorf("the failure does not name the lookup that failed:\n%s", out)
		}
	})
	t.Run("a reviewer's role cannot be read", func(t *testing.T) {
		out, code := runExternalReview(t, [][]review{
			{{"alice", "APPROVED", "2026-01-01T00:00:00Z"}, {"bob", "APPROVED", "2026-01-01T01:00:00Z"}, {"carol", "APPROVED", "2026-01-01T02:00:00Z"}},
		}, collaborators, []string{"alice"})
		if code == 0 {
			t.Errorf("the gate passed although a reviewer's role could not be read:\n%s", out)
		}
		if strings.Contains(out, "not counted: alice") {
			t.Errorf("an API failure was reported as a reviewer without a role:\n%s", out)
		}
		if !strings.Contains(out, "could not read the repository role of 'alice'") {
			t.Errorf("the failure does not name the lookup that failed:\n%s", out)
		}
	})
}
