package question

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/intentdriven/abcd/internal/textwidth"
)

// Rule names one of the limits check's rules (spc-2610030944505997, "The
// limits check"). Each is reported as its own finding.
type Rule string

// The rules, in the spec's order, rule 14 appended by the layout intent's
// decision 20.
const (
	RuleHeader            Rule = "header"               // 1: present, within HeaderColumns, in the chip grammar
	RuleQuestionsPerCall  Rule = "questions-per-call"   // 2: one to four; more than one is tabs
	RuleOptions           Rule = "options"              // 3: two to four, decide-later included
	RuleLabelWords        Rule = "label-words"          // 4: at most LabelWords words
	RuleMeaningSentences  Rule = "meaning-sentences"    // 5: a description present, at most MeaningSentences
	RuleDecideLater       Rule = "decide-later"         // 6: exactly one LaterLabels option, last
	RuleNoBold            Rule = "no-bold"              // 7: no ** or __ outside previews
	RuleNeverRecommended  Rule = "never-recommended"    // 8: no (Recommended), no star mark
	RuleRegister          Rule = "register"             // 9: the product thinker sees no handle or command
	RuleNowAndChangeLater Rule = "now-and-change-later" // 10: a Now: and a Change later: line, each with a value
	RuleThingFirst        Rule = "thing-first"          // 11: material first, the question last
	RuleEmDashListItem    Rule = "em-dash-in-list-item" // 12: no em dash in a list item
	RuleRows              Rule = "rows"                 // 13: within Rows at Columns
	RuleNoPreview         Rule = "no-preview"           // 14: no option carries a side preview
)

// Finding is one broken rule: the question's ordinal (its tab, counted from
// one; zero is the call as a whole), the part, the rule, the offending value
// (sanitised and capped, as the mode store's echo is), the limit, and what to
// do about it.
type Finding struct {
	Tab    int
	Part   string
	Rule   Rule
	Value  string
	Limit  string
	Remedy string
}

// String is the finding as one line of plain text.
func (f Finding) String() string {
	where := "the call"
	if f.Tab > 0 {
		where = fmt.Sprintf("tab %d", f.Tab)
	}
	return fmt.Sprintf("%s, %s (%s): %q; limit: %s. %s", where, f.Part, f.Rule, f.Value, f.Limit, f.Remedy)
}

// echoCap bounds how much of an offending value a finding quotes back, the
// mode store's own echo bound.
const echoCap = 64

// echo is a value as a finding may quote it: one line, sanitised, capped.
func echo(s string) string { return termsafe.CleanProseLine(s, echoCap) }

// EmDashListItemPattern is rule 12's pattern: a line opening a list item that
// carries an em dash. It is the docs-lint token
// punctuation/em-dash-in-list-item, whose definition lives in ahoy's docs-lint
// seed; a test in internal/core/ahoy holds the two equal.
const EmDashListItemPattern = "^\\s*(?:[-*+]|[0-9]+\\.)\\s.*\u2014"

var emDashListItemRe = regexp.MustCompile(EmDashListItemPattern)

// RecommendedRemedy is rule 8's remedy. It names the cause, so the agent does
// not loop (itd-201 criterion R6).
const RecommendedRemedy = "The host's own instruction for this tool asks for a recommended first option; abcd's asking rule reverses it: no option is marked, styled, or ordered as recommended. Remove the mark and ask again; if the person asks for a recommendation, give it in prose beside the question."

// previewRemedy is rule 14's remedy (itd-2610030810350727 decision 20).
const previewRemedy = "abcd's questions carry no side preview (decision 20): put the option's meaning, gain and cost in its description."

// splitRemedy is rule 13's remedy (itd-2610030810350727 decision 9).
const splitRemedy = "Split the material into parts: up to four parts of one thing as tabs, or successive questions, one part per question. Never move it into a message before the question, or into a preview."

// ChipRole reads the role word from a header in the chip grammar,
// "<role> Q<n>" with an optional "/<total>", the role one of l.ChipRoles
// ("Product Q2", "Tech Q3", "Setup Q1/4"). It judges the grammar alone, not the
// width. A header in the grammar is abcd's: only abcd's interview pages are
// taught to write it.
func ChipRole(header string, l Limits) (string, bool) {
	m := chipRe(l).FindStringSubmatch(header)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// chipRes holds the chip pattern compiled once per set of role words, keyed by
// the words joined with a NUL, which no role word carries. The guard reads the
// chip once per tab, and a compile per read made a call of many tabs spend
// seconds compiling (review-askGuard-security finding 6).
var chipRes sync.Map // string -> *regexp.Regexp

func chipRe(l Limits) *regexp.Regexp {
	key := strings.Join(l.ChipRoles, "\x00")
	if re, ok := chipRes.Load(key); ok {
		return re.(*regexp.Regexp)
	}
	roles := make([]string, len(l.ChipRoles))
	for i, r := range l.ChipRoles {
		roles[i] = regexp.QuoteMeta(r)
	}
	re := regexp.MustCompile(`^(` + strings.Join(roles, "|") + `) Q[1-9][0-9]*(?:/[1-9][0-9]*)?$`)
	actual, _ := chipRes.LoadOrStore(key, re)
	return actual.(*regexp.Regexp)
}

// CheckLimits holds a call's fields to l and returns every finding at once, so
// one refusal names every part to fix and the agent fixes them in one retry.
// It refuses and never rewrites: a rewritten question puts words in the
// agent's mouth that neither the agent nor the person chose. No finding means
// the question is admitted.
func CheckLimits(f Fields, l Limits, who Addressee) []Finding {
	var out []Finding
	if n := len(f.Tabs); n < l.QuestionsPerCall[0] || n > l.QuestionsPerCall[1] {
		out = append(out, Finding{
			Part:   "questions",
			Rule:   RuleQuestionsPerCall,
			Value:  fmt.Sprintf("%d questions", n),
			Limit:  fmt.Sprintf("%d to %d questions in one call", l.QuestionsPerCall[0], l.QuestionsPerCall[1]),
			Remedy: "Ask one question, or up to four parts of one thing as tabs; ask a question that depends on an earlier answer alone, after that answer.",
		})
	}
	// Past the count, only the tabs the limits allow are checked field by
	// field: the count finding already refuses the call, and a finding per
	// field of every extra tab would grow with the payload, not the fault.
	c := checker{l: l, who: who, chip: chipRe(l), verb: verbRe(who.Verbs)}
	for i, t := range f.Tabs[:min(len(f.Tabs), max(l.QuestionsPerCall[1], 0))] {
		out = append(out, c.tab(i+1, t)...)
	}
	return out
}

// checker carries what every tab of one call is checked against.
type checker struct {
	l    Limits
	who  Addressee
	chip *regexp.Regexp
	verb *regexp.Regexp // nil when the caller passed no verbs
}

// field is one checked field of a tab, named as a finding names it.
type field struct {
	part, text string
	preview    bool
}

func optionPart(i int, what string) string { return fmt.Sprintf("option %d %s", i+1, what) }

// fields lists a tab's fields in reading order, previews included.
func fields(t Tab) []field {
	out := []field{{part: "header", text: t.Header}, {part: "question text", text: t.Text}}
	for i, o := range t.Options {
		out = append(out,
			field{part: optionPart(i, "label"), text: o.Label},
			field{part: optionPart(i, "description"), text: o.Description})
		if o.Preview != "" {
			out = append(out, field{part: optionPart(i, "preview"), text: o.Preview, preview: true})
		}
	}
	return out
}

func (c checker) tab(n int, t Tab) []Finding {
	var out []Finding
	add := func(part string, r Rule, value, limit, remedy string) {
		out = append(out, Finding{Tab: n, Part: part, Rule: r, Value: echo(value), Limit: limit, Remedy: remedy})
	}
	l := c.l

	// 1. Header.
	chipLimit := fmt.Sprintf("the chip <role> Q<n>, an optional /<total>, the role one of %s", strings.Join(l.ChipRoles, ", "))
	if strings.TrimSpace(t.Header) == "" {
		add("header", RuleHeader, t.Header, chipLimit, "Head the question with its chip, such as "+chipExample(l)+".")
	} else {
		if w := textwidth.Columns(t.Header); w > l.HeaderColumns {
			add("header", RuleHeader, t.Header, fmt.Sprintf("%d columns", l.HeaderColumns),
				fmt.Sprintf("The header is %d columns; shorten it to the chip, such as %s.", w, chipExample(l)))
		}
		if !c.chip.MatchString(t.Header) {
			add("header", RuleHeader, t.Header, chipLimit, "Write the header as the chip, such as "+chipExample(l)+"; a total is added only when the interview's length is known.")
		}
	}

	// 3. Options. A typed part is the host's free-text row: it counts as one
	// option toward the floor (spc-2610031241482088, open question 1), never
	// toward the ceiling, which is what the host lists.
	floor := len(t.Options)
	if strings.TrimSpace(t.Typed) != "" {
		floor++
	}
	if k := len(t.Options); floor < l.OptionsPerQ[0] || k > l.OptionsPerQ[1] {
		add("options", RuleOptions, fmt.Sprintf("%d options", k),
			fmt.Sprintf("%d to %d options, the decide-later option included", l.OptionsPerQ[0], l.OptionsPerQ[1]),
			"Offer only the answers each defensible on the record, then the decide-later option.")
	}
	// Past the count, only the options the limits allow are checked field by
	// field, as CheckLimits does with tabs. Rule 6 still reads every label,
	// since "the last" is the payload's last, but names only kept options.
	all := t.Options
	t.Options = t.Options[:min(len(t.Options), max(l.OptionsPerQ[1], 0))]

	// 4. Label words, and 5. meaning sentences.
	for i, o := range t.Options {
		switch w := len(strings.Fields(o.Label)); {
		case w == 0:
			add(optionPart(i, "label"), RuleLabelWords, o.Label, fmt.Sprintf("1 to %d words", l.LabelWords), "Give the option a label of a few words.")
		case w > l.LabelWords:
			add(optionPart(i, "label"), RuleLabelWords, o.Label, fmt.Sprintf("%d words", l.LabelWords),
				fmt.Sprintf("The label is %d words; shorten it and put the rest in the description.", w))
		}
		switch s := sentences(o.Description); {
		case strings.TrimSpace(o.Description) == "":
			add(optionPart(i, "description"), RuleMeaningSentences, o.Description, fmt.Sprintf("1 to %d sentences", l.MeaningSentences),
				"Say what choosing this option means in its description, with its gain and cost.")
		case s > l.MeaningSentences:
			add(optionPart(i, "description"), RuleMeaningSentences, o.Description, fmt.Sprintf("%d sentences", l.MeaningSentences),
				fmt.Sprintf("The description is %d sentences; keep what choosing it means and its gain and cost.", s))
		}
	}

	// 6. Decide later.
	var later []int
	for i, o := range all {
		if c.isLater(o.Label) {
			later = append(later, i)
		}
	}
	laterLimit := fmt.Sprintf("exactly one option labelled %s, the last", quoteList(l.LaterLabels))
	if len(later) == 0 {
		labels := make([]string, len(t.Options))
		for i, o := range t.Options {
			labels[i] = o.Label
		}
		add("options", RuleDecideLater, strings.Join(labels, " | "), laterLimit,
			"Add "+quoteList(l.LaterLabels)+" as the last option; it is always offered.")
	}
	for _, i := range later {
		if i == len(all)-1 || i >= len(t.Options) {
			continue
		}
		remedy := "Move the decide-later option to the end."
		if len(later) > 1 {
			remedy = "Offer one decide-later option, the last; remove the others."
		}
		add(optionPart(i, "label"), RuleDecideLater, t.Options[i].Label, laterLimit, remedy)
	}

	// 7. No bold markers, and 8. never recommended.
	for _, fl := range fields(t) {
		if !fl.preview && (strings.Contains(fl.text, "**") || strings.Contains(fl.text, "__")) {
			add(fl.part, RuleNoBold, fl.text, "no ** or __ outside a preview",
				"Remove the markers: the host shows them literally.")
		}
	}
	for i, o := range t.Options {
		if recommended(o.Label) {
			add(optionPart(i, "label"), RuleNeverRecommended, o.Label, "no (Recommended) label and no star mark", RecommendedRemedy)
		}
	}

	// 9. The product thinker's register.
	if c.forProductThinker(t.Header) {
		for _, fl := range fields(t) {
			if hit, ok := c.command(fl.text); ok {
				add(fl.part, RuleRegister, hit, "no record number and no command for the product thinker",
					"Say it in product terms: name the outcome or the choice, never a record number or a command.")
			}
		}
	}

	// 10. Now and change later, 11. the thing first, 12. no em dash in a
	// list item.
	lines := strings.Split(t.Text, "\n")
	for _, p := range []string{l.NowPrefix, l.ChangeLaterPrefix} {
		value, found := prefixedValue(lines, p)
		if found && value != "" {
			continue
		}
		v := "no " + p + " line"
		if found {
			v = p
		}
		add("question text", RuleNowAndChangeLater, v,
			fmt.Sprintf("a line starting %q with a value", p),
			fmt.Sprintf("Add a %q line saying %s, writing %q where it does not apply.", p, laterOrNow(p, l), l.NotApplicable))
	}
	last := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			last = i
			break
		}
	}
	lastLine := ""
	if last >= 0 {
		lastLine = strings.TrimSpace(lines[last])
	}
	if !strings.HasSuffix(lastLine, "?") {
		add("question text", RuleThingFirst, lastLine, "the text ends with the question",
			"End the text with the one plain question, on its own line.")
	}
	material := false
	for _, ln := range lines[:max(last, 0)] {
		if tl := strings.TrimSpace(ln); tl != "" && !hasPrefixFold(tl, l.NowPrefix) && !hasPrefixFold(tl, l.ChangeLaterPrefix) {
			material = true
			break
		}
	}
	if !material {
		add("question text", RuleThingFirst, lastLine, "at least one paragraph of material before the question",
			"Quote the thing being decided in full first, in paragraphs and lists, then ask; never refer to it.")
	}
	for _, ln := range lines {
		if emDashListItemRe.MatchString(ln) {
			add("question text", RuleEmDashListItem, ln, "no em dash in a list item",
				"Use a colon at the pivot, with a capital letter after it.")
		}
	}

	// 13. Rows.
	if rows := estimateRows(t, l); rows > l.Rows {
		add("question text", RuleRows, fmt.Sprintf("%d rows", rows),
			fmt.Sprintf("%d rows at %d columns", l.Rows, l.Columns), splitRemedy)
	}

	// 14. No side preview (the layout intent's decision 20): with a preview
	// the host hides every option's description and cuts the preview to the
	// rows it has, so the meaning is put where it always shows.
	for i, o := range t.Options {
		if o.Preview != "" {
			add(optionPart(i, "preview"), RuleNoPreview, o.Preview, "no side preview", previewRemedy)
		}
	}
	return out
}

func chipExample(l Limits) string {
	if len(l.ChipRoles) == 0 {
		return `"Q1"`
	}
	return fmt.Sprintf("%q", l.ChipRoles[0]+" Q1")
}

func quoteList(xs []string) string {
	q := make([]string, len(xs))
	for i, x := range xs {
		q[i] = fmt.Sprintf("%q", x)
	}
	return strings.Join(q, " or ")
}

func laterOrNow(prefix string, l Limits) string {
	if prefix == l.NowPrefix {
		return "what holds now"
	}
	return "how to change the answer later"
}

func (c checker) isLater(label string) bool {
	label = strings.TrimSpace(label)
	for _, ll := range c.l.LaterLabels {
		if strings.EqualFold(label, ll) {
			return true
		}
	}
	return false
}

// forProductThinker reports whether rule 9 applies: the mode names the product
// thinker, or no mode names anyone and the chip's role word is the product
// thinker's.
func (c checker) forProductThinker(header string) bool {
	switch c.who.Person {
	case ProductThinker:
		return true
	case Facilitator:
		return false
	}
	m := c.chip.FindStringSubmatch(header)
	return m != nil && m[1] == ProductRole
}

var (
	goRunRe = regexp.MustCompile(`\bgo\s+run\b`)
	slashRe = regexp.MustCompile(`/abcd:[A-Za-z0-9_-]*`)
)

func verbRe(verbs []string) *regexp.Regexp {
	if len(verbs) == 0 {
		return nil
	}
	q := make([]string, len(verbs))
	for i, v := range verbs {
		q[i] = regexp.QuoteMeta(v)
	}
	return regexp.MustCompile(`(?i)\babcd\s+(?:` + strings.Join(q, "|") + `)\b`)
}

// command returns the first record handle or command s carries: a handle as
// recordid.HandleInText finds one, a backtick span, `go run`, a /abcd: slash
// command, or abcd followed by one of the binary's verbs.
func (c checker) command(s string) (string, bool) {
	if h, ok := recordid.HandleInText(s); ok {
		return h, true
	}
	for i := 0; i < len(s); i++ {
		if s[i] != '`' {
			continue
		}
		if sp, ok := termsafe.PairCodeSpan(s, i); ok {
			return s[sp.Start:sp.End], true
		}
		for i+1 < len(s) && s[i+1] == '`' {
			i++
		}
	}
	for _, re := range []*regexp.Regexp{goRunRe, slashRe, c.verb} {
		if re == nil {
			continue
		}
		if m := re.FindString(s); m != "" {
			return m, true
		}
	}
	return "", false
}

// recommended reports a label marked as recommended: "(Recommended)" in any
// case, or a star opening or closing it.
func recommended(label string) bool {
	if strings.Contains(strings.ToLower(label), "(recommended)") {
		return true
	}
	t := strings.Trim(label, " \t️")
	if t == "" {
		return false
	}
	first, _ := utf8.DecodeRuneInString(t)
	lastR, _ := utf8.DecodeLastRuneInString(t)
	return isStar(first) || isStar(lastR)
}

func isStar(r rune) bool {
	switch r {
	case '*', '★', '☆', '⭐':
		return true
	}
	return false
}

// abbreviations end no sentence (rule 5).
var abbreviations = strings.NewReplacer("e.g.", "eg", "E.g.", "Eg", "i.e.", "ie", "I.e.", "Ie", "etc.", "etc")

var sentenceEndRe = regexp.MustCompile(`[.?!]+(?:\s+|$)`)

// sentences counts the sentences in s: each run ended by '.', '?' or '!'
// before white space or the end, with "e.g.", "i.e." and "etc." masked, and a
// closing fragment without a stop counted as one.
func sentences(s string) int {
	n := 0
	for _, part := range sentenceEndRe.Split(abbreviations.Replace(s), -1) {
		if strings.TrimSpace(part) != "" {
			n++
		}
	}
	return n
}

func hasPrefixFold(s, prefix string) bool {
	return len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix)
}

// prefixedValue finds the first line starting with prefix (case-insensitive)
// and returns the value after it.
func prefixedValue(lines []string, prefix string) (string, bool) {
	for _, ln := range lines {
		if tl := strings.TrimSpace(ln); hasPrefixFold(tl, prefix) {
			return strings.TrimSpace(tl[len(prefix):]), true
		}
	}
	return "", false
}
