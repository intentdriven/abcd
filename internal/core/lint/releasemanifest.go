package lint

// releasemanifest.go — the release gate's refusal of a tree that lost its
// release-gate manifest (iss-130). The manifest's presence is the era marker
// (iss-122): a tree without one is judged by the pre-manifest rules, which is
// right for the releases that predate it and a silent disarm for every later
// one. The release's arming alone (RuleConfig.release) reads the released
// version to tell the two apart; CheckGateReceipt, the reader the doc-fidelity
// gate shares, keeps the era marker as it was.

import (
	"os"

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
// pre-manifest rules and disarmed every manifest-era refusal (iss-130). The
// version is read from the gated tree's CHANGELOG.md by the strict reader the
// release job binds its receipts with; a tree that names no release cannot be
// shown to predate the manifest, so it fails closed.
func missingManifestRefusal(root *os.Root) string {
	const tail = "; a release after v0.4.0 carries the manifest, and without it the gate would fall back to the pre-manifest rules (iss-130), so the release gate fails closed"
	data, err := fsutil.ReadGuardedInRoot(root, "CHANGELOG.md", changelog.MaxChangelogBytes)
	if err != nil {
		return "receipt_gate finds no release-gate manifest " + releaseGateManifestPath + " and cannot read the release version from CHANGELOG.md (" + guardedReason(err) + ")" + tail
	}
	v, found, err := changelog.ReleasedVersionIn(data)
	if err != nil {
		return "receipt_gate finds no release-gate manifest " + releaseGateManifestPath + " and cannot read the release version from CHANGELOG.md (" + err.Error() + ")" + tail
	}
	if !found {
		return "receipt_gate finds no release-gate manifest " + releaseGateManifestPath + " and CHANGELOG.md names no dated release" + tail
	}
	if !launch.CoreGreater(v, lastPreManifestRelease) {
		return ""
	}
	return "receipt_gate finds no release-gate manifest " + releaseGateManifestPath + " in the tree of release v" + v.String() + tail
}
