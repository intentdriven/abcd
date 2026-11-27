package scanner

import "testing"

// A home's received form that ends in a separator, or holds fewer segments
// than the home it came from, names a different folder: the dropped-byte form
// of a home whose last segment holds no valid byte is "/home/", the parent of
// every account. Neither the sweep, the residual check nor the detector takes
// such a form for the home.
func TestHomeFormThatLosesASegmentIsNotTheHome(t *testing.T) {
	for _, c := range []struct{ home, text string }{
		{"/home/\xe8\xe2\xe0\xed", "list /home/ and /home/zed/notes today"},
		{"/h/\xe9/maya", "open /h//maya/x and /h/maya/x"},
	} {
		withHomeSpellings(t, []string{c.home})
		if got := SweepCallerHome(c.text, c.home); got != c.text {
			t.Errorf("home %+q: SweepCallerHome(%q) = %q, want it unchanged", c.home, c.text, got)
		}
		for _, f := range ScanText(c.text, Identity{HomePath: c.home}, nil, DefaultIdentitySeverities(), "f") {
			if f.Kind == kindHomeSelf {
				t.Errorf("home %+q: the detector took %q for the home", c.home, f.Matched)
			}
		}
	}
	withHomeSpellings(t, []string{"/home/\xe8\xe2\xe0\xed"})
	if got, want := SweepCallerHome("open /home/����/x", "/home/\xe8\xe2\xe0\xed"), "open ~/x"; got != want {
		t.Errorf("the replaced form of the home was not swept: %q, want %q", got, want)
	}
}
