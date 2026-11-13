package lint

import (
	"path/filepath"
	"reflect"
	"strconv"
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

// TestStripInlineCodeBlanksAsTheLintDidBefore holds two lines to literal
// text, so the sure lines' blanking does not rest only on a function as new
// as the change: a balanced line, and one whose first backtick is escaped.
func TestStripInlineCodeBlanksAsTheLintDidBefore(t *testing.T) {
	for _, c := range []struct{ line, want string }{
		{"a `x` b ``y`` c", "a     b       c"},
		{"a \\` b `c` d", "a \\` b     d"},
	} {
		if got := stripInlineCode([]string{c.line}, []bool{false})[0]; got != c.want {
			t.Errorf("stripInlineCode(%q) = %q, want %q", c.line, got, c.want)
		}
		if got := stripInlineCodeLine(c.line); got != c.want {
			t.Errorf("stripInlineCodeLine(%q) = %q, want %q", c.line, got, c.want)
		}
	}
}

// A use that can make a finding go away reads each line alone, on every line,
// as the lint read it before (dcn-2611130921444350): below a backtick left
// open, a marker or definition the line alone shows in a span, which the page
// may show as code, quiets nothing.

// TestCitationFootnotesReportADefinitionQuotedOnlyBelowARunLeftOpen: a
// definition whose only marker sits in a one-line span below a backtick left
// open, on a paragraph line or a heading line, is never referenced.
func TestCitationFootnotesReportADefinitionQuotedOnlyBelowARunLeftOpen(t *testing.T) {
	for name, c := range map[string]struct {
		body string
		line int
	}{
		"a paragraph line": {"# C\n\nIntro `a\nSee `[^a]` here.\n\n[^a]: Source A, https://example.org/a.\n", 6},
		"a heading line":   {"# C `a\nSee `[^a]` here.\n\n[^a]: Source A, https://example.org/a.\n", 4},
	} {
		root := citeRepo(t, map[string]string{"docs/c.md": c.body})
		findings, err := LintAt(citeConfig("citation_footnotes", RuleConfig{}), root, fixedNow)
		if err != nil {
			t.Fatalf("%s: LintAt: %v", name, err)
		}
		got := findingsFor(findings, "citation_footnotes")
		if len(got) != 1 || got[0].Line != c.line || !strings.Contains(got[0].Message, "[^a] is never referenced") {
			t.Errorf("%s: want one never-referenced finding for [^a] on line %d: %+v", name, c.line, got)
		}
	}
}

// TestCitationFootnotesDefineAsTheLineAloneReads: below a backtick left open,
// only a definition the line alone reads defines its marker, as before. The
// whole reading can find a definition the line alone lacks, since a footnote
// id may hold a backtick: "[^a`b]: x` y" read alone is "[^a" and a blanked
// span, read whole it defines [^a`b]. That definition defines nothing, so the
// marker is still reported.
func TestCitationFootnotesDefineAsTheLineAloneReads(t *testing.T) {
	for name, c := range map[string]struct {
		body string
		id   string
		want int // the line of the one finding, or 0 for none
	}{
		// The definition pattern opens at the line's start, and this line
		// starts with a backtick, so neither reading finds a definition: the
		// case holds main's answer and passes on every head, the lane's
		// included. It pins that answer, not a change.
		"an opener in a span":         {"# C\n\nA claim.[^b]\n\nIntro `a\n`[^b]: x` y\n", "b", 3},
		"a definition read whole":     {"# C\n\nA claim.[^a`b]\n\nIntro `z\n[^a`b]: x` y\n", "a`b", 3},
		"a definition after a span":   {"# C\n\nA claim.[^b]\n\nIntro `a\n`x`[^b]: Source B, https://example.org/b.\n", "b", 0},
		"a definition on a sure line": {"# C\n\nA claim.[^b]\n\n`x`[^b]: Source B, https://example.org/b.\n", "b", 0},
	} {
		root := citeRepo(t, map[string]string{"docs/c.md": c.body})
		findings, err := LintAt(citeConfig("citation_footnotes", RuleConfig{}), root, fixedNow)
		if err != nil {
			t.Fatalf("%s: LintAt: %v", name, err)
		}
		got := findingsFor(findings, "citation_footnotes")
		switch {
		case c.want == 0 && len(got) != 0:
			t.Errorf("%s: want no finding: %+v", name, got)
		case c.want != 0 && (len(got) != 1 || got[0].Line != c.want || !strings.Contains(got[0].Message, "[^"+c.id+"] has no definition")):
			t.Errorf("%s: want one no-definition finding for [^%s] on line %d: %+v", name, c.id, c.want, got)
		}
	}
}

// A use that can report a finding reports what either reading finds: a
// pattern matched over the whole line can take in what the line alone reads
// as a link or a marker of its own, so the whole reading by itself would lose
// it.

// TestLinksReportWhatTheLineAloneFindsBelowARunLeftOpen: a dead link and a
// fragment naming no heading, each after a span whose text opens a link, are
// reported. The whole reading's longer match names the same written link
// again under a longer target, so line 4 carries two findings for one dead
// link; the count is pinned so a change to it is seen.
func TestLinksReportWhatTheLineAloneFindsBelowARunLeftOpen(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/a.md", "# A\n\nIntro `a\nSee `[y](` and [t](missing.md)` z\nSee `[y](` and [t](#nowhere)` z\n")
	cfg := Config{Roots: []string{"rec"}, Rules: map[string]RuleConfig{
		"links_resolve": {Enabled: true, Severity: "blocker"},
		"link_anchors":  {Enabled: true, Severity: "warn"},
	}}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	var dead, anchor bool
	for _, f := range fs {
		dead = dead || f.RuleID == "links_resolve" && f.Line == 4 && strings.HasSuffix(f.Message, "does not resolve: missing.md")
		anchor = anchor || f.RuleID == "link_anchors" && f.Line == 5 && strings.HasSuffix(f.Message, ": #nowhere")
	}
	if !dead || !anchor {
		t.Fatalf("want missing.md on line 4 and #nowhere on line 5 reported: %+v", fs)
	}
	count := map[string]int{}
	for _, f := range fs {
		count[f.RuleID+":"+strconv.Itoa(f.Line)]++
	}
	want := map[string]int{"links_resolve:4": 2, "links_resolve:5": 1, "link_anchors:5": 1}
	if !reflect.DeepEqual(count, want) {
		t.Fatalf("findings by rule and line = %v, want %v: %+v", count, want, fs)
	}
}

// TestCitationFootnotesReportWhatTheLineAloneFindsBelowARunLeftOpen: a marker
// after a span whose text opens one is reported as the line alone reads it.
func TestCitationFootnotesReportWhatTheLineAloneFindsBelowARunLeftOpen(t *testing.T) {
	root := citeRepo(t, map[string]string{"docs/c.md": "# C\n\nIntro `a\nSee `[^`b[^gone].\n"})
	findings, err := LintAt(citeConfig("citation_footnotes", RuleConfig{}), root, fixedNow)
	if err != nil {
		t.Fatalf("LintAt: %v", err)
	}
	var found bool
	for _, f := range findingsFor(findings, "citation_footnotes") {
		found = found || f.Line == 4 && strings.Contains(f.Message, "footnote marker [^gone] has no definition")
	}
	if !found {
		t.Fatalf("want [^gone] on line 4 reported: %+v", findings)
	}
}

// TestForbiddenSynonymsReportWhatEitherReadingFindsBelowARunLeftOpen: on a line
// below a backtick left open, a word only the line alone finds and a word only
// the whole reading finds are both reported. The line alone differs from the
// line as written only where a span is blanked, so the word only it finds is
// one written with the run of spaces a blanked span leaves; the word only the
// whole reading finds is the one in the span.
func TestForbiddenSynonymsReportWhatEitherReadingFindsBelowARunLeftOpen(t *testing.T) {
	root := t.TempDir()
	glossaryTerm(t, root, "spec", []string{"user   story", "epic"})
	writeFile(t, root, "rec/x.md", "# t\n\nIntro `a\na user` `story and `epic` b\n")
	fs, err := Lint(fsCfg([]string{"user   story", "epic"}, nil, nil), root)
	if err != nil {
		t.Fatal(err)
	}
	var spaced, quoted bool
	for _, f := range fs {
		if f.RuleID != "GL002" || f.Line != 4 {
			t.Errorf("unexpected finding: %+v", f)
			continue
		}
		spaced = spaced || strings.Contains(f.Message, "'user   story'")
		quoted = quoted || strings.Contains(f.Message, "'epic'")
	}
	if len(fs) != 2 || !spaced || !quoted {
		t.Fatalf("want two GL002 findings on line 4, the spaced word and the quoted one: %+v", fs)
	}
}

// TestCitationURLSyntaxReadsALineBelowABacktickLeftOpen: a cited URL in a
// one-line span on a definition line below a backtick left open is checked.
func TestCitationURLSyntaxReadsALineBelowABacktickLeftOpen(t *testing.T) {
	root := citeRepo(t, map[string]string{"docs/c.md": "# C\n\nA claim.[^u]\n\nIntro `a\n[^u]: Source, `http://intranet/x` here.\n"})
	findings, err := LintAt(citeConfig("citation_url_syntax", RuleConfig{}), root, fixedNow)
	if err != nil {
		t.Fatalf("LintAt: %v", err)
	}
	got := findingsFor(findings, "citation_url_syntax")
	if len(got) != 1 || got[0].Line != 6 || !strings.Contains(got[0].Message, "http://intranet/x") {
		t.Fatalf("want one finding for http://intranet/x on line 6: %+v", got)
	}
}

// TestCollectCitedURLsReadsWholeAndMarksTheLineAlone: the collector lists a
// URL only the whole reading finds, for the refresh to check, and marks as
// read alone only the URL a line cites outside a span on that line.
func TestCollectCitedURLsReadsWholeAndMarksTheLineAlone(t *testing.T) {
	root := citeRepo(t, map[string]string{"docs/c.md": "# C\n\nA claim.[^u]\n\nIntro `a\n[^u]: Source, `https://example.org/q` and https://example.org/p.\n"})
	got, err := CollectCitedURLs(Config{Roots: []string{"docs"}}, root)
	if err != nil {
		t.Fatal(err)
	}
	alone := map[string]bool{}
	for _, c := range got {
		alone[c.URL] = c.ReadAlone
	}
	if len(got) != 2 || !alone["https://example.org/p"] || alone["https://example.org/q"] {
		t.Fatalf("want p read alone and q listed but not read alone: %+v", got)
	}
}
