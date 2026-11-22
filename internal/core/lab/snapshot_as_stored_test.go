package lab

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestTheSnapshotDescendsFromThePinAsStored: the snapshot check passes only
// when the snapshot's HEAD stores the pin as an ancestor; a replace ref giving
// HEAD the pin as a parent does not make it pass.
func TestTheSnapshotDescendsFromThePinAsStored(t *testing.T) {
	src := gittest.NewRepo(t)
	src.Commit("base")
	base := src.Git("rev-parse", "HEAD")
	pin := src.Git("commit-tree", base+"^{tree}", "-p", base, "-m", "the pin")
	src.Commit("elsewhere")
	head := src.Git("rev-parse", "HEAD")
	home := t.TempDir()
	if err := os.Rename(src.Root(), filepath.Join(home, snapshotDir)); err != nil {
		t.Fatal(err)
	}
	snap := gittest.OpenRepo(t, filepath.Join(home, snapshotDir))
	l := &lab{dir: home, entry: Entry{Pin: pin}}
	if c := l.checkSnapshot(); c.OK {
		t.Fatalf("a HEAD that does not descend from the pin passed: %+v", c)
	}
	snap.Git("replace", "--graft", head, base, pin)
	if c := l.checkSnapshot(); c.OK {
		t.Fatalf("a replace ref made a HEAD that does not store the pin pass: %+v", c)
	}
	snap.Git("replace", "-d", head)
	snap.Git("merge", "-q", "--no-ff", "-m", "take the pin", pin)
	if c := l.checkSnapshot(); !c.OK {
		t.Fatalf("a HEAD that stores the pin was refused: %+v", c)
	}
}
