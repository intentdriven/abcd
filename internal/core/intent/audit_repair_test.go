// Package intent_test holds the audit-render assertions that must be judged by a
// real markdown reader. internal/core/site imports internal/core/intent, so the
// renderer can only be reached from the external test package.
package intent_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/site"
)

const shippedIntentsDir = ".abcd/development/intents/shipped"

func writeAt(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func owedIntent(id, slug, specID, rcp string) string {
	return "---\nid: " + id + "\nslug: " + slug + "\nspec_id: " + specID + "\nkind: standalone\n---\n" +
		"# " + slug + "\n\n## Scope Conditions\n\n" + intent.NullityToken +
		"\n\n## Acceptance Criteria\n\n- ok\n" +
		"\n## Grounds\n\n- pursued: we expect the recorded conjecture to outlive the session that had it\n" +
		"\n## Audit Notes\n\n<!-- abcd-review: OWED receipt=" + rcp + " -->\nFidelity review OWED.\n"
}

// verdictWithAttestation builds a schema-valid verdict whose ONE input
// attestation carries the caller's three untrusted fields — the three the
// ingested block renders adjacent on a single line as `%s:%s@%s`.
func verdictWithAttestation(rcp, kind, ref, digest string) string {
	v := map[string]any{
		"_type":      "abcd/intent-fidelity-verdict/v1",
		"receipt_id": rcp,
		"verifier":   map[string]any{"id": "intent-fidelity-reviewer", "version": "claude-opus-4-8"},
		"policy":     map[string]any{"rubric_hash": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "prompt_hash": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		"input_attestations": []any{
			map[string]any{"kind": kind, "ref": ref, "digest": digest},
		},
		"criteria": []any{
			map[string]any{
				"criterion_id": "ac-1",
				"verdict":      "MET",
				"rationale":    "the delivered code matches the criterion",
				"evidence": []any{
					map[string]any{"ref": "internal/core/intent/audit.go:1", "quote": "func IngestVerdict("},
				},
			},
		},
		"acceptance_rollup": map[string]any{"MET": 1, "MET_WITH_CONCERNS": 0, "NOT_MET": 0, "INCONCLUSIVE": 0},
		"gap_audit": map[string]any{
			"honoured": []any{
				map[string]any{"claim": "the marker is parked", "evidence": []any{map[string]any{"ref": "internal/core/intent/audit.go:2", "quote": "OWED"}}},
			},
			"diverged": []any{},
			"missing":  []any{},
		},
	}
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

// renderLine renders one line of the committed record through the site renderer,
// which refuses inline HTML outright — the strictest reader the record has.
func renderLine(md string) (string, error) {
	r := &site.Renderer{
		UI:    site.UI{Copy: "copy", Copied: "copied"},
		Image: func(src, alt string, _ site.Source) (string, error) { return "", nil },
		Link:  func(href string, _ site.Source) string { return href },
	}
	return r.RenderBlocks("record.md", site.Blocks(md, 1))
}

// TestIngestedAttestationLineCannotRePairACodeSpan pins the second half of the
// invariant the code-span exemption rests on: a cleaned field is parsed as
// CommonMark as the exact string it was CLEANED as.
//
// Each untrusted field goes through the cleaner ALONE, but the ingested block
// renders three of them adjacent on one line (`%s:%s@%s`). An unpaired backtick
// run left in the first field re-pairs with the OPENING run of a genuine span in
// the second: the span boundary moves, and the `<script>` the cleaner had judged
// sheltered lands outside a span as live inline HTML in a committed record. The
// cleaner closes it by never emitting an unpaired run.
func TestIngestedAttestationLineCannotRePairACodeSpan(t *testing.T) {
	root := t.TempDir()
	const rcp = "rcp-0123456789ab"
	writeAt(t, root, shippedIntentsDir+"/itd-10-alpha.md", owedIntent("itd-10", "alpha", "spc-1", rcp))

	payload := verdictWithAttestation(rcp, "diff`", "`<script>alert(1)`", "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
	p := filepath.Join(root, "verdict.json")
	if err := os.WriteFile(p, []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := intent.IngestVerdict(root, p); err != nil {
		t.Fatalf("ingest: %v", err)
	}

	body, err := os.ReadFile(filepath.Join(root, shippedIntentsDir, "itd-10-alpha.md"))
	if err != nil {
		t.Fatal(err)
	}
	var line string
	for _, ln := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(ln, "Input attestations:") {
			line = ln
			break
		}
	}
	if line == "" {
		t.Fatalf("no input-attestation line in the record:\n%s", body)
	}
	html, err := renderLine(line)
	if err != nil {
		t.Fatalf("the attestation line re-pairs a code span and exposes markup: %v\nline: %s", err, line)
	}
	// The positive form: the tag the payload put inside a span stays inside one.
	if !strings.Contains(html, "<code>&lt;script&gt;alert(1)</code>") {
		t.Errorf("attestation line %q rendered as %q; want the tag sheltered inside a code span", line, html)
	}
}

// TestIngestedEvidenceLineKeepsTheCleanedBytes pins the other half of the same
// invariant: no embedding may REWRITE a cleaned field. renderEvidence used %q,
// which doubles backslashes — so the cleaner's own `\“ escape came back as an
// escaped backslash followed by a LIVE backtick, putting an unpaired run into a
// committed record and reopening exactly the re-pairing hole the cleaner closes.
func TestIngestedEvidenceLineKeepsTheCleanedBytes(t *testing.T) {
	root := t.TempDir()
	const rcp = "rcp-0123456789ab"
	writeAt(t, root, shippedIntentsDir+"/itd-10-alpha.md", owedIntent("itd-10", "alpha", "spc-1", rcp))

	// One stray backtick. The cleaner escapes it; %q would have doubled the escape's
	// backslash and handed the record a live delimiter back.
	payload := verdictWithEvidenceQuote(rcp, "a stray ` backtick")
	p := filepath.Join(root, "verdict.json")
	if err := os.WriteFile(p, []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := intent.IngestVerdict(root, p); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(root, shippedIntentsDir, "itd-10-alpha.md"))
	if err != nil {
		t.Fatal(err)
	}
	var line string
	for _, ln := range strings.Split(string(body), "\n") {
		if strings.Contains(ln, "evidence:") {
			line = strings.TrimSpace(ln)
			break
		}
	}
	if line == "" {
		t.Fatalf("no evidence line in the record:\n%s", body)
	}
	html, err := renderLine(line)
	if err != nil {
		t.Fatalf("the evidence line carries an unpaired backtick run: %v\nline: %s", err, line)
	}
	// The positive form: the stray backtick reaches the reader as the literal
	// character the payload wrote, not as a delimiter.
	if !strings.Contains(html, "a stray ` backtick") {
		t.Errorf("evidence line %q rendered as %q; want the stray backtick as literal text", line, html)
	}
}

// verdictWithEvidenceQuote is a schema-valid verdict whose one criterion carries
// the caller's untrusted evidence quote.
func verdictWithEvidenceQuote(rcp, quote string) string {
	v := map[string]any{
		"_type":      "abcd/intent-fidelity-verdict/v1",
		"receipt_id": rcp,
		"verifier":   map[string]any{"id": "intent-fidelity-reviewer", "version": "claude-opus-4-8"},
		"policy":     map[string]any{"rubric_hash": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "prompt_hash": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		"criteria": []any{
			map[string]any{
				"criterion_id": "ac-1",
				"verdict":      "MET",
				"rationale":    "the delivered code matches the criterion",
				"evidence": []any{
					map[string]any{"ref": "internal/core/intent/audit.go:1", "quote": quote},
				},
			},
		},
		"acceptance_rollup": map[string]any{"MET": 1, "MET_WITH_CONCERNS": 0, "NOT_MET": 0, "INCONCLUSIVE": 0},
		"gap_audit": map[string]any{
			"honoured": []any{
				map[string]any{"claim": "the marker is parked", "evidence": []any{map[string]any{"ref": "internal/core/intent/audit.go:2", "quote": "OWED"}}},
			},
			"diverged": []any{},
			"missing":  []any{},
		},
	}
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
