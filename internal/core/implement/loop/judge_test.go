package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The judge strikes false findings only (itd-2610240922461281,
// spc-2610240951109420 step 5).

// calibratedJudge is the fixture that stands in for itd-81's calibration
// gate: a judge agent has cleared it.
var calibratedJudge = Options{judgeCalibrated: func() (bool, string) { return true, "" }}

// judgedRound starts a run under o, implements its lane and takes the round
// to the point its every validator has returned: the ruthless reviewers
// return ruthless0 and ruthless1, the security reviewers APPROVE both and the
// auditor MET. It returns the run, its stages, the lane and the two ruthless
// awaits.
func judgedRound(t *testing.T, o Options, ruthless0, ruthless1 string) (*gittest.Repo, string, Stages, Lane, [2]Await) {
	t.Helper()
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", o)
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, DefaultStages()
	l := implemented(t, repo, id, stages, "one.txt")
	r0 := stepAwait(t, repo, id, stages, o)
	r1 := stepAwait(t, repo, id, stages, o)
	returnAt(t, repo, id, stages, r0, ruthless0)
	returnAt(t, repo, id, stages, r1, ruthless1)
	s0 := stepAwait(t, repo, id, stages, o)
	s1 := stepAwait(t, repo, id, stages, o)
	returnAt(t, repo, id, stages, s0, reviewerReturn("APPROVE"))
	returnAt(t, repo, id, stages, s1, reviewerReturn("APPROVE"))
	aud := stepAwait(t, repo, id, stages, o)
	returnAt(t, repo, id, stages, aud, "MET")
	return repo, id, stages, l, [2]Await{r0, r1}
}

// judgeReturn writes body under the round's head line as the judge's return
// and hands it back.
func judgeReturn(t *testing.T, repo *gittest.Repo, id string, stages Stages, a Await, head, body string) (StepResult, error) {
	t.Helper()
	path := filepath.Join(repo.Root(), filepath.FromSlash(a.Receipt))
	if err := os.WriteFile(path, []byte("Reviewed head: "+head+"\n\n"+body), 0o600); err != nil {
		t.Fatal(err)
	}
	return Receipt(repo.Root(), id, path, stages, Options{})
}

// readBrief reads a brief an await names.
func readBrief(t *testing.T, repo *gittest.Repo, a Await) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(a.Brief)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestADisagreementHandsAnActiveJudgeBothReturnsAndTheHead: with a calibrated
// judge fixture, a role whose two reviewers disagree hands the run's active
// judge both reviewers' returns and the round's head, once every validator
// of the round has returned, before any fix round; the await names the
// judge's class and its model. Until its return is in, nothing else is
// handed out.
func TestADisagreementHandsAnActiveJudgeBothReturnsAndTheHead(t *testing.T) {
	repo, id, stages, l, r := judgedRound(t, calibratedJudge,
		findingsReturn("SHIP"), findingsReturn("FIX FIRST", "a.go:3", "b.go:5"))
	a := stepAwait(t, repo, id, stages, calibratedJudge)
	if a.Role != RoleJudge || a.Class != ClassTop || a.Model != BundledClasses[ClassTop] || a.Reviewer != nil {
		t.Fatalf("the disagreement hands out the run's judge at its class: %+v", a)
	}
	if !strings.HasSuffix(filepath.Dir(a.Receipt), "/round-1/"+JudgeDirName) {
		t.Fatalf("the judge writes in the round's judge directory: %s", a.Receipt)
	}
	brief := readBrief(t, repo, a)
	for _, want := range []string{abs(repo.Root(), r[0].Receipt), abs(repo.Root(), r[1].Receipt), l.HeadSHA,
		"a.go:3", "b.go:5", "### Struck", "never pass"} {
		if !strings.Contains(brief, want) {
			t.Fatalf("the judge's brief names %q:\n%s", want, brief)
		}
	}
	if res, err := advance(repo.Root(), id, stages, calibratedJudge); err == nil && res.handed {
		t.Fatalf("nothing else is handed out while the judge is out: %+v", res.Awaiting)
	}
	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	j := st.Lanes[0].Validation[0].Judge
	if j == nil || !slices.Equal(j.Roles, []string{RoleRuthless}) || j.Return != a.Receipt || j.Ruled {
		t.Fatalf("the round records its judge, out on the ruthless reviewers' disagreement: %+v", j)
	}
}

// TestAWaitingJudgeIsNeverLaunched: a run started with its judge waiting
// (no agent cleared the gate when it started) hands the disagreement straight
// to a fix round, whatever is calibrated by then; the round records no judge.
func TestAWaitingJudgeIsNeverLaunched(t *testing.T) {
	repo, id, stages, _, _ := judgedRound(t, Options{}, findingsReturn("SHIP"), findingsReturn("FIX FIRST", "a.go:3"))
	a := stepAwait(t, repo, id, stages, calibratedJudge)
	if a.Role != RoleImplementer {
		t.Fatalf("a waiting judge is never launched; the fix round follows: %+v", a)
	}
	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	if st.Review.Judge.Status != JudgeWaiting || st.Lanes[0].Validation[0].Judge != nil {
		t.Fatalf("the judge waits and the round records none: %+v %+v", st.Review.Judge, st.Lanes[0].Validation[0].Judge)
	}
	if !strings.Contains(readBrief(t, repo, a), "located findings: a.go:3") {
		t.Fatalf("every finding goes to the fix round:\n%s", readBrief(t, repo, a))
	}
}

// TestAJudgeWhoseAgentNoLongerClearsTheGateIsNotLaunched: an active judge is
// launched only where its agent clears the calibration gate when it launches;
// one that no longer does is not launched, the round records why, and the
// fix round follows with every finding.
func TestAJudgeWhoseAgentNoLongerClearsTheGateIsNotLaunched(t *testing.T) {
	repo, id, stages, _, _ := judgedRound(t, calibratedJudge, findingsReturn("SHIP"), findingsReturn("FIX FIRST", "a.go:3"))
	lapsed := Options{judgeCalibrated: func() (bool, string) { return false, "the judge agent's calibration record no longer passes" }}
	a := stepAwait(t, repo, id, stages, lapsed)
	if a.Role != RoleImplementer {
		t.Fatalf("a judge that no longer clears the gate is not launched: %+v", a)
	}
	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	j := st.Lanes[0].Validation[0].Judge
	if j == nil || !strings.Contains(j.Unlaunched, "no longer passes") || j.Return != "" || j.Ruled {
		t.Fatalf("the round records why its judge was not launched: %+v", j)
	}
	if !strings.Contains(readBrief(t, repo, a), "located findings: a.go:3") {
		t.Fatalf("every finding goes to the fix round:\n%s", readBrief(t, repo, a))
	}
}

// TestAStruckFindingLeavesTheFixBriefWithItsReasonRecorded: a finding the
// judge strikes is not handed to the fix round, its reason is recorded on the
// finding, and the findings it did not strike are; after the fix the struck
// finding keeps its strike and the others carry what the fix report said.
func TestAStruckFindingLeavesTheFixBriefWithItsReasonRecorded(t *testing.T) {
	repo, id, stages, l, _ := judgedRound(t, calibratedJudge,
		findingsReturn("SHIP"), findingsReturn("FIX FIRST", "a.go:3", "b.go:5"))
	a := stepAwait(t, repo, id, stages, calibratedJudge)
	if _, err := judgeReturn(t, repo, id, stages, a, l.HeadSHA,
		"### Analysis\n\nRead both.\n\n### Struck\n\n- `a.go:3` — the caller already checks it at caller.go:9\n"); err != nil {
		t.Fatal(err)
	}
	fix := stepAwait(t, repo, id, stages, calibratedJudge)
	if fix.Role != RoleImplementer {
		t.Fatalf("the round goes to a fix round: %+v", fix)
	}
	brief := readBrief(t, repo, fix)
	if !strings.Contains(brief, "located findings: b.go:5\n") || strings.Contains(brief, "located findings: a.go:3") {
		t.Fatalf("the struck finding leaves the fix brief, the other stays:\n%s", brief)
	}
	if strings.Contains(brief, "the caller already checks it") {
		t.Fatalf("the judge's reason is recorded, not handed on:\n%s", brief)
	}
	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	r := st.Lanes[0].Validation[0]
	fs := r.Validators[1].Findings
	if fs[0].Status != FindingStruck || fs[0].Reason != "the caller already checks it at caller.go:9" || fs[1].Status != "" || fs[1].Reason != "" {
		t.Fatalf("the struck finding carries its reason: %+v", fs)
	}
	if r.Judge == nil || !r.Judge.Ruled || r.Judge.Struck != 1 {
		t.Fatalf("the round's judge ruled and struck one: %+v", r.Judge)
	}

	commit := laneCommit(t, repo, currentLane(t, repo, id), "b.go")
	fixed(t, repo, id, stages, "- b.go:5 applied in "+commit+"\n- a.go:3 rejected\n", commit)
	st, err = ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	fs = st.Lanes[0].Validation[0].Validators[1].Findings
	if fs[0].Status != FindingStruck || fs[1].Status != FindingApplied {
		t.Fatalf("after the fix the strike stands and the rest carry the report's word: %+v", fs)
	}
	rec, err := ReadRecord(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `{"file":"a.go","line":3,"status":"struck","reason":"the caller already checks it at caller.go:9"}`) ||
		!strings.Contains(string(raw), `"judges":[{"round":1,`) {
		t.Fatalf("implement record --json carries the strike and its reason, and the round's judge:\n%s", raw)
	}
}

// TestTheJudgeNeverPassesARound: a judge's return that states a passing
// verdict and strikes nothing passes nothing: the round goes to a fix round
// with every finding.
func TestTheJudgeNeverPassesARound(t *testing.T) {
	repo, id, stages, l, _ := judgedRound(t, calibratedJudge, findingsReturn("SHIP"), findingsReturn("FIX FIRST", "a.go:3"))
	a := stepAwait(t, repo, id, stages, calibratedJudge)
	if _, err := judgeReturn(t, repo, id, stages, a, l.HeadSHA, "### Verdict\n\n- **SHIP** — "+l.HeadSHA+"\n"); err != nil {
		t.Fatal(err)
	}
	res, err := advance(repo.Root(), id, stages, calibratedJudge)
	if err != nil {
		t.Fatal(err)
	}
	if res.Awaiting == nil || res.Awaiting.Role != RoleImplementer || res.Stage != StageValidate {
		t.Fatalf("the judge never passes a round: %+v", res)
	}
	if !strings.Contains(readBrief(t, repo, *res.Awaiting), "located findings: a.go:3") {
		t.Fatalf("a judge striking nothing hands every finding on:\n%s", readBrief(t, repo, *res.Awaiting))
	}
}

// TestARoundWhoseFailingFindingsAreAllStruckOpensAnEmptyFixRound: a round
// whose every failing finding the judge struck still does not pass: it goes
// to a fix round whose brief hands on no finding.
func TestARoundWhoseFailingFindingsAreAllStruckOpensAnEmptyFixRound(t *testing.T) {
	repo, id, stages, l, _ := judgedRound(t, calibratedJudge, findingsReturn("SHIP"), findingsReturn("FIX FIRST", "a.go:3", "b.go:5"))
	a := stepAwait(t, repo, id, stages, calibratedJudge)
	if _, err := judgeReturn(t, repo, id, stages, a, l.HeadSHA,
		"### Struck\n\n- a.go:3 — not reachable from any caller\n- **b.go:5**: the bound is checked one line above\n"); err != nil {
		t.Fatal(err)
	}
	fix := stepAwait(t, repo, id, stages, calibratedJudge)
	if fix.Role != RoleImplementer {
		t.Fatalf("a round whose every failing finding is struck still goes to a fix round: %+v", fix)
	}
	brief := readBrief(t, repo, fix)
	if strings.Contains(brief, "located findings:") || !strings.Contains(brief, "hands on no finding") {
		t.Fatalf("the fix brief hands on no finding:\n%s", brief)
	}
}

// TestAgreeingReviewersStartNoJudge: two reviewers that agree, both failing,
// start no judge, active as it is: the fix round follows.
func TestAgreeingReviewersStartNoJudge(t *testing.T) {
	repo, id, stages, _, _ := judgedRound(t, calibratedJudge, findingsReturn("FIX FIRST", "a.go:3"), findingsReturn("FIX FIRST", "b.go:5"))
	a := stepAwait(t, repo, id, stages, calibratedJudge)
	if a.Role != RoleImplementer {
		t.Fatalf("agreeing reviewers start no judge: %+v", a)
	}
	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	if st.Review.Judge.Status != JudgeActive || st.Lanes[0].Validation[0].Judge != nil {
		t.Fatalf("the active judge is not started on an agreement: %+v", st.Lanes[0].Validation[0].Judge)
	}
}

// TestAJudgeMayStrikeOnlyAFindingItWasHanded: a strike naming a location no
// failing reviewer of the disagreeing role cites, or a strike with no reason,
// is refused, and nothing is recorded.
func TestAJudgeMayStrikeOnlyAFindingItWasHanded(t *testing.T) {
	repo, id, stages, l, _ := judgedRound(t, calibratedJudge, findingsReturn("SHIP", "c.go:1"), findingsReturn("FIX FIRST", "a.go:3"))
	a := stepAwait(t, repo, id, stages, calibratedJudge)
	for _, body := range []string{
		"### Struck\n\n- c.go:1 — the passing reviewer's own note\n",
		"### Struck\n\n- other.go:7 — never cited\n",
		"### Struck\n\n- a.go:3\n",
	} {
		if _, err := judgeReturn(t, repo, id, stages, a, l.HeadSHA, body); err == nil {
			t.Fatalf("a strike outside the handed findings or with no reason is refused: %q", body)
		}
	}
	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	if j := st.Lanes[0].Validation[0].Judge; j == nil || j.Ruled || st.Lanes[0].Validation[0].Validators[1].Findings[0].Status != "" {
		t.Fatalf("a refused return records nothing: %+v", j)
	}
}

// TestAJudgedRoundIsTheVersionAfterTheUnjudged: a round's judge and a struck
// finding are state schema version 28's; a version-27 file carrying either
// is refused naming it, and a stored judge or strike the loop never writes is
// refused at read.
func TestAJudgedRoundIsTheVersionAfterTheUnjudged(t *testing.T) {
	if schemaVersionUnreasoned != schemaVersionUnjudged+1 {
		t.Fatalf("the judge is the version after %d: %d", schemaVersionUnjudged, schemaVersionUnreasoned)
	}
	repo, id, stages, l, _ := judgedRound(t, calibratedJudge, findingsReturn("SHIP"), findingsReturn("FIX FIRST", "a.go:3", "b.go:5"))
	a := stepAwait(t, repo, id, stages, calibratedJudge)
	if _, err := judgeReturn(t, repo, id, stages, a, l.HeadSHA, "### Struck\n\n- a.go:3 — false\n"); err != nil {
		t.Fatal(err)
	}
	path := runStatePath(repo.Root(), id)
	good, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stateAtVersion(t, repo.Root(), id, schemaVersionUnjudged)
	if _, err := ReadState(repo.Root(), id); err == nil || !strings.Contains(err.Error(), "judge") {
		t.Fatalf("a version-27 file carrying a judge is refused naming it: %v", err)
	}
	for name, edit := range map[string]func(map[string]any){
		"a strike with no reason": func(r map[string]any) {
			f := r["validators"].([]any)[1].(map[string]any)["findings"].([]any)[0].(map[string]any)
			delete(f, "reason")
		},
		"a reason on a finding not struck": func(r map[string]any) {
			f := r["validators"].([]any)[1].(map[string]any)["findings"].([]any)[1].(map[string]any)
			f["reason"] = "x"
		},
		"a strike on a passing reviewer": func(r map[string]any) {
			v := r["validators"].([]any)[1].(map[string]any)
			v["verdict"], v["pass"] = "SHIP", true
		},
		"a judge on no disagreement": func(r map[string]any) {
			delete(r, "comparisons")
		},
		"a judge at another class": func(r map[string]any) {
			r["judge"].(map[string]any)["class"] = ClassSmall
		},
		"a count the judge did not strike": func(r map[string]any) {
			r["judge"].(map[string]any)["struck"] = 3
		},
	} {
		var doc map[string]any
		if err := json.Unmarshal(good, &doc); err != nil {
			t.Fatal(err)
		}
		edit(doc["lanes"].([]any)[0].(map[string]any)["validation"].([]any)[0].(map[string]any))
		data, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadState(repo.Root(), id); err == nil {
			t.Fatalf("%s is refused at read", name)
		}
	}
	if err := os.WriteFile(path, good, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadState(repo.Root(), id); err != nil {
		t.Fatalf("the file the loop wrote reads: %v", err)
	}
}
