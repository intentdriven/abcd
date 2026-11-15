package lint

import "testing"

// TestReadProseLinesReadsCodeSpansOnOneLine: a forward-looking marker inside a
// span that opens and closes on its line is a mention of the marker; a span
// is paired on each line alone, so a marker inside a span over a line break is
// a marker, and on the line below it two backticks pair as on any line.
func TestReadProseLinesReadsCodeSpansOnOneLine(t *testing.T) {
	for _, c := range []struct {
		name   string
		lines  []string
		marker bool
	}{
		{"inside a span on one line", []string{"Intro.", "Write `<!-- record-lint: forward-looking -->` on iss-2611121843363011 to hold it."}, false},
		{"inside a span over a line break", []string{"Write `the marker", "on iss-2611121843363011 <!-- record-lint: forward-looking -->` to hold it."}, true},
		{"between two backticks below a span over a line break", []string{"See `x", "y` iss-2611121843363011 <!-- record-lint: forward-looking --> `z`."}, false},
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

// TestReadProseLinesPairsSpansOnEveryLine: the prose-citation rule pairs a
// span on each line alone, below a backtick run a line leaves open as
// anywhere else, so a marker quoted on its line there is a mention.
func TestReadProseLinesPairsSpansOnEveryLine(t *testing.T) {
	pls := readProseLines([]string{"See `x", "y and `<!-- record-lint: forward-looking -->` on iss-2611121843363011."}, nil)
	if len(pls) != 1 || pls[0].forwardMarker {
		t.Fatalf("readProseLines = %+v, want line 2 with no marker", pls)
	}
}
