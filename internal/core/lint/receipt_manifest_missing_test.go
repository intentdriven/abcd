package lint

import (
	"path/filepath"
	"testing"
)

// iss-130: deleting the manifest from a post-v0.4.0 tree once reverted the
// release gate to the pre-manifest rules, disarming every manifest-era refusal.
// The release's arming refuses such a tree; the unarmed reader the doc-fidelity
// gate shares keeps the era marker as it was.
func TestReleaseGateRefusesAMissingManifestAfterV040(t *testing.T) {
	const sha = "0123456789abcdef0123456789abcdef01234567"
	gates := []string{"iss35-brief-surface-crosscheck", "docs-currency-reviewer"}
	// setup is a tree with no manifest, whose receipts break a manifest-era
	// rule: they echo no manifest hash and pin no tier.
	setup := func(t *testing.T, changelog string) (string, RuleConfig) {
		root := t.TempDir()
		if changelog != "" {
			writeFile(t, root, "CHANGELOG.md", changelog)
		}
		for _, g := range gates {
			writeFile(t, root, filepath.Join(reviewsSubdir, sha, g+".json"), manifestReceipt(sha, g, "", "", `[{"where":"x"}]`))
		}
		return root, ArmReceiptGate(Config{}, sha, gates).Rules["receipt_gate"]
	}
	t.Run("a post-v0.4.0 release is refused", func(t *testing.T) {
		root, cfg := setup(t, "# Changelog\n\n## [Unreleased]\n\n## [0.15.0] - 2026-10-15\n\n## [0.4.0] - 2026-07-01\n")
		if fs := runReceiptGate(t, root, cfg); !findingWith(fs, reviewsSubdir, "receipt_gate", "no release-gate manifest "+releaseGateManifestPath+" in the tree of release v0.15.0") {
			t.Fatalf("a v0.15.0 tree with no manifest must be refused: %+v", fs)
		}
	})
	t.Run("a tree naming no release is refused", func(t *testing.T) {
		root, cfg := setup(t, "")
		if fs := runReceiptGate(t, root, cfg); !findingWith(fs, reviewsSubdir, "receipt_gate", "cannot read the release version from CHANGELOG.md") {
			t.Fatalf("a tree whose release version cannot be read must fail closed: %+v", fs)
		}
	})
	t.Run("v0.4.0 keeps the pre-manifest rules", func(t *testing.T) {
		root, cfg := setup(t, "# Changelog\n\n## [0.4.0] - 2026-07-01\n")
		if fs := runReceiptGate(t, root, cfg); countRule(fs, "receipt_gate") != 0 {
			t.Fatalf("a v0.4.0 tree predates the manifest and keeps the old rules: %+v", fs)
		}
	})
	t.Run("the unarmed reader keeps the pre-manifest rules", func(t *testing.T) {
		root, _ := setup(t, "# Changelog\n\n## [0.15.0] - 2026-10-15\n")
		got, err := CheckGateReceipt(root, reviewsSubdir, sha, gates[0])
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Problems) != 0 {
			t.Fatalf("CheckGateReceipt is not the release gate and must not refuse a missing manifest: %+v", got.Problems)
		}
	})
}
