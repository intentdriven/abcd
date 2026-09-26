package lint

import (
	"path/filepath"
	"strings"
	"testing"
)

// familiesPage is a minimal record-families page: the rule reads its table's
// first column as the closed set of families, exactly as it reads the live page.
const familiesPage = `---
term: record-families
bounded_context: core
not_to_be_confused_with: null
status: stable
---
# record-families

| Family | What it is | Groups | Grouped by | Lifecycle | Moved by |
|---|---|---|---|---|---|
| **intent** | one capability | its specs | a bundle | drafts → planned → shipped | ` + "`intent plan`" + ` |
| **spec** | the design record | its steps | its intent | open → closed | ` + "`spec close`" + ` |
| **step** | one landable piece | nothing | its spec | listed, landed | ` + "`abcd build`" + ` |
| **bundle** | intents that ship as one | intents | nothing | named at plan | ` + "`intent plan --bundle`" + ` |
| **issue** | a captured finding | nothing | nothing | open → resolved | ` + "`capture`" + ` |
| **release** | the derived cut | what shipped | nothing | cut, tagged | ` + "`launch ship`" + ` |
| **status** | Now / Next / Later | nothing | nothing | none | nothing |

**Retired**: phase and milestone.
`

// familyEntry writes one glossary term file whose not_to_be_confused_with line
// is given verbatim (an empty ntbcw omits the key altogether).
func familyEntry(t *testing.T, root, rel, term, status, ntbcw, syns string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("<!-- attribution -->\n---\nterm: " + term + "\n")
	if syns != "" {
		b.WriteString("forbidden_synonyms: " + syns + "\n")
	}
	b.WriteString("status: " + status + "\n")
	if ntbcw != "" {
		b.WriteString("not_to_be_confused_with: " + ntbcw + "\n")
	}
	b.WriteString("---\n# " + term + "\n")
	writeFile(t, root, "gloss/"+rel, b.String())
}

func familiesCfg() Config {
	return Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			ruleGlossaryFamilyPointer: {
				Enabled: true, Severity: severityBlocker,
				GlossaryDir: "gloss", Registry: "gloss/core/record-families.md",
			},
			ruleRecordFamilyKey: {
				Enabled: true, Severity: severityBlocker,
				GlossaryDir: "gloss", Registry: "gloss/core/record-families.md",
				RecordStores: map[string]string{"itd": "rec/intents"},
			},
		},
	}
}

// TestGlossaryFamilyPointerRefusesAnEntryNamingNothingOnThePage is acceptance
// criterion 4's first half (itd-2609211913453478): a glossary entry whose
// not_to_be_confused_with names nothing on the record-families page is refused.
// Naming a family row, or the page itself, anywhere in the value is the pointer;
// a retired word, a term the page has no row for, null and an absent key are not.
func TestGlossaryFamilyPointerRefusesAnEntryNamingNothingOnThePage(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/README.md", "# rec\n")
	writeFile(t, root, "gloss/core/record-families.md", familiesPage)
	writeFile(t, root, "gloss/_template.md", "---\nterm: example-term\nnot_to_be_confused_with: null\n---\n# t\n")
	writeFile(t, root, "gloss/README.md", "# glossary\n")
	writeFile(t, root, "gloss/core/README.md", "# core\n")

	// Pass: a family row, the page itself, a list holding one of them.
	familyEntry(t, root, "core/intent.md", "intent", "stable", "core/spec", "")
	familyEntry(t, root, "core/phase.md", "phase", "superseded", "core/record-families", "")
	familyEntry(t, root, "ledger/position.md", "position", "draft", "[ledger/regime, core/record-families]", "")
	familyEntry(t, root, "distribution/version.md", "version", "stable", `"distribution/release"`, "")

	// Refused: a term with no row, a retired word, null, absent.
	familyEntry(t, root, "ledger/warm.md", "warm", "draft", "ledger/cold-reading", "")
	familyEntry(t, root, "core/loop.md", "loop", "draft", "core/phase", "")
	familyEntry(t, root, "ledger/lapse.md", "lapse", "draft", "null", "")
	familyEntry(t, root, "core/persona.md", "persona", "stable", "", "")

	fs, err := Lint(familiesCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleGlossaryFamilyPointer); n != 4 {
		t.Fatalf("want 4 %s findings, got %d: %+v", ruleGlossaryFamilyPointer, n, fs)
	}
	for _, want := range []struct {
		file string
		line int
	}{
		{"gloss/ledger/warm.md", 5},
		{"gloss/core/loop.md", 5},
		{"gloss/ledger/lapse.md", 5},
		{"gloss/core/persona.md", 3},
	} {
		if !hasFinding(fs, filepath.FromSlash(want.file), ruleGlossaryFamilyPointer, want.line) {
			t.Errorf("missing %s on %s:%d: %+v", ruleGlossaryFamilyPointer, want.file, want.line, fs)
		}
	}
	for _, f := range fs {
		if f.RuleID == ruleGlossaryFamilyPointer && f.Severity != severityBlocker {
			t.Errorf("the pointer check refuses: want severity %s, got %s", severityBlocker, f.Severity)
		}
		if f.RuleID == ruleGlossaryFamilyPointer && !strings.Contains(f.Message, "intent, spec, step, bundle, issue, release, status") {
			t.Errorf("message should name the page's families: %q", f.Message)
		}
	}
}

// TestRecordFamilyKeyReportsAKeyNamingAFamilyThePageDoesNotDefine is acceptance
// criterion 4's second half: a record frontmatter key naming a family the page
// does not define is reported, never refused. A key names a family when one of
// its underscore-separated words (singular or plural) is a grouping word the
// glossary knows: a family on the page, a superseded entry's term or alias, or a
// forbidden synonym of a family entry. Only the words the page has no row for
// are reported; a word the glossary does not know is not a family at all.
func TestRecordFamilyKeyReportsAKeyNamingAFamilyThePageDoesNotDefine(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/README.md", "# rec\n")
	writeFile(t, root, "gloss/core/record-families.md", familiesPage)
	familyEntry(t, root, "core/spec.md", "spec", "stable", "core/intent", `["milestone", "epic"]`)
	familyEntry(t, root, "core/phase.md", "phase", "superseded", "core/record-families", `["version", "release"]`)

	writeFile(t, root, "rec/intents/planned/itd-5-x.md", strings.Join([]string{
		"---",
		"id: itd-5",
		"spec_id: null",               // spec: a family on the page
		"bundle: b",                   // bundle: a family on the page
		"grandfathered_at_phase: p-3", // phase: superseded           -> line 5
		"schema_version: 1",           // version: phase's synonym, not a family word
		"epics: []",                   // epic: a family's forbidden synonym (plural) -> line 7
		"grill_session_id: s",         // session: not a grouping word
		"---",
		"# x",
	}, "\n")+"\n")

	fs, err := Lint(familiesCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordFamilyKey); n != 2 {
		t.Fatalf("want 2 %s findings, got %d: %+v", ruleRecordFamilyKey, n, fs)
	}
	file := filepath.FromSlash("rec/intents/planned/itd-5-x.md")
	for _, line := range []int{5, 7} {
		if !hasFinding(fs, file, ruleRecordFamilyKey, line) {
			t.Errorf("missing %s on line %d: %+v", ruleRecordFamilyKey, line, fs)
		}
	}
	for _, f := range fs {
		if f.RuleID == ruleRecordFamilyKey && f.Severity != severityWarn {
			t.Errorf("the key check reports and never refuses: want %s, got %s (config asked blocker)", severityWarn, f.Severity)
		}
	}
}

// TestRecordFamiliesFailClosedWithoutThePage proves an armed rule never goes
// green on a missing or tableless page: that would disarm the refusal silently.
func TestRecordFamiliesFailClosedWithoutThePage(t *testing.T) {
	for name, page := range map[string]string{
		"missing":   "",
		"tableless": "---\nterm: record-families\n---\n# record-families\n\nNo table here.\n",
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, "rec/README.md", "# rec\n")
			if page != "" {
				writeFile(t, root, "gloss/core/record-families.md", page)
			}
			familyEntry(t, root, "core/intent.md", "intent", "stable", "core/spec", "")
			if _, err := Lint(familiesCfg(), root); err == nil || !strings.Contains(err.Error(), "record-families") {
				t.Fatalf("want an error naming the page, got %v", err)
			}
		})
	}
}
