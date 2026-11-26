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
		if len(cells) != 4 {
			t.Fatalf("a row of the list is not four cells: %q", ln)
		}
		list := func(cell string) []string {
			var out []string
			if cell = strings.TrimSpace(cell); cell != "none" {
				for _, a := range strings.Split(cell, ",") {
					out = append(out, strings.TrimSpace(a))
				}
			}
			return out
		}
		w := FloorWord{Word: strings.TrimSpace(cells[0]), Short: list(cells[2]), Plain: strings.TrimSpace(cells[3])}
		// A negated form carries its own plain words in brackets.
		for _, a := range list(cells[1]) {
			if form, plain, ok := strings.Cut(a, " ("); ok {
				w.Not, w.NotPlain = append(w.Not, form), strings.TrimSuffix(plain, ")")
				continue
			}
			w.Also = append(w.Also, a)
		}
		out = append(out, w)
	}
	return out
}

// TestBelowFloorIsTheFloorPagesList: the words the check refuses are the
// floor page's own list, word for word, in the same order, so the page and
// the check cannot differ.
func TestBelowFloorIsTheFloorPagesList(t *testing.T) {
	page, code := floorPageList(t), BelowFloor()
	if len(code) == 0 {
		t.Fatal("the check holds no word below the floor")
	}
	same := slices.EqualFunc(page, code, func(a, b FloorWord) bool {
		return a.Word == b.Word && a.Plain == b.Plain && slices.Equal(a.Also, b.Also) && slices.Equal(a.Short, b.Short) &&
			slices.Equal(a.Not, b.Not) && a.NotPlain == b.NotPlain
	})
	if !same {
		t.Errorf("the floor page's list and the check's differ:\npage %q\ncode %q", page, code)
	}
	code[0].Word = "changed"
	code[0].Also = append(code[0].Also, "changed")
	if BelowFloor()[0].Word == "changed" || slices.Contains(BelowFloor()[0].Also, "changed") {
		t.Error("a caller's edit to the returned words reached the next caller")
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
	want := []string{"worktree", "checkout", "pull request", "merge queue", "CI", "commit", "checksum"}
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
	// A word in its everyday sense, or the product's own word, is still a
	// word the check cannot tell apart: the remedy says to introduce it once.
	if !strings.Contains(checksum.Remedy, "a shop's checkout") || !strings.Contains(checksum.Remedy, "a bank's branch") || !strings.Contains(checksum.Remedy, "introduce it once") {
		t.Errorf("the remedy does not name the everyday sense and the product's own word: %q", checksum.Remedy)
	}
	// The way out it names is one the check admits: the word bare or after
	// "a" as the subject, or as the first word of the product's own term.
	for _, want := range []string{`"A push is ..."`, `"A push notification is ..."`} {
		if !strings.Contains(checksum.Remedy, want) {
			t.Errorf("the remedy does not show the introduction %s: %q", want, checksum.Remedy)
		}
	}
	if checksum.Plain != "a fingerprint saying two copies are identical" {
		t.Errorf("the finding carries plain words %q", checksum.Plain)
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
// plural, in a possessive and in the other forms the list names, joined to
// another word by a hyphen included, never inside a longer word, and a phrase
// is its own word, so a merge queue is not also a merge.
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
		"A symlink, or a symbolic link.":      {"symlink", "symbolic link"},
		"Two environment variables are set.":  {"environment variables"},
		"The record ids are kept.":            {"record ids"},
		"It is in a nutshell.":                nil,
		"The hookah shop.":                    nil,
		"They emerge, and the emerged merger": nil,
		"The committee is committed to it.":   {"committed"},
		"Let abcd auto-merge with TF==auto.":  {"merge"},
		"A pre-commit check.":                 {"commit"},
		"The work is merged and merging.":     {"merged"},
		"abcd is committing it.":              {"committing"},
		"The repository holds it.":            {"repository"},
		"Two repositories wait.":              {"repositories"},
		"The repo holds it, and the repos.":   {"repo"},
		"Open a PR, or a pull request.":       {"PR", "pull request"},
		"Two PRs wait.":                       {"PRs"},
		"The work is rebased, and a rebase.":  {"rebased"},
		"abcd is rebasing it.":                {"rebasing"},
		"The change is uncommitted.":          {"uncommitted"},
		"Three unmerged changes wait.":        {"unmerged"},
		"The Git history is kept.":            {"Git"},
		"GitHub runs the checks.":             nil,
		"abcd pushes it, and pushed it.":      {"pushes"},
		"abcd is pushing it.":                 {"pushing"},
		"The work is force-pushed.":           {"pushed"},
		// A Unicode space joins a phrase, and the echo shows it as a space.
		"Open a pull\u00a0request for it.":    {"pull request"},
		"The merge\u00a0queue is full.":       {"merge queue"},
		"The merge\u2003queue is full.":       {"merge queue"},
		"abcd sends it to origin.":            {"origin"},
		"The diffs are shown.":                {"diffs"},
		"They are different.":                 nil,
		"The hook-and-loop strap.":            {"hook"},
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
		{"the", "The checksum is the fingerprint saying two copies are identical.", false},
		{"a plain use, then a question", "The checksum is the last step before release. Who may let abcd make the checksum?", false},
		{"after a colon", "Note: the checksum is the last step before release. The checksum waits.", false},
		{"a colon opening", "Note: a checksum is a fingerprint saying two copies are identical.", false},
		{"the word again", "A checksum is a checksum of the two copies. The checksum waits.", false},
		{"the word again, plural", "A checksum is the one of the checksums we keep. The checksum waits.", false},
		{"no determiner", "Checksums are all kept for good. Each checksum stays.", false},
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
	// A form introduced does not introduce another form of the word: an
	// initialism is introduced with it, or it is a use. An inflection is the
	// same word.
	for text, want := range map[string][]string{
		"Continuous integration is the automatic checks that run on every change. Once CI passes, abcd adds the work.":      {"CI"},
		"Continuous integration (CI) is the automatic checks that run on every change. Once CI passes, abcd adds the work.": nil,
		"A symbolic link is a file that stands for another file. The symlink is kept.":                                      {"symlink"},
		"A merge is the step that adds finished work to the project. The work is merged, and each merge waits.":             nil,
	} {
		tab := wellBuiltTab()
		tab.Text = text + "\n\n" + tab.Text
		if got := valuesOf(CheckLimits(one(tab), Default, productThinker())); !slices.Equal(got, want) && len(got)+len(want) > 0 {
			t.Errorf("%q: words named %q, want %q", text, got, want)
		}
	}
	// A short form named in the sentence introducing the word, in any of the
	// shapes the floor's pages state, is introduced with it; a long form
	// named in the sentence introducing its short form is introduced too.
	for _, text := range []string{
		"Continuous integration, or CI, is the automatic checks that run on every change. Once CI passes, abcd adds the work.",
		"Continuous integration is the automatic checks that run on every change, CI for short. Once CI passes, abcd adds the work.",
		"A pull request is a request to add finished work, a PR for short. The PR waits.",
		"A pull request, or PR, is a request to add finished work. The PR waits.",
		"A request to add finished work is called a pull request (PR). The PR waits.",
		"A request to add finished work is called a pull request, or PR. The PR waits.",
		"CI (continuous integration) is the automatic checks that run on every change. Continuous integration must pass.",
		"A symbolic link (a symlink) is a file that stands for another file. The symlink is kept.",
	} {
		tab := wellBuiltTab()
		tab.Text = text + "\n\n" + tab.Text
		if fs := CheckLimits(one(tab), Default, productThinker()); len(fs) != 0 {
			t.Errorf("%q: want it introduced, got:\n%s", text, render(fs))
		}
	}
	for text, want := range map[string][]string{
		"CI is the automatic checks that run on every change. Continuous integration must pass.": {"Continuous integration"},
		"A merge is the step where work is merged into the project. The merge waits.":            {"merge"},
		"CI is the CI checks that run on every change. CI must pass.":                            {"CI"},
		"A PR is a pull request. The PR waits.":                                                  {"PR", "pull request"},
		"CI is continuous integration on every change. CI must pass.":                            {"CI", "continuous integration"},
		"A symlink is a symbolic link to the file. The symlink is kept.":                         {"symlink", "symbolic link"},
		"A push notification is a message the app sends to the phone. Push the change first.":    {"Push"},
		"The origin is the shared copy of the project. The origin waits.":                        {"origin"},
	} {
		tab := wellBuiltTab()
		tab.Text = text + "\n\n" + tab.Text
		if got := valuesOf(CheckLimits(one(tab), Default, productThinker())); !slices.Equal(got, want) {
			t.Errorf("%q: words named %q, want %q", text, got, want)
		}
	}
	// The product's own term, the word its first word, is introduced as that
	// term alone; and the bare form the remedy names is introduced.
	for _, text := range []string{
		"A push notification is a message the app sends to the phone. Each push notification is short.",
		"A push is the step that sends saved changes to the shared copy. The push waits.",
		"Origin is the shared copy of the project. The origin waits.",
	} {
		tab := wellBuiltTab()
		tab.Text = text + "\n\n" + tab.Text
		if fs := CheckLimits(one(tab), Default, productThinker()); len(fs) != 0 {
			t.Errorf("%q: want it introduced, got:\n%s", text, render(fs))
		}
	}
	// A negated form names its own plain words.
	tab = wellBuiltTab()
	tab.Text = "You have uncommitted work and unmerged changes.\n\n" + tab.Text
	var plains []string
	for _, f := range CheckLimits(one(tab), Default, productThinker()) {
		plains = append(plains, f.Plain)
	}
	if want := []string{"not yet saved", "not yet added to the project"}; !slices.Equal(plains, want) {
		t.Errorf("the negated forms name plain words %q, want %q", plains, want)
	}
	// An option's description introduces nothing: the material does.
	tab = wellBuiltTab()
	tab.Options[0].Description = "A checksum is a fingerprint saying two copies are identical."
	onlyRule(t, CheckLimits(one(tab), Default, productThinker()), RuleRegister)
}

// TestRegisterRefusalLineNamesEveryWord: the hook's lines name every word the
// register rule found on a tab with the plain words for each, so one retry
// fixes them all; the words are grouped apart from a record number, whose
// line would otherwise hide their plain words.
func TestRegisterRefusalLineNamesEveryWord(t *testing.T) {
	tab := internalsTab()
	tab.Options[0].Description = "As iss-2609202058058301 asked. It costs a second look."
	tab.Options[1].Description = "As iss-2609202058058302 asked. It costs your time."
	groups := GroupFindings(CheckLimits(one(tab), Default, productThinker()))
	if len(groups) != 2 || groups[0].Rule != RuleRegister || groups[1].Rule != RuleRegister {
		t.Fatalf("want a record-number group and a floor group, got %+v", groups)
	}
	handles, floor := groups[0].String(), groups[1].String()
	if !strings.Contains(handles, "iss-2609202058058301") || !strings.HasSuffix(handles, `(and 1 more such line: "iss-2609202058058302")`) {
		t.Errorf("the record-number line:\n%s", handles)
	}
	if strings.Contains(floor, "iss-") || !strings.Contains(floor, `"a second folder of the project's work"`) {
		t.Errorf("the floor line does not open on the first word's plain words:\n%s", floor)
	}
	want := `(and 6 more such lines: "checkout" (say "the project's folder of work"), "pull request" (say "a request to add finished work"), ` +
		`"merge queue" (say "the line finished work waits in before it is added"), "CI" (say "the automatic checks"), "commit" (say "a saved change"), ` +
		`"checksum" (say "a fingerprint saying two copies are identical"))`
	if !strings.HasSuffix(floor, want) {
		t.Errorf("the floor line does not close on every word with its plain words:\n%s\nwant suffix %s", floor, want)
	}
}

// TestTheReviewProbeRows is the review's probe (iss-2611252121330106, round
// 1), each sentence the material of one Product tab: the words it must name,
// or none. A plain use shaped like a definition introduces nothing, a
// sentence that uses the word it names introduces nothing, and a short form
// is introduced on its own.
func TestTheReviewProbeRows(t *testing.T) {
	for text, want := range map[string][]string{
		"The repository's main line of work is protected: a pull request is merged only after it was rebased and the automatic checks passed. abcd can auto-merge it once the pre-commit checks pass.": {"repository", "pull request", "merged", "rebased", "commit"},
		"Continuous integration is the automatic checks that run on every change. Once CI passes, abcd adds the work.":                                                                                 {"CI"},
		"A symbolic link is a file that stands for another file. The symlink is kept.":                                                                                                                 {"symlink"},
		"Note: the merge is the last step before release. The merge waits.":                                                                                                                            {"merge"},
		"The work is committed, merged and the PR closed. The repo keeps it.":                                                                                                                          {"committed", "merged", "PR", "repo"},
		"A merge is the step that adds finished work to the project. The merge waits.":                                                                                                                 nil,
		"A merge is the step that adds finished work to the project. A second merge is the one that failed.":                                                                                           nil,
		"Each commit is kept.\n\nA commit is a saved change to the project.":                                                                                                                           {"commit"},
		"The merge is the last step before release. Who may let abcd do the merge?":                                                                                                                    {"merge"},
		"The branch is a week old now. The branch waits.":                                                                                                                                              {"branch"},
		"The commit is the one from Tuesday. Keep the commit?":                                                                                                                                         {"commit"},
		"The hook is the one that failed. The hook runs again.":                                                                                                                                        {"hook"},
		"The checkout is the same as before. The checkout stays.":                                                                                                                                      {"checkout"},
		"The merge queue is the place it waits now. The merge queue is long.":                                                                                                                          {"merge queue"},
		"Commits are all kept for good. Each commit stays.":                                                                                                                                            {"Commits"},
		"The shell is where you typed it. The shell is open.":                                                                                                                                          {"shell"},
		"A merge is a merge of the work. The merge waits.":                                                                                                                                             {"merge"},
		"A commit is a commit that is kept. The commit stays.":                                                                                                                                         {"commit"},
		"Two repositories wait, and a PR.":                                                                                                                                                             {"repositories", "PR"},
		"A merge is the step that adds finished work; the merge waits.":                                                                                                                                nil,
		"Let abcd auto-merge the work.":                                                                                                                                                                {"merge"},
		"The pre-commit step runs.":                                                                                                                                                                    {"commit"},
	} {
		tab := wellBuiltTab()
		tab.Text = text + "\n\n" + tab.Text
		if got := valuesOf(CheckLimits(one(tab), Default, productThinker())); !slices.Equal(got, want) && len(got)+len(want) > 0 {
			t.Errorf("%q: words named %q, want %q", text, got, want)
		}
	}
}
