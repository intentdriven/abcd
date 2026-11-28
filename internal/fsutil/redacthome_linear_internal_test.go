package fsutil

import (
	"strings"
	"testing"
)

// The single pass over several spellings walks each spelling's search across
// the text once, not once per replacement: a spelling that has no further
// occurrence is not searched again, and one whose next occurrence lies ahead
// is not searched again until the pass reaches it. Without that, a text naming
// the home thousands of times beside a spelling that never occurs (the macOS
// data-volume spelling, added for every home under /Users) is rescanned to its
// end after every replacement, quadratic in the text: seconds for a file the
// scribe's scrub passes whole. The walk is counted, not timed, so a loaded
// machine cannot make the test flaky.
func TestReplaceRootsWalksEachSpellingAcrossTheTextOnce(t *testing.T) {
	const home = "/Users/alice"
	roots := []string{"/System/Volumes/Data" + home, home, "/Users//alice"}
	var in, want strings.Builder
	for i := range 4000 {
		if i%3 == 0 {
			in.WriteString("open /Users//alice/x: denied\n")
		} else {
			in.WriteString("open " + home + "/x: denied\n")
		}
		want.WriteString("open ~/x: denied\n")
	}
	for _, fold := range []bool{false, true} {
		got, scanned := scanRoots(in.String(), roots, "~", fold)
		if got != want.String() {
			t.Fatalf("fold=%v: the text was not redacted line for line", fold)
		}
		if limit := len(roots) * (in.Len() + 1); scanned > limit {
			t.Errorf("fold=%v: the search walked %d bytes over a %d-byte text with %d spellings, more than once across it each (%d)",
				fold, scanned, in.Len(), len(roots), limit)
		}
	}
}

// Remembering each spelling's next occurrence changes nothing about what is
// redacted: the earliest occurrence still wins, the longest spelling a tie,
// and a span is redacted once.
func TestReplaceRootsKeepsTheEarliestLongestMatchOnce(t *testing.T) {
	roots := []string{"/h", "/h/.", "/h//a"}
	for in, want := range map[string]string{
		"/h/./x /h/x /h//a/y":  "~/x ~/x ~/y",
		"/h//a/h/x":            "~/h/x",
		"x /h. /h/._y /h/../z": "x ~. ~/._y ~/../z",
		"/hh /h2 /h":           "/hh /h2 ~",
	} {
		if got := replaceRoots(in, roots, "~", false); got != want {
			t.Errorf("replaceRoots(%q) = %q, want %q", in, got, want)
		}
	}
}
