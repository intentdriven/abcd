package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestAhoyInstallOffersTheEmDashRuleOnStdin is ruling G1 at the front door: an
// interactive install asks whether the em-dash-in-list-item house-style rule
// blocks or warns, on the same stdin every other question reads. The stream is
// what `yes | abcd ahoy install` sends, so the answer to this question is "y",
// which names neither choice: the rule must be seeded as a warning, never
// guessed into a gate, and the result must say so.
func TestAhoyInstallOffersTheEmDashRuleOnStdin(t *testing.T) {
	hermeticEnv(t)
	// A REAL repository: the docs-lint seed is written only on git's own verdict
	// that its path is not ignored, which a bare `.git` directory cannot give.
	repo := gittest.NewRepo(t).Root()
	t.Chdir(repo)
	out, errOut, err := runCLIPipedStdinSplit(t, strings.Repeat("y\n", 16), "ahoy", "install",
		"--visibility", "private", "--docs-target", "both",
		"--oracle-backend", "host-delegated", "--scan-deep", "false", "--json")
	if err != nil {
		t.Fatalf("install exited non-zero: %v\n%s\n%s", err, out, errOut)
	}
	if !strings.Contains(string(errOut), "docs_lint.em_dash_in_list_item (blocking/warning) [warning]: y") {
		t.Fatalf("the em-dash question was not asked on stdin:\n%s", errOut)
	}
	var res struct {
		Notes []string `json:"notes"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("install output not JSON: %v\n%s", err, out)
	}
	if !strings.Contains(strings.Join(res.Notes, "\n"), "punctuation/em-dash-in-list-item") {
		t.Errorf("the result does not say the rule was seeded as a warning; notes %v", res.Notes)
	}
	data, err := os.ReadFile(filepath.Join(repo, ".abcd", "docs-lint.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		BannedTokens []struct {
			ID       string `json:"id"`
			Severity string `json:"severity"`
		} `json:"banned_tokens"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	for _, tok := range cfg.BannedTokens {
		if tok.ID == "punctuation/em-dash-in-list-item" {
			if tok.Severity != "warn" {
				t.Errorf("seeded severity = %q, want warn", tok.Severity)
			}
			return
		}
	}
	t.Error("the seeded docs-lint config carries no em-dash token")
}
