package reading

// ingest_comparative_test.go holds the ingest half of the comparative channel:
// the two checks against the manifest (ac-7, ac-8), the clean-run idiom the
// framework's section 13 fixes, and the exported durable-tier writer
// (spc-2609020626039834; iss-2609021153269181).

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ac-7. TestIngestRefusesACandidateOutsideTheRun: a comparative item naming a
// candidate the recorded run does not hold is refused BY NAME. The check is
// against the manifest, not against the payload's own account of itself: a
// reading's claim about what it was given establishes nothing, and the manifest
// is the artefact that does.
func TestIngestRefusesACandidateOutsideTheRun(t *testing.T) {
	f := newIngestFixture(t, PositionComparative)
	doc := f.payload(3)
	// The SECOND of three, so the ordinal in the message is not the only one the
	// verb could have printed, and the other two land.
	doc["items"].([]any)[1].(map[string]any)["candidate_id"] = "rdi-999"

	res := f.mustIngest(doc)
	if len(res.Records) != 2 {
		t.Fatalf("an item-level refusal landed %d of the other 2 items", len(res.Records))
	}
	r := f.refusalOf(res, 2)
	if r.Rule != ruleUnknownCandidate {
		t.Errorf("the refusal cites rule %q, want %q", r.Rule, ruleUnknownCandidate)
	}
	if r.Field != "candidate_id" {
		t.Errorf("the refusal names field %q, want candidate_id", r.Field)
	}
	for _, want := range []string{"rdi-999", fixtureIngestCandidateRun} {
		if !strings.Contains(r.Detail, want) {
			t.Errorf("the refusal does not name %q: %s", want, r.Detail)
		}
	}
}

// ac-8. TestIngestRefusesAnUndeclaredCriterion: a criterion the discipline does
// not declare is refused, naming the item and the criterion. The criteria are a
// declared, recorded discipline and a reading never authors one (itd-191).
func TestIngestRefusesAnUndeclaredCriterion(t *testing.T) {
	f := newIngestFixture(t, PositionComparative)
	doc := f.payload(3)
	doc["items"].([]any)[2].(map[string]any)["criterion"] = "Vibes"

	res := f.mustIngest(doc)
	if len(res.Records) != 2 {
		t.Fatalf("an item-level refusal landed %d of the other 2 items", len(res.Records))
	}
	r := f.refusalOf(res, 3)
	if r.Rule != ruleUndeclaredCriterion {
		t.Errorf("the refusal cites rule %q, want %q", r.Rule, ruleUndeclaredCriterion)
	}
	if r.Field != "criterion" {
		t.Errorf("the refusal names field %q, want criterion", r.Field)
	}
	for _, want := range []string{"Vibes", CriteriaDiscipline, fixtureIngestCriteria[0]} {
		if !strings.Contains(r.Detail, want) {
			t.Errorf("the refusal does not name %q: %s", want, r.Detail)
		}
	}
}

// TestEveryDeclaredCriterionIsAccepted is the two checks' non-vacuity: a rule
// refusing everything would pass both cases above. Every criterion the manifest
// records, against every candidate it records, lands.
func TestEveryDeclaredCriterionIsAccepted(t *testing.T) {
	f := newIngestFixture(t, PositionComparative)
	doc := f.payload(len(fixtureIngestCriteria))
	for i, raw := range doc["items"].([]any) {
		item := raw.(map[string]any)
		item["criterion"] = fixtureIngestCriteria[i]
		item["candidate_id"] = fixtureIngestCandidates[i]
	}
	res := f.mustIngest(doc)
	if len(res.Records) != len(fixtureIngestCriteria) {
		t.Fatalf("%d of %d declared criteria landed", len(res.Records), len(fixtureIngestCriteria))
	}
	if len(res.RefusedItems) != 0 {
		t.Errorf("a declared criterion was refused: %v", res.RefusedItems)
	}
}

// TestTheTwoComparativeChecksBindAtNoOtherPosition: the manifest carries a
// candidate set and a slate at the comparative position and at no other, so the
// checks must not fire elsewhere — an item at the detection position naming
// nothing of the kind lands.
func TestTheTwoComparativeChecksBindAtNoOtherPosition(t *testing.T) {
	for _, p := range []Position{PositionWidening, PositionEntailment, PositionDetection} {
		f := newIngestFixture(t, p)
		res := f.mustIngest(f.payload(2))
		if len(res.Records) != 2 {
			t.Errorf("at %s a legal payload landed %d of 2 items", p, len(res.Records))
		}
	}
}

// TestIngestCommitsAnEmptyRunAtEveryPosition is the general rule
// (iss-2609021153269181; the framework's section 13; the corrections ruling (4)
// of 2026-09-02): a run that returns no items is committed AT EVERY POSITION as
// a run with an empty item set, and never refused.
func TestIngestCommitsAnEmptyRunAtEveryPosition(t *testing.T) {
	for _, p := range Positions() {
		t.Run(string(p), func(t *testing.T) {
			f := newIngestFixture(t, p)
			res, err := f.ingest(f.payload(0))
			if err != nil {
				t.Fatalf("an output with no items was refused at the %s position: %v", p, err)
			}
			if len(res.Records) != 0 {
				t.Errorf("an empty run wrote %d record(s)", len(res.Records))
			}
			if res.RunRecordPath == "" {
				t.Fatal("an empty run left no commit marker; a run without one never happened")
			}
			run := f.readRunRecord(f.runID)
			if run.Records == nil {
				t.Error("the run record writes `null` for its records; an empty item set is `[]`, " +
					"so a reader can tell it from a record written before the field existed")
			}
			if len(run.Records) != 0 {
				t.Errorf("the run record names %d record(s)", len(run.Records))
			}
			if len(run.RefusedItems) != 0 {
				t.Errorf("an empty run refused %d item(s); nothing was refused, because nothing "+
					"was returned", len(run.RefusedItems))
			}
		})
	}
}

// TestIngestStillRefusesAMalformedEmptyPayload: the refusal survives for a
// payload that is empty AND malformed. Refusal is reserved for a malformed
// payload, and withdrawing the empty-item refusal must not withdraw the rest.
func TestIngestStillRefusesAMalformedEmptyPayload(t *testing.T) {
	f := newIngestFixture(t, PositionDetection)
	doc := f.payload(0)
	doc["instrument"] = map[string]any{
		"model": "", "definition_sha256": "", "assembler_version": "",
	}
	if _, err := f.ingest(doc); err == nil {
		t.Fatal("an empty payload with no instrument was committed; refusal is reserved for a " +
			"malformed payload, and an empty item list is not malformed")
	}

	// And a run whose every item was refused is still a list-level refusal: that
	// is a different fact from a run that returned nothing, and recording the
	// first as the second would lose it.
	g := newIngestFixture(t, PositionDetection)
	bad := g.payload(1)
	bad["items"].([]any)[0].(map[string]any)["fix"] = "rewrite the constraint"
	if _, err := g.ingest(bad); err == nil {
		t.Fatal("a run whose every item was refused was committed as a run with an empty item set")
	}
}

// ac-5. TestIngestCommitsAnEmptyComparativeRun: the not-exercised outcome is one
// instance of the general rule. The committed run names the widening run it
// characterised, how many candidates that run held, and that it was not
// exercised — so the outcome of a widening run is always the same thing.
func TestIngestCommitsAnEmptyComparativeRun(t *testing.T) {
	f := newIngestFixture(t, PositionComparative)
	// The parked manifest is the staged empty run's: the derived run held one
	// candidate, the bundle carried none, and `exercised` is false.
	notExercised := false
	f.parkComparative(f.runID, fixtureIngestCandidateRun, 1, &notExercised, nil)

	doc := f.payload(0)
	doc["manifest_sha256"] = f.manifestHashOf(f.runID)
	res, err := f.ingest(doc)
	if err != nil {
		t.Fatalf("the staged empty comparative run was refused: %v", err)
	}
	run := f.readRunRecord(f.runID)
	if run.CandidateRun != fixtureIngestCandidateRun {
		t.Errorf("the committed run names the candidate run %q, want %q; the outcome of a widening "+
			"run is a committed comparative run NAMING it", run.CandidateRun, fixtureIngestCandidateRun)
	}
	if run.Candidates != 1 {
		t.Errorf("the committed run records %d candidates, want 1 — the derived run's item count, "+
			"never zero for a run that holds an item", run.Candidates)
	}
	if run.Exercised == nil || *run.Exercised {
		t.Errorf("the committed run does not state exercised:false: %v", run.Exercised)
	}
	if len(res.Records) != 0 {
		t.Errorf("the not-exercised run wrote %d reading record(s)", len(res.Records))
	}
}

// TestAnExercisedComparativeRunCarriesTheJoinForward is the other half: a run
// that WAS exercised carries the same three fields, so a later verb reads one
// shape whichever happened.
func TestAnExercisedComparativeRunCarriesTheJoinForward(t *testing.T) {
	f := newIngestFixture(t, PositionComparative)
	f.mustIngest(f.payload(2))
	run := f.readRunRecord(f.runID)
	if run.CandidateRun != fixtureIngestCandidateRun {
		t.Errorf("the committed run names %q", run.CandidateRun)
	}
	if run.Exercised == nil || !*run.Exercised {
		t.Errorf("the committed run does not state exercised:true: %v", run.Exercised)
	}
}

// TestNoOtherPositionsRunRecordCarriesTheJoin: `candidate_run`, `candidates` and
// `exercised` are the comparative position's alone, so a widening run record
// asserts nothing about a join it has no opinion on.
func TestNoOtherPositionsRunRecordCarriesTheJoin(t *testing.T) {
	for _, p := range []Position{PositionWidening, PositionEntailment, PositionDetection} {
		f := newIngestFixture(t, p)
		f.mustIngest(f.payload(1))
		raw, err := os.ReadFile(filepath.Join(f.root, filepath.FromSlash(
			ReadingsRecordDir+"/"+f.runID+"/"+RunFileName)))
		if err != nil {
			t.Fatal(err)
		}
		var doc map[string]json.RawMessage
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"candidate_run", "candidates", "exercised"} {
			if _, present := doc[key]; present {
				t.Errorf("the %s run record carries %q; the join is the comparative position's alone",
					p, key)
			}
		}
	}
}

// TestWriteRunArtefactIsWriteOnce: the durable tier is write-once at emission,
// so a second write beside a committed run is refused rather than amending it.
func TestWriteRunArtefactIsWriteOnce(t *testing.T) {
	f := newIngestFixture(t, PositionDetection)
	f.mustIngest(f.payload(1))

	rel, err := WriteRunArtefact(f.root, f.runID, "scribe-manifest.json", map[string]any{"a": 1})
	if err != nil {
		t.Fatalf("writing beside a committed run: %v", err)
	}
	if want := ReadingsRecordDir + "/" + f.runID + "/scribe-manifest.json"; rel != want {
		t.Errorf("the artefact landed at %q, want %q", rel, want)
	}
	if _, err := os.Stat(filepath.Join(f.root, filepath.FromSlash(rel))); err != nil {
		t.Fatalf("the artefact is not on disk: %v", err)
	}
	_, err = WriteRunArtefact(f.root, f.runID, "scribe-manifest.json", map[string]any{"a": 2})
	if err == nil {
		t.Fatal("a second write to one artefact was accepted; the durable tier is write-once at " +
			"emission, and an amended artefact is one nothing can date")
	}
	if !strings.Contains(err.Error(), "write-once") {
		t.Errorf("the refusal does not state the rule: %v", err)
	}
}

// TestWriteRunArtefactRefusesAnUncommittedRun: a run without its commit marker
// never happened, and its records are the next sweep's to roll back — so nothing
// is filed beside it.
func TestWriteRunArtefactRefusesAnUncommittedRun(t *testing.T) {
	f := newIngestFixture(t, PositionDetection)
	_, err := WriteRunArtefact(f.root, "rdg-2608310000000099", "scribe-manifest.json", map[string]any{})
	if err == nil {
		t.Fatal("an artefact was filed beside a run that never committed")
	}
	for _, want := range []string{"commit marker", "never happened"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not state %q: %v", want, err)
		}
	}
}

// TestWriteRunArtefactRefusesTheRunsOwnFiles: a run's evidence is the ingest
// verb's to write, and a name that is one of its own three files — or a name
// that is a path at all — is refused.
func TestWriteRunArtefactRefusesTheRunsOwnFiles(t *testing.T) {
	f := newIngestFixture(t, PositionDetection)
	f.mustIngest(f.payload(1))

	for _, name := range []string{RunFileName, ManifestFileName, RefusalFileName} {
		if _, err := WriteRunArtefact(f.root, f.runID, name, map[string]any{}); err == nil {
			t.Errorf("%s was overwritten beside its own run", name)
		}
	}
	for _, name := range []string{
		"../escape.json", "sub/dir.json", "no-extension", "Upper.json", ".hidden.json", "a..b.json",
	} {
		if _, err := WriteRunArtefact(f.root, f.runID, name, map[string]any{}); err == nil {
			t.Errorf("the name %q was accepted; a name is a plain lowercase basename ending in "+
				".json, never a path", name)
		}
	}
}

// A refused item's payload text reaches the DURABLE tier — the run record's
// `refused_items`, and the refusal record's `reason` when every item was refused
// — and it was echoed there through the neutraliser alone, with no privacy
// redaction. The same ingest redacts an ACCEPTED item's body on the way into the
// ledger, so one payload string was treated two ways by one verb depending only
// on whether the verb liked it (AGENTS.md's privacy rule; framework 7.1; brief
// invariant 16; iss-2609022002241168).
//
// The criterion is the field the review named, and it is the worst case: it is
// free text by construction, and it is quoted back verbatim precisely BECAUSE
// the discipline does not declare it.
func TestARefusedItemsTextIsRedactedInTheDurableRecord(t *testing.T) {
	const leak = "/Users/zzotherperson/checkouts/abcd/notes.md"
	f := newIngestFixture(t, PositionComparative)
	doc := f.payload(3)
	doc["items"].([]any)[2].(map[string]any)["criterion"] = "read from " + leak

	res := f.mustIngest(doc)
	r := f.refusalOf(res, 3)
	if r.Rule != ruleUndeclaredCriterion {
		t.Fatalf("the refusal cites rule %q, want %q", r.Rule, ruleUndeclaredCriterion)
	}
	if strings.Contains(r.Detail, "zzotherperson") {
		t.Errorf("the refusal detail carries a third party's home path: %s", r.Detail)
	}
	if !strings.Contains(r.Detail, "[redacted-path]") {
		t.Errorf("the refusal detail carries no redaction mask, so the text was dropped rather "+
			"than redacted: %s", r.Detail)
	}

	// The durable record is the point: a result a surface prints is transient
	// and run.json is committed.
	raw, err := os.ReadFile(filepath.Join(f.root, filepath.FromSlash(
		ReadingsRecordDir+"/"+f.runID+"/"+RunFileName)))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "zzotherperson") {
		t.Errorf("the committed run record carries a third party's home path:\n%s", raw)
	}
}

// The refusal RECORD is the other durable surface: when every item is refused
// the run is refused whole, and the per-item details are carried into
// refusal.json's reason. Redacting one and not the other would leave the leak on
// the path a failed run takes.
func TestARefusedRunsReasonIsRedacted(t *testing.T) {
	const leak = "/Users/zzotherperson/checkouts/abcd/notes.md"
	f := newIngestFixture(t, PositionComparative)
	doc := f.payload(1)
	doc["items"].([]any)[0].(map[string]any)["criterion"] = "read from " + leak

	if _, err := f.ingest(doc); err == nil {
		t.Fatal("a run whose every item was refused was committed")
	}
	rec := f.readRefusalRecord(f.runID)
	if strings.Contains(rec.Reason, "zzotherperson") {
		t.Errorf("the committed refusal record carries a third party's home path: %s", rec.Reason)
	}
	if !strings.Contains(rec.Reason, "[redacted-path]") {
		t.Errorf("the refusal reason carries no redaction mask: %s", rec.Reason)
	}
}

// The payload-supplied INSTRUMENT identity lands in both durable records
// verbatim, and it is the same class: sanitizeInstrument neutralised it and
// nothing redacted it. A model name is agent-supplied text, not a validated
// shape.
func TestTheRecordedInstrumentIsRedacted(t *testing.T) {
	const leak = "/Users/zzotherperson/models/local.gguf"
	f := newIngestFixture(t, PositionComparative)
	doc := f.payload(1)
	doc["instrument"].(map[string]any)["model"] = "local model at " + leak

	res := f.mustIngest(doc)
	if len(res.Records) != 1 {
		t.Fatalf("the run landed %d record(s), want 1", len(res.Records))
	}
	run := f.readRunRecord(f.runID)
	if strings.Contains(run.Instrument.Model, "zzotherperson") {
		t.Errorf("the committed run record names a third party's home path as the model: %s",
			run.Instrument.Model)
	}
	if !strings.Contains(run.Instrument.Model, "[redacted-path]") {
		t.Errorf("the recorded model carries no redaction mask: %s", run.Instrument.Model)
	}
}
