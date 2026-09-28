package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// devInstallOpts is a fully-specified, non-interactive dev-mode install: adopt
// the repo, approve every category, pin the config, and request track-latest.
func devInstallOpts() InstallOptions {
	opts := installOpts()
	opts.Dev = true
	return opts
}

// detectInstallModeSignal returns the install_mode detection signal for repo —
// the honest, filesystem-derived mode that status reports.
func detectInstallModeSignal(t *testing.T, repo string) string {
	t.Helper()
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	m, _ := det.Signals["install_mode"].(string)
	return m
}

// configHasInstallSection reports whether config.json carries an `install`
// section at all. The install mode is derived from the filesystem, never
// recorded in the tracked repo config, so this must always be false — the
// regression pin that a dev install never dirties config.json.
func configHasInstallSection(t *testing.T, repo string) bool {
	t.Helper()
	cfg, err := readConfig(repo)
	if err != nil {
		t.Fatalf("readConfig: %v", err)
	}
	_, ok := cfg["install"]
	return ok
}

// TestDevInstallWritesShimAndSurfacesMode is coverage (a): a --dev install writes
// the rebuild-then-exec shim (not a symlink) at the PATH target, the shim carries
// the loud-fail path, and detection surfaces the mode from the installed shim
// (never recorded in the tracked repo config).
func TestDevInstallWritesShimAndSurfacesMode(t *testing.T) {
	_, pluginRoot := setupHermetic(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := Install(repo, devInstallOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "clean" {
		t.Fatalf("dev install status = %q (remaining=%v), want clean", res.Status, res.Remaining)
	}

	target := binTarget()
	fi, err := os.Lstat(target)
	if err != nil {
		t.Fatalf("PATH target not created: %v", err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("dev mode installed a symlink, want a regular shim file")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	shim := string(data)
	for _, want := range []string{
		devShimMarker,     // recognisable as our shim
		"go build",        // (a) rebuilds from source tip
		"exec ",           // (b) execs the fresh binary
		"refusing to run", // (c) loud-fail message
		"exit 1",          // (c) non-zero exit on build failure
		pluginRoot,        // (d) baked absolute source-repo path
	} {
		if !strings.Contains(shim, want) {
			t.Errorf("shim missing %q:\n%s", want, shim)
		}
	}
	// The loud-fail branch must precede the exec so a broken build never falls
	// through to a stale binary.
	if strings.Index(shim, "exit 1") > strings.LastIndex(shim, "exec ") {
		t.Errorf("exec precedes the loud-fail exit; a broken build could exec a stale binary:\n%s", shim)
	}

	// Detection surfaces the mode honestly from the installed shim — and the
	// tracked repo config is never touched with an install section.
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if m, _ := det.Signals["install_mode"].(string); m != "dev (tip build)" {
		t.Errorf("install_mode signal = %q, want %q", m, "dev (tip build)")
	}
	if configHasInstallSection(t, repo) {
		t.Errorf("dev install wrote an install section into tracked config.json; want none")
	}
	// The dev shim is a valid install, never flagged as a foreign occupant.
	if hasGap(det.Gaps, "symlink.foreign") {
		t.Errorf("dev shim wrongly flagged symlink.foreign: %+v", det.Gaps)
	}
}

// TestNormalInstallRegressionPin is coverage (b): a plain normal install
// creates the pinned entry — the owned copy of the verified release artefact —
// and records NO install section.
func TestNormalInstallRegressionPin(t *testing.T) {
	setupHermetic(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	assertOwnedCopyAt(t, binTarget())
	if configHasInstallSection(t, repo) {
		t.Errorf("normal install wrote an install section; want none (regression)")
	}
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if m, _ := det.Signals["install_mode"].(string); m != "pinned" {
		t.Errorf("install_mode signal = %q, want pinned", m)
	}
}

// TestInstallPinnedToDevTransition is coverage (c), one direction: a normal
// install followed by --dev applies-as-update, replacing the symlink with the
// shim and echoing the mode change.
func TestInstallPinnedToDevTransition(t *testing.T) {
	setupHermetic(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	res, err := Install(repo, devInstallOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "clean" {
		t.Fatalf("transition status = %q, want clean", res.Status)
	}
	if !containsChange(res.Changes, "install_mode: pinned -> dev") {
		t.Errorf("changes = %v, want an install_mode pinned -> dev echo", res.Changes)
	}
	fi, err := os.Lstat(binTarget())
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Errorf("transition left a symlink, want the dev shim")
	}
	if got := detectInstallModeSignal(t, repo); got != "dev (tip build)" {
		t.Errorf("install_mode signal = %q, want %q", got, "dev (tip build)")
	}
	// Idempotent: a second --dev install is a no-op.
	res2, err := Install(repo, devInstallOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res2.Status != "already_up_to_date" {
		t.Errorf("re-dev-install status = %q, want already_up_to_date", res2.Status)
	}
}

// TestInstallDevToPinnedTransition is coverage (c), the other direction: a --dev
// install followed by a plain install applies-as-update, restoring the pinned
// owned copy.
func TestInstallDevToPinnedTransition(t *testing.T) {
	setupHermetic(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(repo, devInstallOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "clean" {
		t.Fatalf("transition status = %q, want clean", res.Status)
	}
	if !containsChange(res.Changes, "install_mode: dev -> pinned") {
		t.Errorf("changes = %v, want an install_mode dev -> pinned echo", res.Changes)
	}
	assertOwnedCopyAt(t, binTarget())
	if got := detectInstallModeSignal(t, repo); got != "pinned" {
		t.Errorf("install_mode signal = %q, want pinned", got)
	}
}

// TestUninstallRemovesDevShim pins that uninstall recognises and removes the
// dev shim (it is ours), not just the pinned symlink.
func TestUninstallRemovesDevShim(t *testing.T) {
	setupHermetic(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(repo, devInstallOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	receipt, err := Uninstall(repo, "")
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Symlink.Removed {
		t.Errorf("uninstall did not remove the dev shim: %+v", receipt.Symlink)
	}
	if _, err := os.Lstat(binTarget()); !os.IsNotExist(err) {
		t.Errorf("dev shim still present after uninstall: %v", err)
	}
}

// TestDevInstallRespectsDeclinedConfigChange pins the consent boundary: the PATH
// entry is a ConfigChange, so a fresh --dev install must NOT write the shim when
// that category is declined — exactly as a plain install honours the decline.
func TestDevInstallRespectsDeclinedConfigChange(t *testing.T) {
	setupHermetic(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	adopt := true
	opts := InstallOptions{
		Adopt: &adopt,
		Dev:   true,
		// Approve every resolvable category EXCEPT ConfigChange (the PATH entry).
		ApprovedCategories: map[GapCategory]bool{
			SafeAutocreate: true,
			PluginOwned:    true,
			UserState:      true,
			Dependency:     true,
		},
	}
	if _, err := Install(repo, opts, RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(binTarget()); !os.IsNotExist(err) {
		t.Errorf("--dev wrote the PATH shim despite a declined ConfigChange approval (consent bypass): %v", err)
	}
}

func containsChange(changes []string, want string) bool {
	for _, c := range changes {
		if c == want {
			return true
		}
	}
	return false
}
