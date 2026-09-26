package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/capture"
)

// TestStrayStoreNotesKnowsTheRootUnderAnotherCase is iss-2609260057123452: on a
// case-insensitive filesystem a working directory spelt with a different case
// from the checkout root is the root, but the walk compared path strings, never
// met the root, and reported the checkout's own ledger as a second one below it.
// A genuine stray store below the root is still reported.
func TestStrayStoreNotesKnowsTheRootUnderAnotherCase(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "CaseRepo")
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(capture.LedgerRelPath)), 0o755); err != nil {
		t.Fatal(err)
	}
	variant := filepath.Join(parent, "caserepo")
	if _, err := os.Stat(variant); err != nil {
		t.Skip("this filesystem is case-sensitive, so no case-variant spelling reaches the root")
	}

	if notes := strayStoreNotes(variant, root, capture.LedgerRelPath, "ledger"); len(notes) != 0 {
		t.Errorf("the checkout's own ledger, reached by a case-variant path, is reported as a stray: %v", notes)
	}

	sub := filepath.Join(variant, "sub")
	if err := os.MkdirAll(filepath.Join(sub, filepath.FromSlash(capture.LedgerRelPath)), 0o755); err != nil {
		t.Fatal(err)
	}
	notes := strayStoreNotes(sub, root, capture.LedgerRelPath, "ledger")
	if len(notes) != 1 || !strings.Contains(notes[0], "sub/"+capture.LedgerRelPath) {
		t.Errorf("a genuine stray ledger below the root is not reported exactly once: %v", notes)
	}
}
