package landings

// since.go resolves the Range a listing reads (spc-2610301950480609 approaches
// 4 and 8): To is the default branch's tip as last fetched (gitutil's
// DefaultRef, with no network), and From is a release tag, the commit before
// the first first-parent commit at or after a date, or by default the last
// release tag reachable from the tip.

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// ErrNoDefaultBranch is resolve's answer for a repository with no default
// branch to list.
var ErrNoDefaultBranch = errors.New("the repository has no default branch to list the landings of")

// ErrNoReleaseTag is resolve's answer when no since is named and no release
// tag is reachable from the default branch: there is no default range.
var ErrNoReleaseTag = errors.New("no release tag (v*) is reachable from the default branch, so there is no range to list by default; name where to start with --since <tag|YYYY-MM-DD>")

// ReleaseTagPattern is the tags `git describe` takes the default range from.
const ReleaseTagPattern = "v*"

// dateRe is the one date shape since takes.
var dateRe = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)

// refCap bounds one ref question's answer.
const refCap = 4096

// resolve is unexported until the verb, its front door, calls it, as list is.
//
// resolve is the Range from since to the default branch's tip as the
// repository at repoRoot last fetched it. since is a tag, a date YYYY-MM-DD
// (taken in UTC as the start of that day, From being the first commit on the
// first-parent line before it, so the range opens at the first first-parent
// commit at or after it), or "" for the last release tag reachable from the
// tip (git describe --tags --match 'v*' --abbrev=0). Every git question is
// bound to ctx.
func resolve(ctx context.Context, repoRoot, since string) (Range, error) {
	ref := gitutil.DefaultRefContext(ctx, repoRoot)
	if err := ctx.Err(); err != nil {
		return Range{}, err
	}
	if ref == "" {
		return Range{}, ErrNoDefaultBranch
	}
	to, err := commitOf(ctx, repoRoot, ref)
	if err != nil {
		return Range{}, ErrNoDefaultBranch
	}
	r := Range{To: to, Tip: gitutil.ShortRef(ref)}
	switch {
	case since == "":
		tag, err := gitutil.RunLimitedContext(ctx, repoRoot, refCap, "describe", "--tags", "--match", ReleaseTagPattern, "--abbrev=0", to)
		if err := ctx.Err(); err != nil {
			return Range{}, err
		}
		tag = strings.TrimSpace(tag)
		if err != nil || tag == "" {
			return Range{}, ErrNoReleaseTag
		}
		if r.From, err = tagCommit(ctx, repoRoot, tag); err != nil {
			return Range{}, ErrNoReleaseTag
		}
		r.Since = tag
	case dateRe.MatchString(since):
		day, err := time.ParseInLocation(time.DateOnly, since, time.UTC)
		if err != nil {
			return Range{}, fmt.Errorf("--since %s is not a date (YYYY-MM-DD)", since)
		}
		if r.From, err = beforeDay(ctx, repoRoot, to, day); err != nil {
			return Range{}, err
		}
		r.Since = since
	default:
		from, err := tagCommit(ctx, repoRoot, since)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Range{}, ctxErr
		}
		if err != nil {
			return Range{}, fmt.Errorf("--since %s is neither a tag this repository holds nor a date (YYYY-MM-DD)", termsafe.Sanitize(boundText(since, maxTextBytes)))
		}
		r.From, r.Since = from, since
	}
	return r, nil
}

// commitOf is the full object name of the commit ref names.
func commitOf(ctx context.Context, repoRoot, ref string) (string, error) {
	out, err := gitutil.RunLimitedContext(ctx, repoRoot, refCap, "rev-parse", "--verify", "--quiet", ref+"^{commit}", "--")
	out = strings.TrimSpace(out)
	if err != nil || !gitutil.IsFullSHA(out) {
		return "", fmt.Errorf("%s names no commit", ref)
	}
	return out, nil
}

// tagCommit is the commit the tag name points at, the name held to a ref's
// shape under refs/tags/ first, so it is never read as anything else.
func tagCommit(ctx context.Context, repoRoot, name string) (string, error) {
	if name == "" || strings.HasPrefix(name, "-") {
		return "", errors.New("not a tag name")
	}
	if _, err := gitutil.RunLimitedContext(ctx, repoRoot, refCap, "check-ref-format", "refs/tags/"+name); err != nil {
		return "", errors.New("not a tag name")
	}
	return commitOf(ctx, repoRoot, "refs/tags/"+name)
}

// beforeDay is the first commit on to's first-parent line committed before
// day: walking back from to, the first one older than the day closes the
// range. "" when every commit on the line is at or after it (the whole line),
// and to itself when to is older (an empty range).
func beforeDay(ctx context.Context, repoRoot, to string, day time.Time) (string, error) {
	out, err := gitutil.RunCappedContext(ctx, repoRoot, maxListBytes, "rev-list", "--first-parent", "--timestamp", to, "--")
	if err != nil {
		return "", fmt.Errorf("listing the default branch's first-parent line: %w", err)
	}
	cut := day.Unix()
	for line := range strings.SplitSeq(out, "\n") {
		ts, sha, ok := strings.Cut(strings.TrimSpace(line), " ")
		if !ok {
			continue
		}
		t, err := strconv.ParseInt(ts, 10, 64)
		if err != nil || !gitutil.IsFullSHA(sha) {
			return "", errors.New("git listed the default branch's first-parent line in a shape it does not read")
		}
		if t < cut {
			return sha, nil
		}
	}
	return "", nil
}
