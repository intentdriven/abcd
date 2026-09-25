package capture

import (
	"os"
	"path/filepath"
	"testing"
)

// TestStatusSkipNamesTheRefusingLayerAndIsCounted is the remaining half of
// iss-2609120452071388: a record the reader refuses is excluded from every
// total, so the board must COUNT what it excluded and SAY which layer refused
// each one. A reader told only "skipped … malformed frontmatter" cannot tell
// whether the writer or the validator is the side that is wrong, and a count
// that silently omits the record under-reports the ledger.
//
// One planted record per layer, each wrong in exactly one way, so the layer
// reported can be attributed to nothing else.
func TestStatusSkipNamesTheRefusingLayerAndIsCounted(t *testing.T) {
	repo, ir := ledger(t)
	if _, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "a well formed finding", Severity: SeverityMinor,
		Category: "bug", Source: "manual-test", Slug: "fine", FoundDuring: "t",
	}); err != nil {
		t.Fatal(err)
	}
	valid := func(id, slug string) string {
		return "---\nschema_version: 1\nid: \"" + id + "\"\nslug: \"" + slug + "\"\n" +
			"severity: \"minor\"\ncategory: \"bug\"\nsource: \"manual-test\"\nfound_during: \"t\"\n"
	}
	open := filepath.Join(ir, "open")
	plant := map[string]string{
		// name: the filename claims a record and is not a well-formed one.
		"iss-11-bad_name.md": valid("iss-11", "bad-name") + "---\n\nbody\n",
		// frontmatter: no closing fence, so nothing parses.
		"iss-12-unclosed.md": valid("iss-12", "unclosed") + "\nbody with no closer\n",
		// schema: parses, and carries a source outside the closed vocabulary.
		"iss-13-bad-source.md": "---\nschema_version: 1\nid: \"iss-13\"\nslug: \"bad-source\"\n" +
			"severity: \"minor\"\ncategory: \"bug\"\nsource: \"autonomous-hunt\"\nfound_during: \"t\"\n---\n\nbody\n",
		// invariant: schema-clean, but the filename slug disagrees with the field.
		"iss-14-other-slug.md": valid("iss-14", "the-real-slug") + "---\n\nbody\n",
	}
	for name, content := range plant {
		if err := os.WriteFile(filepath.Join(open, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	st, err := Status(StatusRequest{RepoRoot: repo, IssuesRoot: ir})
	if err != nil {
		t.Fatal(err)
	}
	if st.OpenCount != 1 {
		t.Fatalf("open count = %d, want the 1 readable record", st.OpenCount)
	}
	if st.SkippedCount != len(plant) || len(st.Skipped) != st.SkippedCount {
		t.Fatalf("skipped_count = %d with %d roster entries; want both %d, so the count and the roster agree",
			st.SkippedCount, len(st.Skipped), len(plant))
	}
	want := map[string]SkipLayer{
		"iss-11-bad_name.md":   SkipLayerName,
		"iss-12-unclosed.md":   SkipLayerFrontmatter,
		"iss-13-bad-source.md": SkipLayerSchema,
		"iss-14-other-slug.md": SkipLayerInvariant,
	}
	for _, sk := range st.Skipped {
		base := filepath.Base(sk.Path)
		if sk.Layer != want[base] {
			t.Errorf("%s skipped by layer %q, want %q (error: %s)", base, sk.Layer, want[base], sk.Error)
		}
	}
}

// TestListSkipOfAnUnreadableLeafNamesTheReadLayer pins the fifth layer: a leaf
// the guarded read refuses (here a dangling symlink) is a read refusal, not a
// statement about the record's content.
func TestListSkipOfAnUnreadableLeafNamesTheReadLayer(t *testing.T) {
	repo, ir := ledger(t)
	if err := ensureLedgerDirs(repo, ir); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(ir, "open", "nowhere.md"), filepath.Join(ir, "open", "iss-1-broken.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	res, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateOpen})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Skipped) != 1 || res.Skipped[0].Layer != SkipLayerRead {
		t.Fatalf("want one skip by the read layer, got %+v", res.Skipped)
	}
}
