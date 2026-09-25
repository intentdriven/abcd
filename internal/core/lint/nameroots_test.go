package lint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func nameToken() BannedToken {
	return BannedToken{
		ID: "names/secret-project", Pattern: `(?i)\bmoonbeam\b`, Message: "banned name",
		Severity: "blocker", Successor: "the project", AllowContext: []string{`docs-lint: allow`},
	}
}

// The name gate (the banned_tokens `names/` family, the public banlist layer)
// read only the docs roots, so the largest public surface — .abcd/**, the root
// prose files, scripts/ — was scanned by no name ban (iss-279). name_roots
// widens the `names/` family alone over those trees, reading every text file
// there (a script is not markdown), honouring exempt_paths, and never running
// the rest of the family (a present-tense or spelling token) outside the roots.
func TestNameRootsCarryTheNamesFamilyOnly(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/page.md", "# Page\n")
	writeFile(t, root, "AGENTS.md", "Moonbeam is the codename; previously it was not.\n")
	writeFile(t, root, "scripts/run.sh", "#!/bin/sh\necho moonbeam\n")
	writeFile(t, root, ".abcd/development/brief.md", "# Brief\n\nthe moonbeam plan\n")
	writeFile(t, root, ".abcd/work/reviews/old.md", "quoted: moonbeam\n")
	writeFile(t, root, ".abcd/blob.bin", "moonbeam\x00\x01")
	cfg := Config{
		Roots: []string{"docs"},
		BannedTokens: []BannedToken{nameToken(), {
			ID: "present_tense/previously", Pattern: `(?i)\bpreviously\b`, Message: "narration",
			Severity: "blocker", Successor: "present tense", AllowContext: []string{`docs-lint: allow`},
		}},
		NameRoots:   []string{".abcd", "AGENTS.md", "scripts"},
		ExemptPaths: []string{".abcd/work/reviews/"},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct {
		file string
		line int
	}{{"AGENTS.md", 1}, {filepath.Join("scripts", "run.sh"), 2}, {filepath.Join(".abcd", "development", "brief.md"), 3}} {
		if !hasFinding(fs, want.file, "names/secret-project", want.line) {
			t.Errorf("the name ban did not reach %s:%d: %+v", want.file, want.line, fs)
		}
	}
	if n := countRule(fs, "names/secret-project"); n != 3 {
		t.Errorf("want 3 name findings (exempt review and binary skipped), got %d: %+v", n, fs)
	}
	if n := countRule(fs, "present_tense/previously"); n != 0 {
		t.Errorf("a non-name token ran outside the roots: %+v", fs)
	}
}

// TestRepoNameRootsCoverThePublicSurface pins this repository's own coverage:
// the name gate reaches .abcd/**, the root prose files and scripts/ (iss-279).
func TestRepoNameRootsCoverThePublicSurface(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", ".abcd", "docs-lint.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	have := map[string]bool{}
	for _, r := range append(append([]string{}, cfg.Roots...), cfg.NameRoots...) {
		have[r] = true
	}
	for _, want := range []string{".abcd", "AGENTS.md", "CONTRIBUTING.md", "scripts", "README.md", "docs"} {
		if !have[want] {
			t.Errorf("the name gate does not reach %s (roots %q, name_roots %q)", want, cfg.Roots, cfg.NameRoots)
		}
	}
}
