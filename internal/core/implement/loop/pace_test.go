package loop

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The pacing intent's tests (itd-2609201925079472, spc-2609202134341288):
// the layered pace resolver (criteria 1 to 3 and 9) and the window clock
// (criteria 4 and 5).

func strp(s string) *string { return &s }

// machineConfig writes ~/.abcd/config.json under the test's HOME with the
// modes the machine layer's guard accepts.
func machineConfig(t *testing.T, body string) {
	t.Helper()
	dir := filepath.Join(os.Getenv("HOME"), ".abcd")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// repoConfig commits .abcd/config.json in the repository.
func repoConfig(t *testing.T, repo *gittest.Repo, body string) {
	t.Helper()
	repo.Write(".abcd/config.json", body)
	repo.Commit("config")
}

func paceRecord(t *testing.T, st State) string {
	t.Helper()
	for _, e := range st.Record {
		if e.Stage == "pace" {
			return e.Note
		}
	}
	t.Fatalf("the run record names no pace: %+v", st.Record)
	return ""
}

func wantPace(t *testing.T, p *Pace, work, pause, subs int, layer string) {
	t.Helper()
	if p == nil {
		t.Fatal("the run carries no pace")
	}
	got := []PaceValue{p.WorkMinutes, p.PauseMinutes, p.SubAgents}
	for i, want := range []int{work, pause, subs} {
		if got[i].Value != want || got[i].Layer != layer {
			t.Fatalf("pace = %+v, want %d/%d with %d sub-agents from the %s layer", *p, work, pause, subs, layer)
		}
	}
}

// TestARunWithNoConfigurationRunsOnTheBundledPace is criterion 1: with no flag
// and no configuration the run is paced 120/300 with two lanes, the run record
// names the bundled layer, and the run's first window opens at its start.
func TestARunWithNoConfigurationRunsOnTheBundledPace(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	now := time.Date(2026, 9, 29, 9, 0, 0, 0, time.UTC)
	res, err := Start(repo.Root(), "itd-10", Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	wantPace(t, res.Pace, 120, 300, 2, "bundled")
	st, err := ReadState(repo.Root(), res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	wantPace(t, st.Pace, 120, 300, 2, "bundled")
	if note := paceRecord(t, st); !strings.Contains(note, "120/300") || !strings.Contains(note, "2 sub-agents") || !strings.Contains(note, "bundled") {
		t.Fatalf("the record names the pace and the bundled layer: %q", note)
	}
	if st.WindowStartedAt == nil || !st.WindowStartedAt.Equal(now) || st.NextEligibleAt != nil {
		t.Fatalf("the first window opens at the start: %v %v", st.WindowStartedAt, st.NextEligibleAt)
	}
	if st.SchemaVersion != SchemaVersion {
		t.Fatalf("a new run is written at the current schema version: %d", st.SchemaVersion)
	}
}

// TestTheRepositoryPaceWinsOverTheMachines is criterion 2: a repository
// configuration and a machine configuration that disagree, and no flag, run on
// the repository's values, and the record names the repository's file.
func TestTheRepositoryPaceWinsOverTheMachines(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	machineConfig(t, `{"pace": {"work_minutes": 60, "pause_minutes": 120, "sub_agents": 1}}`)
	repoConfig(t, repo, `{"pace": {"work_minutes": 100, "pause_minutes": 200, "sub_agents": 4}}`)
	res, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	st, _ := ReadState(repo.Root(), res.RunID)
	wantPace(t, st.Pace, 100, 200, 4, "repo")
	if st.Pace.WorkMinutes.Origin != ".abcd/config.json" {
		t.Fatalf("the origin names the repository's file: %+v", st.Pace)
	}
	if note := paceRecord(t, st); !strings.Contains(note, "100/200") || !strings.Contains(note, ".abcd/config.json") || strings.Contains(note, "~/.abcd") {
		t.Fatalf("the record names the repository's file and not the machine's: %q", note)
	}

	// A key the repository leaves out falls to the machine's, and the record
	// names each value's layer.
	repoConfig(t, repo, `{"pace": {"work_minutes": 100}}`)
	os.RemoveAll(filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir)))
	res, err = Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	st, _ = ReadState(repo.Root(), res.RunID)
	if st.Pace.WorkMinutes.Layer != "repo" || st.Pace.PauseMinutes.Layer != "machine" || st.Pace.PauseMinutes.Value != 120 ||
		st.Pace.SubAgents.Origin != "~/.abcd/config.json" {
		t.Fatalf("each key resolves through its own layers: %+v", st.Pace)
	}
	if note := paceRecord(t, st); !strings.Contains(note, ".abcd/config.json") || !strings.Contains(note, "~/.abcd/config.json") {
		t.Fatalf("the record names both layers that applied: %q", note)
	}
}

// TestThePaceFlagsWinOverEveryLayer is criterion 3: --pace 90/240
// --sub-agents 3 apply over every configured layer, and the record names the
// flags as typed.
func TestThePaceFlagsWinOverEveryLayer(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	machineConfig(t, `{"pace": {"work_minutes": 60, "pause_minutes": 120, "sub_agents": 1}}`)
	repoConfig(t, repo, `{"pace": {"work_minutes": 100, "pause_minutes": 200, "sub_agents": 4}}`)
	res, err := Start(repo.Root(), "itd-10", Options{Pace: strp("90/240"), SubAgents: strp("3")})
	if err != nil {
		t.Fatal(err)
	}
	st, _ := ReadState(repo.Root(), res.RunID)
	wantPace(t, st.Pace, 90, 240, 3, "flag")
	if st.Pace.WorkMinutes.Origin != "--pace 90/240" || st.Pace.SubAgents.Origin != "--sub-agents 3" {
		t.Fatalf("the origins name the flags as typed: %+v", st.Pace)
	}
	if note := paceRecord(t, st); !strings.Contains(note, "90/240") || !strings.Contains(note, "--pace 90/240") || !strings.Contains(note, "--sub-agents 3") {
		t.Fatalf("the record names the flags: %q", note)
	}
}

// TestAMalformedPaceIsRefusedAndWritesNoState is criterion 9: a malformed pace
// or ceiling, typed as a flag or written in a configuration layer, is refused
// naming the value and the accepted form, and no state is written.
func TestAMalformedPaceIsRefusedAndWritesNoState(t *testing.T) {
	cases := []struct {
		name       string
		pace, subs *string
		repo       string
		machine    string
		value      string
	}{
		{name: "one number", pace: strp("90"), value: `"90"`},
		{name: "no pause", pace: strp("90/"), value: `"90/"`},
		{name: "words", pace: strp("ninety/240"), value: `"ninety/240"`},
		{name: "three parts", pace: strp("90/240/3"), value: `"90/240/3"`},
		{name: "a sign", pace: strp("+90/240"), value: `"+90/240"`},
		{name: "no work", pace: strp("0/240"), value: "0"},
		{name: "past a week", pace: strp("90/99999"), value: "99999"},
		{name: "an overflow", pace: strp("99999999999999999999/1"), value: "99999999999999999999/1"},
		{name: "no lane", subs: strp("0"), value: "0"},
		{name: "a word ceiling", subs: strp("two"), value: `"two"`},
		{name: "a fractional ceiling", subs: strp("2.5"), value: `"2.5"`},
		{name: "too many lanes", subs: strp("65"), value: "65"},
		{name: "a repo string", repo: `{"pace": {"work_minutes": "two hours"}}`, value: `"two hours"`},
		{name: "a repo fraction", repo: `{"pace": {"pause_minutes": 1.5}}`, value: "1.5"},
		{name: "a repo misspelling", repo: `{"pace": {"work_mins": 90}}`, value: "work_mins"},
		{name: "a machine negative", machine: `{"pace": {"sub_agents": -1}}`, value: "-1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
			if tc.repo != "" {
				repoConfig(t, repo, tc.repo)
			}
			if tc.machine != "" {
				machineConfig(t, tc.machine)
			}
			_, err := Start(repo.Root(), "itd-10", Options{Pace: tc.pace, SubAgents: tc.subs})
			r := mustRefusal(t, err)
			if r.Stage != "pace" || r.Contention {
				t.Fatalf("want the pace step's refusal: %+v", r)
			}
			if !strings.Contains(r.Reason, strings.Trim(tc.value, `"`)) {
				t.Fatalf("the reason names the value %s: %q", tc.value, r.Reason)
			}
			for _, form := range []string{"<work-minutes>/<pause-minutes>", "sub-agents"} {
				if !strings.Contains(r.Remedy, form) && !strings.Contains(r.Reason, form) {
					t.Fatalf("the refusal names the accepted form %q: %+v", form, r)
				}
			}
			runTierAbsent(t, repo.Root())
		})
	}
}

// TestAResumeKeepsTheRunsPace: a build started again for a run in progress
// reports the run's own pace; a flag naming another pace is refused rather
// than silently ignored, and one naming the same pace is not.
func TestAResumeKeepsTheRunsPace(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	first, err := Start(repo.Root(), "itd-10", Options{Pace: strp("90/240")})
	if err != nil {
		t.Fatal(err)
	}
	before := stateBytes(t, repo.Root(), first.RunID)
	again, err := Start(repo.Root(), "itd-10", Options{Pace: strp("90/240")})
	if err != nil || !again.Resumed {
		t.Fatalf("the same pace resumes: %+v %v", again, err)
	}
	if p := again.Pace; p == nil || p.WorkMinutes.Value != 90 || p.PauseMinutes.Value != 240 || p.WorkMinutes.Layer != "flag" ||
		p.SubAgents.Value != 2 || p.SubAgents.Layer != "bundled" {
		t.Fatalf("a resume reports the run's own pace: %+v", again.Pace)
	}
	_, err = Start(repo.Root(), "itd-10", Options{Pace: strp("60/240")})
	r := mustRefusal(t, err)
	if r.Stage != "pace" || !strings.Contains(r.Reason, "90/240") || !strings.Contains(r.Reason, "60/240") {
		t.Fatalf("a resume with another pace names both: %+v", r)
	}
	_, err = Start(repo.Root(), "itd-10", Options{SubAgents: strp("0")})
	if r := mustRefusal(t, err); r.Stage != "pace" {
		t.Fatalf("a malformed flag is refused on a resume too: %+v", r)
	}
	if !bytes.Equal(before, stateBytes(t, repo.Root(), first.RunID)) {
		t.Fatal("a resume changes nothing")
	}
}

// TestAnElapsedWindowStartsNothingAndWritesNextEligibleAt is criteria 4 and 5:
// once the window has elapsed, a step starts nothing, writes next_eligible_at
// and exits without error naming the time; the running lane's agent still
// hands its receipt back, so the lane finishes its step and checkpoints to its
// branch; before next_eligible_at a step is refused naming the time and the
// state is unchanged; at it, a new window opens and the loop moves again.
func TestAnElapsedWindowStartsNothingAndWritesNextEligibleAt(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps("1. The parser\n2. The loop\n"))
	t0 := time.Date(2026, 9, 29, 9, 0, 0, 0, time.UTC)
	at := func(d time.Duration) Options { return Options{Now: func() time.Time { return t0.Add(d) }} }
	start, err := Start(repo.Root(), "itd-10", Options{Now: at(0).Now, Pace: strp("60/30")})
	if err != nil {
		t.Fatal(err)
	}
	id := start.RunID
	f := &fakeSteps{calls: map[Stage]int{}}
	steps := f.steps()
	for i, want := range []Stage{StageWorktree, StageBrief} {
		res, err := advance(repo.Root(), id, steps, at(time.Duration(i+1)*time.Minute))
		if err != nil || res.PerformedStage != want {
			t.Fatalf("inside the window the loop moves: %+v %v", res, err)
		}
	}
	res, err := advance(repo.Root(), id, steps, at(5*time.Minute))
	if err != nil || res.Awaiting == nil {
		t.Fatalf("the implementer is started inside the window: %+v %v", res, err)
	}
	receipt := res.Awaiting.Receipt

	// The window elapses while the implementer works.
	closed := t0.Add(61 * time.Minute)
	res, err = advance(repo.Root(), id, steps, at(61*time.Minute))
	if err != nil {
		t.Fatalf("closing the window is not a failure: %v", err)
	}
	wantNext := closed.Add(30 * time.Minute)
	if res.PerformedStage != "" || res.NextEligibleAt == nil || !res.NextEligibleAt.Equal(wantNext) ||
		!strings.Contains(res.Next, wantNext.Format(time.RFC3339)) {
		t.Fatalf("an elapsed window performs nothing and names next_eligible_at %s: %+v", wantNext, res)
	}
	st, _ := ReadState(repo.Root(), id)
	if st.NextEligibleAt == nil || !st.NextEligibleAt.Equal(wantNext) {
		t.Fatalf("next_eligible_at is written into the state: %v", st.NextEligibleAt)
	}
	if last := st.Record[len(st.Record)-1]; last.Stage != "pause" || !strings.Contains(last.Note, wantNext.Format(time.RFC3339)) {
		t.Fatalf("the record names the pause: %+v", last)
	}

	// The running lane finishes its step: its receipt is taken inside the pause.
	if err := os.WriteFile(receipt, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Receipt(repo.Root(), id, receipt, steps, at(70*time.Minute))
	if err != nil || got.PerformedStage != StageImplement {
		t.Fatalf("the running lane checkpoints inside the pause: %+v %v", got, err)
	}

	// Before next_eligible_at nothing moves and the state is unchanged.
	before := stateBytes(t, repo.Root(), id)
	_, err = advance(repo.Root(), id, steps, at(80*time.Minute))
	r := mustRefusal(t, err)
	if r.Stage != "pause" || !r.Contention || !strings.Contains(r.Reason, wantNext.Format(time.RFC3339)) {
		t.Fatalf("a step inside the pause is refused naming the time: %+v", r)
	}
	if !bytes.Equal(before, stateBytes(t, repo.Root(), id)) || f.calls[StageValidate] != 0 {
		t.Fatal("a refused step inside the pause changes no state and performs nothing")
	}

	// At next_eligible_at a new window opens and the loop moves again.
	res, err = advance(repo.Root(), id, steps, at(91*time.Minute))
	if err != nil || res.PerformedStage != StageValidate {
		t.Fatalf("after the pause the loop moves: %+v %v", res, err)
	}
	st, _ = ReadState(repo.Root(), id)
	if st.NextEligibleAt != nil || st.WindowStartedAt == nil || !st.WindowStartedAt.Equal(t0.Add(91*time.Minute)) {
		t.Fatalf("a new window opens when the pause ends: %v %v", st.WindowStartedAt, st.NextEligibleAt)
	}
}

// TestAVersionOneStateIsReadAsAnUnpacedRun: a state file the previous schema
// wrote (version 1, no pace) is read as a run started before pacing, is never
// closed by a window it never had, and is written back at the current version;
// a version-1 file carrying a pace is not one version 1 wrote, and is refused.
func TestAVersionOneStateIsReadAsAnUnpacedRun(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(start.RunID)))
	v1 := `{
  "schema_version": 1,
  "run_id": "` + start.RunID + `",
  "key": "itd-10",
  "intent": "itd-10",
  "spec": "spc-1",
  "driver": "host",
  "created_at": "2026-09-20T09:00:00Z",
  "updated_at": "2026-09-20T09:00:00Z",
  "lanes": [{"id": "lane-1", "key": "itd-10", "spec_step": 1, "step_title": "the whole spec", "step": "worktree"}],
  "pending": [],
  "record": [{"at": "2026-09-20T09:00:00Z", "lane": "lane-1", "step": "start", "note": "checks passed"}]
}
`
	if err := os.WriteFile(path, []byte(v1), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatalf("a version-1 state is read: %v", err)
	}
	if st.Pace != nil || st.WindowStartedAt != nil {
		t.Fatalf("a version-1 run is unpaced: %+v", st)
	}
	f := &fakeSteps{calls: map[Stage]int{}}
	res, err := advance(repo.Root(), start.RunID, f.steps(), Options{})
	if err != nil || res.PerformedStage != StageWorktree {
		t.Fatalf("a version-1 run steps on, days after it started: %+v %v", res, err)
	}
	if !bytes.Contains(stateBytes(t, repo.Root(), start.RunID), []byte(`"schema_version": `+strconv.Itoa(SchemaVersion))) {
		t.Fatal("the next write carries the current version")
	}

	withPace := strings.Replace(v1, `"pending": []`, `"pending": [], "pace": {"work_minutes": {"value": 1, "layer": "flag", "origin": "--pace 1/1"}}`, 1)
	if err := os.WriteFile(path, []byte(withPace), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadState(repo.Root(), start.RunID); err == nil {
		t.Fatal("a version-1 file carrying a pace is refused")
	}
}
