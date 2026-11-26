package surface

import (
	"strings"
	"testing"
)

// TestEveryNextEntryIsStepsOrAReason holds each entry of the table to one of
// its two forms: one or more steps, each a command template that starts with
// the binary's name and a line saying when it applies, or the explicit empty
// list with the reason nothing usually follows. An entry with both, or
// neither, says nothing a reader can act on.
func TestEveryNextEntryIsStepsOrAReason(t *testing.T) {
	if len(NextPathsForTest()) == 0 {
		t.Fatal("the next-steps table is empty")
	}
	for _, path := range NextPathsForTest() {
		n, _ := NextFor(path)
		switch {
		case len(n.Steps) == 0 && strings.TrimSpace(n.None) == "":
			t.Errorf("%s: no steps and no reason nothing follows", path)
		case len(n.Steps) > 0 && n.None != "":
			t.Errorf("%s: steps and a reason nothing follows, both", path)
		}
		seen := map[string]bool{}
		for _, s := range n.Steps {
			if s.Command != "abcd" && !strings.HasPrefix(s.Command, "abcd ") {
				t.Errorf("%s: step %q does not start with the binary's name", path, s.Command)
			}
			if strings.TrimSpace(s.When) == "" {
				t.Errorf("%s: step %q says nothing about when it applies", path, s.Command)
			}
			if seen[s.Command+"\x00"+s.When] {
				t.Errorf("%s: step %q is listed twice", path, s.Command)
			}
			seen[s.Command+"\x00"+s.When] = true
		}
	}
}

// TestCanonicalFactsKeepsTheNextStepsInTheirOrder: the steps are the order
// a caller weighs them in, so the canonical form keeps it, and copies them
// rather than sharing the input's backing array.
func TestCanonicalFactsKeepsTheNextStepsInTheirOrder(t *testing.T) {
	in := []Facts{{Path: "abcd z", Next: []NextStep{{Command: "abcd z b", When: "second"}, {Command: "abcd z a", When: "first"}}}}
	out := CanonicalFacts(in)
	if out[0].Next[0].Command != "abcd z b" {
		t.Fatalf("the steps were reordered: %+v", out[0].Next)
	}
	out[0].Next[0].When = "changed"
	if in[0].Next[0].When != "second" {
		t.Error("the canonical facts share the steps' backing array with the input")
	}
}
