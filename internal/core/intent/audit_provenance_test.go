package intent

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// iss-2609100505140261 — the two halves of `intent audit` disagreed about the
// policy hashes: the emit handed the auditor a request carrying neither, and the
// ingest refused a verdict whose hashes were empty. Thirteen auditors each
// invented a value to satisfy the non-empty rule, and the ingest shape-checked
// them and accepted every one — a required attestation with no supported way to
// fill it, which is the false-green shape.

var provenanceLineRe = regexp.MustCompile(`(?m)^- (rubric_hash|prompt_hash): (sha256:[0-9a-f]{64})$`)

// TestAuditRequestCarriesHostIssuedPolicyHashes is the emit half: the request
// the host hands the auditor must state both hashes, so the auditor echoes a
// value the host computed rather than one it chose.
func TestAuditRequestCarriesHostIssuedPolicyHashes(t *testing.T) {
	root := t.TempDir()
	rcp := shipOne(t, root)

	rb, err := os.ReadFile(filepath.Join(root, reviewsDir, rcp+".request.md"))
	if err != nil {
		t.Fatal(err)
	}
	req := string(rb)
	got := map[string]string{}
	for _, m := range provenanceLineRe.FindAllStringSubmatch(req, -1) {
		got[m[1]] = m[2]
	}
	if got["rubric_hash"] == "" {
		t.Fatalf("request states no host-computed rubric_hash; the auditor has nothing to echo:\n%s", req)
	}
	if got["prompt_hash"] == "" {
		t.Fatalf("request states no host-computed prompt_hash; the auditor has nothing to echo:\n%s", req)
	}
	if got["rubric_hash"] == got["prompt_hash"] {
		t.Fatalf("rubric_hash and prompt_hash are the same value, so neither distinguishes what it attests: %s", got["rubric_hash"])
	}
	// The rubric the hashes attest to must be IN the request: a hash over text
	// the auditor never saw attests nothing to the auditor.
	if !strings.Contains(req, rubricText()) {
		t.Fatalf("request does not carry the rubric its rubric_hash is computed over:\n%s", req)
	}
}

// TestIngestRefusesPolicyHashesTheHostDidNotIssue is the ingest half: a verdict
// whose hashes are well-SHAPED but are not the values this receipt's request
// issued is refused. Shape-checking alone let six verdicts across two
// repositories write self-issued provenance into permanent Audit Notes.
func TestIngestRefusesPolicyHashesTheHostDidNotIssue(t *testing.T) {
	root := t.TempDir()
	rcp := shipOne(t, root)
	// validVerdict's placeholder hashes are the very shape the auditors invented:
	// 64 lowercase hex that passes sha256FieldRe and matches nothing. Written RAW,
	// so the fixture writer's host-issued substitution does not repair them.
	vp := writeVerdictRaw(t, root, validVerdict(rcp))

	_, err := IngestVerdict(root, vp)
	if err == nil {
		t.Fatal("ingest accepted policy hashes the host never issued; the attestation attests nothing")
	}
	for _, want := range []string{"rubric_hash", "issued"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("refusal does not name %q, so the writer cannot tell what the host expects to be hashed: %v", want, err)
		}
	}
	// Refused outright, not dead-lettered: the OWED marker survives so a
	// re-emit + re-audit is still possible.
	body, _ := os.ReadFile(filepath.Join(root, shippedDir, "itd-10-alpha.md"))
	if !strings.Contains(string(body), "abcd-review: OWED receipt="+rcp) {
		t.Fatalf("a mismatched attestation must leave the OWED receipt parked:\n%s", body)
	}
}

// TestIngestAcceptsPolicyHashesTheRequestIssued closes the loop: the values the
// emit wrote into the request are exactly the values the ingest accepts, so the
// verb is completable without inventing anything.
func TestIngestAcceptsPolicyHashesTheRequestIssued(t *testing.T) {
	root := t.TempDir()
	rcp := shipOne(t, root)

	rb, err := os.ReadFile(filepath.Join(root, reviewsDir, rcp+".request.md"))
	if err != nil {
		t.Fatal(err)
	}
	issued := map[string]string{}
	for _, m := range provenanceLineRe.FindAllStringSubmatch(string(rb), -1) {
		issued[m[1]] = m[2]
	}
	if issued["rubric_hash"] == "" || issued["prompt_hash"] == "" {
		t.Fatalf("emit issued no hashes to echo:\n%s", rb)
	}

	payload := validVerdict(rcp)
	payload = strings.Replace(payload,
		"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		issued["rubric_hash"], 1)
	payload = strings.Replace(payload,
		"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		issued["prompt_hash"], 1)

	res, err := IngestVerdict(root, writeVerdictRaw(t, root, payload))
	if err != nil {
		t.Fatalf("ingest refused the very hashes its own request issued: %v", err)
	}
	if res.Status != "ingested" {
		t.Fatalf("status = %q, want ingested (%+v)", res.Status, res)
	}
	body, _ := os.ReadFile(filepath.Join(root, shippedDir, "itd-10-alpha.md"))
	if !strings.Contains(string(body), issued["rubric_hash"]) {
		t.Fatalf("the ingested Audit Note does not carry the host-issued rubric_hash:\n%s", body)
	}
}
