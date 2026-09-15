package recordid

import "testing"

// TestCanonCitedIDFoldsTheSpellingsProseUses is the citation-side counterpart of
// the ADR canonicaliser's test: a handle written in a record's body is the same
// handle whatever case and padding the author reached for, and a gate that keys
// on the raw text would report `ADR-02` as naming no record while `adr-2` sits in
// the store.
func TestCanonCitedIDFoldsTheSpellingsProseUses(t *testing.T) {
	cases := map[string]string{
		"adr-35":               "adr-35",
		"ADR-02":               "adr-2",
		"Adr-0035":             "adr-35",
		"itd-007":              "itd-7",
		"ISS-2608231322321751": "iss-2608231322321751",
		"spc-009":              "spc-9",
		// 20 digits: the trim is textual, so a number no integer type holds
		// still canonicalises rather than collapsing to "not a record".
		"adr-99999999999999999999": "adr-99999999999999999999",
	}
	for in, want := range cases {
		if got := CanonCitedID(in); got != want {
			t.Errorf("CanonCitedID(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestCanonCitedIDRefusesWhatIsNotACitation pins the negative side. An all-zero
// number is refused for the reason canonADRNum refuses it — the allocator issues
// no such id, so no record can ever answer to it — and a family the resolver does
// not resolve is not a cited id at all.
func TestCanonCitedIDRefusesWhatIsNotACitation(t *testing.T) {
	for _, in := range []string{
		"", "adr", "adr-", "adr-0", "spc-00", "itd-x", "rdi-4", "adr-4-slug", "4-adr",
	} {
		if got := CanonCitedID(in); got != "" {
			t.Errorf("CanonCitedID(%q) = %q, want \"\"", in, got)
		}
	}
}

// TestCanonCitedIDAgreesWithTheResolverOnEveryLiveID is the anti-drift assertion:
// the canonicaliser and the resolver must key on the same spelling, or a citation
// of a record that plainly exists would be reported as dangling.
func TestCanonCitedIDAgreesWithTheResolverOnEveryLiveID(t *testing.T) {
	r, err := NewResolver("../../..")
	if err != nil {
		t.Fatal(err)
	}
	if r.Len() == 0 {
		t.Fatal("the repository resolved no records; the assertion would be vacuous")
	}
	for id := range r.ids {
		if got := CanonCitedID(id); got != id {
			t.Errorf("resolver key %q canonicalises to %q; the two must agree", id, got)
		}
	}
}
