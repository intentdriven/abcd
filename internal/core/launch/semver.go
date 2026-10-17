package launch

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

// Semver is a parsed strict SemVer 2.0.0 version.
type Semver struct {
	Major, Minor, Patch int
	Prerelease, Build   string
}

// semverRe is the strict SemVer 2.0.0 grammar (semver.org): MAJOR.MINOR.PATCH
// with no leading zeros and no leading v, plus optional -prerelease and +build.
// It is anchored with \A...\z (not ^...$) so a trailing newline is rejected —
// the exact bug the Python semver_grammar module guards against.
var semverRe = func() *regexp.Regexp {
	numID := `0|[1-9]\d*`
	alnumID := `\d*[A-Za-z-][0-9A-Za-z-]*`
	preID := `(?:` + numID + `|` + alnumID + `)`
	buildID := `[0-9A-Za-z-]+`
	return regexp.MustCompile(
		`\A(` + numID + `)\.(` + numID + `)\.(` + numID + `)` +
			`(?:-(` + preID + `(?:\.` + preID + `)*))?` +
			`(?:\+(` + buildID + `(?:\.` + buildID + `)*))?\z`,
	)
}()

// IsStrictSemver reports whether value is a strict SemVer 2.0.0 string.
func IsStrictSemver(value string) bool {
	return semverRe.MatchString(value)
}

// ParseSemver parses a strict SemVer string into its components. It returns an
// error for any non-conforming input (the boundary validator).
func ParseSemver(value string) (Semver, error) {
	m := semverRe.FindStringSubmatch(value)
	if m == nil {
		return Semver{}, fmt.Errorf("not a strict SemVer 2.0.0 version (semver.org, no leading 'v'): %q", value)
	}
	// A component wider than int overflows to MaxInt64 on the discarded error, so
	// two distinct oversized versions collide to one String() and the next Patch++
	// wraps negative — the same defect guarded in spec/spec.go:specNum. A version
	// this tool cannot represent is not a version; fail the parse.
	major, err := strconv.Atoi(m[1])
	if err != nil {
		return Semver{}, fmt.Errorf("semver component out of range: %q: %w", value, err)
	}
	minor, err := strconv.Atoi(m[2])
	if err != nil {
		return Semver{}, fmt.Errorf("semver component out of range: %q: %w", value, err)
	}
	patch, err := strconv.Atoi(m[3])
	if err != nil {
		return Semver{}, fmt.Errorf("semver component out of range: %q: %w", value, err)
	}
	return Semver{Major: major, Minor: minor, Patch: patch, Prerelease: m[4], Build: m[5]}, nil
}

// releaseTagRe is a release tag: exactly vMAJOR.MINOR.PATCH, ASCII digits
// only, no leading zero, no pre-release and no build part, each component at
// most nine digits. The bound keeps every component inside int32 on any
// platform, far above any release this repository will cut, and caps the
// whole tag at 30 bytes, so a value read from a tracked file cannot carry
// anything past the version it claims to be.
var releaseTagRe = regexp.MustCompile(`\Av(0|[1-9][0-9]{0,8})\.(0|[1-9][0-9]{0,8})\.(0|[1-9][0-9]{0,8})\z`)

// errNotReleaseTag is the refusal ParseReleaseTag returns. It does not echo
// the value: the value is often read from a repository's tracked files, and a
// caller relaying the error must not relay it.
var errNotReleaseTag = errors.New("not a release tag of the form vMAJOR.MINOR.PATCH (digits only, no leading zero)")

// ParseReleaseTag parses a release tag, the one strict form every release
// version abcd reads must take (spc-2610170635097509): the running binary's
// core.Version, a repository's recorded setup version, the binary record's
// previous tag, a digest's tags. Anything else, a dev build or a pre-release
// included, is refused and takes no part in a comparison. Order the result
// with CoreGreater.
func ParseReleaseTag(tag string) (Semver, error) {
	m := releaseTagRe.FindStringSubmatch(tag)
	if m == nil {
		return Semver{}, errNotReleaseTag
	}
	var n [3]int
	for i := range n {
		v, err := strconv.Atoi(m[i+1])
		if err != nil {
			return Semver{}, errNotReleaseTag
		}
		n[i] = v
	}
	return Semver{Major: n[0], Minor: n[1], Patch: n[2]}, nil
}

// BumpTier names the SemVer component that moved from prev to next: "major",
// "minor", "patch", or "" when the two cores are equal.
//
// It reads the actual version delta rather than mapping an impact, because the
// mapping is not fixed: pre-1.0, changelog.DeriveNext turns a breaking impact
// into a MINOR bump and an additive one into a PATCH. A tier derived from the
// impact would therefore mislabel every pre-1.0 release in the published
// manifest, while the delta is true at any point on the 0.x/1.x boundary.
func BumpTier(prev, next Semver) string {
	switch {
	case next.Major != prev.Major:
		return "major"
	case next.Minor != prev.Minor:
		return "minor"
	case next.Patch != prev.Patch:
		return "patch"
	}
	return ""
}

// Line is the MAJOR.MINOR retention line a version belongs to.
func (s Semver) Line() string { return fmt.Sprintf("%d.%d", s.Major, s.Minor) }

// String renders the core MAJOR.MINOR.PATCH (retention compares core only).
func (s Semver) String() string { return fmt.Sprintf("%d.%d.%d", s.Major, s.Minor, s.Patch) }

// Tag renders the v-prefixed git tag for the version core.
func (s Semver) Tag() string { return "v" + s.String() }

// coreLess orders two versions by (Major, Minor, Patch); prerelease/build are
// ignored (releases are core versions).
func coreLess(a, b Semver) bool {
	if a.Major != b.Major {
		return a.Major < b.Major
	}
	if a.Minor != b.Minor {
		return a.Minor < b.Minor
	}
	return a.Patch < b.Patch
}

// CoreGreater reports whether a is strictly newer than b by core version. It is
// exported because release ordering is decided in more than one place — the
// retention plan here and the tag anchor the changelog derivation resolves — and
// a second hand-rolled field comparison would be a second, silently divergent
// definition of "newer".
func CoreGreater(a, b Semver) bool { return coreLess(b, a) }
