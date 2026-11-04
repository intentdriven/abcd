package loop

// receipt_standing.go keeps a routed implementer's finished work when the step
// that would have collected it was refused (iss-2611031337115764). A runner's
// agent is handed back by the step that started it, once the agent ends, so a
// refusal at that moment (a run state raised under it, a lock, a verifier's
// passing fault) left the agent finished, its commits on the branch and its
// receipt in place, and every later step asking for a fresh one: the work was
// lost, or paid for twice.
//
// So a later step reads that receipt, and verifies it exactly as `implement
// receipt` would, through the same hand-in under the same lock. It does so only
// for an implementer, whose verifier holds a receipt to the run, the lane, the
// branch and commits on it, and only where the run's own state shows a runner was
// started for exactly this await (startedRunner). That proof speaks for one
// attempt: whatever stands at an await's path is set aside unread as the await is
// handed out and as a runner's claim is taken (setAsideForAgent), so a file there
// came later. A fix round's receipt also needs a commit past the round's head,
// since its verifier asks for commits on the branch and not for new ones.
//
// Every other await keeps the path it always had: a reviewer's runner hands its
// return back when it ends, a host hands one back with `implement receipt`, and a
// hand-out starts its agent.
//
// A receipt that verifies is taken, over a runner's claim still standing if need
// be: a refused step could not drop its claim, and the claim names no process to
// ask. The lane takes the head the receipt verified at, so if the agent was in
// fact still working, its later commits move the branch past that head, which the
// landing's checks of a moved tip then meet. Its later hand-in is read
// (answerCollected): the same bytes are answered, and a changed file is refused.
//
// The route records what is known. The state names the runner that was asked; no
// report from it arrived, so what ran the agent is left empty until a report does.
//
// A receipt that does not verify is left where it stands. While the claim stands
// the agent may still be working, so the step says to wait. After a runner that
// ended and had its answer refused the lane is stopped, with no second runner over
// the work: the step's answer names the ways out alone, and each such step counts
// as a refused one, so a drain hands the lane back at the stuck bounds (stuck.go).
// After a runner that exited badly it may be unfinished anywhere, so the role is
// retried as ever, and the file is set aside as the retry's claim is taken.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/localtier"
)

// startedRunner is the runner the run's state shows was started for await a of
// lane l, and whether it shows one: a claim taken under the lock just before the
// launch, or a pause recording how that attempt ended: its answer refused, or its
// harness exiting badly or printing what cannot be parsed. Since the claim cleared
// the path, a file standing there is that attempt's. It holds only for an
// implementer, at the implement stage or a validate round's fix or sync.
func startedRunner(l Lane, a Await) (string, bool) {
	if a.Role != RoleImplementer || a.Kept != "" || (l.Stage != StageImplement && l.Stage != StageValidate) {
		return "", false
	}
	switch {
	case a.Running != nil:
		return a.Running.Runner, true
	case a.Paused != nil:
		switch a.Paused.Reason {
		case runner.ReasonInvalid, runner.ReasonFailed, runner.ReasonUnparsable:
			return a.Paused.Runner, true
		}
	}
	return "", false
}

// retriedOverReceipt reports whether a receipt standing for await a that does not
// verify leaves the role to be retried: its runner exited badly, so the file is no
// finished work to stop for. An answer that was refused is.
func retriedOverReceipt(a Await) bool {
	return a.Paused != nil && a.Paused.Reason != runner.ReasonInvalid
}

// stoppedReceipt is a receipt a step read and could not take: the lane, what the
// step says of it, the refusal to count against the lane, nil while a runner's
// claim stands and its agent may still be working or when the role will be retried
// over it, and whether it will be.
type stoppedReceipt struct {
	lane    string
	note    string
	refusal error
	retried bool
}

// collectStanding is a step's read of the receipts of the awaits startedRunner
// admits. It hands in the first that stands and verifies, in lane order, and
// reports true with the lane's result: that is the step's one move, and the next
// step takes the lane on. Otherwise it reports false, with each that stands and
// does not verify. A run that cannot be read gives neither, for the step's own
// read to refuse in its own frame.
func collectStanding(repoRoot, runID string, steps Stages, o Options) (StepResult, bool, []stoppedReceipt) {
	st, err := ReadState(repoRoot, runID)
	if err != nil {
		return StepResult{}, false, nil
	}
	var stopped []stoppedReceipt
	for _, i := range st.laneOrder() {
		l := st.Lanes[i]
		for _, a := range l.Awaits {
			name, ran := startedRunner(l, a)
			if !ran || !received(repoRoot, a.Receipt) {
				continue
			}
			route := &runner.RouteRecord{Asked: name}
			res, err := handIn(repoRoot, runID, a.Receipt, steps, o, route, true)
			if err == nil {
				res.Next = fmt.Sprintf("%s's %s, which the %s runner was started for, had written its receipt at %s, which verified, so no fresh agent is needed; %s",
					l.ID, a.Role, name, a.Receipt, res.Next)
				res.Route = route
				return res, true, nil
			}
			rf, refused := AsRefusal(err)
			if refused && rf.Contention {
				// Another invocation holds the run: the step's own write meets
				// the same lock and answers for it.
				return StepResult{}, false, nil
			}
			if !refused {
				rf = refuse("receipt", "", l.ID, err.Error(), "")
			}
			if rf.Lane == "" {
				rf.Lane = l.ID
			}
			if a.Running != nil && o.now().Before(a.Running.Until) {
				stopped = append(stopped, stoppedReceipt{lane: l.ID,
					note: fmt.Sprintf("a receipt stands at %s and does not verify yet (%s); the %s runner may still be working, until its claim lapses at %s, so wait, then step again",
						a.Receipt, refusalReason(rf), a.Running.Runner, stamp(a.Running.Until))})
				continue
			}
			if retriedOverReceipt(a) {
				stopped = append(stopped, stoppedReceipt{lane: l.ID, retried: true,
					note: fmt.Sprintf("a receipt stands at %s and does not verify (%s), so it is set aside when the role is retried on its runner", a.Receipt, refusalReason(rf))})
				continue
			}
			stopped = append(stopped, stoppedReceipt{lane: l.ID, note: stoppedNote(l.ID, a.Receipt, refusalReason(rf)), refusal: rf})
		}
	}
	return StepResult{}, false, stopped
}

// stoppedNote is what a step says of a lane stopped on a receipt that does not
// verify: why, and the ways out.
func stoppedNote(laneID, receipt, reason string) string {
	return fmt.Sprintf("a receipt stands at %s and does not verify (%s); %s is stopped and no runner is started over it. "+
		"Correct the receipt and run `abcd implement receipt %s`, or a plain `abcd implement step` reads it; or start over with `abcd implement step --restart %s`, which sets it aside",
		receipt, reason, laneID, receipt, laneID)
}

// stoppedText is what a step says of every receipt it could not take.
func stoppedText(stopped []stoppedReceipt) string {
	notes := make([]string, len(stopped))
	for i, s := range stopped {
		notes[i] = s.note
	}
	return strings.Join(notes, "; ")
}

// noteStopped counts each stopped lane as a refused step, and reports whether it
// changed the state. A run a drain does not read, an intent's, has no such count,
// so its record says it once.
func noteStopped(st *State, stopped []stoppedReceipt, now time.Time) bool {
	changed := false
	for _, s := range stopped {
		if s.refusal == nil {
			continue
		}
		if noteRefusal(st, s.refusal, now) {
			changed = true
			continue
		}
		if st.Issue() != "" || slices.ContainsFunc(st.Record, func(e Entry) bool { return e.Stage == StageRefused && e.Lane == s.lane && e.Note == s.note }) {
			continue
		}
		st.Record = append(st.Record, Entry{At: now, Lane: s.lane, Stage: StageRefused, Note: s.note})
		st.UpdatedAt = now
		changed = true
	}
	return changed
}

// standingUnderRoute is what a step carrying a --route says of a receipt
// collectStanding would have read, since that step reads none.
func standingUnderRoute(repoRoot, runID string) string {
	st, err := ReadState(repoRoot, runID)
	if err != nil {
		return ""
	}
	var notes []string
	for _, l := range st.Lanes {
		for _, a := range l.Awaits {
			if _, ran := startedRunner(l, a); ran && received(repoRoot, a.Receipt) {
				notes = append(notes, fmt.Sprintf("a receipt stands at %s: a plain `abcd implement step` reads it", a.Receipt))
			}
		}
	}
	return strings.Join(notes, "; ")
}

// noWorkPastRound is why a step cannot take a fix round's receipt as a fix, ""
// when it can, or when the await is no fix round's: the branch must hold a commit
// past the head the round judged, or the round would close on nothing a step can
// show was done. Handed back by `implement receipt`, a fix round that changed
// nothing is the implementer's own word, as it always was.
func noWorkPastRound(c Context, lane Lane, await Await) string {
	n := len(lane.Validation)
	if lane.Stage != StageValidate || n == 0 {
		return ""
	}
	if s := lane.pendingSync(); s != nil && await.Receipt == s.Receipt {
		return ""
	}
	tip, err := branchTip(c, lane)
	if err != nil {
		return "the branch's tip cannot be read to check for a commit past round's head: " + refusalReason(err)
	}
	if head := lane.Validation[n-1].HeadSHA; tip == head {
		return fmt.Sprintf("%s holds no commit past %s, the head round %d judged", lane.Branch, shortSHA(head), lane.Validation[n-1].Round)
	}
	return ""
}

// setAsideForAgent moves whatever stands at path, unread, into the lane's aside
// directory, and logs it in the run record, as an agent is handed the await at
// path or a runner is claimed for it: from here on what stands there is that
// agent's. The caller holds the run's lock and writes the state.
func setAsideForAgent(repoRoot string, st *State, lane Lane, path string, now time.Time) error {
	if !received(repoRoot, path) {
		return nil
	}
	rel, err := filepath.Rel(repoRoot, absIn(repoRoot, path))
	if err != nil {
		return err
	}
	rel = filepath.ToSlash(rel)
	laneDir, err := laneRel(st.RunID, lane.ID, "aside")
	if err != nil {
		return err
	}
	parent := laneDir + "/" + AsideDirName
	if err := fsutil.EnsureRealDirAll(repoRoot, parent, dirPerm); err != nil {
		return err
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return err
	}
	defer root.Close()
	dir, err := asideDir(root, parent, now)
	if err != nil {
		return err
	}
	name := filepath.Base(filepath.FromSlash(rel))
	if err := root.Rename(rel, dir+"/"+name); err != nil {
		return refuse("state", "", lane.ID, fmt.Sprintf("%s could not be set aside before its agent was started: %s", rel, fsutil.RedactHome(err.Error())),
			"move it out of the way yourself, then run `abcd implement step` again")
	}
	st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Stage: StageRunner,
		Note: fmt.Sprintf("what stood at %s before its agent was started is set aside at %s, unread", rel, dir)})
	return nil
}

// carryStanding keeps what from's step said of a receipt it did not take on r,
// the answer a later move of the same call gave in its place.
func carryStanding(r, from StepResult) StepResult {
	if from.standing != "" && !strings.Contains(r.Next, from.standing) {
		r.Next += "; " + from.standing
	}
	return r
}

// refusalReason is a refusal's reason, without its remedy, or an error's text.
func refusalReason(err error) string {
	if rf, ok := AsRefusal(err); ok {
		return rf.Reason
	}
	return err.Error()
}

// collectedMark ends the record's line for a receipt a step collected, before
// the digest of the bytes it took. The record is the one place the run keeps
// them, so the state's shape, and with it its schema, stays as it was.
const collectedMark = "; it stood at the path the lane awaited, so a step collected it and no fresh agent was asked for; sha256 "

// collectedNote is what the record adds for a receipt a step collected.
func collectedNote(digest string) string { return collectedMark + digest }

// standingDigest is the SHA-256, in hex, of the file at path through the guarded
// reader inside the checkout's root. A path that is not the checkout's is an
// error.
func standingDigest(repoRoot string, root *os.Root, path string) (string, error) {
	rel, err := filepath.Rel(repoRoot, absIn(repoRoot, path))
	if err != nil {
		return "", err
	}
	data, err := localtier.ReadInRoot(root, filepath.ToSlash(rel), maxReceiptBytes)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// collectedAt names the lane whose implementer's receipt a step collected at
// path: its index, the path as the lane took it, and the digest of the bytes the
// step took. It reads the record's line for that receipt, a receipt with no such
// line having been handed in rather than collected.
func (s State) collectedAt(repoRoot, path string) (int, string, string, bool) {
	for i, l := range s.Lanes {
		for _, r := range l.Receipts {
			if !samePath(repoRoot, path, r.Receipt) {
				continue
			}
			for k := len(s.Record) - 1; k >= 0; k-- {
				e := s.Record[k]
				if e.Lane != l.ID || e.Stage != "receipt" || !strings.Contains(e.Note, " verified at "+r.Receipt) {
					continue
				}
				if at := strings.LastIndex(e.Note, collectedMark); at >= 0 {
					return i, r.Receipt, e.Note[at+len(collectedMark):], true
				}
			}
		}
	}
	return 0, "", "", false
}

// answerCollected answers a hand-in at taken, a path a step already collected. It
// reads the file first: bytes other than those the step took are refused, and the
// lane stays where it is. The same bytes are answered, and change nothing unless
// the hand-in brings a runner's report, which replaces the route the step could
// not know. It reports whether it changed the state.
func answerCollected(repoRoot string, root *os.Root, st *State, i int, taken, digest string, route *runner.RouteRecord, now time.Time) (StepResult, bool, error) {
	l := st.Lanes[i]
	if got, err := standingDigest(repoRoot, root, taken); err != nil || got != digest {
		return StepResult{}, false, refuse("receipt", "", l.ID,
			fmt.Sprintf("%s has changed since a step collected it, and %s went on from head %s", taken, l.ID, shortSHA(l.HeadSHA)),
			"nothing was taken from it: restore the branch to that head if it has moved, and leave the file as it was collected")
	}
	changed := false
	if route != nil && route.Ran != "" {
		l.Receipts = slices.Clone(l.Receipts)
		for k := range l.Receipts {
			r := l.Receipts[k]
			if r.Route != nil && r.Route.Ran == "" && samePath(repoRoot, r.Receipt, taken) {
				reported := *route
				l.Receipts[k].Route = &reported
				st.Record = append(st.Record, Entry{At: now, Lane: l.ID, Stage: StageRunner, Note: routeNote(RoleImplementer, reported)})
				st.UpdatedAt = now
				changed = true
			}
		}
		st.Lanes[i] = l
	}
	return laneResult(*st, l, "", nil), changed, nil
}
