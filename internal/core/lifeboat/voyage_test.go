package lifeboat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// TestPackAppendsVoyageForGitSource: a git-backed source (has a root-commit SHA)
// records one append-only voyage line keyed on that SHA, carrying the manifest
// hash that ties it to the lifeboat's provenance.
func TestPackAppendsVoyageForGitSource(t *testing.T) {
	repo := packFixture(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	dest := filepath.Join(t.TempDir(), "lb")
	res, err := Pack(repo, dest, okScan)
	if err != nil {
		t.Fatal(err)
	}
	if !res.VoyageAppended {
		t.Fatalf("voyage not appended: note=%q", res.VoyageNote)
	}

	rootSHA := res.ManifestSHA256 // placeholder; real key is source root sha
	_ = rootSHA
	// Find the single ledger under ~/.abcd/voyage/<sha>/disembark/history.jsonl.
	var ledger string
	err = filepath.Walk(abcdhome.Path(home, "voyage"), func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && filepath.Base(p) == "history.jsonl" {
			ledger = p
		}
		return nil
	})
	if err != nil || ledger == "" {
		t.Fatalf("voyage ledger not found: %v", err)
	}
	data, err := os.ReadFile(ledger)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 ledger line, got %d", len(lines))
	}
	var e voyageEntry
	if err := json.Unmarshal([]byte(lines[0]), &e); err != nil {
		t.Fatalf("ledger line is not a voyage entry: %v", err)
	}
	if e.Event != "disembark" || e.ManifestSHA256 != res.ManifestSHA256 {
		t.Errorf("ledger entry mismatch: %+v vs manifest %s", e, res.ManifestSHA256)
	}
	if !rootSHARe.MatchString(e.SourceRootSHA) {
		t.Errorf("ledger source_root_sha is not a 40-hex SHA: %q", e.SourceRootSHA)
	}
	// The ledger path must be keyed on that same root SHA.
	if filepath.Base(filepath.Dir(filepath.Dir(ledger))) != e.SourceRootSHA {
		t.Errorf("ledger not keyed on source_root_sha: path=%s key=%s", ledger, e.SourceRootSHA)
	}
}

// TestPackVoyageAppendsNotRewrites: a second pack appends a second line rather
// than rewriting the ledger.
func TestPackVoyageAppendsNotRewrites(t *testing.T) {
	repo := packFixture(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	dest := filepath.Join(t.TempDir(), "lb")
	for i := 0; i < 2; i++ {
		if _, err := Pack(repo, dest, okScan); err != nil {
			t.Fatalf("pack %d: %v", i, err)
		}
	}
	var ledger string
	filepath.Walk(abcdhome.Path(home, "voyage"), func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && filepath.Base(p) == "history.jsonl" {
			ledger = p
		}
		return nil
	})
	data, _ := os.ReadFile(ledger)
	if n := len(strings.Split(strings.TrimSpace(string(data)), "\n")); n != 2 {
		t.Errorf("voyage ledger has %d lines after 2 packs, want 2 (append-only)", n)
	}
}

// TestPackVoyageRefusesSymlinkedBase: a symlinked ~/.abcd/voyage is refused, and
// crucially no directories are created under the symlink target — the real-dir
// guard runs before any mkdir. The pack itself still succeeds (voyage is
// non-fatal).
func TestPackVoyageRefusesSymlinkedBase(t *testing.T) {
	repo := packFixture(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	abcd := abcdhome.Path(home)
	if err := os.MkdirAll(abcd, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(abcd, "voyage")); err != nil {
		t.Skipf("cannot symlink: %v", err)
	}
	dest := filepath.Join(t.TempDir(), "lb")
	res, err := Pack(repo, dest, okScan)
	if err != nil {
		t.Fatalf("pack must still succeed when voyage is refused: %v", err)
	}
	if res.VoyageAppended {
		t.Error("voyage must refuse a symlinked base")
	}
	if !strings.Contains(res.VoyageNote, "real directory") {
		t.Errorf("want a 'not a real directory' note, got %q", res.VoyageNote)
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Errorf("directories were created under the symlinked voyage target: %v", entries)
	}
}

// TestPackSkipsVoyageWhenNoRootSHA: a non-git source has no root-commit SHA to
// key a voyage, so the pack succeeds but records no ledger line, with a reason.
func TestPackSkipsVoyageWhenNoRootSHA(t *testing.T) {
	repo := nativeTierFixture(t) // not a git repo → RootSHA is empty
	home := t.TempDir()
	t.Setenv("HOME", home)
	dest := filepath.Join(t.TempDir(), "lb")
	res, err := Pack(repo, dest, okScan)
	if err != nil {
		t.Fatal(err)
	}
	if res.VoyageAppended {
		t.Error("a source with no root SHA must not be voyage-logged")
	}
	if !strings.Contains(res.VoyageNote, "root-commit SHA") {
		t.Errorf("want a 'no root-commit SHA' note, got %q", res.VoyageNote)
	}
	if _, err := os.Stat(abcdhome.Path(home, "voyage")); !os.IsNotExist(err) {
		t.Errorf("no voyage directory should be created for an unkeyable source")
	}
}
