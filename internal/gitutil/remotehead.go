package gitutil

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// RefLineMax bounds one line of a ref listing as it is read (NetEach and its
// kin): far past any ref name a forge accepts, so a line naming the ref asked
// for is never cut. A line that reaches it may have been cut, and RefListing
// reads it as naming no ref it asked for.
const RefLineMax = 64 << 10

// RefListing is what a `git ls-remote [--symref] <remote> <ref>` listing says
// of the ref Ref itself, the one reader every such listing abcd asks for is
// read through. git matches the pattern against the tail of every ref name,
// so HEAD also lists refs/heads/x/HEAD, and refs/heads/main also lists
// refs/heads/x/refs/heads/main: a listing can run to any length a pusher
// chooses, and its first line need not be Ref's. So a listing is read one
// line at a time (Line, handed to NetEach, NetEachContext or
// RunNetEachContext with RefLineMax), never whole under a bound and never by
// its first line, and only the lines whose ref is exactly Ref are kept.
type RefListing struct {
	// Ref is the ref name asked for, exactly as git lists it (HEAD,
	// refs/heads/main).
	Ref string
	// Symrefs counts the `ref: <target>\t<Ref>` lines (--symref), and Target
	// is the first one's target.
	Symrefs int
	Target  string
	// Tips counts the `<object>\t<Ref>` lines, and Tip is the first one's
	// object name, as listed (a caller checks its shape).
	Tips int
	Tip  string
}

// Line reads one line of the listing, without its newline.
func (l *RefListing) Line(line string) {
	if len(line) >= RefLineMax {
		return
	}
	line = strings.TrimSuffix(line, "\r")
	if rest, ok := strings.CutPrefix(line, "ref: "); ok {
		if target, name, ok := strings.Cut(rest, "\t"); ok && name == l.Ref {
			if l.Symrefs == 0 {
				l.Target = target
			}
			l.Symrefs++
		}
		return
	}
	if sha, name, ok := strings.Cut(line, "\t"); ok && name == l.Ref {
		if l.Tips == 0 {
			l.Tip = sha
		}
		l.Tips++
	}
}

// Read reads a listing a caller's runner answered whole (a seam's answer).
func (l *RefListing) Read(out string) {
	for _, line := range strings.Split(out, "\n") {
		l.Line(line)
	}
}

// RemoteHeadBranch asks the remote which branch is its default and returns
// the bare branch name: `git ls-remote --symref <remote> HEAD`, read for the
// `ref: refs/heads/<branch>\tHEAD` line the remote advertises, the line whose
// ref is HEAD itself (RefListing). git, the caller's networked git runner,
// asks; nil reads the listing line by line through NetEach (a network call,
// its environment scrubbed and its time bounded). A caller with a seam for its
// networked git (abcd land's Options.Git) passes it, so the call is the
// caller's to record and its answer the caller's to fake; a runner that
// refuses the listing as past its bound (Net's OutputCapError: the remote
// holds more branches whose names end in /HEAD than fit) has it read again
// line by line through NetEach, Net's environment.
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
	args := []string{"ls-remote", "--symref", remote, "HEAD"}
	l := RefListing{Ref: "HEAD"}
	var err error
	if git != nil {
		var out string
		if out, err = git(root, args...); err == nil {
			l.Read(out)
		}
	}
	var capped *OutputCapError
	if git == nil || errors.As(err, &capped) {
		l = RefListing{Ref: "HEAD"}
		err = NetEach(root, RefLineMax, l.Line, args...)
	}
	if err != nil {
		return "", fmt.Errorf("asking %s for its default branch: %w", remote, err)
	}
	return remoteHeadOf(root, remote, l)
}

// remoteHeadOf is the default branch l names, checked locally.
func remoteHeadOf(root, remote string, l RefListing) (string, error) {
	if l.Symrefs == 0 {
		return "", fmt.Errorf("%s did not name a default branch (its HEAD is detached, or it advertises no symref)", remote)
	}
	branch, ok := strings.CutPrefix(l.Target, "refs/heads/")
	if !ok || branch == "" || branch == "HEAD" {
		return "", fmt.Errorf("%s's HEAD does not name a branch", remote)
	}
	if _, err := LocalGit(root, "check-ref-format", "refs/heads/"+branch); err != nil {
		return "", fmt.Errorf("%s's HEAD names a branch whose name is not a valid ref", remote)
	}
	return branch, nil
}

// RemoteHeadTip asks the remote which branch is its default and which commit
// that branch's tip is: `git ls-remote --symref <remote> HEAD`, read for the
// lines whose ref is HEAD itself (RefListing). git, the caller's runner,
// asks; nil reads the listing line by line through NetEachContext (the
// person's own configuration and credentials, none of the checkout's hooks,
// bounded by ctx as well as NetTimeout), so however many branches whose
// names end in /HEAD the remote holds, none is kept and none crowds out the
// answer. A runner that refuses the listing as past its bound (NetContext's
// OutputCapError) has it read again that way. It never fetches and never
// writes.
//
// The answer is the remote's, so it is read as untrusted: it must carry
// exactly one symref line for HEAD, naming a branch under refs/heads/ that
// `git check-ref-format` accepts (asked under ctx), and exactly one object
// line for HEAD, a full object name. Anything else is an error, never a guess.
func RemoteHeadTip(ctx context.Context, git func(ctx context.Context, dir string, args ...string) (string, error), root, remote string) (branch, tip string, err error) {
	if !RefIsSafe(remote) {
		return "", "", errors.New("the remote's name is empty or option-shaped")
	}
	args := []string{"ls-remote", "--symref", remote, "HEAD"}
	l := RefListing{Ref: "HEAD"}
	if git != nil {
		var out string
		if out, err = git(ctx, root, args...); err == nil {
			l.Read(out)
		}
	}
	var capped *OutputCapError
	if git == nil || errors.As(err, &capped) {
		l = RefListing{Ref: "HEAD"}
		err = NetEachContext(ctx, root, RefLineMax, l.Line, args...)
	}
	if err != nil {
		return "", "", fmt.Errorf("asking %s for its default branch: %w", remote, err)
	}
	if l.Symrefs != 1 || l.Tips != 1 {
		return "", "", fmt.Errorf("%s did not name one default branch and one tip for it (%d symref and %d object lines for HEAD)", remote, l.Symrefs, l.Tips)
	}
	branch, ok := strings.CutPrefix(l.Target, "refs/heads/")
	if !ok || branch == "" || branch == "HEAD" {
		return "", "", fmt.Errorf("%s's HEAD does not name a branch", remote)
	}
	if tip = l.Tip; !IsFullSHA(tip) {
		return "", "", fmt.Errorf("%s's HEAD tip is not a full object name", remote)
	}
	if _, err := RunLimitedContext(ctx, root, 4096, "check-ref-format", "refs/heads/"+branch); err != nil {
		return "", "", fmt.Errorf("%s's HEAD names a branch whose name is not a valid ref", remote)
	}
	return branch, tip, nil
}
