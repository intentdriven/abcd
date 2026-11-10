package coord

import (
	"math/rand"
	"reflect"
	"slices"
	"strings"
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
// closed needs nothing. A direct dependency closed unmerged or with no record,
// and a change on or reaching a cycle of unmerged changes, is refused at once,
// before any start or deadline is judged.
func TestPassDecides(t *testing.T) {
	t.Parallel()
	start := at(8, 30)
	deadline := at(10, 30)
	plus2 := time.FixedZone("UTC+2", 2*60*60)
	q1 := Change{Number: 1, State: StateQueued}
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
				Change{Number: 5, State: StateQueued},
				Change{Number: 9, State: StateQueued}),
			[]Decision{{Change: 4, Action: ActionWait, WaitsOn: []int{5, 9}, Until: deadline}}},
		{"a wait one nanosecond before its deadline still waits",
			records(deadline.Add(-time.Nanosecond),
				q1, Change{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: start}),
			[]Decision{{Change: 2, Action: ActionWait, WaitsOn: []int{1}, Until: deadline}}},
		{"a wait at its deadline is refused, naming what and how long",
			records(deadline,
				q1, Change{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: start}),
			[]Decision{{Change: 2, Action: ActionRefuse, WaitsOn: []int{1}, Until: deadline,
				Reason: "waited on #1 for the 2h0m0s limit since 2026-11-10T08:30:00Z; deadline 2026-11-10T10:30:00Z reached"}}},
		{"a wait well past its deadline is refused, naming what and how long",
			records(at(23, 0),
				q1, Change{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: start}),
			[]Decision{{Change: 2, Action: ActionRefuse, WaitsOn: []int{1}, Until: deadline,
				Reason: "waited on #1 for the 2h0m0s limit since 2026-11-10T08:30:00Z; deadline 2026-11-10T10:30:00Z reached"}}},
		{"a wait with no recorded start is refused, naming the stated limit",
			records(t0,
				q1, Change{Number: 2, State: StateOpen, DependsOn: []int{1}}),
			[]Decision{{Change: 2, Action: ActionRefuse, WaitsOn: []int{1},
				Reason: "waits on #1 with no recorded start, so no deadline can be derived from the 2h0m0s limit"}}},
		{"a recorded start later than the records' time is refused, naming both",
			records(t0,
				q1, Change{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: at(9, 0).Add(time.Nanosecond)}),
			[]Decision{{Change: 2, Action: ActionRefuse, WaitsOn: []int{1},
				Reason: "waits on #1 from a recorded start of 2026-11-10T09:00:00.000000001Z, later than the records' time of 2026-11-10T09:00:00Z, so no deadline is derived from it"}}},
		{"a start years ahead is refused, not waited on",
			records(t0,
				q1, Change{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: time.Date(2999, 1, 1, 0, 0, 0, 0, time.UTC)}),
			[]Decision{{Change: 2, Action: ActionRefuse, WaitsOn: []int{1},
				Reason: "waits on #1 from a recorded start of 2999-01-01T00:00:00Z, later than the records' time of 2026-11-10T09:00:00Z, so no deadline is derived from it"}}},
		{"a start equal to the records' time waits",
			records(t0,
				q1, Change{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: t0}),
			[]Decision{{Change: 2, Action: ActionWait, WaitsOn: []int{1}, Until: at(11, 0)}}},
		{"a deadline is named in UTC whatever zone the start was read in",
			records(t0,
				q1, Change{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: time.Date(2026, 11, 10, 10, 30, 0, 0, plus2)}),
			[]Decision{{Change: 2, Action: ActionWait, WaitsOn: []int{1}, Until: deadline}}},
		{"a refusal past a deadline from a zoned start names both times in UTC",
			records(at(23, 0),
				q1, Change{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: time.Date(2026, 11, 10, 10, 30, 0, 0, plus2)}),
			[]Decision{{Change: 2, Action: ActionRefuse, WaitsOn: []int{1}, Until: deadline,
				Reason: "waited on #1 for the 2h0m0s limit since 2026-11-10T08:30:00Z; deadline 2026-11-10T10:30:00Z reached"}}},
		{"a refusal of a zoned start in the future names it in UTC",
			records(t0,
				q1, Change{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: time.Date(2026, 11, 10, 12, 0, 0, 0, plus2)}),
			[]Decision{{Change: 2, Action: ActionRefuse, WaitsOn: []int{1},
				Reason: "waits on #1 from a recorded start of 2026-11-10T10:00:00Z, later than the records' time of 2026-11-10T09:00:00Z, so no deadline is derived from it"}}},
		{"a closed dependency is refused at once, before its deadline",
			records(deadline.Add(-time.Nanosecond),
				Change{Number: 1, State: StateClosed},
				Change{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: start}),
			[]Decision{{Change: 2, Action: ActionRefuse, WaitsOn: []int{1},
				Reason: atOnce + "depends on #1, closed unmerged"}}},
		{"a closed dependency is refused at once, at its deadline",
			records(deadline,
				Change{Number: 1, State: StateClosed},
				Change{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: start}),
			[]Decision{{Change: 2, Action: ActionRefuse, WaitsOn: []int{1},
				Reason: atOnce + "depends on #1, closed unmerged"}}},
		{"a closed dependency is refused at once, past its deadline",
			records(at(23, 0),
				Change{Number: 1, State: StateClosed},
				Change{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: start}),
			[]Decision{{Change: 2, Action: ActionRefuse, WaitsOn: []int{1},
				Reason: atOnce + "depends on #1, closed unmerged"}}},
		{"a closed dependency is refused at once, with no recorded start",
			records(t0,
				Change{Number: 1, State: StateClosed},
				Change{Number: 2, State: StateOpen, DependsOn: []int{1}}),
			[]Decision{{Change: 2, Action: ActionRefuse, WaitsOn: []int{1},
				Reason: atOnce + "depends on #1, closed unmerged"}}},
		{"a closed dependency is refused at once, with a start in the future",
			records(t0,
				Change{Number: 1, State: StateClosed},
				Change{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: at(12, 0)}),
			[]Decision{{Change: 2, Action: ActionRefuse, WaitsOn: []int{1},
				Reason: atOnce + "depends on #1, closed unmerged"}}},
		{"a dependency with no record is refused at once",
			records(t0,
				Change{Number: 2, State: StateOpen, DependsOn: []int{41, 40}, WaitingSince: start}),
			[]Decision{{Change: 2, Action: ActionRefuse, WaitsOn: []int{40, 41},
				Reason: atOnce + "depends on #40, #41, each of which has no record"}}},
		{"closed dependencies read twice and out of order are named sorted and once each",
			records(t0,
				Change{Number: 91, State: StateClosed},
				Change{Number: 92, State: StateClosed},
				Change{Number: 2, State: StateOpen, DependsOn: []int{92, 91, 92, 91}, WaitingSince: start}),
			[]Decision{{Change: 2, Action: ActionRefuse, WaitsOn: []int{91, 92},
				Reason: atOnce + "depends on #91, #92, closed unmerged"}}},
		{"dependencies with no record read twice and out of order are named sorted and once each",
			records(t0,
				Change{Number: 2, State: StateOpen, DependsOn: []int{92, 91, 92, 91}, WaitingSince: start}),
			[]Decision{{Change: 2, Action: ActionRefuse, WaitsOn: []int{91, 92},
				Reason: atOnce + "depends on #91, #92, each of which has no record"}}},
		{"ten distinct dependencies read many times over are judged",
			records(t0, append(merged(1, 10),
				Change{Number: 11, State: StateOpen, DependsOn: repeated(10, 1000), WaitingSince: start})...),
			[]Decision{{Change: 11, Action: ActionQueue}}},
		{"a change on an open change that is itself refused waits to its own deadline",
			records(t0,
				Change{Number: 1, State: StateOpen, DependsOn: []int{99}, WaitingSince: start},
				Change{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: start}),
			[]Decision{
				{Change: 1, Action: ActionRefuse, WaitsOn: []int{99}, Reason: atOnce + "depends on #99, which has no record"},
				{Change: 2, Action: ActionWait, WaitsOn: []int{1}, Until: deadline},
			}},
		{"a change depending on itself is refused at once",
			records(t0,
				Change{Number: 5, State: StateOpen, DependsOn: []int{5}, WaitingSince: start}),
			[]Decision{{Change: 5, Action: ActionRefuse, WaitsOn: []int{5},
				Reason: atOnce + "is on or reaches a cycle of unmerged changes among #5"}}},
		{"each change of a two-change cycle is refused at once",
			records(t0,
				Change{Number: 6, State: StateOpen, DependsOn: []int{7}, WaitingSince: start},
				Change{Number: 7, State: StateOpen, DependsOn: []int{6}, WaitingSince: at(7, 0)}),
			[]Decision{
				{Change: 6, Action: ActionRefuse, WaitsOn: []int{7}, Reason: atOnce + "is on or reaches a cycle of unmerged changes among #6, #7"},
				{Change: 7, Action: ActionRefuse, WaitsOn: []int{6}, Reason: atOnce + "is on or reaches a cycle of unmerged changes among #6, #7"},
			}},
		{"a cycle member with no recorded start is refused at once for the cycle",
			records(t0,
				Change{Number: 6, State: StateOpen, DependsOn: []int{7}},
				Change{Number: 7, State: StateOpen, DependsOn: []int{6}, WaitingSince: start}),
			[]Decision{
				{Change: 6, Action: ActionRefuse, WaitsOn: []int{7}, Reason: atOnce + "is on or reaches a cycle of unmerged changes among #6, #7"},
				{Change: 7, Action: ActionRefuse, WaitsOn: []int{6}, Reason: atOnce + "is on or reaches a cycle of unmerged changes among #6, #7"},
			}},
		{"a cycle member with a start in the future is refused at once for the cycle",
			records(t0,
				Change{Number: 6, State: StateOpen, DependsOn: []int{7}, WaitingSince: at(12, 0)},
				Change{Number: 7, State: StateOpen, DependsOn: []int{6}, WaitingSince: start}),
			[]Decision{
				{Change: 6, Action: ActionRefuse, WaitsOn: []int{7}, Reason: atOnce + "is on or reaches a cycle of unmerged changes among #6, #7"},
				{Change: 7, Action: ActionRefuse, WaitsOn: []int{6}, Reason: atOnce + "is on or reaches a cycle of unmerged changes among #6, #7"},
			}},
		{"each change of a three-change cycle is refused at once",
			records(t0,
				Change{Number: 8, State: StateOpen, DependsOn: []int{9}, WaitingSince: start},
				Change{Number: 9, State: StateOpen, DependsOn: []int{10}, WaitingSince: start},
				Change{Number: 10, State: StateOpen, DependsOn: []int{8}, WaitingSince: start}),
			[]Decision{
				{Change: 8, Action: ActionRefuse, WaitsOn: []int{9}, Reason: atOnce + "is on or reaches a cycle of unmerged changes among #8, #9, #10"},
				{Change: 9, Action: ActionRefuse, WaitsOn: []int{10}, Reason: atOnce + "is on or reaches a cycle of unmerged changes among #8, #9, #10"},
				{Change: 10, Action: ActionRefuse, WaitsOn: []int{8}, Reason: atOnce + "is on or reaches a cycle of unmerged changes among #8, #9, #10"},
			}},
		{"a cycle through a queued change refuses its open member",
			records(t0,
				Change{Number: 30, State: StateOpen, DependsOn: []int{31}, WaitingSince: start},
				Change{Number: 31, State: StateQueued, DependsOn: []int{30}}),
			[]Decision{{Change: 30, Action: ActionRefuse, WaitsOn: []int{31},
				Reason: atOnce + "is on or reaches a cycle of unmerged changes among #30, #31"}}},
		{"a tail reaching a cycle is refused, naming the cycle",
			records(t0,
				Change{Number: 40, State: StateOpen, DependsOn: []int{41}, WaitingSince: start},
				Change{Number: 41, State: StateQueued, DependsOn: []int{42}},
				Change{Number: 42, State: StateQueued, DependsOn: []int{43}},
				Change{Number: 43, State: StateQueued, DependsOn: []int{42}}),
			[]Decision{{Change: 40, Action: ActionRefuse, WaitsOn: []int{41},
				Reason: atOnce + "is on or reaches a cycle of unmerged changes among #42, #43"}}},
		{"a change reaching two disjoint cycles names both",
			records(t0,
				Change{Number: 50, State: StateQueued, DependsOn: []int{51}},
				Change{Number: 51, State: StateQueued, DependsOn: []int{50}},
				Change{Number: 52, State: StateQueued, DependsOn: []int{53}},
				Change{Number: 53, State: StateQueued, DependsOn: []int{52}},
				Change{Number: 54, State: StateOpen, DependsOn: []int{52, 50}, WaitingSince: start}),
			[]Decision{{Change: 54, Action: ActionRefuse, WaitsOn: []int{50, 52},
				Reason: atOnce + "is on or reaches a cycle of unmerged changes among #50, #51, #52, #53"}}},
		{"a diamond is not a cycle",
			records(t0,
				Change{Number: 60, State: StateOpen, DependsOn: []int{61, 62}, WaitingSince: start},
				Change{Number: 61, State: StateOpen, DependsOn: []int{63}, WaitingSince: start},
				Change{Number: 62, State: StateOpen, DependsOn: []int{63}, WaitingSince: start},
				Change{Number: 63, State: StateOpen}),
			[]Decision{
				{Change: 60, Action: ActionWait, WaitsOn: []int{61, 62}, Until: deadline},
				{Change: 61, Action: ActionWait, WaitsOn: []int{63}, Until: deadline},
				{Change: 62, Action: ActionWait, WaitsOn: []int{63}, Until: deadline},
				{Change: 63, Action: ActionQueue},
			}},
		{"a cycle broken by a merged member is no cycle",
			records(t0,
				Change{Number: 70, State: StateOpen, DependsOn: []int{71}, WaitingSince: start},
				Change{Number: 71, State: StateMerged, DependsOn: []int{70}},
				Change{Number: 72, State: StateOpen, DependsOn: []int{73}, WaitingSince: start},
				Change{Number: 73, State: StateOpen, DependsOn: []int{74}, WaitingSince: start},
				Change{Number: 74, State: StateMerged, DependsOn: []int{72}}),
			[]Decision{
				{Change: 70, Action: ActionQueue},
				{Change: 72, Action: ActionWait, WaitsOn: []int{73}, Until: deadline},
				{Change: 73, Action: ActionQueue},
			}},
		{"a closed member of a loop is the cause, not a cycle",
			records(t0,
				Change{Number: 80, State: StateOpen, DependsOn: []int{81}, WaitingSince: start},
				Change{Number: 81, State: StateClosed, DependsOn: []int{80}}),
			[]Decision{{Change: 80, Action: ActionRefuse, WaitsOn: []int{81},
				Reason: atOnce + "depends on #81, closed unmerged"}}},
		{"all three causes on one change are named in a fixed order",
			records(deadline,
				Change{Number: 90, State: StateOpen, DependsOn: []int{93, 92, 91}, WaitingSince: start},
				Change{Number: 91, State: StateClosed},
				Change{Number: 93, State: StateOpen, DependsOn: []int{90}}),
			[]Decision{
				{Change: 90, Action: ActionRefuse, WaitsOn: []int{91, 92, 93},
					Reason: atOnce + "depends on #91, closed unmerged; depends on #92, which has no record; is on or reaches a cycle of unmerged changes among #90, #93"},
				{Change: 93, Action: ActionRefuse, WaitsOn: []int{90},
					Reason: atOnce + "is on or reaches a cycle of unmerged changes among #90, #93"},
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

const (
	atOnce = "refused at once, since no wait can release it: "
	cycle  = "is on or reaches a cycle of unmerged changes among "
)

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
			{Change: 7, Action: ActionRefuse, WaitsOn: []int{40}, Reason: atOnce + "depends on #40, which has no record"},
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
			{Change: 10, Action: ActionRefuse, WaitsOn: []int{10}, Reason: atOnce + cycle + "#10"},
			{Change: 11, Action: ActionRefuse, WaitsOn: []int{12}, Reason: atOnce + cycle + "#11, #12"},
			{Change: 12, Action: ActionRefuse, WaitsOn: []int{11}, Reason: atOnce + cycle + "#11, #12"},
			{Change: 13, Action: ActionRefuse, WaitsOn: []int{14}, Reason: atOnce + cycle + "#13, #14, #15"},
			{Change: 14, Action: ActionRefuse, WaitsOn: []int{15}, Reason: atOnce + cycle + "#13, #14, #15"},
			{Change: 15, Action: ActionRefuse, WaitsOn: []int{13}, Reason: atOnce + cycle + "#13, #14, #15"},
			{Change: 16, Action: ActionQueue},
			{Change: 17, Action: ActionWait, WaitsOn: []int{16}, Until: at(10, 30)},
			{Change: 18, Action: ActionQueue},
			{Change: 19, Action: ActionRefuse, WaitsOn: []int{20}, Reason: atOnce + cycle + "#19, #20"},
			{Change: 20, Action: ActionRefuse, WaitsOn: []int{19}, Reason: atOnce + cycle + "#19, #20"},
		}},
		{"at-once refusals", func() []Change {
			return []Change{
				{Number: 100, State: StateOpen, DependsOn: []int{101}, WaitingSince: at(8, 30)},
				{Number: 101, State: StateClosed},
				{Number: 102, State: StateOpen, DependsOn: []int{103}},
				{Number: 104, State: StateOpen, DependsOn: []int{105}, WaitingSince: at(8, 30)},
				{Number: 105, State: StateOpen, DependsOn: []int{104}, WaitingSince: at(8, 30)},
				{Number: 106, State: StateOpen, DependsOn: []int{104}, WaitingSince: at(8, 30)},
				{Number: 107, State: StateOpen, DependsOn: []int{109, 108}, WaitingSince: at(8, 30)},
				{Number: 108, State: StateOpen, DependsOn: []int{110}, WaitingSince: at(8, 30)},
				{Number: 109, State: StateOpen, DependsOn: []int{110}, WaitingSince: at(8, 30)},
				{Number: 110, State: StateOpen},
				{Number: 111, State: StateOpen, DependsOn: []int{102}, WaitingSince: at(8, 30)},
				{Number: 112, State: StateOpen},
			}
		}, []Decision{
			{Change: 100, Action: ActionRefuse, WaitsOn: []int{101}, Reason: atOnce + "depends on #101, closed unmerged"},
			{Change: 102, Action: ActionRefuse, WaitsOn: []int{103}, Reason: atOnce + "depends on #103, which has no record"},
			{Change: 104, Action: ActionRefuse, WaitsOn: []int{105}, Reason: atOnce + cycle + "#104, #105"},
			{Change: 105, Action: ActionRefuse, WaitsOn: []int{104}, Reason: atOnce + cycle + "#104, #105"},
			{Change: 106, Action: ActionRefuse, WaitsOn: []int{104}, Reason: atOnce + cycle + "#104, #105"},
			{Change: 107, Action: ActionWait, WaitsOn: []int{108, 109}, Until: at(10, 30)},
			{Change: 108, Action: ActionWait, WaitsOn: []int{110}, Until: at(10, 30)},
			{Change: 109, Action: ActionWait, WaitsOn: []int{110}, Until: at(10, 30)},
			{Change: 110, Action: ActionQueue},
			{Change: 111, Action: ActionWait, WaitsOn: []int{102}, Until: at(10, 30)},
			{Change: 112, Action: ActionQueue},
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
		if len(queue) < 2 || len(queue) > 8 {
			t.Fatalf("%s: the sample needs two to eight queue decisions, has %d", s.name, len(queue))
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
// keeps its dependant waiting, the dependency merging releases it, a
// dependency closing unmerged refuses its dependant at once, and a later time
// crossing a deadline turns a wait into a refusal.
func TestPassAcrossPasses(t *testing.T) {
	t.Parallel()
	ch := []Change{
		{Number: 1, State: StateOpen},
		{Number: 2, State: StateOpen, DependsOn: []int{1}, WaitingSince: at(8, 30)},
		{Number: 3, State: StateOpen, DependsOn: []int{2}, WaitingSince: at(8, 30)},
		{Number: 4, State: StateOpen, DependsOn: []int{5}, WaitingSince: at(8, 30)},
		{Number: 5, State: StateOpen},
	}
	closed := Decision{Change: 4, Action: ActionRefuse, WaitsOn: []int{5}, Reason: atOnce + "depends on #5, closed unmerged"}
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
			{Change: 4, Action: ActionWait, WaitsOn: []int{5}, Until: at(10, 30)},
			{Change: 5, Action: ActionQueue},
		}},
		{"#1 queued", t0, func() { ch[0].State = StateQueued }, []Decision{
			{Change: 2, Action: ActionWait, WaitsOn: []int{1}, Until: at(10, 30)},
			{Change: 3, Action: ActionWait, WaitsOn: []int{2}, Until: at(10, 30)},
			{Change: 4, Action: ActionWait, WaitsOn: []int{5}, Until: at(10, 30)},
			{Change: 5, Action: ActionQueue},
		}},
		{"#5 closes unmerged, well before the deadline", at(9, 10), func() { ch[4].State = StateClosed }, []Decision{
			{Change: 2, Action: ActionWait, WaitsOn: []int{1}, Until: at(10, 30)},
			{Change: 3, Action: ActionWait, WaitsOn: []int{2}, Until: at(10, 30)},
			closed,
		}},
		{"#1 merged", at(9, 30), func() { ch[0].State = StateMerged }, []Decision{
			{Change: 2, Action: ActionQueue},
			{Change: 3, Action: ActionWait, WaitsOn: []int{2}, Until: at(10, 30)},
			closed,
		}},
		{"#2 queued, a nanosecond before the deadline", at(10, 30).Add(-time.Nanosecond), func() { ch[1].State = StateQueued }, []Decision{
			{Change: 3, Action: ActionWait, WaitsOn: []int{2}, Until: at(10, 30)},
			closed,
		}},
		{"#2 never merges and the deadline passes", at(11, 30), func() {}, []Decision{
			{Change: 3, Action: ActionRefuse, WaitsOn: []int{2}, Until: at(10, 30),
				Reason: "waited on #2 for the 2h0m0s limit since 2026-11-10T08:30:00Z; deadline 2026-11-10T10:30:00Z reached"},
			closed,
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
		{"an unknown state of 32 runes is quoted whole", records(t0, Change{Number: 3, State: State(strings.Repeat("é", 32))}),
			`change #3 has unknown state "éééééééééééééééééééééééééééééééé"`},
		{"an unknown state of 33 runes is cut to 32", records(t0, Change{Number: 3, State: State(strings.Repeat("é", 33))}),
			`change #3 has unknown state "éééééééééééééééééééééééééééééééé", cut to its first 32 runes`},
		{"a dependency numbered zero", records(t0, Change{Number: 4, State: StateOpen, DependsOn: []int{0}}),
			"change #4 depends on number 0"},
		{"a dependency numbered below zero", records(t0, Change{Number: 4, State: StateMerged, DependsOn: []int{-5}}),
			"change #4 depends on number -5"},
		{"a closed dependency beside a dependency numbered zero is refused whole, not at once",
			records(t0, Change{Number: 1, State: StateClosed}, Change{Number: 2, State: StateOpen, DependsOn: []int{1, 0}, WaitingSince: t0}),
			"change #2 depends on number 0"},
		{"eleven distinct dependencies", records(t0, Change{Number: 4, State: StateOpen, DependsOn: repeated(11, 1)}),
			"change #4 depends on more than 10 distinct numbers"},
		{"eleven distinct dependencies, below zero, are named for their count alone",
			records(t0, Change{Number: 4, State: StateOpen, DependsOn: []int{-1, -2, -3, -4, -5, -6, -7, -8, -9, -10, -11}}),
			"change #4 depends on more than 10 distinct numbers"},
		{"one change more than the most", records(t0, merged(1, maxChanges+1)...),
			"the records hold 501 changes, above the most of 500"},
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
	if maxChanges != 500 {
		t.Fatalf("maxChanges is %d, want 500", maxChanges)
	}
	if _, err := pass(records(t0, merged(1, maxChanges)...)); err != nil {
		t.Fatalf("exactly %d changes were refused: %v", maxChanges, err)
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
		"the records carry no time to judge them by; " +
		"the records state no positive wait limit (0s)"
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
		got, err := pass(Records{Changes: ch})
		if err == nil {
			t.Fatalf("read %d: pass accepted the records and gave %+v", i, got)
		}
		if err.Error() != want {
			t.Fatalf("read %d: error\n got  %q\n want %q", i, err, want)
		}
	}
}

// belowZero is change c depending on -1 to -n, which is n defects.
func belowZero(c, n int) Change {
	ch := Change{Number: c, State: StateOpen}
	for i := 1; i <= n; i++ {
		ch.DependsOn = append(ch.DependsOn, -i)
	}
	return ch
}

// tenBelowZero is the ten defects of belowZero(c, 10), in the order the
// error sorts them, the first k of them.
func tenBelowZero(c string, k int) []string {
	var out []string
	for _, n := range []string{"-1", "-10", "-2", "-3", "-4", "-5", "-6", "-7", "-8", "-9"}[:k] {
		out = append(out, "change #"+c+" depends on number "+n)
	}
	return out
}

// TestPassRefusalIsCapped: the one error names the defects found, sorted and
// once each, at most maxNamed, then how many more, and an unknown state is
// quoted cut to maxStateRunes, so no snapshot makes the error unbounded.
func TestPassRefusalIsCapped(t *testing.T) {
	t.Parallel()
	if maxNamed != 20 || maxStateRunes != 32 {
		t.Fatalf("maxNamed %d, maxStateRunes %d; want 20 and 32", maxNamed, maxStateRunes)
	}
	const prefix = "coord: the records cannot be judged: "
	join := func(parts ...[]string) string { return strings.Join(slices.Concat(parts...), "; ") }
	twice := belowZero(3, 10)
	twice.DependsOn = append(twice.DependsOn, -1)
	cases := []struct {
		name    string
		changes []Change
		want    string
	}{
		{"20 defects are all named", []Change{belowZero(2, 10), belowZero(3, 10)},
			join(tenBelowZero("2", 10), tenBelowZero("3", 10))},
		{"21 defects name 20, then one more", []Change{belowZero(2, 10), belowZero(3, 10), belowZero(4, 1)},
			join(tenBelowZero("2", 10), tenBelowZero("3", 10), []string{"and 1 more"})},
		{"25 defects, one read twice, and a long state name 20, then five more",
			[]Change{{Number: 1, State: State(strings.Repeat("Z", 100))}, belowZero(2, 10), twice, belowZero(4, 4)},
			join([]string{`change #1 has unknown state "ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", cut to its first 32 runes`},
				tenBelowZero("2", 10), tenBelowZero("3", 9), []string{"and 5 more"})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := pass(records(t0, tc.changes...))
			if err == nil {
				t.Fatalf("pass accepted the records and gave %+v", got)
			}
			if want := prefix + tc.want; err.Error() != want {
				t.Fatalf("error\n got  %q\n want %q", err, want)
			}
		})
	}
}

// ring is the open changes first to last, each depending on the next and the
// last on the first, every one with a recorded start.
func ring(first, last int) []Change {
	var ch []Change
	for n := first; n <= last; n++ {
		next := n + 1
		if n == last {
			next = first
		}
		ch = append(ch, Change{Number: n, State: StateOpen, DependsOn: []int{next}, WaitingSince: at(8, 30)})
	}
	return ch
}

// merged is the merged changes first to last.
func merged(first, last int) []Change {
	var ch []Change
	for n := first; n <= last; n++ {
		ch = append(ch, Change{Number: n, State: StateMerged})
	}
	return ch
}

// repeated is the numbers 1 to n, the whole run read times times.
func repeated(n, times int) []int {
	var out []int
	for range times {
		for i := 1; i <= n; i++ {
			out = append(out, i)
		}
	}
	return out
}

// TestPassReasonNamesAtMostTwenty: a cycle's reason lists at most maxNamed of
// its members, then how many more.
func TestPassReasonNamesAtMostTwenty(t *testing.T) {
	t.Parallel()
	const twenty = "#101, #102, #103, #104, #105, #106, #107, #108, #109, #110, " +
		"#111, #112, #113, #114, #115, #116, #117, #118, #119, #120"
	for _, tc := range []struct {
		members int
		on      string
	}{{20, twenty}, {21, twenty + " and 1 more"}} {
		got, err := pass(records(t0, ring(101, 100+tc.members)...))
		if err != nil {
			t.Fatal(err)
		}
		want := Decision{Change: 101, Action: ActionRefuse, WaitsOn: []int{102}, Reason: atOnce + cycle + tc.on}
		if len(got) != tc.members || !reflect.DeepEqual(got[0], want) {
			t.Fatalf("a cycle of %d:\n got  %+v\n want %+v first of %d", tc.members, got[0], want, tc.members)
		}
	}
}

// TestPassLongChainIntoLargeCycle: at the bound of maxChanges, a chain of 300
// open changes running into a cycle of 200 is judged whole: every member of
// both is refused at once, naming the cycle's first twenty members and how
// many more.
func TestPassLongChainIntoLargeCycle(t *testing.T) {
	t.Parallel()
	var ch []Change
	for n := 1; n <= 300; n++ {
		ch = append(ch, Change{Number: n, State: StateOpen, DependsOn: []int{n + 1}, WaitingSince: at(8, 30)})
	}
	ch = append(ch, ring(301, 500)...)
	if len(ch) != maxChanges {
		t.Fatalf("the snapshot holds %d changes, want %d", len(ch), maxChanges)
	}
	got, err := pass(records(t0, ch...))
	if err != nil {
		t.Fatal(err)
	}
	const reason = atOnce + cycle + "#301, #302, #303, #304, #305, #306, #307, #308, #309, #310, " +
		"#311, #312, #313, #314, #315, #316, #317, #318, #319, #320 and 180 more"
	if len(got) != 500 {
		t.Fatalf("pass gave %d decisions, want 500", len(got))
	}
	for _, want := range []Decision{
		{Change: 1, Action: ActionRefuse, WaitsOn: []int{2}, Reason: reason},
		{Change: 300, Action: ActionRefuse, WaitsOn: []int{301}, Reason: reason},
		{Change: 301, Action: ActionRefuse, WaitsOn: []int{302}, Reason: reason},
		{Change: 500, Action: ActionRefuse, WaitsOn: []int{301}, Reason: reason},
	} {
		if d := got[want.Change-1]; !reflect.DeepEqual(d, want) {
			t.Fatalf("change #%d:\n got  %+v\n want %+v", want.Change, d, want)
		}
	}
	for _, d := range got {
		if d.Action != ActionRefuse || d.Reason != reason {
			t.Fatalf("change #%d: %+v, want a refusal for the cycle", d.Change, d)
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
