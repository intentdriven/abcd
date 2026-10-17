package recordid

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// NamesID reports whether text names the record id as a whole token: `itd-4`
// is not named by `itd-48`, `itd-4abc` or `xitd-4`. A hyphen after the id
// still names it, because that is the head of the record's own file name or
// wikilink (`itd-4-slug.md`, `[[itd-4-slug]]`), the way a record most often
// names another; no id continues with a hyphen, so it cannot start a different
// id. Before the id, a letter, a digit or a hyphen makes a longer word. Both
// neighbours are decoded as runes, so a non-ASCII letter is a letter.
//
// It is the one id test record-lint's cross-document rules and the
// consistency filer's prose match both apply.
func NamesID(text, id string) bool {
	if id == "" {
		return false
	}
	for i := 0; ; {
		k := strings.Index(text[i:], id)
		if k < 0 {
			return false
		}
		start, end := i+k, i+k+len(id)
		prev, _ := utf8.DecodeLastRuneInString(text[:start])
		next, _ := utf8.DecodeRuneInString(text[end:])
		before := start == 0 || !(unicode.IsLetter(prev) || unicode.IsDigit(prev) || prev == '-')
		after := end == len(text) || !(unicode.IsLetter(next) || unicode.IsDigit(next))
		if before && after {
			return true
		}
		i = start + 1
	}
}
