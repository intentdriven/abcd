package statusblock

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestALaneAtItsLandingReadsAsLandedOnlyWhenTheDefaultBranchStoresItsTip: a
// lane branch whose tip the default branch does not store has not landed,
// whatever parents a refs/replace entry or a .git/info/grafts line gives the
// default branch's tip; the same tip merged in the stored history has.
func TestALaneAtItsLandingReadsAsLandedOnlyWhenTheDefaultBranchStoresItsTip(t *testing.T) {
	setup := func(t *testing.T) (r *gittest.Repo, base, work, tip string) {
		t.Helper()
		r = gittest.NewRepo(t)
		r.Git("symbolic-ref", "HEAD", "refs/heads/main")
		r.Commit("base")
		base = r.Git("rev-parse", "HEAD")
		work = r.Git("commit-tree", base+"^{tree}", "-p", base, "-m", "work")
		r.Git("branch", "build/run-1-lane-1", work)
		r.Commit("main moves on")
		return r, base, work, r.Git("rev-parse", "HEAD")
	}
	check := func(t *testing.T, r *gittest.Repo, work, tip string) {
		t.Helper()
		if !slices.Contains(strings.Fields(r.Git("rev-list", tip)), work) {
			t.Fatal("git does not present the lane's tip on main")
		}
		if LandedContext(context.Background(), r.Root(), "build/run-1-lane-1", true) {
			t.Fatal("a lane whose tip main does not store reads as landed")
		}
	}
	t.Run("a refs/replace entry", func(t *testing.T) {
		r, base, work, tip := setup(t)
		r.Git("replace", "--graft", tip, base, work)
		check(t, r, work, tip)
	})
	t.Run("a graft file", func(t *testing.T) {
		r, base, work, tip := setup(t)
		info := filepath.Join(r.Root(), ".git", "info")
		if err := os.MkdirAll(info, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(info, "grafts"), []byte(tip+" "+base+" "+work+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		check(t, r, work, tip)
	})
	t.Run("a stored merge", func(t *testing.T) {
		r, _, work, _ := setup(t)
		r.Git("merge", "-q", "--no-ff", "-m", "Merge pull request", work)
		if !LandedContext(context.Background(), r.Root(), "build/run-1-lane-1", true) {
			t.Fatal("a lane whose tip main stores does not read as landed")
		}
	})
}
