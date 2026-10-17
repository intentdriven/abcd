package loop

// narrow.go narrows a lane's reservation to the files its branch changed
// (spc-2610161044461448 step 12; iss-2610152125101468; dcn-2610161040385152).
// A drain reserves an issue with the path its found_at names, which may be a
// whole directory, and the overlap check then holds every other account off
// every issue inside it for the lane's whole life. Once the lane is past its
// implement stage, its branch says which files the work changed, so its next
// renewal writes those files, the ones within the reservation's footprint,
// in place of the directory: lock de-escalation, which never widens a hold
// (claim.Request.Narrow refuses to). Only an issue run narrows: an intent's
// run holds one reservation for every lane of its spec, and a lane not yet
// opened has changed nothing.

import (
	"bytes"
	"path"
	"slices"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// narrowedFootprint is the footprint the renewal of st's reservation own
// narrows to: the files the issue lane's own commits changed, from its base
// to its head, that lie within own's footprint, once the lane is past its
// implement stage. ok is false when there is nothing to narrow: an intent's
// run, a lane still implementing, a branch git cannot read, no changed file
// within the footprint, or a footprint already that narrow.
func narrowedFootprint(repoRoot string, st State, own claim.Reservation) ([]string, bool) {
	i := st.current()
	if st.Issue() == "" || i < 0 {
		return nil, false
	}
	l := st.Lanes[i]
	if (l.Stage != StageValidate && l.Stage != StageLand) || !gitutil.IsFullSHA(l.BaseSHA) || !gitutil.IsFullSHA(l.HeadSHA) {
		return nil, false
	}
	out, err := gitutil.RunCappedBytes(repoRoot, maxGitOutput, "log", "--no-merges", "--no-renames", "--format=", "--name-only", "-z",
		l.BaseSHA+".."+l.HeadSHA, "--")
	if err != nil {
		return nil, false
	}
	var fp []string
	for _, raw := range bytes.Split(out, []byte{0}) {
		p := string(bytes.TrimSpace(raw))
		if p == "" || !fsutil.ValidRelPath(p) {
			continue
		}
		p = path.Clean(p)
		if covers(own.Footprint, []string{p}) && !slices.Contains(fp, p) {
			fp = append(fp, p)
		}
	}
	slices.Sort(fp)
	held := slices.Clone(own.Footprint)
	slices.Sort(held)
	if len(fp) == 0 || slices.Equal(fp, held) {
		return nil, false
	}
	return fp, true
}
