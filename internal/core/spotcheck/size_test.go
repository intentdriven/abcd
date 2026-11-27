package spotcheck

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

func closedClean(id string) cycleTally { return cycleTally{id: id, closed: true} }

func TestSizeHalvesAfterThreeClosedCleanCycles(t *testing.T) {
	size, why, counted := sampleSize(10, []cycleTally{closedClean("c3"), closedClean("c2"), closedClean("c1")})
	if size != 5 || why != SizeHalved || !slices.Equal(counted, []string{"c3", "c2", "c1"}) {
		t.Fatalf("sampleSize = %d %s %v; want 5, halved, the three", size, why, counted)
	}
	if size, _, _ := sampleSize(5, []cycleTally{closedClean("c3"), closedClean("c2"), closedClean("c1")}); size != 3 {
		t.Fatalf("half of five is %d; want it rounded up to 3", size)
	}
	if size, _, _ := sampleSize(1, []cycleTally{closedClean("c3"), closedClean("c2"), closedClean("c1")}); size != 1 {
		t.Fatalf("half of one is %d; want never below 1", size)
	}
	if size, why, _ := sampleSize(10, []cycleTally{closedClean("c2"), closedClean("c1")}); size != 10 || why != SizeFull {
		t.Fatalf("two clean cycles give %d %s; want full", size, why)
	}
}

func TestSizeFullAfterAFinding(t *testing.T) {
	cs := []cycleTally{{id: "c4", closed: true, findings: 1}, closedClean("c3"), closedClean("c2"), closedClean("c1")}
	if size, why, counted := sampleSize(10, cs); size != 10 || why != SizeFull || !slices.Equal(counted, []string{"c4", "c3", "c2"}) {
		t.Fatalf("after a finding: %d %s %v; want full over c4, c3, c2", size, why, counted)
	}
}

func TestOpenCycleCountsAsNeither(t *testing.T) {
	cs := []cycleTally{{id: "c5", findings: 3}, closedClean("c3"), {id: "c4"}, closedClean("c2"), closedClean("c1")}
	if size, why, counted := sampleSize(10, cs); size != 5 || why != SizeHalved || !slices.Equal(counted, []string{"c3", "c2", "c1"}) {
		t.Fatalf("open cycles skipped: %d %s %v; want halved over c3, c2, c1", size, why, counted)
	}
	if size, _, counted := sampleSize(10, []cycleTally{{id: "c1"}, {id: "c2"}}); size != 10 || len(counted) != 0 {
		t.Fatalf("only open cycles: %d %v; want full over none", size, counted)
	}
}

// tallyCase is origin's copy holding n cycles of one drawn item each, for
// the tally: judge(i) is cycle i's judgement file, "" for none.
func tallyCase(t *testing.T, n int, judge func(i int, id, stem string) string) (windowState, []string, string) {
	t.Helper()
	r, _ := originWithRemote(t)
	files := map[string]string{}
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var ids []string
	var cycles []Cycle
	stem := itemStem(1, GroupDrainLanded, "k")
	for i := range n {
		id := cycleID(base.Add(time.Duration(i)*24*time.Hour), fmt.Sprintf("%040x", i+1))
		ids = append(ids, id)
		c := Cycle{SchemaVersion: SchemaVersion, ID: id, Groups: []GroupDraw{{Group: GroupDrainLanded, Drawn: []string{"k"}},
			{Group: GroupAutomaticMerges}, {Group: GroupDrainHandBacks}, {Group: GroupFacilitatorDecisions}}}
		cycles = append(cycles, c)
		it := Item{SchemaVersion: SchemaVersion, Cycle: id, Number: 1, Group: GroupDrainLanded, Key: "k", Roles: []string{RoleFacilitator}}
		files[ItemsDir+"/"+id+"/"+stem+".json"] = encodeOrFail(t, it)
		if j := judge(i, id, stem); j != "" {
			files[JudgementsDir+"/"+id+"/"+stem+"."+RoleFacilitator+".json"] = j
		}
	}
	tracking := setTracking(t, r, files)
	listed, err := listFiles(t.Context(), r.Root(), tracking)
	if err != nil {
		t.Fatal(err)
	}
	return windowState{tracking: tracking, files: listed, cycles: cycles, open: true}, ids, r.Root()
}

// judgementOf is a judgement file as the review lane writes one.
func judgementOf(t *testing.T, id, stem, verdict string) string {
	return encodeOrFail(t, Judgement{SchemaVersion: SchemaVersion, Cycle: id, Item: stem, Role: RoleFacilitator, Verdict: verdict})
}

// TestSizeCountsClosedCyclesInWindowOrder: the tally reads the verified
// cycles newest first by window, and stops once three closed ones are found.
func TestSizeCountsClosedCyclesInWindowOrder(t *testing.T) {
	st, ids, root := tallyCase(t, 5, func(i int, id, stem string) string {
		// The newest (4) is open; 3 found a fault; 2, 1, 0 are clean.
		switch {
		case i == 4:
			return ""
		case i == 3:
			return judgementOf(t, id, stem, VerdictWrong)
		}
		return judgementOf(t, id, stem, VerdictSound)
	})
	tallies, err := tallyCycles(t.Context(), root, st)
	if err != nil {
		t.Fatal(err)
	}
	want := []cycleTally{{id: ids[4]}, {id: ids[3], closed: true, findings: 1}, closedClean(ids[2]), closedClean(ids[1])}
	if !slices.Equal(tallies, want) {
		t.Fatalf("tallies = %+v; want %+v", tallies, want)
	}
	if size, why, _ := sampleSize(10, tallies); size != 10 || why != SizeFull {
		t.Fatalf("size = %d %s; want full, after the finding in %s", size, why, ids[3])
	}
}

// TestAJudgementThatDoesNotReadLeavesTheCycleOpen: a file at a judgement's
// path that does not read as that judgement (empty, a verdict alone, another
// item's, a verdict outside the set) is no judgement, so three such cycles
// never halve the sample.
func TestAJudgementThatDoesNotReadLeavesTheCycleOpen(t *testing.T) {
	for name, body := range map[string]func(id, stem string) string{
		"empty object": func(string, string) string { return "{}\n" },
		"verdict only": func(string, string) string { return `{"verdict":"sound"}` + "\n" },
		"another item": func(id, _ string) string { return judgementOf(t, id, itemStem(2, GroupDrainLanded, "k"), VerdictSound) },
		"odd verdict":  func(id, stem string) string { return judgementOf(t, id, stem, "fine") },
		"repeated key": func(id, stem string) string {
			return strings.Replace(judgementOf(t, id, stem, VerdictWrong), `"verdict": "wrong"`, `"verdict": "wrong", "verdict": "sound"`, 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			st, _, root := tallyCase(t, 3, func(_ int, id, stem string) string { return body(id, stem) })
			tallies, err := tallyCycles(t.Context(), root, st)
			if err != nil {
				t.Fatal(err)
			}
			if size, why, _ := sampleSize(10, tallies); size != 10 || why != SizeFull {
				t.Fatalf("three cycles of judgements that do not read halved the sample: %d %s (%+v)", size, why, tallies)
			}
		})
	}
}
