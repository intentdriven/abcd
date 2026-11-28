package recordid

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// The slug reader and the record reader must agree on where a block ends: an
// indented rule is body to the reader, so a slug after one is still the record's.
func TestAnIndentedRuleDoesNotCloseTheSlugsBlock(t *testing.T) {
	file := filepath.Join(t.TempDir(), "record.md")
	if err := os.WriteFile(file, []byte("---\n  ---\nslug: far-too-long\n---\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	slug, line, ok, err := frontmatterSlug(file)
	if err != nil || !ok || slug != "far-too-long" || line != 3 {
		t.Fatalf("frontmatterSlug = %q, %d, %v; want the slug on line 3", slug, line, ok)
	}
}

// The slug reader reads the block as record-lint's scanner does, so the
// commit-message hook refuses what the lint refuses: a byte-order mark ahead of
// the opening rule, a key written with a space before its colon, a trailing
// comment and a long line above the field are each read as the record's slug
// field, and so is one under an HTML comment above the block, which the lint
// admits (iss-2611070737092992). A record over the record cap is not read at
// all; TestARecordOverTheRecordCapIsRefusedByName holds that.
func TestTheSlugReaderReadsTheBlockAsTheLintDoes(t *testing.T) {
	long := strings.Repeat("x", issueschema.RecordReadLimit/2)
	cases := []struct {
		name, text string
		line       int
	}{
		{"byte-order mark", "\ufeff---\nslug: far-too-long\n---\nbody\n", 2},
		{"spaced key", "---\nslug : far-too-long\n---\nbody\n", 2},
		{"long line under the record cap", "---\ntitle: " + long + "\nslug: far-too-long\n---\nbody\n", 3},
		{"trailing comment", "---\nslug: far-too-long # a note\n---\nbody\n", 2},
		{"leading HTML comment", "<!-- attribution -->\n\n---\nslug: far-too-long\n---\nbody\n", 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "record.md")
			if err := os.WriteFile(file, []byte(c.text), 0o644); err != nil {
				t.Fatal(err)
			}
			slug, line, ok, err := frontmatterSlug(file)
			if err != nil || !ok || slug != "far-too-long" || line != c.line {
				t.Fatalf("frontmatterSlug = %q, %d, %v, %v; want %q on line %d", slug, line, ok, err, "far-too-long", c.line)
			}
		})
	}
}

// A record over the record cap is not read: the slug reader takes a record
// through the shared bounded read, as record-lint's record_schema does, and an
// over-cap record is named rather than passed, so a slug field below a line
// longer than the cap cannot slip the slug cap unseen.
func TestARecordOverTheRecordCapIsRefusedByName(t *testing.T) {
	root := t.TempDir()
	rel := ".abcd/work/decisions/dcn-2611280000000001-short.md"
	file := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\ntitle: " + strings.Repeat("x", issueschema.RecordReadLimit) + "\nslug: far-too-long\n---\nbody\n"
	if err := os.WriteFile(file, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got := SlugsOverCap(root, []string{rel}, 40)
	if len(got) != 1 || got[0].Rel != rel || !got[0].OverRecordCap || got[0].Where != "record" {
		t.Fatalf("SlugsOverCap = %+v; want the one record named as over the record cap", got)
	}
}
