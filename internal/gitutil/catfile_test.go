package gitutil_test

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestCatFileBatchAnswersEachObjectInOrderUnderItsCap: one batch answers every
// object asked, in order, a missing one marked missing and content kept byte
// for byte; an answer past the cap is an error, and a name that is not a full
// object name never reaches git.
func TestCatFileBatchAnswersEachObjectInOrderUnderItsCap(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write("a.txt", "")
	a := strings.TrimSpace(repo.Git("hash-object", "-w", "--", "a.txt"))
	repo.Write("b.txt", "line one\nline two\n\n")
	b := strings.TrimSpace(repo.Git("hash-object", "-w", "--", "b.txt"))
	missing := strings.Repeat("0", len(b))
	got, err := gitutil.CatFileBatch(repo.Root(), []string{b, missing, a}, 1<<10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Name != b || got[0].Type != "blob" || string(got[0].Content) != "line one\nline two\n\n" ||
		!got[1].Missing || got[2].Type != "blob" || len(got[2].Content) != 0 {
		t.Fatalf("the batch = %+v", got)
	}
	if _, err := gitutil.CatFileBatch(repo.Root(), []string{b}, 20); err == nil || !strings.Contains(err.Error(), "cap") {
		t.Fatalf("an answer past the cap = %v; want it refused", err)
	}
	if _, err := gitutil.CatFileBatch(repo.Root(), []string{b + "\n" + a}, 1<<10); err == nil {
		t.Fatal("a name that is not a full object name reached git")
	}
}

// TestTreePathTypesAnswersEachPathOnItsOwn: one batch answers each path's
// type in order, a directory asked beside a file beneath it still a tree and
// an absent path ""; a path that cannot be asked on one line, and a commit
// that is not a full object name, never reach git.
func TestTreePathTypesAnswersEachPathOnItsOwn(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write("internal/x/a.go", "package x\n")
	repo.Commit("a tree")
	head := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	got, err := gitutil.TreePathTypes(repo.Root(), head, []string{"internal/x", "internal/x/a.go", "internal/x/", "gone", "internal/x/a.go/"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"tree", "blob", "tree", "", ""}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("the types = %q; want %q", got, want)
	}
	if _, err := gitutil.TreePathTypes(repo.Root(), head, []string{"a\nHEAD"}); err == nil {
		t.Fatal("a path holding a newline reached git")
	}
	if _, err := gitutil.TreePathTypes(repo.Root(), "HEAD", []string{"internal"}); err == nil {
		t.Fatal("a commit that is not a full object name reached git")
	}
}

// TestTreePathTypesClassifiesAGitlinkAsACommit: a submodule's gitlink is a
// commit entry in its tree, never a file. Git answers "<sha> commit" when the
// repository holds the commit the gitlink names, and "<sha> submodule",
// ignoring the format asked, when it does not; both answer "commit", and a
// path beneath a gitlink is not in the tree. None of it is an error.
func TestTreePathTypesClassifiesAGitlinkAsACommit(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write("a.go", "package a\n")
	repo.Commit("one")
	held := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	absent := strings.Repeat("0123456789", 4)
	repo.Git("update-index", "--add", "--cacheinfo", "160000,"+held+",sub")
	repo.Git("update-index", "--add", "--cacheinfo", "160000,"+absent+",gone")
	repo.Git("-c", "commit.gpgsign=false", "commit", "-q", "-m", "two gitlinks")
	head := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	got, err := gitutil.TreePathTypes(repo.Root(), head, []string{"sub", "gone", "sub/x", "a.go"})
	if err != nil {
		t.Fatalf("a gitlink errored the answer: %v", err)
	}
	if want := []string{"commit", "commit", "", "blob"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("the types = %q; want %q", got, want)
	}
}
