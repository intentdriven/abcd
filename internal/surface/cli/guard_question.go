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
// names whom the question on screen is for. A mode store the gate cannot read
// leaves a chipped question judged all the same, at the limits in force,
// since its chip names whom it is for: one that breaks a rule is refused, and
// one that does not runs with no mode set and no marker, and one line says
// so. A question without a chip is abcd's only through the mode, so with the
// store unread it runs unchecked, loudly, the guard's fail-open-loud contract
// for a question it cannot tell is its own. A marker or mode the gate cannot
// write is not a decision either: the question, already judged, runs and the
// gate says so.
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
// through to the defaults. A layer it could not read (a file that is not a
// regular file, cannot be parsed or is over the size cap, or an unresolved
// home) is loud and holds abcd's question to the defaults: one NOT CHECKED
// line names the file, and a question the defaults admit runs at exit 1.
// Another tool's question runs unchecked whatever either file says.
func questionGate(cmd *cobra.Command, cwd string, raw json.RawMessage) error {
	reads := questionState(cmd, cwd)
	if reads.state.Away {
		return denyCall(awayQuestionRefusal())
	}
	if reads.state.AwayErr == nil {
		return presentQuestionGate(cmd, reads, raw)
	}
	// The away record could not be read safely, so it reads as not away and
	// the question is checked as if the person were present. Said on stderr,
	// and kept loud: an admitted question exits 1, which the host shows,
	// unless the gate already answered on stdout, whose note exit 1 would
	// drop. A deny folds the line into its reason.
	fmt.Fprintf(cmd.ErrOrStderr(), "abcd guard: the away record was not read (%s), so it reads as not away and the question is checked as if the person were present.\n",
		termsafe.Sanitize(scrubPaths(reads.state.AwayErr)))
	out := &wroteWriter{w: cmd.OutOrStdout()}
	cmd.SetOut(out)
	err := presentQuestionGate(cmd, reads, raw)
	cmd.SetOut(out.w)
	if err == nil && !out.wrote {
		return &exitError{Code: 1}
	}
	return err
}

// questionReads is what the gate reads before it decides: the state the
// shared decision takes, and the two facts only the hook's writes need, the
// repository root and the mode as the store spells it.
type questionReads struct {
	state question.State
	root  string
	mode  mode.State
}

// questionState makes every store read the question gate's decision takes
// (spc-2610301917245442, "One store reader"): the away record, the
// repository root, whether the badge shows, the mode where it does, the
// binary's verbs, and the question limits in force (questionLimitsAt,
// spc-2610301950037469 step 2). The hook and `abcd guard question` both call
// it and then question.Decide, so the two read the same state the same way.
// It writes nothing; a store it cannot read is named in the state, never
// fatal.
func questionState(cmd *cobra.Command, cwd string) questionReads {
	r := questionReads{mode: mode.Managed}
	r.state.Away, r.state.AwayErr = awayAtQuestion(cwd)
	r.state.Verbs = verbsOf(cmd.Root())
	questionLimitsAt(cwd, &r.state)
	root, rerr := mode.Root(cwd)
	if rerr != nil {
		// In a tree that looks like a checkout but git would not name the
		// root of, the mode store was never read (iss-2610110432159652).
		// Outside every repository there is no store to miss.
		if gitutil.RepoShaped(cwd) {
			r.state.RootErr = rerr
		}
		return r
	}
	r.root = root
	r.state.Repo = true
	// The badge shows only in a checkout abcd manages with the local tier;
	// only there is there a mode store to read.
	r.state.Badge = ahoy.Managed(root) && mode.HasTier(root)
	if !r.state.Badge {
		return r
	}
	st, err := mode.ReadAt(root)
	if err != nil {
		r.state.ModeErr = err
		return r
	}
	r.mode = st
	r.state.Mode = addresseeOf(st)
	return r
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
// questionGate's comment describes after the away check. The decision is
// question.Decide's; this function only acts on it, and every write it makes
// comes after the decision, so `abcd guard question`, which calls the same
// decision, can never reach them.
func presentQuestionGate(cmd *cobra.Command, reads questionReads, raw json.RawMessage) error {
	stderr := cmd.ErrOrStderr()
	fields, err := decodeQuestions(raw)
	if err != nil {
		return questionFailOpen(stderr, "the question tool's input could not be read (%s)", err)
	}
	v := question.Decide(fields, reads.state, reads.state.Limits)
	lead := unreadLead(reads, v)
	if lead == "" {
		return actOnQuestionVerdict(cmd, reads, v, "")
	}
	// A store abcd's question was checked without could not be read: the
	// mode store, for a chipped question, whose mode is then not set, or a
	// layer of the question limits, whose defaults it was then checked at.
	// The gate says so, one line per store, wherever its answer is read: at
	// the head of a deny's reason or a rows note, which the agent reads, and
	// otherwise on stderr with exit 1, the loud non-blocking status the host
	// shows, never a silent admission.
	out := &wroteWriter{w: cmd.OutOrStdout()}
	cmd.SetOut(out)
	err = actOnQuestionVerdict(cmd, reads, v, lead)
	cmd.SetOut(out.w)
	var deny *hookDeny
	if out.wrote || errors.As(err, &deny) {
		return err
	}
	fmt.Fprint(stderr, lead)
	if err == nil {
		return &exitError{Code: 1}
	}
	return err
}

// unreadLead is the lines the gate leads its answer with for abcd's question
// checked without a store it could not read, one per store, in the
// decision's order: the mode store's for a chipped question where the badge
// shows (modeUnreadLine), and the question limits' (limitsUnreadLine). A
// question without a chip and the mode store unread is not checked at all
// and fails open on its own line instead (actOnQuestionVerdict), so it gets
// neither.
func unreadLead(reads questionReads, v question.Verdict) string {
	var b strings.Builder
	for _, u := range v.Unread {
		switch {
		case u.Store == question.StoreMode && v.Chipped && reads.state.Badge && reads.state.ModeErr != nil:
			b.WriteString(modeUnreadLine(reads.state.ModeErr))
		case u.Store == question.StoreLimits:
			b.WriteString(limitsUnreadLine(reads.state.LimitsErr))
		}
	}
	return b.String()
}

// modeUnreadLine is the gate's loud line for a chipped question while the
// mode store could not be read: the question is judged on its chip, which
// names whom it is for, and the mode and the open-question marker the gate
// would write from that chip are not written.
func modeUnreadLine(err error) string {
	return "abcd guard: the mode store could not be read (" + termsafe.Sanitize(scrubPaths(err)) +
		"), so the mode is not set from this question's chip and the question is not marked open; its chip names whom it is for, so it is judged without the mode.\n"
}

// actOnQuestionVerdict does what the hook does with a decision: a deny for a
// refusal, the marker and the mode for an admission where the badge shows,
// the rows note for an admission with rows findings, and questionFailOpen
// for a question the decision could not tell is abcd's because a store it
// needed could not be read. lead, when set, opens the deny's reason and the
// rows note.
func actOnQuestionVerdict(cmd *cobra.Command, reads questionReads, v question.Verdict, lead string) error {
	stderr := cmd.ErrOrStderr()
	switch v.Outcome {
	case question.OutcomeUnchecked:
		if reads.state.ModeErr != nil && !v.Chipped {
			return questionFailOpen(stderr, "the mode store could not be read (%s)", reads.state.ModeErr)
		}
		if reads.state.RootErr != nil && !v.Chipped {
			return questionFailOpen(stderr, "the repository root could not be found, so the mode store was not read (%s)", reads.state.RootErr)
		}
	case question.OutcomeAdmitUnchecked:
		return nil
	case question.OutcomeRefuse:
		why := ""
		if !v.Chipped {
			why = modeMadeAbcds(reads.mode)
		}
		if v.Refused != nil {
			return denyCall(lead + limitsRefusedReason(v.Refused.Reason, why))
		}
		var reason strings.Builder
		reason.WriteString(lead)
		writeLimitsRefusal(&reason, v.Findings, why)
		return denyCall(reason.String())
	}
	if !v.Abcds {
		// Not abcd's question, unchecked because the away record was not
		// read: the caller's loud status is the whole answer.
		return nil
	}
	if reads.state.Badge && reads.state.ModeErr == nil {
		// With the mode store unread the chipped question was still judged,
		// and runs, but nothing is written beside a store the gate could not
		// read: no marker, and no mode (modeUnreadLine says so).
		want := stateOf(v.ModeAfter)
		// The marker first: a marker written for a mode that then could not
		// be set costs one reset to managed on the answer, while a mode set
		// with no marker would stay parked with nothing left to reset it.
		if err := mode.MarkQuestionOpen(reads.root, want); err != nil {
			return questionFailOpen(stderr, "the question could not be marked open, so the mode was not set from its chip and will not reset on the answer (%s)", err)
		}
		if want != reads.mode {
			if err := mode.SetAt(reads.root, want); err != nil {
				return questionFailOpen(stderr, "the mode could not be set from the question's chip, so the status line does not say whom the question is for (%s)", err)
			}
		}
	}
	if len(v.Findings) > 0 {
		// Only rows findings are left: the question runs, and the agent
		// reads the note after it returns. No permission decision is set,
		// so the host's own permission flow still applies.
		if err := writeHookNote(cmd.OutOrStdout(), lead+rowsNote(v.Findings)); err != nil {
			return questionFailOpen(stderr, "the note on the question's height could not be written (%s)", err)
		}
	}
	return nil
}

// stateOf is the mode a person names: the inverse of addresseeOf.
func stateOf(p question.Person) mode.State {
	switch p {
	case question.ProductThinker:
		return mode.ProductThinker
	case question.Facilitator:
		return mode.Facilitator
	}
	return mode.Managed
}

// chipRoleState is the mode one chip role word names: the product thinker
// for the Product role, the technical facilitator for every other role (Tech
// and Setup). The interview pages' test reads it to accept a page that names
// a question's chip in place of setting the mode.
func chipRoleState(role string) mode.State {
	return stateOf(question.ChipPerson(role))
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

// questionLimitsAt puts the question limits in force for the repository
// holding cwd on st, through the one resolver (layered.QuestionLimits) at the
// root the rules loader resolves, so the gate and the injected GRILL domain
// read one file (itd-2610301914350541 decision 6). The mode store keeps its
// own root (mode.Root, git's toplevel); in a monorepo member with its own
// .abcd/ the two differ, and the limits follow the rules root.
//
// The outcome is one of three. In force: st.Limits and st.LimitsSource are
// the resolver's. Unread (a layer the resolver could not read or parse, or an
// unresolved home): st.LimitsErr and st.LimitsFile name it, and st.Limits is
// the defaults the resolver returns with it, which abcd's question is then
// checked at. Refused: st.LimitsRefused carries the refusal, and st.Limits is
// the defaults, assigned here, read only for the header grammar (whose width
// and role words no file sets, decision 2) and the rows split, since a
// refused value is never applied and abcd's question is refused before any
// limit is checked.
func questionLimitsAt(cwd string, st *question.State) {
	roots, _ := layered.RootsFor(cwd)
	v, err := layered.QuestionLimits(roots)
	var unread *layered.UnreadError
	switch {
	case err == nil, errors.As(err, &unread):
		st.Limits = v.V
		st.LimitsSource = make([]string, len(v.Sources))
		for i, src := range v.Sources {
			st.LimitsSource[i] = src.Origin
		}
		if err != nil {
			st.LimitsErr, st.LimitsFile = err, unread.Origin
		}
	default:
		st.Limits = question.Default
		st.LimitsRefused = limitsRefusalOf(err)
	}
}

// limitsRefusalOf is the resolver's refusal as the decision carries it: the
// key, the value, the bounds and the file of a value refused, the limit in
// force and its file for a personal value that loosens, and the resolver's
// own words for every refusal, an unclaimed key included.
func limitsRefusalOf(err error) *question.LimitsRefusal {
	r := &question.LimitsRefusal{Reason: err.Error()}
	var kr *layered.KeyRefusedError
	if errors.As(err, &kr) {
		r.Key, r.Value, r.Bounds, r.File = kr.Key, kr.Value, kr.Bounds.String(), kr.Origin
		if kr.Loosens() {
			r.Limit, r.LimitFile = kr.Limit, kr.LimitOrigin
		}
	}
	return r
}

// limitsUnreadLine is the gate's loud line for abcd's question while a layer
// of the question limits could not be read: in the NOT CHECKED form, naming
// the file, and saying what was done instead, since the question is not let
// through unchecked but held to abcd's defaults.
func limitsUnreadLine(err error) string {
	return "abcd guard: NOT CHECKED at this repository's question limits — they were not read (" +
		termsafe.Sanitize(scrubPaths(err)) + "), so the question is checked at abcd's defaults instead.\n"
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
func limitsRefusedReason(reason, why string) string {
	var b strings.Builder
	b.WriteString("Blocked by the abcd guard (question tool): a question limit in this repository's configuration is refused, so abcd asks no question until it is fixed.\n")
	if why != "" {
		b.WriteString(why + "\n")
	}
	b.WriteString(termsafe.Sanitize(scrubPaths(errors.New(reason))) + "\n")
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
