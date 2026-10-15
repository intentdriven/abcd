package ahoy

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/decisions"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestInstallCreatesTheCommittedTiers pins iss-2610071538028804: every tier
// the repository lint's three-tier-layout rule requires is a gap on a repo
// without it, install creates each as a real directory and seeds the shared
// tier's decisions folder, and nothing is left for detection to report.
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
	folderReadme := decisions.RelDir + "/README.md"
	if !fileExists(filepath.Join(repo, filepath.FromSlash(folderReadme))) {
		t.Fatalf("the decisions folder was not seeded")
	}
	if !strings.Contains(writes, folderReadme) {
		t.Errorf("the decisions folder's README is not on the receipt: %v", res.Writes)
	}

	readme, err := os.ReadFile(filepath.Join(repo, ".abcd", "development", "README.md"))
	if err != nil {
		t.Fatalf("the durable-record tier was not seeded with its README: %v", err)
	}
	if !strings.Contains(writes, ".abcd/development/README.md") {
		t.Errorf("the README is not on the receipt: %v", res.Writes)
	}
	if regexp.MustCompile(`(?m)^\s*[-*] `).Match(readme) {
		t.Errorf("the seeded README carries a list:\n%s", readme)
	}

	after, _ := Detect(repo)
	for _, s := range tierSeeds() {
		if hasGap(after.Gaps, s.gapID) {
			t.Errorf("the %s gap persists after install", s.gapID)
		}
	}
	for _, tier := range lint.CommittedTiers() {
		if hasGap(after.Gaps, committedTierGapID(tier)) {
			t.Errorf("the %s gap persists after install", tier.Rel)
		}
	}
	if hasGap(after.Gaps, decisionsFolderGapID) {
		t.Error("the decisions-folder gap persists after install")
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

// TestEveryCommittedTierHasASeed: git keeps no empty folder, so a tier install
// creates without a file in it is gone from the next clone. Every tier on the
// lint's list therefore carries a seed file, and a tier added to the list
// without one fails here.
func TestEveryCommittedTierHasASeed(t *testing.T) {
	seeded := map[string]bool{}
	for _, s := range tierSeeds() {
		seeded[s.tier] = true
	}
	for _, tier := range lint.CommittedTiers() {
		if !seeded[tier.Rel] {
			t.Errorf("%s has no seed file, so a clone of a fresh install loses it", tier.Rel)
		}
	}
}

// TestInstallKeepsAnExistingDevelopmentReadme: a README the durable-record
// tier already holds is never overwritten, and is not reported missing.
func TestInstallKeepsAnExistingDevelopmentReadme(t *testing.T) {
	setupHermetic(t)
	harnessFixture(t, "")
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	const own = "# Our record\n"
	p := filepath.Join(repo, ".abcd", "development", "README.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(own), 0o644); err != nil {
		t.Fatal(err)
	}
	det, _ := Detect(repo)
	if hasGap(det.Gaps, developmentReadmeGapID) {
		t.Error("a README that exists is reported missing")
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(p); err != nil || string(got) != own {
		t.Errorf("the existing README was changed: %q (err=%v)", got, err)
	}
}

// TestDevelopmentReadmeGapInAnExistingTier: a durable-record tier with no
// README is a gap, and install seeds the README beside what the tier holds.
func TestDevelopmentReadmeGapInAnExistingTier(t *testing.T) {
	setupHermetic(t)
	harnessFixture(t, "")
	repo := t.TempDir()
	for _, d := range []string{".git", ".abcd/development", ".abcd/work"} {
		if err := os.MkdirAll(filepath.Join(repo, filepath.FromSlash(d)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	det, _ := Detect(repo)
	if !hasGap(det.Gaps, developmentReadmeGapID) {
		t.Fatalf("no %s gap for a durable-record tier without a README: %v", developmentReadmeGapID, gapIDs(det.Gaps))
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	if !fileExists(filepath.Join(repo, ".abcd", "development", "README.md")) {
		t.Fatal("the README was not seeded into the existing durable-record tier")
	}
}

// TestInstallSeedsTheDecisionsFolder is A6 of spc-2610110931381316: adoption
// with no decision log gets the decisions folder, seeded with a README so git
// keeps it, and no single log. The folder's absence is a gap of its own in a
// shared tier that already exists, and install closes it.
func TestInstallSeedsTheDecisionsFolder(t *testing.T) {
	setupHermetic(t)
	harnessFixture(t, "")
	repo := t.TempDir()
	for _, d := range []string{".git", ".abcd/development", ".abcd/work"} {
		if err := os.MkdirAll(filepath.Join(repo, filepath.FromSlash(d)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	const gapID = decisionsFolderGapID
	det, _ := Detect(repo)
	if !hasGap(det.Gaps, gapID) {
		t.Fatalf("no %s gap for a shared tier without the decisions folder: %v", gapID, gapIDs(det.Gaps))
	}
	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	readmeRel := decisions.RelDir + "/README.md"
	readme, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(readmeRel)))
	if err != nil {
		t.Fatalf("the decisions folder was not seeded with its README: %v", err)
	}
	if !strings.Contains(strings.Join(res.Writes, "\n"), readmeRel) {
		t.Errorf("the README is not on the receipt: %v", res.Writes)
	}
	// The seed says what the folder holds and how a decision is recorded, and
	// is no decision itself.
	if !strings.Contains(string(readme), `abcd decide "`) {
		t.Errorf("the seeded README does not name the verb that records a decision:\n%s", readme)
	}
	if ds, skipped, err := decisions.List(repo); err != nil || len(ds) != 0 || len(skipped) != 0 {
		t.Errorf("the seeded folder reads as decisions %v, skipped %v, err %v; want none", ds, skipped, err)
	}
	if _, err := os.Lstat(filepath.Join(repo, ".abcd", "work", "DECISIONS.md")); !os.IsNotExist(err) {
		t.Errorf("install laid a single decisions log: %v", err)
	}
	after, _ := Detect(repo)
	if hasGap(after.Gaps, gapID) {
		t.Errorf("the %s gap persists after install", gapID)
	}
}

// TestInstallLeavesAFrozenLogByteForByte is A5 of spc-2610110931381316: a
// managed repository's existing DECISIONS.md is never opened for writing by
// install, compared by hash, while the folder new decisions go to is seeded
// beside it.
func TestInstallLeavesAFrozenLogByteForByte(t *testing.T) {
	setupHermetic(t)
	harnessFixture(t, "")
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	// No trailing newline and a CRLF line on purpose: a writer that normalised
	// the file in passing would change its hash.
	frozen := []byte("# our decisions\r\n\n- 2026-01-01 — keep it.\n- 2026-02-02 — and this")
	p := filepath.Join(repo, ".abcd", "work", "DECISIONS.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, frozen, 0o644); err != nil {
		t.Fatal(err)
	}
	before := sha256.Sum256(frozen)
	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(got) != before {
		t.Errorf("install changed the frozen log:\n%q", got)
	}
	if strings.Contains(strings.Join(res.Writes, "\n"), "DECISIONS.md") {
		t.Errorf("the receipt claims a write of the frozen log: %v", res.Writes)
	}
	if !fileExists(filepath.Join(repo, filepath.FromSlash(decisions.RelDir), "README.md")) {
		t.Error("the decisions folder was not seeded beside the frozen log")
	}
	if !fsutil.IsRealDir(filepath.Join(repo, ".abcd", "development")) {
		t.Error("the missing durable-record tier was not created")
	}
}

// TestInstallScaffoldsNoMergeAttribute is A5's other half: install lays the
// decisions folder and gives it no merge attribute and no driver, because
// recording a decision adds a file and git has nothing to reconcile. The one
// attribute install writes is the hooks' EOL pin.
func TestInstallScaffoldsNoMergeAttribute(t *testing.T) {
	setupHermetic(t)
	harnessFixture(t, "")
	repo := gittest.NewRepo(t)
	if _, err := Install(repo.Root(), installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	if !fsutil.IsRealDir(filepath.Join(repo.Root(), filepath.FromSlash(decisions.RelDir))) {
		t.Fatalf("install did not lay %s/", decisions.RelDir)
	}
	attrs, err := os.ReadFile(filepath.Join(repo.Root(), ".gitattributes"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(attrs), "\n") {
		if strings.Contains(line, "merge=") {
			t.Errorf(".gitattributes carries a merge attribute: %q", line)
		}
	}
	for _, rel := range []string{
		decisions.RelDir + "/dcn-2610150000001234-a-decision.md",
		decisions.RelDir + "/README.md",
		decisions.FrozenLog,
	} {
		if got := repo.Git("check-attr", "merge", "--", rel); !strings.HasSuffix(got, ": merge: unspecified") {
			t.Errorf("git check-attr merge %s = %q, want unspecified", rel, got)
		}
	}
}
