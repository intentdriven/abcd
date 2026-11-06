package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// A drive holds a run to the profile it started on (drive_profile.go).

// laneLeftOn runs a drive on profile until its implementer is in its call and kills
// it, as a second signal does, leaving the lane in progress and reporting its run.
func laneLeftOn(t *testing.T, w *terminalWorld, profile string) TerminalResult {
	t.Helper()
	w.realWorktree()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	o := w.options(t)
	o.Options.Profile = profile
	o.Runners.Timeout = 40 * time.Second
	done := make(chan TerminalResult, 1)
	go func() {
		res, _ := RunTerminal(ctx, w.repo.Root(), o)
		done <- res
	}()
	waitPid(t, w.env, "claude.pid")
	cancel()
	res := <-done
	if res.Run == "" {
		t.Fatalf("the kill names the lane: %+v", res)
	}
	return res
}

// TestARunRecordsTheProfileItsDriveStartedItOn: the state and the record both hold the
// name, and the listing says it.
func TestARunRecordsTheProfileItsDriveStartedItOn(t *testing.T) {
	w := newTerminalWorld(t, "shape", "late")
	left := laneLeftOn(t, w, "alpha")
	st, err := ReadState(w.repo.Root(), left.Run)
	if err != nil {
		t.Fatal(err)
	}
	if st.Profile != "alpha" {
		t.Fatalf("the state records %q", st.Profile)
	}
	if rec := recordOf(st); rec.Profile != "alpha" {
		t.Fatalf("the record names %q", rec.Profile)
	}
	l, err := ListRuns(w.repo.Root(), left.Run, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Runs) != 1 || l.Runs[0].Profile != "alpha" {
		t.Fatalf("the listing says %+v", l.Runs)
	}
}

// TestADriveOnAnotherProfileStopsBeforeItTakesARunUp: no agent starts, nothing claims
// the run, the state keeps the name, and the words name both and what to type.
func TestADriveOnAnotherProfileStopsBeforeItTakesARunUp(t *testing.T) {
	w := newTerminalWorld(t, "shape", "late")
	left := laneLeftOn(t, w, "alpha")
	before := w.launches(t)
	beforeState := stateBytes(t, w.repo.Root(), left.Run)
	o := w.options(t)
	o.Options.Profile = "beta"
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalWall {
		t.Fatalf("the run ends on the check: %+v", res)
	}
	for _, want := range []string{left.Run, "alpha", "beta", "--profile alpha"} {
		if !strings.Contains(res.Why, want) {
			t.Errorf("the reason %q does not name %q", res.Why, want)
		}
	}
	if w.launches(t) != before {
		t.Fatalf("no agent starts on the other profile: %d launches, was %d", w.launches(t), before)
	}
	var was, now State
	if err := json.Unmarshal(beforeState, &was); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(stateBytes(t, w.repo.Root(), left.Run), &now); err != nil {
		t.Fatal(err)
	}
	if now.Profile != "alpha" {
		t.Fatalf("the run keeps its profile: %q", now.Profile)
	}
	for _, e := range now.Record[len(was.Record):] {
		if e.Stage == StageDriver || e.Stage == StageReservation {
			t.Fatalf("nothing claims or reserves the run: %+v", e)
		}
	}
}

// TestADriveOnTheSameProfileTakesTheRunUp: the check stands in nobody's way.
func TestADriveOnTheSameProfileTakesTheRunUp(t *testing.T) {
	w := newTerminalWorld(t, "shape", "late")
	left := laneLeftOn(t, w, "alpha")
	before := w.launches(t)
	o := w.options(t)
	o.Options.Profile = "alpha"
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended == TerminalWall {
		t.Fatalf("the drive goes on: %+v", res)
	}
	if w.launches(t) == before {
		t.Fatalf("the lane's agents start again: %+v", res)
	}
	st, err := ReadState(w.repo.Root(), left.Run)
	if err != nil {
		t.Fatal(err)
	}
	if st.Profile != "alpha" {
		t.Fatalf("the run keeps its profile: %q", st.Profile)
	}
}

// TestARunThatNamesNoProfileGoesOnAndIsMarked: one started before runs named a
// profile, or by a session, was on no named profile; the drive says what it runs it on
// from here and keeps it, so a later drive on the other is held to it.
func TestARunThatNamesNoProfileGoesOnAndIsMarked(t *testing.T) {
	w := newTerminalWorld(t, "shape", "late")
	left := laneLeftOn(t, w, "")
	st, err := ReadState(w.repo.Root(), left.Run)
	if err != nil {
		t.Fatal(err)
	}
	if st.Profile != "" {
		t.Fatalf("no profile, none recorded: %q", st.Profile)
	}
	o := w.options(t)
	o.Options.Profile = "beta"
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatal(err)
	}
	if st, err = ReadState(w.repo.Root(), left.Run); err != nil {
		t.Fatal(err)
	}
	if st.Profile != "beta" {
		t.Fatalf("the run takes the profile that drove it: %q", st.Profile)
	}
	if !slices.ContainsFunc(st.Record, func(e Entry) bool { return e.Stage == StageProfile && strings.Contains(e.Note, "beta") }) {
		t.Fatalf("the record says so: %+v", st.Record)
	}
	if !slices.ContainsFunc(w.lines, func(l string) bool { return strings.Contains(l, left.Run) && strings.Contains(l, "beta") }) {
		t.Fatalf("the drive says so:\n%s", strings.Join(w.lines, "\n"))
	}
}

// TestLanesOnAnotherProfileAreTheLaneInProgress: a lane a drive would take up is named,
// on any profile but the one it started on; a lane at its pull request needs no agent,
// and a drain that ended takes nothing up.
func TestLanesOnAnotherProfileAreTheLaneInProgress(t *testing.T) {
	w := newTerminalWorld(t, "shape", "late")
	left := laneLeftOn(t, w, "alpha")
	root := w.repo.Root()
	got, err := LanesOnAnotherProfile(root, "beta")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].RunID != left.Run || got[0].Issue != left.Issue || got[0].Profile != "alpha" {
		t.Fatalf("beta finds %+v", got)
	}
	if got, err := LanesOnAnotherProfile(root, "alpha"); err != nil || len(got) != 0 {
		t.Fatalf("alpha finds %+v, %v", got, err)
	}
	st, _, err := readDrain(root)
	if err != nil {
		t.Fatal(err)
	}
	parked := st
	parked.Lanes = slices.Clone(st.Lanes)
	for i := range parked.Lanes {
		parked.Lanes[i].Outcome = DrainLanePullRequest
	}
	if err := writeDrain(root, parked); err != nil {
		t.Fatal(err)
	}
	if got, err := LanesOnAnotherProfile(root, "beta"); err != nil || len(got) != 0 {
		t.Fatalf("a lane at its pull request: %+v, %v", got, err)
	}
	ended := st
	at := w.clock.t
	ended.Stopped, ended.EndedAt = DrainStoppedOutage, &at
	if err := writeDrain(root, ended); err != nil {
		t.Fatal(err)
	}
	if got, err := LanesOnAnotherProfile(root, "beta"); err != nil || len(got) != 0 {
		t.Fatalf("a drain that ended: %+v, %v", got, err)
	}
}

// TestALaterVersionAloneCarriesAProfile: the field joined the state at version 22, so a
// file of an earlier version carrying one is no file that version wrote, one with none
// reads as a run no drive named, and the next write carries the version.
func TestALaterVersionAloneCarriesAProfile(t *testing.T) {
	repo, runID, _, _ := issueAwaiting(t)
	path := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(runID)))
	good := stateBytes(t, repo.Root(), runID)
	cur := fmt.Sprintf(`"schema_version": %d,`, SchemaVersion)
	const old = `"schema_version": 21,`
	write := func(s string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(strings.Replace(string(good), cur, old+` "profile": "alpha",`, 1))
	if _, err := ReadState(repo.Root(), runID); err == nil || !strings.Contains(err.Error(), "profile") {
		t.Fatalf("version 21 never wrote a profile: %v", err)
	}
	write(strings.Replace(string(good), cur, old, 1))
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatalf("version 21 with none reads: %v", err)
	}
	if st.SchemaVersion != SchemaVersion || st.Profile != "" {
		t.Fatalf("read as %d, profile %q", st.SchemaVersion, st.Profile)
	}
	write(strings.Replace(string(good), cur, cur+` "profile": "alpha",`, 1))
	if st, err = ReadState(repo.Root(), runID); err != nil || st.Profile != "alpha" {
		t.Fatalf("the current version carries it: %q %v", st.Profile, err)
	}
}
