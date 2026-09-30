package capture

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestCaptureBranchesSameInstantNeverCollide is itd-114's first acceptance
// criterion at the capture surface, and the successor of the retired
// refs-union test (iss-115/iss-120): two branches cut from one base each mint
// before either commits — the exact residual window max+1-with-refs-union
// accepted, and the mechanism behind the 2026-08-19/20 collisions. Under the
// timestamp-numeric mint the two ids differ even with both clocks pinned to
// the same instant, with no refs scan and no coordination: independent entropy
// alone separates them.
func TestCaptureBranchesSameInstantNeverCollide(t *testing.T) {
	instant := time.Date(2026, 8, 20, 11, 42, 7, 0, time.UTC)
	// The entropy is scripted with two distinct draws (42, then 4369): the test's
	// subject is that the mint consults no refs and no maximum, and only the
	// suffix separates the two branches — asserting that on live crypto/rand
	// would fail on the spec's own accepted ~10^-4 same-suffix residue, blaming
	// a correct implementation. Determinism costs one field.
	setMinter(t, recordid.Minter{
		Now:     func() time.Time { return instant },
		Entropy: bytes.NewReader([]byte{0x00, 0x2A, 0x11, 0x11}),
	})

	r := gittest.NewRepo(t)
	r.Write("README.md", "# base\n")
	r.Commit("base")
	ledger := filepath.Join(r.Root(), LedgerRelPath)

	// Branch A: mint, do NOT commit yet (the uncommitted-mint window).
	resA, err := testCapture(CaptureRequest{
		RepoRoot: r.Root(), IssuesRoot: ledger, Text: "alpha observation",
		Severity: SeverityMinor, Category: "bug", Source: "user-observation",
		FoundDuring: "t", Slug: "alpha",
	})
	if err != nil {
		t.Fatalf("branch A Capture: %v", err)
	}
	r.Commit("A: " + resA.ID)

	// Branch B, cut from the base BEFORE A's commit: its working tree carries no
	// issue file and, unlike the retired scheme, the mint consults no refs — the
	// outcome must not depend on whether A has merged (the field test's P2).
	r.Git("checkout", "-b", "branch-b", "HEAD~1")
	resB, err := testCapture(CaptureRequest{
		RepoRoot: r.Root(), IssuesRoot: ledger, Text: "beta observation",
		Severity: SeverityMinor, Category: "bug", Source: "user-observation",
		FoundDuring: "t", Slug: "beta",
	})
	if err != nil {
		t.Fatalf("branch B Capture: %v", err)
	}
	if resA.ID == resB.ID {
		t.Fatalf("same-instant mints on two branches collided: %q", resA.ID)
	}
	if !reNativeIssID.MatchString(resA.ID) || !reNativeIssID.MatchString(resB.ID) {
		t.Fatalf("mints are not native-shaped: %q, %q", resA.ID, resB.ID)
	}
}

// TestCaptureBranchesMergeUnionPassesGates completes itd-114's first
// acceptance criterion literally: after two branches mint at the same instant,
// MERGING them must pass every uniqueness gate with no renumber. The sibling
// test above stops at id inequality; this one performs the actual git merge
// and runs the armed issue_id_unique detector over the union — then proves the
// gate is not vacuous by planting a same-id duplicate and watching it fire.
func TestCaptureBranchesMergeUnionPassesGates(t *testing.T) {
	instant := time.Date(2026, 8, 20, 11, 42, 7, 0, time.UTC)
	setMinter(t, recordid.Minter{
		Now:     func() time.Time { return instant },
		Entropy: bytes.NewReader([]byte{0x00, 0x2A, 0x11, 0x11}),
	})

	r := gittest.NewRepo(t)
	r.Write("README.md", "# base\n")
	r.Commit("base")
	trunk := r.Git("rev-parse", "--abbrev-ref", "HEAD")
	ledger := filepath.Join(r.Root(), LedgerRelPath)

	resA, err := testCapture(CaptureRequest{
		RepoRoot: r.Root(), IssuesRoot: ledger, Text: "alpha observation",
		Severity: SeverityMinor, Category: "bug", Source: "user-observation",
		FoundDuring: "t", Slug: "alpha",
	})
	if err != nil {
		t.Fatalf("branch A Capture: %v", err)
	}
	r.Commit("A: " + resA.ID)

	r.Git("checkout", "-b", "branch-b", "HEAD~1")
	resB, err := testCapture(CaptureRequest{
		RepoRoot: r.Root(), IssuesRoot: ledger, Text: "beta observation",
		Severity: SeverityMinor, Category: "bug", Source: "user-observation",
		FoundDuring: "t", Slug: "beta",
	})
	if err != nil {
		t.Fatalf("branch B Capture: %v", err)
	}
	r.Commit("B: " + resB.ID)

	// The literal merge of the two minting branches: it must be clean (the two
	// ledger files cannot conflict) and the union must carry both ids unrenumbered.
	r.Git("merge", "--no-edit", trunk)
	for _, id := range []string{resA.ID, resB.ID} {
		if _, status, ferr := findIssue(ledger, id); ferr != nil || status != StateOpen {
			t.Fatalf("after the merge, %s is not intact in open/: status=%s err=%v", id, status, ferr)
		}
	}

	gateCfg := lint.Config{Rules: map[string]lint.RuleConfig{
		"issue_id_unique": {Enabled: true, Severity: "blocker", IssuesDir: LedgerRelPath},
	}}
	findings, err := lint.Lint(gateCfg, r.Root())
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("the merged union must pass the uniqueness gate, got %+v", findings)
	}

	// Negative control — the gate must be armed, not vacuously green: a planted
	// same-id claimant makes it fire on every file in the colliding set.
	dupe := filepath.Join(ledger, "open", resA.ID+"-planted-twin.md")
	if err := os.WriteFile(dupe, []byte("---\nid: \""+resA.ID+"\"\n---\n# twin\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err = lint.Lint(gateCfg, r.Root())
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) < 2 {
		t.Fatalf("the planted duplicate must trip the gate on both claimants, got %+v", findings)
	}
}
