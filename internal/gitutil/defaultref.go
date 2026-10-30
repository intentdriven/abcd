package gitutil

import (
	"context"
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
func DefaultRef(root string) string {
	return defaultRef(func(args ...string) (string, error) { return Run(root, args...) })
}

// DefaultRefContext is DefaultRef with every git question bounded by ctx
// (RunLimitedContext): a git still running when ctx ends is killed, and the
// answer is "". A caller that must tell "no default branch" from "git did not
// answer in time" checks ctx.Err() after it.
func DefaultRefContext(ctx context.Context, root string) string {
	return defaultRef(func(args ...string) (string, error) {
		out, err := RunLimitedContext(ctx, root, refAnswerCap, args...)
		return strings.TrimSpace(out), err
	})
}

// refAnswerCap bounds one ref question's answer: a ref name or an object name.
const refAnswerCap = 4096

// gitAsk is one read-only git question, its answer trimmed.
type gitAsk func(args ...string) (string, error)

func defaultRef(ask gitAsk) string {
	if ref := trackedDefaultBranch(ask, "origin"); ref != "" {
		return ref
	}
	for _, n := range defaultNames {
		if commitRefExists(ask, "refs/heads/"+n) {
			return "refs/heads/" + n
		}
	}
	return ""
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
// target, which asks the remote itself.
func TrackedDefaultBranch(root, remote string) string {
	return trackedDefaultBranch(func(args ...string) (string, error) { return Run(root, args...) }, remote)
}

func trackedDefaultBranch(ask gitAsk, remote string) string {
	prefix := "refs/remotes/" + remote + "/"
	if out, err := ask("symbolic-ref", "--quiet", prefix+"HEAD"); err == nil && strings.HasPrefix(out, prefix) && commitRefExists(ask, out) {
		return out
	}
	for _, n := range defaultNames {
		if commitRefExists(ask, prefix+n) {
			return prefix + n
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
