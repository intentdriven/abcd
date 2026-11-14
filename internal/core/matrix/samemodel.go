package matrix

// samemodel.go answers whether a reviewer's model may count as a model other
// than the one that wrote a change (itd-2610301750162657, spc-2610301925429250
// step 4). The names are written by agents and configuration, so both are
// untrusted, and the answer leans to "the same model": a change recorded as
// reviewed by another model while the writer's own reviewed it is the costly
// failure, and a reviewer refused that was in truth another model only costs a
// switch of class or a hand-back. It is the opposite lean to sameModel's
// (export.go), which must never read a substitution as the asked model.

import (
	"strings"
	"unicode"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// maxComparedModel bounds a name the comparison reads; a longer one is not
// read, and so counts as the writer's.
const maxComparedModel = 200

// OtherModel reports whether reviewer surely names a model other than writer.
// Case is folded and a vendor prefix stripped from each, in either form a name
// carries one (`anthropic/` before the model, or `Claude:` as an Assisted-by
// trailer writes it); a name equal to the other followed by `-`, `@` or `:`
// and anything (a version, a size or a date stamp) is the same model; and an
// empty, overlong or unparseable name on either side (a space, a control rune
// or a rune the terminal cleaning masks in it) is the writer's. Where a colon
// may part a vendor from a model or a model from its tag, both readings are
// compared, and either one matching makes the two the same.
func OtherModel(writer, reviewer string) bool {
	ws, ok := comparedForms(writer)
	if !ok {
		return false
	}
	rs, ok := comparedForms(reviewer)
	if !ok {
		return false
	}
	for _, w := range ws {
		for _, r := range rs {
			if extends(w, r) || extends(r, w) {
				return false
			}
		}
	}
	return true
}

// comparedForms are the readings of a name OtherModel compares, and false for
// a name it cannot read.
func comparedForms(s string) ([]string, bool) {
	if s == "" || len(s) > maxComparedModel || termsafe.Sanitize(s) != s {
		return nil, false
	}
	for _, c := range s {
		if unicode.IsSpace(c) || unicode.IsControl(c) {
			return nil, false
		}
	}
	s = strings.ToLower(s)
	if i := strings.LastIndexByte(s, '/'); i >= 0 {
		s = s[i+1:]
	}
	if s == "" {
		return nil, false
	}
	forms := []string{s}
	if v, rest, ok := strings.Cut(s, ":"); ok && v != "" && rest != "" && !strings.ContainsAny(v, "-@") {
		forms = append(forms, rest)
	}
	return forms, true
}

// extends reports whether long is short, or short followed by `-`, `@` or `:`
// and anything.
func extends(short, long string) bool {
	rest, ok := strings.CutPrefix(long, short)
	return ok && (rest == "" || strings.ContainsRune("-@:", rune(rest[0])))
}
