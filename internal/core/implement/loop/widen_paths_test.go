package loop

import (
	"fmt"
	"strings"
	"testing"
	"time"

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

// TestAWideningOfVeryManyPathsIsRefusedAtOnce: a widening over 200,000
// paths, far past what any reservation message can name, is refused as over
// the cap at once, having compared no more than maxWidenPaths of them, and
// pushes nothing.
func TestAWideningOfVeryManyPathsIsRefusedAtOnce(t *testing.T) {
	want := make([]string, 200_000)
	for i := range want {
		want[i] = fmt.Sprintf("root-file-%06d.txt", i)
	}
	start := time.Now()
	w, err := widenHold(t.TempDir(), "iss-5", claim.Reservation{Footprint: []string{"docs"}}, want, time.Now())
	took := time.Since(start)
	rf, ok := AsRefusal(err)
	if !ok || !rf.overCap || !strings.Contains(rf.Reason, fmt.Sprintf("more than %d", 1+maxWidenPaths)) {
		t.Fatalf("the widening is refused as over the cap: %v", err)
	}
	if len(w.added) != 0 || w.footprint != nil {
		t.Fatalf("nothing is added or held: %+v", w)
	}
	if took > 2*time.Second {
		t.Fatalf("the refusal comes at once, not after comparing every path: %s", took)
	}
	if paths, err := widenPaths("lane-1", want); err != nil || len(paths) != len(want) {
		t.Fatalf("the paths are deduplicated in one pass: %d, %v", len(paths), err)
	}
	if took := time.Since(start); took > 4*time.Second {
		t.Fatalf("checking and deduplicating the paths takes no more than a pass over them: %s", took)
	}
}

// TestPathsTheWriterCanCarryAreReservedAtTheReceipt: the walled implementer
// changes forty-seven files beyond the found_at, a footprint that leaves less
// room in the message than claim.FootprintFits keeps but that the writer can
// carry; the loop reserves each by its own name and the receipt verifies.
func TestPathsTheWriterCanCarryAreReservedAtTheReceipt(t *testing.T) {
	f := newNarrowFixture(t)
	names := []string{"docs/page.md"}
	var beyond []string
	for i := range 47 {
		beyond = append(beyond, fmt.Sprintf("pkg/a-file-with-a-long-enough-name-to-fill-%02d.txt", i))
	}
	if claim.FootprintFits(append([]string{"docs"}, beyond...)) {
		t.Fatal("premise: the footprint leaves less room than FootprintFits keeps")
	}
	path := f.walledStage(t, append(names, beyond...)...)
	if _, err := Receipt(f.repo.Root(), f.runID, path, f.stages, Options{}); err != nil {
		t.Fatalf("the receipt verifies once each path is reserved: %v", err)
	}
	if fp := f.footprint(t); len(fp) != 1+len(beyond) {
		t.Fatalf("the hold names the found_at and each path by its own name: %d paths, %q", len(fp), fp)
	}
}
