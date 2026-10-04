package history

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// TestLegacyRecordMovedIntoTheStoreIsOwnerOnly: a record an earlier binary wrote
// 0o644 at the legacy location arrives in the store owner-only, not carrying its
// old mode through the rename.
func TestLegacyRecordMovedIntoTheStoreIsOwnerOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	legacyRecords := abcdhome.Path(home, "history", testRootSHA, legacyRecordsDirName)
	if err := os.MkdirAll(legacyRecords, 0o755); err != nil {
		t.Fatal(err)
	}
	const name = "20260101T000000.000000000Z-sess-legacy.md"
	src := filepath.Join(legacyRecords, name)
	if err := os.WriteFile(src, []byte("---\nsession_id: sess-legacy\n---\nassistant: hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(src, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Resolve(t.TempDir(), testRootSHA)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	moved := filepath.Join(res.Records, name)
	if got := permOf(t, moved); got != 0o600 {
		t.Errorf("the moved legacy record is mode %#o, want 0o600", got)
	}
}
