package intent

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/record/findingkey"
)

// TestFindingKeyDropsTheIntentBuckets pins the finding key's intent status
// folders to this store's own bucket list: a bucket the key did not know would
// put the folder into the key, and a finding would change key when its intent
// moved into or out of it (itd-2609291923559186, G1).
func TestFindingKeyDropsTheIntentBuckets(t *testing.T) {
	got := findingkey.StatusFoldersForTest(IntentsRelDir)
	if strings.Join(got, ",") != strings.Join(Buckets, ",") {
		t.Fatalf("findingkey drops %v under %s; the store's buckets are %v", got, IntentsRelDir, Buckets)
	}
}
