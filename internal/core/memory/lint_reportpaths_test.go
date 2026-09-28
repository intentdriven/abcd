package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// memory lint reports its run-log directory, the store it read and every
// finding's file without the home path: a checkout under the home named the
// developer in `memory lint --json` through report_dir, store_path and each
// finding's file, against the iss-81 rule (iss-2609261950061900). Each is named
// relative to the repository, and the run log is still written where the
// reported directory says.
func TestLintReportsItsPathsRelativeToTheRepository(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := filepath.Join(home, "src", "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	seedResidueStore(t, repo, false)
	plantResidue(t, filepath.Join(Dir(repo), "topic_auth_tokens.md"), map[string]string{
		"MARKERTOKEN": "ghp_" + strings.Repeat("A", 40),
	})

	res, err := Lint(LintRequest{RepoRoot: repo, Now: fixedNow})
	if err != nil {
		t.Fatalf("lint: %v", err)
	}
	if len(res.Findings) == 0 {
		t.Fatal("fixture drift: lint reported no finding, so no finding's file is under test")
	}

	raw, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range homeSpellings(home) {
		if strings.Contains(string(raw), h) {
			t.Errorf("memory lint --json carries the home directory %q:\n%s", h, raw)
		}
	}
	if res.StorePath != RelDir {
		t.Errorf("store_path = %q, want %q", res.StorePath, RelDir)
	}
	if !strings.HasPrefix(res.ReportDir, ".abcd/.work.local/logs/memory/lint-") {
		t.Errorf("report_dir = %q, want it relative to the repository", res.ReportDir)
	}
	for _, f := range res.Findings {
		if filepath.IsAbs(f.File) {
			t.Errorf("%s finding names its file absolutely: %q", f.Code, f.File)
		}
	}

	for _, name := range []string{"report.json", "report.md"} {
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(res.ReportDir), name))
		if err != nil {
			t.Fatalf("the run log is not where report_dir says: %v", err)
		}
		for _, h := range homeSpellings(home) {
			if strings.Contains(string(data), h) {
				t.Errorf("%s carries the home directory %q", name, h)
			}
		}
	}
}

func homeSpellings(home string) []string {
	out := []string{home}
	if real, err := filepath.EvalSymlinks(home); err == nil && real != home {
		out = append(out, real)
	}
	return out
}

// inRepo is abs as Lint reports it: relative to the repository, slash-separated.
func inRepo(t *testing.T, repo, abs string) string {
	t.Helper()
	rel, err := filepath.Rel(repo, abs)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.ToSlash(rel)
}
