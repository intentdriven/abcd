package cli

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
)

// managedMarker is a conventions file carrying abcd's marker block, which
// makes a checkout one abcd manages.
const managedMarker = "# AGENTS.md\n\n<!-- BEGIN ABCD -->\nabcd's block.\n<!-- END ABCD -->\n"

// TestTheLoopsAutomaticSweepIsThisFrontDoorsSweep: the sweep the loop runs on
// its own after a landing is the one this front door installs: in a managed
// checkout it removes the merged clean worktree, through `abcd sweep
// --apply`'s own readers, keeps the merged local branch abcd did not make for
// the applying sweep (dcn-2611161459048806), and answers the line the
// landing reports.
func TestTheLoopsAutomaticSweepIsThisFrontDoorsSweep(t *testing.T) {
	t.Cleanup(loop.SetAutoSweeper(nil))
	installAutoSweep()
	repo, done := sweepFixture(t)
	repo.Write("AGENTS.md", managedMarker)

	line := loop.AutoSweepForTest(context.Background(), repo.Root())
	if !strings.HasPrefix(line, "abcd swept on its own: ") || !strings.Contains(line, "[removed  worktree  done") || !strings.Contains(line, "[kept  branch  loc") {
		t.Fatalf("the loop's sweep answered %q", line)
	}
	if _, err := os.Stat(done); !os.IsNotExist(err) {
		t.Fatalf("the merged clean worktree stands: %v", err)
	}
	if out := repo.Git("branch", "--list", "loc"); out == "" {
		t.Fatal("the automatic sweep deleted a merged local branch abcd did not make")
	}
}

// TestTheAutomaticSweepHonoursTheRepositorysOff: sweep.automatic false in the
// repository's file removes nothing, and the line says which file turned it
// off; outside a managed checkout nothing is removed and nothing is said.
func TestTheAutomaticSweepHonoursTheRepositorysOff(t *testing.T) {
	t.Cleanup(loop.SetAutoSweeper(nil))
	installAutoSweep()
	repo, done := sweepFixture(t)
	line := loop.AutoSweepForTest(context.Background(), repo.Root())
	if line != "" {
		t.Fatalf("outside a managed checkout the sweep said %q", line)
	}
	if _, err := os.Stat(done); err != nil {
		t.Fatalf("outside a managed checkout the worktree was removed: %v", err)
	}

	repo.Write("AGENTS.md", managedMarker)
	repo.Write(".abcd/config.json", `{"sweep": {"automatic": false}}`)
	line = loop.AutoSweepForTest(context.Background(), repo.Root())
	if !strings.Contains(line, "abcd did not sweep on its own: sweep.automatic is off (repo layer, .abcd/config.json)") {
		t.Fatalf("the off sweep said %q", line)
	}
	if _, err := os.Stat(done); err != nil {
		t.Fatalf("the off sweep removed the worktree: %v", err)
	}
}

// TestEveryRouteThatLandsInstallsTheAutomaticSweep: each verb that can land a
// lane, close a merged one or end a drain installs this front door's sweep as
// its run starts, so the loop sweeps after its landings: `abcd implement
// step`, `abcd drain`, `abcd drain --drive`, `abcd build` (its start and its
// resume) and `abcd build next`. Each route is run where it refuses at once
// (no checkout); the sweep it installed answers, in a managed checkout whose
// setting is off, naming the setting, and nothing is installed after the test.
func TestEveryRouteThatLandsInstallsTheAutomaticSweep(t *testing.T) {
	repo, _ := sweepFixture(t)
	repo.Write("AGENTS.md", managedMarker)
	repo.Write(".abcd/config.json", `{"sweep": {"automatic": false}}`)
	for _, args := range [][]string{
		{"implement", "step"},
		{"drain"},
		{"drain", "--drive"},
		{"build", "itd-1"},
		{"build", "next"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Cleanup(loop.SetAutoSweeper(nil))
			t.Chdir(t.TempDir())
			if _, err := runCLIErr(t, args...); err == nil {
				t.Fatalf("%v outside a checkout ran", args)
			}
			line := loop.AutoSweepForTest(context.Background(), repo.Root())
			if !strings.Contains(line, "abcd did not sweep on its own: sweep.automatic is off") {
				t.Fatalf("after %v the loop's sweep answered %q; want this front door's, installed", args, line)
			}
		})
	}
	if line := loop.AutoSweepForTest(context.Background(), repo.Root()); line != "" {
		t.Fatalf("a sweep is still installed after the routes' tests: %q", line)
	}
}
