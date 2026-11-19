package intent

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// TestItd16AuditUmbrellaTakesNoCheckOfForm holds itd-16 (the /abcd:audit
// umbrella) to adr-40's buckets: an audit compares reality against a recorded
// commitment, and a check of form is a lint, an `abcd lint` target. itd-16 once
// offered a schema audit and a prompt-quality audit as future sub-verbs of the
// umbrella, while 16-lint keeps /abcd:audit for the hash-chain fidelity surface
// and calls such a check a lint (iss-2610262036067608). No line of itd-16 may
// name either as an audit application, and the record must say where a check
// of form goes instead.
func TestItd16AuditUmbrellaTakesNoCheckOfForm(t *testing.T) {
	root, err := fsutil.ModuleRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".abcd", "development", "intents", "drafts", "itd-16-hash-chain-merkle-audit.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read itd-16: %v", err)
	}
	formAudit := regexp.MustCompile(`(?i)\b(schema|prompt-quality) audits?\b`)
	for i, line := range strings.Split(string(raw), "\n") {
		if m := formAudit.FindString(line); m != "" {
			t.Errorf("%s:%d offers %q as an /abcd:audit application, but a check of form is a lint under adr-40: %q", path, i+1, m, strings.TrimSpace(line))
		}
	}
	body := string(raw)
	if !strings.Contains(body, "adr-40") || !strings.Contains(body, "abcd lint") {
		t.Errorf("%s does not say that a check of form is an `abcd lint` target under adr-40", path)
	}
}
