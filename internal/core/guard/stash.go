package guard

import (
	"sync"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// The shared stash list (iss-2609190338340796, iss-2611220942432452). git keeps
// ONE stash list per repository, not one per worktree, so in a clone with
// several worktrees a stash in one lane and a pop in another can hand the
// second lane the first lane's work: the pop succeeds, the work lands in the
// wrong tree, and neither lane has a message it can act on. Naming the entry
// does not help, since `stash@{0}` is whatever entry is newest when the command
// runs. The warning is the registry entry stashEntryID, so the SHELL rules
// teach it; two of its conditions are facts the Pattern language cannot say —
// whether the stash changes the list, and whether the repository it loaded for
// has linked worktrees — and the matcher reads them here (stashEntryStands).

const (
	// stashEntryID is the registry entry that warns on a stash which changes
	// the shared list.
	stashEntryID = "git-stash-shared-stack"

	// maxWorktreeListBytes caps the worktree listing the count is read from.
	maxWorktreeListBytes = 1 << 20
)

// stashValueFlags are git stash's own options that take the next word, so a
// message or a pathspec file is never read as the stash's subcommand (`git
// stash -m list` is a push).
var stashValueFlags = []string{"-m", "--message", "--pathspec-from-file"}

// worktreeCounter returns a function that counts root's working trees (bare
// entries excluded) the first time it is asked, and 1 — a clone whose list is
// not shared — when git cannot answer. Lazy because it costs a git process,
// and only a command holding a stash that changes the list needs the answer.
func worktreeCounter(root string) func() int {
	var (
		once sync.Once
		n    = 1
	)
	return func() int {
		once.Do(func() {
			wts, err := gitutil.ListWorktrees(root, maxWorktreeListBytes)
			if err != nil {
				return
			}
			count := 0
			for _, wt := range wts {
				if !wt.Bare {
					count++
				}
			}
			if count > 0 {
				n = count
			}
		})
		return n
	}
}

// stashEntryStands reports whether a match of entry id on segment s stands. It
// is true for every entry but stashEntryID. That entry's pattern names `git
// stash`, so it stands only where the stash changes the list
// (stashChangesList) and the repository has more than one worktree. A registry
// with no repository behind it (the bundled defaults on their own) cannot
// count, and lets it stand: the warning's side.
func (r Registry) stashEntryStands(id string, s segment) bool {
	if id != stashEntryID {
		return true
	}
	valueFlags := append(append([]string(nil), r.gitValueFlags()...), stashValueFlags...)
	if !stashChangesList(s, valueFlags) {
		return false
	}
	return r.worktrees == nil || r.worktrees() >= 2
}

// stashChangesList reports whether a segment holds a git stash that changes
// the shared list: anything but `git stash list` and `git stash show`, which
// only read it. Every place git can sit and every reading of its operands is
// read (unknown.go); readings past their bound, and a subcommand the line does
// not fix, are taken as changing it, the warning's side.
func stashChangesList(s segment, valueFlags []string) bool {
	for _, site := range sitesNamed(s, "git") {
		args := s.tokens[site.idx+1:]
		readings, complete := operandReadings(args, valueFlags, 2)
		if !complete {
			return true
		}
		for _, idx := range readings {
			if len(idx) == 0 || !wordCouldBe(args[idx[0]], "stash") {
				continue
			}
			if len(idx) == 1 {
				// `git stash [options]` is `git stash push [options]`.
				return true
			}
			op := args[idx[1]]
			// `git stash -- list` pushes the pathspec `list`.
			if isUnknown(op) || (op != "list" && op != "show") || containsString(args[idx[0]+1:idx[1]], "--") {
				return true
			}
		}
	}
	return false
}
