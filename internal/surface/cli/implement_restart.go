package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// runRestart is `implement step --restart <lane-id> [--yielded <line>]`: the
// lane's gone implementer is restarted as a fresh agent from the lane's last
// commit, its uncommitted work saved aside (iss-2610080620372731).
func runRestart(w io.Writer, asJSON bool, root, runID, laneID, yielded string) error {
	const prefix = "abcd implement step"
	res, err := loop.Restart(root, runID, laneID, yielded, loop.Options{})
	if err != nil {
		return loopFail(w, asJSON, prefix, err)
	}
	res.StepResult = redactStep(res.StepResult)
	return render(w, asJSON, res, func(w io.Writer) { renderRestart(w, res) })
}

// renderRestart is the restart's text form: what was saved aside and where,
// then the step's own lines, which name the fresh agent to start.
func renderRestart(w io.Writer, res loop.RestartResult) {
	a := res.Aside
	saved := "nothing was left uncommitted"
	if n := len(a.Files); n > 0 {
		saved = fmt.Sprintf("%d file(s) left uncommitted saved aside", n)
	}
	if a.Receipt != "" {
		saved += ", with the partial receipt"
	}
	fmt.Fprintf(w, "restart: %s of %s from its last commit %s (%s); %s\n",
		res.Lane, res.RunID, shortSHA(a.Head), termsafe.Sanitize(a.Why), saved)
	fmt.Fprintf(w, "aside: %s (for the product thinker's review; the fresh agent is not told of it)\n",
		termsafe.Sanitize(fsutil.RedactHome(a.Path)))
	if len(a.Files) > 0 {
		fmt.Fprintf(w, "  files: %s\n", termsafe.Sanitize(strings.Join(a.Files, ", ")))
	}
	renderStepResult(w, "step", res.StepResult)
}
