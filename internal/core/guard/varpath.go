package guard

import "strings"

// varpath.go reads the one operand shape an entry's arg_shapes can name
// (Pattern.ArgShapes): a word that opens with a variable whose value can be
// empty, directly followed by `/` (iss-2610091942156774). `rm -f "$VAR"/*`
// with VAR empty or unset is `rm -f /*`, and nothing on the line says which:
// the host asks the person to approve that blind, so the guard refuses it
// first and names the rewrite, `"${VAR:?}"/*`, which stops the shell on an
// empty value instead.
//
// The shape is decided while the word is read, from the expansions the
// tokenizer recorded for it (varSite), because the written spelling the
// arg_values compare reads (segment.spelled) cannot tell `${VAR:?}` from
// `${VAR}`: both can print only VAR's value, and the difference is what
// happens when it is empty.

// ShapeUnguardedVariablePath names the shape in an entry's arg_shapes.
const ShapeUnguardedVariablePath = "unguarded-variable-path"

// argShapes are the shapes arg_shapes may name; Validate refuses any other.
var argShapes = map[string]bool{
	ShapeUnguardedVariablePath: true,
}

// guardedValue is the text of the variable's own value, as spellParameter
// writes it (`${X}`), where the `${…}` whose text between the braces is body
// never prints that value empty, and "" otherwise: `${X:?}` stops the shell
// on an empty or unset X, and `${X:-w}` and `${X:=w}` print w then. Whether
// w can be empty is read from w's own texts, which the expansion's texts
// hold beside the value. A plain reference (`${X}`), a test without the
// colon (`${X?}`, `${X-w}`, `${X=w}`, which print a value set to empty), an
// alternative and a subscript keep "". An indirection is not read: the
// value it names is not X's.
func guardedValue(body string) string {
	body = paramText(body)
	indirect, n := paramNameEnd(body)
	if indirect || n <= 0 {
		return ""
	}
	rest := body[n:]
	if strings.HasPrefix(rest, ":?") || strings.HasPrefix(rest, ":-") || strings.HasPrefix(rest, ":=") {
		return "${" + body[:n] + "}"
	}
	return ""
}

// transformsValue reports whether the `${…}` whose text between the braces
// is body is a trim, a replacement, a substring or a case change
// (`${DIR%/}`, `${p%/*}`, `${X/a/b}`, `${X:1}`, `${X^^}`): an expression
// over a value rather than the variable written as a path, which the
// guard leaves allowed as an everyday expansion
// (TestEverydayExpansionsStayAllowed), and which the shape does not read.
// What such an expansion makes of the root is the arg_values compare's to
// read (`${X^}/` is `/`, rm-rf-root-or-home).
func transformsValue(body string) bool {
	body = paramText(body)
	indirect, n := paramNameEnd(body)
	if indirect || n <= 0 || n >= len(body) {
		return false
	}
	rest := body[n:]
	if rest[0] == '[' {
		// An element's trim (`${X[0]%zzz}`) is a trim.
		k := subscriptEnd(rest)
		if k < 0 || k+1 >= len(rest) {
			return false
		}
		rest = rest[k+1:]
	}
	switch {
	case strings.HasPrefix(rest, ":?"), strings.HasPrefix(rest, ":-"), strings.HasPrefix(rest, ":="), strings.HasPrefix(rest, ":+"):
		return false
	}
	return strings.IndexByte(":%#/^,@", rest[0]) >= 0
}

// opensUnguardedPath reports whether a word, as the tokenizer built it (each
// variable's mark at its site), opens with a variable that can print nothing,
// directly followed by `/`, so the word read with the variable empty is a
// path from the root. A word whose first byte is not a variable's is not it,
// however many variables follow (`./build/$name`), and neither is a word the
// variable ends (`"$VAR"`), which empty is no path at all.
func opensUnguardedPath(word []byte, sites []varSite) bool {
	if len(sites) == 0 || sites[0].at != 0 {
		return false
	}
	return siteOpensPath(word, sites, 0)
}

// siteOpensPath reports whether the site sites[k] can print nothing, or a
// text that opens with a variable that can, where what follows in the word
// or the text is `/` — or is another site of which the same holds
// (`"$A$B"/x`). A transformsValue expansion is not read. Each of the
// expansion's texts is read: the value a guardedValue expansion guards is
// skipped, and a text past a bound is read
// as able to print nothing, so a bound refuses only where a `/` follows. A
// site whose texts are not known is not read: a raw 0x01 byte at the top of
// a line is text, and a mark a payload's text carries from the enclosing
// shell is read from the string with its variables written out instead
// (spellPayload).
func siteOpensPath(word []byte, sites []varSite, k int) bool {
	s := sites[k]
	next := s.at + max(s.width, 1)
	// followed is asked once whatever the number of texts, so a run of
	// sites is read once each, never once per combination of their texts.
	asked, answer := false, false
	followed := func() bool {
		if asked {
			return answer
		}
		asked = true
		switch {
		case next >= len(word):
		case word[next] == '/':
			answer = true
		case k+1 < len(sites) && sites[k+1].at == next:
			answer = siteOpensPath(word, sites, k+1)
		}
		return answer
	}
	if len(s.texts) == 0 || s.transform {
		return false
	}
	if capped(s.texts) {
		return followed()
	}
	for _, t := range s.texts {
		if s.guarded != "" && t == s.guarded {
			continue
		}
		rest, stripped := stripEmptyableRefs(t)
		switch {
		case rest == "":
			if followed() {
				return true
			}
		case stripped && rest[0] == '/':
			return true
		}
	}
	return false
}

// stripEmptyableRefs is t with its leading run of variable references that
// can print nothing taken off (`$X`, `${X}`, `$1`), and whether any was.
// A reference that cannot be empty ends the run: a length (`${#X}`), the
// shell's own numbers (`$$`, `$?`, `$#`, `$0`), and the home and the working
// directory, which the login and the shell set to absolute paths and whose
// own entries name a delete of them.
func stripEmptyableRefs(t string) (string, bool) {
	stripped := false
	for {
		end, empty := leadingRef(t)
		if end <= 0 || !empty {
			return t, stripped
		}
		t, stripped = t[end:], true
	}
}

// leadingRef returns the length of the variable reference t opens with, 0
// when it opens with none, and whether its value can be empty.
func leadingRef(t string) (end int, empty bool) {
	if len(t) < 2 || t[0] != '$' {
		return 0, false
	}
	if t[1] == '{' {
		depth := 0
		for i := 1; i < len(t); i++ {
			switch t[i] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					return i + 1, refCanBeEmpty(t[2:i])
				}
			}
		}
		return 0, false
	}
	if strings.IndexByte("$?#!", t[1]) >= 0 {
		return 2, refCanBeEmpty(t[1:2])
	}
	if e := simpleParamEnd(t, 1); e > 1 {
		return e, refCanBeEmpty(t[1:e])
	}
	return 0, false
}

// refCanBeEmpty reports whether the reference whose text after its `$`, or
// between its braces, is body can print nothing. Every form the spelling
// keeps as written other than a length (`${!X}`, `${X@Q}`) is read as able
// to: a guard refuses where it cannot read.
func refCanBeEmpty(body string) bool {
	switch body {
	case "HOME", "PWD", "$", "?", "#", "0":
		return false
	}
	return !(len(body) > 1 && body[0] == '#')
}
