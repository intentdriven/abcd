package loop

// drain_readmit.go is a drain's re-admission of the issues it handed back
// (spc-2610161044461448 step 16; iss-2610161015457468; dcn-2610161040399117).
// A drain hands an issue back to a person, and passes it over for the rest of
// its life while the record it judged stands. A changed record is a new
// question: each route keeps the blob id of the issue's record on origin's
// default branch as it was routed, and once that record's blob there differs,
// the issue is the drain's to consider again, judged afresh, as the tombstone
// on origin lets every other account's drain consider it (handback.go in the
// claim package). The issue run a handed-back lane leaves behind is over: a
// drain neither passes the issue over for it nor resumes it, and opens a new
// run when the issue's lane opens again.

import (
	"time"

	"github.com/intentdriven/abcd/internal/core/peers"
)

// issueHandedBack reports whether s is an issue run whose lane was handed back:
// nothing further runs in it, and its issue is a person's.
func (s State) issueHandedBack() bool {
	outcome, _, _ := laneOutcome(s)
	return s.Issue() != "" && outcome == DrainLaneHandedBack
}

// originBlobs reads, once per move and only when asked, the blob id of each
// issue's record on origin's default branch at the commit the move planned
// from; "" for an issue origin holds in no one file, and for every issue
// without origin or when the tree cannot be read.
type originBlobs struct {
	repoRoot string
	origin   *DrainOrigin
	read     bool
	blobs    map[string]string
}

func (b *originBlobs) of(issue string) string {
	if b.origin == nil {
		return ""
	}
	if !b.read {
		b.read, b.blobs = true, map[string]string{}
		if recs, err := peers.RecordsAt(b.repoRoot, b.origin.Commit); err == nil {
			for id, rs := range recs {
				if len(rs) == 1 {
					b.blobs[id] = rs[0].Blob
				}
			}
		}
	}
	return b.blobs[issue]
}

// lastHandBack is the latest route the drain made for issue, nil when none.
func (st DrainState) lastHandBack(issue string) *DrainRoute {
	for i := len(st.HandBacks) - 1; i >= 0; i-- {
		if st.HandBacks[i].Issue == issue {
			return &st.HandBacks[i]
		}
	}
	return nil
}

// taken reports whether the drain has issue in hand: a lane it opened that
// was not handed back, or a hand-back whose record has not changed on origin
// since it was routed. A route that kept no blob (no origin, or a record
// origin held in no one file) is never re-admitted, nor is one whose record
// origin now holds in no one file.
func (st DrainState) taken(issue string, blobs *originBlobs) bool {
	handedBack := false
	for _, l := range st.Lanes {
		if l.Issue != issue {
			continue
		}
		if l.Outcome != DrainLaneHandedBack {
			return true
		}
		handedBack = true
	}
	r := st.lastHandBack(issue)
	if r == nil {
		return handedBack
	}
	now := blobs.of(issue)
	return r.Blob == "" || now == "" || now == r.Blob
}

// handedBackAt is when the drain last routed issue back, nil when it has not
// or the route kept no time: a judgement taken before then judged a record
// since changed, so it decides nothing now.
func (st DrainState) handedBackAt(issue string) *time.Time {
	if r := st.lastHandBack(issue); r != nil {
		return r.At
	}
	return nil
}
