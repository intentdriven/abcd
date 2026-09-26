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
	if out, err := Run(root, "symbolic-ref", "--quiet", "refs/remotes/origin/HEAD"); err == nil && strings.HasPrefix(out, "refs/remotes/origin/") && commitRefExists(root, out) {
		return out
	}
	names := []string{"main", "master", "trunk", "develop"}
	for _, prefix := range []string{"refs/remotes/origin/", "refs/heads/"} {
		for _, n := range names {
			if commitRefExists(root, prefix+n) {
				return prefix + n
			}
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
