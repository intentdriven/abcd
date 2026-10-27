package cli

import (
	"fmt"
	"io"

	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/peers"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// boardWorktreesLine is the board's worktrees member (spc-2609301811532881,
// criterion 17): how many worktrees this repository's lane in the store
// holds, and how many of them the scan judged spent.
type boardWorktreesLine struct {
	Held        int `json:"held"`
	Reclaimable int `json:"reclaimable"`
}

// boardWorktrees counts, from the scan the board already ran, the linked
// worktrees inside the repository's lane, or nil when the lane holds none.
// It reads nothing of its own but the root commit that keys the lane, and
// only when the scan saw a worktree at all; a lane that cannot be derived
// omits the line, said on stderr.
func boardWorktrees(root string, rep peers.Report, stderr io.Writer) *boardWorktreesLine {
	if len(rep.Worktrees()) == 0 {
		return nil
	}
	lane, err := worktree.LaneFor(root)
	if err != nil {
		fmt.Fprintf(stderr, "abcd: the worktrees line is omitted — %s\n", termsafe.Sanitize(fsutil.DisplayPathsIn(err.Error(), root)))
		return nil
	}
	held, spent := rep.HeldUnder(lane.Dir)
	if held == 0 {
		return nil
	}
	return &boardWorktreesLine{Held: held, Reclaimable: spent}
}
