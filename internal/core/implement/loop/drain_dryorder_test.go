package loop

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The dry run's order on a copy (spc-2610161044461448 step 2): with this
// copy's key kept, the dry run shows the order the next move takes; on a
// fresh copy it mints no key, takes each group oldest first, and says that
// the copy's own order is drawn at its first move.

// groupDrainRepo is reservingDrainRepo with six more documentation issues in
// the documentation issue's group, all naming one file, pushed and fetched.
func groupDrainRepo(t *testing.T) (*gittest.Repo, []string) {
	t.Helper()
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	group := []string{secondIssue}
	for i := 0; i < 6; i++ {
		id := fmt.Sprintf("iss-26093000000002%02d", i)
		fileIssueAt(t, repo, id, capture.SeverityMinor, capture.Category("documentation"), "Name the flag in the page.", "commands/drain.md")
		group = append(group, id)
	}
	repo.Commit("a documentation group")
	repo.Git("push", "-q", "origin", "main")
	repo.Git("fetch", "-q", "origin")
	return repo, group
}

// eligibleIDs is the eligible ids of verdicts, in order.
func eligibleIDs(vs []capture.DrainVerdict) []string {
	var out []string
	for _, v := range vs {
		if v.Outcome == capture.DrainEligible {
			out = append(out, v.ID)
		}
	}
	return out
}

// copyKeyPath is where this copy keeps its key.
func copyKeyPath(repo *gittest.Repo) string {
	return filepath.Join(repo.Root(), ".abcd", ".work.local", "reservation", "copy-key")
}

// TestTheDryRunOrderIsTheNextMovesOnceTheCopyHasAKey: with the key kept, the
// dry run's order is the one the next move plans, and it says nothing of a
// first move.
func TestTheDryRunOrderIsTheNextMovesOnceTheCopyHasAKey(t *testing.T) {
	repo, _ := groupDrainRepo(t)
	if _, err := claim.CopyKey(repo.Root()); err != nil {
		t.Fatal(err)
	}
	pv, err := DrainDryRun(repo.Root(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := eligibleIDs(pv.Dispositions), eligibleIDs(res.Dispositions); !slices.Equal(got, want) {
		t.Fatalf("the dry run's order = %v; the move's = %v", got, want)
	}
	if pv.OrderDrawnAtFirstMove || strings.Contains(pv.Order, "first move") {
		t.Fatalf("a copy with a key is said to draw its order later: %v, %q", pv.OrderDrawnAtFirstMove, pv.Order)
	}
}

// TestTheDryRunOnAFreshCopyMintsNoKeyAndSaysTheOrderIsDrawnLater: with no key
// kept, the dry run mints none, takes the group oldest first, and says so in
// its order and its field; the first move mints the key, and a dry run after
// it shows the move's order.
func TestTheDryRunOnAFreshCopyMintsNoKeyAndSaysTheOrderIsDrawnLater(t *testing.T) {
	repo, group := groupDrainRepo(t)
	pv, err := DrainDryRun(repo.Root(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(copyKeyPath(repo)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the dry run minted the copy's key: %v", err)
	}
	oldest := slices.Clone(group)
	slices.Sort(oldest)
	if got := eligibleIDs(pv.Dispositions); !slices.Equal(got[:len(oldest)], oldest) {
		t.Fatalf("the fresh copy's preview takes the group oldest first: %v", got)
	}
	if !pv.OrderDrawnAtFirstMove || !strings.Contains(pv.Order, "drawn at this copy's first move") {
		t.Fatalf("the preview says the order is drawn at the first move: %v, %q", pv.OrderDrawnAtFirstMove, pv.Order)
	}
	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	after, err := DrainDryRun(repo.Root(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if after.OrderDrawnAtFirstMove || !slices.Equal(eligibleIDs(after.Dispositions), eligibleIDs(res.Dispositions)) {
		t.Fatalf("after the first move the preview shows its order: %v, %v; move %v", after.OrderDrawnAtFirstMove, eligibleIDs(after.Dispositions), eligibleIDs(res.Dispositions))
	}
}
