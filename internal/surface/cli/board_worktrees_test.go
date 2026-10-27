package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/textwidth"
)

// boardWorktreesRow is the line criterion 17 names, at its label's width.
const boardWorktreesRow = "  worktrees:  2 in the store, 1 can be cleared (`abcd ahoy worktree prune`)"

// TestTheBoardCountsTheWorktreesTheLaneHolds is spc-2609301811532881 step 6
// (criterion 17): the board carries one worktrees line, held and reclaimable,
// counted from the one peers scan; the line and its --json member are absent
// while the lane holds nothing; a worktree outside the lane is not counted;
// the Now / Next / Later block the site renders carries none of it; and the
// line wraps at the window like every other row.
func TestTheBoardCountsTheWorktreesTheLaneHolds(t *testing.T) {
	root := managedCheckout(t)
	statusRecord(t, root)
	gitCmd(t, root, "add", "-A")
	gitCommit(t, root, "commit", "-q", "-m", "base record")

	if out := string(runCLI(t, "--view", "facilitator")); strings.Contains(out, "worktrees:") {
		t.Fatalf("the board carries a worktrees line on an empty lane:\n%s", out)
	}
	var board map[string]json.RawMessage
	if err := json.Unmarshal(runCLI(t, "--json"), &board); err != nil {
		t.Fatal(err)
	}
	if _, ok := board["worktrees"]; ok {
		t.Fatalf("the board's --json carries a worktrees member on an empty lane: %s", board["worktrees"])
	}

	// One worktree whose branch is merged into main, one with a commit main
	// lacks, and a peer outside the store, which the lane does not hold.
	// add prints the path home-redacted, so the test derives it from HOME.
	lane := abcdhome.Path(os.Getenv("HOME"), "worktrees", gittest.RootCommit(t, root))
	runCLI(t, "ahoy", "worktree", "add", "done")
	done := filepath.Join(lane, "done")
	writeRel(t, done, "done.txt", "done\n")
	gitCmd(t, done, "add", "-A")
	gitCommit(t, done, "commit", "-q", "-m", "done")
	gitCmd(t, root, "merge", "-q", "--ff-only", "done")
	runCLI(t, "ahoy", "worktree", "add", "kept")
	kept := filepath.Join(lane, "kept")
	writeRel(t, kept, "kept.txt", "kept\n")
	gitCmd(t, kept, "add", "-A")
	gitCommit(t, kept, "commit", "-q", "-m", "kept")
	addPeer(t, os.Getenv("HOME"), root, "outside")

	setBoardWidth(t, wideBoard)
	if out := string(runCLI(t, "--view", "facilitator")); !strings.Contains(out, boardWorktreesRow+"\n") {
		t.Fatalf("the board lacks\n%s\nin\n%s", boardWorktreesRow, out)
	}
	board = nil
	if err := json.Unmarshal(runCLI(t, "--json"), &board); err != nil {
		t.Fatal(err)
	}
	var wt struct {
		Held        *int `json:"held"`
		Reclaimable *int `json:"reclaimable"`
	}
	if err := json.Unmarshal(board["worktrees"], &wt); err != nil || wt.Held == nil || wt.Reclaimable == nil ||
		*wt.Held != 2 || *wt.Reclaimable != 1 {
		t.Fatalf("the board's worktrees member = %s, want held 2, reclaimable 1", board["worktrees"])
	}
	if st := string(board["status"]); st == "" || strings.Contains(st, "worktree") || strings.Contains(st, "reclaimable") || strings.Contains(st, "can be cleared") {
		t.Fatalf("the status block the site renders must not carry the worktrees member: %s", st)
	}

	setBoardWidth(t, 40)
	text := string(runCLI(t, "--view", "facilitator"))
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	for i, l := range lines {
		if n := textwidth.Columns(l); n > 40 {
			t.Errorf("at 40 columns a line takes %d: %q", n, l)
		}
		if strings.HasPrefix(l, "  worktrees:") {
			if i+1 >= len(lines) || !strings.HasPrefix(lines[i+1], "      ") {
				t.Errorf("the worktrees row does not continue indented under its start:\n%s", text)
			}
		}
	}
	if !strings.Contains(text, "  worktrees:") {
		t.Errorf("the 40-column board lacks the worktrees row:\n%s", text)
	}
}
