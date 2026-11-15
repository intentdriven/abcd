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
// while such a report is still to come: once the landing step's save is
// written, its arming is withdrawn (withdrawAfterWrite), and the lane then
// waits at its arming as an unarmed lane does. One a late report hands back
// has the arming withdrawn once the hand-back is written (releaseAfterWrite),
// and the review receipt the loop recorded for its head moved aside
// (voidOwnReview); until the forge reports no arming the lane is not finished:
// each later step and each drain move takes it up again, its issue run's
// reservation is kept and renewed, and the record says what came of the
// withdrawal once it takes.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/localtier"
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
// lane handed back as closed has a pull request the forge already closed. A
// lane that owes it is not finished: a drain reads it as in progress
// (laneOutcome) and routes the hand-back only once the forge reports no
// arming, and its run's reservation is renewed until then (handedBackIssue).
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
// is gone. With void set, which only a late writer's hand-back sets, the review
// receipt the loop recorded for the lane's pushed head is moved aside first
// (voidOwnReview), so no re-arm made of that head, the release hold's
// included, counts the pair that judged it.
func withdrawLateArming(c Context, lane *Lane, stage Stage, why, then string, void bool) (string, error) {
	n := strconv.Itoa(lane.PR)
	voided := ""
	if void {
		voided = voidOwnReview(c.RepoRoot, *lane)
	}
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
//
// The forge is asked here, before the save: a save then refused leaves the
// forge disarmed and the state armed, and the lane handed back, where nothing
// but this move reads its arming. The next step's withdrawal reads the forge
// again, finds no arming left, and records that, so nothing is lost.
//
// Past the stuck bound's minutes since the lane began owing the withdrawal
// (owedSince), the refusal's remedy says a person withdraws the arming or
// decides the lane (owedToPerson); the reservation stays kept and renewed.
func withdrawOwed(repoRoot string, st *State, i int, c Context, now time.Time) (StepResult, error) {
	lane := st.Lanes[i]
	note, err := withdrawLateArming(c, &lane, StageHandedBack, lateHandBack(lane), "it merges only if a person arms or merges it", lateWriterHandedBack(lane))
	if err != nil {
		if r, ok := AsRefusal(err); ok && strings.HasPrefix(r.Reason, owedRetryPrefix(lane)) {
			roots, _ := layered.RootsFor(repoRoot)
			if b, berr := resolveStuckBounds(roots); berr == nil {
				if mins, past := owedPast(*st, lane, b, now); past {
					r.Remedy = owedToPerson(lane, mins, b)
				}
			}
		}
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

// owedRetryPrefix is how a hand-back's withdrawal of lane's arming that did not
// take begins (withdrawLateArming's refusal), in the record line
// releaseAfterWrite writes for it and in each later step's refusal.
func owedRetryPrefix(lane Lane) string {
	return "pull request #" + strconv.Itoa(lane.PR) + " is armed and " + lateHandBack(lane) + ", and "
}

// owedSince is when lane, handed back with its pull request still armed
// (armingOwed), began owing the withdrawal, which its bound counts from, as the
// runner wait counts from its record line (runnerWaitSince): the lane's first
// record line of a withdrawal that did not take since it was handed back, or
// the hand-back's own time where the record holds none. ok is false where
// neither is known.
func owedSince(st State, lane Lane) (since time.Time, ok bool) {
	if lane.HandBack != nil {
		since = lane.HandBack.At
	}
	prefix := owedRetryPrefix(lane)
	for _, e := range st.Record {
		if e.Lane == lane.ID && e.Stage == string(StageHandedBack) && !e.At.Before(since) && strings.HasPrefix(e.Note, prefix) {
			return e.At, true
		}
	}
	return since, !since.IsZero()
}

// owedPast is how many whole minutes lane has owed its withdrawal (owedSince),
// and whether that is at or past the stuck bound's minutes in b.
func owedPast(st State, lane Lane, b stuckBounds, now time.Time) (mins int, past bool) {
	since, ok := owedSince(st, lane)
	if !ok {
		return 0, false
	}
	mins = int(now.Sub(since) / time.Minute)
	return mins, mins >= b.minutes.Value
}

// owedToPerson is what a step's refusal and a drain's next say of a lane whose
// withdrawal has not taken past the bound (owedPast): a person withdraws the
// arming or decides the lane, and the reservation is kept and renewed.
func owedToPerson(lane Lane, mins int, b stuckBounds) string {
	n := strconv.Itoa(lane.PR)
	return fmt.Sprintf("the withdrawal of pull request #%s's arming has not taken in %d minutes of tries, past the bound of %d (%s), so a person withdraws it with `gh pr merge %s --disable-auto`, taking it out of the merge queue on its page if it is there, or decides the lane; until the forge reports no arming the run's reservation is kept and renewed, and each `abcd implement step` still tries the withdrawal again",
		n, mins, b.minutes.Value, describeOrigin(b.minutes), n)
}

// landWithdrawal is a landing's withdrawal of an armed lane's arming while a
// runner's report on one of its implementers is still to come (landMerged):
// why the arming is withdrawn and what comes next once it is gone.
type landWithdrawal struct{ why, then string }

// withdrawAfterWrite registers, on the state a landing step left lane at, the
// step (State.afterWritten) that withdraws the lane's arming once that save's
// write has returned without error, under the lock the save took: a save
// refused or failed asks nothing of the forge, so the state and the forge
// never disagree on whether the lane is armed. The step reads the lane as it
// stands in the state just written and acts only on one still armed at its
// land stage. A withdrawal that takes (withdrawLateArming) clears the arming
// and is recorded by a second write under the same hold; one that does not is
// kept as the step's refusal (handBackRelease.refused), nothing more written,
// the arming as it was, and the next step at the landing tries again. It
// voids no review receipt: no report has yet shown the lane's reviewers to
// share a model with its writers.
func withdrawAfterWrite(repoRoot string, st *State, lane string, w landWithdrawal, now time.Time) *handBackRelease {
	out := &handBackRelease{atLand: true}
	st.afterWritten(func(s *State, rewrite func() error) {
		out.ran = true
		defer func() { out.state = *s }()
		at := slices.IndexFunc(s.Lanes, func(l Lane) bool { return l.ID == lane })
		if at < 0 {
			return
		}
		l := s.Lanes[at]
		if l.Stage != StageLand || l.PR <= 0 || l.Landing == nil || !l.Landing.Armed {
			return
		}
		c := Context{RepoRoot: repoRoot, RunDir: runRel(s.RunID), State: *s, Now: now}
		note, err := withdrawLateArming(c, &l, StageLand, w.why, w.then, false)
		if err != nil {
			out.refused = err
			return
		}
		s.Lanes[at] = l
		s.Record = append(s.Record, Entry{At: now, Lane: l.ID, Stage: string(StageLand), Note: note})
		s.UpdatedAt = now
		out.arming = note
		out.unwritten = rewrite()
	})
	return out
}

// voidedSuffix is added to the name of a review receipt the loop moved aside
// (voidOwnReview). The receipt's readers (landing's findReceipt and
// readReviewReceipts) look for the file named by the head alone, so a receipt
// under this name counts for no head.
const voidedSuffix = ".voided"

// ownReviewEntry is one entry of a review receipt, as landing.RecordReview
// writes it, read for the name and digest of the return it counts.
type ownReviewEntry struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}

// voidOwnReview moves aside the review receipt the loop recorded for lane's
// pushed head in this checkout's local tier (recordOwnReview, through
// landing.RecordReview, from the returns of the lane's last round), and
// returns the clause the record says it with, "" when no receipt names that
// head here; a receipt an earlier try moved aside is said as moved aside again,
// so each line of the withdrawal names the move alike. It is asked only for a lane a late writer handed back
// (lateWriterHandedBack): that report has shown the round's pair cannot be
// shown to differ from every model that wrote the lane, so the merge gate
// asked again of that head (landing.DecideMerge, which the release hold's
// re-arm asks too) is not to count it; a later round's pair, recorded again
// once the lane's reviewers are settled, does.
//
// The receipt is the loop's own only when its commit is the head and its
// reviews are the round's non-auditor returns, each by its name and the
// sha256 of its text, as recordOwnReview handed them in; any other receipt at
// that path (one a person recorded with `abcd land --record-review`, or one
// whose returns cannot be read now) is left in place, and the clause says it
// stands as its recorder's. A receipt in another worktree is not this lane's
// to touch: the loop records its own in this checkout alone, so one found
// elsewhere was recorded by someone else, and it still counts.
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
		// A receipt an earlier try of this withdrawal moved aside, which only
		// this move does, is said in the same words, so the line of a try that
		// takes names the move as the first try's did.
		if _, err := root.Lstat(rel + voidedSuffix); err == nil {
			return movedAside(head)
		}
		return ""
	}
	stands := func(why string) string {
		return "; the review receipt of its head " + shortSHA(head) + " in this checkout is left in place, since " + why + ", so it stands as its recorder's"
	}
	data, err := localtier.ReadInRoot(root, rel, maxReviewReceiptBytes)
	if err != nil {
		return stands("it could not be read (" + termsafe.Sanitize(fsutil.RedactHome(err.Error())) + ") to show it is the one the loop recorded")
	}
	own, why := ownReviewEntries(repoRoot, lane)
	if why != "" {
		return stands("the lane's round cannot be read to show it is the one the loop recorded (" + why + ")")
	}
	var rec struct {
		Commit  string           `json:"commit"`
		Reviews []ownReviewEntry `json:"reviews"`
	}
	if err := json.Unmarshal(data, &rec); err != nil || rec.Commit != head || !sameEntries(rec.Reviews, own) {
		return stands("it is not the one the loop recorded from the lane's round")
	}
	if err := root.Rename(rel, rel+voidedSuffix); err != nil {
		return "; the review receipt the loop recorded for its head " + shortSHA(head) + " could not be moved aside (" + termsafe.Sanitize(fsutil.RedactHome(err.Error())) + "), so a re-arm of that head may still count it"
	}
	return movedAside(head)
}

// movedAside is the clause a withdrawal's line says the loop's own review
// receipt for head was moved aside with (voidOwnReview).
func movedAside(head string) string {
	return "; the review receipt the loop recorded for its head " + shortSHA(head) + " from the lane's round is moved aside to " + head + voidedSuffix +
		" in this checkout, so no re-arm of that head counts the pair that judged it"
}

// maxReviewReceiptBytes bounds the review receipt voidOwnReview reads.
const maxReviewReceiptBytes = 1 << 20

// ownReviewEntries are the entries recordOwnReview would write for lane's
// pushed head from its last round: each non-auditor validator's return, by
// name and digest. why is set where they cannot be read: no round, a round
// that read another head, or a return that cannot be read.
func ownReviewEntries(repoRoot string, lane Lane) ([]ownReviewEntry, string) {
	n := len(lane.Validation)
	if n == 0 {
		return nil, "the lane records no review round"
	}
	round := lane.Validation[n-1]
	if round.HeadSHA != lane.Landing.Pushed {
		return nil, "its latest round read " + shortSHA(round.HeadSHA) + ", not the pushed head"
	}
	var out []ownReviewEntry
	for _, v := range round.Validators {
		if v.Role == RoleAuditor {
			continue
		}
		text, err := readReturn(repoRoot, v.Return, lane.ID)
		if err != nil {
			return nil, refusalReason(err)
		}
		sum := sha256.Sum256(text)
		out = append(out, ownReviewEntry{Name: v.Return, SHA256: hex.EncodeToString(sum[:])})
	}
	return out, ""
}

// sameEntries reports whether a receipt's entries are exactly want, in any
// order: the same count, and each one's name with its digest.
func sameEntries(got, want []ownReviewEntry) bool {
	if len(got) != len(want) || len(want) == 0 {
		return false
	}
	left := slices.Clone(want)
	for _, g := range got {
		i := slices.Index(left, g)
		if i < 0 {
			return false
		}
		left = slices.Delete(left, i, i+1)
	}
	return true
}
