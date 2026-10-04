package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// userScopeEnv is hermeticEnv with the ABCD_BIN_TARGET override cleared and PATH
// stated, so the CLI exercises the real default install location
// (~/.local/bin/abcd) rather than a temp target that hides a home-path leak.
func userScopeEnv(t *testing.T, pathDirs ...string) (home, pluginRoot string) {
	t.Helper()
	home = t.TempDir()
	pluginRoot = t.TempDir()
	if err := os.MkdirAll(filepath.Join(pluginRoot, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginRoot, "hooks", "hooks.json"), []byte(validHooksJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginRoot, "abcd"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", pluginRoot)
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("ABCD_BIN_TARGET", "")
	t.Setenv("PATH", strings.Join(pathDirs, string(os.PathListSeparator)))
	provisionHermeticCache(t, home)
	return home, pluginRoot
}

// TestAhoyUninstallPrintsNoAbsoluteHomePath pins the iss-177 hand-off at the one
// print site outside the receipt's own note seam: `ahoy uninstall` renders the
// symlink target verbatim, and the default install location now lives under
// $HOME, so the line would carry the username unless the target is rendered in
// tilde form.
func TestAhoyUninstallPrintsNoAbsoluteHomePath(t *testing.T) {
	home, pluginRoot := userScopeEnv(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(pluginRoot, "abcd"), filepath.Join(binDir, "abcd")); err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)

	out := string(runCLI(t, "ahoy", "uninstall"))
	if strings.Contains(out, home) {
		t.Fatalf("uninstall printed the absolute home path %q:\n%s", home, out)
	}
	if !strings.Contains(out, "~/.local/bin/abcd") {
		t.Fatalf("uninstall did not name the removed entry in tilde form:\n%s", out)
	}
}

// TestAhoyInstallBinDirFlagIsWired proves --bin-dir reaches the core engine from
// the CLI front door: the entry lands in the named directory, never in the
// default one.
func TestAhoyInstallBinDirFlagIsWired(t *testing.T) {
	home, _ := userScopeEnv(t)
	dir := filepath.Join(t.TempDir(), "opt", "bin")
	t.Setenv("PATH", dir)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)

	runCLI(t, "ahoy", "install", "--yes", "--adopt", "--bin-dir", dir,
		"--visibility", "private", "--docs-target", "agents_md",
		"--oracle-backend", "host-delegated", "--scan-deep", "false", "--json")

	if _, err := os.Lstat(filepath.Join(dir, "abcd")); err != nil {
		t.Fatalf("--bin-dir did not reach the core engine: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(home, ".local", "bin", "abcd")); !os.IsNotExist(err) {
		t.Errorf("--bin-dir install also wrote the default location: %v", err)
	}
}
