package loop

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/peers"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// unreadablePeers refuses a drain move, once and where per-issue contention
// would begin, when the peer listing names a peer it could not read
// (spc-2610161044461448 step 9; iss-2610161015397586, dcn-2610161040378713).
// What such a peer holds is unknown, so the peers check reads it as a holder
// of every record: left to that check, the drain would judge each issue and
// pass it over as contended until nothing eligible was left. The refusal names
// each peer with the repair its reason calls for instead.
//
// A worktree git has locked is not unreadable for being locked: the lock keeps
// `git worktree prune` and `remove` away from it and says nothing of whether
// it reads, so a locked worktree that reads is an ordinary peer.
func unreadablePeers(repoRoot string) error {
	rep, err := peers.Scan(repoRoot)
	if err != nil {
		return err
	}
	return unreadableRefusal(repoRoot, rep.Unjudged())
}

// unreadableRefusal is the move's refusal over the peers bad names, or nil
// when it names none. A start whose own peers check met an unreadable peer
// hands its peers here, so the move stops with this refusal rather than
// passing the issue over.
func unreadableRefusal(repoRoot string, bad []peers.Peer) error {
	if len(bad) == 0 {
		return nil
	}
	var named, remedies []string
	for _, p := range bad {
		who := peerName(p.Source, p.Branch, p.Path)
		named = append(named, who+" could not be read ("+p.NotReadShown(repoRoot)+")")
		remedies = append(remedies, who+": "+peerRemedy(p))
	}
	reason := fmt.Sprintf("%d peer(s) could not be read, so what each holds is unknown and every issue would read as held by it: %s",
		len(bad), strings.Join(named, "; "))
	remedy := "settle each, then `abcd drain` again: " + strings.Join(remedies, "; ") +
		"; `abcd peers` names each peer it cannot read"
	return refuse(StageDrain, CheckPeers, "", reason, remedy)
}

// peerRemedy is the repair p's reason calls for. None removes a worktree:
// what an unreadable peer holds is unknown, and even a plain
// `git worktree remove`, which stops at tracked changes and untracked files,
// deletes the worktree's ignored files without asking, a session's
// .abcd/.work.local/ among them. Nor does any advise --force.
//
//   - A ledger holding one id in two folders: its reason's own advice, to move
//     or remove one of each.
//   - A worktree another account owns: that account owns it. This account
//     cannot repair it, and no git command is offered.
//   - A worktree whose owner cannot be read: it says so, and offers no
//     command, since whose work it is cannot be told.
//   - A worktree git would not answer for: `git worktree repair`, run in this
//     repository with no argument, which points every worktree's git file back
//     at it and touches no work.
//   - Any other branch: settle what its reason names, or delete it once spent.
//   - Any other worktree: settle what its reason names, and before anyone
//     removes it, `git -C "<path>" status --ignored` to list the ignored files
//     a removal would delete, for whoever owns them to keep or move.
func peerRemedy(p peers.Peer) string {
	if strings.HasPrefix(p.NotRead, peers.NotReadSplit) {
		return "move or remove one of each so the ledger says which status the record is in"
	}
	if p.Source != peers.SourceWorktree {
		return "settle what its reason names, or delete the branch once its work is spent"
	}
	switch owner, whose := worktreeOwner(p); whose {
	case ownedElsewhere:
		return "it is owned by " + owner + ", not this account, which cannot read or repair it; ask whoever uses that account to repair or remove it"
	case ownerUnknown:
		return "its owner could not be read, so whose work it holds is unknown and no command is offered; find which account owns it, and leave it to that account to settle"
	}
	if strings.HasPrefix(p.NotRead, peers.NotReadGitRefused) {
		return "`git worktree repair`, run in this repository with no argument, points every worktree's git file back at it and touches no work"
	}
	return "settle what its reason names; no command to remove it is offered, because removing a worktree deletes its ignored files without asking, `.abcd/.work.local/` among them (a session's NEXT.md, scratch/, logs/ and transcripts/): `git -C " +
		commandPath(p.Path) + " status --ignored` lists them, and whoever owns that work keeps or moves them before anyone removes the worktree (a worktree another tool has locked is that tool's: ask it first)"
}

// peerOwnerUID is the owner lookup peerRemedy reads, a var because a second
// account is not something a test process can create.
var peerOwnerUID = fsutil.OwnerUID

// ownership is what can be told of whose a worktree is.
type ownership int

const (
	ownedHere      ownership = iota // this account owns it
	ownedElsewhere                  // another account owns it
	ownerUnknown                    // its owner could not be read
)

// worktreeOwner says whose p's worktree is, naming the account when it is
// another one: by its owner's uid where that can be read, or by git's own word
// that the worktree's owner is another account. A uid that cannot be read,
// with no such word from git, is an owner unknown, never this account.
func worktreeOwner(p peers.Peer) (string, ownership) {
	uid, err := peerOwnerUID(p.Path)
	if err == nil && uid != uint32(os.Getuid()) {
		return accountName(uid), ownedElsewhere
	}
	if strings.Contains(p.NotRead, "dubious ownership") {
		return "another account", ownedElsewhere
	}
	if err != nil {
		return "", ownerUnknown
	}
	return "", ownedHere
}

// accountName names the account uid is: its username, or its uid where the
// system names none.
func accountName(uid uint32) string {
	id := strconv.FormatUint(uint64(uid), 10)
	if u, err := user.LookupId(id); err == nil && u.Username != "" {
		return "the account " + u.Username
	}
	return "the account with uid " + id
}

// commandPath is a path as a command the reader types names it: the path
// itself, with HOME as "$HOME", which a shell expands inside double quotes
// where it would not expand ~, and the rest in single quotes, inside which no
// shell expands anything, the `!` an interactive bash or zsh would read as
// history included, with each ' written '\”. A path outside HOME is
// single-quoted whole. fsutil.DisplayPath is for prose: outside HOME it is a
// folder name, which no command can act on.
//
// HOME is "$HOME" only as the path's leading prefix, the whole path or its
// directory, in its own spelling or its symlink-resolved one: HOME met
// anywhere else in the path is the path's own bytes, and fsutil.RedactHome,
// which redacts it wherever it starts a path in a message, would print a
// different path there.
func commandPath(p string) string {
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }
	for _, h := range homeSpellings() {
		if p == h {
			return `"$HOME"`
		}
		if rest, ok := strings.CutPrefix(p, h+"/"); ok {
			return `"$HOME"` + quote("/"+rest)
		}
	}
	return quote(p)
}

// homeSpellings is HOME, cleaned, and its symlink-resolved spelling when that
// differs: the two fsutil.RedactHome redacts. It is none when HOME cannot be
// read, is relative, or is the root, which prefixes every absolute path.
func homeSpellings() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	home = filepath.Clean(home)
	if len(home) <= 1 || !filepath.IsAbs(home) {
		return nil
	}
	spellings := []string{home}
	if real, err := filepath.EvalSymlinks(home); err == nil && real != home && len(real) > 1 {
		spellings = append(spellings, real)
	}
	return spellings
}

// drainBeforeStart runs, when set, just before the drain starts an issue's
// lane: a test seam for a peer that turns unreadable between the move's peers
// check and the start's own, which no real ordering can be relied on to hit.
var drainBeforeStart func(issue string)
