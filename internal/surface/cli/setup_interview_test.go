package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/decide"
	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/core/interview"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/surface/cli/ask"
	"github.com/intentdriven/abcd/internal/term"
	"github.com/intentdriven/abcd/internal/term/ptytest"
)

var updateSetupGolden = flag.Bool("update-setup-golden", false, "rewrite the setup question goldens under testdata")

// TestSetupQuestionDrawsAt80Columns is B1's layout half through drawnPrompter:
// the visibility question built from core's help (HelpIn, ChangeLaterLine)
// drawn at 80 columns in Mono, against its golden: the chip, About as the
// material, the ask, each answer numbered with its meaning beneath, decide
// later last, and the change-later line naming the flag.
func TestSetupQuestionDrawsAt80Columns(t *testing.T) {
	p := &drawnPrompter{}
	q := p.value("visibility", []string{"private", "public"}, "")
	got := strings.Join(ask.Layout(question.Ask{Questions: []question.Question{q}}, 80, term.Mono), "\n") + "\n"
	path := filepath.Join("testdata", "setup-visibility-80.golden")
	if *updateSetupGolden {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with -update-setup-golden to write it)", err)
	}
	if got != string(want) {
		t.Fatalf("the setup question differs from its golden:\n--- got\n%s--- want\n%s", got, want)
	}
	h, _ := ahoy.HelpIn("", "visibility")
	for _, part := range []string{"› Setup Q1", "Which answer does this repository take?", "1. private", "3. Decide later", h.ChangeLaterLine()} {
		if !strings.Contains(got, part) {
			t.Errorf("the drawing lacks %q", part)
		}
	}
	if strings.Contains(got, "\x1b") {
		t.Fatal("Mono drew an escape sequence")
	}
}

// TestSetupQuestionAnswersByArrowsAndByNumber is B1's answer half through
// drawnPrompter on a pseudo-terminal: Down then Enter, and "2" then Enter,
// both choose the second answer, and each is recorded as answered in
// Terminal under the question's own id.
func TestSetupQuestionAnswersByArrowsAndByNumber(t *testing.T) {
	pt := ptytest.Open(t, 80, 24)
	env := map[string]string{"TERM": "xterm", "LANG": "en_US.UTF-8"}
	p := &drawnPrompter{
		setupQuestions: setupQuestions{w: pt.Terminal},
		term: ask.Terminal{In: pt.Terminal, Out: pt.Terminal, Getenv: func(k string) string { return env[k] },
			Mode: term.Mono, Roots: layered.Roots{Home: t.TempDir()}},
	}
	ask1 := func(keys string) string {
		t.Helper()
		offset := len(pt.Output())
		got := make(chan string, 1)
		go func() { got <- p.Prompt("visibility", []string{"private", "public"}, "") }()
		pt.WaitFor(t, offset, "3. Decide later", 20*time.Second)
		pt.Type(t, keys)
		select {
		case v := <-got:
			return v
		case <-time.After(20 * time.Second):
			t.Fatalf("no answer; the terminal shows:\n%q", pt.Output()[offset:])
		}
		return ""
	}
	byArrows := ask1("\x1b[B\r")
	byNumber := ask1("2\r")
	if byArrows != "public" || byNumber != "public" {
		t.Fatalf("Down+Enter chose %q and 2+Enter chose %q; both should choose public", byArrows, byNumber)
	}
	rec := p.recorded()
	if len(rec) != 2 || rec[0].ID != "visibility" || rec[1].Ask.Questions[0].Chip != "Setup Q2" {
		t.Fatalf("recorded %+v", rec)
	}
	for _, a := range rec {
		if a.Value != "public" || a.AnsweredIn != interview.Terminal {
			t.Fatalf("recorded %+v", a)
		}
	}
	if !strings.Contains(pt.Output(), "› Setup Q1: public") {
		t.Fatalf("the question did not collapse to its answer:\n%q", pt.Output())
	}
}

// TestSetupLaterLeavesTheValueUnset is the decide-later answer through the
// drawn door: the install gets no answer (so a config value stays unset and
// its gap listed), and the record says "later".
func TestSetupLaterLeavesTheValueUnset(t *testing.T) {
	in, w := pipeWith(t, "3\n")
	_ = w
	p := &drawnPrompter{
		setupQuestions: setupQuestions{w: &bytes.Buffer{}},
		term: ask.Terminal{In: in, Out: &bytes.Buffer{}, Getenv: func(k string) string {
			return map[string]string{"ABCD_ACCESSIBLE": "1"}[k]
		}, Mode: term.Mono, Roots: layered.Roots{Home: t.TempDir()}},
	}
	if v := p.Prompt("visibility", []string{"private", "public"}, ""); v != "" {
		t.Fatalf("decide later answered %q", v)
	}
	if rec := p.recorded(); len(rec) != 1 || rec[0].Value != ahoy.SetupLaterValue {
		t.Fatalf("recorded %+v", rec)
	}
}

// pipeWith is a pipe already holding s, its write end left open for more.
func pipeWith(t *testing.T, s string) (*os.File, *os.File) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close(); _ = w.Close() })
	if _, err := w.WriteString(s); err != nil {
		t.Fatal(err)
	}
	return r, w
}

// setupRun is one hermetic install: its own home and a repository named
// "proj", so two runs write the same project name into their settings.
type setupRun struct {
	home, repo string
}

func newSetupRun(t *testing.T) setupRun {
	t.Helper()
	home := t.TempDir()
	pluginRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(pluginRoot, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginRoot, "hooks", "hooks.json"), []byte(validHooksJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginRoot, "abcd"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", pluginRoot)
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("ABCD_BIN_TARGET", filepath.Join(t.TempDir(), "bin", "abcd"))
	provisionHermeticCache(t, home)
	repo := filepath.Join(t.TempDir(), "proj")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)
	// The drain rule is recorded already: its offer is put only to a person at
	// a terminal, so a repository without it asks the drawn run one question
	// more than its replay, and their records could not match.
	if _, err := decide.CreateStated(repo, decide.Stated{
		Title: drainrule.ProposalTitle, Frontmatter: drainrule.ProposalFrontmatter(), Body: drainrule.ProposalBody(),
	}); err != nil {
		t.Fatal(err)
	}
	return setupRun{home: home, repo: repo}
}

// choose is the answer a test gives a setup question: approve every kind of
// change but installing tools, take the first value, install the status
// line with every element on, accept the machine's routing and not the
// repository's, and decide anything else later.
func choose(q question.Question) string {
	switch {
	case q.ID == "approve.dependency", q.ID == ahoy.OracleRoutingRepoGapID:
		return "no"
	case strings.HasPrefix(q.ID, "approve."), q.ID == "adopt", q.ID == ahoy.StatusLineOfferGapID, q.ID == ahoy.OracleRoutingMachineGapID:
		return "yes"
	case strings.HasPrefix(q.ID, "statusline."):
		return "on"
	case q.ID == "visibility":
		return "private"
	case q.ID == "docs_target":
		return "agents_md"
	case len(q.Options) > 0 && !strings.HasPrefix(q.ID, "Q"):
		return q.Options[0].Value
	}
	return ahoy.SetupLaterValue
}

// number is the numbered reader's line choosing value in q.
func number(q question.Question, value string) string {
	for i, o := range append(append([]question.Option(nil), q.Options...), q.Later) {
		if o.Value == value {
			return strconv.Itoa(i + 1)
		}
	}
	return "0"
}

// drawnSetup runs the install through the drawn door, the answer loop reading
// a key stream the test types into it, one line per question as it is put
// (the numbered reader: ABCD_ACCESSIBLE), with every stream standing in for
// a terminal. It returns stderr, where the questions are drawn.
func drawnSetup(t *testing.T, args ...string) (string, error) {
	t.Helper()
	t.Setenv("ABCD_ACCESSIBLE", "1")
	in, keys := pipeWith(t, "")
	stdout, stderr := tempStream(t, "stdout"), tempStream(t, "stderr")
	swapStream, swapPut := isTerminalStream, drawnPut
	t.Cleanup(func() { isTerminalStream, drawnPut = swapStream, swapPut })
	isTerminalStream = func(*os.File) bool { return true }
	drawnPut = func(tm ask.Terminal, a question.Ask) ([]ask.Answer, error) {
		q := a.Questions[0]
		if _, err := fmt.Fprintln(keys, number(q, choose(q))); err != nil {
			return nil, err
		}
		return tm.Put(a)
	}
	cmd := NewRootCommand()
	cmd.SetIn(in)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs(append([]string{"ahoy", "install"}, args...))
	err := cmd.Execute()
	got, _ := os.ReadFile(stderr.Name())
	return string(got), err
}

// pipedSetup runs the install with every stream a pipe or a file, never a
// terminal: stdin empty, answered from the answers file at answers. It
// returns stderr, where the questions are written as plain text.
func pipedSetup(t *testing.T, answers string, args ...string) (string, error) {
	t.Helper()
	in, w := pipeWith(t, "")
	_ = w.Close()
	stdout, stderr := tempStream(t, "stdout"), tempStream(t, "stderr")
	cmd := NewRootCommand()
	cmd.SetIn(in)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs(append([]string{"ahoy", "install", "--answers", answers}, args...))
	err := cmd.Execute()
	got, _ := os.ReadFile(stderr.Name())
	return string(got), err
}

func tempStream(t *testing.T, name string) *os.File {
	t.Helper()
	f, err := os.Create(filepath.Join(t.TempDir(), name))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

// oneRecord reads the one record in dir, or "" when dir holds none.
func oneRecord(t *testing.T, dir string) []byte {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 1 {
		t.Fatalf("%s holds %d records, want one", dir, len(ents))
	}
	b, err := os.ReadFile(filepath.Join(dir, ents[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// records reads a run's two records: the repository's and the machine's.
func (r setupRun) records(t *testing.T) (repo, machine []byte) {
	return oneRecord(t, filepath.Join(r.repo, filepath.FromSlash(interview.RecordsRel))),
		oneRecord(t, filepath.Join(r.home, ".abcd", "interviews"))
}

func (r setupRun) config(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(r.repo, ".abcd", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// answersFrom writes an answers file replaying the records' answers by id,
// stamped answeredIn ("" names no place, so the flag stamps it), leaving out
// the ids in drop.
func answersFrom(t *testing.T, answeredIn string, drop map[string]bool, recs ...[]byte) string {
	t.Helper()
	f := interview.Answers{SchemaVersion: interview.SchemaVersion, Interview: "setup"}
	for _, b := range recs {
		var r interview.Record
		if err := json.Unmarshal(b, &r); err != nil {
			t.Fatal(err)
		}
		for _, a := range r.Answers {
			if !drop[a.ID] {
				f.Answers = append(f.Answers, interview.FileAnswer{ID: a.ID, Value: a.Value, AnsweredIn: answeredIn})
			}
		}
	}
	body, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "answers.json")
	if err := os.WriteFile(p, body, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// machineStream finds what a machine-consumed stream must never carry
// (adr-49): an ESC, a C1 control, or a cursor sequence.
var machineStream = regexp.MustCompile("\x1b|[\u0080-\u009f]|\\[[0-9;]*[ABCDHJK]")

// TestInterviewOffATerminalIsPlainAndMatchesTheTerminalRecord is B3 for setup:
// the install answered through the drawn door, then replayed in a second
// repository and home from an answers file carrying the same choices, every
// stream a pipe. The piped run writes each question as plain text with no
// escape, C1 or cursor byte, and its answers records (the repository's and
// the machine's) and the configuration it writes are byte-identical to the
// drawn run's.
func TestInterviewOffATerminalIsPlainAndMatchesTheTerminalRecord(t *testing.T) {
	drawn := newSetupRun(t)
	drawnOut, err := drawnSetup(t)
	if err != nil {
		t.Fatalf("drawn install: %v\n%s", err, drawnOut)
	}
	dRepo, dMachine := drawn.records(t)
	if dRepo == nil || dMachine == nil {
		t.Fatalf("the drawn run left the records repo=%q machine=%q\n%s", dRepo, dMachine, drawnOut)
	}
	// The routing confirmation is drawn too: the proposal in counts, one
	// paragraph, then the question on its own line.
	plain := regexp.MustCompile("\x1b\\[[0-9;]*m").ReplaceAllString(drawnOut, "")
	if !regexp.MustCompile(`(?s)› Setup Q\d+\n\n  abcd proposes a model tier.*\d+ agents: .*\n\n  Accept abcd's proposed routing for this machine\?\n`).MatchString(plain) {
		t.Fatalf("the routing confirmation was not drawn in counts:\n%s", drawnOut)
	}

	piped := newSetupRun(t)
	pipedOut, err := pipedSetup(t, answersFrom(t, "", nil, dRepo, dMachine))
	if err != nil {
		t.Fatalf("piped install: %v\n%s", err, pipedOut)
	}
	if m := machineStream.FindString(pipedOut); m != "" {
		t.Fatalf("the piped run wrote %q to a machine stream:\n%s", m, pipedOut)
	}
	if !strings.Contains(pipedOut, "Setup Q1") || !strings.Contains(pipedOut, "Adopt this unmanaged repo into abcd?") {
		t.Fatalf("the piped run did not write its questions as plain text:\n%s", pipedOut)
	}
	pRepo, pMachine := piped.records(t)
	if !bytes.Equal(dRepo, pRepo) {
		t.Fatalf("the repository records differ:\n--- drawn\n%s\n--- piped\n%s", dRepo, pRepo)
	}
	if !bytes.Equal(dMachine, pMachine) {
		t.Fatalf("the machine records differ:\n--- drawn\n%s\n--- piped\n%s", dMachine, pMachine)
	}
	if !bytes.Equal(drawn.config(t), piped.config(t)) {
		t.Fatalf("the configurations differ:\n--- drawn\n%s\n--- piped\n%s", drawn.config(t), piped.config(t))
	}
	var rec interview.Record
	if err := json.Unmarshal(dRepo, &rec); err != nil || rec.Answers[0].ID != "adopt" || rec.Answers[0].AnsweredIn != interview.Terminal {
		t.Fatalf("drawn record %+v, %v", rec, err)
	}
}

// TestRecordsDifferOnlyInWhereAnswered is B5 for setup: the drawn run, and
// the host path's run (an answers file whose entries name no place, with
// --answered-in "Claude Code", as the plugin page runs it), write records
// equal field by field but for answered_in, and the same configuration.
func TestRecordsDifferOnlyInWhereAnswered(t *testing.T) {
	drawn := newSetupRun(t)
	if out, err := drawnSetup(t); err != nil {
		t.Fatalf("drawn install: %v\n%s", err, out)
	}
	dRepo, dMachine := drawn.records(t)

	host := newSetupRun(t)
	if out, err := pipedSetup(t, answersFrom(t, "", nil, dRepo, dMachine), "--answered-in", "Claude Code"); err != nil {
		t.Fatalf("host-path install: %v\n%s", err, out)
	}
	hRepo, hMachine := host.records(t)
	for _, pair := range [][2][]byte{{dRepo, hRepo}, {dMachine, hMachine}} {
		var d, h interview.Record
		if err := json.Unmarshal(pair[0], &d); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(pair[1], &h); err != nil {
			t.Fatal(err)
		}
		if len(d.Answers) == 0 || len(d.Answers) != len(h.Answers) {
			t.Fatalf("drawn %d answers, host %d", len(d.Answers), len(h.Answers))
		}
		for i := range d.Answers {
			if d.Answers[i].AnsweredIn != interview.Terminal || h.Answers[i].AnsweredIn != interview.ClaudeCode {
				t.Fatalf("answer %d answered in %q and %q", i, d.Answers[i].AnsweredIn, h.Answers[i].AnsweredIn)
			}
			d.Answers[i].AnsweredIn, h.Answers[i].AnsweredIn = "", ""
		}
		db, _ := json.Marshal(d)
		hb, _ := json.Marshal(h)
		if !bytes.Equal(db, hb) {
			t.Fatalf("the records differ beyond answered_in:\n%s\n%s", db, hb)
		}
	}
	if !bytes.Equal(drawn.config(t), host.config(t)) {
		t.Fatal("the configurations differ")
	}
}

// TestUnansweredSetupQuestionRefusesNamingIt is B3's refusal: off a terminal,
// a question neither a flag nor the answers file answers ends the run with
// exit 2, naming the question's id, the flag that would answer it and the
// answers-file key, and no answers record is written. The adoption question,
// asked before any write, leaves the repository and the home untouched.
func TestUnansweredSetupQuestionRefusesNamingIt(t *testing.T) {
	run := newSetupRun(t)
	before := listTree(t, run.repo)
	answers := answersFrom(t, "", nil)
	out, err := pipedSetup(t, answers)
	var ee *exitError
	if !errors.As(err, &ee) || ee.Code != 2 {
		t.Fatalf("an unanswered question: %v, want exit 2\n%s", err, out)
	}
	for _, part := range []string{"Setup Q1 (adopt) has no answer", "--adopt or --refuse-adopt", `{"id": "adopt"`} {
		if !strings.Contains(ee.Msg, part) {
			t.Errorf("the refusal %q lacks %q", ee.Msg, part)
		}
	}
	if after := listTree(t, run.repo); after != before {
		t.Fatalf("the refused run wrote into the repository:\n--- before\n%s--- after\n%s", before, after)
	}
	if _, err := os.Stat(filepath.Join(run.home, ".abcd", "interviews")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the refused run made the home's interviews: %v", err)
	}

	// A value question names its flag and the values the file may give it.
	out, err = pipedSetup(t, answers, "--adopt")
	if !errors.As(err, &ee) || ee.Code != 2 {
		t.Fatalf("an unanswered approval: %v\n%s", err, out)
	}
	if !strings.Contains(ee.Msg, "(approve.") || !strings.Contains(ee.Msg, "--yes") {
		t.Fatalf("the refusal %q names no approval and its flag", ee.Msg)
	}
	out, err = pipedSetup(t, answers, "--adopt", "--yes")
	if !errors.As(err, &ee) || ee.Code != 2 {
		t.Fatalf("an unanswered value: %v\n%s", err, out)
	}
	if !strings.Contains(ee.Msg, "(visibility) has no answer: pass --visibility <private|public>") ||
		!strings.Contains(ee.Msg, `{"id": "visibility", "value": "<private|public|later>"}`) {
		t.Fatalf("the refusal %q does not name the visibility question, its flag and its key", ee.Msg)
	}
	if rec := oneRecord(t, filepath.Join(run.repo, filepath.FromSlash(interview.RecordsRel))); rec != nil {
		t.Fatalf("a refused run wrote an answers record:\n%s", rec)
	}
}

// TestSetupAnswersFlagsAreChecked refuses an --answered-in that is neither
// place, and an answers file jsonstrict refuses, before anything is asked.
func TestSetupAnswersFlagsAreChecked(t *testing.T) {
	run := newSetupRun(t)
	before := listTree(t, run.repo)
	good := answersFrom(t, "", nil)
	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte(`{"schema_version":1,"interview":"setup","answers":[],"answers":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, args := range map[string][]string{
		"answered elsewhere": {"--answered-in", "a browser"},
		"duplicate key":      {"--answers", bad},
	} {
		out, err := pipedSetup(t, good, args...)
		var ee *exitError
		if !errors.As(err, &ee) || ee.Code != 2 {
			t.Errorf("%s: %v, want exit 2\n%s", name, err, out)
		}
	}
	if after := listTree(t, run.repo); after != before {
		t.Fatalf("a refused flag wrote into the repository")
	}
}

// listTree lists every path under root, for a nothing-written assertion.
func listTree(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil {
			rel, _ := filepath.Rel(root, p)
			fmt.Fprintf(&b, "%s %d\n", rel, info.Size())
		}
		return nil
	})
	return b.String()
}

// TestSetupInterruptKeepsTheAnswersGiven is Ctrl-C at a drawn question, the
// last setup asks: the run ends with exit 130, the answers given before it
// are recorded (the repository's and the machine's), and nothing is recorded
// for the question it interrupted.
func TestSetupInterruptKeepsTheAnswersGiven(t *testing.T) {
	run := newSetupRun(t)
	var at string
	swap := drawnPut
	t.Cleanup(func() { drawnPut = swap })
	put := func(tm ask.Terminal, a question.Ask) ([]ask.Answer, error) {
		q := a.Questions[0]
		if q.ID == "artefact_kind" {
			at = q.ID
			return nil, ask.ErrInterrupted
		}
		return swap(tm, a)
	}
	t.Setenv("ABCD_ACCESSIBLE", "1")
	in, keys := pipeWith(t, "")
	stdout, stderr := tempStream(t, "stdout"), tempStream(t, "stderr")
	swapStream := isTerminalStream
	t.Cleanup(func() { isTerminalStream = swapStream })
	isTerminalStream = func(*os.File) bool { return true }
	drawnPut = func(tm ask.Terminal, a question.Ask) ([]ask.Answer, error) {
		q := a.Questions[0]
		if _, err := fmt.Fprintln(keys, number(q, choose(q))); err != nil {
			return nil, err
		}
		return put(tm, a)
	}
	cmd := NewRootCommand()
	cmd.SetIn(in)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"ahoy", "install"})
	err := cmd.Execute()
	var ee *exitError
	if !errors.As(err, &ee) || ee.Code != ask.ExitInterrupted {
		t.Fatalf("Ctrl-C: %v, want exit %d", err, ask.ExitInterrupted)
	}
	if at == "" {
		t.Fatal("the artefact kind was not asked; the interrupt never happened")
	}
	repo, machine := run.records(t)
	var r, m interview.Record
	if err := json.Unmarshal(repo, &r); err != nil || len(r.Answers) == 0 || r.Answers[0].ID != "adopt" {
		t.Fatalf("the repository's answers were not kept: %v\n%s", err, repo)
	}
	if err := json.Unmarshal(machine, &m); err != nil {
		t.Fatalf("the machine's answers were not kept: %v\n%s", err, machine)
	}
	for _, a := range append(r.Answers, m.Answers...) {
		if a.ID == at {
			t.Fatalf("the interrupted question %s was recorded", at)
		}
	}
	if len(m.Answers) != 1 || m.Answers[0].ID != ahoy.OracleRoutingMachineGapID {
		t.Fatalf("machine record %+v", m.Answers)
	}
}

// TestStoppedQuestionEndsTheVerb holds the other verbs that ask through the
// drawn door (ahoy remote apply, site setup) to the same ending: a question
// stopped by Ctrl-C ends the verb with its exit code, never a crash, and any
// other panic is left to crash.
func TestStoppedQuestionEndsTheVerb(t *testing.T) {
	verb := func(stop any) (err error) {
		defer endOnStop(&err)
		panic(stop)
	}
	var ee *exitError
	if err := verb(&setupStop{code: ask.ExitInterrupted, msg: "interrupted"}); !errors.As(err, &ee) || ee.Code != ask.ExitInterrupted {
		t.Fatalf("a stopped question ended the verb with %v", err)
	}
	defer func() {
		if r := recover(); r != "not a stop" {
			t.Fatalf("another panic was swallowed: %v", r)
		}
	}()
	_ = verb("not a stop")
}
