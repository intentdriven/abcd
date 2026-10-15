package spec

import (
	"strings"
	"testing"
)

// TestParseStepsMasksAStepTitlesHiddenRunes (iss-2610121542268630): a step
// title carrying U+202E or U+009B is read with the rune masked, since the loop
// stores and echoes the title (its brief, state.json, `implement status
// --json`, a pull request's title); a spec is the owner's own text, so the
// step is kept rather than refused, and a copy of the steps still carries the
// author's words as written.
func TestParseStepsMasksAStepTitlesHiddenRunes(t *testing.T) {
	for _, bad := range []string{"‮", "\u009b"} {
		doc := "# s\n\n## Steps\n\n1. The state" + bad + " file\n2. The verb\n"
		steps, err := ParseSteps(doc)
		if err != nil {
			t.Fatalf("%q: the steps parse: %v", bad, err)
		}
		if len(steps) != 2 || steps[0].Title != "The state? file" || steps[1].Title != "The verb" {
			t.Fatalf("%q: the title is masked and the step kept: %+v", bad, steps)
		}
		if got := renderSteps(steps); !strings.Contains(got, "The state"+bad+" file") {
			t.Errorf("%q: a copy of the steps keeps the author's words: %q", bad, got)
		}
	}
}
