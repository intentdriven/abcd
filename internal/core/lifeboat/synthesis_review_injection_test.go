package lifeboat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// synthesis_review_injection_test.go covers GH #325: untrusted provenance fields
// (source_name, manifest_sha256) come verbatim from the reviewed lifeboat's
// _provenance.json and are stamped into the DURABLE review .md. They were cleaned
// with sanitize (terminal-escape defanging), which leaves a CommonMark HTML-block
// opener ('<script', '<!--', …) intact — so a hostile lifeboat could hide or forge
// the attestation/findings when the record renders. The fix routes them through
// the file-write cleaner (termsafe.CleanProse via cleanSynthProse), as the
// delegated findings in the same file already are.

// assertNoRawHTMLOpener fails if the written markdown carries a raw CommonMark
// HTML-block opener from the injected marker, and confirms the neutralised form
// ("< script") is present instead.
func assertNoRawHTMLOpener(t *testing.T, md string) {
	t.Helper()
	if strings.Contains(md, "<script") {
		t.Fatalf("review .md carries a raw <script HTML-block opener from an untrusted provenance field:\n%s", md)
	}
	if !strings.Contains(md, "< script") {
		t.Fatalf("expected the neutralised '< script' form in the review .md, not found:\n%s", md)
	}
}

// TestReviewMDNeutralisesInjectedSourceName is the fnd-source-name vector: a
// source_name that differs from the audited source basename fires the
// deterministic identity finding, whose prose embeds the raw provenance value.
// The lifeboat stays sealed, so the verdict is still SHIP — the injection lands at
// exit 0.
func TestReviewMDNeutralisesInjectedSourceName(t *testing.T) {
	dir := reviewFixture(t, "<script>evil", &Summary{Grounded: 7, Partial: 4, Blank: 3})
	res, err := ReviewLifeboat(dir, realSourceDir(t), nil, okScan) // source basename "src" != "<script>evil"
	if err != nil {
		t.Fatalf("ReviewLifeboat: %v", err)
	}
	if res.Verdict != VerdictShip {
		t.Fatalf("verdict = %q, want SHIP (sealed lifeboat, injection via source_name)", res.Verdict)
	}
	md, err := os.ReadFile(filepath.Join(dir, res.RenderPath))
	if err != nil {
		t.Fatalf("read review .md: %v", err)
	}
	assertNoRawHTMLOpener(t, string(md))
}

// TestReviewMDNeutralisesInjectedManifestSHA is the attestation-header vector: the
// manifest line embeds the raw manifest_sha256. Overwriting it flips
// manifestVerified to false (MAJOR_RETHINK), but the review is still written, so
// the opener still lands in the durable record.
func TestReviewMDNeutralisesInjectedManifestSHA(t *testing.T) {
	dir := reviewFixture(t, "abc", &Summary{Grounded: 7, Partial: 4, Blank: 3})
	// Overwrite the sealed header with a hostile manifest_sha256 (breaks the seal,
	// but the review is written regardless).
	prov := `{"schema_version":2,"generator":"test","source_name":"abc","manifest_sha256":"<script>deadbeef"}`
	writeFile(t, filepath.Join(dir, ProvenanceName), []byte(prov))

	res, err := ReviewLifeboat(dir, realSourceDir(t), nil, okScan)
	if err != nil {
		t.Fatalf("ReviewLifeboat: %v", err)
	}
	md, err := os.ReadFile(filepath.Join(dir, res.RenderPath))
	if err != nil {
		t.Fatalf("read review .md: %v", err)
	}
	assertNoRawHTMLOpener(t, string(md))
}
