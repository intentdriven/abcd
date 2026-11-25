package loop

import (
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/claim"
)

// A bundle's lane read from its pull request (itd-2611201450052326,
// spc-2611201455051897 step 4): a merged bundle is done for every member it
// carried to the merge, a member handed back alone before then staying handed
// back; a pull request closed unmerged is a person's word on the whole lane,
// so every issue still held is handed back as closed and tombstoned once; and
// a member's route is found on the lane that carried it.

// parkedMembers are the two members the parked lane of secondIssue carries in
// these tests, each reserved with a path of its own.
var parkedMembers = []string{bundleTwo, bundleThree}

// parkBundleLane parks the drain's lane of secondIssue at land with its pull
// request open, then gives its run the two members, each reserved on origin
// and recorded as the run's start records it, and names them on the drain's
// lane. With records, each member's record is filed on origin's default
// branch, major so the drain takes neither up, since a tombstone names its
// hand-back's kind only over a record origin holds; a test that merges the
// lane's head leaves them out, the default branch moving only by that merge.
// With alone an issue of the run's, the lead or a member, that issue is
// handed back alone before the pull request is read, its reservation
// tombstoned, as a fix round's receipt leaves it.
func parkBundleLane(t *testing.T, records bool, alone string) (*landFixture, Lane) {
	t.Helper()
	f, l := parkDrainLane(t)
	if records {
		for _, id := range parkedMembers {
			fileIssueAt(t, f.repo, id, capture.SeverityMajor, capture.Category("documentation"), "Correct the sentence.", "AGENTS.md")
		}
		f.repo.Commit("the members' records")
		f.repo.Git("push", "-q", "origin", "main")
		f.repo.Git("fetch", "-q", "origin")
	}
	held := time.Now().UTC().Truncate(time.Second)
	for _, id := range parkedMembers {
		if _, err := claim.Reserve(f.repo.Root(), claim.Request{Record: id, Footprint: []string{bundleFiles[id]}, Lease: 2 * time.Hour}, clockAt(held)); err != nil {
			t.Fatal(err)
		}
	}
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		for _, id := range parkedMembers {
			st.Bundle = append(st.Bundle, Member{Issue: id, Footprint: []string{bundleFiles[id]}, ReservedAt: &held})
			st.Record = append(st.Record, Entry{At: held, Stage: StageReservation, Note: "reserved " + id, Issue: id})
		}
		if alone != "" {
			handBackOneAlone(f.repo.Root(), st, alone, held)
		}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	d, live, err := readDrain(f.repo.Root())
	if err != nil || !live {
		t.Fatalf("a live drain: %v", err)
	}
	i := slices.IndexFunc(d.Lanes, func(dl DrainLane) bool { return dl.RunID == f.runID })
	if i < 0 {
		t.Fatalf("the drain names the parked lane: %+v", d.Lanes)
	}
	d.Lanes[i].Bundle = slices.Clone(parkedMembers)
	if err := writeDrain(f.repo.Root(), d); err != nil {
		t.Fatal(err)
	}
	return f, l
}

// handBackOneAlone hands id back alone on st as a fix round's receipt does,
// and releases it with its own kind.
func handBackOneAlone(repoRoot string, st *State, id string, at time.Time) {
	handBackAlone(st, st.Lanes[0].ID, []issueHandedBack{{issue: id,
		hb: HandBack{Kind: HandBackUserVisible, Reason: "the correction changes what a reader is told"}}}, at)
	releaseAlone(repoRoot, st, at)
}

// releasedOnce reports whether origin's reservation on id ends in a tombstone
// whose parent is not one: the issue was released, and only once.
func releasedOnce(t *testing.T, bare, id string) bool {
	t.Helper()
	tip := tipOf(t, bare, id)
	return tip.released && tip.parent != "" && !reservationCommit(t, bare, tip.parent).released
}

// TestAMergedBundleIsDoneForEveryMember: once the bundle's pushed head is on
// the default branch, the drain's move finishes its lane for the lead and
// both members, names each, marks the drain's lane done, and tombstones every
// issue the run held.
func TestAMergedBundleIsDoneForEveryMember(t *testing.T) {
	f, l := parkBundleLane(t, false, "")
	f.merged(t, l.Landing.Pushed)

	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Finished) != 1 || !res.Finished[0].Done || res.Finished[0].Issue != secondIssue || !slices.Equal(res.Finished[0].Bundle, parkedMembers) {
		t.Fatalf("the merged bundle is finished for its lead and both members: %+v", res.Finished)
	}
	i := slices.IndexFunc(res.Lanes, func(dl DrainLane) bool { return dl.RunID == f.runID })
	if i < 0 || res.Lanes[i].Outcome != DrainLaneDone || res.Lanes[i].PR != 7 {
		t.Fatalf("the drain marks the bundle's lane done with its pull request: %+v", res.Lanes)
	}
	for _, id := range append([]string{secondIssue}, parkedMembers...) {
		if !releasedOnce(t, f.bare, id) {
			t.Fatalf("finishing the merged bundle tombstones %s once: %+v", id, tipOf(t, f.bare, id))
		}
		if !strings.Contains(res.Next, id) {
			t.Fatalf("next names %s among the finished: %s", id, res.Next)
		}
	}
}

// TestAMemberHandedBackAloneBeforeTheMergeStaysHandedBack: the member handed
// back alone in a fix round is not finished with the lane: the finish names
// the lead and the other member alone, the run keeps the member's hand-back,
// and origin keeps the tombstone of its own kind, pushed nothing over it.
func TestAMemberHandedBackAloneBeforeTheMergeStaysHandedBack(t *testing.T) {
	f, l := parkBundleLane(t, false, bundleThree)
	alone := tipOf(t, f.bare, bundleThree)
	if !alone.released {
		t.Fatalf("the member handed back alone is tombstoned: %+v", alone)
	}
	f.merged(t, l.Landing.Pushed)

	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Finished) != 1 || !res.Finished[0].Done || !slices.Equal(res.Finished[0].Bundle, []string{bundleTwo}) {
		t.Fatalf("the merge finishes the lead and the member still carried, not the one handed back: %+v", res.Finished)
	}
	if strings.Contains(finishedNote(res.Finished), bundleThree) {
		t.Fatalf("the finish does not name the member handed back alone as finished: %s", finishedNote(res.Finished))
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if m := st.member(bundleThree); m < 0 || st.Bundle[m].HandBack == nil || st.Bundle[m].HandBack.Kind != HandBackUserVisible {
		t.Fatalf("the run keeps the member's hand-back: %+v", st.Bundle)
	}
	if tip := tipOf(t, f.bare, bundleThree); tip.commit != alone.commit {
		t.Fatalf("the finish pushes nothing over the member handed back alone: before %+v, now %+v", alone, tip)
	}
	for _, id := range []string{secondIssue, bundleTwo} {
		if !releasedOnce(t, f.bare, id) {
			t.Fatalf("finishing the merged bundle tombstones %s once: %+v", id, tipOf(t, f.bare, id))
		}
	}
}

// TestAClosedBundleHandsBackEachHeldIssueAsClosed: a pull request closed
// unmerged hands the bundle's lane back as closed, its record naming every
// issue still held, and tombstones each of them once with the closed kind; a
// member handed back alone before keeps its own tombstone, and a later move
// pushes nothing more.
func TestAClosedBundleHandsBackEachHeldIssueAsClosed(t *testing.T) {
	for name, tc := range map[string]struct {
		alone bool
		held  []string
	}{
		"every member held":           {false, parkedMembers},
		"a member handed back before": {true, []string{bundleTwo}},
	} {
		t.Run(name, func(t *testing.T) {
			alone := ""
			if tc.alone {
				alone = bundleThree
			}
			f, l := parkBundleLane(t, true, alone)
			aloneTip := tipOf(t, f.bare, bundleThree)
			if kind, _ := tombstoneTrailers(t, f.bare, bundleThree); tc.alone && kind != HandBackUserVisible {
				t.Fatalf("the member handed back alone is tombstoned with its own kind: %q", kind)
			}
			f.listPRs(t, stubPR{Number: 7, HeadRefName: l.Branch, State: "CLOSED", Mergeable: "UNKNOWN", MergeStateStatus: "UNKNOWN"})

			res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
			if err != nil {
				t.Fatal(err)
			}
			p := parkedOf(t, res, f.runID)
			if p.State != ParkedClosed || !slices.Equal(p.Bundle, tc.held) {
				t.Fatalf("the closed bundle is read as closed, naming the members it holds: %+v", p)
			}
			st, err := ReadState(f.repo.Root(), f.runID)
			if err != nil {
				t.Fatal(err)
			}
			if hb := st.Lanes[0].HandBack; st.Lanes[0].Stage != StageHandedBack || hb == nil || hb.Kind != claim.HandBackClosed {
				t.Fatalf("the bundle's lane is handed back as closed: %+v", st.Lanes[0])
			}
			for _, id := range append([]string{secondIssue}, tc.held...) {
				if !releasedOnce(t, f.bare, id) {
					t.Fatalf("%s is tombstoned once: %+v", id, tipOf(t, f.bare, id))
				}
				if kind, _ := tombstoneTrailers(t, f.bare, id); kind != claim.HandBackClosed {
					t.Fatalf("%s's tombstone marks it closed: %q", id, kind)
				}
				if !strings.Contains(p.Note, id) {
					t.Fatalf("the hand-back names %s: %s", id, p.Note)
				}
			}
			if tc.alone {
				if tip := tipOf(t, f.bare, bundleThree); tip.commit != aloneTip.commit {
					t.Fatalf("the member handed back alone keeps its own tombstone: before %+v, now %+v", aloneTip, tip)
				}
				if strings.Contains(p.Note, bundleThree) {
					t.Fatalf("the closed hand-back does not name the member handed back before: %s", p.Note)
				}
			}
			tips := tipsOf(t, f.bare)
			if _, err := Drain(f.repo.Root(), Options{}, DrainOptions{}); err != nil {
				t.Fatal(err)
			}
			for id, tip := range tipsOf(t, f.bare) {
				if tip.commit != tips[id].commit {
					t.Fatalf("a later move pushes nothing more over %s: before %+v, now %+v", id, tips[id], tip)
				}
			}
		})
	}
}

// TestRoutedLaneFindsALaneByItsLeadOrAMember: a lane's hand-back of a member
// maps to the lane that carried it, as the lead's does; an issue on no lane,
// and a route no lane made, map to none.
func TestRoutedLaneFindsALaneByItsLeadOrAMember(t *testing.T) {
	opened := time.Date(2026, 11, 25, 2, 0, 0, 0, time.UTC)
	at := opened.Add(time.Hour)
	st := DrainState{Lanes: []DrainLane{
		{Issue: eligibleIssue, RunID: "run-a", OpenedAt: opened.Add(-time.Hour), PR: 6},
		{Issue: bundleLead, RunID: "run-b", OpenedAt: opened, PR: 7, Bundle: []string{bundleTwo, bundleThree}},
	}}
	for _, id := range []string{bundleLead, bundleTwo, bundleThree} {
		if l, ok := routedLane(st, DrainRoute{Issue: id, From: "lane", At: &at}); !ok || l.RunID != "run-b" || l.PR != 7 {
			t.Fatalf("%s's route maps to the bundle's lane: %+v %v", id, l, ok)
		}
	}
	if l, ok := routedLane(st, DrainRoute{Issue: eligibleIssue, From: "lane", At: &at}); !ok || l.RunID != "run-a" {
		t.Fatalf("a lane of one maps as before: %+v %v", l, ok)
	}
	if _, ok := routedLane(st, DrainRoute{Issue: secondIssue, From: "lane", At: &at}); ok {
		t.Fatal("an issue no lane carried maps to none")
	}
	if _, ok := routedLane(st, DrainRoute{Issue: bundleTwo, From: "judgement", At: &at}); ok {
		t.Fatal("a route no lane made maps to none")
	}
	early := opened.Add(-time.Minute)
	if _, ok := routedLane(st, DrainRoute{Issue: bundleTwo, From: "lane", At: &early}); ok {
		t.Fatal("a member's route before its lane opened maps to none")
	}
}

// TestTextWrittenReadsEveryMemberID: a drain state whose lane names a member,
// or whose forming bundle names a lead, a member or a hold, with a rune the
// drain never writes is refused as it is read; the same state written clean is
// not.
func TestTextWrittenReadsEveryMemberID(t *testing.T) {
	hostile := bundleTwo + "‮"
	clean := func() DrainState {
		return DrainState{
			Lanes:   []DrainLane{{Issue: bundleLead, RunID: "run-b", Bundle: []string{bundleTwo, bundleThree}}},
			Forming: &DrainForming{Lead: eligibleIssue, Hold: &DrainHold{Commit: strings.Repeat("a", 40)}, Members: []DrainFormingMember{{Issue: secondIssue, Hold: &DrainHold{Commit: strings.Repeat("b", 40)}}}},
		}
	}
	if !clean().textWritten() {
		t.Fatal("a drain state the drain writes is admitted")
	}
	for name, spoil := range map[string]func(*DrainState){
		"a lane's member":       func(st *DrainState) { st.Lanes[0].Bundle[1] = hostile },
		"a forming lead":        func(st *DrainState) { st.Forming.Lead = hostile },
		"a forming member":      func(st *DrainState) { st.Forming.Members[0].Issue = hostile },
		"a forming lead's hold": func(st *DrainState) { st.Forming.Hold.Commit = hostile },
		"a forming member's hold": func(st *DrainState) {
			st.Forming.Members[0].Hold.ForkAllowed = hostile
		},
	} {
		st := clean()
		spoil(&st)
		if st.textWritten() {
			t.Fatalf("%s carrying a rune the drain never writes is refused", name)
		}
	}
}

// TestALeadHandedBackAloneIsNeitherFinishedNorHandedBackAgain: the lead
// handed back alone in a fix round is not the lane's issue once its pull
// request is read: a merge finishes the two members alone, naming the lane by
// its run, and a closed pull request hands back the two members alone, the
// lead keeping its own tombstone.
func TestALeadHandedBackAloneIsNeitherFinishedNorHandedBackAgain(t *testing.T) {
	t.Run("merged", func(t *testing.T) {
		f, l := parkBundleLane(t, false, secondIssue)
		lead := tipOf(t, f.bare, secondIssue)
		if !lead.released {
			t.Fatalf("the lead handed back alone is tombstoned: %+v", lead)
		}
		f.merged(t, l.Landing.Pushed)

		res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Finished) != 1 || !res.Finished[0].Done || res.Finished[0].Issue != "" || !slices.Equal(res.Finished[0].Bundle, parkedMembers) {
			t.Fatalf("the merge finishes the two members, not the lead handed back alone: %+v", res.Finished)
		}
		note := finishedNote(res.Finished)
		if strings.Contains(note, secondIssue) || !strings.Contains(note, "run "+f.runID) {
			t.Fatalf("the finish names the lane by its run and its members, not the lead: %s", note)
		}
		if tip := tipOf(t, f.bare, secondIssue); tip.commit != lead.commit {
			t.Fatalf("the finish pushes nothing over the lead handed back alone: before %+v, now %+v", lead, tip)
		}
		for _, id := range parkedMembers {
			if !releasedOnce(t, f.bare, id) {
				t.Fatalf("finishing the merged bundle tombstones %s once: %+v", id, tipOf(t, f.bare, id))
			}
		}
	})
	t.Run("closed", func(t *testing.T) {
		f, l := parkBundleLane(t, true, secondIssue)
		lead := tipOf(t, f.bare, secondIssue)
		f.listPRs(t, stubPR{Number: 7, HeadRefName: l.Branch, State: "CLOSED", Mergeable: "UNKNOWN", MergeStateStatus: "UNKNOWN"})

		res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
		if err != nil {
			t.Fatal(err)
		}
		p := parkedOf(t, res, f.runID)
		if p.State != ParkedClosed || p.Issue != "" || !slices.Equal(p.Bundle, parkedMembers) {
			t.Fatalf("the closed bundle names the two members it holds, not the lead: %+v", p)
		}
		if strings.Contains(p.Note, secondIssue) {
			t.Fatalf("the closed hand-back does not name the lead handed back before: %s", p.Note)
		}
		if tip := tipOf(t, f.bare, secondIssue); tip.commit != lead.commit {
			t.Fatalf("the lead handed back alone keeps its own tombstone: before %+v, now %+v", lead, tip)
		}
		for _, id := range parkedMembers {
			if kind, _ := tombstoneTrailers(t, f.bare, id); !releasedOnce(t, f.bare, id) || kind != claim.HandBackClosed {
				t.Fatalf("%s is tombstoned once as closed: %+v %q", id, tipOf(t, f.bare, id), kind)
			}
		}
	})
}

// TestAClosedBundleNamesWhatItsLockedSaveHandsBack: a member handed back
// alone after the move listed the parked runs, and before the closed pull
// request's hand-back takes the run's lock, is left out of the issues the
// reading names, as the save hands them back.
func TestAClosedBundleNamesWhatItsLockedSaveHandsBack(t *testing.T) {
	f, l := parkBundleLane(t, true, "")
	f.listPRs(t, stubPR{Number: 7, HeadRefName: l.Branch, State: "CLOSED", Mergeable: "UNKNOWN", MergeStateStatus: "UNKNOWN"})
	was := afterParkedFetch
	t.Cleanup(func() { afterParkedFetch = was })
	afterParkedFetch = func() {
		if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
			handBackOneAlone(f.repo.Root(), st, bundleThree, time.Now().UTC().Truncate(time.Second))
			return true, nil
		}); err != nil {
			t.Fatal(err)
		}
	}

	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	p := parkedOf(t, res, f.runID)
	if p.State != ParkedClosed || p.Issue != secondIssue || !slices.Equal(p.Bundle, []string{bundleTwo}) {
		t.Fatalf("the reading names what the locked save handed back, not the listing: %+v", p)
	}
	if kind, _ := tombstoneTrailers(t, f.bare, bundleThree); kind != HandBackUserVisible {
		t.Fatalf("the member handed back meanwhile keeps its own tombstone: %q", kind)
	}
}

// TestALaneOfOnesHandBackLineNamesItsIssue: the record's line for a lane of
// one issue handed back as closed opens with the issue, not a blank key.
func TestALaneOfOnesHandBackLineNamesItsIssue(t *testing.T) {
	f, l := parkDrainLane(t)
	f.listPRs(t, stubPR{Number: 7, HeadRefName: l.Branch, State: "CLOSED", Mergeable: "UNKNOWN", MergeStateStatus: "UNKNOWN"})
	if _, err := Drain(f.repo.Root(), Options{}, DrainOptions{}); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	lines := linesAt(st, string(StageHandedBack))
	if len(lines) == 0 || !strings.HasPrefix(lines[len(lines)-1].Note, secondIssue+" is handed back as closed") {
		t.Fatalf("the hand-back's line opens with the issue: %+v", lines)
	}
}
