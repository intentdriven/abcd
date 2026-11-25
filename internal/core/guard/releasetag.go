package guard

import (
	"path"
	"regexp"
	"strings"
)

// releasetag.go reads the one operand `abcd update` takes, a release tag: the
// shape the registry entry abcd-update-tag names (ShapeReleaseTag), and the
// reading of the same act under a program whose name carries abcd's
// (personAct.tagAct).

// ShapeReleaseTag names the shape in an entry's arg_shapes for a release tag
// standing as the first operand after the subcommand.
const ShapeReleaseTag = "release-tag"

// ReleaseTagWord, ReleaseTagStart and VerbWord are the update command's own
// reading of its operand: a word of a tag's alphabet, where a lowercase word that
// does not start as every release tag does is a mistyped sub-verb. Each is the
// pattern the command reads by (TestTheGuardsTagPatternsAreTheirSources).
var (
	ReleaseTagWord  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]{0,63}$`)
	ReleaseTagStart = regexp.MustCompile(`^v[0-9]`)
	VerbWord        = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
)

// releaseTagCouldBe reports whether an operand is a release tag, or can still be
// one: a word with a part the line does not fix, or a glob's, whose fixed text a
// tag can hold. `{}` is such a part, the placeholder find -exec and xargs -I put
// an operand in.
func releaseTagCouldBe(tok string, glob bool) bool {
	tok = strings.ReplaceAll(tok, "{}", unknownText)
	if glob {
		tok = strings.NewReplacer("*", unknownText, "?", unknownText).Replace(bracketsAsStar(tok))
	}
	if !isUnknown(tok) {
		return ReleaseTagWord.MatchString(tok) && (!VerbWord.MatchString(tok) || ReleaseTagStart.MatchString(tok))
	}
	fixed := knownText(tok)
	if knownLead(tok) == "" {
		fixed = "v" + fixed
	}
	return ReleaseTagWord.MatchString(fixed)
}

// unnamedPackage reports whether a word is a package path that names no directory
// (`.`, `./`, `..`), so the program a `go run` of it starts takes the name of the
// directory it runs in.
func unnamedPackage(tok string) bool {
	base := path.Base(tok)
	return base == "." || base == ".."
}

// entersAbcd reports whether a segment is a `cd`, `pushd` or `popd` with a word
// naming abcd's directory.
func entersAbcd(s segment) bool {
	for _, a := range commandSites(s) {
		if nameCouldBeAny(s.tokens[a.idx], directoryChanges) && firstAbcdWord(s, a.idx+1) >= 0 {
			return true
		}
	}
	return false
}

// tagAct reports whether any way the site's words may run is an update with a
// release tag as its operand, under a program whose name carries abcd's, one the
// line does not fix, or a `go run` of an unnamed package where the line enters a
// directory named for abcd. Its words are common to other programs, so a program
// of any other name is left alone.
func (a personAct) tagAct(s segment, site arrival, inAbcd bool) bool {
	for _, r := range actReadings(s, site) {
		if r.from == site.idx+1 && !r.fixedName {
			if lit, has := s.literal[site.idx]; has && !lit.split {
				r.fixedName = carriesAbcd(lit.text)
				r.named = r.fixedName
			}
			if unnamedPackage(s.tokens[site.idx]) && (inAbcd || firstAbcdBefore(s, site.idx)) {
				r.fixedName, r.named = true, true
			}
		}
		if r.named && a.tagFrom(s, r) {
			return true
		}
	}
	return false
}

// firstAbcdBefore reports whether a word before the site names abcd, as a `go run`
// given its directory does (`go run -C cmd/abcd .`).
func firstAbcdBefore(s segment, site int) bool {
	for _, w := range s.tokens[:site] {
		if carriesAbcd(w) {
			return true
		}
	}
	return false
}

// tagFrom reads the words as the update command's parser does: options pass, the
// first other word is the verb and the next its operand, after a "--" or not. An
// option before the verb may take the word after it; none of the verb's own takes
// one. A word the line does not fix may be the verb, an option, an option's value
// or no word, and a value the line assigned may split into several. On a program
// the line does not fix, the verb and a tag must be spelled.
func (a personAct) tagFrom(s segment, r actReading) bool {
	verb := a.verbs[0]
	atVerb, atTag, value := true, false, false
	var words []actWord
	for _, w := range actWordsOf(s, r.from) {
		if w.unfixed || !strings.ContainsAny(w.text, " \t\n") {
			words = append(words, w)
			continue
		}
		for _, f := range strings.Fields(w.text) {
			words = append(words, actWord{text: f})
		}
	}
	for _, w := range words {
		if !w.unfixed && w.text == "--" {
			atVerb = false
			continue
		}
		if known := knownText(w.text); strings.HasPrefix(known, "-") && known != "-" {
			value = atVerb && !strings.Contains(known, "=")
			continue
		}
		if atTag {
			if r.fixedName && releaseTagCouldBe(w.text, w.glob) || !w.unfixed && ReleaseTagStart.MatchString(w.text) {
				return true
			}
		}
		open := w.unfixed && vanishable(w.text)
		isVerb := atVerb && (w.text == verb || w.unfixed && r.fixedName && unknownCouldBe(w.text, verb))
		atTag = isVerb || atTag && open
		atVerb = atVerb && (value || open)
		value = false
		if !atVerb && !atTag {
			return false
		}
	}
	return false
}
