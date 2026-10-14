package lint

import (
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
		x.r.Write(filepath.Join(reviewsSubdir, tagSha, g+".json"), manifestReceipt(tagSha, g, "", "", `[{"disposition":"accepted: x"}]`))
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
