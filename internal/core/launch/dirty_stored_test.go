package launch

import (
	"slices"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestDirtyTreeFilesComparesWithHEADAsStored: a staged edit, with a
// refs/replace entry that stands a commit carrying the staged tree in for
// HEAD, reads as clean to a diff that honours replace refs. The dirty-tree
// read compares with HEAD as stored, so the edit is still listed.
func TestDirtyTreeFilesComparesWithHEADAsStored(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("a.txt", "one\n")
	r.Commit("seed")
	r.Write("a.txt", "two\n")
	r.Git("add", "-A")
	r.Git("replace", r.Git("rev-parse", "HEAD"), r.Git("commit-tree", r.Git("write-tree"), "-p", "HEAD", "-m", "seed"))
	if got := r.Git("diff", "--name-only", "HEAD"); got != "" {
		t.Fatalf("the replace ref does not stand in: the diff lists %q", got)
	}

	dirty, err := DirtyTreeFiles(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(dirty, []string{"a.txt"}) {
		t.Fatalf("dirty = %v, want the staged a.txt HEAD does not store", dirty)
	}
}
