package capture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestCaptureFoundAtMustResolveInTheTree is the guard for iss-2609120511058115:
// nine findings about abcd's own installer were filed into another repository's
// ledger, and nothing refused them because found_at is never checked against the
// tree. A value that names a repo-relative path must resolve in the checkout
// being written to; a conceptual location (anything that is not a lone path
// token) and an absent value stay legitimate. Both sides are asserted
// (guards-prove-themselves): the refusals write nothing, and the permitted
// values are written as given. The caller is a neutral one: a home-relative
// location names abcd's own store, which a caller whose account name is that
// word would read as its login and redact.
func TestCaptureFoundAtMustResolveInTheTree(t *testing.T) {
	gittest.NeutralCaller(t)
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
			_, err := testCapture(CaptureRequest{
				RepoRoot: repo, IssuesRoot: ir, Text: "a finding", Severity: SeverityMinor,
				Category: "bug", Source: "manual-test", Slug: "finding", FoundDuring: "t",
				FoundAt: tc.foundAt,
			})
			if err == nil {
				t.Fatalf("found_at %q names a path that is not in this checkout; the capture must be refused", tc.foundAt)
			}
			// The caller's input, nothing written: a refusal, not a fault
			// (iss-2609261241119343).
			if !errors.Is(err, ErrRequestRefused) {
				t.Errorf("found_at %q is refused as a fault, not as the request's input: %v", tc.foundAt, err)
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
		{"existing root file with no extension", "Makefile"},
		{"missing root name with no extension", "Dockerfile"},
		{"conceptual phrase", "git history"},
		{"conceptual single word", "conventions"},
		{"path with a parenthetical", "internal/core/lint (the <!-- marker scan)"},
		{"list of paths", "internal/a.go, internal/b.go"},
		{"home-relative location", abcdhome.Display("history")},
		{"url", "https://example.com/some/page"},
		{"version string", "v0.9.0"},
	}
	for _, tc := range written {
		t.Run("written/"+tc.name, func(t *testing.T) {
			repo, ir := ledger(t)
			if err := os.MkdirAll(filepath.Join(repo, "internal", "core"), 0o755); err != nil {
				t.Fatal(err)
			}
			for _, f := range []string{"internal/core/x.go", "README.md", "Makefile"} {
				if err := os.WriteFile(filepath.Join(repo, f), []byte("x\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			res, err := testCapture(CaptureRequest{
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

// TestCaptureFoundAtThatCannotBeCheckedIsAFault is the other half of
// iss-2609261241119343: a path the checkout will not let the check read (a
// directory on the way to it that cannot be searched) says nothing about the
// caller's input, so it is refused with nothing written but as a fault, never
// as ErrRequestRefused.
func TestCaptureFoundAtThatCannotBeCheckedIsAFault(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root searches a mode-000 directory")
	}
	repo, ir := ledger(t)
	locked := filepath.Join(repo, "internal", "locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	_, err := testCapture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "a finding", Severity: SeverityMinor,
		Category: "bug", Source: "manual-test", Slug: "finding", FoundDuring: "t",
		FoundAt: "internal/locked/x.go",
	})
	if err == nil || !strings.Contains(err.Error(), "could not be checked") {
		t.Fatalf("a found_at behind an unsearchable directory: %v, want the could-not-be-checked fault", err)
	}
	if errors.Is(err, ErrRequestRefused) {
		t.Errorf("a found_at that could not be checked is refused as the caller's input: %v", err)
	}
	if _, serr := os.Stat(ir); !os.IsNotExist(serr) {
		t.Fatalf("a refused capture touched the ledger root %s (stat err=%v)", ir, serr)
	}
}

// TestFoundAtPathInTheCheckout is iss-2611190625535337: a found_at naming a
// file at the checkout's root with no extension, such as Makefile, names a
// path, so the build and the drain reserve the issue with it. A lone name is
// taken as a path only when a regular file of exactly that name is at the
// root: a word that names no file, a name that is missing, a directory and a
// name differing in case all stay a place in words, and a path-shaped value
// keeps its verdict whether or not it is there.
func TestFoundAtPathInTheCheckout(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "Makefile"), []byte("all:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(repo, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, foundAt, want string
		ok                  bool
	}{
		{"extensionless root file", "Makefile", "Makefile", true},
		{"extensionless root file with a line locator", "Makefile:12", "Makefile", true},
		{"padded extensionless root file", "  Makefile ", "Makefile", true},
		{"a word that names no file", "conventions", "", false},
		{"a missing extensionless name", "Dockerfile", "", false},
		{"a root directory named alone", "scripts", "", false},
		{"a name differing in case from the file", "makefile", "", false},
		{"a phrase", "the Makefile", "", false},
		{"a path-shaped value that is missing", "internal/core/x.go", "internal/core/x.go", true},
		{"a root file with an extension that is missing", "README.md", "README.md", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := FoundAtPathIn(repo, tc.foundAt)
			if got != tc.want || ok != tc.ok {
				t.Errorf("FoundAtPathIn(%q) = %q, %v; want %q, %v", tc.foundAt, got, ok, tc.want, tc.ok)
			}
		})
	}
	// An unreadable root reads as no file there: the lone name stays a place
	// in words rather than failing the verdict.
	if got, ok := FoundAtPathIn(filepath.Join(repo, "absent"), "Makefile"); ok {
		t.Errorf("a lone name under a root that cannot be read is taken as the path %q", got)
	}
}
