package ahoy

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// TestInstallCreatesTheCommittedTiers pins iss-2610071538028804: every tier
// the repository lint's three-tier-layout rule requires is a gap on a repo
// without it, install creates each as a real directory and seeds the shared
// tier's decision log, and nothing is left for detection to report.
func TestInstallCreatesTheCommittedTiers(t *testing.T) {
	setupHermetic(t)
	harnessFixture(t, "")
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	for _, tier := range lint.CommittedTiers() {
		id := committedTierGapID(tier)
		var gap Gap
		for _, g := range det.Gaps {
			if g.ID == id {
				gap = g
			}
		}
		if gap.ID == "" {
			t.Fatalf("no %s gap on a repo without %s: %+v", id, tier.Rel, gapIDs(det.Gaps))
		}
		if !gap.Required || !gap.Resolvable || gap.Category != SafeAutocreate || gap.Scope != "repo" {
			t.Errorf("%s has the wrong contract: %+v", id, gap)
		}
	}

	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "clean" {
		t.Fatalf("status = %q remaining=%v notes=%v", res.Status, res.Remaining, res.Notes)
	}
	writes := strings.Join(res.Writes, "\n")
	for _, tier := range lint.CommittedTiers() {
		if !fsutil.IsRealDir(filepath.Join(repo, filepath.FromSlash(tier.Rel))) {
			t.Errorf("%s is not a real directory after install", tier.Rel)
		}
		if !strings.Contains(writes, tier.Rel) {
			t.Errorf("%s is not on the receipt: %v", tier.Rel, res.Writes)
		}
	}
	ledger, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(lint.DecisionsLedger)))
	if err != nil {
		t.Fatalf("the decision log was not seeded: %v", err)
	}
	if !strings.Contains(writes, lint.DecisionsLedger) {
		t.Errorf("the decision log is not on the receipt: %v", res.Writes)
	}
	// The seed is a header only: it claims no decision was made.
	if regexp.MustCompile(`(?m)^\s*[-*] `).Match(ledger) {
		t.Errorf("the seeded decision log carries an entry:\n%s", ledger)
	}

	after, _ := Detect(repo)
	for _, tier := range lint.CommittedTiers() {
		if hasGap(after.Gaps, committedTierGapID(tier)) {
			t.Errorf("the %s gap persists after install", tier.Rel)
		}
	}
	if hasGap(after.Gaps, decisionsLedgerGapID) {
		t.Error("the decision-log gap persists after install")
	}
}

// TestInstallSeedsTheLedgerIntoAnExistingWorkTier: a shared tier that is
// already there but holds no decision log is a gap of its own, and install
// seeds the log beside what the tier holds.
func TestInstallSeedsTheLedgerIntoAnExistingWorkTier(t *testing.T) {
	setupHermetic(t)
	harnessFixture(t, "")
	repo := t.TempDir()
	for _, d := range []string{".git", ".abcd/development", ".abcd/work"} {
		if err := os.MkdirAll(filepath.Join(repo, filepath.FromSlash(d)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	det, _ := Detect(repo)
	if !hasGap(det.Gaps, decisionsLedgerGapID) {
		t.Fatalf("no %s gap for a work tier without a decision log: %v", decisionsLedgerGapID, gapIDs(det.Gaps))
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	if !fileExists(filepath.Join(repo, filepath.FromSlash(lint.DecisionsLedger))) {
		t.Fatal("the decision log was not seeded into the existing work tier")
	}
}

// TestInstallKeepsAnExistingDecisionLog: install never overwrites a decision
// log the repository already keeps, while it still creates a missing tier.
func TestInstallKeepsAnExistingDecisionLog(t *testing.T) {
	setupHermetic(t)
	harnessFixture(t, "")
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	const own = "# our decisions\n\n- 2026-01-01 — keep it.\n"
	p := filepath.Join(repo, filepath.FromSlash(lint.DecisionsLedger))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(own), 0o644); err != nil {
		t.Fatal(err)
	}
	det, _ := Detect(repo)
	if hasGap(det.Gaps, decisionsLedgerGapID) {
		t.Error("a decision log that exists is reported missing")
	}
	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(p)
	if err != nil || string(got) != own {
		t.Errorf("the existing decision log was changed: %q (err=%v)", got, err)
	}
	if strings.Contains(strings.Join(res.Writes, "\n"), lint.DecisionsLedger) {
		t.Errorf("the receipt claims a write of the existing log: %v", res.Writes)
	}
	if !fsutil.IsRealDir(filepath.Join(repo, ".abcd", "development")) {
		t.Error("the missing durable-record tier was not created")
	}
}

// TestInstallRefusesATierThatIsAFile: a regular file standing at a tier's path
// is refused with a note, never replaced.
func TestInstallRefusesATierThatIsAFile(t *testing.T) {
	setupHermetic(t)
	harnessFixture(t, "")
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	stand := filepath.Join(repo, ".abcd", "work")
	if err := os.WriteFile(stand, []byte("not a tier\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(res.Notes, "\n"), ".abcd/work") {
		t.Errorf("no refusal note for the file standing at the work tier: %v", res.Notes)
	}
	if res.Status != "partial" {
		t.Errorf("status = %q with a tier refused, want partial", res.Status)
	}
	if got, err := os.ReadFile(stand); err != nil || string(got) != "not a tier\n" {
		t.Errorf("the file at the tier's path was changed: %q (err=%v)", got, err)
	}
}
