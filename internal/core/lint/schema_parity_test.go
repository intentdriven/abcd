package lint

import (
	"path/filepath"
	"testing"
)

// seedRecRoot creates the `rec` root the shared schemaConfig declares, so a test
// that populates only the issue store (which sits outside `rec`) still resolves
// every configured root. The README is skipped by the record scan.
func seedRecRoot(t *testing.T, root string) {
	t.Helper()
	writeFile(t, root, "rec/README.md", "# rec\n")
}

// validIssue is a complete, well-formed open-issue record. Each parity test below
// mutates exactly one facet of it, so a single finding is the whole delta.
func validIssue(id, slug string) string {
	return "---\nschema_version: 1\nid: " + id + "\nslug: " + slug +
		"\nseverity: minor\ncategory: bug\nsource: user-observation\nfound_during: t\n---\n\nan issue\n"
}

// TestRecordSchemaFilenameGrammarMatchesRecordid (iss-2608270908346617) pins that
// lint's per-store filename grammar is the recordid resolver's grammar, not a
// looser `^iss-(\d+).*\.md$` that accepts an arbitrary tail. A divergent name is
// silently dropped by capture's scanLedger and hard-errors when the record is
// cited, yet the loose lint regex read it as a well-formed record — so the gate
// passed a record every consumer refuses.
func TestRecordSchemaFilenameGrammarMatchesRecordid(t *testing.T) {
	root := t.TempDir()
	issues := "work/issues"
	seedRecRoot(t, root)
	// `iss-5_bad.md`: the loose grammar captures num 5 and accepts the `_bad`
	// tail; the recordid grammar (and capture) refuse it — the slug must be a
	// kebab tail, not arbitrary bytes after the ordinal.
	writeFile(t, root, issues+"/open/iss-5_bad.md", validIssue("iss-5", "ok"))

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join(issues, "open", "iss-5_bad.md"), ruleRecordSchema, "not a well-formed issue filename") {
		t.Fatalf("expected a malformed-filename finding for iss-5_bad.md (recordid grammar): %+v", fs)
	}
}

// TestRecordSchemaFilenameGrammarAcceptsKebab is the control: a well-formed name
// with a kebab slug stays silent, so the tightened grammar refuses only what
// capture refuses. The bare-ordinal name (iss-6.md) is still grammatically a
// record — the tightened filename grammar draws no "not a well-formed issue
// filename" finding on it — but the issue schema requires a slug the filename
// must carry, so the slug-agreement check (checkRecordFilenameSlug) reports its
// empty filename slug against the frontmatter slug. That mirrors the capture
// reader exactly: validateInvariants refuses the same record because "" is a
// value that names a different record. The two capabilities compose — the
// grammar accepts the bare ordinal, and the reader-mirroring slug check still
// catches the disagreement — so lint refuses only what capture refuses.
func TestRecordSchemaFilenameGrammarAcceptsKebab(t *testing.T) {
	root := t.TempDir()
	issues := "work/issues"
	seedRecRoot(t, root)
	writeFile(t, root, issues+"/open/iss-5-a-good-slug.md", validIssue("iss-5", "a-good-slug"))
	writeFile(t, root, issues+"/open/iss-6.md", validIssue("iss-6", "bare"))

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	// The kebab name agrees on every count and must stay completely silent.
	if findingWith(fs, filepath.Join(issues, "open", "iss-5-a-good-slug.md"), ruleRecordSchema, "") {
		t.Fatalf("well-formed kebab name must stay clean: %+v", fs)
	}
	// The bare-ordinal name is grammatically a record: no malformed-filename finding.
	if findingWith(fs, filepath.Join(issues, "open", "iss-6.md"), ruleRecordSchema, "not a well-formed issue filename") {
		t.Fatalf("bare-ordinal name must clear the filename grammar: %+v", fs)
	}
	// But its empty filename slug disagrees with the required frontmatter slug, the
	// same record the capture reader refuses — so slug-agreement reports it.
	if !findingWith(fs, filepath.Join(issues, "open", "iss-6.md"), ruleRecordSchema, "must be the same value") {
		t.Fatalf("bare-ordinal name with a frontmatter slug must trip slug-agreement, mirroring capture: %+v", fs)
	}
}

// TestRecordSchemaFlagsIdlessADR (iss-2608270908344426) pins that an ADR whose
// frontmatter carries no id is refused. describeADR resolves an ADR by matching
// its filename ordinal AND confirming the frontmatter id, so an id-less ADR reads
// as "not found" through the record dispatcher though its file plainly sits in
// the store — while the lint that never asked for the id stayed green. `id` is a
// required ADR property, the same way intent.Load fail-closes on a missing id.
func TestRecordSchemaFlagsIdlessADR(t *testing.T) {
	root := t.TempDir()
	adrs := "rec/decisions/adrs"
	writeFile(t, root, adrs+"/0007-no-id.md", "---\nstatus: accepted\n---\n# ADR-7\n")

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join(adrs, "0007-no-id.md"), ruleRecordSchema, "missing required property 'id'") {
		t.Fatalf("expected a missing-id finding on the id-less ADR: %+v", fs)
	}
}

// TestRecordSchemaFlagsOrdinalCollision (iss-2608270908346940) pins that two
// records sharing a filename ordinal are reported. The corpus index keys on the
// (prefix, ordinal) handle, so the second record silently overwrites the first
// and becomes unreachable to every cross-reference and index that resolves the
// handle — with no finding to mark it. The collision is now a record_schema
// finding on the later (overwriting) record.
func TestRecordSchemaFlagsOrdinalCollision(t *testing.T) {
	root := t.TempDir()
	adrs := "rec/decisions/adrs"
	writeFile(t, root, adrs+"/0006-a.md", "---\nid: adr-6\nsupersedes: null\nsuperseded_by: null\n---\n# ADR-6 a\n")
	writeFile(t, root, adrs+"/0006-b.md", "---\nid: adr-6\nsupersedes: null\nsuperseded_by: null\n---\n# ADR-6 b\n")

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join(adrs, "0006-b.md"), ruleRecordSchema, "collides") {
		t.Fatalf("expected an ordinal-collision finding on the second adr-6: %+v", fs)
	}
}

// TestRecordSchemaFlagsUnknownIssueKey (iss-2608261447039180) pins that an
// UNKNOWN issue-frontmatter key is refused. capture's reader rejects a record
// with a key outside the additionalProperties:false allow-list and skips it, so
// the record goes invisible to every capture surface — while the lint, which
// flagged only MISSING required properties, stayed green. The allow-list is the
// one shared issueschema set, not a restated copy.
func TestRecordSchemaFlagsUnknownIssueKey(t *testing.T) {
	root := t.TempDir()
	issues := "work/issues"
	seedRecRoot(t, root)
	rec := "---\nschema_version: 1\nid: iss-5\nslug: ok\nseverity: minor\ncategory: bug\nsource: user-observation\nfound_during: t\nbogus: x\n---\n\nan issue\n"
	writeFile(t, root, issues+"/open/iss-5-ok.md", rec)

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !findingWith(fs, filepath.Join(issues, "open", "iss-5-ok.md"), ruleRecordSchema, "unknown frontmatter property 'bogus'") {
		t.Fatalf("expected an unknown-key finding on the issue record: %+v", fs)
	}
}

// TestRecordSchemaFlagsIssueEnumAndSlug (iss-2608270908342889) pins that lint
// mirrors capture's enum-membership and kebab-slug checks for the issue store. A
// record with an out-of-enum severity/category/source, or a non-kebab slug, is
// refused by capture's validateStrict and skipped — invisible everywhere — while
// the lint that never checked the values stayed green.
func TestRecordSchemaFlagsIssueEnumAndSlug(t *testing.T) {
	issues := "work/issues"

	cases := []struct {
		name   string
		file   string
		rec    string
		substr string
	}{
		{"severity", "iss-5-ok.md",
			"---\nschema_version: 1\nid: iss-5\nslug: ok\nseverity: huge\ncategory: bug\nsource: user-observation\nfound_during: t\n---\n\nx\n",
			"severity 'huge'"},
		{"category", "iss-6-ok.md",
			"---\nschema_version: 1\nid: iss-6\nslug: ok\nseverity: minor\ncategory: nonsense\nsource: user-observation\nfound_during: t\n---\n\nx\n",
			"category 'nonsense'"},
		{"source", "iss-7-ok.md",
			"---\nschema_version: 1\nid: iss-7\nslug: ok\nseverity: minor\ncategory: bug\nsource: telepathy\nfound_during: t\n---\n\nx\n",
			"source 'telepathy'"},
		{"slug", "iss-8-ok.md",
			"---\nschema_version: 1\nid: iss-8\nslug: Not_Kebab\nseverity: minor\ncategory: bug\nsource: user-observation\nfound_during: t\n---\n\nx\n",
			"slug 'Not_Kebab'"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			seedRecRoot(t, root)
			writeFile(t, root, issues+"/open/"+c.file, c.rec)
			fs, err := Lint(schemaConfig(), root)
			if err != nil {
				t.Fatal(err)
			}
			if !findingWith(fs, filepath.Join(issues, "open", c.file), ruleRecordSchema, c.substr) {
				t.Fatalf("expected an issue-shape finding quoting %q: %+v", c.substr, fs)
			}
		})
	}
}

// TestIssueRecordShapeFlagsLapseWithoutLapsedAt (spc-60) pins that the
// committed-ledger gate refuses exactly what the reader refuses: a record whose
// category is lapse and which carries a malformed lapsed_at (absence is parked,
// iss-2609091009111294, and reads clean on both sides). capture's
// validateStrict refuses such a record and SKIPS it, so it sits in the ledger
// invisible to every capture surface — the failure mode this rule exists to turn
// into a red gate. The well-formed record beside it is the control: the gate must
// refuse the defect and nothing else.
func TestIssueRecordShapeFlagsLapseWithoutLapsedAt(t *testing.T) {
	issues := "work/issues"
	record := func(id, slug, category, lapsedAt string) string {
		rec := "---\nschema_version: 1\nid: " + id + "\nslug: " + slug +
			"\nseverity: minor\ncategory: " + category + "\nsource: user-observation\nfound_during: preparation\n"
		if lapsedAt != "" {
			rec += "lapsed_at: " + lapsedAt + "\n"
		}
		return rec + "---\n\na record\n"
	}
	lapse := func(id, slug, lapsedAt string) string {
		return record(id, slug, "lapse", lapsedAt)
	}
	// A lapsed_at whose own line carries no value and whose value is an indented
	// continuation on the lines that follow. The shared frontmatter scanner reads
	// same-line values only, so this is the one shape it sees as empty.
	blockValued := func(id, slug, category, continuation string) string {
		return "---\nschema_version: 1\nid: " + id + "\nslug: " + slug +
			"\nseverity: minor\ncategory: " + category +
			"\nsource: user-observation\nfound_during: preparation\n" +
			"lapsed_at:\n  " + continuation + "\n---\n\na record\n"
	}

	cases := []struct {
		name   string
		file   string
		rec    string
		substr string // "" means the record must stay clean
	}{
		// Absent is clean: the absence finding spc-60 raised is parked
		// (iss-2609091009111294), mirroring capture's reader, which accepts the record.
		{"absent", "iss-5-lapse-a.md", lapse("iss-5", "lapse-a", ""), ""},
		{"date only", "iss-6-lapse-b.md", lapse("iss-6", "lapse-b", "2026-08-28"), "is not an RFC 3339 instant"},
		{"free text", "iss-7-lapse-c.md", lapse("iss-7", "lapse-c", "yesterday"), "is not an RFC 3339 instant"},
		{"well-formed", "iss-8-lapse-d.md", lapse("iss-8", "lapse-d", `"2026-08-28T00:00:00Z"`), ""},
		// A quoted all-whitespace value, hand-authored. capture's reader trims
		// before judging, so it reads as ABSENT: on a lapse record that is the
		// missing-instant refusal, and on every other category it is a clean record
		// with an optional property left unset. This gate must reach the same two
		// verdicts, or it reports a reader refusal that does not happen
		// (iss-2608300212513349).
		{"padded on a lapse", "iss-9-lapse-e.md", lapse("iss-9", "lapse-e", `"   "`), ""},
		{"padded on a non-lapse", "iss-10-obs-a.md", record("iss-10", "obs-a", "observation", `"   "`), ""},
		// A list-shaped value. capture's reader parses it as []string and refuses the
		// record outright ("lapsed_at" must be a string), skipping it — so it is
		// invisible to every capture surface. Reading an empty inline list as ABSENT
		// here would leave that record lint-green on any category but lapse, which is
		// the split iss-2608300224316569 records. Both categories must be refused,
		// and for the same reason: the value is present and is no instant.
		{"list on a non-lapse", "iss-11-obs-b.md", record("iss-11", "obs-b", "observation", "[]"), "is not an RFC 3339 instant"},
		{"list on a lapse", "iss-12-lapse-f.md", lapse("iss-12", "lapse-f", "[]"), "is not an RFC 3339 instant"},
		// The block-mapped sibling of the list case. capture's reader builds a map
		// and refuses the record for the same reason ("lapsed_at" must be a string),
		// skipping it; the same-line scanner sees an empty value, so without the
		// look-ahead the gate reads a value that is plainly there as absent and goes
		// green on a record no capture surface can see (iss-2608300234599781).
		{"map on a non-lapse", "iss-13-obs-c.md", blockValued("iss-13", "obs-c", "observation", "intent: itd-1"), "spelled as an indented block"},
		{"map on a lapse", "iss-14-lapse-g.md", blockValued("iss-14", "lapse-g", "lapse", "intent: itd-1"), "spelled as an indented block"},
		// The block continuation that READS as a valid instant. It is the case the
		// block look-ahead must not hand to the format validator: capture splits the
		// indented line on its first colon and builds map["2026-08-28T00"]="00:00Z",
		// then refuses the record because lapsed_at must be a string. A block-spelled
		// value is never a string to the reader, whatever its text spells, so the
		// finding is its presence rather than its content (iss-2608300244489638).
		{"block instant on a non-lapse", "iss-15-obs-d.md", blockValued("iss-15", "obs-d", "observation", "2026-08-28T00:00:00Z"), "spelled as an indented block"},
		{"block instant on a lapse", "iss-16-lapse-h.md", blockValued("iss-16", "lapse-h", "lapse", "2026-08-28T00:00:00Z"), "spelled as an indented block"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			seedRecRoot(t, root)
			writeFile(t, root, issues+"/open/"+c.file, c.rec)
			fs, err := Lint(schemaConfig(), root)
			if err != nil {
				t.Fatal(err)
			}
			rel := filepath.Join(issues, "open", c.file)
			if c.substr == "" {
				if findingWith(fs, rel, ruleRecordSchema, "") {
					t.Fatalf("a record the reader accepts must stay clean: %+v", fs)
				}
				return
			}
			if !findingWith(fs, rel, ruleRecordSchema, c.substr) {
				t.Fatalf("expected a lapse-shape finding quoting %q: %+v", c.substr, fs)
			}
		})
	}
}

// resolvedIssue is a complete, well-formed RESOLVED-issue record carrying the
// given extra frontmatter line, so a grounds spelling is the whole delta.
func resolvedIssue(id, slug, extra string) string {
	return "---\nschema_version: 1\nid: " + id + "\nslug: " + slug +
		"\nseverity: minor\ncategory: bug\nsource: user-observation\nfound_during: t\n" +
		"impact: fix\nresolution: done\n" + extra + "---\n\nan issue\n"
}

// TestRecordSchemaBlocksAFrontmatterGroundsKey is the gate half of the move off
// frontmatter (iss-2608301657354776). Grounds are appended as `## Grounds`
// bullets in the record body; a frontmatter `grounds:` is a value nothing reads.
//
// The gate deliberately does NOT mirror the reader here, and that is the point
// of the rule. capture TOLERATES the key in every spelling — refusing would make
// it skip the record, hiding it from every capture surface while it still sits
// in the ledger — so the reader's verdict cannot be what notices a misplaced
// value. This gate is. The reader half is proved in
// capture.TestReaderToleratesALegacyGroundsKey; package lint cannot import
// capture, so nothing here can speak for it.
//
// The spellings are the ones that used to decide the old grammar rule's verdict,
// including the two it passed (a well-formed value, an empty one). The property
// asserted is that the KEY is what is blocked, so a rule that went back to
// judging the value fails on whichever spelling it started letting through.
func TestRecordSchemaBlocksAFrontmatterGroundsKey(t *testing.T) {
	const issues = "work/issues"
	for _, tc := range []struct {
		name  string
		extra string
	}{
		{"single quoted", "grounds: 'pursued: we expect the reader to see the quote'\n"},
		{"empty list", "grounds: []\n"},
		{"block spelled", "grounds:\n  pursued: we expect a mapping rather than a string\n"},
		{"empty string", "grounds: \"\"\n"},
		{"bare null", "grounds: null\n"},
		{"out of vocabulary", "grounds: \"planned: not a value the vocabulary carries\"\n"},
		{"no token", "grounds: \"no token at all here\"\n"},
		{"well formed", "grounds: \"pursued: we expect the recorded reasoning to outlive the session\"\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			seedRecRoot(t, root)
			writeFile(t, root, issues+"/resolved/iss-1-ok.md", resolvedIssue("iss-1", "ok", tc.extra))

			fs, err := Lint(schemaConfig(), root)
			if err != nil {
				t.Fatal(err)
			}
			if n := countRule(fs, ruleRecordSchema); n != 1 {
				t.Fatalf("a frontmatter grounds key raised %d record_schema finding(s), want 1: %+v", n, fs)
			}
			// The remedy has to be actionable, and the only actionable thing to
			// say about a misplaced value is where it goes.
			if !findingWith(fs, filepath.Join(issues, "resolved", "iss-1-ok.md"), ruleRecordSchema, "## Grounds") {
				t.Fatalf("the finding does not name the section the value belongs in: %+v", fs)
			}
		})
	}
}

// TestRecordSchemaSilentOnAGroundsSection: the body section is where grounds
// live, so a record carrying one and no frontmatter key is clean. Without this
// the rule above could be satisfied by a gate that blocks every record with the
// word "grounds" anywhere in it.
func TestRecordSchemaSilentOnAGroundsSection(t *testing.T) {
	const issues = "work/issues"
	root := t.TempDir()
	seedRecRoot(t, root)
	writeFile(t, root, issues+"/resolved/iss-1-ok.md",
		resolvedIssue("iss-1", "ok", "")+
			"\n## Grounds\n\n- pursued: we expect the recorded reasoning to outlive the session\n")

	fs, err := Lint(schemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleRecordSchema); n != 0 {
		t.Fatalf("a record carrying its grounds in the body raised %d record_schema finding(s): %+v", n, fs)
	}
}
