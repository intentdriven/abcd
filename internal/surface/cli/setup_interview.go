package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/interview"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/surface/cli/ask"
	"github.com/intentdriven/abcd/internal/term"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The setup interview's two recording front doors (spc-2610030911534855,
// "The interviews whose questions abcd fixes"): drawnPrompter draws each
// question at a terminal through the answer loop, and answersPrompter writes
// each one plainly and answers it from an answers file. Both build the
// question from core's words (ahoy.SetupValueQuestion,
// ahoy.SetupConfirmQuestion), number it in the order setup asks, and record
// it; the record is written through interview.Write once the run ends.

// setupInterviewName names the setup interview in its answers record and in
// its answers file.
const setupInterviewName = "setup"

// isTerminalStream reports whether f is a terminal. Tests stand a pipe in for
// one, so the drawn door can be driven by a key stream they write.
var isTerminalStream = term.IsTerminal

// drawnPut puts a question at the terminal; tests wrap it to type the answer
// into the answer loop's input just before it is read.
var drawnPut = func(t ask.Terminal, a question.Ask) ([]ask.Answer, error) { return t.Put(a) }

// drawnTerminal is the terminal the drawn door asks at, when stdin, stdout
// and stderr are all terminals (adr-49 decision 1, scope condition 4): the
// keyboard on stdin, the question drawn on stderr so a --json result on
// stdout stays clean. ok is false anywhere else.
func drawnTerminal(in io.Reader, out, errOut io.Writer, cwd string) (ask.Terminal, []string, bool) {
	inF, ok1 := in.(*os.File)
	outF, ok2 := out.(*os.File)
	errF, ok3 := errOut.(*os.File)
	if !ok1 || !ok2 || !ok3 || !isTerminalStream(inF) || !isTerminalStream(outF) || !isTerminalStream(errF) {
		return ask.Terminal{}, nil, false
	}
	roots, notes := layered.RootsFor(cwd)
	return ask.Terminal{
		In:     inF,
		Out:    errF,
		Getenv: os.Getenv,
		Mode:   term.ResolveColorMode(os.Getenv, false),
		ASCII:  !term.UTF8Locale(os.Getenv),
		Roots:  roots,
	}, notes, true
}

// setupQuestions is what both recording doors share: the numbering, the
// question built from core's words, the --yes line, and the answers given.
type setupQuestions struct {
	cwd string
	w   io.Writer
	n   int
	// yesApproved is a run under --yes, which approves each kind of change and
	// chooses no value; yesTold that it has said so above the first value
	// question it still asks (ahoy.YesStillAsksValues).
	yesApproved, yesTold bool
	answers              []interview.Answer
}

// value is the next question, a value question.
func (s *setupQuestions) value(key string, choices []string, def string) question.Question {
	if h, ok := ahoy.HelpIn(s.cwd, key); ok && h.Flag != "" && s.yesApproved && !s.yesTold {
		s.yesTold = true
		fmt.Fprintf(s.w, "\n%s\n", ahoy.YesStillAsksValues)
	}
	s.n++
	return ahoy.SetupValueQuestion(s.n, s.cwd, key, choices, def)
}

// confirm is the next question, an approval. A text setup does not own has no
// id of its own, so it takes its ordinal, as an AI-written interview's
// question does.
func (s *setupQuestions) confirm(text string) question.Question {
	s.n++
	q := ahoy.SetupConfirmQuestion(s.n, text)
	if q.ID == "" {
		q.ID = fmt.Sprintf("Q%d", s.n)
	}
	return q
}

// record keeps the answer to q: the question as asked, sanitised, the value,
// the note, and where it was answered.
func (s *setupQuestions) record(q question.Question, value, note, answeredIn string) {
	s.answers = append(s.answers, interview.Answer{
		ID:         q.ID,
		Ask:        ask.Safe(question.Ask{Questions: []question.Question{q}}),
		Value:      value,
		Note:       note,
		AnsweredIn: answeredIn,
	})
}

// recorded returns the answers given so far.
func (s *setupQuestions) recorded() []interview.Answer { return s.answers }

// recordingPrompter is a prompter that records the setup interview.
type recordingPrompter interface {
	ahoy.Prompter
	recorded() []interview.Answer
}

// setupStop ends the setup interview from inside a question, which the
// Prompter seam cannot return an error through: the install's front door
// recovers it (runSetup) and ends the run with its code. Every question is
// asked outside a lock and with no temporary file open, so unwinding through
// the install releases nothing a defer must.
type setupStop struct {
	code int
	msg  string
	// keep records the answers given before the stop; a refused answers file
	// records nothing.
	keep bool
}

// drawnPrompter is the setup interview at a terminal: each question drawn by
// the answer loop, arrow keys first, and recorded as answered in Terminal.
type drawnPrompter struct {
	setupQuestions
	term ask.Terminal
}

// AtTerminal reports that a person is answering, which makes the prompter an
// ahoy.TerminalPrompter.
func (p *drawnPrompter) AtTerminal() bool { return true }

// put asks q at the terminal. Ctrl-C ends the run with exit 130, keeping the
// answers already given and recording nothing for q; any other failure ends
// it with exit 1. The words name no verb: ahoy remote apply and site setup
// ask through this door too.
func (p *drawnPrompter) put(q question.Question) ask.Answer {
	got, err := drawnPut(p.term, question.Ask{Questions: []question.Question{q}})
	switch {
	case errors.Is(err, ask.ErrInterrupted):
		panic(&setupStop{code: ask.ExitInterrupted, msg: "interrupted at " + q.Chip +
			"; nothing is recorded for it, and the answers given before it are kept", keep: true})
	case err != nil:
		panic(&setupStop{code: 1, msg: q.Chip + " could not be asked: " + err.Error(), keep: true})
	case len(got) != 1:
		panic(&setupStop{code: 1, msg: q.Chip + " returned no answer", keep: true})
	}
	return got[0]
}

// Prompt draws the value question; deciding later answers nothing, so the
// install treats it as it treats every unanswered question.
func (p *drawnPrompter) Prompt(key string, choices []string, def string) string {
	q := p.value(key, choices, def)
	a := p.put(q)
	p.record(q, a.Value, "", interview.Terminal)
	if a.Later {
		return ""
	}
	return a.Value
}

// Confirm draws the approval; deciding later declines.
func (p *drawnPrompter) Confirm(text string) bool {
	q := p.confirm(text)
	a := p.put(q)
	p.record(q, a.Value, "", interview.Terminal)
	return a.Value == "yes"
}

// confirmTool draws the tool-install question. It is not part of the record:
// an answers file can never answer it (a program is installed only on an
// answer given at a terminal or relayed by --install-tool), so recording it
// would make a drawn run's record differ from its replay.
func (p *drawnPrompter) confirmTool(tool, text string) bool {
	q := ahoy.SetupConfirmQuestion(0, text)
	q.ID, q.Chip = "install_tool."+tool, "Setup tool"
	return p.put(q).Value == "yes"
}

// answersPrompter is the setup interview off a terminal, answered from an
// answers file: each question written as plain text (the drawing in Mono at
// 80 columns: no colour, no cursor code, no escape byte), then its answer,
// and a question the file does not answer refused, naming the question's id,
// the flag that would answer it and the answers-file key.
type answersPrompter struct {
	setupQuestions
	file interview.Answers
	// stamp is --answered-in, for an entry that names no place of its own.
	stamp string
	ascii bool
}

// AtTerminal reports that no person is at a terminal: the questions abcd puts
// only to a person (the git identity, the drain rule) are never answered
// from a file.
func (p *answersPrompter) AtTerminal() bool { return false }

// answer writes q plainly and returns the file's answer to it, refusing an
// unanswered question and a value q does not offer.
func (p *answersPrompter) answer(q question.Question, flag string) (interview.FileAnswer, question.Option) {
	lines := ask.Draw(question.Ask{Questions: []question.Question{q}}, ask.Frame{Width: question.Default.Columns, Mode: term.Mono, ASCII: p.ascii, Current: -1})
	fmt.Fprintf(p.w, "%s\n", strings.Join(lines, "\n"))
	all := append(append([]question.Option(nil), q.Options...), q.Later)
	values := make([]string, len(all))
	for i, o := range all {
		values[i] = o.Value
	}
	e, ok := p.file.Find(q.ID)
	if !ok {
		panic(&setupStop{code: 2, msg: fmt.Sprintf("abcd ahoy install: %s (%s) has no answer: %s, or add "+
			`{"id": %q, "value": "<%s>"} to the answers file; %s`,
			q.Chip, q.ID, flag, q.ID, strings.Join(values, "|"), writtenBefore(q.ID))})
	}
	for _, o := range all {
		if o.Value == strings.TrimSpace(e.Value) {
			glyph := "›"
			if p.ascii {
				glyph = ">"
			}
			fmt.Fprintf(p.w, "%s %s: %s\n\n", glyph, termsafe.Sanitize(q.Chip), termsafe.Sanitize(o.Label))
			return e, o
		}
	}
	panic(&setupStop{code: 2, msg: fmt.Sprintf("abcd ahoy install: the answers file answers %s (%s) with %q, which it does not offer (%s); %s",
		q.Chip, q.ID, termsafe.Sanitize(e.Value), strings.Join(values, "|"), writtenBefore(q.ID))})
}

// writtenBefore says what a run the answers file stopped at the question id
// left behind: no answers record, and either nothing at all, for a question
// the install asks before its first write, or whatever the steps before the
// question changed. The value questions setup settles before its first write
// (settleSetupAnswers) never reach this stop.
func writtenBefore(id string) string {
	if ahoy.SetupAskedBeforeWriting(id) {
		return "nothing was written, and no answers record was written"
	}
	return "no answers record was written, and what the install changed before this question stays: " +
		"the next abcd ahoy install lists what remains"
}

// settleSetupAnswers checks an answers file before the install's first
// write, so a refusal here leaves the repository and the home untouched
// (iss-2610040025088395): every entry whose question has a fixed set of
// answers must give one of them, and every config value the run would ask
// must be answered, by its flag or by the file, walked in the order and as
// far as the install would ask them (ahoy.WalkConfigValueQuestions). The
// questions whose asking the run itself decides are still checked when they
// are put (answersPrompter.answer).
func settleSetupAnswers(cwd string, opts ahoy.InstallOptions, file interview.Answers) error {
	for _, e := range file.Answers {
		id := strings.TrimSpace(e.ID)
		values, fixed := ahoy.SetupFixedValues(id)
		if fixed && !slices.Contains(values, strings.TrimSpace(e.Value)) {
			return &exitError{Code: 2, Msg: fmt.Sprintf("abcd ahoy install: the answers file answers %s with %q, which it does not offer (%s); "+
				"nothing was written, and no answers record was written", termsafe.Sanitize(id), termsafe.Sanitize(e.Value), strings.Join(values, "|"))}
		}
	}
	var missing string
	var missingChoices []string
	err := ahoy.WalkConfigValueQuestions(cwd, opts, func(id string) bool {
		e, ok := file.Find(id)
		return ok && strings.TrimSpace(e.Value) == "yes"
	}, func(key string, choices []string, def string) string {
		e, ok := file.Find(key)
		if !ok {
			if missing == "" {
				missing, missingChoices = key, choices
			}
			return ""
		}
		if v := strings.TrimSpace(e.Value); v != ahoy.SetupLaterValue {
			return v
		}
		return ""
	})
	if err != nil || missing == "" {
		return nil // a detection that fails is the install's to report
	}
	q := ahoy.SetupValueQuestion(0, cwd, missing, missingChoices, "")
	var values []string
	for _, o := range append(append([]question.Option(nil), q.Options...), q.Later) {
		values = append(values, o.Value)
	}
	flag := "no flag answers it"
	if h, ok := ahoy.HelpIn(cwd, missing); ok && h.Flag != "" {
		flag = "pass " + h.Flag + " <" + strings.Join(missingChoices, "|") + ">"
	}
	return &exitError{Code: 2, Msg: fmt.Sprintf("abcd ahoy install: the value question (%s) has no answer, and this run asks it: %s, or add "+
		`{"id": %q, "value": "<%s>"} to the answers file; nothing was written, and no answers record was written`,
		missing, flag, missing, strings.Join(values, "|"))}
}

// place is where the entry e was answered: its own answered_in, else the
// flag's.
func (p *answersPrompter) place(e interview.FileAnswer) string {
	if e.AnsweredIn != "" {
		return e.AnsweredIn
	}
	return p.stamp
}

// Prompt answers the value question from the file.
func (p *answersPrompter) Prompt(key string, choices []string, def string) string {
	q := p.value(key, choices, def)
	flag := "no flag answers it"
	if h, ok := ahoy.HelpIn(p.cwd, key); ok && h.Flag != "" {
		flag = "pass " + h.Flag + " <" + strings.Join(choices, "|") + ">"
	}
	e, o := p.answer(q, flag)
	p.record(q, o.Value, e.Note, p.place(e))
	if o.Value == q.Later.Value {
		return ""
	}
	return o.Value
}

// Confirm answers the approval from the file; deciding later declines.
func (p *answersPrompter) Confirm(text string) bool {
	q := p.confirm(text)
	flag := "no flag answers it"
	switch {
	case q.ID == "adopt":
		flag = "pass --adopt or --refuse-adopt"
	case strings.HasPrefix(q.ID, "approve."):
		flag = "pass --yes to approve every kind of change"
	}
	e, o := p.answer(q, flag)
	p.record(q, o.Value, e.Note, p.place(e))
	return o.Value == "yes"
}

// readSetupAnswers reads --answers for the setup interview: strict JSON, one
// shape for every interview.
func readSetupAnswers(path string) (interview.Answers, error) {
	data, err := fsutil.ReadGuarded(path, interview.MaxAnswersBytes)
	if err != nil {
		return interview.Answers{}, &exitError{Code: 2, Msg: "abcd ahoy install: reading the answers file: " + scrubPaths(err) + " (nothing written)"}
	}
	a, err := interview.ParseAnswers(data, setupInterviewName)
	if err != nil {
		return interview.Answers{}, &exitError{Code: 2, Msg: "abcd ahoy install: " + termsafe.Sanitize(scrubPaths(err)) + " (nothing written)"}
	}
	return a, nil
}

// runSetup runs the install through p and ends the interview: the answers
// given are written to the answers record (the repository's local tier, the
// machine-wide part to the home) and named on errOut, and a question that
// stopped the run ends it with its code. An install that ends aborted (the
// adoption declined) or refused changed nothing, so it writes no record and
// says nothing of one.
func runSetup(cwd string, p ahoy.Prompter, errOut io.Writer, install func() (ahoy.InstallResult, error)) (res ahoy.InstallResult, err error) {
	rp, recording := p.(recordingPrompter)
	defer func() {
		r := recover()
		if r == nil {
			if recording && err == nil && res.Status != "aborted" && res.Status != "refused" {
				writeSetupRecords(cwd, rp.recorded(), errOut)
			}
			return
		}
		stop, ok := r.(*setupStop)
		if !ok {
			panic(r)
		}
		if stop.keep && recording {
			writeSetupRecords(cwd, rp.recorded(), errOut)
		}
		err = &exitError{Code: stop.code, Msg: termsafe.SanitizeBlock(stop.msg)}
	}()
	return install()
}

// endOnStop is deferred by a verb that asks through newPrompter outside
// setup (ahoy remote apply, site setup): a question the drawn door stopped
// (Ctrl-C) ends the verb with its exit code, recording nothing; any other
// panic goes on.
func endOnStop(err *error) {
	r := recover()
	if r == nil {
		return
	}
	stop, ok := r.(*setupStop)
	if !ok {
		panic(r)
	}
	*err = &exitError{Code: stop.code, Msg: termsafe.SanitizeBlock(stop.msg)}
}

// writeSetupRecords writes the answers record: the machine-wide answers to
// the home's interviews, the rest to the repository's local tier, each named
// on w. A record that cannot be written is said, and the run's outcome
// stands.
func writeSetupRecords(cwd string, answers []interview.Answer, w io.Writer) {
	var repo, machine []interview.Answer
	for _, a := range answers {
		if ahoy.SetupMachineWide(a.ID) {
			machine = append(machine, a)
		} else {
			repo = append(repo, a)
		}
	}
	now := time.Now()
	write := func(place interview.Place, target, shown string, as []interview.Answer) {
		if len(as) == 0 {
			return
		}
		p, err := interview.Write(place, interview.Record{Interview: setupInterviewName, Target: target, Answers: as}, now)
		if err != nil {
			fmt.Fprintf(w, "abcd ahoy install: the %s's answers were not recorded: %s\n", target, termsafe.Sanitize(scrubPaths(err)))
			return
		}
		fmt.Fprintf(w, "abcd ahoy install: the %s's answers are recorded in %s/%s\n", target, shown, termsafe.Sanitize(filepath.Base(p)))
	}
	write(interview.Place{Repo: cwd}, interview.TargetRepository, interview.RecordsRel, repo)
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		write(interview.Place{Home: home}, interview.TargetMachine, abcdhome.Display("interviews"), machine)
	} else if len(machine) > 0 {
		fmt.Fprintf(w, "abcd ahoy install: the machine's answers were not recorded: no home directory resolves\n")
	}
}
