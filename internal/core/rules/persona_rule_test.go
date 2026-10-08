package rules

import (
	"regexp"
	"strings"
	"testing"
)

// personaCap matches the wording that closes the persona roster: a rule that
// says the names are "always" or "only" a fixed set, or forbids "other names",
// forbids the fourth persona a repository needs.
var personaCap = regexp.MustCompile(`(?i)\balways\b|\bonly\b|other names|\bfixed\b`)

// TestNoBundledRuleCapsPersonaNames pins that the bundled defaults teach every
// managed repository an OPEN persona sequence it extends, never a fixed roster
// of three it can lift only by overriding the whole INTENTS rules field.
func TestNoBundledRuleCapsPersonaNames(t *testing.T) {
	rs := Defaults()
	var personaRules int
	for name, d := range rs.Domains {
		for _, r := range d.Rules {
			if !strings.Contains(strings.ToLower(r), "persona") {
				continue
			}
			personaRules++
			if m := personaCap.FindString(r); m != "" {
				t.Errorf("%s rule caps the persona names (%q): %s", name, m, r)
			}
		}
	}
	if personaRules == 0 {
		t.Fatal("no bundled rule teaches personas at all")
	}

	r := intentsPersonaRule(t, rs)
	// The sequence continues past the third name and says it goes on.
	for _, want := range []string{"Alice", "Bob", "Carol", "Dave", "and on", "extends"} {
		if !strings.Contains(r, want) {
			t.Errorf("INTENTS persona rule does not read as an open sequence (missing %q): %s", want, r)
		}
	}
}

// TestIntentsPersonaRuleNamesTheRepositoryRoster pins that the bundled rule
// points at the repository's own roster where one is declared, for names and
// roles alike, and names how a repository declares it: the registry file of
// its persona_registry lint rule, in the .abcd/docs-lint.json that ahoy seeds
// and `abcd lint docs` reads. record-lint is abcd's own development gate, which
// a managed repository neither has nor can run, so the rule never names it.
func TestIntentsPersonaRuleNamesTheRepositoryRoster(t *testing.T) {
	r := intentsPersonaRule(t, Defaults())
	for _, want := range []string{"roster", "persona_registry", "registry", "name and role", ".abcd/docs-lint.json"} {
		if !strings.Contains(r, want) {
			t.Errorf("INTENTS persona rule does not point at a declared roster (missing %q): %s", want, r)
		}
	}
	if strings.Contains(r, "record-lint") {
		t.Errorf("INTENTS persona rule names record-lint, abcd's own development gate, which a managed repository does not have: %s", r)
	}
}

func intentsPersonaRule(t *testing.T, rs RuleSet) string {
	t.Helper()
	d, ok := rs.Domains["INTENTS"]
	if !ok {
		t.Fatal("bundled defaults carry no INTENTS domain")
	}
	for _, r := range d.Rules {
		if strings.Contains(strings.ToLower(r), "persona") {
			return r
		}
	}
	t.Fatal("INTENTS carries no persona rule")
	return ""
}
