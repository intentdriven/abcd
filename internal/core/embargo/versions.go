package embargo

import (
	"errors"
	"regexp"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/core/launch"
)

// patchedVersionKeepsV pins the version form an advisory's patched version
// and affected range take: the release tag without its leading v. It is the
// form this repository's own advisories carry: the version bodies the
// 2026-10-09 hand-run sent for the advisories it published name the patched
// release as MAJOR.MINOR.PATCH with no v, and the live run of 2026-10-12 read
// this repository's published advisories back through the API (state
// published) and found every patched version and every range bound in that
// form, with no v. Changing the form is this one constant.
const patchedVersionKeepsV = false

// releaseTagRe is a release tag: strict vMAJOR.MINOR.PATCH, with no leading
// zero, no pre-release and no build part.
var releaseTagRe = regexp.MustCompile(`\Av(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\z`)

// isReleaseTag reports whether tag is a release tag.
func isReleaseTag(tag string) bool { return releaseTagRe.MatchString(tag) }

// sortReleaseTags returns the release tags among tags, lowest version first.
func sortReleaseTags(tags []string) []string {
	type tv struct {
		tag string
		v   launch.Semver
	}
	var vs []tv
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if !isReleaseTag(t) {
			continue
		}
		v, err := launch.ParseSemver(strings.TrimPrefix(t, "v"))
		if err != nil {
			continue
		}
		vs = append(vs, tv{t, v})
	}
	slices.SortFunc(vs, func(a, b tv) int { return compareCore(a.v, b.v) })
	out := make([]string, 0, len(vs))
	for _, x := range vs {
		out = append(out, x.tag)
	}
	return out
}

func compareCore(a, b launch.Semver) int {
	switch {
	case a.Major != b.Major:
		return a.Major - b.Major
	case a.Minor != b.Minor:
		return a.Minor - b.Minor
	}
	return a.Patch - b.Patch
}

// patchedVersion is the patched version a release tag gives, in the pinned form.
func patchedVersion(tag string) string {
	if patchedVersionKeepsV {
		return tag
	}
	return strings.TrimPrefix(tag, "v")
}

// rangeTermRe is one term of GitHub's version range form: an operator, one
// space, and a version of the characters versions are spelled with.
var rangeTermRe = regexp.MustCompile(`\A(=|<|<=|>|>=) ([0-9A-Za-z][0-9A-Za-z.+_-]*)\z`)

// validRange reports whether r is in GitHub's version range form: one or
// more terms, each an operator (=, <, <=, >, >=), a space and a version,
// separated by a comma and a space.
func validRange(r string) bool {
	if r == "" {
		return false
	}
	for _, term := range strings.Split(r, ", ") {
		if !rangeTermRe.MatchString(term) {
			return false
		}
	}
	return true
}

// affectedRange is the affected range a patched version gives: below the
// patched version, keeping a lower bound (> or >=) the advisory already
// carries. An existing range not in GitHub's form is refused rather than
// overwritten, and so is a lower bound that is not below the patched
// version, which would leave no version affected.
func affectedRange(existing *string, patched string) (string, error) {
	var terms []string
	if existing != nil && strings.TrimSpace(*existing) != "" {
		if !validRange(*existing) {
			return "", errors.New("the affected range on GitHub is not in GitHub's range form; correct it there, then run it again")
		}
		pv, perr := launch.ParseSemver(strings.TrimPrefix(patched, "v"))
		for _, term := range strings.Split(*existing, ", ") {
			m := rangeTermRe.FindStringSubmatch(term)
			if m[1] != ">" && m[1] != ">=" {
				continue
			}
			if lv, err := launch.ParseSemver(strings.TrimPrefix(m[2], "v")); err == nil && perr == nil && compareSemver(lv, pv) >= 0 {
				return "", errors.New("the affected range on GitHub has a lower bound that is not below the patched version; correct it there, then run it again")
			}
			terms = append(terms, term)
		}
	}
	terms = append(terms, "< "+patched)
	r := strings.Join(terms, ", ")
	if !validRange(r) {
		return "", errors.New("the computed affected range is not in GitHub's range form")
	}
	return r, nil
}

// compareSemver orders two versions by their core, then a pre-release below
// its release.
func compareSemver(a, b launch.Semver) int {
	if c := compareCore(a, b); c != 0 {
		return c
	}
	switch {
	case a.Prerelease == b.Prerelease:
		return 0
	case a.Prerelease == "":
		return 1
	case b.Prerelease == "":
		return -1
	}
	return strings.Compare(a.Prerelease, b.Prerelease)
}
