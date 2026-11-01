package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/autonomy"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/mode"
	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// The question check, the question gate and its reset (itd-2609212130146198;
// spc-2610030944505997; iss-2610100626211810).
//
// The guard hook, on the host's question tool, holds abcd's own questions to
// the field limits of internal/core/question wherever it runs, and refuses a
// badly built one naming each part to fix. In a checkout abcd manages it also
// keeps the status-line badge true: abcd's question carries a chip naming
// whom it is for, so an admitted question sets the mode from the chip's role
// and is marked open. The answer resets the mode to managed: the question
// tool's PostToolUse hook runs the reset when the answer comes back, and the
// prompt hook runs the same reset on the next human message, for a host that
// does not run PostToolUse. Another tool's question is none of this file's
// business and runs unchecked, except while the person is away
// (spc-2610180914469489): then every question is refused, routed by its act
// to `abcd mode queue` or `abcd decide`.
//
// This file is the whole of the feature on the front-door side. The guard's
// own hook calls questionGate once, before its shell-command path, and touches
// nothing of the shell guard's tokenizer or decision.

// questionTools are the host tool names that put a question to the human. The
// hook manifest's PreToolUse matcher names exactly these beside the shell tool
// (TestGuardHookIsInstalledForBashCalls holds the two together), and its
// PostToolUse matcher names exactly these alone (TestQuestionAnswerHookIsInstalled).
var questionTools = []string{"AskUserQuestion"}

// isQuestionTool reports whether a hook payload's tool is a question tool.
func isQuestionTool(name string) bool {
	for _, q := range questionTools {
		if strings.EqualFold(name, q) {
			return true
		}
	}
	return false
}

// questionGate is the guard hook's answer for a question-tool call
// (itd-2609212130146198; spc-2610030944505997, "The question check in the
// guard hook"). It never rewrites the question: it admits it or refuses it
// with the host's deny, and it never puts a replacement input on stdout. A
// question whose only finding is the rows limit is admitted with a note for
// the agent (rowsNote).
//
// The away record is read first (spc-2610180914469489, "The question
// reroute"). While it holds, every question-tool call is refused with one
// reason, abcd's question or another tool's, chipped or not, readable or not
// (awayQuestionRefusal): the person is not there to answer, and the reason
// routes the question by the act it asks about. Nothing past that point runs,
// so no mode is set from a chip and no question is marked open while away.
//
// While the person is present the order is the spec's. First the questions
// are decoded; a field the check cannot read is not a decision, so the
// question runs on the loud, non-blocking status. Then the gate decides whether the question is abcd's: a header in
// abcd's chip grammar, which only abcd's interview pages are taught to write,
// or a mode naming somebody, which only abcd's mode verb and this gate set
// (itd-201 decision 10). Anything else is another tool's question and runs,
// unchecked and unmarked, wherever it is asked.
//
// abcd's question is held to the field limits wherever the hook runs, managed
// or not: the setup interview asks before a repository is managed, and the
// limits need no store. A chipped question is checked against the addressee
// each tab's chip names, and a question without a chip against the person the
// mode names. Only the field findings refuse: a chipped question is never
// refused on the mode (iss-2610100626211810), because its chip already says
// whom it is for. Where the badge shows (a checkout abcd manages, with the
// local tier the mode verb writes to), an admitted question is marked open and
// a chipped one sets the mode from its chip (chipState), so the status line
// names whom the question on screen is for. A store or marker the gate cannot
// read or write is not a decision: the question runs and the gate says so, the
// guard's fail-open-loud contract.
//
// The limits abcd's question is held to are the repository's
// (itd-2610301914350541): the resolver, layered.QuestionLimits, reads the
// three question keys of .abcd/config.json at the rules root over
// question.Default, tightened where the person's own
// ~/.abcd.noindex/config.json sets one lower. A value it read and refused
// (outside its bounds, not a whole number, a key the namespace does not
// claim, or a personal value looser than the limit in force) refuses every
// abcd question naming the key, the value, the bounds or the limit in force,
// and the file, and no asking rule (limitsRefusedReason): nothing falls
// through to the defaults. A layer it could not read is the fail-open-loud
// case: the question runs, unchecked, and the gate names the file. Another
// tool's question runs unchecked whatever either file says.
func questionGate(cmd *cobra.Command, cwd string, raw json.RawMessage) error {
	away, unread := awayAtQuestion(cwd)
	if away {
		return denyCall(awayQuestionRefusal())
	}
	if unread == nil {
		return presentQuestionGate(cmd, cwd, raw)
	}
	// The away record could not be read safely, so it reads as not away and
	// the question is checked as if the person were present. Said on stderr,
	// and kept loud: an admitted question exits 1, which the host shows,
	// unless the gate already answered on stdout, whose note exit 1 would
	// drop. A deny folds the line into its reason.
	fmt.Fprintf(cmd.ErrOrStderr(), "abcd guard: the away record was not read (%s), so it reads as not away and the question is checked as if the person were present.\n",
		termsafe.Sanitize(scrubPaths(unread)))
	out := &wroteWriter{w: cmd.OutOrStdout()}
	cmd.SetOut(out)
	err := presentQuestionGate(cmd, cwd, raw)
	cmd.SetOut(out.w)
	if err == nil && !out.wrote {
		return &exitError{Code: 1}
	}
	return err
}

// awayAtQuestion reads whether the person is away for the repository holding
// cwd (spc-2610180914469489, "The question reroute"). It fails open: outside
// a checkout, in a repository with no root commit, or with no home, nobody is
// away and nothing is said; a record or lock the store refuses reads as not
// away, and is returned for the one line the gate writes.
func awayAtQuestion(cwd string) (away bool, unread error) {
	root, err := mode.Root(cwd)
	if err != nil {
		return false, nil
	}
	_, holds, err := autonomy.AwayAt(root, autonomy.Options{})
	if err != nil {
		if errors.Is(err, autonomy.ErrUnsafe) {
			return false, err
		}
		return false, nil
	}
	return holds, nil
}

// awayQuestionRefusal is the one reason every question gets while the person
// is away, whatever its chip and whoever wrote it (dcn-2610180937177502): a
// head line saying why, then the route by the act (autonomy.QuestionRoute).
// It names none of the asking rules, so the agent reroutes the question
// rather than redrafting it.
func awayQuestionRefusal() string {
	return "Blocked by the abcd guard (question tool): the person is away (abcd mode autonomous), so no question is put to them, whatever its chip and whoever wrote it.\n" +
		autonomy.QuestionRoute()
}

// wroteWriter records whether anything was written through it.
type wroteWriter struct {
	w     io.Writer
	wrote bool
}

func (o *wroteWriter) Write(p []byte) (int, error) {
	o.wrote = o.wrote || len(p) > 0
	return o.w.Write(p)
}

// presentQuestionGate is the gate while the person is present: everything
// questionGate's comment describes after the away check.
func presentQuestionGate(cmd *cobra.Command, cwd string, raw json.RawMessage) error {
	stderr := cmd.ErrOrStderr()
	fields, err := decodeQuestions(raw)
	if err != nil {
		return questionFailOpen(stderr, "the question tool's input could not be read (%s)", err)
	}

	// The badge shows only in a checkout abcd manages with the local tier;
	// only there is there a mode store to read.
	root, rerr := mode.Root(cwd)
	badge := rerr == nil && ahoy.Managed(root) && mode.HasTier(root)
	st := mode.Managed
	if badge {
		if st, err = mode.ReadAt(root); err != nil {
			return questionFailOpen(stderr, "the mode store could not be read (%s)", err)
		}
	}
	// The limits are resolved before the question is known to be abcd's,
	// since the chip grammar is read through them, but their outcome is acted
	// on only for abcd's question: another tool's runs unchecked whatever the
	// configuration says.
	limits, lerr := questionLimitsAt(cwd)
	chip := hasAbcdChip(fields, limits)
	if !chip && st == mode.Managed {
		// A question without a chip is abcd's only through the mode store,
		// found through git's toplevel. In a tree that looks like a checkout
		// but git would not name the root of, the store was never read, so
		// whether the mode names somebody is unknown: the question runs, and
		// the gate says the check did not (iss-2610110432159652). Outside
		// every repository there is no store to miss, and it stays silent.
		if rerr != nil && gitutil.RepoShaped(cwd) {
			return questionFailOpen(stderr, "the repository root could not be found, so the mode store was not read (%s)", rerr)
		}
		return nil
	}

	if lerr != nil {
		var unread *layered.UnreadError
		if errors.As(lerr, &unread) {
			return questionFailOpen(stderr, "the question limits were not read, so the question was not held to them (%s)", lerr)
		}
		why := ""
		if !chip {
			why = modeMadeAbcds(st)
		}
		return denyCall(limitsRefusedReason(lerr, why))
	}

	// A chip names the addressee of its own tab, and the mode the gate sets
	// follows the chip, so a chipped question is checked against its chips
	// and not against a mode it is about to replace.
	who := addresseeOf(st)
	if chip {
		who = question.Unnamed
	}
	findings := question.CheckLimits(fields, limits, question.Addressee{
		Person: who,
		Verbs:  verbsOf(cmd.Root()),
	})
	// The rows limit alone does not refuse (iss-2610070637562567): a question
	// too tall for the narrow window is shown, and the agent is told
	// afterwards. Refusing it made the agent redraft a question the person
	// was ready to answer.
	if len(findings) > 0 && !rowsOnly(findings) {
		why := ""
		if !chip {
			why = modeMadeAbcds(st)
		}
		var reason strings.Builder
		writeLimitsRefusal(&reason, findings, why)
		return denyCall(reason.String())
	}
	if badge {
		want := st
		if chip {
			want = chipState(fields, limits)
		}
		// The marker first: a marker written for a mode that then could not
		// be set costs one reset to managed on the answer, while a mode set
		// with no marker would stay parked with nothing left to reset it.
		if err := mode.MarkQuestionOpen(root, want); err != nil {
			return questionFailOpen(stderr, "the question could not be marked open, so the mode was not set from its chip and will not reset on the answer (%s)", err)
		}
		if want != st {
			if err := mode.SetAt(root, want); err != nil {
				return questionFailOpen(stderr, "the mode could not be set from the question's chip, so the status line does not say whom the question is for (%s)", err)
			}
		}
	}
	if len(findings) > 0 {
		// Only rows findings are left: the question runs, and the agent
		// reads the note after it returns. No permission decision is set,
		// so the host's own permission flow still applies.
		if err := writeHookNote(cmd.OutOrStdout(), rowsNote(findings)); err != nil {
			return questionFailOpen(stderr, "the note on the question's height could not be written (%s)", err)
		}
	}
	return nil
}

// chipState is the mode a chipped question sets (iss-2610100626211810): the
// person its chips name. The Product chip names the product thinker, and the
// Tech chip the technical facilitator. The Setup chip names the facilitator
// too: the setup interview asks how abcd is installed and configured in a
// repository (its visibility, its keys, the oracles it connects), which is
// the how of the work, the facilitator's half, never the what. A call whose
// tabs name both people sets the product thinker, so the status line never
// says only the facilitator is owed an answer the product thinker owes too.
func chipState(f question.Fields, l question.Limits) mode.State {
	st := mode.Facilitator
	for _, t := range f.Tabs {
		if role, ok := question.ChipRole(t.Header, l); ok && chipRoleState(role) == mode.ProductThinker {
			st = mode.ProductThinker
		}
	}
	return st
}

// chipRoleState is the mode one chip role word names: the product thinker
// for the Product role, the technical facilitator for every other role (Tech
// and Setup). The interview pages' test reads it to accept a page that names
// a question's chip in place of setting the mode.
func chipRoleState(role string) mode.State {
	if role == question.ProductRole {
		return mode.ProductThinker
	}
	return mode.Facilitator
}

// rowsOnly reports whether every finding is the rows limit's.
func rowsOnly(findings []question.Finding) bool {
	for _, f := range findings {
		if f.Rule != question.RuleRows {
			return false
		}
	}
	return true
}

// rowsNote is what the agent is told when a question ran over the rows limit
// and was shown anyway: each tab over it, its rows and the limit, at most
// maxRefusalParts of them, and how to keep the next question within it. It
// never tells the agent to ask again, because the person may already have
// answered the question it was shown.
func rowsNote(findings []question.Finding) string {
	var b strings.Builder
	fmt.Fprintf(&b, "abcd guard (question tool): this question was shown, but %d tab(s) run past abcd's rows limit, so the host may cut them in a narrow window.\n", len(findings))
	for _, f := range findings[:min(len(findings), maxRefusalParts)] {
		fmt.Fprintf(&b, "tab %d is %s; limit: %s.\n", f.Tab, f.Value, f.Limit)
	}
	if more := len(findings) - maxRefusalParts; more > 0 {
		fmt.Fprintf(&b, "... and %d more tab(s).\n", more)
	}
	b.WriteString("The question was shown this time; keep the next question within the limit by drafting it through the abcd:question-drafter agent, which counts rows as this check does.")
	return b.String()
}

// hostQuestionInput is the host's question-tool input as the check reads it:
// the host's own JSON key names, so the decoding lives here in the surface and
// the core's field view knows none of them. Unknown keys are ignored (the host
// owns the schema and adds to it); a known key of the wrong type makes the
// whole input unreadable.
type hostQuestionInput struct {
	Header   string `json:"header"`
	Question string `json:"question"`
	Options  []struct {
		Label       string `json:"label"`
		Description string `json:"description"`
		Preview     string `json:"preview"`
	} `json:"options"`
}

// errUnreadableQuestions is why a questions field cannot be checked. It names
// the shape the check reads and echoes nothing from the payload.
var errUnreadableQuestions = errors.New("tool_input.questions is not a list of questions, each a header, a question and options with a label, a description and an optional preview")

// errUnreadableQuestionPayload is why a question-tool payload whose outer JSON
// the hook could not decode is not checked. It names the shape the hook reads
// and echoes nothing from the decoder, whose text can carry a Go type.
var errUnreadableQuestionPayload = errors.New("it is not a JSON object whose tool_input is an object, nested no deeper than the decoder reads")

// decodeQuestions reads the question tool's questions into the field view the
// limits check reads, one tab per question.
func decodeQuestions(raw json.RawMessage) (question.Fields, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return question.Fields{}, errUnreadableQuestions
	}
	var in []hostQuestionInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return question.Fields{}, errUnreadableQuestions
	}
	f := question.Fields{Tabs: make([]question.Tab, len(in))}
	for i, q := range in {
		t := question.Tab{Header: q.Header, Text: q.Question, Options: make([]question.Choice, len(q.Options))}
		for j, o := range q.Options {
			t.Options[j] = question.Choice{Label: o.Label, Description: o.Description, Preview: o.Preview}
		}
		f.Tabs[i] = t
	}
	return f, nil
}

// hasAbcdChip reports whether any tab's header is in abcd's chip grammar, read
// through l (whose header width and role words are the binary's and no
// repository sets, itd-2610301914350541 decision 2).
func hasAbcdChip(f question.Fields, l question.Limits) bool {
	for _, t := range f.Tabs {
		if _, ok := question.ChipRole(t.Header, l); ok {
			return true
		}
	}
	return false
}

// addresseeOf is the person a mode names; a managed mode, or no mode store,
// names nobody, and the check then reads the chip's role word instead.
func addresseeOf(st mode.State) question.Person {
	switch st {
	case mode.ProductThinker:
		return question.ProductThinker
	case mode.Facilitator:
		return question.Facilitator
	}
	return question.Unnamed
}

// verbsOf is the binary's verb list, read from the command tree, so the
// product thinker's register rule names a command by the verbs the binary
// actually has and the core holds no copy of them.
func verbsOf(root *cobra.Command) []string {
	var out []string
	for _, c := range root.Commands() {
		out = append(out, c.Name())
		out = append(out, c.Aliases...)
	}
	return out
}

// maxRefusalParts bounds the finding lines one refusal names. The host
// replays the refusal to the agent, so its size must follow the limits, not
// the payload: a question of thousands of broken fields is still refused in a
// few lines (review-askGuard-security finding 1).
const maxRefusalParts = 10

// modeMadeAbcds is the line a refusal carries when the question carries no abcd
// chip and is abcd's only because the mode names somebody (itd-201 decision
// 10): it says why, so an agent that did not write the question, another
// tool's, does not loop on rules it never meant to follow
// (review-askGuard-security finding 4).
func modeMadeAbcds(st mode.State) string {
	who := "the product thinker"
	if st == mode.Facilitator {
		who = "the technical facilitator"
	}
	return "This question carries no abcd chip: it is treated as abcd's because the mode names " + who +
		"; another tool's question asked now is held to abcd's rules."
}

// writeLimitsRefusal writes the refusal the host's deny carries: one head line
// counting the parts that refuse, then, when why is set, the one line saying
// why a question without abcd's chip is abcd's, then one line per refusing
// finding naming the tab, the part, the value, the limit and the remedy, at
// most maxRefusalParts of them, and one closing line counting the parts not
// named. Every line passes termsafe.Sanitize, so no value the agent wrote
// reaches the terminal raw.
//
// The rows limit alone never refuses (rowsOnly), so a rows finding is never
// one of the parts to fix and is not counted among them
// (iss-2610100626327722): listed among them, the agent read it as a cause and
// cut a question the gate would have shown. It follows the refusing parts
// instead, after rowsNotRefusingLine, bounded as they are. It is called only
// with at least one refusing finding.
func writeLimitsRefusal(w io.Writer, findings []question.Finding, why string) {
	var refusing, rows []question.Finding
	for _, f := range findings {
		if f.Rule == question.RuleRows {
			rows = append(rows, f)
		} else {
			refusing = append(refusing, f)
		}
	}
	fmt.Fprintf(w, "Blocked by the abcd guard (question tool): %d part(s) of this question break abcd's asking rules; fix each and ask again.\n", len(refusing))
	if why != "" {
		fmt.Fprintln(w, why)
	}
	for _, f := range refusing[:min(len(refusing), maxRefusalParts)] {
		fmt.Fprintln(w, termsafe.Sanitize(f.String()))
	}
	if more := len(refusing) - maxRefusalParts; more > 0 {
		fmt.Fprintf(w, "... and %d more part(s); fix these and ask again to see the rest.\n", more)
	}
	if len(rows) == 0 {
		return
	}
	fmt.Fprintln(w, rowsNotRefusingLine)
	for _, f := range rows[:min(len(rows), maxRefusalParts)] {
		fmt.Fprintln(w, termsafe.Sanitize(f.String()))
	}
	if more := len(rows) - maxRefusalParts; more > 0 {
		fmt.Fprintf(w, "... and %d more tab(s) over the rows limit.\n", more)
	}
}

// rowsNotRefusingLine opens the rows findings a refusal names but does not
// refuse on.
const rowsNotRefusingLine = "The rows limit does not refuse on its own: with only the rows finding(s) below, this question would have been shown, and a narrow window might cut it."

// questionLimitsAt is the question limits in force for the repository
// holding cwd, through the one resolver (layered.QuestionLimits) at the root
// the rules loader resolves, so the gate and the injected GRILL domain read
// one file (itd-2610301914350541 decision 6). The mode store keeps its own
// root (mode.Root, git's toplevel); in a monorepo member with its own .abcd/
// the two differ, and the limits follow the rules root.
//
// The limits returned are always usable for the header grammar: on an unread
// layer they are the defaults, as the resolver returns them, and on a refusal
// they are the defaults too, since a refused value is never applied (the
// question is refused before any limit is checked) and the chip grammar the
// gate reads them for is the binary's alone.
func questionLimitsAt(cwd string) (question.Limits, error) {
	roots, _ := layered.RootsFor(cwd)
	v, err := layered.QuestionLimits(roots)
	var unread *layered.UnreadError
	if err != nil && !errors.As(err, &unread) {
		grammar := question.Default
		return grammar, err
	}
	return v.V, err
}

// limitsRefusedReason is the deny for abcd's question while a question limit
// is refused (itd-2610301914350541 decision 7): one head line saying abcd
// asks nothing until the file is fixed, then, when why is set, the line
// saying why a question without abcd's chip is abcd's, the resolver's own
// refusal naming the file, the key, the value and the bounds (or, for a
// personal value that loosens, the limit in force and the file that set it),
// and one line stating every key's bounds from question.Bounds. It names no
// asking rule, so the agent fixes the configuration rather than redrafting a
// question that may be well built.
func limitsRefusedReason(err error, why string) string {
	var b strings.Builder
	b.WriteString("Blocked by the abcd guard (question tool): a question limit in this repository's configuration is refused, so abcd asks no question until it is fixed.\n")
	if why != "" {
		b.WriteString(why + "\n")
	}
	b.WriteString(termsafe.Sanitize(scrubPaths(err)) + "\n")
	b.WriteString("Fix the value in that file and ask again: " + boundsLine() + ", and a person's own " + abcdhome.Display(layered.Config.MachineRel) + " may set each only lower than the limit otherwise in force.")
	return b.String()
}

// boundsLine states each configurable key's bounds, rendered from
// question.Bounds, the one statement of them.
func boundsLine() string {
	parts := make([]string, len(question.Bounds))
	for i, b := range question.Bounds {
		parts[i] = b.Key + " takes " + b.String()
	}
	return strings.Join(parts, ", ")
}

// questionFailOpen is the gate's failOpen: exit 1, which lets the question run
// and keeps the warning in front of a human.
func questionFailOpen(w io.Writer, format string, err error) error {
	fmt.Fprintf(w, "abcd guard: NOT CHECKED — "+format+". The question runs UNGATED.\n",
		termsafe.Sanitize(scrubPaths(err)))
	return &exitError{Code: 1}
}

// resetModeOnAnswer is the reset an answer triggers, run from two hooks: the
// question tool's PostToolUse hook, when the answer comes back
// (iss-2610100626211810), and the prompt hook, on the next human message, for
// a host that does not run PostToolUse. When a question is marked open the
// mode goes back to managed, the marker is cleared, and one line on stderr
// says so; whichever hook runs first clears the marker, so the other changes
// nothing. Nothing goes to stdout, which the host hands the agent. Every
// failure is named and never stops the session.
func resetModeOnAnswer(w io.Writer, cwd string) {
	root, err := mode.Root(cwd)
	if err != nil {
		return
	}
	reset, err := mode.ResetOnAnswer(root)
	switch {
	case err != nil:
		// The store's error says whether the mode moved; it is the one line.
		fmt.Fprintf(w, "abcd mode: after the question, %s\n", termsafe.Sanitize(scrubPaths(err)))
	case reset:
		fmt.Fprintln(w, "abcd mode: the question was answered, so the mode is reset to managed")
	}
}
