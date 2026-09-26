package lint

import (
	"os"
	"path/filepath"
	"testing"
)

// specLinkRepo plants a minimal record tree: two intents in different buckets,
// each linked to a spec in a different lifecycle bucket.
func specLinkRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("record/intents/planned/itd-94-gate.md", "---\nid: itd-94\nkind: standalone\nspec_id: spc-9\n---\n# gate\n")
	write("record/intents/shipped/itd-80-lifecycle.md", "---\nid: itd-80\nkind: standalone\nspec_id: spc-2\n---\n# lifecycle\n")
	write("record/intents/drafts/itd-95-idea.md", "---\nid: itd-95\nkind: null\nspec_id: null\n---\n# idea\n")
	write("record/intents/planned/README.md", "# planned\n")
	write("record/specs/open/spc-9-gate.md", "---\nid: spc-9\nslug: gate\nintent: itd-94\n---\n# spc-9\n")
	write("record/specs/closed/spc-2-lifecycle.md", "---\nid: spc-2\nslug: lifecycle\nintent: itd-80\n---\n# spc-2\n")
	return root
}

// TestScanSpecLinks pins the ONE traversal of the intent buckets and the spec
// store that both the spec-lifecycle lint and the release cut read. The cut's
// fail-closed refusal (a merged feature whose intent is still in planned/ while
// its spec has closed) can only be asked of an index that carries BOTH sides'
// buckets, so that is what the scan must return.
func TestScanSpecLinks(t *testing.T) {
	root := specLinkRepo(t)
	idx, err := ScanSpecLinks(root, "record/intents", "record/specs", Config{})
	if err != nil {
		t.Fatalf("ScanSpecLinks: %v", err)
	}

	wantIntents := map[string]IntentLink{
		"itd-94": {ID: "itd-94", Bucket: "planned", Path: "record/intents/planned/itd-94-gate.md", SpecID: "spc-9"},
		"itd-80": {ID: "itd-80", Bucket: "shipped", Path: "record/intents/shipped/itd-80-lifecycle.md", SpecID: "spc-2"},
		"itd-95": {ID: "itd-95", Bucket: "drafts", Path: "record/intents/drafts/itd-95-idea.md", SpecID: "null"},
	}
	if len(idx.Intents) != len(wantIntents) {
		t.Fatalf("scanned %d intents (%+v), want %d", len(idx.Intents), idx.Intents, len(wantIntents))
	}
	for _, got := range idx.Intents {
		want, ok := wantIntents[got.ID]
		if !ok {
			t.Fatalf("unexpected intent %q", got.ID)
		}
		if got != want {
			t.Errorf("intent %s = %+v, want %+v", got.ID, got, want)
		}
	}

	for _, tc := range []struct{ specID, wantBucket string }{
		{"spc-9", "open"},
		{"spc-2", "closed"},
		{"spc-2-lifecycle", "closed"}, // a spec_id written with its slug still resolves
	} {
		bucket, found := idx.SpecBucket(tc.specID)
		if !found || bucket != tc.wantBucket {
			t.Errorf("SpecBucket(%q) = %q,%v, want %q,true", tc.specID, bucket, found, tc.wantBucket)
		}
	}
	if _, found := idx.SpecBucket("spc-404"); found {
		t.Error("SpecBucket resolved a spec that does not exist")
	}
}

// TestScanSpecLinksMissingTreesAreSoft mirrors the rest of the record lint: a
// repository without a record tree contributes nothing and is not an error, so a
// consumer never has to special-case an unpopulated repo.
func TestScanSpecLinksMissingTreesAreSoft(t *testing.T) {
	idx, err := ScanSpecLinks(t.TempDir(), "record/intents", "record/specs", Config{})
	if err != nil {
		t.Fatalf("ScanSpecLinks: %v", err)
	}
	if len(idx.Intents) != 0 || len(idx.Specs) != 0 {
		t.Errorf("scanned %+v, want an empty index", idx)
	}
}

// TestScanSpecLinksUnreadableIntentIsHard proves the intent half fails closed
// exactly as the spec half does: a record that lists but cannot be read is a
// fault, not an absence. Swallowing it would hand the release cut's stale-intent
// refusal an index that says "no intents" about a tree it could not read.
func TestScanSpecLinksUnreadableIntentIsHard(t *testing.T) {
	root := specLinkRepo(t)
	planned := filepath.Join(root, "record", "intents", "planned")
	if err := os.Symlink(filepath.Join(planned, "gone.md"), filepath.Join(planned, "itd-96-dangling.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := ScanSpecLinks(root, "record/intents", "record/specs", Config{}); err == nil {
		t.Fatal("ScanSpecLinks must propagate an unreadable intent record")
	}
}

// A bundle's shared spec realises every member its `intents:` list names, not
// only the one its `intent:` names, so the index answers "which specs realise
// this intent" as the spec store does (itd-34).
func TestScanSpecLinksBundleSpecRealisesEveryMember(t *testing.T) {
	root := t.TempDir()
	for rel, content := range map[string]string{
		"record/intents/planned/itd-20-a.md": "---\nid: itd-20\nkind: bundle-member\nbundle: pair\nspec_id: spc-5\n---\n# a\n",
		"record/intents/planned/itd-21-b.md": "---\nid: itd-21\nkind: bundle-member\nbundle: pair\nspec_id: spc-5\n---\n# b\n",
		"record/specs/open/spc-5-pair.md":    "---\nid: spc-5\nslug: pair\nintent: itd-20\nintents: [itd-20, itd-21]\nbundle: pair\n---\n# pair\n",
	} {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	idx, err := ScanSpecLinks(root, "record/intents", "record/specs", Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"itd-20", "itd-21"} {
		if got := idx.SpecsForIntent(id); len(got) != 1 || got[0].ID != "spc-5" {
			t.Errorf("SpecsForIntent(%s) = %+v, want the shared spc-5", id, got)
		}
	}
}
