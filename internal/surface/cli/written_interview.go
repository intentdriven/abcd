package cli

// written_interview.go is the front door of the interviews an AI writes
// (spc-2610030911534855, "The interviews an AI writes, through the person's
// own route"): `abcd reflect interview <release-tag>` and `abcd intent
// interview <itd-N>`. The turn loop is core's (interview.Written); this door
// reads the person's runner configuration, picks how each question is
// answered, relays an interrupt to the runner while it writes, and maps the
// loop's ends to exit codes.
//
// A question is drawn only when stdin, stdout and stderr are all terminals
// (adr-49 decision 1); anywhere else it is written as plain text and answered
// from --answers, by ordinal (Q1, Q2, ...), and a run with neither refuses
// before any runner starts. Every question reaches the person sanitised
// (question.Safe, in the loop's check and again in the drawing), and every
// answer is recorded through interview.Write with where it was given.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/intentdriven/abcd/internal/core/interview"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/surface/cli/ask"
	"github.com/intentdriven/abcd/internal/term"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// writtenFlags are the flags both AI-written interviews take.
type writtenFlags struct {
	answers    string
	answeredIn string
}

func (f *writtenFlags) register(cmd *cobra.Command, name string) {
	cmd.Flags().StringVar(&f.answers, "answers", "", "answer the questions from this answers file (JSON: schema_version, interview \""+name+
		"\", answers by ordinal Q1, Q2, ...) instead of drawing them; the run refuses at the first question the file does not answer")
	cmd.Flags().StringVar(&f.answeredIn, "answered-in", interview.Terminal, "where an answers-file entry that names no place was answered: "+
		"Terminal, or the host whose question tool asked it, as its plugin page passes; recorded with each answer")
}

// finishError is the interview's own writer refusing its outcome; the verb
// maps it as the writer's verb would.
type finishError struct{ err error }

func (e *finishError) Error() string { return e.err.Error() }
func (e *finishError) Unwrap() error { return e.err }

// runWrittenInterview runs w through the person's own route. It fills in the
// runner configuration, the transcript store, the verbs the limits check
// reads, the front door and the interrupt relay, and returns the loop's
// result with its end mapped to an exit: the no-route refusal, a missing
// local tier and a refused answers file exit 2, an interrupt exits 130, a
// runner that did not answer exits 1, and a *finishError is handed back for
// the verb to map. The answers record, when one was written, is named on
// stderr.
func runWrittenInterview(cmd *cobra.Command, w *interview.Written, f writtenFlags) (interview.WrittenResult, error) {
	errOut := cmd.ErrOrStderr()
	if !interview.ValidAnsweredIn(f.answeredIn) {
		return interview.WrittenResult{}, &exitError{Code: 2, Msg: fmt.Sprintf("%s: --answered-in must be %q or %q (nothing was run)",
			w.Verb, interview.Terminal, interview.ClaudeCode)}
	}
	roots, notes := layered.RootsFor(w.Repo)
	for _, n := range notes {
		fmt.Fprintln(errOut, termsafe.Sanitize(n))
	}
	cfg, err := runner.Load(roots)
	if err != nil {
		return interview.WrittenResult{}, &exitError{Code: 2, Msg: w.Verb + ": " + termsafe.Sanitize(fsutil.RedactHome(err.Error())) + " (nothing was run)"}
	}
	for _, d := range cfg.Diagnostics {
		fmt.Fprintln(errOut, termsafe.Sanitize(fsutil.RedactHome(d)))
	}
	if !interview.Reaches(cfg, w.Role) {
		return interview.WrittenResult{}, &exitError{Code: 2, Msg: termsafe.SanitizeBlock(w.NoRoute().Error())}
	}
	w.Config = cfg
	w.Transcripts = &lazyHistoryStore{cmd: cmd}
	w.Verbs = verbsOf(cmd.Root())
	w.DispatchContext = func(ctx context.Context) (context.Context, context.CancelFunc) {
		return signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	}

	switch tm, tnotes, drawn := drawnTerminal(cmd.InOrStdin(), cmd.OutOrStdout(), errOut, w.Repo); {
	case f.answers != "":
		data, err := fsutil.ReadGuarded(f.answers, interview.MaxAnswersBytes)
		if err != nil {
			return interview.WrittenResult{}, &exitError{Code: 2, Msg: w.Verb + ": reading the answers file: " + scrubPaths(err) + " (nothing was run)"}
		}
		file, err := interview.ParseAnswers(data, w.Name)
		if err != nil {
			return interview.WrittenResult{}, &exitError{Code: 2, Msg: w.Verb + ": " + termsafe.Sanitize(scrubPaths(err)) + " (nothing was run)"}
		}
		w.Answer = fileAnswerer(errOut, w.Verb, file, f.answeredIn, !term.UTF8Locale(os.Getenv))
	case drawn:
		for _, n := range tnotes {
			fmt.Fprintln(errOut, termsafe.Sanitize(n))
		}
		w.Answer = drawnAnswerer(tm)
	default:
		return interview.WrittenResult{}, &exitError{Code: 2, Msg: w.Verb + ": the questions are drawn only at a terminal (stdin, stdout and stderr " +
			"all terminals); anywhere else they are answered from --answers <file>, by ordinal (Q1, Q2, ...) (nothing was run)"}
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	res, err := w.Run(ctx)
	if res.Record != "" {
		fmt.Fprintf(errOut, "%s: the answers are recorded in %s/%s\n", w.Verb, interview.RecordsRel, termsafe.Sanitize(filepath.Base(res.Record)))
	}
	if err == nil {
		return res, nil
	}
	msg := w.Verb + ": " + termsafe.SanitizeBlock(scrubPaths(err))
	var noRoute *interview.NoRouteError
	var fin *finishError
	switch {
	case errors.As(err, &noRoute):
		return res, &exitError{Code: 2, Msg: termsafe.SanitizeBlock(err.Error())}
	case errors.Is(err, interview.ErrNoLocalTier):
		return res, &exitError{Code: 2, Msg: msg + " (nothing was run)"}
	case errors.Is(err, ask.ErrInterrupted), errors.Is(err, interview.ErrInterrupted):
		return res, &exitError{Code: ask.ExitInterrupted, Msg: w.Verb + ": interrupted; the answers given before it are kept"}
	case errors.As(err, &fin):
		return res, fin
	case errors.Is(err, interview.ErrNothingRecorded):
		return res, &exitError{Code: 2, Msg: msg}
	}
	return res, &exitError{Code: 1, Msg: msg}
}

// drawnAnswerer puts each ask at the terminal through the answer loop, every
// answer given in Terminal.
func drawnAnswerer(tm ask.Terminal) func(question.Ask) ([]interview.Reply, error) {
	return func(a question.Ask) ([]interview.Reply, error) {
		got, err := drawnPut(tm, a)
		if err != nil {
			return nil, err
		}
		out := make([]interview.Reply, len(got))
		for i, g := range got {
			out[i] = interview.Reply{ID: g.ID, Value: g.Value, AnsweredIn: interview.Terminal}
		}
		return out, nil
	}
}

// fileAnswerer writes each question as plain text on w (the drawing in Mono
// at 80 columns: no colour, no cursor code, no escape byte) and answers it
// from the answers file by its ordinal. The file running out, or giving a
// value the question does not offer, stops the run and records nothing.
func fileAnswerer(w io.Writer, verb string, file interview.Answers, stamp string, ascii bool) func(question.Ask) ([]interview.Reply, error) {
	glyph := "›"
	if ascii {
		glyph = ">"
	}
	return func(a question.Ask) ([]interview.Reply, error) {
		out := make([]interview.Reply, 0, len(a.Questions))
		for _, q := range a.Questions {
			one := question.Ask{Questions: []question.Question{q}}
			fmt.Fprintf(w, "%s\n", strings.Join(ask.Draw(one, ask.Frame{Width: question.Default.Columns, Mode: term.Mono, ASCII: ascii, Current: -1}), "\n"))
			values := interview.Values(q)
			e, ok := file.Find(q.ID)
			if !ok {
				return nil, errors.Join(fmt.Errorf("the answers file runs out at %s (%s): add {\"id\": %q, \"value\": \"<%s>\"} to answer it; "+
					"no answers record was written", q.ID, termsafe.Sanitize(q.Chip), q.ID, termsafe.Sanitize(strings.Join(values, "|"))), interview.ErrNothingRecorded)
			}
			v := strings.TrimSpace(e.Value)
			if !slices.Contains(values, v) {
				return nil, errors.Join(fmt.Errorf("the answers file answers %s (%s) with %q, which it does not offer (%s); no answers record was written",
					q.ID, termsafe.Sanitize(q.Chip), termsafe.Sanitize(v), termsafe.Sanitize(strings.Join(values, "|"))), interview.ErrNothingRecorded)
			}
			fmt.Fprintf(w, "%s %s: %s\n\n", glyph, termsafe.Sanitize(q.Chip), termsafe.Sanitize(labelOf(q, v)))
			place := e.AnsweredIn
			if place == "" {
				place = stamp
			}
			out = append(out, interview.Reply{ID: q.ID, Value: v, Note: e.Note, AnsweredIn: place})
		}
		return out, nil
	}
}

// labelOf is the label q draws for value.
func labelOf(q question.Question, value string) string {
	opts := q.Options
	if q.List != nil {
		opts = q.List.Choices
	}
	for _, o := range append(append([]question.Option(nil), opts...), q.Later) {
		if o.Value == value {
			return o.Label
		}
	}
	return value
}
