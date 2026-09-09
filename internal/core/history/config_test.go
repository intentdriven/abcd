package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeHistoryConfig plants .abcd/config/history.json in a repo root.
func writeHistoryConfig(t *testing.T, repoRoot, body string) {
	t.Helper()
	dir := filepath.Join(repoRoot, ".abcd", "config")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "history.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestLoadConfigDefaultsWhenAbsent: no file is not a fault. It is a repository
// that has not declared anything, which is every repository until it does.
func TestLoadConfigDefaultsWhenAbsent(t *testing.T) {
	cfg, err := LoadConfig(t.TempDir())
	if err != nil {
		t.Fatalf("an absent configuration must not be an error: %v", err)
	}
	if cfg.OnOrphan != OnOrphanIgnore {
		t.Errorf("the default orphan policy must be %q, got %q", OnOrphanIgnore, cfg.OnOrphan)
	}
	if len(cfg.IngestRoots) != 0 || len(cfg.AdoptProjects) != 0 {
		t.Errorf("a repository that declared nothing must claim nothing, got %+v", cfg)
	}
}

// TestLoadConfigReadsTheDeclaredRootsAndClaims.
func TestLoadConfigReadsTheDeclaredRootsAndClaims(t *testing.T) {
	repo := t.TempDir()
	writeHistoryConfig(t, repo, `{"schema_version":1,"ingest_roots":["/somewhere/transcripts"],"adopt_projects":["a-project"],"on_orphan":"prompt"}`)
	cfg, err := LoadConfig(repo)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(cfg.IngestRoots) != 1 || cfg.IngestRoots[0] != "/somewhere/transcripts" {
		t.Errorf("ingest roots = %+v", cfg.IngestRoots)
	}
	if len(cfg.AdoptProjects) != 1 || cfg.AdoptProjects[0] != "a-project" {
		t.Errorf("adopt projects = %+v", cfg.AdoptProjects)
	}
	if cfg.OnOrphan != OnOrphanPrompt {
		t.Errorf("on_orphan = %q", cfg.OnOrphan)
	}
}

// TestLoadConfigRefusesAnUnknownOrphanPolicy: a policy nobody implements is
// not a policy, and silently falling back to the default would ingest under
// terms the operator did not choose.
func TestLoadConfigRefusesAnUnknownOrphanPolicy(t *testing.T) {
	repo := t.TempDir()
	writeHistoryConfig(t, repo, `{"on_orphan":"adopt-everything"}`)
	if _, err := LoadConfig(repo); err == nil {
		t.Fatal("an unknown on_orphan must be refused")
	} else if !strings.Contains(err.Error(), "on_orphan") {
		t.Errorf("the refusal must name the field, got %q", err)
	}
}

// TestLoadConfigRefusesMalformedJSON.
func TestLoadConfigRefusesMalformedJSON(t *testing.T) {
	repo := t.TempDir()
	writeHistoryConfig(t, repo, `{"ingest_roots":`)
	if _, err := LoadConfig(repo); err == nil {
		t.Fatal("a configuration that cannot be parsed must be refused, not ignored")
	}
}

// TestLoadConfigRefusesASymlinkedLeaf: the file lives two directories down, so
// the read is contained in the repository root rather than guarded on the leaf
// alone.
func TestLoadConfigRefusesASymlinkedLeaf(t *testing.T) {
	repo := t.TempDir()
	outside := filepath.Join(t.TempDir(), "elsewhere.json")
	if err := os.WriteFile(outside, []byte(`{"adopt_projects":["stolen"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(repo, ".abcd", "config")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "history.json")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := LoadConfig(repo); err == nil {
		t.Fatal("a symlinked configuration leaf must be refused")
	}
}
