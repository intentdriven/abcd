package issueschema

import "testing"

// TestOccasionedByIsNoLongerReservedOnTheEnvelope: the reservation spc-58 placed
// on the reading envelope is retired with the surprise verb
// (spc-2609020626040342). The join lives on the surprise record and nowhere
// else, so a reading record carrying the key is refused as any unknown key is,
// and the surprise's own required set is where the key is declared.
func TestOccasionedByIsNoLongerReservedOnTheEnvelope(t *testing.T) {
	if ReadingKnown["occasioned_by"] {
		t.Error("occasioned_by is still a known key on the reading record")
	}
	if !SurpriseKnown["occasioned_by"] {
		t.Error("occasioned_by must be declared on the surprise record")
	}
	for _, f := range []string{ReadingItemFamily, AdmissionFamily, DispositionFamily} {
		if !ValidSurpriseOccasion(f + "-12") {
			t.Errorf("%s-12 must be a valid surprise occasion", f)
		}
	}
	for _, v := range []string{"", "prose", "itd-1", "srp-1", "RDI-1", " rdi-1", "rdi-", "rdi-1x"} {
		if ValidSurpriseOccasion(v) {
			t.Errorf("ValidSurpriseOccasion(%q) = true, want false", v)
		}
	}
}
