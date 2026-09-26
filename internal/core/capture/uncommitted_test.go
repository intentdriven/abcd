package capture

import (
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestCaptureAndStatusSayARecordIsUncommitted is iss-2609100508570527: a
// record the ledger holds only as an untracked file is in no state to anyone
// but this checkout — no branch cut from the default branch sees it, and no gate
// that reads the committed tree reads it. The write says so, and the status
// board marks such a record rather than listing it as an equal member of its
// folder. Once committed, neither says it.
func TestCaptureAndStatusSayARecordIsUncommitted(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Commit("root")
	repo := r.Root()
	ir := filepath.Join(repo, LedgerRelPath)

	res, err := Capture(CaptureRequest{RepoRoot: repo, IssuesRoot: ir, Text: "a finding",
		Severity: SeverityMinor, Category: "bug", Source: "manual-test", Slug: "untracked", FoundDuring: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Uncommitted {
		t.Fatal("the capture result does not say the record it just wrote is uncommitted")
	}
	st, err := Status(StatusRequest{RepoRoot: repo, IssuesRoot: ir})
	if err != nil {
		t.Fatal(err)
	}
	if st.UncommittedCount != 1 || len(st.RecentOpen) != 1 || !st.RecentOpen[0].Uncommitted {
		t.Fatalf("status must mark the untracked record: count=%d rows=%+v", st.UncommittedCount, st.RecentOpen)
	}

	r.Commit("file the record")
	st, err = Status(StatusRequest{RepoRoot: repo, IssuesRoot: ir})
	if err != nil {
		t.Fatal(err)
	}
	if st.UncommittedCount != 0 || st.RecentOpen[0].Uncommitted {
		t.Fatalf("a committed record is still marked uncommitted: count=%d rows=%+v", st.UncommittedCount, st.RecentOpen)
	}
}
