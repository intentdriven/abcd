package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/mode"
	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// The question check, the question gate and its reset (itd-2609212130146198;
// spc-2610030944505997).
//
// The guard hook, on the host's question tool, holds abcd's own questions to
// the field limits of internal/core/question wherever it runs, and refuses a
// badly built one naming each part to fix. In a checkout abcd manages it also
// refuses abcd's question while the mode reads managed — the agent has not
// said whom it is asking, so the badge would read "nobody is waiting" while
// somebody is — and marks an admitted question open. The prompt hook, on the
// next human message, resets the mode to managed when a question is marked
// open, because that message is the answer. Another tool's question is none of
// this file's business and runs unchecked.
//
// This file is the whole of the feature on the front-door side. The guard's
// own hook calls questionGate once, before its shell-command path, and touches
// nothing of the shell guard's tokenizer or decision.

// questionTools are the host tool names that put a question to the human. The
// hook manifest's PreToolUse matcher names exactly these beside the shell tool
// (TestGuardHookIsInstalledForBashCalls holds the two together).
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

// questionRefusal is the one line the host replays to the agent when it asks
// while the mode reads managed. It names the two settings and the verb, and
// reminds the agent that choosing is its job.
const questionRefusal = "Blocked by the abcd guard (question tool): the mode reads managed, so the status line says nobody is waiting. " +
	"Before asking, say whom the question is for: run `abcd mode product-thinker` if it is for the product thinker, " +
	"or `abcd mode facilitator` if it is for the technical facilitator, then ask again. " +
	"The mode resets to managed on the next human message."

// questionGate is the guard hook's answer for a question-tool call
// (itd-2609212130146198; spc-2610030944505997, "The question check in the
// guard hook"). It never rewrites the question: it admits it or refuses it
// with the host's deny, and it never puts a replacement input on stdout. A
// question whose only finding is the rows limit is admitted with a note for
// the agent (rowsNote).
//
// The order is the spec's. First the questions are decoded; a field the check
// cannot read is not a decision, so the question runs on the loud, non-blocking
// status. Then the gate decides whether the question is abcd's: a header in
// abcd's chip grammar, which only abcd's interview pages are taught to write,
// or a mode naming somebody, which only abcd's mode verb sets (itd-201 decision
// 10). Anything else is another tool's question and runs, unchecked and
// unmarked, wherever it is asked.
//
// abcd's question is held to the field limits wherever the hook runs, managed
// or not: the setup interview asks before a repository is managed, and the
// limits need no store. Where the badge shows (a checkout abcd manages, with the
// local tier the mode verb writes to), the mode gate runs as well: an abcd
// question while the mode reads managed is refused, naming `abcd mode`, and an
// admitted one is marked open, so the next human message resets the mode. A
// store or marker the gate cannot read or write is not a decision: the question
// runs and the gate says so, the guard's fail-open-loud contract. A tier the
// verb cannot write is the same case seen from the refusal's side: a remedy that
// cannot run would hold the question refused forever, so there the mode gate
// stands down, loudly, and only the field findings, whose remedy is the
// agent's own, can refuse.
func questionGate(cmd *cobra.Command, cwd string, raw json.RawMessage) error {
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
	chip := hasAbcdChip(fields)
	if !chip && st == mode.Managed {
		return nil
	}

	findings := question.CheckLimits(fields, question.Default, question.Addressee{
		Person: addresseeOf(st),
		Verbs:  verbsOf(cmd.Root()),
	})
	// The rows limit alone does not refuse (iss-2610070637562567): a question
	// too tall for the narrow window is shown, and the agent is told
	// afterwards. Refusing it made the agent redraft a question the person
	// was ready to answer.
	refuses := len(findings) > 0 && !rowsOnly(findings)
	modeRefuses := false
	if badge && st == mode.Managed {
		// A refusal whose remedy cannot run would refuse this question
		// forever (iss-2609260100382261), so the gate refuses on the mode
		// only where the verb it names could set the state.
		if err := mode.CanSet(root); err != nil {
			if !refuses {
				return questionFailOpen(stderr, "the mode reads managed but cannot be set here, so `abcd mode` could not answer a refusal (%s)", err)
			}
		} else {
			modeRefuses = true
		}
	}
	if refuses || modeRefuses {
		var reason strings.Builder
		if len(findings) > 0 {
			why := ""
			if !chip {
				why = modeMadeAbcds(st)
			}
			writeLimitsRefusal(&reason, findings, why)
		}
		if modeRefuses {
			fmt.Fprintln(&reason, questionRefusal)
		}
		return denyCall(reason.String())
	}
	if badge {
		if err := mode.MarkQuestionOpen(root, st); err != nil {
			return questionFailOpen(stderr, "the question could not be marked open, so the mode will not reset on the answer (%s)", err)
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

// hasAbcdChip reports whether any tab's header is in abcd's chip grammar.
func hasAbcdChip(f question.Fields) bool {
	for _, t := range f.Tabs {
		if _, ok := question.ChipRole(t.Header, question.Default); ok {
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

// writeLimitsRefusal writes the refusal the host's deny carries: one
// head line counting every part, then, when why is set, the one line saying
// why a question without abcd's chip is abcd's, then one line per finding naming the tab, the
// part, the value, the limit and the remedy, at most maxRefusalParts of them,
// and one closing line counting the parts not named. Every line passes
// termsafe.Sanitize, so no value the agent wrote reaches the terminal raw.
func writeLimitsRefusal(w io.Writer, findings []question.Finding, why string) {
	fmt.Fprintf(w, "Blocked by the abcd guard (question tool): %d part(s) of this question break abcd's asking rules; fix each and ask again.\n", len(findings))
	if why != "" {
		fmt.Fprintln(w, why)
	}
	for _, f := range findings[:min(len(findings), maxRefusalParts)] {
		fmt.Fprintln(w, termsafe.Sanitize(f.String()))
	}
	if more := len(findings) - maxRefusalParts; more > 0 {
		fmt.Fprintf(w, "... and %d more part(s); fix these and ask again to see the rest.\n", more)
	}
}

// questionFailOpen is the gate's failOpen: exit 1, which lets the question run
// and keeps the warning in front of a human.
func questionFailOpen(w io.Writer, format string, err error) error {
	fmt.Fprintf(w, "abcd guard: NOT CHECKED — "+format+". The question runs UNGATED.\n",
		termsafe.Sanitize(scrubPaths(err)))
	return &exitError{Code: 1}
}

// resetModeOnAnswer is the prompt hook's half: when a question is marked open,
// this human message is its answer, so the mode goes back to managed, the
// marker is cleared, and one line on stderr says so. Nothing goes to stdout,
// which the host injects into the session's context. Every failure is named
// and never stops the prompt.
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
