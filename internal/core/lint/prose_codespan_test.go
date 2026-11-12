package lint

import "testing"

// TestReadProseLinesReadsCodeSpansAsThePagePairsThem: a forward-looking marker
// inside a span the page pairs over a line break is a mention of the marker,
// and one the page shows after such a span is a marker.
func TestReadProseLinesReadsCodeSpansAsThePagePairsThem(t *testing.T) {
	for _, c := range []struct {
		name   string
		lines  []string
		marker bool
	}{
		{"inside a span over a line break", []string{"Write `the marker", "on iss-2611121843363011 <!-- record-lint: forward-looking -->` to hold it."}, false},
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
