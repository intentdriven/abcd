package cli

import (
	"fmt"
	"io"

	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/peers"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// boardWorktreesLine is the board's worktrees member (spc-2609301811532881,
// criterion 17): how many worktrees this repository's lane in the store
// holds, and how many of them prune would reclaim with no forge answer.
type boardWorktreesLine struct {
	Held        int `json:"held"`
	Reclaimable int `json:"reclaimable"`
}

// boardWorktrees counts the lane's worktrees from git's listing the board's
// one scan already read, or nil when the lane holds none. Reclaimable is the
// store's own judgement, never the scan's merged-by-ancestry: a branch cut
// and never committed to is merged by ancestry, and prune keeps it. The
// forge is not asked, since the board never asks it; a count that cannot be
// made omits the line, said on stderr.
func boardWorktrees(root, cwd string, rep peers.Report, stderr io.Writer) *boardWorktreesLine {
	if len(rep.Worktrees()) == 0 {
		return nil
	}
	got, err := worktree.Count(root, rep.Listing(), worktree.CountOptions{Held: loop.HeldWorktrees, Cwd: cwd})
	if err != nil {
		fmt.Fprintf(stderr, "abcd: the worktrees line is omitted — %s\n", termsafe.Sanitize(fsutil.DisplayPathsIn(err.Error(), root)))
		return nil
	}
	if got.Held == 0 {
		return nil
	}
	return &boardWorktreesLine{Held: got.Held, Reclaimable: got.Reclaimable}
}
