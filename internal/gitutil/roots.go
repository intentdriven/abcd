package gitutil

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// noGraftFile is a GIT_GRAFT_FILE value naming nothing: git reads no graft
// file at all, so a .git/info/grafts file cannot give a commit parents it
// does not store. /dev/null would do the same, but git still prints its
// deprecation hint for it; a path under it cannot exist.
const noGraftFile = "/dev/null/none"

// maxRootLine bounds one line of the root listing: a timestamp and an object
// name are under a hundred bytes.
const maxRootLine = 4096

// DefaultBranchRoots lists every root commit of origin's default branch as
// last fetched (DefaultRef, when it names a ref under refs/remotes/origin/;
// HEAD otherwise), oldest committer date first, ties in object-name order.
// The history is read as stored: with replace refs off
// (core.useReplaceRefs=false and GIT_NO_REPLACE_OBJECTS) and no graft file
// (noGraftFile), so neither can add a root or hide one.
//
// The list only grows by a merge that brings in an unrelated history (`git
// subtree add`, `--allow-unrelated-histories`, an empty-diff `merge -s
// ours`); in an ordinary repository it holds one root, RootCommit's answer.
// It is read as git writes it, with no cap on its length, so one merge of
// many parentless commits cannot make it unreadable.
//
// No commits to read (HEAD names no commit: `rev-parse --verify --quiet`
// exits 1) is an empty list and no error; any other failure of git, there
// or listing the roots, or a line of a shape git never writes, is an error.
func DefaultBranchRoots(root string) ([]string, error) {
	rev := "HEAD"
	if ref := DefaultRef(root); strings.HasPrefix(ref, "refs/remotes/origin/") {
		rev = ref
	} else if _, noCommit, err := RunCappedExitOne(root, maxRootLine, "rev-parse", "--verify", "--quiet", "HEAD^{commit}", "--"); err != nil {
		return nil, fmt.Errorf("git rev-parse HEAD: %w", err)
	} else if noCommit {
		return nil, nil
	}
	args := []string{"-c", "core.useReplaceRefs=false", "rev-list", "--max-parents=0", "--timestamp", rev, "--"}
	cmd := isolatedGit(root, args...)
	cmd.Env = append(cmd.Env, "GIT_NO_REPLACE_OBJECTS=1", "GIT_GRAFT_FILE="+noGraftFile)
	type dated struct {
		at  int64
		sha string
	}
	var roots []dated
	errShape := fmt.Errorf("git rev-list --max-parents=0: a line of an unexpected shape")
	_, err := streamFields(cmd, '\n', maxRootLine, func(line string) error {
		if line == "" {
			return nil
		}
		f := strings.Fields(line)
		if len(f) != 2 || !IsFullSHA(f[1]) {
			return errShape
		}
		at, err := strconv.ParseInt(f[0], 10, 64)
		if err != nil {
			return errShape
		}
		roots = append(roots, dated{at: at, sha: f[1]})
		return nil
	})
	if err != nil {
		return nil, err
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
