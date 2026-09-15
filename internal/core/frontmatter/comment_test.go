package frontmatter

import (
	"strings"
	"testing"
)

// A trailing comment is not part of the value, and the ONE same-line scanner is
// where that is decided. Before this, Fields handed every gate the raw remainder
// of the line, so `severity: minor # todo` reached the enum leg as
// `minor # todo` and `grounds: {}  # todo` reached the emptiness leg with its
// closing brace no longer last — defeating every blank test at once
// (iss-2608301744268001).
//
// framework 11.2: a disposition's required field is refused when it is "blank or
// whitespace only", which a value the reader cannot even delimit correctly
// cannot be judged against.
func TestFieldsStripsATrailingComment(t *testing.T) {
	for _, c := range []struct{ name, line, want string }{
		// The four blank spellings the record enumerates, each behind a comment.
		{"empty flow mapping", "grounds: {}  # todo", "{}"},
		{"empty flow sequence", "grounds: []  # todo", "[]"},
		{"tilde", "grounds: ~ # todo", "~"},
		{"double-quoted empty", `grounds: "" # todo`, `""`},

		// An ordinary scalar keeps its value and loses the comment.
		{"enum value", "severity: minor # todo", "minor"},
		{"comment with no space after the hash", "severity: minor #todo", "minor"},
		{"tab before the hash", "severity: minor\t# todo", "minor"},
		{"comment only", "severity: # todo", ""},
		{"several hashes", "severity: minor # a # b", "minor"},

		// A hash that is PART of the value. YAML starts a comment only at a `#`
		// preceded by whitespace and outside quotes, so each of these is content.
		{"hash with no preceding whitespace", "slug: a#b", "a#b"},
		{"url fragment", "ref: https://example.com/x#frag", "https://example.com/x#frag"},
		{"quoted url fragment", `ref: "https://example.com/x #frag"`, `"https://example.com/x #frag"`},
		{"hash inside double quotes", `title: "the # sign"`, `"the # sign"`},
		{"hash inside single quotes", "title: 'the # sign'", "'the # sign'"},
		{"quoted hash then a real comment", `title: "the # sign" # todo`, `"the # sign"`},
		{"escaped quote before a hash", `title: "a \" b # c"`, `"a \" b # c"`},
		{"doubled quote inside a single-quoted scalar", "title: 'it''s # here'", "'it''s # here'"},
		{"hash immediately after the colon", "slug:#notacomment", "#notacomment"},
		{"a lone hash as the value", "slug: #", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			key := c.line[:strings.IndexByte(c.line, ':')]
			fields := Fields([]string{"---", c.line, "---"})
			f, ok := fields[key]
			if !ok {
				t.Fatalf("%q yielded no %q field", c.line, key)
			}
			if f.Value != c.want {
				t.Errorf("Fields(%q)[%q] = %q, want %q", c.line, key, f.Value, c.want)
			}
		})
	}
}

// A comment line is not a key, and never was: the key pattern is anchored at
// column 0 on a name character, so a `# note: this` line inside the block is
// skipped rather than harvested as the key `note`. Pinned so the comment strip
// cannot be read as having introduced the behaviour.
func TestFieldsIgnoresAWholeLineComment(t *testing.T) {
	fields := Fields([]string{"---", "# note: not a key", "id: adr-9", "---"})
	if _, ok := fields["note"]; ok {
		t.Error("a whole-line comment was harvested as a field")
	}
	if fields["id"].Value != "adr-9" {
		t.Errorf("id = %q, want %q", fields["id"].Value, "adr-9")
	}
}
