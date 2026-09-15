package capture

import (
	"strings"
	"testing"
)

// The gate exists to refuse exactly what the reader refuses, and a trailing
// comment is where the two parsers could part. record-lint reads a record
// through frontmatter.Fields, which strips a YAML comment before it judges the
// value (iss-2608301744268001); the strict ledger parser here is a second reader
// of the same bytes, and one that kept the comment would read `severity: minor
// # todo` as an out-of-enum value while the gate read it as `minor` — the
// split-verdict shape, with the permissive one on the gate side.
//
// So the strip is the shared scanner's, called from both. The rule is YAML's in
// both places: a comment starts at a `#` preceded by whitespace and outside
// quotes.
func TestLedgerParserStripsATrailingCommentLikeTheGate(t *testing.T) {
	for _, c := range []struct{ name, line, want string }{
		{"enum value", "severity: minor # todo", "minor"},
		{"no space after the hash", "severity: minor #todo", "minor"},
		{"comment only", "found_during: # todo", ""},
		{"hash inside the value", "slug: a#b", "a#b"},
		{"quoted hash", `slug: "a #b"`, "a #b"},
	} {
		t.Run(c.name, func(t *testing.T) {
			key := c.line[:strings.IndexByte(c.line, ':')]
			fm, err := parseFrontmatterBlock([]string{"id: iss-1", c.line})
			if err != nil {
				t.Fatalf("parseFrontmatterBlock(%q): %v", c.line, err)
			}
			got, _ := fm[key].(string)
			if got != c.want {
				t.Errorf("%q parsed %q as %q, want %q", c.line, key, got, c.want)
			}
		})
	}
}
