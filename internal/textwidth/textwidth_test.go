package textwidth

import (
	"strings"
	"testing"
)

// TestColumnsCountsWideRunesAsTwo pins the measure: East Asian wide and
// fullwidth runes take two columns, every other rune one.
func TestColumnsCountsWideRunesAsTwo(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"Product Q2", 10},
		{"日本", 4},  // two wide runes
		{"ＡＢ", 4},  // two fullwidth runes
		{"é—★", 3}, // narrow and ambiguous runes count one each
		{"a日b", 4},
	}
	for _, c := range cases {
		if got := Columns(c.in); got != c.want {
			t.Errorf("Columns(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

// TestGreedyFillsEachLine pins the greedy fill the row estimate counts with:
// as many words as fit, a word wider than the limit alone on its line.
func TestGreedyFillsEachLine(t *testing.T) {
	cases := []struct {
		words []string
		limit int
		want  []string
	}{
		{strings.Fields("one two three four"), 14, []string{"one two three", "four"}},
		{strings.Fields("alpha supercalifragilistic beta"), 8, []string{"alpha", "supercalifragilistic", "beta"}},
		{nil, 10, nil},
	}
	for _, c := range cases {
		got := Greedy(c.words, c.limit)
		if strings.Join(got, "|") != strings.Join(c.want, "|") || len(got) != len(c.want) {
			t.Errorf("Greedy(%q, %d) = %q, want %q", c.words, c.limit, got, c.want)
		}
	}
}

// TestWrapIsBalanced pins the balanced wrap: the fewest lines the greedy fill
// needs, then the narrowest measure that keeps that count, so no last word is
// stranded. The banner's TestWrapWords holds the same behaviour at its seam.
func TestWrapIsBalanced(t *testing.T) {
	got := Wrap("one two three four", 14)
	want := []string{"one two", "three four"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("Wrap = %q, want %q", got, want)
	}
	if got := Wrap("", 66); got != nil {
		t.Errorf("Wrap of empty text = %q, want nil", got)
	}
}
