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
