package intent

import (
	"path/filepath"
	"strings"
	"testing"
)

// blockerRecord is a minimal intent record for the blocked check's corpus: an
// id, a slug and, for a superseded one, the successor it names.
func blockerRecord(id, supersededBy string) string {
	s := "---\nid: " + id + "\nslug: " + strings.ReplaceAll(id, "itd-", "rec-") + "\nkind: standalone\n"
	if supersededBy != "" {
		s += "superseded_by: " + supersededBy + "\nkind_at_supersession: standalone\n"
	}
	return s + "---\n# " + id + "\n"
}

// TestStartBlockedRowFollowsASupersededBlockerToItsReplacement is ruling BZ2
// of 2026-09-29: a blocker that was superseded is followed along
// `superseded_by` to the intent that replaced it, transitively, and the intent
// waits on that replacement. It is blocked exactly when the last intent of the
// chain has not shipped; a chain that loops, ends at a record this checkout
// does not hold, names no successor, or ends at a decision rather than an
// intent refuses naming the chain.
func TestStartBlockedRowFollowsASupersededBlockerToItsReplacement(t *testing.T) {
	type rec struct{ bucket, id, by string }
	cases := []struct {
		name    string
		records []rec
		ok      bool
		want    []string
	}{
		{"replaced by a shipped intent", []rec{
			{BucketSuperseded, "itd-27", "itd-94"},
			{BucketShipped, "itd-94", ""},
		}, true, []string{"itd-27 → itd-94"}},
		{"replaced by an unshipped intent", []rec{
			{BucketSuperseded, "itd-27", "itd-94"},
			{BucketPlanned, "itd-94", ""},
		}, false, []string{"itd-27 → itd-94", "planned"}},
		{"replaced twice, the last shipped", []rec{
			{BucketSuperseded, "itd-27", "itd-94"},
			{BucketSuperseded, "itd-94", "itd-95"},
			{BucketShipped, "itd-95", ""},
		}, true, []string{"itd-27 → itd-94 → itd-95"}},
		{"replaced twice, the last a draft", []rec{
			{BucketSuperseded, "itd-27", "itd-94"},
			{BucketSuperseded, "itd-94", "itd-95"},
			{BucketDrafts, "itd-95", ""},
		}, false, []string{"itd-27 → itd-94 → itd-95", "drafts"}},
		{"a supersession cycle", []rec{
			{BucketSuperseded, "itd-27", "itd-94"},
			{BucketSuperseded, "itd-94", "itd-27"},
		}, false, []string{"itd-27 → itd-94 → itd-27", "cycle"}},
		{"a replacement this checkout does not hold", []rec{
			{BucketSuperseded, "itd-27", "itd-94"},
		}, false, []string{"itd-27 → itd-94", "not in this checkout's intent store"}},
		{"a superseded record naming no successor", []rec{
			{BucketSuperseded, "itd-27", "null"},
		}, false, []string{"itd-27", "names no successor"}},
		{"replaced by a decision", []rec{
			{BucketSuperseded, "itd-27", "adr-37"},
		}, false, []string{"itd-27 → adr-37", "not an intent"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for _, r := range tc.records {
				writeFile(t, root, filepath.Join(IntentsRelDir, r.bucket, r.id+"-"+strings.ReplaceAll(r.id, "itd-", "rec-")+".md"), blockerRecord(r.id, r.by))
			}
			corpus, err := Load(root)
			if err != nil {
				t.Fatal(err)
			}
			row := startBlockedRow(corpus, "itd-10", "---\nid: itd-10\nblocked_by: [itd-27]\n---\n")
			if row.OK != tc.ok {
				t.Fatalf("want OK=%v, got %+v", tc.ok, row)
			}
			for _, w := range tc.want {
				if !strings.Contains(row.Detail, w) {
					t.Errorf("the detail must name %q: %q", w, row.Detail)
				}
			}
			if !tc.ok && !strings.Contains(row.Remedy, "superseded_by") {
				t.Errorf("the remedy must say how to repair or follow the chain: %q", row.Remedy)
			}
		})
	}
}

// TestStartBlockedRowKeepsAPlainUnshippedBlocker holds the unchanged half: a
// blocker that was not superseded blocks until it ships, and one this checkout
// does not hold blocks too.
func TestStartBlockedRowKeepsAPlainUnshippedBlocker(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, filepath.Join(IntentsRelDir, BucketPlanned, "itd-27-rec-27.md"), blockerRecord("itd-27", ""))
	corpus, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	row := startBlockedRow(corpus, "itd-10", "---\nid: itd-10\nblocked_by: [itd-27, itd-99]\n---\n")
	if row.OK {
		t.Fatalf("an unshipped blocker blocks: %+v", row)
	}
	for _, w := range []string{"itd-27 (planned)", "itd-99 (not in this checkout's intent store)"} {
		if !strings.Contains(row.Detail, w) {
			t.Errorf("the detail must name %q: %q", w, row.Detail)
		}
	}
}
