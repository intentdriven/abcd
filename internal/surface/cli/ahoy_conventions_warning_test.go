package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAhoyInstallPrintsTheOwnersFileWarningFirst: a tool's own conventions
// file holding the owner's words is named in a warning the text render prints
// before anything else, the headline included, and the JSON carries it as
// `warnings`; the file is left byte for byte (itd-2610030814013772, A3).
func TestAhoyInstallPrintsTheOwnersFileWarningFirst(t *testing.T) {
	repo := hermeticRepo(t)
	const owners = "Always run make check first\n"
	if err := os.WriteFile(filepath.Join(repo, "CLAUDE.md"), []byte(owners), 0o644); err != nil {
		t.Fatal(err)
	}
	args := []string{"ahoy", "install", "--yes", "--adopt", "--visibility", "private", "--docs-target", "agents_md",
		"--oracle-backend", "host-delegated", "--scan-deep", "false"}

	out, err := runCLIStdinErr(t, "", args...)
	if err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	first, _, _ := strings.Cut(string(out), "\n")
	if !strings.HasPrefix(first, "warning: CLAUDE.md holds your own words, so Claude Code reads it and not AGENTS.md") {
		t.Errorf("the first line is not the warning: %q\n%s", first, out)
	}
	if strings.Count(string(out), "warning: ") != 1 {
		t.Errorf("want exactly one warning:\n%s", out)
	}

	raw, err := runCLIStdinErr(t, "", append(args, "--json")...)
	if err != nil {
		t.Fatalf("install --json: %v\n%s", err, raw)
	}
	var res struct {
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, raw)
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "CLAUDE.md") {
		t.Errorf("warnings = %q, want the one naming CLAUDE.md", res.Warnings)
	}
	got, err := os.ReadFile(filepath.Join(repo, "CLAUDE.md"))
	if err != nil || string(got) != owners {
		t.Errorf("CLAUDE.md changed: %q (%v)", got, err)
	}
}
