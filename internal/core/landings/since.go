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
	"os/exec"
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

// ErrReleaseTagUnnamed is what resolve's refusal is (errors.Is) when the
// last release tag git describes names no tag the repository holds: git
// describes an annotated tag by the name stored inside it, which a ref renamed
// since no longer carries. As with no release tag, there is no range to list
// by default, and the person names one.
var ErrReleaseTagUnnamed = errors.New("the last release tag git describes names no tag this repository holds")

// releaseTagError is a described release tag no ref holds, its text naming
// the tag and the flag.
type releaseTagError struct{ msg string }

func (e *releaseTagError) Error() string        { return e.msg }
func (e *releaseTagError) Is(target error) bool { return target == ErrReleaseTagUnnamed }

// ErrBadSince is what every since that names neither a tag the repository
// holds nor a date is (errors.Is), and one naming a tag git says points at no
// commit, so the verb refuses it as a range it was not given. A since whose
// tag could not be read is a failed read, never this.
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

// tagRefFormat is the line git for-each-ref prints for each tag ref tagCommit
// asks about: its object's full name and the ref.
const tagRefFormat = "--format=%(objectname) %(refname)"

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
	if errors.Is(err, errNoCommit) {
		return Range{}, ErrNoDefaultBranch
	}
	if err != nil {
		return Range{}, err
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
		tag, err := lastReleaseTag(ctx, repoRoot, to)
		if err != nil {
			return Range{}, err
		}
		r.From, err = tagCommit(ctx, repoRoot, tag)
		shown := termsafe.Sanitize(boundText(tag, maxTextBytes))
		switch {
		case errors.Is(err, errNotTagName) || errors.Is(err, errNoTagRef):
			// git describes an annotated tag by the name stored inside it,
			// so a tag whose ref was renamed is described under a name no
			// ref holds: the person names the range, as with no release tag.
			return Range{}, &releaseTagError{fmt.Sprintf("the last release tag git describes, %s, names no tag this repository holds (an annotated tag whose ref was renamed is described by the name stored inside it), so there is no range to list by default; name where to start with --since <tag|YYYY-MM-DD>", shown)}
		case errors.Is(err, errNoCommit):
			// git describes only a tag a commit the tip reaches carries.
			return Range{}, readFailed("reading the last release tag "+shown, errors.New("git describes it, and says its ref points at no commit"))
		case err != nil:
			return Range{}, readFailed("reading the last release tag "+shown, err)
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
		shown := termsafe.Sanitize(boundText(since, maxTextBytes))
		switch {
		case errors.Is(err, errNotTagName) || errors.Is(err, errNoTagRef):
			return Range{}, &sinceError{fmt.Sprintf("--since %s is neither a tag this repository holds nor a date (YYYY-MM-DD)", shown)}
		case errors.Is(err, errNoCommit):
			return Range{}, &sinceError{fmt.Sprintf("--since %s names a tag that points at no commit, so no range opens there", shown)}
		case err != nil:
			return Range{}, readFailed("reading the tag --since names, "+shown, err)
		}
		r.From, r.Since = from, since
	}
	if !slices.ContainsFunc(line, func(c lineCommit) bool { return c.sha == r.From }) {
		r.Note = fmt.Sprintf("the tag %s is not on %s's first-parent line, so the range is every commit %s reaches that the tag does not: it may hold rows older than the tag, and a merge's row may hold only part of its branch",
			termsafe.Sanitize(boundText(r.Since, maxTextBytes)), r.Tip, r.Tip)
	}
	return r, nil
}

// lastReleaseTag is the last release tag reachable from to (git describe
// --tags --match 'v*' --abbrev=0, as stored), ErrNoReleaseTag only when git
// says none is: describe fails both for no tag and for a failed read, so a
// failure is followed by a listing of the release tags to reaches (git tag
// --list --merged, which matches the pattern as describe does), and only an
// empty listing reads as no tag. A listing that fails, or names a tag
// describe did not give, is a failed read of the tags, never "no release
// tag". The name is used as git printed it, the line end alone taken off: a
// tag name may end in a rune Unicode calls a space.
func lastReleaseTag(ctx context.Context, repoRoot, to string) (string, error) {
	out, err := gitutil.RunCappedBytesAsStoredContext(ctx, repoRoot, "", refCap, "describe", "--tags", "--match", ReleaseTagPattern, "--abbrev=0", to)
	tag := strings.TrimSuffix(string(out), "\n")
	if err == nil && strings.ContainsAny(tag, "\r\n") {
		return "", errors.New("reading the last release tag: git described the tip in a shape it does not read")
	}
	if err == nil && tag != "" {
		return tag, nil
	}
	describeErr := err
	listed, err := storedText(ctx, repoRoot, maxListBytes, "tag", "--list", "--merged", to, ReleaseTagPattern)
	switch {
	case err != nil:
		return "", masked("reading the release tags", err)
	case strings.TrimSpace(listed) == "":
		return "", ErrNoReleaseTag
	case describeErr != nil:
		return "", masked("reading the last release tag", describeErr)
	}
	return "", errors.New("reading the last release tag: git described the tip with no tag, and lists release tags it reaches")
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
		if l == "" {
			continue
		}
		ts, sha, ok := strings.Cut(l, " ")
		t, err := strconv.ParseInt(ts, 10, 64)
		if !ok || err != nil || !gitutil.IsFullSHA(sha) {
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

// The answers commitOf and tagCommit give when git says the name is not what
// was asked: errNoCommit, git's verify of the peel saying no commit (its exit
// 1 under --quiet); errNotTagName, a name not in a tag ref's shape (empty,
// starting with a dash, or refused by git check-ref-format's exit 1); and
// errNoTagRef, no ref under refs/tags/ holding the name. Every other error
// either returns is a read that failed or an answer it does not read, and its
// text says so.
var (
	errNoCommit   = errors.New("git says it names no commit")
	errNotTagName = errors.New("not a tag name")
	errNoTagRef   = errors.New("no tag of that name")
)

// exitedWith reports whether err is git's own exit with code.
func exitedWith(err error, code int) bool {
	var ee *exec.ExitError
	return errors.As(err, &ee) && ee.ExitCode() == code
}

// readFailed is err, a read that failed, as resolve returns it: what was
// being read, then err's text, which tagCommit and commitOf mask.
func readFailed(what string, err error) error {
	return &maskedError{msg: what + ": " + err.Error(), cause: err}
}

// commitOf is the full object name of the commit ref names, read as stored
// (storedText), so no replace ref stands one tag in for another on the way to
// it; errNoCommit when git's verify says ref names none.
func commitOf(ctx context.Context, repoRoot, ref string) (string, error) {
	out, err := storedText(ctx, repoRoot, refCap, "rev-parse", "--verify", "--quiet", ref+"^{commit}", "--")
	switch {
	case exitedWith(err, 1):
		return "", errNoCommit
	case err != nil:
		return "", masked("reading the commit "+termsafe.Sanitize(boundText(ref, maxTextBytes))+" names", err)
	case !gitutil.IsFullSHA(out):
		return "", fmt.Errorf("reading the commit %s names: git named it in a shape it does not read", termsafe.Sanitize(boundText(ref, maxTextBytes)))
	}
	return out, nil
}

// tagCommit is the commit the tag name points at, the name held to a ref's
// shape under refs/tags/ first and then looked up as that exact ref among the
// refs git for-each-ref lists under it, so it is never read as anything else:
// rev-parse would take a name no ref holds that ends in -g and hex, as git
// describe prints, for the commit the hex names. The ref's object is then
// peeled by its full name. errNotTagName and errNoTagRef are git's own word
// that the name is no tag, and errNoCommit that the tag points at no commit;
// a check, a listing or a peel that failed, and a listing in a shape it does
// not read, are failed reads.
func tagCommit(ctx context.Context, repoRoot, name string) (string, error) {
	if name == "" || strings.HasPrefix(name, "-") {
		return "", errNotTagName
	}
	ref := "refs/tags/" + name
	switch _, err := storedText(ctx, repoRoot, refCap, "check-ref-format", ref); {
	case exitedWith(err, 1):
		return "", errNotTagName
	case err != nil:
		return "", masked("checking the tag name's shape", err)
	}
	// for-each-ref lists the ref and every ref below it as a directory; a line
	// is cut past maxLine, which only a ref below it reaches.
	var oid string
	odd := false
	err := gitutil.EachAsStoredContext(ctx, repoRoot, len(ref)+128, func(line string) {
		o, got, ok := strings.Cut(line, " ")
		switch {
		case !ok || !gitutil.IsFullSHA(o) || (got != ref && !strings.HasPrefix(got, ref+"/")):
			odd = true
		case got == ref && oid != "":
			odd = true
		case got == ref:
			oid = o
		}
	}, "for-each-ref", tagRefFormat, ref)
	switch {
	case err != nil:
		return "", masked("listing the tag's ref", err)
	case odd:
		return "", errors.New("listing the tag's ref: git listed it in a shape it does not read")
	case oid == "":
		return "", errNoTagRef
	}
	return commitOf(ctx, repoRoot, oid)
}
