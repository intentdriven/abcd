package frontmatter

import "testing"

// TestScalarStringReadsOnlyASingleLineString pins the ONE definition of "a
// single-line string" a command-written key rests on (iss-2609200830076665):
// the writer encodes one, the reader decodes one, and record-lint reports every
// other shape as a state no write path produced. The table is the contract in
// both directions — what is read, and what is refused.
func TestScalarStringReadsOnlyASingleLineString(t *testing.T) {
	for name, tc := range map[string]struct {
		in   string
		want string
		ok   bool
	}{
		"bare":                  {`awaiting the reading rethink`, `awaiting the reading rethink`, true},
		"double quoted":         {`"awaiting the reading rethink"`, `awaiting the reading rethink`, true},
		"double quoted escapes": {`"say \"why\" \\ now"`, `say "why" \ now`, true},
		"single quoted":         {`'it''s held'`, `it's held`, true},
		"padded":                {`  bare  `, `bare`, true},
		"blank":                 {``, ``, false},
		"whitespace":            {`   `, ``, false},
		"null":                  {`null`, ``, false},
		"tilde":                 {`~`, ``, false},
		"tagged null":           {`!!null`, ``, false},
		"empty double quoted":   {`""`, ``, false},
		"empty single quoted":   {`''`, ``, false},
		"flow list":             {`[a, b]`, ``, false},
		"empty flow list":       {`[]`, ``, false},
		"flow map":              {`{why: a}`, ``, false},
		"block scalar literal":  {`|`, ``, false},
		"block scalar folded":   {`>-`, ``, false},
		"block scalar comment":  {`| # why`, ``, false},
		"unterminated double":   {`"abc`, ``, false},
		"unterminated single":   {`'abc`, ``, false},
		"lone double quote":     {`"`, ``, false},
	} {
		got, ok := ScalarString(tc.in)
		if ok != tc.ok || got != tc.want {
			t.Errorf("%s: ScalarString(%q) = (%q, %v), want (%q, %v)", name, tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

// TestQuoteScalarRoundTripsThroughScalarString: what QuoteScalar writes,
// ScalarString reads back byte-for-byte — the encoder and the decoder are one
// pair, so a value the verb wrote is never reported by the gate that reads it.
func TestQuoteScalarRoundTripsThroughScalarString(t *testing.T) {
	for _, in := range []string{
		`plain`,
		`say "why"`,
		`a \ b`,
		`quote at end "`,
		`backslash at end \`,
		`colon: and # hash`,
		`café — unicode stays`,
	} {
		q := QuoteScalar(in)
		if q[0] != '"' || q[len(q)-1] != '"' {
			t.Fatalf("QuoteScalar(%q) = %q is not double-quoted", in, q)
		}
		got, ok := ScalarString(q)
		if !ok || got != in {
			t.Errorf("round trip of %q through %q = (%q, %v)", in, q, got, ok)
		}
	}
}
