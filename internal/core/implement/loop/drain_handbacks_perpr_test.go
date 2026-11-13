package loop

import (
	"slices"
	"testing"
	"time"
)

// TestDrainCountsHandBacksPerPullRequest (itd-2610301750162657): each pull
// request the drain's lanes opened has its line, one with no hand-back
// included, counting its lane's hand-backs and naming their kinds once; a
// hand-back no pull request carries (the judgement's, a reservation's, a
// lane's that opened none) is counted apart; and an issue taken up again is
// counted against the lane its hand-back came from.
func TestDrainCountsHandBacksPerPullRequest(t *testing.T) {
	t0 := time.Date(2026, 11, 13, 7, 0, 0, 0, time.UTC)
	at := func(m int) *time.Time { v := t0.Add(time.Duration(m) * time.Minute); return &v }
	st := DrainState{
		Lanes: []DrainLane{
			{Issue: "iss-1", RunID: "run-1", OpenedAt: t0, Outcome: DrainLaneHandedBack, PR: 10},
			{Issue: "iss-2", RunID: "run-2", OpenedAt: t0.Add(time.Minute), Outcome: DrainLanePullRequest, PR: 11},
			{Issue: "iss-3", RunID: "run-3", OpenedAt: t0.Add(2 * time.Minute), Outcome: DrainLaneHandedBack},
			// iss-1 taken up again after its first hand-back.
			{Issue: "iss-1", RunID: "run-4", OpenedAt: t0.Add(10 * time.Minute), Outcome: DrainLaneHandedBack, PR: 12},
		},
		HandBacks: []DrainRoute{
			{Issue: "iss-1", From: "lane", Kind: HandBackClosed, Route: RouteHome, At: at(5)},
			{Issue: "iss-3", From: "lane", Kind: HandBackStuck, Route: RouteHome, At: at(6)},
			{Issue: "iss-5", From: DrainFromJudgement, Kind: HandBackTrustRule, Route: RouteDecisionRecord, At: at(7)},
			{Issue: "iss-1", From: "lane", Kind: HandBackStuck, Route: RouteHome, At: at(15)},
			{Issue: "iss-1", From: "lane", Kind: HandBackStuck, Route: RouteHome, At: at(16)},
		},
	}
	c := drainCounts(t.TempDir(), st, nil, nil)
	want := []DrainPRHandBacks{
		{PR: 10, Issue: "iss-1", HandBacks: 1, Kinds: []string{HandBackClosed}},
		{PR: 11, Issue: "iss-2", HandBacks: 0, Kinds: []string{}},
		{PR: 12, Issue: "iss-1", HandBacks: 2, Kinds: []string{HandBackStuck}},
	}
	if !slices.EqualFunc(c.HandBacksByPR, want, func(a, b DrainPRHandBacks) bool {
		return a.PR == b.PR && a.Issue == b.Issue && a.HandBacks == b.HandBacks && slices.Equal(a.Kinds, b.Kinds)
	}) {
		t.Fatalf("hand-backs by pull request = %+v, want %+v", c.HandBacksByPR, want)
	}
	if c.HandBacksNoPR != 2 {
		t.Fatalf("hand-backs with no pull request = %d, want 2 (iss-3's lane opened none; iss-5's was the judgement's)", c.HandBacksNoPR)
	}
	if empty := drainCounts(t.TempDir(), DrainState{}, nil, nil); empty.HandBacksByPR == nil || len(empty.HandBacksByPR) != 0 {
		t.Fatalf("a drain with no pull request lists none, as an empty list: %#v", empty.HandBacksByPR)
	}
}

// TestDrainCountsHandBacksPerPullRequestOverOddState: the counts read a
// state another process may have written, so each odd shape has a fixed
// answer: a lane whose pull request is zero or below has no line and its
// hand-backs count as carried by no pull request; a route with no time
// counts against the latest lane on its issue; two lanes that name one pull
// request share its one line, under the first lane's issue; and a field
// hand-back is carried by no pull request.
func TestDrainCountsHandBacksPerPullRequestOverOddState(t *testing.T) {
	t0 := time.Date(2026, 11, 13, 7, 0, 0, 0, time.UTC)
	at := func(m int) *time.Time { v := t0.Add(time.Duration(m) * time.Minute); return &v }
	st := DrainState{
		Lanes: []DrainLane{
			{Issue: "iss-1", RunID: "run-1", OpenedAt: t0, PR: 0},
			{Issue: "iss-2", RunID: "run-2", OpenedAt: t0, PR: -4},
			{Issue: "iss-3", RunID: "run-3", OpenedAt: t0, PR: 20},
			{Issue: "iss-3", RunID: "run-4", OpenedAt: t0.Add(5 * time.Minute), PR: 21},
			{Issue: "iss-4", RunID: "run-5", OpenedAt: t0, PR: 30},
			{Issue: "iss-5", RunID: "run-6", OpenedAt: t0, PR: 30},
		},
		HandBacks: []DrainRoute{
			{Issue: "iss-1", From: "lane", Kind: HandBackStuck, Route: RouteHome, At: at(1)},
			{Issue: "iss-2", From: "lane", Kind: HandBackStuck, Route: RouteHome, At: at(1)},
			{Issue: "iss-3", From: "lane", Kind: HandBackClosed, Route: RouteHome},
			{Issue: "iss-5", From: "lane", Kind: HandBackStuck, Route: RouteHome, At: at(2)},
			{Issue: "iss-6", From: "field", Kind: HandBackTrustRule, Route: RouteDecisionRecord, At: at(3)},
		},
	}
	c := drainCounts(t.TempDir(), st, nil, nil)
	want := []DrainPRHandBacks{
		{PR: 20, Issue: "iss-3", HandBacks: 0, Kinds: []string{}},
		{PR: 21, Issue: "iss-3", HandBacks: 1, Kinds: []string{HandBackClosed}},
		{PR: 30, Issue: "iss-4", HandBacks: 1, Kinds: []string{HandBackStuck}},
	}
	if !slices.EqualFunc(c.HandBacksByPR, want, func(a, b DrainPRHandBacks) bool {
		return a.PR == b.PR && a.Issue == b.Issue && a.HandBacks == b.HandBacks && slices.Equal(a.Kinds, b.Kinds)
	}) {
		t.Fatalf("hand-backs by pull request = %+v, want %+v", c.HandBacksByPR, want)
	}
	if c.HandBacksNoPR != 3 {
		t.Fatalf("hand-backs with no pull request = %d, want 3 (two lanes with no pull request, one field hand-back)", c.HandBacksNoPR)
	}
}
