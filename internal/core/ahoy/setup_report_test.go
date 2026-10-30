package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReportHasThreeParts is criterion 7: the report ends in three parts:
// done, one line per change with where it landed; needs you, a checklist of
// exact steps each with why; and one next step.
func TestReportHasThreeParts(t *testing.T) {
	setupHermetic(t)
	clearIdentityEnv(t)
	repo := idGitRepo(t, "Alice", "alice@example.com")
	res, err := Install(repo, planOpts(), &planPrompter{choice: PlanProceed})
	if err != nil {
		t.Fatal(err)
	}
	rep := res.Report
	if len(rep.Done) == 0 {
		t.Fatalf("a run that wrote %d files reports nothing done", len(res.Writes))
	}
	for _, line := range rep.Done {
		label, where, ok := strings.Cut(line, ": ")
		if !ok || label == "" || where == "" {
			t.Errorf("done line %q is not a change with where it landed", line)
		}
	}
	for i, s := range rep.NeedsYou {
		if s.Step == "" || s.Why == "" {
			t.Errorf("needs-you item %d lacks its step or why: %+v", i+1, s)
		}
	}
	if rep.Next == "" || strings.Contains(rep.Next, "\n") {
		t.Errorf("next = %q, want one step", rep.Next)
	}
	if rep.OneLine != "" {
		t.Errorf("a run that changed things reported one line: %q", rep.OneLine)
	}
}

// TestFullySetUpRerunIsOneLine is criterion 8: run again on a repository that
// is fully set up, setup asks nothing, changes nothing and reports one line.
func TestFullySetUpRerunIsOneLine(t *testing.T) {
	setupHermetic(t)
	// The PATH entry's folder is on PATH, as a set-up account's is.
	t.Setenv("PATH", filepath.Dir(os.Getenv("ABCD_BIN_TARGET"))+string(os.PathListSeparator)+os.Getenv("PATH"))
	repo := idGitRepo(t, "Alice", "alice@example.com")
	// A first commit gives the repository the root commit abcd registers it by.
	idMustGit(t, repo, "commit", "--allow-empty", "-q", "-m", "first")
	clearIdentityEnv(t)
	// Fully set up includes the commit checks git runs (iss-2610241134315056)
	// and the person's own identity record, each answered yes, and the live
	// brief session question answered, since a question left for later is
	// asked again.
	first := func(q string) bool { return isArmHooksQuestion(q) || setupConfirmID(q) == UnrecordedGapID }
	answered := planOpts()
	answered.LiveSessions = LiveSessionsNone
	if _, err := Install(repo, answered, &planPrompter{choice: PlanProceed, confirm: first}); err != nil {
		t.Fatal(err)
	}
	again := &planPrompter{choice: PlanProceed}
	res, err := Install(repo, planOpts(), again)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "already_up_to_date" || len(res.Writes) != 0 {
		t.Fatalf("rerun status %q, writes %v, notes %q, optional %v", res.Status, res.Writes, res.Notes, res.OptionalSkipped)
	}
	if len(again.asked) != 0 {
		t.Errorf("the rerun asked %q", again.asked)
	}
	if res.Report.OneLine != UpToDateLine {
		t.Errorf("one line = %q, needs you %+v", res.Report.OneLine, res.Report.NeedsYou)
	}
}
