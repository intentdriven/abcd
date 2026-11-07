package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// closeNamingIssueRepo lays a planned intent with one open spec, naming iss-5,
// which sits in the ledger folder status names.
func closeNamingIssueRepo(t *testing.T, status string) string {
	t.Helper()
	repo := t.TempDir()
	gitInitAt(t, repo)
	t.Chdir(repo)
	writeRepoFile(t, repo, cliPlanned+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\nimpact: fix\nrelated_issues: [iss-5]\n---\n# alpha\n\n## Acceptance Criteria\n\n- ok\n")
	writeRepoFile(t, repo, cliSpecsOpen+"/spc-1-alpha.md",
		"---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# alpha\n")
	writeRepoFile(t, repo, ".abcd/work/issues/"+status+"/iss-5-five.md",
		driftIssueText("iss-5", "five", "related_intents: [itd-10]\n"))
	return repo
}

// TestSpecCloseRefusesShippingAnIntentNamingAnOpenIssue is iss-2611060603312990:
// a close that would ship an intent whose related_issues names an open issue is
// refused before anything moves, with the open issue and the ways out named,
// rather than left for issue-drift at preflight.
func TestSpecCloseRefusesShippingAnIntentNamingAnOpenIssue(t *testing.T) {
	repo := closeNamingIssueRepo(t, "open")
	_, err := runCLIErr(t, "spec", "close", "spc-1")
	if code := exitCodeOf(err); code != 1 {
		t.Fatalf("exit = %d, want 1 (%v)", code, err)
	}
	for _, want := range []string{"itd-10", "iss-5", "open/", "nothing moved", "capture resolve iss-5", "related_issues", "--remainder"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q:\n%s", want, err)
		}
	}
	for _, rel := range []string{cliSpecsOpen + "/spc-1-alpha.md", cliPlanned + "/itd-10-alpha.md"} {
		if _, statErr := os.Stat(filepath.Join(repo, rel)); statErr != nil {
			t.Errorf("%s moved on a refused close: %v", rel, statErr)
		}
	}
}

// TestSpecCloseRefusesAWontfixLinkAsTheDriftGateDoes: issue-drift's finding is
// "not in resolved/", so a declined issue refuses too, and resolving it is not
// offered as a way out, since `capture resolve` moves an open issue only.
func TestSpecCloseRefusesAWontfixLinkAsTheDriftGateDoes(t *testing.T) {
	closeNamingIssueRepo(t, "wontfix")
	_, err := runCLIErr(t, "spec", "close", "spc-1")
	if code := exitCodeOf(err); code != 1 {
		t.Fatalf("exit = %d, want 1 (%v)", code, err)
	}
	if !strings.Contains(err.Error(), "wontfix/") || strings.Contains(err.Error(), "capture resolve") {
		t.Fatalf("the refusal must name wontfix/ and offer no resolve:\n%s", err)
	}
}

// A resolved issue is what the gate wants, so the close goes through.
func TestSpecCloseShipsAnIntentNamingAResolvedIssue(t *testing.T) {
	repo := closeNamingIssueRepo(t, "resolved")
	runCLI(t, "spec", "close", "spc-1")
	if _, err := os.Stat(filepath.Join(repo, ".abcd/development/intents/shipped/itd-10-alpha.md")); err != nil {
		t.Fatalf("the intent did not ship: %v", err)
	}
}

// A --remainder close ships nothing, so an open link is no reason to refuse it.
func TestSpecCloseWithARemainderIgnoresAnOpenLink(t *testing.T) {
	repo := closeNamingIssueRepo(t, "open")
	runCLI(t, "spec", "close", "spc-1", "--remainder", "the-rest", "--production-mode", "dictated-and-formatted")
	if _, err := os.Stat(filepath.Join(repo, cliPlanned, "itd-10-alpha.md")); err != nil {
		t.Fatalf("the intent must stay planned: %v", err)
	}
}

// A close of one spec while another still names the intent ships nothing either.
func TestSpecCloseLeavingAnotherOpenSpecIgnoresAnOpenLink(t *testing.T) {
	repo := closeNamingIssueRepo(t, "open")
	writeRepoFile(t, repo, cliSpecsOpen+"/spc-2-beta.md", "---\nid: spc-2\nslug: beta\nintent: itd-10\n---\n# beta\n")
	runCLI(t, "spec", "close", "spc-1")
	if _, err := os.Stat(filepath.Join(repo, cliPlanned, "itd-10-alpha.md")); err != nil {
		t.Fatalf("the intent must stay planned: %v", err)
	}
}
