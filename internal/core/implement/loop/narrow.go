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
	def, err := defaultBranch(Context{RepoRoot: repoRoot}, l)
	if err != nil {
		return nil, false, "the default branch on " + Remote + " cannot be read"
	}
	tracking := Remote + "/" + termsafe.Sanitize(def)
	base, err := gitutil.Run(repoRoot, "merge-base", l.HeadSHA, "refs/remotes/"+Remote+"/"+def)
	if base = strings.TrimSpace(base); err != nil || !gitutil.IsFullSHA(base) {
		return nil, false, "git could not read the merge base of the lane's head " + shortSHA(l.HeadSHA) + " and " + tracking
	}
	out, err := gitutil.RunCappedBytes(repoRoot, maxGitOutput, "diff", "--no-renames", "--name-only", "-z", base, l.HeadSHA, "--")
	if err != nil {
		return nil, false, "git could not read the files the lane changed since its merge base with " + tracking
	}
	for _, raw := range bytes.Split(out, []byte{0}) {
		p := string(raw)
		if p == "" {
			continue
		}
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
