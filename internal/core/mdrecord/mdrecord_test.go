package mdrecord

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// This package is a leaf two record families read and write their bodies
// through, and it was covered only transitively, through core/intent and
// core/grounds. A leaf tested only through its consumers is a leaf whose own
// rules nobody states: a consumer's test passes for the consumer's reasons, and
// the rule it happened to exercise can change under it silently. What follows
// asserts THIS package's rules directly.

var groundsRe = regexp.MustCompile(`^#{1,6}\s+Grounds\s*$`)

// lines is the shape every test here works in: a record body written as one
// string, split the way every caller splits it.
func lines(body string) []string { return strings.Split(body, "\n") }

// TestMaskCoversFencesAndCommentsIncludingTheirDelimiters: the delimiter lines
// are masked too, because nothing on them is live markdown either — a fence's
// opener carries its info string and a comment's opener carries `<!--`.
func TestMaskCoversFencesAndCommentsIncludingTheirDelimiters(t *testing.T) {
	body := "live\n" +
		"```go\n" +
		"## Grounds\n" +
		"```\n" +
		"live\n" +
		"<!-- parked\n" +
		"- pursued: a parked bullet\n" +
		"-->\n" +
		"live\n"
	mask := Mask(lines(body))
	want := []uint8{0, MaskFence, MaskFence, MaskFence, 0, MaskComment, MaskComment, MaskComment, 0, 0}
	if !reflect.DeepEqual(mask, want) {
		t.Fatalf("Mask = %v, want %v", mask, want)
	}
}

// TestMaskDoesNotNestOneConstructInTheOther: inside a fence `<!--` is literal
// text and inside a comment a fence delimiter is, so neither opens a span the
// other is already holding.
func TestMaskDoesNotNestOneConstructInTheOther(t *testing.T) {
	fenced := "```\n<!--\n```\nlive\n"
	if m := Mask(lines(fenced)); m[3] != 0 {
		t.Fatalf("a `<!--` inside a fence opened a comment span: %v", m)
	}
	commented := "<!--\n```\n-->\nlive\n"
	if m := Mask(lines(commented)); m[3] != 0 {
		t.Fatalf("a fence delimiter inside a comment opened a fence span: %v", m)
	}
}

// TestMaskRunsAnUnclosedOpenerToEndOfFile is CommonMark's rule for a fence and
// the only safe reading of a comment nobody closed. It is also the whole reason
// an appending writer needs Unclosed: everything below the opener stops being
// live markdown, so a line appended at the end can never read back.
func TestMaskRunsAnUnclosedOpenerToEndOfFile(t *testing.T) {
	for name, body := range map[string]string{
		"fence":   "live\n```go\nbelow\nfurther below\n",
		"comment": "live\n<!-- parked\nbelow\nfurther below\n",
	} {
		mask := Mask(lines(body))
		for i := 1; i < len(mask); i++ {
			if mask[i] == 0 {
				t.Fatalf("%s: line %d is live below an unclosed opener: %v", name, i, mask)
			}
		}
	}
}

// TestUnclosedNamesTheOpenerAndItsConstruct: Mask has to work out which line
// opened the span it is still inside, and a refusal that cannot name that line
// describes the symptom instead of the cause.
func TestUnclosedNamesTheOpenerAndItsConstruct(t *testing.T) {
	for name, tc := range map[string]struct {
		body string
		line int
		flag uint8
	}{
		"fence":            {"a\nb\n```go\nc\n", 2, MaskFence},
		"comment":          {"a\n<!-- parked\nc\n", 1, MaskComment},
		"reopened comment": {"a\n<!-- x --> y <!-- z\nc\n", 1, MaskComment},
	} {
		line, flag, ok := Unclosed(lines(tc.body))
		if !ok || line != tc.line || flag != tc.flag {
			t.Fatalf("%s: Unclosed = (%d, %d, %v), want (%d, %d, true)", name, line, flag, ok, tc.line, tc.flag)
		}
	}
	for name, body := range map[string]string{
		"nothing opened":  "a\nb\nc\n",
		"closed fence":    "```go\nx\n```\nlive\n",
		"closed comment":  "<!-- x -->\nlive\n",
		"reclosed second": "<!-- a\n-->\n```\nx\n```\n",
	} {
		if line, _, ok := Unclosed(lines(body)); ok {
			t.Fatalf("%s: Unclosed reports line %d open, want nothing open", name, line)
		}
	}
}

// TestOpensCommentLetsTheFirstConstructWin is CommonMark's precedence, and it
// diverged in BOTH directions when code spans were resolved first: a backtick
// inside a live comment re-paired the rest of the line, and a comment opener
// quoted in backticks read as live.
func TestOpensCommentLetsTheFirstConstructWin(t *testing.T) {
	for name, tc := range map[string]struct {
		line string
		want bool
	}{
		"plain opener":            {"<!-- parked", true},
		"opener closed":           {"<!-- parked -->", false},
		"opener quoted in a span": {"the `<!--` marker scan", false},
		"backtick inside comment": {"<!-- a ` b -->", false},
		"unmatched backtick run":  {"a ` b <!-- c", true},
		"reopened after close":    {"<!-- a --> b <!-- c", true},
		"no construct":            {"an ordinary sentence", false},
	} {
		if got := OpensComment(tc.line); got != tc.want {
			t.Fatalf("%s: OpensComment(%q) = %v, want %v", name, tc.line, got, tc.want)
		}
	}
}

// TestCodeSpanRangesPairsRunsOfEqualLength: a run closes on a run of the SAME
// length, and an unmatched run is literal backticks rather than an opener that
// swallows the rest of the line.
func TestCodeSpanRangesPairsRunsOfEqualLength(t *testing.T) {
	for name, tc := range map[string]struct {
		line string
		want [][2]int
	}{
		"one span":     {"a `b` c", [][2]int{{2, 5}}},
		"two spans":    {"`a` `b`", [][2]int{{0, 3}, {4, 7}}},
		"double run":   {"a ``b`c`` d", [][2]int{{2, 9}}},
		"unmatched":    {"a ` b", nil},
		"length mixed": {"a ``b` c", nil},
		"none":         {"plain prose", nil},
	} {
		if got := CodeSpanRanges(tc.line); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%s: CodeSpanRanges(%q) = %v, want %v", name, tc.line, got, tc.want)
		}
	}
}

// TestInAnyRangeIsHalfOpen: the end offset is past the span, so a position at it
// is outside.
func TestInAnyRangeIsHalfOpen(t *testing.T) {
	r := [][2]int{{2, 5}}
	for pos, want := range map[int]bool{1: false, 2: true, 4: true, 5: false} {
		if got := InAnyRange(r, pos); got != want {
			t.Fatalf("InAnyRange(%v, %d) = %v, want %v", r, pos, got, want)
		}
	}
}

// TestSectionLineRangeStopsAtTheNextLiveHeading, and a MASKED heading is neither
// the heading that opens a section nor the heading that closes one — a fenced or
// commented `## Grounds` is an example somebody wrote, not the record's own
// section.
func TestSectionLineRangeStopsAtTheNextLiveHeading(t *testing.T) {
	body := "# Title\n" +
		"\n" +
		"## Grounds\n" +
		"\n" +
		"- pursued: the first entry\n" +
		"\n" +
		"## Notes\n" +
		"\n" +
		"prose\n"
	start, end, ok := SectionLineRange(lines(body), groundsRe)
	if !ok || start != 3 || end != 6 {
		t.Fatalf("SectionLineRange = (%d, %d, %v), want (3, 6, true)", start, end, ok)
	}

	shadowed := "```\n## Grounds\n```\n\n## Grounds\n\n- pursued: the live entry\n"
	start, _, ok = SectionLineRange(lines(shadowed), groundsRe)
	if !ok || start != 5 {
		t.Fatalf("a fenced heading was read as the section: start = %d (ok=%v), want 5", start, ok)
	}

	if _, _, ok := SectionLineRange(lines("# Title\n\nprose\n"), groundsRe); ok {
		t.Fatal("an absent section reported as present, which is what separates it from an empty one")
	}
	// An EMPTY section still reports ok, which is what separates it from an
	// absent one. Its bounds run to the end of the input — here the one line the
	// split leaves past the final newline.
	if start, end, ok := SectionLineRange(lines("## Grounds\n"), groundsRe); !ok || start != 1 || end != 2 {
		t.Fatalf("an EMPTY section = (%d, %d, %v), want (1, 2, true)", start, end, ok)
	}
}

// TestCountHeadingsCountsLiveOnes: a second live heading makes "the section"
// ambiguous, and a writer that stamps into it is guessing. A fenced one does not
// count, or every record carrying an example would read as ambiguous.
func TestCountHeadingsCountsLiveOnes(t *testing.T) {
	body := "## Grounds\n- pursued: a\n\n```\n## Grounds\n```\n\n## Grounds\n- pursued: b\n"
	ls := lines(body)
	if n := CountHeadings(ls, Mask(ls), groundsRe); n != 2 {
		t.Fatalf("CountHeadings = %d, want 2 (the fenced heading is not one)", n)
	}
}

// TestBulletBlocksFoldContinuationsAndStopAtAnyBullet: a wrapped bullet is one
// item, and an INDENTED sub-bullet ends its parent's text rather than being
// folded into it — so a reader that rejoins a block gets the line the grammar
// was written on and nothing else.
func TestBulletBlocksFoldContinuationsAndStopAtAnyBullet(t *testing.T) {
	body := "## Grounds\n" +
		"\n" +
		"- pursued: a conjecture that\n" +
		"  wraps onto a second line\n" +
		"  - a sub-bullet, which is detail\n" +
		"\n" +
		"- deferred: the second entry\n"
	ls := lines(body)
	mask := Mask(ls)
	start, end, ok := SectionLineRangeIn(ls, mask, groundsRe)
	if !ok {
		t.Fatal("no section")
	}
	got := BulletBlocks(ls, mask, start, end)
	want := []BulletBlock{{Start: 2, End: 4}, {Start: 6, End: 7}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BulletBlocks = %+v, want %+v", got, want)
	}
}

// TestBulletBlocksSkipMaskedLines: a bullet inside a fence or a comment is an
// example or a parked block, and counting it would let a record's own example
// vote.
func TestBulletBlocksSkipMaskedLines(t *testing.T) {
	body := "## Grounds\n\n```\n- pursued: an example\n```\n\n<!--\n- pursued: parked\n-->\n\n- pursued: the live one\n"
	ls := lines(body)
	mask := Mask(ls)
	start, end, _ := SectionLineRangeIn(ls, mask, groundsRe)
	got := BulletBlocks(ls, mask, start, end)
	if len(got) != 1 || got[0].Start != 10 {
		t.Fatalf("BulletBlocks = %+v, want exactly the live bullet at line 10", got)
	}
}

// TestBulletPredicatesJudgeTheLineAlone: whether the line is LIVE is Mask's
// answer, not theirs, and a top-level bullet is a COLUMN-0 one because a
// record's bullets are counted positionally.
func TestBulletPredicatesJudgeTheLineAlone(t *testing.T) {
	for line, want := range map[string]bool{
		"- an item":     true,
		"* an item":     true,
		"  - an item":   false,
		"-no space":     false,
		"- ":            false,
		"not a bullet":  false,
		"# not either":  false,
		"-   spaced ok": true,
	} {
		if got := IsTopLevelBullet(line); got != want {
			t.Fatalf("IsTopLevelBullet(%q) = %v, want %v", line, got, want)
		}
	}
	for line, want := range map[string]bool{
		"# Title":       true,
		"###### Deep":   true,
		"#NoSpace":      false,
		"####### Seven": false,
		"prose":         false,
	} {
		if got := IsHeading(line); got != want {
			t.Fatalf("IsHeading(%q) = %v, want %v", line, got, want)
		}
	}
	if got := TrimBulletPrefix("-   pursued: a conjecture"); got != "pursued: a conjecture" {
		t.Fatalf("TrimBulletPrefix = %q", got)
	}
}

// TestAnyMaskedReportsOneFlagOverARange: the flags are reported separately so a
// refusal names the construct the reader has to go and look at.
func TestAnyMaskedReportsOneFlagOverARange(t *testing.T) {
	ls := lines("live\n```\nx\n```\n<!--\ny\n-->\nlive\n")
	mask := Mask(ls)
	if !AnyMasked(mask, 0, len(ls), MaskFence) || !AnyMasked(mask, 0, len(ls), MaskComment) {
		t.Fatalf("both constructs are present but AnyMasked missed one: %v", mask)
	}
	if AnyMasked(mask, 4, 7, MaskFence) {
		t.Fatalf("the comment lines reported as fenced: %v", mask)
	}
	if AnyMasked(mask, 0, 1, MaskFence|MaskComment) {
		t.Fatalf("a live line reported as masked: %v", mask)
	}
	// Out-of-range indices are answered rather than panicked on: callers pass
	// ranges derived from a section, and a stale one must not take the process
	// down.
	if AnyMasked(mask, len(ls), len(ls)+9, MaskFence) {
		t.Fatal("AnyMasked past the end reported a flag")
	}
}

// TestMaskLeavesASelfClosingCommentLive is the boundary of what Mask claims. It
// masks the lines a span is OPEN across, so a comment that opens and closes on
// one line is not masked at all — and that is safe rather than a gap, because
// every construct this package recognises must begin its line (a heading and a
// bullet at column 0, a fence at up to three spaces of indent) and a `<!--`
// occupying the line start leaves no room for one.
func TestMaskLeavesASelfClosingCommentLive(t *testing.T) {
	ls := lines("<!-- ## Grounds -->\nlive\n")
	if m := Mask(ls); m[0] != 0 {
		t.Fatalf("a self-closing comment line reported as masked: %v", m)
	}
	if _, _, ok := SectionLineRange(ls, groundsRe); ok {
		t.Fatal("a heading commented out on one line was read as a section heading")
	}
}

// TestPeelTrailingLinkRefsReturnsTheRunAndTrimsBothSides: a `[ref]: url`
// definition parked at the end of a section belongs BELOW the section's prose,
// so an appending writer takes the run off, appends, and puts it back.
func TestPeelTrailingLinkRefsReturnsTheRunAndTrimsBothSides(t *testing.T) {
	section := []string{"- pursued: an entry", "", "", "[iss-80]: ../open/iss-80.md", ""}
	refs := PeelTrailingLinkRefs(&section)
	if !reflect.DeepEqual(section, []string{"- pursued: an entry"}) {
		t.Fatalf("section after the peel = %q", section)
	}
	if !reflect.DeepEqual(refs, []string{"[iss-80]: ../open/iss-80.md", ""}) {
		t.Fatalf("refs = %q", refs)
	}

	// A tail of pure blank lines is not refs, and is left to the caller's own
	// blank-trimming rather than swallowed here.
	blanks := []string{"- pursued: an entry", "", ""}
	if got := PeelTrailingLinkRefs(&blanks); got != nil || len(blanks) != 3 {
		t.Fatalf("a blank tail was peeled as refs: got %q, section %q", got, blanks)
	}
	// A definition that is not at the tail stays where it is.
	mid := []string{"[a]: x", "- pursued: an entry"}
	if got := PeelTrailingLinkRefs(&mid); got != nil || len(mid) != 2 {
		t.Fatalf("a mid-section definition was peeled: got %q, section %q", got, mid)
	}
}

// pathologicalRuns builds a line of k backtick runs of DISTINCT lengths, so no
// run has a closer of its own length. It is the shape a per-run rescan to end of
// line is quadratic on.
func pathologicalRuns(k int) string {
	var b strings.Builder
	for n := 1; n <= k; n++ {
		b.WriteString(strings.Repeat("`", n))
		b.WriteString("x <!- y ")
	}
	return b.String()
}

func BenchmarkOpensCommentDistinctRuns(b *testing.B) {
	ln := pathologicalRuns(120)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if OpensComment(ln) {
			b.Fatal("the fixture must not open a comment")
		}
	}
}

func BenchmarkCodeSpanRangesDistinctRuns(b *testing.B) {
	ln := pathologicalRuns(120)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if r := CodeSpanRanges(ln); len(r) != 0 {
			b.Fatalf("the fixture must pair no spans, got %v", r)
		}
	}
}

// typicalLine is what record prose actually looks like: one or two code spans
// and no pathology. It is here so the run-list allocation the benchmarks above
// pay for is measured on the case that is common rather than only on the case
// that is bad.
const typicalLine = "The reader asks `ParseSectionAboveFloor`, which the intent half also asks, and the writer appends one bullet."

func BenchmarkOpensCommentTypicalLine(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if OpensComment(typicalLine) {
			b.Fatal("the fixture must not open a comment")
		}
	}
}

// TestReadListNestedAdmitsAFenceIndentedUnderAListItem is the one extension the
// tree takes to CommonMark's top-level fence rule. A fence written inside a list
// item sits at the item's content indent, which may be four columns or more, and
// a reader that saw only the left margin reads the fence's own `#` lines as
// headings. TopLevel keeps CommonMark's 0-3 space rule, under which the same run
// is indented code and not a delimiter.
func TestReadListNestedAdmitsAFenceIndentedUnderAListItem(t *testing.T) {
	ls := lines("- item\n\n    ```sh\n    # a shell comment\n    ```\n\n# Real\n")
	nested := Read(ls, ListNested).Mask
	for i := 2; i <= 4; i++ {
		if nested[i]&MaskFence == 0 {
			t.Fatalf("ListNested: line %d of the list-item fence is live: %v", i, nested)
		}
	}
	if nested[6] != 0 {
		t.Fatalf("ListNested: the heading after the fence is masked: %v", nested)
	}
	if top := Read(ls, TopLevel).Mask; top[2] != 0 || top[3] != 0 {
		t.Fatalf("TopLevel: a run indented four columns was read as a fence: %v", top)
	}
}

// TestReadFollowsCommonMarkOnRunLengthCharacterAndInfoString holds under both
// rules: a closer is a run of the opener's character at least as long, with
// nothing after it. A shorter run, the other character, or a run carrying an
// info string is content.
func TestReadFollowsCommonMarkOnRunLengthCharacterAndInfoString(t *testing.T) {
	for _, rule := range []Rule{TopLevel, ListNested} {
		for name, tc := range map[string]struct {
			body string
			live int // a line index that must be live
			dark int // a line index that must be fenced
		}{
			"four-backtick fence quoting a three-backtick line": {"````\n```\n# quoted\n```\n````\n# Real\n", 5, 2},
			"tilde fence holding a backtick line":               {"~~~\n```go\n~~~\nlive\n", 3, 1},
			"backtick fence holding a tilde line":               {"```\n~~~\n```\nlive\n", 3, 1},
			"indented list-item fence closes at its own indent": {"- a\n  ```\n  # x\n  ```\nlive\n", 4, 2},
		} {
			m := Read(lines(tc.body), rule).Mask
			if m[tc.live] != 0 || m[tc.dark]&MaskFence == 0 {
				t.Fatalf("rule %d, %s: mask %v, want line %d live and line %d fenced", rule, name, m, tc.live, tc.dark)
			}
		}
		if line, flag, ok := Read(lines("```\nx\n```go\nstill inside\n"), rule).Unclosed(); !ok || line != 0 || flag != MaskFence {
			t.Fatalf("rule %d: a closer carrying an info string closed the fence: (%d, %d, %v)", rule, line, flag, ok)
		}
	}
}

// TestReadListNestedEndsAFenceWithItsListItem: a fenced block cannot continue
// lazily, so a non-blank line at column 0 ends the list item holding the fence
// and the fence with it (CommonMark 5.2).
func TestReadListNestedEndsAFenceWithItsListItem(t *testing.T) {
	ls := lines("- item\n\n    ```\n    code\n# Heading\nbody\n")
	nested := Read(ls, ListNested)
	if nested.Mask[4] != 0 {
		t.Fatalf("ListNested: the column-0 heading after the item is masked: %v", nested.Mask)
	}
	if _, _, ok := nested.Unclosed(); ok {
		t.Fatal("ListNested: a fence its list item ended is reported unclosed")
	}
	if want := []Span{{Start: 2, End: 4, Closed: true}}; !reflect.DeepEqual(nested.Fences, want) {
		t.Fatalf("ListNested fences = %+v, want %+v", nested.Fences, want)
	}
	// A run indented four or more past the opener is content, not a closer.
	deep := Read(lines("- a\n\n    ```\n        ```\n    x\n    ```\nlive\n"), ListNested)
	if want := []Span{{Start: 2, End: 6, Closed: true}}; !reflect.DeepEqual(deep.Fences, want) {
		t.Fatalf("ListNested: a run indented past the closer's reach closed the fence: %+v", deep.Fences)
	}
}

// TestReadListNestedReadsAShallowRunAsTopLevelDoes: whether a run at one to
// three columns sits in a list item turns on the list around it, which this
// package does not parse. Both exported rules take the top-level reading —
// the one a committed record depends on, whose indented closer is followed by a
// column-0 line inside the same fence — and the list-item reading is left to
// FencedUnderEveryRule.
func TestReadListNestedReadsAShallowRunAsTopLevelDoes(t *testing.T) {
	ls := lines("```\n---\na: |\n  ```\nkey: v\n  ```\n---\n```\nlive\n")
	top, nested := Read(ls, TopLevel), Read(ls, ListNested)
	if !reflect.DeepEqual(top, nested) {
		t.Fatalf("TopLevel = %+v\nListNested = %+v", top, nested)
	}
	if want := []Span{{0, 4, true}, {5, 8, true}}; !reflect.DeepEqual(top.Fences, want) {
		t.Fatalf("Fences = %+v, want %+v", top.Fences, want)
	}
}

// TestReadReportsEachFenceAsItsOwnSpan: two fences back to back are two blocks,
// which a per-line mask cannot say — the renderer needs to know where one
// fence's closer ends it.
func TestReadReportsEachFenceAsItsOwnSpan(t *testing.T) {
	r := Read(lines("```\na\n```\n~~~\nb\n~~~\nlive\n```\nopen\n"), TopLevel)
	want := []Span{{0, 3, true}, {3, 6, true}, {7, 10, false}}
	if !reflect.DeepEqual(r.Fences, want) {
		t.Fatalf("Fences = %+v, want %+v", r.Fences, want)
	}
}

// TestMaskAndUnclosedAreTheTopLevelReading: the existing entry points are the
// TopLevel rule, so every reader that already calls them keeps its answer.
func TestMaskAndUnclosedAreTheTopLevelReading(t *testing.T) {
	ls := lines("- a\n    ```\n    x\n<!-- c\n```\n")
	r := Read(ls, TopLevel)
	if !reflect.DeepEqual(Mask(ls), r.Mask) {
		t.Fatalf("Mask = %v, Read(TopLevel).Mask = %v", Mask(ls), r.Mask)
	}
	l1, f1, ok1 := Unclosed(ls)
	l2, f2, ok2 := r.Unclosed()
	if l1 != l2 || f1 != f2 || ok1 != ok2 {
		t.Fatalf("Unclosed = (%d, %d, %v), Read(TopLevel).Unclosed = (%d, %d, %v)", l1, f1, ok1, l2, f2, ok2)
	}
}

// TestFencedUnderEveryRuleMasksOnlyWhereTheRulesAgree: a line any reading reads
// as live is live — including the list-item reading of a shallow run, under
// which a column-0 line ends the item and its fence — and a comment is not a
// fence.
func TestFencedUnderEveryRuleMasksOnlyWhereTheRulesAgree(t *testing.T) {
	ls := lines("```\nx\n```\n- item\n  ```\n  code\n# Heading\n  ```\n<!--\nparked\n-->\n")
	got := FencedUnderEveryRule(ls)
	want := []bool{true, true, true, false, true, true, false, true, false, false, false, false}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FencedUnderEveryRule = %v, want %v", got, want)
	}
	if m := Read(ls, TopLevel).Mask; m[6]&MaskFence == 0 {
		t.Fatalf("TopLevel reads the heading as fenced, which is the disagreement under test: %v", m)
	}
}

// FirstContent is the one leading-comment locator: the first line, and the
// column in it, holding anything but blanks and HTML comments, comments read
// as Read reads them.
func TestFirstContentSkipsBlanksAndComments(t *testing.T) {
	for name, tc := range map[string]struct {
		lines     []string
		line, col int
	}{
		"no preamble":                {[]string{"---", "id: x"}, 0, 0},
		"blank lines":                {[]string{"", "  ", "---"}, 2, 0},
		"a one-line comment":         {[]string{"<!-- a -->", "---"}, 1, 0},
		"a multi-line comment":       {[]string{"<!--", "text", "-->", "---"}, 3, 0},
		"two comments on one line":   {[]string{"<!-- a --> <!-- b -->", "x"}, 1, 0},
		"content after a comment":    {[]string{"<!-- a --> ---", "x"}, 0, 11},
		"content after a closer":     {[]string{"<!--", "a --> ---"}, 1, 6},
		"a byte-order mark":          {[]string{"\ufeff<!-- a -->", "---"}, 1, 0},
		"a code span quoting a mark": {[]string{"`<!--` is an opener", "x"}, 0, 0},
		"nothing but comments":       {[]string{"<!-- a -->", ""}, 2, 0},
		"an unclosed comment":        {[]string{"<!-- a", "b"}, 2, 0},
	} {
		line, col := FirstContent(tc.lines)
		if line != tc.line || col != tc.col {
			t.Errorf("%s: FirstContent = (%d, %d), want (%d, %d)", name, line, col, tc.line, tc.col)
		}
	}
}
