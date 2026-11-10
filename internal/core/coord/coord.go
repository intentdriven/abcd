// Package coord is the coordination pass of itd-2610170634394993: a pure
// function that reads a snapshot of the records on the shared remote and
// returns the decisions they call for (spc-2610230758581403, step 1).
//
// The pass holds no state and keeps no clock of its own. The time it judges
// by is part of the snapshot, and every deadline it names is derived from the
// records when they are read (dcn-2610180720366282), so any session that reads
// the same records reaches the same decisions, in whatever order it read them,
// and a pass stopped part-way through acting on them, re-run from the records
// its acts left, gives again every decision but those for the changes its acts
// recorded as queued.
//
// What it decides: an open change no dependency holds (every one it names has
// merged) is queued. One that depends directly on a change closed unmerged or
// on a number with no record, or is on or reaches a cycle of unmerged changes,
// itself included, is refused at once, naming the numbers and the cause (story
// 2), since no wait can release it. Any other open change waiting on another
// waits until its recorded start plus the stated wait limit; a wait at or past
// that deadline, one with no recorded start, or one whose recorded start is
// later than the records' time is refused with a reason naming what it waited
// on (adr-2610222026262980, rule 1). Claims and the record version are later
// steps of the spec.
//
// This is staged code: nothing outside the package calls the pass or its
// at-once refusals yet. Step 2a's landing wiring exports them and wires their
// first caller in one commit, and this package's step-2a code is never pushed
// or landed without that wiring (dcn-2611101309224327).
package coord

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// MaxWaitLimit is the longest wait limit a snapshot may state. A longer one is
// refused whole, as a limit that is not positive is, so no record can make a
// wait open-ended by stating a limit of centuries.
const MaxWaitLimit = 24 * time.Hour

const (
	// maxNamed is the most defects the one error names, and the most numbers
	// a reason lists; the rest are counted, never listed.
	maxNamed = 20
	// maxStateRunes is the most of an unknown state the error quotes.
	maxStateRunes = 32
)

// State is where a change stands in the records.
type State string

// The states a change's record can carry.
const (
	StateOpen   State = "open"
	StateQueued State = "queued"
	StateMerged State = "merged"
	StateClosed State = "closed"
)

func (s State) known() bool {
	switch s {
	case StateOpen, StateQueued, StateMerged, StateClosed:
		return true
	}
	return false
}

// Change is one change's record as the pass reads it.
type Change struct {
	// Number is the change's pull request number.
	Number int
	// State is where the change stands.
	State State
	// DependsOn names the changes this one waits on, already read from its
	// record. A number with no record of its own in the snapshot refuses the
	// change at once; why there is none is the caller's to say before the
	// pass.
	DependsOn []int
	// WaitingSince is the recorded time the change's wait began, the start
	// its deadline is derived from. The caller writes it once, before the
	// first pass that finds the change waiting, keeps that first start and
	// never moves it, and takes it from a source the waiting agent cannot
	// rewrite. A start re-stamped at each pass waits forever, and a pure pass
	// cannot see that it was. A waiting change with no recorded start is
	// refused, since no deadline can be derived for it. A start later than
	// the records' Now is refused too, with no allowance for clock skew; any
	// allowance is the caller's to make before the pass.
	WaitingSince time.Time
}

// Records is the snapshot one pass reads. The records are written by other
// agents, so the pass judges these fields before deciding anything: the time,
// the wait limit, each change's number, a number read twice, the state against
// the set of known states, each dependency's number, and each start against
// Now. A field it cannot judge refuses the change or the whole snapshot, never
// queues. That a change is merged or queued is taken as the caller's verified
// fact, not judged: land takes it from the forge's answer, never from a pull
// request body.
//
// The snapshot must hold only changes the caller is entitled to queue: a queue
// decision says only that no dependency holds the change, not whose change it
// is or that it is ready. Rebuilding a dependant on the default branch and the
// full test run before it is queued stay the caller's (the spec's step 2b).
type Records struct {
	// Now is the time the records were read at, which the pass judges by.
	Now time.Time
	// WaitLimit is how long a change may wait on another before its wait is
	// refused: more than zero and at most MaxWaitLimit.
	WaitLimit time.Duration
	// Changes is every change record read.
	Changes []Change
}

// Action is what a decision says to do with a change.
type Action string

// The actions a decision can carry.
const (
	// ActionQueue: no dependency holds the change. It says nothing of whether
	// the change is ready or the caller's to queue (Records).
	ActionQueue Action = "queue"
	// ActionWait: the change waits on WaitsOn until Until.
	ActionWait Action = "wait"
	// ActionRefuse: the change is refused, for Reason.
	ActionRefuse Action = "refuse"
)

// Decision is the pass's ruling on one change.
type Decision struct {
	// Change is the change's number.
	Change int
	// Action is what to do with it.
	Action Action
	// WaitsOn lists the dependencies not yet merged, ascending and once each,
	// for a wait or a refusal.
	WaitsOn []int
	// Until is the wait's deadline in UTC, for a wait, and for a refusal at
	// or past its deadline. It is zero for a refusal at once and for a
	// refusal of a change with no recorded start or with one later than the
	// records' time.
	Until time.Time
	// Reason says why a refusal was made.
	Reason string
}

// pass reads the records and returns one decision for each open change,
// ordered by change number; queued, merged and closed changes need none. A
// snapshot it cannot judge is refused whole: no time, a wait limit that is not
// positive or is above MaxWaitLimit, a change with a number of zero or less,
// one read twice, an unknown state, or a dependency numbered zero or less. The
// one error names every defect found, sorted, so it is the same whatever
// order the records were read in. It names at most maxNamed defects, then how
// many more there are, and quotes an unknown state cut to maxStateRunes. pass
// does not change the records it reads.
//
// A caller whose whole pass is refused refuses its own act, arms nothing, and
// names the defect. It never sets the defective record aside and goes on,
// since a dependency set aside would read as satisfied.
//
// It is unexported because nothing in production calls it yet, only its
// tests. That takes it out of the exported-reach audit's view and does not
// satisfy the audit: step 2a of spc-2610230758581403 exports it and wires its
// first caller in one commit (dcn-2611101309224327).
func pass(r Records) ([]Decision, error) {
	var defects []string
	if r.Now.IsZero() {
		defects = append(defects, "the records carry no time to judge them by")
	}
	switch {
	case r.WaitLimit <= 0:
		defects = append(defects, fmt.Sprintf("the records state no positive wait limit (%s)", r.WaitLimit))
	case r.WaitLimit > MaxWaitLimit:
		defects = append(defects, fmt.Sprintf("the records state a wait limit of %s, above the most of %s", r.WaitLimit, MaxWaitLimit))
	}
	states := make(map[int]State, len(r.Changes))
	seen := make(map[int]int, len(r.Changes))
	for _, c := range r.Changes {
		if c.Number <= 0 {
			defects = append(defects, fmt.Sprintf("a change carries number %d", c.Number))
		} else if seen[c.Number]++; seen[c.Number] == 2 {
			defects = append(defects, fmt.Sprintf("change #%d is in the records more than once", c.Number))
		}
		if !c.State.known() {
			defects = append(defects, unknownState(c))
		}
		for _, n := range c.DependsOn {
			if n <= 0 {
				defects = append(defects, fmt.Sprintf("change #%d depends on number %d", c.Number, n))
			}
		}
		states[c.Number] = c.State
	}
	if len(defects) > 0 {
		slices.Sort(defects)
		defects = slices.Compact(defects)
		if more := len(defects) - maxNamed; more > 0 {
			defects = append(defects[:maxNamed], fmt.Sprintf("and %d more", more))
		}
		return nil, fmt.Errorf("coord: the records cannot be judged: %s", strings.Join(defects, "; "))
	}

	open := make([]Change, 0, len(r.Changes))
	for _, c := range r.Changes {
		if c.State == StateOpen {
			open = append(open, c)
		}
	}
	slices.SortFunc(open, func(a, b Change) int { return a.Number - b.Number })

	cycles := cyclesReached(r.Changes, states)
	decisions := make([]Decision, 0, len(open))
	for _, c := range open {
		decisions = append(decisions, decide(r, states, cycles[c.Number], c))
	}
	return decisions, nil
}

// decide rules on one open change. A direct dependency closed unmerged or
// with no record, or a cycle the change is on or reaches, refuses it at once,
// before its start or deadline is judged: no wait can release it. A change
// depending on an open change that is itself refused is not refused at once;
// it waits to its own deadline.
func decide(r Records, states map[int]State, cycle []int, c Change) Decision {
	var waits, closed, missing []int
	for _, n := range c.DependsOn {
		switch st, ok := states[n]; {
		case !ok:
			missing = append(missing, n)
		case st == StateClosed:
			closed = append(closed, n)
		case st == StateMerged:
			continue
		}
		waits = append(waits, n)
	}
	if len(waits) == 0 {
		return Decision{Change: c.Number, Action: ActionQueue}
	}
	slices.Sort(waits)
	waits = slices.Compact(waits)
	var causes []string
	if len(closed) > 0 {
		slices.Sort(closed)
		causes = append(causes, fmt.Sprintf("depends on %s, closed unmerged", names(slices.Compact(closed))))
	}
	if len(missing) > 0 {
		slices.Sort(missing)
		causes = append(causes, fmt.Sprintf("depends on %s, with no record", names(slices.Compact(missing))))
	}
	if len(cycle) > 0 {
		causes = append(causes, "is on or reaches a cycle of unmerged changes among "+names(cycle))
	}
	if len(causes) > 0 {
		return Decision{Change: c.Number, Action: ActionRefuse, WaitsOn: waits,
			Reason: "refused at once, since no wait can release it: " + strings.Join(causes, "; ")}
	}
	on := names(waits)
	if c.WaitingSince.IsZero() {
		return Decision{Change: c.Number, Action: ActionRefuse, WaitsOn: waits,
			Reason: fmt.Sprintf("waits on %s with no recorded start, so no deadline can be derived from the %s limit", on, r.WaitLimit)}
	}
	if c.WaitingSince.After(r.Now) {
		return Decision{Change: c.Number, Action: ActionRefuse, WaitsOn: waits,
			Reason: fmt.Sprintf("waits on %s from a recorded start of %s, later than the records' time of %s, so no deadline is derived from it",
				on, stamp(c.WaitingSince), stamp(r.Now))}
	}
	until := c.WaitingSince.Add(r.WaitLimit).UTC()
	if !r.Now.Before(until) {
		return Decision{Change: c.Number, Action: ActionRefuse, WaitsOn: waits, Until: until,
			Reason: fmt.Sprintf("waited on %s for the %s limit since %s; deadline %s reached",
				on, r.WaitLimit, stamp(c.WaitingSince), stamp(until))}
	}
	return Decision{Change: c.Number, Action: ActionWait, WaitsOn: waits, Until: until}
}

// cyclesReached maps each open or queued change that is on or reaches a cycle
// to the cycle members it reaches, ascending and once each. Edges run from an
// open or queued change to each dependency that is open or queued; a merged or
// closed change, or a number with no record, ends the walk. A cycle is a
// self-edge or a strongly connected group of two or more, so a diamond is
// none. The groups are found by one iterative walk (Tarjan's), linear in
// changes plus edges; a group's reached members are then joined from the
// groups it depends on, each finished before it.
func cyclesReached(changes []Change, states map[int]State) map[int][]int {
	unmerged := func(st State) bool { return st == StateOpen || st == StateQueued }
	var nodes []Change
	for _, c := range changes {
		if unmerged(c.State) {
			nodes = append(nodes, c)
		}
	}
	slices.SortFunc(nodes, func(a, b Change) int { return a.Number - b.Number })
	index := make(map[int]int, len(nodes))
	for i, c := range nodes {
		index[c.Number] = i
	}
	adj := make([][]int, len(nodes))
	self := make([]bool, len(nodes))
	for i, c := range nodes {
		for _, n := range c.DependsOn {
			if st, ok := states[n]; ok && unmerged(st) {
				adj[i] = append(adj[i], index[n])
				self[i] = self[i] || n == c.Number
			}
		}
	}

	const unseen = -1
	order, low := make([]int, len(nodes)), make([]int, len(nodes))
	for i := range order {
		order[i] = unseen
	}
	onStack := make([]bool, len(nodes))
	group := make([]int, len(nodes))
	var stack []int
	var groups [][]int // each group finished after every group it reaches
	next := 0
	type frame struct{ v, edge int }
	for root := range nodes {
		if order[root] != unseen {
			continue
		}
		order[root], low[root] = next, next
		next++
		stack, onStack[root] = append(stack, root), true
		calls := []frame{{root, 0}}
		for len(calls) > 0 {
			top := len(calls) - 1
			v := calls[top].v
			if e := calls[top].edge; e < len(adj[v]) {
				calls[top].edge++
				w := adj[v][e]
				switch {
				case order[w] == unseen:
					order[w], low[w] = next, next
					next++
					stack, onStack[w] = append(stack, w), true
					calls = append(calls, frame{w, 0})
				case onStack[w]:
					low[v] = min(low[v], order[w])
				}
				continue
			}
			calls = calls[:top]
			if top > 0 {
				p := calls[top-1].v
				low[p] = min(low[p], low[v])
			}
			if low[v] != order[v] {
				continue
			}
			var members []int
			for {
				w := stack[len(stack)-1]
				stack, onStack[w] = stack[:len(stack)-1], false
				group[w] = len(groups)
				members = append(members, w)
				if w == v {
					break
				}
			}
			groups = append(groups, members)
		}
	}

	reached := make([][]int, len(groups))
	out := make(map[int][]int)
	for g, members := range groups {
		var nums []int
		if len(members) > 1 || self[members[0]] {
			for _, m := range members {
				nums = append(nums, nodes[m].Number)
			}
		}
		for _, m := range members {
			for _, w := range adj[m] {
				if group[w] != g {
					nums = append(nums, reached[group[w]]...)
				}
			}
		}
		slices.Sort(nums)
		reached[g] = slices.Compact(nums)
		if len(reached[g]) > 0 {
			for _, m := range members {
				out[nodes[m].Number] = reached[g]
			}
		}
	}
	return out
}

func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

// unknownState names a change's unknown state, quoting at most its first
// maxStateRunes runes, so a state of any length gives a bounded defect.
func unknownState(c Change) string {
	s, runes := string(c.State), 0
	for i := range s {
		if runes == maxStateRunes {
			return fmt.Sprintf("change #%d has unknown state %q, cut to its first %d runes", c.Number, s[:i], maxStateRunes)
		}
		runes++
	}
	return fmt.Sprintf("change #%d has unknown state %q", c.Number, s)
}

// names lists the numbers, at most maxNamed of them, then how many more.
func names(nums []int) string {
	shown := nums[:min(len(nums), maxNamed)]
	parts := make([]string, len(shown))
	for i, n := range shown {
		parts[i] = fmt.Sprintf("#%d", n)
	}
	out := strings.Join(parts, ", ")
	if more := len(nums) - len(shown); more > 0 {
		out += fmt.Sprintf(" and %d more", more)
	}
	return out
}
