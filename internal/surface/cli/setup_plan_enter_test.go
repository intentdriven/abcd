package cli

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
)

// TestPlanScreenBareEnterAsksAgain: at the stdin door's plan screen a bare
// Enter is no answer, as it is at every [y/N] question, so the screen asks
// again rather than proceeding; end of input cancels.
func TestPlanScreenBareEnterAsksAgain(t *testing.T) {
	var w bytes.Buffer
	p := &stdinPrompter{r: bufio.NewReader(strings.NewReader("\nproceed\n")), w: &w, tty: true}
	c, asked := p.Plan(ahoy.SetupPlan{})
	if !asked || c != ahoy.PlanProceed {
		t.Fatalf("choice %q asked %v", c, asked)
	}
	if n := strings.Count(w.String(), "[proceed/customise/cancel]"); n != 2 {
		t.Fatalf("a bare Enter was taken as an answer (asked %d times):\n%s", n, w.String())
	}
	w.Reset()
	p = &stdinPrompter{r: bufio.NewReader(strings.NewReader("\n")), w: &w, tty: true}
	if c, _ := p.Plan(ahoy.SetupPlan{}); c != ahoy.PlanCancel {
		t.Fatalf("a bare Enter then end of input took %q, want cancel", c)
	}
}

// TestPlanScreenQuestionMarkListsEveryItem (iss-2611152326582649): the stdin
// door's plan screen shows the plan in counts, and ? lists every item it
// counts, each default with where to change it, before asking again.
func TestPlanScreenQuestionMarkListsEveryItem(t *testing.T) {
	plan := ahoy.SetupPlan{
		Changes:  []ahoy.PlanGroup{{Where: "In this repository", Items: []string{"write the config", "add the hooks"}}},
		Defaults: []ahoy.PlanDefault{{Key: "scan_deep", What: "the deep secret scan", Value: "off", ChangeWith: "abcd ahoy install --scan-deep true"}},
		AsksYou:  []string{"the commit identity"},
	}
	var w bytes.Buffer
	p := &stdinPrompter{r: bufio.NewReader(strings.NewReader("?\nproceed\n")), w: &w, tty: true}
	if c, asked := p.Plan(plan); !asked || c != ahoy.PlanProceed {
		t.Fatalf("choice %q asked %v", c, asked)
	}
	out := w.String()
	ask := strings.Index(out, "[proceed/customise/cancel]")
	if counts := strings.Index(out, "Setup makes 2 changes in this repository, takes 1 default"); counts < 0 || counts > ask {
		t.Fatalf("the plan was not shown in counts before its one line:\n%s", out)
	}
	if strings.Index(out, planInFullHeading) < ask {
		t.Fatalf("the full list was shown before ? was answered:\n%s", out)
	}
	for _, item := range []string{"write the config", "add the hooks", "the deep secret scan: off (change: abcd ahoy install --scan-deep true)", "the commit identity"} {
		if !strings.Contains(out, "  - "+item+"\n") {
			t.Errorf("? did not list %q:\n%s", item, out)
		}
	}
	if n := strings.Count(out, "[proceed/customise/cancel]"); n != 2 {
		t.Errorf("? was not followed by the question again (asked %d times):\n%s", n, out)
	}
}
