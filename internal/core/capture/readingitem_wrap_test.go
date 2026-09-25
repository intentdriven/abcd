package capture

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/readingitem"
)

// TestCaptureSentinelsWrapLocator holds capture's locator as a thin wrapper over
// the readingitem leaf (spc-2609020626046252): every refusal is still capture's
// own sentinel with capture's own message, so its callers and mintUnusedItemID's
// probing read unchanged, and it is ALSO the leaf's, so a caller that has moved
// to the leaf's sentinels reads the same error through either door.
func TestCaptureSentinelsWrapLocator(t *testing.T) {
	_, ir, item := readingFixture(t, "detection")

	path, err := findReadingItem(ir, item)
	if err != nil {
		t.Fatalf("findReadingItem: %v", err)
	}
	if _, leafPath, _ := readingitem.Locate(ir, item); leafPath != path {
		t.Fatalf("the wrapper and the leaf disagree: %s vs %s", path, leafPath)
	}

	const absent = "rdi-2608300000000077"
	_, err = findReadingItem(ir, absent)
	if !errors.Is(err, ErrUnknownIssueID) || !errors.Is(err, readingitem.ErrUnknown) {
		t.Errorf("an absent item: err = %v, want both ErrUnknownIssueID and readingitem.ErrUnknown", err)
	}
	if err != nil && err.Error() != "unknown issue id: "+absent+" is not a reading item this ledger holds" {
		t.Errorf("the message moved: %q", err.Error())
	}

	// A second run holding the same item is a ledger fault.
	src, _ := os.ReadFile(path)
	dup := filepath.Join(ir, issueschema.ReadingsDir, "rdg-2608300000000099", item+".md")
	if err := os.MkdirAll(filepath.Dir(dup), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dup, src, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = findReadingItem(ir, item)
	if !errors.Is(err, ErrDuplicateIssueID) || !errors.Is(err, readingitem.ErrDuplicate) {
		t.Errorf("a duplicated item: err = %v, want both duplicate sentinels", err)
	}
	if err := os.Remove(dup); err != nil {
		t.Fatal(err)
	}

	// A symlinked run directory is capture's ErrPathUnsafe as well as the leaf's.
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(ir, issueschema.ReadingsDir, "rdg-2608300000000098")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	_, err = findReadingItem(ir, item)
	if !errors.Is(err, ErrPathUnsafe) || !errors.Is(err, readingitem.ErrPathUnsafe) {
		t.Errorf("a symlinked run: err = %v, want both ErrPathUnsafe sentinels", err)
	}
	if _, err := readingItemPaths(ir, item); !errors.Is(err, ErrPathUnsafe) {
		t.Errorf("readingItemPaths through a symlinked run: err = %v, want ErrPathUnsafe", err)
	}
}
