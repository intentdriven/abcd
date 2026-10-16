package loop

import (
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/core/peers"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// unreadablePeers refuses a drain move, once and before any reservation or
// judgement, when the peer listing names a peer it could not read
// (spc-2610161044461448 step 9; iss-2610161015397586, dcn-2610161040378713).
// What such a peer holds is unknown, so the peers check reads it as a holder
// of every record: left to that check, the drain would judge each issue and
// pass it over as contended until nothing eligible was left. The refusal names
// each peer with its repair instead.
//
// A worktree git has locked is not unreadable for being locked: the lock keeps
// `git worktree prune` and `remove` away from it and says nothing of whether
// it reads, so a locked worktree that reads is an ordinary peer.
func unreadablePeers(repoRoot string) error {
	rep, err := peers.Scan(repoRoot)
	if err != nil {
		return err
	}
	bad := rep.Unjudged()
	if len(bad) == 0 {
		return nil
	}
	var named, repairs []string
	branches := false
	for _, p := range bad {
		named = append(named, peerName(p.Source, p.Branch, p.Path)+" could not be read ("+fsutil.DisplayPathsIn(p.NotRead, p.Path)+")")
		if p.Source != peers.SourceWorktree {
			branches = true
			continue
		}
		shown := fsutil.DisplayPath(p.Path)
		repairs = append(repairs, fmt.Sprintf("`git worktree repair %s` to point it back at this repository, or, once its work is spent, `git worktree remove --force %s`", shown, shown))
	}
	if branches {
		repairs = append(repairs, "for a branch, settle what its reason names, or delete the branch once its work is spent")
	}
	reason := fmt.Sprintf("%d peer(s) could not be read, so what each holds is unknown and every issue would read as held by it: %s",
		len(bad), strings.Join(named, "; "))
	remedy := "repair or remove each, then `abcd drain` again: " + strings.Join(repairs, "; ") +
		"; a worktree another tool has locked is that tool's, so ask it, or unlock the worktree, before removing it; `abcd peers` names each peer it cannot read"
	return refuse(StageDrain, CheckPeers, "", reason, remedy)
}
