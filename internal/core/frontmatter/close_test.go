package frontmatter

import (
	"strings"
	"testing"
)

// TestCloseFindsTheBlockFieldsReads: Close is the one answer to "where does the
// leading block close", on Fields' terms exactly — a BOM tolerated at line 0 and
// nowhere else, a delimiter by IsDelimiter, an unclosed block no block at all —
// so a sibling reader that asks it cannot disagree with the reader about where
// the body begins (iss-2608221126066379).
func TestCloseFindsTheBlockFieldsReads(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		doc  string
		want int
	}{
		{"plain", "---\nid: a\n---\nbody\n", 2},
		{"BOM ahead of the opening delimiter", "\ufeff---\nid: a\n---\nbody\n", 2},
		{"trailing whitespace and CRLF on the delimiters", "--- \r\nid: a\r\n---\t\r\nbody\r\n", 2},
		{"no opening delimiter", "id: a\n---\nbody\n", -1},
		{"unclosed", "---\nid: a\nbody\n", -1},
		{"a mid-file ZWNBSP line is not a close", "---\nid: a\n\ufeff---\nb: c\n---\n", 4},
		{"an indented rule is not a close", "---\nid: a\n  ---\n---\n", 3},
		{"empty", "", -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			lines := strings.Split(tc.doc, "\n")
			got := Close(lines)
			if got != tc.want {
				t.Fatalf("Close = %d, want %d", got, tc.want)
			}
			// Agreement with Fields: a block Close finds is one Fields reads.
			if fields := Fields(lines); (got >= 0) != (len(fields) > 0) {
				t.Fatalf("Close = %d but Fields read %d field(s)", got, len(fields))
			}
		})
	}
}
