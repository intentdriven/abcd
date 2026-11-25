package question

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// The product thinker's knowledge floor in the check (iss-2611252121330106):
// the register rule names each word below the floor that a tab addressed to
// the product thinker uses without introducing it in one sentence first.

// floorPage is the floor's own page, read from the repository.
func floorPage(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", filepath.FromSlash(KnowledgeFloorPage)))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// floorPageList is the page's table of words, one FloorWord per row.
func floorPageList(t *testing.T) []FloorWord {
	t.Helper()
	_, section, found := strings.Cut(floorPage(t), "\n## Words below the product thinker's floor\n")
	if !found {
		t.Fatalf("%s has no section listing the words below the product thinker's floor", KnowledgeFloorPage)
	}
	section, _, _ = strings.Cut(section, "\n## ")
	var out []FloorWord
	for _, ln := range strings.Split(section, "\n") {
		if !strings.HasPrefix(ln, "| ") || strings.HasPrefix(ln, "| Word ") {
			continue
		}
		cells := strings.Split(strings.Trim(ln, "| "), "|")
		if len(cells) != 3 {
			t.Fatalf("a row of the list is not three cells: %q", ln)
		}
		w := FloorWord{Word: strings.TrimSpace(cells[0]), Plain: strings.TrimSpace(cells[2])}
		if also := strings.TrimSpace(cells[1]); also != "none" {
			for _, a := range strings.Split(also, ",") {
				w.Also = append(w.Also, strings.TrimSpace(a))
			}
		}
		out = append(out, w)
	}
	return out
}

// TestBelowFloorIsTheFloorPagesList: the words the check refuses are the
// floor page's own list, word for word, in the same order, so the page and
// the check cannot differ.
func TestBelowFloorIsTheFloorPagesList(t *testing.T) {
	page, code := floorPageList(t), belowFloor
	if len(code) == 0 {
		t.Fatal("the check holds no word below the floor")
	}
	same := slices.EqualFunc(page, code, func(a, b FloorWord) bool {
		return a.Word == b.Word && a.Plain == b.Plain && slices.Equal(a.Also, b.Also)
	})
	if !same {
		t.Errorf("the floor page's list and the check's differ:\npage %q\ncode %q", page, code)
	}
}

// TestFloorPageProseAndListNameTheSameWords: the sentence of the page that
// says what the product thinker is not assumed to know names a word of the
// list in every item, and every word of the list is an item of the sentence,
// so the prose and the list on the page cannot differ either.
func TestFloorPageProseAndListNameTheSameWords(t *testing.T) {
	prose := strings.Join(strings.Fields(floorPage(t)), " ")
	_, sentence, found := strings.Cut(prose, "They are not assumed to know ")
	if !found {
		t.Fatal("the floor page has no sentence saying what the product thinker is not assumed to know")
	}
	sentence, _, _ = strings.Cut(sentence, ".")
	named := map[string]bool{}
	for _, item := range regexp.MustCompile(`[,()]|\b(?:or|and)\b`).Split(sentence, -1) {
		item = regexp.MustCompile(`^(?:an?|the) `).ReplaceAllString(strings.TrimSpace(item), "")
		if item == "" {
			continue
		}
		i := slices.IndexFunc(floorPageList(t), func(w FloorWord) bool { return item == w.Word || item == w.Word+"s" })
		if i < 0 {
			t.Errorf("the prose names %q, which the list does not", item)
			continue
		}
		named[floorPageList(t)[i].Word] = true
	}
	for _, w := range floorPageList(t) {
		if !named[w.Word] {
			t.Errorf("the list names %q, which the prose does not", w.Word)
		}
	}
}

// TestPlainWordsAreAboveTheFloor: what the list says to use instead names no
// word below the floor itself.
func TestPlainWordsAreAboveTheFloor(t *testing.T) {
	for _, w := range belowFloor {
		if strings.TrimSpace(w.Plain) == "" {
			t.Errorf("%q has no plain words", w.Word)
		}
		tab := wellBuiltTab()
		tab.Text = "For example: " + w.Plain + ".\n\n" + tab.Text
		if fs := CheckLimits(one(tab), Default, productThinker()); len(fs) != 0 {
			t.Errorf("the plain words for %q are refused:\n%s", w.Word, render(fs))
		}
	}
}

// internalsTab is the question of iss-2611252121330106: put to the product
// thinker, worded in internals.
func internalsTab() Tab {
	return Tab{
		Header: "Q1",
		Text: strings.Join([]string{
			"A lane works in its own worktree of the checkout, and abcd puts the pull request in the merge queue once CI has passed on the commit and the checksum matches.",
			"",
			"Now: not applicable",
			"Change later: not applicable",
			"",
			"Who may let abcd do that?",
		}, "\n"),
		Options: []Choice{
			{Label: "Anyone", Description: "Every person here may. It costs a second look."},
			{Label: "Only me", Description: "You alone may. It costs your time."},
			{Label: "Decide later", Description: "Nothing changes now. The question comes back at the next interview."},
		},
	}
}

// valuesOf lists the values a set of findings names, in order.
func valuesOf(fs []Finding) []string {
	out := make([]string, len(fs))
	for i, f := range fs {
		out[i] = f.Value
	}
	return out
}

// TestProductThinkerQuestionBelowTheFloorIsRefused (iss-2611252121330106):
// each word below the floor is its own finding of the register rule, naming
// the part, the word and the plain words, and each refuses, as a record number
// does; the same question is admitted for the facilitator and where nobody is
// named, and an older-form chip decides its own tab.
func TestProductThinkerQuestionBelowTheFloorIsRefused(t *testing.T) {
	tab := internalsTab()
	fs := CheckLimits(one(tab), Default, productThinker())
	onlyRule(t, fs, RuleRegister)
	want := []string{"worktree", "checkout", "merge queue", "CI", "commit", "checksum"}
	if got := valuesOf(fs); !slices.Equal(got, want) {
		t.Errorf("words named %q, want %q in reading order", got, want)
	}
	for _, f := range fs {
		if f.Tab != 1 || f.Part != "question text" || !RefusingFinding(f) {
			t.Errorf("a finding names the wrong part or does not refuse: %+v", f)
		}
	}
	checksum := fs[len(fs)-1]
	if !strings.Contains(checksum.Remedy, `"a fingerprint saying two copies are identical"`) || !strings.Contains(checksum.Remedy, "one sentence") {
		t.Errorf("the remedy names no plain words, or not the one sentence: %q", checksum.Remedy)
	}
	for name, who := range map[string]Addressee{"facilitator": facilitator(), "unnamed": unnamed()} {
		if fs := CheckLimits(one(tab), Default, who); len(fs) != 0 {
			t.Errorf("%s: refused:\n%s", name, render(fs))
		}
	}
	tab.Header = "Product Q1"
	if fs := CheckLimits(one(tab), Default, facilitator()); len(fs) != len(want) {
		t.Errorf("a Product chip: %d findings, want %d:\n%s", len(fs), len(want), render(fs))
	}
	tab.Header = "Tech Q1"
	if fs := CheckLimits(one(tab), Default, productThinker()); len(fs) != 0 {
		t.Errorf("a Tech chip refused:\n%s", render(fs))
	}
}

// TestBelowFloorWordsAreFoundInEveryField: a word is found in any field of the
// tab, and a word used twice is one finding, at its first use.
func TestBelowFloorWordsAreFoundInEveryField(t *testing.T) {
	for _, c := range []struct {
		name, part string
		edit       func(*Tab)
	}{
		{"text", "question text", func(tb *Tab) { tb.Text = "The commit is kept.\n\n" + tb.Text }},
		{"label", "option 1 label", func(tb *Tab) { tb.Options[0].Label = "Keep the commit" }},
		{"description", "option 2 description", func(tb *Tab) { tb.Options[1].Description = "The commit is kept. It costs room." }},
		{"twice", "option 1 label", func(tb *Tab) {
			tb.Options[0].Label = "Keep the commit"
			tb.Options[1].Description = "Every commit goes. It costs the history."
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			tab := wellBuiltTab()
			c.edit(&tab)
			fs := CheckLimits(one(tab), Default, productThinker())
			onlyRule(t, fs, RuleRegister)
			if len(fs) != 1 || fs[0].Part != c.part || fs[0].Value != "commit" {
				t.Errorf("want one finding for commit in %s, got:\n%s", c.part, render(fs))
			}
		})
	}
}

// TestBelowFloorWordsAreWholeWords: a word is found in either case, in the
// plural and in a possessive, never inside a longer word, a hyphenated one
// included, and a phrase is its own word, so a merge queue is not also a
// merge.
func TestBelowFloorWordsAreWholeWords(t *testing.T) {
	for text, want := range map[string][]string{
		"Two branches wait.":                  {"branches"},
		"The Checkouts are listed.":           {"Checkouts"},
		"The branch's name is kept.":          {"branch"},
		"Both merge\nqueues wait.":            {"merge queues"},
		"A merge-queue waits.":                {"merge-queue"},
		"A merge waits in the merge queue.":   {"merge", "merge queue"},
		"It is kept (a worktree).":            {"worktree"},
		"CI ran.":                             {"CI"},
		"A symlink, or a symbolic link.":      {"symlink"},
		"Two environment variables are set.":  {"environment variables"},
		"The record ids are kept.":            {"record ids"},
		"It is in a nutshell.":                nil,
		"The hookah shop.":                    nil,
		"They emerge, and the emerged merger": nil,
		"The committee is committed to it.":   nil,
		"Let abcd auto-merge with TF==auto.":  nil,
		"A pre-commit check.":                 nil,
		"The merge_queue setting.":            nil,
		"The branché café is open.":           nil,
		"A decision, with precision.":         nil,
		"Check out the checking-out counter.": nil,
		"A file, a folder, a name, a version, an account, a link, and a permission someone grants.": nil,
	} {
		tab := wellBuiltTab()
		tab.Text = text + "\n\n" + tab.Text
		if got := valuesOf(CheckLimits(one(tab), Default, productThinker())); !slices.Equal(got, want) && len(got)+len(want) > 0 {
			t.Errorf("%q: words named %q, want %q", text, got, want)
		}
	}
}

// TestAWordIntroducedInOneSentencePasses is the floor's own rule: a word the
// tab's text introduces in one sentence, before any other use of it, passes
// everywhere on that tab, and on that tab alone; a use before the sentence, an
// annotation beside the use, or a sentence that only uses the word does not.
func TestAWordIntroducedInOneSentencePasses(t *testing.T) {
	for _, c := range []struct {
		name, text string
		passes     bool
	}{
		{"is", "A checksum is a fingerprint saying two copies are identical. Each checksum is kept.", true},
		{"are", "Checksums are the fingerprints saying two copies are identical.", true},
		{"means", "Checksum means a fingerprint saying two copies are identical.", true},
		{"the", "The checksum is the fingerprint saying two copies are identical.", true},
		{"called", "A fingerprint saying two copies are identical is called a checksum. Each checksum is kept.", true},
		{"quoted", "A \"checksum\" is a fingerprint saying two copies are identical.", true},
		{"not the first sentence", "Two copies are kept. A checksum is a fingerprint saying they are identical.", true},
		{"a list item", "Two things are kept:\n- A checksum is a fingerprint saying two copies are identical.", true},
		{"wrapped", "A checksum is a\nfingerprint saying two copies are identical.", true},
		{"used first", "Each checksum is kept. A checksum is a fingerprint saying two copies are identical.", false},
		{"annotated", "Each checksum (a fingerprint saying two copies are identical) is kept.", false},
		{"only used", "The checksum is missing from the list.", false},
		{"asked", "Is the checksum a fingerprint?", false},
		{"too short to say", "The checksum is the same.", false},
		{"inside a sentence", "We know a checksum is a fingerprint saying two copies are identical.", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			tab := wellBuiltTab()
			tab.Text = c.text + "\n\n" + tab.Text
			tab.Options[0].Description = "The checksum is kept. It costs a few characters."
			fs := CheckLimits(one(tab), Default, productThinker())
			if c.passes && len(fs) != 0 {
				t.Errorf("an introduced word refused:\n%s", render(fs))
			}
			if !c.passes {
				onlyRule(t, fs, RuleRegister)
			}
		})
	}
	// On the tab that introduces it alone.
	first, second := wellBuiltTab(), wellBuiltTab()
	first.Text = "A checksum is a fingerprint saying two copies are identical.\n\n" + first.Text
	second.Header = "Product Q3"
	second.Options[0].Description = "The checksum is kept. It costs a few characters."
	fs := CheckLimits(Fields{Tabs: []Tab{first, second}}, Default, productThinker())
	if len(fs) != 1 || fs[0].Tab != 2 || fs[0].Value != "checksum" {
		t.Errorf("want one finding on the second tab, got:\n%s", render(fs))
	}
	// An introduction that itself uses a word below the floor passes the
	// word it introduces and not the one it uses.
	tab := wellBuiltTab()
	tab.Text = "A merge queue is the line where each commit waits its turn.\n\n" + tab.Text
	if got := valuesOf(CheckLimits(one(tab), Default, productThinker())); !slices.Equal(got, []string{"commit"}) {
		t.Errorf("words named %q, want commit alone", got)
	}
	// An option's description introduces nothing: the material does.
	tab = wellBuiltTab()
	tab.Options[0].Description = "A checksum is a fingerprint saying two copies are identical."
	onlyRule(t, CheckLimits(one(tab), Default, productThinker()), RuleRegister)
}

// TestRegisterRefusalLineNamesEveryWord: the hook's one line per rule and tab
// names every value the register rule found there, not the first alone, so
// one retry fixes them all; another rule's line keeps the count.
func TestRegisterRefusalLineNamesEveryWord(t *testing.T) {
	tab := internalsTab()
	tab.Options[0].Description = "As iss-2609202058058301 asked. It costs a second look."
	groups := GroupFindings(CheckLimits(one(tab), Default, productThinker()))
	if len(groups) != 1 {
		t.Fatalf("want one group, got %+v", groups)
	}
	line := groups[0].String()
	for _, w := range []string{"iss-2609202058058301", `"worktree"`, `"checkout"`, `"merge queue"`, `"CI"`, `"commit"`, `"checksum"`} {
		if !strings.Contains(line, w) {
			t.Errorf("the line does not name %s:\n%s", w, line)
		}
	}
	if !strings.HasSuffix(line, `(and 6 more such lines: "worktree", "checkout", "merge queue", "CI", "commit", "checksum")`) {
		t.Errorf("the line does not close on the rest:\n%s", line)
	}
}
