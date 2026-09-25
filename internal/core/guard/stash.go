package guard

import (
	"path"
	"strings"
	"sync"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// The shared stash stack (iss-2609190338340796). git keeps ONE stash stack per
// repository, not one per worktree, so in a clone with several worktrees a
// bare `git stash` in one lane and a bare `git stash pop` in another can hand
// the second lane the first lane's entry: the pop succeeds, the work lands in
// the wrong tree, and neither lane has a message it can act on. That is a fact
// about the repository, not about the command's words, so no Pattern can say
// it; the guard raises it under a reserved id, as a warn, and only when the
// repository it loaded for really has more than one worktree.

const (
	// stashEntryID is the reserved id the shared-stack warning is reported
	// under. It names a verdict the Pattern language cannot express (it turns
	// on the repository's worktree count), so no registry entry may claim it.
	stashEntryID = "git-stash-shared-stack"

	familyStash = "git stash"

	// maxWorktreeListBytes caps the worktree listing the count is read from.
	maxWorktreeListBytes = 1 << 20
)

// worktreeCounter returns a function that counts root's working trees (bare
// entries excluded) the first time it is asked, and 1 — a clone whose stack is
// not shared — when git cannot answer. Lazy because it costs a git process,
// and only a command holding a bare stash needs the answer.
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

// sharedStashSignal returns the warning for the first bare stash segment —
// `git stash` / `git stash push` without a message, or `git stash pop` /
// `git stash apply` without naming the entry — when the registry's repository
// has more than one worktree. A registry with no repository behind it (the
// bundled defaults on their own) cannot tell, and says nothing.
func (r Registry) sharedStashSignal(segs []segment, valueFlags []string) (payloadSignal, bool) {
	if r.worktrees == nil {
		return payloadSignal{}, false
	}
	for _, s := range segs {
		if !bareStash(s, valueFlags) {
			continue
		}
		if r.worktrees() < 2 {
			return payloadSignal{}, false
		}
		return sharedStashWarnSignal(), true
	}
	return payloadSignal{}, false
}

// bareStash reports whether a segment is a git stash that takes or gives the
// TOP of the shared stack without naming it.
func bareStash(s segment, valueFlags []string) bool {
	ci, noglob := commandIndex(s)
	if ci < 0 {
		return false
	}
	base := path.Base(s.tokens[ci])
	if !strings.EqualFold(base, "git") &&
		!(!noglob && s.globAt(ci) && globMatches(strings.ToLower(base), "git")) {
		return false
	}
	args := s.tokens[ci+1:]
	idx := operandIndexes(args, valueFlags)
	if len(idx) == 0 || args[idx[0]] != "stash" {
		return false
	}
	ops := make([]string, 0, len(idx)-1)
	for _, i := range idx[1:] {
		ops = append(ops, args[i])
	}
	rest := args[idx[0]+1:]
	switch {
	case len(ops) == 0:
		// `git stash [options]` is `git stash push [options]`.
		return !stashHasMessage(rest)
	case ops[0] == "push":
		return !stashHasMessage(rest)
	case ops[0] == "save":
		// The deprecated form takes its message as an operand.
		return len(ops) < 2
	case ops[0] == "pop" || ops[0] == "apply":
		return len(ops) < 2
	}
	// `git stash -- <pathspec>`, `git stash -p` and the like: a stash that
	// pushes, with the first operand a pathspec rather than a subcommand.
	if !isStashSubcommand(ops[0]) {
		return !stashHasMessage(rest)
	}
	return false
}

// isStashSubcommand reports whether a word is one of git stash's own
// subcommands rather than a pathspec.
func isStashSubcommand(w string) bool {
	switch w {
	case "list", "show", "drop", "pop", "apply", "branch", "push", "save", "clear", "create", "store", "export", "import":
		return true
	}
	return false
}

// stashHasMessage reports whether a stash's arguments carry a message, in any
// of the spellings git accepts.
func stashHasMessage(args []string) bool {
	for i, a := range args {
		if a == "--" {
			return false
		}
		switch {
		case a == "-m" || a == "--message":
			return i+1 < len(args)
		case strings.HasPrefix(a, "--message="):
			return true
		case strings.HasPrefix(a, "-m") && len(a) > 2 && !strings.HasPrefix(a, "--"):
			return true
		}
	}
	return false
}

// sharedStashWarnSignal is the warning itself. A WARN, not a block: a stash on
// a shared stack is only a hazard when two lanes use it at once, which the
// guard cannot see — but the lane that pops the wrong entry never learns it did.
func sharedStashWarnSignal() payloadSignal {
	return payloadSignal{
		id:      stashEntryID,
		verdict: VerdictWarn,
		family:  familyStash,
		reason: "git keeps one stash stack for the whole repository, and this repository has more than one worktree, " +
			"so a stash or pop that does not name its entry can take another worktree's work and land it in the wrong tree.",
		successor: "Stash with a message and pop that entry by name (`git stash push -m '<lane>: <why>'`, then `git stash list` and `git stash pop stash@{N}`), " +
			"or commit the work to a scratch commit on this worktree's own branch.",
	}
}
