package recordid

import (
	"io/fs"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// TestResolverFromPathsAgreesWithTheScan holds the listing reader to the
// working-tree scan (iss-2611031731385043): a record another branch carries
// must resolve by the same derivation as one in this tree, so the two readers
// of one tree give one answer, id for id and path for path.
func TestResolverFromPathsAgreesWithTheScan(t *testing.T) {
	root := seedRecords(t)
	var paths []string
	if err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			paths = append(paths, filepath.ToSlash(rel))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	scan, err := NewResolver(root)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	listed, err := NewResolverFromPaths(paths)
	if err != nil {
		t.Fatalf("NewResolverFromPaths: %v", err)
	}
	if !reflect.DeepEqual(scan.ids, listed.ids) {
		t.Errorf("the listing reader disagrees with the scan:\nscan:    %v\nlisting: %v", scan.ids, listed.ids)
	}
}

// TestResolverFromPathsReadsOnlyWhereTheScanReads: the scan reads a family
// root and one bucket below it, never deeper and never outside the families,
// so a listing that names a record-shaped file anywhere else names no record.
func TestResolverFromPathsReadsOnlyWhereTheScanReads(t *testing.T) {
	r, err := NewResolverFromPaths([]string{
		".abcd/work/decisions/dcn-2611031615094092-a-ruling.md",              // flat root
		".abcd/work/issues/open/iss-41-a-finding.md",                         // one bucket below
		".abcd/work/issues/open/nested/iss-42-too-deep.md",                   // two below: not read
		".abcd/work/issues-archive/iss-43-a-sibling-dir.md",                  // a sibling, not the family
		"docs/iss-44-outside.md",                                             // outside every family
		".abcd/development/decisions/adrs/2611031615094092-a-decision.md",    // ADR, timestamp vintage
		".abcd/development/intents/planned/not-a-record.md",                  // no id
		".abcd/development/specs/open/spc-0-zero-is-no-id.md",                // id zero names nothing
		".abcd/development/intents/planned/itd-2611031615094092-intent.md",   // planned intent
		".abcd/development/specs/closed/spc-2611031615094092-closed-spec.md", // closed spec
	})
	if err != nil {
		t.Fatalf("NewResolverFromPaths: %v", err)
	}
	var got []string
	for id := range r.ids {
		got = append(got, id)
	}
	sort.Strings(got)
	want := []string{"adr-2611031615094092", "dcn-2611031615094092", "iss-41", "itd-2611031615094092", "spc-2611031615094092"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ids = %v, want %v", got, want)
	}
	if p, ok, err := r.Lookup("iss-41"); err != nil || !ok || p != ".abcd/work/issues/open/iss-41-a-finding.md" {
		t.Errorf("Lookup(iss-41) = %q, %v, %v; want the bucketed path", p, ok, err)
	}
}

// TestFamilyRootDirsNamesEveryResolvedFamily: the pathspecs a tree reader lists
// are exactly the roots the resolver reads, so nothing the resolver would find
// is left out of a listing.
func TestFamilyRootDirsNamesEveryResolvedFamily(t *testing.T) {
	got := FamilyRootDirs()
	want := []string{".abcd/development/intents", ".abcd/development/specs", IssuesRelDir, ADRsRelDir, DecisionsRelDir}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FamilyRootDirs = %v, want %v", got, want)
	}
}
