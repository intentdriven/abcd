package spec

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/record/findingkey"
)

// TestFindingKeyDropsTheSpecStatuses pins the finding key's spec status folders
// to this store's own: a status the key did not know would put the folder into
// the key, and a finding would change key when its spec closed
// (itd-2609291923559186, G1).
func TestFindingKeyDropsTheSpecStatuses(t *testing.T) {
	got := findingkey.StatusFoldersForTest(SpecsRelDir)
	if want := []string{StatusOpen, StatusClosed}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("findingkey drops %v under %s; the store's statuses are %v", got, SpecsRelDir, want)
	}
}
