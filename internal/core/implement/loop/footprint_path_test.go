package loop

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// TestALaneChangingAPathNoReservationMayCarryIsRefusedAndPushesNothing: the
// walled implementer adds a file at the root whose name ends in a space,
// beside another copy's live hold. A footprint path must read back exactly
// as written, so the loop does not reserve it: the receipt is refused,
// naming it, and nothing reaches origin.
func TestALaneChangingAPathNoReservationMayCarryIsRefusedAndPushesNothing(t *testing.T) {
	f := newNarrowFixture(t)
	f.reserveAs(t, "iss-99", "quinn", "tools/run.go", time.Now().Add(-time.Minute))
	before := f.tip(t)
	path := f.walledStage(t, "docs/page.md", "tools ")
	_, err := Receipt(f.repo.Root(), f.runID, path, f.stages, Options{})
	if rf, ok := AsRefusal(err); !ok || !strings.Contains(rf.Reason, "take the change out of the branch") {
		t.Fatalf("the receipt is refused on a path no reservation may carry: %v", err)
	}
	if after := f.tip(t); after.commit != before.commit {
		t.Fatalf("nothing is pushed: the tip moved from %s to %s", before.commit, after.commit)
	}
	if fp := f.footprint(t); !slices.Equal(fp, []string{"docs"}) {
		t.Fatalf("the hold stays on the found_at: %q", fp)
	}
}
