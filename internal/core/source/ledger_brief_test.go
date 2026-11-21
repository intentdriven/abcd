package source

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestTheLedgerGlossaryAllowsTheLibraryChaptersBareLedger pins the library
// surface chapter to the ledger glossary entry. The entry once reserved bare
// "the ledger" for the issue ledger and had every other ledger named in full,
// while the library chapter, where the `abcd library ledger` sub-verb lives,
// says bare "the ledger" for the sources ledger (iss-2610262033569882). The
// entry now allows the bare form there, which holds only while the chapter
// never speaks of the issue ledger, so the bare word cannot mean it. Only the
// chapter's prose is read: the generated surface appendix names the sub-verb.
func TestTheLedgerGlossaryAllowsTheLibraryChaptersBareLedger(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	data, err := os.ReadFile(filepath.Join(root, ".abcd", "development", "brief", "04-surfaces", "07-library.md"))
	if err != nil {
		t.Fatal(err)
	}
	prose := string(data)
	if i := strings.Index(prose, "<!-- surface-appendix:begin"); i >= 0 {
		prose = prose[:i]
	}
	prose = strings.Join(strings.Fields(prose), " ")
	if !regexp.MustCompile(`(?i)\bthe ledgers?\b`).MatchString(prose) {
		return // nothing bare to sanction
	}
	if strings.Contains(strings.ToLower(prose), "issue ledger") {
		t.Error("the library chapter speaks of the issue ledger, so its bare \"the ledger\" is ambiguous: name the sources ledger in full there")
	}
	gloss, err := os.ReadFile(filepath.Join(root, ".abcd", "development", "brief", "glossary", "core", "ledger.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(gloss)
	start := strings.Index(body, "## When to use")
	if start < 0 {
		t.Fatal("the ledger glossary entry has no `## When to use` section")
	}
	section := body[start+len("## When to use"):]
	if end := strings.Index(section, "\n## "); end >= 0 {
		section = section[:end]
	}
	section = strings.Join(strings.Fields(section), " ")
	for _, want := range []string{
		"04-surfaces/07-library.md",
		`bare "the ledger" means the sources ledger`,
	} {
		if !strings.Contains(section, want) {
			t.Errorf("the ledger glossary's `When to use` does not allow the library chapter's bare \"the ledger\" for the sources ledger: it lacks %q", want)
		}
	}
}
