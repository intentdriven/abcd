package lint

import (
	"path/filepath"
	"strings"
	"testing"
)

// A rule whose severity is off-enum would emit findings that serialize yet
// count toward no exit code — a clean exit beside a non-empty findings list,
// the shape the sibling engines (repolint.Evaluate, guard.Validate) fail
// closed on. The loader is the one place that can refuse it before a gate
// runs vacuously green.

func TestLoadConfigRefusesOffEnumRuleSeverity(t *testing.T) {
	path := writeConfig(t, `{
	  "roots": ["rec"],
	  "rules": {"links_resolve": {"enabled": true, "severity": "blocking"}}
	}`)
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("LoadConfig accepted an enabled rule with severity \"blocking\"; want rejection")
	}
	if !strings.Contains(err.Error(), "links_resolve") {
		t.Fatalf("rejection must name the offending rule, got: %v", err)
	}
}

func TestLoadConfigRefusesEnabledRuleWithNoSeverity(t *testing.T) {
	path := writeConfig(t, `{
	  "roots": ["rec"],
	  "rules": {"links_resolve": {"enabled": true}}
	}`)
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("LoadConfig accepted an enabled rule with no severity; want rejection")
	}
}

func TestLoadConfigRefusesOffEnumTokenSeverity(t *testing.T) {
	path := writeConfig(t, `{
	  "roots": ["rec"],
	  "banned_tokens": [
	    {"id":"t1","pattern":"foo","message":"no foo","severity":"Blocker","successor":"bar","allow_context":["ok"]}
	  ]
	}`)
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("LoadConfig accepted a banned token with severity \"Blocker\"; want rejection")
	}
}

func TestLoadConfigRefusesUnknownKeys(t *testing.T) {
	// A misspelt key silently zero-values the field it missed: "enabld" leaves
	// Enabled false (a rule disarmed), "severty" leaves Severity "" (findings
	// that count toward no exit). Strict decoding of the rule and token
	// objects turns both into a refusal.
	path := writeConfig(t, `{
	  "roots": ["rec"],
	  "rules": {"links_resolve": {"enabld": true, "severity": "blocker"}}
	}`)
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("LoadConfig accepted an unknown config key (\"enabld\"); want rejection")
	}
	tok := writeConfig(t, `{
	  "roots": ["rec"],
	  "banned_tokens": [
	    {"id":"t1","pattern":"foo","message":"no","severty":"blocker","severity":"blocker","successor":"bar","allow_context":["ok"]}
	  ]
	}`)
	if _, err := LoadConfig(tok); err == nil {
		t.Fatal("LoadConfig accepted an unknown banned-token key (\"severty\"); want rejection")
	}
}

func TestLoadConfigAcceptsTopLevelAnnotationKey(t *testing.T) {
	// The top level stays lenient: an annotation key is the JSON commentary
	// convention, and the banlist editor pins that such a config still loads.
	path := writeConfig(t, `{
	  "note": "banned_tokens",
	  "roots": ["rec"],
	  "rules": {"links_resolve": {"enabled": true, "severity": "blocker"}}
	}`)
	if _, err := LoadConfig(path); err != nil {
		t.Fatalf("LoadConfig refused a top-level annotation key: %v", err)
	}
}

func TestLoadConfigAcceptsDisabledRuleWithoutSeverity(t *testing.T) {
	// A disabled rule is inert; its severity is not consulted, so absence there
	// is not a fault.
	path := writeConfig(t, `{
	  "roots": ["rec"],
	  "rules": {"links_resolve": {"enabled": false}}
	}`)
	if _, err := LoadConfig(path); err != nil {
		t.Fatalf("LoadConfig refused a disabled rule with no severity: %v", err)
	}
}

func TestLoadConfigAcceptsBothLiveSeverities(t *testing.T) {
	path := writeConfig(t, `{
	  "roots": ["rec"],
	  "rules": {
	    "links_resolve": {"enabled": true, "severity": "blocker"},
	    "no_brittle_line_refs": {"enabled": true, "severity": "warn"}
	  }
	}`)
	if _, err := LoadConfig(path); err != nil {
		t.Fatalf("LoadConfig refused the live severity vocabulary: %v", err)
	}
}

// TestLoadConfigRefusesUnknownRuleName is the rule-name half of the misspelt-key
// refusal above. A rule keyed "links_reslove" decodes cleanly, reads as armed, and
// is counted by ArmedChecks, yet no check runs under that name: the lint reports
// its findings over a tree it never looked at, the false green a checks count
// exists to remove. The loader refuses the name and says which it is, whether the
// entry is enabled or not, because a disabled misspelling is still a rule the
// author believes they can switch on.
func TestLoadConfigRefusesUnknownRuleName(t *testing.T) {
	for _, enabled := range []string{"true", "false"} {
		path := writeConfig(t, `{
	  "roots": ["rec"],
	  "rules": {"links_reslove": {"enabled": `+enabled+`, "severity": "blocker"}}
	}`)
		_, err := LoadConfig(path)
		if err == nil {
			t.Fatalf("LoadConfig accepted the unknown rule name \"links_reslove\" (enabled=%s); want rejection", enabled)
		}
		if !strings.Contains(err.Error(), "links_reslove") || !strings.Contains(err.Error(), "links_resolve") {
			t.Fatalf("rejection must name the unknown rule and list the known ones, got: %v", err)
		}
	}
}

// TestKnownRulesCoverTheCommittedConfigs holds the known-rule set to the two
// configurations this repository runs, so a rule the gates rely on can never be
// refused as unknown: both committed configs load, and every rule they name is
// known.
func TestKnownRulesCoverTheCommittedConfigs(t *testing.T) {
	for _, rel := range []string{".abcd/record-lint.json", ".abcd/docs-lint.json"} {
		cfg, err := LoadConfig(filepath.Join("..", "..", "..", filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("%s does not load: %v", rel, err)
		}
		for name := range cfg.Rules {
			if !knownRules[name] {
				t.Errorf("%s names rule %s, which knownRules does not carry", rel, name)
			}
		}
	}
}
