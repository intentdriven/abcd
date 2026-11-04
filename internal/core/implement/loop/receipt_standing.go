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
// where the run's own state shows a runner was started for exactly this await
// (startedRunner), and only for the implement stage's implementer, whose
// verifier holds a receipt to the run, the lane, the branch and commits on it.
// Every other await keeps the path it always had: its agent's runner hands its
// return back when it ends, a host hands one back with `implement receipt`, and
// a hand-out starts its agent. What stands at a path is not an answer to an
// await nothing has been started for.
//
// A receipt that verifies is taken, over a runner's claim still standing if need
// be: a refused step could not drop its claim, and the claim names no process to
// ask. The lane takes the head the receipt verified at, so if the agent was in
// fact still working, its later commits move the branch past that head, and the
// landing refuses a tip the validators did not judge. Its later hand-in is read
// (answerCollected): the same bytes are answered, and a changed file is refused.
//
// The route records what is known. The state names the runner that was asked;
// no report from it arrived, so what ran the agent is left empty until a report
// does. A receipt that does not verify is left where it stands, the await stays
// out, no second runner is started over it, and the step says why.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/localtier"
)

// startedRunner is the runner the run's state shows was started for await a of
// lane l, and whether it shows one: a claim taken under the lock just before the
// launch, or a pause recording that the runner's answer was refused, which only
// a run that ended gives. It holds only for the implement stage's implementer.
func startedRunner(l Lane, a Await) (string, bool) {
	if l.Stage != StageImplement || a.Role != RoleImplementer || a.Kept != "" {
		return "", false
	}
	switch {
	case a.Running != nil:
		return a.Running.Runner, true
	case a.Paused != nil && a.Paused.Reason == runner.ReasonInvalid:
		return a.Paused.Runner, true
	}
	return "", false
}

// collectStanding is a step's read of the receipts of the awaits startedRunner
// admits. It hands in the first that stands and verifies, in lane order, and
// reports true with the lane's result: that is the step's one move, and the next
// step takes the lane on. Otherwise it reports false, with what the step should
// say of each that stands and does not verify. A run that cannot be read gives
// neither, for the step's own read to refuse in its own frame.
func collectStanding(repoRoot, runID string, steps Stages, o Options) (StepResult, bool, string) {
	st, err := ReadState(repoRoot, runID)
	if err != nil {
		return StepResult{}, false, ""
	}
	var notes []string
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
				return res, true, ""
			}
			if rf, ok := AsRefusal(err); ok && rf.Contention {
				// Another invocation holds the run: the step's own write meets
				// the same lock and answers for it.
				return StepResult{}, false, ""
			}
			notes = append(notes, fmt.Sprintf("a receipt stands at %s and does not verify (%s); it is left there and no runner is started over it: correct it and run `abcd implement receipt %s`, or start over with `abcd implement step --restart %s`",
				a.Receipt, refusalReason(err), a.Receipt, l.ID))
		}
	}
	return StepResult{}, false, strings.Join(notes, "; ")
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

// collectedAt is the lane whose implementer's receipt a step collected at path,
// its index, and the digest of the bytes the step took.
func (s State) collectedAt(repoRoot, path string) (int, string, bool) {
	for i, l := range s.Lanes {
		if l.Receipt == "" || !samePath(repoRoot, path, l.Receipt) {
			continue
		}
		for k := len(s.Record) - 1; k >= 0; k-- {
			e := s.Record[k]
			if e.Lane != l.ID || e.Stage != "receipt" {
				continue
			}
			if at := strings.LastIndex(e.Note, collectedMark); at >= 0 {
				return i, e.Note[at+len(collectedMark):], true
			}
		}
	}
	return 0, "", false
}

// answerCollected answers a hand-in at a path a step already collected. It reads
// the file first: bytes other than those the step took are refused, and the lane
// stays where it is. The same bytes are answered, and change nothing unless the
// hand-in brings a runner's report, which replaces the route the step could not
// know. It reports whether it changed the state.
func answerCollected(repoRoot string, root *os.Root, st *State, i int, digest string, route *runner.RouteRecord, now time.Time) (StepResult, bool, error) {
	l := st.Lanes[i]
	if got, err := standingDigest(repoRoot, root, l.Receipt); err != nil || got != digest {
		return StepResult{}, false, refuse("receipt", "", l.ID,
			fmt.Sprintf("%s has changed since a step collected it, and %s went on from head %s", l.Receipt, l.ID, shortSHA(l.HeadSHA)),
			"nothing was taken from it; decide whether that head stands, and restart the lane if it does not")
	}
	changed := false
	if route != nil && route.Ran != "" {
		for k := range l.Receipts {
			r := l.Receipts[k]
			if r.Route != nil && r.Route.Ran == "" && samePath(repoRoot, r.Receipt, l.Receipt) {
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
