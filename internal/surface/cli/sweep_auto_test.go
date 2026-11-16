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
// checkout it removes the merged clean worktree and deletes the merged local
// branch, through `abcd sweep --apply`'s own readers, and answers the line
// the landing reports.
func TestTheLoopsAutomaticSweepIsThisFrontDoorsSweep(t *testing.T) {
	t.Cleanup(loop.SetAutoSweeper(nil))
	installAutoSweep()
	repo, done := sweepFixture(t)
	repo.Write("AGENTS.md", managedMarker)

	line := loop.AutoSweep(context.Background(), repo.Root())
	if !strings.HasPrefix(line, "abcd swept on its own: ") || !strings.Contains(line, "[removed  worktree  done") || !strings.Contains(line, "[deleted  branch  loc") {
		t.Fatalf("the loop's sweep answered %q", line)
	}
	if _, err := os.Stat(done); !os.IsNotExist(err) {
		t.Fatalf("the merged clean worktree stands: %v", err)
	}
	if out := repo.Git("branch", "--list", "loc"); out != "" {
		t.Fatalf("the merged local branch stands: %q", out)
	}
}

// TestTheAutomaticSweepHonoursTheRepositorysOff: sweep.automatic false in the
// repository's file removes nothing, and the line says which file turned it
// off; outside a managed checkout nothing is removed and nothing is said.
func TestTheAutomaticSweepHonoursTheRepositorysOff(t *testing.T) {
	t.Cleanup(loop.SetAutoSweeper(nil))
	installAutoSweep()
	repo, done := sweepFixture(t)
	line := loop.AutoSweep(context.Background(), repo.Root())
	if line != "" {
		t.Fatalf("outside a managed checkout the sweep said %q", line)
	}
	if _, err := os.Stat(done); err != nil {
		t.Fatalf("outside a managed checkout the worktree was removed: %v", err)
	}

	repo.Write("AGENTS.md", managedMarker)
	repo.Write(".abcd/config.json", `{"sweep": {"automatic": false}}`)
	line = loop.AutoSweep(context.Background(), repo.Root())
	if !strings.Contains(line, "abcd did not sweep on its own: sweep.automatic is off (repo layer, .abcd/config.json)") {
		t.Fatalf("the off sweep said %q", line)
	}
	if _, err := os.Stat(done); err != nil {
		t.Fatalf("the off sweep removed the worktree: %v", err)
	}
}
