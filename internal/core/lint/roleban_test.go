package lint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// TestRepoRoleWordIsRefusedOnEveryLintRoot pins this repository's own ban
// (itd-2609212137129937 AC2): the one role-vocabulary entry in the docs lint's
// banned_tokens refuses the retired word on every root the spec names — the
// docs roots, the command pages, this repository's rules overrides and the
// bundled rules source — as a blocker, and the marked escape is the only way
// past it.
func TestRepoRoleWordIsRefusedOnEveryLintRoot(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", ".abcd", "docs-lint.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	word := "main" + "tainer" // spelled apart so this file never trips the ban it tests
	var ban *BannedToken
	for i, tok := range cfg.BannedTokens {
		if regexp.MustCompile(tok.Pattern).MatchString("a " + word + "'s ruling") {
			if ban != nil {
				t.Fatalf("two banned tokens match the role word (%s, %s); the list holds one", ban.ID, tok.ID)
			}
			ban = &cfg.BannedTokens[i]
		}
	}
	if ban == nil {
		t.Fatal("no docs-lint banned token refuses the role word")
	}
	if ban.Severity != "blocker" {
		t.Errorf("the role ban is %q, want blocker", ban.Severity)
	}
	for _, want := range []string{"commands", ".abcd/rules.json", "internal/core/rules/defaults"} {
		found := false
		for _, r := range ban.ExtraRoots {
			found = found || r == want
		}
		if !found {
			t.Errorf("the role ban does not reach %s (extra_roots %q)", want, ban.ExtraRoots)
		}
	}

	// Run the repository's own entry over a tree shaped like this one.
	root := t.TempDir()
	pages := map[string]string{
		"docs/page.md":     "Ask the " + word + ".\n",
		"README.md":        "The " + word + " decides.\n",
		"commands/verb.md": "the " + word + "'s move\n",
		".abcd/rules.json": "{\"r\": \"a " + word + " decision\"}\n",
		"internal/core/rules/defaults/rules.json": "{\"r\": \"a " + word + " decision\"}\n",
		"commands/credit.md":                      "the " + word + " of a tool we use <!-- docs-lint: allow -->\n",
	}
	for p, body := range pages {
		writeFile(t, root, p, body)
	}
	lintCfg := Config{Roots: []string{"docs", "README.md"}, BannedTokens: []BannedToken{*ban}}
	fs, err := Lint(lintCfg, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"docs/page.md", "README.md", "commands/verb.md", ".abcd/rules.json", "internal/core/rules/defaults/rules.json"} {
		if !hasFinding(fs, filepath.FromSlash(p), ban.ID, 1) {
			t.Errorf("the role word in %s was not refused: %+v", p, fs)
		}
	}
	if hasFinding(fs, filepath.FromSlash("commands/credit.md"), ban.ID, 1) {
		t.Errorf("the marked escape did not suppress the finding: %+v", fs)
	}
}
