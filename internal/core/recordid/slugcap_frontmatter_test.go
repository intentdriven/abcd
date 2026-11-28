package recordid

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The slug reader and the record reader must agree on where a block ends: an
// indented rule is body to the reader, so a slug after one is still the record's.
func TestAnIndentedRuleDoesNotCloseTheSlugsBlock(t *testing.T) {
	file := filepath.Join(t.TempDir(), "record.md")
	if err := os.WriteFile(file, []byte("---\n  ---\nslug: far-too-long\n---\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	slug, line, ok := frontmatterSlug(file)
	if !ok || slug != "far-too-long" || line != 3 {
		t.Fatalf("frontmatterSlug = %q, %d, %v; want the slug on line 3", slug, line, ok)
	}
}

// The slug reader reads the block as record-lint's scanner does, so the
// commit-message hook refuses what the lint refuses: a byte-order mark ahead of
// the opening rule, a key written with a space before its colon, and a slug
// below a line longer than any buffer the reader keeps are each the record's
// slug field, and so is one under an HTML comment above the block, which the
// lint admits (iss-2611070737092992).
func TestTheSlugReaderReadsTheBlockAsTheLintDoes(t *testing.T) {
	long := strings.Repeat("x", 1<<20+1)
	cases := []struct {
		name, text string
		line       int
	}{
		{"byte-order mark", "\ufeff---\nslug: far-too-long\n---\nbody\n", 2},
		{"spaced key", "---\nslug : far-too-long\n---\nbody\n", 2},
		{"line over 1 MiB", "---\ntitle: " + long + "\nslug: far-too-long\n---\nbody\n", 3},
		{"trailing comment", "---\nslug: far-too-long # a note\n---\nbody\n", 2},
		{"leading HTML comment", "<!-- attribution -->\n\n---\nslug: far-too-long\n---\nbody\n", 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "record.md")
			if err := os.WriteFile(file, []byte(c.text), 0o644); err != nil {
				t.Fatal(err)
			}
			slug, line, ok := frontmatterSlug(file)
			if !ok || slug != "far-too-long" || line != c.line {
				t.Fatalf("frontmatterSlug = %q, %d, %v; want %q on line %d", slug, line, ok, "far-too-long", c.line)
			}
		})
	}
}
