package lint_test

import (
	"encoding/json"
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

// TestPersonaGlossaryScopesItsRuleToAModelledUser holds the persona glossary
// entry to the practice the intents follow (iss-2610262041388352): a persona
// is the voice of a modelled user, and a real role abcd addresses, the
// product thinker, may be quoted by role, unnamed, without one. Its "When to
// use" section says both.
func TestPersonaGlossaryScopesItsRuleToAModelledUser(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	body := readRepoFile(t, root, ".abcd/development/brief/glossary/core/persona.md")
	section := body
	if i := strings.Index(section, "## When to use"); i >= 0 {
		section = section[i:]
	} else {
		t.Fatalf("persona.md has no \"When to use\" section")
	}
	if i := strings.Index(section, "## When NOT to use"); i >= 0 {
		section = section[:i]
	}
	flat := strings.Join(strings.Fields(section), " ")
	for _, want := range []string{"modelled user", "product thinker", "by role, unnamed"} {
		if !strings.Contains(flat, want) {
			t.Errorf("persona.md's \"When to use\" does not say %q: the rule is scoped to a modelled user's voice, and a real role abcd addresses may be quoted by role, unnamed:\n%s", want, section)
		}
	}
}

// TestTheProductThinkerQuotesNameTheRegisteredPersona holds the press release
// and itd-106 to one name for the product-thinker role (iss-2610262031403463):
// the registry picks the name from the role, so each quote attributed to "a
// product thinker" names the persona personas.json gives that role, and the
// two quotes agree.
func TestTheProductThinkerQuotesNameTheRegisteredPersona(t *testing.T) {
	root := filepath.Join("..", "..", "..")

	var reg struct {
		Personas []struct {
			Name      string   `json:"name"`
			RoleHints []string `json:"role_hints"`
		} `json:"personas"`
	}
	if err := json.Unmarshal([]byte(readRepoFile(t, root, ".abcd/development/personas.json")), &reg); err != nil {
		t.Fatal(err)
	}
	var want string
	for _, p := range reg.Personas {
		for _, h := range p.RoleHints {
			if h == "product thinker" {
				want = p.Name
			}
		}
	}
	if want == "" {
		t.Fatal("personas.json gives no persona the \"product thinker\" role hint")
	}

	for _, rel := range []string{
		".abcd/development/brief/01-product/01-press-release.md",
		".abcd/development/intents/drafts/itd-106-abcd-sets-up-the-ci-a-repo-requires-and.md",
	} {
		found := false
		for i, line := range strings.Split(readRepoFile(t, root, rel), "\n") {
			if !strings.Contains(line, ", a product thinker") {
				continue
			}
			name, ok := lint.PersonaAttribution(line)
			if !ok {
				continue
			}
			found = true
			if name != want {
				t.Errorf("%s:%d attributes the product thinker's quote to %s; personas.json gives the role to %s", rel, i+1, name, want)
			}
		}
		if !found {
			t.Errorf("%s holds no quote attributed to a named product thinker", rel)
		}
	}
}

// TestProductThinkerGlossaryLetsAPersonaCarryTheRoleHint holds the
// product-thinker glossary entry to the registry (iss-2610262041032173):
// personas.json gives Iris "product thinker" among its role hints, so the
// entry's "When NOT to use" no longer calls a persona's role hint an outside
// job title. It says a persona such as Iris may carry the hint, while the
// persona stays a modelled archetype and not the role abcd addresses.
func TestProductThinkerGlossaryLetsAPersonaCarryTheRoleHint(t *testing.T) {
	root := filepath.Join("..", "..", "..")

	var reg struct {
		Personas []struct {
			Name      string   `json:"name"`
			RoleHints []string `json:"role_hints"`
		} `json:"personas"`
	}
	if err := json.Unmarshal([]byte(readRepoFile(t, root, ".abcd/development/personas.json")), &reg); err != nil {
		t.Fatal(err)
	}
	var holder string
	for _, p := range reg.Personas {
		for _, h := range p.RoleHints {
			if h == "product thinker" {
				holder = p.Name
			}
		}
	}
	if holder == "" {
		t.Fatal("personas.json gives no persona the \"product thinker\" role hint")
	}

	body := readRepoFile(t, root, ".abcd/development/brief/glossary/core/product-thinker.md")
	section := body
	if i := strings.Index(section, "## When NOT to use"); i >= 0 {
		section = section[i:]
	} else {
		t.Fatalf("product-thinker.md has no \"When NOT to use\" section")
	}
	if i := strings.Index(section, "## Examples"); i >= 0 {
		section = section[:i]
	}
	flat := strings.Join(strings.Fields(section), " ")
	if strings.Contains(flat, "outside job title") {
		t.Errorf("product-thinker.md's \"When NOT to use\" still calls a persona's role hint an outside job title, but personas.json gives %s the \"product thinker\" hint:\n%s", holder, section)
	}
	for _, want := range []string{
		"such as " + holder,
		"\"product thinker\" among its role hints",
		"personas.json",
		"modelled archetype",
		"not the role abcd addresses",
	} {
		if !strings.Contains(flat, want) {
			t.Errorf("product-thinker.md's \"When NOT to use\" does not say %q:\n%s", want, section)
		}
	}
}
