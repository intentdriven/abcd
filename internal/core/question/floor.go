package question

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// FloorWord is one word below the product thinker's knowledge floor: the word
// in the singular, the other ways it is written, and the plain words a
// question uses instead.
type FloorWord struct {
	Word  string
	Also  []string
	Plain string
}

// belowFloor is the words below the product thinker's knowledge floor, the
// list the floor's own page carries (KnowledgeFloorPage), in its order. A test
// holds the two equal, so the check and the floor cannot differ
// (iss-2611252121330106).
var belowFloor = []FloorWord{
	{Word: "version control", Plain: "the kept history of every change"},
	{Word: "checkout", Plain: "the project's folder of work"},
	{Word: "branch", Plain: "a separate line of work"},
	{Word: "merge", Plain: "adding finished work to the project"},
	{Word: "commit", Plain: "a saved change"},
	{Word: "worktree", Plain: "a second folder of the project's work"},
	{Word: "shell", Plain: "the window where commands are typed"},
	{Word: "file ownership", Plain: "which account a file belongs to"},
	{Word: "permission bit", Plain: "who may open or change a file"},
	{Word: "continuous integration", Also: []string{"CI"}, Plain: "the automatic checks"},
	{Word: "merge queue", Plain: "the line finished work waits in before it is added"},
	{Word: "hook", Plain: "a step that runs by itself at a set moment"},
	{Word: "environment variable", Plain: "a setting a program reads when it starts"},
	{Word: "checksum", Plain: "a fingerprint saying two copies are identical"},
	{Word: "symbolic link", Also: []string{"symlink"}, Plain: "a file that stands for another file"},
	{Word: "record id", Plain: "the record's title"},
}

// floorForm is one written form of a word below the floor, with its pattern.
// Forms are tried longest first, so a merge queue is found as itself and not
// also as a merge.
type floorForm struct {
	word int // the index in belowFloor
	form string
	re   *regexp.Regexp
}

var floorForms = compileFloorForms()

func compileFloorForms() []floorForm {
	var out []floorForm
	for i, w := range belowFloor {
		for _, f := range append([]string{w.Word}, w.Also...) {
			parts := strings.Fields(f)
			for j, p := range parts {
				parts[j] = regexp.QuoteMeta(p)
			}
			plural := "s?"
			for _, end := range []string{"ch", "sh", "s", "x"} {
				if strings.HasSuffix(f, end) {
					plural = "(?:es)?"
				}
			}
			// An initialism is read in capitals alone; "ci" is no word of it.
			fold := "(?i)"
			if f == strings.ToUpper(f) {
				fold = ""
			}
			out = append(out, floorForm{word: i, form: f, re: regexp.MustCompile(fold + strings.Join(parts, `[\s-]+`) + plural)})
		}
	}
	slices.SortStableFunc(out, func(a, b floorForm) int { return len(b.form) - len(a.form) })
	return out
}

// floorUse is one use of a word below the floor: the word's index in
// belowFloor and where the use starts and ends.
type floorUse struct{ word, start, end int }

// floorUses is every use of a word below the floor in s, in reading order. A
// use is a whole word in any case, the plural included; a word inside a longer
// one, one joined to another by a hyphen included ("auto-merge",
// "pre-commit"), is that longer word and no use.
func floorUses(s string) []floorUse {
	var out []floorUse
	taken := make([]bool, len(s)) // the bytes a longer form already holds
	for _, f := range floorForms {
		for _, m := range f.re.FindAllStringIndex(s, -1) {
			if insideWord(s, m[0], m[1]) || slices.Contains(taken[m[0]:m[1]], true) {
				continue
			}
			for i := m[0]; i < m[1]; i++ {
				taken[i] = true
			}
			out = append(out, floorUse{f.word, m[0], m[1]})
		}
	}
	slices.SortFunc(out, func(a, b floorUse) int { return a.start - b.start })
	return out
}

// insideWord reports whether s[start:end] is part of a longer word: a letter,
// a digit or an underscore touches it, or a hyphen joins it to one.
func insideWord(s string, start, end int) bool {
	r, n := utf8.DecodeLastRuneInString(s[:start])
	if r == '-' {
		r, _ = utf8.DecodeLastRuneInString(s[:start-n])
	}
	if wordRune(r) {
		return true
	}
	r, n = utf8.DecodeRuneInString(s[end:])
	if r == '-' {
		r, _ = utf8.DecodeRuneInString(s[end+n:])
	}
	return wordRune(r)
}

func wordRune(r rune) bool {
	return r != utf8.RuneError && (r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r))
}

// The one sentence that introduces a word (the floor's own rule: "a checksum
// is a fingerprint saying two copies are identical"), read by its shape: the
// word opens a sentence, which goes on to say what it is or means; or a
// sentence saying what it is ends on what it is called.
var (
	introOpensRe = regexp.MustCompile(`(?i)(?:^|[.?!:]["')\]]*\s+|\n[ \t]*\n\s*|\n[ \t]*(?:[-*+]|[0-9]+\.)[ \t]+)(?:(?:an?|the)\s+)?["'\x{201c}\x{2018}]?$`)
	introSaysRe  = regexp.MustCompile(`(?i)^["'\x{201d}\x{2019}]?(?:\s*\([^()]*\))?\s+(?:is|are|means?)\s+((?:an?|the|one|two|what|where|when|who|how|that|any|every|each|all)\b[^.?!]*?)(?:[.!]|\n[ \t]*\n|$)`)
	introNamesRe = regexp.MustCompile(`(?i)([^.?!]*)\s(?:is|are)\s+(?:called|known\s+as)\s+(?:(?:an?|the)\s+)?["'\x{201c}\x{2018}]?$`)
	introEndsRe  = regexp.MustCompile(`^["'\x{201d}\x{2019}]?\s*[.!]`)
)

// introWords is the fewest words that say what a word is: fewer ("is the
// same") use the word and introduce nothing.
const introWords = 3

// introWindow bounds how much text on each side of a use is read for its
// sentence, so a payload of any size costs the same; no sentence that
// introduces a word is longer.
const introWindow = 1 << 10

// introducedAt reports whether the use of a word at s[start:end] is in the
// sentence that introduces it.
func introducedAt(s string, start, end int) bool {
	before, after := s[:start], s[end:]
	if len(before) > introWindow {
		before = "x" + before[len(before)-introWindow:] // mid-sentence, so never an opening
	}
	if len(after) > introWindow {
		after = after[:introWindow] + "?" // cut, so never an end
	}
	if introOpensRe.MatchString(before) {
		if m := introSaysRe.FindStringSubmatch(after); m != nil && len(strings.Fields(m[1])) >= introWords {
			return true
		}
	}
	m := introNamesRe.FindStringSubmatch(before)
	return m != nil && len(strings.Fields(m[1])) >= introWords && introEndsRe.MatchString(after)
}

// floorFindings is a tab's words below the floor, one finding per word at its
// first use in reading order. A word whose first use in the question text is
// the sentence introducing it passes everywhere on the tab.
func floorFindings(n int, t Tab) []Finding {
	introduced := map[int]bool{}
	for _, u := range floorUses(t.Text) {
		if _, met := introduced[u.word]; !met {
			introduced[u.word] = introducedAt(t.Text, u.start, u.end)
		}
	}
	var out []Finding
	named := map[int]bool{}
	for _, fl := range fields(t) {
		for _, u := range floorUses(fl.text) {
			if introduced[u.word] || named[u.word] {
				continue
			}
			w := belowFloor[u.word]
			named[u.word] = true
			out = append(out, Finding{Tab: n, Part: fl.part, Rule: RuleRegister, Value: echo(fl.text[u.start:u.end]),
				Limit: "no word below the product thinker's knowledge floor that the question does not introduce",
				Remedy: fmt.Sprintf("Say %q instead, wherever the tab uses the word, or introduce it in one sentence of the question text before its first use, as in \"A checksum is a fingerprint saying two copies are identical.\"; the floor is stated at %s under abcd's plugin root.",
					w.Plain, KnowledgeFloorPage)})
		}
	}
	return out
}
