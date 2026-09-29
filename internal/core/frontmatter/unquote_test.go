package frontmatter

import "testing"

// TestUnquoteReversesTheEmittedEscaping pins the ONE decoder for a
// double-quoted frontmatter scalar's backslash escaping (iss-2608301212424896).
// capture's serialiser escapes a backslash and a double quote; every reader of
// that value — the ledger's own parser and the committed-record gate that must
// refuse exactly what the parser refuses — reverses it here rather than each
// keeping a private replica of the loop.
func TestUnquoteReversesTheEmittedEscaping(t *testing.T) {
	for name, tc := range map[string]struct{ in, want string }{
		"plain":            {`nothing to undo`, `nothing to undo`},
		"escaped quote":    {`he said \"hi\"`, `he said "hi"`},
		"escaped slash":    {`a \\ b`, `a \ b`},
		"escaped colon":    {`pursued\: the token`, `pursued: the token`},
		"trailing escape":  {`ends with \`, `ends with \`},
		"escape then text": {`\a\b`, `ab`},
		"empty":            {``, ``},
	} {
		if got := Unquote(tc.in); got != tc.want {
			t.Fatalf("%s: Unquote(%q) = %q, want %q", name, tc.in, got, tc.want)
		}
	}
}

// TestUnquoteScalarStripsAMatchedPairThenDecodes pins the idiom every reader of
// a possibly-quoted value needs (iss-2608311039531552). Unquote takes the
// scalar's INNER text, so a caller holding the raw value strips the quotes
// first; three readers each held a private copy of that strip, and a caller
// that forgot it compared a value against itself in its refusal. The strip
// lives beside the decoder so no caller re-derives it.
func TestUnquoteScalarStripsAMatchedPairThenDecodes(t *testing.T) {
	for name, tc := range map[string]struct{ in, want string }{
		"quoted":               {`"major"`, `major`},
		"quoted with escapes":  {`"he said \"hi\""`, `he said "hi"`},
		"empty quoted":         {`""`, ``},
		"bare token":           {`itd-5`, `itd-5`},
		"lone quote":           {`"`, `"`},
		"unclosed":             {`"open`, `"open`},
		"closing only":         {`close"`, `close"`},
		"single-quoted stays":  {`'single'`, `'single'`},
		"untrimmed stays":      {` "x" `, ` "x" `},
		"empty":                {``, ``},
		"inner quotes at ends": {`"a" and "b"`, `a" and "b`},
	} {
		got, quoted := UnquoteScalar(tc.in)
		if got != tc.want {
			t.Errorf("%s: UnquoteScalar(%q) = %q, want %q", name, tc.in, got, tc.want)
		}
		if wantQuoted := got != tc.in; quoted != wantQuoted {
			t.Errorf("%s: UnquoteScalar(%q) reports quoted=%v, want %v", name, tc.in, quoted, wantQuoted)
		}
	}
}
