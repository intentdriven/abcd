package spec

import (
	"slices"
	"strings"
	"testing"
)

// TestRenderSpecSeedsAnEmptyFootprintSection is itd-2609211116005482's
// criterion 7: a minted spec stub carries an empty `## Footprint` section
// (packages, tests) for the author to fill, above `## Steps`, and the section
// reads as present and empty.
func TestRenderSpecSeedsAnEmptyFootprintSection(t *testing.T) {
	minted := renderSpec("spc-1", "a-slug", "itd-9", mustStamp(t))
	if !strings.Contains(minted, "\n"+FootprintHeading+"\n\n- packages:\n- tests:\n") {
		t.Fatalf("the minted stub must carry an empty `## Footprint` section (packages, tests):\n%s", minted)
	}
	if strings.Index(minted, FootprintHeading) > strings.Index(minted, StepsHeading) {
		t.Fatalf("`## Steps` stays last, below `## Footprint`:\n%s", minted)
	}
	fp := ReadFootprint(minted)
	if !fp.Present || len(fp.Packages) != 0 || fp.Tests != "" {
		t.Fatalf("the seeded section is present and empty: %+v", fp)
	}
	if !BodyIsStub(minted) {
		t.Fatal("the Footprint section must not hide the Summary stub marker")
	}
	// The seeded bullets are not steps: the minted spec still lists none.
	if listed, err := ParseSteps(minted); err != nil || len(listed) != 0 {
		t.Fatalf("the minted spec lists no step: %v %v", listed, err)
	}
}

// TestReadFootprintReadsTheSection reads a written section: the package list
// split on commas, the tests text with its indented continuation, and the
// absence of the section told apart from an empty one.
func TestReadFootprintReadsTheSection(t *testing.T) {
	written := "# a\n\n## Summary\n\nText.\n\n## Footprint\n\n" +
		"- packages: internal/core/implement, internal/core/intent,  internal/surface/cli\n" +
		"- tests: the candidate filter over a fixture store;\n  the score on fixtures\n\n## Steps\n\n1. One\n"
	fp := ReadFootprint(written)
	if !fp.Present {
		t.Fatal("the section is present")
	}
	if want := []string{"internal/core/implement", "internal/core/intent", "internal/surface/cli"}; !slices.Equal(fp.Packages, want) {
		t.Fatalf("packages = %q, want %q", fp.Packages, want)
	}
	if fp.Tests != "the candidate filter over a fixture store; the score on fixtures" {
		t.Fatalf("tests = %q", fp.Tests)
	}
	if got := ReadFootprint("# a\n\n## Summary\n\nText.\n"); got.Present {
		t.Fatalf("a spec without the section reads absent: %+v", got)
	}
	// A heading inside a fence is an example, not the section.
	if got := ReadFootprint("# a\n\n```md\n## Footprint\n\n- packages: x\n```\n"); got.Present {
		t.Fatalf("a fenced heading is not the section: %+v", got)
	}
}
