package ahoy

import (
	"path"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// The newest-release mark (iss-2609151847117977): ~/.abcd.noindex/release-mark
// names the newest release this machine has installed or verified. The
// bootstrap raises it whenever it provisions a release it authenticated
// against the published manifest, and `abcd update` records it whenever it
// installs or verifies one. The cache attestation pins a binary hash, so it
// keeps vouching for a release after a newer one has been installed; the mark
// is what lets a promotion tell "the release this machine authenticated last"
// from "a release this machine authenticated once" (the monotonic version check
// TUF answers rollback with). A cache whose attested release is older than the
// mark is not promoted, and a latest release that has fallen back below it —
// a publisher withdrawal — is refused, naming the one command that lowers the
// mark on purpose: `abcd update <tag>` with the tag named.

// releaseMarkFile is the mark's name under ~/.abcd.noindex.
const releaseMarkFile = "release-mark"

// releaseMarkRel is the mark's home-relative path.
var releaseMarkRel = abcdhome.Rel(releaseMarkFile)

// LowerReleaseMarkCommand is the one command that lowers the mark on purpose,
// for a tag: naming the tag is the person's own choice of an older release.
func LowerReleaseMarkCommand(tag string) string { return "abcd update " + tag }

// parseReleaseTag parses a vMAJOR.MINOR.PATCH release tag.
func parseReleaseTag(tag string) (launch.Semver, bool) {
	if !ReleaseTagShape.MatchString(tag) {
		return launch.Semver{}, false
	}
	v, err := launch.ParseSemver(strings.TrimPrefix(tag, "v"))
	if err != nil {
		return launch.Semver{}, false
	}
	return v, true
}

// ReleaseOlder reports whether release tag a is older than tag b. Two tags of
// the same core are equal unless only a carries a pre-release suffix. A tag
// that does not parse is never older: the comparison refuses only on a fact.
func ReleaseOlder(a, b string) bool {
	va, okA := parseReleaseTag(a)
	vb, okB := parseReleaseTag(b)
	if !okA || !okB {
		return false
	}
	if launch.CoreGreater(vb, va) {
		return true
	}
	if launch.CoreGreater(va, vb) {
		return false
	}
	return va.Prerelease != "" && vb.Prerelease == ""
}

// ReadReleaseMark returns the recorded newest release, read through the same
// home-scoped declaration read as the cache attestation (owned by this uid,
// writable by no one else, a regular file). Absent, unreadable or malformed
// reads as no mark.
func ReadReleaseMark() (string, bool) {
	home, refused := homeScope()
	if refused != "" {
		return "", false
	}
	raw, _, err := fsutil.ReadHomeDeclaration(home, releaseMarkRel, maxPathEntryBytes)
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if k, v, ok := strings.Cut(strings.TrimSpace(line), "="); ok && k == "release_tag" {
			if _, ok := parseReleaseTag(v); ok {
				return v, true
			}
			return "", false
		}
	}
	return "", false
}

// RecordReleaseMark records tag as this machine's newest release. With lower
// false it only RAISES the mark (a tag at or below the current one changes
// nothing); with lower true — an explicitly named tag, the person's own
// choice — it records tag whatever the mark said. The write is atomic and
// mode 0600 under ~/.abcd.noindex, judged and opened relative to home's
// descriptor like path-entry.
func RecordReleaseMark(tag string, lower bool) error {
	if _, ok := parseReleaseTag(tag); !ok {
		return nil
	}
	if cur, ok := ReadReleaseMark(); ok && !lower && !ReleaseOlder(cur, tag) {
		return nil
	}
	home, err := homeScopeErr()
	if err != nil {
		return err
	}
	dir, err := fsutil.EnsureHomeScope(home, path.Dir(releaseMarkRel), abcdhome.DirMode)
	if err != nil {
		return err
	}
	defer dir.Close()
	return fsutil.WriteFileAtomicInRoot(dir, path.Base(releaseMarkRel), []byte("release_tag="+tag+"\n"), abcdhome.FileMode)
}
