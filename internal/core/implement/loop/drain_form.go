package loop

// drain_form.go is the drain forming and opening a bundle
// (itd-2611201450052326, spc-2611201455051897 step 4). After a lead's
// judgement answers no, a drain whose cap is above 1 does not open the lane at
// once when the lead has candidates (bundleCandidates): it records the bundle
// as forming, the lead and its hold, and takes each candidate after it the way
// it takes an issue, passing over one held, skipped, lapsed or already run,
// reserving it before its judgement and asking it. A no adds the candidate to
// the bundle with its hold; a yes routes it as today, and it leaves. The holds
// are kept across moves (judgeHolds.settle leaves what forming records), and
// renewed at half their lease (renewForming). The lane opens when no
// candidate is left, the cap is reached, or the person's live reservations
// would pass reservation.max_live with the next one: the bundle then opens as
// it stands rather than waiting. A drain that ends lets every forming hold go
// plainly, with no tombstone.
//
// A cap of 1, a lead that is not bundleable and a lead with no candidates go
// the way the drain went before bundles, with no forming written.

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/core/landings"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// startBundle starts the lane of lead with members, a seam a test replaces to
// read what a bundle opens with. The start carries the members on the new
// run, each with the hold it was reserved on (Options.members, loop.go); a
// lane with no members starts as startLane's does.
var startBundle = func(repoRoot, lead string, o Options, members []laneMember) (StartResult, error) {
	o.members = members
	return startLane(repoRoot, lead, o)
}

// formingEntry is one issue of a forming bundle and its hold.
type formingEntry struct {
	issue string
	hold  *DrainHold
}

// entries is the forming bundle's issues, the lead first, each with its hold;
// none for a nil bundle.
func (f *DrainForming) entries() []formingEntry {
	if f == nil {
		return nil
	}
	out := []formingEntry{{issue: f.Lead, hold: f.Hold}}
	for _, m := range f.Members {
		out = append(out, formingEntry{issue: m.Issue, hold: m.Hold})
	}
	return out
}

// ids is the forming bundle's issues, the lead first.
func (f *DrainForming) ids() []string {
	var out []string
	for _, e := range f.entries() {
		out = append(out, e.issue)
	}
	return out
}

// has reports whether the forming bundle carries id, as its lead or a member.
func (f *DrainForming) has(id string) bool {
	return slices.ContainsFunc(f.ids(), func(i string) bool { return recordid.SameID(i, id) })
}

// holds reports whether the forming bundle records issue's hold at commit.
func (f *DrainForming) holds(issue, commit string) bool {
	return slices.ContainsFunc(f.entries(), func(e formingEntry) bool {
		return e.hold != nil && e.issue == issue && e.hold.Commit == commit
	})
}

// clone is a copy of the forming bundle that shares nothing with it, so the
// copy a move keeps of what is on disk is not changed by the move's own
// changes to its state.
func (f *DrainForming) clone() *DrainForming {
	if f == nil {
		return nil
	}
	c := &DrainForming{Lead: f.Lead, Hold: cloneHold(f.Hold), Members: make([]DrainFormingMember, len(f.Members))}
	for i, m := range f.Members {
		c.Members[i] = DrainFormingMember{Issue: m.Issue, Hold: cloneHold(m.Hold)}
	}
	return c
}

func cloneHold(h *DrainHold) *DrainHold {
	if h == nil {
		return nil
	}
	c := *h
	return &c
}

// size is how many issues the forming bundle holds, its lead included.
func (f *DrainForming) size() int { return len(f.entries()) }

// former is what one move's forming reads: the drain's state, its plan, how
// it reads an issue's area, and the issues passed over as a bundle's lead, by
// this move or by the drive, which join no bundle (nil passes none over).
type former struct {
	repoRoot string
	st       *DrainState
	plan     capture.DrainPlan
	area     func(id string) string
	passed   func(id string) bool
}

// standing reports whether id still stands in a bundle: it is eligible in the
// plan, its found_at still names a path to reserve it with, and the judgement
// over its remedy as it now stands answered no.
func (m former) standing(id string) bool {
	i := slices.IndexFunc(m.plan.Dispositions, func(v capture.DrainVerdict) bool { return recordid.SameID(v.ID, id) })
	if i < 0 || m.plan.Dispositions[i].Outcome != capture.DrainEligible {
		return false
	}
	iss, ok := m.plan.OpenIssue(m.plan.Dispositions[i].ID)
	if !ok {
		return false
	}
	if _, ok := issueFootprint(m.repoRoot, iss); !ok {
		return false
	}
	j := judgementFor(*m.st, iss.ID, remedyDigest(iss.Remedy))
	return j != nil && j.Answer == JudgementNo
}

// candidates is every issue that would join id's lane, whatever the cap: the
// cap is counted as the bundle forms, since a candidate passed over or
// answered yes leaves room for the next.
func (m former) candidates(id string) []string {
	out := bundleCandidates(m.plan.Dispositions, id, len(m.plan.Dispositions)+1, m.area)
	if m.passed != nil {
		out = slices.DeleteFunc(out, m.passed)
	}
	return out
}

// queue is what a move takes next for the bundle st is forming: the
// candidates after its lead in the drain order that it does not carry yet,
// none once the cap is reached. A member that no longer stands (it left the
// eligible set, or its remedy changed since its no) is let go first, its hold
// released on the move's way out; a lead that no longer stands lets the whole
// bundle go, and ok is false: the move takes the drain's order as before. note
// says what was let go, "" when nothing was.
func (m former) queue() (queue []capture.DrainVerdict, note string, ok bool) {
	f := m.st.Forming
	if !m.standing(f.Lead) {
		m.st.Forming = nil
		return nil, fmt.Sprintf("the bundle forming on %s is let go: %s is no longer eligible, its found_at names no path, or its remedy changed since its judgement, so its reservations are released", f.Lead, f.Lead), false
	}
	var gone []string
	f.Members = slices.DeleteFunc(f.Members, func(mb DrainFormingMember) bool {
		if m.standing(mb.Issue) {
			return false
		}
		gone = append(gone, mb.Issue)
		return true
	})
	if len(gone) > 0 {
		note = fmt.Sprintf("%s left the bundle forming on %s: no longer eligible, its found_at names no path, or its remedy changed since its judgement", strings.Join(gone, ", "), f.Lead)
	}
	if f.size() >= m.st.bundleCap() {
		return nil, note, true
	}
	for _, id := range m.candidates(f.Lead) {
		if f.has(id) {
			continue
		}
		if i := slices.IndexFunc(m.plan.Dispositions, func(v capture.DrainVerdict) bool { return v.ID == id }); i >= 0 {
			queue = append(queue, m.plan.Dispositions[i])
		}
	}
	return queue, note, true
}

// forms reports whether a no over id would form or join a bundle: one is
// forming, or id would lead one, having candidates under a cap above 1.
func (m former) forms(id string) bool {
	return m.st.Forming != nil || m.st.bundleCap() > 1 && len(m.candidates(id)) > 0
}

// join is the forming step for an issue whose judgement answered no, as the
// move reaches the point where its lane would open, on hold (nil when the
// drain took none). With no bundle forming, an issue with candidates under a
// cap above 1 becomes a bundle's lead, and lead is true: the move goes on to
// its candidates. With one forming, the issue joins it as a member, and full
// is true once the bundle has reached the cap. Neither is true for an issue
// whose lane opens alone, as before bundles.
func (m former) join(id string, hold *DrainHold) (lead, full bool) {
	if f := m.st.Forming; f != nil {
		if !f.has(id) {
			f.Members = append(f.Members, DrainFormingMember{Issue: id, Hold: cloneHold(hold)})
		}
		return false, f.size() >= m.st.bundleCap()
	}
	if m.st.bundleCap() <= 1 || len(m.candidates(id)) == 0 {
		return false, false
	}
	m.st.Forming = &DrainForming{Lead: id, Hold: cloneHold(hold), Members: []DrainFormingMember{}}
	return true, false
}

// formingHold is the drain's hold on an issue as the bundle records it: the
// one the judgement was asked on when it is the reservation the view shows
// (own), a routed judge's hold the view, fetched before it, could not show,
// or, with no view to judge by, the one the judgement records; nil when the
// drain holds none. byHand is true when the view shows this copy's live
// reservation at a tip the drain's own hold is not: this person made or
// renewed it by hand, so it is never the drain's (drain_hold.go), and the
// issue does not join a bundle, whose holds the drain renews and releases.
func formingHold(own claim.Reservation, owned, viewed bool, judged *DrainJudgement, routed *DrainHold) (hold *DrainHold, byHand bool) {
	if owned {
		if judged != nil && judged.Hold != nil && judged.Hold.Commit == own.Commit {
			return cloneHold(judged.Hold), false
		}
		return nil, true
	}
	if routed == nil && !viewed && judged != nil {
		routed = judged.Hold
	}
	return cloneHold(routed), false
}

// handHeldPassOver is the pass-over of a candidate whose reservation this person
// holds by hand: it is left as it stands, and the bundle forms without it.
func handHeldPassOver(id, lead string) Excluded {
	return Excluded{ID: id, Check: CheckReservation, Reason: "it is your own reservation from this copy, made or renewed by hand since the drain took its hold, so it does not join the bundle forming on " +
		lead + " and is left as it stands"}
}

// heldOn is the lane's start on the drain's hold h of id, reserved with
// footprint: what the lane records of the reservation, and the fork check it
// could not make or waived.
func heldOn(id string, h DrainHold, footprint []string) *heldReservation {
	held := &heldReservation{at: h.ClaimedAt, note: fmt.Sprintf("reserved %s on %s for this copy until %s before its judgement, footprint: %s",
		id, claim.Remote, stamp(h.ExpiresAt), strings.Join(footprint, ", ")),
		forkUnchecked: h.ForkUnchecked, forkAllowed: h.ForkAllowed}
	if h.ForkUnchecked != "" {
		held.note += "; " + claim.ForkUncheckedLine(h.ForkUnchecked)
	}
	if h.ForkAllowed != "" {
		held.note += "; " + claim.ForkAllowedLine(h.ForkAllowed)
	}
	return held
}

// drainLaneOptions are the options a lane the drain opens starts on: the
// drain's pace, not what the configuration resolves when it opens
// (drain_pace.go); footprint, the issue's found_at path, reserved; the plan
// the move read; and the rule the move loaded and judged the issue by, which
// its records commit names (itd-2610301850454860).
func drainLaneOptions(o Options, st DrainState, footprint []string, plan *capture.DrainPlan, rule drainrule.Rule) Options {
	lane := o
	lane.Pace, lane.SubAgents, lane.FixRounds, lane.pace = nil, nil, nil, st.Pace
	lane.reserve = footprint
	lane.plan = plan
	lane.drain = true
	lane.admitted = &Admitted{Kind: landings.KindDrain, Record: rule.Record, Path: rule.Path}
	return lane
}

// bundleOpened is what opening a forming bundle came to: the lane's start, or
// the lead passed over and why, or the reservations the person is at the cap
// with, or the refusal that stops the move; left says which members left the
// bundle before it opened, and why. A lead passed over names every issue the
// bundle carried as it came to open, the lead first; contention is true when
// another's hold refused its start, and moved when the lead's found_at no
// longer names the footprint its hold reserved, which the next move reserves
// again rather than a drive passing it over.
type bundleOpened struct {
	start      *StartResult
	passed     *Excluded
	issues     []string
	contention bool
	moved      bool
	capped     []string
	err        error
	left       string
}

// reservedFootprint reports whether fp, the footprint an issue's found_at now
// names, is the footprint the drain's hold on it reserved, as view shows the
// tip at hold; true with no view or no hold to judge by.
func reservedFootprint(view *claim.Snapshot, id string, hold *DrainHold, fp []string) bool {
	if view == nil || hold == nil {
		return true
	}
	r, ok := view.Lookup(id)
	if !ok || r.Commit != hold.Commit {
		return true
	}
	a, b := slices.Clone(r.Footprint), slices.Clone(fp)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

// openBundle opens the lane of the bundle st is forming: the lead's lane, with
// each member on the hold it was judged on, every hold kept by the lane once
// it starts. Forming is cleared either way: a start that fails releases every
// hold on the move's way out (judgeHolds.settle, starting), and the
// judgements are kept, so a later move forms the bundle again without asking
// any of them again. A start refused for a reason the drain passes a single
// lane's lead over for passes this lead over, and the move releases the
// bundle's holds and goes on in the drain's order (Drain); a lead whose
// found_at moved ends the move, and the next one forms the bundle again.
func openBundle(repoRoot string, o Options, st *DrainState, plan *capture.DrainPlan, rule drainrule.Rule, holds *judgeHolds, view *claim.Snapshot, now time.Time) bundleOpened {
	f := st.Forming
	st.Forming = nil
	all := f.ids()
	// footprintOf is the footprint id's found_at names now, and whether it
	// is still the one its hold reserved: an issue whose found_at no longer
	// names a path, or names another since it was reserved, is not opened on
	// that hold.
	footprintOf := func(id string, hold *DrainHold) ([]string, bool) {
		iss, ok := plan.OpenIssue(id)
		if !ok {
			return nil, false
		}
		fp, ok := issueFootprint(repoRoot, iss)
		return fp, ok && reservedFootprint(view, id, hold, fp)
	}
	leadFP, ok := footprintOf(f.Lead, f.Hold)
	if !ok {
		return bundleOpened{issues: all, moved: true, passed: &Excluded{ID: f.Lead, Check: CheckReservation, Reason: "its found_at names no path, or another than its reservation's footprint, since the drain reserved it" +
			"; the bundle with " + strings.Join(f.ids()[1:], ", ") + " did not open, and its reservations are released"}}
	}
	var left []string
	f.Members = slices.DeleteFunc(f.Members, func(m DrainFormingMember) bool {
		if _, ok := footprintOf(m.Issue, m.Hold); ok {
			return false
		}
		left = append(left, m.Issue)
		return true
	})
	leftNote := ""
	if len(left) > 0 {
		leftNote = fmt.Sprintf("%s left the bundle on %s before its lane opened: its found_at names no path, or another than its reservation's footprint, since the drain reserved it, so its reservation is released",
			strings.Join(left, ", "), f.Lead)
	}
	lead := drainLaneOptions(o, *st, leadFP, plan, rule)
	if f.Hold != nil {
		lead.held = heldOn(f.Lead, *f.Hold, lead.reserve)
	}
	members := make([]laneMember, 0, len(f.Members))
	for _, m := range f.Members {
		fp, _ := footprintOf(m.Issue, m.Hold)
		lm := laneMember{issue: m.Issue, footprint: fp}
		if m.Hold != nil {
			lm.held = heldOn(m.Issue, *m.Hold, lm.footprint)
		}
		members = append(members, lm)
	}
	for _, id := range f.ids() {
		holds.starting(id)
	}
	if drainBeforeStart != nil {
		drainBeforeStart(f.Lead)
	}
	started, err := startBundle(repoRoot, f.Lead, lead, members)
	if err != nil {
		r, ok := AsRefusal(err)
		if ok && r.capped != nil {
			return bundleOpened{capped: r.capped, left: leftNote}
		}
		if !ok || (r.Stage == StageReservation && !r.Contention) {
			return bundleOpened{err: err}
		}
		if err := unreadableRefusal(repoRoot, r.unjudged); err != nil {
			return bundleOpened{err: err}
		}
		return bundleOpened{left: leftNote, issues: all, contention: r.Contention, passed: &Excluded{ID: f.Lead, Check: r.Check, Reason: termsafe.Sanitize(r.Reason) +
			"; the bundle with " + strings.Join(f.ids()[1:], ", ") + " did not open, and its reservations are released"}}
	}
	ids := f.ids()
	for _, id := range ids {
		holds.keep(id)
	}
	lane := DrainLane{Issue: f.Lead, RunID: started.RunID, OpenedAt: now, Outcome: DrainLaneInProgress}
	if len(ids) > 1 {
		lane.Bundle = ids[1:]
	}
	st.Lanes = append(st.Lanes, lane)
	return bundleOpened{start: &started, left: leftNote}
}

// bundleNext is the move's next once a bundle's lane opened: a lane of one
// issue, every candidate having left, is said as one is.
func bundleNext(l DrainLane, started StartResult) string {
	if len(l.Bundle) == 0 {
		return fmt.Sprintf("drive %s's lane, run %s: %s; run `abcd drain` again once it is handed back or its pull request is open",
			l.Issue, started.RunID, started.Next)
	}
	return fmt.Sprintf("drive %s's lane, a bundle with %s, run %s: %s; run `abcd drain` again once it is handed back or its pull request is open",
		l.Issue, strings.Join(l.Bundle, ", "), started.RunID, started.Next)
}

// endForming is the clause for a forming bundle a drain that ends lets go:
// its holds are released plainly, with no tombstone, on the move's way out.
func endForming(f *DrainForming) string {
	return fmt.Sprintf("the bundle forming on %s ended with the drain: the reservations on %s are released with no tombstone",
		f.Lead, strings.Join(f.ids(), ", "))
}

// routedInLane reports whether st has routed a hand-back of id from the lane
// l since it opened.
func routedInLane(st DrainState, l DrainLane, id string) bool {
	return slices.ContainsFunc(st.HandBacks, func(r DrainRoute) bool {
		return r.From == "lane" && recordid.SameID(r.Issue, id) && r.At != nil && !r.At.Before(l.OpenedAt)
	})
}

// routeAlone routes each issue of l's run handed back alone (step 2's
// record: the lead's LeadHandBack, a member's HandBack) that the drain has not
// routed since the lane opened, as a lane's hand-back of its kind: the issue
// is a person's again while the lane lands the rest. The step that handed it
// back released its reservation with its tombstone. A run with no members
// routes nothing here.
func routeAlone(repoRoot string, st DrainState, l DrainLane, run State, origin *DrainOrigin, blobs *originBlobs, now time.Time) ([]DrainRoute, error) {
	var out []DrainRoute
	for _, a := range run.aloneHandBacks() {
		if routedInLane(st, l, a.id) {
			continue
		}
		r, err := routeHandBack(repoRoot, a.id, "lane", a.hb, origin, now)
		if err != nil {
			return out, err
		}
		r.Blob = blobs.of(a.id)
		out = append(out, r)
	}
	return out, nil
}

// routeAloneFinished is routeAlone over a lane of a bundle that merged: a
// member handed back alone that no move routed while the lane ran is routed
// as the lane is marked done, so it does not stay in the lane's hand. A lane
// of one issue reads nothing.
func routeAloneFinished(repoRoot string, st DrainState, l DrainLane, origin *DrainOrigin, blobs *originBlobs, now time.Time) ([]DrainRoute, error) {
	if len(l.Bundle) == 0 {
		return nil, nil
	}
	run, err := ReadState(repoRoot, l.RunID)
	if err != nil {
		return nil, err
	}
	return routeAlone(repoRoot, st, l, run, origin, blobs, now)
}

// aloneHandBacks is each issue of the run handed back alone, the lead first,
// with its hand-back, whether or not its reservation is recorded.
func (s State) aloneHandBacks() []aloneHandBack {
	var out []aloneHandBack
	if s.LeadHandBack != nil {
		out = append(out, aloneHandBack{id: s.Key, hb: *s.LeadHandBack})
	}
	for _, m := range s.Bundle {
		if m.HandBack != nil {
			out = append(out, aloneHandBack{id: m.Issue, hb: *m.HandBack})
		}
	}
	return out
}

// routeLaneHandBack routes the hand-back of l's whole lane: each issue handed
// back alone and not yet routed by its own (routeAlone), then every issue the
// run still carries by the lane's hand-back, the lead first; until the split
// (step 5), every member of a bundle is routed so. The reservation is released
// as a lane's is (releaseHandedBack), its clause on the first route the lane's
// hand-back makes. A run with no members routes its lane's issue alone, as
// before bundles; an issue the lane names as none is never routed.
func routeLaneHandBack(repoRoot string, st DrainState, l DrainLane, run State, hb HandBack, origin *DrainOrigin, blobs *originBlobs, now time.Time) ([]DrainRoute, error) {
	out, err := routeAlone(repoRoot, st, l, run, origin, blobs, now)
	if err != nil {
		return out, err
	}
	ids := []string{l.Issue}
	if run.hasMembers() {
		ids = run.Issues()
	}
	first := true
	for _, id := range ids {
		if id == "" {
			continue
		}
		r, err := routeHandBack(repoRoot, id, "lane", hb, origin, now)
		if err != nil {
			return out, err
		}
		if first {
			first = false
			if rel := releaseHandedBack(repoRoot, run, hb, now); rel != "" {
				r.Wrote += "; " + rel
			}
		}
		r.Blob = blobs.of(id)
		out = append(out, r)
	}
	return out, nil
}

// laneCarries reports whether l, a lane the drain opened, took id: as its
// issue, the lead, or as a member.
func laneCarries(l DrainLane, id string) bool {
	return l.Issue == id || slices.Contains(l.Bundle, id)
}

// handedBackAloneIn reports whether id, a member or the lead of l's bundle,
// was handed back alone and routed so since the lane opened: the lane no
// longer has it in hand, whatever it comes to. A lane of one issue has
// nothing handed back alone.
func (st DrainState) handedBackAloneIn(l DrainLane, id string) bool {
	return len(l.Bundle) > 0 && routedInLane(st, l, id)
}
