package question

import (
	"fmt"
	"regexp"
	"strings"
)

// Ask is what one turn puts to the person: one question, or up to four parts
// of one thing shown as tabs (itd-2610030810350727 decision 10). It is the
// structured question both front doors use (spc-2610030911534855, "One
// question type, defined here first"): abcd draws it in a plain Terminal, and
// Fields maps it onto the host's question tool.
type Ask struct {
	Questions []Question `json:"questions"`
}

// Question is one question: the thing being decided first, then the one plain
// question, then the answers and the way to decide later.
type Question struct {
	ID          string   `json:"id"`                     // stable: a setup key, or the turn's ordinal
	Chip        string   `json:"chip"`                   // who it is for and which question: "Setup Q1"
	Material    []Block  `json:"material"`               // the thing being decided: paragraphs and lists
	Ask         string   `json:"ask"`                    // the one plain question
	Options     []Option `json:"options"`                // the substantive answers
	Later       Option   `json:"later"`                  // the way to decide later, always present
	Now         string   `json:"now,omitempty"`          // what holds now (itd-2610030810350727 decision 4)
	ChangeLater string   `json:"change_later,omitempty"` // how to change the answer later (decision 4)
	List        *List    `json:"list,omitempty"`         // the long-list variant
	// Typed is the prompt for a typed answer: a question with a typed part
	// takes any text as its answer besides its options (spc-2610031241482088,
	// "The typed part on the question type"). The host's question tool takes
	// it in its free-text row, so such a question carries no side preview,
	// which would remove that row; the Terminal draws it as one line after
	// the options. It counts as one option toward the limits' floor.
	Typed string `json:"typed,omitempty"`
}

// TypedTextPrefix opens the line of the question text that says what the
// typed part takes.
const TypedTextPrefix = "Or type in the row below:"

// TypedRowLabel is the label of the one option a front door adds to a
// question with a typed part when the host's question tool takes fewer
// listed options than the question has (a typed part and decide later alone):
// it points at the host's free-text row and answers nothing itself, so an
// interview that receives it asks the question again.
const TypedRowLabel = "Type my own answer"

// Block is one piece of a question's material: a paragraph, or a list.
type Block struct {
	Kind  string   `json:"kind"` // KindParagraph or KindList
	Text  string   `json:"text,omitempty"`
	Items []string `json:"items,omitempty"`
}

// The kinds a material block takes.
const (
	KindParagraph = "paragraph"
	KindList      = "list"
)

// Option is one answer: the value recorded, a label of a few words, and what
// choosing it means, with its trade-off.
type Option struct {
	Value   string `json:"value"`
	Label   string `json:"label"`
	Meaning string `json:"meaning"`
}

// List is a choice among many (300 models): the question's Options stay empty
// and the choices live here; the Later option still applies.
type List struct {
	Choices []Option `json:"choices"`
}

// RuleStructure names a structural finding: a part a question cannot be
// drawn, answered or recorded without. Check reports it; the limits are
// CheckLimits's.
const RuleStructure Rule = "structure"

// Check is the structural check: every question has an id (unique within the
// Ask), a chip, an ask, a Later option with a value, and options, a list, or a
// typed part; every option and list choice has a value and a label; values are
// unique within a question, Later's included; no answer's meaning points at
// text above a question that carries no material (pointsAbove); an Ask holds
// as many questions as Default.QuestionsPerCall admits. It returns every finding at once, each
// naming the question (its tab, counted from one; zero is the Ask as a whole)
// and the part, and holds no limit of its own: how much each part may hold is
// CheckLimits's, run on the field view (Fields).
func Check(a Ask) []Finding {
	var out []Finding
	if n, span := len(a.Questions), Default.QuestionsPerCall; n < span[0] || n > span[1] {
		out = append(out, Finding{
			Part:   "questions",
			Rule:   RuleStructure,
			Value:  fmt.Sprintf("%d questions", n),
			Limit:  fmt.Sprintf("%d to %d questions in one ask", span[0], span[1]),
			Remedy: "Ask one question, or up to four parts of one thing as tabs.",
		})
	}
	ids := map[string]bool{}
	for i, q := range a.Questions {
		tab := i + 1
		add := func(part, value, limit, remedy string) {
			out = append(out, Finding{Tab: tab, Part: part, Rule: RuleStructure, Value: echo(value), Limit: limit, Remedy: remedy})
		}
		switch id := strings.TrimSpace(q.ID); {
		case id == "":
			add("id", q.ID, "required", "Give the question a stable id: its setup key, or the turn's ordinal.")
		case ids[id]:
			add("id", q.ID, "unique within the ask", "Give each question of the ask its own id.")
		default:
			ids[id] = true
		}
		if strings.TrimSpace(q.Chip) == "" {
			add("chip", q.Chip, "required", "Head the question with its chip, such as \"Setup Q1\".")
		}
		for j, b := range q.Material {
			if b.Kind != KindParagraph && b.Kind != KindList {
				add(fmt.Sprintf("material block %d kind", j+1), b.Kind, fmt.Sprintf("%q or %q", KindParagraph, KindList),
					"Write the material as paragraphs and lists.")
			}
		}
		if strings.TrimSpace(q.Ask) == "" {
			add("ask", q.Ask, "required", "End the question with the one plain question it asks.")
		}
		switch {
		case len(q.Options) == 0 && (q.List == nil || len(q.List.Choices) == 0) && strings.TrimSpace(q.Typed) == "":
			add("options", "", "options, a list or a typed part", "Offer the answers as options, a long list as a list, or a typed answer as the typed part.")
		case len(q.Options) > 0 && q.List != nil:
			add("options", "", "options or a list, not both", "Offer the answers as options or as a list, never both.")
		}
		seen := map[string]bool{}
		value := func(part string, o Option) {
			v := strings.TrimSpace(o.Value)
			switch {
			case v == "":
				add(part+" value", o.Value, "required", "Give the answer the value that is recorded when it is chosen.")
			case seen[v]:
				add(part+" value", o.Value, "unique within the question", "Give each answer of the question its own value.")
			default:
				seen[v] = true
			}
		}
		labelled := func(part string, o Option) {
			if strings.TrimSpace(o.Label) == "" {
				add(part+" label", o.Label, "required", "Give the answer a label of a few words.")
			}
		}
		for j, o := range q.Options {
			part := fmt.Sprintf("option %d", j+1)
			value(part, o)
			labelled(part, o)
		}
		if len(q.Material) == 0 {
			for j, o := range append(append([]Option(nil), q.Options...), q.Later) {
				if !pointsAbove(o.Meaning) {
					continue
				}
				part := fmt.Sprintf("option %d meaning", j+1)
				if j == len(q.Options) {
					part = "later meaning"
				}
				add(part, o.Meaning, "material above the question for the meaning to point at",
					"Quote what the answer acts on in the material, or say what it does without pointing above.")
			}
		}
		if q.List != nil {
			for j, o := range q.List.Choices {
				part := fmt.Sprintf("list choice %d", j+1)
				value(part, o)
				labelled(part, o)
			}
		}
		switch v := strings.TrimSpace(q.Later.Value); {
		case v == "":
			add("later", q.Later.Value, "a decide-later option with a value", "Offer the way to decide later, with the value recorded when it is chosen.")
		case seen[v]:
			add("later", q.Later.Value, "unique within the question", "Give the decide-later option a value no other answer has.")
		default:
			labelled("later", q.Later)
		}
	}
	return out
}

// aboveRe finds a meaning pointing at text above its question: "the text
// above", "listed above", "described above" (iss-2610071528375981).
var aboveRe = regexp.MustCompile(`(?i)\b(?:text|list|listed|described|shown|named|set out)\s+above\b`)

// pointsAbove reports whether meaning points at text above its question.
func pointsAbove(meaning string) bool { return aboveRe.MatchString(meaning) }

// Fields maps the Ask onto the field view the limits are checked on
// (spc-2610030944505997, "The field view"): the chip to the header; the
// material's blocks, then the Now: and Change later: lines, then the ask, to
// the question text; the options (a long list's choices) and then Later to the
// options; the typed part to the free-text row (Tab.Typed), its prompt said
// in the text just before the ask. A line the question does not carry is left out, so the limits
// check names the gap rather than the mapping inventing a value.
func (a Ask) Fields() Fields {
	f := Fields{Tabs: make([]Tab, 0, len(a.Questions))}
	for _, q := range a.Questions {
		var parts []string
		for _, b := range q.Material {
			parts = append(parts, b.text())
		}
		var state []string
		if q.Now != "" {
			state = append(state, Default.NowPrefix+" "+q.Now)
		}
		if q.ChangeLater != "" {
			state = append(state, Default.ChangeLaterPrefix+" "+q.ChangeLater)
		}
		if len(state) > 0 {
			parts = append(parts, strings.Join(state, "\n"))
		}
		if q.Typed != "" {
			// The host's free-text row carries no prompt of its own, so the
			// typed part's is said in the text, just before the question.
			parts = append(parts, TypedTextPrefix+" "+q.Typed+".")
		}
		parts = append(parts, q.Ask)
		options := q.Options
		if q.List != nil {
			options = q.List.Choices
		}
		choices := make([]Choice, 0, len(options)+1)
		for _, o := range append(append([]Option(nil), options...), q.Later) {
			choices = append(choices, Choice{Label: o.Label, Description: o.Meaning})
		}
		f.Tabs = append(f.Tabs, Tab{Header: q.Chip, Text: strings.Join(parts, "\n\n"), Options: choices, Typed: q.Typed})
	}
	return f
}

// text is the block as question text: a paragraph as it is, a list one
// "- item" line per item.
func (b Block) text() string {
	if b.Kind != KindList {
		return b.Text
	}
	lines := make([]string, len(b.Items))
	for i, it := range b.Items {
		lines[i] = "- " + it
	}
	return strings.Join(lines, "\n")
}
