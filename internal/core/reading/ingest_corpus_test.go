package reading

// ingest_corpus_test.go is the realistic corpus the supply-regime gate is held
// to (iss-2608311518056854, ruled 2026-09-01: the gate refuses only a real
// decision field).
//
// A reading REPORTS. It quotes the document it read, says what a clause settles,
// what a paper recommends, what a suite scores, and which section says a fix is
// merged — and that is most of what a reading legitimately does. A reading that
// DECIDES carries the decision as a field of its OWN output: a key on the item
// named `disposition`, `score`, `fix` and the rest. The first is honest at every
// regime; only the second is the licence breach.
//
// So the corpus has two halves and the gate is held to both. Every honest report
// lands, and the run record it leaves says nothing against it. Every real
// decision field refuses under the reserved-name rule, naming itself — including
// the ones spelled in code points that render the same, and the one carried
// inside a nested object the contract does not define.
//
// The honest half is the thirty-four realistic outputs the itd-185 fidelity
// audit measured (receipt rcp-fe3450ca55ff). Fourteen of them are the shapes it
// measured caught; the other twenty are ordinary reporting prose that already
// landed, kept so a fix that widened the refusal instead of narrowing it would
// show.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// corpusCase is one realistic item: a position, the body field the prose sits
// in, and the prose. The rest of the item is the fixture's legal body.
type corpusCase struct {
	name     string
	position Position
	field    string
	text     string
}

// honestReports is the thirty-four-case corpus. Every one lands, at every
// position, with nothing raised against it.
var honestReports = []corpusCase{
	// The fourteen the audit measured caught: a reading quoting or reporting.
	{"quotes a record's disposition line", PositionEntailment, "claim_surfaced",
		"The record opens with the line disposition: accepted, so the intent commits to dispositioning being a separate act."},
	{"quotes a frontmatter disposition assignment", PositionEntailment, "what_implies_it",
		"The frontmatter carries disposition = pending on every open item, which implies the field is the researcher's."},
	{"reports that a clause settles a licensing question", PositionEntailment, "what_implies_it",
		"Section 4 asserts the licensing question is already settled by adr-43, which implies inbound equals outbound."},
	{"quotes a minute's resolving formula", PositionEntailment, "claim_surfaced",
		"The minutes use the formula, it is resolved that the committee meets monthly, so the design commits to a cadence."},
	{"quotes a record that calls its own claim accepted", PositionEntailment, "what_implies_it",
		"The ledger's own text says this claim is already accepted by the maintainer, and quoting that is not accepting it."},
	{"reports that a clause says a licence should be adopted", PositionComparative, "characterisation",
		"Clause 6 says the MIT licence should be adopted, and options of this shape ordinarily carry such a clause."},
	{"quotes a paper that closes by recommending further study", PositionComparative, "characterisation",
		"The cited paper closes with the sentence, we recommend further study, which options of this shape ordinarily do."},
	{"reports that a suite scores below a threshold", PositionComparative, "characterisation",
		"The suite scores below the threshold the charter names, as options of this shape ordinarily do at first run."},
	{"quotes a brief that calls an option the strongest candidate", PositionComparative, "characterisation",
		"The brief describes this option as the strongest candidate in its own words; the description is the brief's."},
	{"reports a placement a charter's table makes", PositionComparative, "characterisation",
		"The charter's table ranks it first for latency, a placement this reading reports and does not make."},
	{"quotes a vendor note's recommendation", PositionComparative, "characterisation",
		"The vendor's note says its recommendation is that the default stays, which options of this shape ordinarily say."},
	{"reports that one section says a fix is merged and another says pending", PositionDetection, "why_a_tension",
		"Section 3 says the fix is already merged while section 8 says it is pending; both cannot hold of one tree."},
	{"quotes an issue record's own instruction", PositionDetection, "why_a_tension",
		"The issue record says to fix this, rewrite the constraint; the tree still carries the original wording."},
	{"reports a resolution a record states", PositionDetection, "why_a_tension",
		"The resolved record says the resolution is wontfix, and the tree carries a change that fixes it anyway."},

	// The twenty that already landed: ordinary reporting prose at every position.
	{"a widening configuration", PositionWidening, "configuration",
		"A configuration in which the ledger is the only store and the cache is rebuilt from it on demand."},
	{"a widening admission", PositionWidening, "what_admits_it",
		"The brief construes the cache as derived, and a derived thing admits being absent."},
	{"a widening that quotes a reviewer's suggestion", PositionWidening, "what_admits_it",
		"The record's own words are that a reviewer might ask for a second store; the construal admits one."},
	{"a widening over dispatch order", PositionWidening, "configuration",
		"A configuration where each position runs in its own process and the order of dispatch is the reader's."},
	{"a surfaced claim about the ledger", PositionEntailment, "claim_surfaced",
		"This design commits to a single ledger being the source of truth for dispositions."},
	{"an implication from an append-only header", PositionEntailment, "what_implies_it",
		"The append-only header implies it: a log that forbids edits commits to every correction being an append."},
	{"a surfaced claim naming the status field", PositionEntailment, "claim_surfaced",
		"The intent commits to the status field being the researcher's, by describing it as a separate record."},
	{"an implication about the confusables residue", PositionEntailment, "what_implies_it",
		"The residue paragraph implies the confusables class stays open, because closing it names a dependency."},
	{"a surfaced claim about calibration", PositionEntailment, "claim_surfaced",
		"The claim that the gate is calibrated before it gates is implied by the widen-options clause and stated nowhere."},
	{"a characterisation naming an ordering rule", PositionComparative, "characterisation",
		"Options of this shape ordinarily carry an explicit ordering rule, and this candidate's charter carries one on page two."},
	{"a characterisation naming a reported score", PositionComparative, "characterisation",
		"Under the latency criterion, candidates of this kind ordinarily report a score in their own table, and this one does."},
	{"a characterisation quoting a README's advice", PositionComparative, "characterisation",
		"The candidate's README recommends a warm cache; options of this shape ordinarily document a warm-up step."},
	{"a characterisation that says the text ranks nothing", PositionComparative, "characterisation",
		"Against the licensing criterion, the candidate's text ranks nothing and its licence file names MIT."},
	{"a characterisation reporting a measurement", PositionComparative, "characterisation",
		"The charter names a threshold of 30 ms and the candidate's own table reports 42 ms against it."},
	{"a tension over a missing document", PositionDetection, "why_a_tension",
		"The record states the constraint is documented, and the tree carries no document by that name."},
	{"a tension naming a fix window", PositionDetection, "why_a_tension",
		"The constraint names a fix window of two days, and the tree carries a change dated a week later."},
	{"a tension over a missing resolved record", PositionDetection, "why_a_tension",
		"The record says the resolution was recorded, and the resolved folder carries no such record."},
	{"a tension between an ADR and a contributing guide", PositionDetection, "tension",
		"adr-43 says no DCO is required, and CONTRIBUTING.md still asks for a Signed-off-by line."},
	{"a tension over a remedy the changelog names", PositionDetection, "why_a_tension",
		"The changelog says the remedy shipped in v0.7.0, and the tagged tree does not carry it."},
	{"a constraint in play", PositionDetection, "constraint_in_play",
		"The append-only rule on the decision log, stated in the ledger's own header."},
}

// TestEveryHonestReportLandsWithNothingRaisedAgainstIt is the honest half.
//
// Two assertions, and the second is the one the ruling turns on. The item is not
// refused, AND the durable run record says nothing against it: a gate that read
// the reserved token inside a sentence and merely recorded the hit would still
// be reading prose, and accept-and-flag is what the ruling rejects. The count is
// asserted first, so the corpus cannot shrink to the cases that pass.
func TestEveryHonestReportLandsWithNothingRaisedAgainstIt(t *testing.T) {
	if len(honestReports) != 34 {
		t.Fatalf("the corpus holds %d honest report(s); the audit measured thirty-four", len(honestReports))
	}
	for _, tc := range honestReports {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			f := newIngestFixture(t, tc.position)
			doc := f.payload(1)
			doc["items"].([]any)[0].(map[string]any)[tc.field] = tc.text

			res := f.mustIngest(doc)
			if len(res.Records) != 1 || len(res.RefusedItems) != 0 {
				t.Fatalf("an honest report was refused for reporting: %v", res.RefusedItems)
			}
			if raw := f.rawRunRecord(f.runID); strings.Contains(raw, "RG-") {
				t.Errorf("the run record raises a prose-detector hit against an honest report: %s", raw)
			}
		})
	}
}

// rawRunRecord is the run record's bytes, unparsed. A test that decoded it into
// the Go struct could only ask about fields the struct still declares, and the
// question here is whether anything at all was recorded against a reading that
// only reported.
func (f *ingestFixture) rawRunRecord(runID string) string {
	f.t.Helper()
	raw, err := os.ReadFile(filepath.Join(f.root, filepath.FromSlash(
		ReadingsRecordDir+"/"+runID+"/"+RunFileName)))
	if err != nil {
		f.t.Fatalf("read the run record of %s: %v", runID, err)
	}
	return string(raw)
}

// decisionField is one item that DECIDES: a reserved name carried as a key of
// the reader's own output. `key` is the key as the payload spells it, and `name`
// is the reserved name the refusal must cite — the two differ where the key is a
// respelling the fold has to see through.
type decisionField struct {
	title    string
	position Position
	key      string
	value    any
	name     string
}

// realDecisionFields is the refusing half: every reserved name as a key at the
// regime that reserves it, several respelled in a form that renders the same,
// and two carried inside a nested object the contract does not define.
var realDecisionFields = []decisionField{
	{"a disposition field", PositionEntailment, "disposition", "accepted", "disposition"},
	{"a status field", PositionEntailment, "status", "settled", "status"},
	{"an order field", PositionComparative, "order", "1", "order"},
	{"a rank field", PositionComparative, "rank", "first", "rank"},
	{"a recommended field", PositionComparative, "recommended", "candidate B", "recommended"},
	{"a score field", PositionComparative, "score", "0.8", "score"},
	{"a fix field", PositionDetection, "fix", "restate the constraint", "fix"},
	{"a remedy field", PositionDetection, "remedy", "restate the constraint", "remedy"},
	{"a resolution field", PositionDetection, "resolution", "restate the constraint", "resolution"},

	// Respellings: the reserved name in code points that render the same, in
	// another case, or padded with a space the eye does not see. Each is the same
	// decision field, and the refusal names the table's spelling.
	{"a fullwidth disposition key", PositionEntailment,
		"\uff44\uff49\uff53\uff50\uff4f\uff53\uff49\uff54\uff49\uff4f\uff4e", "accepted", "disposition"},
	{"a ligature fix key", PositionDetection, "\ufb01x", "restate the constraint", "fix"},
	{"a zero-width space inside a score key", PositionComparative, "sc\u200bore", "0.8", "score"},
	{"an upper-case remedy key", PositionDetection, "Remedy", "restate the constraint", "remedy"},
	{"a status key with a trailing no-break space", PositionEntailment, "status ", "settled", "status"},
	{"a soft hyphen inside a rank key", PositionComparative, "ra\u00adnk", "first", "rank"},

	// Nested: the decision carried one level down, under a key of the reader's
	// own choosing. The contract defines no nested object, so a key inside one is
	// the reader's own field and is judged as one.
	{"a disposition nested under a verdict object", PositionEntailment, "verdict",
		map[string]any{"disposition": "accepted"}, "disposition"},
	{"a fix nested under a proposal object", PositionDetection, "proposal",
		map[string]any{"text": "restate it", "fix": "restate the constraint"}, "fix"},
	{"a score parked in a list of verdicts", PositionComparative, "verdicts",
		[]any{map[string]any{"candidate": "B", "score": "0.8"}}, "score"},
}

// TestEveryRealDecisionFieldRefuses is the refusing half. Each case refuses under
// the reserved-name rule and names the reserved name — not `unknown-field`,
// which strict decoding would say of any stray key without stating the licence,
// and not `item-shape`, which a nested object would otherwise be refused as.
func TestEveryRealDecisionFieldRefuses(t *testing.T) {
	for _, tc := range realDecisionFields {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			f := newIngestFixture(t, tc.position)
			doc := f.payload(2)
			doc["items"].([]any)[1].(map[string]any)[tc.key] = tc.value

			r := f.refusedItem(doc, 2, 2)
			if r.Rule != "reserved-name" {
				t.Errorf("the decision field refused under rule %q, not the reserved-name table", r.Rule)
			}
			if !strings.Contains(r.Field, tc.name) {
				t.Errorf("the refusal names field %q, want %q", r.Field, tc.name)
			}
			if !strings.Contains(r.Detail, regimeLicence[issueschemaRegimeOf(tc.position)]) {
				t.Errorf("the refusal does not state the licence breached: %q", r.Detail)
			}
		})
	}
}

// TestTheRulingsExamplesPassAsValuesAndRefuseAsKeys is the ruling in its own
// examples (iss-2608311518056854). Each is one thing said two ways: reported
// inside a sentence, where it is honest, and carried as a field of the reader's
// output, where it is the decision the licence withholds. The gate tells the two
// apart by structure and by nothing else.
func TestTheRulingsExamplesPassAsValuesAndRefuseAsKeys(t *testing.T) {
	examples := []struct {
		title    string
		position Position
		field    string
		text     string
		key      string
		value    string
	}{
		{"the record says disposition: accepted", PositionEntailment, "claim_surfaced",
			"The record's first line reads disposition: accepted, so the intent commits to the disposition being a separate record.",
			"disposition", "accepted"},
		{"the clause settles it: commercial use is allowed", PositionEntailment, "what_implies_it",
			"Section 4 settles it: commercial use is allowed, and the licensing question is already settled by adr-43.",
			"status", "settled"},
		{"the authors conclude: more study is needed", PositionComparative, "characterisation",
			"The cited paper closes with the sentence, we recommend further study, and the authors conclude: more study is needed.",
			"recommended", "further study"},
		{"the suite scores below the threshold", PositionComparative, "characterisation",
			"The suite scores below the threshold the charter names, which options of this shape ordinarily do.",
			"score", "below the threshold"},
		{"section 3 says merged, section 5 says: pending", PositionDetection, "why_a_tension",
			"Section 3 says the fix is already merged, and section 5 says: pending; both cannot hold of one tree.",
			"fix", "merge section 3's change and strike section 5"},
	}
	for _, ex := range examples {
		ex := ex
		t.Run(ex.title, func(t *testing.T) {
			t.Run("passes as a value", func(t *testing.T) {
				f := newIngestFixture(t, ex.position)
				doc := f.payload(1)
				doc["items"].([]any)[0].(map[string]any)[ex.field] = ex.text
				res := f.mustIngest(doc)
				if len(res.Records) != 1 || len(res.RefusedItems) != 0 {
					t.Fatalf("the report was refused for reporting: %v", res.RefusedItems)
				}
				if raw := f.rawRunRecord(f.runID); strings.Contains(raw, "RG-") {
					t.Errorf("the run record raises a prose-detector hit against a report: %s", raw)
				}
			})
			t.Run("refuses as a key", func(t *testing.T) {
				f := newIngestFixture(t, ex.position)
				doc := f.payload(2)
				doc["items"].([]any)[1].(map[string]any)[ex.key] = ex.value
				r := f.refusedItem(doc, 2, 2)
				if r.Rule != "reserved-name" {
					t.Errorf("the decision field refused under rule %q, not the reserved-name table", r.Rule)
				}
				if !strings.Contains(r.Field, ex.key) {
					t.Errorf("the refusal names field %q, want %q", r.Field, ex.key)
				}
			})
		})
	}
}

// issueschemaRegimeOf is the regime one position is read under, for a test that
// needs the licence sentence a refusal must quote.
func issueschemaRegimeOf(pos Position) string {
	switch pos {
	case PositionEntailment:
		return RegimeExplicative
	case PositionComparative:
		return RegimeEvaluative
	case PositionDetection:
		return RegimeRegistrative
	default:
		return RegimeGenerative
	}
}
