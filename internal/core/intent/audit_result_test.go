package intent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// audit_result_test.go — what the emit and the ingest results say they did.
// A result names the receipt's state and the act separately, because the two
// differ, and a reader who takes one for the other acts on something that did
// not happen.

// TestIngestResultJSONCarriesARollupOnlyForARecordedVerdict is
// iss-2609190337545165: one result struct serves every outcome, and a
// quarantined verdict's JSON carried zero-valued criteria counters beside a
// conditions count read from the intent — a rollup that reads as a result. The
// JSON now says what the ingest recorded (`recorded`), carries the rollup only
// when that is a verdict, and states a quarantine's untested split under its
// own name.
func TestIngestResultJSONCarriesARollupOnlyForARecordedVerdict(t *testing.T) {
	rollup := []string{"criteria", "met", "met_with_concerns", "not_met", "inconclusive",
		"conditions", "survived", "narrowed", "falsified", "untested"}
	decode := func(t *testing.T, res IngestVerdictResult) map[string]any {
		t.Helper()
		b, err := json.Marshal(res)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatal(err)
		}
		return m
	}

	root := t.TempDir()
	rcp := shipWithConditions(t, root,
		stampedCondition(condOne, "holds on POSIX"),
		stampedCondition(condTwo, "holds below 10k records"),
	)
	bad := verdictWithConditions(t, rcp, dispositionOf(condOne, "MET"), dispositionOf(condTwo, "survived"))
	dl, err := IngestVerdict(root, writeVerdict(t, root, bad))
	if err != nil || dl.Status != "dead_letter" {
		t.Fatalf("setup: want a dead_letter, got %+v, %v", dl, err)
	}
	m := decode(t, dl)
	for _, k := range rollup {
		if _, ok := m[k]; ok {
			t.Errorf("the dead_letter JSON carries the rollup member %q:\n%v", k, m)
		}
	}
	if m["recorded"] != "quarantine" {
		t.Errorf("the dead_letter JSON says recorded = %v, want quarantine", m["recorded"])
	}
	if m["conditions_untested"] != float64(2) {
		t.Errorf("the dead_letter JSON must state the 2 conditions its quarantine recorded untested: %v", m)
	}
	if m["dead_letter_path"] == nil || m["reason"] == nil {
		t.Errorf("the dead_letter JSON must still say where the payload went and why: %v", m)
	}

	root = t.TempDir()
	rcp = shipOne(t, root)
	vp := writeVerdict(t, root, validVerdict(rcp))
	ok, err := IngestVerdict(root, vp)
	if err != nil || ok.Status != "ingested" {
		t.Fatalf("setup: want ingested, got %+v, %v", ok, err)
	}
	m = decode(t, ok)
	for _, k := range rollup {
		if _, ok := m[k]; !ok {
			t.Errorf("the ingested JSON lacks the rollup member %q (a zero count is a count):\n%v", k, m)
		}
	}
	if m["recorded"] != "verdict" {
		t.Errorf("the ingested JSON says recorded = %v, want verdict", m["recorded"])
	}
	if _, ok := m["conditions_untested"]; ok {
		t.Errorf("the ingested JSON carries the quarantine's member: %v", m)
	}

	noop, err := IngestVerdict(root, vp)
	if err != nil || noop.Status != "noop" {
		t.Fatalf("setup: want noop, got %+v, %v", noop, err)
	}
	m = decode(t, noop)
	for _, k := range rollup {
		if _, ok := m[k]; ok {
			t.Errorf("the noop JSON carries the rollup member %q:\n%v", k, m)
		}
	}
	if m["recorded"] != "nothing" {
		t.Errorf("the noop JSON says recorded = %v, want nothing", m["recorded"])
	}
}

// TestReEmitOfAnOwedReceiptSaysItRewroteTheRequest is iss-2609190337598356: a
// re-emit on an OWED receipt rewrites the request and reported only
// already_owed, which a caller read as "nothing happened". The result now says
// it wrote the request, and a terminal receipt — whose emit writes nothing —
// names no request path at all rather than one it did not write.
func TestReEmitOfAnOwedReceiptSaysItRewroteTheRequest(t *testing.T) {
	root := t.TempDir()
	rcp := shipOne(t, root)
	reqPath := filepath.Join(root, reviewsDir, rcp+".request.md")
	if err := os.Remove(reqPath); err != nil {
		t.Fatal(err)
	}

	res, err := ReEmitAudit(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "already_owed" || !auditEmitJSON(t, res)["request_written"].(bool) || res.RequestPath == "" {
		t.Fatalf("an OWED re-emit = %+v, want already_owed with the request it wrote", res)
	}
	if _, err := os.Stat(reqPath); err != nil {
		t.Fatalf("the re-emit reported a request it did not write: %v", err)
	}

	if _, err := IngestVerdict(root, writeVerdict(t, root, validVerdict(rcp))); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(reqPath); err != nil {
		t.Fatal(err)
	}
	done, err := ReEmitAudit(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	if m := auditEmitJSON(t, done); done.Status != "already_ingested" || m["request_written"] != false || m["request_path"] != nil {
		t.Fatalf("a terminal re-emit = %+v, want already_ingested naming no request", done)
	}
	if _, err := os.Stat(reqPath); err == nil {
		t.Fatal("a terminal re-emit wrote a request")
	}
}

// auditEmitJSON is an emit result as its --json reader sees it.
func auditEmitJSON(t *testing.T, res AuditEmitResult) map[string]any {
	t.Helper()
	b, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]any{"request_written": false}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}
