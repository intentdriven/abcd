package ahoy

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/question"
)

// The question pair at a terminal and through the plugin page
// (spc-2610031704338417 step 4, for itd-2610031348087517).

// helpedAsk is one question a helpedPrompter was asked: its key, the help it
// came with (zero for a question asked through Prompt), its choices and its
// default.
type helpedAsk struct {
	key     string
	help    PromptHelp
	helped  bool
	choices []string
	def     string
}

// helpedPrompter is a person at a terminal: it answers each approval with
// confirm, each value question from answers (its default when absent), and
// keeps every question in the order asked, approvals by their text.
type helpedPrompter struct {
	confirm func(q string) bool
	answers map[string]string
	asked   []helpedAsk
}

func (p *helpedPrompter) AtTerminal() bool { return true }

func (p *helpedPrompter) Confirm(q string) bool {
	p.asked = append(p.asked, helpedAsk{key: q})
	return p.confirm != nil && p.confirm(q)
}

func (p *helpedPrompter) Prompt(key string, choices []string, def string) string {
	p.asked = append(p.asked, helpedAsk{key: key, choices: choices, def: def})
	return p.answer(key, def)
}

func (p *helpedPrompter) PromptHelped(h PromptHelp, choices []string, def string) string {
	p.asked = append(p.asked, helpedAsk{key: h.Key, help: h, helped: true, choices: choices, def: def})
	return p.answer(h.Key, def)
}

func (p *helpedPrompter) answer(key, def string) string {
	if v, ok := p.answers[key]; ok {
		return v
	}
	return def
}

// keys is every question asked, in order.
func (p *helpedPrompter) keys() []string {
	out := make([]string, len(p.asked))
	for i, a := range p.asked {
		out[i] = a.key
	}
	return out
}

// readmeKeys is the README questions asked, in order.
func (p *helpedPrompter) readmeKeys() []string {
	var out []string
	for _, a := range p.asked {
		if a.key == ReadmePromptKey || strings.HasPrefix(a.key, ReadmePromptKey+".") {
			out = append(out, a.key)
		}
	}
	return out
}

// find is the question asked by key.
func (p *helpedPrompter) find(t *testing.T, key string) helpedAsk {
	t.Helper()
	for _, a := range p.asked {
		if a.key == key {
			return a
		}
	}
	t.Fatalf("%s was not asked; asked %q", key, p.keys())
	return helpedAsk{}
}

// approveCategories approves every kind of change and declines every offer.
func approveCategories(q string) bool { return askedCategory(q) != "" }

// terminalInstall runs an interactive install at a terminal through p.
func terminalInstall(t *testing.T, repo string, p *helpedPrompter) InstallResult {
	t.Helper()
	res, err := Install(repo, InstallOptions{}, p)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// TestReadmeQuestionPairOrder: at a terminal the pair is asked after the
// configuration values (the conventions-file question among them) and the
// house-style question, and before the status-line offer and the artefact
// kind: Q1 first, through the helped prompter, then one question per offered
// part in the fixed order, each with its current setting and an example. A
// project not declared public is not asked about the release and build
// badges; one declared public is asked about all six. The answers are stored
// as the chosen set and written.
func TestReadmeQuestionPairOrder(t *testing.T) {
	for _, tc := range []struct {
		vis   string
		parts []string
	}{
		{"private", []string{"abcd-badge", "licence", "links", "contents"}},
		{"public", []string{"release-badge", "build-badge", "abcd-badge", "licence", "links", "contents"}},
	} {
		t.Run(tc.vis, func(t *testing.T) {
			setupHermetic(t)
			repo := t.TempDir()
			if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
				t.Fatal(err)
			}
			writeFixture(t, repo, "LICENSE", mitLicence, 0o644)
			writeFixture(t, repo, "CHANGELOG.md", "# Changelog\n", 0o644)
			writeFixture(t, repo, "README.md", "# Project\n\nprose\n", 0o644)
			adopt := true
			p := &helpedPrompter{confirm: approveCategories, answers: map[string]string{
				"visibility": tc.vis, "docs_target": "skip", "scan_deep": "false",
				ReadmePromptKey: "yes", "readme.licence": "on", "readme.links": "on",
				"readme.contents": "off", "readme.abcd-badge": "off",
				"readme.release-badge": "off", "readme.build-badge": "off",
			}}
			if _, err := Install(repo, InstallOptions{Adopt: &adopt}, p); err != nil {
				t.Fatal(err)
			}
			want := append([]string{ReadmePromptKey}, prefixed("readme.", tc.parts)...)
			if got := p.readmeKeys(); !reflect.DeepEqual(got, want) {
				t.Fatalf("README questions asked %q, want %q", got, want)
			}
			keys := p.keys()
			at := slices.Index(keys, ReadmePromptKey)
			last := at + len(want) - 1
			if !reflect.DeepEqual(keys[at:last+1], want) {
				t.Errorf("the pair was not asked as one run: %q", keys)
			}
			if i := slices.Index(keys, "docs_target"); i < 0 || i > at {
				t.Errorf("the conventions-file question (at %d) is not asked before the README question (at %d): %q", i, at, keys)
			}
			// The house-style question is asked where setup seeds the docs
			// lint; where it is asked, it comes first.
			if i := slices.Index(keys, emDashPromptKey); i > at {
				t.Errorf("the house-style question (at %d) is asked after the README question (at %d)", i, at)
			}
			if i := slices.Index(keys, artefactKindKey); i >= 0 && i < last {
				t.Errorf("the artefact kind (at %d) is asked before the README pair ends (at %d)", i, last)
			}
			for i, k := range keys {
				if isStatusLineOfferQuestion(k) && i < last {
					t.Errorf("the status-line offer (at %d) is asked before the README pair ends (at %d)", i, last)
				}
			}
			for _, k := range want {
				a := p.find(t, k)
				if !a.helped {
					t.Errorf("%s was asked through Prompt, not with its help for this project", k)
				}
				if a.help.Now == "" || a.help.ChangeLater == "" {
					t.Errorf("%s carries no current setting or change-later line: %+v", k, a.help)
				}
				if k != ReadmePromptKey && len(a.help.Example) == 0 && a.help.ExampleNone == "" {
					t.Errorf("%s carries no example and no reason it writes none", k)
				}
				if !slices.Contains(a.choices, SetupLaterValue) {
					t.Errorf("%s offers no decide-later answer: %q", k, a.choices)
				}
			}
			if ex := p.find(t, "readme.licence").help.Example; len(ex) != 1 || !strings.Contains(ex[0], "the MIT License") {
				t.Errorf("the licence example is %q, want the line naming MIT", ex)
			}
			if parts, ok := storedReadmeParts(t, repo); !ok || !reflect.DeepEqual(parts, []string{"licence", "links"}) {
				t.Errorf("readme.parts = %v (stored %v), want [licence links]", parts, ok)
			}
			if got := readReadme(t, filepath.Join(repo, "README.md")); !strings.Contains(got, "ABCD:licence") || !strings.Contains(got, "ABCD:links") {
				t.Errorf("the chosen parts were not written:\n%s", got)
			}
		})
	}
}

func prefixed(prefix string, names []string) []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = prefix + n
	}
	return out
}

// TestReadmeNoIsStored: a no to Q1 stores an empty set, asks no part, writes
// nothing into README.md, and the next setup run asks neither question and
// reports nothing skipped about the README.
func TestReadmeNoIsStored(t *testing.T) {
	setupHermetic(t)
	repo := installedRepo(t)
	writeFixture(t, repo, "LICENSE", mitLicence, 0o644)
	writeFixture(t, repo, "README.md", "# Project\n", 0o644)
	p := &helpedPrompter{confirm: approveCategories, answers: map[string]string{ReadmePromptKey: "no"}}
	res := terminalInstall(t, repo, p)
	if got := p.readmeKeys(); !reflect.DeepEqual(got, []string{ReadmePromptKey}) {
		t.Fatalf("README questions asked %q, want Q1 alone", got)
	}
	if parts, ok := storedReadmeParts(t, repo); !ok || len(parts) != 0 {
		t.Fatalf("a no stored %v (stored %v), want []", parts, ok)
	}
	if !containsString(res.Changes, "readme.parts: not set -> none") {
		t.Errorf("changes = %q", res.Changes)
	}
	if got := readReadme(t, filepath.Join(repo, "README.md")); got != "# Project\n" {
		t.Errorf("README.md changed: %q", got)
	}

	again := &helpedPrompter{confirm: approveCategories}
	res = terminalInstall(t, repo, again)
	if got := again.readmeKeys(); len(got) != 0 {
		t.Errorf("a stored no was asked again: %q", got)
	}
	if containsString(res.OptionalSkipped, ReadmeOfferGapID) {
		t.Errorf("a stored no is still reported skipped: %v", res.OptionalSkipped)
	}
}

// TestReadmeLaterStoresNothing: deciding later, at Q1 or at any part, stores
// nothing and writes nothing, the parts after a part decided later are not
// asked, and the next setup run asks the pair again. Ending the input takes
// Q1's default, which is to decide later.
func TestReadmeLaterStoresNothing(t *testing.T) {
	for name, answers := range map[string]map[string]string{
		"Q1 later":            {ReadmePromptKey: SetupLaterValue},
		"Q1 unanswered":       {},
		"a part later":        {ReadmePromptKey: "yes", "readme.licence": SetupLaterValue},
		"a part unanswered":   {ReadmePromptKey: "yes", "readme.licence": ""},
		"a part another word": {ReadmePromptKey: "yes", "readme.licence": "maybe"},
	} {
		t.Run(name, func(t *testing.T) {
			setupHermetic(t)
			repo := installedRepo(t)
			writeFixture(t, repo, "LICENSE", mitLicence, 0o644)
			writeFixture(t, repo, "CHANGELOG.md", "# Changelog\n", 0o644)
			writeFixture(t, repo, "README.md", "# Project\n", 0o644)
			p := &helpedPrompter{confirm: approveCategories, answers: answers}
			res := terminalInstall(t, repo, p)
			if _, ok := storedReadmeParts(t, repo); ok {
				t.Errorf("deciding later stored a README answer")
			}
			if got := readReadme(t, filepath.Join(repo, "README.md")); got != "# Project\n" {
				t.Errorf("README.md changed: %q", got)
			}
			if _, ok := answers["readme.licence"]; ok && slices.Contains(p.readmeKeys(), "readme.links") {
				t.Errorf("a part after the one decided later was asked: %q", p.readmeKeys())
			}
			if containsString(res.OptionalSkipped, ReadmeOfferGapID) {
				t.Errorf("an asked question is reported as not asked: %v", res.OptionalSkipped)
			}
			again := &helpedPrompter{confirm: approveCategories, answers: map[string]string{ReadmePromptKey: SetupLaterValue}}
			terminalInstall(t, repo, again)
			if got := again.readmeKeys(); !reflect.DeepEqual(got, []string{ReadmePromptKey}) {
				t.Errorf("the next run asked %q, want Q1 again", got)
			}
		})
	}
}

// TestAbcdBadgeOffByDefault: the managed-with-abcd badge is offered in every
// project, off: its question's default answer is off, its current setting
// reads off, and a person who takes every default stores the licence and the
// links (on by default where they can be written) without it. Every part's
// answers come in one order, on, off, then deciding later, whatever its
// default: an answer put first would read as recommended, and the options
// widen rather than recommend.
func TestAbcdBadgeOffByDefault(t *testing.T) {
	setupHermetic(t)
	repo := installedRepo(t)
	writeFixture(t, repo, "LICENSE", mitLicence, 0o644)
	writeFixture(t, repo, "CHANGELOG.md", "# Changelog\n", 0o644)
	writeFixture(t, repo, "README.md", "# Project\n", 0o644)
	p := &helpedPrompter{confirm: approveCategories, answers: map[string]string{ReadmePromptKey: "yes"}}
	terminalInstall(t, repo, p)
	badge := p.find(t, "readme.abcd-badge")
	if badge.def != "off" {
		t.Errorf("the badge's default is %q, want off", badge.def)
	}
	for _, k := range p.readmeKeys()[1:] {
		if got := p.find(t, k).choices; !reflect.DeepEqual(got, []string{"on", "off", SetupLaterValue}) {
			t.Errorf("%s offers %q, want on, off, then deciding later", k, got)
		}
	}
	v := buildReadmeView(repo, "public")
	for _, a := range v.Asks[1:] {
		for _, q := range a.Questions {
			if got := choiceLabels(q); !reflect.DeepEqual(got, []string{"on", "off", question.Default.LaterLabels[0]}) {
				t.Errorf("the page's %s offers %q, want on, off, then deciding later", q.ID, got)
			}
		}
	}
	if !strings.Contains(badge.help.Now, "off") {
		t.Errorf("the badge's current setting reads %q, want it to show off", badge.help.Now)
	}
	if want := []string{"on", "off", "on", "off"}; !reflect.DeepEqual([]string{
		p.find(t, "readme.licence").def, p.find(t, "readme.abcd-badge").def,
		p.find(t, "readme.links").def, p.find(t, "readme.contents").def,
	}, want) {
		t.Errorf("the defaults of licence, abcd-badge, links and contents are not %q", want)
	}
	if parts, _ := storedReadmeParts(t, repo); !reflect.DeepEqual(parts, []string{"licence", "links"}) {
		t.Errorf("taking every default stored %v, want [licence links]", parts)
	}
	view, ok := readmeView(repo)
	if ok {
		t.Errorf("an answered pair still has a view to relay: %+v", view)
	}
}

// TestReadmeOfferView: while the pair is unanswered, the detection the plugin
// page reads carries the README view: per part its name, whether it is
// offered, its current setting, its default, and its example or the reason
// it writes none, every line terminal-safe; the declaration line when the
// project is not declared public; and the questions themselves, built from
// the same help the terminal asks with, Q1 alone and the offered parts in
// calls of at most four. Once the pair is answered the view is gone.
func TestReadmeOfferView(t *testing.T) {
	readme := "# Project\n\n## Install \x1b[31mnow\n"
	t.Run("not declared public", func(t *testing.T) {
		repo := readmeRepo(t, "https://github.com/owner/repo.git", map[string]string{
			"LICENSE": mitLicence, "README.md": readme, ".abcd/config.json": `{"repo": {"visibility": "private"}}`,
		})
		v, ok := readmeView(repo)
		if !ok {
			t.Fatal("no view while the pair is unanswered")
		}
		if !strings.Contains(v.Declaration, "not declared public") {
			t.Errorf("declaration = %q, want it to name the declaration", v.Declaration)
		}
		names := make([]string, len(v.Parts))
		for i, pv := range v.Parts {
			names[i] = pv.Name
		}
		if !reflect.DeepEqual(names, readmePartOrder) {
			t.Fatalf("parts %q, want every part in the fixed order", names)
		}
		byName := map[string]ReadmePartView{}
		for _, pv := range v.Parts {
			byName[pv.Name] = pv
			if pv.Now == "" {
				t.Errorf("%s has no current setting", pv.Name)
			}
		}
		for _, n := range []string{"release-badge", "build-badge"} {
			if byName[n].Offered || !strings.Contains(byName[n].Reason, "not declared public") {
				t.Errorf("%s: %+v, want not offered, naming the declaration", n, byName[n])
			}
		}
		if b := byName["abcd-badge"]; !b.Offered || b.Default != "off" || len(b.Example) != 1 || !strings.Contains(b.Example[0], "Managed with abcd") {
			t.Errorf("abcd-badge: %+v", b)
		}
		if l := byName["licence"]; !l.Offered || l.Default != "on" || len(l.Example) != 1 || !strings.Contains(l.Example[0], "the MIT License") {
			t.Errorf("licence: %+v", l)
		}
		if l := byName["links"]; !l.Offered || len(l.Example) != 0 || !strings.Contains(l.Reason, "CHANGELOG.md") {
			t.Errorf("links: %+v, want offered with the reason it writes nothing", l)
		}
		c := byName["contents"]
		if len(c.Example) != 1 || strings.ContainsRune(c.Example[0], 0x1b) || !strings.Contains(c.Example[0], "Install") {
			t.Errorf("contents example %q, want the heading listed with its escape byte made safe", c.Example)
		}
		if got := askShape(v); !reflect.DeepEqual(got, [][]string{{"readme"}, {"readme.abcd-badge", "readme.licence", "readme.links", "readme.contents"}}) {
			t.Errorf("asks %q, want Q1 alone, then the four offered parts", got)
		}
		q1 := v.Asks[0].Questions[0]
		if !strings.Contains(materialText(q1), "not declared public") {
			t.Errorf("Q1 does not say it goes by the declaration: %q", materialText(q1))
		}
		tab := v.Asks[1].Questions[1]
		if tab.Now == "" || tab.ChangeLater == "" || !strings.Contains(materialText(tab), "the MIT License") {
			t.Errorf("the licence tab lacks its current setting, change-later line or example: %+v", tab)
		}
		if h, ok := HelpIn(repo, "readme.licence"); !ok || !strings.HasPrefix(materialText(tab), strings.TrimSuffix(firstSentence(h.About), ".")) {
			t.Errorf("the licence tab is not built from the core help %q: %q", h.About, materialText(tab))
		}
	})
	t.Run("declared public", func(t *testing.T) {
		repo := readmeRepo(t, "https://github.com/owner/repo.git", map[string]string{
			"LICENSE": mitLicence, "README.md": readme, ".github/workflows/ci.yml": "on: push\n",
			".abcd/config.json": `{"repo": {"visibility": "public"}}`,
		})
		v, ok := readmeView(repo)
		if !ok {
			t.Fatal("no view while the pair is unanswered")
		}
		if v.Declaration != "" {
			t.Errorf("a public project carries the declaration line %q", v.Declaration)
		}
		if got := askShape(v); !reflect.DeepEqual(got, [][]string{
			{"readme"},
			{"readme.release-badge", "readme.build-badge", "readme.abcd-badge", "readme.licence"},
			{"readme.links", "readme.contents"},
		}) {
			t.Errorf("asks %q, want Q1 alone, then the six parts as four and two", got)
		}
		if rb := v.Parts[0]; !rb.Offered || len(rb.Example) != 1 || !strings.Contains(rb.Example[0], "github.com/owner/repo/releases/latest") {
			t.Errorf("release-badge: %+v", rb)
		}
	})
	t.Run("answered", func(t *testing.T) {
		repo := readmeRepo(t, "", map[string]string{".abcd/config.json": `{"readme": {"parts": []}}`})
		if v, ok := readmeView(repo); ok {
			t.Errorf("an answered pair has a view: %+v", v)
		}
	})
	t.Run("on the detection the page reads", func(t *testing.T) {
		setupHermetic(t)
		repo := installedRepo(t)
		det, err := DryRun(repo)
		if err != nil {
			t.Fatal(err)
		}
		if det.Readme == nil || len(det.Readme.Asks) == 0 {
			t.Fatalf("the detection carries no README view: %+v", det.Readme)
		}
		setReadmeConfig(t, repo, map[string]any{"parts": []string{}})
		if det, _ = DryRun(repo); det.Readme != nil {
			t.Errorf("an answered pair is still on the detection: %+v", det.Readme)
		}
	})
}

// askShape is the view's asks as their question ids.
func askShape(v ReadmeView) [][]string {
	var out [][]string
	for _, a := range v.Asks {
		var ids []string
		for _, q := range a.Questions {
			ids = append(ids, q.ID)
		}
		out = append(out, ids)
	}
	return out
}

// firstSentence is s up to its first ". ".
func firstSentence(s string) string {
	if i := strings.Index(s, ". "); i >= 0 {
		return s[:i+1]
	}
	return s
}

// materialText is a question's material as one text, a list's items one a
// line.
func materialText(q question.Question) string {
	var parts []string
	for _, b := range q.Material {
		parts = append(parts, b.Text)
		parts = append(parts, b.Items...)
	}
	return strings.Join(parts, "\n")
}

// TestReadmeViewIsLinearInTheHeadings: the table of contents lists one line a
// heading, and a README of thousands of headings is read whole (up to the
// 1 MiB cap), so cutting its example to the rows limit must not try every
// length from the longest down: no more lines than the limit has rows can fit.
// Building the view of a README of 6000 headings measures a bounded number of
// candidate questions, as many as for a short README, where trying every
// length measured one for each line and took 17 s. The count, not the time,
// is asserted: a race-enabled run on a loaded machine is slow either way.
func TestReadmeViewIsLinearInTheHeadings(t *testing.T) {
	measured := func(headings int) int {
		var readme strings.Builder
		readme.WriteString("# Project\n")
		for i := range headings {
			fmt.Fprintf(&readme, "\n## Heading number %d\n", i)
		}
		repo := readmeRepo(t, "", map[string]string{"README.md": readme.String(), ".abcd/config.json": `{"repo": {"visibility": "private"}}`})
		calls := 0
		swap := measureFits
		t.Cleanup(func() { measureFits = swap })
		measureFits = func(h PromptHelp, def string) bool {
			calls++
			return swap(h, def)
		}
		start := time.Now()
		v, ok := readmeView(repo)
		measureFits = swap
		if !ok {
			t.Fatal("no view while the pair is unanswered")
		}
		for _, pv := range v.Parts {
			if pv.Name == "contents" && len(pv.Example) != headings {
				t.Errorf("the view's contents example holds %d lines, want all %d", len(pv.Example), headings)
			}
		}
		t.Logf("%d headings: %d questions measured in %s", headings, calls, time.Since(start))
		return calls
	}
	short, long := measured(60), measured(6000)
	if bound := len(readmePartOrder) * (question.Default.Rows + 2); long > bound || long != short {
		t.Errorf("6000 headings measured %d candidate questions and 60 measured %d, want the same, at most %d", long, short, bound)
	}
}

// TestFitLinesChecksABoundedNumberOfCandidates: however many lines an example
// has, the cut asks whether at most the limit's rows plus one candidates fit,
// so its cost does not grow with the lines.
func TestFitLinesChecksABoundedNumberOfCandidates(t *testing.T) {
	limit := question.Default.Rows + 1
	for _, n := range []int{3, 30, 3000, 30000} {
		lines := make([]string, n)
		for i := range lines {
			lines[i] = fmt.Sprintf("- [Heading %d](#heading-%d)", i, i)
		}
		calls := 0
		got, left := fitLines(lines, func(ex []string, left int) bool {
			calls++
			return len(ex) <= 5
		})
		if calls > limit {
			t.Errorf("%d lines: fits was called %d times, want at most %d", n, calls, limit)
		}
		if want := min(n, 5); len(got) != want || left != n-want {
			t.Errorf("%d lines: cut to %d lines leaving %d, want %d leaving %d", n, len(got), left, want, n-want)
		}
	}
}

// TestReadmeReasonIsFittedToTheRows: a part that writes nothing says why in
// its question as fully as the rows limit allows. Eight long-named workflows
// and no ci.yml make the build badge's whole reason too tall, so its tab says
// how many workflows there are, and the view's reason still names them; a
// reason with no short form that does not fit is not shown.
func TestReadmeReasonIsFittedToTheRows(t *testing.T) {
	files := map[string]string{"README.md": "# Project\n", ".abcd/config.json": `{"repo": {"visibility": "public"}}`}
	for i := range 8 {
		files[fmt.Sprintf(".github/workflows/a-workflow-with-a-long-descriptive-name-number-%d.yml", i)] = "on: push\n"
	}
	repo := readmeRepo(t, "https://github.com/owner/repo.git", files)
	v, ok := readmeView(repo)
	if !ok {
		t.Fatal("no view while the pair is unanswered")
	}
	var bb ReadmePartView
	for _, pv := range v.Parts {
		if pv.Name == "build-badge" {
			bb = pv
		}
	}
	if !strings.Contains(bb.Reason, "number-7.yml") {
		t.Errorf("the view's reason %q does not name the workflows", bb.Reason)
	}
	if got := bb.help().ExampleNone; !strings.HasPrefix(got, "8 workflows in .github/workflows/") {
		t.Errorf("the build badge's tab says %q, want the short form counting the workflows", got)
	}
	long := strings.Repeat("a reason that goes on ", 200)
	if got := fitReason(bb.help(), "on", long, ""); got != readmeReasonNotShown {
		t.Errorf("a reason too long to fit is said as %q, want %q", got, readmeReasonNotShown)
	}
	if got := fitReason(bb.help(), "on", ""); got != "" {
		t.Errorf("no reason is said as %q", got)
	}
}

// TestReadmeExampleQuotesTheExactLines: a part's example is quoted as the
// exact lines it would write, as the host's question tool shows the tab's
// text: a links line is "- [Changelog](CHANGELOG.md)", never with a second
// bullet before it, a nested contents line keeps its indent, and an example
// cut to the rows limit says in its lead how many lines it writes, rather
// than in a line that reads as one of them.
func TestReadmeExampleQuotesTheExactLines(t *testing.T) {
	var readme strings.Builder
	readme.WriteString("# Project\n")
	for i := range 30 {
		fmt.Fprintf(&readme, "\n## Section %d\n\n### Sub %d\n", i, i)
	}
	repo := readmeRepo(t, "", map[string]string{
		"CHANGELOG.md": "x\n", "CONTRIBUTING.md": "x\n", "README.md": readme.String(),
		".abcd/config.json": `{"repo": {"visibility": "private"}}`,
	})
	v, ok := readmeView(repo)
	if !ok {
		t.Fatal("no view while the pair is unanswered")
	}
	tabs := map[string]question.Tab{}
	for _, a := range v.Asks {
		f := a.Fields()
		for i, q := range a.Questions {
			tabs[q.ID] = f.Tabs[i]
		}
	}
	for _, pv := range v.Parts {
		tab, asked := tabs[readmePartKeyPrefix+pv.Name]
		if !asked || len(pv.Example) == 0 {
			continue
		}
		lead, quoted, ok := quotedExample(tab.Text)
		if !ok {
			t.Errorf("%s: no example quoted after its lead in:\n%s", pv.Name, tab.Text)
			continue
		}
		if len(quoted) > len(pv.Example) || !reflect.DeepEqual(quoted, pv.Example[:len(quoted)]) {
			t.Errorf("%s: the tab quotes\n%s\nwant the first lines of\n%s", pv.Name, strings.Join(quoted, "\n"), strings.Join(pv.Example, "\n"))
		}
		if len(quoted) < len(pv.Example) && !strings.Contains(lead, fmt.Sprintf("%d lines", len(pv.Example))) {
			t.Errorf("%s: an example cut to %d of %d lines does not say so in its lead %q", pv.Name, len(quoted), len(pv.Example), lead)
		}
	}
	if !strings.Contains(tabs["readme.links"].Text, "\n- [Changelog](CHANGELOG.md)\n") || strings.Contains(tabs["readme.links"].Text, "- - ") {
		t.Errorf("the links tab does not quote its lines exactly:\n%s", tabs["readme.links"].Text)
	}
	if !strings.Contains(tabs["readme.contents"].Text, "\n  - [Sub 0](#sub-0)\n") {
		t.Errorf("the contents tab does not keep a nested line's indent:\n%s", tabs["readme.contents"].Text)
	}
}

// quotedExample splits a tab's text into the paragraph that leads its example
// and the example's lines, the block after it.
func quotedExample(text string) (lead string, lines []string, ok bool) {
	blocks := strings.Split(text, "\n\n")
	for i, b := range blocks {
		if strings.Contains(b, "Here it writes") && strings.HasSuffix(b, ":") && i+1 < len(blocks) {
			return b, strings.Split(blocks[i+1], "\n"), true
		}
	}
	return "", nil, false
}

// choiceLabels is a question's options' labels, deciding later last.
func choiceLabels(q question.Question) []string {
	var out []string
	for _, o := range append(append([]question.Option(nil), q.Options...), q.Later) {
		out = append(out, o.Label)
	}
	return out
}

// TestReadmeCutsStopBeforeAStyleFinding: a quoted example or reason is held to
// every limit the host's question hook checks, not the rows alone
// (iss-2610120129098206). A table of contents whose second line has an em
// dash in a list item and whose third a bold heading shows the line before
// them; one whose first line trips a rule shows none and says how many it
// writes; a reason that trips a rule is not shown. Every tab then passes the
// limits, so the page can ask it.
func TestReadmeCutsStopBeforeAStyleFinding(t *testing.T) {
	contents := func(readme string) (ReadmePartView, PromptHelp) {
		t.Helper()
		repo := readmeRepo(t, "", map[string]string{"README.md": readme, ".abcd/config.json": `{"repo": {"visibility": "private"}}`})
		v, ok := readmeView(repo)
		if !ok {
			t.Fatal("no view while the pair is unanswered")
		}
		for _, a := range v.Asks {
			for _, f := range question.CheckLimits(a.Fields(), question.Default, question.Addressee{}) {
				t.Errorf("%s: %s", askLabel(a, f.Tab), f)
			}
		}
		for _, pv := range v.Parts {
			if pv.Name == "contents" {
				return pv, v.helps["contents"]
			}
		}
		t.Fatal("no contents part")
		return ReadmePartView{}, PromptHelp{}
	}
	pv, h := contents("# Project\n\n## Usage\n\n## Install — fast\n\n## The **core** of __init__\n")
	if len(pv.Example) != 3 {
		t.Errorf("the view's example holds %d lines, want all 3", len(pv.Example))
	}
	if want := []string{"- [Usage](#usage)"}; strings.Join(h.Example, "\n") != strings.Join(want, "\n") || h.ExampleLeft != 2 {
		t.Errorf("the tab quotes %q leaving %d, want %q leaving 2", h.Example, h.ExampleLeft, want)
	}
	if got, want := h.ExampleIntro(), "Here it writes 3 lines, the first 1 of them:"; got != want {
		t.Errorf("lead %q, want %q", got, want)
	}
	_, h = contents("# Project\n\n## Install — fast\n\n## Usage\n")
	if len(h.Example) != 0 || h.ExampleLeft != 2 {
		t.Errorf("the tab quotes %q leaving %d, want none leaving 2", h.Example, h.ExampleLeft)
	}
	if got, want := h.ExampleIntro(), "Here it writes 2 lines, not shown here."; got != want {
		t.Errorf("lead %q, want %q", got, want)
	}
	h.Example, h.ExampleLeft = nil, 1
	if got, want := h.ExampleIntro(), "Here it writes 1 line, not shown here."; got != want {
		t.Errorf("lead %q, want %q", got, want)
	}
	h.Example, h.ExampleLeft = nil, 0
	if got, want := fitReason(h, "off", "no **LICENSE** file", ""), "its reason is not shown here"; got != want {
		t.Errorf("a reason tripping a style rule is said as %q, want %q", got, want)
	}
	if got, want := fitReason(h, "off", "no **LICENSE** file", "no licence"), "no licence"; got != want {
		t.Errorf("a reason tripping a style rule falls back to %q, want its short form %q", got, want)
	}
}
