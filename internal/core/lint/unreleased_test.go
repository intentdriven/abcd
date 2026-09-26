package lint

import (
	"path/filepath"
	"strings"
	"testing"
)

// A hand-written entry under `## [Unreleased]` wedges every derived cut: the
// release ingest refuses a non-empty section by design, so the change that adds
// one is refused here, at its record gate, with a message naming the
// record-first flow (iss-256).
func TestChangelogUnreleasedMustStayEmpty(t *testing.T) {
	rule := map[string]RuleConfig{ruleChangelogUnreleasedEmpty: {Enabled: true, Severity: severityBlocker}}
	for name, c := range map[string]struct {
		text     string
		wantLine int
	}{
		"empty":            {"# Changelog\n\n## [Unreleased]\n\n## [0.1.0] - 2026-01-01\n\n- shipped\n", 0},
		"hand entry":       {"# Changelog\n\n## [Unreleased]\n\n- **A thing.** It works.\n\n## [0.1.0] - 2026-01-01\n", 5},
		"subheading only":  {"# Changelog\n\n## [Unreleased]\n### Added\n## [0.1.0] - 2026-01-01\n", 4},
		"no anchor at all": {"# Changelog\n\n## [0.1.0] - 2026-01-01\n", 1},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, "CHANGELOG.md", c.text)
			fs, err := Lint(Config{Rules: rule}, root)
			if err != nil {
				t.Fatal(err)
			}
			if c.wantLine == 0 {
				if len(fs) != 0 {
					t.Fatalf("want no finding, got %+v", fs)
				}
				return
			}
			if len(fs) != 1 || fs[0].Line != c.wantLine || fs[0].File != filepath.FromSlash("CHANGELOG.md") {
				t.Fatalf("want one finding at CHANGELOG.md:%d, got %+v", c.wantLine, fs)
			}
			if name == "hand entry" && !strings.Contains(fs[0].Message, "abcd capture resolve") {
				t.Errorf("the finding must name the record-first flow: %s", fs[0].Message)
			}
		})
	}
}

// The shipped config arms the rule as a blocker.
func TestChangelogUnreleasedArmedInRealConfig(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("..", "..", "..", ".abcd", "record-lint.json"))
	if err != nil {
		t.Fatal(err)
	}
	if rc := cfg.Rules[ruleChangelogUnreleasedEmpty]; !rc.Enabled || rc.Severity != severityBlocker {
		t.Fatalf("record-lint.json must arm %s as a blocker, got %+v", ruleChangelogUnreleasedEmpty, rc)
	}
}
