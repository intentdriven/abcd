package decisions

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRecordCutsTheSlugAtTheRepositorysCap is iss-2611052241331003 at the decision
// writer: the slug meets records.slug_cap, and a nonsense value writes nothing.
func TestRecordCutsTheSlugAtTheRepositorysCap(t *testing.T) {
	root := sharedTierRepo(t)
	if err := os.MkdirAll(filepath.Join(root, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(root, ".abcd", "config.json")
	if err := os.WriteFile(cfg, []byte(`{"records": {"slug_cap": 20}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := Record(root, "A record's file name has no cap tied to the length of its full path")
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if d.Slug != "a-record-s-file-name" {
		t.Fatalf("slug = %q", d.Slug)
	}
	if err := os.WriteFile(cfg, []byte(`{"records": {"slug_cap": 0}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Record(root, "Another decision"); err == nil {
		t.Fatal("a nonsense cap recorded a decision")
	}
}
