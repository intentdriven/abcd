package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/update"
)

// runUpdateHermetic drives `abcd update` in a hermetic PATH/HOME with the
// updater-construction seam counted, so a dispatch refusal is proven to
// happen BEFORE anything network-capable exists.
func runUpdateHermetic(t *testing.T, args ...string) (string, error, int) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("ABCD_BIN_TARGET", "")
	t.Setenv("PATH", filepath.Join(home, ".local", "bin"))

	calls := 0
	orig := newUpdater
	newUpdater = func() *update.Updater { calls++; return orig() }
	t.Cleanup(func() { newUpdater = orig })

	var out bytes.Buffer
	cmd := NewRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(append([]string{"update"}, args...))
	err := cmd.Execute()
	return out.String(), err, calls
}

// TestUpdateRefusesWithNothingOnPath: the absent shape refuses loudly, names
// the install remedy, exits non-zero — and never constructs the updater, so
// the refusal path is provably network-free (the adr-38 seam).
func TestUpdateRefusesWithNothingOnPath(t *testing.T) {
	out, err, calls := runUpdateHermetic(t)
	if err == nil {
		t.Fatal("a refusal must exit non-zero")
	}
	if !strings.Contains(out, "nothing to update") || !strings.Contains(out, "install") {
		t.Errorf("the refusal is not loud or names no remedy:\n%s", out)
	}
	if calls != 0 {
		t.Errorf("the refusal path constructed the network updater %d times; the dispatch must refuse first", calls)
	}
}

// TestUpdateRefusesPluginRootEntry: an owned plugin-root symlink names the
// host's plugin-update path and touches nothing.
func TestUpdateRefusesPluginRootEntry(t *testing.T) {
	home := t.TempDir()
	pluginRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(pluginRoot, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginRoot, "abcd"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(binDir, "abcd")
	if err := os.Symlink(filepath.Join(pluginRoot, "abcd"), link); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", pluginRoot)
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("ABCD_BIN_TARGET", "")
	t.Setenv("PATH", binDir)

	calls := 0
	orig := newUpdater
	newUpdater = func() *update.Updater { calls++; return orig() }
	t.Cleanup(func() { newUpdater = orig })

	var out bytes.Buffer
	cmd := NewRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"update"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("the plugin-root shape must refuse")
	}
	if !strings.Contains(out.String(), "plugin update") {
		t.Errorf("the refusal does not name the plugin-update path:\n%s", out.String())
	}
	if calls != 0 {
		t.Errorf("the plugin-root refusal constructed the updater %d times", calls)
	}
	if fi, serr := os.Lstat(link); serr != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the refusal touched the entry: %v %v", fi, serr)
	}
}

// TestUpdateSeamUntouchedByOtherVerbs is the adr-38 zero-network extension:
// no verb but update ever constructs the updater.
func TestUpdateSeamUntouchedByOtherVerbs(t *testing.T) {
	calls := 0
	orig := newUpdater
	newUpdater = func() *update.Updater { calls++; return orig() }
	t.Cleanup(func() { newUpdater = orig })

	for _, args := range [][]string{{"version"}, {"rules"}} {
		cmd := NewRootCommand()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs(args)
		_ = cmd.Execute()
	}
	if calls != 0 {
		t.Errorf("a non-update verb constructed the updater %d times", calls)
	}
}

// TestUpdateRejectsMalformedTag: a path-shaped tag refuses before any request.
func TestUpdateRejectsMalformedTag(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "abcd"), []byte("\x7fELFfake"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("ABCD_BIN_TARGET", "")
	t.Setenv("PATH", binDir)

	var out bytes.Buffer
	cmd := NewRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"update", "v1/../evil"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "not a release tag") {
		t.Fatalf("a malformed tag must refuse by shape, got: %v\n%s", err, out.String())
	}
}

// TestUpdateReceiptNamesTheUnpublishedBuildItReplaced: when the file abcd
// replaced belongs to no release the forge still serves, the receipt must say
// so by digest and name the proof that let abcd touch it at all. Rendering
// "updated ~/.local/bin/abcd:  -> v0.7.0" with an empty left-hand side would
// read as a bug in the very receipt that has to be trusted here
// (iss-2609012000222546).
func TestUpdateReceiptNamesTheUnpublishedBuildItReplaced(t *testing.T) {
	digest := strings.Repeat("ab", 32)
	var out bytes.Buffer
	renderUpdateReport(&out, false, update.Report{
		Action:     update.ActionSwapped,
		Origin:     "https://example.invalid/abcd",
		TargetPath: "~/.local/bin/abcd",
		OldDigest:  digest,
		Ownership:  update.OwnedByRunningExecutable,
		NewVersion: "v0.7.0",
		Digest:     strings.Repeat("cd", 32),
	})
	got := out.String()
	if !strings.Contains(got, "an unpublished build") {
		t.Errorf("the receipt must name what it replaced when no version could be derived:\n%s", got)
	}
	if !strings.Contains(got, digest) {
		t.Errorf("the receipt must carry the replaced file's digest:\n%s", got)
	}
	if !strings.Contains(got, update.OwnedByRunningExecutable.Prose()) {
		t.Errorf("the receipt must name the ownership proof that allowed the swap:\n%s", got)
	}
}

// TestUpdateReceiptKeepsTheOrdinaryVersionLine: the digest line is the
// exception, not the new normal — a provable old build still renders as a
// plain old -> new version pair with no digest noise.
func TestUpdateReceiptKeepsTheOrdinaryVersionLine(t *testing.T) {
	var out bytes.Buffer
	renderUpdateReport(&out, false, update.Report{
		Action:     update.ActionSwapped,
		Origin:     "https://example.invalid/abcd",
		TargetPath: "~/.local/bin/abcd",
		OldVersion: "v0.6.9",
		Ownership:  update.OwnedByManifest,
		NewVersion: "v0.7.0",
		Digest:     strings.Repeat("cd", 32),
	})
	got := out.String()
	if !strings.Contains(got, "v0.6.9 -> v0.7.0") {
		t.Errorf("the ordinary receipt line changed:\n%s", got)
	}
	if strings.Contains(got, "unpublished") {
		t.Errorf("a provable old build must not be reported as unpublished:\n%s", got)
	}
}
