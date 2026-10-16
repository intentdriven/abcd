package gitutil

import "strings"

// DefaultRef is the full ref name of the repository's default branch as last
// fetched, with no network: origin/HEAD's target, then the conventional names
// on the remote, then the same names locally. "" when none resolves.
//
// It is the one resolution the readers that measure against the default
// branch share: the peers reader judges a branch merged into it, and the
// reviews board counts the commits it has moved since a review's pin. Two
// readers of one repository asking the same question must get one answer.
func DefaultRef(root string) string {
	if ref := RemoteDefaultBranch(root, "origin"); ref != "" {
		return ref
	}
	for _, n := range defaultNames {
		if commitRefExists(root, "refs/heads/"+n) {
			return "refs/heads/" + n
		}
	}
	return ""
}

// defaultNames are the conventional default-branch names, in the order a
// probe tries them.
var defaultNames = []string{"main", "master", "trunk", "develop"}

// RemoteDefaultBranch is the named remote's default branch as this checkout
// last fetched it, with no network: the remote's HEAD target, then the
// conventional names under the remote, and never a local branch. It is a ref
// under refs/remotes/<remote>/, or "" when none resolves: a verb that acts on
// what the remote holds (a reservation's release, a land) must not take a
// local branch the remote may not have for the remote's.
func RemoteDefaultBranch(root, remote string) string {
	prefix := "refs/remotes/" + remote + "/"
	if out, err := Run(root, "symbolic-ref", "--quiet", prefix+"HEAD"); err == nil && strings.HasPrefix(out, prefix) && commitRefExists(root, out) {
		return out
	}
	for _, n := range defaultNames {
		if commitRefExists(root, prefix+n) {
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

func commitRefExists(root, ref string) bool {
	_, err := Run(root, "rev-parse", "--verify", "--quiet", ref+"^{commit}", "--")
	return err == nil
}
