package spotcheck

import (
	"fmt"
	"slices"
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

// TestSizeCountsClosedCyclesInWindowOrder: the tally reads the cycles origin's
// copy holds newest first by window, whatever order the files are listed in,
// and stops once three closed ones are found.
func TestSizeCountsClosedCyclesInWindowOrder(t *testing.T) {
	r, _ := originWithRemote(t)
	files := map[string]string{}
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var ids []string
	for i := range 5 {
		id := cycleID(base.Add(time.Duration(i)*24*time.Hour), fmt.Sprintf("%040x", i+1))
		ids = append(ids, id)
		files[CyclesDir+"/"+id+".json"] = encodeOrFail(t, Cycle{SchemaVersion: SchemaVersion, ID: id, ToCommit: fmt.Sprintf("%040x", i+1), ToTime: base})
		it := Item{SchemaVersion: SchemaVersion, Cycle: id, Number: 1, Group: GroupDrainLanded, Key: "k", Roles: []string{RoleFacilitator}}
		files[ItemsDir+"/"+id+"/"+itemStem(1, GroupDrainLanded, "k")+".json"] = encodeOrFail(t, it)
		// The newest (4) is open; 3 found a fault; 2, 1, 0 are clean.
		if i < 4 {
			verdict := "sound"
			if i == 3 {
				verdict = "wrong"
			}
			files[JudgementsDir+"/"+id+"/"+itemStem(1, GroupDrainLanded, "k")+"."+RoleFacilitator+".json"] = `{"verdict":"` + verdict + `"}` + "\n"
		}
	}
	setTracking(t, r, files)
	st, err := readWindow(t.Context(), r.Root())
	if err != nil {
		t.Fatal(err)
	}
	tallies, err := tallyCycles(t.Context(), r.Root(), st)
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
