package guard

import (
	"strings"
	"testing"
)

// TestFixingAProgramCostsWhatTheLineHolds: reading whether the line fixes each
// command's program reads each word of the line a bounded number of times, however
// many commands use the variable.
func TestFixingAProgramCostsWhatTheLineHolds(t *testing.T) {
	line := "PM=npm; " + strings.Repeat(`"$PM" update v1; `, 2000)
	segs, err := tokenize(line)
	if err != nil {
		t.Fatal(err)
	}
	f := newFixedReader(line, segs)
	words := len(line)
	for i := range segs {
		f.program(i)
		words += len(segs[i].tokens)
	}
	if f.read > 4*words {
		t.Errorf("fixing %d commands' programs read %d words and bytes of a line of %d", len(segs), f.read, words)
	}
}
