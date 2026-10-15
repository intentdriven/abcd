package lint

import "testing"

// A release branch could set the manifest back to schemaVersion 1, re-hash
// its receipts and so turn the closed disposition vocabulary off for every
// later release. A release newer than v0.14.1, the last to predate the
// vocabulary, is refused with a manifest below schemaVersion 2.
func TestReleaseGateRefusesASchemaVersionDowngrade(t *testing.T) {
	setup := func(t *testing.T, changelog string) *dispFixture {
		f := newDispFixture(t, "1")
		if changelog != "" {
			writeFile(t, f.root, "CHANGELOG.md", changelog)
		}
		f.put(dispGateA, `[{"disposition":"accepted: knowingly shipped unfixed"}]`)
		return f
	}
	t.Run("a v0.15.0 tree at schemaVersion 1 is refused", func(t *testing.T) {
		f := setup(t, "# Changelog\n\n## [Unreleased]\n\n## [0.15.0] - 2026-10-15\n\n## [0.14.1] - 2026-10-14\n")
		if fs := f.run(); !refusal(fs, "v0.15.0", "schemaVersion 1", releaseGateManifestPath) {
			t.Fatalf("a v0.15.0 tree whose manifest is schemaVersion 1 must be refused: %+v", fs)
		}
	})
	t.Run("a tree naming no release is refused", func(t *testing.T) {
		f := setup(t, "")
		if fs := f.run(); !refusal(fs, "schemaVersion 1", "CHANGELOG.md") {
			t.Fatalf("a schemaVersion-1 tree whose release cannot be read must fail closed: %+v", fs)
		}
	})
	t.Run("v0.14.1 at schemaVersion 1 keeps the free-text dispositions", func(t *testing.T) {
		f := setup(t, "# Changelog\n\n## [0.14.1] - 2026-10-14\n")
		if fs := f.run(); countRule(fs, "receipt_gate") != 0 {
			t.Fatalf("v0.14.1 predates the vocabulary and keeps its old rules: %+v", fs)
		}
	})
	t.Run("the unarmed reader is untouched", func(t *testing.T) {
		f := setup(t, "# Changelog\n\n## [0.15.0] - 2026-10-15\n")
		got, err := CheckGateReceipt(f.root, reviewsSubdir, dispSha, dispGateA)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Problems) != 0 {
			t.Fatalf("CheckGateReceipt is not the release gate and must not refuse a schemaVersion: %+v", got.Problems)
		}
	})
}
