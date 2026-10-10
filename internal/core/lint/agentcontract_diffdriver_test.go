package lint

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestAgentContractDiffRunsNoRepoDiffDriver is iss-2610090821531570. The
// armed bump check reads a unified diff, and without --no-ext-diff and
// --no-textconv git hands that diff to the repository's own diff.external,
// diff.<driver>.command or diff.<driver>.textconv program, whose stdout the
// check then parses: a driver that prints a version line hides an unbumped
// prompt. A range of one revision diffs the working tree, which git re-reads
// through the repository's clean filter. The check must run none of them and
// still see the missing bump.
func TestAgentContractDiffRunsNoRepoDiffDriver(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sentinel script")
	}
	for _, tc := range []struct {
		name string
		key  string
		attr string
	}{
		{"diff.external", "diff.external", ""},
		{"diff.<driver>.command", "diff.evil.command", "* diff=evil\n"},
		{"diff.<driver>.textconv", "diff.evil.textconv", "* diff=evil\n"},
		// A single-revision range diffs the working tree, which git re-reads
		// through the repository's clean filter.
		{"filter.<name>.clean", "filter.evil.clean", "* filter=evil\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := newAgentRepo(t)
			mark := filepath.Join(t.TempDir(), "driver-ran")
			script := filepath.Join(t.TempDir(), "evil-diff.sh")
			body := "#!/bin/sh\ntouch " + mark + "\necho '+prompt_version: 9.9.9'\n"
			if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
				t.Fatal(err)
			}
			cfg := exec.Command("git", "-C", root, "config", tc.key, script)
			cfg.Env = gittest.Env(t)
			if out, err := cfg.CombinedOutput(); err != nil {
				t.Fatalf("git config: %v: %s", err, out)
			}
			if tc.attr != "" {
				if err := os.WriteFile(filepath.Join(root, ".git", "info", "attributes"), []byte(tc.attr), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			writeFile(t, root, filepath.Join("agents", "ruthless-reviewer.md"),
				"---\nname: ruthless-reviewer\n"+conformingAgent+"---\n\n# ruthless-reviewer\n\nA changed prompt body.\n")

			fs, err := Lint(ArmAgentDiff(agentCfg(), "HEAD"), root)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(mark); err == nil {
				t.Fatalf("the armed bump check ran the repository's %s program", tc.key)
			}
			if !messageContains(fs, "without a 'prompt_version' bump") {
				t.Fatalf("the unbumped edit must still be reported; got %+v", fs)
			}
		})
	}
}
