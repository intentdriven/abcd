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
// started for exactly this await (startedRunner). A receipt carrying a hand-back
// claims no commits and discards the lane, which is a decision rather than a
// report of work, so a step does not take it: `implement receipt` does, or the
// runner's own hand-in as its agent ends. The proof speaks for one
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
//
// A drive is the exception to the stop (iss-2611071242184002). Where the drive
// itself drives the run (Options.driverIsSelf), a receipt refused for the work it
// reports (Refusal.work) is consumed in the same write as the refusal
// (consumeRefused): set aside into a new directory under the lane's aside
// directory, kept and never deleted, with the refusal and the digest of the bytes
// judged in the record's line. The receipt's text never chooses what else goes
// with it, and the run's state decides whether anything does
// (implementStageAwait). At the implement stage's own await, the report and the
// definition of done's output go with it, only at the names the loop assigned
// that attempt (ReportFileName, DoDFileName), directly in the lane's own
// directory, each a regular file and no link, moved through that directory opened
// as its own root (moveNamed); a path the receipt names for either that is
// anything else is left, and the record's line says so. At a fix or sync round's
// await nothing but the receipt goes: the round's own report and output are left
// where they are, and so are the bare names in the lane's directory, which are
// the implement stage's and which later rounds read; the record's line says only
// the receipt was set aside at that round. Only the bytes judged are
// moved: a file replaced since it was judged, or an await a runner has claimed
// since, is left standing for the next step to read, uncounted. The digest check
// guards against the run's own steps and a file replaced between the judging and
// the lock; it does not guard against a writer outside the run racing the rename
// itself, which the run's lock does not stop, and nothing is deleted either way.
//
// Every consumed receipt is counted as a refused step under one check and reason
// (consumedRefusal), whatever it lacked, so the drain's stuck bound of identical
// refusals, three by default, is a bound of consumed receipts in a row. Only an
// issue run keeps that count, and only the drain's issue runs are driven this
// way; an intent's run is never driven so, and keeps its stop. The third makes
// the lane stuck, with or without the drive's --until, and since the count is the
// lane's in the run's state, a fresh drive on the same state carries it on. Time
// between them does not matter: the count of identical refusals is never reset by
// a quiet, which restarts only the stuck minutes. A step of the lane that
// succeeds clears it, and a refusal of another kind restarts it. That is the
// count's limit: such a lane is still ended by three failed launches in a row
// under a bound, and by the outage's limit without one.
//
// Nothing then stands at the path, so below the bound the next step starts the
// role again on its runner (heldRole). A receipt is consumed at two places: the
// runner's own hand-in (pauseAwait), which under a bound is also a failed launch
// of the drive's, and a receipt a step finds standing, left by a step that was
// killed or by an older abcd (advanceWith), which is a refused step and no
// launch, and whose role a drive retries in the same call. A consumed receipt
// that brings the lane to the stuck bound is followed by no launch on either
// route: after the hand-in the drive's next turn gives the lane back
// (stuckBack), and after a receipt found standing the role is not retried in that
// call (stuckAfterConsumed), so the drive's next turn gives it back. A receipt
// refused for anything else (unreadable, another run's, a hand-back, a fix round
// with no commit) is not consumed: it stops the lane as above, and each step
// counts it until the stuck bounds hand the lane back. A host-stepped run keeps
// the stop.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/localtier"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// startedRunner is the runner the run's state shows was started for await a of
// lane l, and whether it shows one: a claim taken under the lock just before the
// launch, or a pause recording how that attempt ended: its answer refused, its
// harness reporting an error or exiting badly, or printing what cannot be parsed.
// It holds only for an implementer, at the implement stage or a validate round's
// fix or sync.
//
// What it proves has a limit. A claim was taken, not that an agent ran: a claim can
// outlive a step killed before its launch, and a `failed` pause also covers a
// harness that could not start. So a file at the path came after the claim, from
// whatever could write there, not necessarily from an agent of that attempt. What
// bounds it is the verifier, which is why a receipt that claims no commits is not
// taken (handIn) and why the route leaves what ran it empty.
func startedRunner(l Lane, a Await) (string, bool) {
	if a.Role != RoleImplementer || a.Kept != "" || (l.Stage != StageImplement && l.Stage != StageValidate) {
		return "", false
	}
	switch {
	case a.Running != nil:
		return a.Running.Runner, true
	case a.Paused != nil:
		switch a.Paused.Reason {
		case runner.ReasonInvalid, runner.ReasonFailed, runner.ReasonUnparsable, runner.ReasonRefused:
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
// over it, and whether the note is the whole of what is true of the lane. One that
// is not sits beside the lane's own move and is true only until the step's later
// moves, so it is not carried past them.
type stoppedReceipt struct {
	lane    string
	note    string
	refusal error
	whole   bool
	// receipt is the await's path, and consume says a drive sets what stands there
	// aside, its refusal being for the work (consumeStopped).
	receipt string
	consume bool
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
				res.Next = fmt.Sprintf("%s's %s was routed to the %s runner, and a receipt stood at %s and verified, so no fresh agent is needed; %s",
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
				stopped = append(stopped, stoppedReceipt{lane: l.ID, whole: true,
					note: fmt.Sprintf("a receipt stands at %s and does not verify yet (%s); the %s runner may still be working, until its claim lapses at %s, so wait, then step again",
						a.Receipt, refusalReason(rf), a.Running.Runner, stamp(a.Running.Until))})
				continue
			}
			if retriedOverReceipt(a) {
				stopped = append(stopped, stoppedReceipt{lane: l.ID,
					note: fmt.Sprintf("a receipt stands at %s and is not taken (%s), and is set aside if the role is retried on its runner", a.Receipt, refusalReason(rf))})
				continue
			}
			note := stoppedNote(l.ID, a.Receipt, refusalReason(rf))
			if rf.handBack {
				note = stoppedHandBackNote(l.ID, a.Receipt)
			}
			stopped = append(stopped, stoppedReceipt{lane: l.ID, note: note, refusal: rf, whole: true,
				receipt: a.Receipt, consume: o.driverIsSelf && rf.work})
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

// stoppedHandBackNote is stoppedNote for a receipt that hands its lane back, which
// correcting does not cure and no step takes.
func stoppedHandBackNote(laneID, receipt string) string {
	return fmt.Sprintf("the receipt at %s hands %s back, which a step does not do on its own; %s is stopped and no runner is started over it. "+
		"To hand it back, run `abcd implement receipt %s`; or start over with `abcd implement step --restart %s`, which sets the receipt aside",
		receipt, laneID, laneID, receipt, laneID)
}

// carriedText is what a step's later moves in the same call keep saying.
func carriedText(stopped []stoppedReceipt) string {
	var notes []string
	for _, s := range stopped {
		if s.whole {
			notes = append(notes, s.note)
		}
	}
	return strings.Join(notes, "; ")
}

// stoppedText is what a step says of every receipt it could not take.
func stoppedText(stopped []stoppedReceipt) string {
	notes := make([]string, len(stopped))
	for i, s := range stopped {
		notes[i] = s.note
	}
	return strings.Join(notes, "; ")
}

// consumeStopped consumes each receipt a drive's step read and found refused for
// the work (stoppedReceipt.consume), in the step's own write, through
// consumeRefused, so noteStopped does not count it again and the step's answer
// says what became of it. It reports whether it changed the state, and whether
// it counted one as a refused step. The caller holds the run's lock and writes
// the state.
func consumeStopped(repoRoot string, root *os.Root, st *State, stopped []stoppedReceipt, now time.Time) (changed, counted bool) {
	for k := range stopped {
		s := &stopped[k]
		if !s.consume || s.refusal == nil {
			continue
		}
		i, at := st.findAwait(repoRoot, s.receipt)
		if i < 0 || st.Lanes[i].ID != s.lane {
			continue
		}
		switch consumeRefused(repoRoot, root, st, i, at, "", s.refusal, now) {
		case consumedAside:
			s.note = fmt.Sprintf("a receipt stood at %s and was refused for the work (%s), so it is set aside, kept, and the implementer is started again on its runner, unless this brings the lane to the drain's stuck bounds",
				s.receipt, refusalReason(s.refusal))
			counted = true
		case consumedStands:
			counted = true
		case consumedLeft:
			s.note = fmt.Sprintf("a receipt stands at %s that is not the one this step judged, or its role has been claimed since, so it is left for the next step to read", s.receipt)
		}
		// Counted in consumeRefused, or left uncounted: never again here.
		s.refusal = nil
		st.UpdatedAt = now
		changed = true
	}
	return changed, counted
}

// consumed is what consumeRefused made of a receipt refused for the work.
type consumed int

const (
	// consumedAside: set aside, kept, and counted.
	consumedAside consumed = iota
	// consumedStands: the set-aside failed; counted, and the receipt stands.
	consumedStands
	// consumedLeft: not the bytes judged, or the role claimed since; left
	// standing for the next step to read, and not counted.
	consumedLeft
)

// consumeRefused consumes the receipt await k of lane i awaits, refused for the
// work. Under the caller's lock it first checks that what it would move is what was
// judged: no runner's claim but session's own stands on the await, and the file at
// the path still has the digest of the bytes the verifier judged (Refusal.judged).
// Either failing, the file is left standing for the next step to read, and nothing
// is counted. Otherwise the receipt is set aside into a new directory under the
// lane's aside directory, at the implement stage's own await with the report and
// the definition of done's output only at the names the loop assigned them
// (moveNamed), and at a fix or sync round's alone, the record's line names the
// refusal, the digest and what went with it, and the step is counted under the one
// identity every consumed receipt shares (consumedRefusal), whatever it lacked, so
// the drain's stuck bound of identical refusals is a bound of consumed receipts in a
// row. A set-aside that fails takes nothing else from the caller's write: the
// failure is the record's line, the receipt stands, and it is counted the same way,
// as each later step that reads and consumes it again is. The caller holds the
// run's lock and writes the state.
func consumeRefused(repoRoot string, root *os.Root, st *State, i, k int, session string, refusal error, now time.Time) consumed {
	lane, a := st.Lanes[i], st.Lanes[i].Awaits[k]
	path := a.Receipt
	judged := ""
	var named []laneNamed
	if rf, ok := AsRefusal(refusal); ok {
		judged, named = rf.judged, rf.named
	}
	why := bounded(refusalReason(refusal), maxConsumedReason)
	left := func(because string) consumed {
		st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Stage: StageRunner,
			Note: fmt.Sprintf("the receipt at %s was refused for the work (%s), but %s, so it is left standing for the next step to read and is not counted",
				path, why, because)})
		return consumedLeft
	}
	if c := a.Running; c != nil && c.Session != session && now.Before(c.Until) {
		return left(fmt.Sprintf("the %s runner has claimed the role since, until %s", termsafe.Sanitize(c.Runner), stamp(c.Until)))
	}
	data, err := standingBytes(repoRoot, root, path)
	if err != nil || judged == "" || answerDigest(data) != judged {
		return left(fmt.Sprintf("what stands there is not the bytes judged (sha256 %s)", judgedOrNone(judged)))
	}
	rel, dir, err := moveAside(repoRoot, st, lane, path, now, "after it was refused for the work")
	switch {
	case err != nil:
		st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Stage: StageRunner,
			Note: fmt.Sprintf("the receipt at %s was refused for the work (%s; sha256 %s) and could not be set aside (%s); it stands, and each later step counts it as a refused one until the drain's stuck bounds hand %s back",
				path, why, judged, bounded(refusalReason(err), maxConsumedReason), lane.ID)})
		noteRefusal(st, consumedRefusal(lane.ID), now)
		return consumedStands
	case rel == "":
		return left("nothing stands there now")
	}
	note := fmt.Sprintf("the receipt at %s was refused for the work (%s), so it is set aside at %s, kept (sha256 %s)", rel, why, dir, judged)
	laneDir, lerr := laneRel(st.RunID, lane.ID, "aside")
	var moved, kept []string
	switch {
	case lerr != nil:
	case !implementStageAwait(lane, rel, laneDir):
		// A fix or sync round's report and output are the round's, and the bare
		// names in the lane's directory are the implement stage's, which later
		// rounds read: nothing but the receipt goes. The line's words are for a
		// fix or sync round. That rests on startedRunner admitting an
		// implementer at the implement and validate stages alone
		// (TestStartedRunnerAdmitsAnImplementerAtImplementAndValidateOnly), and
		// on the validate stage's implementer awaits being a fix or sync
		// round's, which nothing here checks; where the round's directory holds
		// no report or output, nothing else stands to be left. Another
		// implementer await added later at either stage needs its own words
		// here.
		note += "; only the receipt is set aside at this round, and the round's report and the definition of done's output are left where they are"
	default:
		moved, kept = moveNamed(root, laneDir, dir, named)
	}
	if len(moved) > 0 {
		note += ", with " + strings.Join(moved, " and ")
	}
	note += ", and the implementer is started again on its runner, unless this brings the lane to the drain's stuck bounds, which hand it back"
	if len(kept) > 0 {
		note += "; left in place: " + strings.Join(kept, "; ")
	}
	st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Stage: StageRunner, Note: note})
	noteRefusal(st, consumedRefusal(lane.ID), now)
	return consumedAside
}

// implementStageAwait reports whether the receipt at rel, relative to the
// checkout, is the implement stage's own, read from the run's state alone: the
// lane is at the implement stage and the receipt stands directly in the lane's
// directory laneDir, where the loop assigned that attempt its report and output
// (ReportFileName, DoDFileName). A fix or sync round's receipt stands in the
// round's own directory, and the bare names beside the lane's receipt are the
// implement stage's, not that round's.
func implementStageAwait(lane Lane, rel, laneDir string) bool {
	return lane.Stage == StageImplement && path.Dir(rel) == laneDir
}

// judgedOrNone is a judged digest, or says there is none.
func judgedOrNone(d string) string {
	if d == "" {
		return "none recorded"
	}
	return d
}

// consumedRefusal is the refused step every consumed receipt of a lane is counted
// as: one stage, check and reason, whatever the receipt lacked, which the record's
// line for it keeps instead. A step of the lane that succeeds clears the count, and
// any refusal of another kind restarts it.
func consumedRefusal(laneID string) *Refusal {
	return refuse("receipt", consumedCheck, laneID, consumedReason, "")
}

// consumedReason is the reason every consumed receipt is counted under.
const consumedReason = "an implementer's receipt was refused for the work it reports, so it was set aside and the implementer started again; the run record keeps what each one lacked"

// moveNamed moves into dir, the consumed receipt's new directory under the lane's
// aside directory, each file the receipt names that is the one the loop assigned
// it: the receipt's path, cleaned, is exactly the name the loop gave that file
// (laneNamed.assigned), directly in the lane's own directory laneDir. It is called
// only for the implement stage's own await (implementStageAwait), the one whose
// assigned names those are; at a fix or sync round's nothing but the receipt
// moves. The receipt's text never chooses what moves: a path naming anything
// else, the brief, a receipt, a validate, sync or land round's file, anything
// under the aside directory in any letter case, is left where it is. The one it
// may name is moved only as a regular file, no link, and only when its
// destination in dir can be checked and nothing stands there.
// The check and the rename go through laneDir opened as its own root, so a
// directory swapped for a link cannot reach outside it. It reports what moved and
// what was left in place, and why, each in the record's words. The caller holds
// the run's lock.
func moveNamed(root *os.Root, laneDir, dir string, named []laneNamed) (moved, kept []string) {
	into, inLane := strings.CutPrefix(dir, laneDir+"/")
	var lane *os.Root
	if inLane {
		var err error
		if lane, err = root.OpenRoot(laneDir); err != nil {
			lane = nil
		} else {
			defer lane.Close()
		}
	}
	for _, n := range named {
		if n.rel == "" {
			continue
		}
		shown := bounded(termsafe.Sanitize(n.rel), maxNamedShown)
		if !fsutil.ValidRelPath(n.rel) {
			shown = termsafe.DescribeRefused(n.rel)
		}
		if path.Clean(n.rel) != n.assigned {
			kept = append(kept, fmt.Sprintf("%s %s (not %s, the name the loop assigned it in the lane's directory)", n.what, shown, n.assigned))
			continue
		}
		if lane == nil {
			kept = append(kept, fmt.Sprintf("%s %s (the lane's directory could not be opened)", n.what, shown))
			continue
		}
		fi, err := lane.Lstat(n.assigned)
		switch {
		case err != nil:
			kept = append(kept, fmt.Sprintf("%s %s (it is not there)", n.what, shown))
			continue
		case fi.Mode()&fs.ModeSymlink != 0:
			kept = append(kept, fmt.Sprintf("%s %s (a link, which is not followed)", n.what, shown))
			continue
		case !fi.Mode().IsRegular():
			kept = append(kept, fmt.Sprintf("%s %s (not a regular file)", n.what, shown))
			continue
		}
		dest := into + "/" + n.assigned
		switch _, err := lane.Lstat(dest); {
		case err == nil:
			kept = append(kept, fmt.Sprintf("%s %s (a file of its name is already set aside there)", n.what, shown))
			continue
		case !errors.Is(err, fs.ErrNotExist):
			kept = append(kept, fmt.Sprintf("%s %s (where it would be set aside cannot be checked: %s)", n.what, shown, fsutil.RedactHome(err.Error())))
			continue
		}
		if err := lane.Rename(n.assigned, dest); err != nil {
			kept = append(kept, fmt.Sprintf("%s %s (it could not be moved: %s)", n.what, shown, fsutil.RedactHome(err.Error())))
			continue
		}
		moved = append(moved, n.what+" "+n.assigned)
	}
	return moved, kept
}

// maxNamedShown bounds, in runes, a named file's path in the record's line.
const maxNamedShown = 200

// maxConsumedReason bounds, in runes, a refusal the record's line for a consumed
// receipt carries.
const maxConsumedReason = 600

// laneNamed is a file a receipt names in the lane's directory: what it is to the
// receipt, its path as the receipt names it, relative to that directory, and the
// name the loop assigned that file there (ReportFileName, DoDFileName), the only
// one a drive moves aside with the receipt.
type laneNamed struct {
	what, rel, assigned string
}

// consumedCheck is the check every consumed receipt is counted under, whatever it
// lacked, so the drain's stuck bound of identical refusals counts consumed
// receipts in a row (noteRefusal).
const consumedCheck = "refused for the work"

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
	return setAside(repoRoot, st, lane, path, now, "before its agent was started", func(rel, dir string) string {
		return fmt.Sprintf("what stood at %s before its agent was started is set aside at %s, unread", rel, dir)
	})
}

// setAsideCutShort moves a return whose launch reports it stopped short out of the
// path a later hand-in would read it at, with no report to say so. The caller holds
// the run's lock and writes the state.
func setAsideCutShort(repoRoot string, st *State, lane Lane, path string, now time.Time) error {
	return setAside(repoRoot, st, lane, path, now, "after its launch reported it stopped short", func(rel, dir string) string {
		return fmt.Sprintf("the return at %s, whose launch reports it stopped short, is set aside at %s", rel, dir)
	})
}

// setAside moves whatever stands at path into the lane's aside directory, and logs it
// in the run record in the words note gives it (moveAside). The caller holds the
// run's lock and writes the state.
func setAside(repoRoot string, st *State, lane Lane, path string, now time.Time, when string, note func(rel, dir string) string) error {
	rel, dir, err := moveAside(repoRoot, st, lane, path, now, when)
	if err != nil || rel == "" {
		return err
	}
	st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Stage: StageRunner, Note: note(rel, dir)})
	return nil
}

// moveAside moves whatever stands at path into a new directory under the lane's
// aside directory, made exclusively, and reports the path it moved, relative to the
// checkout, and that directory; rel is "" when nothing stood there. A move that
// fails is refused in the caller's words: when says at what point it was being set
// aside. The caller holds the run's lock and writes the state.
func moveAside(repoRoot string, st *State, lane Lane, path string, now time.Time, when string) (rel, dir string, err error) {
	if !received(repoRoot, path) {
		return "", "", nil
	}
	rel, err = filepath.Rel(repoRoot, absIn(repoRoot, path))
	if err != nil {
		return "", "", err
	}
	rel = filepath.ToSlash(rel)
	laneDir, err := laneRel(st.RunID, lane.ID, "aside")
	if err != nil {
		return "", "", err
	}
	parent := laneDir + "/" + AsideDirName
	if err := fsutil.EnsureRealDirAll(repoRoot, parent, dirPerm); err != nil {
		return "", "", err
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return "", "", err
	}
	defer root.Close()
	if dir, err = asideDir(root, parent, now); err != nil {
		return "", "", err
	}
	name := filepath.Base(filepath.FromSlash(rel))
	if err := root.Rename(rel, dir+"/"+name); err != nil {
		return "", "", refuse("state", "", lane.ID, fmt.Sprintf("%s could not be set aside %s: %s", rel, when, fsutil.RedactHome(err.Error())),
			"move it out of the way yourself, then run `abcd implement step` again")
	}
	return rel, dir, nil
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

// standingBytes reads the file at path through the guarded reader inside the
// checkout's root. A path that is not the checkout's is an error.
func standingBytes(repoRoot string, root *os.Root, path string) ([]byte, error) {
	rel, err := filepath.Rel(repoRoot, absIn(repoRoot, path))
	if err != nil {
		return nil, err
	}
	return localtier.ReadInRoot(root, filepath.ToSlash(rel), maxReceiptBytes)
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
	if data, err := standingBytes(repoRoot, root, taken); err != nil || answerDigest(data) != digest {
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
