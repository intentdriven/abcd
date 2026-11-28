package scanner

import (
	"testing"
)

// withHomeSpellings hands the sweep and the detector a fixed spelling list
// for the test, and forgets it after.
func withHomeSpellings(t *testing.T, spellings []string) {
	t.Helper()
	restore := homeSpellingsOf
	reset := func() {
		spellingsMemo.Lock()
		spellingsMemo.home, spellingsMemo.spellings = "", nil
		spellingsMemo.Unlock()
	}
	reset()
	homeSpellingsOf = func(string) []string { return spellings }
	t.Cleanup(func() { homeSpellingsOf = restore; reset() })
}

// A home spelling that is not valid UTF-8 — a HOME, or a symlink target, that
// holds a Latin-1 byte — cannot be held in a regular expression, so the
// detector leaves it out of its pattern rather than panic, and builds no
// pattern when none is left; the literal sweep, which compares bytes, still
// removes it. The account-name pattern does the same for a HOME whose last
// segment is not valid UTF-8.
func TestScanTextSkipsAHomeSpellingNoPatternCanHold(t *testing.T) {
	for _, id := range []Identity{
		{HomePath: "/h/donn\xe9es/maya"},
		{HomePath: "/h/donn\xe9es", HomeUser: "donn\xe9es"},
	} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("ScanText with HomePath %q panicked: %v", id.HomePath, r)
				}
			}()
			ScanText("x", id, nil, nil, "f")
		}()
	}

	withHomeSpellings(t, []string{"/h/maya", "/h/donn\xe9es"})
	var found bool
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("ScanText with a resolved spelling that is not UTF-8 panicked: %v", r)
			}
		}()
		for _, f := range ScanText("open /h/maya/x", Identity{HomePath: "/h/maya"}, nil, DefaultIdentitySeverities(), "f") {
			found = found || f.Kind == kindHomeSelf
		}
	}()
	if !found {
		t.Error("the spelling a pattern can hold was not detected beside one it cannot")
	}
	if got, want := SweepCallerHome("open /h/donn\xe9es/x", "/h/maya"), "open ~/x"; got != want {
		t.Errorf("SweepCallerHome = %q, want the byte spelling swept to %q", got, want)
	}
}
