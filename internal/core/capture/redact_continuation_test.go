package capture

import (
	"strings"
	"testing"
)

// TestLedgerRedactsHomeWithANameSuffix pins the ledger redactor to the three
// continuation shapes main redacted and the trailing anchor let through:
// the caller's home followed by ".zip", "-old" and "_snapshot", behind a URL
// host and as a plain path. The ledger has no backstop, so this is its only gate.
func TestLedgerRedactsHomeWithANameSuffix(t *testing.T) {
	repo := t.TempDir()
	home := "/Users/zzhomeuser42" // abcd-audit:allow
	t.Setenv("HOME", home)
	for _, text := range []string{
		"the archive is at https://ci.example.com" + home + ".zip for review",
		"old copy under " + home + "-old/x here",
		"snapshot under /Volumes/T7" + home + "_snapshot/x here",
	} {
		out, _, degraded := redactLedgerText(repo, text)
		if degraded != "" {
			t.Fatalf("scanner degraded: %s", degraded)
		}
		if strings.Contains(out, "zzhomeuser42") {
			t.Errorf("%q: the caller's name reached the ledger text:\n%s", text, out)
		}
	}
}
