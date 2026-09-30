package capture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/record/match"
	"github.com/intentdriven/abcd/internal/gittest"
)

// consistency_test.go covers the ledger half of the consistency pass (itd-48
// AC 2): one capture per finding, evidenced by the report, and a finding an
// open record already holds linked to that record rather than filed twice.

const (
	cxA      = ".abcd/development/intents/planned/itd-10-one-spec.md"
	cxB      = ".abcd/development/intents/shipped/itd-11-many-specs.md"
	cxQuoteA = "An intent carries exactly one spec for its whole life."
	cxQuoteB = "An intent owns one or more specs, each closed in turn."
)

func consistencyLedgerRepo(t *testing.T) string {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write(cxA, "---\nid: itd-10\nslug: one-spec\nkind: standalone\nspec_id: spc-1\n---\n\n# One spec\n\n## Press Release\n\n"+cxQuoteA+"\n")
	r.Write(cxB, "---\nid: itd-11\nslug: many-specs\nkind: standalone\nspec_id: spc-2\n---\n\n# Many specs\n\n## Decisions\n\n1. "+cxQuoteB+"\n")
	r.Commit("fixture")
	return r.Root()
}

// consistencyPayload emits a corpus request and returns a findings payload
// echoing its provenance, carrying the one contradiction between itd-10 and itd-11.
func consistencyPayload(t *testing.T, root string) []byte {
	t.Helper()
	return consistencyPayloadWith(t, root, []any{map[string]any{
		"class": "premise_contradiction", "severity": "major",
		"summary":     "itd-10 and itd-11 disagree on how many specs an intent owns",
		"explanation": "One says exactly one spec for life; the other says one or more.",
		"ends": []any{
			map[string]any{"path": cxA, "quote": cxQuoteA},
			map[string]any{"path": cxB, "quote": cxQuoteB},
		},
	}})
}

// consistencyPayloadWith is consistencyPayload carrying the findings given.
func consistencyPayloadWith(t *testing.T, root string, findings []any) []byte {
	t.Helper()
	em, err := intent.EmitConsistency(root, "", intent.ConsistencyEmitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	req, err := os.ReadFile(filepath.Join(root, em.RequestPath))
	if err != nil {
		t.Fatal(err)
	}
	rubric := regexp.MustCompile(`(?m)^- rubric_hash: (\S+)$`).FindStringSubmatch(string(req))[1]
	prompt := regexp.MustCompile(`(?m)^- prompt_hash: (\S+)$`).FindStringSubmatch(string(req))[1]
	b, err := json.Marshal(map[string]any{
		"_type":      intent.ConsistencyType,
		"receipt_id": em.ReceiptID,
		"verifier":   map[string]any{"id": "intent-auditor", "version": "claude-opus-5-5"},
		"policy":     map[string]any{"rubric_hash": rubric, "prompt_hash": prompt},
		"findings":   findings,
	})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestConsistencyFilesOneCapturePerFinding: the finding lands as an open
// inconsistency record from an agent, found during the pass that names the
// report, located at its first end, naming both ends and the report as its
// evidence, and related to the intents it sits in.
func TestConsistencyFilesOneCapturePerFinding(t *testing.T) {
	root := consistencyLedgerRepo(t)
	res, err := IngestConsistency(root, consistencyPayload(t, root), "2026-09-26", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "ingested" || len(res.Filed) != 1 || len(res.Linked) != 0 {
		t.Fatalf("ingest = %+v; want one record filed", res)
	}
	list, err := List(ListRequest{RepoRoot: root, State: StateOpen})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Issues) != 1 || list.Issues[0].ID != res.Filed[0] {
		t.Fatalf("open ledger = %+v; want exactly %s", list.Issues, res.Filed[0])
	}
	iss := list.Issues[0]
	if iss.Category != "inconsistency" || iss.Source != "agent-finding" || iss.Severity != "major" {
		t.Errorf("record classified %s/%s/%s; want inconsistency/agent-finding/major", iss.Category, iss.Source, iss.Severity)
	}
	if !strings.Contains(iss.FoundDuring, res.ReportPath) || iss.FoundAt != cxA+":12" {
		t.Errorf("found_during %q / found_at %q; want the report named and the first end located", iss.FoundDuring, iss.FoundAt)
	}
	if strings.Join(iss.RelatedIntents, ",") != "itd-10,itd-11" {
		t.Errorf("related_intents = %v; want itd-10,itd-11", iss.RelatedIntents)
	}
	for _, want := range []string{cxA + ":12", cxB + ":12", cxQuoteA, cxQuoteB, res.ReportPath, "premise contradiction"} {
		if !strings.Contains(iss.Body, want) {
			t.Errorf("record body lacks %q:\n%s", want, iss.Body)
		}
	}
}

// TestConsistencyLinksAFindingAnOpenRecordHolds: an open record quoting either
// end in its document is linked and nothing is filed; a record that only names
// the document, or that is no longer open, is not a match.
func TestConsistencyLinksAFindingAnOpenRecordHolds(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		resolve  bool
		wantLink bool
	}{
		{"open, quotes end B in its document", "Seen in " + cxB + ": \"" + cxQuoteB + "\" — contradicts the planned record.", false, true},
		{"open, quotes end A by its intent id", "itd-10 says: " + cxQuoteA, false, true},
		{"open, names the document without the quote", "Something else is wrong in " + cxB + ".", false, false},
		{"open, quote under another intent's id", "itd-100 says: " + cxQuoteA, false, false},
		{"resolved, quotes end B", "Seen in " + cxB + ": \"" + cxQuoteB + "\".", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := consistencyLedgerRepo(t)
			held, err := testCapture(CaptureRequest{RepoRoot: root, Text: tc.body, Severity: "minor",
				Category: "inconsistency", Source: "user-observation", FoundDuring: "fixture"})
			if err != nil {
				t.Fatal(err)
			}
			if tc.resolve {
				if _, err := Resolve(ResolveRequest{RepoRoot: root, ID: held.ID, Resolution: "fixed by hand", Impact: "internal"}); err != nil {
					t.Fatal(err)
				}
			}
			res, err := IngestConsistency(root, consistencyPayload(t, root), "2026-09-26", nil)
			if err != nil {
				t.Fatal(err)
			}
			linked := len(res.Linked) == 1 && res.Linked[0] == held.ID && len(res.Filed) == 0
			if linked != tc.wantLink {
				t.Fatalf("ingest linked %v filed %v; want linked to %s: %v", res.Linked, res.Filed, held.ID, tc.wantLink)
			}
		})
	}
}

// The pass's words for a finding a person already filed in other words: the
// held record doubles it without quoting either end, so the pass's own
// end-matching (openRecordHolding) does not hold it, and only the filing-time
// match can.
const (
	cxDoubleSummary = "The planned intent and the shipped intent disagree on how many specs an intent owns over its lifetime"
	cxDoubleExplain = "The planned record binds every intent to exactly one spec for its whole lifetime, " +
		"while the shipped decision lets an intent own several specs closed in sequence."
	cxHeld = "Planned and shipped intents disagree on how many specs an intent owns over its lifetime: " +
		"the planned record binds each intent to exactly one spec for its whole lifetime, and the " +
		"shipped decision lets an intent own several specs closed in sequence."
)

func cxFinding(summary, explanation string) map[string]any {
	return map[string]any{
		"class": "premise_contradiction", "severity": "major",
		"summary": summary, "explanation": explanation,
		"ends": []any{
			map[string]any{"path": cxA, "quote": cxQuoteA},
			map[string]any{"path": cxB, "quote": cxQuoteB},
		},
	}
}

// A finding the pass files that doubles an open record is written with
// capture's typed link naming it, and the row carries the match.
func TestConsistencyLinksANearDuplicateAtFiling(t *testing.T) {
	root := consistencyLedgerRepo(t)
	held := captureText(t, root, "", cxHeld, nil)
	captureText(t, root, "", matchFiller1, nil)
	captureText(t, root, "", matchFiller2, nil)

	res, err := IngestConsistency(root, consistencyPayloadWith(t, root, []any{cxFinding(cxDoubleSummary, cxDoubleExplain)}), "2026-09-26", bundled())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Filed) != 1 || len(res.Rows) != 1 {
		t.Fatalf("ingest = %+v; want one record filed", res)
	}
	o := res.Rows[0].Match
	if o == nil || len(o.Matches) == 0 {
		t.Fatalf("no match on the row: %+v", o)
	}
	m := o.Matches[0]
	if m.ID != held.ID || !m.Linked || (m.Relation != match.Duplicates && m.Relation != match.Refines) {
		t.Fatalf("match = %+v, want %s linked", m, held.ID)
	}
	list, err := List(ListRequest{RepoRoot: root, State: StateOpen})
	if err != nil {
		t.Fatal(err)
	}
	for _, iss := range list.Issues {
		if iss.ID != res.Filed[0] {
			continue
		}
		got := iss.Duplicates
		if m.Relation == match.Refines {
			got = iss.Refines
		}
		if len(got) != 1 || got[0] != held.ID {
			t.Fatalf("filed record's %s = %v, want [%s]", m.Relation, got, held.ID)
		}
		return
	}
	t.Fatalf("filed record %s not listed", res.Filed[0])
}

// The match reads the finding's own words, not the lines every finding of a
// pass is filed with, and never the records this pass has already filed: two
// findings of one pass are two findings, as the pass's own end-matching has
// it.
func TestConsistencyNeverLinksTwoFindingsOfOnePass(t *testing.T) {
	root := consistencyLedgerRepo(t)
	captureText(t, root, "", matchFiller1, nil)
	captureText(t, root, "", matchFiller2, nil)
	// One finding under two classes: the pass files them as two records.
	second := cxFinding(cxDoubleSummary, cxDoubleExplain)
	second["class"] = "scope_leakage"
	payload := consistencyPayloadWith(t, root, []any{cxFinding(cxDoubleSummary, cxDoubleExplain), second})
	res, err := IngestConsistency(root, payload, "2026-09-26", bundled())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Filed) != 2 {
		t.Fatalf("ingest = %+v; want two records filed", res)
	}
	for _, r := range res.Rows {
		if r.Match == nil {
			t.Fatalf("row %d was not matched at all", r.Number)
		}
		for _, m := range r.Match.Matches {
			if m.ID == res.Filed[0] || m.ID == res.Filed[1] {
				t.Fatalf("finding %d was linked to a record of its own pass: %+v", r.Number, m)
			}
		}
	}
}
