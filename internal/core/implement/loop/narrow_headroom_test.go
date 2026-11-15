package loop

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// TestALandingsNarrowingKeepsRoomForItsRecords: the lane's found_at is the
// directory docs/, and its branch changes forty-five files under it and one
// beyond it. A narrowing to those files would leave the reservation message
// no room for the two record paths the landing's records commit adds at the
// push fence, so the landing's half-lease renewal declines it and the hold
// stays as it stands; the push fence then holds the record paths and the
// push goes on.
func TestALandingsNarrowingKeepsRoomForItsRecords(t *testing.T) {
	f := newNarrowFixture(t)
	names := []string{"README.txt"}
	for i := range 45 {
		names = append(names, fmt.Sprintf("docs/a-page-with-a-long-enough-name-to-fill-the-msg-%02d.md", i))
	}
	if err := f.handBack(t, Options{}, names...); err != nil {
		t.Fatal(err)
	}
	f.toLand(t)
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := claim.View(f.repo.Root(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	own, ok := snap.Lookup(secondIssue)
	if !ok {
		t.Fatal("premise: this copy holds the issue")
	}
	if fp, ok, declined := narrowedFootprint(f.repo.Root(), st, own); ok || !strings.Contains(declined, "no room for the records") {
		t.Fatalf("the narrowing is declined for want of room: %d paths, %v, %q", len(fp), ok, declined)
	}
	due := f.tip(t).claimed.Add(61 * time.Minute)
	f.stepAt(t, due)
	if fp := sortedFootprint(t, f); !slices.Equal(fp, []string{"README.txt", "docs"}) {
		t.Fatalf("the landing's renewal leaves the hold as it stands: %q", fp)
	}
	l, err := stepToThePush(t, f, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := gitIn(t, f.bare, "for-each-ref", "refs/heads/"+l.Branch); got == "" {
		t.Fatalf("the lane's branch is pushed: %+v", l.Landing)
	}
	moved := landingMoved(t, f, l)
	if len(moved) != 2 {
		t.Fatalf("premise: the records commit moves the issue's record, two paths: %q", moved)
	}
	want := heldWith([]string{"docs", "README.txt"}, moved)
	slices.Sort(want)
	if fp := sortedFootprint(t, f); !slices.Equal(fp, want) {
		t.Fatalf("the push fence holds the record paths before the push: %q", fp)
	}
}
