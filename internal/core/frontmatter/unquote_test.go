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
