package worktree_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestPruneLeavesTheTiersBuiltBinaryOutOfTheNotes: the binary scripts/abcd.sh
// builds into the tier's bin/ is a build product, past the per-file cap and
// not text, so the archive leaves it out instead of keeping the worktree
// (iss-2611022247131866). The tier's notes beside it are still archived, and a
// bin/ deeper in the tier is still judged as notes.
func TestPruneLeavesTheTiersBuiltBinaryOutOfTheNotes(t *testing.T) {
	repo, home := pruneFixture(t)
	p := mergedWorktree(t, repo, "built")
	tier := filepath.Join(p, ".abcd", ".work.local")
	if err := os.MkdirAll(filepath.Join(tier, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	big := make([]byte, 5<<20) // over the 4 MiB per-file cap, and NUL bytes: not text
	if err := os.WriteFile(filepath.Join(tier, "bin", "abcd"), big, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tier, "NEXT.md"), []byte("next: land it\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := prune(t, repo, worktree.PruneOptions{})
	r := pruneRow(t, got, "built")
	if r.Outcome != "reclaimed" {
		t.Fatalf("a merged, clean worktree whose tier holds the built binary is reclaimed: %+v", r)
	}
	if exists(p) || listed(t, repo, p) {
		t.Fatal("the reclaimed worktree is gone")
	}
	entries := notesEntries(t, repo, home)
	if len(entries) != 1 {
		t.Fatalf("one archive entry: %v", entries)
	}
	entry := filepath.Join(abcdhome.Path(home, "notes", gittest.RootCommit(t, repo.Root())), entries[0])
	if _, err := os.Stat(filepath.Join(entry, "NEXT.md")); err != nil {
		t.Fatalf("the notes beside the binary are archived: %v", err)
	}
	if exists(filepath.Join(entry, "bin")) {
		t.Fatal("the built binary is left out of the archive")
	}
	b, err := os.ReadFile(filepath.Join(entry, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "bin/abcd") {
		t.Fatalf("the manifest does not list the binary: %s", b)
	}
}

// TestPruneStillJudgesABinDeeperInTheTier: only the tier's top-level bin/ is
// the build product; a bin/ under scratch/ is notes like any other, and a
// binary there still keeps the worktree.
func TestPruneStillJudgesABinDeeperInTheTier(t *testing.T) {
	repo, _ := pruneFixture(t)
	p := mergedWorktree(t, repo, "deep")
	deep := filepath.Join(p, ".abcd", ".work.local", "scratch", "bin")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deep, "tool"), []byte{0, 1, 2}, 0o755); err != nil {
		t.Fatal(err)
	}
	got := prune(t, repo, worktree.PruneOptions{})
	r := pruneRow(t, got, "deep")
	if r.Outcome != "kept" || !strings.Contains(r.Reason, "scratch/bin/tool") {
		t.Fatalf("a binary under scratch/bin keeps the worktree: %+v", r)
	}
}
