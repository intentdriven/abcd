package loop

import (
	"os"
	"testing"
	"time"
)

// TestRunsListsTwoRunsMintedInOneSecondInStartOrder: a run id is the second it
// was minted in plus four random digits, so two runs minted in one second sort
// by the digits, not by which started first. Runs lists them oldest first by
// when each was created, the id breaking only a tie.
func TestRunsListsTwoRunsMintedInOneSecondInStartOrder(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	now := time.Date(2026, 10, 26, 14, 13, 40, 0, time.UTC)
	start, err := Start(repo.Root(), "itd-10", Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	older, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	// The later run's id sorts first: same second, smaller digits.
	newer := older
	newer.RunID = "run-2610261413400000"
	if newer.RunID >= older.RunID {
		older.RunID = "run-2610261413409999"
		if err := os.Rename(repo.Root()+"/"+runRel(start.RunID), repo.Root()+"/"+runRel(older.RunID)); err != nil {
			t.Fatal(err)
		}
	}
	newer.CreatedAt = now.Add(time.Minute)
	newer.UpdatedAt = newer.CreatedAt
	root, err := os.OpenRoot(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := root.Mkdir(runRel(newer.RunID), dirPerm); err != nil {
		t.Fatal(err)
	}
	for _, st := range []State{older, newer} {
		if err := writeState(root, st); err != nil {
			t.Fatal(err)
		}
	}
	runs, err := Runs(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 2 || runs[0].RunID != older.RunID || runs[1].RunID != newer.RunID {
		var got []string
		for _, r := range runs {
			got = append(got, r.RunID)
		}
		t.Fatalf("Runs lists %v, want the run created first (%s) before the later one (%s)", got, older.RunID, newer.RunID)
	}
}
