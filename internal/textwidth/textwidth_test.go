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

// TestHangKeepsSpacingAndBreaksAtSpaces pins the hanging wrap the status board
// lays its rows with (iss-2610031207397996): a line that fits comes back
// byte-identical, runs of spaces and a leading indent included; a longer one
// breaks only at a space, its first line at most first columns and every
// later line at most rest, with no blank carried to either side of a break.
func TestHangKeepsSpacingAndBreaksAtSpaces(t *testing.T) {
	cases := []struct {
		in          string
		first, rest int
		want        []string
	}{
		{"  git repo:   yes", 80, 76, []string{"  git repo:   yes"}},
		{"", 80, 76, []string{""}},
		{"      itd-5  The head  [next up]", 22, 16, []string{"      itd-5  The head", "[next up]"}},
		{"    receipts: 25 release receipts — the oldest", 28, 21, []string{"    receipts: 25 release", "receipts — the oldest"}},
		{"a  b", 2, 2, []string{"a", "b"}},
		{"word supercalifragilistic", 10, 6, []string{"word", "supercalifragilistic"}},
		{"日本 日本 日本", 9, 5, []string{"日本 日本", "日本"}},
	}
	for _, c := range cases {
		got := Hang(c.in, c.first, c.rest)
		if strings.Join(got, "|") != strings.Join(c.want, "|") {
			t.Errorf("Hang(%q, %d, %d) = %q, want %q", c.in, c.first, c.rest, got, c.want)
		}
	}
}

// TestFitCutsByDisplayWidth is A7's helper (spc-2610031844142274): a text that
// fits comes back unchanged; a longer one is cut by display width and ends in
// the ellipsis, never splitting a wide rune, the whole within the limit.
func TestFitCutsByDisplayWidth(t *testing.T) {
	cases := []struct {
		in, ellipsis string
		limit        int
		want         string
	}{
		{"short", "…", 10, "short"},
		{"exactly ten", "…", 11, "exactly ten"},
		{"a title that runs past", "…", 10, "a title t…"},
		{"a title that runs past", "...", 10, "a title..."},
		{"日本語の題名", "…", 7, "日本語…"},    // a cut that would split 語's second column keeps it whole
		{"日本語の題名", "…", 8, "日本語…"},    // 日本語 is 6 columns, the ellipsis 1, a fourth wide rune does not fit
		{"日本語の題名", "...", 7, "日本..."}, // 4 + 3
		{"anything", "…", 0, ""},
		{"trailing space before the cut", "…", 10, "trailing…"},
	}
	for _, c := range cases {
		got := Fit(c.in, c.limit, c.ellipsis)
		if got != c.want || Columns(got) > c.limit {
			t.Errorf("Fit(%q, %d, %q) = %q (%d columns), want %q", c.in, c.limit, c.ellipsis, got, Columns(got), c.want)
		}
	}
}

// TestBreakSplitsByDisplayWidth: a run of text wider than a line is split into
// pieces of at most limit columns, never splitting a wide rune.
func TestBreakSplitsByDisplayWidth(t *testing.T) {
	cases := []struct {
		in    string
		limit int
		want  []string
	}{
		{"fits", 10, []string{"fits"}},
		{"abcdefghij", 4, []string{"abcd", "efgh", "ij"}},
		{"日本語の", 5, []string{"日本", "語の"}},
		{"", 4, []string{""}},
	}
	for _, c := range cases {
		if got := Break(c.in, c.limit); strings.Join(got, "|") != strings.Join(c.want, "|") {
			t.Errorf("Break(%q, %d) = %q, want %q", c.in, c.limit, got, c.want)
		}
	}
}
