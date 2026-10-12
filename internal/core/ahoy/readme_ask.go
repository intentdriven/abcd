package ahoy

// The README question pair, asked at a terminal and relayed through the
// plugin page (spc-2610031704338417 step 4, for itd-2610031348087517).
//
// Q1, keyed readme, stands alone: yes asks Q2; no stores readme.parts: [],
// which is never asked again; deciding later (the default, and end of input)
// stores nothing. Q2 asks one on/off question per offered part, keyed
// readme.<part>, in the fixed order: the host's question tool shows them as
// tabs of at most four, a terminal one at a time. A project not declared
// public is not asked about the release and build badges. Any decide-later
// answer stores nothing, since the stored set is one answer.
//
// Each part's question carries what the part is (its help here), its current
// setting, the change-later line naming --readme-parts, and an example: the
// exact lines it would write in this project, or the reason it writes none.
// Both front doors build their words from this help: the terminal through
// HelpedPrompter, the plugin page from the README view detection carries.

import (
	"strings"

	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// ReadmePromptKey keys Q1; each part's question is keyed ReadmePromptKey + "."
// + the part's name, as the status line's elements are keyed statusline.<el>.
const ReadmePromptKey = "readme"

// readmePartKeyPrefix keys each part's question.
const readmePartKeyPrefix = ReadmePromptKey + "."

// readmeChangeLater is every README question's change-later line.
const readmeChangeLater = "abcd ahoy install --readme-parts <list>|none"

// readmeNowUnset is a part's current setting while the pair is unanswered:
// abcd keeps no part until it is, so every part is off.
const readmeNowUnset = "not set, so off"

// readmeNotPublic is the one line saying the badges go by the declaration in
// abcd's settings, said where the project is not declared public.
const readmeNotPublic = "No release or build badge is offered: the project is not declared public in abcd's settings."

// readmeQ1Choices are Q1's answers; deciding later is its default.
var readmeQ1Choices = []string{"yes", "no", SetupLaterValue}

// readmeQ1 is Q1's help as every project sees it. readmeQ1For adds, for this
// project, which parts are offered.
var readmeQ1 = PromptHelp{
	Key: ReadmePromptKey,
	About: "abcd can keep parts of README.md current, read from the project, each in its own marked place; " +
		"your own words are never touched. Should abcd keep parts of README.md current?",
	ChangeLater: readmeChangeLater,
	Choices: []ChoiceHelp{
		{Value: "yes", Meaning: "Asks next which parts, one question for each part."},
		{Value: "no", Meaning: "Keeps no part, and is not asked again; abcd's parts are removed."},
		{Value: SetupLaterValue, Meaning: "Stores nothing, so the next setup run asks again."},
	},
}

// readmePartAbout says what each part is, ending on its own question.
var readmePartAbout = map[string]string{
	"release-badge": "A badge of the latest release, kept live by the forge. Keep the release badge in README.md?",
	"build-badge":   "A badge of whether the build passes, drawn by the forge. Keep the build badge in README.md?",
	"abcd-badge":    "A badge saying the project is managed with abcd. Keep the managed-with-abcd badge in README.md?",
	"licence":       "A line naming the licence, read from LICENSE. Keep the licence line in README.md?",
	"links":         "Links to the project's guides, such as its changelog. Keep the links in README.md?",
	"contents":      "A table of contents for the README's sections. Keep the table of contents in README.md?",
}

// readmePartOn is each part's default: the licence and the links on, the
// table of contents off ("if the owner wants one"), the release and build
// badges on where they are offered, and the managed-with-abcd badge off
// (decision 2).
var readmePartOn = map[string]bool{
	"release-badge": true, "build-badge": true, "abcd-badge": false,
	"licence": true, "links": true, "contents": false,
}

// readmePartChoices are a part's answers, in the order the question offers
// them.
var readmePartChoices = []ChoiceHelp{
	{Value: "on", Meaning: "Writes it, and rewrites it from the project on every setup run."},
	{Value: "off", Meaning: "Leaves it out; a copy abcd wrote before is removed."},
	{Value: SetupLaterValue, Meaning: "Stores no parts, so the next setup run asks again."},
}

// readmeForgeBadge reports whether the part is a forge badge, offered only in
// a project declared public.
func readmeForgeBadge(name string) bool { return name == "release-badge" || name == "build-badge" }

// readmePartDefault is the part's default answer, on or off.
func readmePartDefault(name string) string {
	if readmePartOn[name] {
		return "on"
	}
	return "off"
}

// readmeHelp is the help for a README question key as every project sees it,
// with no current setting and no example: those are per run (ReadmeView).
func readmeHelp(key string) (PromptHelp, bool) {
	if key == ReadmePromptKey {
		return readmeQ1, true
	}
	name, ok := strings.CutPrefix(key, readmePartKeyPrefix)
	if !ok {
		return PromptHelp{}, false
	}
	about, known := readmePartAbout[name]
	if !known {
		return PromptHelp{}, false
	}
	return PromptHelp{
		Key:         key,
		About:       about,
		ChangeLater: readmeChangeLater,
		Choices:     readmeChoicesFor(name),
	}, true
}

// readmeChoicesFor is a part's answers with its default first, then the
// other, then deciding later: a terminal's cursor starts on the default.
func readmeChoicesFor(name string) []ChoiceHelp {
	out := append([]ChoiceHelp(nil), readmePartChoices...)
	if !readmePartOn[name] {
		out[0], out[1] = out[1], out[0]
	}
	return out
}

// choiceValues is the values of choices, in order.
func choiceValues(choices []ChoiceHelp) []string {
	out := make([]string, len(choices))
	for i, c := range choices {
		out[i] = c.Value
	}
	return out
}

// ReadmeView is the README question as detection carries it while the pair
// is unanswered, for a front door that relays it: each part, and the
// questions themselves, built from the same help the terminal asks with.
type ReadmeView struct {
	// Public is the owner's declaration in abcd's settings (repo.visibility);
	// Declaration, where it is not public, is the one line saying the badges
	// go by it, said above Q2's first tab.
	Public      bool             `json:"public"`
	Declaration string           `json:"declaration,omitempty"`
	Parts       []ReadmePartView `json:"parts"`
	// Asks are Q1 alone, then the offered parts in calls of at most four
	// tabs, each as the shared question type.
	Asks []question.Ask `json:"asks"`
	// helps is each offered part's help, its example cut to the rows limit
	// once, which the terminal asks with.
	helps map[string]PromptHelp
}

// ReadmePartView is one part as the view carries it. Every string is
// terminal-safe: an example line and a reason are read from the project.
type ReadmePartView struct {
	Name    string `json:"name"`
	Offered bool   `json:"offered"`
	Now     string `json:"now"`
	Default string `json:"default"`
	// Example is the exact lines the part would write in this project;
	// Reason, where it would write none, why.
	Example []string `json:"example,omitempty"`
	Reason  string   `json:"reason,omitempty"`
	// short is Reason in a few words, where the derivation gives one; the
	// question says it when the whole reason does not fit.
	short string
}

// readmeView is the README view of the project at cwd, and false once the
// pair is answered (or the config cannot be read): only an unanswered pair is
// asked.
func readmeView(cwd string) (ReadmeView, bool) {
	cfg, err := readConfig(cwd)
	if err != nil {
		return ReadmeView{}, false
	}
	if _, answered := subMap(cfg, "readme")["parts"]; answered {
		return ReadmeView{}, false
	}
	vis, _ := stringVal(subMap(cfg, "repo"), "visibility")
	return buildReadmeView(cwd, vis), true
}

// buildReadmeView derives every part as it would be written in the project at
// cwd, whose declared visibility is vis, and builds the questions.
func buildReadmeView(cwd, vis string) ReadmeView {
	s := readmeSettings{Parts: readmePartOrder, Visibility: vis}
	v := ReadmeView{Public: s.public(), Parts: []ReadmePartView{}, Asks: []question.Ask{}, helps: map[string]PromptHelp{}}
	if !v.Public {
		v.Declaration = readmeNotPublic
	}
	for _, d := range deriveReadmeParts(cwd, s) {
		if readmePartIndex(d.Name) < 0 {
			continue
		}
		pv := ReadmePartView{
			Name:    d.Name,
			Offered: v.Public || !readmeForgeBadge(d.Name),
			Now:     readmeNowUnset,
			Default: readmePartDefault(d.Name),
		}
		if d.Reason == "" {
			for _, ln := range strings.Split(d.Body, "\n") {
				pv.Example = append(pv.Example, termsafe.Sanitize(ln))
			}
		} else {
			pv.Reason = termsafe.Sanitize(strings.TrimPrefix(d.Reason, d.Name+" omitted: "))
			pv.short = termsafe.Sanitize(d.Short)
		}
		v.Parts = append(v.Parts, pv)
	}
	n := 1
	v.Asks = append(v.Asks, question.Ask{Questions: []question.Question{
		valueQuestion(n, v.q1Help(), readmeQ1Choices, SetupLaterValue),
	}})
	var call []question.Question
	for _, pv := range v.offered() {
		n++
		h := pv.help()
		v.helps[pv.Name] = h
		call = append(call, valueQuestion(n, h, choiceValues(h.Choices), pv.Default))
		if len(call) == question.Default.QuestionsPerCall[1] {
			v.Asks = append(v.Asks, question.Ask{Questions: call})
			call = nil
		}
	}
	if len(call) > 0 {
		v.Asks = append(v.Asks, question.Ask{Questions: call})
	}
	for i, a := range v.Asks {
		v.Asks[i] = question.Safe(a)
	}
	return v
}

// offered is the parts the pair asks about, in the fixed order.
func (v ReadmeView) offered() []ReadmePartView {
	var out []ReadmePartView
	for _, pv := range v.Parts {
		if pv.Offered {
			out = append(out, pv)
		}
	}
	return out
}

// q1Help is Q1's help in this project: which parts are offered, and, where
// the project is not declared public, that the badges go by that declaration.
func (v ReadmeView) q1Help() PromptHelp {
	h := readmeQ1
	body, ask, _ := strings.Cut(h.About, " Should ")
	offers := "It offers a licence line, links to the guides, a table of contents, release and build badges, " +
		"and a managed-with-abcd badge, off unless you turn it on."
	if !v.Public {
		offers = "It offers a licence line, links to the guides, a table of contents and a managed-with-abcd " +
			"badge, off unless you turn it on. " + readmeNotPublic
	}
	h.About = body + " " + offers + " Should " + ask
	h.Now = "not set"
	return h
}

// help is the part's help in this project: its current setting, and its
// example cut to the lines that fit the rows a question may take, or the
// reason it writes none, said as fully as fits.
func (pv ReadmePartView) help() PromptHelp {
	h, _ := readmeHelp(readmePartKeyPrefix + pv.Name)
	h.Now = pv.Now
	h.ExampleNone = fitReason(h, pv.Default, pv.Reason, pv.short)
	h.Example, h.ExampleLeft = fitExample(h, pv.Example, pv.Default)
	return h
}

// readmeReasonTooLong is said where neither a part's reason nor its short form
// fits the rows a question may take; the view's reason keeps it whole.
const readmeReasonTooLong = "its reason is too long to show here"

// fitReason is the first of reasons (the whole reason, then its short form)
// that the question h asks still fits the rows limit with, else
// readmeReasonTooLong; "" where there is no reason.
func fitReason(h PromptHelp, def string, reasons ...string) string {
	if len(reasons) == 0 || reasons[0] == "" {
		return ""
	}
	for _, r := range reasons {
		h.ExampleNone = r
		if r != "" && fitsRows(h, def) {
			return r
		}
	}
	return readmeReasonTooLong
}

// fitsRows reports whether the question h asks, with def its default, fits
// the rows limit (question.Default.Rows at its columns).
func fitsRows(h PromptHelp, def string) bool {
	q := valueQuestion(0, h, choiceValues(h.Choices), def)
	q.Chip = setupChip(1)
	for _, f := range question.CheckLimits(question.Ask{Questions: []question.Question{q}}.Fields(), question.Default, question.Addressee{}) {
		if f.Rule == question.RuleRows {
			return false
		}
	}
	return true
}

// fitExample is lines, or as many of them as fit the question h asks within
// the rows limit (question.Default.Rows at its columns), and how many it
// leaves out, which the question's lead says (PromptHelp.ExampleIntro). The
// lines shown are exact; none is cut.
func fitExample(h PromptHelp, lines []string, def string) ([]string, int) {
	return fitLines(lines, func(ex []string, left int) bool {
		h.Example, h.ExampleLeft = ex, left
		return fitsRows(h, def)
	})
}

// fitLines is fitExample's cut, with fits judging the first lines of an
// example and how many it leaves out. Every line takes at least one row, so
// no example of more lines than the limit has rows can fit: the cut starts
// there, not at the last line, and fits is called at most
// question.Default.Rows+1 times however many lines there are (a README of
// thousands of headings, read whole, is one contents line each). Where not
// even one line fits, it shows none and leaves them all out.
func fitLines(lines []string, fits func(ex []string, left int) bool) ([]string, int) {
	if len(lines) == 0 || len(lines) <= question.Default.Rows && fits(lines, 0) {
		return lines, 0
	}
	for k := min(len(lines)-1, question.Default.Rows); k >= 1; k-- {
		if fits(lines[:k:k], len(lines)-k) {
			return lines[:k:k], len(lines) - k
		}
	}
	return nil, len(lines)
}

// asksReadme reports whether this run puts the pair to a person: the pair is
// unanswered (no flag, nothing stored), a person is answering at a terminal,
// the run is not --yes, and the config the answer is stored in exists.
func (a *applyCtx) asksReadme(ans readmeAnswer) bool {
	if ans.flagSet || ans.storedSet || a.autoYes || !atTerminal(a.prompter) || !a.has(ReadmeOfferGapID) {
		return false
	}
	cfg, err := readConfig(a.cwd)
	return err == nil && cfg != nil
}

// askReadme asks the pair and returns the chosen parts, and false when the
// answer is to decide later, at Q1 or at any part, which stores nothing.
// Taking a part's default answers it: on or off. Any answer but on or off
// (no answer at all, which a door that draws the question gives for deciding
// later, included) decides later, and the parts after it are not asked.
func (a *applyCtx) askReadme(vis string) ([]string, bool) {
	v := buildReadmeView(a.cwd, vis)
	switch strings.ToLower(strings.TrimSpace(promptHelped(a.prompter, v.q1Help(), readmeQ1Choices, SetupLaterValue))) {
	case "yes":
	case "no":
		return []string{}, true
	default:
		return nil, false
	}
	parts := []string{}
	for _, pv := range v.offered() {
		h := v.helps[pv.Name]
		switch strings.ToLower(strings.TrimSpace(promptHelped(a.prompter, h, choiceValues(h.Choices), pv.Default))) {
		case "on":
			parts = append(parts, pv.Name)
		case "off":
		default:
			return nil, false
		}
	}
	return parts, true
}
