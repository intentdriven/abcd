package repolint_test

import "testing"

// The persona exemption yields to the caller's own home whatever spelling
// HOME is written in: a trailing dot segment is no part of the user's name, so
// /Users/alice is still this machine's own home under HOME=/Users/alice/. and
// still a real leak (iss-2609290055092630).
func TestAC_PrivacyPersonaExemptionYieldsToAnUncleanlySpelledOwnHome(t *testing.T) {
	b := newFixtureRepo(t).conforming().
		file("reference/paths.md", "the fixture lives at /Users/alice/notes.md\n").
		commit()
	t.Setenv("HOME", "/Users/alice/.")
	res := b.run()
	if f := findingFor(res, "privacy-hygiene"); f == nil {
		t.Fatal("the caller's own home, spelled with a trailing dot segment, was exempted as a persona")
	}
}
