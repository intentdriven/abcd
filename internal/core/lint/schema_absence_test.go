package lint

import (
	"path/filepath"
	"strings"
	"testing"
)

// Absence is decided by the CLASS of YAML node a value spells, not by the
// literal it is written with. `!!null`, `!!null null` and
// `!<tag:yaml.org,2002:null>` are one node written three ways, and the
// enumeration this replaces accepted the first and passed the other two
// (iss-2608301808198621) — so closing the tenth spelling left the eleventh open,
// the arms race adr-56 ruled against one workstream over.
//
// The predicate now consults the shared reader (internal/core/frontmatter), so
// the spellings nobody enumerated are decided by the same rule as the ones that
// were. This is the direct successor of
// TestIsAbsentValueIsASpellingTestNotANullTest, which pinned the boundary the
// old prose claimed and said the day the predicate learned the class it would
// have to change with it.
//
// framework 11.2: a disposition's required field is refused when it is "blank or
// whitespace only", which is a question about what the field carries, not about
// how the author spelled nothing.
func TestAbsenceIsDecidedByClassNotBySpelling(t *testing.T) {
	// Carries nothing, whatever it is spelled with.
	for _, v := range []string{
		"", "   ", `""`, `"  "`, "''", "[]", "{}", "[ ]", "{ }",
		"~", "null", "Null", "NULL", "!!null",
		// The spellings the old enumeration let through, each the same node as
		// one it already accepted.
		"!!null null", "!<tag:yaml.org,2002:null>", "!!str ''", "&anchor", "*alias",
		"!!seq []", "!!map {}", "!!null ~", `!!str ""`,
	} {
		if !isAbsentValue(v) {
			t.Errorf("the gate reads %q as carrying a value; it carries nothing", v)
		}
	}
	// Carries something. A tag over a value is the value it tags, and a
	// collection that HOLDS something is a value of the wrong shape rather than
	// an absence — a different question, asked by a different leg.
	for _, v := range []string{
		"adr-9", "minor", "{a: b}", "[adr-14, adr-15]", `"minor"`, `"''"`,
		"!!int 3", "!!str minor", "&a minor", "nullify", "nUlL",
	} {
		if isAbsentValue(v) {
			t.Errorf("the gate reads %q as carrying nothing; it carries a value", v)
		}
	}
}

// End to end through Lint, which is the only proof that matters: the four blank
// spellings the record enumerates each draw a `record_schema` finding when a
// trailing comment follows them. Before the strip landed in the shared scanner,
// every one of these was lint-green — each emptiness test anchors on the value's
// LAST byte, and a comment moved it (iss-2608301744268001) — so an admission
// admitting nothing was keyed as admitted and the proposal it names dropped out
// of the outstanding report with no line saying an answer was written.
//
// framework 11.2: "blank or whitespace only" is the refusal, and a value a
// reader cannot delimit cannot be held to it.
func TestATrailingCommentDoesNotHideABlankFromTheGate(t *testing.T) {
	for _, c := range []struct {
		name, spelling string
		absent         bool
	}{
		{"empty flow mapping behind a comment", "{}  # todo", true},
		{"empty flow sequence behind a comment", "[]  # todo", true},
		{"tilde behind a comment", "~ # todo", true},
		{"double-quoted empty behind a comment", `"" # todo`, true},
		// The control: a comment does not make a real value disappear.
		{"a value behind a comment", "the frame does not already hold it # todo", false},
		// And a hash that is part of the value is part of the value.
		{"a value that contains a hash", "issue a#b is the ground", false},
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
			if got := findingWith(fs, rel, ruleRecordSchema, "'grounds'"); got != c.absent {
				t.Fatalf("grounds: %s — absent=%v, gate said %v: %+v", c.spelling, c.absent, got, fs)
			}
		})
	}
}

// The scanner question is not scoped to `grounds`, which is what made it a
// scanner question rather than a predicate one: `severity: minor # todo` reached
// the enum leg as the value `minor # todo` and drew an invalid-severity finding
// on a record whose severity is plainly `minor`. Every same-line scalar in every
// store was read that way.
func TestATrailingCommentIsNotPartOfAnEnumValue(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/.keep", "")
	writeFile(t, root, "work/issues/open/iss-1-a-finding.md",
		"---\nschema_version: 1\nid: iss-1\nslug: a-finding\nseverity: minor # todo\n"+
			"category: bug\nsource: user-observation\nfound_during: t\n---\n\nan issue\n")

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	rel := filepath.Join("work", "issues", "open", "iss-1-a-finding.md")
	if findingWith(fs, rel, ruleRecordSchema, "invalid severity") {
		t.Errorf("a trailing comment was read as part of the severity: %+v", fs)
	}
	if n := countRule(fs, ruleRecordSchema); n != 0 {
		t.Errorf("a well-formed record with a commented field drew %d finding(s): %+v", n, fs)
	}

	// The other direction: a hash with no whitespace in front of it is part of
	// the value, so an out-of-enum value containing one is still refused, and the
	// message quotes the whole value rather than a truncation of it.
	writeFile(t, root, "work/issues/open/iss-2-a-finding.md",
		"---\nschema_version: 1\nid: iss-2\nslug: a-finding\nseverity: mi#nor\n"+
			"category: bug\nsource: user-observation\nfound_during: t\n---\n\nan issue\n")
	fs, err = Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	rel2 := filepath.Join("work", "issues", "open", "iss-2-a-finding.md")
	if !findingWith(fs, rel2, ruleRecordSchema, "invalid severity 'mi#nor'") {
		t.Errorf("a hash inside a value is part of the value: %+v", fs)
	}
}

// A quoted value carrying a hash is untouched, in the store where a URL fragment
// actually turns up. The strip is YAML's rule, so a `#` inside quotes is content
// and never a comment.
func TestAQuotedHashSurvivesIntoTheValue(t *testing.T) {
	root := admissionCorpus(t)
	grounds := `"the frame does not hold https://example.com/spec#clause-4"`
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-3.md",
		"---\nschema_version: 1\nid: adm-3\nrun: rdg-1\nproposal: rdi-2\ngrounds: "+grounds+"\n---\n\n")

	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	rel := filepath.Join("work", "issues", "admissions", "rdg-1", "adm-3.md")
	if findingWith(fs, rel, ruleRecordSchema, "'grounds'") {
		t.Errorf("a quoted URL fragment was read as a comment and the value as blank: %+v", fs)
	}
	for _, f := range fs {
		if strings.Contains(f.Message, "clause-4") {
			t.Errorf("unexpected finding quoting the value: %+v", f)
		}
	}
}
