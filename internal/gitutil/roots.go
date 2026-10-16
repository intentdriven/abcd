package gitutil

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// NoGraftFile is a GIT_GRAFT_FILE value naming nothing: git reads no graft
// file at all, so a .git/info/grafts file cannot give a commit parents it
// does not store. /dev/null would do the same, but git still prints its
// deprecation hint for it; a path under it cannot exist.
const NoGraftFile = "/dev/null/none"

// maxRootListing caps DefaultBranchRoots' answer: about a thousand roots,
// far past any real repository's, and an error past it, never a short list.
const maxRootListing = 64 << 10

// DefaultBranchRoots lists every root commit of origin's default branch as
// last fetched (DefaultRef, when it names a ref under refs/remotes/origin/;
// HEAD otherwise), oldest committer date first, ties in object-name order.
// The history is read as stored: with replace refs off
// (core.useReplaceRefs=false and GIT_NO_REPLACE_OBJECTS) and no graft file
// (NoGraftFile), so neither can add a root or hide one.
//
// A store keyed on a root commit uses the whole list, not the first root
// `rev-list` meets: that one is the newest by date, so a merge bringing in an
// unrelated history (`git subtree add`, `--allow-unrelated-histories`, an
// empty-diff `merge -s ours`) changes it, and a store looked for under it is
// not found. The list only grows by such a merge. In an ordinary repository
// it holds one root, RootCommit's answer.
//
// No commits to read is an empty list and no error; a git failure, or a list
// past the cap, is an error.
func DefaultBranchRoots(root string) ([]string, error) {
	rev := "HEAD"
	if ref := DefaultRef(root); strings.HasPrefix(ref, "refs/remotes/origin/") {
		rev = ref
	} else if _, err := Run(root, "rev-parse", "--verify", "--quiet", "HEAD^{commit}", "--"); err != nil {
		return nil, nil
	}
	args := []string{"-c", "core.useReplaceRefs=false", "rev-list", "--max-parents=0", "--timestamp", rev, "--"}
	cmd := isolatedGit(root, args...)
	cmd.Env = append(cmd.Env, "GIT_NO_REPLACE_OBJECTS=1", "GIT_GRAFT_FILE="+NoGraftFile)
	out, overflowed, err := runBoundedCmd(cmd, maxRootListing)
	if err != nil {
		return nil, err
	}
	if overflowed {
		return nil, fmt.Errorf("git rev-list --max-parents=0: more root commits than the %d-byte cap holds", maxRootListing)
	}
	type dated struct {
		at  int64
		sha string
	}
	var roots []dated
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		f := strings.Fields(line)
		if len(f) != 2 || !IsFullSHA(f[1]) {
			return nil, fmt.Errorf("git rev-list --max-parents=0: a line of an unexpected shape")
		}
		at, err := strconv.ParseInt(f[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("git rev-list --max-parents=0: a line of an unexpected shape")
		}
		roots = append(roots, dated{at: at, sha: f[1]})
	}
	slices.SortFunc(roots, func(a, b dated) int {
		return cmp.Or(cmp.Compare(a.at, b.at), strings.Compare(a.sha, b.sha))
	})
	shas := make([]string, 0, len(roots))
	for _, r := range roots {
		shas = append(shas, r.sha)
	}
	return shas, nil
}
