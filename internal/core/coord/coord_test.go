package coord

import (
	"math/rand"
	"reflect"
	"testing"
	"time"
)

var (
	t0    = time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	limit = 2 * time.Hour
)

func records(now time.Time, changes ...Change) Records {
	return Records{Now: now, WaitLimit: limit, Changes: changes}
}

// at is 10 November 2026 at hh:mm in UTC, the day every test judges on.
func at(hh, mm int) time.Time { return time.Date(2026, 11, 10, hh, mm, 0, 0, time.UTC) }

// TestPassDecides: the table of what the records call for. An open change no
// dependency holds is queued; one waiting on anything else waits until a
// deadline derived from the records; a wait at or past its deadline is
// refused, naming what it waited on and for how long; a recorded start later
// than the records' time is refused; a change already queued, merged or
// closed needs nothing. A change depending on itself or on a cycle is never
// queued: it waits, then is refused at its deadline.
func TestPassDecides(t *testing.T) {
	t.Parallel()
	start := at(8, 30)
	deadline := at(10, 30)
	plus2 := time.FixedZone("UTC+2", 2*60*60)
	cases := []struct {
		name string
		in   Records
		want []Decision
	}{
		{"no records", records(t0), []Decision{}},
		{"open, no dependency, is queued",
			records(t0, Change{Number: 7, State: StateOpen}),
			[]Decision{{Change: 7, Action: ActionQueue}}},
		{"open, every dependency merged, is queued",
			records(t0,
				Change{Number: 1, State: StateMerged},
				Change{Number: 2, State: StateMerged},
				Change{Number: 3, State: StateOpen, DependsOn: []int{2, 1}, WaitingSince: start}),
			[]Decision{{Change: 3, Action: ActionQueue}}},
		{"open on an open dependency waits until the derived deadline",
			records(t0,
				Change{Number: 1, State: StateOpen},
				Change{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: start}),
			[]Decision{
				{Change: 1, Action: ActionQueue},
				{Change: 2, Action: ActionWait, WaitsOn: []int{1}, Until: deadline},
			}},
		{"a queued dependency still holds the wait",
			records(t0,
				Change{Number: 1, State: StateQueued},
				Change{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: start}),
			[]Decision{{Change: 2, Action: ActionWait, WaitsOn: []int{1}, Until: deadline}}},
		{"only the unmerged dependencies are waited on, sorted and once each",
			records(t0,
				Change{Number: 1, State: StateMerged},
				Change{Number: 4, State: StateOpen, DependsOn: []int{9, 1, 5, 9}, WaitingSince: start},
				Change{Number: 5, State: StateClosed}),
			[]Decision{{Change: 4, Action: ActionWait, WaitsOn: []int{5, 9}, Until: deadline}}},
		{"a wait one nanosecond before its deadline still waits",
			records(deadline.Add(-time.Nanosecond),
				Change{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: start}),
			[]Decision{{Change: 2, Action: ActionWait, WaitsOn: []int{1}, Until: deadline}}},
		{"a wait at its deadline is refused, naming what and how long",
			records(deadline,
				Change{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: start}),
			[]Decision{{Change: 2, Action: ActionRefuse, WaitsOn: []int{1}, Until: deadline,
				Reason: "waited on #1 for the 2h0m0s limit since 2026-11-10T08:30:00Z; deadline 2026-11-10T10:30:00Z reached"}}},
		{"a wait well past its deadline is refused, naming what and how long",
			records(at(23, 0),
				Change{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: start}),
			[]Decision{{Change: 2, Action: ActionRefuse, WaitsOn: []int{1}, Until: deadline,
				Reason: "waited on #1 for the 2h0m0s limit since 2026-11-10T08:30:00Z; deadline 2026-11-10T10:30:00Z reached"}}},
		{"a wait with no recorded start is refused, naming the stated limit",
			records(t0,
				Change{Number: 2, State: StateOpen, DependsOn: []int{1}}),
			[]Decision{{Change: 2, Action: ActionRefuse, WaitsOn: []int{1},
				Reason: "waits on #1 with no recorded start, so no deadline can be derived from the 2h0m0s limit"}}},
		{"a recorded start later than the records' time is refused, naming both",
			records(t0,
				Change{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: at(9, 0).Add(time.Nanosecond)}),
			[]Decision{{Change: 2, Action: ActionRefuse, WaitsOn: []int{1},
				Reason: "waits on #1 from a recorded start of 2026-11-10T09:00:00.000000001Z, later than the records' time of 2026-11-10T09:00:00Z, so no deadline is derived from it"}}},
		{"a start years ahead is refused, not waited on",
			records(t0,
				Change{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: time.Date(2999, 1, 1, 0, 0, 0, 0, time.UTC)}),
			[]Decision{{Change: 2, Action: ActionRefuse, WaitsOn: []int{1},
				Reason: "waits on #1 from a recorded start of 2999-01-01T00:00:00Z, later than the records' time of 2026-11-10T09:00:00Z, so no deadline is derived from it"}}},
		{"a start equal to the records' time waits",
			records(t0,
				Change{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: t0}),
			[]Decision{{Change: 2, Action: ActionWait, WaitsOn: []int{1}, Until: at(11, 0)}}},
		{"a deadline is named in UTC whatever zone the start was read in",
			records(t0,
				Change{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: time.Date(2026, 11, 10, 10, 30, 0, 0, plus2)}),
			[]Decision{{Change: 2, Action: ActionWait, WaitsOn: []int{1}, Until: deadline}}},
		{"a change depending on itself waits, never queued",
			records(t0,
				Change{Number: 5, State: StateOpen, DependsOn: []int{5}, WaitingSince: start}),
			[]Decision{{Change: 5, Action: ActionWait, WaitsOn: []int{5}, Until: deadline}}},
		{"a change depending on itself is refused at its deadline",
			records(deadline,
				Change{Number: 5, State: StateOpen, DependsOn: []int{5}, WaitingSince: start}),
			[]Decision{{Change: 5, Action: ActionRefuse, WaitsOn: []int{5}, Until: deadline,
				Reason: "waited on #5 for the 2h0m0s limit since 2026-11-10T08:30:00Z; deadline 2026-11-10T10:30:00Z reached"}}},
		{"a two-change cycle is never queued: one waits, the other is refused at its deadline",
			records(t0,
				Change{Number: 6, State: StateOpen, DependsOn: []int{7}, WaitingSince: start},
				Change{Number: 7, State: StateOpen, DependsOn: []int{6}, WaitingSince: at(7, 0)}),
			[]Decision{
				{Change: 6, Action: ActionWait, WaitsOn: []int{7}, Until: deadline},
				{Change: 7, Action: ActionRefuse, WaitsOn: []int{6}, Until: at(9, 0),
					Reason: "waited on #6 for the 2h0m0s limit since 2026-11-10T07:00:00Z; deadline 2026-11-10T09:00:00Z reached"},
			}},
		{"each change of a three-change cycle waits, none queued",
			records(t0,
				Change{Number: 8, State: StateOpen, DependsOn: []int{9}, WaitingSince: start},
				Change{Number: 9, State: StateOpen, DependsOn: []int{10}, WaitingSince: start},
				Change{Number: 10, State: StateOpen, DependsOn: []int{8}, WaitingSince: start}),
			[]Decision{
				{Change: 8, Action: ActionWait, WaitsOn: []int{9}, Until: deadline},
				{Change: 9, Action: ActionWait, WaitsOn: []int{10}, Until: deadline},
				{Change: 10, Action: ActionWait, WaitsOn: []int{8}, Until: deadline},
			}},
		{"queued, merged and closed changes need nothing",
			records(t0,
				Change{Number: 1, State: StateQueued},
				Change{Number: 2, State: StateMerged},
				Change{Number: 3, State: StateClosed, DependsOn: []int{4}}),
			[]Decision{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := pass(tc.in)
			if err != nil {
				t.Fatalf("pass: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("pass:\n got  %+v\n want %+v", got, tc.want)
			}
		})
	}
}

// TestPassDeadlineIsDerivedFromTheRecords: two passes at different times
// before the deadline name the same deadline, because it comes from the
// recorded start and the stated limit, never from when a process started.
func TestPassDeadlineIsDerivedFromTheRecords(t *testing.T) {
	t.Parallel()
	ch := []Change{
		{Number: 1, State: StateOpen},
		{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: at(8, 0)},
	}
	want := []Decision{
		{Change: 1, Action: ActionQueue},
		{Change: 2, Action: ActionWait, WaitsOn: []int{1}, Until: at(10, 0)},
	}
	for _, now := range []time.Time{t0, at(9, 59)} {
		got, err := pass(records(now, ch...))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("pass at %s:\n got  %+v\n want %+v", now.Format(time.RFC3339), got, want)
		}
	}
}

// sample is a snapshot of records at t0 with the decisions it calls for,
// written out by hand.
type sample struct {
	name    string
	changes func() []Change
	want    []Decision
}

func samples() []sample {
	return []sample{
		{"mixed", func() []Change {
			return []Change{
				{Number: 1, State: StateMerged},
				{Number: 2, State: StateOpen},
				{Number: 3, State: StateOpen, DependsOn: []int{1}, WaitingSince: at(8, 20)},
				{Number: 4, State: StateOpen, DependsOn: []int{2, 3}, WaitingSince: at(8, 20)},
				{Number: 5, State: StateQueued},
				{Number: 6, State: StateOpen, DependsOn: []int{5}, WaitingSince: at(6, 0)},
				{Number: 7, State: StateOpen, DependsOn: []int{40}},
				{Number: 8, State: StateClosed},
				{Number: 9, State: StateOpen},
			}
		}, []Decision{
			{Change: 2, Action: ActionQueue},
			{Change: 3, Action: ActionQueue},
			{Change: 4, Action: ActionWait, WaitsOn: []int{2, 3}, Until: at(10, 20)},
			{Change: 6, Action: ActionRefuse, WaitsOn: []int{5}, Until: at(8, 0),
				Reason: "waited on #5 for the 2h0m0s limit since 2026-11-10T06:00:00Z; deadline 2026-11-10T08:00:00Z reached"},
			{Change: 7, Action: ActionRefuse, WaitsOn: []int{40},
				Reason: "waits on #40 with no recorded start, so no deadline can be derived from the 2h0m0s limit"},
			{Change: 9, Action: ActionQueue},
		}},
		{"self and cycles", func() []Change {
			return []Change{
				{Number: 10, State: StateOpen, DependsOn: []int{10}, WaitingSince: at(8, 30)},
				{Number: 11, State: StateOpen, DependsOn: []int{12}, WaitingSince: at(8, 30)},
				{Number: 12, State: StateOpen, DependsOn: []int{11}, WaitingSince: at(8, 30)},
				{Number: 13, State: StateOpen, DependsOn: []int{14}, WaitingSince: at(8, 30)},
				{Number: 14, State: StateOpen, DependsOn: []int{15}, WaitingSince: at(8, 30)},
				{Number: 15, State: StateOpen, DependsOn: []int{13}, WaitingSince: at(8, 30)},
				{Number: 16, State: StateOpen},
				{Number: 17, State: StateOpen, DependsOn: []int{16, 16}, WaitingSince: at(8, 30)},
				{Number: 18, State: StateOpen},
				{Number: 19, State: StateOpen, DependsOn: []int{20}, WaitingSince: at(6, 0)},
				{Number: 20, State: StateOpen, DependsOn: []int{19}, WaitingSince: at(8, 30)},
			}
		}, []Decision{
			{Change: 10, Action: ActionWait, WaitsOn: []int{10}, Until: at(10, 30)},
			{Change: 11, Action: ActionWait, WaitsOn: []int{12}, Until: at(10, 30)},
			{Change: 12, Action: ActionWait, WaitsOn: []int{11}, Until: at(10, 30)},
			{Change: 13, Action: ActionWait, WaitsOn: []int{14}, Until: at(10, 30)},
			{Change: 14, Action: ActionWait, WaitsOn: []int{15}, Until: at(10, 30)},
			{Change: 15, Action: ActionWait, WaitsOn: []int{13}, Until: at(10, 30)},
			{Change: 16, Action: ActionQueue},
			{Change: 17, Action: ActionWait, WaitsOn: []int{16}, Until: at(10, 30)},
			{Change: 18, Action: ActionQueue},
			{Change: 19, Action: ActionRefuse, WaitsOn: []int{20}, Until: at(8, 0),
				Reason: "waited on #20 for the 2h0m0s limit since 2026-11-10T06:00:00Z; deadline 2026-11-10T08:00:00Z reached"},
			{Change: 20, Action: ActionWait, WaitsOn: []int{19}, Until: at(10, 30)},
		}},
		{"future start and zones", func() []Change {
			return []Change{
				{Number: 21, State: StateOpen, DependsOn: []int{22}, WaitingSince: at(10, 0)},
				{Number: 22, State: StateOpen},
				{Number: 23, State: StateOpen, DependsOn: []int{22}, WaitingSince: time.Date(2026, 11, 10, 10, 0, 0, 0, time.FixedZone("UTC+2", 2*60*60))},
				{Number: 24, State: StateMerged},
				{Number: 25, State: StateOpen, DependsOn: []int{24}, WaitingSince: at(8, 0)},
				{Number: 26, State: StateOpen, DependsOn: []int{22}, WaitingSince: at(7, 0).Add(time.Nanosecond)},
			}
		}, []Decision{
			{Change: 21, Action: ActionRefuse, WaitsOn: []int{22},
				Reason: "waits on #22 from a recorded start of 2026-11-10T10:00:00Z, later than the records' time of 2026-11-10T09:00:00Z, so no deadline is derived from it"},
			{Change: 22, Action: ActionQueue},
			{Change: 23, Action: ActionWait, WaitsOn: []int{22}, Until: at(10, 0)},
			{Change: 25, Action: ActionQueue},
			{Change: 26, Action: ActionWait, WaitsOn: []int{22}, Until: at(9, 0).Add(time.Nanosecond)},
		}},
	}
}

func shuffled(rng *rand.Rand, ch []Change) []Change {
	rng.Shuffle(len(ch), func(a, b int) { ch[a], ch[b] = ch[b], ch[a] })
	for j := range ch {
		d := ch[j].DependsOn
		rng.Shuffle(len(d), func(a, b int) { d[a], d[b] = d[b], d[a] })
	}
	return ch
}

// TestPassSameRecordsSameDecisions: over each sample, the records give the
// decisions written out for them in any order the records and their
// dependencies are read in, on every run, and the pass leaves the records it
// read as it found them.
func TestPassSameRecordsSameDecisions(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewSource(1))
	for _, s := range samples() {
		for i := 0; i < 50; i++ {
			ch := s.changes()
			if i > 0 {
				ch = shuffled(rng, ch)
			}
			before := cloneChanges(ch)
			got, err := pass(records(t0, ch...))
			if err != nil {
				t.Fatalf("%s, read %d: %v", s.name, i, err)
			}
			if !reflect.DeepEqual(got, s.want) {
				t.Fatalf("%s, read %d:\n got  %+v\n want %+v", s.name, i, got, s.want)
			}
			if !reflect.DeepEqual(ch, before) {
				t.Fatalf("%s, read %d: the pass changed the records it read", s.name, i)
			}
		}
	}
}

// TestPassStoppedMidwayReachesTheSameDecisions: over each sample, a pass whose
// acts stopped after any subset of its queue decisions, re-run from the records
// those acts left, gives every decision written out for the sample except
// those for the changes the acts recorded as queued. A wait or a refusal
// records nothing in this step, so it is given again. This is the
// same-moment half of technical criterion 2 of the spec's step 1;
// TestPassAcrossPasses holds the records and the time moving between passes.
func TestPassStoppedMidwayReachesTheSameDecisions(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewSource(2))
	for _, s := range samples() {
		var queue []int
		for _, d := range s.want {
			if d.Action == ActionQueue {
				queue = append(queue, d.Change)
			}
		}
		if len(queue) < 2 {
			t.Fatalf("%s: the sample needs at least two queue decisions, has %d", s.name, len(queue))
		}
		for mask := 0; mask < 1<<len(queue); mask++ {
			acted := map[int]bool{}
			for i, n := range queue {
				if mask&(1<<i) != 0 {
					acted[n] = true
				}
			}
			ch := s.changes()
			for j := range ch {
				if acted[ch[j].Number] {
					ch[j].State = StateQueued
				}
			}
			remaining := []Decision{}
			for _, d := range s.want {
				if !acted[d.Change] {
					remaining = append(remaining, d)
				}
			}
			got, err := pass(records(t0, shuffled(rng, ch)...))
			if err != nil {
				t.Fatalf("%s, acts %b: %v", s.name, mask, err)
			}
			if !reflect.DeepEqual(got, remaining) {
				t.Fatalf("%s, acts %b:\n got  %+v\n want %+v", s.name, mask, got, remaining)
			}
		}
	}
}

// TestPassAcrossPasses: a sequence of passes over records that move between
// them, each pass's decisions written out by hand. Queueing a dependency
// keeps its dependant waiting, the dependency merging releases it, and a later
// time crossing a deadline turns that wait into a refusal.
func TestPassAcrossPasses(t *testing.T) {
	t.Parallel()
	ch := []Change{
		{Number: 1, State: StateOpen},
		{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: at(8, 30)},
		{Number: 3, State: StateOpen, DependsOn: []int{2}, WaitingSince: at(8, 30)},
	}
	steps := []struct {
		name string
		now  time.Time
		act  func()
		want []Decision
	}{
		{"first pass", t0, func() {}, []Decision{
			{Change: 1, Action: ActionQueue},
			{Change: 2, Action: ActionWait, WaitsOn: []int{1}, Until: at(10, 30)},
			{Change: 3, Action: ActionWait, WaitsOn: []int{2}, Until: at(10, 30)},
		}},
		{"#1 queued", t0, func() { ch[0].State = StateQueued }, []Decision{
			{Change: 2, Action: ActionWait, WaitsOn: []int{1}, Until: at(10, 30)},
			{Change: 3, Action: ActionWait, WaitsOn: []int{2}, Until: at(10, 30)},
		}},
		{"#1 merged", at(9, 30), func() { ch[0].State = StateMerged }, []Decision{
			{Change: 2, Action: ActionQueue},
			{Change: 3, Action: ActionWait, WaitsOn: []int{2}, Until: at(10, 30)},
		}},
		{"#2 queued, a nanosecond before the deadline", at(10, 30).Add(-time.Nanosecond), func() { ch[1].State = StateQueued }, []Decision{
			{Change: 3, Action: ActionWait, WaitsOn: []int{2}, Until: at(10, 30)},
		}},
		{"#2 never merges and the deadline passes", at(11, 30), func() {}, []Decision{
			{Change: 3, Action: ActionRefuse, WaitsOn: []int{2}, Until: at(10, 30),
				Reason: "waited on #2 for the 2h0m0s limit since 2026-11-10T08:30:00Z; deadline 2026-11-10T10:30:00Z reached"},
		}},
	}
	for _, st := range steps {
		st.act()
		got, err := pass(records(st.now, cloneChanges(ch)...))
		if err != nil {
			t.Fatalf("%s: %v", st.name, err)
		}
		if !reflect.DeepEqual(got, st.want) {
			t.Fatalf("%s:\n got  %+v\n want %+v", st.name, got, st.want)
		}
	}
}

// TestPassRefusesRecordsItCannotRead: a snapshot the pass cannot judge is
// refused whole, naming why, rather than half-decided. A limit of exactly the
// most is judged.
func TestPassRefusesRecordsItCannotRead(t *testing.T) {
	t.Parallel()
	const prefix = "coord: the records cannot be judged: "
	cases := []struct {
		name string
		in   Records
		want string
	}{
		{"no time", Records{WaitLimit: limit}, "the records carry no time to judge them by"},
		{"no wait limit", Records{Now: t0}, "the records state no positive wait limit (0s)"},
		{"negative wait limit", Records{Now: t0, WaitLimit: -time.Second}, "the records state no positive wait limit (-1s)"},
		{"a wait limit over the most", Records{Now: t0, WaitLimit: 24*time.Hour + time.Nanosecond},
			"the records state a wait limit of 24h0m0.000000001s, above the most of 24h0m0s"},
		{"a change with no number", records(t0, Change{State: StateOpen}), "a change carries number 0"},
		{"a change read twice", records(t0,
			Change{Number: 3, State: StateOpen}, Change{Number: 3, State: StateMerged}), "change #3 is in the records more than once"},
		{"an unknown state", records(t0, Change{Number: 3, State: "draft"}), `change #3 has unknown state "draft"`},
		{"a dependency numbered zero", records(t0, Change{Number: 4, State: StateOpen, DependsOn: []int{0}}),
			"change #4 depends on number 0"},
		{"a dependency numbered below zero", records(t0, Change{Number: 4, State: StateMerged, DependsOn: []int{-5}}),
			"change #4 depends on number -5"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := pass(tc.in)
			if err == nil {
				t.Fatalf("pass accepted the records and gave %+v", got)
			}
			if want := prefix + tc.want; err.Error() != want {
				t.Fatalf("error\n got  %q\n want %q", err, want)
			}
		})
	}
	if _, err := pass(Records{Now: t0, WaitLimit: MaxWaitLimit}); err != nil {
		t.Fatalf("a limit of exactly %s was refused: %v", MaxWaitLimit, err)
	}
	if MaxWaitLimit != 24*time.Hour {
		t.Fatalf("MaxWaitLimit is %s, want 24h0m0s", MaxWaitLimit)
	}
}

// TestPassRefusalNamesEveryDefectInAnyOrder: a snapshot with several defects
// is refused with one error naming all of them, the same error whatever order
// the records and their dependencies are read in.
func TestPassRefusalNamesEveryDefectInAnyOrder(t *testing.T) {
	t.Parallel()
	const want = "coord: the records cannot be judged: " +
		"a change carries number 0; " +
		`change #3 has unknown state "draft"; ` +
		"change #3 is in the records more than once; " +
		"change #4 depends on number -1; " +
		"change #4 depends on number 0; " +
		"the records carry no time to judge them by"
	defective := func() []Change {
		return []Change{
			{Number: 0, State: StateOpen},
			{Number: 0, State: StateOpen},
			{Number: 3, State: "draft"},
			{Number: 3, State: StateMerged},
			{Number: 4, State: StateOpen, DependsOn: []int{-1, 0, 2, 0}},
			{Number: 5, State: StateOpen},
		}
	}
	rng := rand.New(rand.NewSource(3))
	for i := 0; i < 50; i++ {
		ch := defective()
		if i > 0 {
			ch = shuffled(rng, ch)
		}
		got, err := pass(Records{WaitLimit: limit, Changes: ch})
		if err == nil {
			t.Fatalf("read %d: pass accepted the records and gave %+v", i, got)
		}
		if err.Error() != want {
			t.Fatalf("read %d: error\n got  %q\n want %q", i, err, want)
		}
	}
}

func cloneChanges(in []Change) []Change {
	out := make([]Change, len(in))
	for i, c := range in {
		c.DependsOn = append([]int(nil), c.DependsOn...)
		out[i] = c
	}
	return out
}
