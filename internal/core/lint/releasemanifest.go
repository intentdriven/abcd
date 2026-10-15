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
	"strings"

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
	tags, err := listReleaseTags(repoRoot)
	if err != nil {
		return head + " and cannot read the release tags that show whether an earlier release carried it (" + bareCause(err) + ")" + tail
	}
	v, why := releasedVersionInTree(root)
	for _, tag := range tags {
		if why == "" && !launch.CoreGreater(v, tag.version) {
			continue
		}
		held, err := commitCarriesManifest(repoRoot, tag.commit)
		if err != nil {
			return head + " and cannot read the tree of release v" + tag.version.String() + " (" + bareCause(err) + ")" + tail
		}
		if !held {
			continue
		}
		if why != "" {
			return head + " and " + why + ", though release v" + tag.version.String() + " carried it" + tail
		}
		return head + " in the tree of release v" + v.String() + ", though release v" + tag.version.String() + " carried it" + tail
	}
	return ""
}

// backdatedReleaseRefusal is the release gate's verdict on the release's own
// version against the repository's release tags. Both era refusals key on the
// version the gated tree's CHANGELOG.md names, so a release that named an
// unused version older than the newest release, and pushed a matching new tag
// (creating a tag is open), would be judged as that old release: a manifest
// set back below schemaVersion 2 or deleted would pass. So in the manifest era
// — the tree carries the manifest, or a release tag's tree did — the release's
// version is newer than every release tag, or the release is a re-gate: its
// version is already tagged at the commit gated, and every newer release tag
// descends from that commit. A version tagged at another commit is never
// released again. "" admits; a repository whose releases never carried the
// manifest keeps its own versioning; tags that cannot be listed inside a
// checkout fail closed, and a tree outside git is left to the
// missing-manifest refusal, which fails closed on it.
func backdatedReleaseRefusal(repoRoot string, root *os.Root, manifestPresent bool) string {
	const tail = ", so the release gate fails closed"
	tags, err := listReleaseTags(repoRoot)
	if err != nil {
		// Outside a checkout nothing resolves, and the missing-manifest
		// refusal fails closed on its own; inside one, a listing that fails
		// is not an absence of releases.
		if _, herr := gitutil.Run(repoRoot, "rev-parse", "--verify", "HEAD^{commit}"); herr != nil {
			return ""
		}
		return "receipt_gate cannot read the release tags to judge the release's version against the earlier releases (" + bareCause(err) + ")" + tail
	}
	inEra := manifestPresent
	for _, tag := range tags {
		if inEra {
			break
		}
		held, err := commitCarriesManifest(repoRoot, tag.commit)
		if err != nil {
			return "receipt_gate cannot read the tree of release v" + tag.version.String() + " (" + bareCause(err) + ") to tell whether the release is in the manifest era" + tail
		}
		inEra = held
	}
	if !inEra {
		return ""
	}
	v, why := releasedVersionInTree(root)
	if why != "" {
		return "" // the era refusals read the same version and fail closed on it
	}
	head, err := gitutil.Run(repoRoot, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return "receipt_gate cannot resolve the commit gated (" + bareCause(err) + ")" + tail
	}
	var own *releaseTag
	var newer []releaseTag
	for i, tag := range tags {
		switch {
		case launch.CoreGreater(tag.version, v):
			newer = append(newer, tag)
		case !launch.CoreGreater(v, tag.version):
			own = &tags[i]
		}
	}
	if own != nil && own.commit != head {
		return "receipt_gate finds release v" + v.String() + " is already tagged at another commit (" + own.commit +
			"); a released version is never released again from a different commit" + tail
	}
	if len(newer) == 0 {
		return ""
	}
	if own == nil {
		return "receipt_gate finds release v" + v.String() + " is not newer than release v" + newer[0].version.String() +
			"; a release's version is newer than every release tag, so the manifest-era refusals cannot be stepped around with an older version" + tail
	}
	for _, tag := range newer {
		_, notAncestor, err := gitutil.RunCappedExitOne(repoRoot, 4096, "merge-base", "--is-ancestor", head, tag.commit)
		if err != nil {
			return "receipt_gate cannot tell whether release v" + tag.version.String() + " descends from the commit gated (" + bareCause(err) + ")" + tail
		}
		if notAncestor {
			return "receipt_gate finds release v" + v.String() + " is not newer than release v" + tag.version.String() +
				", which does not descend from the commit gated; only a re-gate of an earlier release, at the commit its tag names, may carry an older version" + tail
		}
	}
	return ""
}

// releaseTag is one release as the repository's tags record it: the version a
// vX.Y.Z tag names and the commit it peels to.
type releaseTag struct {
	version launch.Semver
	commit  string
}

// listReleaseTags is releaseTags, the one reader the era refusals use; a test
// replaces it to fail the listing inside a checkout.
var listReleaseTags = releaseTags

// releaseTags lists the repository's releases, newest first, in one
// for-each-ref: every tag named v followed by an X.Y.Z version — a tag without
// the v prefix, a pre-release or a build tag is not a release, as
// launch.GitExistingTags reads them — that peels to a commit. A tag naming a blob or a tree, or a tag of a tag, is not a
// release: the tag rules forbid deleting it, so reading it as one would let a
// single stray tag refuse every later release for good.
func releaseTags(repoRoot string) ([]releaseTag, error) {
	out, err := gitutil.Run(repoRoot, "for-each-ref",
		"--format=%(refname:strip=2)%09%(objecttype)%09%(objectname)%09%(*objecttype)%09%(*objectname)", "refs/tags/v*")
	if err != nil {
		return nil, err
	}
	var tags []releaseTag
	for _, line := range strings.Split(out, "\n") {
		// The peeled fields are empty for a lightweight tag, and the output's
		// trimming drops them from the last line, so they are padded back.
		f := append(strings.Split(line, "\t"), "", "", "", "")[:5]
		v, err := launch.ParseSemver(strings.TrimPrefix(f[0], "v"))
		if err != nil || !strings.HasPrefix(f[0], "v") || v.Prerelease != "" || v.Build != "" {
			continue
		}
		var commit string
		switch {
		case f[1] == "commit":
			commit = f[2]
		case f[1] == "tag" && f[3] == "commit":
			commit = f[4]
		default:
			continue
		}
		tags = append(tags, releaseTag{version: v, commit: commit})
	}
	sort.Slice(tags, func(i, j int) bool { return launch.CoreGreater(tags[i].version, tags[j].version) })
	return tags, nil
}

// commitCarriesManifest reports whether commit's tree carries the release-gate
// manifest. A git failure is an error, never an absence.
func commitCarriesManifest(repoRoot, commit string) (bool, error) {
	out, err := gitutil.Run(repoRoot, "ls-tree", "--name-only", commit, "--", releaseGateManifestPath)
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
