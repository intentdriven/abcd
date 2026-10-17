package claim

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestStoredReadsIgnoreAGraftFile: a .git/info/grafts line that gives a
// status tip another parent changes neither what the tip holds nor its
// first-parent line. The reads that judge whether a push landed, and the
// history walk, read commits as stored, with no graft file, so an
// unrelated commit grafted under the tip is not on the branch.
func TestStoredReadsIgnoreAGraftFile(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("README.md", "a project\n")
	r.Commit("a")
	a := r.Git("rev-parse", "HEAD")
	r.Commit("b")
	b := r.Git("rev-parse", "HEAD")
	other := r.Git("commit-tree", r.Git("hash-object", "-w", "-t", "tree", "/dev/null"), "-m", "an unrelated commit")
	if err := os.WriteFile(filepath.Join(r.Root(), ".git", "info", "grafts"), []byte(b+" "+other+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := r.Git("-c", "advice.graftFileDeprecated=false", "rev-parse", b+"^"); got != other {
		t.Fatalf("the graft does not stand in: %s's parent reads as %s", b, got)
	}

	held, err := statusHolds(r.Root(), other, b)
	if err != nil || held {
		t.Fatalf("statusHolds(the grafted commit, tip) = %v, %v; want false", held, err)
	}
	line, err := firstParents(r.Root(), "the branch", b, 5)
	if err != nil || !slices.Equal(line, []string{b, a}) {
		t.Fatalf("firstParents = %q, %v; want the stored line [%s %s]", line, err, b, a)
	}
}
