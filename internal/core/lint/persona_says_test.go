package lint

import "testing"

// The persona-attribution checks read `says <Name>,` as well as `said <Name>,`:
// the release page's verbatim-quote check accepts both verbs, so a check that
// read only `said` let an unregistered persona quoted with `says` pass both the
// page's headline refusal and persona_registry (iss-2609231715081185).
func TestPersonaRegistryReadsSaysAsWellAsSaid(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/development/personas.json",
		`{"schema_version":2,"personas":[{"name":"Alice","role_hints":["solo founder"]}]}`)
	writeFile(t, root, "rec/says.md", "# says\n\n> \"Nope,\" says Zorro, pirate captain.\n")
	writeFile(t, root, "rec/ok.md", "# ok\n\n> \"Fine,\" says Alice, solo founder.\n")
	cfg := Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			"persona_registry": {Enabled: true, Severity: "blocker", Registry: ".abcd/development/personas.json"},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "persona_registry"); n != 1 || !hasFinding(fs, "rec/says.md", "persona_registry", 3) {
		t.Fatalf("want the one `says` attribution of an unregistered persona, got %d: %+v", n, fs)
	}
	if name, ok := PersonaAttribution("Nobody types a version, says Iris, a product thinker."); !ok || name != "Iris" {
		t.Errorf("PersonaAttribution missed `says`: %q %v", name, ok)
	}
}
