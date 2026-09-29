package spec

// footprint.go is the spec's `## Footprint` section (itd-2609211116005482,
// criteria 7 and 8): the packages the spec expects to touch and the tests
// that will show it works, written by the author before a lane starts. `abcd
// build next` reads it as two parts of an intent's readiness score, a test
// path and an expected footprint; a spec minted by `intent plan` carries the
// section empty for the author to fill.
//
//	## Footprint
//
//	- packages: internal/core/implement, internal/core/intent
//	- tests: the candidate filter over a fixture store
//
// Only the two keyed bullets are read, each once; any other line is the
// author's. A bullet's indented continuation lines are part of its value.

import (
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/mdrecord"
)

// FootprintHeading is the section a spec declares its footprint under.
const FootprintHeading = "## Footprint"

// footprintHeadingRe matches the heading as a line.
var footprintHeadingRe = regexp.MustCompile(`^## Footprint[ \t]*$`)

// footprintKeyRe is a keyed bullet of the section.
var footprintKeyRe = regexp.MustCompile(`^[-*+][ \t]+(?i:(packages|tests))[ \t]*:[ \t]*(.*)$`)

// footprintStubBody is the empty section a minted spec carries.
const footprintStubBody = "- packages:\n- tests:"

// Footprint is what a spec's `## Footprint` section declares.
type Footprint struct {
	// Present is false when the spec carries no `## Footprint` section.
	Present bool `json:"present"`
	// Packages are the packages the spec expects to touch, in the order
	// written; empty when the section names none.
	Packages []string `json:"packages"`
	// Tests is the tests text, one line; empty when the section names none.
	Tests string `json:"tests"`
}

// ReadFootprint reads the first live `## Footprint` section of a spec: a
// heading inside a fence or a comment is an example, not the section. A key
// given twice keeps its first value, the one a reader meets first.
func ReadFootprint(content string) Footprint {
	fp := Footprint{Packages: []string{}}
	lines := strings.Split(content, "\n")
	mask := mdrecord.Mask(lines)
	start, end, ok := mdrecord.SectionLineRangeIn(lines, mask, footprintHeadingRe)
	if !ok {
		return fp
	}
	fp.Present = true
	var gotPackages, gotTests bool
	for _, b := range mdrecord.BulletBlocks(lines, mask, start, end) {
		m := footprintKeyRe.FindStringSubmatch(strings.TrimRight(lines[b.Start], "\r"))
		if m == nil {
			continue
		}
		parts := []string{strings.TrimSpace(m[2])}
		for i := b.Start + 1; i < b.End; i++ {
			parts = append(parts, strings.TrimSpace(lines[i]))
		}
		value := strings.Join(strings.Fields(strings.Join(parts, " ")), " ")
		switch strings.ToLower(m[1]) {
		case "packages":
			if gotPackages {
				continue
			}
			gotPackages = true
			for _, p := range strings.Split(value, ",") {
				if p = strings.TrimSpace(p); p != "" {
					fp.Packages = append(fp.Packages, p)
				}
			}
		case "tests":
			if gotTests {
				continue
			}
			gotTests = true
			fp.Tests = value
		}
	}
	return fp
}
