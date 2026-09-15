package recordid

import (
	"os"
	"path/filepath"
	"testing"
)

// seedRecords lays out a miniature record tree: the four id-bearing families in
// their canonical homes, each with a bucket level, so the resolver is exercised
// against the shape a real repository has.
func seedRecords(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := []string{
		".abcd/development/intents/drafts/itd-7-a-draft.md",
		".abcd/development/intents/planned/itd-104-the-ideate-gate.md",
		".abcd/development/intents/shipped/itd-88.md",
		".abcd/development/specs/open/spc-18-ideate.md",
		".abcd/development/specs/closed/spc-6-issue-capture.md",
		".abcd/work/issues/open/iss-146-a-guard.md",
		".abcd/work/issues/resolved/iss-32-consolidation.md",
		".abcd/development/decisions/adrs/0001-three-layer-mental-model.md",
		".abcd/development/decisions/adrs/0037-release-workflow.md",
	}
	for _, rel := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte("# record\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// TestResolverFindsEveryFamily proves one resolver recognises all four cited
// families — including an ADR, whose id lives in the NNNN- filename prefix and
// not in the id grammar the filename uses.
func TestResolverFindsEveryFamily(t *testing.T) {
	root := seedRecords(t)
	r, err := NewResolver(root)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	want := map[string]string{
		"itd-7":   ".abcd/development/intents/drafts/itd-7-a-draft.md",
		"itd-104": ".abcd/development/intents/planned/itd-104-the-ideate-gate.md",
		"itd-88":  ".abcd/development/intents/shipped/itd-88.md",
		"spc-18":  ".abcd/development/specs/open/spc-18-ideate.md",
		"spc-6":   ".abcd/development/specs/closed/spc-6-issue-capture.md",
		"iss-146": ".abcd/work/issues/open/iss-146-a-guard.md",
		"iss-32":  ".abcd/work/issues/resolved/iss-32-consolidation.md",
		"adr-1":   ".abcd/development/decisions/adrs/0001-three-layer-mental-model.md",
		"adr-37":  ".abcd/development/decisions/adrs/0037-release-workflow.md",
	}
	for id, path := range want {
		got, ok := r.Lookup(id)
		if !ok {
			t.Errorf("Lookup(%q): not resolved", id)
			continue
		}
		if got != path {
			t.Errorf("Lookup(%q) = %q, want %q", id, got, path)
		}
	}
}

// TestResolverRejectsAbsentAndMalformed proves an id that names no record does
// not resolve, and that a well-formed-but-absent id is reported the same way as
// a malformed one — the caller decides what to do with either.
func TestResolverRejectsAbsentAndMalformed(t *testing.T) {
	root := seedRecords(t)
	r, err := NewResolver(root)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	for _, id := range []string{"itd-9999", "adr-2", "spc-1", "iss-1", "itd-0007", "", "itd", "ITD-7", "itd-7-a-draft"} {
		if path, ok := r.Lookup(id); ok {
			t.Errorf("Lookup(%q) resolved to %q; want unresolved", id, path)
		}
	}
}

// TestResolverEmptyRepo proves a repository with no record tree at all is not an
// error: there are simply no ids to resolve.
func TestResolverEmptyRepo(t *testing.T) {
	r, err := NewResolver(t.TempDir())
	if err != nil {
		t.Fatalf("NewResolver over an empty repo: %v", err)
	}
	if n := r.Len(); n != 0 {
		t.Errorf("Len() = %d over an empty repo, want 0", n)
	}
}

// TestResolverIgnoresSymlinkedEntries proves the scan never follows a symlink,
// so a planted link cannot make a record appear to exist (nor make the scan walk
// out of the repository).
func TestResolverIgnoresSymlinkedEntries(t *testing.T) {
	root := seedRecords(t)
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "itd-500-planted.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, ".abcd/development/intents/drafts/itd-500-planted.md")
	if err := os.Symlink(filepath.Join(outside, "itd-500-planted.md"), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	dirLink := filepath.Join(root, ".abcd/development/intents/elsewhere")
	if err := os.Symlink(outside, dirLink); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	r, err := NewResolver(root)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	if path, ok := r.Lookup("itd-500"); ok {
		t.Errorf("a symlinked record resolved to %q; want unresolved", path)
	}
}

// TestCitedIDReGrammar pins the cited-id grammar: the four id-bearing families,
// bare decimal N, nothing else.
func TestCitedIDReGrammar(t *testing.T) {
	for _, ok := range []string{"itd-1", "iss-146", "spc-18", "adr-37"} {
		if !CitedIDRe.MatchString(ok) {
			t.Errorf("CitedIDRe rejected %q", ok)
		}
	}
	for _, bad := range []string{"ITD-1", "itd_1", "itd-", "itd-1-slug", "prn-1", "fnd-1", "../itd-1", "itd-1 "} {
		if CitedIDRe.MatchString(bad) {
			t.Errorf("CitedIDRe accepted %q", bad)
		}
	}
}

// TestResolverAdmitsBothADRVintages is the 2026-09-01 ruling at the read side:
// the ADR store now holds two id vintages side by side — the hand-numbered
// ordinals 0001–0058, which keep their ids and filenames, and the minted
// timestamp form — and one resolver answers for both. Nothing here parses the
// filename as anything but a run of digits, so a 4-digit name and a 16-digit
// one are judged on the same axis.
func TestResolverAdmitsBothADRVintages(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{
		".abcd/development/decisions/adrs/0058-a-reading-is-commissioned.md",
		".abcd/development/decisions/adrs/2609012206053814-adrs-mint-timestamp-ids.md",
	} {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte("# record\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r, err := NewResolver(root)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	for id, want := range map[string]string{
		"adr-58":               ".abcd/development/decisions/adrs/0058-a-reading-is-commissioned.md",
		"adr-2609012206053814": ".abcd/development/decisions/adrs/2609012206053814-adrs-mint-timestamp-ids.md",
	} {
		got, ok := r.Lookup(id)
		if !ok {
			t.Errorf("%s did not resolve", id)
			continue
		}
		if got != want {
			t.Errorf("%s resolved to %q, want %q", id, got, want)
		}
	}
	// The cited-id grammar every ingest boundary bounds a citation with must
	// admit the minted spelling too, or a cite of a fresh decision is refused
	// before it is ever looked up.
	if !CitedIDRe.MatchString("adr-2609012206053814") {
		t.Error("CitedIDRe refuses a minted ADR id")
	}
}

// TestADRFileIDIsTheOneDerivation pins the shared ADR filename→id derivation:
// the ordinal's padding is trimmed, the stamp is passed through, and a name that
// is not an ADR record yields nothing. It is exported because the mint's
// presence check and the read-side resolver must judge exactly the same files.
func TestADRFileIDIsTheOneDerivation(t *testing.T) {
	cases := map[string]string{
		"0001-three-layer-mental-model.md":            "adr-1",
		"0058-a-reading-is-commissioned.md":           "adr-58",
		"2609012206053814-adrs-mint-timestamp-ids.md": "adr-2609012206053814",
		"README.md":    "",
		"0000-zero.md": "",
		"adr-58-x.md":  "",
		"0058-a-slug":  "",
	}
	for name, want := range cases {
		if got := ADRFileID(name); got != want {
			t.Errorf("ADRFileID(%q) = %q, want %q", name, got, want)
		}
	}
}
