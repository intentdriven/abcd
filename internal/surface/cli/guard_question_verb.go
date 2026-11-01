package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// `abcd guard question` (spc-2610301917245442, step 1; itd-2610230756259215
// decisions 1 to 10): the read-only sibling of `guard check` for a draft
// question. It reads the draft, makes the store reads the hook makes
// (questionState), calls the one decision the hook calls (question.Decide),
// and reports every finding, the rows of each tab, what the hook would do with
// the draft in this checkout now, the binary's version, and the input ready to
// ask. It writes nothing: no open-question marker, no mode, no branch record,
// no log (decision 9), because a check that marked a question open would leave
// the status line naming a person for a question never asked.

// questionCheckType is the JSON result's _type.
const questionCheckType = "abcd/question-check/v1"

// The two input forms the verb reads (decision 4). The flag chooses, and
// nothing is guessed: both open with a questions key, and a wrong guess
// reports the wrong faults.
const (
	questionFormHost = "host"
	questionFormAsk  = "ask"
)

// questionCheckResult is the verb's JSON (decision 6): the fields the drafter
// copies rather than translates.
type questionCheckResult struct {
	Type             string                  `json:"_type"`
	Version          string                  `json:"version"`
	Form             string                  `json:"form"`
	Hook             question.Outcome        `json:"hook"`
	Abcds            bool                    `json:"abcds"`
	DependsOnState   bool                    `json:"depends_on_state"`
	Partial          bool                    `json:"partial"`
	Unread           []questionUnreadJSON    `json:"unread"`
	Addressee        []questionAddresseeJSON `json:"addressee"`
	AddresseeChanges string                  `json:"addressee_changes,omitempty"`
	Findings         []questionFindingJSON   `json:"findings"`
	Rows             []questionRowsJSON      `json:"rows"`
	RowsLimit        int                     `json:"rows_limit"`
	// Limits and LimitsSource are the question limits the check judged at,
	// as the hook judges at them, and where each came from
	// (spc-2610301950037469 step 2). Both are absent while a limit is
	// refused, since no limit is then in force and Refused says why.
	Limits       *questionLimitsJSON       `json:"limits,omitempty"`
	LimitsSource *questionLimitsSourceJSON `json:"limits_source,omitempty"`
	Refused      *questionRefusedJSON      `json:"refused,omitempty"`
	Ready        json.RawMessage           `json:"ready,omitempty"`
}

type questionUnreadJSON struct {
	Store string   `json:"store"`
	Why   string   `json:"why"`
	Rules []string `json:"rules"`
	// File is the file that could not be read, for the question limits.
	File string `json:"file,omitempty"`
}

// questionLimitsJSON is the three figures a repository sets, in force.
type questionLimitsJSON struct {
	Rows                 int `json:"rows"`
	LabelWords           int `json:"label_words"`
	DescriptionSentences int `json:"description_sentences"`
}

// questionLimitsSourceJSON is where each figure came from: "bundled", the
// repository's .abcd/config.json, or the person's own
// ~/.abcd.noindex/config.json where a personal value tightened it.
type questionLimitsSourceJSON struct {
	Rows                 string `json:"rows"`
	LabelWords           string `json:"label_words"`
	DescriptionSentences string `json:"description_sentences"`
}

// questionRefusedJSON is a question limit read and refused: the key, the
// value, the bounds and the file, and for a personal value that loosens the
// limit in force and the file that set it. Why is the resolver's whole
// refusal, which alone names an unclaimed key.
type questionRefusedJSON struct {
	Key       string `json:"key,omitempty"`
	Value     string `json:"value,omitempty"`
	Bounds    string `json:"bounds,omitempty"`
	File      string `json:"file,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	LimitFile string `json:"limit_file,omitempty"`
	Why       string `json:"why"`
}

type questionAddresseeJSON struct {
	Tab    int    `json:"tab"`
	Person string `json:"person"`
}

type questionFindingJSON struct {
	Tab     int    `json:"tab"`
	Part    string `json:"part"`
	Rule    string `json:"rule"`
	Value   string `json:"value"`
	Limit   string `json:"limit"`
	Remedy  string `json:"remedy"`
	Refuses bool   `json:"refuses"`
}

type questionRowsJSON struct {
	Tab int `json:"tab"`
	question.RowSplit
}

// hostQuestionOutput is one question of the host question tool's input as
// the verb composes it from abcd's structured type: the host's own key names,
// and multiSelect, which the host's tool takes on every question.
type hostQuestionOutput struct {
	Header      string             `json:"header"`
	Question    string             `json:"question"`
	MultiSelect bool               `json:"multiSelect"`
	Options     []hostOptionOutput `json:"options"`
}

type hostOptionOutput struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// errQuestionDraftShape is why a host-form draft cannot be judged. It names
// the shape the verb reads and echoes nothing from the draft.
var errQuestionDraftShape = errors.New(`the draft is not the host question tool's input, {"questions": [...]}, each question a header, a question and options with a label and a description; a draft in abcd's structured question type is read with --ask`)

// errQuestionAskShape is the same for a draft read with --ask.
var errQuestionAskShape = errors.New(`the draft is not abcd's structured question type, {"questions": [...]}, each question an id, a chip, material, an ask, options and a later option, with no other key`)

func newGuardQuestionCommand(asJSON *bool) *cobra.Command {
	var ask bool
	cmd := &cobra.Command{
		Use: "question [<file>|-]",
		Long: "Checks a draft question as the guard hook will judge it when it is asked,\n" +
			"and writes nothing: no mode, no open-question marker, no branch record and\n" +
			"no log. The draft is read from the file named, or from stdin when the\n" +
			"argument is - or absent, under the hook's own 1 MiB bound. Without --ask it\n" +
			"is the host question tool's input, {\"questions\": [...]}, read as the hook\n" +
			"reads it; with --ask it is abcd's structured question type, decoded\n" +
			"strictly, its structural findings reported under the rule structure, and\n" +
			"composed into the host's input (the material first, the Now: and Change\n" +
			"later: lines next, the question last, the decide-later option last) before\n" +
			"it is judged. A line the draft does not carry is never invented: the\n" +
			"check names it.\n\n" +
			"The verb reads the stores the hook reads (the away record, and the mode\n" +
			"where the status line's badge shows) and calls the one decision the hook\n" +
			"calls, so it reports every finding, the rows each tab takes at 80 columns\n" +
			"(header and frame, question text, options, total), what the hook would do\n" +
			"with the draft in this checkout now (admit, admit-with-rows-note, refuse,\n" +
			"refuse-away, admit-unchecked or unchecked), whether that answer depends on\n" +
			"the mode or the away record, the binary's version, and the input ready to\n" +
			"ask. The ready input is the draft's own bytes for the host form, every key\n" +
			"the check does not read kept, and the composed input for --ask; it is left\n" +
			"out when the hook would refuse. The verdict is evidence about a hook of the\n" +
			"same version; a mode set, or the person marked away, before the ask can\n" +
			"change it.\n\n" +
			"The limits judged at are the ones the hook holds abcd's question to in\n" +
			"this checkout: abcd's defaults, with the rows, label words and description\n" +
			"sentences the repository's .abcd/config.json sets under question, each\n" +
			"tightened where the person's own " + abcdhome.Display(layered.Config.MachineRel) + " sets it\n" +
			"lower. The result reports them (limits) and the file that set each\n" +
			"(limits_source). A limit the hook refuses (a value outside its bounds, not\n" +
			"a whole number, another key under question, or a personal value looser\n" +
			"than the limit in force) is reported as refused, with no ready input, and\n" +
			"a file that could not be read is named in the partial result, the draft\n" +
			"checked at abcd's defaults, as the hook checks it.\n\n" +
			"Exit 0 when the draft has no finding at all and every store was read,\n" +
			"whatever the hook would do with it: the hook field says whether that is\n" +
			"an admission, a refusal while the person is away, or a pass unchecked as\n" +
			"not abcd's. Exit 1 on any finding, a tab over the rows limit alone\n" +
			"included.\n" +
			"Exit 2 when the draft is not JSON, is over the bound or has neither shape,\n" +
			"when a question limit is refused, whatever the draft, and when a store the\n" +
			"decision needed, the question limits' files included, could not be read:\n" +
			"the result is then partial, names each store and the rules it could not\n" +
			"apply or applied at the defaults, and never reads as clean, even with no\n" +
			"finding.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			src := "-"
			if len(args) == 1 {
				src = args[0]
			}
			raw, err := readQuestionDraft(cmd, src)
			if err != nil {
				return &exitError{Code: 2, Msg: "guard question: " + scrubPaths(err)}
			}
			form, fields, structural, ready, err := decodeQuestionDraft(raw, ask)
			if err != nil {
				return &exitError{Code: 2, Msg: "guard question: " + err.Error()}
			}
			cwd, _ := os.Getwd()
			reads := questionState(cmd, cwd)
			v := question.Decide(fields, reads.state, reads.state.Limits)
			res := questionCheckOf(form, v, structural, ready, reads)
			if rerr := render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				writeQuestionCheck(w, res)
			}); rerr != nil {
				return rerr
			}
			return questionCheckExit(res)
		},
	}
	cmd.Flags().BoolVar(&ask, "ask", false, "read the draft as abcd's structured question type and judge it as composed into the host's input")
	return cmd
}

// readQuestionDraft reads the draft from the file named, or stdin for "-",
// one byte past the hook's bound, so an over-bound draft is named as such
// rather than misread as broken JSON.
func readQuestionDraft(cmd *cobra.Command, src string) ([]byte, error) {
	var r io.Reader = cmd.InOrStdin()
	if src != "-" {
		f, err := os.Open(src)
		if err != nil {
			return nil, fmt.Errorf("the draft could not be opened: %w", err)
		}
		defer f.Close()
		r = f
	}
	raw, err := io.ReadAll(io.LimitReader(r, maxHookStdinBytes+1))
	if err != nil {
		return nil, fmt.Errorf("the draft could not be read: %w", err)
	}
	if len(raw) > maxHookStdinBytes {
		return nil, fmt.Errorf("the draft is over the hook's %d-byte bound; it was not checked", maxHookStdinBytes)
	}
	return raw, nil
}

// decodeQuestionDraft reads the draft in the form the flag names into the
// field view the decision reads, with the structural findings of an --ask
// draft and the input ready to ask.
func decodeQuestionDraft(raw []byte, ask bool) (form string, f question.Fields, structural []question.Finding, ready json.RawMessage, err error) {
	if !json.Valid(raw) {
		shape := errQuestionDraftShape
		if ask {
			shape = errQuestionAskShape
		}
		return "", f, nil, nil, fmt.Errorf("the draft is not JSON; %w", shape)
	}
	if !ask {
		var in struct {
			Questions json.RawMessage `json:"questions"`
		}
		if json.Unmarshal(raw, &in) != nil || len(in.Questions) == 0 {
			return "", f, nil, nil, errQuestionDraftShape
		}
		if f, err = decodeQuestions(in.Questions); err != nil {
			return "", f, nil, nil, errQuestionDraftShape
		}
		// The draft's own bytes: rebuilding the input from the decoded view
		// would drop keys the check does not read, such as multiSelect,
		// which the host still needs (decision 7).
		return questionFormHost, f, nil, json.RawMessage(raw), nil
	}
	var a question.Ask
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(&a) != nil || dec.More() {
		return "", f, nil, nil, errQuestionAskShape
	}
	f = a.Fields()
	out := struct {
		Questions []hostQuestionOutput `json:"questions"`
	}{Questions: make([]hostQuestionOutput, len(f.Tabs))}
	for i, t := range f.Tabs {
		q := hostQuestionOutput{Header: t.Header, Question: t.Text, Options: make([]hostOptionOutput, len(t.Options))}
		for j, o := range t.Options {
			q.Options[j] = hostOptionOutput{Label: o.Label, Description: o.Description}
		}
		out.Questions[i] = q
	}
	composed, err := json.Marshal(out)
	if err != nil {
		return "", f, nil, nil, fmt.Errorf("the draft could not be composed into the host's input: %w", err)
	}
	return questionFormAsk, f, question.Check(a), composed, nil
}

// personName is the mode's word for a person, the word `abcd mode` takes.
func personName(p question.Person) string {
	if p == question.Unnamed {
		return "unnamed"
	}
	return string(stateOf(p))
}

// questionCheckOf builds the result from the verdict.
func questionCheckOf(form string, v question.Verdict, structural []question.Finding, ready json.RawMessage, reads questionReads) questionCheckResult {
	res := questionCheckResult{
		Type:           questionCheckType,
		Version:        core.Version,
		Form:           form,
		Hook:           v.Outcome,
		Abcds:          v.Abcds,
		DependsOnState: v.DependsOnState,
		Partial:        v.Partial,
		Unread:         []questionUnreadJSON{},
		Addressee:      []questionAddresseeJSON{},
		Findings:       []questionFindingJSON{},
		Rows:           []questionRowsJSON{},
		RowsLimit:      reads.state.Limits.Rows,
	}
	limitsUnreadSeen := false
	for _, u := range v.Unread {
		rules := make([]string, len(u.Rules))
		for i, r := range u.Rules {
			rules[i] = string(r)
		}
		res.Unread = append(res.Unread, questionUnreadJSON{Store: string(u.Store), Why: termsafe.Sanitize(scrubPaths(u.Err)), Rules: rules, File: u.File})
		limitsUnreadSeen = limitsUnreadSeen || u.Store == question.StoreLimits
	}
	if st := reads.state; st.LimitsErr != nil && !limitsUnreadSeen {
		// A draft that is not abcd's is not checked, so the decision read no
		// limit; the verb still says the limits could not be read, since a
		// draft relabelled as abcd's would meet them.
		res.Partial = true
		res.Unread = append(res.Unread, questionUnreadJSON{Store: string(question.StoreLimits), Why: termsafe.Sanitize(scrubPaths(st.LimitsErr)), Rules: []string{}, File: st.LimitsFile})
	}
	if r := reads.state.LimitsRefused; r != nil {
		res.Refused = &questionRefusedJSON{Key: r.Key, Value: termsafe.Sanitize(r.Value), Bounds: r.Bounds, File: r.File,
			Limit: r.Limit, LimitFile: r.LimitFile, Why: termsafe.Sanitize(scrubPaths(errors.New(r.Reason)))}
	} else {
		c := reads.state.Limits.Configurable()
		res.Limits = &questionLimitsJSON{Rows: c[0], LabelWords: c[1], DescriptionSentences: c[2]}
		src := make([]string, 3)
		for i := range src {
			src[i] = "bundled"
			if i < len(reads.state.LimitsSource) {
				src[i] = reads.state.LimitsSource[i]
			}
		}
		res.LimitsSource = &questionLimitsSourceJSON{Rows: src[0], LabelWords: src[1], DescriptionSentences: src[2]}
	}
	for i, p := range v.Addressees {
		res.Addressee = append(res.Addressee, questionAddresseeJSON{Tab: i + 1, Person: personName(p)})
	}
	if v.ModeChanges && !v.Outcome.Refuses() {
		res.AddresseeChanges = fmt.Sprintf("asked now, this question sets the mode from %s to %s, from its chip", reads.mode, stateOf(v.ModeAfter))
	}
	for _, f := range structural {
		res.Findings = append(res.Findings, findingJSON(f, false))
	}
	for _, f := range v.Findings {
		res.Findings = append(res.Findings, findingJSON(f, question.RefusingFinding(f)))
	}
	for i, r := range v.Rows {
		res.Rows = append(res.Rows, questionRowsJSON{Tab: i + 1, RowSplit: r})
	}
	if !v.Outcome.Refuses() && res.Refused == nil {
		res.Ready = ready
	}
	return res
}

func findingJSON(f question.Finding, refuses bool) questionFindingJSON {
	return questionFindingJSON{Tab: f.Tab, Part: f.Part, Rule: string(f.Rule), Value: f.Value, Limit: f.Limit, Remedy: f.Remedy, Refuses: refuses}
}

// questionCheckExit maps the result onto the verb's exit (decision 8): 2 for
// a refused question limit and for a partial result, whatever it found; 1
// for any finding; 0 for a result with no finding at all and every store
// read. The exit answers whether the draft
// has anything to fix, not what the hook does with it: a draft refused while
// the person is away, or let through unchecked as not abcd's, carries no
// finding, so it exits 0 and the hook field names the outcome.
func questionCheckExit(res questionCheckResult) error {
	if res.Refused != nil {
		return &exitError{Code: 2, Msg: "guard question: a question limit is refused (" + res.Refused.Why +
			"), so the hook refuses every abcd question until the file is fixed; no draft is ready"}
	}
	if res.Partial {
		stores := make([]string, len(res.Unread))
		for i, u := range res.Unread {
			stores[i] = "the " + u.Store + " store"
			if u.Store == string(question.StoreLimits) {
				stores[i] = "the question limits in " + u.File
			}
		}
		return &exitError{Code: 2, Msg: "guard question: the check is partial: " + strings.Join(stores, " and ") +
			" could not be read, so this draft is not known to be ready"}
	}
	if len(res.Findings) > 0 {
		return &exitError{Code: 1}
	}
	return nil
}

// outcomeLine says what the hook would do with the draft, in a person's words.
func outcomeLine(o question.Outcome) string {
	switch o {
	case question.OutcomeAdmit:
		return "the hook would let this question through"
	case question.OutcomeAdmitWithRowsNote:
		return "the hook would show this question, with a note that a tab runs past the rows limit"
	case question.OutcomeRefuse:
		return "the hook would refuse this question"
	case question.OutcomeRefuseAway:
		return "the person is away (abcd mode autonomous), so the hook would refuse every question; route it with abcd mode queue or abcd decide"
	case question.OutcomeAdmitUnchecked:
		return "this is not abcd's question (no header in abcd's chip grammar, and no mode naming anyone), so the hook would let it through unchecked and nothing was checked; head it with abcd's chip, such as \"Product Q1\", to have it checked"
	case question.OutcomeUnchecked:
		return "a store the check needed could not be read, so the hook would let this question through unchecked, on its loud status"
	}
	return string(o)
}

// outcomeLineOf is outcomeLine for the result, saying so where the question
// limits decided the outcome: a refused limit refuses the draft whatever it
// holds, and limits that could not be read leave it checked at the defaults
// rather than unchecked.
func outcomeLineOf(res questionCheckResult) string {
	switch {
	case res.Refused != nil && res.Hook == question.OutcomeRefuse:
		return "a question limit in this repository's configuration is refused, so the hook would refuse this question, and every abcd question, until the file is fixed"
	case res.Hook == question.OutcomeUnchecked && len(res.Unread) > 0 && !slices.ContainsFunc(res.Unread, func(u questionUnreadJSON) bool {
		return u.Store != string(question.StoreLimits)
	}):
		return "the question limits could not be read, so the hook would check this question at abcd's defaults and let it through on its loud status"
	}
	return outcomeLine(res.Hook)
}

// limitsLine is the text render's statement of the question limits: each
// figure in force, the file that set it after each figure a file set and
// nothing after a default, or the refusal.
func limitsLine(res questionCheckResult) string {
	if res.Refused != nil {
		return "limits: refused: " + res.Refused.Why + "; the hook refuses every abcd question until the file is fixed"
	}
	l, s := res.Limits, res.LimitsSource
	part := func(n int, what, src string) string {
		p := fmt.Sprintf("%d %s", n, what)
		if src != "bundled" {
			p += " (" + termsafe.Sanitize(src) + ")"
		}
		return p
	}
	return "limits: " + part(l.Rows, "rows", s.Rows) + ", " + part(l.LabelWords, "label words", s.LabelWords) + ", " +
		part(l.DescriptionSentences, "description sentences", s.DescriptionSentences)
}

// writeQuestionCheck is the text render: the outcome, the rows of each tab,
// every finding, what could not be read, and the ready input, verbatim.
func writeQuestionCheck(w io.Writer, res questionCheckResult) {
	fmt.Fprintf(w, "hook: %s: %s.\n", res.Hook, outcomeLineOf(res))
	fmt.Fprintln(w, limitsLine(res))
	fmt.Fprintf(w, "version: %s\n", termsafe.Sanitize(res.Version))
	for _, a := range res.Addressee {
		fmt.Fprintf(w, "tab %d is for: %s\n", a.Tab, a.Person)
	}
	if res.AddresseeChanges != "" {
		fmt.Fprintln(w, res.AddresseeChanges+".")
	}
	fmt.Fprintf(w, "rows (limit %d at %d columns):\n", res.RowsLimit, question.Default.Columns)
	for _, r := range res.Rows {
		fmt.Fprintf(w, "  tab %d: %d rows: header and frame %d, question text %d, options %d\n",
			r.Tab, r.Total, r.HeaderAndFrame, r.QuestionText, r.Options)
	}
	refusing := 0
	for _, f := range res.Findings {
		if f.Refuses {
			refusing++
		}
	}
	fmt.Fprintf(w, "findings: %d, %d of them refusing\n", len(res.Findings), refusing)
	for _, f := range res.Findings {
		fmt.Fprintln(w, "  "+termsafe.Sanitize(question.Finding{Tab: f.Tab, Part: f.Part, Rule: question.Rule(f.Rule), Value: f.Value, Limit: f.Limit, Remedy: f.Remedy}.String()))
	}
	for _, u := range res.Unread {
		rules := "none"
		if len(u.Rules) > 0 {
			rules = strings.Join(u.Rules, ", ")
		}
		if u.Store == string(question.StoreLimits) {
			fmt.Fprintf(w, "partial: the question limits were not read (%s); rules checked at abcd's defaults instead: %s\n", u.Why, rules)
			continue
		}
		fmt.Fprintf(w, "partial: the %s store was not read (%s); rules not applied: %s\n", u.Store, u.Why, rules)
	}
	if res.DependsOnState {
		fmt.Fprintln(w, "This answer holds for the checkout as it is now: a mode set, or the person marked away, before the ask can change it.")
	}
	if len(res.Ready) > 0 {
		// The ready input as the draft wrote it, each line through the
		// terminal mask, so a draft free of control and hidden runes prints
		// byte for byte.
		fmt.Fprintln(w, "ready:")
		fmt.Fprintln(w, termsafe.SanitizeBlock(strings.TrimRight(string(res.Ready), "\n")))
	}
}
