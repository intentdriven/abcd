package reading

// ingest_schema_test.go is itd-185's ac-1 and the structural half of ac-8: a
// malformed output is refused, the refusal names the offending field, and
// nothing durable exists anywhere for that run.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// TestPatternFieldIsTheRecordEnvelopeField is the executable form of a claim
// this contract would otherwise only assert in prose: the payload's provenance
// key is the reading RECORD's own envelope field, so the payload, the record and
// the four definitions say one word for one thing.
func TestPatternFieldIsTheRecordEnvelopeField(t *testing.T) {
	found := false
	for _, f := range issueschema.ReadingRequired {
		if f == PatternField {
			found = true
		}
	}
	if !found {
		t.Fatalf("the payload's provenance key is %q, which is not one of the reading record's envelope "+
			"fields %v; a rename between the payload and the record is a translation nobody asked for",
			PatternField, issueschema.ReadingRequired)
	}
	for _, p := range Positions() {
		for _, field := range issueschema.ReadingBodyFields[string(p)] {
			if field == PatternField {
				t.Errorf("the %s position declares %q as a BODY field; the pattern is an envelope field, "+
					"because a universal core condition must not live in a variant part", p, field)
			}
		}
	}
}

// TestMalformedPayloadWritesNothing is ac-1. Four shapes of malformed, and after
// each the run has left nothing in the reading-record family, nothing in the
// readings tree, and nothing in the stage.
func TestMalformedPayloadWritesNothing(t *testing.T) {
	cases := []struct {
		name string
		doc  func(f *ingestFixture) any
		want string
	}{
		{"not json", func(f *ingestFixture) any { return nil }, ""},
		{"wrong type tag", func(f *ingestFixture) any {
			d := f.payload(1)
			d["_type"] = "abcd.reading.output/9"
			return d
		}, "_type"},
		{"run id is not a run id", func(f *ingestFixture) any {
			d := f.payload(1)
			d["run_id"] = "rdg-../../etc"
			return d
		}, "run_id"},
		{"unknown position", func(f *ingestFixture) any {
			d := f.payload(1)
			d["position"] = "speculative"
			return d
		}, "speculative"},
		// An EMPTY item list is not malformed. It was in this table, and the
		// refusal read the framework's clean-run contingency backwards: a run
		// that returned nothing is committed as a run with an empty item set, at
		// every position (framework section 13; the corrections ruling (4) of
		// 2026-09-02; iss-2609021153269181). What replaces it here is a payload
		// that is empty AND malformed, so the refusal this test asserts stays
		// reachable and cannot be read as covering emptiness itself —
		// TestIngestCommitsAnEmptyRunAtEveryPosition holds the other side.
		{"no items and no instrument", func(f *ingestFixture) any {
			d := f.payload(0)
			d["instrument"] = map[string]any{"model": "", "definition_sha256": "", "assembler_version": ""}
			return d
		}, "instrument"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newIngestFixture(t, "detection")
			var err error
			if tc.name == "not json" {
				path := filepath.Join(t.TempDir(), "output.json")
				if writeErr := os.WriteFile(path, []byte("{not json"), 0o644); writeErr != nil {
					t.Fatal(writeErr)
				}
				_, err = Ingest(IngestRequest{RepoRoot: f.root, OutputPath: path})
			} else {
				_, err = f.ingest(tc.doc(f))
			}
			if err == nil {
				t.Fatal("a malformed output was accepted")
			}
			if tc.want != "" && !strings.Contains(err.Error(), tc.want) {
				t.Errorf("the refusal does not name %q: %v", tc.want, err)
			}
			f.nothingDurable(f.runID)
		})
	}
}

// TestUnknownFieldRefusedAtEveryLevel is ac-1's "names the offending field" at
// each of the three levels a payload has: the envelope, the instrument, and the
// item. The adjacent legal payload is accepted in the same test, so a verb that
// refused everything could not pass it.
func TestUnknownFieldRefusedAtEveryLevel(t *testing.T) {
	f := newIngestFixture(t, "detection")

	envelope := f.payload(1)
	envelope["reviewer_notes"] = "smuggled"
	if _, err := f.ingest(envelope); err == nil {
		t.Error("an unknown ENVELOPE field was accepted")
	} else if !strings.Contains(err.Error(), "reviewer_notes") {
		t.Errorf("the envelope refusal does not name the field: %v", err)
	}

	instrument := f.payload(1)
	instrument["instrument"].(map[string]any)["temperature"] = "0.2"
	if _, err := f.ingest(instrument); err == nil {
		t.Error("an unknown INSTRUMENT field was accepted")
	} else if !strings.Contains(err.Error(), "temperature") {
		t.Errorf("the instrument refusal does not name the field: %v", err)
	}

	// Neither of the two above reached the item level, so nothing at all is
	// durable yet: an envelope that does not decode names no run to record
	// against.
	f.nothingDurable(f.runID)

	// The item level, where the refusal is the ITEM's and the rest of the
	// payload still lands.
	item := f.payload(2)
	item["items"].([]any)[1].(map[string]any)["confidence"] = "high"
	r := f.refusedItem(item, 2, 2)
	if r.Rule != "unknown-field" {
		t.Errorf("the item refusal cites rule %q", r.Rule)
	}
	if !strings.Contains(r.Field, "confidence") {
		t.Errorf("the item refusal does not name the field: %q", r.Field)
	}
}

// TestWrongPositionBodyIsUndecodable is the second half of ac-8: an item
// carrying another position's body is refused field by field, so a licence
// cannot be smuggled in under a body that belongs somewhere else.
func TestWrongPositionBodyIsUndecodable(t *testing.T) {
	f := newIngestFixture(t, "detection")
	doc := f.payload(1)
	item := map[string]any{PatternField: "a pattern"}
	// The comparative position's body, at the detection position.
	for _, field := range issueschema.ReadingBodyFields["comparative"] {
		item[field] = "text"
	}
	doc["items"] = []any{item}

	_, err := f.ingest(doc)
	if err == nil {
		t.Fatal("an item carrying another position's body was accepted")
	}
	for _, field := range issueschema.ReadingBodyFields["comparative"] {
		if !strings.Contains(err.Error(), field) {
			t.Errorf("the refusal does not name the foreign field %q: %v", field, err)
		}
	}
	f.nothingDurableInTheLedger(f.runID)
	f.mustIngest(f.nextRun(f.payload(1)))
}

// TestMissingBodyFieldRefusesTheItem is the body schema's other direction, and
// it exists because removing the guard left every other case green: an item
// carrying a FOREIGN body trips the unknown-key check first, so nothing reached
// the missing-field branch (iss-2608311206480573). What it catches is a reading
// that returned a partial item at its OWN position — a tension named with no
// account of why it is one.
//
// Both the empty and the absent form are tried; only one of them is an obviously
// missing field in the payload bytes.
func TestMissingBodyFieldRefusesTheItem(t *testing.T) {
	fields := issueschema.ReadingBodyFields["detection"]
	if len(fields) < 2 {
		t.Fatalf("the detection body declares %d field(s); this case needs one to remove", len(fields))
	}
	// The invisible forms are here for the reason they are on the pattern: a
	// declared body field holding one zero-width rune states nothing, and
	// strings.TrimSpace does not treat it as blank. U+034F is a MARK and U+FE00 a
	// variation selector, so the Cf category alone does not reach either.
	forms := map[string]string{
		"empty": "", "absent": "", "zero-width space": "\u200b",
		"soft hyphen": "\u00ad", "combining grapheme joiner": "\u034f",
		"variation selector": "\ufe00", "non-breaking space": "\u00a0\u00a0",
	}
	for _, field := range fields {
		field := field
		for _, form := range []string{"empty", "absent", "zero-width space",
			"soft hyphen", "combining grapheme joiner", "variation selector", "non-breaking space"} {
			t.Run(field+"/"+form, func(t *testing.T) {
				f := newIngestFixture(t, "detection")
				doc := f.payload(2)
				item := doc["items"].([]any)[1].(map[string]any)
				if form == "absent" {
					delete(item, field)
				} else {
					item[field] = forms[form]
				}

				r := f.refusedItem(doc, 2, 2)
				if r.Rule != "missing-body-field" {
					t.Errorf("the refusal cites rule %q", r.Rule)
				}
				if !strings.Contains(r.Field, field) {
					t.Errorf("the refusal names field %q, want %q", r.Field, field)
				}
				if !strings.Contains(r.Detail, "item 2") {
					t.Errorf("the refusal does not name the item's ordinal: %q", r.Detail)
				}
			})
		}
	}
}

// TestNoDurableWriteBeforeValidation is ac-1 stated as an ordering: a payload
// whose LAST item is illegal leaves nothing behind, even though its earlier
// items are perfectly legal. Validation is step one of four, and the durable
// move happens only after the whole payload has been judged.
func TestNoDurableWriteBeforeValidation(t *testing.T) {
	f := newIngestFixture(t, "detection")
	doc := f.payload(3)
	// Make every item illegal, so the run is refused at list level rather than
	// landing two of three.
	for _, raw := range doc["items"].([]any) {
		delete(raw.(map[string]any), PatternField)
	}
	if _, err := f.ingest(doc); err == nil {
		t.Fatal("a payload whose every item was illegal was accepted")
	}
	if got := f.ledgerRecords(f.runID); len(got) != 0 {
		t.Errorf("the refused run wrote %v into the reading-record family", got)
	}
	if f.exists(IngestStageDir + "/" + f.runID) {
		t.Error("the refused run left a stage behind")
	}
	f.mustIngest(f.nextRun(f.payload(3)))
}

// hostileRune reports whether r is one of the classes termsafe.Sanitize masks:
// a C0 control (ESC among them), DEL, the C1 range, a bidi override or isolate,
// or a zero-width character. The list is stated here rather than imported so
// this case fails if the sanitiser stops masking one of them.
func hostileRune(r rune) bool {
	switch {
	case r < 0x20 && r != '\n':
		return true
	case r == 0x7f, r >= 0x80 && r <= 0x9f:
		return true
	case r >= 0x202a && r <= 0x202e, r >= 0x2066 && r <= 0x2069:
		return true
	case r == 0x200b, r == 0x200c, r == 0x200d, r == 0xfeff:
		return true
	}
	return false
}

// hostileText carries one rune from each class the sanitiser masks: an ANSI
// escape that could recolour or overprint the message reporting it, a bell, a
// right-to-left override, and a zero-width space.
const hostileText = "\x1b[31m\x07\u202ereversed\u200b\x1b[2K"

// TestARefusalNeverEchoesRawPayloadBytes is the trust boundary at the OUTPUT
// end. A refusal quotes model-produced text back to a terminal and writes it
// into a durable record, so a payload carrying an escape sequence could rewrite
// the very message that reports it, and a payload carrying megabytes in one
// field could drown it.
//
// Both guards were mutation-vacuous when they were written — neutralising the
// sanitiser and neutralising the cap each left every test green
// (iss-2608311211235195) — so this case walks the fields an untrusted value
// actually reaches a message through, one at a time.
func TestARefusalNeverEchoesRawPayloadBytes(t *testing.T) {
	long := strings.Repeat("z", 3000)

	for _, field := range []string{"_type", "run_id", "position", "regime", "manifest_sha256"} {
		field := field
		for _, oversized := range []bool{false, true} {
			name, value := field+"/hostile", hostileText
			if oversized {
				name, value = field+"/oversized", long
			}
			t.Run(name, func(t *testing.T) {
				f := newIngestFixture(t, "detection")
				doc := f.payload(1)
				doc[field] = value

				_, err := f.ingest(doc)
				if err == nil {
					t.Fatalf("%s carrying untrusted text was accepted", field)
				}
				assertSafeEcho(t, err.Error(), oversized)
			})
		}
	}

	// An item's KEY is payload text too, and it is the one refusal field built
	// from payload-chosen NAMES rather than from a table. Two items, so the run
	// lands and the refusal reaches the DURABLE run record and the render — where
	// encoding/json escapes C0 and leaves C1, DEL, bidi overrides and zero-width
	// runes raw, so a cap and a mask are the only things standing there.
	t.Run("item key", func(t *testing.T) {
		f := newIngestFixture(t, "detection")
		doc := f.payload(2)
		doc["items"].([]any)[1].(map[string]any)[hostileText+strings.Repeat("k", 3000)] = "x"

		r := f.refusedItem(doc, 2, 2)
		assertSafeEcho(t, r.Field, true)
		assertSafeEcho(t, r.Detail, false)

		run := f.readRunRecord(f.runID)
		if len(run.RefusedItems) != 1 {
			t.Fatalf("the run record carries %d refusal(s)", len(run.RefusedItems))
		}
		assertSafeEcho(t, run.RefusedItems[0].Field, true)
	})

	// And the NUMBER of names is capped as well as each name: a per-name cap
	// bounds nothing when the payload chooses how many names there are.
	t.Run("many item keys", func(t *testing.T) {
		f := newIngestFixture(t, "detection")
		doc := f.payload(2)
		item := doc["items"].([]any)[1].(map[string]any)
		for i := 0; i < 500; i++ {
			item[fmt.Sprintf("smuggled_%03d", i)] = "x"
		}
		r := f.refusedItem(doc, 2, 2)
		if n := strings.Count(r.Field, ", ") + 1; n > maxQuotedNames+1 {
			t.Errorf("the refusal quotes %d names, past the %d-name cap: %q", n, maxQuotedNames, r.Field)
		}
		if !strings.Contains(r.Field, "more") {
			t.Errorf("the refusal does not say how many names it left out: %q", r.Field)
		}
	})

	// And the DURABLE record: a list-level refusal writes the instrument's model
	// into refusal.json, where the same rule has to hold.
	t.Run("durable refusal record", func(t *testing.T) {
		f := newIngestFixture(t, "detection")
		doc := f.payload(1)
		doc["regime"] = RegimeEvaluative
		doc["instrument"].(map[string]any)["model"] = hostileText + long

		if _, err := f.ingest(doc); err == nil {
			t.Fatal("a regime mismatch was accepted")
		}
		rec := f.readRefusalRecord(f.runID)
		assertSafeEcho(t, rec.Instrument.Model, true)
		assertSafeEcho(t, rec.Reason, false)
	})
}

// assertSafeEcho holds one rendered message to both halves of the rule: no
// terminal-display attack rune survives, and a payload value that arrived
// oversized is capped rather than reproduced.
func assertSafeEcho(t *testing.T, msg string, expectCapped bool) {
	t.Helper()
	for i, r := range msg {
		if hostileRune(r) {
			t.Errorf("the message carries U+%04X at byte %d, which the sanitiser masks: %q", r, i, msg)
			return
		}
	}
	if expectCapped && strings.Contains(msg, strings.Repeat("z", maxEchoedBytes+1)) {
		t.Errorf("an oversized payload value reached the message uncapped (%d bytes): %.200q", len(msg), msg)
	}
}

// TestAnOversizeItemIsRefusedRatherThanWrittenUnreadable: nothing between the
// payload cap and the record write enforced the family's read limit, so an item
// large enough became a committed record every reader then refuses — including
// the disposition that is the only way to answer it. Durable, and permanently
// unanswerable.
func TestAnOversizeItemIsRefusedRatherThanWrittenUnreadable(t *testing.T) {
	f := newIngestFixture(t, "detection")
	doc := f.payload(2)
	doc["items"].([]any)[1].(map[string]any)["why_a_tension"] =
		strings.Repeat("the record and the tree disagree. ", issueschema.RecordReadLimit/16)

	r := f.refusedItem(doc, 2, 2)
	if r.Rule != "record-too-large" {
		t.Errorf("the refusal cites rule %q", r.Rule)
	}

	f.assertEveryRecordIsReadable()
}

// TestEscapingCannotPushARecordPastTheLimit is the same criterion at the
// boundary, and it is the case the first estimate failed.
//
// The size is measured before the record writer escapes, so an item of double
// quotes — each written as two bytes — landed a record roughly twice the
// measured size, straight past the limit. Durable, committed, and unreadable by
// every reader of the family, which is the outcome the check exists to prevent.
// This body passes a single-counted estimate and fails a double-counted one.
func TestEscapingCannotPushARecordPastTheLimit(t *testing.T) {
	f := newIngestFixture(t, "detection")
	doc := f.payload(2)
	doc["items"].([]any)[1].(map[string]any)["why_a_tension"] =
		strings.Repeat(`"`, 3*issueschema.RecordReadLimit/4)

	r := f.refusedItem(doc, 2, 2)
	if r.Rule != "record-too-large" {
		t.Errorf("the refusal cites rule %q", r.Rule)
	}
	f.assertEveryRecordIsReadable()
}

// TestTheRefusalListIsBoundedInCount: the item COUNT is payload-chosen, so a cap
// on the names quoted inside one refusal bounds nothing. A payload of many
// illegal items produced a refusal record and a terminal message hundreds of
// kilobytes long — a record whose whole purpose is to be read.
//
// The total is reported separately, so bounding the list hides nothing.
func TestTheRefusalListIsBoundedInCount(t *testing.T) {
	t.Run("an accepted run with many refused items", func(t *testing.T) {
		f := newIngestFixture(t, "detection")
		const items = 200
		doc := f.payload(items)
		for i, raw := range doc["items"].([]any) {
			if i == 0 {
				continue
			}
			delete(raw.(map[string]any), PatternField)
		}

		res := f.mustIngest(doc)
		if res.RefusedCount != items-1 {
			t.Errorf("refused_count is %d, want %d: the total is what nothing truncates",
				res.RefusedCount, items-1)
		}
		if len(res.RefusedItems) > maxReportedRefusals+1 {
			t.Errorf("the result carries %d refusals, past the %d-refusal cap",
				len(res.RefusedItems), maxReportedRefusals)
		}
		run := f.readRunRecord(f.runID)
		if len(run.RefusedItems) > maxReportedRefusals+1 {
			t.Errorf("the run record carries %d refusals, past the cap", len(run.RefusedItems))
		}
		// The DURABLE record must not carry the elision entry as an item
		// either: an ordinal of zero in run.json names an item 0 as surely as
		// the text render would (iss-2608311518250688).
		rawRun, err := os.ReadFile(filepath.Join(f.root, filepath.FromSlash(
			ReadingsRecordDir+"/"+f.runID+"/"+RunFileName)))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(rawRun), `"ordinal": 0`) {
			t.Errorf("the run record carries an item 0 in its refused_items")
		}
		if run.RefusedCount != items-1 {
			t.Errorf("the run record's refused_count is %d, want %d", run.RefusedCount, items-1)
		}
		// The elision entry names no item. Rendering it as "item 0" would send a
		// reader looking for one that does not exist.
		for _, r := range res.RefusedItems {
			if r.Rule == "refusals-elided" {
				if r.Ordinal != 0 {
					t.Errorf("the elision entry claims ordinal %d", r.Ordinal)
				}
				if strings.Contains(r.Detail, "refused_count") {
					t.Errorf("the elision entry refers the reader to a field the text surface does "+
						"not print: %q", r.Detail)
				}
			}
		}
	})

	t.Run("a run in which every item is refused", func(t *testing.T) {
		f := newIngestFixture(t, "detection")
		doc := f.payload(200)
		for _, raw := range doc["items"].([]any) {
			delete(raw.(map[string]any), PatternField)
		}
		if _, err := f.ingest(doc); err == nil {
			t.Fatal("a payload whose every item was illegal was accepted")
		}

		info, err := os.Stat(filepath.Join(f.root, filepath.FromSlash(
			ReadingsRecordDir+"/"+f.runID+"/"+RefusalFileName)))
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() > 64<<10 {
			t.Errorf("the refusal record is %d bytes; a record whose purpose is to be read has to stay "+
				"readable", info.Size())
		}
		rec := f.readRefusalRecord(f.runID)
		if !strings.Contains(rec.Reason, "more item(s) refused") {
			t.Errorf("the bounded reason does not say how many refusals it left out: %q", rec.Reason)
		}
		// The elision entry is not an item, and the durable record renders it
		// under the same rule as the terminal: no ordinal, so no "item 0". The
		// terminal had the branch and the record writer did not
		// (iss-2608311518250688).
		if strings.Contains(rec.Reason, "item 0") {
			t.Errorf("the refusal record names an item 0, which does not exist: %q", rec.Reason)
		}
		if !strings.Contains(rec.Reason, "(refusals-elided) and 180 more item(s) refused") {
			t.Errorf("the refusal record does not render the elision entry as an elision: %q", rec.Reason)
		}
	})
}

// TestAClosedBodyVocabularyIsEnforced: claim_type's three tokens are instructed
// by the entailment definition and tabled in the spec, and were enforced
// nowhere — not by this verb, not by the record writer, not by the schema — so
// an arbitrary value landed in the durable record.
func TestAClosedBodyVocabularyIsEnforced(t *testing.T) {
	if _, ok := ClosedVocabularies["claim_type"]; !ok {
		t.Fatal("claim_type declares no closed vocabulary")
	}
	f := newIngestFixture(t, "entailment")
	doc := f.payload(2)
	doc["items"].([]any)[1].(map[string]any)["claim_type"] = "assertion"

	r := f.refusedItem(doc, 2, 2)
	if r.Rule != "closed-vocabulary" {
		t.Errorf("the refusal cites rule %q", r.Rule)
	}
	if !strings.Contains(r.Detail, "assertion") || !strings.Contains(r.Detail, "criterion") {
		t.Errorf("the refusal names neither the value nor the set: %q", r.Detail)
	}

	// Every token in the set is accepted, so the check cannot be a blanket refusal.
	for _, token := range ClosedVocabularies["claim_type"] {
		g := newIngestFixture(t, "entailment")
		ok := g.payload(1)
		ok["items"].([]any)[0].(map[string]any)["claim_type"] = token
		if res := g.mustIngest(ok); len(res.Records) != 1 {
			t.Errorf("claim_type %q landed %d record(s)", token, len(res.Records))
		}
	}
}

// TestTheCommittedRecordCarriesNoHiddenRunes is the trust boundary at the RECORD
// end. The record writer's own scalar guard refuses runes below 0x20 and nothing
// above, so a bidi override, a C1 control or a zero-width rune in an item body
// would land verbatim in a committed markdown file a reviewer reads in a
// terminal — Trojan Source, in the ledger.
//
// The encoding is lossless, so the bytes are still recoverable; what is gone is
// their ability to reorder or hide what the reader sees.
func TestTheCommittedRecordCarriesNoHiddenRunes(t *testing.T) {
	f := newIngestFixture(t, "detection")
	doc := f.payload(1)
	item := doc["items"].([]any)[0].(map[string]any)
	item["tension"] = "the record says " + hostileText + " and the tree says otherwise"
	item[PatternField] = "a pattern " + hostileText

	res := f.mustIngest(doc)
	if len(res.Records) != 1 {
		t.Fatalf("landed %d record(s)", len(res.Records))
	}
	raw, err := os.ReadFile(filepath.Join(f.root, filepath.FromSlash(res.Records[0].Path)))
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range string(raw) {
		if hostileRune(r) {
			t.Fatalf("the committed record carries U+%04X at byte %d", r, i)
		}
	}
	// And the text is still there, encoded rather than dropped.
	if !strings.Contains(string(raw), "the record says") {
		t.Error("the record lost the item's text")
	}
}

// TestTheRefusalRecordCarriesTheWholeReason: the record whose stated purpose is
// to carry the named reason has to carry it. A second sanitise-and-cap over the
// composed cause truncated the repository's OWN prose — every payload-derived
// substring inside it is already cleaned where it is interpolated — and cut a
// 338-rune refusal to 123 runes, mid-word.
func TestTheRefusalRecordCarriesTheWholeReason(t *testing.T) {
	f := newIngestFixture(t, "detection")
	doc := f.payload(1)
	doc["regime"] = RegimeEvaluative

	_, err := f.ingest(doc)
	if err == nil {
		t.Fatal("a regime mismatch was accepted")
	}
	rec := f.readRefusalRecord(f.runID)

	if len([]rune(rec.Reason)) <= maxEchoedBytes {
		t.Errorf("the recorded reason is %d runes, at or under the per-VALUE cap (%d): the whole "+
			"sentence is what the record exists to carry", len([]rune(rec.Reason)), maxEchoedBytes)
	}
	// The terminal message is the recorded reason plus the pointer at the record.
	if !strings.Contains(err.Error(), rec.Reason) {
		t.Errorf("the recorded reason is not what the operator was told:\n record: %q\n told:   %q",
			rec.Reason, err.Error())
	}
	for _, want := range []string{RegimeEvaluative, RegimeRegistrative, "refuses the run"} {
		if !strings.Contains(rec.Reason, want) {
			t.Errorf("the recorded reason does not carry %q: %q", want, rec.Reason)
		}
	}
	assertSafeEcho(t, rec.Reason, false)
}
