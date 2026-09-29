package scribe

import (
	"os"
	"path/filepath"
	"testing"
)

// TestScribeAssembleRefusesASymlinkedLocalTier: the default run directory is
// relative, inside the checkout's local tier, and was created with os.MkdirAll,
// which follows a symlinked ancestor — a hostile clone that force-adds
// `.abcd/.work.local` as a link parks the context and the manifest wherever the
// link points. The reading assembler proves each level of its in-repo run
// directory (iss-2609262231500173); found on the sweep for iss-2609012037137250,
// the scribe's is proved the same way.
func TestScribeAssembleRefusesASymlinkedLocalTier(t *testing.T) {
	f := newFixture(t, positionDetection, 1)
	outside := t.TempDir()
	local := filepath.Join(f.repo, ".abcd", ".work.local")
	if err := os.RemoveAll(local); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, local); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	if _, err := Assemble(AssembleRequest{RepoRoot: f.repo, Run: fixtureRun,
		DispositionsPath: supply(t, suppliedText)}); err == nil {
		t.Fatal("Assemble parked the run through a symlinked .abcd/.work.local")
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("the refused assembly still created %d entr(y|ies) under the symlink's target", len(entries))
	}
}
