package decide

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCreateCutsTheSlugAtTheRepositorysCap is iss-2611052241331003 at the ADR
// mint: the slug is cut at records.slug_cap, and a nonsense value refuses.
func TestCreateCutsTheSlugAtTheRepositorysCap(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(root, ".abcd", "config.json")
	if err := os.WriteFile(cfg, []byte(`{"records": {"slug_cap": 20}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	title := "A record's file name has no cap tied to the length of its full path"
	d, err := Create(root, title)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if d.Slug != "a-record-s-file-name" {
		t.Fatalf("slug = %q", d.Slug)
	}
	if err := os.WriteFile(cfg, []byte(`{"records": {"slug_cap": -1}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(root, title); err == nil {
		t.Fatal("a cap of -1 minted an ADR")
	}
}
