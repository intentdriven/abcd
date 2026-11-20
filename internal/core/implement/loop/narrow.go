package loop

// narrow.go narrows a lane's reservation to the files its branch changed
// (spc-2610161044461448 step 12; iss-2610152125101468; dcn-2610161040385152;
// dcn-2610170919040486, dcn-2610170947246937).
// A drain reserves an issue with the path its found_at names, which may be a
// whole directory, and the overlap check then holds every other account off
// every issue inside it for the lane's whole life. Once the lane's validators
// have passed and it is at its landing, its branch says which files the work
// changed, and no fix round is left to change another, so its next renewal
// writes those files, the ones within the reservation's footprint, in place of
// the directory: lock de-escalation, which never widens a hold
// (claim.Request.Narrow refuses to). A lane its landing's sync sends back to
// a fresh round reserves its whole found_at again before it goes on
// (reserveOnLeavingLand), since a fix round may follow. An intent's run holds
// one reservation for every lane of its spec, and a lane not yet opened has
// changed nothing, so it narrows another way: once a step has landed, its
// next renewal drops what only the landed steps held (landedNarrowing).

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"slices"
	"strconv"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/spec"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// narrowedFootprint is the footprint the renewal of st's reservation own
// narrows to: the files the issue lane changed, as a diff of its head against
// its merge base with origin's default branch, that lie within own's
// footprint, once the lane is at its landing, its validators passed. The diff
// names the lane's own work, a conflict resolved inside a sync's merge commit
// included, and none of the files a merged default branch brought (which a
// log of the lane's commits from its base would list). ok is false when
// there is nothing to narrow: an intent's run, a lane short of its landing, a
// branch or default branch git cannot read, no changed file within the
// footprint, a footprint already that narrow, or a changed file within it
// that no reservation may carry (claim.ValidFootprintPath), since a
// narrowing that left it out would free a file the lane still changes. A
// narrowing whose footprint fails claim.FootprintFits is declined too, the
// hold left as it stands: that measure keeps room in the message for the
// record paths the land fence adds before the push (holdLaneChanges).
// declined, set only when a narrowing was due and could not be made (the
// default branch, the merge base or the diff git could not read, or such a
// path), says why in plain, terminal-safe words, for the renewal's note.
func narrowedFootprint(repoRoot string, st State, own claim.Reservation) (fp []string, ok bool, declined string) {
	i := st.current()
	if st.Issue() == "" || i < 0 {
		return nil, false, ""
	}
	l := st.Lanes[i]
	if l.Stage != StageLand || !gitutil.IsFullSHA(l.HeadSHA) {
		return nil, false, ""
	}
	changed, declined := laneChangedFiles(repoRoot, l, l.HeadSHA)
	if declined != "" {
		return nil, false, declined
	}
	for _, p := range changed {
		in := covers(own.Footprint, []string{p})
		if !claim.ValidFootprintPath(p) {
			if in {
				return nil, false, "the lane changed " + strconv.Quote(p) + ", a path no reservation may carry"
			}
			continue
		}
		p = path.Clean(p)
		if in && !slices.Contains(fp, p) {
			fp = append(fp, p)
		}
	}
	slices.Sort(fp)
	held := slices.Clone(own.Footprint)
	slices.Sort(held)
	if len(fp) == 0 || slices.Equal(fp, held) {
		return nil, false, ""
	}
	if !claim.FootprintFits(fp) {
		return nil, false, fmt.Sprintf("a narrowing to the %d files the lane changed would leave the reservation message no room for the records its landing adds, so the hold stays as it stands", len(fp))
	}
	return fp, true, ""
}

// landedNarrowing is the footprint the renewal of an intent run's
// reservation own narrows to once one of its steps has landed
// (dcn-2611012049050703, dcn-2611201503122480; iss-2611201418535169): the
// run widens its hold to each step's packages as the step's lane opens
// (holdOpeningStep), and narrows it to the steps still in flight as each
// lands, so a run waiting at a step's opening never holds a landed step's
// packages while it waits. Without the narrowing, two runs whose step orders
// cross would each hold the package the other's next step needs and wait on
// each other for ever.
//
// A lane in flight is one neither done nor discarded. What the lanes in
// flight need is each one's step packages (stepPackages, the `## Footprint`
// packages line where the step cannot say), the paths the run record says it
// widened to, the files its branch changed (laneChangedFiles, at its tip),
// and the spec's own record, which each landing marks. The changed files
// count because a lane widens only to a path the hold does not already
// cover, so an edit to a file under another step's packages leaves no
// record. Nothing narrows while a lane in flight is mid-edit: neither still
// at its worktree or brief, where no implementer has been handed out, nor
// past editing (pastEditing), since an implementer's edits stay uncommitted
// until its receipt and its branch does not show them; two automated lanes
// must never edit one file. A path of
// own any of them covers is kept; one above some of them is narrowed to
// those (lock de-escalation); one a landed lane's packages or widenings
// cover, and nothing in flight needs, is dropped; and a path neither side
// names is kept, since the run cannot say which lane needs it. The
// narrowing is a renewal's (claim.Request.Narrow), so it is never judged for
// overlap and never widens; renewReservation makes it whenever it is due,
// not only at half the lease. ok is false
// when there is nothing to narrow: an issue run (narrowedFootprint is its
// narrowing), an auto-plan run, no lane landed yet, a lane mid-edit, a lane
// whose step cannot be read (its footprint is then unknown), a lane past
// editing whose branch or changed files git cannot read, nothing to drop, or nothing left
// to hold, which claim's renewal cannot write (a reservation names at least
// one path), nor a narrowing it would decline (a path no reservation may
// carry, or more than its message fits), which would otherwise push a
// renewal on every step.
func landedNarrowing(repoRoot string, st State, own claim.Reservation) (fp []string, ok bool) {
	if st.Issue() != "" || st.Spec == "" || st.autoPlanLane() != nil {
		return nil, false
	}
	var flight, landed []string
	var changedBy []Lane
	for i, l := range st.Lanes {
		if l.Stage == StageDiscarded {
			continue
		}
		if l.SpecStep <= 0 {
			return nil, false
		}
		pk, err := stepPackages(repoRoot, st.Spec, l.SpecStep)
		if err != nil || len(pk) == 0 {
			return nil, false
		}
		pk = append(pk, laneWidenedPaths(st, l.ID)...)
		switch {
		case l.Stage == StageDone:
			landed = append(landed, pk...)
		case l.Stage == StageWorktree || l.Stage == StageBrief:
			// No implementer has been handed out: the lane has edited
			// nothing, and what it edits later outside the hold is widened
			// to, judged for overlap, at its receipt.
			flight = append(flight, pk...)
		case pastEditing(st, i):
			flight = append(flight, pk...)
			if l.Branch != "" {
				changedBy = append(changedBy, l)
			}
		default:
			// A lane mid-edit may hold edits its branch does not show yet,
			// to a file the hold already covered, so none was recorded as
			// widened to: nothing narrows until it is past editing.
			return nil, false
		}
	}
	if len(landed) == 0 {
		return nil, false
	}
	// A lane past editing may have changed a file under a landed step's
	// packages, which the hold already covered, so it widened to nothing:
	// the files its branch changed are what it needs too.
	for _, l := range changedBy {
		head, err := branchTip(Context{RepoRoot: repoRoot}, l)
		if err != nil {
			return nil, false
		}
		changed, declined := laneChangedFiles(repoRoot, l, head)
		if declined != "" {
			return nil, false
		}
		for _, p := range changed {
			flight = append(flight, path.Clean(p))
		}
	}
	if store, err := spec.Load(repoRoot); err == nil {
		if sp, ok := store.Lookup(st.Spec); ok && sp.Path != "" {
			flight = append(flight, sp.Path)
		}
	}
	need, done := newPathSet(flight), newPathSet(landed)
	keep := pathSet{}
	for _, p := range own.Footprint {
		p = path.Clean(p)
		if need.covers(p) || !done.covers(p) {
			keep.add(p)
			continue
		}
		above := newPathSet([]string{p})
		for _, t := range flight {
			if above.covers(t) {
				keep.add(t)
			}
		}
	}
	for p := range keep {
		if q := path.Dir(p); q != "." && keep.covers(q) {
			delete(keep, p)
		}
	}
	for p := range keep {
		fp = append(fp, p)
	}
	slices.Sort(fp)
	held := make([]string, 0, len(own.Footprint))
	for _, p := range own.Footprint {
		held = append(held, path.Clean(p))
	}
	slices.Sort(held)
	held = slices.Compact(held)
	if len(fp) == 0 || slices.Equal(fp, held) || !claim.FootprintFits(fp) || slices.ContainsFunc(fp, func(p string) bool { return !claim.ValidFootprintPath(p) }) {
		return nil, false
	}
	return fp, true
}

// inFlightFootprint is fp less what only an intent run's landed steps
// needed, by the rule a landing's renewal narrows by (landedNarrowing): what a
// resume reserves, from the record and from this copy's tip alike, so a
// resume never takes back a path the narrowing gave up, and is never refused
// over a hold a peer took on it since. fp is returned as it stands wherever
// landedNarrowing has nothing to narrow.
func inFlightFootprint(repoRoot string, st State, fp []string) []string {
	if in, ok := landedNarrowing(repoRoot, st, claim.Reservation{Footprint: fp}); ok {
		return in
	}
	return fp
}

// laneWidenedPaths is each path the run record says lane laneID widened to.
func laneWidenedPaths(st State, laneID string) []string {
	var out []string
	for _, e := range st.Record {
		if e.Stage == StageWidenedTo && e.Lane == laneID && !slices.Contains(out, e.Note) {
			out = append(out, e.Note)
		}
	}
	return out
}

// laneChangedFiles is the files lane l changed at head, as git names them:
// the diff of head against its merge base with origin's default branch, the
// one reading narrowedFootprint and a lapsed revival's overlap
// (revivalChangedFiles) share. It names the lane's own work, a conflict
// resolved inside a sync's merge commit included, and none of the files a
// merged default branch brought. declined, set when the files cannot be
// read (the default branch, the merge base or the diff), says why in plain,
// terminal-safe words, and files is then nil.
func laneChangedFiles(repoRoot string, l Lane, head string) (files []string, declined string) {
	def, err := defaultBranch(Context{RepoRoot: repoRoot}, l)
	if err != nil {
		return nil, "the default branch on " + Remote + " cannot be read"
	}
	tracking := Remote + "/" + termsafe.Sanitize(def)
	base, err := gitutil.RunCappedAsStored(repoRoot, maxGitOutput, "merge-base", head, "refs/remotes/"+Remote+"/"+def)
	if err != nil || !gitutil.IsFullSHA(base) {
		return nil, "git could not read the merge base of the lane's head " + shortSHA(head) + " and " + tracking
	}
	out, err := gitutil.RunCappedBytesAsStoredContext(context.Background(), repoRoot, "", maxGitOutput, "diff", "--no-renames", "--ignore-submodules=none", "--name-only", "-z", base, head, "--")
	if err != nil {
		return nil, "git could not read the files the lane changed since its merge base with " + tracking
	}
	files = []string{}
	for _, raw := range bytes.Split(out, []byte{0}) {
		if p := string(raw); p != "" {
			files = append(files, p)
		}
	}
	return files, ""
}

// revivalChangedFiles is the files a lapsed revival of st's reservation is
// judged on against the same person's other copy (claim.Request.OwnOverlapOn;
// iss-2610261837062463, dcn-2610291352512058, dcn-2610291925508286,
// dcn-2610292238191496, dcn-2610300422481486): the files each lane of the run still at work
// changed, read as laneChangedFiles reads them, at its branch's tip, which
// holds the implementer's commits before any receipt records a head. It reads
// them only once no spec step is pending and every lane is past all editing
// (pastEditing: done, discarded, handed back, or at its landing with no sync
// pending; dcn-2610300422481486): an implementer's edits stay uncommitted
// until its receipt, so the tip of a lane that has one at work, or may hand
// one out next, does not show them, and a lane yet to open has changed
// nothing it can name. Until
// then nil is returned and the whole footprint is judged, since two automated
// lanes must never edit one file. nil too when any lane's files cannot be
// read; a non-nil empty list when no lane changed a file.
func revivalChangedFiles(repoRoot string, st State) []string {
	if len(st.Pending) > 0 {
		return nil
	}
	for i := range st.Lanes {
		if !pastEditing(st, i) {
			return nil
		}
	}
	files := []string{}
	for _, l := range st.Lanes {
		if l.Stage == StageDone || l.Stage == StageDiscarded || l.Branch == "" {
			continue
		}
		head, err := branchTip(Context{RepoRoot: repoRoot}, l)
		if err != nil {
			return nil
		}
		changed, declined := laneChangedFiles(repoRoot, l, head)
		if declined != "" {
			return nil
		}
		for _, p := range changed {
			if p = path.Clean(p); !slices.Contains(files, p) {
				files = append(files, p)
			}
		}
	}
	slices.Sort(files)
	return files
}

// pastEditing reports whether lane i of st can hold no edit its branch does
// not show, now or by the run's next hand-out: an allow-list, so a stage or
// state not named here (held, validate, an unset stage, one at or before its
// implement stage) counts as editing. A lane is past editing when it is done,
// discarded or handed back, or at its landing with no sync conflict pending.
// A lane at validate is not, even awaiting its reviewers alone: one verdict
// from them hands a fix implementer any file of the footprint
// (dcn-2610300422481486). A lane its landing sends back to a fresh round
// has its run's overlap with the same person's other copy judged again on
// the whole footprint as it leaves (reserveOnLeavingLand).
func pastEditing(st State, i int) bool {
	l := st.Lanes[i]
	switch l.Stage {
	case StageDone, StageDiscarded, StageHandedBack:
		return true
	case StageLand:
		return l.pendingSync() == nil
	}
	return false
}
