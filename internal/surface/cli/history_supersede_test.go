package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHistoryCaptureSupersededPathIsHomeRedacted holds the success-envelope
// rule on the field supersession added. `history capture --json` already
// home-redacts the stored record's absolute path, because a machine-readable
// success envelope carrying the caller's home root is a developer-identity
// leak the CLI error scrub never sees. The superseded record's path is the same
// absolute path from the same store and needs the same treatment.
func TestHistoryCaptureSupersededPathIsHomeRedacted(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := t.TempDir()
	gitCmd(t, repo, "init")
	gitCmd(t, repo, "config", "user.email", "test@example.com")
	gitCmd(t, repo, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(repo, "f.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, repo, "add", ".")
	gitCommit(t, repo, "commit", "-m", "init")
	t.Chdir(repo)

	rootSHA := gitCmd(t, repo, "rev-list", "--max-parents=0", "HEAD")
	if err := os.MkdirAll(filepath.Join(home, ".abcd", "history", rootSHA, "transcripts"), 0o755); err != nil {
		t.Fatal(err)
	}

	first := "user: turn one\n"
	second := first + "user: turn two\n"
	runCLIStdin(t, first, "history", "capture", "--session", "sess-sup", "--json")
	out := runCLIStdin(t, second, "history", "capture", "--session", "sess-sup", "--json")

	var res struct {
		Wrote      bool `json:"wrote"`
		Superseded *struct {
			Path string `json:"path"`
		} `json:"superseded"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("capture output not JSON: %v\n%s", err, out)
	}
	if !res.Wrote || res.Superseded == nil {
		t.Fatalf("a longer second capture must supersede the first; got %s", out)
	}
	if !strings.HasPrefix(res.Superseded.Path, "~/") {
		t.Errorf("the superseded path must be home-redacted, got %q", res.Superseded.Path)
	}
	if h, err := os.UserHomeDir(); err == nil && h != "" && strings.Contains(string(out), h) {
		t.Errorf("the success envelope carries the absolute home root:\n%s", out)
	}
}
