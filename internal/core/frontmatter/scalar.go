package frontmatter

import (
	"regexp"
	"strings"
)

// BlockScalarHeaderRe matches a YAML block-scalar header and nothing else: `|`,
// `>`, with the chomping and indentation indicators the spelling allows (`|-`,
// `>+`, `|2-`) and the trailing comment YAML permits after the header. A key
// carrying one holds its value on the lines BELOW it, so the same-line scanner
// reports the header as the value — which is why a reader asking "is this a
// single-line string?" has to recognise the header rather than take the byte
// for a value. It lives here, beside the scanner whose reading it corrects, so
// record-lint's block-emptiness leg and the record readers test one pattern.
var BlockScalarHeaderRe = regexp.MustCompile(`^[|>][0-9+-]*(?:\s+#.*)?$`)

// ScalarString reads a raw same-line value — exactly as Fields returned it —
// as a populated single-line string scalar, and returns the string it spells.
// ok is false for every other shape: a blank or null value, an explicitly
// empty string, an empty or populated flow collection (`[…]`, `{…}`), a
// block-scalar header, and a quoted scalar its own line does not close.
//
// It is the ONE definition of "a single-line string" for a frontmatter key
// whose value a command writes as one — the writer (core/intent's hold) and
// the gate that judges committed bytes (record-lint's record_provenance) both
// come here, so the gate refuses exactly the shapes the reader does not read,
// and a shape the reader accepts is never reported. Quoting is decoded by the
// package's own decoders: a double-quoted scalar through Unquote (the mirror
// of QuoteScalar), a single-quoted one by folding the doubled apostrophe.
//
// It is a line scanner's reading, not a YAML parser's: a node tag or anchor in
// front of a bare scalar is returned as part of the string, because the tools
// that write these keys never emit one and a reader that stripped it would be
// claiming a grammar the scanner does not have.
func ScalarString(raw string) (value string, ok bool) {
	v := strings.TrimSpace(raw)
	if EmptinessOf(v) != Populated {
		return "", false
	}
	if BlockScalarHeaderRe.MatchString(v) {
		return "", false
	}
	switch v[0] {
	case '[', '{':
		return "", false
	case '"':
		if len(v) < 2 || v[len(v)-1] != '"' {
			return "", false
		}
		return Unquote(v[1 : len(v)-1]), true
	case '\'':
		if len(v) < 2 || v[len(v)-1] != '\'' {
			return "", false
		}
		return strings.ReplaceAll(v[1:len(v)-1], "''", "'"), true
	}
	return v, true
}

// QuoteScalar renders s as a double-quoted frontmatter scalar: a backslash and
// a double quote are each written `\`-prefixed, and nothing else is escaped. It
// is the encoder Unquote mirrors, spelled once so a value written here reads
// back byte-for-byte through ScalarString.
//
// It does not judge s. A caller writing a single-line key refuses a newline or
// any other control character BEFORE encoding, because a newline inside the
// quotes is a second frontmatter line to the same-line scanner, whatever YAML
// would make of it.
func QuoteScalar(s string) string {
	esc := strings.ReplaceAll(s, `\`, `\\`)
	esc = strings.ReplaceAll(esc, `"`, `\"`)
	return `"` + esc + `"`
}
