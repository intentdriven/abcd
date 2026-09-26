package reading

import (
	"strings"
	"testing"
)

// TestAnUnresolvedCitationRefusesTheRunAndWritesNothing: an item is the host's
// words, bound for a reading record that record-lint's prose_citation_resolves
// reads, so an item citing a record id that names no record refuses the ingest
// naming the item and the id (iss-2609261835118276) — before the sweep, before
// the stage, and with no refusal record, so the run stays parked and the same
// run, re-worded, ingests.
func TestAnUnresolvedCitationRefusesTheRunAndWritesNothing(t *testing.T) {
	const dangling = "iss-2609999999999999"
	const orphan, orphanItem = "rdg-2608310000000031", "rdi-2608310000000032"
	f := newIngestFixture(t, "detection")
	f.write(".abcd/record-lint.json", []byte(`{
  "roots": [".abcd/work"],
  "rules": {
    "prose_citation_resolves": {
      "enabled": true,
      "severity": "blocker",
      "record_stores": {"iss": ".abcd/work/issues", "rdi": ".abcd/work/issues/readings"}
    }
  }
}
`))
	rel, body := f.plantOrphan(orphan, orphanItem)

	doc := f.payload(1)
	doc["items"].([]any)[0].(map[string]any)[PatternField] = "the pattern " + dangling + " names"
	res, err := f.ingest(doc)
	if err == nil || !strings.Contains(err.Error(), dangling) || !strings.Contains(err.Error(), "reading item 1's pattern") {
		t.Fatalf("err = %v, want a refusal naming item 1's pattern and %s", err, dangling)
	}
	if res.RefusalPath != "" {
		t.Errorf("the refusal was recorded at %s; it must write nothing", res.RefusalPath)
	}
	f.nothingDurable(f.runID)
	if string(f.bytesAt(rel)) != string(body) || !f.exists(IngestStageDir+"/"+orphan) {
		t.Error("a citation refusal swept another run's orphan")
	}

	// The same run, re-worded, ingests: the refusal gave it no outcome.
	f.mustIngest(f.payload(1))
}

// The citation refusal is returned unrecorded, but it owes the rollback every
// refusal past the identity point owes (refuse's rollbackThisRun): an earlier
// attempt at this run id that died between its ledger write and its commit
// marker left records the run never committed, and a refused run leaves no
// reading records. The rollback writes no outcome, so the same run re-worded
// still ingests.
func TestACitationRefusalRollsBackTheRunsOwnCrashedAttempt(t *testing.T) {
	const dangling = "iss-2609999999999999"
	f := newIngestFixture(t, "detection")
	f.write(".abcd/record-lint.json", []byte(`{
  "roots": [".abcd/work"],
  "rules": {
    "prose_citation_resolves": {
      "enabled": true,
      "severity": "blocker",
      "record_stores": {"iss": ".abcd/work/issues", "rdi": ".abcd/work/issues/readings"}
    }
  }
}
`))
	withFault(t, faultAfterLedger)
	if _, err := f.ingest(f.payload(2)); err == nil {
		t.Fatal("the injected fault did not stop the first attempt")
	}
	ingestFault = nil
	if got := f.ledgerRecords(f.runID); len(got) != 2 {
		t.Fatalf("the crashed attempt left %v in the ledger, want its 2 records", got)
	}

	doc := f.payload(1)
	doc["items"].([]any)[0].(map[string]any)[PatternField] = "the pattern " + dangling + " names"
	res, err := f.ingest(doc)
	if err == nil || !strings.Contains(err.Error(), dangling) {
		t.Fatalf("err = %v, want a refusal naming %s", err, dangling)
	}
	if res.RefusalPath != "" {
		t.Errorf("the refusal was recorded at %s; it must stay unrecorded", res.RefusalPath)
	}
	f.nothingDurableInTheLedger(f.runID)
	if len(res.RolledBack) != 2 || f.exists(IngestStageDir+"/"+f.runID) {
		t.Errorf("the refusal rolled back %v (stage standing: %v); it removes the earlier attempt's 2 "+
			"records and its stage and says so", res.RolledBack, f.exists(IngestStageDir+"/"+f.runID))
	}

	f.mustIngest(f.payload(1))
}
