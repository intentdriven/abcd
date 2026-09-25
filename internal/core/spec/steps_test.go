package spec

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// steppedSpec is a spec body listing three steps, the second marked landed.
const steppedSpec = `---
id: spc-1
slug: alpha
intent: itd-9
---
# alpha

## Summary

The design record.

## Steps

_Placeholder guidance an author left above the list._

1. **The parser**
   - packages: internal/core/spec
   - tests: a stepped and an unstepped spec
2. The template
   - packages: internal/core/intent
   - landed: #101
3. The remainder copy
   - packages: internal/core/intent
   - tests: the remainder carries the unlanded steps
   A note the author kept under the step.

## Approach

Prose.
`

// A spec that lists its steps yields them in document order, each with its
// footprint and its landed marker.
func TestParseStepsReadsTheOrderedList(t *testing.T) {
	steps, err := ParseSteps(steppedSpec)
	if err != nil {
		t.Fatalf("ParseSteps: %v", err)
	}
	if len(steps) != 3 {
		t.Fatalf("got %d steps, want 3: %+v", len(steps), steps)
	}
	want := []Step{
		{Number: 1, Title: "The parser", Packages: "internal/core/spec", Tests: "a stepped and an unstepped spec"},
		{Number: 2, Title: "The template", Packages: "internal/core/intent", Landed: "#101"},
		{Number: 3, Title: "The remainder copy", Packages: "internal/core/intent", Tests: "the remainder carries the unlanded steps"},
	}
	for i, w := range want {
		g := steps[i]
		if g.Number != w.Number || g.Title != w.Title || g.Packages != w.Packages || g.Tests != w.Tests || g.Landed != w.Landed || g.Implicit {
			t.Errorf("step %d = %+v, want %+v", i+1, g, w)
		}
	}
}

// A spec with no steps listed is built as one step: an absent section, an
// empty one and the minted placeholder all read as a single implicit step.
func TestStepsOfAnUnsteppedSpecIsOneImplicitStep(t *testing.T) {
	for name, content := range map[string]string{
		"absent":      "---\nid: spc-1\n---\n# a\n\n## Summary\n\nText.\n",
		"empty":       "# a\n\n## Steps\n\n## Approach\n\nText.\n",
		"placeholder": renderSpec("spc-1", "a-slug", "itd-9", mustStamp(t)),
	} {
		t.Run(name, func(t *testing.T) {
			listed, err := ParseSteps(content)
			if err != nil {
				t.Fatalf("ParseSteps: %v", err)
			}
			if len(listed) != 0 {
				t.Fatalf("no step is listed, got %+v", listed)
			}
			steps, err := Steps(content)
			if err != nil {
				t.Fatalf("Steps: %v", err)
			}
			if len(steps) != 1 || !steps[0].Implicit || steps[0].Number != 1 {
				t.Fatalf("an unstepped spec is one implicit step, got %+v", steps)
			}
		})
	}
	// A stepped spec's Steps is its list, with no implicit step added.
	steps, err := Steps(steppedSpec)
	if err != nil || len(steps) != 3 || steps[0].Implicit {
		t.Fatalf("Steps(stepped) = %+v, %v; want the three listed steps", steps, err)
	}
}

// The minted stub carries an empty `## Steps` section, placed after every
// other stub section so a later section slots in before it.
func TestRenderSpecSeedsAnEmptyStepsSection(t *testing.T) {
	minted := renderSpec("spc-1", "a-slug", "itd-9", mustStamp(t))
	if !strings.Contains(minted, "\n## Steps\n") {
		t.Fatalf("the minted stub must carry a `## Steps` section:\n%s", minted)
	}
	if strings.Index(minted, "## Summary") > strings.Index(minted, "## Steps") {
		t.Fatalf("`## Steps` must follow `## Summary`:\n%s", minted)
	}
	if !BodyIsStub(minted) {
		t.Fatal("the Steps placeholder must not hide the Summary stub marker")
	}
}

// A section that is not a numbered list is refused, never guessed at: the
// remainder copy writes from this parse.
func TestParseStepsRefusesAMalformedSection(t *testing.T) {
	for name, content := range map[string]string{
		"prose":           "## Steps\n\nFirst we build the parser, then the loop.\n",
		"unordered":       "## Steps\n\n- The parser\n- The loop\n",
		"prose after":     "## Steps\n\n1. The parser\n\nThen the loop.\n",
		"duplicate key":   "## Steps\n\n1. The parser\n   - tests: a\n   - tests: b\n",
		"empty title":     "## Steps\n\n1. **  **\n",
		"two sections":    "## Steps\n\n1. A\n\n## Steps\n\n1. B\n",
		"fenced in steps": "## Steps\n\n```\n1. A\n```\n",
		"comment in step": "## Steps\n\n1. A\n   <!--\n2. B\n   -->\n",
	} {
		t.Run(name, func(t *testing.T) {
			if steps, err := ParseSteps(content); err == nil {
				t.Fatalf("a malformed Steps section must be refused, got %+v", steps)
			}
			if _, err := Steps(content); err == nil {
				t.Fatal("Steps must refuse what ParseSteps refuses")
			}
		})
	}
}

// A `## Steps` heading inside a fenced example elsewhere is an example, not
// the section.
func TestParseStepsIgnoresAFencedHeading(t *testing.T) {
	content := "## Approach\n\n```markdown\n## Steps\n\n1. An example\n```\n"
	steps, err := ParseSteps(content)
	if err != nil || len(steps) != 0 {
		t.Fatalf("a fenced heading is not the section: %+v, %v", steps, err)
	}
}

// The section's bounds are mdrecord's: a fence closes only on its own marker,
// so a `~~~` line inside a backtick fence above the section neither closes it
// nor hides the live `## Steps` below. Read wrong, the section vanishes with
// no error and `spec close --remainder` drops the unlanded steps.
func TestParseStepsReadsPastAMixedMarkerFence(t *testing.T) {
	content := "## Approach\n\n```text\n~~~\nan example\n```\n\n## Steps\n\n1. The parser\n   - packages: internal/core/spec\n2. The loop\n   - packages: internal/core/loop\n\n## Footprint\n\nText.\n"
	steps, err := ParseSteps(content)
	if err != nil {
		t.Fatalf("ParseSteps: %v", err)
	}
	if len(steps) != 2 || steps[0].Title != "The parser" || steps[1].Title != "The loop" {
		t.Fatalf("the live section lists two steps, got %+v", steps)
	}
}

// A `## Steps` heading parked inside an HTML comment is not the section, and
// not a second one: the live heading below it is read.
func TestParseStepsIgnoresACommentedHeading(t *testing.T) {
	content := "## Approach\n\n<!--\n## Steps\n\n1. A parked draft\n-->\n\n## Steps\n\n1. The parser\n2. The loop\n"
	steps, err := ParseSteps(content)
	if err != nil {
		t.Fatalf("a commented heading is not a second section: %v", err)
	}
	if len(steps) != 2 || steps[0].Title != "The parser" || steps[1].Title != "The loop" {
		t.Fatalf("the live section lists two steps, got %+v", steps)
	}
}

// The remainder copy: the steps not marked landed, renumbered from one, each
// carrying the lines the author wrote under it.
func TestRenderStepsCarriesTheUnlandedStepsVerbatim(t *testing.T) {
	steps, err := ParseSteps(steppedSpec)
	if err != nil {
		t.Fatal(err)
	}
	unlanded := Unlanded(steps)
	if len(unlanded) != 2 || unlanded[0].Title != "The parser" || unlanded[1].Title != "The remainder copy" {
		t.Fatalf("Unlanded = %+v, want steps 1 and 3", unlanded)
	}
	out := RenderSteps(unlanded)
	for _, want := range []string{
		"1. **The parser**\n   - packages: internal/core/spec\n",
		"2. The remainder copy\n",
		"   A note the author kept under the step.\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered steps lack %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "The template") || strings.Contains(out, "landed: #101") {
		t.Errorf("a landed step must not be carried:\n%s", out)
	}
	// The rendering reads back as the same steps, renumbered.
	back, err := ParseSteps("## Steps\n\n" + out)
	if err != nil || len(back) != 2 || back[1].Number != 2 || back[1].Tests != unlanded[1].Tests {
		t.Fatalf("rendered steps do not read back: %+v, %v\n%s", back, err, out)
	}
}

// A minted remainder carries the steps it is given in its Steps section.
func TestRenderSpecWithStepsCarriesThem(t *testing.T) {
	steps, err := ParseSteps(steppedSpec)
	if err != nil {
		t.Fatal(err)
	}
	minted := renderSpecWithSteps("spc-2", "the-rest", "itd-9", mustStamp(t), Unlanded(steps))
	back, err := ParseSteps(minted)
	if err != nil || len(back) != 2 {
		t.Fatalf("the minted remainder must carry the two unlanded steps: %+v, %v\n%s", back, err, minted)
	}
}

// An unclosed fence or HTML comment runs to end of file, so a `## Steps`
// heading below it is masked and the section would read as absent: zero steps
// and no error, and `spec close --remainder` would mint the placeholder and
// drop the unlanded steps. The reader refuses instead, naming the opener's
// line — the thing to close — wherever in the document it sits.
func TestParseStepsRefusesAnUnclosedSpan(t *testing.T) {
	const live = "## Steps\n\n1. The parser\n2. The loop\n"
	for name, tc := range map[string]struct {
		content, construct string
		line               int
	}{
		"fence above":   {"# a\n\n## Approach\n\n```text\nan example\n\n" + live, "fenced code block", 5},
		"comment above": {"# a\n\n## Approach\n\n<!-- parked\n\n" + live, "HTML comment", 5},
		"fence below":   {"# a\n\n" + live + "\n## Approach\n\n~~~\n", "fenced code block", 10},
	} {
		t.Run(name, func(t *testing.T) {
			steps, err := ParseSteps(tc.content)
			if err == nil {
				t.Fatalf("an unclosed span must be refused, got %d step(s) and no error: %+v", len(steps), steps)
			}
			for _, want := range []string{fmt.Sprintf("line %d", tc.line), tc.construct} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("the refusal must name %q: %v", want, err)
				}
			}
			if !errors.Is(err, ErrUnclosedSpan) {
				t.Errorf("the refusal must be ErrUnclosedSpan, so a caller can name the remedy: %v", err)
			}
			if _, err := Steps(tc.content); err == nil {
				t.Fatal("Steps must refuse what ParseSteps refuses")
			}
		})
	}
}
