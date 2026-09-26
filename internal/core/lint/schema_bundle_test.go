package lint

import (
	"testing"
)

// Criterion 5, the bundle half: a bundle-member names a bundle at least one
// other record names, or carries the history line saying the bundle now has one
// member; a bundle-member whose bundle nobody else names is refused naming the
// record (itd-34).
func TestRecordSchemaBundleMemberNamesABundleOthersName(t *testing.T) {
	intents := "rec/intents"
	member := func(id, bundle, extra string) string {
		return "---\nid: " + id + "\nkind: bundle-member\nbundle: " + bundle + "\nspec_id: spc-5\n" + extra + "---\n# " + id + "\n"
	}

	t.Run("a bundle nobody else names", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, intents+"/planned/itd-20-a.md", member("itd-20", "lonely", ""))
		fs, err := Lint(schemaConfig(), root)
		if err != nil {
			t.Fatal(err)
		}
		if !findingWith(fs, intents+"/planned/itd-20-a.md", ruleRecordSchema, "'lonely'") {
			t.Fatalf("a bundle-member naming a bundle no other record names must be refused: %+v", fs)
		}
	})

	t.Run("two members name one bundle", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, intents+"/planned/itd-20-a.md", member("itd-20", "pair", ""))
		writeFile(t, root, intents+"/planned/itd-21-b.md", member("itd-21", "pair", ""))
		fs, err := Lint(schemaConfig(), root)
		if err != nil {
			t.Fatal(err)
		}
		if n := countRule(fs, ruleRecordSchema); n != 0 {
			t.Fatalf("a bundle two records name is well-formed: %+v", fs)
		}
	})

	t.Run("a survivor says its bundle has one member", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, intents+"/planned/itd-21-b.md", member("itd-21", "pair",
			"reclassification_history:\n  - { date: 2026-09-26, from: bundle-member, to: bundle-member, reason: \"bundle pair now has one member: itd-20 was superseded by itd-30\" }\n"))
		fs, err := Lint(schemaConfig(), root)
		if err != nil {
			t.Fatal(err)
		}
		if n := countRule(fs, ruleRecordSchema); n != 0 {
			t.Fatalf("a survivor whose history declares a bundle of one is well-formed: %+v", fs)
		}
	})

	t.Run("a history line for another bundle does not count", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, intents+"/planned/itd-21-b.md", member("itd-21", "pair",
			"reclassification_history:\n  - { date: 2026-09-26, from: bundle-member, to: bundle-member, reason: \"bundle pairs now has one member\" }\n"))
		fs, err := Lint(schemaConfig(), root)
		if err != nil {
			t.Fatal(err)
		}
		if !findingWith(fs, intents+"/planned/itd-21-b.md", ruleRecordSchema, "'pair'") {
			t.Fatalf("only a line naming THIS bundle declares it a bundle of one: %+v", fs)
		}
	})
}

// Criterion 5, the kind half: a planned intent whose kind does not match its
// shelf is refused naming the record. intent_lifecycle holds the kind every
// bucket admits, so this pins that rule against the criterion rather than
// adding a second finding for the same line to record_schema.
func TestIntentLifecycleRefusesAKindThatDoesNotMatchItsShelf(t *testing.T) {
	for _, tc := range []struct{ bucket, kind string }{
		{"planned", "discipline"},
		{"shipped", "discipline"},
		{"disciplines", "standalone"},
	} {
		root := t.TempDir()
		rel := "rec/intents/" + tc.bucket + "/itd-20-a.md"
		spec := "spc-5"
		if tc.bucket == "disciplines" {
			spec = "null"
		}
		writeFile(t, root, rel, "---\nid: itd-20\nkind: "+tc.kind+"\nspec_id: "+spec+"\n---\n# a\n")
		cfg := Config{Roots: []string{"rec"}, Rules: map[string]RuleConfig{
			"intent_lifecycle": {Enabled: true, Severity: severityBlocker, IntentsDir: "intents"},
		}}
		fs, err := Lint(cfg, root)
		if err != nil {
			t.Fatal(err)
		}
		if !findingWith(fs, rel, "intent_lifecycle", "kind must be") {
			t.Errorf("%s with kind %s must be refused naming the record: %+v", tc.bucket, tc.kind, fs)
		}
	}
}
