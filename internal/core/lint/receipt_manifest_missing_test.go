package lint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// iss-130: deleting the manifest from a tree that had been in the manifest era
// once reverted the release gate to the pre-manifest rules, disarming every
// manifest-era refusal. The release's arming refuses such a tree. The era is
// read from history: a manifest that an earlier release tag carried was the
// era marker, and a repository whose releases never carried one, such as a
// managed repository whose release workflow abcd scaffolded, keeps the
// pre-manifest rules. The unarmed reader the doc-fidelity gate shares keeps
// the era marker as it was.
func TestReleaseGateRefusesAMissingManifestAfterTheManifestEra(t *testing.T) {
	const sha = "0123456789abcdef0123456789abcdef01234567"
	gates := []string{"iss35-brief-surface-crosscheck", "docs-currency-reviewer"}
	changelog := func(versions ...string) string {
		body := "# Changelog\n\n## [Unreleased]\n"
		for _, v := range versions {
			body += "\n## [" + v + "] - 2026-10-01\n"
		}
		return body
	}
	// release tags each version in turn, writing the manifest into the tree of
	// those listed in withManifest, then rolls the tree to the release under
	// gate with no manifest, holding receipts that break a manifest-era rule:
	// they echo no manifest hash and pin no tier.
	release := func(t *testing.T, tags []string, withManifest map[string]bool, gated string) (string, RuleConfig) {
		r := gittest.NewRepo(t)
		drop := func(rel string) {
			if err := os.Remove(filepath.Join(r.Root(), filepath.FromSlash(rel))); err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
		}
		for i, v := range tags {
			r.Write("CHANGELOG.md", changelog(tags[i]))
			if withManifest[v] {
				r.Write(releaseGateManifestPath, `{"schemaVersion":2}`+"\n")
			} else {
				drop(releaseGateManifestPath)
			}
			r.Commit("release " + v)
			r.Git("tag", "v"+v)
		}
		drop(releaseGateManifestPath)
		if gated != "" {
			r.Write("CHANGELOG.md", changelog(gated))
		} else {
			drop("CHANGELOG.md")
		}
		for _, g := range gates {
			r.Write(filepath.Join(reviewsSubdir, sha, g+".json"), manifestReceipt(sha, g, "", "", `[{"where":"x"}]`))
		}
		r.Commit("roll")
		return r.Root(), ArmReceiptGate(Config{}, sha, gates).Rules["receipt_gate"]
	}

	t.Run("a release after a tag that carried the manifest is refused", func(t *testing.T) {
		root, cfg := release(t, []string{"0.4.0", "0.14.1"}, map[string]bool{"0.14.1": true}, "0.15.0")
		if fs := runReceiptGate(t, root, cfg); !findingWith(fs, reviewsSubdir, "receipt_gate",
			"no release-gate manifest "+releaseGateManifestPath+" in the tree of release v0.15.0, though release v0.14.1 carried it") {
			t.Fatalf("a v0.15.0 tree that lost the manifest v0.14.1 carried must be refused: %+v", fs)
		}
	})
	t.Run("a tree naming no release in the manifest era is refused", func(t *testing.T) {
		root, cfg := release(t, []string{"0.14.1"}, map[string]bool{"0.14.1": true}, "")
		if fs := runReceiptGate(t, root, cfg); !findingWith(fs, reviewsSubdir, "receipt_gate", "cannot read the release version from CHANGELOG.md") {
			t.Fatalf("a manifest-era tree whose release version cannot be read must fail closed: %+v", fs)
		}
	})
	t.Run("a repository whose releases never carried the manifest keeps the pre-manifest rules", func(t *testing.T) {
		root, cfg := release(t, []string{"1.0.0", "1.4.0"}, nil, "1.5.0")
		if fs := runReceiptGate(t, root, cfg); countRule(fs, "receipt_gate") != 0 {
			t.Fatalf("a managed repository that never had the manifest must keep the old rules: %+v", fs)
		}
	})
	t.Run("a repository with no release yet keeps the pre-manifest rules", func(t *testing.T) {
		root, cfg := release(t, nil, nil, "")
		if fs := runReceiptGate(t, root, cfg); countRule(fs, "receipt_gate") != 0 {
			t.Fatalf("a repository with no tag and no manifest must keep the old rules: %+v", fs)
		}
	})
	t.Run("a re-gate of a release older than the first manifest keeps the pre-manifest rules", func(t *testing.T) {
		// Re-gating v0.3.0, at its own tagged commit, after v0.5.0 introduced
		// the manifest: only the tags older than the release judged can show it
		// was in the era.
		x := newTagRepo(t)
		x.commit("0.2.0", "")
		x.tag("0.2.0")
		x.commit("0.3.0", "")
		x.tag("0.3.0")
		x.commit("0.5.0", "2")
		x.tag("0.5.0")
		x.r.Git("checkout", "-q", "--detach", "v0.3.0")
		if fs := x.run(); countRule(fs, "receipt_gate") != 0 {
			t.Fatalf("a release whose older tags never carried the manifest keeps the old rules: %+v", fs)
		}
	})
	t.Run("a tree outside git fails closed", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, "CHANGELOG.md", changelog("0.15.0"))
		for _, g := range gates {
			writeFile(t, root, filepath.Join(reviewsSubdir, sha, g+".json"), manifestReceipt(sha, g, "", "", `[]`))
		}
		cfg := ArmReceiptGate(Config{}, sha, gates).Rules["receipt_gate"]
		if fs := runReceiptGate(t, root, cfg); !findingWith(fs, reviewsSubdir, "receipt_gate", "cannot read the release tags") {
			t.Fatalf("a tree whose release history cannot be read must fail closed: %+v", fs)
		}
	})
	t.Run("the unarmed reader keeps the pre-manifest rules", func(t *testing.T) {
		root, _ := release(t, []string{"0.14.1"}, map[string]bool{"0.14.1": true}, "0.15.0")
		got, err := CheckGateReceipt(root, reviewsSubdir, sha, gates[0])
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Problems) != 0 {
			t.Fatalf("CheckGateReceipt is not the release gate and must not refuse a missing manifest: %+v", got.Problems)
		}
	})
}
