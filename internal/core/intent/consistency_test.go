package intent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// consistency_test.go covers Role 2's binary half (itd-48, spc-2609211921272106):
// the corpus assembly and request emit, the fail-closed validation of a findings
// payload, the dated report on the reviews shelf, and the filer seam that files
// or links one ledger record per finding.

const (
	cxBrief      = ".abcd/development/brief/01-product/01-review-queue.md"
	cxTemplate   = ".abcd/development/brief/glossary/_template.md"
	cxPlanned    = ".abcd/development/intents/planned/itd-10-one-spec.md"
	cxShipped    = ".abcd/development/intents/shipped/itd-11-many-specs.md"
	cxSuperseded = ".abcd/development/intents/superseded/itd-12-retired.md"
	cxDate       = "2026-09-26"

	cxQuoteBrief   = "The review queue drains on every close of a spec record."
	cxQuotePlanned = "An intent carries exactly one spec for its whole life."
	cxQuoteShipped = "An intent owns one or more specs, each closed in turn."
)

// consistencyRepo builds a committed checkout carrying a brief page, a template
// the corpus must skip, and three intents: one planned, one shipped and one
// superseded, which the corpus must also skip.
func consistencyRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write(cxBrief, "# Review queue\n\nIntro line.\n\n"+cxQuoteBrief+"\n")
	r.Write(cxTemplate, "# Template\n\nTEMPLATE-ONLY-TEXT should never reach the corpus.\n")
	r.Write(cxPlanned, "---\nid: itd-10\nslug: one-spec\nkind: standalone\nspec_id: spc-1\n---\n\n# One spec\n\n"+
		"## Press Release\n\n"+cxQuotePlanned+"\n\n### A sub-heading travels with its section\n\nSUBSECTION-TEXT here.\n\n"+
		"## Why This Matters\n\nWHY-TEXT is not compared.\n\n"+
		"## What's In Scope\n\n- the planned scope bullet\n\n"+
		"## Acceptance Criteria\n\n- ACCEPTANCE-TEXT is not compared.\n")
	r.Write(cxShipped, "---\nid: itd-11\nslug: many-specs\nkind: standalone\nspec_id: spc-2\n---\n\n# Many specs\n\n"+
		"## Press Release\n\nThe shipped press release.\n\n"+
		"## Decisions\n\n1. "+cxQuoteShipped+"\n\n"+
		"## Audit Notes\n\nAUDIT-TEXT is not compared.\n")
	r.Write(cxSuperseded, "---\nid: itd-12\nslug: retired\nkind: standalone\nsuperseded_by: itd-11\n---\n\n# Retired\n\n"+
		"## Press Release\n\nSUPERSEDED-TEXT never reaches the corpus.\n")
	r.Commit("fixture")
	return r
}

// recordDigest hashes every brief page and intent, so a test can assert the pass
// left the record exactly as it found it.
func recordDigest(t *testing.T, root string) string {
	t.Helper()
	h := sha256.New()
	var paths []string
	for _, dir := range []string{".abcd/development/brief", ".abcd/development/intents"} {
		_ = filepath.Walk(filepath.Join(root, dir), func(p string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				paths = append(paths, p)
			}
			return nil
		})
	}
	sort.Strings(paths)
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(h, "%s\x00%x\n", p, sha256.Sum256(data))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// provenanceOf reads the two hashes a request's Provenance block states.
func provenanceOf(t *testing.T, root, requestRel string) (rubric, prompt string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, requestRel))
	if err != nil {
		t.Fatal(err)
	}
	rm := regexp.MustCompile(`(?m)^- rubric_hash: (\S+)$`).FindStringSubmatch(string(data))
	pm := regexp.MustCompile(`(?m)^- prompt_hash: (\S+)$`).FindStringSubmatch(string(data))
	if rm == nil || pm == nil {
		t.Fatalf("request %s states no provenance pair:\n%s", requestRel, data)
	}
	return rm[1], pm[1]
}

type cxEnd struct{ path, quote string }

type cxFinding struct {
	class, severity, summary, explanation string
	ends                                  []cxEnd
}

// findingsPayload builds a findings JSON for an emitted request, echoing the
// provenance the request states.
func findingsPayload(t *testing.T, root string, em ConsistencyEmitResult, fs ...cxFinding) []byte {
	t.Helper()
	rubric, prompt := provenanceOf(t, root, em.RequestPath)
	list := []any{}
	for _, f := range fs {
		var ends []any
		for _, e := range f.ends {
			ends = append(ends, map[string]any{"path": e.path, "quote": e.quote})
		}
		list = append(list, map[string]any{
			"class": f.class, "severity": f.severity, "summary": f.summary,
			"explanation": f.explanation, "ends": ends,
		})
	}
	b, err := json.MarshalIndent(map[string]any{
		"_type":      ConsistencyType,
		"receipt_id": em.ReceiptID,
		"verifier":   map[string]any{"id": "intent-auditor", "version": "claude-opus-5-5"},
		"policy":     map[string]any{"rubric_hash": rubric, "prompt_hash": prompt},
		"findings":   list,
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func contradiction() cxFinding {
	return cxFinding{
		class: "premise_contradiction", severity: "major",
		summary:     "itd-10 and itd-11 disagree on how many specs an intent owns",
		explanation: "One says exactly one spec for life; the other says one or more, closed in turn.",
		ends:        []cxEnd{{cxPlanned, cxQuotePlanned}, {cxShipped, cxQuoteShipped}},
	}
}

func briefDrift() cxFinding {
	return cxFinding{
		class: "sequencing_impossibility", severity: "minor",
		summary:     "the brief drains the queue on a close that itd-11 spreads over several specs",
		explanation: "A queue drained on every close runs once per spec, not once per intent.",
		ends:        []cxEnd{{cxBrief, cxQuoteBrief}, {cxShipped, cxQuoteShipped}},
	}
}

// fakeFiler records what it is asked to file and answers with sequential ids;
// the findings whose number is in link are answered as already held.
type fakeFiler struct {
	calls   []ConsistencyFinding
	reports []string
	link    map[int]string
	failAt  int
	onFile  func(reportRel string)
}

func (f *fakeFiler) file(fd ConsistencyFinding, reportRel string) (ConsistencyFiling, error) {
	f.calls = append(f.calls, fd)
	f.reports = append(f.reports, reportRel)
	if f.onFile != nil {
		f.onFile(reportRel)
	}
	if f.failAt == fd.Number {
		return ConsistencyFiling{}, fmt.Errorf("ledger unavailable")
	}
	if id, ok := f.link[fd.Number]; ok {
		return ConsistencyFiling{IssueID: id, Linked: true}, nil
	}
	return ConsistencyFiling{IssueID: fmt.Sprintf("iss-90%d", len(f.calls))}, nil
}

func ingest(t *testing.T, root string, payload []byte, f *fakeFiler) (ConsistencyIngestResult, error) {
	t.Helper()
	return IngestConsistency(ConsistencyIngestRequest{RepoRoot: root, Payload: payload, Date: cxDate, File: f.file})
}

// TestConsistencyEmitAssemblesTheCorpusAndWritesOnlyTheLocalTier: the bare emit
// reads every brief page and every live intent's compared sections, skips the
// template and the superseded record, names the commit it read, and writes the
// request and the corpus under the local tier and nothing else.
func TestConsistencyEmitAssemblesTheCorpusAndWritesOnlyTheLocalTier(t *testing.T) {
	r := consistencyRepo(t)
	root := r.Root()
	before := recordDigest(t, root)
	head := r.Git("rev-parse", "HEAD")

	em, err := EmitConsistency(root, "", ConsistencyEmitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if em.Status != "issued" || em.Scope != ConsistencyScopeCorpus || em.ReviewOfCommit != head {
		t.Fatalf("emit = %+v, want issued over the corpus at %s", em, head)
	}
	if em.BriefDocuments != 1 || em.IntentDocuments != 2 || em.Documents != 3 {
		t.Fatalf("emit counted %d brief + %d intents = %d, want 1 + 2 = 3 (template and superseded skipped)",
			em.BriefDocuments, em.IntentDocuments, em.Documents)
	}
	corpus, err := os.ReadFile(filepath.Join(root, em.CorpusPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{cxQuoteBrief, cxQuotePlanned, cxQuoteShipped, "SUBSECTION-TEXT", "the planned scope bullet", cxBrief, cxPlanned, cxShipped} {
		if !strings.Contains(string(corpus), want) {
			t.Errorf("corpus lacks %q", want)
		}
	}
	for _, not := range []string{"TEMPLATE-ONLY-TEXT", "SUPERSEDED-TEXT", "WHY-TEXT", "ACCEPTANCE-TEXT", "AUDIT-TEXT"} {
		if strings.Contains(string(corpus), not) {
			t.Errorf("corpus carries %q, which the pass does not compare", not)
		}
	}
	req, err := os.ReadFile(filepath.Join(root, em.RequestPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{em.ReceiptID, "- scope: corpus", "- review_of_commit: " + head, "rubric_hash: sha256:", "prompt_hash: sha256:",
		"terminology_drift", "premise_contradiction", "scope_leakage", "sequencing_impossibility", "naming_conflict",
		"abcd intent consistency ingest --findings-json"} {
		if !strings.Contains(string(req), want) {
			t.Errorf("request lacks %q:\n%s", want, req)
		}
	}
	if !strings.HasPrefix(em.RequestPath, ".abcd/.work.local/reviews/") || !strings.HasPrefix(em.CorpusPath, ".abcd/.work.local/reviews/") {
		t.Fatalf("emit wrote outside the local tier: %s, %s", em.RequestPath, em.CorpusPath)
	}
	if after := recordDigest(t, root); after != before {
		t.Fatal("the emit changed the brief or an intent")
	}
	again, err := EmitConsistency(root, "", ConsistencyEmitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if again.ReceiptID != em.ReceiptID {
		t.Fatalf("a re-emit over an unchanged corpus minted %s, want %s", again.ReceiptID, em.ReceiptID)
	}
}

// TestConsistencyEmitScopesToOneIntent: a scoped emit names its intent and
// mints a receipt distinct from the corpus run; a superseded or unknown intent
// is refused before anything is written.
func TestConsistencyEmitScopesToOneIntent(t *testing.T) {
	r := consistencyRepo(t)
	root := r.Root()
	bare, err := EmitConsistency(root, "", ConsistencyEmitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	em, err := EmitConsistency(root, "itd-10", ConsistencyEmitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if em.Scope != "itd-10" || em.ReceiptID == bare.ReceiptID {
		t.Fatalf("scoped emit = %+v; want scope itd-10 and a receipt distinct from %s", em, bare.ReceiptID)
	}
	req, _ := os.ReadFile(filepath.Join(root, em.RequestPath))
	if !strings.Contains(string(req), "- scope: itd-10 ("+cxPlanned+" against the rest of the corpus)") {
		t.Fatalf("scoped request does not say what it is scoped to:\n%s", req)
	}
	for _, id := range []string{"itd-12", "itd-99", "iss-1"} {
		if _, err := EmitConsistency(root, id, ConsistencyEmitOptions{}); err == nil {
			t.Errorf("EmitConsistency(%s) succeeded; want a refusal", id)
		}
	}
}

// TestConsistencyIngestWritesTheDatedReport is AC 1 and AC 2: a validated
// payload lands as a dated report on the reviews shelf naming the commit it
// read, with both ends of every finding quoted and located, and one filing per
// finding whose evidence is that report.
func TestConsistencyIngestWritesTheDatedReport(t *testing.T) {
	r := consistencyRepo(t)
	root := r.Root()
	head := r.Git("rev-parse", "HEAD")
	em, err := EmitConsistency(root, "", ConsistencyEmitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	before := recordDigest(t, root)
	f := &fakeFiler{}
	res, err := ingest(t, root, findingsPayload(t, root, em, contradiction(), briefDrift()), f)
	if err != nil {
		t.Fatal(err)
	}
	wantRel := ".abcd/work/reviews/" + cxDate + "-consistency/00-summary.md"
	if res.Status != "ingested" || res.ReportPath != wantRel || res.Findings != 2 || res.ReviewOfCommit != head {
		t.Fatalf("ingest = %+v, want ingested at %s", res, wantRel)
	}
	if len(f.calls) != 2 || f.reports[0] != wantRel || f.reports[1] != wantRel {
		t.Fatalf("filer calls = %d with reports %v; want two, each evidenced by %s", len(f.calls), f.reports, wantRel)
	}
	if got := strings.Join(res.Filed, ","); got != "iss-901,iss-902" || len(res.Linked) != 0 {
		t.Fatalf("filed %v linked %v; want iss-901,iss-902 and none linked", res.Filed, res.Linked)
	}
	body, err := os.ReadFile(filepath.Join(root, wantRel))
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	for _, want := range []string{
		"---\nreview_of_commit: " + head + "\n---\n",
		"- receipt: " + em.ReceiptID,
		"premise contradiction", "sequencing impossibility",
		"`" + cxPlanned + ":12`", "`" + cxShipped + ":16`", "`" + cxBrief + ":5`",
		"\u201c" + cxQuotePlanned + "\u201d", "\u201c" + cxQuoteShipped + "\u201d", "\u201c" + cxQuoteBrief + "\u201d",
		"iss-901 (filed)", "iss-902 (filed)",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("report lacks %q:\n%s", want, s)
		}
	}
	// AC 4: the brief and the intents are unchanged.
	if after := recordDigest(t, root); after != before {
		t.Fatal("the ingest changed the brief or an intent")
	}
	// The ends reach the filer located, with the intents they sit in.
	c := f.calls[0]
	if c.Ends[0].Line != 12 || c.Ends[1].Line != 16 || strings.Join(c.IntentIDs(), ",") != "itd-10,itd-11" {
		t.Fatalf("filer got ends %+v (intents %v); want lines 12/16 in itd-10/itd-11", c.Ends, c.IntentIDs())
	}
}

// TestConsistencyIngestLinksAFindingTheLedgerHolds is AC 2's dedup half at the
// report: a finding the filer answers as already held is linked, not filed.
func TestConsistencyIngestLinksAFindingTheLedgerHolds(t *testing.T) {
	r := consistencyRepo(t)
	root := r.Root()
	em, err := EmitConsistency(root, "", ConsistencyEmitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeFiler{link: map[int]string{1: "iss-42"}}
	res, err := ingest(t, root, findingsPayload(t, root, em, contradiction(), briefDrift()), f)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(res.Linked, ",") != "iss-42" || len(res.Filed) != 1 {
		t.Fatalf("linked %v filed %v; want iss-42 linked and one filed", res.Linked, res.Filed)
	}
	body, _ := os.ReadFile(filepath.Join(root, res.ReportPath))
	if !strings.Contains(string(body), "iss-42 (already open)") {
		t.Fatalf("report does not say the finding was linked:\n%s", body)
	}
}

// TestConsistencyIngestScopedRun is AC 3: a scoped run's report says so and
// lives in its own directory, and a finding with no end in the scoped intent is
// refused.
func TestConsistencyIngestScopedRun(t *testing.T) {
	r := consistencyRepo(t)
	root := r.Root()
	em, err := EmitConsistency(root, "itd-10", ConsistencyEmitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeFiler{}
	if _, err := ingest(t, root, findingsPayload(t, root, em, briefDrift()), f); err == nil ||
		!strings.Contains(err.Error(), "no end in itd-10") {
		t.Fatalf("a finding outside the scope was not refused: %v", err)
	}
	if len(f.calls) != 0 {
		t.Fatal("a refused payload reached the filer")
	}
	res, err := ingest(t, root, findingsPayload(t, root, em, contradiction()), f)
	if err != nil {
		t.Fatal(err)
	}
	if want := ".abcd/work/reviews/" + cxDate + "-consistency-itd-10/00-summary.md"; res.ReportPath != want || res.Scope != "itd-10" {
		t.Fatalf("scoped ingest = %+v, want report %s", res, want)
	}
	body, _ := os.ReadFile(filepath.Join(root, res.ReportPath))
	if !strings.Contains(string(body), "# Consistency review — itd-10 against the corpus") ||
		!strings.Contains(string(body), "- scope: itd-10 (`"+cxPlanned+"`) against the rest of the corpus") {
		t.Fatalf("scoped report does not say it was scoped:\n%s", body)
	}
}

// TestConsistencyIngestRefusesWithNothingWritten: every malformed, forged or
// stale payload is refused before the filer runs and before any report exists.
func TestConsistencyIngestRefusesWithNothingWritten(t *testing.T) {
	good := func(t *testing.T, root string, em ConsistencyEmitResult) map[string]any {
		var m map[string]any
		if err := json.Unmarshal(findingsPayload(t, root, em, contradiction()), &m); err != nil {
			t.Fatal(err)
		}
		return m
	}
	finding := func(m map[string]any) map[string]any { return m["findings"].([]any)[0].(map[string]any) }
	cases := []struct {
		name   string
		mutate func(t *testing.T, root string, m map[string]any)
		want   string
	}{
		{"wrong type", func(_ *testing.T, _ string, m map[string]any) { m["_type"] = VerdictType }, "is not"},
		{"malformed receipt", func(_ *testing.T, _ string, m map[string]any) { m["receipt_id"] = "rcp-../../x" }, "no resolvable receipt_id"},
		{"unsolicited receipt", func(_ *testing.T, _ string, m map[string]any) { m["receipt_id"] = "rcp-000000000000" }, "no consistency request was issued"},
		{"forged prompt hash", func(_ *testing.T, _ string, m map[string]any) {
			m["policy"].(map[string]any)["prompt_hash"] = "sha256:" + strings.Repeat("b", 64)
		}, "never issued"},
		{"unknown field", func(_ *testing.T, _ string, m map[string]any) { m["verdict"] = "SHIP" }, "malformed findings JSON"},
		{"class outside the set", func(_ *testing.T, _ string, m map[string]any) { finding(m)["class"] = "kind_change" }, "has class"},
		{"severity outside the set", func(_ *testing.T, _ string, m map[string]any) { finding(m)["severity"] = "high" }, "has severity"},
		{"no explanation", func(_ *testing.T, _ string, m map[string]any) { finding(m)["explanation"] = " " }, "no explanation"},
		{"one end", func(_ *testing.T, _ string, m map[string]any) {
			finding(m)["ends"] = finding(m)["ends"].([]any)[:1]
		}, "exactly two"},
		{"path outside the manifest", func(_ *testing.T, _ string, m map[string]any) {
			finding(m)["ends"].([]any)[1].(map[string]any)["path"] = cxSuperseded
		}, "not a document in the corpus manifest"},
		{"quote not in the document", func(_ *testing.T, _ string, m map[string]any) {
			finding(m)["ends"].([]any)[1].(map[string]any)["quote"] = "A sentence nobody ever wrote down."
		}, "does not occur"},
		{"quote from an uncompared section", func(_ *testing.T, _ string, m map[string]any) {
			finding(m)["ends"].([]any)[1].(map[string]any)["quote"] = "AUDIT-TEXT is not compared."
		}, "does not occur"},
		{"quote too short", func(_ *testing.T, _ string, m map[string]any) {
			finding(m)["ends"].([]any)[1].(map[string]any)["quote"] = "An intent"
		}, "shorter than"},
		{"the same end twice", func(_ *testing.T, _ string, m map[string]any) {
			ends := finding(m)["ends"].([]any)
			ends[1] = ends[0]
		}, "same end twice"},
		{"a repeated finding", func(_ *testing.T, _ string, m map[string]any) {
			m["findings"] = append(m["findings"].([]any), finding(m))
		}, "repeats finding 1"},
		{"the corpus moved", func(t *testing.T, root string, _ map[string]any) {
			if err := os.WriteFile(filepath.Join(root, cxBrief), []byte("# Review queue\n\nRewritten after the emit.\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, "the corpus has moved"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := consistencyRepo(t)
			root := r.Root()
			em, err := EmitConsistency(root, "", ConsistencyEmitOptions{})
			if err != nil {
				t.Fatal(err)
			}
			m := good(t, root, em)
			tc.mutate(t, root, m)
			payload, _ := json.Marshal(m)
			f := &fakeFiler{}
			_, err = ingest(t, root, payload, f)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ingest error = %v, want one naming %q", err, tc.want)
			}
			if len(f.calls) != 0 {
				t.Fatal("a refused payload reached the filer")
			}
			if _, err := os.Stat(filepath.Join(root, ReviewsShelfRelDir)); !os.IsNotExist(err) {
				t.Fatal("a refused payload left something on the reviews shelf")
			}
		})
	}
}

// TestConsistencyIngestIsIdempotentAndAppendOnly: the same bytes again are a
// noop naming the report that holds them; a different payload for the same
// receipt is a second review in its own directory, never an edit of the first.
func TestConsistencyIngestIsIdempotentAndAppendOnly(t *testing.T) {
	r := consistencyRepo(t)
	root := r.Root()
	em, err := EmitConsistency(root, "", ConsistencyEmitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	payload := findingsPayload(t, root, em, contradiction())
	f := &fakeFiler{}
	first, err := ingest(t, root, payload, f)
	if err != nil {
		t.Fatal(err)
	}
	firstBody, _ := os.ReadFile(filepath.Join(root, first.ReportPath))
	again, err := ingest(t, root, payload, f)
	if err != nil {
		t.Fatal(err)
	}
	if again.Status != "noop" || again.ReportPath != first.ReportPath || len(f.calls) != 1 {
		t.Fatalf("re-ingest = %+v after %d filer calls; want a noop at %s and one call", again, len(f.calls), first.ReportPath)
	}
	second, err := ingest(t, root, findingsPayload(t, root, em, contradiction(), briefDrift()), f)
	if err != nil {
		t.Fatal(err)
	}
	if want := ".abcd/work/reviews/" + cxDate + "-consistency-2/00-summary.md"; second.ReportPath != want {
		t.Fatalf("a second review the same day landed at %s, want %s", second.ReportPath, want)
	}
	if now, _ := os.ReadFile(filepath.Join(root, first.ReportPath)); string(now) != string(firstBody) {
		t.Fatal("the second review edited the first report")
	}
}

// TestConsistencyIngestAnEmptyPassIsAReport: a pass that finds nothing still
// leaves its dated report, and files nothing.
func TestConsistencyIngestAnEmptyPassIsAReport(t *testing.T) {
	r := consistencyRepo(t)
	root := r.Root()
	em, err := EmitConsistency(root, "", ConsistencyEmitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeFiler{}
	res, err := ingest(t, root, findingsPayload(t, root, em), f)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(filepath.Join(root, res.ReportPath))
	if res.Findings != 0 || len(f.calls) != 0 || !strings.Contains(string(body), "None: the pass found no contradiction") {
		t.Fatalf("empty pass = %+v, %d calls:\n%s", res, len(f.calls), body)
	}
}

// TestConsistencyIngestAFilingFailureWritesNoReport: when the ledger refuses a
// finding midway, no report is written and the error names what was filed.
func TestConsistencyIngestAFilingFailureWritesNoReport(t *testing.T) {
	r := consistencyRepo(t)
	root := r.Root()
	em, err := EmitConsistency(root, "", ConsistencyEmitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeFiler{failAt: 2}
	_, err = ingest(t, root, findingsPayload(t, root, em, contradiction(), briefDrift()), f)
	if err == nil || !strings.Contains(err.Error(), "filed before it: iss-901") {
		t.Fatalf("ingest error = %v; want one naming the record filed before the failure", err)
	}
	if _, err := os.Stat(filepath.Join(root, ReviewsShelfRelDir)); !os.IsNotExist(err) {
		t.Fatal("a failed filing left a report on the shelf")
	}
}

// TestConsistencyReportRedactsTheReviewersProse: an explanation carrying a home
// path, a hostname and a person's name reaches the committed report redacted.
func TestConsistencyReportRedactsTheReviewersProse(t *testing.T) {
	r := consistencyRepo(t)
	root := r.Root()
	r.Git("config", "user.name", "Jonathan Kensington-Pryce")
	em, err := EmitConsistency(root, "", ConsistencyEmitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	leak := contradiction()
	leak.explanation = "Checked against /Users/zzotherperson/checkouts/abcd/x.md on buildbox.local with Jonathan Kensington-Pryce."
	res, err := ingest(t, root, findingsPayload(t, root, em, leak), &fakeFiler{})
	if err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(filepath.Join(root, res.ReportPath))
	assertNoLeak(t, string(body))
}

// TestConsistencyQuoteMatchesAcrossANoBreakSpace: a document whose sentence
// carries a no-break space is located by a quote copied from it verbatim, since
// the quote and the document collapse whitespace the same way.
func TestConsistencyQuoteMatchesAcrossANoBreakSpace(t *testing.T) {
	r := consistencyRepo(t)
	root := r.Root()
	nbsp := "The review queue drains on every close of a spec record."
	r.Write(cxBrief, "# Review queue\n\n"+nbsp+"\n")
	r.Commit("no-break space")
	em, err := EmitConsistency(root, "", ConsistencyEmitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	f := briefDrift()
	f.ends[0].quote = nbsp
	res, err := ingest(t, root, findingsPayload(t, root, em, f), &fakeFiler{})
	if err != nil {
		t.Fatalf("a verbatim quote carrying a no-break space was refused: %v", err)
	}
	if res.Rows[0].Ends[0].Line != 3 {
		t.Fatalf("the quote was located at line %d, want 3", res.Rows[0].Ends[0].Line)
	}
}

// TestConsistencyADirtyCorpusIsMarkedDirty: the report pins HEAD, but the pass
// reads the corpus from the working tree, so an uncommitted or untracked corpus
// document means the quoted text may not be at the pinned commit. itd-28's
// dirty-tree policy for a review pin is to tag the review `dirty: true` and not
// block: the emit names the uncommitted corpus paths, the request carries the
// mark, and the report says so beside the pin. A change outside the corpus —
// a brief template the pass skips, a file elsewhere in the tree — leaves the
// report clean.
func TestConsistencyADirtyCorpusIsMarkedDirty(t *testing.T) {
	r := consistencyRepo(t)
	root := r.Root()
	head := r.Git("rev-parse", "HEAD")

	// Outside the corpus: not dirty.
	r.Write(cxTemplate, "# Template\n\nAn uncommitted edit to a template the pass skips.\n")
	r.Write("README.md", "an untracked file outside the corpus\n")
	clean, err := EmitConsistency(root, "", ConsistencyEmitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if m := emitJSON(t, clean); m["dirty"] != false {
		t.Fatalf("emit over a tree dirty only outside the corpus = %v; want dirty false", m)
	}

	// Inside the corpus: an uncommitted edit and an untracked page.
	const edited = "An uncommitted sentence the pinned commit does not hold."
	const untracked = ".abcd/development/brief/01-product/02-new-page.md"
	r.Write(cxBrief, "# Review queue\n\nIntro line.\n\n"+cxQuoteBrief+"\n\n"+edited+"\n")
	r.Write(untracked, "# New page\n\nAn untracked brief page.\n")
	em, err := EmitConsistency(root, "", ConsistencyEmitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	m := emitJSON(t, em)
	if m["dirty"] != true || m["review_of_commit"] != head {
		t.Fatalf("emit over an uncommitted corpus = %v; want dirty true at %s", m, head)
	}
	if got := fmt.Sprint(m["dirty_paths"]); got != "["+cxBrief+" "+untracked+"]" {
		t.Fatalf("emit dirty_paths = %s; want the edited page and the untracked page", got)
	}
	req, err := os.ReadFile(filepath.Join(root, em.RequestPath))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(req), "- dirty: true\n") {
		t.Fatalf("request does not carry the dirty mark:\n%s", req)
	}

	f := &fakeFiler{}
	drift := cxFinding{
		class: "terminology_drift", severity: "minor",
		summary:     "the brief quotes a sentence the pinned commit does not hold",
		explanation: "The quoted end is an uncommitted edit.",
		ends:        []cxEnd{{cxBrief, edited}, {cxShipped, cxQuoteShipped}},
	}
	res, err := ingest(t, root, findingsPayload(t, root, em, drift), f)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(root, res.ReportPath))
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	for _, want := range []string{
		"---\nreview_of_commit: " + head + "\ndirty: true\n---\n",
		"`" + cxBrief + "`", "`" + untracked + "`",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("report lacks %q:\n%s", want, s)
		}
	}
}

// emitJSON renders an emit result as its JSON object, the shape a front door's
// --json carries.
func emitJSON(t *testing.T, em ConsistencyEmitResult) map[string]any {
	t.Helper()
	b, err := json.Marshal(em)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

// TestConsistencyIngestRefusesARequestSilentOnDirtiness: a request that does
// not say whether the corpus it read was committed, or says dirty without
// naming a path, is refused with nothing written, so a report never pins a
// commit without saying whether the read matched it.
func TestConsistencyIngestRefusesARequestSilentOnDirtiness(t *testing.T) {
	for name, edit := range map[string]func(string) string{
		"no dirty line":  func(s string) string { return strings.Replace(s, "- dirty: false\n", "", 1) },
		"dirty, no path": func(s string) string { return strings.Replace(s, "- dirty: false\n", "- dirty: true\n", 1) },
	} {
		t.Run(name, func(t *testing.T) {
			r := consistencyRepo(t)
			root := r.Root()
			em, err := EmitConsistency(root, "", ConsistencyEmitOptions{})
			if err != nil {
				t.Fatal(err)
			}
			p := filepath.Join(root, em.RequestPath)
			req, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(edit(string(req))), 0o644); err != nil {
				t.Fatal(err)
			}
			f := &fakeFiler{}
			if _, err := ingest(t, root, findingsPayload(t, root, em, contradiction()), f); err == nil {
				t.Fatal("ingest accepted a request that does not state the tree's dirtiness")
			}
			if len(f.calls) != 0 {
				t.Fatalf("a refused ingest filed %d finding(s)", len(f.calls))
			}
			if _, err := os.Stat(filepath.Join(root, ReviewsShelfRelDir)); !os.IsNotExist(err) {
				t.Fatal("a refused ingest left a report on the shelf")
			}
		})
	}
}

// TestConsistencyIngestAReportFailureNamesTheFiledRecords: the findings are
// filed before the report is created, so when the create fails — here a
// concurrent ingest took the same report path while the findings were being
// filed — the error names the records already filed and linked, whose evidence
// line cites a report this ingest did not write.
func TestConsistencyIngestAReportFailureNamesTheFiledRecords(t *testing.T) {
	r := consistencyRepo(t)
	root := r.Root()
	em, err := EmitConsistency(root, "", ConsistencyEmitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeFiler{link: map[int]string{2: "iss-777"}}
	f.onFile = func(reportRel string) {
		p := filepath.Join(root, filepath.FromSlash(reportRel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("a concurrent ingest's report\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, err = ingest(t, root, findingsPayload(t, root, em, contradiction(), briefDrift()), f)
	if err == nil {
		t.Fatal("ingest succeeded over a report path another ingest holds")
	}
	for _, want := range []string{"iss-901", "iss-777", "creating report"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("report-failure error does not name %q: %v", want, err)
		}
	}
}

// TestConsistencyIngestRecomputesTheDirtyMark: the request's dirty mark is the
// emit's reading of the tree, carried as local-tier text anyone can edit, so the
// ingest reads the tree again against the pinned commit and takes the union with
// the request's paths — it marks, and never unmarks. A request hand-edited to
// `dirty: false` over an uncommitted corpus still yields a report marked dirty
// that names the real path, and so does one whose edit was committed between
// the emit and the ingest (the tree then matches HEAD, not the pinned commit).
// A path the request names that the tree no longer shows stays named.
func TestConsistencyIngestRecomputesTheDirtyMark(t *testing.T) {
	const edited = "An uncommitted sentence the pinned commit does not hold."
	drift := cxFinding{
		class: "terminology_drift", severity: "minor",
		summary:     "the brief quotes a sentence the pinned commit does not hold",
		explanation: "The quoted end is an uncommitted edit.",
		ends:        []cxEnd{{cxBrief, edited}, {cxShipped, cxQuoteShipped}},
	}
	forgeClean := func(s string) string {
		s = regexp.MustCompile(`(?m)^- dirty_path: .*\n`).ReplaceAllString(s, "")
		return strings.Replace(s, "- dirty: true\n", "- dirty: false\n", 1)
	}
	for name, tc := range map[string]struct {
		dirty   bool                  // edit the corpus before the emit
		forge   func(string) string   // rewrite the issued request
		since   func(r *gittest.Repo) // act on the tree between emit and ingest
		finding cxFinding
		want    []string
	}{
		"forged clean over a dirty tree": {
			dirty: true, forge: forgeClean, finding: drift, want: []string{cxBrief},
		},
		"forged clean, the edit committed since": {
			dirty: true, forge: forgeClean, finding: drift, want: []string{cxBrief},
			since: func(r *gittest.Repo) {
				r.Git("add", "--", cxBrief)
				r.Git("commit", "-m", "commit the edit the pass read")
			},
		},
		"a named path the tree no longer shows": {
			forge: func(s string) string {
				return strings.Replace(s, "- dirty: false\n", "- dirty: true\n- dirty_path: "+cxPlanned+"\n", 1)
			},
			finding: contradiction(), want: []string{cxPlanned},
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := consistencyRepo(t)
			root := r.Root()
			head := r.Git("rev-parse", "HEAD")
			if tc.dirty {
				r.Write(cxBrief, "# Review queue\n\nIntro line.\n\n"+cxQuoteBrief+"\n\n"+edited+"\n")
			}
			em, err := EmitConsistency(root, "", ConsistencyEmitOptions{})
			if err != nil {
				t.Fatal(err)
			}
			p := filepath.Join(root, em.RequestPath)
			req, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			forged := tc.forge(string(req))
			if forged == string(req) {
				t.Fatalf("the forge left the request unchanged:\n%s", req)
			}
			if err := os.WriteFile(p, []byte(forged), 0o644); err != nil {
				t.Fatal(err)
			}
			if tc.since != nil {
				tc.since(r)
			}
			res, err := ingest(t, root, findingsPayload(t, root, em, tc.finding), &fakeFiler{})
			if err != nil {
				t.Fatal(err)
			}
			if !res.Dirty || res.ReviewOfCommit != head {
				t.Fatalf("ingest = dirty %t at %s; want dirty true at %s", res.Dirty, res.ReviewOfCommit, head)
			}
			body, err := os.ReadFile(filepath.Join(root, res.ReportPath))
			if err != nil {
				t.Fatal(err)
			}
			s := string(body)
			if !strings.Contains(s, "---\nreview_of_commit: "+head+"\ndirty: true\n---\n") {
				t.Errorf("report is not marked dirty at %s:\n%s", head, s)
			}
			for _, w := range tc.want {
				if !strings.Contains(s, "`"+w+"`") {
					t.Errorf("report does not name dirty path %s:\n%s", w, s)
				}
			}
		})
	}
}
