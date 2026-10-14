package recordid

import (
	"os"
	"path/filepath"
	"testing"
)

// TestResolverFindsADecisionRecord pins the dcn family as a record family
// (itd-2609151138388536, step 1): a decision record in the flat folder
// `.abcd/work/decisions/` resolves through both doors, every spelling of its
// handle canonicalises to the one the resolver keys on, and the cited-id
// grammar every ingest boundary bounds a citation with admits it.
func TestResolverFindsADecisionRecord(t *testing.T) {
	root := t.TempDir()
	rel := DecisionsRelDir + "/dcn-2610141200001234-ship-the-folder.md"
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte("---\nid: dcn-2610141200001234\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	r, err := NewResolver(root)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	if got, ok, err := r.Lookup("dcn-2610141200001234"); err != nil || !ok || got != rel {
		t.Errorf("Lookup(dcn-2610141200001234) = %q, %v, %v; want %q", got, ok, err, rel)
	}
	if got, ok, err := LookupOne(root, "dcn-2610141200001234"); err != nil || !ok || got != rel {
		t.Errorf("LookupOne(dcn-2610141200001234) = %q, %v, %v; want %q", got, ok, err, rel)
	}
	if !CitedIDRe.MatchString("dcn-2610141200001234") {
		t.Error("CitedIDRe must admit a dcn handle")
	}
	if got := CanonCitedID("DCN-02610141200001234"); got != "dcn-2610141200001234" {
		t.Errorf("CanonCitedID(DCN-0…) = %q, want dcn-2610141200001234", got)
	}
	// The lifeboat's frozen-line key is a different thing and stays one: dec is
	// no family, so dec-L12 resolves nothing and canonicalises to nothing.
	if CanonCitedID("dec-12") != "" {
		t.Error("dec is not a record family; only dcn is")
	}
}

// TestDateFromStampReadsTheMintStamp pins the helper decide and the decisions
// writer share (spec decision 2): the date is the stamp's own, read off the id's
// digits, and anything that is not a stamp has no date.
func TestDateFromStampReadsTheMintStamp(t *testing.T) {
	for in, want := range map[string]string{
		"2610141200001234": "2026-10-14",
		"260915000000":     "2026-09-15",
		"2609150000000003": "2026-09-15",
		"261314000000":     "", // month 13
		"2610":             "", // too short to be a stamp
		"26101412000x":     "",
		"":                 "",
	} {
		if got := DateFromStamp(in); got != want {
			t.Errorf("DateFromStamp(%q) = %q, want %q", in, got, want)
		}
	}
}
