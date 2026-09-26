package lint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

// TestNameRootsRefuseAFileTheyCannotExamine: a file the walk lists but cannot
// stat (under a directory that can be listed but not searched) was skipped
// silently, so the leak gate passed a file it never read (iss-2609252251320497).
// It fails loud instead, naming the file, as an unreadable file already does.
func TestNameRootsRefuseAFileTheyCannotExamine(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root searches any directory, so the unexaminable file cannot be built")
	}
	root := t.TempDir()
	writeFile(t, root, "docs/page.md", "# Page\n")
	writeFile(t, root, "scripts/locked/run.sh", "echo moonbeam\n")
	locked := filepath.Join(root, "scripts", "locked")
	if err := os.Chmod(locked, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	cfg := Config{Roots: []string{"docs"}, BannedTokens: []BannedToken{nameToken()}, NameRoots: []string{"scripts"}}
	fs, err := Lint(cfg, root)
	if err == nil {
		t.Fatalf("a file the name gate could not examine passed silently: %+v", fs)
	}
	if !strings.Contains(err.Error(), filepath.Join("scripts", "locked", "run.sh")) {
		t.Fatalf("the refusal does not name the file: %v", err)
	}
}

// TestNameBansReadInsideCodeFences: the names family is a leak gate over the
// whole public surface, and a fenced block is published as readily as prose, so
// a name ban reads inside fences by default wherever it runs, under roots and
// name_roots alike (iss-2609252251320133). The rest of the family keeps the
// documentation default (a fenced example is not prose), and a name ban that
// declares skip_code_fences: true keeps the declaration.
func TestNameBansReadInsideCodeFences(t *testing.T) {
	root := t.TempDir()
	fenced := "# Page\n\n```sh\necho moonbeam previously\n```\n"
	writeFile(t, root, "docs/page.md", fenced)
	writeFile(t, root, "AGENTS.md", fenced)
	skip := true
	declared := nameToken()
	declared.ID, declared.Pattern, declared.SkipCodeFences = "names/declared-skip", `(?i)\becho\b`, &skip
	cfg := Config{
		Roots: []string{"docs"},
		BannedTokens: []BannedToken{nameToken(), declared, {
			ID: "present_tense/previously", Pattern: `(?i)\bpreviously\b`, Message: "narration",
			Severity: "blocker", Successor: "present tense", AllowContext: []string{`docs-lint: allow`},
		}},
		NameRoots: []string{"AGENTS.md"},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{filepath.Join("docs", "page.md"), "AGENTS.md"} {
		if !hasFinding(fs, file, "names/secret-project", 4) {
			t.Errorf("a banned name inside a code fence in %s passed the name gate: %+v", file, fs)
		}
	}
	if n := countRule(fs, "present_tense/previously"); n != 0 {
		t.Errorf("a documentation token read inside a fence: %+v", fs)
	}
	if n := countRule(fs, "names/declared-skip"); n != 0 {
		t.Errorf("a name ban declaring skip_code_fences: true read inside a fence: %+v", fs)
	}
}
