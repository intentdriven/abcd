package loop

// schedule.go is how a run works in parallel up to its ceiling (ruling DR6,
// 2026-09-29; spc-2609202134341288, "Concurrent lanes and validators"). A slot
// is one agent the run has handed work to and not yet taken a verified receipt
// from: an outstanding await on any lane. The count is the number of awaits in
// the state file, and nothing else is counted. The ceiling is the run's
// pace.sub_agents; implementers and validators take the same slots.
//
// Each `implement step` performs one move. It first performs any stage of any
// lane the binary owns (the worktree, the brief, the landing's steps, a round's
// close, a sync, a hold), which takes no slot and is never held by the ceiling.
// When the move needs an agent, it takes the first waiting item in this order:
//
//  1. work on a lane already open, before any new lane: a round's validators
//     and the fix or sync implementers;
//  2. among open lanes, the lane of the lower-numbered spec step first;
//  3. within one lane's round, the validators in the order the round lists
//     them;
//  4. then the first implementer of a new lane, lowest spec step first.
//
// The order is a function of the state alone. A lane opens for a spec step once
// every step it needs has landed (its `- needs:` line, or by default every
// earlier step, ruling DR6b), and only when a helper is free to take it (ruling
// DR6d-1, the product thinker, 2026-10-02: "a step's worktree is made just
// before an agent takes it; at most the agent ceiling's worth on disk"): a slot
// is left for its implementer beside every lane opened whose implementer is not
// out yet, and fewer step worktrees than the ceiling are on disk. A step that
// waits for a helper has no worktree. No lane opens after a hand-back (ruling
// DR6c). Landing is one lane at a time: a lane waits at its landing while a
// sibling's landing is under way, and holds no slot while it waits.

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// AliveLane is one lane of a run with anything left, as a result reports it.
type AliveLane struct {
	Lane     string  `json:"lane"`
	SpecStep int     `json:"spec_step"`
	Stage    Stage   `json:"stage"`
	Awaits   []Await `json:"awaits"`
	Hold     *Hold   `json:"hold,omitempty"`
	// Waiting is the lane's wait for its full check, with the time it began
	// (ruling DR6d-2); empty when it waits for none.
	Waiting string `json:"waiting,omitempty"`
}

// LaneAwait is one outstanding await with the lane it belongs to.
type LaneAwait struct {
	Lane  string
	Await Await
}

// ceiling is the most agents the run may have out at once: its pace's
// sub-agents, or one for a run started before the loop paced a run.
func (s State) ceiling() int {
	if s.Pace == nil || s.Pace.SubAgents.Value < 1 {
		return 1
	}
	return s.Pace.SubAgents.Value
}

// slotsInUse is the number of outstanding awaits on every lane of the run.
func (s State) slotsInUse() int {
	n := 0
	for _, l := range s.Lanes {
		n += len(l.Awaits)
	}
	return n
}

// allAwaits lists every outstanding await, lane by lane.
func (s State) allAwaits() []LaneAwait {
	var out []LaneAwait
	for _, l := range s.Lanes {
		for _, a := range l.Awaits {
			out = append(out, LaneAwait{Lane: l.ID, Await: a})
		}
	}
	return out
}

// alive lists every lane with anything left, the held lanes included.
func (s State) alive() []AliveLane {
	var out []AliveLane
	for _, i := range s.laneOrder() {
		l := s.Lanes[i]
		if l.Stage == StageDone || l.Stage == StageDiscarded {
			continue
		}
		aw := append([]Await{}, l.Awaits...)
		out = append(out, AliveLane{Lane: l.ID, SpecStep: l.SpecStep, Stage: l.Stage, Awaits: aw, Hold: l.Hold, Waiting: l.CheckWait()})
	}
	return out
}

// laneOrder is the lanes' indices by spec step, then by the order they opened.
func (s State) laneOrder() []int {
	idx := make([]int, len(s.Lanes))
	for i := range idx {
		idx[i] = i
	}
	slices.SortStableFunc(idx, func(a, b int) int { return s.Lanes[a].SpecStep - s.Lanes[b].SpecStep })
	return idx
}

// findAwait is the lane and the await a receipt path names, or -1.
func (s State) findAwait(repoRoot, receipt string) (int, int) {
	for i, l := range s.Lanes {
		for k, a := range l.Awaits {
			if samePath(repoRoot, receipt, a.Receipt) {
				return i, k
			}
		}
	}
	return -1, -1
}

// unknownReceipt refuses a receipt path no outstanding await names, naming the
// awaits there are; it frees nothing.
func unknownReceipt(st State) error {
	out := st.allAwaits()
	switch len(out) {
	case 0:
		return refuse("receipt", "", "", "no lane of "+st.RunID+" awaits a receipt",
			"run `abcd implement step`; it names the receipt when a stage hands work to an agent")
	case 1:
		stage := ""
		for _, l := range st.Lanes {
			if l.ID == out[0].Lane {
				stage = string(l.Stage)
			}
		}
		return refuse("receipt", "", out[0].Lane, "the "+stage+" stage awaits its receipt at "+out[0].Await.Receipt+", not at the path given",
			"hand back `abcd implement receipt "+out[0].Await.Receipt+"`")
	}
	parts := make([]string, 0, len(out))
	for _, a := range out {
		parts = append(parts, fmt.Sprintf("%s's %s at %s", a.Lane, a.Await.Role, a.Await.Receipt))
	}
	return refuse("receipt", "", "", "no outstanding await of "+st.RunID+" names the path given; it awaits "+strings.Join(parts, "; "),
		"hand back one of those paths with `abcd implement receipt <path>`")
}

// requires is the spec steps a pending step waits for: its resolved needs, or,
// in a state file before version 8, every earlier step (ruling DR6b).
func (p PendingStep) requires() []int {
	if p.Needs != nil {
		return p.Needs
	}
	out := []int{}
	for n := 1; n < p.Number; n++ {
		out = append(out, n)
	}
	return out
}

// landedInRun reports whether spec step n has nothing left in the run: no
// pending entry for it, and every lane that builds it done (its pull request
// an ancestor of the default branch). A step landed before the run is neither.
func (s State) landedInRun(n int) bool {
	for _, p := range s.Pending {
		if p.Number == n {
			return false
		}
	}
	for _, l := range s.Lanes {
		if l.SpecStep == n && l.Stage != StageDone {
			return false
		}
	}
	return true
}

// readyPending lists the indices of the pending steps every need of which has
// landed, in order; after a hand-back there are none (ruling DR6c: no new lane
// opens, and pending steps stay pending).
func (s State) readyPending() []int {
	if s.handedBack() {
		return nil
	}
	var out []int
	for k, p := range s.Pending {
		ready := true
		for _, n := range p.requires() {
			if !s.landedInRun(n) {
				ready = false
				break
			}
		}
		if ready {
			out = append(out, k)
		}
	}
	return out
}

// reserved counts the lanes opened whose first implementer has not been
// handed out yet: each will take a slot, so a new lane opens only while the
// slots in use and these leave one free.
func (s State) reserved() int {
	n := 0
	for _, l := range s.Lanes {
		switch {
		case l.Stage == StageWorktree, l.Stage == StageBrief:
			n++
		case l.Stage == StageImplement && len(l.Awaits) == 0 && l.Receipt == "":
			n++
		}
	}
	return n
}

// stepWorktrees counts the lanes that hold a step worktree, or are opened to
// make one: a lane holds it from its worktree stage until it lands or is
// discarded. A lane handed back with its worktree discarded holds none.
func (s State) stepWorktrees() int {
	n := 0
	for _, l := range s.Lanes {
		switch {
		case l.Stage == StageDone, l.Stage == StageDiscarded:
		case l.HandBack != nil && l.HandBack.Discarded != "":
		default:
			n++
		}
	}
	return n
}

// helperFree reports whether a new lane may open (ruling DR6d-1): a slot is
// left for its implementer beside the slots in use and the lanes opened whose
// implementer is not out yet, and fewer step worktrees than the ceiling are on
// disk.
func (s State) helperFree() bool {
	return s.slotsInUse()+s.reserved() < s.ceiling() && s.stepWorktrees() < s.ceiling()
}

// openLaneRecorded opens the lane for the pending step at index k and records
// it: the run record lists the spec's steps as it lists the lanes
// (itd-2609212103565953, criterion 4). The new lane's waiting item, keyed on
// its spec step, is served by the open.
func openLaneRecorded(st *State, k int, now time.Time) int {
	key := stepKey(st.Pending[k].Number)
	openLane(st, k)
	tookSlot(st, key, RoleImplementer, now)
	i := len(st.Lanes) - 1
	l := st.Lanes[i]
	st.Record = append(st.Record, Entry{At: now, Lane: l.ID, Stage: "open",
		Note: fmt.Sprintf("%s opened for step %d of %s (%s)", l.ID, l.SpecStep, st.Spec, l.StepTitle)})
	return i
}

// openNext opens, state-only, the lane a freed helper would take next when that
// is a new lane: a lane that lands makes its dependants ready and frees its
// worktree, and the run record lists the next lane beside the landing that let
// it open. The next call makes its worktree.
func openNext(st *State, now time.Time) {
	if wants := agentWants(*st); len(wants) > 0 && wants[0].open >= 0 && st.helperFree() {
		openLaneRecorded(st, wants[0].open, now)
	}
}

// stepKey names the waiting work of a spec step no lane is open for yet.
func stepKey(n int) string { return fmt.Sprintf("step %d", n) }

// waitKey names a piece of waiting work: its lane, or the spec step a new lane
// will open for.
func waitKey(st State, w want) string {
	if w.open >= 0 {
		return stepKey(st.Pending[w.open].Number)
	}
	return st.Lanes[w.lane].ID
}

// The kinds of move a lane wants next.
type wantKind int

const (
	wantNone wantKind = iota
	wantBinary
	wantAgent
)

// want is one lane's next move: nothing, a stage the binary performs, or an
// agent in a role. hold marks the binary move that holds the lane (DR6c); new
// marks the first implementer of a lane not yet handed to one, or of a lane to
// open, ordered by its spec step.
type want struct {
	lane int
	kind wantKind
	role string
	hold bool
	new  bool
	// open is the index of the pending step a new lane opens for, or -1;
	// step is the spec step a new lane's work is ordered by.
	open int
	step int
}

// awaited reports whether the lane has an agent out writing path.
func (l Lane) awaited(path string) bool {
	for _, a := range l.Awaits {
		if a.Receipt == path {
			return true
		}
	}
	return false
}

// awaitsRole reports whether the lane has an agent out in role.
func (l Lane) awaitsRole(role string) bool {
	for _, a := range l.Awaits {
		if a.Role == role {
			return true
		}
	}
	return false
}

// pendingSync is the lane's conflicting sync whose resolution has not been
// verified, or nil.
func (l Lane) pendingSync() *Sync {
	if n := len(l.Syncs); n > 0 && l.Syncs[n-1].Conflicted && l.Syncs[n-1].Head == "" {
		return &l.Syncs[n-1]
	}
	return nil
}

// syncRoundDue reports whether the lane's last sync produced a head no round
// has opened over yet: a fresh round judges it.
func (l Lane) syncRoundDue() bool {
	n := len(l.Syncs)
	return n > 0 && l.Syncs[n-1].Head != "" && l.Syncs[n-1].Round == 0
}

// fixRoundsTaken is the fix rounds the lane has taken before its current
// round: the rounds whose findings went to a fresh implementer. A round opened
// after a sync follows a passing round and counts none.
func (l Lane) fixRoundsTaken() int {
	n := 0
	for i := 0; i+1 < len(l.Validation); i++ {
		if l.Validation[i].Fix != "" {
			n++
		}
	}
	return n
}

// laneWant is what lane i wants next, a function of the state alone.
func laneWant(st State, i int) want {
	l := st.Lanes[i]
	w := want{lane: i, open: -1, step: l.SpecStep}
	switch l.Stage {
	case StageWorktree, StageBrief:
		w.kind = wantBinary
	case StageImplement:
		if len(l.Awaits) == 0 {
			w.kind, w.role, w.new = wantAgent, RoleImplementer, l.Receipt == ""
		}
	case StageValidate:
		if l.awaitsRole(RoleImplementer) {
			return w
		}
		if l.pendingSync() != nil {
			w.kind, w.role = wantAgent, RoleImplementer
			return w
		}
		n := len(l.Validation)
		if n == 0 || l.Validation[n-1].Fix != "" || l.syncRoundDue() {
			w.kind, w.role = wantAgent, RoleRuthless
			return w
		}
		cur := l.Validation[n-1]
		out := false
		for _, v := range cur.Validators {
			if v.Verdict != "" {
				continue
			}
			if !l.awaited(v.Return) {
				w.kind, w.role = wantAgent, v.Role
				return w
			}
			out = true
		}
		if out {
			return w
		}
		for _, v := range cur.Validators {
			if !v.Pass {
				if l.fixRoundsTaken() >= st.FixRoundCap() {
					w.kind = wantBinary
				} else {
					w.kind, w.role = wantAgent, RoleImplementer
				}
				return w
			}
		}
		w.kind = wantBinary
	case StageLand:
		if st.handedBack() && (l.Hold == nil || !l.Hold.Released) {
			w.kind, w.hold = wantBinary, true
			return w
		}
		// One landing at a time: a sibling's landing under way holds this one.
		for j, o := range st.Lanes {
			if j != i && o.Stage == StageLand && o.Landing != nil && o.Landing.Merged == "" {
				return w
			}
		}
		w.kind = wantBinary
	}
	return w
}

// agentWants lists the work that needs an agent, in the order a freed slot
// takes it: open lanes before new ones, each by spec step; a new lane is the
// first implementer of a lane not yet handed to one, or a lane to open for a
// ready pending step, which opens only while a helper is free for it.
func agentWants(st State) []want {
	var open, fresh []want
	for _, i := range st.laneOrder() {
		w := laneWant(st, i)
		if w.kind != wantAgent {
			continue
		}
		if w.new {
			fresh = append(fresh, w)
		} else {
			open = append(open, w)
		}
	}
	for _, k := range st.readyPending() {
		fresh = append(fresh, want{lane: -1, kind: wantAgent, role: RoleImplementer, new: true, open: k, step: st.Pending[k].Number})
	}
	slices.SortStableFunc(fresh, func(a, b want) int { return a.step - b.step })
	return append(open, fresh...)
}

// move performs the run's next move and reports what it did. An error that is
// the call's answer, a wait or a refusal, reports true beside it when the call
// wrote the time a lane began waiting for its full check, which the caller
// writes: the time is when the wait began, so a later refusal in the same call
// does not let it drift to the next call's.
func move(repoRoot string, st *State, steps Stages, now time.Time) (StepResult, bool, error) {
	// A landing waiting on the forge's merge or on its full check (a
	// contention refusal, ruling DR6d-2) holds only its own lane: the call
	// moves the next lane and names the wait beside what it did; when nothing
	// else moves, the first wait is the call's answer. The time a lane began
	// waiting for its full check is written once, so the call that finds it
	// writes the state even when the wait is its answer. Any other refusal of a
	// stage the binary performs is the call's answer, and no other lane moves:
	// a disarm the forge refuses stops the step.
	var blocked []Refusal
	var waitErr error
	began := false
	moved := func(res StepResult) StepResult {
		res.Blocked = blocked
		res.Next = withBlocked(res.Next, blocked)
		return res
	}
	for _, i := range st.laneOrder() {
		w := laneWant(*st, i)
		if w.kind != wantBinary {
			continue
		}
		res, err := performMove(repoRoot, st, steps, w, now)
		if err != nil {
			r, ok := AsRefusal(err)
			if !ok || !r.Contention {
				return StepResult{}, began, err
			}
			if beganCheckWait(st, i, r, now) {
				began = true
			}
			if waitErr == nil {
				waitErr = err
			}
			blocked = append(blocked, *r)
			continue
		}
		return moved(res), true, nil
	}
	// A refused agent move writes nothing of its own: a lane it opened for a
	// pending step stays unopened. Every change moveAgent makes before its
	// stage body succeeds replaces a slice or appends past its length, so the
	// shallow copy is the state as it was.
	before := *st
	res, did, err := moveAgent(repoRoot, st, steps, now)
	if err != nil {
		*st = before
		return StepResult{}, began, err
	}
	if did {
		return moved(res), true, nil
	}
	if waitErr != nil {
		return StepResult{}, began, waitErr
	}
	if st.slotsInUse() == 0 && st.handedBack() {
		return StepResult{}, false, handedBackRefusal(*st)
	}
	return res, false, nil
}

// withBlocked names, after a call's next move, each lane whose landing waits
// on the forge while the call moved another.
func withBlocked(next string, blocked []Refusal) string {
	for _, r := range blocked {
		next += fmt.Sprintf("; meanwhile %s waits: %s (%s)", r.Lane, r.Reason, r.Remedy)
	}
	return next
}

// moveAgent hands the first waiting work an agent when a slot is free, or
// records the work the ceiling, or the want of a free helper, holds back; it
// reports whether it moved or wrote.
func moveAgent(repoRoot string, st *State, steps Stages, now time.Time) (StepResult, bool, error) {
	wants := agentWants(*st)
	if len(wants) == 0 {
		return idleResult(*st), false, nil
	}
	if st.slotsInUse() >= st.ceiling() {
		changed := holdWaiting(st, wants, now)
		res := idleResult(*st)
		res.CeilingReached = true
		res.Next = ceilingMove(*st)
		return res, changed, nil
	}
	// A new lane opens only while a helper is free for it (ruling DR6d-1). When
	// every piece of waiting work is a step no helper is free for (the slots
	// left are promised to lanes opened, or the step worktrees on disk are at
	// the ceiling), each waits under `waiting` as `step <n>`, as work the
	// ceiling holds back does.
	k := slices.IndexFunc(wants, func(w want) bool { return w.open < 0 || st.helperFree() })
	if k < 0 {
		changed := holdWaiting(st, wants, now)
		return idleResult(*st), changed, nil
	}
	w := wants[k]
	opened := w.open >= 0
	if opened {
		// A new lane opens, state-only, and its first stage (the worktree,
		// which the binary performs) is this call's move, just before its
		// implementer takes the slot left for it.
		w = want{lane: openLaneRecorded(st, w.open, now), kind: wantBinary, open: -1}
	}
	res, err := performMove(repoRoot, st, steps, w, now)
	if err != nil {
		return StepResult{}, false, err
	}
	if !opened {
		tookSlot(st, st.Lanes[w.lane].ID, w.role, now)
	}
	return res, true, nil
}

// performMove runs one lane's next stage body and applies its outcome.
func performMove(repoRoot string, st *State, steps Stages, w want, now time.Time) (StepResult, error) {
	i := w.lane
	lane := st.Lanes[i]
	c := Context{RepoRoot: repoRoot, RunDir: runRel(st.RunID), State: *st, Now: now}
	var out Outcome
	var err error
	if w.hold {
		out, err = holdLane(c, &lane)
	} else {
		def, ok := steps.lookup(lane.Stage)
		if !ok || def.Run == nil {
			piece := ""
			if ok {
				piece = fmt.Sprintf(" (piece %d of %s delivers it)", def.Piece, specOf(*st))
			}
			return StepResult{}, refusef(string(lane.Stage), lane.ID,
				"use an abcd that carries the stage; the run is unchanged and resumes here",
				"the %s stage is not built in this abcd%s", lane.Stage, piece)
		}
		out, err = def.Run(c, &lane)
	}
	if err != nil {
		return StepResult{}, err
	}
	performed := Stage("")
	var handed *Await
	stage := string(lane.Stage)
	switch {
	case out.HandBack != nil:
		handBackLane(st, &lane, *out.HandBack, out.Note, now)
		st.Lanes[i] = lane
		st.UpdatedAt = now
		res := laneResult(*st, lane, "", nil)
		res.HandBack = lane.HandBack
		return res, nil
	case out.Stay:
		st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Stage: stage, Note: out.Note})
	case out.Goto != "":
		st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Stage: stage, Note: out.Note})
		lane.Stage = out.Goto
	case out.Await != nil:
		if out.Await.Since.IsZero() {
			out.Await.Since = now
		}
		lane.Awaits = append(slices.Clone(lane.Awaits), *out.Await)
		handed = out.Await
		note := out.Note
		if note == "" {
			note = "awaiting the " + out.Await.Role + "'s receipt at " + out.Await.Receipt
		}
		st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Stage: stage, Note: note})
	default:
		performed = lane.Stage
		st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Stage: stage, Note: out.Note})
		lane.Stage = after(lane.Stage)
	}
	st.Lanes[i] = lane
	if lane.Stage == StageDone {
		openNext(st, now)
	}
	st.UpdatedAt = now
	return laneResult(*st, lane, performed, handed), nil
}

// holdWaiting writes the work the ceiling holds back into the run's waiting
// list, keeping the time each item was first held. It reports whether the list
// changed, so a second call before any receipt writes nothing.
func holdWaiting(st *State, wants []want, now time.Time) bool {
	next := make([]Waiting, 0, len(wants))
	for _, w := range wants {
		id := waitKey(*st, w)
		since := now
		for _, old := range st.Waiting {
			if old.Lane == id && old.Role == w.role {
				since = old.Since
			}
		}
		next = append(next, Waiting{Lane: id, Role: w.role, Since: since})
	}
	if slices.Equal(next, st.Waiting) {
		return false
	}
	st.Waiting = next
	st.UpdatedAt = now
	return true
}

// tookSlot removes the waiting item a move just served and records how long
// it waited, in whole minutes. A new lane's item is keyed on its spec step and
// served when the lane opens.
func tookSlot(st *State, key, role string, now time.Time) {
	for k, w := range st.Waiting {
		if w.Lane != key || w.Role != role {
			continue
		}
		laneID := key
		if !ValidLaneID(key) {
			laneID = ""
		}
		st.Waiting = slices.Delete(slices.Clone(st.Waiting), k, k+1)
		if len(st.Waiting) == 0 {
			st.Waiting = nil
		}
		mins := int(now.Sub(w.Since) / time.Minute)
		st.Record = append(st.Record, Entry{At: now, Lane: laneID, Stage: "slot",
			Note: fmt.Sprintf("the %s of %s took a freed slot after waiting %d minute(s) at the run's ceiling of %d", role, key, mins, st.ceiling())})
		return
	}
}

// idleResult reports a run a call moved no lane of: the lanes alive and the
// slots in use.
func idleResult(st State) StepResult {
	res := StepResult{RunID: st.RunID, Slots: st.slotsInUse(), Ceiling: st.ceiling(), Alive: st.alive()}
	if i := st.current(); i >= 0 {
		l := st.Lanes[i]
		res.Lane, res.Stage, res.Awaiting = l.ID, l.Stage, l.awaiting()
		res.Next = nextMove(st, l)
	}
	if out := st.allAwaits(); len(out) > 1 {
		res.Next = awaitsMove(st)
	}
	return res
}

// awaitsMove names every outstanding await.
func awaitsMove(st State) string {
	parts := []string{}
	for _, a := range st.allAwaits() {
		parts = append(parts, fmt.Sprintf("%s's %s (brief %s) hands back `abcd implement receipt %s`", a.Lane, a.Await.Role, a.Await.Brief, a.Await.Receipt))
	}
	return "wait for the agents out: " + strings.Join(parts, "; ")
}

// ceilingMove is the next move of a call that found the ceiling reached.
func ceilingMove(st State) string {
	return fmt.Sprintf("nothing new: the run's ceiling of %d agent(s) is reached (%d in use). %s; each verified receipt frees a slot the next `abcd implement step` fills",
		st.ceiling(), st.slotsInUse(), awaitsMove(st))
}

// SlotsInUse is the agents the run has out: its outstanding awaits.
func (s State) SlotsInUse() int { return s.slotsInUse() }

// Ceiling is the most agents the run may have out at once.
func (s State) Ceiling() int { return s.ceiling() }

// checkWaitText is a lane's wait for its full check as the status shows it
// (ruling DR6d-2), the time the wait began in UTC.
func checkWaitText(since time.Time) string {
	return "waiting for its full check (since " + since.UTC().Format("15:04") + ")"
}

// CheckWait is the lane's wait for its full check, as the status shows it, or
// "" when it waits for none: a landing whose push found no preflight receipt
// naming its head.
func (l Lane) CheckWait() string {
	if l.Stage != StageLand || l.Landing == nil || l.Landing.Pushed != "" || l.Landing.CheckWaitSince == nil {
		return ""
	}
	return checkWaitText(*l.Landing.CheckWaitSince)
}

// beganCheckWait writes, once, the time lane i began waiting for its full
// check, with a record entry naming it, and reports whether it wrote.
func beganCheckWait(st *State, i int, r *Refusal, now time.Time) bool {
	l := st.Lanes[i]
	if r.checkWait.IsZero() || l.Landing == nil || l.Landing.CheckWaitSince != nil {
		return false
	}
	ld := *l.Landing
	since := r.checkWait
	ld.CheckWaitSince = &since
	l.Landing = &ld
	st.Lanes[i] = l
	st.Record = append(st.Record, Entry{At: now, Lane: l.ID, Stage: string(StageLand),
		Note: "waiting for its full check: no preflight receipt names the lane's head " + shortSHA(l.HeadSHA)})
	st.UpdatedAt = now
	return true
}
