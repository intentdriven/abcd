package lint

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// schemaStores is the store map every record_schema fixture below shares.
func schemaStores() map[string]string {
	return map[string]string{
		"adr": "rec/decisions/adrs",
		"itd": "rec/intents",
		"spc": "rec/specs",
		"iss": "work/issues",
	}
}

func schemaConfig() Config {
	return Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			ruleRecordSchema: {Enabled: true, Severity: severityBlocker, RecordStores: schemaStores()},
		},
	}
}

// findingWith reports whether a finding on file:line quotes the given substring.
func findingWith(fs []Finding, file, ruleID string, substr string) bool {
	for _, f := range fs {
		if f.File == file && f.RuleID == ruleID && strings.Contains(f.Message, substr) {
			return true
		}
	}
	return false
}

// Corpus fixture (a): the shape of the real adr-6 defect — two ADRs name adr-6 as
// a live design input in related_adrs, no 0006-*.md is in the store, and no record
// declares it superseded. The control in the same fixture is adr-8, which is
// equally absent but IS declared retired by its successor's `supersedes` (the ADR
// store's documented pruning convention), and must stay silent.
func TestRecordSchemaPhantomCrossReference(t *testing.T) {
	root := t.TempDir()
	adrs := "rec/decisions/adrs"
	writeFile(t, root, adrs+"/0022-adapters.md", "---\nid: adr-22\nsupersedes: [adr-14]\nsuperseded_by: null\nrelated_adrs: []\n---\n# ADR-22\n")
	writeFile(t, root, adrs+"/0025-host-delegated.md", "---\nid: adr-25\nsupersedes: [adr-8]\nsuperseded_by: null\nrelated_adrs: [adr-6, adr-22]\n---\n# ADR-25\n")
	writeFile(t, root, adrs+"/0029-transcripts.md", "---\nid: adr-29\nsupersedes: null\nsuperseded_by: null\nrelated_adrs: [adr-6, adr-22]\n---\n# ADR-29\n")
	// A retired handle cited from an intent must resolve through the successor's
	// declaration, not a file.
	writeFile(t, root, "rec/intents/drafts/itd-17-tracking.md", "---\nid: itd-17\nkind: null\nspec_id: null\nrelated_adrs: [adr-8, adr-25]\n---\n# draft\n")

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordSchema); n != 2 {
		t.Fatalf("expected exactly 2 record_schema findings (the two adr-6 citations), got %d: %+v", n, fs)
	}
	for _, f := range []string{"0025-host-delegated.md", "0029-transcripts.md"} {
		if !findingWith(fs, filepath.Join(adrs, f), ruleRecordSchema, "'adr-6'") {
			t.Errorf("expected an unresolved adr-6 finding on %s: %+v", f, fs)
		}
	}
}

// Corpus fixture (b): the shape of the real adr-12 / adr-32 defect — a supersession
// declared from one side only. Both directions are checked, because either half
// alone leaves the record contradicting itself about which decision is in force.
func TestRecordSchemaSupersessionIsBidirectional(t *testing.T) {
	adrs := "rec/decisions/adrs"

	t.Run("forward half only", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, adrs+"/0012-issue-ledger.md", "---\nid: adr-12\nsupersedes: null\nsuperseded_by: adr-32\n---\n# ADR-12\n")
		writeFile(t, root, adrs+"/0032-working-tier.md", "---\nid: adr-32\nsupersedes: null\nsuperseded_by: null\n---\n# ADR-32\n")
		fs, err := Lint(schemaConfig(), root)
		if err != nil {
			t.Fatal(err)
		}
		if n := countRule(fs, ruleRecordSchema); n != 1 {
			t.Fatalf("expected 1 one-way supersession finding, got %d: %+v", n, fs)
		}
		if !findingWith(fs, filepath.Join(adrs, "0012-issue-ledger.md"), ruleRecordSchema, "one-way supersession") {
			t.Errorf("expected the finding on the declaring record: %+v", fs)
		}
	})

	t.Run("reverse half only", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, adrs+"/0012-issue-ledger.md", "---\nid: adr-12\nsupersedes: null\nsuperseded_by: null\n---\n# ADR-12\n")
		writeFile(t, root, adrs+"/0032-working-tier.md", "---\nid: adr-32\nsupersedes: adr-12\nsuperseded_by: null\n---\n# ADR-32\n")
		fs, err := Lint(schemaConfig(), root)
		if err != nil {
			t.Fatal(err)
		}
		if !findingWith(fs, filepath.Join(adrs, "0032-working-tier.md"), ruleRecordSchema, "one-way supersession") {
			t.Fatalf("expected a one-way finding on the supersedes side: %+v", fs)
		}
	})

	t.Run("both halves present", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, adrs+"/0012-issue-ledger.md", "---\nid: adr-12\nsupersedes: null\nsuperseded_by: adr-32\n---\n# ADR-12\n")
		writeFile(t, root, adrs+"/0032-working-tier.md", "---\nid: adr-32\nsupersedes: adr-12\nsuperseded_by: null\n---\n# ADR-32\n")
		fs, err := Lint(schemaConfig(), root)
		if err != nil {
			t.Fatal(err)
		}
		if n := countRule(fs, ruleRecordSchema); n != 0 {
			t.Fatalf("a fully-linked pair must be clean, got %d: %+v", n, fs)
		}
	})
}

// A supersession may cross stores: an ADR that redecides the question an intent
// rested on retires that intent (the real itd-47 → adr-22, itd-49 → adr-26). The
// bidirectional rule then binds the ADR to name the intent back.
func TestRecordSchemaSupersessionCrossesStores(t *testing.T) {
	root := t.TempDir()
	adrs := "rec/decisions/adrs"
	sup := "rec/intents/superseded"
	writeFile(t, root, adrs+"/0022-adapters.md", "---\nid: adr-22\nsupersedes: [itd-47]\nsuperseded_by: null\n---\n# ADR-22\n")
	writeFile(t, root, adrs+"/0026-spec-layer.md", "---\nid: adr-26\nsupersedes: null\nsuperseded_by: null\n---\n# ADR-26\n")
	writeFile(t, root, sup+"/itd-47-oracle-gates.md", "---\nid: itd-47\nkind: standalone\nsuperseded_by: adr-22\n---\n# ok\n")
	writeFile(t, root, sup+"/itd-49-flow-drift.md", "---\nid: itd-49\nkind: standalone\nsuperseded_by: adr-26\n---\n# bad\n")

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordSchema); n != 1 {
		t.Fatalf("expected 1 finding (adr-26 does not name itd-49 back), got %d: %+v", n, fs)
	}
	if !findingWith(fs, filepath.Join(sup, "itd-49-flow-drift.md"), ruleRecordSchema, "one-way supersession") {
		t.Errorf("expected the one-way finding on itd-49: %+v", fs)
	}
}

// A record answers to exactly one handle: the id its filename claims and the id
// its frontmatter declares must agree, because half the rules key on one and half
// on the other.
func TestRecordSchemaFilenameMatchesID(t *testing.T) {
	root := t.TempDir()
	adrs := "rec/decisions/adrs"
	writeFile(t, root, adrs+"/0012-issue-ledger.md", "---\nid: adr-21\n---\n# ADR-12\n")
	writeFile(t, root, adrs+"/0013-memory.md", "---\nid: adr-13\n---\n# ADR-13\n")
	// A record with no id at all: the filename-AGREEMENT rule still invents no
	// agreement for it (no "filename claims" mismatch) — but an ADR's id is a
	// required property, so its absence is now its own finding
	// (iss-2608270908344426), because the record dispatcher confirms the id before
	// it will render the record.
	writeFile(t, root, adrs+"/0007-grill.md", "# ADR-7\n")
	writeFile(t, root, "rec/intents/planned/itd-4-capture.md", "---\nid: itd-5\nkind: standalone\nspec_id: null\n---\n# x\n")

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordSchema); n != 3 {
		t.Fatalf("expected 3 findings (two filename/id mismatches + the id-less ADR), got %d: %+v", n, fs)
	}
	if !findingWith(fs, filepath.Join(adrs, "0012-issue-ledger.md"), ruleRecordSchema, "filename claims id 'adr-12'") {
		t.Errorf("expected the ADR mismatch: %+v", fs)
	}
	if !findingWith(fs, filepath.Join("rec", "intents", "planned", "itd-4-capture.md"), ruleRecordSchema, "filename claims id 'itd-4'") {
		t.Errorf("expected the intent mismatch: %+v", fs)
	}
	// The id-less ADR is caught by the required-property check, NOT by an invented
	// filename agreement.
	grill := filepath.Join(adrs, "0007-grill.md")
	if !findingWith(fs, grill, ruleRecordSchema, "missing required property 'id'") {
		t.Errorf("expected a missing-id finding on the id-less ADR: %+v", fs)
	}
	if findingWith(fs, grill, ruleRecordSchema, "filename claims") {
		t.Errorf("the filename rule must not invent an agreement for an id-less record: %+v", fs)
	}
}

// Every lifecycle directory is enumerated, so a bucket nobody declared cannot hold
// records that no rule ever reads — and a record cannot escape its lifecycle by
// sitting in the store root.
func TestRecordSchemaLifecycleDirectoriesAreDeclared(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/intents/shipped/itd-1-good.md", "---\nid: itd-1\nkind: standalone\nspec_id: spc-1-x\n---\n# ok\n")
	writeFile(t, root, "rec/intents/archived/itd-2-hidden.md", "---\nid: itd-2\nkind: standalone\n---\n# hidden\n")
	writeFile(t, root, "rec/intents/itd-3-loose.md", "---\nid: itd-3\nkind: standalone\n---\n# loose\n")
	writeFile(t, root, "rec/intents/README.md", "# intents\n")
	writeFile(t, root, "rec/specs/open/notes.md", "# not a spec filename\n")

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join("rec", "intents", "archived"), ruleRecordSchema, "is not a declared intent bucket") {
		t.Errorf("expected an undeclared-bucket finding: %+v", fs)
	}
	if !findingWith(fs, filepath.Join("rec", "intents", "itd-3-loose.md"), ruleRecordSchema, "rather than a lifecycle bucket") {
		t.Errorf("expected a store-root record finding: %+v", fs)
	}
	if !findingWith(fs, filepath.Join("rec", "specs", "open", "notes.md"), ruleRecordSchema, "not a well-formed spec filename") {
		t.Errorf("expected a malformed-filename finding: %+v", fs)
	}
	// The README and the well-formed record are clean.
	if n := countRule(fs, ruleRecordSchema); n != 3 {
		t.Fatalf("expected exactly 3 walk findings, got %d: %+v", n, fs)
	}
}

// A FLAT store declares no buckets, so any subdirectory of it is undeclared. Left
// unsaid, a directory one level down hides every record inside it from every check
// in the rule — the same escape the bucketed branch closes, in the branch that
// looked like it had nothing to close.
func TestRecordSchemaFlatStoreHasNoSubdirectories(t *testing.T) {
	root := t.TempDir()
	adrs := "rec/decisions/adrs"
	writeFile(t, root, adrs+"/0013-memory.md", "---\nid: adr-13\n---\n# ADR-13\n")
	// Three defects, each of a class the rule checks, all one directory down.
	writeFile(t, root, adrs+"/archive/0012-issue-ledger.md", "---\nid: adr-21\nsuperseded_by: adr-9999\n---\n# ADR-12\n")
	writeFile(t, root, adrs+"/archive/loose-notes.md", "# notes\n")

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join(adrs, "archive"), ruleRecordSchema, "store is flat, so subdirectory 'archive' is undeclared") {
		t.Fatalf("expected an undeclared-subdirectory finding on the flat store: %+v", fs)
	}
}

// A DECLARED bucket holds its records directly, so a directory inside one is the
// same escape as a directory inside a flat store — every check in the rule stops
// at it. Closing it only for the store roots left three of the four stores open.
func TestRecordSchemaBucketHasNoSubdirectories(t *testing.T) {
	root := t.TempDir()
	shipped := "rec/intents/shipped"
	writeFile(t, root, shipped+"/itd-1-good.md", "---\nid: itd-1\nkind: standalone\nspec_id: spc-1-x\n---\n# ok\n")
	// Two defects, each of a class the rule checks, one directory down.
	writeFile(t, root, shipped+"/archive/itd-2-bad.md", "---\nid: itd-7\nkind: standalone\nsuperseded_by: itd-9999\n---\n# hidden\n")

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join(shipped, "archive"), ruleRecordSchema, "holds records directly, so subdirectory 'archive' is undeclared") {
		t.Fatalf("expected an undeclared-subdirectory finding inside the bucket: %+v", fs)
	}
	if n := countRule(fs, ruleRecordSchema); n != 1 {
		t.Fatalf("expected exactly 1 finding, got %d: %+v", n, fs)
	}
}

// The spec store is indexed by this rule, so a spec handle must PARSE as one. A
// prefix the rule indexes but the pattern omits reads as "no handle at all": the
// forward direction becomes a false blocker and the reverse is never checked.
func TestRecordSchemaResolvesSpecHandles(t *testing.T) {
	specs := "rec/specs"

	t.Run("a bidirectional spec supersession is clean", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, specs+"/closed/spc-5-old.md", "---\nid: spc-5\nintent: itd-1\nslug: old\nsuperseded_by: spc-6\n---\n# x\n")
		writeFile(t, root, specs+"/open/spc-6-new.md", "---\nid: spc-6\nintent: itd-1\nslug: new\nsupersedes: [spc-5]\n---\n# x\n")
		fs, err := Lint(schemaConfig(), root)
		if err != nil {
			t.Fatal(err)
		}
		if n := countRule(fs, ruleRecordSchema); n != 0 {
			t.Fatalf("a fully-linked spec pair must be clean, got %d: %+v", n, fs)
		}
	})

	t.Run("a one-way spec supersession is caught", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, specs+"/closed/spc-5-old.md", "---\nid: spc-5\nintent: itd-1\nslug: old\nsuperseded_by: spc-6\n---\n# x\n")
		writeFile(t, root, specs+"/open/spc-6-new.md", "---\nid: spc-6\nintent: itd-1\nslug: new\nsupersedes: null\n---\n# x\n")
		fs, err := Lint(schemaConfig(), root)
		if err != nil {
			t.Fatal(err)
		}
		if !findingWith(fs, filepath.Join(specs, "closed", "spc-5-old.md"), ruleRecordSchema, "one-way supersession") {
			t.Fatalf("expected the reverse direction to be checked for specs: %+v", fs)
		}
	})
}

// An empty flow sequence is this record's house spelling for an empty list, so it
// says "nothing here" — not "a value that is not a handle".
func TestRecordSchemaEmptyListIsAbsent(t *testing.T) {
	root := t.TempDir()
	adrs := "rec/decisions/adrs"
	writeFile(t, root, adrs+"/0001-model.md", "---\nid: adr-1\nsupersedes: []\nsuperseded_by: []\nrelated_adrs: []\n---\n# ADR-1\n")
	writeFile(t, root, adrs+"/0002-kinds.md", "---\nid: adr-2\nsuperseded_by: [ ]\n---\n# ADR-2\n")

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordSchema); n != 0 {
		t.Fatalf("an empty list is an absence, got %d: %+v", n, fs)
	}
}

// A zero-padded id and its bare spelling are one handle — the rule says so about
// its cross-references, and must not contradict itself about its filenames.
func TestRecordSchemaFilenameIDComparesNumerically(t *testing.T) {
	root := t.TempDir()
	adrs := "rec/decisions/adrs"
	writeFile(t, root, adrs+"/0012-issue-ledger.md", "---\nid: adr-0012\n---\n# ADR-12\n")
	writeFile(t, root, adrs+"/0013-memory.md", "---\nid: ADR-13\n---\n# ADR-13\n")
	// Still a mismatch: a different number, and a value that is no handle at all.
	writeFile(t, root, adrs+"/0021-go.md", "---\nid: adr-0022\n---\n# ADR-21\n")
	writeFile(t, root, adrs+"/0023-core.md", "---\nid: core\n---\n# ADR-23\n")

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordSchema); n != 2 {
		t.Fatalf("expected 2 filename/id findings, got %d: %+v", n, fs)
	}
	for _, f := range []string{"0021-go.md", "0023-core.md"} {
		if !findingWith(fs, filepath.Join(adrs, f), ruleRecordSchema, "filename claims id") {
			t.Errorf("expected a mismatch on %s: %+v", f, fs)
		}
	}
}

// The retirement escape hatch is bounded by what the store has actually issued.
// Unbounded it is self-attesting: one record naming a phantom in `supersedes`
// would make that phantom resolvable in every other record's cross-references.
func TestRecordSchemaRetirementIsBoundedByAllocation(t *testing.T) {
	adrs := "rec/decisions/adrs"

	t.Run("an id the store never issued", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, adrs+"/0025-host-delegated.md", "---\nid: adr-25\nsupersedes: [adr-9999]\nrelated_adrs: [adr-9999]\n---\n# ADR-25\n")
		fs, err := Lint(schemaConfig(), root)
		if err != nil {
			t.Fatal(err)
		}
		if n := countRule(fs, ruleRecordSchema); n != 2 {
			t.Fatalf("expected 2 findings (the declaration and the reference it must not legitimise), got %d: %+v", n, fs)
		}
		if !findingWith(fs, filepath.Join(adrs, "0025-host-delegated.md"), ruleRecordSchema, "nor an id the adr store has issued") {
			t.Errorf("expected the supersedes declaration itself to be refused: %+v", fs)
		}
		if !findingWith(fs, filepath.Join(adrs, "0025-host-delegated.md"), ruleRecordSchema, "related_adrs names 'adr-9999'") {
			t.Errorf("expected the cross-reference to stay unresolved: %+v", fs)
		}
	})

	t.Run("a genuinely pruned id", func(t *testing.T) {
		root := t.TempDir()
		// adr-8 has no file: it was pruned when adr-25 landed. adr-25 says so, and
		// 8 is below the store's high-water mark, so citing it elsewhere resolves.
		writeFile(t, root, adrs+"/0025-host-delegated.md", "---\nid: adr-25\nsupersedes: [adr-8]\n---\n# ADR-25\n")
		writeFile(t, root, "rec/intents/drafts/itd-17-tracking.md", "---\nid: itd-17\nkind: null\nspec_id: null\nrelated_adrs: [adr-8]\n---\n# draft\n")
		fs, err := Lint(schemaConfig(), root)
		if err != nil {
			t.Fatal(err)
		}
		if n := countRule(fs, ruleRecordSchema); n != 0 {
			t.Fatalf("a pruned id below the high-water mark must resolve, got %d: %+v", n, fs)
		}
	})
}

// YAML spells a list two ways and the record uses both. Reading only the same-line
// form makes a block sequence look empty — and an empty read here does not merely
// under-report, it makes the bidirectional check assert that ANOTHER file omits a
// link that file plainly carries.
func TestRecordSchemaReadsBlockSequences(t *testing.T) {
	adrs := "rec/decisions/adrs"

	t.Run("a linked pair written in block form is clean", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, adrs+"/0012-issue-ledger.md", "---\nid: adr-12\nsuperseded_by: adr-32\n---\n# ADR-12\n")
		writeFile(t, root, adrs+"/0032-working-tier.md", "---\nid: adr-32\nsupersedes:\n  - adr-12\nrelated_adrs:\n  - adr-12\n---\n# ADR-32\n")
		fs, err := Lint(schemaConfig(), root)
		if err != nil {
			t.Fatal(err)
		}
		if n := countRule(fs, ruleRecordSchema); n != 0 {
			t.Fatalf("block-sequence links must read as links, got %d: %+v", n, fs)
		}
	})

	t.Run("a one-way pair written in block form is still caught", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, adrs+"/0012-issue-ledger.md", "---\nid: adr-12\nsuperseded_by:\n  - adr-32\n---\n# ADR-12\n")
		writeFile(t, root, adrs+"/0032-working-tier.md", "---\nid: adr-32\nsupersedes: null\n---\n# ADR-32\n")
		fs, err := Lint(schemaConfig(), root)
		if err != nil {
			t.Fatal(err)
		}
		if !findingWith(fs, filepath.Join(adrs, "0012-issue-ledger.md"), ruleRecordSchema, "one-way supersession") {
			t.Fatalf("expected the one-way finding to survive the block spelling: %+v", fs)
		}
	})

	// YAML nests a block sequence under a mapping key with NO extra indentation:
	// `supersedes:\n- adr-12` is the same list as `supersedes:\n  - adr-12`, and the
	// record writes both. Reading only the indented spelling makes the column-0 one
	// look empty — the same false assertion about ANOTHER file, reached by a purely
	// cosmetic difference in how the list was typed.
	t.Run("the column-0 spelling reads the same as the indented one", func(t *testing.T) {
		lint := func(t *testing.T, supersedes, related string) []Finding {
			t.Helper()
			root := t.TempDir()
			writeFile(t, root, adrs+"/0012-issue-ledger.md", "---\nid: adr-12\nsuperseded_by: adr-32\n---\n# ADR-12\n")
			writeFile(t, root, adrs+"/0032-working-tier.md",
				"---\nid: adr-32\nsupersedes:\n"+supersedes+"related_adrs:\n"+related+"---\n# ADR-32\n")
			fs, err := Lint(schemaConfig(), root)
			if err != nil {
				t.Fatal(err)
			}
			return fs
		}

		indented := lint(t, "  - adr-12\n", "  - adr-12\n")
		column0 := lint(t, "- adr-12\n", "- adr-12\n")
		if n := countRule(indented, ruleRecordSchema); n != 0 {
			t.Fatalf("the indented spelling must be clean, got %d: %+v", n, indented)
		}
		if n := countRule(column0, ruleRecordSchema); n != 0 {
			t.Fatalf("the column-0 spelling is the same list, got %d: %+v", n, column0)
		}
	})

	t.Run("a one-way pair written at column 0 is still caught", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, adrs+"/0012-issue-ledger.md", "---\nid: adr-12\nsuperseded_by:\n- adr-32\n---\n# ADR-12\n")
		writeFile(t, root, adrs+"/0032-working-tier.md", "---\nid: adr-32\nsupersedes: null\n---\n# ADR-32\n")
		fs, err := Lint(schemaConfig(), root)
		if err != nil {
			t.Fatal(err)
		}
		if !findingWith(fs, filepath.Join(adrs, "0012-issue-ledger.md"), ruleRecordSchema, "one-way supersession") {
			t.Fatalf("expected the one-way finding to survive the column-0 spelling: %+v", fs)
		}
	})

	// A blank line or a comment interrupts a sequence without ending it. Stopping
	// at the interruption drops the tail — and the dropped handle then reads as a
	// link the OTHER file omits, which is the false claim this parse prevents.
	t.Run("a blank line or comment does not truncate the sequence", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, adrs+"/0012-a.md", "---\nid: adr-12\nsuperseded_by: adr-32\n---\n# a\n")
		writeFile(t, root, adrs+"/0013-b.md", "---\nid: adr-13\nsuperseded_by: adr-32\n---\n# b\n")
		writeFile(t, root, adrs+"/0032-new.md",
			"---\nid: adr-32\nsupersedes:\n- adr-12\n\n# the ledger-location pair\n- adr-13\nrelated_adrs: []\n---\n# new\n")
		fs, err := Lint(schemaConfig(), root)
		if err != nil {
			t.Fatal(err)
		}
		if n := countRule(fs, ruleRecordSchema); n != 0 {
			t.Fatalf("an interrupted sequence is still one list, got %d: %+v", n, fs)
		}
	})

	// The interruption skip must not run past the frontmatter: the closing `---`
	// is neither blank nor a comment, so it still ends the scan.
	t.Run("the scan still stops at the frontmatter boundary", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, adrs+"/0032-new.md",
			"---\nid: adr-32\nsupersedes:\n\n---\n\n- adr-9999\n")
		fs, err := Lint(schemaConfig(), root)
		if err != nil {
			t.Fatal(err)
		}
		if n := countRule(fs, ruleRecordSchema); n != 0 {
			t.Fatalf("a body list is not the key's sequence, got %d: %+v", n, fs)
		}
	})

	t.Run("an unrelated block key does not leak into the next", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, "rec/intents/superseded/itd-31-cross-doc.md",
			"---\nid: itd-31\nkind: standalone\nreclassification_history:\n  - { date: 2026-05-27, reason: \"absorbed by itd-48\" }\nsupersedes: null\nsuperseded_by: itd-48\n---\n# x\n")
		writeFile(t, root, "rec/intents/shipped/itd-48-successor.md", "---\nid: itd-48\nkind: standalone\nspec_id: spc-2-x\nsupersedes:\n  - itd-31\n---\n# ok\n")
		fs, err := Lint(schemaConfig(), root)
		if err != nil {
			t.Fatal(err)
		}
		if n := countRule(fs, ruleRecordSchema); n != 0 {
			t.Fatalf("expected a clean pair, got %d: %+v", n, fs)
		}
	})
}

// A dot-directory is tooling state, not a lifecycle the record authored, so it is
// not an undeclared bucket.
func TestRecordSchemaIgnoresDotDirectories(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/intents/shipped/itd-1-good.md", "---\nid: itd-1\nkind: standalone\nspec_id: spc-1-x\n---\n# ok\n")
	writeFile(t, root, "rec/intents/.obsidian/workspace.json", "{}\n")
	writeFile(t, root, "rec/decisions/adrs/.vscode/settings.json", "{}\n")

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordSchema); n != 0 {
		t.Fatalf("dot-directories are not undeclared buckets, got %d: %+v", n, fs)
	}
}

// Corpus fixture (c)+(d): the superseded/ exemption is a CONTENT exemption. A
// record in an exempt bucket still answers the schema rules — which is the only
// reason the missing supersession on the two real superseded intents is visible at
// all — while the content-authoring ban above it stays suppressed.
func TestSupersededBucketExemptsContentNotSchema(t *testing.T) {
	root := t.TempDir()
	sup := "rec/intents/superseded"
	writeFile(t, root, "rec/intents/shipped/itd-48-successor.md", "---\nid: itd-48\nkind: standalone\nspec_id: spc-2-x\nsupersedes: itd-31\n---\n# ok\n")
	// Blockquote prose only, exactly as the two real records carried it.
	writeFile(t, root, sup+"/itd-47-oracle-gates.md",
		"---\nid: itd-47\nkind: standalone\n---\n\n> **Superseded by ADR-22** — intent_lint.py is gone.\n")
	writeFile(t, root, sup+"/itd-31-cross-doc.md",
		"---\nid: itd-31\nkind: standalone\nsuperseded_by: itd-48\n---\n\n> intent_lint.py is gone.\n")

	cfg := Config{
		Roots: []string{"rec"},
		BannedTokens: []BannedToken{
			{ID: "py", Pattern: `intent_lint\.py`, Message: "no python name", Severity: severityBlocker,
				Successor: "internal/core/lint", AllowContext: []string{"historical"}},
		},
		Rules: map[string]RuleConfig{
			"intent_lifecycle": {Enabled: true, Severity: severityBlocker, IntentsDir: "intents"},
			ruleRecordSchema:   {Enabled: true, Severity: severityBlocker, RecordStores: schemaStores()},
		},
		ExemptPaths: []string{filepath.Join("rec", "intents", "superseded")},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}

	// Content stays exempt: neither superseded record trips the banned token.
	if n := countRule(fs, "py"); n != 0 {
		t.Errorf("the superseded bucket must stay exempt from content bans, got %d: %+v", n, fs)
	}
	// Schema does not: the record with prose but no superseded_by field is caught.
	if !hasFinding(fs, filepath.Join(sup, "itd-47-oracle-gates.md"), "intent_lifecycle", 1) {
		t.Errorf("expected a lifecycle finding on the superseded record with no supersession field: %+v", fs)
	}
	// And the well-formed superseded record stays clean.
	for _, f := range fs {
		if filepath.Base(f.File) == "itd-31-cross-doc.md" {
			t.Errorf("unexpected finding on the well-formed superseded record: %+v", f)
		}
	}
}

// A record whose lifecycle bucket is superseded/ must name its successor; an ADR
// is a legal successor, so the lifecycle rule accepts an adr-N target and only
// resolves an intent one against the intent tree.
func TestIntentLifecycleAcceptsAnADRSuccessor(t *testing.T) {
	root := t.TempDir()
	base := "rec/intents"
	writeFile(t, root, base+"/superseded/itd-47-adr-successor.md", "---\nid: itd-47\nkind: standalone\nsuperseded_by: adr-22\n---\n# ok\n")
	writeFile(t, root, base+"/superseded/itd-49-no-successor.md", "---\nid: itd-49\nkind: standalone\n---\n# bad\n")
	writeFile(t, root, base+"/superseded/itd-50-ghost.md", "---\nid: itd-50\nkind: standalone\nsuperseded_by: itd-999\n---\n# bad\n")

	cfg := Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			"intent_lifecycle": {Enabled: true, Severity: severityBlocker, IntentsDir: "intents"},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "intent_lifecycle"); n != 2 {
		t.Fatalf("expected 2 lifecycle findings, got %d: %+v", n, fs)
	}
	if !hasFinding(fs, filepath.Join(base, "superseded", "itd-49-no-successor.md"), "intent_lifecycle", 1) {
		t.Errorf("expected a finding for the missing superseded_by: %+v", fs)
	}
	if !findingWith(fs, filepath.Join(base, "superseded", "itd-50-ghost.md"), "intent_lifecycle", "does not exist in any bucket") {
		t.Errorf("expected a finding for the unresolvable intent successor: %+v", fs)
	}
}

// TestRecordSchemaRequiresIssueFrontmatter (iss-2608261437041050) pins the
// required-property invariant for the issue store. The ledger reader validates
// every record it reads and SKIPS the ones that fail, so a committed record
// missing a required property is invisible to every capture surface while the
// record lint — which never asked the question — stays green. A store that
// declares no required properties (adr/itd/spc, whose schemas differ) is
// untouched, which the well-formed records in the same fixture prove.
func TestRecordSchemaRequiresIssueFrontmatter(t *testing.T) {
	root := t.TempDir()
	issues := "work/issues"
	full := "---\nschema_version: 1\nid: iss-1\nslug: ok\nseverity: minor\ncategory: bug\nsource: user-observation\nfound_during: t\n---\n\nan issue\n"
	writeFile(t, root, issues+"/open/iss-1-ok.md", full)
	// The live shape of the defect: the record is complete but for schema_version,
	// so every other gate reads it and only the reader drops it.
	writeFile(t, root, issues+"/resolved/iss-2-stripped.md",
		"---\nid: iss-2\nslug: stripped\nseverity: minor\ncategory: bug\nsource: user-observation\nfound_during: t\nresolution: done\n---\n\nan issue\n")
	// A store with a different schema must not be judged against the issue's.
	writeFile(t, root, "rec/decisions/adrs/0001-model.md", "---\nid: adr-1\n---\n# ADR-1\n")

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join(issues, "resolved", "iss-2-stripped.md"), ruleRecordSchema, "schema_version") {
		t.Errorf("expected a missing-schema_version finding on the stripped record: %+v", fs)
	}
	if n := countRule(fs, ruleRecordSchema); n != 1 {
		t.Fatalf("expected exactly 1 record_schema finding (the stripped record), got %d: %+v", n, fs)
	}
}

// TestRecordSchemaGuardsTheRealRecord loads the committed record-lint config and
// asserts the rule is armed as a blocker over the real stores. Deleting the rule,
// dropping a store, or downgrading its severity fails here rather than silently
// re-opening the drift iss-39 closed.
func TestRecordSchemaGuardsTheRealRecord(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("..", "..", "..", ".abcd", "record-lint.json"))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	rc, ok := cfg.Rules[ruleRecordSchema]
	if !ok || !rc.Enabled {
		t.Fatalf("record_schema must be enabled in the committed config: %+v", rc)
	}
	if rc.Severity != severityBlocker {
		t.Errorf("record_schema severity = %q, want blocker", rc.Severity)
	}
	// Derived from the code's own store list rather than a literal: a store added
	// to recordStores and forgotten in the config is scanned nowhere, which is the
	// silent half of exactly the drift this rule exists to catch.
	for _, store := range recordStores {
		if rc.RecordStores[store.prefix] == "" {
			t.Errorf("record_schema declares no store for %q", store.prefix)
		}
	}
}

// TestRecordSchemaFilenameSlugAgrees pins the slug half of the filename ↔
// frontmatter agreement, the sibling of the id half checkRecordFilename already
// asserts. Both halves are one value written twice, and until this landed only
// the CAPTURE READER asked the slug question — so a record renamed by hand
// passed record-lint and every CI gate, then dropped to a Skipped line at read
// time. A silent skip is exactly the failure mode this gate exists to convert
// into a red one.
//
// The comparison is EXACT, with no prefix tolerance, because the stores apply
// their length cap BEFORE the value forks into the two writers, never after:
// capture derives and caps at internal/core/capture/roots.go:107-113
// (deriveSlug) and then hands the one capped string to both
// internal/core/capture/alloc.go:150 (the filename) and
// internal/core/capture/workflow.go:143 (fm["slug"]); the intent store caps the
// same way at internal/core/intent/create.go:185-186. A filename is therefore
// never a truncated form of a longer field, and tolerating a prefix would
// license precisely the drift this rule catches. Both packages call the ONE
// splitter (recordid.SplitRecordFilename) rather than restating the pattern, so
// the gate and the reader cannot reach different verdicts on one record.
func TestRecordSchemaFilenameSlugAgrees(t *testing.T) {
	root := t.TempDir()
	issues := "work/issues"
	iss := func(id, slug string) string {
		return "---\nschema_version: 1\nid: " + id + "\nslug: " + slug +
			"\nseverity: minor\ncategory: bug\nsource: user-observation\nfound_during: t\n---\n\nan issue\n"
	}
	// The control: name and field agree, and the rule must stay silent.
	writeFile(t, root, issues+"/open/iss-1-broken-thing.md", iss("iss-1", "broken-thing"))
	// The live ledger defect (iss-2608231237300997): the filename kept a handle
	// the frontmatter had already moved off.
	writeFile(t, root, issues+"/open/iss-2-inspirations-lead-removal.md", iss("iss-2", "no-way-to-edit-a-file"))
	// The live intent defect (itd-47): a stale FOREIGN handle in the field, where
	// spc-12 now names an unrelated live spec — so the record answers to a name
	// that points at someone else's record.
	writeFile(t, root, "rec/intents/superseded/itd-47-oracle-gates-autonomous-mode.md",
		"---\nid: itd-47\nslug: spc-12-oracle-gates-autonomous-mode\nkind: null\nspec_id: null\nsuperseded_by: adr-22\n---\n# superseded\n")
	// An ADR whose zero-padded numeric filename agrees with its slug: the check
	// must read the id half of a bare-numeric name without tripping on it.
	writeFile(t, root, "rec/decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md",
		"---\nid: adr-22\nslug: bundled-deps-as-pluggable-adapters\nsupersedes: [itd-47]\nsuperseded_by: null\nrelated_adrs: []\n---\n# ADR-22\n")
	// A record carrying NO slug at all is a different (and larger) schema
	// question, owned by checkRecordRequiredFields for the stores that declare
	// one. Absence must not be read as disagreement, or every ADR-shaped store
	// without the property turns red.
	writeFile(t, root, "rec/specs/open/spc-12-something-else.md",
		"---\nid: spc-12\nintent_id: null\n---\n# spec\n")

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ file, fnSlug, fmSlug string }{
		{filepath.Join(issues, "open", "iss-2-inspirations-lead-removal.md"),
			"inspirations-lead-removal", "no-way-to-edit-a-file"},
		{filepath.Join("rec", "intents", "superseded", "itd-47-oracle-gates-autonomous-mode.md"),
			"oracle-gates-autonomous-mode", "spc-12-oracle-gates-autonomous-mode"},
	} {
		// The message must name BOTH halves: told only that "the slug disagrees",
		// a reader still has to open the file to learn which side to correct.
		if !findingWith(fs, c.file, ruleRecordSchema, c.fnSlug) {
			t.Errorf("no record_schema finding on %s naming the filename slug %q: %+v", c.file, c.fnSlug, fs)
		}
		if !findingWith(fs, c.file, ruleRecordSchema, c.fmSlug) {
			t.Errorf("no record_schema finding on %s naming the frontmatter slug %q: %+v", c.file, c.fmSlug, fs)
		}
	}
	// Exactly the two drifted records — the agreeing issue, the agreeing ADR and
	// the slugless spec must all stay silent, or the rule is a false-blocker
	// generator on the committed tree.
	if n := countRule(fs, ruleRecordSchema); n != 2 {
		t.Fatalf("expected exactly 2 record_schema findings (the two drifted records), got %d: %+v", n, fs)
	}
}

// readingStores is schemaStores plus the three reading families, laid out as
// they are in a real ledger: two of them NESTED inside the issue store's own
// root, which is the arrangement the first change below exists for.
func readingStores() map[string]string {
	stores := schemaStores()
	stores["rdi"] = "work/issues/readings"
	stores["dsp"] = "work/issues/dispositions"
	stores["rdg"] = "rec/readings"
	return stores
}

func readingSchemaConfig() Config {
	return Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			ruleRecordSchema: {Enabled: true, Severity: severityBlocker, RecordStores: readingStores()},
		},
	}
}

// The reading families sit INSIDE the issue store's root, so without this the
// day they appear is the day record_schema calls each of them an undeclared
// issue bucket — a blocker over a directory the configuration itself declares.
//
// The fix is general rather than a special case: a directory that is itself a
// configured store root is not an undeclared bucket of its parent. The set is
// derived from the configuration, so the config stays the single declaration of
// what a store is.
func TestNestedStoreRootIsNotAnUndeclaredBucket(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/.keep", "")
	writeFile(t, root, "work/issues/open/iss-1-a-finding.md",
		"---\nschema_version: 1\nid: iss-1\nslug: a-finding\nseverity: minor\ncategory: bug\nsource: user-observation\nfound_during: t\n---\n\nan issue\n")
	writeFile(t, root, "work/issues/readings/rdg-1/rdi-2.md",
		"---\nschema_version: 1\nid: rdi-2\nrun: rdg-1\nmanifest: sha256:beef\nposition: detection\nregime: registrative\npattern: a stated constraint\ntension: t\nconstraint_in_play: c\nwhy_a_tension: w\n---\n\n")
	writeFile(t, root, "work/issues/dispositions/rdi-2/dsp-3.md",
		"---\nschema_version: 1\nid: dsp-3\nitem: rdi-2\nstate: accepted\ndisposition_grounds: worth acting on\n---\n\n")

	fs, err := Lint(readingSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordSchema); n != 0 {
		t.Fatalf("a configured store root nested in another store must not be an undeclared bucket, got %d finding(s): %+v", n, fs)
	}

	// The check must stay a check: a directory that is NOT a configured store
	// root is still an undeclared bucket, or the fix would have disarmed the
	// escape it was protecting.
	writeFile(t, root, "work/issues/scratch/notes.md", "notes\n")
	fs, err = Lint(readingSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join("work", "issues", "scratch"), ruleRecordSchema, "undeclared") {
		t.Fatalf("an undeclared sibling directory must still be reported: %+v", fs)
	}
}

// A store whose buckets are MINTED rather than enumerated declares them by
// grammar. A run directory and an item-keyed disposition directory are both that
// shape: nobody can list them ahead of time, and a store that could not say so
// would have to leave its whole tree undeclared.
func TestReadingRunBucketDeclaredByGrammar(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/.keep", "")
	writeFile(t, root, "work/issues/readings/rdg-1/rdi-2.md",
		"---\nschema_version: 1\nid: rdi-2\nrun: rdg-1\nmanifest: sha256:beef\nposition: detection\nregime: registrative\npattern: a stated constraint\ntension: t\nconstraint_in_play: c\nwhy_a_tension: w\n---\n\n")

	fs, err := Lint(readingSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordSchema); n != 0 {
		t.Fatalf("a minted run bucket must be declared by grammar, got %d finding(s): %+v", n, fs)
	}

	// A bucket the grammar does not describe is still undeclared: declaring by
	// grammar widens what a store can say, it does not stop it saying anything.
	writeFile(t, root, "work/issues/readings/draft-run/rdi-3.md", "---\nid: rdi-3\n---\n\n")
	fs, err = Lint(readingSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join("work", "issues", "readings", "draft-run"), ruleRecordSchema, "undeclared") {
		t.Fatalf("a bucket outside the declared grammar must be reported: %+v", fs)
	}
}

// The run's manifest is JSON, and the record scan reads markdown. It must pass
// through untouched rather than be reported as a malformed record filename —
// otherwise committing the manifest, which is what makes a run re-runnable and
// diffable, would trip the gate.
func TestManifestJSONIsSkippedByTheRecordScan(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/.keep", "")
	writeFile(t, root, "rec/readings/rdg-1/rdg-1.md",
		"---\nschema_version: 1\nid: rdg-1\n---\n\n# the run\n")
	writeFile(t, root, "rec/readings/rdg-1/manifest.json", "{\"items\": []}\n")

	fs, err := Lint(readingSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordSchema); n != 0 {
		t.Fatalf("manifest.json must be skipped by the record scan, got %d finding(s): %+v", n, fs)
	}
}

// The nested-store-root exemption must be derived from the stores the SCANNER
// knows, never from every value in the config map. Otherwise a committed config
// line naming no store at all — a prefix the code has never heard of, pointed at
// a directory inside a real store — exempts that directory from the
// undeclared-bucket blocker while nothing scans it: a lifecycle state no rule
// reads, which is the exact escape this rule exists to close. The file's own
// comment already says which lifecycle states exist is code, not config, "and a
// config that could add a bucket could also hide one".
func TestUnknownRecordStoreKeyCannotHideADirectory(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/.keep", "")
	writeFile(t, root, "work/issues/open/iss-1-a-finding.md",
		"---\nschema_version: 1\nid: iss-1\nslug: a-finding\nseverity: minor\ncategory: bug\nsource: user-observation\nfound_during: t\n---\n\nan issue\n")
	writeFile(t, root, "work/issues/anything/notes.md", "notes\n")

	stores := readingStores()
	stores["zzz"] = "work/issues/anything"
	cfg := Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			ruleRecordSchema: {Enabled: true, Severity: severityBlocker, RecordStores: stores},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join("work", "issues", "anything"), ruleRecordSchema, "undeclared") {
		t.Fatalf("a record_stores key naming no store must not exempt its directory: %+v", fs)
	}
}

// The same hole closed at the other end: a config carrying such a key is refused
// at parse, so it cannot reach the scanner at all. Two ends because they fail
// differently — a hand-built Config (this package's own callers, and its tests)
// never passes through the loader.
func TestConfigRefusesAnUnknownRecordStoreKey(t *testing.T) {
	_, err := parseConfig([]byte(`{
	  "roots": ["rec"],
	  "rules": {
	    "record_schema": {
	      "enabled": true,
	      "severity": "blocker",
	      "record_stores": {"iss": ".abcd/work/issues", "zzz": ".abcd/work/issues/anything"}
	    }
	  }
	}`))
	if err == nil {
		t.Fatal("a record_stores key naming no store must be refused at parse")
	}
	if !strings.Contains(err.Error(), "zzz") {
		t.Fatalf("the refusal must name the offending key; got %v", err)
	}
}

// malformedIssueRecord is a record missing every required property but one, so a
// store that actually scans it produces several findings and a store that skips
// it produces none. The gap between those two is what the exemption can hide.
const malformedIssueRecord = "---\nid: iss-1\n---\n\nan issue nobody validates\n"

// The nested-store-root exemption still blinded the gate through a KNOWN prefix.
// Pointing rdi at .abcd/work/issues/open passes the unknown-key check — rdi is a
// real store — marks `open` a nested root so the issue store skips it, and then
// the misdirected store ignores every file that is not rdi-N.md. A malformed
// issue record with five missing properties yields zero findings, and the config
// line that did it reads as ordinary configuration.
//
// The exemption exists to say "something else scans this directory". A bucket the
// parent ALREADY declares is scanned by the parent, so there is nothing to
// exempt: granting it there only ever removes coverage.
func TestAKnownPrefixCannotHideADeclaredBucket(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/.keep", "")
	writeFile(t, root, "work/issues/open/iss-1-a-finding.md", malformedIssueRecord)

	stores := readingStores()
	stores["rdi"] = "work/issues/open"
	cfg := Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			ruleRecordSchema: {Enabled: true, Severity: severityBlocker, RecordStores: stores},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join("work", "issues", "open", "iss-1-a-finding.md"), ruleRecordSchema, "required property") {
		t.Fatalf("a store root aimed at another store's declared bucket must not exempt it from its parent: %+v", fs)
	}
}

// The same hole closed at parse, where it can be refused outright rather than
// merely survived. Three shapes, each a way for one line to remove a store's
// coverage without naming anything that looks wrong.
func TestConfigRefusesAStoreRootInsideAnotherStoresBucket(t *testing.T) {
	cases := []struct {
		name, stores, want string
	}{
		{
			name:   "a list-declared bucket",
			stores: `{"iss": ".abcd/work/issues", "rdi": ".abcd/work/issues/open"}`,
			want:   ".abcd/work/issues/open",
		},
		{
			name:   "inside a list-declared bucket",
			stores: `{"iss": ".abcd/work/issues", "rdi": ".abcd/work/issues/open/deeper"}`,
			want:   ".abcd/work/issues/open",
		},
		{
			name:   "a grammar-declared bucket",
			stores: `{"rdi": ".abcd/work/issues/readings", "itd": ".abcd/work/issues/readings/rdg-1"}`,
			want:   ".abcd/work/issues/readings/rdg-1",
		},
		{
			name:   "two prefixes on one path",
			stores: `{"iss": ".abcd/work/issues", "rdi": ".abcd/work/issues"}`,
			want:   ".abcd/work/issues",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := parseConfig([]byte(`{
			  "roots": ["rec"],
			  "rules": {"record_schema": {"enabled": true, "severity": "blocker", "record_stores": ` + c.stores + `}}
			}`))
			if err == nil {
				t.Fatal("a store root that removes another store's coverage must be refused at parse")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("the refusal must name the offending path %q; got %v", c.want, err)
			}
		})
	}
}

// And the legitimate arrangement still loads: the reading families sit inside the
// issue store's root but beside its buckets, not in one.
func TestConfigAcceptsASiblingNestedStoreRoot(t *testing.T) {
	if _, err := parseConfig([]byte(`{
	  "roots": ["rec"],
	  "rules": {"record_schema": {"enabled": true, "severity": "blocker", "record_stores": {
	    "iss": ".abcd/work/issues",
	    "rdi": ".abcd/work/issues/readings",
	    "dsp": ".abcd/work/issues/dispositions"
	  }}}
	}`)); err != nil {
		t.Fatalf("the shipped layout must load: %v", err)
	}
}

// admissionStores is readingStores plus the two step-2 families (spc-67), laid
// out as a real ledger holds them: both nested inside the issue store's root,
// beside its buckets rather than in one.
func admissionStores() map[string]string {
	stores := readingStores()
	stores["adm"] = "work/issues/admissions"
	stores["srp"] = "work/issues/surprises"
	return stores
}

func admissionSchemaConfig() Config {
	return Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			ruleRecordSchema: {Enabled: true, Severity: severityBlocker, RecordStores: admissionStores()},
		},
	}
}

// wellFormedAdmission is the control every case below carries: a rule watched
// only failing is a rule that might refuse everything.
const wellFormedAdmission = "---\nschema_version: 1\nid: adm-2\nrun: rdg-1\nproposal: rdi-2\n" +
	"grounds: the configuration it admits is one the frame does not already hold\n---\n\n"

// admissionCorpus is the ledger every admission case starts from: the reading run
// and the widening proposal the admissions below are keyed to. The proposal has
// to be IN the corpus, because the join that keys an admission to it is resolved
// — a fixture that omitted it would be testing the schema against a record that
// admits nothing.
func admissionCorpus(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, "rec/.keep", "")
	writeFile(t, root, "work/issues/readings/rdg-1/rdi-2.md",
		"---\nschema_version: 1\nid: rdi-2\nrun: rdg-1\nmanifest: sha256:beef\nposition: widening\n"+
			"regime: constitutive\npattern: a stated constraint\n---\n\n")
	return root
}

// Grounds are the whole point of an admission: declining a proposal costs
// nothing epistemically, while admitting one is where the frame is engaged, so an
// admission recording no grounds records nothing. The refusal is armed at the
// GATE rather than at a verb — no reading has run, so nothing writes these
// records yet, and a schema no code reads would be dead scaffolding. A blank
// value and an absent one are alike here for the reason they are alike
// everywhere in this rule: no reader can make a value out of either.
func TestAdmissionRecordRequiresGrounds(t *testing.T) {
	root := admissionCorpus(t)
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-2.md", wellFormedAdmission)
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-3.md",
		"---\nschema_version: 1\nid: adm-3\nrun: rdg-1\nproposal: rdi-2\ngrounds:\n---\n\n")
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-4.md",
		"---\nschema_version: 1\nid: adm-4\nrun: rdg-1\nproposal: rdi-2\n---\n\n")

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	for _, rec := range []string{"adm-3.md", "adm-4.md"} {
		if !findingWith(fs, filepath.Join("work", "issues", "admissions", "rdg-1", rec), ruleRecordSchema, "'grounds'") {
			t.Errorf("a blank or absent grounds must be a finding on %s: %+v", rec, fs)
		}
	}
	if n := countRule(fs, ruleRecordSchema); n != 2 {
		t.Fatalf("expected exactly 2 record_schema findings (the blank and the absent grounds), got %d: %+v", n, fs)
	}
}

// The proposal is what an admission is keyed to. An admission naming none admits
// nothing in particular, so the candidate set it claims to have joined cannot be
// reconstructed from it.
func TestAdmissionRecordRequiresProposal(t *testing.T) {
	root := admissionCorpus(t)
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-2.md",
		"---\nschema_version: 1\nid: adm-2\nrun: rdg-1\ngrounds: it widens the frame\n---\n\n")

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join("work", "issues", "admissions", "rdg-1", "adm-2.md"), ruleRecordSchema, "'proposal'") {
		t.Fatalf("an admission naming no proposal must be a finding: %+v", fs)
	}
}

// The admission store's allow-list is closed, so a key outside it is a field
// nothing reads sitting in a record the gate passed.
func TestAdmissionRecordRefusesUnknownProperty(t *testing.T) {
	root := admissionCorpus(t)
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-2.md",
		"---\nschema_version: 1\nid: adm-2\nrun: rdg-1\nproposal: rdi-2\ngrounds: it widens the frame\nverdict: yes\n---\n\n")

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join("work", "issues", "admissions", "rdg-1", "adm-2.md"), ruleRecordSchema,
		"unknown frontmatter property 'verdict'") {
		t.Fatalf("a key outside the admission allow-list must be a finding: %+v", fs)
	}
}

// An admission is bucketed by RUN, exactly as the reading store is: it is
// meaningful only against the run whose proposals it admits, and nobody can list
// those directories ahead of time, so the store declares them by grammar. A
// directory the grammar does not describe is still undeclared — declaring by
// grammar widens what a store can say, it does not stop it saying anything.
func TestAdmissionStoreBucketsByRun(t *testing.T) {
	root := admissionCorpus(t)
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-2.md", wellFormedAdmission)

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordSchema); n != 0 {
		t.Fatalf("a minted run bucket must be declared by grammar, got %d finding(s): %+v", n, fs)
	}

	writeFile(t, root, "work/issues/admissions/draft-run/adm-3.md", wellFormedAdmission)
	fs, err = Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join("work", "issues", "admissions", "draft-run"), ruleRecordSchema, "undeclared") {
		t.Fatalf("a bucket outside the declared grammar must be reported: %+v", fs)
	}
}

// A surprise is its own record: separate store, separate family prefix, and a
// join key rather than the disposition's key. So neither can be filed where the
// other is read — a surprise in the disposition store would be answered for by
// the standing-disposition reader, and a disposition in the surprise store would
// be an answer nobody reads.
func TestSurpriseRecordIsNotADisposition(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/.keep", "")
	writeFile(t, root, "work/issues/surprises/srp-4.md",
		"---\nschema_version: 1\nid: srp-4\noccasioned_by: a consequence nobody predicted\n---\n\n")
	writeFile(t, root, "work/issues/dispositions/rdi-2/srp-5.md",
		"---\nschema_version: 1\nid: srp-5\noccasioned_by: a consequence nobody predicted\n---\n\n")
	writeFile(t, root, "work/issues/surprises/dsp-6.md",
		"---\nschema_version: 1\nid: dsp-6\nitem: rdi-2\nstate: accepted\ndisposition_grounds: worth acting on\n---\n\n")

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join("work", "issues", "dispositions", "rdi-2", "srp-5.md"), ruleRecordSchema,
		"not a well-formed disposition filename") {
		t.Errorf("a surprise filed in the disposition store must be refused: %+v", fs)
	}
	if !findingWith(fs, filepath.Join("work", "issues", "surprises", "dsp-6.md"), ruleRecordSchema,
		"not a well-formed surprise filename") {
		t.Errorf("a disposition filed in the surprise store must be refused: %+v", fs)
	}
	if n := countRule(fs, ruleRecordSchema); n != 2 {
		t.Fatalf("expected exactly 2 findings (the two misfiled records), got %d: %+v", n, fs)
	}
}

// occasioned_by is the surprise's whole join. Where it names a RECORD, that
// record must be in the corpus: a join naming nothing joins nothing, and the
// surprise then sits beside the thing it claims to have arisen from with no way
// back to it. Prose naming a consequence is legitimate and stays silent — a
// surprise is keyed to whatever occasioned it, and not everything that occasions
// one has an id.
func TestSurpriseOccasionedByResolves(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/.keep", "")
	writeFile(t, root, "work/issues/readings/rdg-1/rdi-2.md",
		"---\nschema_version: 1\nid: rdi-2\nrun: rdg-1\nmanifest: sha256:beef\nposition: detection\nregime: registrative\npattern: a stated constraint\ntension: t\nconstraint_in_play: c\nwhy_a_tension: w\n---\n\n")
	writeFile(t, root, "work/issues/dispositions/rdi-2/dsp-3.md",
		"---\nschema_version: 1\nid: dsp-3\nitem: rdi-2\nstate: accepted\ndisposition_grounds: worth acting on\n---\n\n")
	writeFile(t, root, "work/issues/surprises/srp-4.md",
		"---\nschema_version: 1\nid: srp-4\noccasioned_by: rdi-2\n---\n\n")
	writeFile(t, root, "work/issues/surprises/srp-5.md",
		"---\nschema_version: 1\nid: srp-5\noccasioned_by: the consequence nobody predicted\n---\n\n")
	writeFile(t, root, "work/issues/surprises/srp-6.md",
		"---\nschema_version: 1\nid: srp-6\noccasioned_by: rdi-9999\n---\n\n")

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join("work", "issues", "surprises", "srp-6.md"), ruleRecordSchema, "rdi-9999") {
		t.Errorf("an occasioned_by naming no record in the corpus must be a finding: %+v", fs)
	}
	if n := countRule(fs, ruleRecordSchema); n != 1 {
		t.Fatalf("expected exactly 1 finding (the dangling join), got %d: %+v", n, fs)
	}
}

// A required property is missing when its value is empty ONCE THE YAML SCALAR IS
// READ, not when its raw bytes happen to be empty. `grounds: ""` carries five
// bytes and no value: the reader that validates before it reads makes nothing out
// of it, so the record is skipped and invisible to every surface of its family —
// the exact defect checkRecordRequiredFields exists to catch, walked straight past
// because the check tested the quotes rather than what they contain.
//
// The whole class is pinned here, not the reported spelling alone: an empty
// double-quoted value, an empty single-quoted one, whitespace inside quotes,
// whitespace with no quotes, and the explicit nulls. A fix that answered `""` and
// let `'  '` through would be the same bug with a narrower mouth.
func TestRequiredFieldIsEmptyOnceTheScalarIsRead(t *testing.T) {
	empties := map[string]string{
		"adm-3": `""`,
		"adm-4": `''`,
		"adm-5": `"   "`,
		"adm-6": `'	'`,
		"adm-7": `~`,
		"adm-8": `null`,
	}
	root := admissionCorpus(t)
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-2.md", wellFormedAdmission)
	for id, spelling := range empties {
		writeFile(t, root, "work/issues/admissions/rdg-1/"+id+".md",
			"---\nschema_version: 1\nid: "+id+"\nrun: rdg-1\nproposal: rdi-2\ngrounds: "+spelling+"\n---\n\n")
	}

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	for id := range empties {
		if !findingWith(fs, filepath.Join("work", "issues", "admissions", "rdg-1", id+".md"), ruleRecordSchema, "'grounds'") {
			t.Errorf("grounds spelled %s carries no value and must be a finding on %s.md: %+v", empties[id], id, fs)
		}
	}
	if n := countRule(fs, ruleRecordSchema); n != len(empties) {
		t.Fatalf("expected exactly %d record_schema findings (one per empty spelling), got %d: %+v",
			len(empties), n, fs)
	}
}

// The sibling shape, and the reason the fix is at the ONE place this rule decides
// emptiness rather than in the admission store's branch of it: the issue ledger
// has carried the identical gap since required fields were declared, so a
// committed `found_during: ""` is lint-green while capture's reader refuses the
// record and skips it — invisible to `capture list`, `capture status` and every
// other ledger surface while it still sits in `open/`.
func TestQuotedEmptyRequiredFieldIsRefusedInTheIssueStore(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/.keep", "")
	writeFile(t, root, "work/issues/open/iss-2-a-finding.md",
		"---\nschema_version: 1\nid: \"iss-2\"\nslug: \"a-finding\"\nseverity: \"minor\"\n"+
			"category: \"bug\"\nsource: \"impl-review\"\nfound_during: \"\"\n---\n\nbody\n")

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join("work", "issues", "open", "iss-2-a-finding.md"), ruleRecordSchema, "'found_during'") {
		t.Fatalf("a quoted-empty found_during must be a finding: %+v", fs)
	}
}

// The block-scalar neighbour. `grounds: |` puts the value on the lines BELOW the
// key, so the same-line scanner reads `|` — a non-empty byte that is not a value
// at all. A block carrying text is present; a block carrying nothing is the empty
// string, which is the same absence every spelling above is, and must be refused
// on the same terms.
func TestBlockScalarRequiredFieldIsJudgedByWhatTheBlockHolds(t *testing.T) {
	root := admissionCorpus(t)
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-2.md",
		"---\nschema_version: 1\nid: adm-2\nrun: rdg-1\nproposal: rdi-2\ngrounds: |\n  the frame does not already hold it\n---\n\n")
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-3.md",
		"---\nschema_version: 1\nid: adm-3\nrun: rdg-1\nproposal: rdi-2\ngrounds: >-\n---\n\n")

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join("work", "issues", "admissions", "rdg-1", "adm-3.md"), ruleRecordSchema, "'grounds'") {
		t.Errorf("a block scalar holding nothing is an empty value and must be a finding: %+v", fs)
	}
	if n := countRule(fs, ruleRecordSchema); n != 1 {
		t.Fatalf("a block scalar holding text is a value; expected exactly 1 finding, got %d: %+v", n, fs)
	}
}

// `proposal` is the admission's whole join, and it was the one join this rule
// did not resolve: a surprise's occasioned_by had to name a record in the corpus
// while an admission could name anything at all and pass. An admission naming no
// record admits nothing in particular, and the candidate set it claims to have
// joined cannot be reconstructed from it — which is the same defect, so it is
// the same check, declared by the store rather than written twice.
func TestAdmissionProposalResolves(t *testing.T) {
	root := admissionCorpus(t)
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-3.md",
		"---\nschema_version: 1\nid: adm-3\nrun: rdg-1\nproposal: rdi-2\ngrounds: it widens the frame\n---\n\n")
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-4.md",
		"---\nschema_version: 1\nid: adm-4\nrun: rdg-1\nproposal: rdi-9999\ngrounds: it widens the frame\n---\n\n")

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join("work", "issues", "admissions", "rdg-1", "adm-4.md"), ruleRecordSchema, "rdi-9999") {
		t.Errorf("a proposal naming no record in the corpus must be a finding: %+v", fs)
	}
	if n := countRule(fs, ruleRecordSchema); n != 1 {
		t.Fatalf("expected exactly 1 finding (the dangling join), got %d: %+v", n, fs)
	}
}

// An admission is bucketed by the run whose candidate set it joins AND carries
// that run as a field, so the record states one fact twice. A disagreement means
// the record contradicts itself about which set it joined, and the report — which
// keys the admitted set on the pair — then honours neither claim. Left unnamed,
// the admission would simply admit nothing, silently.
func TestAdmissionRunFieldMustAgreeWithItsBucket(t *testing.T) {
	root := admissionCorpus(t)
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-2.md", wellFormedAdmission)
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-3.md",
		"---\nschema_version: 1\nid: adm-3\nrun: rdg-7\nproposal: rdi-2\ngrounds: it widens the frame\n---\n\n")

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join("work", "issues", "admissions", "rdg-1", "adm-3.md"), ruleRecordSchema, "rdg-7") {
		t.Errorf("a run field contradicting its bucket must be a finding: %+v", fs)
	}
	if n := countRule(fs, ruleRecordSchema); n != 1 {
		t.Fatalf("expected exactly 1 finding (the contradiction), got %d: %+v", n, fs)
	}
}

// A block-scalar header may carry a trailing comment, and the header is still
// not the value. `grounds: | # nothing` over an empty block is the empty string
// exactly as a bare `grounds: |` is — so a pattern that read only the bare
// spellings left four legal spellings of the reported defect passing the gate,
// and each of them silences the report that the admission leg exists to make.
func TestACommentedBlockScalarHeaderIsJudgedByItsBlock(t *testing.T) {
	headers := map[string]string{
		"adm-3": "| # nothing",
		"adm-4": "|- # nothing",
		"adm-5": ">+ # nothing",
		"adm-6": "|2 # nothing",
	}
	root := admissionCorpus(t)
	for id, header := range headers {
		writeFile(t, root, "work/issues/admissions/rdg-1/"+id+".md",
			"---\nschema_version: 1\nid: "+id+"\nrun: rdg-1\nproposal: rdi-2\ngrounds: "+header+"\n---\n\n")
	}

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	for id, header := range headers {
		if !findingWith(fs, filepath.Join("work", "issues", "admissions", "rdg-1", id+".md"), ruleRecordSchema, "'grounds'") {
			t.Errorf("grounds spelled %q holds nothing and must be a finding on %s.md: %+v", header, id, fs)
		}
	}
	if n := countRule(fs, ruleRecordSchema); n != len(headers) {
		t.Fatalf("expected exactly %d findings, got %d: %+v", len(headers), n, fs)
	}
}

// Inside a literal block a `#` is CONTENT, not a comment. The walker the block
// look-ahead reuses was written for block SEQUENCES, where skipping comments is
// right — reading a scalar with it makes a legal record's whole value vanish and
// reports a required property missing that the file plainly carries. A rule whose
// design argument is that it must never make a confident false statement cannot
// make this one.
func TestABlockScalarWhoseOnlyLineIsHashLedIsAValue(t *testing.T) {
	root := admissionCorpus(t)
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-3.md",
		"---\nschema_version: 1\nid: adm-3\nrun: rdg-1\nproposal: rdi-2\ngrounds: |\n"+
			"  # the frame does not already hold the configuration it admits\n---\n\n")

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordSchema); n != 0 {
		t.Fatalf("a block scalar whose content begins with '#' is a value, got %d finding(s): %+v", n, fs)
	}
}

// issueScalar strips ONE surrounding quote pair, not every quote character at
// either end. A required value that is itself two apostrophes inside double
// quotes is a value, and eating it down to nothing turns a present property into
// a blocker that names a field the record carries.
func TestARequiredValueMadeOfQuoteCharactersIsPresent(t *testing.T) {
	root := admissionCorpus(t)
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-3.md",
		"---\nschema_version: 1\nid: adm-3\nrun: rdg-1\nproposal: rdi-2\ngrounds: \"''\"\n---\n\n")

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordSchema); n != 0 {
		t.Fatalf("a value made of quote characters is still a value, got %d finding(s): %+v", n, fs)
	}
}

// The filename<->slug and filename<->id checks skip an ABSENT value and delegate
// the emptiness question to checkRecordRequiredFields — which is sound only where
// the store declares a required set. The intent and spec stores declare none, so
// widening what those two checks read as "absent" sent the delegation nowhere and
// left `slug: ""` and `id: ""` green in every rule. Absence here means NOT
// WRITTEN; an empty value is a value that disagrees.
func TestQuotedEmptyIdAndSlugStillDisagreeWithTheFilename(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/.keep", "")
	writeFile(t, root, "rec/specs/open/spc-12-something-else.md",
		"---\nid: \"\"\nslug: \"\"\nintent_id: null\n---\n# spec\n")

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join("rec", "specs", "open", "spc-12-something-else.md"), ruleRecordSchema, "something-else") {
		t.Errorf("a quoted-empty slug disagrees with the filename and must be a finding: %+v", fs)
	}
	if !findingWith(fs, filepath.Join("rec", "specs", "open", "spc-12-something-else.md"), ruleRecordSchema, "spc-12") {
		t.Errorf("a quoted-empty id disagrees with the filename and must be a finding: %+v", fs)
	}

	// The control the delegation rests on: a record carrying NEITHER property is
	// silent here, because absence is not disagreement.
	other := t.TempDir()
	writeFile(t, other, "rec/.keep", "")
	writeFile(t, other, "rec/specs/open/spc-13-something-else.md", "---\nintent_id: null\n---\n# spec\n")
	fs, err = Lint(schemaConfig(), other)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordSchema); n != 0 {
		t.Fatalf("a record carrying neither id nor slug is a different question, got %d finding(s): %+v", n, fs)
	}
}

// The direction an empty cross-reference is read in, pinned because widening the
// absence test changed it silently. `superseded_by: ""` is the quoted spelling of
// the `superseded_by: null` the record writes everywhere: an optional link left
// unset, not a malformed handle. Reporting it as "not a record handle" would put
// a blocker on a record that declares no supersession at all.
func TestQuotedEmptySupersededByIsUnsetNotMalformed(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/.keep", "")
	writeFile(t, root, "rec/decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md",
		"---\nid: adr-22\nslug: bundled-deps-as-pluggable-adapters\nsupersedes: []\nsuperseded_by: \"\"\nrelated_adrs: []\n---\n# ADR-22\n")

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordSchema); n != 0 {
		t.Fatalf("an empty superseded_by is unset, got %d finding(s): %+v", n, fs)
	}
}

// The scalar reader must not trim INSIDE the quotes. capture's parse path trims
// the raw value before decoding (parse.go, `rest := strings.TrimSpace(...)`) and
// then strips the quotes without trimming again, so `severity: "  minor  "`
// reaches validateStrict as `  minor  `, fails the enum map lookup outright, and
// the record is refused and skipped — invisible to every capture surface while it
// still sits in the ledger. A gate that trimmed the padding away would call that
// record clean, which is the exact silence this rule exists to break.
//
// Emptiness is the one question that does trim, and it is asked separately:
// a value of nothing but padding carries nothing whatever a reader does with it.
func TestQuotedPaddingIsNotTrimmedOutOfAValueTheReaderRefuses(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/.keep", "")
	writeFile(t, root, "work/issues/open/iss-2-a-finding.md",
		"---\nschema_version: 1\nid: \"iss-2\"\nslug: \"a-finding\"\nseverity: \"  minor  \"\n"+
			"category: \"bug\"\nsource: \"impl-review\"\nfound_during: \"a session\"\n---\n\nbody\n")

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join("work", "issues", "open", "iss-2-a-finding.md"), ruleRecordSchema, "severity") {
		t.Fatalf("capture refuses a padded enum value, so the gate must too: %+v", fs)
	}
}

// An ABSENT property and an EMPTY one are both findings, and they are not the
// same finding — WHERE the store's reader validates before it reads. The issue
// ledger's does (capture's validateStrict), and it refuses a record that omits a
// required property outright and skips it, so the missing-property message's
// account of the consequence is true there and true of every field, because that
// loop asks nothing about which property it is.
//
// A BLANK has no such account. The same reader type-checks a present value
// without judging it while the checks either side of it DO judge, so what a blank
// does to the record is a property of the FIELD. `found_during` is a field no leg
// judges, so the finding states what a blank IS and claims nothing about the
// reader in either direction: neither the refusal that would send the author
// looking for a record that is being read, nor the acceptance that would send
// them looking for one that is not. The parity across all forty-two
// combinations is pinned against the reader itself in core/capture
// (TestBlankRequiredPropertyFindingsMatchTheReadersVerdict); the store half of
// the question is TestMissingPropertyClaimsNoReaderWhereTheStoreHasNone.
func TestAbsentAndEmptyRequiredPropertiesGiveDifferentReasons(t *testing.T) {
	root := t.TempDir()
	seedRecRoot(t, root)
	writeFile(t, root, "work/issues/open/iss-3-absent.md",
		"---\nschema_version: 1\nid: iss-3\nslug: absent\nseverity: minor\ncategory: bug\n"+
			"source: user-observation\n---\n\nan issue\n")
	writeFile(t, root, "work/issues/open/iss-4-blank.md",
		"---\nschema_version: 1\nid: iss-4\nslug: blank\nseverity: minor\ncategory: bug\n"+
			"source: user-observation\nfound_during: \"\"\n---\n\nan issue\n")

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	absent := filepath.Join("work", "issues", "open", "iss-3-absent.md")
	empty := filepath.Join("work", "issues", "open", "iss-4-blank.md")
	if !findingWith(fs, absent, ruleRecordSchema, "skipped") {
		t.Errorf("this reader DOES refuse a record that omits a required property, and the finding should say so: %+v", fs)
	}
	if findingWith(fs, empty, ruleRecordSchema, "skipped") {
		t.Errorf("no leg judges a blank found_during, so the finding must not claim the record is skipped: %+v", fs)
	}
	if findingWith(fs, empty, ruleRecordSchema, "this record is read") {
		t.Errorf("no leg judges a blank found_during, so the finding must not claim the record is read either: %+v", fs)
	}
	if !findingWith(fs, empty, ruleRecordSchema, "'found_during'") {
		t.Errorf("a blank required property is still a finding: %+v", fs)
	}
	if n := countRule(fs, ruleRecordSchema); n != 2 {
		t.Fatalf("expected exactly 2 findings, got %d: %+v", n, fs)
	}
}

// An admission is meaningful only against the run whose proposals it admits, and
// the outstanding report keys the admitted set on the (run, proposal) pair — so an
// admission filed under one run that names an item belonging to another is keyed
// on a pair nothing ever queries. It admits nothing, the proposal it names goes on
// being reported as unadmitted, and no line says an answer was written: exactly
// the "record that quietly stops counting" this rule exists to make loud.
//
// The run-field agreement does not cover it. That check enforces FIELD ==
// DIRECTORY; this is DIRECTORY == THE TARGET'S OWN RUN, and a record can satisfy
// the first while failing the second — which is what the fixture below is
// (iss-2608301327013320).
func TestAdmissionNamingAnotherRunsProposalIsAFinding(t *testing.T) {
	root := admissionCorpus(t)
	// A second run with its own proposal. rdi-7 is filed under rdg-9 and belongs
	// to that run alone.
	writeFile(t, root, "work/issues/readings/rdg-9/rdi-7.md",
		"---\nschema_version: 1\nid: rdi-7\nrun: rdg-9\nmanifest: sha256:feed\nposition: widening\n"+
			"regime: constitutive\npattern: a stated constraint\n---\n\n")
	// The control: an admission naming the proposal in its OWN run stays silent.
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-2.md", wellFormedAdmission)
	// The fault: filed under rdg-1, carrying run: rdg-1 (so the run-field
	// agreement is satisfied), naming a proposal that lives under rdg-9.
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-3.md",
		"---\nschema_version: 1\nid: adm-3\nrun: rdg-1\nproposal: rdi-7\n"+
			"grounds: the configuration it admits is one the frame does not already hold\n---\n\n")

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	crossRun := filepath.Join("work", "issues", "admissions", "rdg-1", "adm-3.md")
	if !findingWith(fs, crossRun, ruleRecordSchema, "rdg-9") {
		t.Errorf("an admission naming another run's proposal must be a finding naming the run it reaches into: %+v", fs)
	}
	if findingWith(fs, filepath.Join("work", "issues", "admissions", "rdg-1", "adm-2.md"), ruleRecordSchema, "") {
		t.Errorf("an admission naming a proposal in its own run must stay silent: %+v", fs)
	}
	if n := countRule(fs, ruleRecordSchema); n != 1 {
		t.Fatalf("expected exactly 1 finding (the cross-run join), got %d: %+v", n, fs)
	}
}

// The blocker states the CONSEQUENCE the walk establishes and nothing more. Its
// previous spellings asserted a mechanism instead — that ids in the family are
// minted per bucket and collide across them — which this same rule refuses as a
// fault 440 lines away (the duplicate-ordinal leg fires on a cross-run rdi), and
// which mintUnusedItemID had already made false before this branch began: it
// probes every run under the ledger lock and redraws on a hit
// (iss-2608300227228575).
//
// Scoping that claim to the one family where it was believed true is how the
// round before last closed it (iss-2608301411017768), which is exactly why it
// survived to be found again. So this test pins the WORDING: a message that
// reintroduces the mechanism fails here, rather than being rediscovered as a
// fresh finding a round later (iss-2608301519253368).
func TestBucketJoinBlockerAssertsNoIDCollision(t *testing.T) {
	root := admissionCorpus(t)
	writeFile(t, root, "work/issues/readings/rdg-9/rdi-7.md",
		"---\nschema_version: 1\nid: rdi-7\nrun: rdg-9\nmanifest: sha256:feed\nposition: widening\n"+
			"regime: constitutive\npattern: a stated constraint\n---\n\n")
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-3.md",
		"---\nschema_version: 1\nid: adm-3\nrun: rdg-1\nproposal: rdi-7\n"+
			"grounds: the configuration it admits is one the frame does not already hold\n---\n\n")

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	rel := filepath.Join("work", "issues", "admissions", "rdg-1", "adm-3.md")
	if !findingWith(fs, rel, ruleRecordSchema, "keyed on a pair nothing ever queries") {
		t.Fatalf("the blocker must state the consequence the walk establishes: %+v", fs)
	}
	for _, mechanism := range []string{"minted per", "collide", "collision", "allocate the same"} {
		if findingWith(fs, rel, ruleRecordSchema, mechanism) {
			t.Errorf("the blocker asserts a minting mechanism (%q) that is not the case: %+v", mechanism, fs)
		}
	}
}

// One rule must give ONE answer about one pruned handle. The ADR lifecycle prunes
// a superseded record once its successor lands and keeps the trace in the
// successor's `supersedes`, so such a handle resolves to that declaration rather
// than to a file — which the cross-reference loop has always read that way. The
// join check did not, so `related_adrs: [adr-5]` was accepted on one record while
// `occasioned_by: adr-5` was a blocker on the next (iss-2608301327012166).
func TestJoinsResolveARetiredHandleTheWayCrossReferencesDo(t *testing.T) {
	root := admissionCorpus(t)
	// adr-25 declares it replaced adr-5, which is therefore pruned rather than
	// missing. Its ordinal also puts adr-5 below the store's high-water mark, so
	// the retirement is one the store could have issued.
	writeFile(t, root, "rec/decisions/adrs/0025-host-delegated.md",
		"---\nid: adr-25\nsupersedes: [adr-5]\nsuperseded_by: null\nrelated_adrs: [adr-5]\n---\n# ADR-25\n")
	writeFile(t, root, "work/issues/surprises/srp-4.md",
		"---\nschema_version: 1\nid: srp-4\noccasioned_by: adr-5\n---\n\n")
	// The control: a handle nothing declares retired is still a finding, so the
	// skip is a resolution and not a blanket amnesty.
	writeFile(t, root, "work/issues/surprises/srp-5.md",
		"---\nschema_version: 1\nid: srp-5\noccasioned_by: adr-6\n---\n\n")

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if findingWith(fs, filepath.Join("work", "issues", "surprises", "srp-4.md"), ruleRecordSchema, "adr-5") {
		t.Errorf("a pruned handle resolves through the successor's declaration in a join too: %+v", fs)
	}
	if !findingWith(fs, filepath.Join("work", "issues", "surprises", "srp-5.md"), ruleRecordSchema, "adr-6") {
		t.Errorf("a handle nothing declares retired is still a finding: %+v", fs)
	}
	if n := countRule(fs, ruleRecordSchema); n != 1 {
		t.Fatalf("expected exactly 1 finding (the undeclared adr-6), got %d: %+v", n, fs)
	}
}

// The bucket comparison speaks about a mechanism that belongs to ONE family: the
// outstanding report keys reading items on the run they sit in, so an admission
// reaching into another run is keyed on a pair nothing queries. That is not true
// of the issue ledger — no reader keys an issue on its status directory, and the
// outstanding report walks reading items only and never reports an issue at all.
// Inventing a consequence the operator then goes looking for is the confident
// false statement this rule must not make (iss-2608301411017768).
//
// So a cross-family target is not judged by THAT message. It is still a fault —
// `proposal` names the reading item an admission admits, and an issue is not one
// — and the spelling leg says so in the terms that are true of it: the value is
// not the spelling the reader of that family matches.
func TestABucketJoinDoesNotDescribeThePairMechanismOverAnotherFamily(t *testing.T) {
	root := admissionCorpus(t)
	writeFile(t, root, "work/issues/open/iss-42-a-finding.md", validIssue("iss-42", "a-finding"))
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-3.md",
		"---\nschema_version: 1\nid: adm-3\nrun: rdg-1\nproposal: iss-42\n"+
			"grounds: the configuration it admits is one the frame does not already hold\n---\n\n")

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	rel := filepath.Join("work", "issues", "admissions", "rdg-1", "adm-3.md")
	if findingWith(fs, rel, ruleRecordSchema, "keys it on the pair") {
		t.Errorf("no reader keys an issue on its status directory, and the report never names an issue; "+
			"the bucket leg must not describe that mechanism over a target of another family: %+v", fs)
	}
	if !findingWith(fs, rel, ruleRecordSchema, "not a reading item handle") {
		t.Errorf("an issue is not the reading item a proposal names, and the spelling leg must say so: %+v", fs)
	}
	if n := countRule(fs, ruleRecordSchema); n != 1 {
		t.Fatalf("expected exactly 1 finding (the spelling), got %d: %+v", n, fs)
	}
}

// The six spellings a NUMERIC resolution let through, and the padded one that
// only the target file can judge. checkRecordJoins parsed the
// value with a case-insensitive handle pattern and compared it as a number, while
// the report that reads the same field keys it as the string it is written as — so
// an upper-cased, mixed-cased or zero-padded proposal resolved and matched its
// bucket, and a spaced or prose one fell through the "prose is legitimate" branch.
// Either way the gate was green and the admission admitted nothing, permanently
// and silently (iss-2608301519255871).
//
// A join that declares sameBucketAs is one whose value the schema says IS a handle
// of that family, so the spelling is judged before anything is resolved: the
// family's own prefix, lower case, unpadded, and nothing around it. The seventh
// case carried here is the cross-family handle, which is not one of the six but
// falls in the same set and is refused on the same terms — the case
// TestABucketJoinDoesNotDescribeThePairMechanismOverAnotherFamily reads for its
// wording.
func TestABucketJoinRefusesEverySpellingButTheFamilysOwn(t *testing.T) {
	// The value alone settles these: no prefix of the family, or something around
	// it.
	malformed := map[string]string{
		"adm-3": "RDI-2",
		"adm-4": "Rdi-2",
		"adm-6": "\"rdi-2 \"",
		"adm-7": "\" rdi-2\"",
		"adm-8": "the item we discussed",
		"adm-9": "iss-42",
	}
	// And the family's own prefix with NOTHING after it. `rdi-` carries the prefix
	// and the hyphen, so strings.CutPrefix succeeds and leaves an empty rest — which
	// a digit loop over no bytes accepts. Without spellsHandleOf's own `rest == ""`
	// the value spells a handle, resolves to nothing, and takes the silence prose is
	// owed: the guard survived its own deletion until this case was carried here
	// (iss-2608301656202623).
	malformed["adm-10"] = "rdi-"
	// And this one the target's FILE settles: rdi-2.md carries no padding, so the
	// padded spelling is not a name the family's reader ever matches.
	const padded = "adm-5"
	root := admissionCorpus(t)
	writeFile(t, root, "work/issues/open/iss-42-a-finding.md", validIssue("iss-42", "a-finding"))
	// The control every case is read against: the family's own spelling is silent,
	// so the leg is a spelling check and not a refusal of everything.
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-2.md", wellFormedAdmission)
	writeFile(t, root, "work/issues/admissions/rdg-1/"+padded+".md",
		"---\nschema_version: 1\nid: "+padded+"\nrun: rdg-1\nproposal: rdi-02\n"+
			"grounds: the frame does not already hold it\n---\n\n")
	for id, proposal := range malformed {
		writeFile(t, root, "work/issues/admissions/rdg-1/"+id+".md",
			"---\nschema_version: 1\nid: "+id+"\nrun: rdg-1\nproposal: "+proposal+"\n"+
				"grounds: the frame does not already hold it\n---\n\n")
	}

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	for id, proposal := range malformed {
		if !findingWith(fs, filepath.Join("work", "issues", "admissions", "rdg-1", id+".md"),
			ruleRecordSchema, "not a reading item handle") {
			t.Errorf("proposal spelled %s admits nothing and must be a finding on %s.md: %+v", proposal, id, fs)
		}
	}
	if !findingWith(fs, filepath.Join("work", "issues", "admissions", "rdg-1", padded+".md"),
		ruleRecordSchema, "is filed as 'rdi-2.md'") {
		t.Errorf("a padded proposal against an unpadded file must be a finding naming that file: %+v", fs)
	}
	if n := countRule(fs, ruleRecordSchema); n != len(malformed)+1 {
		t.Fatalf("expected exactly %d findings (one per spelling), got %d: %+v", len(malformed)+1, n, fs)
	}
}

// The spelling leg names the target family's RECORD KIND, which it reads off the
// store the family names. A join declaring a family no store declares would render
// that message with an empty noun, so the lookup's success is pinned here rather
// than guarded by a branch no fixture can enter.
func TestEveryJoinFamilyNamesADeclaredStore(t *testing.T) {
	for _, store := range recordStores {
		for _, join := range store.joins {
			if join.sameBucketAs == "" {
				continue
			}
			target, ok := storeByPrefix(join.sameBucketAs)
			if !ok {
				t.Errorf("%s's %s join declares family %q, which no store declares",
					store.prefix, join.field, join.sameBucketAs)
				continue
			}
			if target.noun == "" {
				t.Errorf("store %q declares no noun, so the %s join's message would name nothing",
					target.prefix, join.field)
			}
		}
	}
}

// The position leg quotes the position it requires, and a target's own position is
// compared against it as a string. A declared position outside the closed set
// would refuse every target the family can hold while quoting a position no record
// may carry, so membership is pinned rather than trusted.
func TestEveryJoinTargetPositionIsADeclaredPosition(t *testing.T) {
	for _, store := range recordStores {
		for _, join := range store.joins {
			if join.targetPosition == "" {
				continue
			}
			if !inSet(join.targetPosition, issueschema.Positions) {
				t.Errorf("%s's %s join requires position %q, which is not one of %v",
					store.prefix, join.field, join.targetPosition, issueschema.Positions)
			}
			// The leg reads the target's position only where the target's FILENAME is
			// a bare handle, and that test is computed inside the sameBucketAs branch.
			// A join declaring a position without a family would therefore be inert,
			// with no fixture able to enter it and nothing saying so.
			if join.sameBucketAs == "" {
				t.Errorf("%s's %s join requires position %q but declares no family, so the leg "+
					"that reads the position never runs", store.prefix, join.field, join.targetPosition)
			}
		}
	}
}

// The missing-property message names a reader that refuses the record and skips
// it. Two stores have such a reader — the issue ledger (capture's validateStrict)
// and the ADR store (the dispatcher confirms the frontmatter id) — and for those
// the account is true. The admission store has none: its only reader in the tree
// honours a record carrying nothing but its run and its proposal
// (TestAdmissionReaderHonoursARecordMissingRequiredProperties), and the surprise
// store has no reader at all.
//
// So the clause is a property of the STORE, exactly as the blank branch's is a
// property of the field. A record is still a finding either way — a required
// property exists because the record has to state something — but a gate that
// names a refusal nobody performs sends the author looking for a record that is
// being read (iss-2608301411010342).
func TestMissingPropertyClaimsNoReaderWhereTheStoreHasNone(t *testing.T) {
	root := admissionCorpus(t)
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-3.md",
		"---\nschema_version: 1\nid: adm-3\nrun: rdg-1\nproposal: rdi-2\n---\n\n")
	writeFile(t, root, "work/issues/surprises/srp-4.md",
		"---\nschema_version: 1\nid: srp-4\n---\n\n")

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ rel, prop string }{
		{filepath.Join("work", "issues", "admissions", "rdg-1", "adm-3.md"), "'grounds'"},
		{filepath.Join("work", "issues", "surprises", "srp-4.md"), "'occasioned_by'"},
	} {
		if !findingWith(fs, c.rel, ruleRecordSchema, c.prop) {
			t.Errorf("an omitted required property is still a finding on %s: %+v", c.rel, fs)
		}
		for _, claim := range []string{"skipped", "invisible"} {
			if findingWith(fs, c.rel, ruleRecordSchema, claim) {
				t.Errorf("this store's reader performs no such refusal, so the finding on %s must not claim %q: %+v",
					c.rel, claim, fs)
			}
		}
	}
}

// The closed-schema leg and the duplicate-key leg make the SAME account of the
// consequence the missing-property leg makes: the reader refuses the record and
// skips it, so it is invisible to every surface of its family. That account is a
// property of the STORE, and the store declares it — readerFailsClosed for the
// two legs that read the record's PROPERTIES, readerRefusesDuplicateKey for the
// duplicate — rather than any leg assuming it.
//
// The admission store's only reader COUNTS a record carrying an unknown key or a
// duplicated one, and no reader of surprise records exists anywhere in the tree,
// so for those two stores both legs must state what a closed schema and a
// duplicated key ARE — a key nothing reads, and a second line the lenient scanner
// drops — and not a refusal nobody performs (iss-2608301519254418).
//
// The ADR store is why the duplicate is declared apart. Its dispatcher DOES
// validate the record before it renders it, and reads the frontmatter with the
// lenient scanner: an ADR carrying `status` twice renders, with the first value
// and a nil error, so one flag across both questions made this store claim a
// refusal nobody performs (iss-2608301656200729). Both halves of the split run
// here, on the same store.
//
// The issue store is the control on the same run: its reader does perform the
// refusal on both, so it keeps the claim, and the gate is not simply mute
// everywhere.
func TestClosedSchemaAndDuplicateKeyClaimNoReaderWhereTheStoreHasNone(t *testing.T) {
	root := admissionCorpus(t)
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-3.md",
		"---\nschema_version: 1\nid: adm-3\nrun: rdg-1\nproposal: rdi-2\n"+
			"grounds: it widens the frame\nverdict: yes\n---\n\n")
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-4.md",
		"---\nschema_version: 1\nid: adm-4\nrun: rdg-1\nproposal: rdi-2\n"+
			"grounds: it widens the frame\ngrounds: and again\n---\n\n")
	writeFile(t, root, "work/issues/surprises/srp-5.md",
		"---\nschema_version: 1\nid: srp-5\noccasioned_by: rdi-2\nverdict: yes\n---\n\n")
	writeFile(t, root, "work/issues/surprises/srp-6.md",
		"---\nschema_version: 1\nid: srp-6\noccasioned_by: rdi-2\noccasioned_by: rdi-2\n---\n\n")
	// The control: the issue ledger's reader validates before it reads, so the
	// claim is true of it and must survive.
	writeFile(t, root, "work/issues/open/iss-42-a-finding.md",
		strings.Replace(validIssue("iss-42", "a-finding"), "found_during: t\n", "found_during: t\nverdict: yes\n", 1))
	writeFile(t, root, "work/issues/open/iss-43-a-finding.md",
		strings.Replace(validIssue("iss-43", "a-finding"), "found_during: t\n", "found_during: t\nseverity: major\n", 1))
	// The ADR store is the fourth, and it parts the two legs. Its reader confirms
	// the frontmatter id before it will render the record, so the MISSING-PROPERTY
	// claim is true of it — but record.readRecordHead reads with frontmatter.Fields,
	// the lenient scanner, and no ADR reader anywhere refuses a duplicated key: an
	// ADR carrying `status` twice renders, with the first value and a nil error. The
	// duplicate-key leg must therefore not claim the refusal on this store's behalf
	// (iss-2608301656200729). Its closed-schema leg is not exercised: the store
	// declares no knownFields, so checkRecordUnknownFields returns before it speaks.
	writeFile(t, root, "rec/decisions/adrs/0009-a-decision.md",
		"---\nid: adr-9\nstatus: accepted\nstatus: draft\n---\n\n# A decision\n")
	writeFile(t, root, "rec/decisions/adrs/0010-another-decision.md",
		"---\nstatus: accepted\n---\n\n# Another decision\n")

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ rel, quote string }{
		{filepath.Join("work", "issues", "admissions", "rdg-1", "adm-3.md"), "unknown frontmatter property 'verdict'"},
		{filepath.Join("work", "issues", "admissions", "rdg-1", "adm-4.md"), "duplicate top-level key 'grounds'"},
		{filepath.Join("work", "issues", "surprises", "srp-5.md"), "unknown frontmatter property 'verdict'"},
		{filepath.Join("work", "issues", "surprises", "srp-6.md"), "duplicate top-level key 'occasioned_by'"},
		{filepath.Join("rec", "decisions", "adrs", "0009-a-decision.md"), "duplicate top-level key 'status'"},
	} {
		if !findingWith(fs, c.rel, ruleRecordSchema, c.quote) {
			t.Errorf("the finding on %s must still be raised: %+v", c.rel, fs)
		}
		for _, claim := range []string{"refuses", "skipped", "skips", "invisible"} {
			if findingWith(fs, c.rel, ruleRecordSchema, claim) {
				t.Errorf("this store's reader performs no such refusal, so the finding on %s must not claim %q: %+v",
					c.rel, claim, fs)
			}
		}
	}
	for _, c := range []struct{ rel, claim string }{
		{filepath.Join("work", "issues", "open", "iss-42-a-finding.md"), "invisible"},
		// The duplicate-key refusal names the LEDGER reader and the verbs that skip
		// the record, rather than "every issue surface": the release cut reads the
		// same file leniently (iss-2608301901260678). The refusal itself is what
		// this row pins, and it is still made.
		{filepath.Join("work", "issues", "open", "iss-43-a-finding.md"), "capture skips the file"},
		// The other half of the ADR split: the reader DOES confirm the id, so the
		// missing-property leg keeps its account on the same store the duplicate-key
		// leg above must not make it on.
		{filepath.Join("rec", "decisions", "adrs", "0010-another-decision.md"), "skipped"},
	} {
		if !findingWith(fs, c.rel, ruleRecordSchema, c.claim) {
			t.Errorf("the issue reader does perform the refusal, so the finding on %s must keep %q: %+v",
				c.rel, c.claim, fs)
		}
	}
}

// A join names a family this scan does not read, in both the ways a value can:
// a prefix no store declares at all, and a store this configuration does not
// point at. Neither supports a verdict — the record might be perfectly present in
// a store nobody configured — so reporting it missing would be a confident false
// statement, and `occasioned_by`, which declares no family, keeps the prose
// tolerance its leg is built on.
//
// The stand-down was correct code no test killed: deleting it left the suite
// green while `occasioned_by: spike-3` drew a blocker saying it is not a record
// in the corpus (iss-2608301519254240).
//
// It is killed as a WHOLE, and the two halves are not separably killable. A
// configuration naming a store no prefix declares is refused by LoadConfig's
// validateRecordStores, so for any configuration a production caller can hold, an
// unread family is also an unknown one: the `!known` half is defence against a
// hand-built Config alone, and deleting it on its own leaves this test green.
func TestAJoinIsSilentOnAFamilyThisScanDoesNotRead(t *testing.T) {
	root := admissionCorpus(t)
	writeFile(t, root, "work/issues/surprises/srp-4.md",
		"---\nschema_version: 1\nid: srp-4\noccasioned_by: spike-3\n---\n\n")
	writeFile(t, root, "work/issues/surprises/srp-5.md",
		"---\nschema_version: 1\nid: srp-5\noccasioned_by: adr-9999\n---\n\n")

	cfg := admissionSchemaConfig()
	rule := cfg.Rules[ruleRecordSchema]
	delete(rule.RecordStores, "adr")
	cfg.Rules[ruleRecordSchema] = rule

	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordSchema); n != 0 {
		t.Fatalf("a family this scan does not read supports no verdict either way, got %d finding(s): %+v", n, fs)
	}
}

// The reader of the family keys on the FILENAME, not on the record's `id`
// property: the outstanding report captures the item name out of `rdi-<N>.md` and
// never opens the item's frontmatter id. And this rule's own filename <-> id leg
// compares those two numerically, so a zero-padded FILE is lint-green.
//
// So which spelling of a padded item admits is decided by the file. Refusing a
// leading zero on the value alone refuses the one spelling that admits and passes
// the one that does not, which inverts the defect the spelling leg exists to
// close.
func TestABucketJoinReadsPaddingOffTheTargetsFilename(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/.keep", "")
	writeFile(t, root, "work/issues/readings/rdg-1/rdi-02.md",
		"---\nschema_version: 1\nid: rdi-2\nrun: rdg-1\nmanifest: sha256:beef\nposition: widening\n"+
			"regime: constitutive\npattern: a stated constraint\n---\n\n")
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-3.md",
		"---\nschema_version: 1\nid: adm-3\nrun: rdg-1\nproposal: rdi-02\n"+
			"grounds: the frame does not already hold it\n---\n\n")

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordSchema); n != 0 {
		t.Fatalf("the proposal is spelled exactly as the item's file names it and admits, got %d finding(s): %+v", n, fs)
	}

	// And the unpadded spelling, which the report cannot match against that file,
	// is the finding.
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-3.md",
		"---\nschema_version: 1\nid: adm-3\nrun: rdg-1\nproposal: rdi-2\n"+
			"grounds: the frame does not already hold it\n---\n\n")
	fs, err = Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join("work", "issues", "admissions", "rdg-1", "adm-3.md"),
		ruleRecordSchema, "rdi-02.md") {
		t.Fatalf("a spelling the item's file does not carry admits nothing and must name that file: %+v", fs)
	}
}

// A target whose filename is not a bare handle is one the reader of the family
// does not read at all, so no spelling of this join admits it and none is more
// right than another. The gate says nothing rather than issuing a blocker whose
// remedy the spelling leg would itself refuse. The divergence between this rule's
// filename grammar and the report's is iss-2608300929274006's to close.
func TestABucketJoinIsSilentOnATargetTheFamilysReaderDoesNotRead(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/.keep", "")
	writeFile(t, root, "work/issues/readings/rdg-1/rdi-2-widen-the-frame.md",
		"---\nschema_version: 1\nid: rdi-2\nrun: rdg-1\nmanifest: sha256:beef\nposition: widening\n"+
			"regime: constitutive\npattern: a stated constraint\n---\n\n")
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-3.md",
		"---\nschema_version: 1\nid: adm-3\nrun: rdg-1\nproposal: rdi-2\n"+
			"grounds: the frame does not already hold it\n---\n\n")

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordSchema); n != 0 {
		t.Fatalf("no spelling admits a file the family's reader never reads, got %d finding(s): %+v", n, fs)
	}
}

// isAbsentValue does its own stripping, so a leg that hands it an ALREADY
// STRIPPED scalar strips twice and reaches a different verdict from the leg
// beside it. A value that is two apostrophes inside double quotes is the shape
// that parts them: the required-fields leg strips once and calls it present,
// while the join and bucket legs stripped twice and called it absent — so every
// leg stood down and an admission that admits nothing drew no finding at all
// (iss-2608301656192369).
//
// The claim under test is that SOMETHING speaks on each record, which is the
// silence the defect was; the quoted leg substrings pin which voice it is.
func TestADoublyQuotedEmptyJoinAndBucketFieldAreJudgedOnOneStrip(t *testing.T) {
	root := admissionCorpus(t)
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-3.md",
		"---\nschema_version: 1\nid: adm-3\nrun: rdg-1\nproposal: \"''\"\n"+
			"grounds: the configuration it admits is one the frame does not already hold\n---\n\n")
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-4.md",
		"---\nschema_version: 1\nid: adm-4\nrun: \"''\"\nproposal: rdi-2\n"+
			"grounds: the configuration it admits is one the frame does not already hold\n---\n\n")

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	adm3 := filepath.Join("work", "issues", "admissions", "rdg-1", "adm-3.md")
	adm4 := filepath.Join("work", "issues", "admissions", "rdg-1", "adm-4.md")
	for _, c := range []struct{ rel, field, leg string }{
		{adm3, "proposal", "is not a reading item handle"},
		{adm4, "run", "is filed under 'rdg-1'"},
	} {
		if !findingWith(fs, c.rel, ruleRecordSchema, c.field) {
			t.Errorf("a doubly quoted empty %s admits nothing and no leg spoke about %s: %+v", c.field, c.rel, fs)
		}
		if !findingWith(fs, c.rel, ruleRecordSchema, c.leg) {
			t.Errorf("the leg that judges %s must be the one that speaks on %s: %+v", c.field, c.rel, fs)
		}
	}
}

// The empty flow MAPPING and the explicit null tag are absences, on the same
// terms the empty flow sequence already was. `grounds: {}` states nothing, and
// `grounds: !!null` states nothing twice over — but isAbsentValue special-cased
// one flow collection and not the other and knew no tag, so a groundless
// admission passed the gate, the report keyed its pair as admitted, and the
// proposal it names vanished from the outstanding report with no line saying an
// answer was written (iss-2608301649337965).
//
// The control is the last case: a flow mapping that HOLDS something is a value,
// wrong for the field perhaps, but not an absence.
func TestAnEmptyFlowMappingAndAnExplicitNullTagAreAbsences(t *testing.T) {
	for _, c := range []struct {
		name, spelling string
		absent         bool
	}{
		{"empty flow mapping", "{}", true},
		{"padded flow mapping", "{ }", true},
		{"explicit null tag", "!!null", true},
		{"populated flow mapping", "{a: b}", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := admissionCorpus(t)
			writeFile(t, root, "work/issues/admissions/rdg-1/adm-3.md",
				"---\nschema_version: 1\nid: adm-3\nrun: rdg-1\nproposal: rdi-2\ngrounds: "+c.spelling+"\n---\n\n")

			fs, err := Lint(admissionSchemaConfig(), root)
			if err != nil {
				t.Fatal(err)
			}
			rel := filepath.Join("work", "issues", "admissions", "rdg-1", "adm-3.md")
			got := findingWith(fs, rel, ruleRecordSchema, "'grounds'")
			if got != c.absent {
				t.Fatalf("grounds: %s — absent=%v, gate said %v: %+v", c.spelling, c.absent, got, fs)
			}
		})
	}
}

// The admission's `proposal` join constrains the target's FAMILY and its BUCKET,
// and the third coordinate of the same pair is its POSITION: the outstanding
// report opens the admissions tree only for an item at `widening`, so an
// admission naming a detection, an entailment or a comparative is keyed on a pair
// nothing ever queries. It resolved, matched its bucket, spelled correctly and
// drew nothing, while admitting nothing (iss-2608301649339636).
//
// An item whose file declares NO position reads the same way, because the report
// compares the position it read — the empty string, there — against `widening`.
//
// The stand-down is the padding leg's: a target whose filename is not itself a
// bare handle is a file the family's reader never opens, so no claim about what
// the report does with it is available to make.
func TestAnAdmissionNamingAnItemOutsideTheWideningPositionIsRefused(t *testing.T) {
	root := admissionCorpus(t)
	writeFile(t, root, "work/issues/readings/rdg-1/rdi-3.md",
		"---\nschema_version: 1\nid: rdi-3\nrun: rdg-1\nmanifest: sha256:beef\nposition: detection\n"+
			"regime: registrative\npattern: a stated constraint\n---\n\n")
	writeFile(t, root, "work/issues/readings/rdg-1/rdi-5.md",
		"---\nschema_version: 1\nid: rdi-5\nrun: rdg-1\nmanifest: sha256:beef\n---\n\n")
	writeFile(t, root, "work/issues/readings/rdg-1/rdi-7-widen-the-frame.md",
		"---\nschema_version: 1\nid: rdi-7\nrun: rdg-1\nmanifest: sha256:beef\nposition: detection\n---\n\n")

	adm := func(id, proposal string) string {
		return "---\nschema_version: 1\nid: " + id + "\nrun: rdg-1\nproposal: " + proposal +
			"\ngrounds: the configuration it admits is one the frame does not already hold\n---\n\n"
	}
	for _, c := range []struct{ id, proposal string }{
		{"adm-2", "rdi-2"}, {"adm-3", "rdi-3"}, {"adm-5", "rdi-5"}, {"adm-7", "rdi-7"},
	} {
		writeFile(t, root, "work/issues/admissions/rdg-1/"+c.id+".md", adm(c.id, c.proposal))
	}

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	onFile := func(name string) int {
		rel := filepath.Join("work", "issues", "admissions", "rdg-1", name)
		n := 0
		for _, f := range fs {
			if f.File == rel && f.RuleID == ruleRecordSchema {
				n++
			}
		}
		return n
	}
	for _, c := range []struct{ file, quote string }{
		{"adm-3.md", "'detection'"},
		{"adm-5.md", "declares no position"},
	} {
		rel := filepath.Join("work", "issues", "admissions", "rdg-1", c.file)
		if !findingWith(fs, rel, ruleRecordSchema, "position") {
			t.Errorf("an admission naming an item outside the widening position must be a finding on %s: %+v", c.file, fs)
		}
		if !findingWith(fs, rel, ruleRecordSchema, c.quote) {
			t.Errorf("the finding on %s must name what the target's file says: %+v", c.file, fs)
		}
	}
	// The widening control, and the file the family's reader never opens.
	for _, name := range []string{"adm-2.md", "adm-7.md"} {
		if n := onFile(name); n != 0 {
			t.Errorf("%s must draw no finding, got %d: %+v", name, n, fs)
		}
	}
}

// The bucket blocker's leading clause — this record is keyed on a pair nothing
// queries — is true of every cross-bucket target. Its TAIL is not: it says the
// item goes on being reported as unanswered, and the padding leg one block above
// stands down on a target whose filename is not a bare handle precisely because
// what reads the family never opens such a file. The report emits nothing at all
// about `rdi-7-widen-the-frame.md`, so the operator was sent to find a report line
// that does not exist (iss-2608301656193936).
//
// The tail is therefore appended on the same test the padding leg computes. The
// two cases run together so the negative assertion cannot go vacuous: the control
// pins the tail's current wording, so a rewording fails there before the
// stand-down's absence can pass for nothing.
func TestTheBucketBlockerClaimsAReportLineOnlyForAFileTheFamilyReads(t *testing.T) {
	const tail = "goes on being reported as unanswered"
	root := admissionCorpus(t)
	// Read by the family: its filename is a bare handle.
	writeFile(t, root, "work/issues/readings/rdg-9/rdi-8.md",
		"---\nschema_version: 1\nid: rdi-8\nrun: rdg-9\nmanifest: sha256:beef\nposition: widening\n---\n\n")
	// Never opened by the family: its filename carries a slug.
	writeFile(t, root, "work/issues/readings/rdg-9/rdi-7-widen-the-frame.md",
		"---\nschema_version: 1\nid: rdi-7\nrun: rdg-9\nmanifest: sha256:beef\nposition: widening\n---\n\n")

	adm := func(id, proposal string) string {
		return "---\nschema_version: 1\nid: " + id + "\nrun: rdg-1\nproposal: " + proposal +
			"\ngrounds: the configuration it admits is one the frame does not already hold\n---\n\n"
	}
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-8.md", adm("adm-8", "rdi-8"))
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-7.md", adm("adm-7", "rdi-7"))

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	read := filepath.Join("work", "issues", "admissions", "rdg-1", "adm-8.md")
	unread := filepath.Join("work", "issues", "admissions", "rdg-1", "adm-7.md")

	// Both are keyed on a pair nothing queries, so both are still blocked.
	for _, rel := range []string{read, unread} {
		if !findingWith(fs, rel, ruleRecordSchema, "keyed on a pair nothing ever queries") {
			t.Errorf("a cross-bucket target is a finding whatever its filename: %s: %+v", rel, fs)
		}
	}
	if !findingWith(fs, read, ruleRecordSchema, tail) {
		t.Errorf("the family reads rdi-8.md, so the blocker may say the item %s: %+v", tail, fs)
	}
	if findingWith(fs, unread, ruleRecordSchema, tail) {
		t.Errorf("nothing reads rdi-7-widen-the-frame.md, so no report line names it and the blocker "+
			"must not say the item %s: %+v", tail, fs)
	}
}

// checkRecordBucketField stands down on an ABSENT bucket field, so the blank is
// reported once — by checkRecordRequiredFields, which is whose business absence
// is — and not twice. Without that stand-down a blank `run` draws a second,
// weaker finding beside the first, saying the record declares a bucket it does
// not: a blank declares nothing, and the two findings then disagree about what
// the record says. The guard survived its own deletion (iss-2608301656202623).
func TestABlankBucketFieldIsReportedOnceByTheLegAbsenceBelongsTo(t *testing.T) {
	root := admissionCorpus(t)
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-3.md",
		"---\nschema_version: 1\nid: adm-3\nrun: \"\"\nproposal: rdi-2\n"+
			"grounds: the configuration it admits is one the frame does not already hold\n---\n\n")

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	rel := filepath.Join("work", "issues", "admissions", "rdg-1", "adm-3.md")
	if !findingWith(fs, rel, ruleRecordSchema, "required property 'run' carries no value") {
		t.Errorf("a blank required property is still a finding: %+v", fs)
	}
	if n := countRule(fs, ruleRecordSchema); n != 1 {
		t.Fatalf("a blank bucket field is one finding, not two; got %d: %+v", n, fs)
	}
}

// The duplicate-key finding's fallback branch told the author that "the lenient
// scanner every record surface reads with keeps only the first value". That is a
// universal over readers, and the disposition store falsifies it:
// issueschema.ParseDisposition — the ONE reader of which disposition is in
// force, shared by core/capture and core/lint precisely so the two cannot
// diverge — returns DispositionRecord{ID: id} on a duplicated top-level key,
// keeping NEITHER value, and the outstanding report routes the item to
// Unreadable. So one lint run emitted two findings on one file that contradicted
// each other: record_schema said the first value stands, reading_outstanding said
// no reader can read the record at all, and the author acts on the false one
// (iss-2608301813253101).
//
// What this rule may claim is what its OWN scanner does, which is what its godoc
// said all along: it keeps the first value, so a second line can silence a
// blocker armed on the value the first hides. That account is true of every
// store, because this rule is the one doing that reading — and it is the whole of
// what the fallback branch may say.
//
// Setting readerRefusesDuplicateKey on the dsp store is the wrong fix and is not
// what this pins: that branch says the file is "skipped by every disposition
// surface", and a duplicated-key disposition is not skipped — it is read, found
// illegible, and reported as an unreadable answer. It would be a second wrong
// sentence.
//
// The dsp record is the falsifier and carries the reader's own answer beside it.
// The adm record is a store where the first-value account IS true of a real
// reader (admittedProposals reads with the lenient scanner and counts the
// record), so the narrowing is watched not to have gone mute on the store it was
// always true of.
func TestDuplicateKeyClaimIsScopedToThisRulesOwnScanner(t *testing.T) {
	root := admissionCorpus(t)
	const dupDisposition = "---\nschema_version: 1\nid: dsp-3\nitem: rdi-2\n" +
		"state: accepted\nstate: declined\ndisposition_grounds: worth acting on\n---\n\n"
	writeFile(t, root, "work/issues/dispositions/rdi-2/dsp-3.md", dupDisposition)
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-3.md",
		"---\nschema_version: 1\nid: adm-3\nrun: rdg-1\nproposal: rdi-2\n"+
			"grounds: it widens the frame\ngrounds: and again\n---\n\n")

	// The evidence the claim has to answer to, read from the reader itself rather
	// than asserted about it: neither value survives, and the record is not
	// well-formed. If a first-value disposition reader ever lands, this goes red
	// and points at the message that would have to change with it.
	if got := issueschema.ParseDisposition("dsp-3", dupDisposition); got.WellFormed || got.State != "" {
		t.Fatalf("the disposition reader discards BOTH values on a duplicated key, got %+v", got)
	}

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		filepath.Join("work", "issues", "dispositions", "rdi-2", "dsp-3.md"),
		filepath.Join("work", "issues", "admissions", "rdg-1", "adm-3.md"),
	} {
		if !findingWith(fs, rel, ruleRecordSchema, "duplicate top-level key") {
			t.Errorf("the duplicate is still a finding on %s: %+v", rel, fs)
		}
		if !findingWith(fs, rel, ruleRecordSchema, "silence a blocker armed on the value the first hides") {
			t.Errorf("the finding on %s keeps the account this rule can make: %+v", rel, fs)
		}
		for _, claim := range []string{"every record surface", "every disposition surface", "skipped", "refuses"} {
			if findingWith(fs, rel, ruleRecordSchema, claim) {
				t.Errorf("no reader of %s performs that, so the finding must not claim %q: %+v", rel, claim, fs)
			}
		}
	}
}

// The predicate's enumeration was a SPELLING test rather than a null test, and
// TestIsAbsentValueIsASpellingTestNotANullTest pinned that boundary in both
// directions so the widening could not land quietly: the day the predicate
// learned the null CLASS, it would go red on the prose that still told a reader
// which spellings pass. That day came (iss-2608301808198621). Its successor is
// TestAbsenceIsDecidedByClassNotBySpelling in schema_absence_test.go, which
// asserts the same table with the tagged, anchored and aliased rows moved across
// to the absent side; the godoc and the `grounds` paragraph in commands/capture.md
// moved with them.

// provenanceStores adds the two reading families to the schema fixture's store
// map, so a reading pointer has something to resolve against.
func provenanceStores() map[string]string {
	stores := schemaStores()
	stores["rdi"] = "work/issues/readings"
	stores["rdg"] = "rec/readings"
	return stores
}

// provenanceConfig arms record_provenance ALONE, so a finding in these tests can
// only come from the rule under test.
func provenanceConfig() Config {
	return Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			ruleRecordProvenance: {Enabled: true, Severity: severityBlocker, RecordStores: provenanceStores()},
		},
	}
}

// TestRecordProvenanceSilentOnUnstampedRecord is the rule's most important case,
// because it is the state of every record in the corpus: population is
// forward-only, an absent stamp is never backfilled, and sparseness is
// information. A rule that reported an unstamped record would be a wall of
// blockers over records nobody is going to change.
func TestRecordProvenanceSilentOnUnstampedRecord(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/intents/drafts/itd-1-unstamped.md", "---\nid: itd-1\nkind: null\nspec_id: null\n---\n# draft\n")
	writeFile(t, root, "work/issues/open/iss-1-unstamped.md", "---\nid: iss-1\nslug: unstamped\n---\nbody\n")
	fs, err := Lint(provenanceConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordProvenance); n != 0 {
		t.Fatalf("an unstamped record is not a finding, got %d: %+v", n, fs)
	}
}

// TestRecordProvenanceLoneKey: every write path stamps both keys together, so
// one without the other is a state no command produces.
func TestRecordProvenanceLoneKey(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/intents/drafts/itd-1-half.md",
		"---\nid: itd-1\nkind: null\nspec_id: null\norigin: researcher-authored\n---\n# draft\n")
	writeFile(t, root, "work/issues/open/iss-1-half.md",
		"---\nid: iss-1\nslug: half\nproduction_mode: hand-written\n---\nbody\n")
	fs, err := Lint(provenanceConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordProvenance); n != 2 {
		t.Fatalf("expected one finding per half-stamped record, got %d: %+v", n, fs)
	}
	if !findingWith(fs, filepath.Join("rec/intents/drafts", "itd-1-half.md"), ruleRecordProvenance, "production_mode") {
		t.Errorf("expected the missing half named: %+v", fs)
	}
	if !findingWith(fs, filepath.Join("work/issues/open", "iss-1-half.md"), ruleRecordProvenance, "origin") {
		t.Errorf("expected the missing half named: %+v", fs)
	}
}

// TestRecordProvenanceOutOfVocabulary: a value outside its closed set could not
// have been written by a command, which validates before it writes.
func TestRecordProvenanceOutOfVocabulary(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/intents/drafts/itd-1-bad-origin.md",
		"---\nid: itd-1\nkind: null\nspec_id: null\norigin: invented-by-hand\nproduction_mode: hand-written\n---\n# draft\n")
	writeFile(t, root, "rec/intents/drafts/itd-2-bad-mode.md",
		"---\nid: itd-2\nkind: null\nspec_id: null\norigin: researcher-authored\nproduction_mode: typed\n---\n# draft\n")
	fs, err := Lint(provenanceConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordProvenance); n != 2 {
		t.Fatalf("expected one finding per out-of-vocabulary value, got %d: %+v", n, fs)
	}
	if !findingWith(fs, filepath.Join("rec/intents/drafts", "itd-1-bad-origin.md"), ruleRecordProvenance, "invented-by-hand") {
		t.Errorf("expected the refused origin quoted: %+v", fs)
	}
	if !findingWith(fs, filepath.Join("rec/intents/drafts", "itd-2-bad-mode.md"), ruleRecordProvenance, "typed") {
		t.Errorf("expected the refused production mode quoted: %+v", fs)
	}
}

// TestRecordProvenanceExtractedWithoutPromotedFrom: promote writes the back-edge
// and the origin in one act, so the origin without the back-edge is a record no
// promote could have produced.
func TestRecordProvenanceExtractedWithoutPromotedFrom(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/intents/drafts/itd-1-orphan.md",
		"---\nid: itd-1\nkind: null\nspec_id: null\norigin: extracted-from-record\nproduction_mode: hand-written\n---\n# draft\n")
	writeFile(t, root, "rec/intents/drafts/itd-2-promoted.md",
		"---\nid: itd-2\nkind: null\nspec_id: null\npromoted_from: iss-1\norigin: extracted-from-record\nproduction_mode: hand-written\n---\n# draft\n")
	fs, err := Lint(provenanceConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordProvenance); n != 1 {
		t.Fatalf("expected exactly the back-edge-less record to be reported, got %d: %+v", n, fs)
	}
	if !findingWith(fs, filepath.Join("rec/intents/drafts", "itd-1-orphan.md"), ruleRecordProvenance, "promoted_from") {
		t.Errorf("expected the missing back-edge named: %+v", fs)
	}
}

// TestRecordProvenanceReportsUnresolvableReading proves the fourth check against
// a FIXTURE reading record. No command in this repository mints
// contributed-by-reading — the reading-ingest verb that carries the run and item
// identifiers does not exist yet — so the check ships armed with no production
// input, which is a sequencing fact rather than a discovery.
func TestRecordProvenanceReportsUnresolvableReading(t *testing.T) {
	root := t.TempDir()
	// The fixture reading item: run rdg-3 holds item rdi-17.
	writeFile(t, root, "work/issues/readings/rdg-3/rdi-17.md", "---\nid: rdi-17\npattern: a thing the instrument returned\n---\nbody\n")
	// Resolves: the pointer names the run the item actually sits in. Each draft
	// carries the back-edge promote writes beside the origin, so the only finding
	// these fixtures can raise is the pointer's own resolution.
	writeFile(t, root, "rec/intents/drafts/itd-1-resolves.md",
		"---\nid: itd-1\nkind: null\nspec_id: null\npromoted_from: rdi-17\norigin: contributed-by-reading rdg-3/rdi-17\nproduction_mode: hand-written\n---\n# draft\n")
	// Dangling item.
	writeFile(t, root, "rec/intents/drafts/itd-2-dangling.md",
		"---\nid: itd-2\nkind: null\nspec_id: null\npromoted_from: rdi-99\norigin: contributed-by-reading rdg-3/rdi-99\nproduction_mode: hand-written\n---\n# draft\n")
	// The item exists, but in a different run: the pair is what resolves, not
	// either id alone.
	writeFile(t, root, "rec/intents/drafts/itd-3-wrong-run.md",
		"---\nid: itd-3\nkind: null\nspec_id: null\npromoted_from: rdi-17\norigin: contributed-by-reading rdg-4/rdi-17\nproduction_mode: hand-written\n---\n# draft\n")
	fs, err := Lint(provenanceConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordProvenance); n != 2 {
		t.Fatalf("expected the two unresolvable pointers, got %d: %+v", n, fs)
	}
	if !findingWith(fs, filepath.Join("rec/intents/drafts", "itd-2-dangling.md"), ruleRecordProvenance, "rdi-99") {
		t.Errorf("expected the dangling item named: %+v", fs)
	}
	if !findingWith(fs, filepath.Join("rec/intents/drafts", "itd-3-wrong-run.md"), ruleRecordProvenance, "rdg-4") {
		t.Errorf("expected the wrong run named: %+v", fs)
	}
}

// TestRecordProvenanceIsArmedInThisRepo is the wiring evidence for the gate: a
// rule the repository's own record-lint configuration does not name runs
// nowhere, and a unit test over a fixture would go on passing while the corpus
// went unchecked.
//
// It is armed as a blocker from the start rather than staged, because
// forward-only population means it is silent on every record already committed:
// there is no wall of findings to work through, and nothing to backfill.
func TestRecordProvenanceIsArmedInThisRepo(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("..", "..", "..", ".abcd", "record-lint.json"))
	if err != nil {
		t.Fatalf("loading the repo's record-lint config: %v", err)
	}
	rc, ok := cfg.Rules[ruleRecordProvenance]
	if !ok || !rc.Enabled {
		t.Fatalf("%s is not armed in .abcd/record-lint.json: %+v", ruleRecordProvenance, rc)
	}
	if rc.Severity != severityBlocker {
		t.Errorf("%s severity = %q, want %q", ruleRecordProvenance, rc.Severity, severityBlocker)
	}
	// The stores come from record_schema's declaration, so the rule cannot drift
	// from the corpus the record gate walks.
	if len(rc.RecordStores) == 0 && len(cfg.Rules[ruleRecordSchema].RecordStores) == 0 {
		t.Error("the rule has no stores to walk from either its own config or record_schema's")
	}
}

// TestRecordProvenanceRequiresTheBackEdgeBesideAReadingOrigin — framework 11.3
// (linkage): the origin's item and the `promoted_from` back-edge are one join
// written twice, on the same footing as extracted-from-record with no back-edge.
// Promote writes both in one act, so a record carrying the origin alone, or the
// two in disagreement, is a state no command produced.
func TestRecordProvenanceRequiresTheBackEdgeBesideAReadingOrigin(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "work/issues/readings/rdg-3/rdi-17.md", "---\nid: rdi-17\npattern: a thing the instrument returned\n---\nbody\n")
	writeFile(t, root, "work/issues/readings/rdg-3/rdi-18.md", "---\nid: rdi-18\npattern: another thing\n---\nbody\n")
	// No back-edge at all.
	writeFile(t, root, "rec/intents/drafts/itd-1-no-back-edge.md",
		"---\nid: itd-1\nkind: null\nspec_id: null\norigin: contributed-by-reading rdg-3/rdi-17\nproduction_mode: hand-written\n---\n# draft\n")
	// A back-edge naming a different item from the one the origin names.
	writeFile(t, root, "rec/intents/drafts/itd-2-disagrees.md",
		"---\nid: itd-2\nkind: null\nspec_id: null\npromoted_from: rdi-18\norigin: contributed-by-reading rdg-3/rdi-17\nproduction_mode: hand-written\n---\n# draft\n")
	// The shape promote writes: both halves, agreeing.
	writeFile(t, root, "rec/intents/drafts/itd-3-agrees.md",
		"---\nid: itd-3\nkind: null\nspec_id: null\npromoted_from: rdi-17\norigin: contributed-by-reading rdg-3/rdi-17\nproduction_mode: hand-written\n---\n# draft\n")

	fs, err := Lint(provenanceConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordProvenance); n != 2 {
		t.Fatalf("expected exactly the two records missing or contradicting the back-edge, got %d: %+v", n, fs)
	}
	if !findingWith(fs, filepath.Join("rec/intents/drafts", "itd-1-no-back-edge.md"), ruleRecordProvenance, "promoted_from") {
		t.Errorf("expected the absent back-edge named: %+v", fs)
	}
	if !findingWith(fs, filepath.Join("rec/intents/drafts", "itd-2-disagrees.md"), ruleRecordProvenance, "rdi-18") {
		t.Errorf("expected the disagreeing back-edge named: %+v", fs)
	}
}

// TestRecordProvenanceChecksTheForwardEdge — framework 11.3: the join is
// redundant by design, so the gate can check it from both ends. An item whose
// `promoted_to` names a record other than the one whose origin names the item is
// reported once, on the draft.
//
// The reverse direction is deliberately not a finding: an item whose
// `promoted_to` names a researcher-authored draft is link mode working as
// designed, and so is a draft whose `promoted_from` names another item, which is
// the several-items case the intent's first scope condition describes.
func TestRecordProvenanceChecksTheForwardEdge(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "work/issues/readings/rdg-3/rdi-17.md",
		"---\nid: rdi-17\npattern: a thing the instrument returned\npromoted_to: itd-9\n---\nbody\n")
	writeFile(t, root, "work/issues/readings/rdg-3/rdi-18.md",
		"---\nid: rdi-18\npattern: another thing\npromoted_to: itd-2\n---\nbody\n")
	// The origin names rdi-17, whose forward stamp names a different record.
	writeFile(t, root, "rec/intents/drafts/itd-1-forward-disagrees.md",
		"---\nid: itd-1\nkind: null\nspec_id: null\npromoted_from: rdi-17\norigin: contributed-by-reading rdg-3/rdi-17\nproduction_mode: hand-written\n---\n# draft\n")
	// The pair agrees in both directions.
	writeFile(t, root, "rec/intents/drafts/itd-2-agrees.md",
		"---\nid: itd-2\nkind: null\nspec_id: null\npromoted_from: rdi-18\norigin: contributed-by-reading rdg-3/rdi-18\nproduction_mode: hand-written\n---\n# draft\n")
	// Link mode working as designed: an item points forward at a draft that was
	// filed by hand and says so. Not a finding.
	writeFile(t, root, "work/issues/readings/rdg-3/rdi-19.md",
		"---\nid: rdi-19\npattern: a third thing\npromoted_to: itd-3\n---\nbody\n")
	writeFile(t, root, "rec/intents/drafts/itd-3-hand-filed.md",
		"---\nid: itd-3\nkind: null\nspec_id: null\npromoted_from: rdi-19\norigin: researcher-authored\nproduction_mode: hand-written\n---\n# draft\n")

	fs, err := Lint(provenanceConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordProvenance); n != 1 {
		t.Fatalf("expected exactly the one disagreeing forward edge, got %d: %+v", n, fs)
	}
	if !findingWith(fs, filepath.Join("rec/intents/drafts", "itd-1-forward-disagrees.md"), ruleRecordProvenance, "itd-9") {
		t.Errorf("expected the record the item points at named, on the draft: %+v", fs)
	}
}

// TestRecordSchemaAdmitsBothADRVintages is the 2026-09-01 ruling at the gate:
// the ADR store holds the hand-numbered ordinals and the minted timestamp form
// in one flat directory, they cite each other, and the rule reports nothing. A
// gate that admitted only the zero-padded ordinal would refuse every decision
// minted from here on, and a lint-red record is a record nobody can commit.
func TestRecordSchemaAdmitsBothADRVintages(t *testing.T) {
	root := t.TempDir()
	adrs := "rec/decisions/adrs"
	writeFile(t, root, adrs+"/0058-a-reading-is-commissioned.md",
		"---\nid: adr-58\nslug: a-reading-is-commissioned\nstatus: accepted\nsupersedes: null\nsuperseded_by: null\nrelated_adrs: [adr-2609012206053814]\n---\n# ADR-58\n")
	writeFile(t, root, adrs+"/2609012206053814-decisions-mint-through-the-seam.md",
		"---\nid: adr-2609012206053814\nslug: decisions-mint-through-the-seam\nstatus: proposed\nsupersedes: null\nsuperseded_by: null\nrelated_adrs: [adr-58]\n---\n# ADR-2609012206053814\n")

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordSchema); n != 0 {
		t.Fatalf("expected no record_schema findings over the two ADR vintages, got %d: %+v", n, fs)
	}
}
