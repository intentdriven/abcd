package ahoy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/banlist"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/gittest"
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

// TestSeedRootsAreWhatExistsAtInstall (iss-2610040758095861): the seeded
// docs-lint config lists docs only when that folder exists and README.md only
// when that file does, so the documentation check setup has just armed runs
// rather than refusing over a root it named itself. With neither, roots is
// empty: the check runs, finds no document and says loudly that nothing was
// checked, at exit 0, which is the honest report for a repository with no
// documentation yet. In every case the check runs to completion (no refusal,
// the exit-2 case) and ahoy raises no dangling-root gap.
func TestSeedRootsAreWhatExistsAtInstall(t *testing.T) {
	for _, tc := range []struct {
		name      string
		docs      bool
		readme    bool
		wantRoots []string
	}{
		{name: "neither", wantRoots: []string{}},
		{name: "readme only", readme: true, wantRoots: []string{"README.md"}},
		{name: "docs only", docs: true, wantRoots: []string{"docs"}},
		{name: "both", docs: true, readme: true, wantRoots: []string{"docs", "README.md"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupHermetic(t)
			repo := gittest.NewRepo(t).Root()
			if tc.docs {
				if err := os.MkdirAll(filepath.Join(repo, "docs"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(repo, "docs", "guide.md"), []byte("# Guide\n\nRun the check.\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if tc.readme {
				if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# Example\n\nThe tool reports what it finds.\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(repo, filepath.FromSlash(banlist.PublicConfigRelPath))
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("no seeded docs-lint config: %v", err)
			}
			var seeded struct {
				Roots []string `json:"roots"`
			}
			if err := json.Unmarshal(raw, &seeded); err != nil {
				t.Fatalf("seeded config is not JSON: %v\n%s", err, raw)
			}
			if seeded.Roots == nil || !slices.Equal(seeded.Roots, tc.wantRoots) {
				t.Errorf("seeded roots = %#v, want %#v", seeded.Roots, tc.wantRoots)
			}
			cfg, err := lint.LoadConfig(path)
			if err != nil {
				t.Fatalf("seeded config does not load: %v", err)
			}
			if _, err := lint.Lint(cfg, repo); err != nil {
				t.Errorf("the seeded check refuses to run (abcd lint docs would exit 2): %v", err)
			}
			if gaps := detectDocsLintRoots(repo); len(gaps) != 0 {
				t.Errorf("a fresh install leaves a dangling docs-lint root: %+v", gaps)
			}
		})
	}
}
