package capture

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestOrphanSweepDoesNotDeleteCommittedFile is the iss-102 repro. It forces the
// documented interleaving: the orphan sweep classifies a >60s-stalled, zero-byte
// placeholder as removable, then — in the residual window immediately before the
// unlink — a stalled commit fills that exact placeholder. Pre-fix (sweep and
// commit both lockless) the sweep's os.Remove deletes the file the commit just
// wrote, AFTER the commit reported success — silent data loss. Post-fix the sweep
// runs under the ledger lock and the commit write does too, so the commit cannot
// land inside the sweep's window: it blocks on the lock, the sweep removes the
// still-empty placeholder, and the commit then fails loudly (missing placeholder)
// instead of the sweep destroying a committed file.
//
// The invariant asserted is exactly the issue's failure mode: a commit that
// reports success must leave its file present. Pre-fix that invariant is violated;
// post-fix the racing commit never reports success.
func TestOrphanSweepDoesNotDeleteCommittedFile(t *testing.T) {
	repo, ir := ledger(t)
	if err := ensureLedgerDirs(repo, ir); err != nil {
		t.Fatal(err)
	}

	// Reserve a placeholder by hand and age it past the sweep threshold, standing
	// in for a capture that stalled >60s between reservation and commit.
	placeholder := filepath.Join(ir, "open", "iss-1-stall.md")
	if err := os.WriteFile(placeholder, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * orphanAgeThreshold)
	if err := os.Chtimes(placeholder, old, old); err != nil {
		t.Fatal(err)
	}

	req := CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "stalled commit body\n",
		Severity: SeverityMinor, Category: "bug", Source: "manual-test",
		Slug: "stall", FoundDuring: "iss-102 repro",
	}

	// The committer fires when the sweep reaches its pre-unlink window, and reports
	// its outcome once. Buffered so the send never blocks whether the hook or the
	// test drains it.
	trigger := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		<-trigger
		_, err := commitCapture(repo, ir, req, "iss-1", "stall", placeholder)
		result <- err
	}()

	// Shorten the lock budget so the post-fix "commit blocked on the sweep's lock"
	// path resolves within the test window instead of the 5s default.
	origTimeout := lockTimeout
	lockTimeout = 200 * time.Millisecond
	defer func() { lockTimeout = origTimeout }()

	var commitErr error
	commitDone := false
	beforeOrphanRemoveHook = func(cand string) {
		beforeOrphanRemoveHook = nil // fire exactly once
		close(trigger)
		select {
		case commitErr = <-result:
			commitDone = true // pre-fix: the commit wrote and returned before the unlink
		case <-time.After(500 * time.Millisecond):
			// post-fix: the commit is blocked on the ledger lock this sweep holds.
		}
	}
	defer func() { beforeOrphanRemoveHook = nil }()

	// Run the sweep through the production preamble so post-fix it holds the lock.
	if err := mutationPreamble(repo, ir); err != nil {
		t.Fatalf("mutationPreamble: %v", err)
	}

	if !commitDone {
		commitErr = <-result // post-fix: the commit unblocks once the sweep released
	}

	// The core invariant: if the commit reported success, its file must exist and
	// be non-empty. Pre-fix the sweep deleted the just-committed file, so a nil
	// commitErr coincides with a missing file — the violation.
	if commitErr == nil {
		fi, err := os.Stat(placeholder)
		if err != nil {
			t.Fatalf("commit reported success but the committed file is gone: %v", err)
		}
		if fi.Size() == 0 {
			t.Fatalf("commit reported success but the committed file is empty")
		}
	}
}
