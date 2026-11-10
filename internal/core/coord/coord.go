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
// What this step decides: an open change no dependency holds (every one it
// names has merged) is queued; an open change waiting on any other change
// waits until its recorded start plus the stated wait limit; a wait at or past
// that deadline, one with no recorded start, or one whose recorded start is
// later than the records' time is refused with a reason naming what it waited
// on (adr-2610222026262980, rule 1). A change depending on itself or on a
// cycle is never queued: it waits, then is refused at its deadline. Reading
// the dependency line, refusing a closed, unresolvable or cyclic dependency at
// once, claims and the record version are later steps of the spec, not this
// package yet. Nothing outside the package calls the pass yet (see pass).
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
	// record. A number with no record of its own counts as not merged.
	DependsOn []int
	// WaitingSince is the recorded time the change's wait began, the start
	// its deadline is derived from. The caller records it before the first
	// pass that finds the change waiting: a waiting change with no recorded
	// start is refused, since no deadline can be derived for it. A start later
	// than the records' Now is refused too, with no allowance for clock skew;
	// any allowance is the caller's to make before the pass.
	WaitingSince time.Time
}

// Records is the snapshot one pass reads. The records are written by other
// agents, so the pass trusts none of them: a field it cannot judge refuses the
// change or the whole snapshot, never queues.
//
// The snapshot must hold only changes the caller is entitled to queue: a queue
// decision says only that no dependency holds the change, not whose change it
// is or that it is ready. Rebuilding a dependant on the default branch and the
// full test run before it is queued stay the caller's (the spec's part 2).
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
	// ActionRefuse: the change's wait is refused, for Reason.
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
	// or past its deadline. It is zero for a refusal of a change with no
	// recorded start or with one later than the records' time.
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
// order the records were read in. pass does not change the records it reads.
//
// It is unexported because nothing in production calls it yet, only its
// tests. That takes it out of the exported-reach audit's view and does not
// satisfy the audit: step 2 of spc-2610230758581403 exports it and wires its
// first caller, or removes it (dcn-2611101309224327).
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
			defects = append(defects, fmt.Sprintf("change #%d has unknown state %q", c.Number, c.State))
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
		return nil, fmt.Errorf("coord: the records cannot be judged: %s", strings.Join(slices.Compact(defects), "; "))
	}

	open := make([]Change, 0, len(r.Changes))
	for _, c := range r.Changes {
		if c.State == StateOpen {
			open = append(open, c)
		}
	}
	slices.SortFunc(open, func(a, b Change) int { return a.Number - b.Number })

	decisions := make([]Decision, 0, len(open))
	for _, c := range open {
		decisions = append(decisions, decide(r, states, c))
	}
	return decisions, nil
}

func decide(r Records, states map[int]State, c Change) Decision {
	var waits []int
	for _, n := range c.DependsOn {
		if states[n] != StateMerged {
			waits = append(waits, n)
		}
	}
	if len(waits) == 0 {
		return Decision{Change: c.Number, Action: ActionQueue}
	}
	slices.Sort(waits)
	waits = slices.Compact(waits)
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

func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func names(nums []int) string {
	parts := make([]string, len(nums))
	for i, n := range nums {
		parts[i] = fmt.Sprintf("#%d", n)
	}
	return strings.Join(parts, ", ")
}
