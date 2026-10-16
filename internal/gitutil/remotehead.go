package gitutil

import (
	"errors"
	"fmt"
	"strings"
)

// RemoteHeadBranch asks the remote which branch is its default and returns
// the bare branch name: `git ls-remote --symref <remote> HEAD`, through Net (a
// network call, its environment scrubbed and its time bounded), read for the
// `ref: refs/heads/<branch>\tHEAD` line the remote advertises.
//
// It is the authority for an act on the remote, unlike DefaultRef, which
// reads the tracked refs/remotes/origin/HEAD as this checkout last fetched it
// (or a conventional name), and so can be stale or missing. It never fetches
// and never writes: origin/HEAD stays as the repository has it.
//
// An answer is an error, never a guess, when the remote cannot be asked, when
// it advertises no symref for HEAD (a detached HEAD), when the target is
// outside refs/heads/, when the name is empty or HEAD (whose tracking ref
// would be origin/HEAD itself), or when `git check-ref-format` refuses it.
func RemoteHeadBranch(root, remote string) (string, error) {
	if !RefIsSafe(remote) {
		return "", errors.New("the remote's name is empty or option-shaped")
	}
	out, err := Net(root, "ls-remote", "--symref", remote, "HEAD")
	if err != nil {
		return "", fmt.Errorf("asking %s for its default branch: %w", remote, err)
	}
	return remoteHeadFrom(root, remote, out)
}

// remoteHeadFrom reads the default branch out of out, the remote's answer to
// `git ls-remote --symref <remote> HEAD`, checking the name locally.
func remoteHeadFrom(root, remote, out string) (string, error) {
	for _, line := range strings.Split(out, "\n") {
		rest, ok := strings.CutPrefix(line, "ref: ")
		if !ok {
			continue
		}
		target, name, ok := strings.Cut(rest, "\t")
		if !ok || name != "HEAD" {
			continue
		}
		branch, ok := strings.CutPrefix(target, "refs/heads/")
		if !ok || branch == "" || branch == "HEAD" {
			return "", fmt.Errorf("%s's HEAD does not name a branch", remote)
		}
		if _, err := LocalGit(root, "check-ref-format", "refs/heads/"+branch); err != nil {
			return "", fmt.Errorf("%s's HEAD names a branch whose name is not a valid ref", remote)
		}
		return branch, nil
	}
	return "", fmt.Errorf("%s did not name a default branch (its HEAD is detached, or it advertises no symref)", remote)
}
