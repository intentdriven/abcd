package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// The MR001 a lint raises because a git identity value is not valid UTF-8
// names that setting to re-save, not the scanner's configuration; one raised
// for a degraded pattern set still names pii.json.
func TestLintMR001NamesTheFixForItsCause(t *testing.T) {
	mr001 := func(t *testing.T, repo string) Finding {
		t.Helper()
		res, err := Lint(LintRequest{RepoRoot: repo, Now: fixedNow})
		if err != nil {
			t.Fatalf("lint: %v", err)
		}
		for _, f := range res.Findings {
			if f.Code == "MR001" && strings.Contains(f.Message, "secret scan unavailable") {
				return f
			}
		}
		t.Fatalf("no degraded-scanner MR001: %+v", res.Findings)
		return Finding{}
	}
	t.Run("identity", func(t *testing.T) {
		repo := t.TempDir()
		if err := os.MkdirAll(Dir(repo), 0o755); err != nil {
			t.Fatal(err)
		}
		for _, kv := range os.Environ() {
			if k, _, _ := strings.Cut(kv, "="); strings.HasPrefix(k, "GIT_") {
				t.Setenv(k, "")
				os.Unsetenv(k)
			}
		}
		global := filepath.Join(t.TempDir(), "global.gitconfig")
		if err := os.WriteFile(global, []byte("[user]\n\tname = Zorv\xe9n Quill\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("GIT_CONFIG_GLOBAL", global)
		t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
		t.Setenv(abcdhome.StageHomeEnv, "")
		f := mr001(t, repo)
		if !strings.Contains(f.Suggestion, "re-save git config user.name as UTF-8") || strings.Contains(f.Suggestion, "pii.json") {
			t.Errorf("MR001 suggests %q; want git config user.name named to re-save as UTF-8, not pii.json", f.Suggestion)
		}
	})
	t.Run("patterns", func(t *testing.T) {
		repo := t.TempDir()
		if err := os.MkdirAll(Dir(repo), 0o755); err != nil {
			t.Fatal(err)
		}
		cfg := filepath.Join(repo, ".abcd", "config")
		if err := os.MkdirAll(cfg, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cfg, "pii.json"), []byte(`{"patterns": {"p": {"regex": ""}}}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if f := mr001(t, repo); !strings.Contains(f.Suggestion, ".abcd/config/pii.json") {
			t.Errorf("MR001 suggests %q; want the scanner's configuration named", f.Suggestion)
		}
	})
}
