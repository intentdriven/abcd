package intent

import "testing"

// handaudit_test.go pins the one rule for when an Audit Notes section with no
// review marker counts as an audit (iss-2610241431221889): it names the
// acceptance criteria and gives a verdict on them. The positive shapes are the
// ones the record actually holds — itd-3's hand-written audit of 2026-07-11,
// whose verdicts sit in a numbered list under a sub-heading, and the
// reviewer's own per-criterion bullet — and a wrapped paragraph. The negative
// shapes are the stray notes real records carry beside or instead of an audit.

// withAuditNotes is a record body whose Audit Notes section is notes, followed
// by a References section.
func withAuditNotes(notes string) string {
	return "---\nid: itd-9\n---\n# A promise\n\n## Acceptance Criteria\n\n- ok\n\n## Audit Notes\n\n" +
		notes + "\n\n## References\n\n- none\n"
}

func TestHandAuditedNamesCriteriaWithAVerdict(t *testing.T) {
	cases := []struct {
		name, notes string
	}{
		{"a numbered verdict list under a sub-heading, as itd-3's hand audit reads",
			"> _Manual fidelity audit, performed by hand before the reviewer existed._\n\n" +
				"### Per-criterion verdicts\n\n" +
				"1. **MET.** The hook injects the domain on the turn.\n" +
				"2. **MET_WITH_CONCERNS.** The refresh is event-driven, not every fifth prompt.\n" +
				"3. **NOT_MET.** The defaults omit the naming rules.\n\n" +
				"**Rollup:** 1 MET · 1 MET_WITH_CONCERNS · 1 NOT_MET.\n\n" +
				"### Three-bucket audit\n\n- **Honoured.** Recall injection.\n"},
		{"a paragraph naming the criteria, wrapped across lines",
			"Audited by hand on 2026-07-01 against the merged code: criteria 1 and 2\n" +
				"are MET, and criterion 3 is NOT_MET because the refusal names no tag."},
		{"the reviewer's per-criterion bullet",
			"Per-criterion verdicts:\n- ac-2 — INCONCLUSIVE: no test reaches the refusal."},
		{"a criterion named by its upper-case id",
			"AC 4 is NOT MET: the star command is not wired."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if !HandAudited(withAuditNotes(c.notes)) {
				t.Errorf("not read as a hand-written audit:\n%s", c.notes)
			}
		})
	}
}

func TestHandAuditedRefusesWhatIsNotAnAudit(t *testing.T) {
	cases := []struct {
		name, notes string
	}{
		{"the template's placeholder with a parked sub-heading",
			"_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._\n\n" +
				"### Implementation notes\n\nA subsection is not an audit."},
		{"a ruling's amendment line",
			"- 2026-10-02 — Amendment to decision 8, recorded by lane rd2 of autonomous run A."},
		{"a reading note that says a criterion is met in running prose",
			"Criterion 9 read on 2026-09-30, recorded under a ruling: it is met as stdout\n" +
				"carrying only the receipt in every mode. The criterion text above stands as shipped."},
		{"a note that every criterion holds without naming one",
			"Audited by hand on 2026-07-01: every criterion holds."},
		{"a rollup that names no criterion",
			"**Rollup:** 3 MET · 1 NOT_MET."},
		{"a criterion in one paragraph and a verdict in another",
			"Criterion 2 was reread on 2026-09-30.\n\nThe rollup stays MET 3."},
		{"a verdict quoted as code",
			"Criterion 2 now names the `NOT_MET` token in its refusal."},
		{"a verdict inside a fenced example",
			"Criterion 2 was reread on 2026-09-30.\n```\n- ac-2 — NOT_MET: example\n```"},
		{"a verdict inside an HTML comment",
			"<!-- ac-2 — NOT_MET: parked -->\nNothing audited yet."},
		{"a receipt id that is not a criterion",
			"Receipt rcp-ac1234567890 is MET by nothing here."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if HandAudited(withAuditNotes(c.notes)) {
				t.Errorf("read as a hand-written audit:\n%s", c.notes)
			}
		})
	}
}

// The section ends at the next heading of level one or two: a verdict under
// the record's next section is not the Audit Notes', and a record with no
// Audit Notes section has no audit.
func TestHandAuditedReadsOnlyItsOwnSection(t *testing.T) {
	after := "---\nid: itd-9\n---\n# A promise\n\n## Audit Notes\n\nNothing yet.\n\n" +
		"## Grounds\n\n- criterion 2 is NOT_MET in the predecessor\n"
	if HandAudited(after) {
		t.Error("a verdict under the next section was read as the Audit Notes'")
	}
	none := "---\nid: itd-9\n---\n# A promise\n\n## Grounds\n\n- ac-1 — MET: elsewhere\n"
	if HandAudited(none) {
		t.Error("a record with no Audit Notes section was read as hand-audited")
	}
}
