package memory

import (
	"strings"
	"testing"
)

// pem_identity_test.go — the memory-store legs of two scanner defects.
// GHSA-5qr6-f78x-g2cx: the store redacted a PEM BEGIN header and wrote the
// key body and END line into the page and the --keep-original copy.
// GHSA-gxhr-pmwv-r99p: identity redaction followed the repo's effective git
// identity and stored the caller's other one. Markers are assembled from
// halves and bodies are repeated letters: nothing here is a key.

// TestIngestRedactsPEMBodyFromPageAndKeptOriginal: a key block in the source
// reaches neither the page body nor the kept original, and the prose after
// the block survives.
func TestIngestRedactsPEMBodyFromPageAndKeptOriginal(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	header := "-----BEGIN " + "OPENSSH PRIVATE KEY-----"
	body1, body2 := strings.Repeat("Q", 64), strings.Repeat("R", 64)
	end := "-----END " + "OPENSSH PRIVATE KEY-----"

	stores := ingestEcho(t, repo, "The deploy key was:\n"+header+"\n"+body1+"\n"+body2+"\n"+end+"\nIt has been rotated.\n")
	for label, content := range stores {
		for _, leak := range []string{body1, body2, end} {
			if strings.Contains(content, leak) {
				t.Errorf("%s persists the key line %q…", label, leak[:8])
			}
		}
		if !strings.Contains(content, "It has been rotated.") {
			t.Errorf("%s lost the prose after the block:\n%s", label, content)
		}
	}
}
