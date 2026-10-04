package ahoy

import (
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/core/question"
)

// The setup interview's questions as the shared question type
// (spc-2610030911534855, "The interviews whose questions abcd fixes"). The
// install asks through Prompter, whose seam is a key, its choices and a
// default, or the text of an approval; a front door that draws a question, or
// writes it plainly off a terminal, or records it, builds it here, so every
// door shows and records core's own words and none invents its own (iss-163).

// SetupLaterValue is the value recorded when a setup question is answered
// "decide later". It is never handed to the install: a door answers the
// prompt with no answer (""), and the install treats that as it treats every
// unanswered question.
const SetupLaterValue = "later"

// setupAsk is the plain question every value question ends on.
const setupAsk = "Which answer does this repository take?"

// setupChip is the chip of the n-th question setup asks.
func setupChip(n int) string { return fmt.Sprintf("Setup Q%d", n) }

// SetupValueQuestion is the n-th question of setup, Prompt(key, choices, def)
// as the shared type: the key the id; the material the About of the help
// HelpIn gives for the repository at cwd (its last sentence the question,
// where it asks one); the choices the prompt offers as
// options, in its order, each labelled by its value with core's meaning
// beneath; the change-later line from ChangeLaterLine; and the decide-later
// answer, whose meaning says what the install does with no answer. For a
// config value (a question a flag answers) no answer leaves the value unset,
// so the gap stays listed; for any other question no answer takes def. A key
// core has no help for is asked bare.
func SetupValueQuestion(n int, cwd, key string, choices []string, def string) question.Question {
	h, ok := HelpIn(cwd, key)
	if !ok {
		h = PromptHelp{Key: key}
	}
	return valueQuestion(n, h, choices, def)
}

// valueQuestion is SetupValueQuestion over the help h.
//
// A help that ends on its own question (its About's last sentence asks it)
// is asked by that question, and a choice whose value is SetupLaterValue is
// the question's own decide-later answer, with its own meaning, never a
// second one beside it.
func valueQuestion(n int, h PromptHelp, choices []string, def string) question.Question {
	later := question.Option{Value: SetupLaterValue, Label: question.Default.LaterLabels[0],
		Meaning: "Leaves the value unset, so the install lists it again next time."}
	if h.Flag == "" && def != "" {
		later.Meaning = "Takes " + def + " for now, as an unanswered question does."
	}
	opts := make([]question.Option, 0, len(choices))
	for _, c := range choices {
		if c == SetupLaterValue {
			if m := h.Meaning(c); m != "" {
				later.Meaning = m
			}
			continue
		}
		opts = append(opts, question.Option{Value: c, Label: c, Meaning: h.Meaning(c)})
	}
	about, ask := h.About, setupAsk
	if body, own := splitConfirm(about); strings.HasSuffix(about, "?") && len(body) == 1 && own != "" {
		about, ask = body[0].Text, own
	}
	var material []question.Block
	if about != "" {
		material = []question.Block{{Kind: question.KindParagraph, Text: about}}
	}
	change := h.ChangeLaterLine()
	if change == "" {
		change = question.Default.NotApplicable
	}
	return question.Question{
		ID:          h.Key,
		Chip:        setupChip(n),
		Material:    material,
		Ask:         ask,
		Options:     opts,
		Later:       later,
		Now:         "not set",
		ChangeLater: change,
	}
}

// setupConfirmIDs gives each approval setup asks its stable id, matched on
// the text core composes, so an answers file can answer it by id. A text that
// matches none (a verb outside setup asking through the same prompter) has no
// id, and the door numbers it.
var setupConfirmIDs = []struct {
	id    string
	match func(string) bool
}{
	{"adopt", func(s string) bool { return s == "Adopt this unmanaged repo into abcd?" }},
	{OracleRoutingMachineGapID, func(s string) bool { return s == machineRoutingQuestion() }},
	{OracleRoutingRepoGapID, func(s string) bool { return s == repoRoutingQuestion() }},
	{StatusLineOfferGapID, func(s string) bool { return s == statusLineOfferQuestion }},
	{DrainRuleOfferGapID, func(s string) bool { return s == drainRuleQuestion() }},
	{"history.lineage", func(s string) bool {
		return strings.HasPrefix(s, "Re-founded from ") && strings.HasSuffix(s, "? Link lineage?")
	}},
	{"git_identity.establish", func(s string) bool { return strings.HasPrefix(s, "Commit to this repository as ") }},
	{"remote.apply", func(s string) bool { return strings.HasPrefix(s, "Change GitHub settings on ") }},
}

// approvePrefix ids a category approval: "Apply <category> changes?" is
// approve.<category>.
const approvePrefix = "approve."

// setupConfirmID is the id of the approval whose text is text, or "".
func setupConfirmID(text string) string {
	if c, ok := strings.CutPrefix(text, "Apply "); ok {
		if cat, ok := strings.CutSuffix(c, " changes?"); ok && !strings.ContainsAny(cat, " ?") {
			return approvePrefix + cat
		}
	}
	for _, c := range setupConfirmIDs {
		if c.match(text) {
			return c.id
		}
	}
	return ""
}

// SetupConfirmQuestion is the n-th question of setup, Confirm(text) as the
// shared type: its id the approval's (setupConfirmID; "" for a text setup
// does not own); the text split into the material and the one plain question
// it ends on (the last line, or else the last sentence of a text that ends in
// a question, a line indented by two spaces an item of a list); and the
// answers "Yes, make the change" and "No, leave it", with deciding later
// declining.
func SetupConfirmQuestion(n int, text string) question.Question {
	material, ask := splitConfirm(text)
	return question.Question{
		ID:       setupConfirmID(text),
		Chip:     setupChip(n),
		Material: material,
		Ask:      ask,
		Options: []question.Option{
			{Value: "yes", Label: "Yes, make the change", Meaning: "Writes what the text above describes."},
			{Value: "no", Label: "No, leave it", Meaning: "Writes nothing, so the next install asks again."},
		},
		Later: question.Option{Value: SetupLaterValue, Label: question.Default.LaterLabels[0],
			Meaning: "Declines for now and writes nothing, so the next install asks again."},
		Now:         question.Default.NotApplicable,
		ChangeLater: question.Default.NotApplicable,
	}
}

// splitConfirm splits an approval's text into its material and its question:
// the question is the last line, or, in a one-line text that ends in a
// question, its last sentence; a text with no question to split off is asked
// whole.
func splitConfirm(text string) ([]question.Block, string) {
	body, ask := "", text
	if i := strings.LastIndex(text, "\n"); i >= 0 {
		body, ask = text[:i], text[i+1:]
	} else if strings.HasSuffix(text, "?") {
		if i := strings.LastIndex(text, ". "); i >= 0 {
			body, ask = text[:i+1], text[i+2:]
		}
	}
	var blocks []question.Block
	var items []string
	flush := func() {
		if len(items) > 0 {
			blocks = append(blocks, question.Block{Kind: question.KindList, Items: items})
			items = nil
		}
	}
	for _, ln := range strings.Split(body, "\n") {
		if strings.HasPrefix(ln, "  ") {
			items = append(items, strings.TrimSpace(ln))
			continue
		}
		flush()
		if strings.TrimSpace(ln) != "" {
			blocks = append(blocks, question.Block{Kind: question.KindParagraph, Text: ln})
		}
	}
	flush()
	return blocks, strings.TrimSpace(ask)
}

// SetupMachineWide reports whether the setup question id changes the machine
// rather than the repository: the status line, which every repository on the
// machine shows, and the machine's routing table. Its answer is recorded in
// the home's interviews, the rest in the repository's local tier.
func SetupMachineWide(id string) bool {
	return strings.HasPrefix(id, elementPromptPrefix) || id == OracleRoutingMachineGapID
}

// SetupFixedValues returns the values an answers file may give the setup
// question id when that question's answers are fixed before the run, as
// SetupValueQuestion and SetupConfirmQuestion build them (the options, then
// deciding later): a config value's, the house-style question's, a status
// line element's, and every approval's. ok is false for a question whose
// answers the run decides (the artefact kind, a conventions file's
// retirement) or an id setup does not ask.
func SetupFixedValues(id string) (values []string, ok bool) {
	var q question.Question
	switch {
	case id == "adopt", strings.HasPrefix(id, approvePrefix), confirmIDKnown(id):
		q = SetupConfirmQuestion(0, "")
	case strings.HasPrefix(id, elementPromptPrefix):
		q = valueQuestion(0, PromptHelp{Key: id}, []string{"on", "off"}, "on")
	default:
		choices, known := fixedValueChoices[id]
		if !known {
			return nil, false
		}
		q = valueQuestion(0, PromptHelp{Key: id}, choices, "")
	}
	for _, o := range append(append([]question.Option(nil), q.Options...), q.Later) {
		values = append(values, o.Value)
	}
	return values, true
}

// fixedValueChoices are the choices the value questions with a fixed set
// offer, keyed by the key the install asks them by.
var fixedValueChoices = map[string][]string{
	"visibility":     visibilityChoices,
	"docs_target":    docsTargetWritable,
	"oracle_backend": oracleBackendChoices,
	"scan_deep":      scanDeepChoices,
	emDashPromptKey:  emDashChoices,
}

// confirmIDKnown reports whether id is one of the approvals setupConfirmIDs
// names.
func confirmIDKnown(id string) bool {
	for _, c := range setupConfirmIDs {
		if c.id == id {
			return true
		}
	}
	return false
}

// SetupAskedBeforeWriting reports whether the install asks the setup question
// id before its first write: the adoption and the approvals of each kind of
// change, which install asks before its first apply step. A stop at one of
// them leaves the repository untouched; a stop at any other question leaves
// what the steps before it changed.
func SetupAskedBeforeWriting(id string) bool {
	return id == "adopt" || strings.HasPrefix(id, approvePrefix)
}
