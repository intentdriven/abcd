package capture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCaptureFoundAtMustResolveInTheTree is the guard for iss-2609120511058115:
// nine findings about abcd's own installer were filed into another repository's
// ledger, and nothing refused them because found_at is never checked against the
// tree. A value that names a repo-relative path must resolve in the checkout
// being written to; a conceptual location (anything that is not a lone path
// token) and an absent value stay legitimate. Both sides are asserted
// (guards-prove-themselves): the refusals write nothing, and the permitted
// values are written as given.
func TestCaptureFoundAtMustResolveInTheTree(t *testing.T) {
	refused := []struct{ name, foundAt string }{
		{"missing file", "internal/core/ahoy/install.go"},
		{"missing file with a line locator", "hooks/bootstrap.sh:12"},
		{"missing file with a line range", "README.md:66-120"},
		{"missing file with a symbol locator", "internal/scanner.go:scanAllPatterns"},
		{"missing directory", "internal/core/ahoy"},
		{"missing directory with a trailing slash", "internal/core/ahoy/"},
		{"missing root file", "CHANGELOG.md"},
		{"missing dot-directory path", ".github/workflows/ci.yml"},
		{"escapes the checkout", "../elsewhere/notes.md"},
		{"padded path", "  internal/core/ahoy/install.go  "},
	}
	for _, tc := range refused {
		t.Run("refused/"+tc.name, func(t *testing.T) {
			repo, ir := ledger(t)
			_, err := Capture(CaptureRequest{
				RepoRoot: repo, IssuesRoot: ir, Text: "a finding", Severity: SeverityMinor,
				Category: "bug", Source: "manual-test", Slug: "finding", FoundDuring: "t",
				FoundAt: tc.foundAt,
			})
			if err == nil {
				t.Fatalf("found_at %q names a path that is not in this checkout; the capture must be refused", tc.foundAt)
			}
			msg := err.Error()
			for _, want := range []string{"found_at", strings.TrimSpace(tc.foundAt), "nothing written"} {
				if !strings.Contains(msg, want) {
					t.Errorf("refusal must contain %q, got: %v", want, err)
				}
			}
			// Nothing written: no record, and not even the ledger directories.
			if _, serr := os.Stat(ir); !os.IsNotExist(serr) {
				t.Fatalf("a refused capture touched the ledger root %s (stat err=%v)", ir, serr)
			}
		})
	}

	written := []struct{ name, foundAt string }{
		{"absent", ""},
		{"existing file", "internal/core/x.go"},
		{"existing file with a line locator", "internal/core/x.go:103"},
		{"existing file with a line list", "internal/core/x.go:72,92"},
		{"existing directory", "internal/core"},
		{"existing directory with a trailing slash", "internal/core/"},
		{"existing root file", "README.md"},
		{"conceptual phrase", "git history"},
		{"conceptual single word", "conventions"},
		{"path with a parenthetical", "internal/core/lint (the <!-- marker scan)"},
		{"list of paths", "internal/a.go, internal/b.go"},
		{"home-relative location", "~/.abcd/history"},
		{"url", "https://example.com/some/page"},
		{"version string", "v0.9.0"},
	}
	for _, tc := range written {
		t.Run("written/"+tc.name, func(t *testing.T) {
			repo, ir := ledger(t)
			if err := os.MkdirAll(filepath.Join(repo, "internal", "core"), 0o755); err != nil {
				t.Fatal(err)
			}
			for _, f := range []string{"internal/core/x.go", "README.md"} {
				if err := os.WriteFile(filepath.Join(repo, f), []byte("x\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			res, err := Capture(CaptureRequest{
				RepoRoot: repo, IssuesRoot: ir, Text: "a finding", Severity: SeverityMinor,
				Category: "bug", Source: "manual-test", Slug: "finding", FoundDuring: "t",
				FoundAt: tc.foundAt,
			})
			if err != nil {
				t.Fatalf("found_at %q must be written as it is today, got: %v", tc.foundAt, err)
			}
			lr, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateOpen})
			if err != nil {
				t.Fatal(err)
			}
			if len(lr.Issues) != 1 || lr.Issues[0].ID != res.ID {
				t.Fatalf("want the one captured issue back, got %+v (skipped=%v)", lr.Issues, lr.Skipped)
			}
			if got := lr.Issues[0].FoundAt; got != tc.foundAt {
				t.Errorf("found_at read back as %q, want %q", got, tc.foundAt)
			}
		})
	}
}
