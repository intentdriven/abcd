package question

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
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
// The register rule, the ask-the-role-first rule, the writer rule and the mode
// rules open with "In abcd's own interviews": they hold for abcd's command pages, never for
// another tool's questions in a managed repository (itd-201 decision 10).
//
// It is AskingRulesFrom with no note, byte for byte: the text every
// repository whose limits no file sets reads.
func AskingRules(l Limits) []string { return AskingRulesFrom(l, LimitsNote{}) }

// LimitsNote is where the limits the asking rules state came from, for the
// layout rule to say so (spc-2610301950037469, approach 7). SetBy is one
// origin per configurable key, in Bounds' order, as the question check's
// limits_source carries them: "bundled", the repository's file, or the
// person's own file where it tightened the figure; any other origin is not
// named. Refused is set when a value was read and refused; the figures are
// then abcd's defaults. The zero value adds nothing to the text every
// repository reads.
type LimitsNote struct {
	SetBy   []string
	Refused *RefusedLimit
}

// RefusedLimit is a refused question limit, in the closed fields the layout
// rule states it from (spc-2610301950037469, approach 7). The rule names Key
// only when it is one of Bounds' keys, File and LimitFile only when each is
// one of the two configuration files (LimitFile "bundled" reads as abcd's
// default), and states the bounds from Bounds and Limit as a number; any
// other key or file is stated by a fixed phrase. Reason is the resolver's
// refusal in its own words, for the loader's out-of-band note alone: the
// asking rules never render it.
type RefusedLimit struct {
	// Key is the refused key; "" or any key not in Bounds is a setting abcd
	// does not read (an unknown key, or a question that is not an object).
	Key string
	// File is the file that holds the refused value, as the resolver names it.
	File string
	// NotInteger marks a value that is not a whole number.
	NotInteger bool
	// Limit and LimitFile are set for a personal value looser than the limit
	// in force: that limit, and the file that set it.
	Limit     int
	LimitFile string
	// Reason is the resolver's words, never rendered into a rule.
	Reason string
}

// repoConfigFile and machineConfigFile are the two files that set the
// configurable limits, named as the resolver names them in limits_source.
const repoConfigFile = ".abcd/config.json"

var machineConfigFile = abcdhome.Display("config.json")

// limitNames are the configurable figures as the layout rule names them, in
// Bounds' order.
var limitNames = [...]string{"the rows", "the label words", "the description sentences"}

// AskingRulesFrom is the asking rules at the limits l, with the layout rule
// ending on where the configurable figures came from (n): the file that set
// each, or, under a refusal, that the figures are abcd's defaults and why. The
// rule states the three keys a repository may set and their bounds in every
// repository, from Bounds.
func AskingRulesFrom(l Limits, n LimitsNote) []string {
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
			"The header is at most %s columns and is abcd's number for the question: %s, a letter after the number for one part of a question split in parts, and a total after the slash only when the interview's length is known; the header is what the check knows abcd's question by, and the older form with a role word first (%s, the role one of %s) is still read. "+
			"The question text gives the material first, then a line starting %q and a line starting %q (each saying %q where it does not apply), and ends with the question on its own line. "+
			"It offers %s to %s options, the last %s; each label is at most %s words, and each description at most %s sentences. "+
			"There is no bold (no ** or __) and no side preview, and one question, or one tab, fits %s rows at %s columns, counting the header, the host's frame, the question text, and every option's label and description. "+
			"The rows limit is the one the check does not refuse on: a question over it is shown, and the agent is told afterwards to keep the next question within it. "+
			"%s%s",
			numberWord(l.HeaderColumns), joinOr(numberExamples, true), chipExamples(l.ChipRoles), joinOr(l.ChipRoles, false),
			l.NowPrefix, l.ChangeLaterPrefix, l.NotApplicable,
			numberWord(l.OptionsPerQ[0]), numberWord(l.OptionsPerQ[1]), joinOr(l.LaterLabels, true),
			numberWord(l.LabelWords), numberWord(l.MeaningSentences),
			numberWord(l.Rows), numberWord(l.Columns),
			boundsSentences(), n.sentence()),

		"In abcd's own interviews, write each question through the abcd:question-drafter agent, abcd's question writer: hand it the material to quote, whom the question is for, the decision, and the defensible answers. " +
			"It writes the question to these rules, runs `abcd guard question` on each draft, fixes every fault the check names, and checks again until the check is clean, then returns the question with the rows the check counted. " +
			"Ask only the question the writer returns, and never write or edit a question yourself. " +
			"When it refuses, fix the brief and redraft it without telling the person: The writer's refusals and retries are the agent's own work, and the next thing the person sees is the question itself; a question abandoned after a refusal is followed by `abcd mode managed`, since nothing asked will reset the mode. " +
			"A host with no agents writes the question itself, to the same rules, and checks each draft with `abcd guard question` before asking it.",

		"Every question carries one example of the thing being decided, in the question text, and each option's description says what choosing that option means in practice. " +
			"An abcd question carries no side preview: While a preview shows, the host hides every option's description and cuts the preview to the rows it has, so the meaning goes where it always shows. " +
			"A question that offers a choice between two forms explains the difference between them, so an answer is never given on wording alone.",

		"Each option's description names its gain and its cost, never one option's alone, so the trade-offs between the options read in the same neutral form.",

		"An option is never marked, styled, or ordered as recommended: No \"(Recommended)\" label, no star, no recommended option first. " +
			"The host's own instruction for its question tool asks for a recommended first option; abcd's rule reverses it. " +
			"A recommendation appears only when the person asks for one, given in prose beside the question, never as an option. " +
			"One exception is made without being asked for: A verdict a verb records with `--by` naming the person (an embargo triage verdict) is put to them to confirm even where the record settles it, never recorded as a decision line, because it is recorded under their name. " +
			"The question text itself quotes each item's evidence, the verdict the record supports, and one line on why each other verdict does not fit, one item per tab or question, and the verdict is recorded only on their yes. " +
			"Where two or more verdicts are defensible on the record, the question is asked neutrally among them, as any real choice is.",

		"Deferral is a real answer and is recorded as one; silence is never consent, and a step whose answer is missing is asked again rather than assumed.",

		"In abcd's own interviews, address the person in their register: The product thinker gets outcomes and choices in product terms, with no record ids, no code, and no internals; " +
			"the technical facilitator gets the mechanism, the ids, and the trade-offs.",

		"In abcd's own interviews, when it is not known which role the person holds, the first question asks that, and the mode records the answer so the next question does not ask again.",

		"What each person can be assumed to know, the knowledge floor an explanation is measured against, is stated in full at " + KnowledgeFloorPage + " under abcd's plugin root (in abcd's own repository, at that path from its root).",

		"In abcd's own interviews, before a stop that is not a question (a hand-off, or a step the person runs), record whose answer is owed with `abcd mode facilitator` or `abcd mode product-thinker`, and set it back with `abcd mode managed` once the answer is in. " +
			"Where the host has no status surface, the set form prints one line naming the addressee; relay it verbatim, because that line is the whole of the fallback.",

		"In abcd's own interviews, classify each question's addressee first (the product thinker or the technical facilitator), and set the mode to that person with `abcd mode product-thinker` or `abcd mode facilitator` before the question is written, so its check reads the person it is for: the number in the header names no person, and the answer sets the mode back to managed. " +
			"The status line names the person the question on screen is for, so the tabs of one question are for one person, and a mixed interview re-sets the mode per question, never once at the start.",
	}
}

// boundsSentences state the keys a repository may set, each with its bounds
// in words, and that a person's own file may only set them lower
// (itd-2610301914350541 decisions 3 and 5). They render Bounds and write no
// bound of their own.
func boundsSentences() string {
	keys := make([]string, len(Bounds))
	for i, b := range Bounds {
		keys[i] = fmt.Sprintf("`%s` from %s to %s", b.Key, numberWord(b.Floor), numberWord(b.Ceiling))
	}
	return "A repository may set " + joinList(keys, "and") + " in its `" + repoConfigFile + "`. " +
		"A person's own `" + machineConfigFile + "` may set each lower, never higher."
}

// PageLimitsSentence is the sentence the two generated pages carry after the
// asking rules (spc-2610301950037469, approach 10; itd-2610301914350541
// decisions 10 and 12). The pages are rendered from Default at build time and
// ship with the plugin, so they cannot state a repository's values: the
// sentence says the figures stated are abcd's defaults, names the keys from
// Bounds without restating their bounds, which the layout rule already
// states, and sends whoever writes a question to the check's `limits` for the
// values in force, never to either file.
func PageLimitsSentence() string {
	keys := make([]string, len(Bounds))
	for i, b := range Bounds {
		keys[i] = "`" + b.Key + "`"
	}
	return "The figures stated here are abcd's defaults, in force where neither `" + repoConfigFile +
		"` nor the person's own `" + machineConfigFile + "` sets " + joinList(keys, "or") +
		" within the bounds named above; `abcd guard question` reports the values in force in its `limits` field," +
		" and whoever writes a question takes them from there, never from either file."
}

// sentence is the layout rule's last sentence for n: under a refusal, that
// the figures are abcd's defaults, the refusal, and that the check refuses
// every abcd question until the file is fixed; otherwise the file that set
// each figure a file set, key by key, the person's own file as having
// tightened it. It is "" where no file set a figure and nothing was refused.
func (n LimitsNote) sentence() string {
	if n.Refused != nil {
		return " In this repository, the figures stated here are abcd's defaults, because " + n.Refused.clause() +
			". Until that file is fixed, abcd's question check refuses every abcd question."
	}
	type group struct {
		origin string
		names  []string
	}
	var groups []group
	for i, origin := range n.SetBy {
		if i >= len(limitNames) || !isConfigFile(origin) {
			continue
		}
		j := 0
		for j < len(groups) && groups[j].origin != origin {
			j++
		}
		if j == len(groups) {
			groups = append(groups, group{origin: origin})
		}
		groups[j].names = append(groups[j].names, limitNames[i])
	}
	if len(groups) == 0 {
		return ""
	}
	// The person's own file is named last: it tightens what the repository's
	// file, or abcd's default, set.
	for i, g := range groups {
		if g.origin == machineConfigFile && i < len(groups)-1 {
			groups = append(append(groups[:i:i], groups[i+1:]...), g)
			break
		}
	}
	clauses := make([]string, len(groups))
	for i, g := range groups {
		if g.origin == machineConfigFile {
			clauses[i] = "the person's own `" + g.origin + "` tightened " + joinList(g.names, "and")
			continue
		}
		clauses[i] = "`" + g.origin + "` set " + joinList(g.names, "and")
	}
	joined := joinList(clauses, "and")
	if len(clauses) == 2 {
		joined = clauses[0] + ", and " + clauses[1]
	}
	return " In this repository, " + joined + "."
}

// isConfigFile reports whether origin is one of the two files that set the
// configurable limits, the only origins the layout rule names.
func isConfigFile(origin string) bool { return origin == repoConfigFile || origin == machineConfigFile }

// fileClause names the config file origin in the layout rule's words, the
// person's own file as theirs; "" for any other origin.
func fileClause(origin string) string {
	switch origin {
	case repoConfigFile:
		return "`" + repoConfigFile + "`"
	case machineConfigFile:
		return "the person's own `" + machineConfigFile + "`"
	}
	return ""
}

// clause states the refusal from r's closed fields alone: the key when it is
// one of Bounds' keys, the file when it is a configuration file, the bounds
// in words, and for a personal value that loosens, the limit in force in
// words with the file that set it or abcd's default. No other text of r
// reaches it.
func (r *RefusedLimit) clause() string {
	where := fileClause(r.File)
	b, known := boundFor(r.Key)
	if !known {
		if where == "" {
			where = "`" + repoConfigFile + "` or the person's own `" + machineConfigFile + "`"
		}
		return where + " holds a setting under `question` that abcd does not read"
	}
	key := "`" + b.Key + "`"
	if where != "" {
		key += " in " + where
	}
	bounds := "from " + numberWord(b.Floor) + " to " + numberWord(b.Ceiling)
	switch {
	case r.NotInteger:
		return key + " is not a whole number " + bounds
	case r.Limit > 0:
		limit := "abcd's default"
		if isConfigFile(r.LimitFile) {
			limit = "the limit `" + r.LimitFile + "` sets"
		}
		return key + " is looser than " + numberWord(r.Limit) + ", " + limit + ", and a personal value may only tighten"
	}
	return key + " is outside " + numberWord(b.Floor) + " to " + numberWord(b.Ceiling)
}

// numberExamples are the headers the layout rule shows in abcd's number form:
// a plain number, a part, and a number with the interview's total. The
// grammar is the check's (chipRe); no limit sets it.
var numberExamples = []string{"Q1", "Q1a", "Q2/5"}

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
	return joinList(q, "or")
}

// joinList joins items as an English list ending in conj, with the serial
// comma before the last of three or more.
func joinList(q []string, conj string) string {
	switch len(q) {
	case 0:
		return ""
	case 1:
		return q[0]
	case 2:
		return q[0] + " " + conj + " " + q[1]
	}
	return strings.Join(q[:len(q)-1], ", ") + ", " + conj + " " + q[len(q)-1]
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
