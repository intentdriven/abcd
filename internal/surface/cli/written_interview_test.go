package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

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
// receipt), keeps the turn's brief and its argv beside the script, and
// reports success. A turn-<n>.also.json, a map of path to content, makes the
// turn write those files too, relative to the directory the runner was
// started in; a turn-<n>.sleep makes it mark turn-<n>.started beside the
// script and sleep, so a test can stop it mid-run.
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
	if argv, err := json.Marshal(os.Args[1:]); err == nil {
		_ = os.WriteFile(filepath.Join(dir, turn+".argv.json"), argv, 0o600)
	}
	if body, err := os.ReadFile(filepath.Join(dir, turn+".json")); err == nil {
		if err := os.WriteFile(receipt, body, 0o600); err != nil {
			return 3
		}
	}
	if raw, err := os.ReadFile(filepath.Join(dir, turn+".also.json")); err == nil {
		var also map[string]string
		if err := json.Unmarshal(raw, &also); err != nil {
			return 3
		}
		for p, body := range also {
			target := filepath.FromSlash(p)
			if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
				return 3
			}
			if err := os.WriteFile(target, []byte(body), 0o600); err != nil {
				return 3
			}
		}
	}
	if _, err := os.Stat(filepath.Join(dir, turn+".sleep")); err == nil {
		_ = os.WriteFile(filepath.Join(dir, turn+".started"), nil, 0o600)
		time.Sleep(30 * time.Second)
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
	record := ".abcd/development/intents/drafts/itd-5-a-draft.md"
	interviewStubAlso(t, script, 1, map[string]string{record: "---\nid: itd-5\nslug: a-draft\nkind: standalone\n---\n\n# A draft to plan\n\n## Press Release\n\n> A line the person confirmed.\n\n## Acceptance Criteria\n\n- Given a list, when it is long, then it narrows.\n"})
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
		!strings.HasPrefix(res.Record, interview.RecordsRel+"/planning-") || !reflect.DeepEqual(res.ChangedPaths, []string{record}) {
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

// interviewStubAlso makes the script's turn write files besides its receipt,
// each path relative to the repository the runner runs in.
func interviewStubAlso(t *testing.T, script string, turn int, files map[string]string) {
	t.Helper()
	b, err := json.Marshal(files)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(script, fmt.Sprintf("turn-%d.also.json", turn)), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestInterruptWhileARunnerWritesExits130: an interrupt while the second
// turn's runner writes exits 130, keeps the first answer in the record, and
// records no runner failure; nothing is filed.
func TestInterruptWhileARunnerWritesExits130(t *testing.T) {
	routeMachine(t, interview.RoleReflectionComposer)
	script := interviewStub(t, retroReceipts()...)
	if err := os.WriteFile(filepath.Join(script, "turn-2.sleep"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	swap := interruptContext
	t.Cleanup(func() { interruptContext = swap })
	n := 0
	interruptContext = func(ctx context.Context) (context.Context, context.CancelFunc) {
		n++
		dctx, cancel := context.WithCancel(ctx)
		if n == 2 {
			go func() {
				for dctx.Err() == nil {
					if _, err := os.Stat(filepath.Join(script, "turn-2.started")); err == nil {
						cancel()
						return
					}
					time.Sleep(10 * time.Millisecond)
				}
			}()
		}
		return dctx, cancel
	}
	r := retroRepo(t)
	stderr, err := interviewRun(t, false, term.Mono, "reflect", "interview", "v0.2.0", "--proceed", "--answers", retroAnswersFile(t, "Q1", "Q2"))
	if exitCodeOf(err) != ask.ExitInterrupted {
		t.Fatalf("exit %d, err %v\n%s", exitCodeOf(err), err, stderr)
	}
	if rec := readInterviewRecord(t, r.Root()); len(rec.Answers) != 1 || len(rec.Fallbacks) != 0 {
		t.Fatalf("record %+v; want the first answer and no fallback", rec)
	}
	if _, err := os.Stat(retroPath(r, "v0.2.0")); !os.IsNotExist(err) {
		t.Fatal("a retrospective was written")
	}
}

// TestAnswersFileValueNotOfferedRefusesRecordingNothing: an answers-file
// entry whose value the question does not offer refuses at the front door,
// exit 2, naming the values it offers, and writes no answers record.
func TestAnswersFileValueNotOfferedRefusesRecordingNothing(t *testing.T) {
	routeMachine(t, interview.RoleReflectionComposer)
	interviewStub(t, retroReceipts()...)
	r := retroRepo(t)
	p := filepath.Join(t.TempDir(), "answers.json")
	if err := os.WriteFile(p, []byte(`{"schema_version":1,"interview":"retrospective","answers":[{"id":"Q1","value":"nope"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	stderr, err := interviewRun(t, false, term.Mono, "reflect", "interview", "v0.2.0", "--proceed", "--answers", p)
	if exitCodeOf(err) != 2 || !strings.Contains(err.Error(), `"nope"`) || !strings.Contains(err.Error(), "kept|more|later") {
		t.Fatalf("exit %d, err %v\n%s", exitCodeOf(err), err, stderr)
	}
	if b := oneRecord(t, filepath.Join(r.Root(), filepath.FromSlash(interview.RecordsRel))); b != nil {
		t.Fatalf("a record was written:\n%s", b)
	}
}

// TestReflectInterviewWriterFaultExitsOneKeepingTheRecord: a fault of the
// retrospective's writer that is not a refusal the person answers (here the
// retrospectives directory cannot be written in) exits 1, since the answers
// record was written, never 2, which promises nothing was.
func TestReflectInterviewWriterFaultExitsOneKeepingTheRecord(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("a directory's mode does not stop root writing in it")
	}
	routeMachine(t, interview.RoleReflectionComposer)
	interviewStub(t, retroReceipts()...)
	r := retroRepo(t)
	shut := filepath.Dir(filepath.Dir(retroPath(r, "v0.2.0")))
	if err := os.MkdirAll(shut, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(shut, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(shut, 0o755) })
	stderr, err := interviewRun(t, false, term.Mono, "reflect", "interview", "v0.2.0", "--proceed", "--answers", retroAnswersFile(t, "Q1", "Q2"))
	if exitCodeOf(err) != 1 {
		t.Fatalf("exit %d, err %v\n%s", exitCodeOf(err), err, stderr)
	}
	if rec := readInterviewRecord(t, r.Root()); len(rec.Answers) != 2 {
		t.Fatalf("record %+v", rec)
	}
}

// planningAnswersFileQ1Q2 answers the planning interview's first two
// questions.
func planningAnswersFileQ1Q2(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "answers.json")
	if err := os.WriteFile(p, []byte(`{"schema_version":1,"interview":"planning","answers":[{"id":"Q1","value":"kept"},{"id":"Q2","value":"kept"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestIntentInterviewRoleChangingAnotherFileIsRefused: the planning role may
// change the intent's record and nothing else. A turn that also writes a
// file elsewhere in the repository (a hook script a contributed criterion
// asked for) stops the interview, exit 1, naming it, with the answer given
// before it recorded and no readiness reported; under --json the refusal
// carries every path the role changed.
func TestIntentInterviewRoleChangingAnotherFileIsRefused(t *testing.T) {
	for _, asJSON := range []bool{false, true} {
		t.Run(fmt.Sprintf("json=%v", asJSON), func(t *testing.T) {
			routeMachine(t, interview.RolePlanningInterviewer)
			script := interviewStub(t, stubQuestion("Product Q1", "Does the press release stand?", ""),
				stubQuestion("Product Q2", "Does the first criterion stand?", ""), `{"done":{"summary":"x"}}`)
			r := planningRepo(t)
			record := ".abcd/development/intents/drafts/itd-5-a-draft.md"
			interviewStubAlso(t, script, 1, map[string]string{record: "---\nid: itd-5\nslug: a-draft\nkind: standalone\n---\n\n# A draft to plan\n"})
			interviewStubAlso(t, script, 2, map[string]string{".githooks/pre-push": "#!/bin/sh\nexit 0\n"})
			stdout, stderr := tempStream(t, "stdout"), tempStream(t, "stderr")
			in, w := pipeWith(t, "")
			_ = w.Close()
			cmd := NewRootCommand()
			cmd.SetIn(in)
			cmd.SetOut(stdout)
			cmd.SetErr(stderr)
			args := []string{"intent", "interview", "itd-5", "--answers", planningAnswersFileQ1Q2(t)}
			if asJSON {
				args = append(args, "--json")
			}
			cmd.SetArgs(args)
			err := cmd.Execute()
			errText, _ := os.ReadFile(stderr.Name())
			out, _ := os.ReadFile(stdout.Name())
			if exitCodeOf(err) != 1 {
				t.Fatalf("exit %d, err %v\n%s", exitCodeOf(err), err, errText)
			}
			if asJSON {
				var ref planningChangesRefusal
				if jerr := json.Unmarshal(out, &ref); jerr != nil {
					t.Fatalf("%v\n%s", jerr, out)
				}
				if ref.Refused != "unexpected_changes" || !reflect.DeepEqual(ref.ChangedPaths, []string{record, ".githooks/pre-push"}) ||
					!strings.Contains(ref.Message, ".githooks/pre-push") || !strings.HasPrefix(ref.Record, interview.RecordsRel+"/planning-") {
					t.Fatalf("refusal %+v", ref)
				}
			} else {
				if !strings.Contains(err.Error(), ".githooks/pre-push") || strings.Contains(err.Error(), record) {
					t.Fatalf("the refusal does not name the one ungranted path alone: %v", err)
				}
				if strings.Contains(string(out), "READY") {
					t.Fatalf("readiness was reported after the refusal:\n%s", out)
				}
			}
			if rec := readInterviewRecord(t, r.Root()); len(rec.Answers) != 1 {
				t.Fatalf("record %+v", rec)
			}
		})
	}
}

// TestIntentInterviewRoleWritingWhereGitOrAPushRunsFromIsRefused: what git
// status does not list is watched too. A planning role that plants a git
// hook, edits the git configuration, writes a push receipt into the local
// tier or a sibling worktree's, repoints a sibling worktree's common
// directory, or writes a gitignored file stops the interview, exit 1, naming
// the path, with the answer given before it recorded and no readiness
// reported.
func TestIntentInterviewRoleWritingWhereGitOrAPushRunsFromIsRefused(t *testing.T) {
	// laneToken in a case's path or body is the sibling worktree's path.
	const laneToken = "{lane}"
	for _, c := range []struct{ name, path, body string }{
		{"hook", ".git/hooks/pre-commit", "#!/bin/sh\nexit 0\n"},
		{"config", ".git/config", "[core]\n\trepositoryformatversion = 0\n[alias]\n\tst = !sh -c true\n"},
		{"receipt", ".abcd/.work.local/preflight-receipts/0123456789abcdef0123456789abcdef01234567", "ok\n"},
		{"ignored", "build.log", "planted\n"},
		{"sibling receipt", laneToken + "/.abcd/.work.local/preflight-receipts/0123456789abcdef0123456789abcdef01234567", "ok\n"},
		{"sibling commondir", ".git/worktrees/lane/commondir", laneToken + "-fake\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			routeMachine(t, interview.RolePlanningInterviewer)
			script := interviewStub(t, stubQuestion("Product Q1", "Does the press release stand?", ""),
				stubQuestion("Product Q2", "Does the first criterion stand?", ""), `{"done":{"summary":"x"}}`)
			r := planningRepo(t)
			r.Write(".gitignore", "*.log\n.abcd/.work.local/\n")
			r.Commit("ignore")
			lane := filepath.Join(t.TempDir(), "lane")
			r.Git("worktree", "add", "-q", "-b", "lane", lane)
			c.path = strings.ReplaceAll(c.path, laneToken, filepath.ToSlash(lane))
			c.body = strings.ReplaceAll(c.body, laneToken, filepath.ToSlash(lane))
			interviewStubAlso(t, script, 2, map[string]string{c.path: c.body})
			stdout, stderr := tempStream(t, "stdout"), tempStream(t, "stderr")
			in, w := pipeWith(t, "")
			_ = w.Close()
			cmd := NewRootCommand()
			cmd.SetIn(in)
			cmd.SetOut(stdout)
			cmd.SetErr(stderr)
			cmd.SetArgs([]string{"intent", "interview", "itd-5", "--answers", planningAnswersFileQ1Q2(t)})
			err := cmd.Execute()
			errText, _ := os.ReadFile(stderr.Name())
			out, _ := os.ReadFile(stdout.Name())
			if exitCodeOf(err) != 1 {
				t.Fatalf("exit %d, err %v\n%s", exitCodeOf(err), err, errText)
			}
			if !strings.Contains(err.Error(), c.path) {
				t.Fatalf("the refusal does not name %s: %v", c.path, err)
			}
			if strings.Contains(string(out), "READY") {
				t.Fatalf("readiness was reported after the refusal:\n%s", out)
			}
			if rec := readInterviewRecord(t, r.Root()); len(rec.Answers) != 1 {
				t.Fatalf("record %+v", rec)
			}
		})
	}
}

// TestInterviewRolesAreGrantedReadForTheirBrief: the runner's prompt tells
// the role to read its turn's brief, and the claude runner denies every tool
// its launch does not grant (dontAsk), so both roles are launched with Read
// granted beside Write for the receipt: the retrospective's role with those
// two alone, the planning role with its contract's tools as well.
func TestInterviewRolesAreGrantedReadForTheirBrief(t *testing.T) {
	for _, c := range []struct {
		name, role, want string
		args             []string
		answers          func(*testing.T) string
		setup            func(*testing.T)
	}{
		{"retrospective", interview.RoleReflectionComposer, "--allowedTools=Read,Write",
			[]string{"reflect", "interview", "v0.2.0", "--proceed"}, func(t *testing.T) string { return retroAnswersFile(t, "Q1", "Q2") }, func(t *testing.T) { retroRepo(t) }},
		{"planning", interview.RolePlanningInterviewer, "--allowedTools=Read,Edit,Grep,Glob,Write",
			[]string{"intent", "interview", "itd-5"}, planningAnswersFile, func(t *testing.T) { planningRepo(t) }},
	} {
		t.Run(c.name, func(t *testing.T) {
			routeMachine(t, c.role)
			script := interviewStub(t, `{"done":{"summary":"nothing to ask"}}`)
			c.setup(t)
			_, _ = interviewRun(t, false, term.Mono, append(c.args, "--answers", c.answers(t))...)
			raw, err := os.ReadFile(filepath.Join(script, "turn-1.argv.json"))
			if err != nil {
				t.Fatalf("the runner was not started: %v", err)
			}
			var argv []string
			if err := json.Unmarshal(raw, &argv); err != nil {
				t.Fatal(err)
			}
			if !slices.Contains(argv, c.want) {
				t.Fatalf("argv %q does not grant %s", argv, c.want)
			}
		})
	}
}
