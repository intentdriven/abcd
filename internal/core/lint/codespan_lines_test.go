package lint

import (
	"path/filepath"
	"strings"
	"testing"
)

// The lint's prose checks pair inline code spans on one line, as the page
// readers do, and where a line above leaves a backtick open, to the next blank
// line, they blank no span and read the line whole (iss-2611130850006749). The
// fixture's first line leaves a backtick open; the page pairs it with the first
// backtick on the line below, so the text after that backtick is prose there,
// while the second line read alone pairs its two backticks and blanks it.
const openThenClosed = "Intro `a\nx` "

// TestLinksResolveReadsALineBelowABacktickLeftOpen: a dead link the page shows
// as prose, on a line below a backtick left open, is reported; past a blank
// line the same line is blanked as a line read alone.
func TestLinksResolveReadsALineBelowABacktickLeftOpen(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/doc.md", openThenClosed+"[t](missing.md) `y`\n\nIntro `a\n\nx` [t](quoted.md) `y`\n")
	cfg := Config{Roots: []string{"rec"}, Rules: map[string]RuleConfig{"links_resolve": {Enabled: true, Severity: "blocker"}}}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "links_resolve"); n != 1 || !hasFinding(fs, filepath.Join("rec", "doc.md"), "links_resolve", 2) {
		t.Fatalf("want one links_resolve finding, on line 2: %+v", fs)
	}
}

// TestLinkAnchorsReadsALineBelowABacktickLeftOpen: the same for a fragment
// naming no heading.
func TestLinkAnchorsReadsALineBelowABacktickLeftOpen(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/a.md", "# A\n\n"+openThenClosed+"[t](#nowhere) `y`\n")
	cfg := Config{Roots: []string{"rec"}, Rules: map[string]RuleConfig{"link_anchors": {Enabled: true, Severity: "warn"}}}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "link_anchors"); n != 1 || !hasFinding(fs, filepath.Join("rec", "a.md"), "link_anchors", 4) {
		t.Fatalf("want one link_anchors finding, on line 4: %+v", fs)
	}
}

// TestCitationFootnotesReadALineBelowABacktickLeftOpen: a footnote marker the
// page shows as prose, with no definition, is reported.
func TestCitationFootnotesReadALineBelowABacktickLeftOpen(t *testing.T) {
	root := citeRepo(t, map[string]string{"docs/c.md": "# C\n\n" + openThenClosed + "a claim.[^gone] `y`\n"})
	findings, err := LintAt(citeConfig("citation_footnotes", RuleConfig{}), root, fixedNow)
	if err != nil {
		t.Fatalf("LintAt: %v", err)
	}
	got := findingsFor(findings, "citation_footnotes")
	if len(got) != 1 || got[0].Line != 4 || !strings.Contains(got[0].Message, "[^gone]") {
		t.Fatalf("want one finding for [^gone] on line 4: %+v", got)
	}
}

// TestForbiddenSynonymsReadALineBelowABacktickLeftOpen: an enforced synonym
// the page shows as prose is reported; one in a span on a line whose stretch
// leaves nothing open is not.
func TestForbiddenSynonymsReadALineBelowABacktickLeftOpen(t *testing.T) {
	root := t.TempDir()
	glossaryTerm(t, root, "spec", []string{"epic"})
	writeFile(t, root, "rec/x.md", "# t\n\n"+openThenClosed+"an epic `y` here\n\nA `b` line.\nThe `epic` is quoted.\n")
	fs, err := Lint(fsCfg([]string{"epic"}, nil, nil), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "GL002"); n != 1 || !hasFinding(fs, filepath.Join("rec", "x.md"), "GL002", 4) {
		t.Fatalf("want one GL002 finding, on line 4: %+v", fs)
	}
}

// TestCitationCrosswalkRowsReadEachRowAlone: the crosswalk rule reports a row
// with no footnote, so reading a marker the page shows as code is what would
// let an uncited row through; a table row is its own line on the page, and the
// rule pairs each row's spans alone, below a backtick left open as anywhere.
func TestCitationCrosswalkRowsReadEachRowAlone(t *testing.T) {
	root := citeRepo(t, map[string]string{
		"docs/cw.md": "# Crosswalk\nIntro `a\n" +
			"| Term | Meaning |\n|---|---|\n" +
			"| **A** | Quoted `[^a]` only. |\n" +
			"| **B** | Cited.[^a] |\n\n" +
			"[^a]: Source A, https://example.org/a.\n",
	})
	findings, err := LintAt(citeConfig("citation_crosswalk_rows", RuleConfig{}), root, fixedNow)
	if err != nil {
		t.Fatalf("LintAt: %v", err)
	}
	got := findingsFor(findings, "citation_crosswalk_rows")
	if len(got) != 1 || got[0].Line != 5 {
		t.Fatalf("want one finding, the row **A** on line 5: %+v", got)
	}
}

// TestStripInlineCodeLinesKeepsSureLines: a balanced line after a balanced
// line, and the line a backtick is left open on, blank exactly as the line
// read alone does; a blank line ends the stretch below an open run, and so
// does a blank line inside a fence the mask skips.
func TestStripInlineCodeLinesKeepsSureLines(t *testing.T) {
	for _, c := range []struct {
		name   string
		lines  []string
		mask   []bool
		unsure []int
	}{
		{"balanced lines", []string{"a `x` b", "c ``y`` d `z`", "e `epic` f"}, nil, nil},
		{"the open run's own line", []string{"a `x` b `c", "d` e `f`"}, nil, []int{1}},
		{"to the next blank line", []string{"a `c", "d` e `f`", "g` h `i`", " \t", "j` k `l`"}, nil, []int{1, 2}},
		{"a blank line in a fence", []string{"a `c", "```", "", "```", "d` e `f`"}, []bool{false, true, true, true, false}, nil},
		{"a fence with no blank line", []string{"a `c", "```", "x", "```", "d` e `f`"}, []bool{false, true, true, true, false}, []int{4}},
	} {
		mask := c.mask
		if mask == nil {
			mask = make([]bool, len(c.lines))
		}
		got := stripInlineCode(c.lines, mask)
		unsure := map[int]bool{}
		for _, i := range c.unsure {
			unsure[i] = true
		}
		for i, ln := range c.lines {
			want := stripInlineCodeLine(ln)
			if unsure[i] {
				want = ln
			}
			if got[i] != want {
				t.Errorf("%s: line %d %q blanked to %q, want %q", c.name, i, ln, got[i], want)
			}
		}
	}
}
