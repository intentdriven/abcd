package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeDocsLintRoots writes repo's .abcd/docs-lint.json with roots and one
// rule armed, so it loads.
func writeDocsLintRoots(t *testing.T, repo string, roots ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"roots": ["` + strings.Join(roots, `", "`) + `"], ` +
		`"rules": {"links_resolve": {"enabled": true, "severity": "blocker"}}}`
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "docs-lint.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestDetectReportsADocsLintRootThatDoesNotResolve (iss-2610100649479892):
// CLAUDE.md retired, by setup or by hand, while .abcd/docs-lint.json still
// names it in roots. The documentation check refuses to run there, so ahoy
// and its doctor name it: one report-only gap per dangling root, whose fix
// names the file to edit and the entry to take out.
func TestDetectReportsADocsLintRootThatDoesNotResolve(t *testing.T) {
	setupHermetic(t)
	repo := committedRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# readme\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeDocsLintRoots(t, repo, "CLAUDE.md", "README.md")

	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	var got []Gap
	for _, g := range det.Gaps {
		if g.ID == DocsLintRootMissingGapID {
			got = append(got, g)
		}
	}
	if len(got) != 1 {
		t.Fatalf("want one %s gap, for CLAUDE.md alone; got %+v", DocsLintRootMissingGapID, got)
	}
	g := got[0]
	if g.Resolvable {
		t.Error("the gap is resolvable, but install never edits the person's docs-lint config")
	}
	for _, want := range []string{".abcd/docs-lint.json", `"CLAUDE.md"`} {
		if !strings.Contains(g.Title, want) || !strings.Contains(g.FixHint, want) {
			t.Errorf("title and fix hint must both name %s:\ntitle: %s\nfix: %s", want, g.Title, g.FixHint)
		}
	}
	if strings.Contains(g.Title+g.FixHint, "README.md") {
		t.Errorf("a root that resolves was named: %+v", g)
	}
}

// Roots that all resolve, and a repository with no docs-lint config, raise
// nothing.
func TestDetectQuietWhenDocsLintRootsResolve(t *testing.T) {
	setupHermetic(t)
	repo := committedRepo(t)
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if hasGap(det.Gaps, DocsLintRootMissingGapID) {
		t.Fatalf("no docs-lint config, yet %s was raised", DocsLintRootMissingGapID)
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# readme\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeDocsLintRoots(t, repo, "README.md", "")
	if det, err = Detect(repo); err != nil {
		t.Fatal(err)
	}
	if hasGap(det.Gaps, DocsLintRootMissingGapID) {
		t.Fatalf("every root resolves, yet %s was raised: %+v", DocsLintRootMissingGapID, det.Gaps)
	}
}
