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
// in the singular, its other written forms (an inflection or a clipping, the
// same word to a reader who knows it), its short forms (an initialism or a
// coinage, which a reader does not link to the word unless told), and the
// plain words a question uses instead.
type FloorWord struct {
	Word  string
	Also  []string
	Short []string
	Plain string
	// Not is the word's negated forms ("uncommitted"), still the word to a
	// reader who knows it, and NotPlain the plain words they say instead.
	Not      []string
	NotPlain string
}

// belowFloor is the words below the product thinker's knowledge floor, the
// list the floor's own page carries (KnowledgeFloorPage), in its order. A test
// holds the two equal, so the check and the floor cannot differ
// (iss-2611252121330106).
var belowFloor = []FloorWord{
	{Word: "version control", Plain: "the kept history of every change"},
	{Word: "checkout", Plain: "the project's folder of work"},
	{Word: "branch", Plain: "a separate line of work"},
	{Word: "merge", Also: []string{"merged", "merging"}, Not: []string{"unmerged"}, NotPlain: "not yet added to the project", Plain: "adding finished work to the project"},
	{Word: "commit", Also: []string{"committed", "committing"}, Not: []string{"uncommitted"}, NotPlain: "not yet saved", Plain: "a saved change"},
	{Word: "worktree", Plain: "a second folder of the project's work"},
	{Word: "repository", Also: []string{"repo"}, Plain: "the project's files and their history"},
	{Word: "pull request", Short: []string{"PR"}, Plain: "a request to add finished work"},
	{Word: "rebase", Also: []string{"rebased", "rebasing"}, Plain: "moving work onto the newest version"},
	{Word: "git", Plain: "the program that keeps the project's history"},
	{Word: "push", Also: []string{"pushed", "pushing"}, Plain: "sending saved changes to the shared copy"},
	{Word: "origin", Plain: "the shared copy of the project"},
	{Word: "diff", Plain: "the lines a change alters"},
	{Word: "shell", Plain: "the window where commands are typed"},
	{Word: "file ownership", Plain: "which account a file belongs to"},
	{Word: "permission bit", Plain: "who may open or change a file"},
	{Word: "continuous integration", Short: []string{"CI"}, Plain: "the automatic checks"},
	{Word: "merge queue", Plain: "the line finished work waits in before it is added"},
	{Word: "hook", Plain: "a step that runs by itself at a set moment"},
	{Word: "environment variable", Plain: "a setting a program reads when it starts"},
	{Word: "checksum", Plain: "a fingerprint saying two copies are identical"},
	{Word: "symbolic link", Short: []string{"symlink"}, Plain: "a file that stands for another file"},
	{Word: "record id", Plain: "the record's title"},
}

// BelowFloor returns the words below the product thinker's knowledge floor, a
// fresh copy each call.
func BelowFloor() []FloorWord {
	out := slices.Clone(belowFloor)
	for i := range out {
		out[i].Also = slices.Clone(out[i].Also)
		out[i].Short = slices.Clone(out[i].Short)
		out[i].Not = slices.Clone(out[i].Not)
	}
	return out
}

// floorForm is one written form of a word below the floor, with its pattern.
// Forms are tried longest first, so a merge queue is found as itself and not
// also as a merge. group is 0 for the word and its other forms, and one more
// than the index of a short form, which is introduced on its own. not marks a
// negated form, which names its own plain words.
type floorForm struct {
	word, group int
	not         bool
	form        string
	re          *regexp.Regexp
}

var floorForms = compileFloorForms()

func compileFloorForms() []floorForm {
	var out []floorForm
	for i, w := range belowFloor {
		forms := append([]string{w.Word}, w.Also...)
		groups := make([]int, len(forms))
		for j, s := range w.Short {
			forms, groups = append(forms, s), append(groups, j+1)
		}
		for k, f := range forms {
			out = append(out, floorForm{word: i, group: groups[k], form: f, re: formRe(f)})
		}
		for _, f := range w.Not {
			out = append(out, floorForm{word: i, not: true, form: f, re: formRe(f)})
		}
	}
	slices.SortStableFunc(out, func(a, b floorForm) int { return len(b.form) - len(a.form) })
	return out
}

// formRe is the pattern of one form: its words apart by white space or a
// hyphen, in either case (an initialism in capitals alone, since "ci" is no
// word of it), its last word in the singular or the plural.
func formRe(f string) *regexp.Regexp {
	parts := strings.Fields(f)
	for j, p := range parts {
		parts[j] = regexp.QuoteMeta(p)
	}
	last := len(parts) - 1
	switch p := parts[last]; {
	case strings.HasSuffix(p, "y") && len(p) > 1 && !strings.ContainsRune("aeiou", rune(p[len(p)-2])):
		parts[last] = p[:len(p)-1] + "(?:y|ies)"
	case strings.HasSuffix(p, "ch"), strings.HasSuffix(p, "sh"), strings.HasSuffix(p, "s"), strings.HasSuffix(p, "x"):
		parts[last] = p + "(?:es)?"
	default:
		parts[last] = p + "s?"
	}
	fold := "(?i)"
	if f == strings.ToUpper(f) {
		fold = ""
	}
	return regexp.MustCompile(fold + strings.Join(parts, `[\s\p{Zs}-]+`))
}

// floorUse is one use of a word below the floor: the word's index in
// belowFloor, the form's group, and where the use starts and ends.
type floorUse struct {
	word, group, start, end int
	not                     bool
}

// floorUses is every use of a word below the floor in s, in reading order. A
// use is a whole word in any case, the plural included; a word inside a longer
// one ("emerged", "merge_queue") is that longer word and no use. A word joined
// to another by a hyphen is still the word: "auto-merge" is a merge and
// "pre-commit" a commit, to a reader who knows neither.
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
			out = append(out, floorUse{word: f.word, group: f.group, start: m[0], end: m[1], not: f.not})
		}
	}
	slices.SortFunc(out, func(a, b floorUse) int { return a.start - b.start })
	return out
}

// insideWord reports whether s[start:end] is part of a longer word: a letter,
// a digit or an underscore touches it.
func insideWord(s string, start, end int) bool {
	before, _ := utf8.DecodeLastRuneInString(s[:start])
	after, _ := utf8.DecodeRuneInString(s[end:])
	return wordRune(before) || wordRune(after)
}

func wordRune(r rune) bool {
	return r != utf8.RuneError && (r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r))
}

// The one sentence that introduces a word (the floor's own rule: "a checksum
// is a fingerprint saying two copies are identical"), read by its shape: the
// word, with no article or an indefinite one, opens a sentence, which goes on
// to say what it is or means without using the word again; or a sentence
// saying what a thing is ends on what it is called. A short form may be named
// beside the word ("a pull request (PR)", "a pull request, or PR,") or at the
// end of what it is ("..., a PR for short"). "The merge is the last
// step" uses a merge the reader is assumed to know, so it introduces nothing,
// and neither does a sentence after a colon, which goes on with the one before.
var (
	introOpensRe = regexp.MustCompile(`(?i)(?:^|[.?!]["')\]]*\s+|\n[ \t]*\n\s*|\n[ \t]*(?:[-*+]|[0-9]+\.)[ \t]+)(?:an?\s+)?["'\x{201c}\x{2018}]?$`)
	introSaysRe  = regexp.MustCompile(`(?i)^["'\x{201d}\x{2019}]?(?:\s*\([^()]*\)|,\s*or\s+[^,.?!;]{1,40},)?\s+(?:is|are|means?)\s+((?:an?|the|one|what|where|when|who|how)\b[^.?!;]*?)(?:[.!;]|\n[ \t]*\n|$)`)
	introNamesRe = regexp.MustCompile(`(?i)([^.?!]*)\s(?:is|are)\s+(?:called|known\s+as)\s+(?:(?:an?|the)\s+)?["'\x{201c}\x{2018}]?$`)
	introEndsRe  = regexp.MustCompile(`^["'\x{201d}\x{2019}]?(?:\s*\([^()]*\)|,\s*or\s+[^,.?!;]{1,40})?\s*[.!]`)
)

// introWords is the fewest words that say what a word is: fewer ("is the
// same") use the word and introduce nothing.
const introWords = 3

// introWindow bounds how much text on each side of a use is read for its
// sentence, so a payload of any size costs the same; no sentence that
// introduces a word is longer.
const introWindow = 1 << 10

// introPhraseRe is a sentence opening on the product's own term whose first
// word is the word below the floor ("A push notification is a message the app
// sends."): one or two more words, then what the term is.
var introPhraseRe = regexp.MustCompile(`(?i)^((?:[ \t]+\p{L}+){1,2}?)\s+(?:is|are|means?)\s+((?:an?|the|one|what|where|when|who|how)\b[^.?!;]*?)(?:[.!;]|\n[ \t]*\n|$)`)

// introducedAt reports whether the use u of a word in s is in the sentence
// that introduces it, and where that sentence ends; phrase is the words after
// it when the sentence introduces a term the word opens, which introduces that
// term alone.
func introducedAt(s string, u floorUse) (end int, phrase string, ok bool) {
	before, after := s[:u.start], s[u.end:]
	if len(before) > introWindow {
		before = "x" + before[len(before)-introWindow:] // mid-sentence, so never an opening
	}
	if len(after) > introWindow {
		after = after[:introWindow] + "?" // cut, so never an end
	}
	// What the word is said to be uses none of the word's own forms, and no
	// long form defines its short one; a short form named in it ("CI for
	// short") is the naming, not a use.
	says := func(complement string) bool {
		return len(strings.Fields(complement)) >= introWords &&
			!slices.ContainsFunc(floorUses(complement), func(v floorUse) bool {
				return v.word == u.word && (v.group == u.group || v.group == 0)
			})
	}
	if introOpensRe.MatchString(before) {
		if m := introSaysRe.FindStringSubmatchIndex(after); m != nil && says(after[m[2]:m[3]]) {
			return u.end + m[1], "", true
		}
		if m := introPhraseRe.FindStringSubmatchIndex(after); m != nil && says(after[m[4]:m[5]]) {
			return u.end + m[1], strings.ToLower(strings.Join(strings.Fields(after[m[2]:m[3]]), " ")), false
		}
	}
	if m := introNamesRe.FindStringSubmatch(before); m != nil && says(m[1]) {
		if e := introEndsRe.FindStringIndex(after); e != nil {
			return u.end + e[1], "", true
		}
	}
	return 0, "", false
}

// introductions is what a tab's question text introduces: for each word and
// form group, whether it is introduced, and the terms it opens that are.
type introductions struct {
	word  map[[2]int]bool
	terms map[[2]int][]string
}

// covers reports whether the use u in s is introduced: its word, or the term
// it opens there.
func (in introductions) covers(s string, u floorUse) bool {
	k := [2]int{u.word, u.group}
	if in.word[k] {
		return true
	}
	next := strings.ToLower(strings.Join(strings.Fields(s[u.end:min(len(s), u.end+introWindow)]), " "))
	return slices.ContainsFunc(in.terms[k], func(t string) bool {
		rest, ok := strings.CutPrefix(next, t)
		r, _ := utf8.DecodeRuneInString(rest)
		return ok && !wordRune(r)
	})
}

// introduced is, for each word and form group, whether the tab's question text
// introduces it: its first use there is the sentence introducing it, or it is
// named in the sentence introducing another form of the same word
// ("Continuous integration (CI) is the automatic checks.", "CI (continuous
// integration) is ..."). An initialism the sentence does not name is not
// introduced with the word. A first use introducing a term the word opens
// introduces that term.
func introduced(text string) introductions {
	in := introductions{word: map[[2]int]bool{}, terms: map[[2]int][]string{}}
	sentences := map[int][][2]int{} // the word's introducing sentences, by word
	for _, u := range floorUses(text) {
		k := [2]int{u.word, u.group}
		if _, met := in.word[k]; met {
			continue
		}
		if slices.ContainsFunc(sentences[u.word], func(sp [2]int) bool { return u.start >= sp[0] && u.start < sp[1] }) {
			in.word[k] = true
			continue
		}
		end, term, ok := introducedAt(text, u)
		in.word[k] = ok
		if term != "" {
			in.terms[k] = append(in.terms[k], term)
		}
		if ok {
			sentences[u.word] = append(sentences[u.word], [2]int{u.start, end})
		}
	}
	return in
}

// floorRemedy is the remedy of a word below the floor whose plain words are
// plain.
func floorRemedy(plain string) string {
	return fmt.Sprintf("Say %q instead, wherever the tab uses the word, or introduce it in one sentence of the question text before its first use, as in \"A checksum is a fingerprint saying two copies are identical.\" "+
		"A word meant in its everyday sense or as the product's own word (a shop's checkout, a bank's branch) reads the same to the check, so introduce it once: "+
		"the sentence opens on the word bare or after \"a\" (\"A push is ...\"), never after \"the\", or on the product's own term the word opens (\"A push notification is ...\"), which introduces that term alone. "+
		"The floor is stated at %s under abcd's plugin root.", plain, KnowledgeFloorPage)
}

// floorFindings is a tab's words below the floor, one finding per word and
// form group at its first use in reading order. A word whose first use in the
// question text is the sentence introducing it passes everywhere on the tab.
func floorFindings(n int, t Tab) []Finding {
	intro := introduced(t.Text)
	var out []Finding
	type key struct {
		word, group int
		not         bool
	}
	named := map[key]bool{}
	for _, fl := range fields(t) {
		for _, u := range floorUses(fl.text) {
			k := key{u.word, u.group, u.not}
			if intro.covers(fl.text, u) || named[k] {
				continue
			}
			named[k] = true
			plain := belowFloor[u.word].Plain
			if u.not {
				plain = belowFloor[u.word].NotPlain
			}
			out = append(out, Finding{Tab: n, Part: fl.part, Rule: RuleRegister, Value: echo(fl.text[u.start:u.end]),
				Limit: "no word below the product thinker's knowledge floor that the question does not introduce", Remedy: floorRemedy(plain), Plain: plain})
		}
	}
	return out
}
