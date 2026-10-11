package mdrecord

import (
	"strings"
	"testing"
)

// TestTheLinkPassIsLinear: a heading title is the document's to choose, so the
// link scanner's pass must cost a bounded multiple of the title's length on
// any input. The shapes below are the ones that make a naive scanner re-read:
// inline tails that open and never close, unbalanced parentheses, an unclosed
// angle destination or title, and brackets nested deep. The pass charges
// every failed tail to a budget of the title's length and copies the rest as
// written once it is spent, so its work stays under three times the length.
func TestTheLinkPassIsLinear(t *testing.T) {
	const reps = 4000
	for _, s := range []string{
		strings.Repeat("[a](x", reps),
		strings.Repeat("[a](x(", reps) + " ",
		strings.Repeat("[a](<x", reps),
		strings.Repeat("[a](x \"", reps),
		strings.Repeat("[a](x (", reps),
		strings.Repeat("[", reps) + strings.Repeat("]", reps),
		strings.Repeat("![[a]", reps),
		strings.Repeat("[a]", reps),
	} {
		for _, shortcuts := range []bool{true, false} {
			if _, work := UnwrapLinkPass(s, shortcuts); work > 3*len(s) {
				t.Errorf("a pass over %q... (%d bytes, shortcuts %v) did %d bytes of work, over three times its length",
					s[:12], len(s), shortcuts, work)
			}
		}
	}
}

// TestIsASCIIPunctIsCommonMarksClass holds isASCIIPunct to the 32 bytes
// CommonMark 2.1 names as ASCII punctuation, and to nothing else in a byte.
func TestIsASCIIPunctIsCommonMarksClass(t *testing.T) {
	const commonMark = "!\"#$%&'()*+,-./:;<=>?@[\\]^_\x60{|}~"
	for c := 0; c < 256; c++ {
		want := strings.IndexByte(commonMark, byte(c)) >= 0
		if got := isASCIIPunct(byte(c)); got != want {
			t.Errorf("isASCIIPunct(%#x) = %v, want %v", c, got, want)
		}
	}
}

// TestAShortcutIsKeptOnlyWhenAsked: the reading floor reads a shortcut
// reference as its label, the side a floor errs on, and a principle statement
// keeps it as the literal text it is without a definition; either way the
// links inside its brackets are unwrapped.
func TestAShortcutIsKeptOnlyWhenAsked(t *testing.T) {
	for _, c := range []struct {
		in        string
		shortcuts bool
		want      string
	}{
		{"[Audit Notes]", true, "Audit Notes"},
		{"[Audit Notes]", false, "[Audit Notes]"},
		{"![alt]", false, "![alt]"},
		{"[see [a](x/(y))]", false, "[see a]"},
		{"[see [a](x/(y))]", true, "see [a](x/(y))"},
	} {
		if got, _ := UnwrapLinkPass(c.in, c.shortcuts); got != c.want {
			t.Errorf("UnwrapLinkPass(%q, %v) = %q, want %q", c.in, c.shortcuts, got, c.want)
		}
	}
}
