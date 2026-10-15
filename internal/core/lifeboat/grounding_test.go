package lifeboat

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// gitFixtureWithRevert builds a git-only repo whose history contains a reverted
// commit and a deleted file — the raw material the Tier-0 graveyard adapter
// reads. No README, no .abcd.
func gitFixtureWithRevert(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	repo := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(),
			"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(repo, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("init", "-q")
	run("commit", "-q", "--allow-empty", "-m", "root")
	write("feature.txt", "an experiment\n")
	run("add", "-A")
	run("commit", "-q", "-m", "add feature")
	// Revert it: history now records the abandonment explicitly.
	run("revert", "--no-edit", "HEAD")
	// Delete a long-lived file after substantial history.
	write("legacy.txt", "old\n")
	run("add", "-A")
	run("commit", "-q", "-m", "add legacy")
	run("rm", "-q", "legacy.txt")
	run("commit", "-q", "-m", "drop legacy")
	return repo
}

// findSection returns the coverage row for a section, failing if it is absent.
func findSection(t *testing.T, cov Coverage, name Section) SectionCoverage {
	t.Helper()
	for _, s := range cov.Sections {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("section %s not in report", name)
	return SectionCoverage{}
}

// TestProbeGroundsRichRepoFromEveryTier is the M2 milestone behaviour: probed
// against this repository — which has git history, conventional files, AND a
// full abcd record — the coverage report is mostly filled, and specific
// sections are grounded from the tier that owns them. With the adapter stubs
// this fails (every section blank); it passes once the tiered adapters land.
func TestProbeGroundsRichRepoFromEveryTier(t *testing.T) {
	cov := probeThisRepo(t)

	if cov.Summary.Grounded < 5 {
		t.Errorf("rich repo grounded only %d sections; the record should ground many", cov.Summary.Grounded)
	}
	nonBlank := cov.Summary.Grounded + cov.Summary.Partial
	if nonBlank < 10 {
		t.Errorf("rich repo has only %d non-blank sections of %d", nonBlank, len(cov.Sections))
	}

	// docs/adrs is grounded from the native record (this repo keeps ADRs under
	// .abcd/development/decisions/adrs/).
	if adrs := findSection(t, cov, "docs/adrs"); adrs.Status != StatusGrounded {
		t.Errorf("docs/adrs = %s, want grounded from the native record", adrs.Status)
	}
	// activity/issues is grounded from the capture ledger under .abcd/work/issues.
	if iss := findSection(t, cov, "activity/issues"); iss.Status == StatusBlank {
		t.Errorf("activity/issues is blank; the capture ledger should ground it")
	}
	// product/context is at least partial from the README (a conventions tier).
	if ctx := findSection(t, cov, "product/context"); ctx.Status == StatusBlank {
		t.Errorf("product/context is blank; the README should ground it")
	}
	// NB: this test deliberately does NOT assert on git-history-derived sections
	// (graveyard, rescue/spine, build-sequence). CI's `check` job checks out at
	// the default shallow depth (a single commit), so the Tier-0 adapters see no
	// reverts, deletions, or tags on this repo — those sections degrade to blank
	// there through no fault of the probe. The git-tier grounding property is
	// proved deterministically against a full-history fixture in
	// TestProbeGraveyardFromGitAlone instead.

	// Every grounded/partial row must cite evidence — the anti-fiction rule.
	for _, s := range cov.Sections {
		if s.Status != StatusBlank && len(s.Evidence) == 0 {
			t.Errorf("section %s is %s but cites no evidence", s.Name, s.Status)
		}
	}
}

// TestProbeGraveyardFromGitAlone proves the Tier-0 thesis directly: a git-only
// repo with a reverted commit grounds the graveyard from history, with no record
// of any kind. This is the section that most distinguishes git from nothing.
func TestProbeGraveyardFromGitAlone(t *testing.T) {
	repo := gitFixtureWithRevert(t)
	cov, err := Probe(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(cov.TiersPresent) != 1 || cov.TiersPresent[0] != TierGit {
		t.Fatalf("fixture should be git-only, tiers = %v", cov.TiersPresent)
	}
	grave := findSection(t, cov, "graveyard")
	if grave.Status == StatusBlank {
		t.Errorf("graveyard blank despite a reverted commit in history")
	}
	if grave.Status != StatusBlank && grave.Tier != TierGit {
		t.Errorf("graveyard grounded at tier %s, want git", grave.Tier)
	}
	if grave.Status != StatusBlank && len(grave.Evidence) == 0 {
		t.Errorf("graveyard grounded but cites no evidence")
	}
}
