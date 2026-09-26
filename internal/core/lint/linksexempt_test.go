package lint

import (
	"path/filepath"
	"testing"
)

// A tool-mandated mirror of a root file (a byte-identical copy of AGENTS.md a
// tool reads from .github/, because it follows no pointer) carries links that
// resolve from the root and not from the mirror's directory. links_resolve's own
// `exempt` globs excuse such a file from the link check; every other file under
// the roots is still checked (iss-2609151150180583). Before the fix the key
// decoded cleanly and was read by nothing, so it looked like an exemption and
// excused no link.
func TestLinksResolveExemptGlobExcusesAMirrorFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "AGENTS.md", "see [guide](docs/guide.md)\n")
	writeFile(t, root, "docs/guide.md", "# Guide\n")
	writeFile(t, root, ".github/copilot-instructions.md", "see [guide](docs/guide.md)\n")
	writeFile(t, root, ".github/other.md", "see [guide](docs/guide.md)\n")
	cfg := Config{
		Roots: []string{"AGENTS.md", "docs", ".github"},
		Rules: map[string]RuleConfig{"links_resolve": {
			Enabled: true, Severity: "blocker",
			Exempt: []string{".github/copilot-instructions.md"},
		}},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if hasFinding(fs, filepath.Join(".github", "copilot-instructions.md"), "links_resolve", 1) {
		t.Errorf("the exempt mirror file still raised links_resolve: %+v", fs)
	}
	if !hasFinding(fs, filepath.Join(".github", "other.md"), "links_resolve", 1) {
		t.Errorf("a file the exemption does not name was excused too: %+v", fs)
	}
	if n := countRule(fs, "links_resolve"); n != 1 {
		t.Errorf("want exactly one links_resolve finding, got %d: %+v", n, fs)
	}
}
