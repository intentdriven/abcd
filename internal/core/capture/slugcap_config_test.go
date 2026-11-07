package capture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const longSlugText = "A record's file name has no cap tied to the length of its full path on Windows"

func repoWithSlugCap(t *testing.T, cfg string) string {
	t.Helper()
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "config.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	return repo
}

// TestCaptureCutsTheSlugAtTheRepositorysCap is iss-2611052241331003: the capture
// verb reads records.slug_cap from the repository, whichever way the slug
// arrives, wider than the default as well as narrower.
func TestCaptureCutsTheSlugAtTheRepositorysCap(t *testing.T) {
	const given = "a-record-s-file-name-has-no-cap-tied-to-the-length-of-its-full-path-on-windows"
	for _, c := range []struct {
		name, cfg, slug, want string
	}{
		{"wider, derived", `{"records": {"slug_cap": 60}}`, "", "a-record-s-file-name-has-no-cap-tied-to-the-length-of-its"},
		{"wider, given", `{"records": {"slug_cap": 60}}`, given, "a-record-s-file-name-has-no-cap-tied-to-the-length-of-its"},
		{"narrower, derived", `{"records": {"slug_cap": 20}}`, "", "a-record-s-file-name"},
		{"narrower, given", `{"records": {"slug_cap": 20}}`, given, "a-record-s-file-name"},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo := repoWithSlugCap(t, c.cfg)
			res, err := testCapture(CaptureRequest{RepoRoot: repo, Text: longSlugText, Slug: c.slug,
				Severity: SeverityMinor, Category: "process", Source: "user-observation", FoundDuring: "unit-test"})
			if err != nil {
				t.Fatalf("Capture: %v", err)
			}
			if res.Slug != c.want {
				t.Fatalf("slug = %q, want %q", res.Slug, c.want)
			}
			if !strings.HasSuffix(filepath.Base(res.Path), "-"+c.want+".md") {
				t.Fatalf("filename %q does not carry %q", res.Path, c.want)
			}
		})
	}
}

// A cap the repository cannot have meant refuses the capture, naming the file
// and the bounds, and writes no record.
func TestCaptureRefusesANonsenseSlugCap(t *testing.T) {
	repo := repoWithSlugCap(t, `{"records": {"slug_cap": 0}}`)
	_, err := testCapture(CaptureRequest{RepoRoot: repo, Text: longSlugText,
		Severity: SeverityMinor, Category: "process", Source: "user-observation", FoundDuring: "unit-test"})
	if err == nil || !strings.Contains(err.Error(), "records.slug_cap") || !strings.Contains(err.Error(), ".abcd/config.json") {
		t.Fatalf("Capture with a cap of 0: %v; want a refusal naming the key and the file", err)
	}
	if entries, _ := os.ReadDir(filepath.Join(repo, ".abcd/work/issues/open")); len(entries) != 0 {
		t.Fatalf("a refused capture wrote %v", entries)
	}
}
