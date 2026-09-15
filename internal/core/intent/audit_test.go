package intent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
)

const reviewsDir = ".abcd/.work.local/reviews"

// validVerdict builds a schema-valid single-criterion (ac-1) verdict JSON for
// the given receipt id.
func validVerdict(receiptID string) string {
	v := map[string]any{
		"_type":      "abcd/intent-fidelity-verdict/v1",
		"receipt_id": receiptID,
		// The verifier id deliberately keeps the PRE-rename agent name: stored
		// verdicts and their markers are format-frozen across the spc-28 rename,
		// so this fixture doubles as the stays-valid assertion.
		"verifier": map[string]any{"id": "intent-fidelity-reviewer", "version": "claude-opus-4-8"},
		"policy":   map[string]any{"rubric_hash": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "prompt_hash": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		"input_attestations": []any{
			map[string]any{"kind": "diff", "ref": "main..auto/x", "digest": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
		},
		"criteria": []any{
			map[string]any{
				"criterion_id": "ac-1",
				"verdict":      "MET",
				"rationale":    "the ship-move writes the OWED stub and request file",
				"evidence": []any{
					map[string]any{"ref": "internal/core/intent/review.go:42", "quote": "func emitAuditForIntent("},
				},
			},
		},
		"acceptance_rollup": map[string]any{"MET": 1, "MET_WITH_CONCERNS": 0, "NOT_MET": 0, "INCONCLUSIVE": 0},
		"gap_audit": map[string]any{
			"honoured": []any{
				map[string]any{"claim": "OWED stub emitted at ship", "evidence": []any{map[string]any{"ref": "internal/core/intent/review.go:50", "quote": "OWED"}}},
			},
			"diverged": []any{},
			"missing":  []any{},
		},
	}
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

// writeVerdict writes a verdict payload to a scratch file and returns its path,
// substituting validVerdict's PLACEHOLDER policy hashes for the pair the host
// actually issued for the payload's receipt (iss-2609100505140261).
//
// The substitution lives in the shared fixture writer rather than at each call
// site because a real verdict cannot carry anything else: the ingest recomputes
// both hashes and refuses an echo it did not issue. Keeping the invented
// `sha256:aa…`/`sha256:bb…` values in every fixture is what let the ingest's
// shape-only check read as verified for so long, so the fixture writer is the
// one place that can stop a future test reintroducing them by accident.
//
// It is keyed on the exact placeholder strings, so a test that deliberately
// plants a malformed, empty or foreign hash keeps the value it planted —
// writeVerdictRaw is the explicit form for that.
func writeVerdict(t *testing.T, root, payload string) string {
	t.Helper()
	if p, ok := issuedPolicy(t, root, receiptIDOf(payload)); ok {
		payload = strings.Replace(payload, placeholderRubricHash, p.RubricHash, 1)
		payload = strings.Replace(payload, placeholderPromptHash, p.PromptHash, 1)
	}
	return writeVerdictRaw(t, root, payload)
}

// writeVerdictRaw writes a payload verbatim — the form for a test whose point IS
// the policy hashes it planted.
func writeVerdictRaw(t *testing.T, root, payload string) string {
	t.Helper()
	p := filepath.Join(root, "verdict.json")
	if err := os.WriteFile(p, []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

const (
	placeholderRubricHash = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	placeholderPromptHash = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

// receiptIDOf recovers the receipt id a payload keys on, best-effort: a fixture
// that is not parseable JSON has no receipt to resolve and needs no substitution.
func receiptIDOf(payload string) string {
	var lenient struct {
		ReceiptID string `json:"receipt_id"`
	}
	if err := json.Unmarshal([]byte(payload), &lenient); err != nil {
		return ""
	}
	return lenient.ReceiptID
}

// issuedPolicy returns the provenance the host issues for rcp in this repo,
// resolved the same way the ingest resolves it: the intent carrying the marker.
// ok is false when no intent carries it (the unsolicited-receipt fixtures).
func issuedPolicy(t *testing.T, root, rcp string) (auditPolicy, bool) {
	t.Helper()
	if rcp == "" {
		return auditPolicy{}, false
	}
	corpus, err := Load(root)
	if err != nil {
		return auditPolicy{}, false
	}
	for _, it := range corpus.Intents {
		data, err := os.ReadFile(filepath.Join(root, it.Path))
		if err != nil {
			continue
		}
		if _, ok := markerState(string(data), rcp); ok {
			realised, err := deliveredSpecs(root, it)
			if err != nil {
				t.Fatal(err)
			}
			return auditPolicyFor(it, rcp, string(data), realised), true
		}
	}
	return auditPolicy{}, false
}

// shipOne reconciles a fresh planned intent/spec pair and returns the receipt id
// the ship-move emitted.
func shipOne(t *testing.T, root string) string {
	t.Helper()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))
	res, err := Reconcile(root, "spc-1", "", RemainderRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if res.ReceiptID == "" {
		t.Fatal("Reconcile must emit a receipt id at the ship move")
	}
	return res.ReceiptID
}

// TestReconcileEmitsOwedStubAndRequest proves the ship move parks an OWED marker
// in the shipped intent's Audit Notes and writes the ephemeral request file.
func TestReconcileEmitsOwedStubAndRequest(t *testing.T) {
	root := t.TempDir()
	rcp := shipOne(t, root)

	body, err := os.ReadFile(filepath.Join(root, shippedDir, "itd-10-alpha.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	if !strings.Contains(s, "<!-- abcd-review: OWED receipt="+rcp+" -->") {
		t.Fatalf("shipped intent missing OWED marker for %s:\n%s", rcp, s)
	}
	if !strings.Contains(s, "Fidelity review OWED") {
		t.Fatalf("shipped intent missing human OWED line:\n%s", s)
	}
	reqPath := filepath.Join(root, reviewsDir, rcp+".request.md")
	rb, err := os.ReadFile(reqPath)
	if err != nil {
		t.Fatalf("request file should exist at %s: %v", reqPath, err)
	}
	if !strings.Contains(string(rb), rcp) || !strings.Contains(string(rb), "Acceptance Criteria") {
		t.Fatalf("request file missing receipt/AC:\n%s", rb)
	}
}

// TestReconcileEmitDeterministicReceipt proves the receipt id is stable across a
// re-run (idempotent emit does not fork the receipt or duplicate the stub).
func TestReconcileEmitDeterministicReceipt(t *testing.T) {
	root := t.TempDir()
	rcp := shipOne(t, root)
	// Re-run reconcile (idempotent): same receipt, single OWED marker.
	res, err := Reconcile(root, "spc-1", "", RemainderRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if res.ReceiptID != rcp {
		t.Fatalf("receipt id changed on re-run: %q != %q", res.ReceiptID, rcp)
	}
	body, _ := os.ReadFile(filepath.Join(root, shippedDir, "itd-10-alpha.md"))
	if n := strings.Count(string(body), "abcd-review: OWED"); n != 1 {
		t.Fatalf("expected exactly 1 OWED marker after re-run, got %d:\n%s", n, body)
	}
}

// TestIngestHappyPath ingests a valid verdict: the OWED stub flips to INGESTED,
// per-criterion verdicts and the gap audit render with cited evidence.
func TestIngestHappyPath(t *testing.T) {
	root := t.TempDir()
	rcp := shipOne(t, root)
	vp := writeVerdict(t, root, validVerdict(rcp))

	res, err := IngestVerdict(root, vp)
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if res.Status != "ingested" || res.ReceiptID != rcp || res.IntentID != "itd-10" {
		t.Fatalf("ingest result = %+v", res)
	}
	body, _ := os.ReadFile(filepath.Join(root, shippedDir, "itd-10-alpha.md"))
	s := string(body)
	if !strings.Contains(s, "<!-- abcd-review: INGESTED receipt="+rcp+" -->") {
		t.Fatalf("intent missing INGESTED marker:\n%s", s)
	}
	if strings.Contains(s, "abcd-review: OWED") {
		t.Fatalf("OWED stub should be gone after ingest:\n%s", s)
	}
	if !strings.Contains(s, "ac-1") || !strings.Contains(s, "MET") {
		t.Fatalf("per-criterion verdict not rendered:\n%s", s)
	}
	if !strings.Contains(s, "internal/core/intent/review.go:42") {
		t.Fatalf("cited evidence not rendered:\n%s", s)
	}
	if !strings.Contains(s, "OWED stub emitted at ship") {
		t.Fatalf("gap-audit honoured claim not rendered:\n%s", s)
	}
	// The pinned provenance (policy hashes + input-attestation digest) is rendered
	// — and it is the pair the HOST issued, not a value the payload chose. The
	// assertion used to pin the placeholder, which is precisely why nothing
	// noticed that the ingest never checked it (iss-2609100505140261).
	issued, ok := issuedPolicy(t, root, rcp)
	if !ok {
		t.Fatal("no host-issued provenance for the parked receipt")
	}
	if !strings.Contains(s, "Provenance:") || !strings.Contains(s, issued.RubricHash) {
		t.Fatalf("provenance line (verifier + host-issued rubric_hash %s) not rendered:\n%s", issued.RubricHash, s)
	}
	if !strings.Contains(s, "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc") {
		t.Fatalf("input-attestation digest not rendered:\n%s", s)
	}
}

// TestIngestIdempotentNoOp proves re-ingesting the same verdict is a no-op.
func TestIngestIdempotentNoOp(t *testing.T) {
	root := t.TempDir()
	rcp := shipOne(t, root)
	vp := writeVerdict(t, root, validVerdict(rcp))
	if _, err := IngestVerdict(root, vp); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(filepath.Join(root, shippedDir, "itd-10-alpha.md"))
	res, err := IngestVerdict(root, vp)
	if err != nil {
		t.Fatalf("second ingest must not error: %v", err)
	}
	if res.Status != "noop" {
		t.Fatalf("second ingest status = %q, want noop", res.Status)
	}
	after, _ := os.ReadFile(filepath.Join(root, shippedDir, "itd-10-alpha.md"))
	if string(before) != string(after) {
		t.Fatalf("no-op re-ingest mutated the intent:\nBEFORE\n%s\nAFTER\n%s", before, after)
	}
	if n := strings.Count(string(after), "abcd-review: INGESTED"); n != 1 {
		t.Fatalf("expected 1 INGESTED marker, got %d", n)
	}
}

// TestIngestUnsolicitedVerdictRejected rejects a verdict whose receipt matches no
// parked OWED marker (nothing is written).
func TestIngestUnsolicitedVerdictRejected(t *testing.T) {
	root := t.TempDir()
	shipOne(t, root)
	vp := writeVerdict(t, root, validVerdict("rcp-000000000000"))
	if _, err := IngestVerdict(root, vp); err == nil {
		t.Fatal("ingest must reject an unsolicited verdict (no matching OWED receipt)")
	}
	// The shipped intent still shows OWED, untouched.
	body, _ := os.ReadFile(filepath.Join(root, shippedDir, "itd-10-alpha.md"))
	if !strings.Contains(string(body), "abcd-review: OWED") {
		t.Fatalf("unsolicited rejection must not disturb the OWED stub:\n%s", body)
	}
}

// TestIngestUnknownCriterionDeadLetters dead-letters a verdict citing a criterion
// the intent does not have (ac-9 for a single-bullet intent).
func TestIngestUnknownCriterionDeadLetters(t *testing.T) {
	root := t.TempDir()
	rcp := shipOne(t, root)
	payload := strings.Replace(validVerdict(rcp), `"criterion_id": "ac-1"`, `"criterion_id": "ac-9"`, 1)
	vp := writeVerdict(t, root, payload)

	res, err := IngestVerdict(root, vp)
	if err != nil {
		t.Fatalf("dead-letter path must not error: %v", err)
	}
	if res.Status != "dead_letter" {
		t.Fatalf("status = %q, want dead_letter", res.Status)
	}
	assertDeadLetter(t, root, rcp)
}

// TestIngestOutOfEnumVerdictDeadLetters dead-letters an out-of-enum verdict token.
func TestIngestOutOfEnumVerdictDeadLetters(t *testing.T) {
	root := t.TempDir()
	rcp := shipOne(t, root)
	payload := strings.Replace(validVerdict(rcp), `"verdict": "MET"`, `"verdict": "SHIP"`, 1)
	vp := writeVerdict(t, root, payload)

	res, err := IngestVerdict(root, vp)
	if err != nil {
		t.Fatalf("dead-letter path must not error: %v", err)
	}
	if res.Status != "dead_letter" {
		t.Fatalf("status = %q, want dead_letter", res.Status)
	}
	assertDeadLetter(t, root, rcp)
}

// TestIngestMalformedBodyDeadLetters dead-letters a resolvable-but-malformed
// payload (criteria is not an array), retaining the raw payload.
func TestIngestMalformedBodyDeadLetters(t *testing.T) {
	root := t.TempDir()
	rcp := shipOne(t, root)
	payload := `{"_type":"abcd/intent-fidelity-verdict/v1","receipt_id":"` + rcp + `","criteria":"not-an-array"}`
	vp := writeVerdict(t, root, payload)

	res, err := IngestVerdict(root, vp)
	if err != nil {
		t.Fatalf("dead-letter path must not error: %v", err)
	}
	if res.Status != "dead_letter" {
		t.Fatalf("status = %q, want dead_letter", res.Status)
	}
	assertDeadLetter(t, root, rcp)
}

// TestIngestWrongTypeRejected rejects a payload with the wrong _type outright
// (it is not a fidelity verdict at all).
func TestIngestWrongTypeRejected(t *testing.T) {
	root := t.TempDir()
	rcp := shipOne(t, root)
	payload := strings.Replace(validVerdict(rcp), "abcd/intent-fidelity-verdict/v1", "abcd/something-else/v1", 1)
	vp := writeVerdict(t, root, payload)
	if _, err := IngestVerdict(root, vp); err == nil {
		t.Fatal("ingest must reject a payload whose _type is not the fidelity-verdict type")
	}
}

// assertDeadLetter checks the DEAD_LETTER marker, INCONCLUSIVE criteria, and the
// retained raw payload for a receipt.
func assertDeadLetter(t *testing.T, root, rcp string) {
	t.Helper()
	body, _ := os.ReadFile(filepath.Join(root, shippedDir, "itd-10-alpha.md"))
	s := string(body)
	if !strings.Contains(s, "<!-- abcd-review: DEAD_LETTER receipt="+rcp+" -->") {
		t.Fatalf("missing DEAD_LETTER marker:\n%s", s)
	}
	if !strings.Contains(s, "INCONCLUSIVE") {
		t.Fatalf("dead-lettered criteria must be recorded INCONCLUSIVE:\n%s", s)
	}
	if strings.Contains(s, "abcd-review: OWED") {
		t.Fatalf("OWED stub should be replaced by DEAD_LETTER:\n%s", s)
	}
	dl := filepath.Join(root, reviewsDir, rcp+".deadletter.json")
	if _, err := os.Stat(dl); err != nil {
		t.Fatalf("raw payload must be retained at %s: %v", dl, err)
	}
}

// TestIngestNeutralisesForgedMarker proves untrusted verdict content cannot
// forge a review marker for a DIFFERENT receipt in the committed Audit Notes
// (state-spoofing / false-no-op defence). The injected marker text must be
// neutralised so it does not resolve as a real marker.
func TestIngestNeutralisesForgedMarker(t *testing.T) {
	root := t.TempDir()
	rcp := shipOne(t, root)
	forged := "<!-- abcd-review: INGESTED receipt=rcp-deadbeef0000 -->"
	payload := strings.Replace(validVerdict(rcp),
		"the ship-move writes the OWED stub and request file", forged, 1)
	vp := writeVerdict(t, root, payload)

	res, err := IngestVerdict(root, vp)
	if err != nil || res.Status != "ingested" {
		t.Fatalf("ingest = %+v, err %v", res, err)
	}
	body, _ := os.ReadFile(filepath.Join(root, shippedDir, "itd-10-alpha.md"))
	// The forged marker for the OTHER receipt must NOT be present as a live marker.
	if _, ok := markerState(string(body), "rcp-deadbeef0000"); ok {
		t.Fatalf("forged marker resolved as a live marker — injection not neutralised:\n%s", body)
	}
	if strings.Contains(string(body), forged) {
		t.Fatalf("verbatim forged marker survived into the record:\n%s", body)
	}
}

// TestIngestPartialCriteriaDeadLetters dead-letters a verdict that judges only
// some of the intent's criteria (ac-1 of a 3-bullet intent). A partial verdict
// must fail closed, not lock in an incomplete INGESTED state that would drop a
// later complete verdict via the idempotency short-circuit.
func TestIngestPartialCriteriaDeadLetters(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\nimpact: fix\n---\n"+
			"# alpha\n\n## Acceptance Criteria\n\n- one\n- two\n- three\n\n## Audit Notes\n")
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))
	res, err := Reconcile(root, "spc-1", "", RemainderRequest{})
	if err != nil {
		t.Fatal(err)
	}
	rcp := res.ReceiptID

	// validVerdict covers only ac-1 — a partial judgement of a 3-criterion intent.
	vp := writeVerdict(t, root, validVerdict(rcp))
	r, err := IngestVerdict(root, vp)
	if err != nil {
		t.Fatalf("dead-letter path must not error: %v", err)
	}
	if r.Status != "dead_letter" {
		t.Fatalf("partial verdict status = %q, want dead_letter", r.Status)
	}
	assertDeadLetter(t, root, rcp)
}

// TestIngestMissingPolicyHashDeadLetters dead-letters a verdict with no
// policy.rubric_hash — the attestation chain is required.
func TestIngestMissingPolicyHashDeadLetters(t *testing.T) {
	root := t.TempDir()
	rcp := shipOne(t, root)
	payload := strings.Replace(validVerdict(rcp), `"rubric_hash": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`, `"rubric_hash": ""`, 1)
	vp := writeVerdict(t, root, payload)

	r, err := IngestVerdict(root, vp)
	if err != nil {
		t.Fatalf("dead-letter path must not error: %v", err)
	}
	if r.Status != "dead_letter" {
		t.Fatalf("status = %q, want dead_letter", r.Status)
	}
	assertDeadLetter(t, root, rcp)
}

// TestReconcileEmitIdempotent proves a second emit on an already-parked intent
// neither forks the receipt nor appends a second OWED stub — even when the
// intent had no `## Audit Notes` and its Acceptance Criteria was the last
// section (the case where a naive receipt recompute would shift). Extends
// TestReconcileEmitDeterministicReceipt.
func TestReconcileEmitIdempotent(t *testing.T) {
	root := t.TempDir()
	// Acceptance Criteria is the file's last section (no `## Audit Notes`), with a
	// trailing blank line — creating the section on emit shifts the AC body a naive
	// recompute would digest.
	writeFile(t, root, shippedDir+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n---\n"+
			"# alpha\n\n## Acceptance Criteria\n\n- ok\n\n")

	c1, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	it, ok := c1.Lookup("itd-10")
	if !ok {
		t.Fatal("shipped intent not loaded")
	}
	r1, err := emitAuditForIntent(root, it)
	if err != nil {
		t.Fatalf("first emit: %v", err)
	}

	// Re-load the mutated content and re-emit.
	c2, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	it2, _ := c2.Lookup("itd-10")
	r2, err := emitAuditForIntent(root, it2)
	if err != nil {
		t.Fatalf("second emit: %v", err)
	}
	if r2.ReceiptID != r1.ReceiptID {
		t.Fatalf("receipt shifted across re-emit: %q != %q", r2.ReceiptID, r1.ReceiptID)
	}
	if r2.Status != "already_owed" {
		t.Fatalf("second emit status = %q, want already_owed", r2.Status)
	}
	body, _ := os.ReadFile(filepath.Join(root, shippedDir, "itd-10-alpha.md"))
	if n := strings.Count(string(body), "abcd-review: OWED"); n != 1 {
		t.Fatalf("expected exactly 1 OWED marker after re-emit, got %d:\n%s", n, body)
	}
}

// TestFullReviewCycle drives drafts -> plan -> close(reconcile -> OWED) ->
// ingest, asserting the shipped intent lands with a populated Audit Notes and the
// lifecycle lint rules stay green throughout.
func TestFullReviewCycle(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))

	pr, err := Plan(root, "itd-10", "")
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	// The seeded draft declares no impact, so the close supplies the judgement —
	// the drafts -> shipped path a real intent takes when the seed deferred it.
	rr, err := Reconcile(root, pr.Spec.ID, "fix", RemainderRequest{})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	rcp := rr.ReceiptID
	if rcp == "" {
		t.Fatal("no receipt id emitted")
	}

	vp := writeVerdict(t, root, validVerdict(rcp))
	res, err := IngestVerdict(root, vp)
	if err != nil {
		t.Fatalf("IngestVerdict: %v", err)
	}
	if res.Status != "ingested" {
		t.Fatalf("cycle ingest status = %q", res.Status)
	}

	body, _ := os.ReadFile(filepath.Join(root, shippedDir, "itd-10-alpha.md"))
	s := string(body)
	if !strings.Contains(s, "## Audit Notes") || !strings.Contains(s, "INGESTED receipt="+rcp) {
		t.Fatalf("shipped intent Audit Notes not populated:\n%s", s)
	}

	cfg := lint.Config{
		Roots: []string{".abcd/development"},
		Rules: map[string]lint.RuleConfig{
			"intent_lifecycle": {Enabled: true, Severity: "blocker", IntentsDir: "intents"},
			"spec_lifecycle":   {Enabled: true, Severity: "blocker", SpecsDir: "specs", IntentsDir: "intents"},
		},
	}
	findings, err := lint.Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		if f.RuleID == "intent_lifecycle" || f.RuleID == "spec_lifecycle" {
			t.Fatalf("post-cycle lifecycle finding: %s:%d %s %s", f.File, f.Line, f.RuleID, f.Message)
		}
	}
}

// TestFirstReviewBlockClearsPlaceholder (iss-67 seed3) proves the intent
// template's Audit Notes placeholder ("_Empty. Populated by ..._") is dropped when
// the first review block lands, so a populated audit carries no stale "Empty"
// claim above the real verdict.
func TestFirstReviewBlockClearsPlaceholder(t *testing.T) {
	// Both template placeholder styles the record has used must be cleared.
	for _, placeholder := range []string{
		"_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._",
		"<Empty until intent moves to shipped/. intent-fidelity-reviewer populates this.>",
	} {
		content := "---\nid: itd-9\n---\n# alpha\n\n## Acceptance Criteria\n\n- one\n\n" +
			"## Audit Notes\n\n" + placeholder + "\n"
		out := upsertReviewBlock(content, "rcp-abc123", owedBlock("rcp-abc123"))
		if strings.Contains(out, "Empty until") || strings.Contains(out, "_Empty. Populated by") {
			t.Fatalf("the placeholder %q must be cleared once a review block lands:\n%s", placeholder, out)
		}
		if !strings.Contains(out, "OWED receipt=rcp-abc123") {
			t.Fatalf("the OWED block must be present:\n%s", out)
		}
	}
}

// TestIngestVerdictRefusesSymlink is the S6 regression: the verdict operand read
// goes through fsutil.ReadGuarded (O_NOFOLLOW), so a symlinked verdict path is
// refused rather than followed. The prior Lstat-then-os.ReadFile form left a
// window in which a symlink swapped in after the Lstat was followed by ReadFile.
func TestIngestVerdictRefusesSymlink(t *testing.T) {
	root := t.TempDir()
	rcp := shipOne(t, root)
	real := writeVerdict(t, root, validVerdict(rcp))
	link := filepath.Join(root, "verdict-link.json")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	_, err := IngestVerdict(root, link)
	if err == nil {
		t.Fatal("expected a symlinked verdict path to be refused, got nil error")
	}
	if !strings.Contains(err.Error(), "not a regular file") {
		t.Errorf("unexpected error for symlinked verdict: %v", err)
	}
}

// TestReEmitAuditToleratesSpecIDSpelling proves the manual re-emit verb accepts
// the same lint-green spec_id spellings the ship move does.
func TestReEmitAuditToleratesSpecIDSpelling(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, shippedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1-alpha"))

	res, err := ReEmitAudit(root, "itd-10")
	if err != nil {
		t.Fatalf("ReEmitAudit must accept a slug-suffixed spec_id: %v", err)
	}
	if res.ReceiptID == "" || res.Status != "owed" {
		t.Fatalf("ReEmitAudit result = %+v, want an owed receipt", res)
	}
}

// TestIngestNeutralisesLinkSyntax (iss-2608311504353427) proves a verdict that
// faithfully quotes code of the shape `items[0](itm-0001)` — a bracket followed
// immediately by a parenthesis — cannot write a live markdown link into the
// shipped intent's Audit Notes. Before the fix the record-lint links_resolve
// gate refused the whole tree on the record the ingest had just written, because
// the link's target resolves to nothing. The neutralised text must stay legible
// (the quoted code is still readable) and the gate must pass over the result.
func TestIngestNeutralisesLinkSyntax(t *testing.T) {
	root := t.TempDir()
	rcp := shipOne(t, root)
	const quoted = "items[0](itm-0001)"
	payload := strings.Replace(validVerdict(rcp),
		"the ship-move writes the OWED stub and request file",
		"the element path "+quoted+" is quoted verbatim, as is [text][label] and <https://example.invalid/x>", 1)
	payload = strings.Replace(payload, "func emitAuditForIntent(", quoted, 1)
	payload = strings.Replace(payload, "OWED stub emitted at ship", "the path "+quoted+" resolves", 1)
	vp := writeVerdict(t, root, payload)

	res, err := IngestVerdict(root, vp)
	if err != nil || res.Status != "ingested" {
		t.Fatalf("ingest = %+v, err %v", res, err)
	}
	body, _ := os.ReadFile(filepath.Join(root, shippedDir, "itd-10-alpha.md"))
	s := string(body)
	if strings.Contains(s, "](") || strings.Contains(s, "][") || strings.Contains(s, "<https://") {
		t.Fatalf("live link syntax survived into the record:\n%s", s)
	}
	// Legibility: the quoted code is still readable as the code it quotes.
	if !strings.Contains(s, "items[0]") || !strings.Contains(s, "(itm-0001)") {
		t.Fatalf("neutralisation destroyed the quoted code:\n%s", s)
	}

	cfg := lint.Config{
		Roots: []string{".abcd/development"},
		Rules: map[string]lint.RuleConfig{
			"links_resolve": {Enabled: true, Severity: "blocker"},
		},
	}
	findings, err := lint.Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		if f.RuleID == "links_resolve" {
			t.Fatalf("links_resolve finding on the ingested record: %s:%d %s\n%s", f.File, f.Line, f.Message, s)
		}
	}
}
