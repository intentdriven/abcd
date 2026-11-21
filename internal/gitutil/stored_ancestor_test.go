package gitutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestIsAncestorAsStoredReadsTheStoredParents: a graft that parents one
// commit on another makes IsAncestor answer yes; IsAncestorAsStored reads the
// parents the commits store and answers no. A refs/replace entry that stands
// a child of the other commit in for the first does the same.
func TestIsAncestorAsStoredReadsTheStoredParents(t *testing.T) {
	setup := func(t *testing.T) (r *gittest.Repo, side, main string) {
		t.Helper()
		r = gittest.NewRepo(t)
		r.Write("README.md", "a project\n")
		r.Commit("start")
		start := r.Git("rev-parse", "HEAD")
		r.Write("side.txt", "a side line\n")
		r.Commit("side")
		side = r.Git("rev-parse", "HEAD")
		r.Git("checkout", "-q", "--detach", start)
		r.Write("main.txt", "the main line\n")
		r.Commit("main")
		return r, side, r.Git("rev-parse", "HEAD")
	}
	check := func(t *testing.T, root, side, main string) {
		t.Helper()
		if ok, err := gitutil.IsAncestor(root, side, main); err != nil || !ok {
			t.Fatalf("IsAncestor = %v, %v; the stand-in does not stand in", ok, err)
		}
		if ok, err := gitutil.IsAncestorAsStored(root, side, main); err != nil || ok {
			t.Fatalf("IsAncestorAsStored = %v, %v; want no from the stored parents", ok, err)
		}
		if ok, err := gitutil.IsAncestorAsStored(root, "HEAD~1", main); err != nil || !ok {
			t.Fatalf("IsAncestorAsStored(HEAD~1) = %v, %v; want yes", ok, err)
		}
	}
	t.Run("a graft", func(t *testing.T) {
		r, side, main := setup(t)
		info := filepath.Join(r.Root(), ".git", "info")
		if err := os.MkdirAll(info, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(info, "grafts"), []byte(main+" "+side+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		check(t, r.Root(), side, main)
	})
	t.Run("a refs/replace entry", func(t *testing.T) {
		r, side, main := setup(t)
		r.Git("replace", main, r.Git("commit-tree", main+"^{tree}", "-p", side, "-m", "main"))
		check(t, r.Root(), side, main)
	})
}
