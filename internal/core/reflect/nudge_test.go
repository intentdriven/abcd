package reflect

import (
	"strings"
	"testing"
)

// Criterion 8's line (spec scope 6): one line that says a retrospective for the
// release is owed and names the command. Printing it once, at the end of the
// cut, is the front door's; the text is here so every door says the same.
func TestNudgeSaysOnceThatARetrospectiveIsOwedAndNamesTheCommand(t *testing.T) {
	got := Nudge("v0.11.0")
	if strings.Contains(got, "\n") {
		t.Errorf("nudge spans lines: %q", got)
	}
	for _, want := range []string{"retrospective", "v0.11.0", "owed", "/abcd:reflect v0.11.0"} {
		if !strings.Contains(got, want) {
			t.Errorf("nudge %q lacks %q", got, want)
		}
	}
}
