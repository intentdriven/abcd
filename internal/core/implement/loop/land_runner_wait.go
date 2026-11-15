package loop

// land_runner_wait.go holds a lane's arming until every runner that ran one of
// its implementers has said which model ran. A step can collect an
// implementer's receipt where it stood before the runner's own report reaches
// the loop (receipt_standing.go); the report, when it comes, names a writer of
// the lane, and the reviewers handed out since are settled against it
// (answerCollected). Until it has come, the lane's reviewers have been judged
// against a writer the loop does not know, so the landing does not arm the
// lane's pull request: it waits at the arming, as its other waits do, with one
// record line naming the receipt, and once the stuck bound's minutes have
// passed since that line it hands the lane back as stuck, naming the receipt.
//
// A lane armed before this wait existed can still meet a late report that hands
// it back: the hand-back then withdraws the arming through the forge, as a held
// sibling's is withdrawn (hold.go), and the record says what came of it.

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/core/layered"
)

// unreportedRunner is the lane's first implementer receipt launched through a
// runner whose report of what ran it has not reached the loop.
func unreportedRunner(lane Lane) (ReceiptRecord, bool) {
	for _, r := range lane.Receipts {
		if r.Role == RoleImplementer && r.Route != nil && r.Route.Ran == "" {
			return r, true
		}
	}
	return ReceiptRecord{}, false
}

// runnerWaitNote is the record's line for a lane waiting at its arming for the
// report of the runner that ran the implementer whose receipt is r.
func runnerWaitNote(r ReceiptRecord) string {
	return "waiting for the runner's report on " + r.Receipt + ": the implementer was routed to " + r.Route.Asked +
		", and no report of which model ran it has reached the loop, so the pull request is not armed until it does"
}

// runnerWaitSince is when the lane began its current wait at the arming with
// note, read from the record line that began it. The lane's lines at its land
// stage, its refusals and its runners' reports are passed over; any other line
// of the lane after the wait's means the lane left the landing since, and the
// wait it is in, if any, has not begun.
func runnerWaitSince(st State, laneID, note string) (time.Time, bool) {
	for k := len(st.Record) - 1; k >= 0; k-- {
		e := st.Record[k]
		if e.Lane != laneID {
			continue
		}
		if e.Stage == string(StageLand) && e.Note == note {
			return e.At, true
		}
		if e.Stage != string(StageLand) && e.Stage != StageRefused && e.Stage != StageRunner {
			return time.Time{}, false
		}
	}
	return time.Time{}, false
}

// awaitRunnerReport is the arming's wait for a runner's report: waits is false
// when every implementer receipt of the lane has its runner's report, or was
// run by the host. Otherwise the lane waits, a contention whose first call
// writes the record line (beganCheckWait), and once the stuck bound's minutes
// have passed since that line it is handed back as stuck, naming the receipt.
func awaitRunnerReport(c Context, lane Lane) (out Outcome, waits bool, err error) {
	r, ok := unreportedRunner(lane)
	if !ok {
		return Outcome{}, false, nil
	}
	note := runnerWaitNote(r)
	if since, began := runnerWaitSince(c.State, lane.ID, note); began {
		roots, _ := layered.RootsFor(c.RepoRoot)
		b, err := resolveStuckBounds(roots)
		if err != nil {
			return Outcome{}, true, err
		}
		if mins := int(c.Now.Sub(since) / time.Minute); mins >= b.minutes.Value {
			return Outcome{HandBack: &HandBack{Kind: HandBackStuck,
				Reason: fmt.Sprintf("its pull request was not armed: the implementer whose receipt is %s was routed to %s, and no report of which model ran it has reached the loop in %d minutes, past the bound of %d (%s), so the lane's reviewers cannot be shown to differ from every model that wrote it",
					r.Receipt, r.Route.Asked, mins, b.minutes.Value, describeOrigin(b.minutes)),
				Home: "a person: the lane's run " + c.State.RunID + " keeps its worktree and branch until `abcd implement step --run " + c.State.RunID +
					" --discard " + lane.ID + "` discards them; build " + keyOf(c.State) + " again, and its writers' models are recorded as its runners report them"}}, true, nil
		}
	}
	rf := contend(string(StageLand), "", lane.ID, note,
		"the runner that ran the implementer hands its report in as its agent ends; run `abcd implement step` again once it has. The lane is handed back as stuck if it has not come within the stuck bound's minutes")
	rf.runnerWait = note
	return Outcome{}, true, rf
}

// beganRunnerWait writes, once, the record line that begins lane i's wait at
// its arming for a runner's report, and reports whether it wrote.
func beganRunnerWait(st *State, i int, r *Refusal, now time.Time) bool {
	if r.runnerWait == "" {
		return false
	}
	l := st.Lanes[i]
	if _, began := runnerWaitSince(*st, l.ID, r.runnerWait); began {
		return false
	}
	st.Record = append(st.Record, Entry{At: now, Lane: l.ID, Stage: string(StageLand), Note: r.runnerWait})
	st.UpdatedAt = now
	return true
}

// withdrawLateArming withdraws the arming of a lane a runner's late report
// handed back after its pull request was armed, through the forge, as a held
// sibling's arming is withdrawn (holdLane), and returns the record's line on
// what came of it. A withdrawal the forge takes and stops reporting clears the
// lane's arming; any other answer leaves it as it was, and the line names the
// command that withdraws it.
func withdrawLateArming(repoRoot string, st State, lane *Lane, now time.Time) string {
	c := Context{RepoRoot: repoRoot, State: st, Now: now}
	n := strconv.Itoa(lane.PR)
	still := "; it is still armed and can merge: withdraw it with `gh pr merge " + n + " --disable-auto`, and take it out of the merge queue on its page if it is there"
	v, unread, err := readQueue(c, *lane)
	switch {
	case err != nil:
		return "pull request #" + n + " was armed when " + lane.ID + " was handed back, and its arming could not be read (" + refusalReason(err) + ")" + still
	case unread != "":
		return "pull request #" + n + " was armed when " + lane.ID + " was handed back, and its arming could not be read (" + unread + ")" + still
	case strings.EqualFold(v.State, "MERGED"):
		return "pull request #" + n + " had already merged when " + lane.ID + " was handed back, so there was no arming to withdraw"
	}
	if err := landing.WithdrawArming(gateForge(c, *lane), n, v.ID, v.AutoMergeRequest != nil, v.queued()); err != nil {
		return "pull request #" + n + " was armed when " + lane.ID + " was handed back, and the forge refused to withdraw the arming: " + forgeWhy(err) + still
	}
	if err := confirmWithdrawn(c, *lane, StageHandedBack); err != nil {
		return refusalReason(err) + still
	}
	ld := *lane.Landing
	ld.Merge, ld.Armed = "", false
	lane.Landing = &ld
	return "withdrew pull request #" + n + "'s arming, since " + lane.ID + " was handed back after it was armed; it merges only if a person arms or merges it"
}
