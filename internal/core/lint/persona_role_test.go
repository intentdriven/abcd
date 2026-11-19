package lint_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
)

// TestAQuoteMayBeAttributedToARoleAlone holds itd-79 to the ruling that a
// press-release quote may be attributed to a nameless role
// (iss-2610262043159613, dcn-2610271112330536): a role with no name invents
// no name, so it does not offend the rule's reason. The discipline says so,
// and the persona_registry rule, armed
// against the real roster, leaves such a quote alone while it still flags an
// invented name.
func TestAQuoteMayBeAttributedToARoleAlone(t *testing.T) {
	root := filepath.Join("..", "..", "..")

	itd79 := readRepoFile(t, root, ".abcd/development/intents/disciplines/itd-79-persona-registry.md")
	rule := itd79
	if i := strings.Index(rule, "## Rule"); i >= 0 {
		rule = rule[i:]
	}
	if i := strings.Index(rule, "## Why"); i >= 0 {
		rule = rule[:i]
	}
	if !strings.Contains(rule, "a role alone") {
		t.Errorf("itd-79's Rule does not allow a quote attributed to a role alone; "+
			"dcn-2610271112330536 ruled that a nameless role is allowed:\n%s", rule)
	}

	scratch := t.TempDir()
	roster := readRepoFile(t, root, ".abcd/development/personas.json")
	for rel, body := range map[string]string{
		"personas.json":   roster,
		"rec/role.md":     "# role\n\n> \"It costs what it says,\" said an AI/agent researcher who runs cold readings.\n",
		"rec/named.md":    "# named\n\n> \"It costs what it says,\" said Maya, an AI/agent researcher.\n",
		"rec/invented.md": "# invented\n\n> \"It costs what it says,\" said Zorro, an AI/agent researcher.\n",
	} {
		p := filepath.Join(scratch, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	fs, err := lint.Lint(lint.Config{
		Roots: []string{"rec"},
		Rules: map[string]lint.RuleConfig{
			"persona_registry": {Enabled: true, Severity: "blocker", Registry: "personas.json"},
		},
	}, scratch)
	if err != nil {
		t.Fatal(err)
	}
	var flagged []string
	for _, f := range fs {
		if f.RuleID == "persona_registry" {
			flagged = append(flagged, f.File)
		}
	}
	if len(flagged) != 1 || flagged[0] != "rec/invented.md" {
		t.Errorf("persona_registry flagged %v, want only rec/invented.md: a role alone and a registered name both pass, an invented name does not", flagged)
	}
}
