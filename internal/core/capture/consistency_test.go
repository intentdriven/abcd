package capture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/record/findingkey"
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

// cxKey is the key the pass derives for consistencyPayload's one finding.
func cxKey() string {
	return findingkey.Derive("premise_contradiction", findingkey.End{Path: cxA, Quote: cxQuoteA}, findingkey.End{Path: cxB, Quote: cxQuoteB})
}

// TestConsistencyFiledRecordCarriesTheKeyAndTheNextRunLinksIt is G1's
// criterion 9: the record a finding files carries the finding's key as
// `finding_ref:`, and the same contradiction found on a second run — reported
// in other words, so the payload is not the first one again — carries the same
// key and links to that record rather than filing a second.
func TestConsistencyFiledRecordCarriesTheKeyAndTheNextRunLinksIt(t *testing.T) {
	root := consistencyLedgerRepo(t)
	first, err := IngestConsistency(root, consistencyPayload(t, root), "2026-09-26", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Filed) != 1 || first.Rows[0].Key != cxKey() {
		t.Fatalf("first run = %+v; want one record filed under key %s", first, cxKey())
	}
	list, err := List(ListRequest{RepoRoot: root, State: StateOpen})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Issues) != 1 || list.Issues[0].FindingRef != cxKey() {
		t.Fatalf("open ledger = %+v; want one record carrying finding_ref %s", list.Issues, cxKey())
	}
	if raw := readLedgerFile(t, root, list.Issues[0].Path); !strings.Contains(raw, "\nfinding_ref: "+cxKey()+"\n") {
		t.Fatalf("the filed record does not carry finding_ref in its frontmatter:\n%s", raw)
	}

	second, err := IngestConsistency(root, consistencyPayloadWith(t, root, []any{cxFinding(
		"the two intents still disagree about specs", "Reported again, in other words, on a later run.")}), "2026-09-27", nil)
	if err != nil {
		t.Fatal(err)
	}
	if second.Rows[0].Key != first.Rows[0].Key {
		t.Fatalf("second run's key %s, first run's %s; want one key for one contradiction", second.Rows[0].Key, first.Rows[0].Key)
	}
	if len(second.Filed) != 0 || strings.Join(second.Linked, ",") != first.Filed[0] || second.Rows[0].State != intent.ConsistencyStateLinked {
		t.Fatalf("second run filed %v linked %v (state %q); want it linked to %s", second.Filed, second.Linked, second.Rows[0].State, first.Filed[0])
	}
}

// TestConsistencyAnOpenKeyedRecordLinksWithoutTheProse: the key alone joins a
// finding to its open record, even one whose body no longer quotes either end,
// and linking writes nothing to it.
func TestConsistencyAnOpenKeyedRecordLinksWithoutTheProse(t *testing.T) {
	root := consistencyLedgerRepo(t)
	held, err := testCapture(CaptureRequest{RepoRoot: root, Text: "Reworded by a person; neither end is quoted any more.",
		Severity: "minor", Category: "inconsistency", Source: "agent-finding", FoundDuring: "fixture", FindingRef: cxKey()})
	if err != nil {
		t.Fatal(err)
	}
	before := readLedgerFile(t, root, held.Path)
	res, err := IngestConsistency(root, consistencyPayload(t, root), "2026-09-26", nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(res.Linked, ",") != held.ID || len(res.Filed) != 0 {
		t.Fatalf("linked %v filed %v; want linked to %s by its key", res.Linked, res.Filed, held.ID)
	}
	if after := readLedgerFile(t, root, held.Path); after != before {
		t.Fatalf("linking rewrote the record:\n%s", after)
	}
}

// TestConsistencyAKeyedClosedRecordAddressesTheFinding is G1's criterion 10: a
// resolved or won't-fix record carrying the finding's key makes the row
// `addressed`, naming that record and its folder; nothing is filed, and the
// record is left exactly as it was.
func TestConsistencyAKeyedClosedRecordAddressesTheFinding(t *testing.T) {
	for _, folder := range []string{"resolved", "wontfix"} {
		t.Run(folder, func(t *testing.T) {
			root := consistencyLedgerRepo(t)
			held, err := testCapture(CaptureRequest{RepoRoot: root, Text: "Seen in " + cxB + ": \"" + cxQuoteB + "\".",
				Severity: "minor", Category: "inconsistency", Source: "agent-finding", FoundDuring: "fixture", FindingRef: cxKey()})
			if err != nil {
				t.Fatal(err)
			}
			var moved TransitionResult
			if folder == "resolved" {
				moved, err = Resolve(ResolveRequest{RepoRoot: root, ID: held.ID, Resolution: "reconciled the two intents", Impact: "internal"})
			} else {
				moved, err = Wontfix(WontfixRequest{RepoRoot: root, ID: held.ID, Reason: "the two readings are both intended"})
			}
			if err != nil {
				t.Fatal(err)
			}
			before := readLedgerFile(t, root, moved.Path)
			res, err := IngestConsistency(root, consistencyPayload(t, root), "2026-09-26", nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(res.Filed) != 0 || len(res.Linked) != 0 || strings.Join(res.Addressed, ",") != held.ID {
				t.Fatalf("filed %v linked %v addressed %v; want only %s addressed", res.Filed, res.Linked, res.Addressed, held.ID)
			}
			row := res.Rows[0]
			if row.State != intent.ConsistencyStateAddressed || row.AddressedBy == nil ||
				*row.AddressedBy != (intent.ConsistencyAddress{IssueID: held.ID, Folder: folder}) {
				t.Fatalf("row = state %q addressed_by %+v; want addressed by %s in %s", row.State, row.AddressedBy, held.ID, folder)
			}
			list, err := List(ListRequest{RepoRoot: root, State: StateOpen})
			if err != nil {
				t.Fatal(err)
			}
			if len(list.Issues) != 0 {
				t.Fatalf("the ingest filed %+v; an addressed finding files nothing", list.Issues)
			}
			if after := readLedgerFile(t, root, moved.Path); after != before {
				t.Fatalf("the ingest rewrote the addressing record:\n%s", after)
			}
			body := readLedgerFile(t, root, res.ReportPath)
			if !strings.Contains(body, held.ID+" (addressed: "+folder+")") {
				t.Fatalf("report does not mark the finding addressed:\n%s", body)
			}
		})
	}
}

// TestConsistencyAnOpenKeyedRecordWinsOverAClosedOne: when an open record and
// a resolved one both carry the key, the finding is still being answered, so it
// links to the open record.
func TestConsistencyAnOpenKeyedRecordWinsOverAClosedOne(t *testing.T) {
	root := consistencyLedgerRepo(t)
	old, err := testCapture(CaptureRequest{RepoRoot: root, Text: "An earlier filing of the same contradiction.",
		Severity: "minor", Category: "inconsistency", Source: "agent-finding", FoundDuring: "fixture", FindingRef: cxKey()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(ResolveRequest{RepoRoot: root, ID: old.ID, Resolution: "thought fixed", Impact: "internal"}); err != nil {
		t.Fatal(err)
	}
	reopened, err := testCapture(CaptureRequest{RepoRoot: root, Text: "It came back.",
		Severity: "minor", Category: "inconsistency", Source: "agent-finding", FoundDuring: "fixture", FindingRef: cxKey()})
	if err != nil {
		t.Fatal(err)
	}
	res, err := IngestConsistency(root, consistencyPayload(t, root), "2026-09-26", nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(res.Linked, ",") != reopened.ID || len(res.Addressed) != 0 {
		t.Fatalf("linked %v addressed %v; want linked to the open %s", res.Linked, res.Addressed, reopened.ID)
	}
}

// TestConsistencyALegacyOpenRecordLinksByProse: a record filed before keys
// existed carries none; the prose match still links it, and nothing — no key —
// is written into it.
func TestConsistencyALegacyOpenRecordLinksByProse(t *testing.T) {
	root := consistencyLedgerRepo(t)
	held, err := testCapture(CaptureRequest{RepoRoot: root, Text: "Seen in " + cxB + ": \"" + cxQuoteB + "\".",
		Severity: "minor", Category: "inconsistency", Source: "user-observation", FoundDuring: "fixture"})
	if err != nil {
		t.Fatal(err)
	}
	before := readLedgerFile(t, root, held.Path)
	res, err := IngestConsistency(root, consistencyPayload(t, root), "2026-09-26", nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(res.Linked, ",") != held.ID || len(res.Filed) != 0 {
		t.Fatalf("linked %v filed %v; want the legacy %s linked by prose", res.Linked, res.Filed, held.ID)
	}
	if after := readLedgerFile(t, root, held.Path); after != before || strings.Contains(after, "finding_ref") {
		t.Fatalf("linking a legacy record wrote to it:\n%s", after)
	}
}

// TestNamesIDIsTheRecordIDToken: the consistency filer's id test is the one
// record-lint's A1 rules use (recordid.NamesID): a trailing letter or digit
// makes another word, a hyphen starts the record's own slug.
func TestNamesIDIsTheRecordIDToken(t *testing.T) {
	for text, want := range map[string]bool{
		"see itd-4.":        true,
		"see itd-4-slug.md": true,
		"see itd-48":        false,
		"see xitd-4":        false,
		"see itd-4abc":      false,
		"see itd-4étude":    false,
	} {
		if got := namesID(text, "itd-4"); got != want {
			t.Errorf("namesID(%q, itd-4) = %t, want %t", text, got, want)
		}
	}
}
