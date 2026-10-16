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
