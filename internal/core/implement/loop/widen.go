package loop

// widen.go is an issue lane's widening of its hold before an edit beyond it
// (iss-2610311224444576; dcn-2610311251002713, dcn-2610311544325397). A
// build and a drain reserve an issue with the path its found_at names, and a
// fix may need to change a further file. Before the implementer's first edit
// there it runs `abcd implement widen`, which reserves the issue again with
// the footprint its reservation holds and the further paths, judged for
// overlap and against the cap like any reservation (widenReservation), so a
// clash is met before the work, not once it is done. Another copy's live
// hold on one of them is contention: nothing is pushed, the run record says
// the widening backed off, and the implementer edits none of them. The
// receipt is the backstop (unheldGaps): one is refused while the lane's
// branch changes a path the reservation does not cover, which catches a
// widening forgotten and an edit made through the shell. The fence before a
// landing's push and its arming judges the same paths (holdLaneChanges), for
// a lane past its last receipt. The run record keeps each path widened to
// (StageWidenedTo), so a reservation made again names it whatever became of
// the reservation branch.

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/shellquote"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// StageWidenedTo is the run record's stage for a path a lane's reservation was
// widened to; its note is the path and nothing else. A reservation made again
// names these beside the found_at (widenedPaths), so they outlive a deleted tip.
const StageWidenedTo = "reservation_widened_to"

// WidenResult is a widening's outcome: the run and lane, the paths newly held
// (none when the reservation already covered every one), the footprint the
// reservation holds after, and the run record's line.
type WidenResult struct {
	RunID     string   `json:"run_id"`
	Lane      string   `json:"lane"`
	Record    string   `json:"record"`
	Added     []string `json:"added"`
	Footprint []string `json:"footprint"`
	Note      string   `json:"note"`
}

// widenStages are the stages at which a lane's implementer edits: the
// implement stage, and the validate stage, whose fix rounds hand a fresh
// implementer the lane.
var widenStages = []Stage{StageImplement, StageValidate}

// Widen widens the reservation of issue run runID, for its current lane
// laneID, to cover paths before the lane's implementer edits them. A path
// the reservation already covers adds nothing, and when it covers every one
// nothing is pushed. A run that holds no reservation (no origin) has nothing
// to widen and says so. Refused for a person (exit 2): an intent's run,
// whose footprint is its spec's; a lane that is not the run's current one or
// not at an editing stage; a path no reservation may carry; a reservation
// the view does not show this copy holding. Contention (exit 3): another
// copy's live hold on one of the paths, with nothing pushed and a run record
// line saying the widening backed off; or a locked run state. Its brief has
// the lane's implementer run it, so the save is the reservation's alone
// (mutateReservation, iss-2611020719494343).
func Widen(repoRoot, runID, laneID string, paths []string, o Options) (WidenResult, error) {
	res := WidenResult{RunID: runID, Lane: laneID, Added: []string{}}
	var backedOff error
	err := mutateReservation(repoRoot, runID, func(st *State) (bool, error) {
		now := o.now()
		res.Record = st.Key
		if st.Issue() == "" {
			return false, refuse(StageReservation, "", laneID, st.Key+"'s run holds the footprint its spec names, and only an issue lane widens its hold",
				"name the path on the `- packages:` line of "+specOf(*st)+"'s `## Footprint` section, then run `abcd build "+st.Key+"` again")
		}
		i := st.current()
		if i < 0 || st.Lanes[i].ID != laneID {
			named := laneID
			if !ValidLaneID(laneID) {
				named = termsafe.DescribeRefused(laneID)
			}
			return false, refuse(StageReservation, "", "", named+" is not the current lane of "+runID,
				"name the lane the brief names; `abcd implement status` lists the run's lanes")
		}
		lane := st.Lanes[i]
		if !slices.Contains(widenStages, lane.Stage) {
			return false, refuse(StageReservation, "", laneID, fmt.Sprintf("%s is at its %s stage, where no implementer edits, so its hold is not widened", laneID, lane.Stage),
				"widen only before an edit an implementer makes at the implement or validate stage")
		}
		want, err := widenPaths(laneID, paths)
		if err != nil {
			return false, err
		}
		if st.ReservedAt == nil {
			res.Note = runID + " holds no reservation, so there is nothing to widen"
			return false, nil
		}
		snap, err := claim.View(repoRoot, now)
		if err != nil {
			return false, refuse(StageReservation, "", laneID, "the reservation on "+st.Key+" cannot be read to widen it: "+fsutil.RedactHome(err.Error()),
				"settle what it names, then run `abcd implement widen` again")
		}
		own, ok := snap.Lookup(st.Key)
		if !ok || own.Copy != claim.CopyThis || (own.State != claim.StateLive && own.State != claim.StateExpired) {
			return false, refuse(StageReservation, "", laneID, "this copy holds no reservation on "+st.Key+" to widen",
				"run `abcd implement step`, whose land fence settles a lost reservation, before editing further")
		}
		res.Footprint = slices.Clone(own.Footprint)
		for _, p := range want {
			if !covers(own.Footprint, []string{p}) {
				res.Added = append(res.Added, p)
			}
		}
		if len(res.Added) == 0 {
			res.Note = "the reservation on " + st.Key + " already holds " + strings.Join(want, ", ")
			return false, nil
		}
		fp := append(slices.Clone(own.Footprint), res.Added...)
		r, err := widenReservation(repoRoot, st.Key, fp, false, false, own, now)
		if err != nil {
			rf, ok := AsRefusal(err)
			if !ok || !rf.Contention {
				return false, err
			}
			note := fmt.Sprintf("the widening of %s's hold to %s backed off: %s; nothing was pushed, and the lane edits none of them",
				st.Key, strings.Join(res.Added, ", "), rf.Reason)
			st.Record = append(st.Record, Entry{At: now, Lane: laneID, Stage: StageReservation, Note: note})
			st.UpdatedAt = now
			backedOff = contend(StageReservation, CheckReservation, laneID, note,
				"edit none of them; stop without a receipt, and the lane is handed to an implementer again on a later step")
			return true, nil
		}
		res.Footprint = fp
		res.Note = r.note + ", widened to " + strings.Join(res.Added, ", ") + " before the lane's implementer edits it"
		for _, p := range res.Added {
			st.Record = append(st.Record, Entry{At: now, Lane: laneID, Stage: StageWidenedTo, Note: p})
		}
		st.Record = append(st.Record, Entry{At: now, Lane: laneID, Stage: StageReservation, Note: res.Note})
		st.UpdatedAt = now
		return true, nil
	})
	if err != nil {
		return WidenResult{}, err
	}
	if backedOff != nil {
		return WidenResult{}, backedOff
	}
	return res, nil
}

// widenedPaths is each path the run record says the run's lanes widened to.
func widenedPaths(st State) []string {
	var out []string
	for _, e := range st.Record {
		if e.Stage == StageWidenedTo && !slices.Contains(out, e.Note) {
			out = append(out, e.Note)
		}
	}
	return out
}

// widenPaths is paths cleaned, deduplicated and sorted, refused when one is
// empty or a path no reservation may carry.
func widenPaths(laneID string, paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, refuse(StageReservation, "", laneID, "no path was named to widen the hold to",
			"name each repository-relative path the lane is about to edit")
	}
	var out []string
	for _, p := range paths {
		if !claim.ValidFootprintPath(p) {
			return nil, refuse(StageReservation, "", laneID, termsafe.DescribeRefused(p)+" is a path no reservation may carry",
				"name each path relative to the repository root, without `..`, a leading `/` or a control character")
		}
		if p = path.Clean(p); !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	slices.Sort(out)
	return out, nil
}

// unheldGaps is the receipt's backstop: a gap naming each path lane's branch
// changes, at its tip, that the issue run's reservation does not cover. An
// intent's run, a run holding no reservation, and a reservation the view
// does not show this copy holding (the land fence's to settle) give none.
// The paths the landing's own records commit changed are left out
// (landingRecordPaths): a lane leaving its landing after that commit, its
// pull request having started to conflict, carries the loop's move of the
// issue's record, which no implementer made and no hold names.
func unheldGaps(c Context, lane Lane, now time.Time) []string {
	st := c.State
	own, paths, gap := unheldPaths(c, lane, now)
	if gap != "" {
		return []string{gap}
	}
	if len(paths) == 0 {
		return nil
	}
	unheld := make([]string, len(paths))
	quoted := make([]string, len(paths))
	for i, p := range paths {
		unheld[i] = termsafe.Sanitize(p)
		quoted[i] = shellquote.Single(unheld[i])
	}
	return []string{fmt.Sprintf("a hold on %s, which the lane's branch changes beyond its reservation on %s (%s): run `%s` from the run's checkout before the edit, or take the change out of the branch",
		strings.Join(unheld, ", "), st.Key, strings.Join(own.Footprint, ", "), widenCommand(st.RunID, lane.ID, strings.Join(quoted, " ")))}
}

// unheldPaths is the reading unheldGaps and holdLaneChanges share: the reservation
// this copy holds on the issue, and each path lane's branch changes at its tip,
// as git names it, that the reservation does not cover. gap, set when one of them
// cannot be read, says what is missing in a receipt's words. See unheldGaps for
// what gives none.
func unheldPaths(c Context, lane Lane, now time.Time) (own claim.Reservation, unheld []string, gap string) {
	st := c.State
	if st.Issue() == "" || st.ReservedAt == nil {
		return own, nil, ""
	}
	snap, err := claim.View(c.RepoRoot, now)
	if err != nil {
		return own, nil, "a reservation that can be read to check the files the lane changed (" + fsutil.RedactHome(err.Error()) + ")"
	}
	own, ok := snap.Lookup(st.Key)
	if !ok || own.Copy != claim.CopyThis || (own.State != claim.StateLive && own.State != claim.StateExpired) {
		return own, nil, ""
	}
	head, err := branchTip(c, lane)
	if err != nil {
		return own, nil, "a branch whose tip can be read to check the files the lane changed (" + fsutil.RedactHome(err.Error()) + ")"
	}
	changed, declined := laneChangedFiles(c.RepoRoot, lane, head)
	if declined != "" {
		return own, nil, "the files the lane changed, to check them against its reservation (" + declined + ")"
	}
	records, declined := landingRecordPaths(c.RepoRoot, lane)
	if declined != "" {
		return own, nil, "the files the lane changed, to check them against its reservation (" + declined + ")"
	}
	for _, p := range changed {
		if !records[p] && !covers(own.Footprint, []string{p}) {
			unheld = append(unheld, p)
		}
	}
	return own, unheld, ""
}

// holdLaneChanges is the fence's judgement of what the lane's branch changes: no
// receipt stands between a lane at its validators and its landing, so a path its
// reservation came to leave out would otherwise land unjudged. A path the
// reservation does not cover is reserved now, judged for overlap and against the
// cap like any reservation, keeping the tip's term; another's hold on it, or a
// refusal of the reservation, stops act, naming what it met. note is the run
// record's clause for a reservation made, "" when every path was held.
func holdLaneChanges(c Context, lane Lane, act fenceAct) (note string, err error) {
	own, paths, gap := unheldPaths(c, lane, c.Now)
	if gap != "" {
		return "", refuse(string(StageLand), "", lane.ID, "the land fence is missing "+gap+", so this lane does not "+act.not,
			"settle what it names, then run `abcd implement step` again; "+act.stays)
	}
	if len(paths) == 0 {
		return "", nil
	}
	shown := make([]string, len(paths))
	for i, p := range paths {
		shown[i] = termsafe.Sanitize(p)
	}
	named := strings.Join(shown, ", ")
	r, err := widenReservation(c.RepoRoot, c.State.Key, heldWith(own.Footprint, paths), false, false, own, c.Now)
	if err != nil {
		reason := fsutil.RedactHome(err.Error())
		rf, ok := AsRefusal(err)
		if ok {
			reason = rf.Reason
		}
		what := fmt.Sprintf("the lane's branch changes %s beyond its reservation on %s, which could not be reserved: %s; so this lane does not %s",
			named, c.State.Key, reason, act.not)
		if ok && rf.Contention {
			return "", contend(string(StageLand), "", lane.ID, what,
				act.stays+"; wait for the reservation it names to end, or settle it with its holder, then run `abcd implement step` again")
		}
		return "", refuse(string(StageLand), "", lane.ID, what, "settle what it names, then run `abcd implement step` again; "+act.stays)
	}
	return r.note + ", widened to " + named + ", which the lane's branch changes beyond it", nil
}

// landingRecordPaths is the set of paths lane's landing records commit
// changed against its parent, or nil when the landing has made none.
func landingRecordPaths(repoRoot string, lane Lane) (map[string]bool, string) {
	if lane.Landing == nil || !gitutil.IsFullSHA(lane.Landing.Records) {
		return nil, ""
	}
	out, err := gitutil.RunCappedBytes(repoRoot, maxGitOutput, "diff-tree", "--no-commit-id", "--no-renames", "--name-only", "-r", "-z", lane.Landing.Records, "--")
	if err != nil {
		return nil, "git could not read the files the landing's records commit " + shortSHA(lane.Landing.Records) + " changed"
	}
	paths := map[string]bool{}
	for _, raw := range bytes.Split(out, []byte{0}) {
		if p := string(raw); p != "" {
			paths[p] = true
		}
	}
	return paths, ""
}

// widenCommand is the widen step's command line for runID's lane laneID over
// paths, spelt with an abcd the implementer can run from the run's checkout:
// `go run ./cmd/abcd` when this abcd is a build of a source checkout's tree
// (go run's, or scripts/abcd.sh's), whose executable go removes or the tree
// outgrows; the running binary's absolute path otherwise, since abcd need not
// be on PATH (a plugin install); bare `abcd` when the system cannot name it.
func widenCommand(runID, laneID, paths string) string {
	return abcdCommand() + " implement widen --run " + runID + " --lane " + laneID + " " + paths
}

// abcdCommand is how a command line the loop renders for an agent names abcd
// (widenCommand).
func abcdCommand() string {
	if sourceBuild() {
		return "go run ./cmd/abcd"
	}
	if p, err := os.Executable(); err == nil && filepath.IsAbs(p) {
		return shellquote.Single(filepath.Clean(p))
	}
	return "abcd"
}

// heldWith is fp with every path of held that fp does not cover appended.
func heldWith(fp, held []string) []string {
	out := slices.Clone(fp)
	for _, p := range held {
		if !covers(out, []string{p}) {
			out = append(out, p)
		}
	}
	return out
}
