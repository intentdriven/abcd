package loop

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// TestManyChangedPathsAreHeldByTheirDirectory: the walled implementer changes
// sixty files under pkg/ beyond the found_at, more than one reservation
// message can name; the loop holds their directory instead, and the receipt
// verifies.
func TestManyChangedPathsAreHeldByTheirDirectory(t *testing.T) {
	f := newNarrowFixture(t)
	names := []string{"docs/page.md"}
	for i := range 60 {
		names = append(names, fmt.Sprintf("pkg/sub/a-file-with-a-long-enough-name-to-fill-the-message-%02d.txt", i))
	}
	path := f.walledStage(t, names...)
	if _, err := Receipt(f.repo.Root(), f.runID, path, f.stages, Options{}); err != nil {
		t.Fatalf("the receipt verifies once the paths are held by their directory: %v", err)
	}
	if fp := sortedFootprint(t, f); !slices.Equal(fp, []string{"docs", "pkg/sub"}) {
		t.Fatalf("the hold names the files' directory: %q", fp)
	}
}

// TestPathsTooManyForAReservationAreAGapNamingTheCap: sixty files at the
// repository's root cannot be held by a directory, since a hold never goes
// up to the root; the receipt is refused with a gap saying the reservation
// cannot carry that many paths and naming the cap, and nothing is pushed.
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

// TestACoarseningNeverReachesTheRootOrAGateFile: paths go up one level at a
// time until they fit, never to the repository's root and never to a gate
// file, which keeps its own path.
func TestACoarseningNeverReachesTheRootOrAGateFile(t *testing.T) {
	long := func(dir string, n int) []string {
		var out []string
		for i := range n {
			out = append(out, fmt.Sprintf("%sa-file-with-a-long-enough-name-to-fill-the-message-%02d.txt", dir, i))
		}
		return out
	}
	if got, ok := coarsened([]string{"docs"}, []string{"README.txt"}, ""); !ok || !slices.Equal(got, []string{"README.txt"}) {
		t.Fatalf("paths that fit are left as they are: %q %v", got, ok)
	}
	if got, ok := coarsened([]string{"docs"}, long("a/b/c/", 60), ""); !ok || !slices.Equal(got, []string{"a/b/c"}) {
		t.Fatalf("paths go up one level at a time: %q %v", got, ok)
	}
	if got, ok := coarsened(nil, append(long("top/", 30), long("top/sub/", 30)...), ""); !ok || !slices.Equal(got, []string{"top"}) {
		t.Fatalf("a path a coarsened one covers is dropped: %q %v", got, ok)
	}
	if _, ok := coarsened([]string{"docs"}, long("", 60), ""); ok {
		t.Fatal("files at the root never go up to it")
	}
	for _, dir := range []string{"scripts/tools/", ".githooks/", "hooks/", "custom-hooks/"} {
		got, ok := coarsened([]string{"docs"}, long(dir, 60), "custom-hooks")
		if ok || slices.ContainsFunc(got, func(p string) bool { return DriveGate(p, "custom-hooks") && !strings.HasSuffix(p, ".txt") }) {
			t.Fatalf("paths under %s never go up to a gate: %q %v", dir, got, ok)
		}
	}
}
