package loop

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// A finished run keeps its judge's strike reasons (iss-2611182238519860): the
// round's judge keeps each location it struck and its reason, which outlive
// the round's findings once the next round opens.

// struckReason is the reason the strike fixtures give.
const struckReason = "the caller already checks it at caller.go:9"

// struckThenFixed strikes a.go:3 of a round whose failing reviewer cites
// a.go:3 and b.go:5, answers the fix round, and, where next, opens the next
// round by handing out its first reviewer. It returns the run.
func struckThenFixed(t *testing.T, next bool) (*gittest.Repo, string) {
	t.Helper()
	repo, id, stages, l, _ := judgedRound(t, calibratedJudge,
		findingsReturn("SHIP"), findingsReturn("FIX FIRST", "a.go:3", "b.go:5"))
	a := stepAwait(t, repo, id, stages, calibratedJudge)
	if _, err := judgeReturn(t, repo, id, stages, a, l.HeadSHA, "### Struck\n\n- `a.go:3` — "+struckReason+"\n"); err != nil {
		t.Fatal(err)
	}
	stepAwait(t, repo, id, stages, calibratedJudge)
	commit := laneCommit(t, repo, currentLane(t, repo, id), "b.go")
	fixed(t, repo, id, stages, "- b.go:5 applied in "+commit+"\n", commit)
	if next {
		if a := stepAwait(t, repo, id, stages, calibratedJudge); a.Role != RoleRuthless {
			t.Fatalf("premise: the fix opens the next round's reviewers: %+v", a)
		}
	}
	return repo, id
}

// TestAStrikeKeepsItsLocationAndReasonOnceTheNextRoundOpens: once the fix
// round opens the next round and the struck round's findings are dropped, the
// round's judge still keeps the location it struck and its reason, which
// `implement record --json` shows under the lane's judges, and the judge's
// line in the run record says where the reason is kept.
func TestAStrikeKeepsItsLocationAndReasonOnceTheNextRoundOpens(t *testing.T) {
	repo, id := struckThenFixed(t, true)
	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Lanes[0].Validation) < 2 || st.Lanes[0].Validation[0].carriesFindings() {
		t.Fatalf("premise: the next round is open and the struck round's findings are dropped: %+v", st.Lanes[0].Validation)
	}
	j := st.Lanes[0].Validation[0].Judge
	if j == nil || j.Struck != 1 || len(j.Strikes) != 1 || j.Strikes[0] != (JudgeStrike{File: "a.go", Line: 3, Reason: struckReason}) {
		t.Fatalf("the round's judge keeps its strike and reason: %+v", j)
	}
	rec, err := ReadRecord(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(rec.Lanes[0].Judges)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"strikes":[{"file":"a.go","line":3,"reason":"`+struckReason+`"}]`) {
		t.Fatalf("implement record --json shows the strike under the lane's judges:\n%s", raw)
	}
	var note string
	for _, e := range rec.Record {
		if strings.Contains(e.Note, "round's judge") && strings.Contains(e.Note, "a.go:3") {
			note = e.Note
		}
	}
	if !strings.Contains(note, "kept on the round's judge") || strings.Contains(note, "reason on the finding") {
		t.Fatalf("the judge's line says the reason is kept on the round's judge: %q", note)
	}
}

// TestAVersion28FileReadsAsTheCurrentVersion: a version-28 file, which keeps
// no strike on its judge, reads as the current version; where the struck
// round still holds its findings the judge's strikes are taken from them, and
// where they were dropped the judge keeps its count alone.
func TestAVersion28FileReadsAsTheCurrentVersion(t *testing.T) {
	if SchemaVersion != schemaVersionUnreasoned+1 || schemaVersionUnreasoned != schemaVersionUnjudged+1 {
		t.Fatalf("a judge's kept strikes are the version after %d: %d", schemaVersionUnreasoned, SchemaVersion)
	}
	for _, next := range []bool{false, true} {
		repo, id := struckThenFixed(t, next)
		older(t, repo, id, schemaVersionUnreasoned, func(j map[string]any) { delete(j, "strikes") })
		st, err := ReadState(repo.Root(), id)
		if err != nil {
			t.Fatalf("a version-28 file reads (next round open: %v): %v", next, err)
		}
		j := st.Lanes[0].Validation[0].Judge
		if st.SchemaVersion != SchemaVersion || j == nil || j.Struck != 1 {
			t.Fatalf("it reads as the current version, its judge's count kept: %d %+v", st.SchemaVersion, j)
		}
		want := 1
		if next {
			want = 0
		}
		if len(j.Strikes) != want || want == 1 && j.Strikes[0] != (JudgeStrike{File: "a.go", Line: 3, Reason: struckReason}) {
			t.Fatalf("the strikes are taken from the round's findings where it holds them (next round open: %v): %+v", next, j.Strikes)
		}
	}
}

// TestAVersion28FileKeepingStrikesOnItsJudgeIsRefused: a judge's kept strikes
// are version 29's, so a version-28 file carrying them is refused naming them.
func TestAVersion28FileKeepingStrikesOnItsJudgeIsRefused(t *testing.T) {
	repo, id := struckThenFixed(t, true)
	older(t, repo, id, schemaVersionUnreasoned, func(map[string]any) {})
	if _, err := ReadState(repo.Root(), id); err == nil || !strings.Contains(err.Error(), "strikes") || !strings.Contains(err.Error(), "never wrote") {
		t.Fatalf("a version-28 file keeping strikes on its judge is refused naming them: %v", err)
	}
}

// TestAHandEditedStrikeOnTheJudgeIsRefused: a strike on the round's judge is
// one the loop writes: while the round holds its findings, exactly the ones
// it struck with their reasons; and always a plain location and reason, once
// each, as many as it struck.
func TestAHandEditedStrikeOnTheJudgeIsRefused(t *testing.T) {
	for _, next := range []bool{false, true} {
		repo, id := struckThenFixed(t, next)
		path := runStatePath(repo.Root(), id)
		good, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		edits := map[string]func(j map[string]any){
			"a strike past the count": func(j map[string]any) {
				j["strikes"] = append(j["strikes"].([]any), map[string]any{"file": "c.go", "line": 1.0, "reason": "x"})
			},
			"a strike with no reason": func(j map[string]any) { delete(j["strikes"].([]any)[0].(map[string]any), "reason") },
			"a strike at no plain location": func(j map[string]any) {
				j["strikes"].([]any)[0].(map[string]any)["file"] = "../a.go"
			},
			"a reason carrying a control": func(j map[string]any) {
				j["strikes"].([]any)[0].(map[string]any)["reason"] = "false\x1b[2J"
			},
		}
		if !next {
			edits["a strike the round's findings never held"] = func(j map[string]any) {
				j["strikes"].([]any)[0].(map[string]any)["file"] = "other.go"
			}
			edits["a reason other than the finding's"] = func(j map[string]any) {
				j["strikes"].([]any)[0].(map[string]any)["reason"] = "another reason"
			}
			edits["no strike kept where the round's findings carry one"] = func(j map[string]any) { delete(j, "strikes") }
		}
		for name, edit := range edits {
			var doc map[string]any
			if err := json.Unmarshal(good, &doc); err != nil {
				t.Fatal(err)
			}
			j := doc["lanes"].([]any)[0].(map[string]any)["validation"].([]any)[0].(map[string]any)["judge"].(map[string]any)
			if _, ok := j["strikes"].([]any); !ok {
				t.Fatalf("premise: the file keeps the judge's strikes: %v", j)
			}
			edit(j)
			data, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := ReadState(repo.Root(), id); err == nil || !strings.Contains(err.Error(), "strike") {
				t.Fatalf("%s is refused at read (next round open: %v): %v", name, next, err)
			}
		}
		if err := os.WriteFile(path, good, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadState(repo.Root(), id); err != nil {
			t.Fatalf("the file the loop wrote reads: %v", err)
		}
	}
}

// TestTrimmingDropsAJudgesStrikesOnceNoFindingsAreLeft: a state still past its
// room drops every round's findings before any judge's strikes, so a judge is
// never left without its strikes while its round's findings still mark them
// struck, and the file of every step reads; once no findings are left, the
// oldest round's judge's strikes go, its count kept. The struck round is the
// latest, still holding its findings, and an earlier one whose findings the
// next round already dropped.
func TestTrimmingDropsAJudgesStrikesOnceNoFindingsAreLeft(t *testing.T) {
	for _, next := range []bool{false, true} {
		repo, id := struckThenFixed(t, next)
		st, err := ReadState(repo.Root(), id)
		if err != nil {
			t.Fatal(err)
		}
		if len(st.Lanes[0].Validation[0].Judge.Strikes) != 1 {
			t.Fatalf("premise: the judge keeps its strike (next round open: %v): %+v", next, st.Lanes[0].Validation[0].Judge)
		}
		if held := st.Lanes[0].Validation[0].carriesFindings(); held == next {
			t.Fatalf("premise: the struck round holds its findings only while it is the latest (next round open: %v)", next)
		}
		path := runStatePath(repo.Root(), id)
		for step := 1; st.trimOldestFindings(); step++ {
			r := st.Lanes[0].Validation[0]
			if r.carriesFindings() && len(r.Judge.Strikes) == 0 {
				t.Fatalf("trim step %d drops the judge's strikes while its round's findings remain (next round open: %v)", step, next)
			}
			if !st.judgesWritten() {
				t.Fatalf("trim step %d leaves a judge the loop never writes (next round open: %v): %+v", step, next, r.Judge)
			}
			data, err := json.MarshalIndent(st, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := ReadState(repo.Root(), id); err != nil {
				t.Fatalf("the file of trim step %d reads (next round open: %v): %v", step, next, err)
			}
		}
		if j := st.Lanes[0].Validation[0].Judge; j.Struck != 1 || len(j.Strikes) != 0 {
			t.Fatalf("the trim drops the judge's strikes and keeps its count (next round open: %v): %+v", next, j)
		}
	}
}

// older rewrites the run's state file at version, editing the first round's
// judge with edit.
func older(t *testing.T, repo *gittest.Repo, id string, version int, edit func(map[string]any)) {
	t.Helper()
	data := stateAtVersion(t, repo.Root(), id, version)
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	edit(doc["lanes"].([]any)[0].(map[string]any)["validation"].([]any)[0].(map[string]any)["judge"].(map[string]any))
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runStatePath(repo.Root(), id), out, 0o600); err != nil {
		t.Fatal(err)
	}
}
