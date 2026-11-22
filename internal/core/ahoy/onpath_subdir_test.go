//go:build unix

package ahoy

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestAProgramAtTheCheckoutRootIsNotRunFromASubdirectory extends
// TestAProgramInsideTheCheckoutIsNotInstalled to a pass run from a
// subdirectory: a gitleaks and a claude that PATH finds at the checkout's root
// are held to the whole checkout, not to the directory the pass runs in, so
// neither is run, gitleaks is still reported missing, and the agent tool's
// version is not read.
func TestAProgramAtTheCheckoutRootIsNotRunFromASubdirectory(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	repo := t.TempDir()
	sub := filepath.Join(repo, "pkg")
	for _, d := range []string{filepath.Join(repo, ".git"), sub} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	ran := filepath.Join(t.TempDir(), "ran")
	for name, version := range map[string]string{"gitleaks": "8.30.1", claudeCommand: "2.1.300 (Claude Code)"} {
		script := "#!/bin/sh\n: > " + ran + "\necho " + version + "\n"
		if err := os.WriteFile(filepath.Join(repo, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", repo)

	if onPath(sub, "gitleaks") {
		t.Error("onPath from a subdirectory counts a gitleaks at the checkout's root as installed")
	}
	depGap(t, detectDependencies(sub, true), "deps.gitleaks_missing")
	if _, err := readHostVersion(claudeCommand, sub); !errors.Is(err, errHostRefused) {
		t.Errorf("readHostVersion from a subdirectory = %v, want the refusal of a program inside the project", err)
	}
	if _, err := os.Stat(ran); err == nil {
		t.Fatal("a program at the checkout's root was run from a subdirectory")
	}
}
