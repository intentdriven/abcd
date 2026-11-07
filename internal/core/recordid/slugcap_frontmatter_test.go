package recordid

import (
	"os"
	"path/filepath"
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
