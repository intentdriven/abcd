package spec

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCreateCutsTheSlugAtTheRepositorysCap is iss-2611052241331003 at the spec
// mint: a slug carried from an intent is cut at records.slug_cap, and a nonsense
// value refuses.
func TestCreateCutsTheSlugAtTheRepositorysCap(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(root, ".abcd", "config.json")
	if err := os.WriteFile(cfg, []byte(`{"records": {"slug_cap": 20}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	sp, err := Create(root, "itd-9", "a-record-s-file-name-has-no-cap-tied-to-the-length-of-its", "")
	if err != nil {
		t.Fatal(err)
	}
	if sp.Slug != "a-record-s-file-name" {
		t.Fatalf("slug = %q", sp.Slug)
	}
	if err := os.WriteFile(cfg, []byte(`{"records": {"slug_cap": 9000}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(root, "itd-9", "another", ""); err == nil {
		t.Fatal("a cap of 9000 minted a spec")
	}
}
