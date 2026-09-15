package reading

// ingest_regime_test.go is the supply-regime gate: itd-185's ac-4 through ac-9
// and ac-12.
//
// Every refusal case here carries the adjacent LEGAL payload in the same test.
// ac-7 is that assertion promoted to a criterion of its own, and the same shape
// is owed to the rest: a refusal test that would pass against a verb refusing
// everything establishes nothing about the rule it names.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// TestRegimeComesFromTheDefinitionNotThePayload is ac-12 from the verb's side:
// the regime is READ, from the position's definition file, and a run whose
// definition the locator cannot resolve has no regime at all.
//
// Removing the file is the discriminator. A verb that answered from
// issueschema's position table would carry straight on — the table still binds
// detection to registrative whether or not any file exists — so this case
// separates a verb that reads the definition from one that only says it does.
func TestRegimeComesFromTheDefinitionNotThePayload(t *testing.T) {
	f := newIngestFixture(t, "detection")
	legal := f.payload(1)
	f.mustIngest(legal)

	if err := os.Remove(filepath.Join(f.root, filepath.FromSlash(DefinitionPath("detection")))); err != nil {
		t.Fatal(err)
	}
	f.parkRun("rdg-2608310000000002", "detection", AssemblerVersion())
	doc := f.payload(1)
	doc["run_id"] = "rdg-2608310000000002"
	doc["manifest_sha256"] = f.manifestHashOf("rdg-2608310000000002")

	res, err := f.ingest(doc)
	if err == nil {
		t.Fatal("a run whose definition is absent was accepted; the regime then came from somewhere " +
			"other than the definition")
	}
	if !strings.Contains(err.Error(), DefinitionPath("detection")) {
		t.Errorf("the refusal does not name the definition it could not resolve: %v", err)
	}
	// The run's identity was proven before the definition was looked up, so
	// this is a list-level refusal past the identity point, and it records.
	assertUnresolvedDefinitionRecorded(t, f, res, err, "rdg-2608310000000002")
	f.nothingDurableInTheLedger("rdg-2608310000000002")
}

// assertUnresolvedDefinitionRecorded holds a run whose definition did not
// resolve to the plugin page's sentence: from the identity point a list-level
// refusal writes refusal.json, and the message names it. The record states no
// regime, because the verb could not resolve one — a regime it wrote there would
// be a claim about a definition it never read.
func assertUnresolvedDefinitionRecorded(t *testing.T, f *ingestFixture, res IngestResult, err error, runID string) {
	t.Helper()
	wantPath := ReadingsRecordDir + "/" + runID + "/" + RefusalFileName
	if res.RefusalPath != wantPath {
		t.Fatalf("a list-level refusal after the run's identity was proven wrote no refusal record "+
			"(refusal_record is %q, want %s); the run was refused and nothing says so", res.RefusalPath, wantPath)
	}
	if !strings.Contains(err.Error(), wantPath) {
		t.Errorf("the refusal message does not name the record it wrote: %v", err)
	}
	rec := f.readRefusalRecord(runID)
	if rec.Type != RefusalType || rec.RunID != runID || rec.Position != "detection" {
		t.Errorf("the refusal record carries the wrong run metadata: %+v", rec)
	}
	if rec.Regime != "" {
		t.Errorf("the refusal record states the %q regime, which the verb never resolved", rec.Regime)
	}
	if !strings.Contains(rec.Reason, DefinitionPath("detection")) {
		t.Errorf("the recorded reason does not name the definition that did not resolve: %q", rec.Reason)
	}
	if strings.HasPrefix(rec.Reason, "reading: ") {
		t.Errorf("the recorded reason carries the core's own prefix: %q", rec.Reason)
	}
}

// TestADriftedDefinitionRefusesTheRunRatherThanChangingTheLicence is the
// adversarial half of ac-12, which the criterion does not state: ac-12 is the
// PAYLOAD lying about its regime, and this is the DEFINITION lying.
//
// A file stating a legal regime under the wrong position would hand this verb
// the wrong licence with no refusal anywhere in the path — the gate whose whole
// purpose is to catch a reading that exceeded its licence would then be
// enforcing a different one, silently. The refusal lives in the locator, which
// is the one thing that claims to resolve a position to its regime
// (iss-2608311145258479); this case holds the ingest path to it, so the fix
// cannot be undone in the locator without a reading-side test going red.
//
// The refusal IS recorded. The run's identity is proven before the definition is
// resolved — it names a parked run and cites that run's manifest by content hash
// — and from that point every list-level refusal leaves a record, which is what
// ac-10 and the plugin page both say. The earlier reading, that a run whose
// definition does not resolve has no regime the verb could honestly write down,
// argued for writing nothing at all; the honest form is a record whose `regime`
// is EMPTY and whose reason says why, because that is a fact an operator can
// find, and a refusal nobody recorded is not (iss-2608311518250688).
func TestADriftedDefinitionRefusesTheRunRatherThanChangingTheLicence(t *testing.T) {
	f := newIngestFixture(t, "detection")
	f.mustIngest(f.payload(1))

	// The detection definition now states the evaluative licence. Under it, the
	// evaluative reserved names would be enforced against a registrative
	// reading, and the registrative ones would not be enforced at all.
	f.writeDefinition("detection", RegimeEvaluative)

	f.parkRun("rdg-2608310000000007", "detection", AssemblerVersion())
	doc := f.payload(1)
	doc["run_id"] = "rdg-2608310000000007"
	doc["manifest_sha256"] = f.manifestHashOf("rdg-2608310000000007")

	res, err := f.ingest(doc)
	if err == nil {
		t.Fatal("a definition stating another position's regime was resolved rather than refused, so " +
			"the run read under a licence nothing checked")
	}
	for _, want := range []string{RegimeEvaluative, RegimeRegistrative, DefinitionPath("detection")} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}
	assertUnresolvedDefinitionRecorded(t, f, res, err, "rdg-2608310000000007")
	rec := f.readRefusalRecord("rdg-2608310000000007")
	if rec.ManifestSHA256 == "" || rec.TargetCommit == "" {
		t.Errorf("the refusal record does not carry the run's manifest reference: %+v", rec)
	}
	// The reason names both regimes and says the definition was refused rather
	// than resolved: the record has to carry what the message carries.
	for _, want := range []string{RegimeEvaluative, RegimeRegistrative, "refused rather than resolved"} {
		if !strings.Contains(rec.Reason, want) {
			t.Errorf("the recorded reason does not name %q: %q", want, rec.Reason)
		}
	}
	f.nothingDurableInTheLedger("rdg-2608310000000007")
}

// TestSelfDeclaredRegimeMismatchRefusesRun is ac-12: a self-declared regime that
// disagrees with the definition refuses the RUN, at list level, not one item.
func TestSelfDeclaredRegimeMismatchRefusesRun(t *testing.T) {
	f := newIngestFixture(t, "detection")
	doc := f.payload(3)
	doc["regime"] = RegimeGenerative

	res, err := f.ingest(doc)
	if err == nil {
		t.Fatal("an output declaring a regime the definition does not state was accepted")
	}
	for _, want := range []string{RegimeGenerative, RegimeRegistrative} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}
	if len(res.RefusedItems) != 0 {
		t.Errorf("a list-level refusal reported %d item refusals; the run is wrong in whole",
			len(res.RefusedItems))
	}
	if got := f.ledgerRecords(f.runID); len(got) != 0 {
		t.Errorf("the refused run wrote %v", got)
	}
	f.mustIngest(f.nextRun(f.payload(3)))
}

// TestEvaluativeRankScoreRecommendedRefused is ac-6: each reserved name on the
// evaluative table refuses, and the refusal names the field.
func TestEvaluativeRankScoreRecommendedRefused(t *testing.T) {
	for _, field := range ReservedNames[RegimeEvaluative] {
		field := field
		t.Run(field, func(t *testing.T) {
			f := newIngestFixture(t, "comparative")
			doc := f.payload(2)
			doc["items"].([]any)[1].(map[string]any)[field] = "1"

			r := f.refusedItem(doc, 2, 2)
			if !strings.Contains(r.Field, field) {
				t.Errorf("the refusal names field %q, want %q", r.Field, field)
			}
			if !strings.Contains(r.Detail, field) {
				t.Errorf("the refusal does not name the field: %q", r.Detail)
			}
			if r.Rule != "reserved-name" {
				t.Errorf("the refusal cites rule %q, not the reserved-name table", r.Rule)
			}
		})
	}
}

// TestEvaluativeDocumentOrderIsNeverRefused is ac-7, and it is the reason the
// reserved table names a FIELD rather than a property of the list: items arrive
// in document order by mandate, and arrangement order is never inspected.
func TestEvaluativeDocumentOrderIsNeverRefused(t *testing.T) {
	f := newIngestFixture(t, "comparative")
	doc := f.payload(3)
	// A deliberate arrangement: the three items are ordered, and none carries a
	// reserved field. Nothing about the ORDER may be refused.
	// Three DIFFERENT candidates of the derived run, in an order chosen by this
	// test. The ids are the run's own, because a comparative item names a
	// candidate the manifest records (spc-2609020626039834); what this case is
	// about is the ARRANGEMENT, which nothing may inspect.
	for i, raw := range doc["items"].([]any) {
		raw.(map[string]any)["candidate_id"] = fixtureIngestCandidates[i]
	}
	res := f.mustIngest(doc)
	if len(res.Records) != 3 {
		t.Fatalf("an arranged evaluative output landed %d of 3 items", len(res.Records))
	}
	if len(res.RefusedItems) != 0 {
		t.Errorf("arrangement order was refused: %v", res.RefusedItems)
	}
}

// TestRegistrativeResolutionFieldRefused is ac-4: a registrative reserved name
// refuses, and the refusal names the item's ordinal, the field, and the licence.
func TestRegistrativeResolutionFieldRefused(t *testing.T) {
	for _, field := range ReservedNames[RegimeRegistrative] {
		field := field
		t.Run(field, func(t *testing.T) {
			f := newIngestFixture(t, "detection")
			doc := f.payload(3)
			// The SECOND of three, so the ordinal in the message is not the only
			// one a verb could have printed.
			doc["items"].([]any)[1].(map[string]any)[field] = "rewrite the constraint"

			r := f.refusedItem(doc, 2, 3)
			if r.Ordinal != 2 {
				t.Errorf("the refusal names ordinal %d, want 2", r.Ordinal)
			}
			if !strings.Contains(r.Field, field) {
				t.Errorf("the refusal names field %q, want %q", r.Field, field)
			}
			if !strings.Contains(r.Detail, "item 2") {
				t.Errorf("the refusal does not name the item's ordinal: %q", r.Detail)
			}
			if !strings.Contains(r.Detail, "not its licence") {
				t.Errorf("the refusal does not state the licence breached: %q", r.Detail)
			}
		})
	}
}

// TestExplicativeDispositionRefused is the structural half of ac-8: a
// disposition-bearing FIELD on a surfaced claim refuses, and the refusal names
// the field.
func TestExplicativeDispositionRefused(t *testing.T) {
	for _, field := range ReservedNames[RegimeExplicative] {
		field := field
		t.Run(field, func(t *testing.T) {
			f := newIngestFixture(t, "entailment")
			doc := f.payload(2)
			doc["items"].([]any)[1].(map[string]any)[field] = "accepted"

			r := f.refusedItem(doc, 2, 2)
			if !strings.Contains(r.Field, field) {
				t.Errorf("the refusal names field %q, want %q", r.Field, field)
			}
			if r.Rule != "reserved-name" {
				t.Errorf("the refusal cites rule %q, not the reserved-name table", r.Rule)
			}
		})
	}
}

// TestGenerativeHasNoRegimeRefusal is the generative path, which carries no
// criterion of its own and is recorded on spc-63 rather than left to be
// discovered at the audit. The generative licence is the widest and the
// constraint on it falls at admission, so no reserved-name row exists for it and
// an item proposing a configuration lands.
func TestGenerativeHasNoRegimeRefusal(t *testing.T) {
	f := newIngestFixture(t, PositionWidening)
	doc := f.payload(1)
	doc["items"].([]any)[0].(map[string]any)["what_admits_it"] =
		"the criterion the record states; we recommend this configuration"

	res := f.mustIngest(doc)
	if len(res.Records) != 1 {
		t.Fatalf("the generative item did not land: %d record(s)", len(res.Records))
	}
	if len(res.RefusedItems) != 0 {
		t.Errorf("the generative regime refused an item: %v", res.RefusedItems)
	}
	if _, has := ReservedNames[RegimeGenerative]; has {
		t.Error("the generative regime carries a reserved-name row; its constraint falls at admission")
	}
}

// TestReservedNamesNeverCollideWithABodyField holds the reserved tables honest:
// a reserved name that was also a position's body field would refuse every legal
// output at that position, and the reserved-name refusal would be unreachable.
func TestReservedNamesNeverCollideWithABodyField(t *testing.T) {
	for regime, names := range ReservedNames {
		for _, p := range Positions() {
			if issueschema.ReadingRegime(string(p)) != regime {
				continue
			}
			for _, field := range issueschema.ReadingBodyFields[string(p)] {
				for _, name := range names {
					if field == name {
						t.Errorf("%q is both a reserved name at the %s regime and a body field of the "+
							"%s position", name, regime, p)
					}
				}
			}
		}
	}
}

// TestInnocentProseCarryingFoldedRunesStillLands is the fold's other half, and
// the more important one.
//
// `foldForMatching` widens what a comparison sees — every Unicode space to
// ASCII, every invisible rune dropped, NFKC over the compatibility forms — and
// it now serves two rules: the blankness rules, and the reserved-name rule over
// an item's KEYS. Neither reads prose. So every case here is ordinary prose that
// legitimately carries one of the runes being folded, and every one must LAND: a
// widening of the fold that reached the values would cost a reading its
// findings, which is the failure the 2026-09-01 ruling closed.
//
// The evasion side — a reserved name spelled as a KEY in code points that render
// the same — is in ingest_corpus_test.go, beside the corpus it is measured on.
func TestInnocentProseCarryingFoldedRunesStillLands(t *testing.T) {
	const (
		nbsp       = "\u00a0"
		narrowNbsp = "\u202f"
	)
	innocent := []struct{ name, text string }{
		{"non-breaking space in a measurement", "the constraint names a budget of 30" + nbsp + "ms"},
		{"soft hyphen at a line break", "the constraint is docu\u00admented in the record"},
		{"narrow no-break space in a figure", "the budget is 30" + narrowNbsp + "%, and the path exceeds it"},
		{"zero-width non-joiner in a compound", "the record names a well\u200cknown constraint"},
		{"combining acute", "the re\u0301cord and the tree disagree about the constraint"},
		{"variation selector after a glyph", "the record marks it \u2714\ufe0f and the tree does not"},
		{"the bare word fix", "the constraint names a fix window, and the tree has none"},
		{"the bare word order", "the record states an order of operations the tree does not keep"},
		{"the word recommend in quoted material", "the passed material uses the word recommend"},
		{"ideographic text", "the record states \u5236\u7d04 and the tree does not"},
		{"an em dash and curly quotes", "the record says \u201cone thing\u201d \u2014 the tree says another"},
		{"a ligature in ordinary prose", "the record de\ufb01nes the constraint the tree ignores"},
	}
	for _, tc := range innocent {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			f := newIngestFixture(t, "detection")
			doc := f.payload(1)
			doc["items"].([]any)[0].(map[string]any)["why_a_tension"] = tc.text

			res := f.mustIngest(doc)
			if len(res.Records) != 1 {
				t.Fatalf("innocent prose was refused: %v", res.RefusedItems)
			}
			if len(res.RefusedItems) != 0 {
				t.Errorf("the fold reached a value and refused it: %v", res.RefusedItems)
			}
		})
	}
}

// TestTheProvenanceFieldIsJudgedByTheSameRuleAsAnyOther keeps
// iss-2608311517547712's finding answered under the new rule.
//
// That issue found `pattern` to be a channel no detector read, so an item whose
// provenance field carried the registry's own phrasing landed unremarked. There
// is no prose detector now, at any field, so the asymmetry is gone by
// construction: what the gate judges is the item's KEY SET, and `pattern` is one
// key in it. A reserved name reaches the same refusal whichever key it is under,
// and an ordinary pattern — whatever it says — lands.
func TestTheProvenanceFieldIsJudgedByTheSameRuleAsAnyOther(t *testing.T) {
	t.Run("a pattern quoting a fix proposal lands", func(t *testing.T) {
		f := newIngestFixture(t, "detection")
		doc := f.payload(1)
		doc["items"].([]any)[0].(map[string]any)[PatternField] =
			"P-1. The record says the fix is to rewrite the charter."

		res := f.mustIngest(doc)
		if len(res.Records) != 1 || len(res.RefusedItems) != 0 {
			t.Fatalf("a pattern reporting what the material says was refused: %v", res.RefusedItems)
		}
	})

	t.Run("a reserved name beside the pattern still refuses", func(t *testing.T) {
		f := newIngestFixture(t, "detection")
		doc := f.payload(2)
		item := doc["items"].([]any)[1].(map[string]any)
		item[PatternField] = "P-2. The record and the tree disagree."
		item["fix"] = "rewrite the charter"

		r := f.refusedItem(doc, 2, 2)
		if r.Rule != "reserved-name" {
			t.Errorf("the refusal cites rule %q, not the reserved-name table", r.Rule)
		}
	})
}

// TestTheElisionEntryNamesNoItemInAnyDurableRecord.
//
// A bounded refusal list ends in an entry that is not an item: "and N more
// item(s) refused". It carries no ordinal, and the zero value of an ordinal is
// 0 — so both durable records asserted an item 0 that does not exist. The
// terminal renderer had the branch that suppresses it and neither record writer
// did, which is the shape a claim takes when it lives in one surface instead of
// in the value.
func TestTheElisionEntryNamesNoItemInAnyDurableRecord(t *testing.T) {
	t.Run("the run record", func(t *testing.T) {
		f := newIngestFixture(t, "detection")
		// More refusals than the cap, and survivors, so the run lands and its
		// commit marker carries the bounded list.
		doc := f.payload(maxReportedRefusals + 5)
		items := doc["items"].([]any)
		for i := 0; i < maxReportedRefusals+3; i++ {
			items[i].(map[string]any)[PatternField] = ""
		}
		res := f.mustIngest(doc)
		if res.RefusedCount != maxReportedRefusals+3 {
			t.Fatalf("the run refused %d item(s), want %d", res.RefusedCount, maxReportedRefusals+3)
		}
		run := f.readRunRecord(f.runID)
		last := run.RefusedItems[len(run.RefusedItems)-1]
		if last.Rule != "refusals-elided" {
			t.Fatalf("the bounded list does not end in the elision entry: %+v", last)
		}
		if last.Ordinal != 0 {
			t.Fatalf("the elision entry carries ordinal %d", last.Ordinal)
		}
		raw, err := os.ReadFile(filepath.Join(f.root, filepath.FromSlash(
			ReadingsRecordDir+"/"+f.runID+"/"+RunFileName)))
		if err != nil {
			t.Fatal(err)
		}
		var doc2 struct {
			RefusedItems []map[string]any `json:"refused_items"`
		}
		if err := json.Unmarshal(raw, &doc2); err != nil {
			t.Fatal(err)
		}
		entry := doc2.RefusedItems[len(doc2.RefusedItems)-1]
		if _, named := entry["ordinal"]; named {
			t.Errorf("the run record's elision entry states an ordinal: %v; there is no item 0, and a "+
				"reader sent looking for one finds nothing", entry)
		}
	})

	t.Run("the refusal record", func(t *testing.T) {
		f := newIngestFixture(t, "detection")
		// Every item refused, so the run is refused at list level and the reason
		// carries the bounded list into refusal.json.
		doc := f.payload(maxReportedRefusals + 5)
		for _, it := range doc["items"].([]any) {
			it.(map[string]any)[PatternField] = ""
		}
		if _, err := f.ingest(doc); err == nil {
			t.Fatal("a run in which every item was refused was accepted")
		}
		rec := f.readRefusalRecord(f.runID)
		if !strings.Contains(rec.Reason, "more item(s) refused") {
			t.Fatalf("the reason does not carry the elision entry: %q", rec.Reason)
		}
		if strings.Contains(rec.Reason, "item 0") {
			t.Errorf("the refusal record names an item 0: %q", rec.Reason)
		}
	})
}

// TestAPatternThatRendersAsNothingIsRefused.
//
// U+2800 BRAILLE PATTERN BLANK renders as nothing in every common font, but it
// is a graphic character: not Cf, not Other_Default_Ignorable_Code_Point, not a
// Variation_Selector, and unicode.IsSpace is false for it. So it cleared the
// blankness test, and a record asserted a provenance that renders blank — the
// failure the U+200B fix closed, one category further out.
func TestAPatternThatRendersAsNothingIsRefused(t *testing.T) {
	for _, p := range Positions() {
		p := p
		t.Run(string(p), func(t *testing.T) {
			f := newIngestFixture(t, p)
			doc := f.payload(2)
			doc["items"].([]any)[1].(map[string]any)[PatternField] = "⠀⠀"

			r := f.refusedItem(doc, 2, 2)
			if r.Rule != "named-provenance" {
				t.Errorf("the refusal cites rule %q, want named-provenance", r.Rule)
			}
			if r.Field != PatternField {
				t.Errorf("the refusal names field %q", r.Field)
			}
		})
	}

	// A braille pattern with dots is ordinary text and still lands: the fold
	// closes the blank, not the block.
	t.Run("a braille pattern with dots lands", func(t *testing.T) {
		f := newIngestFixture(t, "detection")
		doc := f.payload(1)
		doc["items"].([]any)[0].(map[string]any)[PatternField] = "⠁⠃"
		if res := f.mustIngest(doc); len(res.Records) != 1 {
			t.Errorf("a braille pattern carrying dots was refused: %v", res.RefusedItems)
		}
	})
}
