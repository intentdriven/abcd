package lint

import (
	"path/filepath"
	"testing"
)

// The shipped config bans the slash spellings of the five plugin pages the
// verb split retired (itd-2610090831227812, spc-2610100613109045 decision 11),
// and only those: /abcd:memory is left free for memory notes, and a line that
// says the page retired, a historical line, or a quoted audit-evidence line
// passes.
func TestRetiredPluginPagesBannedInRealConfig(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("..", "..", "..", ".abcd", "record-lint.json"))
	if err != nil {
		t.Fatal(err)
	}
	var entry *BannedToken
	for i := range cfg.BannedTokens {
		if cfg.BannedTokens[i].ID == "retired-plugin-pages" {
			entry = &cfg.BannedTokens[i]
		}
	}
	if entry == nil {
		t.Fatal("record-lint.json must carry the retired-plugin-pages banned token")
	}
	if entry.Severity != severityBlocker {
		t.Fatalf("retired-plugin-pages must be a blocker, got %q", entry.Severity)
	}
	checks, err := compileTokens([]BannedToken{*entry})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		line string
		flag bool
	}{
		{"Run `/abcd:consult` to read the corpus.", true},
		{"Run `/abcd:ingest` on the file.", true},
		{"Run `/abcd:prepare-this-repo` first.", true},
		{"`/abcd:version` prints the version.", true},
		{"`/abcd:source` renders the corpus.", true},
		{"the retired `/abcd:version` page is gone", false},
		{"the page retires with `/abcd:source`", false},
		{"historical: `/abcd:consult` ran the corpus", false},
		{"  evidence: commands/x.md:28 — \"| 13 | `/abcd:consult` | shipped |\"", false},
		{"`/abcd:memory` is kept for memory notes", false},
		{"`/abcd:library` renders both stores", false},
		{"`/abcd:sources` is no page name", false},
	}
	for _, c := range cases {
		got := checkBannedTokens("x.md", []string{c.line}, []bool{false}, checks)
		if (len(got) > 0) != c.flag {
			t.Errorf("line %q: flagged=%v, want %v", c.line, len(got) > 0, c.flag)
		}
	}
}
