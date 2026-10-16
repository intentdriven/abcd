package lint

import (
	"path/filepath"
	"strings"
	"testing"
)

// decisionRecord renders a decision record in the shape the folder's writer
// will lay down (spc-2610110931381316, "The record").
func decisionRecord(id, slug, date, body string) string {
	return "---\nid: " + id + "\nslug: " + slug + "\ndate: " + date + "\n---\n" + body + "\n"
}

// decisionSchemaConfig is schemaConfig with the dcn store beside the four.
func decisionSchemaConfig() Config {
	stores := schemaStores()
	stores["dcn"] = "work/decisions"
	return Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			ruleRecordSchema: {Enabled: true, Severity: severityBlocker, RecordStores: stores},
		},
	}
}

// TestRecordSchemaRequiresDecisionFields pins the dcn store's schema
// (spc-2610110931381316, step 1): a decision record states its id, slug and
// date, and its date is the one its id's stamp carries and the one its body
// opens with. A record missing the date, and one whose date disagrees with
// its stamp or with its body, are each a blocker; a record that agrees with
// itself throughout is clean.
func TestRecordSchemaRequiresDecisionFields(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/.keep", "")
	files := map[string]struct{ name, content string }{
		"good": {"dcn-2610141200001111-a-good-one.md", decisionRecord("dcn-2610141200001111", "a-good-one", "2026-10-14",
			"2026-10-14 — a decision that agrees with itself")},
		"no-date": {"dcn-2610141200002222-no-date.md",
			"---\nid: dcn-2610141200002222\nslug: no-date\n---\n2026-10-14 — a decision with no date\n"},
		"stamp": {"dcn-2610141200003333-stamp.md", decisionRecord("dcn-2610141200003333", "stamp", "2026-10-13",
			"2026-10-13 — a date its stamp does not carry")},
		"body": {"dcn-2610141200004444-body.md", decisionRecord("dcn-2610141200004444", "body", "2026-10-14",
			"2026-10-12 — a body dated another day")},
	}
	rel := map[string]string{}
	for key, f := range files {
		rel[key] = filepath.Join("work", "decisions", f.name)
		writeFile(t, root, "work/decisions/"+f.name, f.content)
	}

	fs, err := Lint(decisionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		if f.File == rel["good"] {
			t.Errorf("a self-consistent decision record drew a finding: %+v", f)
		}
	}
	if !findingWith(fs, rel["no-date"], ruleRecordSchema, "missing required property 'date'") {
		t.Errorf("a decision with no date must be a finding: %+v", fs)
	}
	if !findingWith(fs, rel["stamp"], ruleRecordSchema, "its id's stamp carries 2026-10-14") {
		t.Errorf("a date its stamp does not carry must be a finding: %+v", fs)
	}
	if !findingWith(fs, rel["body"], ruleRecordSchema, "its body opens with 2026-10-12") {
		t.Errorf("a date its body contradicts must be a finding: %+v", fs)
	}
	for _, f := range fs {
		if f.RuleID == ruleRecordSchema && f.Severity != severityBlocker {
			t.Errorf("a decision-record finding is a blocker: %+v", f)
		}
	}
}

// TestProseCitationOfAnAbsentDecisionIsABlocker pins step 1's point: once dcn is
// a record family, a decision handle in prose is a checked citation. One that
// names a record in the folder is quiet, in any spelling; one that names no
// record is a blocker, exactly as an invented issue id is.
func TestProseCitationOfAnAbsentDecisionIsABlocker(t *testing.T) {
	root := t.TempDir()
	proseCorpus(t, root)
	writeFile(t, root, ".abcd/work/decisions/dcn-2610141200001111-a-ruling.md",
		decisionRecord("dcn-2610141200001111", "a-ruling", "2026-10-14", "2026-10-14 — a ruling"))
	writeProseIssue(t, root, "open", "iss-92-cites-decisions.md", strings.Join([]string{
		"The ruling is dcn-2610141200001111, also written DCN-02610141200001111.",
		"",
		"An invented one: dcn-2610141200009999.",
	}, "\n"))

	cfg := proseCfg()
	rc := cfg.Rules[ruleProseCitationResolves]
	rc.RecordStores["dcn"] = ".abcd/work/decisions"
	cfg.Rules[ruleProseCitationResolves] = rc

	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleProseCitationResolves); n != 1 {
		t.Fatalf("expected the absent decision to fire once and the present one to stay quiet, got %d: %+v", n, fs)
	}
	rel := filepath.Join(".abcd", "work", "issues", "open", "iss-92-cites-decisions.md")
	if !findingWith(fs, rel, ruleProseCitationResolves, "dcn-2610141200009999") {
		t.Errorf("the finding names the absent decision: %+v", fs)
	}
	for _, f := range fs {
		if f.RuleID == ruleProseCitationResolves && f.Severity != severityBlocker {
			t.Errorf("an absent decision is a blocker: %+v", f)
		}
	}
}

// TestCrossStoreIDClaimNamesADecisionHandle pins cross_store_id_claim's reach
// to the dcn family: a decision-shaped document outside the decisions folder
// whose heading claims a decision id a real record holds is a second document
// answering to one handle, as it is for an ADR.
func TestCrossStoreIDClaimNamesADecisionHandle(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/work/decisions/dcn-2610141200001111-a-ruling.md",
		decisionRecord("dcn-2610141200001111", "a-ruling", "2026-10-14", "2026-10-14 — a ruling"))
	writeFile(t, root, "research/notes/a-copy.md",
		"# DCN-2610141200001111: a ruling, copied\n\n## Status\n\nAccepted\n")

	cfg := crossStoreCfg()
	rc := cfg.Rules[ruleRecordSchema]
	rc.RecordStores["dcn"] = ".abcd/work/decisions"
	cfg.Rules[ruleRecordSchema] = rc

	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if !hasFinding(fs, filepath.Join("research", "notes", "a-copy.md"), ruleCrossStoreIDClaim, 1) ||
		!messageContains(fs, "dcn-2610141200001111") {
		t.Errorf("a copy claiming a held decision id must be named: %+v", fs)
	}
}

// TestRecordSchemaRefusesWhatTheDecisionsReaderRefuses is the dcn store's
// reader-parity leg, the issue store's twin: the record is handed to
// decisions.Parse itself, so a record the folder's reader skips cannot pass the
// gate. The fixture is one the other legs read as well-formed — a filename id
// written with a leading zero is the same handle to the numeric compare, but not
// to the reader, which holds the filename and the frontmatter to one spelling.
func TestRecordSchemaRefusesWhatTheDecisionsReaderRefuses(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/.keep", "")
	writeFile(t, root, "work/decisions/dcn-02610141200001111-padded.md",
		decisionRecord("dcn-2610141200001111", "padded", "2026-10-14", "2026-10-14 — a padded filename"))

	fs, err := Lint(decisionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	rel := filepath.Join("work", "decisions", "dcn-02610141200001111-padded.md")
	if !findingWith(fs, rel, ruleRecordSchema, "the decisions reader refuses this record") {
		t.Errorf("a record the decisions reader refuses must not pass the gate: %+v", fs)
	}
}
