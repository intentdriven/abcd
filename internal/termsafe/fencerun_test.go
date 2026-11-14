package termsafe

import (
	"slices"
	"testing"
)

// TestLongestFenceRun pins the one counter every span and fence writer sizes
// its delimiter from: the longest run of ONE fence character among those
// asked for, with a backtick next to a tilde starting a new run.
func TestLongestFenceRun(t *testing.T) {
	backtick := []byte{FenceBacktick}
	both := []byte{FenceTilde, FenceBacktick}
	for _, c := range []struct {
		name   string
		s      string
		fences []byte
		want   int
	}{
		{"empty input", "", backtick, 0},
		{"empty input, both fences", "", both, 0},
		{"no run", "plain text", backtick, 0},
		{"no fences asked for", "``` ~~~", nil, 0},
		{"single backtick", "a`b", backtick, 1},
		{"run at start", "```abc", backtick, 3},
		{"run in the middle", "ab````cd", backtick, 4},
		{"run at end", "abc``", backtick, 2},
		{"whole string is a run", "`````", backtick, 5},
		{"longest of several runs", "` `` ```` ``` `", backtick, 4},
		{"tilde ignored when only backtick asked", "~~~~~ ``", backtick, 2},
		{"tilde run counted when asked", "~~~~~ ``", both, 5},
		{"tilde run at end", "x`~~~~", both, 4},
		{"mixed stretch counts per character", "``~~~`", both, 3},
		{"alternating characters never lengthen", "`~`~`~", both, 1},
		{"mixed stretch, backtick only", "``~``~``", backtick, 2},
		{"multi-byte text around a run", "é```ü", backtick, 3},
		{"multi-byte text between runs", "``日本``語```", backtick, 3},
		{"multi-byte text around a tilde run", "→~~~~←", both, 4},
		{"invalid UTF-8 breaks a run", "``\xff```", backtick, 3},
		{"newline breaks a run", "``\n``", both, 2},
	} {
		got := LongestFenceRun(c.s, c.fences...)
		if got != c.want {
			t.Errorf("%s: LongestFenceRun(%q, %q) = %d, want %d", c.name, c.s, c.fences, got, c.want)
		}
		// The writers moved onto this counter walked runes before; a bytewise
		// count must give them the same answer, multi-byte and invalid text
		// included.
		if ref := runeWiseLongestRun(c.s, c.fences); got != ref {
			t.Errorf("%s: LongestFenceRun(%q, %q) = %d, a rune-wise count gives %d", c.name, c.s, c.fences, got, ref)
		}
	}
}

// runeWiseLongestRun is the rune-by-rune count the rules fence writer held
// before it moved onto LongestFenceRun, kept here as the reference the
// bytewise scan is checked against.
func runeWiseLongestRun(s string, fences []byte) int {
	longest, run, prev := 0, 0, rune(0)
	for _, c := range s {
		isFence := c < 0x80 && slices.Contains(fences, byte(c))
		switch {
		case isFence && c == prev:
			run++
		case isFence:
			run = 1
		default:
			run = 0
		}
		prev = c
		longest = max(longest, run)
	}
	return longest
}
