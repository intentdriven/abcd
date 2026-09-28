package termsafe

import "testing"

// TestPairCodeSpanClosesOnARunOfTheSameLength pins CommonMark's pairing rule:
// the closer is the first later run of EXACTLY the opening length, a longer run
// never closes by its prefix, and a run no such closer follows opens nothing
// (iss-2609262322244502).
func TestPairCodeSpanClosesOnARunOfTheSameLength(t *testing.T) {
	for _, c := range []struct {
		s    string
		i    int
		ok   bool
		raw  string
		endS string // s[End:]
	}{
		{"`a` b", 0, true, "a", " b"},
		{"``a```b`` c", 0, true, "a```b", " c"},
		{"`a``b` c", 0, true, "a``b", " c"},
		{"x `a` y", 2, true, "a", " y"},
		{"```` ``` ````", 0, true, " ``` ", ""},
		{"`a\nb`", 0, true, "a\nb", ""},
		{"`a``", 0, false, "", ""},
		{"``a`", 0, false, "", ""},
		{"``a```", 0, false, "", ""},
		{"`", 0, false, "", ""},
		{"a", 0, false, "", ""},
		{"`a`", 5, false, "", ""},
	} {
		sp, ok := PairCodeSpan(c.s, c.i)
		if ok != c.ok {
			t.Errorf("PairCodeSpan(%q, %d) ok = %v, want %v", c.s, c.i, ok, c.ok)
			continue
		}
		if !ok {
			continue
		}
		if sp.Start != c.i || sp.Raw(c.s) != c.raw || c.s[sp.End:] != c.endS {
			t.Errorf("PairCodeSpan(%q, %d) = %+v (raw %q, after %q), want raw %q, after %q",
				c.s, c.i, sp, sp.Raw(c.s), c.s[sp.End:], c.raw, c.endS)
		}
	}
}

// TestCodeSpanTextAppliesTheContentRules pins CommonMark's two content rules: a
// line ending is a space, and ONE space comes off each side only when both are
// there and the content is not all spaces.
func TestCodeSpanTextAppliesTheContentRules(t *testing.T) {
	for raw, want := range map[string]string{
		"a":        "a",
		" a ":      "a",
		"  a  ":    " a ",
		" a":       " a",
		"a ":       "a ",
		" ":        " ",
		"  ":       "  ",
		" `` ":     "``",
		"a\nb":     "a b",
		"a\r\nb":   "a b",
		"\na\n":    "a",
		"\n":       " ",
		"":         "",
		" \n ":     "   ",
		" a\r":     "a",
		"x\ry  z ": "x y  z ",
	} {
		if got := CodeSpanText(raw); got != want {
			t.Errorf("CodeSpanText(%q) = %q, want %q", raw, got, want)
		}
	}
}
