package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// validHooksJSON is a structurally-sound plugin hook manifest.
const validHooksJSON = `{
  "hooks": {
    "UserPromptSubmit": [{"hooks": [{"type": "command", "command": "\"$CLAUDE_PLUGIN_ROOT/abcd\" hook prompt-router"}]}],
    "SessionStart":     [{"hooks": [{"type": "command", "command": "\"$CLAUDE_PLUGIN_ROOT/abcd\" hook prompt-router-reset"}]}],
    "PreCompact":       [{"hooks": [{"type": "command", "command": "\"$CLAUDE_PLUGIN_ROOT/abcd\" hook prompt-router-reset"}]}]
  }
}`

// setupHermetic redirects HOME, the plugin root, and the PATH symlink target to
// temp locations so a test never touches the real machine.
func setupHermetic(t *testing.T) (home, pluginRoot string) {
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
	binTargetPath := filepath.Join(t.TempDir(), "bin", "abcd")
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", pluginRoot)
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_DATA", "")
	// The harness's settings resolve under HOME once this is empty, so a
	// machine that names its own configuration directory never leaks its real
	// status line into a hermetic detection.
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("ABCD_BIN_TARGET", binTargetPath)
	// The detector walks PATH looking for abcd entries, so a machine that has a
	// real install — exactly the machines that dogfood the install — leaks
	// "(shadowed on PATH)" into every hermetic assertion (iss-249). Keep the
	// utilities the code under test shells out to; drop only the directories
	// that resolve an abcd.
	var kept []string
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		// Lstat, not Stat: the scanner under test classifies a DANGLING abcd symlink
		// as an entry that still shadows PATH, so a Stat here — which follows the link
		// to a not-exist error and keeps the directory — would leak that entry into
		// every hermetic assertion. Lstat sees the link itself and drops it.
		if fi, err := os.Lstat(filepath.Join(dir, "abcd")); err == nil && !fi.IsDir() {
			continue
		}
		kept = append(kept, dir)
	}
	t.Setenv("PATH", strings.Join(kept, string(os.PathListSeparator)))
	return home, pluginRoot
}

func TestClassifyUnmanagedFolder(t *testing.T) {
	setupHermetic(t)
	dir := t.TempDir()
	det, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if det.FolderKind != UnmanagedFolder {
		t.Errorf("kind = %q, want %q", det.FolderKind, UnmanagedFolder)
	}
}

func TestClassifyUnmanagedRepo(t *testing.T) {
	setupHermetic(t)
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	det, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if det.FolderKind != UnmanagedRepo {
		t.Errorf("kind = %q, want %q", det.FolderKind, UnmanagedRepo)
	}
}

// TestClassifyGitfileWorktreeIsARepo pins that a linked worktree or submodule —
// where `.git` is a regular gitfile ("gitdir: …"), not a directory — is detected
// as a git checkout, not an unmanaged folder. The old isDir check misread it and
// silently aborted `ahoy install` there.
func TestClassifyGitfileWorktreeIsARepo(t *testing.T) {
	setupHermetic(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: /somewhere/.git/worktrees/wt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	det, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if det.FolderKind != UnmanagedRepo {
		t.Errorf("gitfile worktree kind = %q, want %q", det.FolderKind, UnmanagedRepo)
	}
	if reg, _ := det.Signals["git_repo"].(bool); !reg {
		t.Errorf("git_repo signal = false over a gitfile worktree, want true")
	}
}

// TestGuardOmittedForUnmanagedFolder pins that the guard-health object is omitted
// for an unmanaged folder rather than serialising a never-computed all-false zero
// value, which would report a broken guard in a document that also says the
// plugin root is resolved. It mirrors the Banlist pointer's treatment.
func TestGuardOmittedForUnmanagedFolder(t *testing.T) {
	setupHermetic(t)
	det, err := Detect(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if det.FolderKind != UnmanagedFolder {
		t.Fatalf("precondition: kind = %q, want %q", det.FolderKind, UnmanagedFolder)
	}
	if det.Guard != nil {
		t.Errorf("Guard = %+v for an unmanaged folder, want nil (omitted)", *det.Guard)
	}
	// A repo, by contrast, carries a computed guard object.
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	rdet, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if rdet.Guard == nil {
		t.Errorf("Guard = nil for a repo, want a computed object")
	}
}

// TestClassifyAbcdDirInRepoIsNotManaged pins iss-88: a git repo carrying a stray
// .abcd/ directory but no index registration and no marker block is an
// unmanaged-repo, not a managed-repo — the .abcd/ dir alone never promotes.
func TestClassifyAbcdDirInRepoIsNotManaged(t *testing.T) {
	setupHermetic(t)
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	det, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if det.FolderKind != UnmanagedRepo {
		t.Errorf("kind = %q, want %q", det.FolderKind, UnmanagedRepo)
	}
}

// TestClassifyStrayAbcdDirIsNotManaged pins iss-88: a bare .abcd/ directory with
// no index registration and no marker block must NOT overclaim managed-repo. With
// no .git present it is an unmanaged folder.
func TestClassifyStrayAbcdDirIsNotManaged(t *testing.T) {
	setupHermetic(t)
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	det, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if det.FolderKind == ManagedRepo {
		t.Errorf("stray .abcd/ overclaims managed: kind = %q, want not %q", det.FolderKind, ManagedRepo)
	}
	if det.FolderKind != UnmanagedFolder {
		t.Errorf("kind = %q, want %q", det.FolderKind, UnmanagedFolder)
	}
}

func TestClassifyManagedRepoByMarker(t *testing.T) {
	setupHermetic(t)
	dir := t.TempDir()
	// A CLAUDE.md carrying a BEGIN fence is a strong managed signal even with
	// no .git and no .abcd dir.
	body := "# Project\n\n<!-- BEGIN ABCD -->\nx\n<!-- END ABCD -->\n"
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	det, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if det.FolderKind != ManagedRepo {
		t.Errorf("kind = %q, want %q", det.FolderKind, ManagedRepo)
	}
}

func TestSymlinkedMarkerNotAManagedSignal(t *testing.T) {
	setupHermetic(t)
	dir := t.TempDir()
	// Plant a real file elsewhere, symlink CLAUDE.md to it. A symlinked marker
	// doc must NOT promote the folder to managed.
	real := filepath.Join(t.TempDir(), "real.md")
	if err := os.WriteFile(real, []byte("<!-- BEGIN ABCD -->\nx\n<!-- END ABCD -->\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(dir, "CLAUDE.md")); err != nil {
		t.Fatal(err)
	}
	det, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if det.FolderKind != UnmanagedFolder {
		t.Errorf("kind = %q, want %q (symlinked marker must not count)", det.FolderKind, UnmanagedFolder)
	}
}

func TestDetectUnmanagedFolderShortCircuits(t *testing.T) {
	setupHermetic(t)
	dir := t.TempDir()
	det, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	// An unmanaged folder runs no other detection checks — no gaps at all.
	if len(det.Gaps) != 0 {
		t.Errorf("unmanaged folder produced gaps: %+v", det.Gaps)
	}
}

func TestDetectHookManifestGapOnBrokenPlugin(t *testing.T) {
	home := t.TempDir()
	pluginRoot := t.TempDir()
	// hooks dir exists (so the root validates) but hooks.json is absent.
	if err := os.MkdirAll(filepath.Join(pluginRoot, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", pluginRoot)
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("ABCD_BIN_TARGET", filepath.Join(t.TempDir(), "abcd"))

	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A marker block makes the folder genuinely managed (post iss-88 the .abcd/
	// dir alone no longer promotes), so the deeper gap checks run.
	body := "# Project\n\n<!-- BEGIN ABCD -->\nx\n<!-- END ABCD -->\n"
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	det, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !hasGap(det.Gaps, "hooks.manifest_missing") {
		t.Errorf("expected hooks.manifest_missing gap; got %+v", det.Gaps)
	}
	// It is a non-resolvable diagnostic, never actionable.
	for _, g := range det.Gaps {
		if g.ID == "hooks.manifest_missing" && (g.Resolvable || g.Required) {
			t.Errorf("hooks.manifest_missing must be non-resolvable/advisory: %+v", g)
		}
	}
}

func hasGap(gaps []Gap, id string) bool {
	for _, g := range gaps {
		if g.ID == id {
			return true
		}
	}
	return false
}

// TestShippedHookManifestVerifies pins the real hooks/hooks.json against the
// verifier, so the manifest and requiredHookCommand can never drift apart.
func TestShippedHookManifestVerifies(t *testing.T) {
	if reason := verifyHookManifest("../../.."); reason != "" {
		t.Fatalf("shipped hooks/hooks.json fails verification: %s", reason)
	}
}
