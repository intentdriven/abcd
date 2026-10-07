package ahoy

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/question"
)

// The approval of each kind of change (iss-2610071528375981). The install
// asks one approval per category of change present, and the approval shows
// what it would do: one line per change, the gap's title, which names the
// file or the setting, then the question in plain words, never the
// category's internal name. The words go through Prompter.Confirm as one
// text, every line but the last an item of the list (indented two spaces, as
// splitConfirm reads it), so every door shows the same list: the drawn
// question at a terminal, the plain one an answers file is read against, and
// the line-per-answer stream.

// approvalWords is how the approval of one category is asked: the question,
// the label of yes, and whether yes writes the listed changes itself (else it
// goes on to the questions that decide each, and yes says so).
type approvalWords struct {
	ask    string
	yes    string // the yes label
	writes bool
	// next is what yes does where it does not write: the questions it goes
	// on to.
	next string
}

// writesLabel and nextLabel are the two labels yes takes.
const (
	writesLabel = "Yes, make the changes"
	nextLabel   = "Yes, go on"
)

// categoryApprovalWords is each category's approval. A category missing here
// is asked by its name (categoryWords), so a category added to the type is
// still asked with its list.
var categoryApprovalWords = map[GapCategory]approvalWords{
	Dependency: {ask: "Offer to install the tools listed above?", yes: nextLabel,
		next: "Asks before installing each tool listed above, one question per tool."},
	SafeAutocreate: {ask: "Create the files and folders listed above?", yes: writesLabel, writes: true},
	ConfigChange: {ask: "Change the settings listed above?", yes: nextLabel,
		next: "Asks first for each value not yet chosen; a setting left unanswered is not written."},
	StatusLine: {ask: "Go on to the status line offer listed above?", yes: nextLabel,
		next: "Asks next whether to install it and what it shows; nothing is written without a yes there."},
	OracleRouting: {ask: "Go on to the model routing offers listed above?", yes: nextLabel,
		next: "Asks about each offer listed above in turn; nothing is written without a yes there."},
	DrainRule: {ask: "Go on to the drain rule offer listed above?", yes: nextLabel,
		next: "Asks next whether to record the rule; nothing is written without a yes there."},
	ConventionsFile: {ask: "Go on to the conventions files listed above?", yes: nextLabel,
		next: "Asks about each file listed above in turn; a file is removed only on your answer there."},
	UserState:   {ask: "Update abcd's records listed above?", yes: writesLabel, writes: true},
	PluginOwned: {ask: "Update abcd's own parts listed above?", yes: writesLabel, writes: true},
}

// unknownAskFormat asks the approval of a category categoryApprovalWords does
// not name.
const unknownAskFormat = "Make the %s changes listed above?"

var unknownAskRe = regexp.MustCompile(`^Make the ([^ ?]+) changes listed above\?$`)

// categoryWords is the approval of c.
func categoryWords(c GapCategory) approvalWords {
	if w, ok := categoryApprovalWords[c]; ok {
		return w
	}
	return approvalWords{ask: fmt.Sprintf(unknownAskFormat, c), yes: writesLabel, writes: true}
}

// approvalCategory is the category whose approval asks ask.
func approvalCategory(ask string) (GapCategory, bool) {
	for c, w := range categoryApprovalWords {
		if w.ask == ask {
			return c, true
		}
	}
	if m := unknownAskRe.FindStringSubmatch(ask); m != nil {
		return GapCategory(m[1]), true
	}
	return "", false
}

// categoryApprovalText is the approval of the kind of change c over its
// changes, lines: each line an item, then the question. Every change is
// listed, on every route, never cut to fit: the person sees every file and
// setting before approving, and every door records the same question
// (spc-2610030911534855 B3, B5). A list taller than the host's rows is shown
// with the guard's note rather than refused.
func categoryApprovalText(c GapCategory, lines []string) string {
	var b strings.Builder
	for _, ln := range lines {
		if ln = strings.Join(strings.Fields(ln), " "); ln != "" {
			b.WriteString("  " + ln + "\n")
		}
	}
	return b.String() + categoryWords(c).ask
}

// approvalCount is how many changes the material of an approval lists.
func approvalCount(material []question.Block) int {
	n := 0
	for _, b := range material {
		if b.Kind == question.KindList {
			n += len(b.Items)
		}
	}
	return n
}

// approvalYes is the yes of the approval of c whose material is material.
func approvalYes(c GapCategory, material []question.Block) question.Option {
	w := categoryWords(c)
	meaning := w.next
	if w.writes {
		switch n := approvalCount(material); n {
		case 1:
			meaning = "Writes the change listed above."
		default:
			meaning = fmt.Sprintf("Writes the %d changes listed above.", n)
		}
	}
	return question.Option{Value: "yes", Label: w.yes, Meaning: meaning}
}
