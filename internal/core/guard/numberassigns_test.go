package guard

import (
	"slices"
	"strings"
	"testing"
)

// numberAssignsShapes are texts whose arithmetic openers a reading that
// scanned from every opener to its close would read in time growing with
// the square of their length: openers that never close, inside quotes and
// out, nested openers, and nested subscripts.
var numberAssignsShapes = map[string]func(n int) string{
	"open parens inside quotes": func(n int) string { return `echo "` + strings.Repeat("(", n) + `"` },
	"open parens":               func(n int) string { return "echo " + strings.Repeat("((", n/2) },
	"open brackets":             func(n int) string { return "echo " + strings.Repeat("$[", n/2) },
	"nested parens":             func(n int) string { return strings.Repeat("((", n/4) + "x=1" + strings.Repeat("))", n/4) },
	"nested subscripts":         func(n int) string { return "((" + strings.Repeat("a[", n/2) + "))" },
}

// TestNumberAssignsReadsEachByteAFewTimes: the reading of the names an
// arithmetic expression or a descriptor redirection assigns visits each byte
// of the text a bounded number of times, whatever openers it holds, so 64
// KiB of `(` inside quotes is read in time growing with its length.
func TestNumberAssignsReadsEachByteAFewTimes(t *testing.T) {
	for name, build := range numberAssignsShapes {
		t.Run(name, func(t *testing.T) {
			text := build(64 << 10)
			if _, work := numberAssignsWork(text); work > 8*len(text) {
				t.Fatalf("%d units of work on %d bytes, want at most %d", work, len(text), 8*len(text))
			}
		})
	}
}

// TestNumberAssignsStillReadsEveryTarget: reading each byte in one span
// keeps every target: one inside a nested expression, one after a span that
// closes, one in a subscript, a descriptor redirection inside an arithmetic
// expression's substitution, and one after an opener that never closes,
// where the text after it is read as the expression it opens.
func TestNumberAssignsStillReadsEveryTarget(t *testing.T) {
	for text, want := range map[string][]string{
		`(( a = (( b += 1 )) ))`:              {"a", "b"},
		`(( 1 )); echo $(( c++ ))`:            {"c"},
		`(( x[i=2] ))`:                        {"i"},
		`: $(( 1 + $(exec {fd}>f; echo 1) ))`: {"fd"},
		`echo "((" ; (( d=1 ))`:               {"d"},
		`echo $[ e = 1 ]`:                     {"e"},
		`echo "(( "; f=1`:                     {"f"},
		`(( g == 1 ))`:                        nil,
	} {
		got := numberAssigns(text)
		slices.Sort(got)
		if !slices.Equal(got, want) {
			t.Errorf("%q: names %v, want %v", text, got, want)
		}
	}
}

// BenchmarkNumberAssignsOnOpenParensInQuotes reads 64 KiB of `(` inside
// quotes.
func BenchmarkNumberAssignsOnOpenParensInQuotes(b *testing.B) {
	text := numberAssignsShapes["open parens inside quotes"](64 << 10)
	for b.Loop() {
		numberAssigns(text)
	}
}
