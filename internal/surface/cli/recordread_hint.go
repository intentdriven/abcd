package cli

import (
	"fmt"
	"strings"
	"unicode"
)

// recordread_hint.go — the record dispatcher named where a caller reaches for
// a "status" or "show" sub-verb (iss-2609190337466942).
//
// The record verbs carry no status or show sub-verb, because the bare
// dispatcher answers that question for every family: `abcd <itd-N>` prints the
// record's bucket, its path and the next move. A fresh operator tries
// `abcd intent status itd-5` first, and the refusal listed the sub-verbs and
// said nothing of the form that works. The refusal is unchanged; it gains the
// dispatcher invocation, filled with the id the caller gave when they gave one.

// recordReadFamilies maps a record verb to the id placeholder its records take.
var recordReadFamilies = map[string]string{
	"capture": "iss-N",
	"intent":  "itd-N",
	"spec":    "spc-N",
}

// recordReadWords are the sub-verb spellings read as "show me one record".
var recordReadWords = map[string]bool{"status": true, "show": true}

// recordReadHint returns the sentence naming the record dispatcher when args,
// under the record verb parent, are shaped like a status/show sub-verb call —
// the word alone, or the word before one record id — and "" otherwise, so
// prose that merely begins with the word is left to the create path.
func recordReadHint(parent string, args []string) string {
	family, ok := recordReadFamilies[parent]
	if !ok || len(args) == 0 || len(args) > 2 || !recordReadWords[args[0]] {
		return ""
	}
	if strings.IndexFunc(args[0], unicode.IsSpace) >= 0 {
		return ""
	}
	target := "<" + family + ">"
	if len(args) == 2 {
		if !recordIDRe.MatchString(args[1]) {
			return ""
		}
		target = args[1]
	}
	return fmt.Sprintf("there is no %s sub-verb — to see one record's bucket, path and next move, run `abcd %s`",
		args[0], target)
}

// positionalsFrom returns the positional tokens of args from the first one
// equal to word onward, flags dropped: the shape recordReadHint reads, recovered
// from a cobra usage error that names only the unknown word.
func positionalsFrom(args []string, word string) []string {
	var out []string
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		if len(out) == 0 && a != word {
			continue
		}
		out = append(out, a)
	}
	return out
}
