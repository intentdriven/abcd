package lint

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// tagRepo is a git repository the release gate's history reads run against:
// each step writes a CHANGELOG naming one release, with or without the
// manifest, commits it and optionally tags it.
type tagRepo struct {
	t *testing.T
	r *gittest.Repo
}

const tagSha = "0123456789abcdef0123456789abcdef01234567"

var tagGates = []string{"iss35-brief-surface-crosscheck", "docs-currency-reviewer"}

func newTagRepo(t *testing.T) *tagRepo { return &tagRepo{t: t, r: gittest.NewRepo(t)} }

func (x *tagRepo) drop(rel string) {
	x.t.Helper()
	if err := os.Remove(filepath.Join(x.r.Root(), filepath.FromSlash(rel))); err != nil && !os.IsNotExist(err) {
		x.t.Fatal(err)
	}
}

// commit writes the tree of release v and commits it. manifest is the
// manifest's schemaVersion, or "" for no manifest. The receipts are
// pre-manifest shaped, so only a refusal of the tree itself can fire on them
// in a tree with no manifest.
func (x *tagRepo) commit(v, manifest string) {
	x.t.Helper()
	x.r.Write("CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n\n## ["+v+"] - 2026-10-01\n")
	if manifest == "" {
		x.drop(releaseGateManifestPath)
	} else {
		x.r.Write(releaseGateManifestPath, `{"schemaVersion":`+manifest+`}`+"\n")
	}
	for _, g := range tagGates {
		receipt := manifestReceipt(tagSha, g, "", "", `[{"disposition":"accepted: x"}]`)
		if manifest != "" {
			// Conforming manifest-era receipts, so only a refusal of the
			// tree's release history can fire.
			receipt = manifestReceipt(tagSha, g, releaseTierFull, hashManifest([]byte(`{"schemaVersion":`+manifest+`}`+"\n")), `[]`)
		}
		x.r.Write(filepath.Join(reviewsSubdir, tagSha, g+".json"), receipt)
	}
	x.r.Commit("release " + v)
}

func (x *tagRepo) tag(v string) { x.t.Helper(); x.r.Git("tag", "v"+v) }

func (x *tagRepo) run() []Finding {
	x.t.Helper()
	return runReceiptGate(x.t, x.r.Root(), ArmReceiptGate(Config{}, tagSha, tagGates).Rules["receipt_gate"])
}

// A v* tag that does not peel to a commit is not a release: it once made the
// manifest-era read fail on every later release, and the tag rules mean no one
// can delete it, so a managed repository was blocked for good.
func TestReleaseGateTreatsATagThatIsNoCommitAsNoRelease(t *testing.T) {
	plant := func(x *tagRepo) {
		x.r.Git("tag", "v1.2.0", x.r.Git("rev-parse", "HEAD:CHANGELOG.md"))
		x.r.Git("tag", "-a", "-m", "a tree", "v1.3.0", x.r.Git("rev-parse", "HEAD^{tree}"))
	}
	t.Run("a managed repository keeps the pre-manifest rules", func(t *testing.T) {
		x := newTagRepo(t)
		x.commit("1.0.0", "")
		x.tag("1.0.0")
		plant(x)
		x.commit("1.5.0", "")
		if fs := x.run(); countRule(fs, "receipt_gate") != 0 {
			t.Fatalf("a tag that is no commit must not block a managed repository's release: %+v", fs)
		}
	})
	t.Run("a manifest-era repository is still judged by its real releases", func(t *testing.T) {
		x := newTagRepo(t)
		x.commit("0.14.1", "2")
		x.tag("0.14.1")
		plant(x)
		x.commit("1.5.0", "")
		if fs := x.run(); !findingWith(fs, reviewsSubdir, "receipt_gate", "in the tree of release v1.5.0, though release v0.14.1 carried it") {
			t.Fatalf("the manifest-era refusal must name the real release that carried the manifest: %+v", fs)
		}
	})
}

// A cutter could top the CHANGELOG with an unused version older than the
// newest release, push a matching new tag (creating a tag is open), and so
// step around both era refusals, which key on the release version: a
// schemaVersion-1 manifest or a deleted one would pass as an old release. In
// the manifest era the release's version must be newer than every release
// tag, except a re-gate of a release already tagged at the commit gated, whose
// commit the newer releases descend from.
func TestReleaseGateRefusesABackdatedRelease(t *testing.T) {
	era := func(t *testing.T) *tagRepo {
		x := newTagRepo(t)
		x.commit("0.13.0", "2")
		x.tag("0.13.0")
		x.commit("0.14.1", "2")
		x.tag("0.14.1")
		return x
	}
	t.Run("a backdated release with a fresh tag is refused", func(t *testing.T) {
		x := era(t)
		x.commit("0.13.5", "1")
		x.tag("0.13.5")
		if fs := x.run(); !findingWith(fs, reviewsSubdir, "receipt_gate", "release v0.13.5 is not newer than release v0.14.1") {
			t.Fatalf("a backdated release must be refused: %+v", fs)
		}
	})
	t.Run("a backdated release before its tag is refused", func(t *testing.T) {
		x := era(t)
		x.commit("0.4.5", "")
		if fs := x.run(); !findingWith(fs, reviewsSubdir, "receipt_gate", "release v0.4.5 is not newer than release v0.14.1") {
			t.Fatalf("a backdated release that deleted the manifest must be refused: %+v", fs)
		}
	})
	t.Run("a newer tag planted at the backdated commit does not help", func(t *testing.T) {
		x := era(t)
		x.commit("0.13.5", "1")
		x.tag("0.13.5")
		x.tag("99.0.0")
		if fs := x.run(); !findingWith(fs, reviewsSubdir, "receipt_gate", "release v0.13.5 is not newer than release v0.14.1") {
			t.Fatalf("a planted newer tag must not admit a backdated release: %+v", fs)
		}
	})
	t.Run("a version already tagged at another commit is refused", func(t *testing.T) {
		x := era(t)
		x.commit("0.14.1", "1")
		if fs := x.run(); !findingWith(fs, reviewsSubdir, "receipt_gate", "release v0.14.1 is already tagged at another commit") {
			t.Fatalf("a release reusing a tagged version must be refused: %+v", fs)
		}
	})
	t.Run("a fresh release passes", func(t *testing.T) {
		x := era(t)
		x.commit("0.15.0", "2")
		if fs := x.run(); countRule(fs, "receipt_gate") != 0 {
			t.Fatalf("a fresh v0.15.0 must pass: %+v", fs)
		}
		// verify on a tag push, and a re-run once the tag exists.
		x.tag("0.15.0")
		if fs := x.run(); countRule(fs, "receipt_gate") != 0 {
			t.Fatalf("v0.15.0 gated at its own tag must pass: %+v", fs)
		}
	})
	t.Run("a re-gate of an existing release at its own commit passes", func(t *testing.T) {
		x := era(t)
		x.r.Git("checkout", "-q", "--detach", "v0.13.0")
		if fs := x.run(); countRule(fs, "receipt_gate") != 0 {
			t.Fatalf("re-gating v0.13.0 at its tagged commit must pass: %+v", fs)
		}
	})
	t.Run("a managed repository is untouched", func(t *testing.T) {
		x := newTagRepo(t)
		x.commit("1.4.0", "")
		x.tag("1.4.0")
		x.commit("1.3.9", "")
		if fs := x.run(); countRule(fs, "receipt_gate") != 0 {
			t.Fatalf("a repository that never had the manifest keeps its own versioning: %+v", fs)
		}
	})
}

// Inside a checkout, a release-tag listing that fails is not an absence of
// releases: the backdated-version check fails closed on it, as the
// missing-manifest refusal does. Outside a checkout, where nothing resolves,
// it is left to the missing-manifest refusal.
func TestReleaseGateFailsClosedWhenTheTagsCannotBeListed(t *testing.T) {
	saved := listReleaseTags
	t.Cleanup(func() { listReleaseTags = saved })
	listReleaseTags = func(string) ([]releaseTag, error) { return nil, errors.New("for-each-ref failed") }

	x := newTagRepo(t)
	x.commit("0.15.0", "2")
	if fs := x.run(); !findingWith(fs, reviewsSubdir, "receipt_gate", "cannot read the release tags") {
		t.Fatalf("a checkout whose tags cannot be listed must fail closed: %+v", fs)
	}
}
