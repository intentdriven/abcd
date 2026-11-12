package lint

import "testing"

// TestReadProseLinesReadsCodeSpansOnOneLine: a forward-looking marker inside a
// span that opens and closes on its line is a mention of the marker; below a
// backtick run a line leaves open, nothing is read as a span, so a marker there
// is a marker, whether the page shows it inside a span over the line break or
// after one.
func TestReadProseLinesReadsCodeSpansOnOneLine(t *testing.T) {
	for _, c := range []struct {
		name   string
		lines  []string
		marker bool
	}{
		{"inside a span on one line", []string{"Intro.", "Write `<!-- record-lint: forward-looking -->` on iss-2611121843363011 to hold it."}, false},
		{"inside a span over a line break", []string{"Write `the marker", "on iss-2611121843363011 <!-- record-lint: forward-looking -->` to hold it."}, true},
		{"after a span over a line break", []string{"See `x", "y` iss-2611121843363011 <!-- record-lint: forward-looking --> `z`."}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			pls := readProseLines(c.lines, nil)
			if len(pls) != 1 || pls[0].line != 2 {
				t.Fatalf("readProseLines = %+v, want line 2 alone", pls)
			}
			if pls[0].forwardMarker != c.marker {
				t.Fatalf("forwardMarker = %v, want %v", pls[0].forwardMarker, c.marker)
			}
		})
	}
}
