package cli

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestDocsLintNamesTheGitignoredPathsItPruned: `abcd lint docs` prunes what git
// ignores under a root (a cached clone is not the repository's documentation),
// and says so in both renders rather than reading as a smaller tree
// (iss-2609151952353626).
func TestDocsLintNamesTheGitignoredPathsItPruned(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write(".gitignore", "docs/cache/\n")
	repo.Write(".abcd/docs-lint.json", `{"roots": ["docs"], "rules": {"links_resolve": {"enabled": true, "severity": "blocker"}}}`)
	repo.Write("docs/page.md", "# Page\n")
	repo.Commit("docs")
	repo.Write("docs/cache/clone/x.md", "[far](../../elsewhere/y.md)\n")
	t.Chdir(repo.Root())

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"lint", "docs"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d: the ignored clone's broken link was read\n%s%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "skipped 1 gitignored path(s) under the roots: docs/cache/") {
		t.Errorf("the text render must name what it pruned, got:\n%s", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"lint", "docs", "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("--json exit %d", code)
	}
	var res struct {
		Pruned []string `json:"pruned"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		t.Fatalf("--json output does not parse: %v\n%s", err, stdout.String())
	}
	if want := []string{"docs/cache/"}; !reflect.DeepEqual(res.Pruned, want) {
		t.Errorf("--json pruned = %q, want %q", res.Pruned, want)
	}
}
