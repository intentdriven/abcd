package scanner

import "testing"

// A store never receives the raw bytes of a home spelling that is not valid
// UTF-8: a JSON transcript carries U+FFFD in place of each invalid byte, and
// the history reader's clipped scalars drop them. The sweep, the residual
// check and the detector read the home in those received forms too.
func TestHomeSpellingIsMatchedInTheFormsAStoreReceives(t *testing.T) {
	withHomeSpellings(t, []string{"/h/maya", "/h/donn\xe9es"})
	for _, in := range []string{"open /h/donn�es/x", "open /h/donnes/x", "open /h/donn\xe9es/x"} {
		if got, want := SweepCallerHome(in, "/h/maya"), "open ~/x"; got != want {
			t.Errorf("SweepCallerHome(%+q) = %+q, want %q", in, got, want)
		}
		if _, resid := SurvivingCallerHome(in, "/h/maya"); len(resid) == 0 {
			t.Errorf("SurvivingCallerHome(%+q) reported nothing", in)
		}
	}
	for _, in := range []string{"open /h/donn�es/x", "open /h/donnes/x"} {
		got, _ := Redact(in, ScanText(in, Identity{HomePath: "/h/maya"}, nil, DefaultIdentitySeverities(), "f"))
		if want := "open ~/x"; got != want {
			t.Errorf("the detector left %+q as %+q, want %q", in, got, want)
		}
	}
}
