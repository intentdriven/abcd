package lint

import (
	"path/filepath"
	"strings"
	"testing"
)

func roleToken(extra ...string) BannedToken {
	return BannedToken{
		ID: "roles/test", Pattern: `(?i)\bgatekeeper`, Message: "names neither role",
		Severity: "blocker", Successor: "the product thinker or the technical facilitator",
		AllowContext: []string{`(?i)<!--\s*docs-lint:\s*allow\b`}, ExtraRoots: extra,
	}
}

// A banned token's extra_roots widen THAT token alone beyond the configuration's
// roots (itd-2609212137129937): a role word is refused in the command pages and
// the rules files, which are not documentation, without arming the rest of the
// family (a spelling or present-tense token) there. Every text file under an
// extra root is read, so a JSON rules file is checked as a page is; the escape,
// fenced code and exempt_paths behave as they do under roots; and a file both
// walks reach is reported once.
func TestTokenExtraRootsCarryThatTokenOnly(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/page.md", "# Page\n\nask the gatekeeper\n")
	writeFile(t, root, "commands/verb.md", "# Verb\n\nthe gatekeeper decides; previously not.\n\n```\ngatekeeper in a fence\n```\n")
	writeFile(t, root, "commands/escaped.md", "the gatekeeper <!-- docs-lint: allow -->\n")
	writeFile(t, root, "commands/old/history.md", "the gatekeeper said so\n")
	writeFile(t, root, "rules/rules.json", "{\n  \"rule\": \"a gatekeeper decision\"\n}\n")
	cfg := Config{
		Roots: []string{"docs"},
		BannedTokens: []BannedToken{roleToken("commands", "rules/rules.json", "docs"), {
			ID: "present_tense/previously", Pattern: `(?i)\bpreviously\b`, Message: "narration",
			Severity: "blocker", Successor: "present tense", AllowContext: []string{`docs-lint: allow`},
		}},
		ExemptPaths: []string{filepath.Join("commands", "old") + string(filepath.Separator)},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct {
		file string
		line int
	}{{filepath.Join("docs", "page.md"), 3}, {filepath.Join("commands", "verb.md"), 3}, {filepath.Join("rules", "rules.json"), 2}} {
		if !hasFinding(fs, want.file, "roles/test", want.line) {
			t.Errorf("the role token did not reach %s:%d: %+v", want.file, want.line, fs)
		}
	}
	if n := countRule(fs, "roles/test"); n != 3 {
		t.Errorf("want 3 role findings (fence, escape and exempt path skipped, docs reported once), got %d: %+v", n, fs)
	}
	if n := countRule(fs, "present_tense/previously"); n != 0 {
		t.Errorf("another token ran over the role token's extra roots: %+v", fs)
	}
}

// An extra root that does not resolve would disarm the token for that tree while
// the lint reported clean, so it is a configuration error, as a missing root is;
// an extra root reaching outside the repository is refused before it is read.
func TestTokenExtraRootsRefuseAMissingOrEscapingRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/page.md", "# Page\n")
	for _, extra := range []string{"commands", "../outside"} {
		cfg := Config{Roots: []string{"docs"}, BannedTokens: []BannedToken{roleToken(extra)}}
		if _, err := Lint(cfg, root); err == nil || !strings.Contains(err.Error(), "extra_roots") {
			t.Errorf("extra root %q: want a configuration error naming extra_roots, got %v", extra, err)
		}
	}
}

// The configuration decoder is strict, so the key must load from JSON.
func TestTokenExtraRootsLoadFromConfig(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "commands/verb.md", "the gatekeeper\n")
	writeFile(t, root, "docs/page.md", "# Page\n")
	writeFile(t, root, "cfg.json", `{"roots":["docs"],"banned_tokens":[{"id":"roles/test","pattern":"(?i)\\bgatekeeper","message":"m","severity":"blocker","successor":"s","allow_context":["x"],"extra_roots":["commands"]}],"rules":{}}`)
	cfg, err := LoadConfig(filepath.Join(root, "cfg.json"))
	if err != nil {
		t.Fatal(err)
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if !hasFinding(fs, filepath.Join("commands", "verb.md"), "roles/test", 1) {
		t.Errorf("extra_roots from the config file did not reach the command page: %+v", fs)
	}
}

// A file named in roots that is not markdown was walked, kept by the markdown
// filter as nothing, and reported clean: a ban meant to reach a rules file read
// zero lines of it and said so with a green. roots holds markdown; a
// non-markdown file there is a configuration error that points at extra_roots.
func TestRootsRefuseANonMarkdownFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/page.md", "# Page\n")
	writeFile(t, root, "rules.json", "{\"r\": \"a gatekeeper decision\"}\n")
	cfg := Config{Roots: []string{"docs", "rules.json"}, BannedTokens: []BannedToken{roleToken()}}
	if _, err := Lint(cfg, root); err == nil || !strings.Contains(err.Error(), "extra_roots") {
		t.Errorf("want a configuration error naming the non-markdown root and extra_roots, got %v", err)
	}
	if _, err := DocumentsInRoots(cfg, root); err == nil {
		t.Error("DocumentsInRoots counted a non-markdown root as zero documents instead of refusing it")
	}
}
