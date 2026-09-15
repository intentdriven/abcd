package capture

import (
	"path/filepath"
	"strings"
	"testing"
)

// pem_identity_test.go — the issue-ledger legs of two scanner defects.
// GHSA-29jw-3jg9-qmhx: the ledger redactor masked a PEM BEGIN header and
// committed the key body, the END line, a body-prefixed filename, and — via
// promote — a body-prefixed intent. GHSA-rvhr-3455-c5jw: identity redaction
// followed the repo's effective git identity, committing the caller's other
// one. Markers are assembled from halves and bodies are repeated letters.

func pemBlock() (header, body, end string) {
	header = "-----BEGIN " + "OPENSSH PRIVATE KEY-----"
	body = strings.Repeat("Q", 64) + "\n" + strings.Repeat("R", 64)
	end = "-----END " + "OPENSSH PRIVATE KEY-----"
	return
}

// TestCaptureRedactsPEMBodyFromRecordFilenameNoteAndPromote drives a key
// block through every ledger write the advisory names: the capture body and
// the slug derived from it, a one-line PEM in a resolve note, and the intent
// promote mints from the record.
func TestCaptureRedactsPEMBodyFromRecordFilenameNoteAndPromote(t *testing.T) {
	repo, ir := ledger(t)
	t.Setenv("HOME", t.TempDir())
	header, body, end := pemBlock()
	bodyPrefix := strings.ToLower(strings.Repeat("Q", 8))

	res, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir,
		Text:     "found this key:\n" + header + "\n" + body + "\n" + end + "\nrotate it",
		Severity: SeverityMinor, Category: "security", Source: "user-observation", FoundDuring: "t",
	})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if res.Redacted == 0 {
		t.Fatalf("Capture reported no redaction")
	}
	record := readLedgerFile(t, repo, res.Path)
	for _, line := range append(strings.Split(body, "\n"), end) {
		if strings.Contains(record, line) {
			t.Errorf("record carries the key line %q…:\n%s", line[:8], record)
		}
	}
	if !strings.Contains(record, "rotate it") {
		t.Errorf("the prose after the block was lost:\n%s", record)
	}
	if strings.Contains(strings.ToLower(filepath.Base(res.Path)), bodyPrefix) {
		t.Errorf("record filename carries the key body prefix: %s", res.Path)
	}

	tr, err := Resolve(ResolveRequest{
		Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: res.ID,
		Resolution: "note: " + header + " " + strings.Repeat("Q", 64) + " rotated", Impact: "internal",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved := readLedgerFile(t, repo, tr.Path); strings.Contains(resolved, strings.Repeat("Q", 64)) {
		t.Errorf("resolution note carries the one-line key body:\n%s", resolved)
	}

	pr, err := Promote(PromoteRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: res.ID})
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if strings.Contains(strings.ToLower(filepath.Base(pr.IntentPath)), bodyPrefix) {
		t.Errorf("minted intent filename carries the key body prefix: %s", pr.IntentPath)
	}
	if draft := readLedgerFile(t, repo, pr.IntentPath); strings.Contains(draft, strings.Repeat("Q", 8)) {
		t.Errorf("minted intent carries key body bytes:\n%s", draft)
	}
}
