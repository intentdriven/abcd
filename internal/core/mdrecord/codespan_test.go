package mdrecord

import (
	"reflect"
	"strings"
	"testing"
)

type spanCase struct {
	name string
	body string
	want [][][2]int // per line; nil for a line holding none
}

// checkSpans holds CodeSpans and ReadPage's Spans to the same want.
func checkSpans(t *testing.T, cases []spanCase) {
	t.Helper()
	norm := func(s [][][2]int) [][][2]int {
		for i := range s {
			if len(s[i]) == 0 {
				s[i] = nil
			}
		}
		return s
	}
	for _, c := range cases {
		body := lines(c.body)
		want := c.want
		for len(want) < len(body) {
			want = append(want, nil)
		}
		if got := norm(CodeSpans(body)); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: CodeSpans(%q) = %v, want %v", c.name, c.body, got, want)
		}
		if got := norm(ReadPage(body).Spans); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: ReadPage(%q).Spans = %v, want %v", c.name, c.body, got, want)
		}
	}
}

// TestCodeSpansPairOnOneLine: a run closes at the next run of its length on
// the same line, and a backtick behind an odd run of backslashes opens none.
func TestCodeSpansPairOnOneLine(t *testing.T) {
	checkSpans(t, []spanCase{
		{"one backtick", "a `x` b", [][][2]int{{{2, 5}}}},
		{"two backticks, one inside", "a ``x` y`` b", [][][2]int{{{2, 10}}}},
		{"three backticks, two inside", "a ```x`` y``` b", [][][2]int{{{2, 13}}}},
		{"a balanced line after a balanced line", "a `x` b\nc `y` d", [][][2]int{{{2, 5}}, {{2, 5}}}},
		{"an escaped backtick opens nothing", "a \\`x` b `y` c", [][][2]int{{{5, 10}}}},
		{"an escaped backslash leaves the backtick open", "a \\\\`x` b", [][][2]int{{{4, 7}}}},
	})
}

// TestCodeSpansReadAStretchWholeBelowARunLeftOpen: below a backtick run a line
// leaves open, every line to the next blank line reports no span, whatever
// block it starts; the line holding the run keeps its own spans on either side
// of it; a fence's lines leave the stretch as it was.
func TestCodeSpansReadAStretchWholeBelowARunLeftOpen(t *testing.T) {
	checkSpans(t, []spanCase{
		{"a span over a line break", "a `x\ny` b", nil},
		{"a span after it", "a `x\ny` b `z`", nil},
		{"over three lines", "a `x\nmore\ny` b `z`", nil},
		{"a heading, a list item and a blockquote", "a `x\n# h `y`\n- i `z`\n> q `w`", nil},
		{"a fence", "a `x\n```\n`y`\n```\nc `z` d", nil},
		{"a comment line", "a `x\n<!-- c -->\nd `y` e", nil},
		{"spans after the run on its line", "a `` b `c` d\ne `f` g", [][][2]int{{{7, 10}}}},
		{"spans either side of the run", "`a` `` `b`\nc `d`", [][][2]int{{{0, 3}, {7, 10}}}},
		{"a blank line ends the stretch", "a `x\n\nc `y` d", [][][2]int{nil, nil, {{2, 5}}}},
		{"a line of spaces and tabs ends it", "a `x\n \t\nc `y` d", [][][2]int{nil, nil, {{2, 5}}}},
		{"a line of no-break spaces does not", "a `x\n \nc `y` d", nil},
		{"a fence in a sure stretch", "a `x` b\n```\n`\n```\nc `y` d", [][][2]int{{{2, 5}}, nil, nil, nil, {{2, 5}}}},
	})
}

// TestCodeSpansStepOverComments: backticks inside an HTML comment are literal,
// and a comment opener inside a span opens nothing.
func TestCodeSpansStepOverComments(t *testing.T) {
	checkSpans(t, []spanCase{
		{"a backtick inside a comment", "<!-- `x --> `y` z", [][][2]int{{{12, 15}}}},
		{"an opener inside a span", "`<!--` x <!-- `y` -->", [][][2]int{{{0, 6}}}},
		{"a comment over two lines", "a <!-- b\n`c` -->\nd `e`", [][][2]int{nil, nil, {{2, 5}}}},
		{"an opener after a run left open", "a `x <!-- b\nc --> `d`", nil},
	})
}

// TestReadOpensNoCommentBelowARunLeftOpen: from a backtick run a line leaves
// open to the next blank line, a comment opener opens no comment, on the run's
// own line or below it, and the page reads those lines whole; after a blank
// line, and after balanced lines, an opener opens one as before.
func TestReadOpensNoCommentBelowARunLeftOpen(t *testing.T) {
	for _, c := range []struct {
		name string
		body string
		mask []uint8
	}{
		{"an opener on the line below", "Intro `a\nb <!-- c\nd -->\ne", []uint8{0, 0, 0, 0}},
		{"an opener after the run on its line", "Intro `a <!-- c\nd -->", []uint8{0, 0}},
		{"an opener inside a span over a line break", "Intro `a <!-- b\nc` d --> e", []uint8{0, 0}},
		{"a blockquote", "> a `x\n> y` <!-- b\n> c -->\nd", []uint8{0, 0, 0, 0}},
		{"an opener after a blank line", "Intro `a\n\nb <!-- c\nd -->", []uint8{0, 0, MaskComment, MaskComment}},
		{"an opener after balanced lines", "Intro `a` b\nc <!-- d\ne -->", []uint8{0, MaskComment, MaskComment}},
		{"an opener inside a one-line span", "`<!--` b\nc", []uint8{0, 0}},
	} {
		body := lines(c.body)
		if m := Mask(body); !reflect.DeepEqual(m, c.mask) {
			t.Errorf("%s: Mask(%q) = %v, want %v", c.name, c.body, m, c.mask)
		}
		p := ReadPage(body)
		if _, ok := p.Unsure(); ok {
			t.Errorf("%s: ReadPage(%q) is unsure, want it read", c.name, c.body)
		}
		for i, m := range c.mask {
			if m == 0 && p.Text[i] != body[i] {
				t.Errorf("%s: Text[%d] = %q, want the line whole", c.name, i, p.Text[i])
			}
		}
	}
}

// TestOpensCommentReadsTheLineAlone: a line read alone opens a comment after
// a backtick run it leaves open, the answer a writer asking before it appends
// the line needs, since on the page the run may pair with nothing.
func TestOpensCommentReadsTheLineAlone(t *testing.T) {
	for ln, want := range map[string]bool{
		"a `x <!-- y":     true,
		"a `x` <!-- y":    true,
		"a `<!--` y":      false,
		"a <!-- `x -->":   false,
		"a `x <!-- y -->": false,
	} {
		if got := OpensComment(ln); got != want {
			t.Errorf("OpensComment(%q) = %v, want %v", ln, got, want)
		}
	}
}

// TestCodeSpansStepInLineWithTheBody: every step the line walk takes is
// counted, and each shape takes steps in line with the body: a body twice or
// four times as long takes at most half as many steps again per byte. A scan
// from each run to the line's end, or from each comment opener back to the
// line's start, grows with the square of a line on the first shape and the
// comment shapes.
func TestCodeSpansStepInLineWithTheBody(t *testing.T) {
	steps := 0
	stepHook = func() { steps++ }
	walkHook = func() { steps++ }
	defer func() { stepHook, walkHook = nil, nil }()
	shapes := map[string]func(n int) string{
		"runs of every length on one line": func(n int) string {
			var b strings.Builder
			for k := 1; k <= n/10; k++ {
				b.WriteString(strings.Repeat("`", k) + " a ")
			}
			return b.String()
		},
		"runs of every length, one a line, in one stretch": func(n int) string {
			var b strings.Builder
			for k := 1; k <= n/10; k++ {
				b.WriteString("t " + strings.Repeat("`", k) + "\n")
			}
			return b.String()
		},
		"an unpaired backtick in every paragraph": func(n int) string {
			return strings.Repeat("a ` b <!-- c --> d\n\n", n)
		},
		"one long stretch of single backticks": func(n int) string {
			return strings.Repeat("a ` b\n", n)
		},
		"empty comments, then a backtick left open, then a blockquote": func(n int) string {
			return strings.Repeat("<!---->", n) + " `\n> x\n"
		},
		"a backtick left open, then comment openers": func(n int) string {
			return "` " + strings.Repeat("<!---->", n) + "\n> x\n"
		},
		"comment openers none closes": func(n int) string {
			return strings.Repeat("<!--", n) + "\n> x\n"
		},
		"escaped backticks": func(n int) string {
			return strings.Repeat("\\`", n) + "\n" + strings.Repeat("\\\\`x` ", n) + "\n"
		},
	}
	for name, shape := range shapes {
		count := func(n int) (int, int) {
			body := lines(shape(n))
			steps = 0
			ReadPage(body)
			CodeSpans(body)
			return steps, len(strings.Join(body, "\n"))
		}
		n := 1000
		small, ns := count(n)
		large, nl := count(2 * n)
		if small == 0 || float64(large)/float64(nl) > 1.5*float64(small)/float64(ns) {
			t.Errorf("%s: %d steps over %d bytes, %d over %d: want steps in line with the body", name, small, ns, large, nl)
		}
	}
}
