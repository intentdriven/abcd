package loop

import (
	"fmt"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// TestPathsTooManyForAReservationAreAGapNamingTheCap: the walled implementer
// changes sixty files beyond the found_at, more than one reservation message
// can name, each held by its own name; the receipt is refused with a gap
// saying the reservation cannot carry that many paths and naming the cap,
// never asking for the change to be taken out, and nothing is pushed.
func TestPathsTooManyForAReservationAreAGapNamingTheCap(t *testing.T) {
	f := newNarrowFixture(t)
	before := f.tip(t)
	names := []string{"docs/page.md"}
	for i := range 60 {
		names = append(names, fmt.Sprintf("a-root-file-with-a-long-enough-name-to-fill-the-message-%02d.txt", i))
	}
	path := f.walledStage(t, names...)
	_, err := Receipt(f.repo.Root(), f.runID, path, f.stages, Options{})
	rf, ok := AsRefusal(err)
	if !ok || !strings.Contains(rf.Reason, "cannot carry that many paths") || !strings.Contains(rf.Reason, fmt.Sprintf("%d-byte cap", claim.MaxMessageBytes)) ||
		strings.Contains(rf.Reason, "take the change out") {
		t.Fatalf("the receipt is refused naming the cap, not the change: %v", err)
	}
	if after := f.tip(t); after.commit != before.commit {
		t.Fatalf("nothing is pushed: the tip moved from %s to %s", before.commit, after.commit)
	}
}
