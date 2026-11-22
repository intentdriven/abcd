package lint_test

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
)

// TestCheckReleaseReceiptsComparesWithHEADAsStored: a staged edit to a
// verdict input, with a refs/replace entry that stands a commit carrying the
// staged tree in for HEAD, reads as committed to a status that honours
// replace refs. The check compares the inputs with HEAD as stored, so the
// edit still refuses as uncommitted.
func TestCheckReleaseReceiptsComparesWithHEADAsStored(t *testing.T) {
	r := committedDeferral(t)
	rel, body := openIssue("iss-5")
	r.Write(rel, body+"more\n")
	r.Git("add", "-A")
	r.Git("replace", r.Git("rev-parse", "HEAD"), r.Git("commit-tree", r.Git("write-tree"), "-p", "HEAD", "-m", "capture"))
	if got := r.Git("status", "--porcelain", "--", rel); got != "" {
		t.Fatalf("the replace ref does not stand in: status reads %q", got)
	}

	check, err := lint.CheckReleaseReceipts(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	named := false
	for _, u := range check.Uncommitted {
		named = named || strings.Contains(u, rel)
	}
	if check.Pass || !named {
		t.Fatalf("a staged edit HEAD does not store must refuse, naming %s, got %+v", rel, check)
	}
}
