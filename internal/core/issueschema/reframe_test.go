package issueschema_test

import (
	"slices"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// TestReframeKnownIsRequiredPlusAfter pins the reframe record's two lists
// (spc-2609020626048705): the required set is what a first half carries, and
// the allow-list is that set plus the after half, so an open record and a
// complete one are both well-formed and nothing else is.
func TestReframeKnownIsRequiredPlusAfter(t *testing.T) {
	wantRequired := []string{"schema_version", "id", "occasioned_by", "construal_before", "glossary_before", "scope_before", "grounds"}
	if !slices.Equal(issueschema.ReframeRequired, wantRequired) {
		t.Fatalf("ReframeRequired = %v, want %v", issueschema.ReframeRequired, wantRequired)
	}
	after := []string{"construal_after", "glossary_after", "scope_after", "changed"}
	if !slices.Equal(issueschema.ReframeAfter, after) {
		t.Fatalf("ReframeAfter = %v, want %v", issueschema.ReframeAfter, after)
	}
	assertKnownCoversRequired(t, "ReframeKnown", issueschema.ReframeKnown, issueschema.ReframeRequired)
	for _, f := range after {
		if !issueschema.ReframeKnown[f] {
			t.Errorf("ReframeKnown omits the after-half property %q", f)
		}
	}
	if got, want := len(issueschema.ReframeKnown), len(wantRequired)+len(after); got != want {
		t.Errorf("ReframeKnown holds %d keys, want exactly required plus after (%d)", got, want)
	}
	// The record carries no text of any surface: a key that could hold the
	// abandoned framing is the one thing adr-55 forbids this record.
	for k := range issueschema.ReframeKnown {
		for _, banned := range []string{"text", "body", "prior", "construal"} {
			if k == banned {
				t.Errorf("ReframeKnown admits %q, a key that would carry a surface's text", k)
			}
		}
	}

	if issueschema.ReframeFamily != "rfm" || issueschema.ReframesDir != "reframes" {
		t.Errorf("family/store = %q/%q, want rfm/reframes", issueschema.ReframeFamily, issueschema.ReframesDir)
	}
	wantOcc := []string{issueschema.ReadingItemFamily, issueschema.DispositionFamily, issueschema.SurpriseFamily}
	if !slices.Equal(issueschema.ReframeOccasionFamilies, wantOcc) {
		t.Errorf("ReframeOccasionFamilies = %v, want %v", issueschema.ReframeOccasionFamilies, wantOcc)
	}
	if !slices.Equal(issueschema.FrameSurfaceNames, []string{"construal", "glossary", "scope"}) {
		t.Errorf("FrameSurfaceNames = %v", issueschema.FrameSurfaceNames)
	}
}

// TestValidReframeOccasionIsVerbatimAndClosed holds the occasion to the three
// families by shape, with nothing around the handle.
func TestValidReframeOccasionIsVerbatimAndClosed(t *testing.T) {
	for _, ok := range []string{"rdi-1", "dsp-22", "srp-2609251200001234"} {
		if !issueschema.ValidReframeOccasion(ok) {
			t.Errorf("ValidReframeOccasion(%q) = false, want true", ok)
		}
	}
	for _, bad := range []string{"", "adm-1", "iss-1", "rdi-", "RDI-1", " rdi-1", "rdi-1 ", "rdi-1a", "a reading"} {
		if issueschema.ValidReframeOccasion(bad) {
			t.Errorf("ValidReframeOccasion(%q) = true, want false", bad)
		}
	}
}

// TestLedgerDirectoriesCarryReframes: the reframe store is registered with the
// ledger's one directory list, which is what the comparative exclusion rows and
// the scribe's allow list derive from.
func TestLedgerDirectoriesCarryReframes(t *testing.T) {
	if !slices.Contains(issueschema.LedgerDirs(), issueschema.ReframesDir) {
		t.Fatalf("LedgerDirs() = %v omits %q", issueschema.LedgerDirs(), issueschema.ReframesDir)
	}
	if slices.Contains(issueschema.StatusDirs, issueschema.ReframesDir) {
		t.Error("a reframe is not an issue; its store must stay out of StatusDirs")
	}
}
