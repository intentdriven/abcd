package capture

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// TestRefusedDispositionOfAHeldItemLeavesNoLock — a disposition the schema
// refuses (no grounds, a held state with no exit condition, a state the
// item's position does not offer) is the caller's input, and the request
// alone decides it. Disposition checked it only after mutationPreamble, which
// opens the allocator lock file, so a refusal of an item the ledger does hold
// left .abcd/work/issues/.iss-alloc.lock behind. The check runs before the
// preamble, as admit's requireWidening does, and a refusal writes nothing.
func TestRefusedDispositionOfAHeldItemLeavesNoLock(t *testing.T) {
	for _, tc := range []struct {
		name string
		req  DispositionRequest
	}{
		{"empty grounds", DispositionRequest{State: issueschema.DispositionAccepted, Grounds: "   "}},
		{"held without exit condition", DispositionRequest{State: issueschema.DispositionHeld}},
		{"state unavailable at the position", DispositionRequest{State: issueschema.DispositionDeclined, Grounds: "because"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, ir, item := readingFixture(t, "detection")
			lock := filepath.Join(ir, lockFilename)
			// The fixture's own ingest took the lock; start from a ledger
			// without one, as a fresh clone of a committed ledger is.
			if err := os.Remove(lock); err != nil && !os.IsNotExist(err) {
				t.Fatalf("removing the fixture's lock: %v", err)
			}
			req := tc.req
			req.RepoRoot, req.IssuesRoot, req.Item = repo, ir, item
			if _, err := Disposition(req); err == nil {
				t.Fatalf("Disposition(%s) succeeded; want a refusal", tc.name)
			}
			if _, err := os.Lstat(lock); !os.IsNotExist(err) {
				t.Errorf("a refused disposition (%s) left %s behind (Lstat err = %v); a refusal writes nothing", tc.name, lockFilename, err)
			}
			if _, err := os.Lstat(filepath.Join(ir, issueschema.DispositionsDir)); !os.IsNotExist(err) {
				t.Errorf("a refused disposition (%s) created %s (Lstat err = %v)", tc.name, issueschema.DispositionsDir, err)
			}
		})
	}
}
