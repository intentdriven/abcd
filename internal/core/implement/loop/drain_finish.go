package loop

// drain_finish.go finishes merged lanes (spc-2609212015054359 step 3,
// iss-2610090936460789). A lane's worktree and branch are cleaned up only by
// the landing's last step, run once its pull request has merged, and nothing
// came back to run it: the land stage waits until the merge, the drain read the
// lane as its pull request and moved on, and a run paused by its window clock
// refused the step for hours. So every drain move, before its window clock is
// read, takes the landing's last step for each issue lane parked at land whose
// pushed head is on the default branch: the cleanup does no new work, so the
// pace does not hold it.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// DrainFinished is one merged lane a drain move took the landing's last step
// for: Done when its worktree and branch were removed and the lane marked
// done, otherwise Note says why the landing refused.
type DrainFinished struct {
	Issue string `json:"issue"`
	RunID string `json:"run_id"`
	PR    int    `json:"pr,omitempty"`
	Done  bool   `json:"done"`
	Note  string `json:"note"`
}

// DrainParked is one lane parked at land whose head is not on the default
// branch, as the move read its pull request (spc-2610161044461448 step 8).
type DrainParked struct {
	Issue string `json:"issue"`
	RunID string `json:"run_id"`
	PR    int    `json:"pr"`
	State string `json:"state"`
	Note  string `json:"note"`
}

// HandBackClosed is the kind a parked lane whose pull request was closed
// unmerged is handed back as; a lane's receipt cannot name it.
const HandBackClosed = claim.HandBackClosed

// What a move read a parked lane's pull request as.
const (
	ParkedWaiting     = "waiting"
	ParkedConflicting = "conflicting"
	ParkedClosed      = "closed"
	ParkedMerged      = "merged"
	ParkedUnread      = "unread"
	// ParkedSyncing: the pull request conflicts only on files origin's
	// default branch declares generated, or not at all as this checkout
	// merges, so the lane went back through its sync (step 13).
	ParkedSyncing = "syncing"
)

// parkedAtLand is the issue lane of run waiting on its pull request's merge:
// at land, its pull request armed or left open, its head pushed, and not yet
// cleaned up. It reports -1 for any other run.
func parkedAtLand(run State) int {
	if run.Issue() == "" {
		return -1
	}
	i := run.current()
	if i < 0 {
		return -1
	}
	l := run.Lanes[i]
	if l.Stage != StageLand || l.PR <= 0 || l.Landing == nil || l.Landing.Merge == "" || l.Landing.Merged != "" ||
		!gitutil.IsFullSHA(l.Landing.Pushed) {
		return -1
	}
	return i
}

// parkedListLimit is how many of the person's latest pull requests the one
// listing a move reads its parked lanes from asks for: one page of the
// forge's API. A parked lane's pull request older than that is read as
// unread, and kept held as it was.
const parkedListLimit = 100

// parkedFields are the listing's fields. gh's `pr list --json` names no
// merge-queue entry (ghcontract pins its list), so the arming
// (autoMergeRequest) stands for it.
const parkedFields = "number,headRefName,state,mergeable,mergeStateStatus,autoMergeRequest"

// listedPR is one pull request as the forge lists it.
type listedPR struct {
	Number           int             `json:"number"`
	HeadRefName      string          `json:"headRefName"`
	State            string          `json:"state"`
	Mergeable        string          `json:"mergeable"`
	MergeStateStatus string          `json:"mergeStateStatus"`
	AutoMergeRequest json.RawMessage `json:"autoMergeRequest"`
}

// armed reports whether the forge lists the pull request armed to merge.
func (p listedPR) armed() bool {
	return len(p.AutoMergeRequest) > 0 && string(p.AutoMergeRequest) != "null"
}

// listParked reads the person's latest pull requests, open, closed and
// merged, in one forge listing, by number; unread says why it could not.
func listParked(repoRoot string) (map[int]listedPR, string) {
	out, err := landing.GH(repoRoot, "pr", "list", "--state", "all", "--author", "@me",
		"--limit", strconv.Itoa(parkedListLimit), "--json", parkedFields)
	if errors.Is(err, landing.ErrNoForgeClient) {
		return nil, "the forge client `gh` is not installed"
	}
	if err != nil {
		return nil, termsafe.Sanitize(gitutil.ScrubURLCredentials(fsutil.RedactHome(err.Error())))
	}
	var prs []listedPR
	if err := json.Unmarshal([]byte(out), &prs); err != nil {
		return nil, "the forge's listing of the pull requests does not parse"
	}
	byNumber := map[int]listedPR{}
	for _, p := range prs {
		if p.Number > 0 {
			byNumber[p.Number] = p
		}
	}
	return byNumber, ""
}

// readParked is what the listing says of run's parked lane: closed, merged
// (by a squash or rebase, its head not on the default branch), conflicting,
// waiting on its merge, or unread when the listing could not be read or does
// not show the lane's pull request, which keeps the lane held as before.
func readParked(run State, lane Lane, listed map[int]listedPR, unread string) DrainParked {
	p := DrainParked{Issue: run.Issue(), RunID: run.RunID, PR: lane.PR}
	n := "pull request #" + strconv.Itoa(lane.PR)
	if unread != "" {
		p.State, p.Note = ParkedUnread, n+" could not be read from the forge ("+unread+"), so its lane is kept held"
		return p
	}
	pr, ok := listed[lane.PR]
	if !ok || pr.State == "" || (pr.HeadRefName != "" && pr.HeadRefName != lane.Branch) {
		p.State, p.Note = ParkedUnread, fmt.Sprintf("%s is not among the forge's listing of your latest %d pull requests, so its lane is kept held", n, parkedListLimit)
		return p
	}
	status := termsafe.Sanitize(strings.ToLower(pr.MergeStateStatus))
	switch {
	case pr.State == "CLOSED":
		p.State, p.Note = ParkedClosed, n+" was closed without merging, so its lane is handed back as closed and its reservation released"
	case pr.State == "MERGED":
		p.State, p.Note = ParkedMerged, n+" merged, but the lane's head is not on the default branch (a squash or rebase rewrote it), so the loop cannot prove the lane landed and does not hold it longer: confirm the change is on the default branch, then remove the lane's worktree and branch yourself"
	case pr.Mergeable == MergeableConflicting:
		p.State, p.Note = ParkedConflicting, n+" conflicts with the default branch as the forge computes it (merge state "+status+"), so its lane is no longer renewed: a person merges the default branch into it, or closes it"
	case pr.armed():
		p.State, p.Note = ParkedWaiting, n+" is open and armed, waiting on its merge (merge state "+status+"), so its lane is kept held"
	default:
		p.State, p.Note = ParkedWaiting, n+" is open, waiting on a person's merge (merge state "+status+"), so its lane is kept held"
	}
	return p
}

// held reports whether a parked lane read as p is still renewed: one waiting
// on its merge, or one whose pull request could not be read.
func (p DrainParked) held() bool { return p.State == ParkedWaiting || p.State == ParkedUnread }

// finishMerged takes the landing's last step for every issue lane in the
// checkout parked at land whose pushed head is on the default branch as the
// remote holds it, whatever the drain's or the run's window clock says, and
// reads every other parked lane from its pull request
// (spc-2610161044461448 step 8; dcn-2610161040380948). The forge lists the
// pull requests once, and the default branch is fetched once, both only when
// a lane is parked. A lane whose pull request is open, or could not be read,
// is renewed and named as waiting; a conflicting one is not renewed, and is
// named; a closed one is handed back as closed with its reservation released
// (handBackClosed), which the drain then routes. A lane the landing refuses
// to finish (a branch moved past what landed, a worktree with changes) is
// named, not fatal: the drain's own move goes on.
func finishMerged(repoRoot string, now time.Time) ([]DrainFinished, []DrainParked, error) {
	runs, err := Runs(repoRoot)
	if err != nil {
		return nil, nil, err
	}
	var parked []State
	for _, run := range runs {
		if parkedAtLand(run) >= 0 {
			parked = append(parked, run)
		}
	}
	out, reads := []DrainFinished{}, []DrainParked{}
	if len(parked) == 0 {
		return out, reads, nil
	}
	listed, unread := listParked(repoRoot)
	read := make(map[string]DrainParked, len(parked))
	for _, run := range parked {
		read[run.RunID] = readParked(run, run.Lanes[parkedAtLand(run)], listed, unread)
	}
	// Every drain move keeps its parked lanes held (spc-2610031753359849): a
	// pull request waiting on a person's merge can wait overnight, and
	// nothing else steps the run, so each parked run's reservation is renewed
	// here at half its lease, logged in the run's record. A lane whose pull
	// request is closed, merged or conflicting is not: it waits on nothing
	// that will come.
	// The view is read once, from disk, outside the run lock, and only a run
	// it shows due (or held by another) is locked and renewed; the renewal's
	// own fetch and push are made for that run alone.
	if snap, err := claim.View(repoRoot, now); err == nil {
		for _, run := range parked {
			if run.ReservedAt == nil || !read[run.RunID].held() {
				continue
			}
			if _, due, lost := renewalDue(run, snap, now); !due && lost == "" {
				continue
			}
			if err := mutate(repoRoot, run.RunID, func(_ *os.Root, st *State) (bool, error) {
				if parkedAtLand(*st) < 0 {
					return false, nil
				}
				logged, updated := len(st.Record), st.UpdatedAt
				renewReservation(repoRoot, st, now, &snap)
				return len(st.Record) != logged || !st.UpdatedAt.Equal(updated), nil
			}); err != nil {
				if _, ok := AsRefusal(err); !ok {
					return nil, nil, err
				}
			}
		}
	}
	refused := func(run State, r *Refusal) DrainFinished {
		return DrainFinished{Issue: run.Issue(), RunID: run.RunID, PR: run.Lanes[parkedAtLand(run)].PR,
			Note: fsutil.RedactHome(r.Reason + "; " + r.Remedy)}
	}
	first := parked[0].Lanes[parkedAtLand(parked[0])]
	def, err := defaultBranch(Context{RepoRoot: repoRoot}, first)
	if err != nil {
		r, ok := AsRefusal(err)
		if !ok {
			return nil, nil, err
		}
		for _, run := range parked {
			out = append(out, refused(run, r))
		}
		return out, reads, nil
	}
	tracking, err := landing.FetchDefault(gitutil.Net, repoRoot, def)
	if err != nil {
		r := refuse(StageDrain, "", "", "git could not fetch "+def+" from "+Remote+" to find the merged lanes: "+errors.Unwrap(err).Error(),
			"the next `abcd drain` looks again")
		for _, run := range parked {
			out = append(out, refused(run, r))
		}
		return out, reads, nil
	}
	for _, run := range parked {
		lane := run.Lanes[parkedAtLand(run)]
		f := DrainFinished{Issue: run.Issue(), RunID: run.RunID, PR: lane.PR}
		on, err := gitutil.IsAncestor(repoRoot, lane.Landing.Pushed, tracking)
		if err != nil {
			// git could not answer, which is not a no: the lane is named, so
			// an unreadable pushed head does not drop it from finished.
			f.Note = fsutil.RedactHome("git could not place " + shortSHA(lane.Landing.Pushed) + " on " + tracking + ": " + err.Error() +
				"; settle what git reports, then run `abcd drain` again")
			out = append(out, f)
			continue
		}
		if !on {
			p := read[run.RunID]
			if p.State == ParkedConflicting && syncable(repoRoot, tracking, lane.Landing.Pushed) {
				// A conflict confined to declared generated files, or one
				// this checkout cannot reproduce, goes back through the
				// lane's sync (sync.go, step 13): the run's own step takes it.
				note, moved, err := resyncParked(repoRoot, run.RunID, now)
				if err != nil {
					return nil, nil, err
				}
				if moved {
					p.State, p.Note = ParkedSyncing, note
				} else if note != "" {
					// The step that would enter the sync refused, or did
					// not move the lane: what it said is named, not
					// dropped behind the forge's reading.
					p.Note += "; its sync was not entered: " + note
				}
			}
			if p.State == ParkedClosed {
				if p.Note, err = handBackClosed(repoRoot, run.RunID, now); err != nil {
					return nil, nil, err
				}
				if p.Note == "" {
					continue
				}
			}
			p.Note = fsutil.RedactHome(p.Note)
			reads = append(reads, p)
			continue
		}
		err = mutate(repoRoot, run.RunID, func(_ *os.Root, st *State) (bool, error) {
			i := parkedAtLand(*st)
			if i < 0 {
				return false, nil
			}
			w := laneWant(*st, i)
			if w.kind != wantBinary || w.hold {
				return false, nil
			}
			res, err := performMove(repoRoot, st, DefaultStages(), w, now, nil)
			if err != nil {
				return false, err
			}
			if st.Lanes[i].Stage != StageDone {
				// The landing's step completed something short of the cleanup;
				// what it wrote is kept, and the lane is not claimed done.
				f.Note = res.Next
				return true, nil
			}
			f.Done = true
			f.Note = st.Record[len(st.Record)-1].Note
			return true, nil
		})
		if err != nil {
			r, ok := AsRefusal(err)
			if !ok {
				return nil, nil, err
			}
			out = append(out, refused(run, r))
			continue
		}
		if f.Done || f.Note != "" {
			out = append(out, f)
		}
	}
	return out, reads, nil
}

// syncable reports whether a parked lane whose pull request the forge lists
// conflicting goes back through its sync: a merge of tracking and its pushed
// head, as the forge merges, conflicts in no file, or only in files origin's
// default branch declares generated. Every misreading answers no, which
// leaves the lane a person's.
func syncable(repoRoot, tracking, pushed string) bool {
	f := localConflict(repoRoot, tracking, pushed)
	if f.err != nil {
		return false
	}
	gen, _ := claim.GeneratedFiles(repoRoot)
	return len(undeclared(f.paths, gen)) == 0
}

// resyncParked takes the run's own landing step for its parked lane, under
// the run's lock, which enters the lane's sync again over its conflicting
// pull request (resyncLane). moved is true when the lane left its merge wait
// for the sync; note is the record's line, or why the step refused.
func resyncParked(repoRoot, runID string, now time.Time) (note string, moved bool, err error) {
	err = mutate(repoRoot, runID, func(_ *os.Root, st *State) (bool, error) {
		i := parkedAtLand(*st)
		if i < 0 {
			return false, nil
		}
		w := laneWant(*st, i)
		if w.kind != wantBinary || w.hold {
			return false, nil
		}
		res, err := performMove(repoRoot, st, DefaultStages(), w, now, nil)
		if err != nil {
			return false, err
		}
		moved = parkedAtLand(*st) < 0
		note = res.Next
		if n := len(st.Record); n > 0 {
			note = st.Record[n-1].Note
		}
		return true, nil
	})
	if r, ok := AsRefusal(err); ok {
		return fsutil.RedactHome(r.Reason + "; " + r.Remedy), false, nil
	}
	return note, moved, err
}

// handBackClosed hands run's parked lane back as closed, its pull request
// closed without merging, and releases the issue's reservation with the
// hand-back's tombstone (releaseOnHandBack), under the run's lock. It returns
// the record's line, "" when the lane is no longer parked (a step running at
// the same time moved it).
func handBackClosed(repoRoot, runID string, now time.Time) (string, error) {
	note := ""
	err := mutate(repoRoot, runID, func(_ *os.Root, st *State) (bool, error) {
		i := parkedAtLand(*st)
		if i < 0 {
			return false, nil
		}
		lane := st.Lanes[i]
		hb := HandBack{Kind: HandBackClosed, Reason: "pull request #" + strconv.Itoa(lane.PR) + " was closed without merging",
			Home: "a person: the issue stays open, and the lane's run " + st.RunID + " keeps its worktree and branch until `abcd implement step --run " +
				st.RunID + " --discard " + lane.ID + "` discards them"}
		handBackLane(st, &lane, hb, "", now)
		releaseOnHandBack(repoRoot, st, *lane.HandBack, now)
		st.Lanes[i] = lane
		st.UpdatedAt = now
		note = st.Record[len(st.Record)-1].Note
		if k := len(st.Record) - 2; k >= 0 && st.Record[k].Stage == string(StageHandedBack) {
			note = st.Record[k].Note + "; " + note
		}
		return true, nil
	})
	return note, err
}

// parkedNote names, before a drain move's next move, the parked lanes it read
// from their pull requests.
func parkedNote(ps []DrainParked) string {
	var parts []string
	for _, p := range ps {
		parts = append(parts, fmt.Sprintf("%s's run %s: %s", p.Issue, p.RunID, p.Note))
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("read %d parked lane(s) from their pull requests: %s", len(parts), strings.Join(parts, "; "))
}

// finishedNote names, before a drain move's next move, the merged lanes it
// finished and those it could not.
func finishedNote(fs []DrainFinished) string {
	var done, not []string
	for _, f := range fs {
		pr := ""
		if f.PR > 0 {
			pr = ", pull request #" + strconv.Itoa(f.PR)
		}
		if f.Done {
			done = append(done, fmt.Sprintf("%s's run %s%s", f.Issue, f.RunID, pr))
		} else {
			not = append(not, fmt.Sprintf("%s's run %s%s (%s)", f.Issue, f.RunID, pr, f.Note))
		}
	}
	var parts []string
	if len(done) > 0 {
		parts = append(parts, fmt.Sprintf("finished %d merged lane(s), each worktree and branch removed and the lane done: %s",
			len(done), strings.Join(done, "; ")))
	}
	if len(not) > 0 {
		parts = append(parts, fmt.Sprintf("could not finish %d merged lane(s): %s", len(not), strings.Join(not, "; ")))
	}
	return strings.Join(parts, "; ")
}
