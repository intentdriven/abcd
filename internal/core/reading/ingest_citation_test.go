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
