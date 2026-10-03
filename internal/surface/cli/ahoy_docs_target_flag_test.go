package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
)

// TestDocsTargetFlagRefusesRetiredValues is A6's flag: --docs-target claude_md
// or both is refused at the flag, before the install runs, with the one
// explanation the core holds, so the front door restates nothing and writes
// nothing (itd-2610030814013772).
func TestDocsTargetFlagRefusesRetiredValues(t *testing.T) {
	for _, v := range []string{"claude_md", "both"} {
		t.Run(v, func(t *testing.T) {
			repo := hermeticRepo(t)
			out, err := runCLIStdinErr(t, "", "ahoy", "install", "--yes", "--adopt",
				"--visibility", "private", "--docs-target", v,
				"--oracle-backend", "host-delegated", "--scan-deep", "false", "--json")
			if err == nil {
				t.Fatalf("--docs-target %s was accepted:\n%s", v, out)
			}
			why, retired := ahoy.RetiredDocsTarget(v)
			if !retired {
				t.Fatalf("%s is not retired in the core", v)
			}
			if !strings.Contains(err.Error(), why) {
				t.Errorf("the refusal does not carry the core's explanation:\n got: %s\nwant: %s", err, why)
			}
			for _, rel := range []string{".abcd", "CLAUDE.md", "AGENTS.md"} {
				if _, err := os.Lstat(filepath.Join(repo, rel)); !os.IsNotExist(err) {
					t.Errorf("%s exists after a refused flag (err=%v)", rel, err)
				}
			}
		})
	}
}
