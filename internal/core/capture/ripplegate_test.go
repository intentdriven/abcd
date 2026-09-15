// Package capture_test holds the itd-114 ripple gate: the acceptance criterion
// that the repository's own id consumers hold unchanged against freshly minted
// timestamp-numeric ids and a mixed (legacy + native) ledger. It is an external
// test package because the consumers it drives (internal/core/record) import
// capture back.
package capture_test

import (
	"path/filepath"
	"regexp"
	"testing"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/record"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/gittest"
)

var reNative = regexp.MustCompile(`^iss-[0-9]{16}$`)

// mustCapture wraps capture.Capture with the boilerplate request fields.
func mustCapture(t *testing.T, root, ledger, slug, forceID string) capture.CaptureResult {
	t.Helper()
	res, err := capture.Capture(capture.CaptureRequest{
		RepoRoot: root, IssuesRoot: ledger, Text: "ripple gate fixture: " + slug,
		Severity: capture.SeverityMinor, Category: "bug", Source: "manual-test",
		FoundDuring: "ripple-gate", Slug: slug, ForceID: forceID,
	})
	if err != nil {
		t.Fatalf("capture %s: %v", slug, err)
	}
	return res
}

// TestRippleGateConsumersHoldOnMintedIDs executes itd-114's consumer criterion
// rather than asserting it: a mixed ledger — legacy sequential ids below a
// tagged release, native timestamp-numeric ids minted on top by the real mint
// (production clock and entropy) — is run through the release-cut derivation,
// the canonical resolver, the record-lint uniqueness/impact/schema rules,
// `capture list` ordering, and the `abcd <id>` dispatch. Every consumer must
// resolve, cite, sort, and gate the new ids with no modification, and the
// legacy ids must stay valid and unrenumbered beside them.
func TestRippleGateConsumersHoldOnMintedIDs(t *testing.T) {
	r := gittest.NewRepo(t)
	root := r.Root()
	ledger := filepath.Join(root, capture.LedgerRelPath)

	// The legacy era: two sequential-id records and a tagged release base.
	legacyOpen := mustCapture(t, root, ledger, "legacy-open", "iss-3")
	legacyFixed := mustCapture(t, root, ledger, "legacy-fixed", "iss-374")
	if _, err := capture.Resolve(capture.ResolveRequest{
		Grounds:  "pursued: we expect the ledger to keep the reasoning the session would otherwise lose",
		RepoRoot: root, IssuesRoot: ledger, ID: legacyFixed.ID,
		Resolution: "fixed in the legacy era", Impact: "fix",
	}); err != nil {
		t.Fatal(err)
	}
	r.Write("CHANGELOG.md", "# Changelog\n\n## [0.1.0] - 2026-08-01\n\n- base\n")
	r.Commit("legacy ledger")
	r.Git("tag", "v0.1.0")

	// The native era: the real production mint (no injected seams), one record
	// left open and one resolved with a release-relevant impact.
	nativeOpen := mustCapture(t, root, ledger, "native-open", "")
	nativeFixed := mustCapture(t, root, ledger, "native-fixed", "")
	for _, id := range []string{nativeOpen.ID, nativeFixed.ID} {
		if !reNative.MatchString(id) {
			t.Fatalf("mint produced %q, want the 16-digit native shape", id)
		}
	}
	if _, err := capture.Resolve(capture.ResolveRequest{
		Grounds:  "pursued: we expect the ledger to keep the reasoning the session would otherwise lose",
		RepoRoot: root, IssuesRoot: ledger, ID: nativeFixed.ID,
		Resolution: "fixed in the native era", Impact: "fix",
	}); err != nil {
		t.Fatalf("resolve native id: %v", err)
	}
	r.Commit("native records")

	// Consumer 1 — capture list ordering: ascending numeric order must put the
	// mixed ledger in correct time order, legacy era first, and the legacy ids
	// must still be present under their original ids (capture-stable).
	lr, err := capture.List(capture.ListRequest{RepoRoot: root, IssuesRoot: ledger})
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, iss := range lr.Issues {
		order = append(order, iss.ID)
	}
	if len(order) != 4 {
		t.Fatalf("list = %v (skipped %v), want 4 issues", order, lr.Skipped)
	}
	pos := map[string]int{}
	for i, id := range order {
		pos[id] = i
	}
	if !(pos[legacyOpen.ID] < pos[legacyFixed.ID] && pos[legacyFixed.ID] < pos[nativeOpen.ID] && pos[legacyFixed.ID] < pos[nativeFixed.ID]) {
		t.Fatalf("mixed ledger out of time order: %v", order)
	}

	// Consumer 2 — the status board's newest-first view leads with a native id.
	st, err := capture.Status(capture.StatusRequest{RepoRoot: root, IssuesRoot: ledger})
	if err != nil {
		t.Fatal(err)
	}
	if len(st.RecentOpen) == 0 || st.RecentOpen[0].ID != nativeOpen.ID {
		t.Fatalf("recent-open must lead with the native mint, got %+v", st.RecentOpen)
	}

	// Consumer 3 — the canonical resolver answers for both eras.
	resolver, err := recordid.NewResolver(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{legacyOpen.ID, nativeOpen.ID, nativeFixed.ID} {
		if _, ok := resolver.Lookup(id); !ok {
			t.Fatalf("resolver does not resolve %s", id)
		}
	}

	// Consumer 4 — the `abcd <id>` dispatch describes a native id.
	d, err := record.Describe(root, nativeOpen.ID)
	if err != nil {
		t.Fatalf("Describe(%s): %v", nativeOpen.ID, err)
	}
	if d.Family != "issue" || d.Status != "open" || d.ID != nativeOpen.ID {
		t.Fatalf("Describe(%s) = %+v", nativeOpen.ID, d)
	}

	// Consumer 5 — the record-lint uniqueness, impact, and schema rules gate the
	// mixed ledger clean: the armed detectors accept the scheme's output.
	findings, err := lint.Lint(lint.Config{
		Roots: []string{".abcd"},
		Rules: map[string]lint.RuleConfig{
			"issue_id_unique":    {Enabled: true, Severity: "blocker", IssuesDir: capture.LedgerRelPath},
			"issue_impact_valid": {Enabled: true, Severity: "blocker", IssuesDir: capture.LedgerRelPath},
			"record_schema":      {Enabled: true, Severity: "blocker", RecordStores: map[string]string{"iss": capture.LedgerRelPath}},
		},
	}, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("record-lint must gate the mixed ledger clean, got %+v", findings)
	}

	// Consumer 6 — the release-cut derivation: the cut since v0.1.0 contains
	// exactly the native resolved record (open issues are not release records),
	// its id parsed from the filename, its impact read, and the version bumped.
	shipped, err := changelog.ShippedSince(root, "v0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(shipped.Added) != 1 || shipped.Added[0].ID != nativeFixed.ID {
		t.Fatalf("cut since v0.1.0 = %+v, want exactly the native resolved record %s", shipped.Added, nativeFixed.ID)
	}
	if string(shipped.Added[0].Impact) != "fix" {
		t.Fatalf("cut record impact = %q, want fix", shipped.Added[0].Impact)
	}
	deriv, err := changelog.Derive(root)
	if err != nil {
		t.Fatal(err)
	}
	if deriv.Refused {
		t.Fatalf("derivation refused over the native record: %s", deriv.RefusalReason)
	}
	if !deriv.Bumped || deriv.NextTag != "v0.1.1" {
		t.Fatalf("derivation = bump %v next %q, want a patch bump to v0.1.1 from the fix-impact native record", deriv.Bumped, deriv.NextTag)
	}
}
