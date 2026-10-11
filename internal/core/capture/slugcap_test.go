package capture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/recordid"
)

// TestCaptureCutsTheSlugAtTheRecordCap is iss-2610100626320367 at the capture
// verb: a slug derived from long text, and a long slug given explicitly, are
// both cut at recordid.MaxSlugLen on a hyphen, and the one value lands in the
// filename and the frontmatter alike.
func TestCaptureCutsTheSlugAtTheRecordCap(t *testing.T) {
	long := "A record's file name has no cap tied to the length of its full path on Windows"
	cases := []struct {
		name, slug, want string
	}{
		{"derived from the text", "", "a-record-s-file-name-has-no-cap-tied-to"},
		{"given explicitly", "a-record-s-file-name-has-no-cap-tied-to-the-length-of-its", "a-record-s-file-name-has-no-cap-tied-to"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo := t.TempDir()
			res, err := testCapture(CaptureRequest{
				RepoRoot: repo, Text: long, Slug: c.slug,
				Severity: SeverityMinor, Category: "process",
				Source: "user-observation", FoundDuring: "unit-test",
			})
			if err != nil {
				t.Fatalf("Capture: %v", err)
			}
			if res.Slug != c.want {
				t.Fatalf("slug = %q, want %q", res.Slug, c.want)
			}
			if len(res.Slug) > recordid.MaxSlugLen {
				t.Fatalf("slug %q is %d characters, over the %d cap", res.Slug, len(res.Slug), recordid.MaxSlugLen)
			}
			if !strings.HasSuffix(filepath.Base(res.Path), "-"+c.want+".md") {
				t.Fatalf("filename %q does not carry the capped slug %q", res.Path, c.want)
			}
			data, err := os.ReadFile(filepath.Join(repo, res.Path))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), "slug: \""+c.want+"\"\n") && !strings.Contains(string(data), "slug: "+c.want+"\n") {
				t.Fatalf("frontmatter does not carry the capped slug %q:\n%s", c.want, data)
			}
		})
	}
}
