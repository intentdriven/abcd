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
// A lane armed before this wait existed is not asked, recorded or armed again
// while such a report is still to come: the landing withdraws its arming
// (withdrawLateArming) and the lane then waits at its arming as an unarmed lane
// does. One a late report hands back has the arming withdrawn once the
// hand-back is written (releaseAfterWrite), and until the forge reports no
// arming the lane is retried by each later step and an issue run's reservation
// is kept; the record says what came of each try.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
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
// of the lane after the wait's (a receipt's or a validate line among them)
// means the lane left the landing since, and the wait it is in, if any, has not
// begun. The bound's clock therefore starts again after such a line, and when
// the first unreported receipt changes, since note names that receipt.
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
// writes the record line (beganRunnerWait), and once the stuck bound's minutes
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

// armingOwed reports whether l was handed back with its pull request still
// armed, so the hand-back owes the arming's withdrawal (withdrawLateArming): a
// lane handed back as closed has a pull request the forge already closed.
func armingOwed(l Lane) bool {
	return l.Stage == StageHandedBack && l.PR > 0 && l.Landing != nil && l.Landing.Armed &&
		(l.HandBack == nil || l.HandBack.Kind != HandBackClosed)
}

// lateHandBack is why a handed-back lane's arming is withdrawn.
func lateHandBack(lane Lane) string { return lane.ID + " was handed back after it was armed" }

// withdrawLateArming takes the arming off lane's pull request through the
// forge, as a held sibling's is taken off (holdLane), for why, and returns the
// record's line on what came of it, with the lane's arming cleared: withdrawn
// and read back as gone; nothing left to withdraw (no auto-merge and no queue
// entry, or the pull request closed); or merged already, which is said as
// merged. Any other answer (the arming unread, the forge refusing, the
// read-back failing or still armed) is a contention refusal that leaves the
// lane's arming as it was, for a later step to try again; a lost network waits
// on the shared probe. then is what the line says comes next once the arming
// is gone. The review receipt this checkout holds for the lane's pushed head is
// removed first (voidOwnReview), so no re-arm made of that head, the release
// hold's included, counts the pair that judged it.
func withdrawLateArming(c Context, lane *Lane, stage Stage, why, then string) (string, error) {
	n := strconv.Itoa(lane.PR)
	voided := voidOwnReview(c.RepoRoot, *lane)
	retry := func(reason string) error {
		return contend(string(stage), "", lane.ID, "pull request #"+n+" is armed and "+why+", and "+reason+"; until the forge reports no arming it can still merge"+voided,
			"run `abcd implement step` again: the step tries the withdrawal again, and releases an issue run's reservation only once the forge reports no arming; or withdraw it yourself with `gh pr merge "+n+
				" --disable-auto`, taking it out of the merge queue on its page if it is there")
	}
	clear := func() {
		ld := *lane.Landing
		ld.Merge, ld.Armed = "", false
		lane.Landing = &ld
	}
	v, unread, err := readQueue(c, *lane)
	switch {
	case err != nil:
		if r, ok := AsRefusal(err); ok && r.outage != nil {
			return "", r
		}
		return "", retry("its arming could not be read (" + refusalReason(err) + ")")
	case unread != "":
		return "", retry("its arming could not be read (" + unread + ")")
	case strings.EqualFold(v.State, "MERGED"):
		clear()
		return "pull request #" + n + " had already merged when " + why + ", so there was no arming to withdraw" + voided, nil
	case strings.EqualFold(v.State, "CLOSED") || (v.AutoMergeRequest == nil && !v.queued()):
		clear()
		return "pull request #" + n + " was recorded armed when " + why + ", and the forge reports no arming left on it (no auto-merge and no merge queue entry, or the pull request closed), so there was none to withdraw; " + then + voided, nil
	}
	if err := landing.WithdrawArming(gateForge(c, *lane), n, v.ID, v.AutoMergeRequest != nil, v.queued()); err != nil {
		if r := netFailure(stage, lane.ID, "gh pr merge", err); r != nil {
			return "", r
		}
		return "", retry("the forge refused to withdraw the arming: " + forgeWhy(err))
	}
	if err := confirmWithdrawn(c, *lane, stage); err != nil {
		if r, ok := AsRefusal(err); ok && r.outage != nil {
			return "", r
		}
		return "", retry(refusalReason(err))
	}
	clear()
	return "withdrew pull request #" + n + "'s arming, since " + why + "; " + then + voided, nil
}

// withdrawOwed is the step's move on lane i, handed back with its pull request
// still armed (armingOwed): the withdrawal tried again (withdrawLateArming).
// One that does not take is the step's contention, nothing written; one that
// does is recorded, and the run's reservation is released once the step's save
// has written it, under the same lock (releaseAfterWrite).
func withdrawOwed(repoRoot string, st *State, i int, c Context, now time.Time) (StepResult, error) {
	lane := st.Lanes[i]
	note, err := withdrawLateArming(c, &lane, StageHandedBack, lateHandBack(lane), "it merges only if a person arms or merges it")
	if err != nil {
		return StepResult{}, err
	}
	st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Stage: string(StageHandedBack), Note: note})
	st.Lanes[i] = lane
	st.UpdatedAt = now
	res := laneResult(*st, lane, "", nil)
	res.Next = note + "; " + res.Next
	res.release = releaseAfterWrite(repoRoot, st, lane.ID, now)
	return res, nil
}

// voidOwnReview removes the review receipt this checkout's local tier holds
// for lane's pushed head (landing.RecordReview, which recordOwnReview writes
// from the round's pair), and returns the clause the record says it with, ""
// when there was none. A report since has shown that pair cannot be shown to
// differ from every model that wrote the lane, so the merge gate asked again
// of that head (landing.DecideMerge, which the release hold's re-arm asks
// too) is not to count it; a later round's pair, recorded again once the
// lane's reviewers are settled, does.
func voidOwnReview(repoRoot string, lane Lane) string {
	if lane.Landing == nil || !gitutil.IsFullSHA(lane.Landing.Pushed) {
		return ""
	}
	head := lane.Landing.Pushed
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return "; the review receipt of its head " + shortSHA(head) + " could not be checked (" + termsafe.Sanitize(fsutil.RedactHome(err.Error())) + "), so a re-arm of that head may still count it"
	}
	defer root.Close()
	rel := landing.ReviewReceiptsRelDir + "/" + head
	if _, err := root.Lstat(rel); errors.Is(err, fs.ErrNotExist) {
		return ""
	}
	if err := root.Remove(rel); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "; the review receipt of its head " + shortSHA(head) + " could not be removed (" + termsafe.Sanitize(fsutil.RedactHome(err.Error())) + "), so a re-arm of that head may still count it"
	}
	return "; the review receipt of its head " + shortSHA(head) + " in this checkout is removed, so no re-arm of that head counts the pair that judged it"
}
