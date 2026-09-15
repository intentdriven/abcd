package reading

// ingest_sweep_fence_test.go holds the orphan sweep behind the whole payload
// validating (iss-2608311517509690).
//
// The sweep is a DELETE in the committed tier, and it used to be step one of
// the verb: with an orphaned stage present and its records in the ledger, an
// ingest whose payload failed at the type check removed those records and
// printed only the type error. The durable tier is never mutated by a run that
// is refused, so the sweep now runs only once the payload has validated — and a
// refused run says which orphans it saw and left in place, so nothing about the
// tier's state is silent on either path.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// plantOrphan writes an orphaned stage and one reading record its run had
// landed, returning the record's repo-relative path and the exact bytes written.
func (f *ingestFixture) plantOrphan(runID, itemID string) (rel string, body []byte) {
	f.t.Helper()
	f.write(IngestStageDir+"/"+runID+"/"+stageFileName,
		[]byte(`{"_type":"`+StageType+`","run_id":"`+runID+`","records":["`+itemID+`"]}`))
	rel = ".abcd/work/issues/readings/" + runID + "/" + itemID + ".md"
	body = []byte("---\nid: " + itemID + "\nrun: " + runID + "\n---\n\nthe committed body\n")
	f.write(rel, body)
	return rel, body
}

// bytesAt reads a repo-relative file back.
func (f *ingestFixture) bytesAt(rel string) []byte {
	f.t.Helper()
	raw, err := os.ReadFile(filepath.Join(f.root, filepath.FromSlash(rel)))
	if err != nil {
		f.t.Fatalf("read %s: %v", rel, err)
	}
	return raw
}

// TestARefusedRunLeavesTheOrphanInPlace is the invariant at the two kinds of
// refusal: one before the run's identity is proven (the type check, which
// writes no refusal record) and one after it (the regime check, which does).
// In both, the committed record survives byte for byte, the named error is the
// one reported, and the result says the orphan was seen and left — so the
// surface has the fields to render on the refusal path.
func TestARefusedRunLeavesTheOrphanInPlace(t *testing.T) {
	const orphan, item = "rdg-2608310000000031", "rdi-2608310000000032"

	cases := []struct {
		name   string
		mutate func(doc map[string]any)
		wants  string
		// recorded says whether the refusal reaches a refusal record, which
		// decides whether the adjacent legal payload can reuse the run id.
		recorded bool
	}{
		{"at the type check", func(doc map[string]any) { doc["_type"] = "not-the-output-type" }, "_type", false},
		{"at the regime check", func(doc map[string]any) { doc["regime"] = RegimeEvaluative }, RegimeEvaluative, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newIngestFixture(t, "detection")
			rel, body := f.plantOrphan(orphan, item)

			doc := f.payload(1)
			tc.mutate(doc)
			res, err := f.ingest(doc)
			if err == nil {
				t.Fatal("the illegal payload was accepted")
			}
			if !strings.Contains(err.Error(), tc.wants) {
				t.Errorf("the refusal does not name the rule: %v", err)
			}

			// (a) the committed record is untouched, byte for byte.
			if !f.exists(rel) {
				t.Fatalf("a refused run deleted %s from the committed ledger", rel)
			}
			if got := f.bytesAt(rel); string(got) != string(body) {
				t.Errorf("the committed record changed under a refused run:\n%s", got)
			}
			if !f.exists(IngestStageDir + "/" + orphan) {
				t.Error("a refused run cleared the orphaned stage")
			}

			// (c) and the result says so: the orphan was seen and left, and
			// nothing was cleared or rolled back for the surface to hide.
			if len(res.PendingStages) != 1 || res.PendingStages[0] != orphan {
				t.Errorf("a refused run reports pending stages %v, want exactly [%s]", res.PendingStages, orphan)
			}
			if len(res.ClearedStages) != 0 || len(res.RolledBack) != 0 {
				t.Errorf("a refused run reports cleared %v and rolled back %v; it must mutate nothing",
					res.ClearedStages, res.RolledBack)
			}
			if !res.HasDisclosure() {
				t.Error("the result carries a pending stage and says it has nothing to disclose")
			}
			if tc.recorded && res.RefusalPath == "" {
				t.Error("a refusal after the run's identity was proven recorded nothing")
			}

			// The adjacent LEGAL payload sweeps what the refused one left: the
			// orphan is cleared, its record is rolled back and named, and
			// nothing is pending any more.
			next := f.payload(1)
			if tc.recorded {
				next = f.nextRun(next)
			}
			swept := f.mustIngest(next)
			if len(swept.ClearedStages) != 1 || swept.ClearedStages[0] != orphan {
				t.Errorf("the validated run cleared %v, want exactly [%s]", swept.ClearedStages, orphan)
			}
			if len(swept.RolledBack) != 1 || swept.RolledBack[0] != item {
				t.Errorf("the validated run rolled back %v, want exactly [%s]", swept.RolledBack, item)
			}
			if len(swept.PendingStages) != 0 {
				t.Errorf("the validated run left %v pending after sweeping", swept.PendingStages)
			}
			if f.exists(rel) {
				t.Error("the orphaned record survived the validated run's sweep")
			}
		})
	}
}

// TestTheSweepRunsOnlyAfterTheWholePayloadValidates is the ordering stated as
// a property over the fault seam: when the sweep's unlink fires, the payload
// has already been validated in full — an item-level refusal has been decided —
// so a delete in the committed tier is never reached on a payload that will be
// refused.
func TestTheSweepRunsOnlyAfterTheWholePayloadValidates(t *testing.T) {
	f := newIngestFixture(t, "detection")
	rel, _ := f.plantOrphan("rdg-2608310000000033", "rdi-2608310000000034")

	// A payload with one illegal item among three: item-level, so the run
	// lands, and the sweep must have waited for that decision.
	doc := f.payload(3)
	doc["items"].([]any)[1].(map[string]any)[PatternField] = ""

	unlinked := false
	prior := ingestFault
	ingestFault = func(at string) error {
		if at == faultDuringRollback {
			unlinked = true
		}
		return nil
	}
	t.Cleanup(func() { ingestFault = prior })

	res := f.mustIngest(doc)
	if !unlinked {
		t.Fatal("the sweep never unlinked the orphan on a validated run")
	}
	if len(res.RefusedItems) != 1 || len(res.Records) != 2 {
		t.Errorf("the run refused %d and landed %d; the item decision was not made", len(res.RefusedItems), len(res.Records))
	}
	if f.exists(rel) {
		t.Error("the orphaned record survived a validated run")
	}
}

// TestTheDisclosurePredicateNamesEveryMutationField pins HasDisclosure to the
// fields it exists for. The surface renders on the error path only when this
// says so, so a field it does not cover is a mutation the surface can hide.
func TestTheDisclosurePredicateNamesEveryMutationField(t *testing.T) {
	if (IngestResult{}).HasDisclosure() {
		t.Error("an empty result claims a disclosure")
	}
	for name, res := range map[string]IngestResult{
		"a refusal record":   {RefusalPath: "x"},
		"a cleared stage":    {ClearedStages: []string{"rdg-2608310000000035"}},
		"a rolled-back id":   {RolledBack: []string{"rdi-2608310000000036"}},
		"a pending stage":    {PendingStages: []string{"rdg-2608310000000037"}},
		"a degraded outcome": {Degraded: "the stage could not be cleared"},
	} {
		if !res.HasDisclosure() {
			t.Errorf("%s is not disclosed", name)
		}
	}
}
