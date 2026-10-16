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
		named = append(named, who+" could not be read ("+notReadShown(repoRoot, p)+")")
		remedies = append(remedies, who+": "+peerRemedy(p))
	}
	reason := fmt.Sprintf("%d peer(s) could not be read, so what each holds is unknown and every issue would read as held by it: %s",
		len(bad), strings.Join(named, "; "))
	remedy := "settle each, then `abcd drain` again: " + strings.Join(remedies, "; ") +
		"; `abcd peers` names each peer it cannot read"
	return refuse(StageDrain, CheckPeers, "", reason, remedy)
}

// peerRemedy is the repair p's reason calls for. None forces anything: a
// forced `git worktree remove` deletes the uncommitted work of a worktree,
// a peer's included, so where removal is apt the remedy gives the plain form,
// which refuses while there is any.
//
//   - A ledger holding one id in two folders: its reason's own advice, to move
//     or remove one of each.
//   - A worktree another account owns: that account owns it. This account
//     cannot repair it, and no git command is offered.
//   - A worktree git would not answer for: `git worktree repair`, run in this
//     repository with no argument, which points every worktree's git file back
//     at it and touches no work.
//   - Any other branch: settle what its reason names, or delete it once spent.
//   - Any other worktree: settle what its reason names, or, once its work is
//     spent, the plain `git worktree remove`.
func peerRemedy(p peers.Peer) string {
	if strings.HasPrefix(p.NotRead, peers.NotReadSplit) {
		return "move or remove one of each so the ledger says which status the record is in"
	}
	if p.Source != peers.SourceWorktree {
		return "settle what its reason names, or delete the branch once its work is spent"
	}
	if owner, ok := foreignOwner(p); ok {
		return "it is owned by " + owner + ", not this account, which cannot read or repair it; ask whoever uses that account to repair or remove it"
	}
	if strings.HasPrefix(p.NotRead, peers.NotReadGitRefused) {
		return "`git worktree repair`, run in this repository with no argument, points every worktree's git file back at it and touches no work"
	}
	return "settle what its reason names, or, once its work is spent, `git worktree remove " + commandPath(p.Path) +
		"`, which refuses while it holds uncommitted work (a worktree another tool has locked is that tool's: ask it before removing it)"
}

// peerOwnerUID is the owner lookup peerRemedy reads, a var because a second
// account is not something a test process can create.
var peerOwnerUID = fsutil.OwnerUID

// foreignOwner names the account that owns a worktree when it is not this
// one: by its owner's uid where that can be read, or, where it cannot, by
// git's own word that the worktree's owner is another account.
func foreignOwner(p peers.Peer) (string, bool) {
	if uid, err := peerOwnerUID(p.Path); err == nil && uid != uint32(os.Getuid()) {
		return accountName(uid), true
	}
	if strings.Contains(p.NotRead, "dubious ownership") {
		return "another account", true
	}
	return "", false
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
// itself, with HOME as $HOME, which a shell expands inside double quotes where
// it would not expand ~, quoted whole with what a double quote reads escaped.
// fsutil.DisplayPath is for prose: outside HOME it is a folder name, which no
// command can act on.
func commandPath(p string) string {
	shown := fsutil.RedactHome(p)
	if shown == "~" {
		return `"$HOME"`
	}
	prefix := ""
	if rest, ok := strings.CutPrefix(shown, "~/"); ok {
		prefix, shown = "$HOME/", rest
	}
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `\$`, "`", "\\`")
	return `"` + prefix + r.Replace(shown) + `"`
}

// notReadShown is p's not-read reason with no absolute local path in it: the
// peer's own directory as fsutil.DisplayPathsIn names it, HOME as ~, this
// checkout as ., and any other absolute path, such as where a broken git file
// points, by its last element alone.
func notReadShown(repoRoot string, p peers.Peer) string {
	s := fsutil.DisplayPathsIn(p.NotRead, p.Path)
	if filepath.IsAbs(repoRoot) {
		s = fsutil.RedactRoot(s, repoRoot, ".")
		if real, err := filepath.EvalSymlinks(repoRoot); err == nil && real != repoRoot {
			s = fsutil.RedactRoot(s, real, ".")
		}
	}
	return displayAbsPaths(s)
}

// displayAbsPaths replaces each absolute path left in s, one starting where a
// path can (the start, or after a space, a quote, a bracket or an equals
// sign, so a URL's // is left alone) and running to the next space or quote, with
// fsutil.DisplayPath's form of it. A path in a message has no reliable end, so
// one holding a space keeps the words after it; what it never keeps is the
// directories above them.
func displayAbsPaths(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '/' || (i > 0 && !strings.ContainsRune(" \t\n'\"([=", rune(s[i-1]))) {
			b.WriteByte(s[i])
			i++
			continue
		}
		end := i + strings.IndexFunc(s[i:]+" ", func(r rune) bool { return strings.ContainsRune(" \t\n'\")]", r) })
		tok := strings.TrimRight(s[i:end], ".,;:")
		if tok == "/" {
			b.WriteString(tok)
		} else {
			b.WriteString(fsutil.DisplayPath(tok))
		}
		i += len(tok)
	}
	return b.String()
}

// drainBeforeStart runs, when set, just before the drain starts an issue's
// lane: a test seam for a peer that turns unreadable between the move's peers
// check and the start's own, which no real ordering can be relied on to hit.
var drainBeforeStart func(issue string)
