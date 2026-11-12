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
	"slices"
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

// ErrBadSince is what every since that names neither a tag the repository
// holds nor a date is (errors.Is), so the verb refuses it as a range it was
// not given.
var ErrBadSince = errors.New("the range's start names neither a tag nor a date")

// sinceError is a refused since, its text saying which.
type sinceError struct{ msg string }

func (e *sinceError) Error() string        { return e.msg }
func (e *sinceError) Is(target error) bool { return target == ErrBadSince }

// ReleaseTagPattern is the tags `git describe` takes the default range from.
const ReleaseTagPattern = "v*"

// dateRe is the one date shape since takes.
var dateRe = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)

// refCap bounds one ref question's answer.
const refCap = 4096

// Resolve is the Range from since to the default branch's tip as the
// repository at repoRoot last fetched it, as abcd landings names it.
//
// since is a tag, a date YYYY-MM-DD (taken in UTC as the start of that day:
// the range opens at the first commit on the tip's first-parent line, oldest
// first, committed at or after it, so From is the commit before that one), or
// "" for the last release tag reachable from the tip (git describe --tags
// --match 'v*' --abbrev=0). A committer date is whatever the commit's maker
// set, so where a commit inside a date's range is dated before the day, the
// range's Note says so; where a tag is not on the tip's first-parent line, the
// Note says what that range then holds. Every git question is bound to ctx, or
// to listBound when ctx sets no deadline, and a ctx that ends first is the
// error, whatever the question it ended was.
func Resolve(ctx context.Context, repoRoot, since string) (Range, error) {
	ctx, cancel := bounded(ctx)
	defer cancel()
	r, err := resolveRange(ctx, repoRoot, since)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return Range{}, ctxErr
	}
	return r, err
}

// resolveRange is resolve's work, under the ctx resolve bounds.
func resolveRange(ctx context.Context, repoRoot, since string) (Range, error) {
	ref := gitutil.DefaultRefContext(ctx, repoRoot)
	if ref == "" {
		return Range{}, ErrNoDefaultBranch
	}
	to, err := commitOf(ctx, repoRoot, ref)
	if err != nil {
		return Range{}, ErrNoDefaultBranch
	}
	r := Range{To: to, Tip: gitutil.ShortRef(ref)}
	line, err := firstParentLine(ctx, repoRoot, to)
	if err != nil {
		return Range{}, err
	}
	switch {
	case since == "":
		// As stored, so no replace ref or graft moves which tags the tip
		// reaches.
		tag, err := storedText(ctx, repoRoot, refCap, "describe", "--tags", "--match", ReleaseTagPattern, "--abbrev=0", to)
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
			return Range{}, &sinceError{fmt.Sprintf("--since %s is not a date (YYYY-MM-DD)", since)}
		}
		r.From, r.Note = sinceDay(line, day)
		r.Since = since
		return r, nil
	default:
		from, err := tagCommit(ctx, repoRoot, since)
		if err != nil {
			return Range{}, &sinceError{fmt.Sprintf("--since %s is neither a tag this repository holds nor a date (YYYY-MM-DD)", termsafe.Sanitize(boundText(since, maxTextBytes)))}
		}
		r.From, r.Since = from, since
	}
	if !slices.ContainsFunc(line, func(c lineCommit) bool { return c.sha == r.From }) {
		r.Note = fmt.Sprintf("the tag %s is not on %s's first-parent line, so the range is every commit %s reaches that the tag does not: it may hold rows older than the tag, and a merge's row may hold only part of its branch",
			termsafe.Sanitize(boundText(r.Since, maxTextBytes)), r.Tip, r.Tip)
	}
	return r, nil
}

// lineCommit is one commit on the first-parent line and its committer date.
type lineCommit struct {
	sha string
	at  int64
}

// firstParentLine is to's first-parent line, newest first, with each commit's
// committer date.
func firstParentLine(ctx context.Context, repoRoot, to string) ([]lineCommit, error) {
	out, err := storedText(ctx, repoRoot, maxListBytes, "rev-list", "--first-parent", "--timestamp", to, "--")
	if err != nil {
		return nil, masked("listing the default branch's first-parent line", err)
	}
	var line []lineCommit
	for l := range strings.SplitSeq(out, "\n") {
		ts, sha, ok := strings.Cut(strings.TrimSpace(l), " ")
		if !ok {
			continue
		}
		t, err := strconv.ParseInt(ts, 10, 64)
		if err != nil || !gitutil.IsFullSHA(sha) {
			return nil, errors.New("git listed the default branch's first-parent line in a shape it does not read")
		}
		line = append(line, lineCommit{sha, t})
	}
	return line, nil
}

// sinceDay is the From of the range that opens at day on line (newest first):
// walking the line oldest first, the first commit at or after the day opens
// the range, so From is the commit before it, "" when that is the line's first
// commit, and line's tip itself when no commit is at or after the day (an
// empty range). note counts the commits inside the range dated before the day,
// "" for none.
func sinceDay(line []lineCommit, day time.Time) (from, note string) {
	if len(line) == 0 {
		return "", ""
	}
	cut := day.Unix()
	start := -1
	for i := len(line) - 1; i >= 0; i-- {
		if line[i].at >= cut {
			start = i
			break
		}
	}
	if start < 0 {
		return line[0].sha, ""
	}
	if start+1 < len(line) {
		from = line[start+1].sha
	}
	older := 0
	for _, c := range line[:start] {
		if c.at < cut {
			older++
		}
	}
	if older > 0 {
		note = fmt.Sprintf("%d of the range's first-parent commits are dated before %s, after the first one dated on or after it: committer dates on the line are not in order, so the range opens at the oldest commit dated on or after the day", older, day.Format(time.DateOnly))
	}
	return from, note
}

// commitOf is the full object name of the commit ref names, read as stored
// (storedText), so no replace ref stands one tag in for another on the way to
// it.
func commitOf(ctx context.Context, repoRoot, ref string) (string, error) {
	out, err := storedText(ctx, repoRoot, refCap, "rev-parse", "--verify", "--quiet", ref+"^{commit}", "--")
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
	if _, err := storedText(ctx, repoRoot, refCap, "check-ref-format", "refs/tags/"+name); err != nil {
		return "", errors.New("not a tag name")
	}
	return commitOf(ctx, repoRoot, "refs/tags/"+name)
}
