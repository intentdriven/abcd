package decide

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCreateRefusesASymlinkedAncestor: the mint lock refused a symlinked
// LEAF and then called os.MkdirAll, which follows a symlinked ANCESTOR, so a
// committed `.abcd/development -> <elsewhere>` put the ADR outside the
// checkout. Found on the sweep for iss-2609012037137250: the same create
// sequence the spec store carried.
func TestCreateRefusesASymlinkedAncestor(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, ".abcd", "development")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	if d, err := Create(root, "A decision minted through a planted link"); err == nil {
		t.Fatalf("Create minted %s through a symlinked .abcd/development", d.Path)
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("the refused mint still created %d entr(y|ies) under the symlink's target", len(entries))
	}
}
