package ahoy

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/gittest"
)

// hooksJSONWithGuard is a manifest that also arms the execution-time guard.
const hooksJSONWithGuard = `{
  "hooks": {
    "UserPromptSubmit": [{"hooks": [{"type": "command", "command": "\"$CLAUDE_PLUGIN_ROOT/abcd\" hook prompt-router"}]}],
    "SessionStart":     [{"hooks": [{"type": "command", "command": "\"$CLAUDE_PLUGIN_ROOT/abcd\" hook prompt-router-reset"}]}],
    "PreCompact":       [{"hooks": [{"type": "command", "command": "\"$CLAUDE_PLUGIN_ROOT/abcd\" hook prompt-router-reset"}]}],
    "PreToolUse":       [{"matcher": "Bash", "hooks": [{"type": "command", "command": "\"$CLAUDE_PLUGIN_ROOT/abcd\" guard hook"}]}]
  }
}`

// managedRepoAt turns dir into a folder ahoy classifies as a managed repo, so the
// full detection pass runs over it.
func managedRepoAt(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := append(append(append([]byte(nil), markerBegin...), []byte("\nx\n")...), markerEnd...)
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), append(body, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestGuardHealthArmed is the good state: the manifest carries the PreToolUse
// entry, the plugin-root binary is present, and the registry loads. Everything a
// facilitator needs to believe the guard is actually on.
func TestGuardHealthArmed(t *testing.T) {
	_, pluginRoot := setupHermetic(t)
	if err := os.WriteFile(filepath.Join(pluginRoot, "hooks", "hooks.json"), []byte(hooksJSONWithGuard), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	managedRepoAt(t, dir)

	det, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	g := det.Guard
	if !g.HookInstalled || !g.BinaryReachable || !g.RegistryLoadable {
		t.Errorf("a fully armed guard must report healthy; got %+v", g)
	}
	if !g.Healthy() {
		t.Errorf("Healthy() must agree with the three checks; got %+v", g)
	}
	for _, gap := range det.Gaps {
		if strings.HasPrefix(gap.ID, "guard.") {
			t.Errorf("a healthy guard must emit no guard gap; got %q", gap.ID)
		}
	}
}

// TestGuardHealthHookNotInstalled is AC 1's visibility half: a manifest without
// the PreToolUse entry means commands run unguarded, and ahoy must say so rather
// than report a clean bill of health.
func TestGuardHealthHookNotInstalled(t *testing.T) {
	setupHermetic(t) // validHooksJSON: no PreToolUse entry
	dir := t.TempDir()
	managedRepoAt(t, dir)

	det, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if det.Guard.HookInstalled {
		t.Error("a manifest with no PreToolUse guard entry must report hook_installed=false")
	}
	if det.Guard.Healthy() {
		t.Error("a guard with no hook is not healthy")
	}
	if !hasGap(det.Gaps, "guard.hook_missing") {
		t.Errorf("a missing guard hook must surface as a gap; gaps = %v", gapIDs(det.Gaps))
	}
}

// TestGuardHealthBinaryUnreachable covers the fail-open case ahoy exists to make
// visible: the hook is installed but the binary it calls is gone, so every
// session runs unguarded behind a shim warning.
func TestGuardHealthBinaryUnreachable(t *testing.T) {
	_, pluginRoot := setupHermetic(t)
	if err := os.WriteFile(filepath.Join(pluginRoot, "hooks", "hooks.json"), []byte(hooksJSONWithGuard), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(pluginRoot, "abcd")); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	managedRepoAt(t, dir)

	det, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if det.Guard.BinaryReachable {
		t.Error("an absent plugin-root binary must report binary_reachable=false")
	}
	if !hasGap(det.Gaps, "guard.binary_unreachable") {
		t.Errorf("an unreachable guard binary must surface as a gap; gaps = %v", gapIDs(det.Gaps))
	}
	// spc-21: the binary is provisioned by hooks/bootstrap.sh at session start, so
	// "why is the guard down and what do I do" is answered in one place rather than
	// leaving a reader to guess that a reinstall is the only route.
	if !strings.Contains(det.Guard.Detail, "bootstrap.sh") {
		t.Errorf("the health reason must name the script that re-provisions the binary; detail = %q", det.Guard.Detail)
	}
	for _, g := range det.Gaps {
		if g.ID != "guard.binary_unreachable" {
			continue
		}
		if !strings.Contains(g.FixHint, "bootstrap.sh") {
			t.Errorf("the fix hint must name hooks/bootstrap.sh; fix hint = %q", g.FixHint)
		}
		if !strings.Contains(g.FixHint, "session") {
			t.Errorf("the fix hint must say the bootstrap runs at session start; fix hint = %q", g.FixHint)
		}
	}
}

// TestGuardHealthRegistryUnloadable is the third failure mode, refined by the
// fail-safe load (iss-2608261551087492): the hook runs, the binary runs, and the
// registry the repo committed does not parse — so the repo's OVERRIDES are
// dropped while the bundled hazards stay armed. A mild, expected state: still
// reported loudly (the broken file needs fixing), but never as "commands run
// unchecked", because they do not (iss-2608281222011114).
func TestGuardHealthRegistryUnloadable(t *testing.T) {
	_, pluginRoot := setupHermetic(t)
	if err := os.WriteFile(filepath.Join(pluginRoot, "hooks", "hooks.json"), []byte(hooksJSONWithGuard), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	managedRepoAt(t, dir)
	if err := os.WriteFile(filepath.Join(dir, ".abcd", "guard.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	det, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Load fell back to the bundled defaults, so a registry IS armed.
	if !det.Guard.RegistryLoadable {
		t.Error("a malformed .abcd/guard.json drops only the repo layer; the bundled registry is armed and registry_loadable must say so")
	}
	if det.Guard.Entries == 0 {
		t.Error("the bundled hazards remain armed on the fail-safe path, so entries must be non-zero")
	}
	if det.Guard.Detail == "" {
		t.Error("a dropped repo layer must carry a reason a human can act on")
	}
	if strings.Contains(det.Guard.Detail, "unchecked") {
		t.Errorf("the fail-safe load keeps commands checked; the reason must not claim otherwise: %q", det.Guard.Detail)
	}
	if !strings.Contains(det.Guard.Detail, "bundled hazards remain armed") {
		t.Errorf("the reason must say the bundled hazards remain armed; detail = %q", det.Guard.Detail)
	}
	if !det.Guard.RepoOverridesDropped {
		t.Error("the dropped repo layer must be reported as repo_overrides_dropped, not hidden inside a healthy report")
	}
	if !det.Guard.Healthy() {
		t.Errorf("commands are still checked against the bundled hazards, so the guard is healthy (degraded, not down); got %+v", det.Guard)
	}
	if !hasGap(det.Gaps, "guard.registry_unloadable") {
		t.Errorf("a repo guard file that does not load must still surface as a gap; gaps = %v", gapIDs(det.Gaps))
	}
	for _, g := range det.Gaps {
		if g.ID == "guard.registry_unloadable" && strings.Contains(g.Detail, "unchecked") {
			t.Errorf("the gap must not claim commands run unchecked when the bundled hazards are armed: %q", g.Detail)
		}
	}
}

// TestGuardHealthEmptyRegistryIsUnguarded pins the second half of the two-state
// model: an empty registry — no bundled layer to fall back to — is the only
// genuinely-unguarded registry state. The embedded defaults make it unreachable
// through guard.Load, so it is injected at the fold-in seam; a health check
// earns trust by stating what it would say if the impossible happened.
func TestGuardHealthEmptyRegistryIsUnguarded(t *testing.T) {
	var h GuardHealth
	reason := applyRegistryHealth(&h, guard.Loaded{Posture: guard.LoadUnavailable})

	if h.RegistryLoadable {
		t.Error("an empty registry must report registry_loadable=false")
	}
	if h.RepoOverridesDropped {
		t.Error("an empty registry is not the dropped-overrides state; nothing is armed at all")
	}
	if !strings.Contains(reason, "unchecked") {
		t.Errorf("no registry at all is the unguarded state and the reason must say so; got %q", reason)
	}

	gaps := registryGap(h)
	if len(gaps) != 1 || gaps[0].ID != "guard.registry_empty" {
		t.Fatalf("an empty registry must surface as the guard.registry_empty gap; got %v", gaps)
	}
	if !strings.Contains(gaps[0].Detail, "unchecked") {
		t.Errorf("the empty-registry gap must carry the unguarded reason; got %q", gaps[0].Detail)
	}
}

// TestGuardHealthRepoBrokenFoldIn pins the fold-in seam directly: an error
// alongside a non-empty registry (what guard.Load returns for a broken repo
// file since the fail-safe change) is the mild state, and an error-free load
// reports nothing.
func TestGuardHealthRepoBrokenFoldIn(t *testing.T) {
	var h GuardHealth
	reason := applyRegistryHealth(&h, guard.Loaded{Registry: guard.Defaults(), Posture: guard.LoadRepoDropped, Err: errors.New(".abcd/guard.json: boom")})
	if !h.RegistryLoadable || !h.RepoOverridesDropped {
		t.Errorf("error + non-empty registry is the dropped-overrides state; got %+v", h)
	}
	if reason != guardRepoOverridesDroppedReason {
		t.Errorf("the mild state must carry its one shared reason; got %q", reason)
	}

	var clean GuardHealth
	if reason := applyRegistryHealth(&clean, guard.Loaded{Registry: guard.Defaults()}); reason != "" {
		t.Errorf("a clean load must report nothing; got %q", reason)
	}
	if !clean.RegistryLoadable || clean.RepoOverridesDropped {
		t.Errorf("a clean load is fully loadable with nothing dropped; got %+v", clean)
	}
}

// TestGuardHealthUnresolvablePluginRootAssertsNothing is the honesty rule: when
// the plugin root cannot be resolved, the manifest is never opened and the binary
// is never looked for, so ahoy knows NOTHING about the hook wiring. Reporting
// "hook not installed" there would accuse a plugin install that may be perfectly
// armed — the state a dev build (`go run ./cmd/abcd`) is in every time.
func TestGuardHealthUnresolvablePluginRootAssertsNothing(t *testing.T) {
	setupHermetic(t)
	t.Setenv("ABCD_PLUGIN_ROOT", t.TempDir()) // no hooks/ layout: not a plugin root
	dir := t.TempDir()
	managedRepoAt(t, dir)

	det, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if det.PluginRootStatus != "missing" {
		t.Fatalf("fixture is wrong: plugin root resolved as %q", det.PluginRootStatus)
	}
	if det.Guard.PluginRootResolved {
		t.Error("an unresolvable plugin root must be reported as such, not folded into the other checks")
	}
	if hasGap(det.Gaps, "guard.hook_missing") || hasGap(det.Gaps, "guard.binary_unreachable") {
		t.Errorf("ahoy must not accuse a manifest it never opened; gaps = %v", gapIDs(det.Gaps))
	}
	if !strings.Contains(det.Guard.Detail, "plugin root") {
		t.Errorf("the reason must name what is actually unknown; detail = %q", det.Guard.Detail)
	}
	// The registry is a repo fact and stays answerable regardless of the plugin.
	if !det.Guard.RegistryLoadable {
		t.Error("the registry is readable without a plugin root and must still be reported")
	}
}

// TestGuardHealthDisabledIsReported is the committed escape hatch: a repo that
// deliberately turned the guard off is not broken, but the fact must still be on
// the status board — a disabled guard that looks armed is the failure mode.
func TestGuardHealthDisabledIsReported(t *testing.T) {
	_, pluginRoot := setupHermetic(t)
	if err := os.WriteFile(filepath.Join(pluginRoot, "hooks", "hooks.json"), []byte(hooksJSONWithGuard), 0o644); err != nil {
		t.Fatal(err)
	}
	// A real repository with the kill switch COMMITTED: an uncommitted one is
	// refused and the guard stays armed (iss-147).
	repo := gittest.NewRepo(t)
	dir := repo.Root()
	managedRepoAt(t, dir)
	repo.Write(".abcd/guard.json", `{"schema_version":1,"disabled":true,"entries":{}}`)
	repo.Commit("switch the guard off")

	det, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !det.Guard.RegistryLoadable {
		t.Error("a deliberately disabled registry still loads; it is not a fault")
	}
	if !det.Guard.Disabled {
		t.Error("a committed kill switch must be reported by ahoy, not hidden")
	}
}
