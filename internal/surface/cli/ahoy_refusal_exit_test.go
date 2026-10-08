package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/vintage"
)

// TestAhoyInstallRefusalExitsTwo pins the exit code of every refusal `ahoy
// install` renders from the core (iss-2610031915386832, iss-2610020705158672).
// Each one stops the run before its first write and names the reason in the
// result's notes; an exit 0 there let a script read an install that wrote
// nothing as one that landed. Exit 2 is the code every other verb gives a
// refusal, and the result is still rendered on stdout so the reason reaches the
// caller.
func TestAhoyInstallRefusalExitsTwo(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, repo string)
		why   string
	}{
		{
			name: "symlinked .abcd",
			setup: func(t *testing.T, repo string) {
				if err := os.Symlink(t.TempDir(), filepath.Join(repo, ".abcd")); err != nil {
					t.Fatal(err)
				}
			},
			why: ".abcd exists but is not a real directory",
		},
		{
			name: "stale binary",
			setup: func(t *testing.T, _ string) {
				t.Cleanup(ahoy.SetCurrentVintageForTest(func() vintage.Current {
					return vintage.Current{Known: false}
				}))
			},
			why: "--allow-stale-binary",
		},
		{
			name: "saved claude_md",
			setup: func(t *testing.T, repo string) {
				if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
					t.Fatal(err)
				}
				cfg := `{"repo":{"visibility":"private"},"docs":{"target":"claude_md"},"oracle":{"backend":"host-delegated"}}`
				if err := os.WriteFile(filepath.Join(repo, ".abcd", "config.json"), []byte(cfg), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			why: "claude_md",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := hermeticRepo(t)
			tc.setup(t, repo)
			out, err := runCLIErr(t, "ahoy", "install", "--yes", "--adopt",
				"--visibility", "private", "--oracle-backend", "host-delegated",
				"--scan-deep", "false", "--json")
			var ee *exitError
			if !errors.As(err, &ee) || ee.Code != 2 {
				t.Fatalf("a refused install: %v, want exit 2\n%s", err, out)
			}
			var res ahoy.InstallResult
			if jerr := json.Unmarshal(out, &res); jerr != nil {
				t.Fatalf("the refused result is not rendered on stdout: %v\n%s", jerr, out)
			}
			if res.Status != "refused" || !strings.Contains(strings.Join(res.Notes, "\n"), tc.why) {
				t.Errorf("status %q, notes lack %q: %v", res.Status, tc.why, res.Notes)
			}
		})
	}

	// The text render carries the reason too, before the non-zero exit.
	t.Run("text", func(t *testing.T) {
		repo := hermeticRepo(t)
		if err := os.Symlink(t.TempDir(), filepath.Join(repo, ".abcd")); err != nil {
			t.Fatal(err)
		}
		out, err := runCLIErr(t, "ahoy", "install", "--yes", "--adopt")
		var ee *exitError
		if !errors.As(err, &ee) || ee.Code != 2 {
			t.Fatalf("a refused install: %v, want exit 2\n%s", err, out)
		}
		if !strings.Contains(string(out), "abcd ahoy install — refused") || !strings.Contains(string(out), "note: refused to install") {
			t.Errorf("the refusal's reason is not rendered:\n%s", out)
		}
	})
}

// TestAhoyRemoteApplyExitCodes pins the sibling verb to the same refusal code:
// a refused apply exits 2 like a refused install. An aborted one (an
// unanswered confirmation) keeps exit 1, where scripts already read it, which
// TestAhoyRemoteApplyExitsNonZeroWhenItChangesNothing pins.
func TestAhoyRemoteApplyExitCodes(t *testing.T) {
	hermeticEnv(t)
	t.Chdir(t.TempDir())
	out, err := runCLIErr(t, "ahoy", "remote", "apply")
	var ee *exitError
	if !errors.As(err, &ee) || ee.Code != 2 {
		t.Fatalf("a refused apply: %v, want exit 2\n%s", err, out)
	}
	if !strings.Contains(string(out), "refused") {
		t.Errorf("the refusal is not rendered:\n%s", out)
	}
}
