package loop

import (
	"strings"
	"testing"
)

// The build loop's trial mode (itd-2610161733316518 decision 4,
// spc-2610161949559050 step 3): a model-matrix copy runs the loop's stages
// taking no reservation and landing nothing, so copies of one intent never
// collide on its footprint and none of them lands.

// TestATrialRunTakesNoReservation: a run started in trial mode on a checkout
// with an origin pushes no reservation, records none in its state, and says
// in its record that it is a trial; resumed, it reserves nothing either.
func TestATrialRunTakesNoReservation(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), footprintedSpec)
	bare := onOrigin(t, repo)

	res, err := Start(repo.Root(), "itd-10", Options{Trial: true})
	if err != nil {
		t.Fatal(err)
	}
	if tip := originReservation(t, bare, reservedRef); tip.commit != "" {
		t.Fatalf("a trial run pushes no reservation: %+v", tip)
	}
	st, err := ReadState(repo.Root(), res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Trial || st.ReservedAt != nil {
		t.Fatalf("the state is a trial's and records no reservation: trial %v, reserved %v", st.Trial, st.ReservedAt)
	}
	if res.Reservation != "" {
		t.Fatalf("the start names no reservation: %q", res.Reservation)
	}
	if note := recordNote(t, repo, res.RunID, StageTrial); !strings.Contains(note, "takes no reservation and lands nothing") {
		t.Fatalf("the record says the run is a trial: %q", note)
	}

	again, err := Start(repo.Root(), "itd-10", Options{Trial: true})
	if err != nil || !again.Resumed || again.RunID != res.RunID {
		t.Fatalf("starting again resumes the trial: %+v %v", again, err)
	}
	if tip := originReservation(t, bare, reservedRef); tip.commit != "" {
		t.Fatalf("a resumed trial reserves nothing either: %+v", tip)
	}
}

// TestATrialRunIsNeverResumedAsABuildNorABuildAsATrial: a trial and a build of
// one intent are different runs, so neither start resumes the other's.
func TestATrialRunIsNeverResumedAsABuildNorABuildAsATrial(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), footprintedSpec)
	trial, err := Start(repo.Root(), "itd-10", Options{Trial: true})
	if err != nil {
		t.Fatal(err)
	}
	build, err := Start(repo.Root(), "itd-10", Options{})
	if err == nil && build.Resumed && build.RunID == trial.RunID {
		t.Fatalf("a build resumed the trial %s", trial.RunID)
	}
}

// TestATrialLaneLandsNothing: the landing stage of a trial lane pushes,
// merges and records nothing to land, and completes with a line saying so,
// whatever the lane holds.
func TestATrialLaneLandsNothing(t *testing.T) {
	lane := &Lane{ID: "lane-1", Branch: "main", Stage: StageLand}
	out, err := landStage(Context{State: State{Trial: true}}, lane)
	if err != nil {
		t.Fatal(err)
	}
	if out.Await != nil || out.HandBack != nil || out.Stay || out.Goto != "" {
		t.Fatalf("a trial's landing completes at once: %+v", out)
	}
	if !strings.Contains(out.Note, "lands nothing") {
		t.Fatalf("the landing says it landed nothing: %q", out.Note)
	}
	if lane.Landing != nil {
		t.Fatalf("a trial's lane starts no landing: %+v", lane.Landing)
	}
}
