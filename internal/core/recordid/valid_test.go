package recordid

import "testing"

// TestValidIDPredicates pins the exact accepted set of the one well-formedness
// rule the loaders and the record-lint gate now share. The absent-value
// spellings (empty, YAML nulls) are listed explicitly because they are the shape
// a hand-edited record actually carries.
func TestValidIDPredicates(t *testing.T) {
	intentOK := []string{"itd-1", "itd-999", "itd-0007"}
	intentBad := []string{"", "null", "~", "TBD", "itd-", "itd-1-slug", " itd-1", "itd-1\n", "ITD-1", "spc-1"}
	for _, id := range intentOK {
		if !ValidIntentID(id) {
			t.Errorf("ValidIntentID(%q) = false, want true", id)
		}
	}
	for _, id := range intentBad {
		if ValidIntentID(id) {
			t.Errorf("ValidIntentID(%q) = true, want false", id)
		}
	}

	specOK := []string{"spc-1", "spc-999", "spc-0007"}
	specBad := []string{"", "null", "~", "TBD", "spc-", "spc-1-slug", " spc-1", "spc-1\n", "SPC-1", "itd-1"}
	for _, id := range specOK {
		if !ValidSpecID(id) {
			t.Errorf("ValidSpecID(%q) = false, want true", id)
		}
	}
	for _, id := range specBad {
		if ValidSpecID(id) {
			t.Errorf("ValidSpecID(%q) = true, want false", id)
		}
	}
}

// TestAdmissionAndSurpriseIDGrammars pins the two families the admission and
// surprise verbs build paths from (spc-2609020626040342): an id nothing has
// matched never becomes a filename, so each family's grammar sits beside the
// four this package already holds.
func TestAdmissionAndSurpriseIDGrammars(t *testing.T) {
	cases := []struct {
		name  string
		valid func(string) bool
		ok    []string
		bad   []string
	}{
		{"ValidAdmissionID", ValidAdmissionID,
			[]string{"adm-1", "adm-2609251200001234", "adm-0007"},
			[]string{"", "null", "~", "adm-", "adm-1-slug", " adm-1", "adm-1\n", "ADM-1", "srp-1", "adm-../x", "adm-1/..", "rdi-1"}},
		{"ValidSurpriseID", ValidSurpriseID,
			[]string{"srp-1", "srp-2609251200001234", "srp-0007"},
			[]string{"", "null", "~", "srp-", "srp-1-slug", " srp-1", "srp-1\n", "SRP-1", "adm-1", "srp-../x", "dsp-1"}},
		// The reframe record (spc-2609020626048705) writes reframes/rfm-N.md and
		// the dispatcher reads it back, so its grammar joins the two above.
		{"ValidReframeID", ValidReframeID,
			[]string{"rfm-1", "rfm-2609251200001234", "rfm-0007"},
			[]string{"", "null", "~", "rfm-", "rfm-1-slug", " rfm-1", "rfm-1\n", "RFM-1", "srp-1", "rfm-../x", "rfm-1/.."}},
	}
	for _, c := range cases {
		for _, id := range c.ok {
			if !c.valid(id) {
				t.Errorf("%s(%q) = false, want true", c.name, id)
			}
		}
		for _, id := range c.bad {
			if c.valid(id) {
				t.Errorf("%s(%q) = true, want false", c.name, id)
			}
		}
	}
}
