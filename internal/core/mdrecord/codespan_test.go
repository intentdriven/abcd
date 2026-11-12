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

func checkSpans(t *testing.T, cases []spanCase) {
	t.Helper()
	for _, c := range cases {
		body := lines(c.body)
		got := CodeSpans(body)
		for i := range got {
			if len(got[i]) == 0 {
				got[i] = nil
			}
		}
		want := c.want
		for len(want) < len(body) {
			want = append(want, nil)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: CodeSpans(%q) = %v, want %v", c.name, c.body, got, want)
		}
		page := ReadPage(body).Spans
		for i := range page {
			if len(page[i]) == 0 {
				page[i] = nil
			}
		}
		if !reflect.DeepEqual(page, want) {
			t.Errorf("%s: ReadPage(%q).Spans = %v, want %v", c.name, c.body, page, want)
		}
	}
}

// TestCodeSpansPairOverALineBreak: a backtick run closes at the next run of
// its length on a later line of the same paragraph, as the page pairs it.
func TestCodeSpansPairOverALineBreak(t *testing.T) {
	checkSpans(t, []spanCase{
		{"one backtick", "a `x\ny` b", [][][2]int{{{2, 4}}, {{0, 2}}}},
		{"two backticks, one inside", "a ``x\n` y`` b", [][][2]int{{{2, 5}}, {{0, 5}}}},
		{"three backticks, two inside", "a ```x\n`` y``` b", [][][2]int{{{2, 6}}, {{0, 7}}}},
		{"over three lines", "a `x\nmore\ny` b", [][][2]int{{{2, 4}}, {{0, 4}}, {{0, 2}}}},
		{"a list item's continuation", "- a `x\n  y` b", [][][2]int{{{4, 6}}, {{0, 4}}}},
		{"an unpaired run, then a span on the next line", "a `x\ny` b `z`", [][][2]int{{{2, 4}}, {{0, 2}, {5, 8}}}},
		{"a span on its own line", "a `x` b", [][][2]int{{{2, 5}}}},
		{"an escaped backtick opens nothing", "a \\`x` b `y`", [][][2]int{{{5, 10}}}},
	})
}

// TestCodeSpansEndAtTheParagraph: a blank line, a heading, a fence, a list
// item, a thematic break or a comment line ends the paragraph, and a run left
// open there is literal backticks.
func TestCodeSpansEndAtTheParagraph(t *testing.T) {
	checkSpans(t, []spanCase{
		{"a blank line", "a `x\n\ny` b", nil},
		{"a heading", "a `x\n# y` b", nil},
		{"a fence", "a `x\n```\ny`\n```\nz` c", nil},
		{"a list item", "a `x\n- y` b", nil},
		{"a numbered item", "a `x\n1. y` b", nil},
		{"a thematic break", "a `x\n***\ny` b", nil},
		{"a setext underline", "a `x\n---\ny` b", nil},
		{"a comment line", "a `x\n<!-- c -->\ny` b", nil},
		{"after a heading", "# a `x\ny` b", nil},
	})
}

// TestCodeSpansReadWhereThePairingIsOpen: where the paragraph's extent turns
// on a construct the package does not parse, and a run there could pair over
// it, the lines hold no span, so a reader reads them.
func TestCodeSpansReadWhereThePairingIsOpen(t *testing.T) {
	checkSpans(t, []spanCase{
		{"a blockquote line after an unpaired run", "> a `x\n> y` b `z`", nil},
		{"a table cell holds the run", "| h | i |\n|---|---|\n| `a | b` |", nil},
		{"an HTML block", "<div>\na `x` b\n</div>", nil},
		{"an indented first line", "    a `x` b", nil},
		{"a tag holding a backtick", "a <b title=\"`\">x</b> `y", nil},
		{"self-contained lines across a blockquote line", "> a `x` b\n> c `y` d", [][][2]int{{{4, 7}}, {{4, 7}}}},
	})
}

// TestReadOpensNoCommentInsideASpanOverALineBreak: an opener inside a span
// the page pairs over a line break opens no comment, so the text after the
// span is read.
func TestReadOpensNoCommentInsideASpanOverALineBreak(t *testing.T) {
	body := lines("Intro `a <!-- b\nc` d --> e")
	if m := Mask(body); m[0] != 0 || m[1] != 0 {
		t.Fatalf("Mask = %v, want both lines live", m)
	}
	p := ReadPage(body)
	if p.Text[1] != "c` d --> e" {
		t.Fatalf("Text[1] = %q, want the line whole", p.Text[1])
	}
	if _, ok := p.Unsure(); ok {
		t.Fatalf("the page is unsure, want it read whole")
	}
}

// TestReadPageReadsACommentWhosePairingIsOpen: a comment opener on a line
// whose spans the package cannot settle, with a backtick before it, makes the
// page unsure from that line, so its text is read.
func TestReadPageReadsACommentWhosePairingIsOpen(t *testing.T) {
	p := ReadPage(lines("> a `x\n> y` <!-- b\n> c -->\nd"))
	if l, ok := p.Unsure(); !ok || l > 1 {
		t.Fatalf("Unsure = %d %v, want line 1", l, ok)
	}
}

// TestCodeSpansStepInLineWithTheBody: a paragraph of runs of every length,
// none of which closes, a body of many paragraphs each left with an unpaired
// backtick and a comment opener, and one long paragraph of single backticks
// take steps in line with the body: twice or four times the body, at most
// about as many times the steps. A scan from each run to the paragraph's end
// takes about eight times the steps on the first shape.
func TestCodeSpansStepInLineWithTheBody(t *testing.T) {
	steps := 0
	stepHook = func() { steps++ }
	defer func() { stepHook = nil }()
	shapes := map[string]func(n int) string{
		"runs of every length in one paragraph": func(n int) string {
			var b strings.Builder
			for k := 1; k <= n; k++ {
				b.WriteString("t " + strings.Repeat("`", k) + "\n")
			}
			return b.String()
		},
		"an unpaired backtick in every paragraph": func(n int) string {
			return strings.Repeat("a ` b <!-- c --> d\n\n", n)
		},
		"one long paragraph of single backticks": func(n int) string {
			return strings.Repeat("a ` b\n", n)
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
		if strings.HasPrefix(name, "runs") {
			n = 100 // the body grows with the square of n
		}
		small, ns := count(n)
		large, nl := count(2 * n)
		if small == 0 || float64(large)/float64(nl) > 1.5*float64(small)/float64(ns) {
			t.Errorf("%s: %d steps over %d bytes, %d over %d: want steps in line with the body", name, small, ns, large, nl)
		}
	}
}
