package question_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/question"
)

// keyQuestion is a well-built question of the shared type: a chip, a paragraph
// and a list of material, the ask, two options, the decide-later option, and
// the Now: and Change later: lines.
func keyQuestion() question.Question {
	return question.Question{
		ID:   "key-home",
		Chip: "Setup Q1",
		Material: []question.Block{
			{Kind: question.KindParagraph, Text: "abcd needs the key your model service gave you, to ask it questions on your behalf."},
			{Kind: question.KindList, Items: []string{"The system keychain asks for your login password once.", "A file in your home folder is read by any program you run."}},
		},
		Ask: "Where should abcd keep the key?",
		Options: []question.Option{
			{Value: "keychain", Label: "In the system keychain", Meaning: "The key is locked with your login. Other programs must ask for it."},
			{Value: "file", Label: "In a file", Meaning: "The key sits in your home folder. It works where no keychain runs."},
		},
		Later:       question.Option{Value: "later", Label: "Decide later", Meaning: "Nothing is stored now. abcd asks again at the next setup."},
		Now:         "no key is stored",
		ChangeLater: "run the setup again",
	}
}

func one(q question.Question) question.Ask { return question.Ask{Questions: []question.Question{q}} }

// findingAt reports whether fs holds a structural finding for the tab and part.
func findingAt(fs []question.Finding, tab int, part string) bool {
	for _, f := range fs {
		if f.Tab == tab && f.Part == part && f.Rule == question.RuleStructure {
			return true
		}
	}
	return false
}

func listFindings(fs []question.Finding) string {
	var b strings.Builder
	for _, f := range fs {
		b.WriteString(f.String() + "\n")
	}
	return b.String()
}

// TestStructuralCheckAdmitsAWellBuiltAsk is the admitted case: a question with
// every structural part, and a long-list question, draw no finding.
func TestStructuralCheckAdmitsAWellBuiltAsk(t *testing.T) {
	long := keyQuestion()
	long.ID, long.Chip = "model", "Setup Q2"
	long.Options = nil
	long.List = &question.List{Choices: []question.Option{
		{Value: "m1", Label: "Model one", Meaning: "The first."},
		{Value: "m2", Label: "Model two", Meaning: "The second."},
	}}
	for name, a := range map[string]question.Ask{
		"one question":       one(keyQuestion()),
		"a long-list option": one(long),
		"two tabs":           {Questions: []question.Question{keyQuestion(), long}},
	} {
		if fs := question.Check(a); len(fs) != 0 {
			t.Errorf("%s: refused:\n%s", name, listFindings(fs))
		}
	}
}

// TestStructuralCheckRefusesNamingThePart is the step's refusal list: a
// question without Later, an empty ask, and duplicate values, each refused
// naming the question (its tab) and the part, with the rest of the structure.
func TestStructuralCheckRefusesNamingThePart(t *testing.T) {
	cases := []struct {
		name string
		edit func(*question.Ask)
		tab  int
		part string
	}{
		{"no Later value", func(a *question.Ask) { a.Questions[0].Later.Value = "" }, 1, "later"},
		{"no Later at all", func(a *question.Ask) { a.Questions[0].Later = question.Option{} }, 1, "later"},
		{"an empty ask", func(a *question.Ask) { a.Questions[0].Ask = "  " }, 1, "ask"},
		{"a value twice", func(a *question.Ask) { a.Questions[0].Options[1].Value = "keychain" }, 1, "option 2 value"},
		{"Later repeating an option's value", func(a *question.Ask) { a.Questions[0].Later.Value = "file" }, 1, "later"},
		{"a block of no known kind", func(a *question.Ask) { a.Questions[0].Material[1].Kind = "table" }, 1, "material block 2 kind"},
		{"no id", func(a *question.Ask) { a.Questions[0].ID = "" }, 1, "id"},
		{"no chip", func(a *question.Ask) { a.Questions[0].Chip = "" }, 1, "chip"},
		{"neither options nor a list", func(a *question.Ask) { a.Questions[0].Options = nil }, 1, "options"},
		{"both options and a list", func(a *question.Ask) {
			a.Questions[0].List = &question.List{Choices: []question.Option{{Value: "x", Label: "X"}}}
		}, 1, "options"},
		{"an option without a value", func(a *question.Ask) { a.Questions[0].Options[0].Value = "" }, 1, "option 1 value"},
		{"an option without a label", func(a *question.Ask) { a.Questions[0].Options[0].Label = "" }, 1, "option 1 label"},
		{"a list choice twice", func(a *question.Ask) {
			a.Questions[0].Options = nil
			a.Questions[0].List = &question.List{Choices: []question.Option{{Value: "m", Label: "M"}, {Value: "m", Label: "N"}}}
		}, 1, "list choice 2 value"},
		{"the second tab's ask empty", func(a *question.Ask) {
			q := keyQuestion()
			q.ID, q.Ask = "second", ""
			a.Questions = append(a.Questions, q)
		}, 2, "ask"},
		{"two questions with one id", func(a *question.Ask) { a.Questions = append(a.Questions, keyQuestion()) }, 2, "id"},
		{"no questions", func(a *question.Ask) { a.Questions = nil }, 0, "questions"},
		{"five questions", func(a *question.Ask) {
			for i := range 4 {
				q := keyQuestion()
				q.ID = q.ID + string(rune('a'+i))
				a.Questions = append(a.Questions, q)
			}
		}, 0, "questions"},
	}
	for _, c := range cases {
		a := one(keyQuestion())
		c.edit(&a)
		fs := question.Check(a)
		if !findingAt(fs, c.tab, c.part) {
			t.Errorf("%s: want a structural finding on tab %d, part %q; got:\n%s", c.name, c.tab, c.part, listFindings(fs))
		}
		for _, f := range fs {
			if f.Remedy == "" || f.Limit == "" {
				t.Errorf("%s: finding %q carries no limit or remedy", c.name, f.String())
			}
		}
	}
}

// TestAskJSONShapeIsTheSpecs holds the type's JSON keys to the shape the spec
// writes down, so a runner's receipt and the host path read the same names.
func TestAskJSONShapeIsTheSpecs(t *testing.T) {
	q := keyQuestion()
	b, err := json.Marshal(one(q))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string][]map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	got := raw["questions"][0]
	for _, k := range []string{"id", "chip", "material", "ask", "options", "later", "now", "change_later"} {
		if _, ok := got[k]; !ok {
			t.Errorf("key %q missing from %s", k, b)
		}
	}
	if _, ok := got["list"]; ok {
		t.Errorf("an options question carries a list key: %s", b)
	}
	block := got["material"].([]any)[0].(map[string]any)
	if block["kind"] != "paragraph" || block["text"] == nil {
		t.Errorf("paragraph block shape: %v", block)
	}
	opt := got["later"].(map[string]any)
	for _, k := range []string{"value", "label", "meaning"} {
		if _, ok := opt[k]; !ok {
			t.Errorf("option key %q missing: %v", k, opt)
		}
	}
	var back question.Ask
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if fs := question.Check(back); len(fs) != 0 {
		t.Errorf("the round-tripped question is refused:\n%s", listFindings(fs))
	}
}

// TestFieldsPlacesNowAndChangeLaterBeforeTheAsk is the field view's mapping
// (spc-2610030944505997, "The field view"): the chip to the header; the
// material's blocks, then the Now: and Change later: lines, then the ask, to
// the question text; the options and then Later to the options.
func TestFieldsPlacesNowAndChangeLaterBeforeTheAsk(t *testing.T) {
	f := one(keyQuestion()).Fields()
	if len(f.Tabs) != 1 {
		t.Fatalf("tabs = %d, want 1", len(f.Tabs))
	}
	tab := f.Tabs[0]
	if tab.Header != "Setup Q1" {
		t.Errorf("header = %q", tab.Header)
	}
	want := strings.Join([]string{
		"abcd needs the key your model service gave you, to ask it questions on your behalf.",
		"",
		"- The system keychain asks for your login password once.",
		"- A file in your home folder is read by any program you run.",
		"",
		"Now: no key is stored",
		"Change later: run the setup again",
		"",
		"Where should abcd keep the key?",
	}, "\n")
	if tab.Text != want {
		t.Errorf("text =\n%s\nwant\n%s", tab.Text, want)
	}
	labels := []string{}
	for _, o := range tab.Options {
		labels = append(labels, o.Label)
	}
	if strings.Join(labels, "|") != "In the system keychain|In a file|Decide later" {
		t.Errorf("options = %v, want the options then Later", labels)
	}
	if tab.Options[0].Description != keyQuestion().Options[0].Meaning {
		t.Errorf("an option's meaning is not its description: %q", tab.Options[0].Description)
	}
}

// TestFieldsOmitsAnAbsentNowLine maps a question without Now: or Change
// later: faithfully, so the limits check names the gap rather than the mapping
// inventing a value.
func TestFieldsOmitsAnAbsentNowLine(t *testing.T) {
	q := keyQuestion()
	q.Now, q.ChangeLater = "", ""
	text := one(q).Fields().Tabs[0].Text
	if strings.Contains(text, question.Default.NowPrefix) || strings.Contains(text, question.Default.ChangeLaterPrefix) {
		t.Errorf("an absent line was invented:\n%s", text)
	}
	fs := question.CheckLimits(one(q).Fields(), question.Default, question.Addressee{})
	if !hasRule(fs, question.RuleNowAndChangeLater) {
		t.Errorf("the limits check does not name the missing lines:\n%s", listFindings(fs))
	}
}

func hasRule(fs []question.Finding, r question.Rule) bool {
	for _, f := range fs {
		if f.Rule == r {
			return true
		}
	}
	return false
}

// TestStructuralCheckRefusesAMeaningPointingAtMissingMaterial is
// iss-2610071528375981's last clause: an answer whose meaning points at text
// above ("Writes what the text above describes.") is refused when the
// question carries no material, since there is nothing above it to read. The
// same meaning over material is admitted.
func TestStructuralCheckRefusesAMeaningPointingAtMissingMaterial(t *testing.T) {
	confirm := question.Question{
		ID:   "approve.config-change",
		Chip: "Setup Q3",
		Ask:  "Apply config-change changes?",
		Options: []question.Option{
			{Value: "yes", Label: "Yes, make the change", Meaning: "Writes what the text above describes."},
			{Value: "no", Label: "No, leave it", Meaning: "Writes nothing, so the next install asks again."},
		},
		Later: question.Option{Value: "later", Label: "Decide later", Meaning: "Declines for now and writes nothing."},
	}
	fs := question.Check(one(confirm))
	if !findingAt(fs, 1, "option 1 meaning") {
		t.Fatalf("a meaning pointing at text above, over no material, is admitted:\n%s", listFindings(fs))
	}
	listed := confirm
	listed.Options = append([]question.Option(nil), confirm.Options...)
	listed.Options[0].Meaning = "Writes the 2 changes listed above."
	if fs := question.Check(one(listed)); !findingAt(fs, 1, "option 1 meaning") {
		t.Fatalf("a meaning pointing at a list above, over no material, is admitted:\n%s", listFindings(fs))
	}
	listed.Material = []question.Block{{Kind: question.KindList, Items: []string{"repo.visibility not set", "docs.target not set"}}}
	if fs := question.Check(one(listed)); len(fs) > 0 {
		t.Fatalf("the same meaning over material is refused:\n%s", listFindings(fs))
	}
}
