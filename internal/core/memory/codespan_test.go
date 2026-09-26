package memory

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// TestRenderIndexCannotHaveItsCodeSpanBrokenByAPageName is
// iss-2609020539188868: termsafe's guarantees hold over the EXACT string
// CleanProse returned, and a renderer that adds its own delimiters is parsing a
// different string than the cleaner reasoned about.
//
// The cleaner shelters what sits inside the FIELD's own code spans — no raw HTML
// is parsed there — and escapes an unpaired backtick run so two cleaned fields on
// one line cannot re-pair. Wrapping the field in the renderer's own single
// backticks defeats the first half: with `a`<script>`b` as the page name, the
// render's opening backtick pairs with the field's first one, so the boundaries
// move and the `<script>` the cleaner sheltered is live prose again.
func TestRenderIndexCannotHaveItsCodeSpanBrokenByAPageName(t *testing.T) {
	const payload = "<script>"
	name := "topic_a`" + payload + "`b.md"
	out := RenderIndex([]PageInfo{{Filename: name, Domain: "d", Summary: "s"}})

	line := ""
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "- ") {
			line = l
			break
		}
	}
	if line == "" {
		t.Fatalf("no index row rendered:\n%s", out)
	}
	if !insideOneCodeSpan(line, payload) {
		t.Errorf("the page name's sheltered content escaped the render's code span, so raw HTML is live in the committed index:\n%s", line)
	}
}

// TestRenderContradictionsCannotHaveItsCodeSpansBroken is the same defect on the
// other committed derived page, where TWO page names share a line.
func TestRenderContradictionsCannotHaveItsCodeSpansBroken(t *testing.T) {
	const payload = "<script>"
	out := RenderContradictions([]PageInfo{{
		Filename:    "topic_a`" + payload + "`b.md",
		Contradicts: []string{"topic_c`" + payload + "`d.md"},
	}})
	line := ""
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "- ") {
			line = l
			break
		}
	}
	if line == "" {
		t.Fatalf("no contradictions row rendered:\n%s", out)
	}
	if strings.Count(line, payload) != 2 {
		t.Fatalf("expected both names rendered:\n%s", line)
	}
	if !insideOneCodeSpan(line, payload) {
		t.Errorf("a page name's sheltered content escaped its code span in the committed contradictions page:\n%s", line)
	}
}

// insideOneCodeSpan reports whether EVERY occurrence of needle in line sits
// inside a CommonMark code span. It walks the line's backtick runs by the spec's
// own rule — a run of N opens a span closed by the next run of exactly N — which
// is the rule the cleaner's own invariant is stated against.
func insideOneCodeSpan(line, needle string) bool {
	spans := codeSpanRanges(line)
	for from := 0; ; {
		i := strings.Index(line[from:], needle)
		if i < 0 {
			return true
		}
		at := from + i
		covered := false
		for _, s := range spans {
			if at >= s[0] && at+len(needle) <= s[1] {
				covered = true
				break
			}
		}
		if !covered {
			return false
		}
		from = at + len(needle)
	}
}

// codeSpanRanges returns the [start,end) byte ranges of each code span's CONTENT.
func codeSpanRanges(s string) [][2]int {
	var out [][2]int
	i := 0
	for i < len(s) {
		if s[i] != '`' {
			i++
			continue
		}
		n := 0
		for i+n < len(s) && s[i+n] == '`' {
			n++
		}
		start := i + n
		j := start
		for j < len(s) {
			if s[j] != '`' {
				j++
				continue
			}
			m := 0
			for j+m < len(s) && s[j+m] == '`' {
				m++
			}
			if m == n {
				out = append(out, [2]int{start, j})
				i = j + m
				break
			}
			j += m
		}
		if j >= len(s) {
			// An unclosed run is literal backticks; nothing opens here.
			i = start
		}
	}
	return out
}

// TestCodeSpanRoundTripsEveryShape is the primitive's own corpus, both sides:
// the value's bytes survive verbatim inside the span, and the span cannot be
// broken by whatever the value carries.
func TestCodeSpanRoundTripsEveryShape(t *testing.T) {
	for _, v := range []string{
		"plain.md",
		"a`b",
		"a``b",
		"`leading",
		"trailing`",
		"``",
		" padded ",
		"a`b``c```d",
	} {
		got := termsafe.CodeSpan(v)
		if v == "" {
			// An empty value has no span to draw; CommonMark cannot express one.
			if got != "" {
				t.Errorf("CodeSpan(\"\") = %q, want \"\" — a wrapper must not invent a span around nothing", got)
			}
			continue
		}
		spans := codeSpanRanges(got)
		if len(spans) != 1 {
			t.Errorf("CodeSpan(%q) = %q parses as %d code spans, want exactly 1", v, got, len(spans))
			continue
		}
		content := got[spans[0][0]:spans[0][1]]
		// CommonMark strips one space from each end when the content begins AND
		// ends with a space and is not all spaces.
		if strings.HasPrefix(content, " ") && strings.HasSuffix(content, " ") && strings.TrimSpace(content) != "" {
			content = content[1 : len(content)-1]
		}
		if content != v {
			t.Errorf("CodeSpan(%q) = %q renders content %q; a wrapper must not alter the value's bytes", v, got, content)
		}
	}
}

// TestRenderCitedMatchesCleansEveryFieldAndOwnsNoDelimiter is the ask half of
// iss-2609020539188868: every untrusted field on the answer's markdown lines
// goes through CleanProse, not Sanitize alone (which leaves HTML openers and
// link syntax live), and the filename's code span is CodeSpan's.
func TestRenderCitedMatchesCleansEveryFieldAndOwnsNoDelimiter(t *testing.T) {
	const payload = "<script>"
	out := RenderCitedMatches("what [x](http://example.com)?", []MatchedPage{{
		Filename: "topic_a`" + payload + "`b.md",
		Score:    1,
		Summary:  "see [here](http://example.com) " + payload,
	}})
	var row string
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "- ") {
			row = l
		}
	}
	if row == "" {
		t.Fatalf("no match row rendered:\n%s", out)
	}
	if !insideOneCodeSpan(row[:strings.Index(row, " (score")], payload) {
		t.Errorf("the filename's sheltered content escaped its code span:\n%s", row)
	}
	if strings.Contains(out, "](http://example.com)") {
		t.Errorf("link syntax from an untrusted field survived into the answer:\n%s", out)
	}
	if tail := row[strings.Index(row, " — "):]; strings.Contains(tail, payload) {
		t.Errorf("raw HTML from the summary is live in the answer:\n%s", row)
	}
}
