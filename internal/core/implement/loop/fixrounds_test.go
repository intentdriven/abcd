package loop

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The fix-round rulings of 2026-09-29: DQ1a (an undecided criterion in the
// fidelity audit reopens the work exactly as a not-met one does) and DR1 (the
// fix rounds a lane may take before it is handed back are a per-run value set
// beside --pace, default 3), on itd-50's criteria 1 and 2 and
// itd-2609211116005482's falsified pick.

// TestAnUndecidedCriterionReopensTheWorkLikeANotMet is ruling DQ1a: an audit
// whose criterion reads INCONCLUSIVE fails the round, so the lane goes to a
// fresh implementer briefed on that criterion, and never to its landing.
func TestAnUndecidedCriterionReopensTheWorkLikeANotMet(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, DefaultStages()
	implemented(t, repo, id, stages, "one.txt")
	passRound(t, repo, id, stages, RoleRuthless, RoleSecurity)
	handBack(t, repo, id, stages, RoleAuditor, "INCONCLUSIVE")

	st, _ := ReadState(repo.Root(), id)
	a := st.Lanes[0].Validation[0].Validators[2]
	if a.Verdict != "INCONCLUSIVE" || a.Pass {
		t.Fatalf("an undecided audit does not pass the round: %+v", a)
	}
	res, err := Advance(repo.Root(), id, stages, Options{})
	if err != nil || res.Awaiting == nil || res.Awaiting.Role != RoleImplementer || res.Stage != StageValidate {
		t.Fatalf("an undecided audit hands the lane to a fresh implementer, never to its landing: %+v %v", res, err)
	}
	brief, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(res.Awaiting.Brief)))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"undecided", "ac-1"} {
		if !strings.Contains(string(brief), want) {
			t.Fatalf("the fix brief names the undecided criterion (%q):\n%s", want, brief)
		}
	}
}

// TestTheFixRoundCapIsSetBesideThePace is ruling DR1: a run's fix-round cap is
// 3 when nothing sets it, the repository's pace.fix_rounds over the bundled
// value, --fix-rounds over every layer; the run keeps it in its state and its
// record names it; a malformed value is refused naming the accepted form and
// writes no state; a resume naming another cap is refused.
func TestTheFixRoundCapIsSetBesideThePace(t *testing.T) {
	t.Run("bundled", func(t *testing.T) {
		repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
		res, err := Start(repo.Root(), "itd-10", Options{})
		if err != nil {
			t.Fatal(err)
		}
		st, _ := ReadState(repo.Root(), res.RunID)
		if f := st.Pace.FixRounds; f.Value != BundledFixRounds || f.Layer != "bundled" || BundledFixRounds != 3 {
			t.Fatalf("the bundled cap is 3 fix rounds: %+v", f)
		}
		if note := paceRecord(t, st); !strings.Contains(note, "3 fix rounds") {
			t.Fatalf("the run record names the cap: %q", note)
		}
	})
	t.Run("repo", func(t *testing.T) {
		repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
		repoConfig(t, repo, `{"pace": {"fix_rounds": 5}}`)
		res, err := Start(repo.Root(), "itd-10", Options{})
		if err != nil {
			t.Fatal(err)
		}
		if f := res.Pace.FixRounds; f.Value != 5 || f.Layer != "repo" {
			t.Fatalf("the repository's cap wins over the bundled one: %+v", f)
		}
	})
	t.Run("flag", func(t *testing.T) {
		repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
		repoConfig(t, repo, `{"pace": {"fix_rounds": 5}}`)
		res, err := Start(repo.Root(), "itd-10", Options{FixRounds: strp("1")})
		if err != nil {
			t.Fatal(err)
		}
		st, _ := ReadState(repo.Root(), res.RunID)
		if f := st.Pace.FixRounds; f.Value != 1 || f.Layer != "flag" || f.Origin != "--fix-rounds 1" {
			t.Fatalf("--fix-rounds wins over every layer and is kept in the run: %+v", f)
		}
		again, err := Start(repo.Root(), "itd-10", Options{FixRounds: strp("1")})
		if err != nil || !again.Resumed {
			t.Fatalf("the same cap resumes: %+v %v", again, err)
		}
		before := stateBytes(t, repo.Root(), res.RunID)
		_, err = Start(repo.Root(), "itd-10", Options{FixRounds: strp("4")})
		if r := mustRefusal(t, err); r.Stage != StagePace || !strings.Contains(r.Reason, "--fix-rounds 4") {
			t.Fatalf("a resume naming another cap is refused, naming it: %+v", r)
		}
		if !bytes.Equal(before, stateBytes(t, repo.Root(), res.RunID)) {
			t.Fatal("a refused resume changes nothing")
		}
	})
	for _, bad := range []string{"three", "-1", "65", "1.5", ""} {
		t.Run("malformed "+bad, func(t *testing.T) {
			repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
			_, err := Start(repo.Root(), "itd-10", Options{FixRounds: strp(bad)})
			r := mustRefusal(t, err)
			if r.Stage != StagePace || !strings.Contains(r.Remedy, "--fix-rounds") {
				t.Fatalf("a malformed cap is refused naming the accepted form: %+v", r)
			}
			runTierAbsent(t, repo.Root())
		})
	}
}

// TestALaneThatExhaustsItsFixRoundsIsHandedBack is DR1's bound on itd-50's
// criterion 2: once a lane has taken the run's cap of fix rounds and the next
// round still does not pass, no further fix round starts: the lane stops as
// unachievable, the run starts nothing further, and the intent is handed back
// to the person, loudly, with the last round's findings; every later step
// refuses naming the hand-back.
func TestALaneThatExhaustsItsFixRoundsIsHandedBack(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{FixRounds: strp("1")})
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, DefaultStages()
	l := implemented(t, repo, id, stages, "one.txt")

	handBack(t, repo, id, stages, RoleRuthless, reviewerReturn("FIX FIRST"))
	passRound(t, repo, id, stages, RoleSecurity, RoleAuditor)
	fix := laneCommit(t, repo, l, "fix.txt")
	fixed(t, repo, id, stages, "applied the finding in "+fix+"\n", fix)

	passRound(t, repo, id, stages, RoleRuthless, RoleSecurity)
	handBack(t, repo, id, stages, RoleAuditor, "NOT_MET")
	res, err := Advance(repo.Root(), id, stages, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Awaiting != nil || res.Stage != StageHandedBack || res.Complete || res.HandBack == nil {
		t.Fatalf("a round past the cap hands the lane back and starts no fix round: %+v", res)
	}
	st, _ := ReadState(repo.Root(), id)
	lane := st.Lanes[0]
	hb := lane.HandBack
	if lane.Stage != StageHandedBack || hb == nil || hb.Verdict != VerdictUnachievable || hb.Round != 2 || hb.FixRounds != 1 {
		t.Fatalf("the lane stops as unachievable after its one fix round: %+v", lane)
	}
	if len(hb.Findings) != 1 || !strings.HasSuffix(hb.Findings[0], "round-2/intent-auditor/"+VerdictFileName) || len(hb.NotMet) != 1 || hb.NotMet[0] != "ac-1" {
		t.Fatalf("the hand-back carries the last round's findings: %+v", hb)
	}
	if _, err := os.Stat(filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), id, lane.ID, "validate", "round-2", FixDirName)); !os.IsNotExist(err) {
		t.Fatalf("no fix brief is written past the cap: %v", err)
	}
	for _, want := range []string{"itd-10", "handed back", "unachievable", "1 fix round", "ac-1", hb.Findings[0]} {
		if !strings.Contains(res.Next, want) {
			t.Fatalf("the hand-back is loud, naming %q: %s", want, res.Next)
		}
	}
	var record strings.Builder
	for _, e := range st.Record {
		record.WriteString(e.Stage + " " + e.Note + "\n")
	}
	if !strings.Contains(record.String(), "handed-back") || !strings.Contains(record.String(), "unachievable") {
		t.Fatalf("the run record names the hand-back:\n%s", record.String())
	}

	before := stateBytes(t, repo.Root(), id)
	_, err = Advance(repo.Root(), id, stages, Options{})
	if r := mustRefusal(t, err); r.Stage != string(StageHandedBack) || !strings.Contains(r.Reason, "unachievable") || !strings.Contains(r.Remedy, "itd-10") {
		t.Fatalf("a handed-back lane starts nothing further, and says why: %+v", r)
	}
	if !bytes.Equal(before, stateBytes(t, repo.Root(), id)) {
		t.Fatal("a step on a handed-back lane changes nothing")
	}
	again, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil || !again.Resumed || !strings.Contains(again.Next, "handed back") {
		t.Fatalf("building the intent again names the hand-back rather than starting over: %+v %v", again, err)
	}
}

// TestAHandedBackPickNamesThePickFalsified is itd-2609211116005482's criterion
// on the pick's falsifier: a run `abcd build next` started whose lane is handed
// back past the cap records the pick as falsified, and the intent's grounds
// entry is not edited.
func TestAHandedBackPickNamesThePickFalsified(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{FixRounds: strp("0")})
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, DefaultStages()
	implemented(t, repo, id, stages, "one.txt")
	// Mark the run as one a pick started, as `abcd build next` writes it.
	path := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(id)))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var st State
	if err := json.Unmarshal(raw, &st); err != nil {
		t.Fatal(err)
	}
	st.Pick = &RunPick{Lane: "lane-1", Entry: "picked by run " + id}
	out, _ := json.Marshal(st)
	if err := os.WriteFile(path, out, 0o600); err != nil {
		t.Fatal(err)
	}
	handBack(t, repo, id, stages, RoleRuthless, reviewerReturn("FIX FIRST"))
	passRound(t, repo, id, stages, RoleSecurity, RoleAuditor)
	res, err := Advance(repo.Root(), id, stages, Options{})
	if err != nil || res.HandBack == nil {
		t.Fatalf("with no fix round allowed, the first failing round hands the lane back: %+v %v", res, err)
	}
	st, _ = ReadState(repo.Root(), id)
	found := false
	for _, e := range st.Record {
		if e.Stage == "pick" && strings.Contains(e.Note, "falsified") {
			found = true
		}
	}
	if !found {
		t.Fatalf("the run record names the pick as falsified: %+v", st.Record)
	}
}

// TestAVersion5StateRunsOnTheBundledCap: version 6 added the fix-round cap, so
// a version-5 file is read as a run on the bundled cap and written back at
// version 6, and a version-5 file carrying a cap or a hand-back is not one
// version 5 wrote, and is refused.
func TestAVersion5StateRunsOnTheBundledCap(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	start, err := Start(repo.Root(), "itd-10", Options{FixRounds: strp("1")})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(start.RunID)))
	current := stateBytes(t, repo.Root(), start.RunID)
	if err := os.WriteFile(path, downgraded(t, current, 5), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil || st.SchemaVersion != SchemaVersion || st.FixRoundCap() != BundledFixRounds {
		t.Fatalf("a version-5 file reads as a run on the bundled cap: %+v %v", st.Pace, err)
	}
	carrying := strings.Replace(string(current), fmt.Sprintf(`"schema_version": %d,`, SchemaVersion), `"schema_version": 5,`, 1)
	if err := os.WriteFile(path, []byte(carrying), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = ReadState(repo.Root(), start.RunID)
	if r := mustRefusal(t, err); r.Stage != "state" || !strings.Contains(r.Reason, "fix-round cap") {
		t.Fatalf("a version-5 file carrying a cap is refused: %+v", r)
	}
}
