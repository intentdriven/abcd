package lint

// releasemanifest.go — the release gate's refusals of a tree that lost its
// release-gate manifest (iss-130) or set it below the schemaVersion that arms
// the closed disposition vocabulary. The manifest's presence is the era marker
// (iss-122): a tree without one is judged by the pre-manifest rules, which is
// right for the releases that predate it and a silent disarm for every later
// one. The release's arming alone (RuleConfig.release) reads the release tags
// to tell the two apart; CheckGateReceipt, the reader the doc-fidelity
// gate shares, keeps the era marker as it was.

import (
	"os"
	"sort"
	"strconv"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// missingManifestRefusal is the release gate's verdict on a tree with no
// release-gate manifest. Without it, deleting the manifest reverted a release
// to the pre-manifest rules and disarmed every manifest-era refusal (iss-130).
// The manifest's absence is a deletion only where the manifest had been the
// era marker, which history shows: a release tag older than the release
// judged whose tree carried the manifest. So "" when no such tag exists — a
// repository whose releases predate the manifest, or a managed repository
// whose scaffolded release workflow runs the gate and never had abcd's
// manifest — and a refusal naming the tag otherwise. The release job checks
// out full history with its tags, and the release version is read from the
// gated tree's CHANGELOG.md; release tags that cannot be read, or a manifest-era
// tree that names no release, fail closed.
func missingManifestRefusal(repoRoot string, root *os.Root) string {
	const head = "receipt_gate finds no release-gate manifest " + releaseGateManifestPath
	const tail = "; a release after one that carried the manifest is in the manifest era, and without it the gate would fall back to the pre-manifest rules (iss-130), so the release gate fails closed"
	tags, err := launch.GitExistingTags(repoRoot)
	if err != nil {
		return head + " and cannot read the release tags that show whether an earlier release carried it (" + bareCause(err) + ")" + tail
	}
	v, why := releasedVersionInTree(root)
	sort.Slice(tags, func(i, j int) bool { return launch.CoreGreater(tags[i], tags[j]) })
	for _, tag := range tags {
		if why == "" && !launch.CoreGreater(v, tag) {
			continue
		}
		held, err := tagCarriesManifest(repoRoot, tag)
		if err != nil {
			return head + " and cannot read the tree of release v" + tag.String() + " (" + bareCause(err) + ")" + tail
		}
		if !held {
			continue
		}
		if why != "" {
			return head + " and " + why + ", though release v" + tag.String() + " carried it" + tail
		}
		return head + " in the tree of release v" + v.String() + ", though release v" + tag.String() + " carried it" + tail
	}
	return ""
}

// tagCarriesManifest reports whether the tree of release tag v carries the
// release-gate manifest. A git failure is an error, never an absence.
func tagCarriesManifest(repoRoot string, v launch.Semver) (bool, error) {
	out, err := gitutil.Run(repoRoot, "ls-tree", "--name-only", "refs/tags/v"+v.String(), "--", releaseGateManifestPath)
	if err != nil {
		return false, err
	}
	return out != "", nil
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
