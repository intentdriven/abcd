package worktree

// count.go is the /abcd board's worktrees line (spc-2609301811532881,
// criterion 17): how many worktrees the lane holds, and how many of them prune
// would reclaim. The board never asks the forge, so reclaimable is prune's
// answer with no forge: the store's own merged judgement (the start record,
// the own-commit rule, the rebase and squash tests), and every refusal prune
// keeps a worktree for. A branch git calls merged by ancestry alone, such as
// one cut and never committed to, is not counted.

import (
	"errors"
	"fmt"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// CountOptions is what Count needs beyond the listing.
type CountOptions struct {
	// Held reads what the build loop holds, as Prune's Held does. It is read
	// only when some worktree would otherwise be reclaimable.
	Held HeldReader
	// Cwd is the caller's working directory: prune keeps its own, so a
	// worktree it stands in is not counted reclaimable.
	Cwd string
}

// Counted is Count's answer.
type Counted struct {
	// Held is the worktrees git lists in the lane whose directory is there.
	Held int
	// Reclaimable is how many of them a prune with no forge answer would
	// reclaim.
	Reclaimable int
}

// Count judges listed, git's worktree listing the caller already read (main
// working tree first, as gitutil.ListWorktrees returns it), against the
// repository at repoRoot's lane, without listing again. A lane level the store
// would refuse is refused here too; an absent lane holds nothing.
func Count(repoRoot string, listed []gitutil.Worktree, opts CountOptions) (Counted, error) {
	if opts.Held == nil {
		return Counted{}, errors.New("worktree: Count needs a reader of what the build loop holds")
	}
	lane, err := LaneFor(repoRoot)
	if err != nil {
		return Counted{}, err
	}
	if ok, err := lane.present(); err != nil || !ok {
		return Counted{}, err
	}
	own, err := gitutil.CommonDir(repoRoot)
	if err != nil {
		return Counted{}, fmt.Errorf("reading the checkout's common directory: %w", err)
	}
	own = fsutil.RealExistingPath(own)
	realLane := fsutil.RealExistingPath(lane.Dir)
	var es []entry
	for i, wt := range listed {
		if i == 0 || wt.Bare || wt.Prunable {
			continue
		}
		if e := listedEntry(wt, own, realLane); e.where == WhereStore {
			es = append(es, e)
		}
	}
	judgeAll(repoRoot, lane, unjudged(es), nil)

	var out Counted
	var held *Held
	cwd := ""
	if opts.Cwd != "" {
		cwd = fsutil.RealExistingPath(opts.Cwd)
	}
	dry := PruneOptions{DryRun: true}
	for _, e := range es {
		out.Held++
		// Only a clean, merged worktree can be one an unnamed prune
		// reclaims; the rest skip the build loop's read and the notes read.
		if e.reason != "" || !e.clean || !e.merged.merged {
			continue
		}
		if held == nil {
			h, err := opts.Held(repoRoot)
			if err != nil {
				return Counted{}, fmt.Errorf("reading what the build loop holds: %w", err)
			}
			held = &h
		}
		row := PruneRow{Outcome: OutcomeKept}
		if row.Reason = refusedFor(e, *held, cwd); row.Reason == "" {
			decide(repoRoot, lane, e, dry, false, &row)
		}
		if row.Outcome == OutcomeWouldReclaim {
			out.Reclaimable++
		}
	}
	return out, nil
}
