package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// docsLintExitFixture writes a docs-lint config whose one banned token carries
// severity, and one docs page that trips it, under a fresh temp root. It
// returns the config path and the root.
func docsLintExitFixture(t *testing.T, severity string) (string, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(root, "docs-lint.json")
	body := `{
  "roots": ["docs"],
  "banned_tokens": [
    {"id":"present_tense/previously","pattern":"(?i)\\bpreviously\\b","severity":"` + severity + `","message":"change-narration","successor":"present-tense phrasing","allow_context":["(?i)<!--\\s*docs-lint:\\s*allow\\b"]}
  ]
}`
	if err := os.WriteFile(cfg, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "page.md"),
		[]byte("# Page\n\nThis was previously X.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return cfg, root
}

// TestDocsLintExitCodes pins the tree gate's exit grammar (iss-2610241550303964):
// `abcd lint docs` exits 1 on any blocker and 0 when only warnings remain. The
// 2/1/0 tri-state, where a warning alone raises the exit to 1, is bare
// `abcd lint`'s, not this gate's.
func TestDocsLintExitCodes(t *testing.T) {
	for _, tc := range []struct {
		severity string
		want     int
	}{
		{"blocker", 1},
		{"warn", 0},
	} {
		t.Run(tc.severity, func(t *testing.T) {
			cfg, root := docsLintExitFixture(t, tc.severity)
			var stdout, stderr bytes.Buffer
			code := Run([]string{"lint", "docs", "--config", cfg, "--root", root}, &stdout, &stderr)
			if code != tc.want {
				t.Fatalf("lint docs over a one-%s fixture exited %d, want %d\n%s%s",
					tc.severity, code, tc.want, stdout.String(), stderr.String())
			}
			if !strings.Contains(stdout.String(), "present_tense/previously") {
				t.Fatalf("the %s finding was not reported:\n%s", tc.severity, stdout.String())
			}
		})
	}
}

// TestTheLintBriefStatesTheTreeGatesExit pins the lint chapter to the exits the
// two tree gates return, so the brief cannot again send a CI job to key on exit
// 2 for a blocker, which would pass a tree full of them (iss-2610241550303964).
func TestTheLintBriefStatesTheTreeGatesExit(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRootFromTest(t), ".abcd", "development", "brief", "05-internals", "06-lint.md"))
	if err != nil {
		t.Fatal(err)
	}
	brief := string(data)
	for _, wrong := range []string{
		"Raises the exit code to 1 on an otherwise-clean run",
		"Their exit code follows the standalone tri-state grammar",
	} {
		if strings.Contains(brief, wrong) {
			t.Errorf("the lint brief still says %q, which neither tree gate does", wrong)
		}
	}
	for _, want := range []string{
		"**blocker → 1, could not run → 2, otherwise 0**",
		"bare `abcd lint`",
	} {
		if !strings.Contains(brief, want) {
			t.Errorf("the lint brief does not state %q", want)
		}
	}
}
