package lint

import (
	"path/filepath"
	"testing"
)

// TestADRIDUniqueRefusesTwoClaimantsOfOneID is adr_id_unique: two files in the
// ADR store that answer to one decision are both refused, because every ADR
// reader resolves an id to ONE file and a second claimant is read first-wins —
// a proposed decision can be settled by an accepted twin nobody reviewed. The
// claim is read from the filename's number AND from the frontmatter `id:`, in
// both id vintages (the 0001–0058 ordinals and the minted stamp), with the
// handle compared case- and padding-insensitively, the reading every ADR reader
// already takes.
func TestADRIDUniqueRefusesTwoClaimantsOfOneID(t *testing.T) {
	root := t.TempDir()
	base := ".abcd/development/decisions/adrs"
	w := func(name, id string) {
		writeFile(t, root, base+"/"+name, "---\nid: "+id+"\nstatus: accepted\n---\n\n# A decision\n")
	}
	// Two files with the same filename number (the review's shape): one proposed,
	// one accepted, both claiming adr-37.
	w("0037-x.md", "adr-37")
	w("0037-a.md", "adr-37")
	// A frontmatter id that claims ANOTHER file's id, spelled in another case.
	w("0038-real.md", "adr-38")
	w("0039-impostor.md", "ADR-38")
	// The minted vintage: the same stamp twice, one padded.
	w("2609012206053814-first.md", "adr-2609012206053814")
	w("02609012206053814-second.md", "adr-2609012206053814")
	// The minted vintage through the frontmatter, quoted and case-shifted.
	w("2609012206053816-real.md", "adr-2609012206053816")
	w("2609012206053817-claims.md", `"Adr-2609012206053816"`)
	// A decision with one claimant, and a non-record page, stay clean.
	w("0041-solo.md", "adr-41")
	writeFile(t, root, base+"/README.md", "# Decisions\n")

	cfg := Config{Rules: map[string]RuleConfig{
		"adr_id_unique": {Enabled: true, Severity: "blocker"},
	}}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"0037-x.md", "0037-a.md",
		"0038-real.md", "0039-impostor.md",
		"2609012206053814-first.md", "02609012206053814-second.md",
		"2609012206053816-real.md", "2609012206053817-claims.md",
	} {
		if !hasFinding(fs, filepath.Join(base, name), "adr_id_unique", 2) {
			t.Errorf("expected an %s finding on %s:2; got %+v", "adr_id_unique", name, fs)
		}
	}
	for _, f := range fs {
		if b := filepath.Base(f.File); b == "0041-solo.md" || b == "README.md" {
			t.Errorf("unexpected finding on %s: %+v", b, f)
		}
		if f.RuleID == "adr_id_unique" && f.Severity != "blocker" {
			t.Errorf("finding carries severity %q, want the configured blocker: %+v", f.Severity, f)
		}
	}
}

// TestADRIDUniqueIsAKnownRule: a configuration naming the rule loads, so the
// repository's own record-lint.json can arm it.
func TestADRIDUniqueIsAKnownRule(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "record-lint.json",
		`{"roots":["rec"],"rules":{"adr_id_unique":{"enabled":true,"severity":"blocker"}}}`)
	if _, err := LoadConfig(filepath.Join(root, "record-lint.json")); err != nil {
		t.Fatalf("LoadConfig refused adr_id_unique: %v", err)
	}
}
