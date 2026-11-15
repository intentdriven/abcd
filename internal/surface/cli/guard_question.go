package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
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
// keeps the status-line badge true: the agent sets the mode to the person a
// question is for before asking it (decision 37), an older-form chip still
// sets it from its role word, and an admitted question is marked open. The answer resets the mode to managed: the question
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
// question runs on the loud, non-blocking status. Then the gate decides
// whether the question is abcd's by its header alone (spc-2610301917245442,
// step 3; itd-2610230756259215 decision 36): a header in abcd's grammar, the
// number form ("Q1", "Q1a", "Q2/5") or the older form opened by a role word
// ("Product Q2"). Anything else is another tool's question and runs unchecked
// wherever it is asked and whatever the mode names; where the badge shows and
// the mode names somebody it is marked open under that mode, with nothing
// else written (markUncheckedOpen), so the answer resets a mode the agent set
// before a question whose header then missed the grammar.
//
// abcd's question is held to the field limits wherever the hook runs, managed
// or not: the setup interview asks before a repository is managed, and the
// limits need no store. The product thinker's register rule is decided tab by
// tab: an older-form tab against the person its chip names, and every other
// tab against the person the mode names where the badge shows.
// Only the field findings refuse: abcd's question is never refused on the
// mode (iss-2610100626211810). Where the badge shows (a checkout abcd
// manages, with the local tier the mode verb writes to), an admitted question
// is marked open, and an older-form chip sets the mode from its role word
// (Verdict.ModeAfter); a number-form question sets nothing and is marked open under
// the mode the agent set before asking, which the answer resets. A mode store
// the gate cannot read leaves abcd's question judged all the same, at the
// limits in force, the register rule of a number header aside, and with no
// mode set and no marker, and one line says so. A marker or mode the gate
// cannot write is not a decision either: the question, already judged, runs
// and the gate says so.
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
func questionGate(cmd *cobra.Command, cwd string, raw json.RawMessage, outer error) error {
	reads := questionState(cmd, cwd)
	if reads.state.Away {
		return denyCall(awayQuestionRefusal())
	}
	if reads.state.AwayErr == nil {
		return presentQuestionGate(cmd, reads, raw, outer)
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
	err := presentQuestionGate(cmd, reads, raw, outer)
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
func presentQuestionGate(cmd *cobra.Command, reads questionReads, raw json.RawMessage, outer error) error {
	stderr := cmd.ErrOrStderr()
	err := outer
	var fields question.Fields
	if err == nil {
		fields, err = decodeQuestions(raw)
	}
	var collision *questionKeyCollision
	if errors.As(err, &collision) {
		// A key the check reads, repeated or under another case, is known
		// for certain and its remedy is to drop it, so the hook denies, as
		// the verb exits 2 on the same input: no question is shown that the
		// check judged on one value while the host could show another.
		// Refused under a mode naming somebody, it is marked open like any
		// refusal (markRefusedOpen), so the next message resets the mode.
		return denyCall(collision.refusal() + markRefusedOpen(reads))
	}
	if err != nil {
		// No marker is written here: the host's own schema rejects such an
		// input before the hook runs, so no question reaches the person.
		return questionFailOpen(stderr, "the question tool's input could not be read (%s)", err)
	}
	v := question.Decide(fields, reads.state, reads.state.Limits)
	lead := unreadLead(reads, v)
	if lead == "" {
		return actOnQuestionVerdict(cmd, reads, v, "")
	}
	// A store abcd's question was checked without could not be read: the
	// mode store, whose mode is then not set from a chip and whose person a
	// number header's register rule then lacks, or a layer of the question
	// limits, whose defaults it was then checked at.
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
// decision's order: the mode store's (modeUnreadLine) and the question
// limits' (limitsUnreadLine). The decision names the mode store only for
// abcd's question: one with no header in abcd's grammar never reads it.
func unreadLead(reads questionReads, v question.Verdict) string {
	var b strings.Builder
	for _, u := range v.Unread {
		switch u.Store {
		case question.StoreMode:
			b.WriteString(modeUnreadLine(reads.state, u))
		case question.StoreLimits:
			b.WriteString(limitsUnreadLine(reads.state.LimitsErr))
		}
	}
	return b.String()
}

// modeUnreadLine is the gate's loud line for abcd's question while the mode
// store could not be read, or, for a number header, its root could not be
// found: the question is judged by every rule it needs no mode for, the
// register rule of a number header, which follows the mode, is not applied,
// and neither the mode nor the open-question marker is written.
func modeUnreadLine(st question.State, u question.Unread) string {
	head := "abcd guard: the mode store could not be read (" + termsafe.Sanitize(scrubPaths(u.Err)) + ")"
	if st.ModeErr == nil && st.RootErr != nil {
		head = "abcd guard: the repository root could not be found, so the mode store was not read (" + termsafe.Sanitize(scrubPaths(u.Err)) + ")"
	}
	if len(u.Rules) > 0 {
		return head + ", so the register rule, which follows the mode for a question headed with abcd's number, was not applied, no mode is set and the question is not marked open; every other rule was.\n"
	}
	return head + ", so the mode is not set from this question's chip and the question is not marked open; its chip names whom it is for, so it is judged without the mode.\n"
}

// actOnQuestionVerdict does what the hook does with a decision: a deny for a
// refusal, marked open under a mode naming somebody with no mode written
// (markRefusedOpen), the marker and the mode for an admission where the badge shows,
// the rows note for an admission with rows findings, and, for a question
// that is not abcd's, only the marker under a mode naming somebody
// (markUncheckedOpen). A marker or mode it cannot write fails open loudly
// (questionFailOpen). lead, when set, opens the deny's reason and the rows
// note.
func actOnQuestionVerdict(cmd *cobra.Command, reads questionReads, v question.Verdict, lead string) error {
	stderr := cmd.ErrOrStderr()
	switch v.Outcome {
	case question.OutcomeAdmitUnchecked:
		return markUncheckedOpen(stderr, reads)
	case question.OutcomeRefuse:
		var reason strings.Builder
		reason.WriteString(lead)
		if v.Refused != nil {
			reason.WriteString(limitsRefusedReason(v.Refused.Reason))
		} else {
			writeLimitsRefusal(&reason, v.Findings)
		}
		reason.WriteString(markRefusedOpen(reads))
		return denyCall(reason.String())
	}
	if !v.Abcds {
		// Not abcd's question, unchecked because the away record was not
		// read: the caller's loud status is the rest of the answer.
		return markUncheckedOpen(stderr, reads)
	}
	if reads.state.Badge && reads.state.ModeErr == nil {
		// With the mode store unread abcd's question was still judged, and
		// runs, but nothing is written beside a store the gate could not
		// read: no marker, and no mode (modeUnreadLine says so). A
		// number-form question leaves the mode as it is (ModeAfter is the
		// mode in force) and is marked open under it, so the answer resets
		// the mode the agent set before asking (decision 37).
		want := stateOf(v.ModeAfter)
		// The marker first: a marker written for a mode that then could not
		// be set no longer names the mode in force, so the answer clears it
		// and leaves the mode alone (mode.ResetOnAnswer), while a mode set
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

// markUncheckedOpen marks a question that is not abcd's open where the badge
// shows and the mode, read, names somebody, under that mode, and writes
// nothing else: no mode, and no rule is read. The agent sets the mode before
// asking abcd's question (decision 37), so a question whose header then
// missed abcd's grammar ("Q 1", "q1", "Q1A", or none) would otherwise leave
// the mode parked, never reset by the answer, for the next question to
// inherit. Under managed, or with no badge, nothing is written. A marker that
// cannot be written is said loudly, since the mode then stays parked.
func markUncheckedOpen(stderr io.Writer, reads questionReads) error {
	if !namesSomebody(reads) {
		return nil
	}
	if err := mode.MarkQuestionOpen(reads.root, reads.mode); err != nil {
		// The question was never abcd's to check, so it ran as any other
		// tool's does; what failed is the reset, and only that is said, on
		// exit 1, loud and never blocking.
		fmt.Fprintf(stderr, "abcd guard: this question was not abcd's (no header in abcd's number grammar), so it ran as normal; but the mode, %s, could not be marked for reset at %s, so the answer will not return it to managed (%s).\n",
			reads.mode, mode.QuestionOpenRelPath, termsafe.Sanitize(scrubPaths(err)))
		return &exitError{Code: 1}
	}
	return nil
}

// namesSomebody reports whether the badge shows and the mode, read, names a
// person: the only place the gate marks a question open under the mode in
// force with no mode written.
func namesSomebody(reads questionReads) bool {
	return reads.state.Badge && reads.state.ModeErr == nil && addresseeOf(reads.mode) != question.Unnamed
}

// markRefusedOpen marks abcd's refused question open under the mode in force
// where it names somebody, and writes no mode. The agent sets the mode before
// asking (decision 37), so a refusal it then abandons would leave the mode
// parked; the marker lets the next human message's reset return it to
// managed (the host runs no PostToolUse for a denied call). It returns the
// line the deny's reason ends with when the marker cannot be written, and ""
// otherwise: the call is refused either way.
func markRefusedOpen(reads questionReads) string {
	if !namesSomebody(reads) {
		return ""
	}
	if err := mode.MarkQuestionOpen(reads.root, reads.mode); err != nil {
		return "\nabcd guard: the mode, " + string(reads.mode) + ", could not be marked for reset at " + mode.QuestionOpenRelPath +
			", so the next message will not return it to managed (" + termsafe.Sanitize(scrubPaths(err)) + ")."
	}
	return ""
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
// and was shown anyway: each tab over it, its rows and the limit, and how to
// keep the next question within it. The rows rule draws at most one finding
// per tab, and the check reads at most four tabs, so the note is bounded by
// the limits, not the payload. It never tells the agent to ask again, because
// the person may already have answered the question it was shown.
func rowsNote(findings []question.Finding) string {
	var b strings.Builder
	fmt.Fprintf(&b, "abcd guard (question tool): this question was shown, but %d tab(s) run past abcd's rows limit, so the host may cut them in a narrow window.\n", len(findings))
	for _, f := range findings {
		fmt.Fprintf(&b, "tab %d is %s; limit: %s.\n", f.Tab, f.Value, f.Limit)
	}
	b.WriteString("The question was shown this time; keep the next question within the limit by checking it with `abcd guard question` before asking it, or by drafting it through the abcd:question-drafter agent, which runs that check.")
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

// errQuestionKeyCollision is why a question whose object carries a key twice,
// or a key that matches one the check reads only under case folding, is not
// checked: encoding/json reads such keys case-insensitively, the last one
// winning, so the check would judge one value while a host reading exact keys
// shows the other. It echoes nothing from the payload; questionKeyCollision
// carries it with the one key, sanitised and bounded, that the refusal names.
var errQuestionKeyCollision = errors.New("an object of the question tool's input carries a key twice, or a key that differs from questions, header, question, options, label, description or preview only in case, so the check would read one value and the host could show another")

// questionKeyCollision is errQuestionKeyCollision for one key: key as the
// input spells it, and read, the key the check reads that it collides with
// (the same key when it is repeated). errors.Is matches it to
// errQuestionKeyCollision.
type questionKeyCollision struct {
	key, read string
}

func (c *questionKeyCollision) Error() string {
	return errQuestionKeyCollision.Error() + " (the key " + quoteKey(c.key) + ")"
}

func (c *questionKeyCollision) Is(target error) bool { return target == errQuestionKeyCollision }

// refusal is the hook's deny for the collision: it names the key and says to
// drop it.
func (c *questionKeyCollision) refusal() string {
	const head = "Blocked by the abcd guard (question tool): "
	k := quoteKey(c.key)
	if c.key == c.read {
		return head + "the key " + k + " appears twice in one object of the question tool's input, so the check would judge one value and the host could show the other. Drop one " + k + " and ask again."
	}
	return head + "the key " + k + " in the question tool's input differs from " + quoteKey(c.read) + ", a key the check reads, only in case, so the check would judge one value and the host could show the other. Drop " + k + " and ask again."
}

// quoteKey is a key from the payload as a refusal names it: sanitised, cut
// at maxQuotedKey bytes on a character boundary, and quoted.
func quoteKey(k string) string {
	k = termsafe.Sanitize(k)
	for i := range k {
		if i > maxQuotedKey {
			k = k[:i] + "..."
			break
		}
	}
	return strconv.Quote(k)
}

// maxQuotedKey bounds the key a refusal echoes from the payload.
const maxQuotedKey = 40

// questionReadKeys and optionReadKeys are the keys the check reads from a
// question and from an option, as hostQuestionInput spells them.
var (
	questionReadKeys = []string{"header", "question", "options"}
	optionReadKeys   = []string{"label", "description", "preview"}
)

// collidingKeys returns the first key of the JSON object raw that it carries
// twice, or that equals one of read under case folding but not byte for
// byte, and nil when none does. Any key that collides with nothing is kept,
// as the host owns the schema and adds to it. A value that is not an object
// collides with nothing; the decode after it decides whether it is readable.
func collidingKeys(raw json.RawMessage, read []string) *questionKeyCollision {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return nil
	}
	seen := map[string]bool{}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil
		}
		k, _ := tok.(string)
		if seen[k] {
			return &questionKeyCollision{key: k, read: k}
		}
		seen[k] = true
		for _, r := range read {
			if k != r && strings.EqualFold(k, r) {
				return &questionKeyCollision{key: k, read: r}
			}
		}
		var skip json.RawMessage
		if dec.Decode(&skip) != nil {
			return nil
		}
	}
	return nil
}

// toolInputKeysCollide reports whether a hook payload's tool_input carries
// a key twice, or its questions key under another case, which the payload
// decode would read as one of them: the hook then denies the question, after
// the away check, as it denies a collision inside a question.
func toolInputKeysCollide(payload []byte) error {
	var p struct {
		ToolInput json.RawMessage `json:"tool_input"`
	}
	if json.Unmarshal(payload, &p) != nil {
		return nil
	}
	if c := collidingKeys(p.ToolInput, []string{"questions"}); c != nil {
		return c
	}
	return nil
}

// questionKeysCollide applies collidingKeys to every question of raw and to
// every option of each, so a question judged is the question shown.
func questionKeysCollide(raw json.RawMessage) *questionKeyCollision {
	var qs []json.RawMessage
	if json.Unmarshal(raw, &qs) != nil {
		return nil
	}
	for _, q := range qs {
		if c := collidingKeys(q, questionReadKeys); c != nil {
			return c
		}
		var in struct {
			Options []json.RawMessage `json:"options"`
		}
		if json.Unmarshal(q, &in) != nil {
			continue
		}
		for _, o := range in.Options {
			if c := collidingKeys(o, optionReadKeys); c != nil {
				return c
			}
		}
	}
	return nil
}

// decodeQuestions reads the question tool's questions into the field view the
// limits check reads, one tab per question. A question or option object
// whose keys collide (questionKeysCollide) is never judged on one of its
// values: it returns the collision, which the hook denies and the verb
// refuses with exit 2.
func decodeQuestions(raw json.RawMessage) (question.Fields, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return question.Fields{}, errUnreadableQuestions
	}
	var in []hostQuestionInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return question.Fields{}, errUnreadableQuestions
	}
	if c := questionKeysCollide(raw); c != nil {
		return question.Fields{}, c
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
// names nobody, and the check then reads an older-form chip's role word
// instead.
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

// writeLimitsRefusal writes the refusal the host's deny carries
// (spc-2610301917245442, step 2): one head line counting every finding that
// refuses, then one line per refusing rule and tab
// (question.GroupFindings, written by question.RenderGroups, the renderer the
// verb's text shares), then the rows findings under rowsNotRefusingLine, and
// last the pointer at the verb and the question-writing agent
// (questionPointerLine). Every line passes termsafe.Sanitize, so no value the
// agent wrote reaches the terminal raw.
//
// The refusal names every rule on every tab it hits: there is no cut. Its
// size follows the limits, not the payload, because the check reads at most
// QuestionsPerCall[1] tabs and OptionsPerQ[1] options, and a rule that breaks
// many times on one tab (an em dash on each of a thousand list lines) is one
// line naming the first and counting the rest. So the lines are at most the
// rules times the checked tabs, the call's count finding, and the three fixed
// lines (review-askGuard-security finding 1).
//
// The rows limit alone never refuses (rowsOnly), so a rows finding is never
// one of the parts to fix and is not counted among them
// (iss-2610100626327722): listed among them, the agent read it as a cause and
// cut a question the gate would have shown. It follows the refusing parts
// instead, after rowsNotRefusingLine. It is called only with at least one
// refusing finding.
func writeLimitsRefusal(w io.Writer, findings []question.Finding) {
	refusing, rows := splitRefusing(findings)
	fmt.Fprintf(w, "Blocked by the abcd guard (question tool): %d part(s) of this question break abcd's asking rules; fix each and ask again.\n", len(refusing))
	question.RenderGroups(w, question.GroupFindings(refusing))
	if len(rows) > 0 {
		fmt.Fprintln(w, rowsNotRefusingLine)
		question.RenderGroups(w, question.GroupFindings(rows))
	}
	fmt.Fprint(w, questionPointerLine)
}

// splitRefusing parts findings into those that refuse the call and those that
// do not: the rows limit's (question.RefusingFinding).
func splitRefusing(findings []question.Finding) (refusing, rest []question.Finding) {
	for _, f := range findings {
		if question.RefusingFinding(f) {
			refusing = append(refusing, f)
		} else {
			rest = append(rest, f)
		}
	}
	return refusing, rest
}

// rowsNotRefusingLine opens the rows findings a refusal names but does not
// refuse on.
const rowsNotRefusingLine = "The rows limit does not refuse on its own: with only the rows finding(s) below, this question would have been shown, and a narrow window might cut it."

// questionPointerLine closes a refusal on the asking rules, and the verb's
// text render of a draft with findings: where the next draft is checked
// before it is asked, and the agent that drafts one through that check
// (spc-2610301917245442, step 2).
const questionPointerLine = "Check the next draft with `abcd guard question` before asking it, or draft it through the abcd:question-drafter agent, which does."

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
	return limitsUnreadLineFor(err, "the question is checked")
}

// limitsUnreadLineFor is that line for the binary's own composers (setup
// and the written interviews), which measure their questions at abcd's
// defaults rather than let them run ungated: subject says what was measured.
func limitsUnreadLineFor(err error, subject string) string {
	return "abcd guard: NOT CHECKED at this repository's question limits — they were not read (" +
		termsafe.SanitizeLine(scrubPaths(err)) + "), so " + subject + " at abcd's defaults instead.\n"
}

// limitsRefusedReason is the deny for abcd's question while a question limit
// is refused (itd-2610301914350541 decision 7): one head line saying abcd
// asks nothing until the file is fixed, then the resolver's own
// refusal naming the file, the key, the value and the bounds (or, for a
// personal value that loosens, the limit in force and the file that set it),
// and one line stating every key's bounds from question.Bounds. It names no
// asking rule, so the agent fixes the configuration rather than redrafting a
// question that may be well built.
func limitsRefusedReason(reason string) string {
	var b strings.Builder
	b.WriteString("Blocked by the abcd guard (question tool): a question limit in this repository's configuration is refused, so abcd asks no question until it is fixed.\n")
	b.WriteString(termsafe.SanitizeLine(scrubPaths(errors.New(reason))) + "\n")
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
		termsafe.SanitizeLine(scrubPaths(err)))
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
