package scanner

import (
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// TestHomeRedactorsAgreeOnTheTrailingBoundary holds the two home redactors to
// one trailing rule: for every printable byte after the home, fsutil.RedactRoot
// (the CLI error scrub, the install receipt) redacts exactly when
// SweepCallerHome (the store redactors) sweeps. The two carried the predicate
// in opposite polarities and disagreed on '.', '-' and '_', so "cannot access
// <home>." was scrubbed by one and left whole by the other
// (iss-2608292037564347). '\\' and '%' are left out: the sweep also reads the
// escape-decoded views of a line carrying them, which fsutil has no
// counterpart for, so they judge a different text rather than a different
// boundary.
func TestHomeRedactorsAgreeOnTheTrailingBoundary(t *testing.T) {
	const home = "/srv/qzhome"
	for b := byte(0x20); b < 0x7f; b++ {
		if b == '\\' || b == '%' {
			continue
		}
		in := "cannot access " + home + string(b) + "tail"
		scrubbed := fsutil.RedactRoot(in, home, "~") != in
		swept := SweepCallerHome(in, home) != in
		if scrubbed != swept {
			t.Errorf("byte %q after the home: RedactRoot redacts=%v, SweepCallerHome sweeps=%v", b, scrubbed, swept)
		}
	}
}
