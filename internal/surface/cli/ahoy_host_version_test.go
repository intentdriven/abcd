package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
)

var cliInstallArgs = []string{"ahoy", "install", "--yes", "--adopt", "--visibility", "private", "--docs-target", "agents_md",
	"--oracle-backend", "host-delegated", "--scan-deep", "false"}

// TestNoCLITestRunsTheMachinesAgentTool: an install run in-process by this
// package's tests never starts the agent tool on PATH to read its version,
// so no unit test runs a vendor binary of the machine's own and no test's
// output depends on which release the developer has installed.
func TestNoCLITestRunsTheMachinesAgentTool(t *testing.T) {
	hermeticRepo(t)
	bin := t.TempDir()
	ran := filepath.Join(t.TempDir(), "ran")
	script := "#!/bin/sh\n: > '" + ran + "'\necho '2.1.200 (Claude Code)'\n"
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	out, err := runCLIStdinErr(t, "", cliInstallArgs...)
	if err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	if _, err := os.Stat(ran); err == nil {
		t.Error("an install in a unit test ran the agent tool on PATH")
	}
	if strings.Contains(string(out), "older than the release that reads AGENTS.md") {
		t.Errorf("the version warning rendered from the machine's agent tool:\n%s", out)
	}
}

// TestAhoyInstallPrintsTheHostVersionWarning: an agent tool older than the
// release that reads AGENTS.md on its own, as a stub reading reports it, is
// one warning line the text render prints before the headline, naming no
// version (itd-2610030814013772, A5).
func TestAhoyInstallPrintsTheHostVersionWarning(t *testing.T) {
	hermeticRepo(t)
	t.Cleanup(ahoy.OldHostVersionForTest())

	out, err := runCLIStdinErr(t, "", cliInstallArgs...)
	if err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	first, _, _ := strings.Cut(string(out), "\n")
	if !strings.HasPrefix(first, "warning: The Claude Code on this computer is older than the release that reads AGENTS.md on its own") {
		t.Errorf("the first line is not the version warning: %q\n%s", first, out)
	}
	if strings.Count(string(out), "warning: ") != 1 {
		t.Errorf("want exactly one warning:\n%s", out)
	}
	if regexp.MustCompile(`\d+\.\d+\.\d+`).MatchString(first) {
		t.Errorf("the warning names a version: %q", first)
	}
}
