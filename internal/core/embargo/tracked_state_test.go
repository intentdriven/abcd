package embargo

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestAStateFileACommitCarriesIsRefused: the store is a worktree at the
// default branch's tip, so a state file or closing draft git tracks there came
// with a commit, not from this account's abcd, and is refused before it is
// read, however its owner and mode look.
func TestAStateFileACommitCarriesIsRefused(t *testing.T) {
	s := openedStore(t)
	if err := s.SaveState(sampleState()); err != nil {
		t.Fatal(err)
	}
	if err := s.writeDraft("GHSA-mmmm-mmmm-mmmm", "a closing draft\n"); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{StateRel, DraftDirRel + "/GHSA-mmmm-mmmm-mmmm.md"} {
		cmd := exec.Command("git", "-C", s.Root, "add", "-f", "--", rel)
		cmd.Env = gittest.Env(t)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git add -f %s: %v: %s", rel, err, out)
		}
	}
	// The store's own rule (its index tracks nothing in its local tier) may
	// refuse first; either way the state a commit carries is never read.
	if _, err := s.Verify(); err == nil || !strings.Contains(err.Error(), "tracks") {
		t.Fatalf("Verify over a tracked state file = %v; want it refused as tracked", err)
	}
	if _, err := s.readDraft("GHSA-mmmm-mmmm-mmmm"); err == nil || !strings.Contains(err.Error(), "git tracks") {
		t.Fatalf("readDraft over a tracked draft = %v; want it refused as tracked", err)
	}
	if _, err := readOtherStore(s.Root); err == nil || !strings.Contains(err.Error(), "tracks") {
		t.Fatalf("readOtherStore over a tracked state file = %v; want it refused as tracked", err)
	}
}
