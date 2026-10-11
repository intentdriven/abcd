package recordid

import (
	"regexp"
	"strings"
)

// MaxSlugLen is the one cap on a minted record slug, shared by every verb that
// mints a record filename: capture, intent, spec and decide alike. A record's
// path is `<store>/<family>-<id>-<slug>.md`, and a checkout on Windows refuses a
// path past 260 characters unless core.longpaths is set; the checkout's own
// location and a worktree store's prefix come on top of the path inside the
// repository, so the slug is the part held short. Ruled by the product thinker
// on 2026-10-10 (iss-2610100626320367), down from the 60 the stores used.
const MaxSlugLen = 40

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

// CapSlug cuts a slug that is already kebab-case to at most max characters, for
// a slug that arrives whole rather than as free text: one given explicitly, one
// carried from another record (an intent's slug into its spec), and an existing
// record's slug being renamed to the cap. Text has words; a slug has only its
// hyphens, so the cut lands on the last hyphen that keeps the slug within the
// cap and drops it, and never leaves a trailing hyphen. A first segment that
// alone exceeds the cap is cut inside, as Slug does, so the result is never
// empty for a non-empty slug. A slug within the cap, or a max of zero or less,
// is returned unchanged, so the cut is idempotent and a slug Slug derived at the
// same cap passes through it untouched.
func CapSlug(slug string, max int) string {
	if max <= 0 || len(slug) <= max {
		return slug
	}
	cut := slug[:max+1]
	if i := strings.LastIndexByte(cut, '-'); i > 0 {
		return strings.TrimRight(slug[:i], "-")
	}
	return strings.Trim(slug[:max], "-")
}
