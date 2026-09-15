package intent

import (
	"strings"
	"testing"
)

// redact_pem_identity_test.go — the intent store is the fourth committed
// sink on the shared scanner, the sibling the six advisories did not name.
// It redacts through the same scanner.Redact primitive as the ledger, the
// memory store and the transcript store, so the two primitive fixes reach
// it by construction; these tests are the proof, not a fourth patch.

func TestCreateFromTextRedactsPEMBody(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	header := "-----BEGIN " + "OPENSSH PRIVATE KEY-----"
	body := strings.Repeat("Q", 64)
	end := "-----END " + "OPENSSH PRIVATE KEY-----"
	draft := createDraft(t, root, "I want the deploy key\n"+header+"\n"+body+"\n"+end+"\nto be rotated on every release")
	if strings.Contains(draft, body) || strings.Contains(draft, end) || strings.Contains(strings.ToLower(draft), "qqqqqqqq") {
		t.Fatalf("committed draft carries key body bytes:\n%s", draft)
	}
	if !strings.Contains(draft, "to be rotated on every release") {
		t.Fatalf("the prose after the block was lost:\n%s", draft)
	}
}
