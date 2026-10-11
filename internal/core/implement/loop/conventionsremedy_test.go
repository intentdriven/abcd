package loop

import (
	"strings"
	"testing"
)

// assertConventionsRemedy checks a brief refusal for a base without AGENTS.md
// names the command that writes one. The preparation workflow that merges the
// conventions into AGENTS.md runs in the host agent from the ahoy page's
// install section (spc-2610100613109045, step 4), so the remedy names
// /abcd:ahoy install. The retired prepare-this-repo page is named nowhere: its
// slash form is a page that no longer exists, and `abcd prepare-this-repo` was
// never a verb, so either sends the person to a command that refuses.
func assertConventionsRemedy(t *testing.T, err error) {
	t.Helper()
	r := mustRefusal(t, err)
	if r.Stage != string(StageBrief) || !strings.Contains(r.Reason, ConventionsFile) {
		t.Fatalf("want the missing %s refused at the brief: %+v", ConventionsFile, r)
	}
	if !strings.Contains(r.Remedy, "/abcd:ahoy install") {
		t.Errorf("the remedy names the plugin command /abcd:ahoy install: %q", r.Remedy)
	}
	if strings.Contains(r.Remedy, "prepare-this-repo") {
		t.Errorf("the remedy names the retired prepare-this-repo: %q", r.Remedy)
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
