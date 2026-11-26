package loop

import (
	"testing"
	"time"
)

// The front door's read of a forming bundle (itd-2611201450052326,
// spc-2611201455051897 step 4): the summary names the bundle a drain in
// progress is forming, read from the drain's state after the move.

// TestReadDrainFormingNamesTheBundleADrainInProgressForms: a drain in progress
// that is forming a bundle reads back its lead and members; one forming none,
// one that has ended and a checkout with no drain read as nil.
func TestReadDrainFormingNamesTheBundleADrainInProgressForms(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	root := repo.Root()

	if f, err := ReadDrainForming(root); err != nil || f != nil {
		t.Fatalf("a checkout with no drain forms nothing: %+v, %v", f, err)
	}

	writeDrainAt(t, root, bundledDrain(), DrainSchemaVersion)
	f, err := ReadDrainForming(root)
	if err != nil {
		t.Fatalf("a drain forming a bundle reads: %v", err)
	}
	if f == nil || f.Lead != "iss-2610010000000006" || len(f.Members) != 1 || f.Members[0].Issue != "iss-2610010000000007" {
		t.Fatalf("the forming bundle's lead and members are read back: %+v", f)
	}

	none := bundledDrain()
	none.Forming = nil
	writeDrainAt(t, root, none, DrainSchemaVersion)
	if f, err := ReadDrainForming(root); err != nil || f != nil {
		t.Fatalf("a drain forming no bundle reads as nil: %+v, %v", f, err)
	}

	ended := bundledDrain()
	at := time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC)
	ended.Stopped, ended.EndedAt = DrainStoppedEmpty, &at
	writeDrainAt(t, root, ended, DrainSchemaVersion)
	if f, err := ReadDrainForming(root); err != nil || f != nil {
		t.Fatalf("a drain that has ended forms nothing: %+v, %v", f, err)
	}
}
