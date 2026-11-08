package matrix

// cell.go makes one copy inside its cell (decisions 4 and 6,
// spc-2610161949559050 step 3). The cell is the route under test: the copy is
// dispatched with the runner package's cell mode (runner.Dispatcher.Cell), so
// a runner that fails, refuses or answers unparsably is the cell's failure and
// nothing else runs the role. A provider that is down before the copy began is
// retried on the same pin, its attempts counted as not started and never
// scored. A clone bound for a provider passes the launch bundle's scan first.

import (
	"context"
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// outageTries is how many times a copy is tried on its pin while the provider
// is down before it is left not started.
const outageTries = 3

// copyResult is how one copy's making came out. A copy that was made (Made)
// carries no status yet: it is judged after, by its task's judging. Every other
// copy carries the status its trial record takes.
type copyResult struct {
	// Status is StatusFailed, StatusExcluded or StatusNotStarted for a copy
	// that was not made; "" for one that was.
	Status TrialStatus
	// Reason says why a copy was not made.
	Reason string
	// Outcome is the last dispatch's outcome: the answer, or the handoff to
	// the host session for a host cell, with the role's receipt.
	Outcome runner.Outcome
	// Fallbacks are the fallback receipts the copy carries, which exclude it.
	Fallbacks []runner.FallbackReceipt
	// NotStarted counts the tries the outage ended before the copy began.
	NotStarted int
}

// Made reports whether the copy was made: its runner answered, or a host
// cell's role was handed to the host session.
func (r copyResult) Made() bool { return r.Status == "" }

// makeCopy dispatches one try of a copy on pin, in the runner's cell mode.
type makeCopy func(ctx context.Context, pin CellPin) (runner.Outcome, error)

// runCopy makes one copy on pin, trying at most tries times while the
// provider is down and waiting wait(ctx, n) before try n+1. The pin is the
// same value on every try, since a fallback to another provider would change
// the cell (decision 6). A dispatch error is the run's, never the copy's
// score: a rate limit, or a route the configuration refuses. So is the end of
// a wait, which is how a stopped run ends a copy waiting out an outage.
func runCopy(ctx context.Context, pin CellPin, tries int, wait func(ctx context.Context, n int) error, make makeCopy) (copyResult, error) {
	var res copyResult
	for n := 1; ; n++ {
		out, err := make(ctx, pin)
		if err != nil {
			return copyResult{}, err
		}
		res.Outcome = out
		switch {
		case out.Fallback != nil:
			// A cell's dispatch writes none; one that carries a receipt never
			// counts as the model's work.
			res.Status, res.Fallbacks = StatusExcluded, []runner.FallbackReceipt{*out.Fallback}
			res.Reason = "a fallback ran the copy (" + string(out.Fallback.Reason) + " on " + out.Fallback.Asked +
				"); it counts as the cell's failure"
			return res, nil
		case out.Failure != nil && out.Failure.ProviderDown():
			res.NotStarted++
			if n >= tries {
				res.Status = StatusNotStarted
				res.Reason = fmt.Sprintf("the provider was down before the copy began, on all %d tries on the cell's pin", n)
				return res, nil
			}
			if err := wait(ctx, n); err != nil {
				return copyResult{}, err
			}
		case out.Failure != nil:
			res.Status = StatusFailed
			res.Reason = "runner " + out.Failure.Runner + " " + string(out.Failure.Reason) + " inside the cell: " + out.Failure.Detail
			return res, nil
		default:
			return res, nil
		}
	}
}

// admitClone passes a copy's clone through the launch bundle's scan when the
// cell sends it to a provider (an OpenRouter or other provider cell), and
// refuses the cell on anything that scan refuses, naming the cell and the
// scan's reasons, never a value. A local or host cell's clone is not sent
// anywhere, so it is not scanned. scanRoot is the checkout whose scanner
// configuration applies: the run's own.
func admitClone(scanRoot, clone string, pin CellPin) error {
	if !pin.Source.Metered() {
		return nil
	}
	refusals, err := launch.ScanTree(scanRoot, clone)
	if err != nil {
		return fmt.Errorf("cell %s is refused: its clone could not be scanned before it is sent to %s: %w", pin.Cell, pin.Source, err)
	}
	if len(refusals) > 0 {
		return fmt.Errorf("cell %s is refused: its clone fails the launch bundle's secret scan, so it is not sent to %s: %s",
			pin.Cell, pin.Source, strings.Join(refusals, "; "))
	}
	return nil
}
