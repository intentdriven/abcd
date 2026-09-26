package lint

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// A gitignored path is by definition not the repository's documentation. A
// fetch script that caches a clone of another repository under a lint root must
// not have that clone linted: its records link to paths that resolve only in
// the other checkout, and its folders carry no README (iss-2609151952353626).
// The per-file walk and directory_coverage prune what git ignores, a committed
// file a pattern matches is still read, and PrunedInRoots names what was pruned.
func TestLintPrunesGitignoredDirectoriesUnderARoot(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write(".gitignore", "docs/cache/\ndocs/kept/\n")
	repo.Write("docs/README.md", "# Docs\n\nsee [guide](guide.md)\n")
	repo.Write("docs/guide.md", "# Guide\n")
	repo.Write("docs/kept/README.md", "tracked though ignored: [gone](gone.md)\n")
	repo.Commit("docs")
	repo.Git("add", "-f", "docs/kept/README.md")
	repo.Commit("a committed file an ignore pattern matches")
	// The cached clone: untracked and ignored, with a broken link and a
	// README-less directory.
	repo.Write("docs/cache/clone/records/x.md", "[far](../../../elsewhere/y.md)\n")
	root := repo.Root()

	cfg := Config{
		Roots: []string{"docs"},
		Rules: map[string]RuleConfig{
			"links_resolve":      {Enabled: true, Severity: "blocker"},
			"directory_coverage": {Enabled: true, Severity: "blocker"},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		if strings.HasPrefix(filepath.ToSlash(f.File), "docs/cache") {
			t.Errorf("a finding from inside the ignored cache: %+v", f)
		}
	}
	if !hasFinding(fs, filepath.Join("docs", "kept", "README.md"), "links_resolve", 1) {
		t.Errorf("a committed file an ignore pattern matches lost its link check: %+v", fs)
	}
	pruned, err := PrunedInRoots(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"docs/cache/"}; !reflect.DeepEqual(pruned, want) {
		t.Errorf("PrunedInRoots = %q, want %q", pruned, want)
	}
	n, err := DocumentsInRoots(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("DocumentsInRoots = %d, want 3 (the ignored clone is not a document)", n)
	}
}
