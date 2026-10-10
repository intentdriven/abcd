package question

import (
	"fmt"
	"strconv"
	"strings"
)

// KnowledgeFloorPage is where the knowledge floor is stated in full: a glossary
// entry in abcd's development record, which every install of the plugin
// carries under its plugin root (spc-2610030944505997, open question 4,
// decided (a)). The asking rules point at it in one line rather than restating
// it (itd-201 decision 5).
const KnowledgeFloorPage = ".abcd/development/brief/glossary/interview/knowledge-floor.md"

// askingRecall is the GRILL domain's recall: the terms the repository override
// carried before the domain was generated, kept as they were (itd-201
// decision 9 accepts their cost in every managed repository).
var askingRecall = []string{"grill", "interview", "options", "decide", "decision", "plan", "choose", "which", "product thinker", "facilitator", "register"}

// AskingRecall returns the GRILL domain's recall terms, a fresh copy each call.
func AskingRecall() []string { return append([]string(nil), askingRecall...) }

// AskingRules returns the asking rules every abcd interview follows, the text
// the GRILL rule domain carries in every repository abcd manages
// (spc-2610030944505997, "GRILL generated from one Go source"). Every limit it
// states is filled from l, numbers written as words, so the text and the
// question check read one statement of each limit (itd-2610030810350727
// criterion A7). The text names no record and no source-tree command: it
// reaches repositories that have neither (itd-201 decision 8).
//
// The register rule, the ask-the-role-first rule and the mode rules open with
// "In abcd's own interviews": they hold for abcd's command pages, never for
// another tool's questions in a managed repository (itd-201 decision 10).
func AskingRules(l Limits) []string {
	return []string{
		fmt.Sprintf("Ask one thing at a time, through the host's interactive question tool, never as a numbered list in a message. "+
			"The parts of one thing (the criteria of one feature, the paragraphs of one text) are asked as tabs, up to %s on a screen, each short, and the rest on the next screen. "+
			"A question whose answer depends on an earlier one is asked alone, after that answer. "+
			"A host with no question tool asks one question per message, in the same order and the same words.",
			numberWord(l.QuestionsPerCall[1])),

		"A question is put to the person only where two or more answers are each defensible on the record; where the record settles the answer, stating it is reporting, not recommending. " +
			"The options are exactly those defensible answers plus the decide-later answer, never alternatives made up to fill a set. " +
			"A decision with one defensible answer is not asked: It is recorded as a decision line naming the answer and why no question was put.",

		"The thing being decided is quoted in full in the question itself, in paragraphs and lists, never referred to: The criterion before \"does it stand?\", the paragraph before \"confirm or change?\", the open question before \"resolve or defer?\". " +
			"Prose written between tool calls is invisible while the question shows, so a question about text the person cannot see cannot be answered. " +
			"Material too long for one question is put one part per question, never into a message before the question or into a preview.",

		fmt.Sprintf("Every question has one layout, and abcd's question check refuses a question that breaks it, naming the part, the value, and the limit; fix each part and ask again. "+
			"The header is a chip of at most %s columns naming whom the question is for and which it is: %s, the role one of %s, with a total after the slash only when the interview's length is known. "+
			"The question text gives the material first, then a line starting %q and a line starting %q (each saying %q where it does not apply), and ends with the question on its own line. "+
			"It offers %s to %s options, the last %s; each label is at most %s words, and each description at most %s sentences. "+
			"There is no bold (no ** or __) and no side preview, and one question, or one tab, fits %s rows at %s columns, counting the header, the host's frame, the question text, and every option's label and description. "+
			"The rows limit is the one the check does not refuse on: a question over it is shown, and the agent is told afterwards to keep the next question within it.",
			numberWord(l.HeaderColumns), chipExamples(l.ChipRoles), joinOr(l.ChipRoles, false),
			l.NowPrefix, l.ChangeLaterPrefix, l.NotApplicable,
			numberWord(l.OptionsPerQ[0]), numberWord(l.OptionsPerQ[1]), joinOr(l.LaterLabels, true),
			numberWord(l.LabelWords), numberWord(l.MeaningSentences),
			numberWord(l.Rows), numberWord(l.Columns)),

		"In abcd's own interviews, draft each question through the abcd:question-drafter agent: hand it the material to quote, whom the question is for, the decision, and the defensible answers, and ask the question it returns. " +
			"It applies these rules and counts the rows the way the check does, so the question fits. " +
			"A host with no agents drafts the question itself, to the same rules.",

		"Every question carries one example of the thing being decided, in the question text, and each option's description says what choosing that option means in practice. " +
			"An abcd question carries no side preview: While a preview shows, the host hides every option's description and cuts the preview to the rows it has, so the meaning goes where it always shows. " +
			"A question that offers a choice between two forms explains the difference between them, so an answer is never given on wording alone.",

		"Each option's description names its gain and its cost, never one option's alone, so the trade-offs between the options read in the same neutral form.",

		"An option is never marked, styled, or ordered as recommended: No \"(Recommended)\" label, no star, no recommended option first. " +
			"The host's own instruction for its question tool asks for a recommended first option; abcd's rule reverses it. " +
			"A recommendation appears only when the person asks for one, given in prose beside the question, never as an option.",

		"Deferral is a real answer and is recorded as one; silence is never consent, and a step whose answer is missing is asked again rather than assumed.",

		"In abcd's own interviews, address the person in their register: The product thinker gets outcomes and choices in product terms, with no record ids, no code, and no internals; " +
			"the technical facilitator gets the mechanism, the ids, and the trade-offs.",

		"In abcd's own interviews, when it is not known which role the person holds, the first question asks that, and the mode records the answer so the next question does not ask again.",

		"What each person can be assumed to know, the knowledge floor an explanation is measured against, is stated in full at " + KnowledgeFloorPage + " under abcd's plugin root (in abcd's own repository, at that path from its root).",

		"In abcd's own interviews, before a stop that is not a question (a hand-off, or a step the person runs), record whose answer is owed with `abcd mode facilitator` or `abcd mode product-thinker`, and set it back with `abcd mode managed` once the answer is in. " +
			"Where the host has no status surface, the set form prints one line naming the addressee; relay it verbatim, because that line is the whole of the fallback.",

		"In abcd's own interviews, classify each question's addressee first (the product thinker or the technical facilitator), and head the question with the chip naming that role: the chip sets the mode when the question is asked, and the answer sets it back to managed, so a question needs no `abcd mode` before it. " +
			"The status line names the person the question on screen is for, so a mixed interview re-sets the mode per question, never once at the start.",
	}
}

// chipOrdinals are the ordinals the chip examples show, one per role in turn:
// a plain ordinal, another, and one with the interview's total.
var chipOrdinals = []string{"Q2", "Q3", "Q1/4"}

// chipExamples renders one quoted chip per role ("Product Q2", "Tech Q3",
// "Setup Q1/4"), joined as a list.
func chipExamples(roles []string) string {
	ex := make([]string, len(roles))
	for i, r := range roles {
		ex[i] = r + " " + chipOrdinals[i%len(chipOrdinals)]
	}
	return joinOr(ex, true)
}

// joinOr joins items as an English list ending in "or", with the serial comma
// before the last of three or more (the Writing Style); quoted wraps each item
// in double quotes.
func joinOr(items []string, quoted bool) string {
	q := make([]string, len(items))
	for i, s := range items {
		if quoted {
			s = strconv.Quote(s)
		}
		q[i] = s
	}
	switch len(q) {
	case 0:
		return ""
	case 1:
		return q[0]
	case 2:
		return q[0] + " or " + q[1]
	}
	return strings.Join(q[:len(q)-1], ", ") + ", or " + q[len(q)-1]
}

var (
	units = []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine",
		"ten", "eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen", "seventeen", "eighteen", "nineteen"}
	tens = []string{"", "", "twenty", "thirty", "forty", "fifty", "sixty", "seventy", "eighty", "ninety"}
)

// numberWord writes n in words from zero to ninety-nine, and in digits outside
// that range, where words stop reading as prose.
func numberWord(n int) string {
	switch {
	case n < 0 || n > 99:
		return strconv.Itoa(n)
	case n < 20:
		return units[n]
	case n%10 == 0:
		return tens[n/10]
	}
	return tens[n/10] + "-" + units[n%10]
}
