package changelog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// MaxChangelogBytes caps the guarded CHANGELOG read. The file is a few hundred
// kilobytes of prose at most; the cap exists so a symlinked or replaced
// CHANGELOG.md cannot stream an unbounded file into a preview command. It is
// exported because the ship verb that WRITES the file reads it first, and one
// ceiling read two different ways is a ceiling that eventually disagrees.
const MaxChangelogBytes = 4 << 20

// datedHeadingRe matches a dated release heading in CHANGELOG.md, and it is a
// CONTRACT, not a convenience: .github/workflows/auto-release.yml greps
// `^## \[v?[0-9]+\.[0-9]+\.[0-9]+\] - ` for the heading it turns into a git tag.
// Reading a different set of headings here would let derivation reason about a
// release the tagger never sees. The only difference from the workflow's pattern
// is the capture group (which changes nothing about what matches); a test pins
// both the string and their agreement over fixtures. The trailing " - " is
// load-bearing: it is what skips the undated "## [Unreleased]" heading.
var datedHeadingRe = regexp.MustCompile(`^## \[v?([0-9]+\.[0-9]+\.[0-9]+)\] - `)

// IsDatedHeading reports whether a line is a dated release heading — the same
// predicate this package reads the CHANGELOG with, exposed so the ship verb that
// WRITES a heading can assert its output against the reader's rule rather than
// against a second copy of the pattern.
func IsDatedHeading(line string) bool {
	return datedHeadingRe.MatchString(strings.TrimRight(line, "\r"))
}

// LatestReleaseTag resolves the base a release is derived against: the newest
// `vX.Y.Z` git tag under root, by SemVer order.
//
// The tag — not the newest CHANGELOG heading — is the anchor, and the
// distinction is load-bearing. auto-release.yml creates the tag AFTER the ship
// PR merges, so between the merge and the tag the heading is one release ahead
// of the tag and the two describe different releases; deriving from the heading
// in that window would compute the cut against a base that does not exist yet.
// The tag is the only immutable anchor.
//
// Tag listing and strictness are delegated to launch.GitExistingTags, the
// repo's one tag reader: it keeps only strict `v`-prefixed SemVer release cores,
// dropping prereleases (whose core would be a phantom release) and anything
// else. Ordering is launch.CoreGreater, the repo's one version comparison, so
// v0.10.0 wins over v0.9.0 where a lexical maximum would not.
//
// A repo with no release tag returns found=false and no error: an absent anchor
// is a state the caller decides about, not a failure to read git.
func LatestReleaseTag(root string) (launch.Semver, bool, error) {
	tags, err := launch.GitExistingTags(root)
	if err != nil {
		return launch.Semver{}, false, err
	}
	var newest launch.Semver
	found := false
	for _, tag := range tags {
		if !found || launch.CoreGreater(tag, newest) {
			newest, found = tag, true
		}
	}
	return newest, found, nil
}

// DatedRelease is one dated release heading: the version it names and the date
// it carries, exactly as written.
type DatedRelease struct {
	Version string `json:"version"`
	Date    string `json:"date"`
}

// DatedReleases returns every dated heading in root's CHANGELOG.md, in file
// order — newest first, the record's own Keep-a-Changelog convention.
//
// It reads the headings through datedHeadingRe, the same predicate
// LatestChangelogVersion and the tagger use, and takes the date as the rest of
// the line after that match rather than through a second pattern: two regexps
// over one heading is two definitions of what a release is, and they drift.
//
// An absent CHANGELOG.md returns found=false and no error. A repository that
// records no releases is a state — the site omits the sections that depend on
// them and renders the rest.
func DatedReleases(root string) ([]DatedRelease, bool, error) {
	data, err := fsutil.ReadGuarded(filepath.Join(root, "CHANGELOG.md"), MaxChangelogBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var out []DatedRelease
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		loc := datedHeadingRe.FindStringSubmatchIndex(line)
		if loc == nil {
			continue
		}
		out = append(out, DatedRelease{
			Version: line[loc[2]:loc[3]],
			Date:    strings.TrimSpace(line[loc[1]:]),
		})
	}
	return out, true, nil
}

// LatestChangelogVersion returns the version of the newest dated heading in
// root's CHANGELOG.md — the heading auto-release.yml would tag on the next push
// to main.
//
// "Newest" is POSITIONAL, not the SemVer maximum: like the workflow's `grep
// -m1`, the first dated heading in the file wins, because Keep-a-Changelog order
// (newest first) is the record's own convention and mirroring the tagger exactly
// matters more here than being independently clever about ordering.
//
// The read is guarded and READ-ONLY. Nothing in this package writes to
// CHANGELOG.md: the whole point of derived releases is that changelog lines are
// generated from records at the ship, never edited in place by a preview.
//
// An absent CHANGELOG.md returns found=false and no error — the file is not this
// package's to require; the ship verb that writes the heading is.
func LatestChangelogVersion(root string) (launch.Semver, bool, error) {
	data, err := fsutil.ReadGuarded(filepath.Join(root, "CHANGELOG.md"), MaxChangelogBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return launch.Semver{}, false, nil
		}
		return launch.Semver{}, false, err
	}
	return LatestVersionIn(data)
}

// LatestVersionIn is LatestChangelogVersion over CHANGELOG bytes the caller
// already holds — a blob read out of a commit rather than the working tree,
// which is how the release gate compares the version a receipt's commit carries
// with the version being released.
func LatestVersionIn(data []byte) (launch.Semver, bool, error) {
	for _, line := range strings.Split(string(data), "\n") {
		m := datedHeadingRe.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil {
			continue
		}
		v, err := launch.ParseSemver(m[1])
		if err != nil {
			// The regexp already constrains the shape, so this is only
			// reachable for a leading-zero component ("## [0.04.0] - …") that
			// strict SemVer rejects. Naming it beats deriving against a version
			// the tagger and this package would read differently.
			return launch.Semver{}, false, err
		}
		return v, true, nil
	}
	return launch.Semver{}, false, nil
}

// releaseHeadingPrefix opens every Keep-a-Changelog version heading, dated or
// not, and unreleasedHeading is the one such heading that names no release.
const (
	releaseHeadingPrefix = "## ["
	unreleasedHeading    = "## [Unreleased]"
)

// ErrUnreadableReleaseHeading is returned by ReleasedVersionIn when the newest
// release heading is one the dated-heading reader does not parse.
var ErrUnreadableReleaseHeading = errors.New("the newest CHANGELOG release heading is not a dated vX.Y.Z heading")

// ReleasedVersionIn is the strict reading of the version a released tree names,
// for a caller that BINDS something to that version rather than merely reports
// it. LatestVersionIn skips every heading datedHeadingRe does not match, which is
// right for a preview but wrong for a binding: a pre-release head ("## [1.0.0-rc.1]
// - …"), a build-metadata head or an undated version head would be skipped, and
// the reader would answer with the PREVIOUS release's version.
//
// So this reader takes the newest "## [" heading other than "## [Unreleased]"
// and requires it to be a dated heading LatestVersionIn parses; anything else is
// ErrUnreadableReleaseHeading, naming the line. found=false means the file names
// no release heading at all, which the caller decides about.
func ReleasedVersionIn(data []byte) (launch.Semver, bool, error) {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		if !strings.HasPrefix(line, releaseHeadingPrefix) || strings.HasPrefix(line, unreleasedHeading) {
			continue
		}
		if !datedHeadingRe.MatchString(line) {
			return launch.Semver{}, false, fmt.Errorf("%w: %q", ErrUnreadableReleaseHeading, line)
		}
		return LatestVersionIn([]byte(line))
	}
	return launch.Semver{}, false, nil
}
