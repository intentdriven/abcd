package claim

import (
	"errors"
	"fmt"
	"testing"
)

// TestAReservationPastTheMessageCapIsMarkedOverTheCap: a footprint too large
// for one reservation message is refused as ErrRefused and marked
// ErrOverCap, so a caller can tell it from any other refusal, and one the
// writer can carry, though FootprintFits keeps room beside it, is written.
func TestAReservationPastTheMessageCapIsMarkedOverTheCap(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	paths := func(n int) []string {
		var out []string
		for i := range n {
			out = append(out, fmt.Sprintf("pkg/a-file-with-a-long-enough-name-to-fill-%02d.txt", i))
		}
		return out
	}
	_, err := p.reserve("iss-5", paths(80)...)
	if !errors.Is(err, ErrRefused) || !errors.Is(err, ErrOverCap) {
		t.Fatalf("a footprint past the cap is refused, marked over the cap: %v", err)
	}
	if _, err := p.reserve("iss-6", "../docs"); !errors.Is(err, ErrRefused) || errors.Is(err, ErrOverCap) {
		t.Fatalf("another refusal is not marked over the cap: %v", err)
	}
	between := paths(47)
	if FootprintFits(between) {
		t.Fatal("premise: the footprint leaves less room than FootprintFits keeps")
	}
	if _, err := p.reserve("iss-7", between...); err != nil {
		t.Fatalf("a footprint the writer can carry is written: %v", err)
	}
}
