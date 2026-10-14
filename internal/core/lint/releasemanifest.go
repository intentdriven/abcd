package lint

// releasemanifest.go — the release gate's refusals of a tree that lost its
// release-gate manifest (iss-130) or set it below the schemaVersion that arms
// the closed disposition vocabulary. The manifest's presence is the era marker
// (iss-122): a tree without one is judged by the pre-manifest rules, which is
// right for the releases that predate it and a silent disarm for every later
// one. The release's arming alone (RuleConfig.release) reads the released
// version to tell the two apart; CheckGateReceipt, the reader the doc-fidelity
// gate shares, keeps the era marker as it was.

import (
	"os"
	"strconv"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// lastPreManifestRelease is the newest release whose tree predates the
// release-gate manifest (iss-122): its receipts, and every older one, are
// judged by the pre-manifest rules.
var lastPreManifestRelease = launch.Semver{Major: 0, Minor: 4, Patch: 0}

// missingManifestRefusal is the release gate's verdict on a tree with no
// release-gate manifest: "" when the release is v0.4.0 or older, which
// predates the manifest and keeps the pre-manifest rules, and a refusal
// otherwise. Without it, deleting the manifest reverted a later release to the
// pre-manifest rules and disarmed every manifest-era refusal (iss-130). A tree
// that names no release cannot be shown to predate the manifest, so it fails
// closed.
func missingManifestRefusal(root *os.Root) string {
	const head = "receipt_gate finds no release-gate manifest " + releaseGateManifestPath
	const tail = "; a release after v0.4.0 carries the manifest, and without it the gate would fall back to the pre-manifest rules (iss-130), so the release gate fails closed"
	v, why := releasedVersionInTree(root)
	if why != "" {
		return head + " and " + why + tail
	}
	if !launch.CoreGreater(v, lastPreManifestRelease) {
		return ""
	}
	return head + " in the tree of release v" + v.String() + tail
}

// lastFreeTextRelease is the newest release whose receipts were written to the
// free-text dispositions: the closed vocabulary arms at manifest
// schemaVersion 2 for every release after it.
var lastFreeTextRelease = launch.Semver{Major: 0, Minor: 14, Patch: 1}

// schemaDowngradeRefusal is the release gate's verdict on a manifest below
// schemaVersion 2: "" when the release is v0.14.1 or older, whose receipts
// keep the free-text dispositions, and a refusal otherwise. Only a test held
// the committed manifest at schemaVersion 2, so a release branch could set it
// back, re-hash its receipts and turn the closed vocabulary off for every
// later release. A tree that names no release fails closed.
func schemaDowngradeRefusal(root *os.Root, version int) string {
	head := "receipt_gate finds the release-gate manifest " + releaseGateManifestPath + " at schemaVersion " + strconv.Itoa(version)
	const tail = "; a release after v0.14.1 is judged by the closed disposition vocabulary, which arms at schemaVersion 2, so the release gate fails closed"
	v, why := releasedVersionInTree(root)
	if why != "" {
		return head + " and " + why + tail
	}
	if !launch.CoreGreater(v, lastFreeTextRelease) {
		return ""
	}
	return head + " in the tree of release v" + v.String() + tail
}

// releasedVersionInTree reads the version the gated tree releases from its
// CHANGELOG.md, by the strict reader the release job binds its receipts with,
// guarded on the gate's own os.Root. When it cannot, it returns why, phrased
// to follow "and".
func releasedVersionInTree(root *os.Root) (launch.Semver, string) {
	data, err := fsutil.ReadGuardedInRoot(root, "CHANGELOG.md", changelog.MaxChangelogBytes)
	if err != nil {
		return launch.Semver{}, "cannot read the release version from CHANGELOG.md (" + guardedReason(err) + ")"
	}
	v, found, err := changelog.ReleasedVersionIn(data)
	if err != nil {
		return launch.Semver{}, "cannot read the release version from CHANGELOG.md (" + err.Error() + ")"
	}
	if !found {
		return launch.Semver{}, "CHANGELOG.md names no dated release"
	}
	return v, ""
}
