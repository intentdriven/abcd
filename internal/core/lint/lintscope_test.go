package lint_test

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// linkScopeExempt names every committed markdown path no links_resolve walk
// reads, each with the reason. A key ending in "/" names a tree.
var linkScopeExempt = map[string]string{
	// Fixture repositories the evals read as input: a planted link that does
	// not resolve is part of what some of them test.
	"evals/testdata/": "eval fixtures, whose content is test input",
	// A block ahoy writes into another repository's CLAUDE.md; its links
	// resolve there, not here.
	"internal/core/ahoy/defaults/": "templates written into other repositories",
	// A managed project's conventions file as setup planted it before the home
	// was renamed, which the ahoy tests read as input.
	"internal/core/ahoy/testdata/": "another repository's file, read as test input",
}

// TestEveryCommittedMarkdownFileHasItsLinksChecked is the link half of iss-46's
// scope matrix: every committed markdown file sits under a tree some
// links_resolve walk reads — record-lint's roots or extra roots, or docs-lint's —
// or is named in linkScopeExempt with its reason. The roster is git's own list
// of committed files, so a markdown file added anywhere joins by existing, and
// a relative link in it that breaks is refused instead of shipping, which is
// what the root prose, the agent prompts and the plugin command pages did
// before the docs-lint walk reached them.
func TestEveryCommittedMarkdownFileHasItsLinksChecked(t *testing.T) {
	root := filepath.Join("..", "..", "..")

	var scope []string
	for _, rel := range []string{".abcd/record-lint.json", ".abcd/docs-lint.json"} {
		var cfg struct {
			Roots []string `json:"roots"`
			Rules map[string]struct {
				Enabled    bool     `json:"enabled"`
				ExtraRoots []string `json:"extra_roots"`
			} `json:"rules"`
		}
		if err := json.Unmarshal([]byte(readRepoFile(t, root, rel)), &cfg); err != nil {
			t.Fatalf("decoding %s: %v", rel, err)
		}
		links, ok := cfg.Rules["links_resolve"]
		if !ok || !links.Enabled {
			t.Fatalf("%s does not enable links_resolve; this test reads the scope that rule walks", rel)
		}
		scope = append(scope, cfg.Roots...)
		scope = append(scope, links.ExtraRoots...)
	}

	cmd := exec.Command("git", "-C", root, "ls-files", "-z", "--", "*.md")
	cmd.Env = gittest.Env(t)
	out, err := cmd.Output()
	if err != nil {
		t.Skipf("git ls-files unavailable in %s: %v", root, err)
	}
	files := strings.Split(strings.TrimRight(string(out), "\x00"), "\x00")
	if len(files) < 100 {
		t.Fatalf("git listed %d committed markdown files; the listing or the pathspec changed shape", len(files))
	}

	within := func(rel string, set []string) bool {
		for _, s := range set {
			s = strings.TrimSuffix(s, "/")
			if rel == s || strings.HasPrefix(rel, s+"/") {
				return true
			}
		}
		return false
	}
	var exempt []string
	for k := range linkScopeExempt {
		exempt = append(exempt, k)
	}

	used := map[string]bool{}
	for _, rel := range files {
		if rel == "" || within(rel, scope) {
			continue
		}
		matched := false
		for _, k := range exempt {
			if within(rel, []string{k}) {
				used[k], matched = true, true
			}
		}
		if !matched {
			t.Errorf("%s is committed markdown that no links_resolve walk reads.\n\n"+
				"Add its tree to docs-lint's links_resolve extra_roots (.abcd/docs-lint.json), or name "+
				"it in linkScopeExempt with the reason its links are not this repository's to check.", rel)
		}
	}
	for k, why := range linkScopeExempt {
		if !used[k] {
			t.Errorf("linkScopeExempt names %s (%s), which matches no committed markdown file; "+
				"a stale exemption would pre-approve an unchecked file nobody has ruled on", k, why)
		}
	}
}

// TestDocsLintArmsThePersonaRule is the persona half of iss-46: the
// persona_registry rule (a quote attributed to a persona outside the roster)
// ran over the design record alone, while docs/ is where the user-facing
// persona prose lives. docs-lint arms it against the same roster record-lint
// reads, at the same severity, so the two walks cannot disagree about who a
// persona may be.
func TestDocsLintArmsThePersonaRule(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	type persona struct {
		Enabled  bool   `json:"enabled"`
		Severity string `json:"severity"`
		Registry string `json:"registry"`
	}
	read := func(rel string) (persona, bool) {
		var cfg struct {
			Rules map[string]persona `json:"rules"`
		}
		if err := json.Unmarshal([]byte(readRepoFile(t, root, rel)), &cfg); err != nil {
			t.Fatalf("decoding %s: %v", rel, err)
		}
		p, ok := cfg.Rules["persona_registry"]
		return p, ok
	}
	record, ok := read(".abcd/record-lint.json")
	if !ok || !record.Enabled {
		t.Fatal(".abcd/record-lint.json does not arm persona_registry; this test holds docs-lint to it")
	}
	docs, ok := read(".abcd/docs-lint.json")
	if !ok || docs != record {
		t.Fatalf(".abcd/docs-lint.json arms persona_registry as %+v (present=%v), want record-lint's %+v: "+
			"docs/ carries persona prose, and a walk that never reads it passes a persona the roster "+
			"does not hold", docs, ok, record)
	}
}
