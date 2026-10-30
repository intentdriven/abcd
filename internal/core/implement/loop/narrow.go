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
// (reserveOnLeavingLand), since a fix round may follow. Only an issue run
// narrows: an intent's run holds one reservation for every lane of its spec,
// and a lane not yet opened has changed nothing.

import (
	"bytes"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/claim"
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
// narrowing that left it out would free a file the lane still changes.
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
	return fp, true, ""
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
	base, err := gitutil.Run(repoRoot, "merge-base", head, "refs/remotes/"+Remote+"/"+def)
	if base = strings.TrimSpace(base); err != nil || !gitutil.IsFullSHA(base) {
		return nil, "git could not read the merge base of the lane's head " + shortSHA(head) + " and " + tracking
	}
	out, err := gitutil.RunCappedBytes(repoRoot, maxGitOutput, "diff", "--no-renames", "--name-only", "-z", base, head, "--")
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
// dcn-2610292238191496): the files each lane of the run still at work
// changed, read as laneChangedFiles reads them, at its branch's tip, which
// holds the implementer's commits before any receipt records a head. It reads
// them only once no spec step is pending and every lane is past all editing
// (pastEditing): an implementer's edits stay uncommitted until its receipt, so
// the tip of a lane that has one at work, or may hand one out next, does not
// show them, and a lane yet to open has changed nothing it can name. Until
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
// state not named here (held, an unset stage, one still at or before its
// implement stage) counts as editing. A lane is past editing when it is done,
// discarded or handed back, at its landing (a landing's conflicting sync sends
// it back to validate first), or at validate awaiting its reviewers alone: no
// implementer out (a fix round's, or a sync conflict's), no sync conflict
// pending, and no fix round its findings hand an implementer next.
func pastEditing(st State, i int) bool {
	l := st.Lanes[i]
	switch l.Stage {
	case StageDone, StageDiscarded, StageHandedBack, StageLand:
		return true
	case StageValidate:
		return !l.awaitsRole(RoleImplementer) && l.pendingSync() == nil && laneWant(st, i).role != RoleImplementer
	}
	return false
}
