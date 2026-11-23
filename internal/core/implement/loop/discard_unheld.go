package loop

// discard_unheld.go is what `implement step --discard` ends beyond a held lane
// (iss-2611231404319245): a lane at work whose run nothing drives, and a lane
// handed back for any reason, unachievable included. Before this, such a lane
// had no verb that ended it: it kept its worktree and branch, prune counted it
// held, and its run stayed in progress. Every discard saves the lane's commits
// not on the default branch to a bundle in the run's directory before it
// removes anything (saveLaneBundle), and once no lane of the run is left at
// work or held it releases what the run still holds (releaseOnDiscard).

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// atWork reports whether l is at one of the lane stages a step performs:
// worktree, brief, implement, validate or land. An auto-plan lane's stages are
// not among them.
func atWork(l Lane) bool {
	if l.AutoPlan != nil {
		return false
	}
	for _, s := range Sequence {
		if l.Stage == s {
			return true
		}
	}
	return false
}

// unheldEnd reports whether --discard may take l beyond the held lane and the
// lanes decidable names: a lane at work, or a lane handed back for any reason.
// Whether its run lets it go is judged under the lock (mayEndUnheld).
func unheldEnd(l Lane) bool {
	return atWork(l) || (l.Stage == StageHandedBack && l.HandBack != nil)
}

// unheldOnly reports whether l is one --discard takes by unheldEnd alone: no
// other rule of decidable admits it, so mayEndUnheld must.
func unheldOnly(repoRoot string, st State, l Lane) bool {
	return unheldEnd(l) && !stuckLane(l) && !driveStopped(st.RunID) && !letGoAtLand(repoRoot, st, l)
}

// mayEndUnheld judges, under the run's lock, a lane unheldOnly admits. It is
// refused while the run is live (runLive); a handed-back lane also while a
// sibling is still at work, since the run holds its siblings only while a
// lane stands handed back; and while the lane's pull request is open or
// merged (unheldPR). closed is set when the forge reports that pull request
// closed, so the discard closes nothing.
func mayEndUnheld(c Context, root *os.Root, lane Lane, g *outageGate, o Options) (closed bool, err error) {
	if err := runLive(c.RepoRoot, root, c.State, lane.ID, o, c.Now); err != nil {
		return false, err
	}
	if lane.Stage == StageHandedBack {
		if busy := busyLanes(c.State); len(busy) > 0 {
			return false, contend(string(StageHeld), "", lane.ID, "lanes of "+c.State.RunID+" are still at work: "+strings.Join(busy, "; ")+
				"; while "+lane.ID+" stands handed back their passing rounds are held rather than landed",
				"run `abcd implement step` until they finish or are held, or discard them first; nothing was changed")
		}
	}
	return unheldPR(c, lane, g)
}

// runLive refuses, as contention, a run something still drives: a recorded
// driver that is alive and is not the session this call runs under, or an
// agent of any lane still at work that is running or may still be in its call
// (judgeAgent). The caller's own session is judged as no driver, so its own
// agents go by their runner claims and the lease. A process table that cannot
// be read is refused too: nothing then says the run is not live. A run a stop
// holds, or halted and no take-over has cleared, is refused as well: the stop
// settles its lanes, and its take-over is how the run moves on.
func runLive(repoRoot string, root *os.Root, st State, laneID string, o Options, now time.Time) error {
	again := "run `abcd implement step --run " + st.RunID + " --discard " + laneID + "` again"
	tab, err := readProcs()
	if err != nil {
		return contend(string(StageHeld), "", laneID, "the process table cannot be read, so whether anything still drives "+st.RunID+" cannot be told",
			again+" once it can be read; nothing was changed")
	}
	c, err := readControl(root, st.RunID)
	if err != nil {
		return err
	}
	if c.Stop != nil || o.stopRead(c.Shared).Stop != nil {
		return refuse(string(StageHeld), "", laneID, "a stop holds "+st.RunID+" or halted it, and its lanes are the stop's to settle",
			"once the stop is lifted, take the run over with `abcd implement resume --take "+st.RunID+"`, then "+again+"; nothing was changed")
	}
	if d := c.Driver; d != nil && tab.Alive(d.host()) {
		if here, _, herr := o.driverHere(tab); herr != nil || here.PID != d.PID {
			return contend(string(StageHeld), "", laneID, fmt.Sprintf("%s is live: it is driven by %s, which is alive", st.RunID, driverName(*d)),
				"back off: only a run nothing drives has its lanes ended this way; once that driver is gone, "+again+"; nothing was changed")
		}
		c.Driver = nil
	}
	procs := ledgerOf(o, runner.LedgerQuery{Run: st.RunID})
	var live []string
	for _, l := range st.Lanes {
		if !atWork(l) {
			continue
		}
		for _, a := range l.Awaits {
			switch judgeAgent(procs, repoRoot, st.RunID, l.ID, a, c, &tab, now) {
			case AgentAlive:
				live = append(live, l.ID+"'s "+a.Role+" is running")
			case AgentUnknown:
				live = append(live, l.ID+"'s "+a.Role+" may still be in its call")
			}
		}
	}
	if len(live) > 0 {
		return contend(string(StageHeld), "", laneID, st.RunID+" is live: "+strings.Join(live, "; "),
			"let the agent finish and hand its receipt in; an agent that is gone counts as gone once its runner's claim ends or, for a host's agent, a reservation lease after it was handed out, and then "+
				again+"; nothing was changed")
	}
	return nil
}

// unheldPR asks the forge what became of the lane's pull request: one open,
// armed or not, is refused, since its work can still merge; one merged has
// landed, which a step records; one closed is reported as such. A lane with
// none, or handed back as closed, asks nothing. While the network is down the
// read waits on the shared probe.
func unheldPR(c Context, lane Lane, g *outageGate) (closed bool, err error) {
	if lane.PR <= 0 {
		return false, nil
	}
	if lane.HandBack != nil && lane.HandBack.Kind == HandBackClosed {
		return true, nil
	}
	n := strconv.Itoa(lane.PR)
	if g.down(implement.ServiceNetwork) {
		return false, g.networkWait(string(StageHeld), lane.ID, fmt.Sprintf("the discard of %s (reading pull request #%s)", lane.ID, n))
	}
	out, err := forge(c, lane, "pr", "view", n, "--json", "state", "--jq", ".state")
	if err != nil {
		if r, ok := AsRefusal(err); ok {
			g.take(r)
		}
		return false, err
	}
	again := "`abcd implement step --run " + c.State.RunID + " --discard " + lane.ID + "`"
	switch strings.TrimSpace(out) {
	case "CLOSED":
		return true, nil
	case "MERGED":
		return false, refuse(string(StageHeld), "", lane.ID, fmt.Sprintf("%s's pull request #%s has merged, so its work has landed and is not the discard's to end", lane.ID, n),
			"run `abcd implement step --run "+c.State.RunID+"`, which records the landing; nothing was changed")
	}
	armed := ""
	if lane.Landing != nil && lane.Landing.Armed {
		armed = " and armed to merge"
	}
	return false, refuse(string(StageHeld), "", lane.ID, fmt.Sprintf("%s's pull request #%s is open%s, so its work can still merge", lane.ID, n, armed),
		"close it on the forge (`gh pr close "+n+"`), or let the run land it; then run "+again+" again; nothing was changed")
}

// saveLaneBundle saves the commits of the lane's branch that the default
// branch does not hold (origin's, as gitutil.DefaultRef resolves it; the whole
// branch where none resolves) to a git bundle in the run's own directory,
// named for the lane and its tip, before a discard removes the branch. It
// returns the bundle's path relative to the checkout and how many commits it
// holds; "" and 0 when the lane names no branch, the branch is gone, or it
// holds no commit of its own. History is read as stored, so no graft or
// replace ref hides a commit from the bundle.
func saveLaneBundle(repoRoot, runID string, lane Lane) (rel string, n int, err error) {
	if lane.Branch == "" {
		return "", 0, nil
	}
	ref := "refs/heads/" + lane.Branch
	tip, err := gitutil.RunCappedAsStored(repoRoot, maxGitOutput, "rev-parse", "--verify", "--quiet", ref+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(tip) {
		return "", 0, nil
	}
	revs := []string{ref}
	if def := gitutil.DefaultRef(repoRoot); def != "" && def != ref {
		revs = append(revs, "^"+def)
	}
	count, err := gitutil.RunCappedAsStored(repoRoot, 64, append(append([]string{"rev-list", "--count"}, revs...), "--")...)
	if err != nil {
		return "", 0, fmt.Errorf("counting %s's commits not on the default branch: %w", lane.Branch, err)
	}
	if n, err = strconv.Atoi(strings.TrimSpace(count)); err != nil || n == 0 {
		return "", 0, err
	}
	name := "lane"
	if ValidLaneID(lane.ID) {
		name = lane.ID
	}
	rel = runRel(runID) + "/" + name + "-" + tip[:12] + ".bundle"
	abs := filepath.Join(repoRoot, filepath.FromSlash(rel))
	if _, err := gitutil.RunCappedAsStored(repoRoot, maxGitOutput, append([]string{"bundle", "create", "-q", abs}, revs...)...); err != nil {
		return "", 0, fmt.Errorf("saving %s's commits to %s: %w", lane.Branch, rel, err)
	}
	return rel, n, nil
}

// bundleNote is the discard record's clause for a saved bundle.
func bundleNote(rel string, n int, branch string) string {
	return fmt.Sprintf("saved its %d commit(s) not on the default branch to %s (`git fetch %s %s` restores them)", n, rel, rel, branch)
}

// releaseOnDiscard registers, on the state a discard's callback has just
// changed, the release of every reservation the run still holds once that
// state is written, under the same lock (State.afterWritten), when no lane of
// the run is left at work or held: nothing of the run will land, so the
// record is no longer the run's to hold. A reservation the record already
// shows released since it was taken is not released again. An issue run's
// release publishes the discarded lane's hand-back, where it carries one, as
// releaseOnHandBack would have. Each release is written to the run record, and
// the run's reserved_at cleared, by a second write under the same hold, as a
// stop's release is. *said is what the releases came to, "" for none.
func releaseOnDiscard(repoRoot string, st *State, laneID string, now time.Time, said *string) {
	st.afterWritten(func(s *State, rewrite func() error) {
		kind := ""
		for _, l := range s.Lanes {
			switch l.Stage {
			case StageDone, StageDiscarded, StageHandedBack:
			default:
				return
			}
			if l.ID == laneID && l.HandBack != nil && s.Issue() != "" {
				kind = tombstoneKind(*l.HandBack)
			}
		}
		var out []string
		for _, id := range s.reservedIssues() {
			if releasedThisHoldFor(*s, id) {
				continue
			}
			clause, failed := releaseReservation(repoRoot, id, kind, now)
			stage := StageReservationReleased
			if failed {
				tail, _, _ := lapseTail(repoRoot, s.forIssue(id), now)
				clause += tail
				stage = StageReservationReleaseFailed
			}
			s.Record = append(s.Record, Entry{At: now, Stage: stage, Note: clause, Issue: s.issueOn(id)})
			out = append(out, clause)
		}
		if len(out) == 0 {
			return
		}
		s.ReservedAt = nil
		for i := range s.Bundle {
			s.Bundle[i].ReservedAt = nil
		}
		s.UpdatedAt = now
		*said = strings.Join(out, "; ")
		if err := rewrite(); err != nil {
			*said += "; the run record could not take this: " + err.Error()
		}
	})
}
