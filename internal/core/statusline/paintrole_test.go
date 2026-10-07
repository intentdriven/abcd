package statusline

import "testing"

// TestPaintRoleUsesTheBadgePair: a role's text is painted in the badge's own
// fixed pair, padded as the badge pads its word, so the board's view label and
// the badge are one colour set (spc-2610031844142274, decision 10); a state
// with no fixed pair is left unpainted.
func TestPaintRoleUsesTheBadgePair(t *testing.T) {
	for _, s := range []State{StateProductThinker, StateFacilitator} {
		if got, want := PaintRole(s, "view"), paint(rolePairs[s], " view "); got != want {
			t.Errorf("PaintRole(%v) = %q, want the badge pair %q", s, got, want)
		}
	}
	if got := PaintRole(StateManaged, "view"); got != "view" {
		t.Errorf("PaintRole(managed) = %q, want the text unpainted", got)
	}
}
