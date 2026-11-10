package gitutil

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// RemoteHeadBranch asks the remote which branch is its default and returns
// the bare branch name: `git ls-remote --symref <remote> HEAD`, run through
// git, the caller's networked git runner (nil is Net: a network call, its
// environment scrubbed and its time bounded), read for the
// `ref: refs/heads/<branch>\tHEAD` line the remote advertises. A caller with
// a seam for its networked git (abcd land's Options.Git) passes it, so the
// call is the caller's to record and its answer the caller's to fake.
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
func RemoteHeadBranch(git func(dir string, args ...string) (string, error), root, remote string) (string, error) {
	if !RefIsSafe(remote) {
		return "", errors.New("the remote's name is empty or option-shaped")
	}
	if git == nil {
		git = Net
	}
	out, err := git(root, "ls-remote", "--symref", remote, "HEAD")
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

// RemoteHeadTip asks the remote which branch is its default and which commit
// that branch's tip is: `git ls-remote --symref <remote> HEAD`, run through
// git, the caller's runner (nil is NetContext: the person's own configuration
// and credentials, none of the checkout's hooks, bounded by ctx as well as
// NetTimeout, its answer refused whole past NetContext's output cap). It never
// fetches and never writes.
//
// The answer is the remote's, so it is read as untrusted: it must carry
// exactly one symref line for HEAD, naming a branch under refs/heads/ that
// `git check-ref-format` accepts (asked under ctx), and exactly one object
// line for HEAD, a full object name. Anything else is an error, never a guess.
func RemoteHeadTip(ctx context.Context, git func(ctx context.Context, dir string, args ...string) (string, error), root, remote string) (branch, tip string, err error) {
	if !RefIsSafe(remote) {
		return "", "", errors.New("the remote's name is empty or option-shaped")
	}
	if git == nil {
		git = NetContext
	}
	out, err := git(ctx, root, "ls-remote", "--symref", remote, "HEAD")
	if err != nil {
		return "", "", fmt.Errorf("asking %s for its default branch: %w", remote, err)
	}
	var symrefs, tips int
	for _, line := range strings.Split(out, "\n") {
		if rest, ok := strings.CutPrefix(line, "ref: "); ok {
			if target, name, ok := strings.Cut(rest, "\t"); ok && name == "HEAD" {
				symrefs++
				branch = target
			}
		} else if sha, name, ok := strings.Cut(line, "\t"); ok && name == "HEAD" {
			tips++
			tip = sha
		}
	}
	if symrefs != 1 || tips != 1 {
		return "", "", fmt.Errorf("%s did not name one default branch and one tip for it (%d symref and %d object lines for HEAD)", remote, symrefs, tips)
	}
	branch, ok := strings.CutPrefix(branch, "refs/heads/")
	if !ok || branch == "" || branch == "HEAD" {
		return "", "", fmt.Errorf("%s's HEAD does not name a branch", remote)
	}
	if !IsFullSHA(tip) {
		return "", "", fmt.Errorf("%s's HEAD tip is not a full object name", remote)
	}
	if _, err := RunLimitedContext(ctx, root, 4096, "check-ref-format", "refs/heads/"+branch); err != nil {
		return "", "", fmt.Errorf("%s's HEAD names a branch whose name is not a valid ref", remote)
	}
	return branch, tip, nil
}
