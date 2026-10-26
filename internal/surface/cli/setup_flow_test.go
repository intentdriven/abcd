package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/interview"
	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/surface/cli/ask"
)

// TestOldApprovalIdIsRefusedNamingThePlan: the plan screen answers every kind
// of change at once, so an answers file naming an approval of one kind by its
// old id is refused before anything is written, naming approve.plan.
func TestOldApprovalIdIsRefusedNamingThePlan(t *testing.T) {
	run := newSetupRun(t)
	before := listTree(t, run.repo)
	out, err := pipedSetup(t, answersFileWith(t, interview.FileAnswer{ID: "approve.config-change", Value: "yes"}))
	var ee *exitError
	if !errors.As(err, &ee) || ee.Code != 2 {
		t.Fatalf("an old approval id: %v, want exit 2\n%s", err, out)
	}
	for _, want := range []string{"approve.config-change", ahoy.PlanQuestionID, "nothing was written"} {
		if !strings.Contains(ee.Msg, want) {
			t.Errorf("the refusal %q does not name %q", ee.Msg, want)
		}
	}
	if after := listTree(t, run.repo); after != before {
		t.Fatal("the refused run wrote into the repository")
	}
}

// TestQuestionMarkDrawsTheWholeQuestion is criterion 6 at the terminal: each
// question is drawn on its one line, and ? draws the whole question, its
// answers' meanings included, and asks it again.
func TestQuestionMarkDrawsTheWholeQuestion(t *testing.T) {
	newSetupRun(t)
	t.Setenv("ABCD_ACCESSIBLE", "1")
	in, keys := pipeWith(t, "")
	stdout, stderr := tempStream(t, "stdout"), tempStream(t, "stderr")
	swapStream, swapPut := isTerminalStream, drawnPut
	t.Cleanup(func() { isTerminalStream, drawnPut = swapStream, swapPut })
	isTerminalStream = func(*os.File) bool { return true }
	explained := map[string]bool{}
	drawnPut = func(tm ask.Terminal, a question.Ask) ([]ask.Answer, error) {
		q := a.Questions[0]
		line := number(q, choose(q))
		if tm.Explain && !explained[q.ID] {
			explained[q.ID] = true
			line = "?"
		}
		if _, err := fmt.Fprintln(keys, line); err != nil {
			return nil, err
		}
		return tm.Put(a)
	}
	cmd := NewRootCommand()
	cmd.SetIn(in)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"ahoy", "install"})
	err := cmd.Execute()
	got, _ := os.ReadFile(stderr.Name())
	if err != nil {
		t.Fatalf("install: %v\n%s", err, got)
	}
	out := string(got)
	brief := strings.Index(out, "Proceed with this plan? "+ahoy.ExplainHint)
	whole := strings.Index(out, "Makes every change and takes every default on the plan")
	if brief < 0 || whole < 0 || whole < brief {
		t.Fatalf("the plan was not drawn on its one line and then whole after ?:\n%s", out)
	}
	if strings.Count(out, "Makes every change and takes every default on the plan") != 1 {
		t.Errorf("the meanings were drawn before ? was answered:\n%s", out)
	}
}
