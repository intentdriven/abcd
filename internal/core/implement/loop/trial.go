package loop

// trial.go is the build loop's trial mode (itd-2610161733316518 decision 4,
// spc-2610161949559050 step 3). A model-matrix copy runs the loop's stages in
// it: the run takes no reservation on the shared remote and no claim in the
// shared run, and its landing lands nothing. Copies of one intent are many
// runs of one record, so a reservation would make the second copy back off the
// first's footprint, and a copy that landed would contaminate every re-run of
// the feature (decision 3: no copy lands). The feature itself is built
// afterwards through the ordinary loop, and a trial run is never resumed as a
// build of its record nor a build as a trial.
//
// Inside a trial the route is the cell under test: its dispatch runs in the
// runners' cell mode (runner.Dispatcher.Cell), a runner's failure is the
// cell's (cellFailed), and no role of a trial is ever released to the host.
//
// A trial runs in its copy's own clone of the frozen commit
// (internal/core/matrix), never in the person's checkout: its start names the
// frozen commit, which must be the checkout's head, and every lane is cut
// from it, so no copy starts anywhere else and no copy's lanes share another
// copy's refs. A trial builds one spec step: a step that needs another would
// be cut from a commit that lacks it, since a trial lands nothing.

import (
	"fmt"
	"os"

	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// StageTrial is the run record's stage for the line a trial run's start writes.
const StageTrial = "trial"

// trialNote is the record's line at a trial run's start.
const trialNote = "a model-matrix trial: the run takes no reservation and lands nothing"

// trialLandNote is the landing's line for a trial lane.
const trialLandNote = "a model-matrix trial lands nothing: no push, no merge, and no record moved"

// VerdictCellFailed is the verdict a trial's lane is handed back with when the
// runner of the cell under test failed, refused or answered unparsably: the
// cell's failure, scored as such and run nowhere else.
const VerdictCellFailed = "cell-failed"

// cellFailed is a trial's answer to its routed runner's failure fl, the
// dispatch having run in the runners' cell mode. A provider down, or
// throttling it (a 429), before the copy began is an outage, not the model's
// failure (runner.Failure.ProviderDown): the role is paused for its own
// runner, which the next step retries on the same route, with no fallback
// receipt (pauseAwait). Any other failure is the cell's: the lane is
// handed back as VerdictCellFailed, naming the runner's reason and its first
// stderr line, and nothing retries it or hands it to the host. An await that
// is gone by now is named as ended, as pauseAwait names one.
func cellFailed(repoRoot, runID, laneID string, aw Await, session, asked string, fl runner.Failure, stderr string, handIn error, o Options) (StepResult, *RunnerPause, error) {
	fb := runner.FallbackReceipt{Role: aw.Role, Asked: asked, Reason: fl.Reason, Detail: fl.Detail, Spent: fl.Spent}
	if fl.ProviderDown() {
		return pauseAwait(repoRoot, runID, laneID, aw, session, fb, stderr, handIn, o)
	}
	var res StepResult
	err := mutate(repoRoot, runID, func(_ *os.Root, st *State) (bool, error) {
		now := o.now()
		i, _ := st.findAwait(repoRoot, aw.Receipt)
		if i < 0 || st.Lanes[i].ID != laneID {
			res = noteRunnerEnded(st, laneID, aw, fb, handIn, now)
			return true, nil
		}
		reason := fmt.Sprintf("the %s on the %s runner %s inside the cell (%s)", aw.Role, asked, fl.Reason, fsutil.RedactHome(fl.Detail))
		if stderr != "" {
			reason += ": " + stderr
		}
		lane := st.Lanes[i]
		handBackLane(st, &lane, HandBack{Verdict: VerdictCellFailed, Reason: reason}, "", now)
		st.Lanes[i] = lane
		st.UpdatedAt = now
		res = laneResult(*st, lane, "", nil)
		res.HandBack = lane.HandBack
		return true, nil
	})
	return res, nil, err
}

// trialFrozen checks a start's frozen commit: a trial names one, the full id
// of the checkout's head, and nothing else names one.
func trialFrozen(repoRoot string, o Options) error {
	if !o.Trial {
		if o.Frozen != "" {
			return refuse(StageTrial, "", "", "only a model-matrix trial names a frozen commit, and this start is a build",
				"start the build without one; nothing was written")
		}
		return nil
	}
	if !gitutil.IsFullSHA(o.Frozen) {
		return refuse(StageTrial, "", "", "a trial names the full id of its frozen commit, which every lane is cut from, and this one names "+
			quoteOrNone(shortSHA(o.Frozen)), "start the trial with its frozen commit's full id; nothing was written")
	}
	head, err := gitutil.Run(repoRoot, "rev-parse", "--verify", "--quiet", "HEAD^{commit}", "--")
	if err != nil || head != o.Frozen {
		return refuse(StageTrial, "", "", fmt.Sprintf("a trial runs in its copy's clone, checked out at its frozen commit %s, and this checkout's head is %s",
			shortSHA(o.Frozen), quoteOrNone(shortSHA(head))),
			"start the trial in the copy's clone of its frozen commit; nothing was written")
	}
	return nil
}

// trialSteps refuses a trial over a spec that leaves more than one step to
// build: it lands nothing, so a later step's lane would be cut from a commit
// without the earlier one.
func trialSteps(chk CheckResult) error {
	return refuse(StageTrial, "", "", fmt.Sprintf("a trial builds one step, and %s leaves %d to build: a trial lands nothing, so a later step would start without the earlier one",
		chk.Spec, len(chk.steps)), "trial a spec that leaves one step to build; nothing was written")
}

// trialBase is the commit a trial's lane is cut from: its frozen commit, as
// the run's state records it and the checkout holds it.
func trialBase(c Context, laneID string) (string, error) {
	frozen := c.State.Frozen
	if !gitutil.IsFullSHA(frozen) {
		return "", refuse(string(StageWorktree), "", laneID, "the trial records no frozen commit to cut the lane from",
			"a trial's start records it; restore the run's state file")
	}
	base, err := gitutil.Run(c.RepoRoot, "rev-parse", "--verify", "--quiet", frozen+"^{commit}", "--")
	if err != nil || base != frozen {
		return "", refuse(string(StageWorktree), "", laneID, "the trial's frozen commit "+shortSHA(frozen)+" is not a commit this checkout holds",
			"start the trial in the copy's clone of its frozen commit")
	}
	return base, nil
}
