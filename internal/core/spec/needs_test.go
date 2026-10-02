package spec

import (
	"slices"
	"strings"
	"testing"
)

// A step's `- needs:` line (ruling DR6, spc-2609202134341288): `none`, or the
// earlier steps it waits for. A step without the line needs every step before
// it (ruling DR6b), which the parse reports as an undeclared need.
func TestParseStepsReadsNeeds(t *testing.T) {
	steps, err := ParseSteps("## Steps\n\n1. One\n2. Two\n   - needs: none\n3. Three\n   - needs: 1\n4. Four\n   - needs: 1, 3\n")
	if err != nil {
		t.Fatalf("ParseSteps: %v", err)
	}
	if steps[0].NeedsDeclared || steps[0].Needs != nil {
		t.Errorf("step 1 declares nothing: %+v", steps[0])
	}
	if !steps[1].NeedsDeclared || len(steps[1].Needs) != 0 {
		t.Errorf("step 2 needs none: %+v", steps[1])
	}
	if !steps[2].NeedsDeclared || !slices.Equal(steps[2].Needs, []int{1}) {
		t.Errorf("step 3 needs 1: %+v", steps[2])
	}
	if !slices.Equal(steps[3].Needs, []int{1, 3}) {
		t.Errorf("step 4 needs 1, 3: %+v", steps[3])
	}
	// The default is every step before it.
	if got := steps[0].Requires(); len(got) != 0 {
		t.Errorf("step 1 requires nothing, got %v", got)
	}
	if got := (Step{Number: 3}).Requires(); !slices.Equal(got, []int{1, 2}) {
		t.Errorf("an undeclared step 3 requires 1 and 2, got %v", got)
	}
	if got := steps[1].Requires(); len(got) != 0 {
		t.Errorf("needs: none requires nothing, got %v", got)
	}
}

// C6: the parser refuses a need naming the step itself, a later step, or a step
// the spec does not list, naming the line.
func TestParseStepsRefusesABadNeeds(t *testing.T) {
	for name, section := range map[string]string{
		"itself":    "## Steps\n\n1. One\n2. Two\n   - needs: 2\n",
		"later":     "## Steps\n\n1. One\n2. Two\n   - needs: 3\n3. Three\n",
		"unlisted":  "## Steps\n\n1. One\n2. Two\n   - needs: 0\n",
		"not a num": "## Steps\n\n1. One\n2. Two\n   - needs: first\n",
		"twice":     "## Steps\n\n1. One\n2. Two\n   - needs: 1\n   - needs: none\n",
		"none and":  "## Steps\n\n1. One\n2. Two\n   - needs: none, 1\n",
	} {
		_, err := ParseSteps(section)
		if err == nil {
			t.Errorf("%s: a bad needs line must be refused", name)
			continue
		}
		want := "line 5"
		if name == "twice" {
			want = "line 6"
		}
		if !strings.Contains(err.Error(), want) {
			t.Errorf("%s: the refusal names the line: %v", name, err)
		}
	}
}

// C6's remainder: steps 1 and 3 landed, step 2 needs 1, step 4 needs 1 and 2.
// The remainder lists old step 2 as step 1 with `- needs: none` and old step 4
// as step 2 with `- needs: 1`, names both rewritten lines, and parses.
func TestCarryUnlandedRewritesNeeds(t *testing.T) {
	const src = "## Steps\n\n1. One\n   - landed: #1\n2. Two\n   - needs: 1\n   - tests: two\n3. Three\n   - landed: #3\n4. Four\n   - needs: 1, 2\n5. Five\n"
	steps, err := ParseSteps(src)
	if err != nil {
		t.Fatal(err)
	}
	carried, rewrites := CarryUnlanded(steps)
	if len(carried) != 3 || carried[0].Title != "Two" || carried[1].Title != "Four" || carried[2].Title != "Five" {
		t.Fatalf("carried = %+v", carried)
	}
	if len(rewrites) != 2 {
		t.Fatalf("two needs lines are rewritten, got %+v", rewrites)
	}
	if rewrites[0].Step != 1 || rewrites[0].Before != "- needs: 1" || rewrites[0].After != "- needs: none" {
		t.Errorf("old step 2's line: %+v", rewrites[0])
	}
	if rewrites[1].Step != 2 || rewrites[1].Before != "- needs: 1, 2" || rewrites[1].After != "- needs: 1" {
		t.Errorf("old step 4's line: %+v", rewrites[1])
	}
	out := renderSteps(carried)
	back, err := ParseSteps("## Steps\n\n" + out)
	if err != nil {
		t.Fatalf("the remainder must parse: %v\n%s", err, out)
	}
	if !back[0].NeedsDeclared || len(back[0].Needs) != 0 || !slices.Equal(back[1].Needs, []int{1}) || back[2].NeedsDeclared {
		t.Fatalf("the remainder's needs: %+v\n%s", back, out)
	}
	if !strings.Contains(out, "   - tests: two\n") {
		t.Errorf("every other line is carried verbatim:\n%s", out)
	}
}
