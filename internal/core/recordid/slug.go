package recordid

import (
	"regexp"
	"strings"
)

// slugWordSepRe matches every run that separates words in a slug: anything
// but a lowercase letter, a digit or a hyphen. A hyphen inside a word (ac-10,
// spec-kit) is part of the word, so the cap below never splits it.
var slugWordSepRe = regexp.MustCompile(`[^a-z0-9-]+`)

// slugHyphenRunRe collapses a run of hyphens inside a word to one.
var slugHyphenRunRe = regexp.MustCompile(`-{2,}`)

// Slug derives a record filename's slug from free text: lowercased, split into
// words on every non-alphanumeric run, each word's hyphen runs collapsed and
// its ends trimmed, the words joined by hyphens, and the length capped at max.
// The cap lands between words — as many whole words as fit — so a slug never
// ends in a token the text did not contain: a hard cut turned "…criterion
// ac-10" into "…-ac-1", which reads as a record about a different criterion.
// Only a first word that alone exceeds the cap is cut inside, since there is
// no boundary to land on. The result may be empty when the text has no
// slug-able characters; callers refuse that case in their own words.
func Slug(text string, max int) string {
	var words []string
	for _, w := range slugWordSepRe.Split(strings.ToLower(text), -1) {
		w = strings.Trim(slugHyphenRunRe.ReplaceAllString(w, "-"), "-")
		if w != "" {
			words = append(words, w)
		}
	}
	if len(words) == 0 {
		return ""
	}
	if max <= 0 {
		return strings.Join(words, "-")
	}
	if len(words[0]) > max {
		return strings.Trim(words[0][:max], "-")
	}
	out := words[0]
	for _, w := range words[1:] {
		if len(out)+1+len(w) > max {
			break
		}
		out += "-" + w
	}
	return out
}
