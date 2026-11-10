package gitutil

import (
	"context"
	"errors"
	"strings"
)

// DefaultRef is the full ref name of the repository's default branch as last
// fetched, with no network: origin/HEAD's target, then the conventional names
// on the remote, then the same names locally. "" when none resolves.
//
// It is the one resolution the readers that measure against the default
// branch share: the peers reader judges a branch merged into it, and the
// reviews board counts the commits it has moved since a review's pin. Two
// readers of one repository asking the same question must get one answer.
// A listing past its 64 KiB cap is refused whole, and the branch is then asked
// ref by ref: the same answer, for a few more git processes.
func DefaultRef(root string) string {
	return defaultRef(askCapped(root))
}

// DefaultRefContext is DefaultRef with every git question bounded by ctx
// (RunLimitedContext): a git still running when ctx ends is killed, and the
// answer is "". A caller that must tell "no default branch" from "git did not
// answer in time" checks ctx.Err() after it.
func DefaultRefContext(ctx context.Context, root string) string {
	return defaultRef(askContext(ctx, root))
}

// askCapped is one ref question with no deadline, its answer refused whole
// past refAnswerCap: the cap askContext holds, so a default-branch form and
// its Context form ask the same questions of one repository and differ in the
// context alone.
func askCapped(root string) gitAsk {
	return func(args ...string) (string, error) {
		return refAnswer(runBounded(root, refAnswerCap, args...))
	}
}

// askContext is one ref question bounded by ctx (RunLimitedContext), its
// answer refused whole past refAnswerCap.
func askContext(ctx context.Context, root string) gitAsk {
	return func(args ...string) (string, error) {
		return refAnswer(runLimitedOverflow(ctx, root, refAnswerCap, false, args))
	}
}

// refAnswer is one ref question's answer, trimmed, or errRefAnswerCut for an
// answer that ran past its cap: askCapped's and askContext's one body.
func refAnswer(out string, overflowed bool, err error) (string, error) {
	if err == nil && overflowed {
		return "", errRefAnswerCut
	}
	return strings.TrimSpace(out), err
}

// refAnswerCap bounds one ref question's answer: a ref name, an object name, or
// the listing, which also holds every ref below a conventional name. An answer
// past it is refused whole, so the resolution asks ref by ref rather than reading
// the part it got as the whole.
const refAnswerCap = 64 << 10

var errRefAnswerCut = errors.New("git's ref answer ran past its bound")

// gitAsk is one read-only git question, its answer trimmed.
type gitAsk func(args ...string) (string, error)

func defaultRef(ask gitAsk) string {
	return resolveDefault(ask, "origin", true)
}

// defaultNames are the conventional default-branch names, in the order a
// probe tries them.
var defaultNames = []string{"main", "master", "trunk", "develop"}

// TrackedDefaultBranch is the named remote's default branch as this checkout
// last fetched it, with no network: the remote's HEAD target, then the
// conventional names under the remote, and never a local branch. It is a ref
// under refs/remotes/<remote>/, or "" when none resolves.
//
// It is a local reading, so it can be stale (the remote's HEAD moved since the
// last fetch) or guessed (no HEAD recorded, so the first conventional name
// present wins). It serves readers that measure against the default branch;
// it must never decide an act on the remote, such as which branch a push may
// target, which asks the remote itself. A listing past its 64 KiB cap is
// refused whole, and the branch is then asked ref by ref: the same answer, for
// a few more git processes.
func TrackedDefaultBranch(root, remote string) string {
	return trackedDefaultBranch(askCapped(root), remote)
}

// TrackedDefaultBranchContext is TrackedDefaultBranch with every git question
// bounded by ctx, as DefaultRefContext bounds its own: a ctx already ended
// starts no git, a git still running when it ends is killed, and the answer
// is "". Its answers are capped as TrackedDefaultBranch's are. A caller that
// must tell "no default branch" from "git did not answer in time" checks
// ctx.Err() after it.
func TrackedDefaultBranchContext(ctx context.Context, root, remote string) string {
	return trackedDefaultBranch(askContext(ctx, root), remote)
}

func trackedDefaultBranch(ask gitAsk, remote string) string {
	return resolveDefault(ask, remote, false)
}

// defaultRefFormat is one row of the default-branch listing. Refnames hold no
// space, and the last field keeps a row's trailing empties through the answer's
// trim.
const defaultRefFormat = "--format=%(refname) %(symref) %(objecttype) %(*objecttype) ."

// resolveDefault answers from one listing of every ref the resolution can name,
// where a question per ref cost up to ten processes on every call, and most of
// the git a repository with no origin/HEAD spends. The order is the remote's HEAD
// target, the conventional names under the remote, then, when local says so, the
// same names locally.
//
// A listing has no row for what a ref's own read could still find. It shows no
// HEAD that is a symlink, and it fails whole on one ref with a missing object, so
// with no HEAD row symbolic-ref asks. On a file system that folds case,
// refs/heads/main opens a loose ref spelt Main, so when the listing names nothing
// it asks ref by ref, as every read before it did, and only a repository with no
// default branch pays for that. A dangling HEAD has no row, and falls through to
// the names as its failed read did. One answer differs from a question per ref:
// a loose origin/Main beside origin/master on such a file system now answers
// master, which is the branch that exists under that name.
func resolveDefault(ask gitAsk, remote string, local bool) string {
	prefix := "refs/remotes/" + remote + "/"
	candidates := make([]string, 0, 2*len(defaultNames))
	for _, n := range defaultNames {
		candidates = append(candidates, prefix+n)
	}
	if local {
		for _, n := range defaultNames {
			candidates = append(candidates, "refs/heads/"+n)
		}
	}
	rows := map[string][]string{}
	args := append([]string{"for-each-ref", defaultRefFormat, prefix + "HEAD"}, candidates...)
	if out, err := ask(args...); err == nil {
		for _, line := range strings.Split(out, "\n") {
			if f := strings.Split(line, " "); len(f) == 5 {
				rows[f[0]] = f[1:4]
			}
		}
	}
	peelsToCommit := func(ref string, row []string) bool {
		switch row[1] {
		case "commit":
			return true
		case "tag":
			// A tag of a tag peels on: older gits list only its first level.
			return row[2] == "commit" || commitRefExists(ask, ref)
		}
		return false
	}
	if head, ok := rows[prefix+"HEAD"]; ok && strings.HasPrefix(head[0], prefix) && peelsToCommit(head[0], head) {
		return head[0]
	}
	if _, ok := rows[prefix+"HEAD"]; !ok {
		target, err := ask("symbolic-ref", "--quiet", prefix+"HEAD")
		if err == nil && strings.HasPrefix(target, prefix) && commitRefExists(ask, target) {
			return target
		}
	}
	for _, ref := range candidates {
		if row, ok := rows[ref]; ok && peelsToCommit(ref, row) {
			return ref
		}
	}
	for _, ref := range candidates {
		if commitRefExists(ask, ref) {
			return ref
		}
	}
	return ""
}

// ShortRef renders a full ref the way a person names it: origin/main for a
// remote-tracking ref, main for a local branch.
func ShortRef(ref string) string {
	if s, ok := strings.CutPrefix(ref, "refs/remotes/"); ok {
		return s
	}
	return strings.TrimPrefix(ref, "refs/heads/")
}

func commitRefExists(ask gitAsk, ref string) bool {
	_, err := ask("rev-parse", "--verify", "--quiet", ref+"^{commit}", "--")
	return err == nil
}
