package memory

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestStoreHandleReadsOnlyTheDirectoryItOpened pins iss-2608291814572914:
// containment is a property of the handle, not a check each verb remembers.
// A store directory replaced by a symlink AFTER the handle was opened must not
// redirect a single read: the handle keeps reading the directory it vetted.
func TestStoreHandleReadsOnlyTheDirectoryItOpened(t *testing.T) {
	root := t.TempDir()
	mem := filepath.Join(root, ".abcd", "memory")
	if err := os.MkdirAll(mem, 0o755); err != nil {
		t.Fatal(err)
	}
	page := "---\nsource: sha256:0000\n---\n\nbody\n"
	if err := os.WriteFile(filepath.Join(mem, "fact_eng_inside.md"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "fact_eng_outside.md"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := openStore(root)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	// Swap the vetted directory for a symlink to one outside the repository.
	if err := os.Rename(mem, filepath.Join(root, ".abcd", "memory.moved")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, mem); err != nil {
		t.Fatal(err)
	}

	for name := range existingPageFrontmatter(store) {
		if name == "fact_eng_outside.md" {
			t.Fatal("a read through the handle followed a store directory swapped in after it was opened")
		}
	}
	if _, err := store.read("fact_eng_outside.md", maxMemoryPageBytes); err == nil {
		t.Fatal("the handle read a page from outside the directory it opened")
	}
	if _, err := store.read("fact_eng_inside.md", maxMemoryPageBytes); err != nil {
		t.Fatalf("the handle lost the directory it opened: %v", err)
	}
}

// TestFileBackRefusesASymlinkedStore: fileBack read the existing pages and the
// registry through Dir(root) with no check at all, so only the later write
// refused a symlinked store, after the registry beyond it had been read. It
// now opens the store handle first, and a symlinked store is refused before
// anything under it is read.
func TestFileBackRefusesASymlinkedStore(t *testing.T) {
	repo := t.TempDir()
	seedAskStore(t, repo)
	matches, err := QueryPages(repo, "how does token rotation work?", AskTopN)
	if err != nil || len(matches) == 0 {
		t.Fatalf("fixture: QueryPages = %d matches, %v", len(matches), err)
	}
	// The store becomes a symlink to a directory whose registry does not parse:
	// reaching it surfaces as a registry format error, not as the refusal.
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, ".sources_index.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	mem := Dir(repo)
	if err := os.Rename(mem, mem+".moved"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, mem); err != nil {
		t.Fatal(err)
	}
	page := map[string]any{
		"type": "topic", "domain": "auth", "slug": "rotation-summary",
		"body": "# Rotation summary\nRotate daily.\n",
	}
	_, err = fileBack(repo, matches, page, nil, fixedNow)
	var unsafe *UnsafeStorePathError
	if !errors.As(err, &unsafe) {
		t.Fatalf("fileBack on a symlinked store = %v (%T); want *UnsafeStorePathError before any read", err, err)
	}
}
