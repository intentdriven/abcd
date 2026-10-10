package loop

import (
	"strings"
	"testing"
)

// assertConventionsRemedy checks a brief refusal for a base without AGENTS.md
// names the plugin command that writes one. prepare-this-repo has no binary
// verb, so a remedy telling the person to run `abcd prepare-this-repo` sends
// them to a command that exits 2 as unknown.
func assertConventionsRemedy(t *testing.T, err error) {
	t.Helper()
	r := mustRefusal(t, err)
	if r.Stage != string(StageBrief) || !strings.Contains(r.Reason, ConventionsFile) {
		t.Fatalf("want the missing %s refused at the brief: %+v", ConventionsFile, r)
	}
	if !strings.Contains(r.Remedy, "/abcd:prepare-this-repo") {
		t.Errorf("the remedy names the plugin command /abcd:prepare-this-repo: %q", r.Remedy)
	}
	if strings.Contains(r.Remedy, "`abcd prepare-this-repo`") {
		t.Errorf("the remedy names a binary verb that does not exist: %q", r.Remedy)
	}
}

func TestAnIntentBriefWithoutConventionsNamesThePluginCommand(t *testing.T) {
	repo := briefRepo(t, "")
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	advanceTo(t, repo, start.RunID, StageBrief)
	_, err = advance(repo.Root(), start.RunID, DefaultStages(), Options{})
	assertConventionsRemedy(t, err)
}

func TestAnIssueBriefWithoutConventionsNamesThePluginCommand(t *testing.T) {
	repo := issueRepo(t)
	repo.Git("rm", "-q", ConventionsFile)
	repo.Commit("no conventions")
	start, err := Start(repo.Root(), eligibleIssue, Options{})
	if err != nil {
		t.Fatal(err)
	}
	advanceTo(t, repo, start.RunID, StageBrief)
	_, err = advance(repo.Root(), start.RunID, DefaultStages(), Options{})
	assertConventionsRemedy(t, err)
}
