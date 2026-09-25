package termsafe

import "testing"

// TestEncodeHiddenRunesBlockKeepsLineStructure: a record body is multi-line
// prose, so the block form encodes every rune EncodeHiddenRunes encodes EXCEPT
// the three that are the body's own structure — the line feed, the tab, and a
// carriage return that is half of a CRLF pair. Everything else hidden is
// percent-encoded losslessly (iss-2608301206073609).
func TestEncodeHiddenRunesBlockKeepsLineStructure(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"clean text is untouched", "a line\n\tindented\nlast", "a line\n\tindented\nlast"},
		{"bidi override is encoded", "x‮y", "x%E2%80%AEy"},
		{"zero-width space is encoded", "a​b", "a%E2%80%8Bb"},
		{"C1 and DEL are encoded", "a\u0085b\x7fc", "a%C2%85b%7Fc"},
		{"an escape is encoded, the newline kept", "a\x1b[2J\nb", "a%1B[2J\nb"},
		{"CRLF is kept, a bare CR is encoded", "a\r\nb\rc", "a\r\nb%0Dc"},
		{"invalid UTF-8 is encoded raw", "a\xffb", "a%FFb"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := EncodeHiddenRunesBlock(c.in); got != c.want {
				t.Errorf("EncodeHiddenRunesBlock(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
