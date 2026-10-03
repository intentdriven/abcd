package question

import "testing"

// TestMatchesIsTheSharedNarrowingRule holds the narrowing rule the
// plain-Terminal list and the guided connect session share
// (spc-2610030911534855 step 2; spc-2610031241482088 "Narrowing uses
// question.Matches"): the typed text, matched case-insensitively as a
// substring of the label or of the value; empty text matches everything.
func TestMatchesIsTheSharedNarrowingRule(t *testing.T) {
	o := Option{Value: "anthropic/claude-sonnet", Label: "Claude Sonnet"}
	for _, tc := range []struct {
		filter string
		want   bool
	}{
		{"", true},
		{"claude", true},
		{"CLAUDE", true},
		{"sonnet", true},
		{"anthropic", true}, // the value alone holds it
		{"de so", true},     // a substring of the label, across the space
		{"gpt", false},
		{"claude-sonnet-4", false},
	} {
		if got := Matches(tc.filter, o); got != tc.want {
			t.Errorf("Matches(%q) = %v, want %v", tc.filter, got, tc.want)
		}
	}
}
