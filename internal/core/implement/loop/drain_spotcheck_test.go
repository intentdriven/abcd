package loop

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/spotcheck"
)

// TestADrainMoveIsObservedOnce: a drain move that read origin's default
// branch observes it for the spot-check once, from the drain's checkout at
// that branch, and the observation's note joins the move's; a move that could
// not read origin is not observed.
func TestADrainMoveIsObservedOnce(t *testing.T) {
	var seen []spotcheck.Observed
	saved := observeLanding
	t.Cleanup(func() { observeLanding = saved })
	observeLanding = func(_ context.Context, o spotcheck.Observed) string {
		seen = append(seen, o)
		return "spot-check: observed"
	}
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(seen) != 1 || seen[0].Site != spotcheck.SiteDrain || seen[0].Root != repo.Root() || seen[0].Default != "main" {
		t.Fatalf("the move was observed as %+v; want once, from the drain site at main", seen)
	}
	if !strings.Contains(res.Next, "spot-check: observed") {
		t.Fatalf("the move's next = %q; want the observation's note in it", res.Next)
	}

	seen = nil
	gone, _ := reservingDrainRepo(t, "commands/drain.md")
	gone.Git("remote", "set-url", "origin", filepath.Join(t.TempDir(), "gone.git"))
	if _, err := Drain(gone.Root(), Options{}, DrainOptions{}); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 0 {
		t.Fatalf("a move that could not read origin was observed: %+v", seen)
	}
}
