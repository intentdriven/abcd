package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/interview"
	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/surface/cli/ask"
	"github.com/intentdriven/abcd/internal/term"
)

// written_interview_test.go holds `abcd reflect interview` and `abcd intent
// interview` to step 4 of spc-2610030911534855 at the front door: a stub
// runner's questions drawn and recorded, the outcome filed through the
// writer's own path, the no-route refusal, the invalid question recorded as
// a fallback, the answers-file replay by ordinal, and the runner's text
// sanitised before it is drawn.

// interviewStubScriptEnv names the stub runner's script: TestMain plays the
// stub runner when it is set and the binary was started as claude is.
const interviewStubScriptEnv = "ABCD_CLI_INTERVIEW_STUB"

// interviewStubRunner plays the claude runner for one turn: it copies the
// script's turn-<n>.json to the receipt path the prompt names (no file, no
// receipt), keeps the turn's brief beside the script, and reports success.
func interviewStubRunner(dir string) int {
	prompt := os.Args[len(os.Args)-1]
	field := func(prefix string) string {
		sc := bufio.NewScanner(strings.NewReader(prompt))
		for sc.Scan() {
			if v, ok := strings.CutPrefix(sc.Text(), prefix); ok {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	receipt, brief := field("Receipt: "), field("Brief: ")
	turn := strings.TrimSuffix(filepath.Base(receipt), ".receipt.json")
	if b, err := os.ReadFile(brief); err == nil {
		_ = os.WriteFile(filepath.Join(dir, turn+".brief.seen.md"), b, 0o600)
	}
	if body, err := os.ReadFile(filepath.Join(dir, turn+".json")); err == nil {
		if err := os.WriteFile(receipt, body, 0o600); err != nil {
			return 3
		}
	}
	fmt.Println(`{"type":"system","subtype":"init","session_id":"stub-session-1","model":"stub-model"}`)
	fmt.Println(`{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"stub-session-1"}`)
	return 0
}

// interviewStub puts the stub runner on PATH as claude, ahead of the rest of
// PATH (the verbs still run git), with one receipt per turn.
func interviewStub(t *testing.T, receipts ...string) string {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin, script := t.TempDir(), t.TempDir()
	if err := os.Symlink(self, filepath.Join(bin, "claude")); err != nil {
		t.Fatal(err)
	}
	for i, r := range receipts {
		if err := os.WriteFile(filepath.Join(script, fmt.Sprintf("turn-%d.json", i+1)), []byte(r), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv(interviewStubScriptEnv, script)
	return script
}

// stubQuestion is an ask receipt the stub runner writes: one question that
// passes the structural check and the asking limits, its material carrying
// extra when given.
func stubQuestion(chip, ask, extra string) string {
	a := question.Ask{Questions: []question.Question{{
		ID:   "Q1",
		Chip: chip,
		Material: []question.Block{{Kind: question.KindParagraph,
			Text: "The release shipped the seed builder and the refusals that name the tag. For example" + extra + ", the seed named every intent the tag carried."}},
		Ask: ask,
		Options: []question.Option{
			{Value: "kept", Label: "Keep it" + extra, Meaning: "The answer stands" + extra + ". Nothing more is asked about it."},
			{Value: "more", Label: "Add more to it", Meaning: "The next question asks for the rest. It takes one more turn."},
		},
		Later:       question.Option{Value: "later", Label: "Decide later", Meaning: "Nothing is recorded for it now. The interview asks again."},
		Now:         "not applicable",
		ChangeLater: "not applicable",
	}}}
	b, err := json.Marshal(map[string]question.Ask{"ask": a})
	if err != nil {
		panic(err)
	}
	return string(b)
}

// routeMachine writes the machine's configuration under a fresh HOME: role
// routed to the claude runner, enabled; nothing when role is "".
func routeMachine(t *testing.T, role string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if role == "" {
		return home
	}
	p := abcdhome.Path(home, "config.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	body := `{"roles":{"` + role + `":{"runner":"claude"}},"runner":{"claude":{}}}`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return home
}

// retroRepo is reflectRepo with a local tier, the working directory set to
// it.
func retroRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	r := reflectRepo(t)
	if err := os.MkdirAll(filepath.Join(r.Root(), ".abcd", ".work.local"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(r.Root())
	return r
}

// interviewRun runs args with every stream a file or a pipe. drawn makes the
// streams stand in for terminals and types each question's first option
// into the numbered reader; mode is the colour rung it draws at. It returns
// stderr, where the questions are drawn or written.
func interviewRun(t *testing.T, drawn bool, mode term.ColorMode, args ...string) (string, error) {
	t.Helper()
	in, keys := pipeWith(t, "")
	stdout, stderr := tempStream(t, "stdout"), tempStream(t, "stderr")
	if drawn {
		t.Setenv("ABCD_ACCESSIBLE", "1")
		swapStream, swapPut := isTerminalStream, drawnPut
		t.Cleanup(func() { isTerminalStream, drawnPut = swapStream, swapPut })
		isTerminalStream = func(*os.File) bool { return true }
		drawnPut = func(tm ask.Terminal, a question.Ask) ([]ask.Answer, error) {
			tm.Mode = mode
			for _, q := range a.Questions {
				if _, err := fmt.Fprintln(keys, number(q, q.Options[0].Value)); err != nil {
					return nil, err
				}
			}
			return tm.Put(a)
		}
	} else {
		_ = keys.Close()
	}
	cmd := NewRootCommand()
	cmd.SetIn(in)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	got, _ := os.ReadFile(stderr.Name())
	return string(got), err
}

func readInterviewRecord(t *testing.T, repo string) interview.Record {
	t.Helper()
	b := oneRecord(t, filepath.Join(repo, filepath.FromSlash(interview.RecordsRel)))
	if b == nil {
		t.Fatal("no answers record was written")
	}
	var rec interview.Record
	if err := json.Unmarshal(b, &rec); err != nil {
		t.Fatal(err)
	}
	return rec
}

func retroAnswersFile(t *testing.T, ids ...string) string {
	t.Helper()
	var entries []interview.FileAnswer
	for _, id := range ids {
		entries = append(entries, interview.FileAnswer{ID: id, Value: "kept"})
	}
	b, err := json.Marshal(interview.Answers{SchemaVersion: interview.SchemaVersion, Interview: interview.Retrospective, Answers: entries})
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "answers.json")
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// retroReceipts are the stub runner's turns: two questions, then the
// answers object as done.
func retroReceipts() []string {
	return []string{
		stubQuestion("Product Q1", "Is that the whole of what went well?", ""),
		stubQuestion("Product Q2", "Is that the whole of what could improve?", ""),
		`{"done":` + fullAnswers + `}`,
	}
}

// TestReflectInterviewDrawsTwoAsksThenWritesTheRetrospective is step 4's
// first test: a stub runner's two ask receipts are drawn and recorded with
// where they were answered, and its done writes the retrospective through
// reflect write's own path.
func TestReflectInterviewDrawsTwoAsksThenWritesTheRetrospective(t *testing.T) {
	routeMachine(t, interview.RoleReflectionComposer)
	script := interviewStub(t, retroReceipts()...)
	r := retroRepo(t)
	stderr, err := interviewRun(t, true, term.Mono, "reflect", "interview", "v0.2.0", "--proceed")
	if err != nil {
		t.Fatalf("err = %v\n%s", err, stderr)
	}
	for _, want := range []string{"Is that the whole of what went well?", "Is that the whole of what could improve?", "the answers are recorded in " + interview.RecordsRel} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr does not carry %q:\n%s", want, stderr)
		}
	}
	rec := readInterviewRecord(t, r.Root())
	if rec.Interview != interview.Retrospective || rec.Target != "v0.2.0" || len(rec.Answers) != 2 {
		t.Fatalf("record %+v", rec)
	}
	for i, a := range rec.Answers {
		if a.ID != fmt.Sprintf("Q%d", i+1) || a.Value != "kept" || a.AnsweredIn != interview.Terminal {
			t.Fatalf("answer %d: %+v", i, a)
		}
	}
	retro, err := os.ReadFile(retroPath(r, "v0.2.0"))
	if err != nil {
		t.Fatalf("the retrospective was not written: %v", err)
	}
	if !strings.Contains(string(retro), "The seed builder read the release from the tag tree") {
		t.Fatalf("the retrospective does not carry the done outcome:\n%s", retro)
	}
	if b, err := os.ReadFile(filepath.Join(script, "turn-3.brief.seen.md")); err != nil || !strings.Contains(string(b), `"id": "Q2"`) {
		t.Fatalf("the third brief does not carry the answers so far: %v\n%s", err, b)
	}
}

// TestRetrospectiveRecordsDifferOnlyInWhereAnswered is B5's second case:
// the retrospective interview against a stub runner returning the same two
// questions and then done, once drawn and once from an answers file marked
// Claude Code, gives answers records equal but for answered_in and the same
// retrospective.
func TestRetrospectiveRecordsDifferOnlyInWhereAnswered(t *testing.T) {
	run := func(drawn bool) (interview.Record, []byte) {
		routeMachine(t, interview.RoleReflectionComposer)
		interviewStub(t, retroReceipts()...)
		r := retroRepo(t)
		args := []string{"reflect", "interview", "v0.2.0", "--proceed"}
		if !drawn {
			args = append(args, "--answers", retroAnswersFile(t, "Q1", "Q2"), "--answered-in", interview.ClaudeCode)
		}
		if stderr, err := interviewRun(t, drawn, term.Mono, args...); err != nil {
			t.Fatalf("drawn %v: err = %v\n%s", drawn, err, stderr)
		}
		retro, err := os.ReadFile(retroPath(r, "v0.2.0"))
		if err != nil {
			t.Fatal(err)
		}
		return readInterviewRecord(t, r.Root()), retro
	}
	drawnRec, drawnRetro := run(true)
	fileRec, fileRetro := run(false)
	for i := range drawnRec.Answers {
		if drawnRec.Answers[i].AnsweredIn != interview.Terminal || fileRec.Answers[i].AnsweredIn != interview.ClaudeCode {
			t.Fatalf("answered_in: drawn %q, file %q", drawnRec.Answers[i].AnsweredIn, fileRec.Answers[i].AnsweredIn)
		}
		drawnRec.Answers[i].AnsweredIn, fileRec.Answers[i].AnsweredIn = "", ""
	}
	if !reflect.DeepEqual(drawnRec, fileRec) {
		t.Fatalf("the records differ beyond answered_in:\n drawn %+v\n file  %+v", drawnRec, fileRec)
	}
	if string(drawnRetro) != string(fileRetro) {
		t.Fatalf("the retrospectives differ:\n--- drawn\n%s\n--- file\n%s", drawnRetro, fileRetro)
	}
}

// TestInterviewWithNoRouteRefusesWritingNothing: with no route of the
// person's to a runner, both interviews refuse before anything runs, exit
// 2, naming the role key, the machine's file and `abcd ahoy install`, and
// write nothing.
func TestInterviewWithNoRouteRefusesWritingNothing(t *testing.T) {
	for _, c := range []struct {
		name, role string
		args       []string
	}{
		{"retrospective", interview.RoleReflectionComposer, []string{"reflect", "interview", "v0.2.0", "--proceed"}},
		{"planning", interview.RolePlanningInterviewer, []string{"intent", "interview", "itd-5"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			home := routeMachine(t, "")
			script := interviewStub(t, retroReceipts()...)
			r := retroRepo(t)
			before := listTree(t, r.Root())
			stderr, err := interviewRun(t, false, term.Mono, append(c.args, "--answers", planningAnswersFile(t))...)
			if exitCodeOf(err) != 2 {
				t.Fatalf("exit %d, err %v\n%s", exitCodeOf(err), err, stderr)
			}
			for _, want := range []string{"roles." + c.role + ".runner", abcdhome.Display("config.json"), "`abcd ahoy install`"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("the refusal does not name %q:\n%s", want, err)
				}
			}
			if after := listTree(t, r.Root()); after != before {
				t.Fatalf("the refusal wrote into the repository:\n--- before\n%s--- after\n%s", before, after)
			}
			if _, err := os.Stat(filepath.Join(script, "turn-1.brief.seen.md")); !os.IsNotExist(err) {
				t.Fatal("the runner was started")
			}
			if _, err := os.Stat(abcdhome.Path(home, "transcripts")); !os.IsNotExist(err) {
				t.Fatal("a transcript was stored")
			}
		})
	}
}

// TestInvalidRunnerQuestionIsAFallbackAndRefused: an ask the check refuses
// is recorded as the dispatcher's invalid fallback, and with no fallback
// host the interview stops, exit 1, naming the runner and the reason, with
// the record holding the receipt and no retrospective written.
func TestInvalidRunnerQuestionIsAFallbackAndRefused(t *testing.T) {
	routeMachine(t, interview.RoleReflectionComposer)
	interviewStub(t, strings.Replace(stubQuestion("Product Q1", "Is that the whole of it?", ""), `"Decide later"`, `""`, 1))
	r := retroRepo(t)
	stderr, err := interviewRun(t, false, term.Mono, "reflect", "interview", "v0.2.0", "--proceed", "--answers", retroAnswersFile(t, "Q1"))
	if exitCodeOf(err) != 1 || !strings.Contains(err.Error(), "claude") || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("exit %d, err %v\n%s", exitCodeOf(err), err, stderr)
	}
	rec := readInterviewRecord(t, r.Root())
	if len(rec.Answers) != 0 || len(rec.Fallbacks) != 1 || rec.Fallbacks[0].Reason != "invalid" || rec.Fallbacks[0].Ran != "none" {
		t.Fatalf("record %+v", rec)
	}
	if _, err := os.Stat(retroPath(r, "v0.2.0")); !os.IsNotExist(err) {
		t.Fatal("a retrospective was written")
	}
}

// TestAnswersFileReplaysByOrdinalAndRefusesWhenItRunsOut: off a terminal the
// answers come from the file by ordinal; a file that runs out refuses at the
// question it cannot answer, exit 2, naming it, and records nothing.
func TestAnswersFileReplaysByOrdinalAndRefusesWhenItRunsOut(t *testing.T) {
	routeMachine(t, interview.RoleReflectionComposer)
	interviewStub(t, retroReceipts()...)
	r := retroRepo(t)
	stderr, err := interviewRun(t, false, term.Mono, "reflect", "interview", "v0.2.0", "--proceed", "--answers", retroAnswersFile(t, "Q1"))
	if exitCodeOf(err) != 2 || !strings.Contains(err.Error(), "runs out at Q2 (Product Q2)") || !strings.Contains(err.Error(), `"id": "Q2"`) {
		t.Fatalf("exit %d, err %v\n%s", exitCodeOf(err), err, stderr)
	}
	if strings.ContainsRune(stderr, 0x1b) {
		t.Fatalf("the plain-text questions carry an escape byte:\n%q", stderr)
	}
	if b := oneRecord(t, filepath.Join(r.Root(), filepath.FromSlash(interview.RecordsRel))); b != nil {
		t.Fatalf("a record was written:\n%s", b)
	}
	if _, err := os.Stat(retroPath(r, "v0.2.0")); !os.IsNotExist(err) {
		t.Fatal("a retrospective was written")
	}
}

// TestInterviewOffATerminalWithNoAnswersRunsNothing: with no terminal to
// draw on and no answers file, the interview refuses before any runner
// starts.
func TestInterviewOffATerminalWithNoAnswersRunsNothing(t *testing.T) {
	routeMachine(t, interview.RoleReflectionComposer)
	script := interviewStub(t, retroReceipts()...)
	retroRepo(t)
	_, err := interviewRun(t, false, term.Mono, "reflect", "interview", "v0.2.0", "--proceed")
	if exitCodeOf(err) != 2 || !strings.Contains(err.Error(), "--answers") || !strings.Contains(err.Error(), "nothing was run") {
		t.Fatalf("exit %d, err %v", exitCodeOf(err), err)
	}
	if _, err := os.Stat(filepath.Join(script, "turn-1.brief.seen.md")); !os.IsNotExist(err) {
		t.Fatal("the runner was started")
	}
}

// ownSGRRe is an escape sequence the drawing composes: a colour, and nothing
// else.
var ownSGRRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

// TestRunnerQuestionIsSanitisedBeforeItIsDrawn is B7's runner case: a stub
// runner's ask whose material, label and meaning carry an escape sequence, a
// C1 control, a bidi override, a zero-width space and a bare carriage return
// is drawn in colour with only the drawing's own colour sequences, each
// injected rune drawn as '?'.
func TestRunnerQuestionIsSanitisedBeforeItIsDrawn(t *testing.T) {
	hostile := "\x1b[31m\u009b\u202e\u200b\r"
	routeMachine(t, interview.RoleReflectionComposer)
	interviewStub(t, stubQuestion("Product Q1", "Is that the whole of it?", hostile), `{"done":`+fullAnswers+`}`)
	retroRepo(t)
	stderr, err := interviewRun(t, true, term.Ansi16, "reflect", "interview", "v0.2.0", "--proceed")
	if err != nil {
		t.Fatalf("err = %v\n%s", err, stderr)
	}
	if !ownSGRRe.MatchString(stderr) {
		t.Fatalf("the question was not drawn in colour:\n%q", stderr)
	}
	stripped := ownSGRRe.ReplaceAllString(stderr, "")
	for _, r := range []rune{0x1b, 0x9b, 0x202e, 0x200b, '\r'} {
		if strings.ContainsRune(stripped, r) {
			t.Errorf("U+%04X reached the screen:\n%q", r, stripped)
		}
	}
	for _, part := range []string{"For example", "Keep it", "The answer stands"} {
		if !strings.Contains(stripped, part+"?[31m????") {
			t.Errorf("%q is not drawn with each injected rune as '?':\n%s", part, stripped)
		}
	}
}

// planningRepo is a repository holding one draft and one shipped intent,
// with a local tier, the working directory set to it.
func planningRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write(".abcd/development/intents/drafts/itd-5-a-draft.md",
		"---\nid: itd-5\nslug: a-draft\nkind: standalone\n---\n\n# A draft to plan\n\n## Press Release\n\n> A line.\n\n## Acceptance Criteria\n\n- Given a list, when it is long, then it narrows.\n")
	r.Write(".abcd/development/intents/shipped/itd-6-shipped.md", reflectIntent("itd-6", "A shipped promise", "", reflectEmpty))
	r.Commit("intents")
	if err := os.MkdirAll(filepath.Join(r.Root(), ".abcd", ".work.local"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(r.Root())
	return r
}

func planningAnswersFile(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "answers.json")
	if err := os.WriteFile(p, []byte(`{"schema_version":1,"interview":"planning","answers":[{"id":"Q1","value":"kept"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestIntentInterviewRecordsAndReportsReadiness: the planning interview
// draws the planning-interviewer's question, records the answer, hands the
// role the record's path in its seed, and ends by reporting the readiness
// gate on the record as the role left it.
func TestIntentInterviewRecordsAndReportsReadiness(t *testing.T) {
	routeMachine(t, interview.RolePlanningInterviewer)
	script := interviewStub(t, stubQuestion("Product Q1", "Does the press release stand?", ""), `{"done":{"summary":"The press release stands as written."}}`)
	r := planningRepo(t)
	stdout := tempStream(t, "stdout")
	stderr := tempStream(t, "stderr")
	in, w := pipeWith(t, "")
	_ = w.Close()
	cmd := NewRootCommand()
	cmd.SetIn(in)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"intent", "interview", "itd-5", "--answers", planningAnswersFile(t), "--json"})
	if err := cmd.Execute(); err != nil {
		errText, _ := os.ReadFile(stderr.Name())
		t.Fatalf("err = %v\n%s", err, errText)
	}
	out, _ := os.ReadFile(stdout.Name())
	var res planningInterviewResult
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if res.Intent != "itd-5" || res.Summary != "The press release stands as written." || res.Ready.Bucket != "drafts" ||
		!strings.HasPrefix(res.Record, interview.RecordsRel+"/planning-") {
		t.Fatalf("result %+v", res)
	}
	if rec := readInterviewRecord(t, r.Root()); rec.Interview != interview.Planning || rec.Target != "itd-5" || len(rec.Answers) != 1 {
		t.Fatalf("record %+v", rec)
	}
	brief, err := os.ReadFile(filepath.Join(script, "turn-1.brief.seen.md"))
	if err != nil || !strings.Contains(string(brief), `"path":".abcd/development/intents/drafts/itd-5-a-draft.md"`) {
		t.Fatalf("the seed does not name the record: %v\n%s", err, brief)
	}
}

// TestIntentInterviewRefusesAShippedIntent: the planning interview runs on a
// draft or a planned intent; anything else refuses before anything runs.
func TestIntentInterviewRefusesAShippedIntent(t *testing.T) {
	routeMachine(t, interview.RolePlanningInterviewer)
	script := interviewStub(t, `{"done":{"summary":"x"}}`)
	planningRepo(t)
	for _, id := range []string{"itd-6", "itd-77"} {
		_, err := interviewRun(t, false, term.Mono, "intent", "interview", id, "--answers", planningAnswersFile(t))
		if exitCodeOf(err) != 2 || !strings.Contains(err.Error(), "nothing was run") {
			t.Fatalf("%s: exit %d, err %v", id, exitCodeOf(err), err)
		}
	}
	if _, err := os.Stat(filepath.Join(script, "turn-1.brief.seen.md")); !os.IsNotExist(err) {
		t.Fatal("the runner was started")
	}
}
