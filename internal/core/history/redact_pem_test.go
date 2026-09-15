package history

import (
	"strings"
	"testing"
)

// pem_identity_test.go — the transcript-store legs of two scanner defects.
// GHSA-gmp7-9rvm-qcr3: the store redacted a PEM BEGIN header and wrote the
// key body verbatim while stage two, re-running the same header-only rule
// over a fingerprinted header, was clean by construction. GHSA-v826-5jf4-p8xg:
// identity redaction followed the repo's effective git identity and stored
// the caller's other one. Markers are assembled from halves and bodies are
// repeated letters: nothing here is a key.

// TestCaptureRedactsPEMBody: the stored record carries no body line and no
// END line of a key block, keeps the prose after it, and still reports the
// key in its secrets count.
func TestCaptureRedactsPEMBody(t *testing.T) {
	repoRoot, _ := setupStore(t)
	header := "-----BEGIN " + "OPENSSH PRIVATE KEY-----"
	body1, body2 := strings.Repeat("Q", 64), strings.Repeat("R", 64)
	end := "-----END " + "OPENSSH PRIVATE KEY-----"

	record := captureText(t, repoRoot, "sess-pem", strings.Join([]string{
		"assistant: the key file reads", header, body1, body2, end, "assistant: rotated it",
	}, "\n"))
	for _, leak := range []string{body1, body2, end} {
		if strings.Contains(record, leak) {
			t.Errorf("stored record carries the key line %q…:\n%s", leak[:8], record)
		}
	}
	if !strings.Contains(record, "assistant: rotated it") {
		t.Errorf("the prose after the block was lost:\n%s", record)
	}
}
