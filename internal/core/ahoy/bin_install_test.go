package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupUserScope drives the DEFAULT (no-sudo) install location: HOME is
// redirected, the ABCD_BIN_TARGET test override is cleared so binTarget()
// resolves to ~/.local/bin/abcd, and PATH is exactly the dirs given — so a test
// can state, rather than inherit, what the machine has on PATH.
func setupUserScope(t *testing.T, pathDirs ...string) (home, pluginRoot string) {
	t.Helper()
	home, pluginRoot = setupHermetic(t)
	t.Setenv("ABCD_BIN_TARGET", "")
	t.Setenv("PATH", strings.Join(pathDirs, string(os.PathListSeparator)))
	return home, pluginRoot
}

// managedRepo returns a temp dir that classifies as a managed repo (a marker
// block fired), so the deeper gap checks run.
func managedRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	body := "# Project\n\n<!-- BEGIN ABCD -->\nx\n<!-- END ABCD -->\n"
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// linkOwned plants an owned install (a symlink to the plugin binary) at path.
func linkOwned(t *testing.T, path, pluginRoot string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(pluginBinaryPath(pluginRoot), path); err != nil {
		t.Fatal(err)
	}
}

func gapByID(gaps []Gap, id string) *Gap {
	for i := range gaps {
		if gaps[i].ID == id {
			return &gaps[i]
		}
	}
	return nil
}

// TestDetectAdoptsOwnedInstallOnPath is iss-171's headline defect: a working
// ~/.local/bin/abcd — the field-standard single-user location — must read as
// installed, not as symlink.missing, while the detector is itself running as
// abcd from PATH.
func TestDetectAdoptsOwnedInstallOnPath(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	linkOwned(t, filepath.Join(binDir, "abcd"), pluginRoot)

	det, err := Detect(managedRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	if hasGap(det.Gaps, "symlink.missing") {
		t.Errorf("a working %s reported symlink.missing: %+v", filepath.Join(binDir, "abcd"), det.Gaps)
	}
	if m, _ := det.Signals["install_mode"].(string); m != "pinned" {
		t.Errorf("install_mode = %q, want pinned", m)
	}
}

// TestDetectAdoptsOwnedInstallAnywhereOnPath proves the detector scans PATH
// rather than recognising one blessed target: an owned install in any PATH
// directory counts.
func TestDetectAdoptsOwnedInstallAnywhereOnPath(t *testing.T) {
	_, pluginRoot := setupUserScope(t)
	other := filepath.Join(t.TempDir(), "opt", "bin")
	t.Setenv("PATH", other)
	linkOwned(t, filepath.Join(other, "abcd"), pluginRoot)

	det, err := Detect(managedRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	if hasGap(det.Gaps, "symlink.missing") {
		t.Errorf("owned install at %s reported symlink.missing: %+v", other, det.Gaps)
	}
}

// TestDetectDanglingOwnedSymlinkIsItsOwnGap pins the shadowing failure mode: an
// owned symlink whose target has gone is NOT a healthy install, and it is not
// silence either — it is its own gap.
func TestDetectDanglingOwnedSymlinkIsItsOwnGap(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	linkOwned(t, filepath.Join(binDir, "abcd"), pluginRoot)
	if err := os.Remove(pluginBinaryPath(pluginRoot)); err != nil {
		t.Fatal(err)
	}

	det, err := Detect(managedRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	g := gapByID(det.Gaps, "symlink.dangling")
	if g == nil {
		t.Fatalf("dangling owned symlink produced no symlink.dangling gap: %+v", det.Gaps)
	}
	if !g.Required {
		t.Errorf("symlink.dangling must be required: %+v", g)
	}
	if m, _ := det.Signals["install_mode"].(string); m == "pinned" {
		t.Errorf("a dangling symlink reported install_mode=pinned")
	}
}

// linkStranded plants the entry a plugin update leaves behind: an abcd symlink
// into a SIBLING plugin cache dir that no longer exists. Every plugin update
// produces this state — the harness provisions a fresh cache dir and deletes
// the old one, and the PATH entry keeps pointing into the old one (iss-345).
func linkStranded(t *testing.T, path, pluginRoot string) {
	t.Helper()
	oldRoot := filepath.Join(filepath.Dir(pluginRoot), "0badc0ffee12")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(oldRoot, "abcd"), path); err != nil {
		t.Fatal(err)
	}
}

// TestDetectStrandedEntryIsDanglingNotForeign is iss-345: the entry a plugin
// update strands must classify as OURS (dangling), not as a foreign occupant
// the user is told to resolve by hand.
func TestDetectStrandedEntryIsDanglingNotForeign(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	linkStranded(t, filepath.Join(binDir, "abcd"), pluginRoot)

	det, err := Detect(managedRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	if g := gapByID(det.Gaps, "symlink.dangling"); g == nil {
		t.Fatalf("a stranded plugin-update entry produced no symlink.dangling gap: %+v", det.Gaps)
	}
	if hasGap(det.Gaps, "symlink.foreign") {
		t.Errorf("a stranded entry of our own was described as foreign: %+v", det.Gaps)
	}
}

// TestInstallRelativePluginRootWritesResolvableEntry is gh-334: a RELATIVE
// ABCD_PLUGIN_ROOT (the variable abcd's own fix hint tells users to set, with no
// absoluteness contract) must not defeat the anti-dangling guard. The guard
// stats the symlink source against the process CWD while the kernel resolves the
// same relative string against the LINK's directory (~/.local/bin), so a relative
// root wrote a dangling PATH entry (nested root) or a self-referential
// abcd -> abcd loop (root "."), reported it as written, then reclassified its own
// link as FOREIGN so neither doctor nor uninstall could repair it. The entry abcd
// writes must resolve to the real plugin binary and be recognised as owned.
func TestInstallRelativePluginRootWritesResolvableEntry(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)

	// Point ABCD_PLUGIN_ROOT at the very same plugin root, but by a RELATIVE path
	// from the process CWD — the shape a `cd abcd-cli && ABCD_PLUGIN_ROOT=.` user
	// produces. resolvePluginRoot takes it first in the ladder.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(cwd, pluginRoot)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.IsAbs(rel) {
		t.Fatalf("test setup: %q is not relative", rel)
	}
	t.Setenv("ABCD_PLUGIN_ROOT", rel)

	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(binDir, "abcd")
	// The link the guard was supposed to protect: it must resolve to an existing
	// file when the kernel resolves it (against the link's own directory), not
	// dangle or loop. os.Stat follows the link exactly as the kernel would.
	if fi, serr := os.Stat(target); serr != nil {
		dest, _ := os.Readlink(target)
		t.Fatalf("install wrote a broken PATH entry from a relative plugin root: %s -> %s does not resolve (%v); notes: %v",
			target, dest, serr, res.Notes)
	} else if !fi.Mode().IsRegular() {
		t.Fatalf("PATH entry %s did not resolve to a regular file (mode %v)", target, fi.Mode())
	}

	// And abcd must still own the link it wrote — not misclassify it as foreign,
	// which is what wedges doctor and uninstall.
	det, err := Detect(managedRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	if hasGap(det.Gaps, "symlink.foreign") {
		t.Errorf("abcd misclassified its own relative-root link as foreign: %+v", det.Gaps)
	}
	if m, _ := det.Signals["install_mode"].(string); m != "pinned" {
		t.Errorf("install_mode = %q, want pinned (a mis-owned link reports '')", m)
	}
}

// TestInstallRepointsEntryStrandedByPluginUpdate is iss-345's repair half: with
// the fresh plugin root holding a binary, install repoints the stranded entry
// in place — no refusal, and no second entry planted at the default location.
// The entry deliberately lives OFF ~/.local/bin so adopt-in-place and
// write-the-default cannot land on the same path and mask each other.
func TestInstallRepointsEntryStrandedByPluginUpdate(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	other := filepath.Join(t.TempDir(), "opt", "bin")
	t.Setenv("PATH", other)
	link := filepath.Join(other, "abcd")
	linkStranded(t, link, pluginRoot)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	dest, rerr := os.Readlink(link)
	if rerr != nil {
		t.Fatalf("the stranded entry is no longer a symlink: %v", rerr)
	}
	if resolveSymlinkDest(link, dest) != resolvePath(pluginBinaryPath(pluginRoot)) {
		t.Errorf("entry was not repointed at the fresh plugin binary: %s -> %s (notes: %v)", link, dest, res.Notes)
	}
	if _, err := os.Lstat(filepath.Join(home, ".local", "bin", "abcd")); !os.IsNotExist(err) {
		t.Errorf("install planted a second entry at ~/.local/bin beside the repointed one: %v", err)
	}
}

// TestUninstallRemovesStrandedEntry is iss-345's uninstall half: the
// symlink.dangling fix hint names `ahoy uninstall` as the remedy, so uninstall
// must recognise the stranded entry as ours — not report it foreign and leave
// the dead link shadowing PATH with no remedy anywhere.
func TestUninstallRemovesStrandedEntry(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	link := filepath.Join(binDir, "abcd")
	linkStranded(t, link, pluginRoot)

	receipt, err := Uninstall(managedRepo(t), "")
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Symlink.Removed {
		t.Fatalf("uninstall left the stranded entry in place: %+v", receipt.Symlink)
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Errorf("the stranded entry survived uninstall: %v", err)
	}
}

// TestDevInstallOnStrandedEntryStillNeedsApproval: a stranded entry now
// classifies as owned, which made modeWouldChange force the --dev shim write
// PAST a declined ConfigChange approval (iss-345 security review). The entry
// must flow through the symlink.dangling gap instead, so consent still gates
// the write.
func TestDevInstallOnStrandedEntryStillNeedsApproval(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	link := filepath.Join(binDir, "abcd")
	linkStranded(t, link, pluginRoot)
	before, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	adopt := true
	opts := InstallOptions{
		Adopt: &adopt,
		Dev:   true,
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
	after, rerr := os.Readlink(link)
	if rerr != nil || after != before {
		t.Errorf("--dev rewrote the stranded entry despite a declined ConfigChange approval: %q -> %q (%v)", before, after, rerr)
	}
}

// TestStrandedOwnershipScope pins both halves of strandedSiblingDest's "and no
// further" contract: a RELATIVE dest into a dead sibling root is ours, and a
// dangling dest anywhere else stays foreign — so a later widening of the
// predicate fails here, not in the field.
func TestStrandedOwnershipScope(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	link := filepath.Join(binDir, "abcd")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}

	rel, err := filepath.Rel(binDir, filepath.Join(filepath.Dir(pluginRoot), "0badc0ffee12", "abcd"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(rel, link); err != nil {
		t.Fatal(err)
	}
	if kind := classifyBinTarget(link, pluginRoot); kind != binTargetOwnedSymlink {
		t.Errorf("relative stranded dest classified %v, want owned", kind)
	}

	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(t.TempDir(), "gone", "abcd"), link); err != nil {
		t.Fatal(err)
	}
	if kind := classifyBinTarget(link, pluginRoot); kind != binTargetForeign {
		t.Errorf("dangling non-sibling dest classified %v, want foreign", kind)
	}
}

// TestDetectBinDirNotOnPathIsALoudGap pins the script-first remedy: an install
// directory that is not on PATH is its own loud gap carrying the one-line export
// fix, and abcd never offers to patch a shell profile (the gap is not resolvable
// by an apply step).
func TestDetectBinDirNotOnPathIsALoudGap(t *testing.T) {
	home, pluginRoot := setupUserScope(t, filepath.Join(t.TempDir(), "elsewhere"))
	linkOwned(t, filepath.Join(home, ".local", "bin", "abcd"), pluginRoot)

	det, err := Detect(managedRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	g := gapByID(det.Gaps, "path.bin_dir_not_on_path")
	if g == nil {
		t.Fatalf("no path.bin_dir_not_on_path gap: %+v", det.Gaps)
	}
	if !g.Required {
		t.Errorf("the PATH gap must be required (loud): %+v", g)
	}
	if g.Resolvable {
		t.Errorf("the PATH gap must NOT be resolvable — abcd prints the fix, it never patches a shell profile: %+v", g)
	}
	if want := `export PATH="$HOME/.local/bin:$PATH"`; !strings.Contains(g.FixHint, want) {
		t.Errorf("fix hint = %q, want it to carry %q", g.FixHint, want)
	}
}

// TestGapTextCarriesNoAbsoluteHomePath pins the receipt-hygiene half of the
// hand-off from iss-177: the moment the default install location moves under
// $HOME, any gap line embedding it carries the developer's username. Gap text
// renders user-scope paths in tilde form.
func TestGapTextCarriesNoAbsoluteHomePath(t *testing.T) {
	home, _ := setupUserScope(t, filepath.Join(t.TempDir(), "elsewhere"))

	det, err := Detect(managedRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	sawTilde := false
	for _, g := range det.Gaps {
		for _, field := range []string{g.Title, g.Detail, g.FixHint} {
			if strings.Contains(field, home) {
				t.Errorf("gap %s leaks the absolute home path %q: %q", g.ID, home, field)
			}
		}
		if strings.Contains(g.Detail, "~/.local/bin/abcd") || strings.Contains(g.Title, "~/.local/bin") {
			sawTilde = true
		}
	}
	if !sawTilde {
		t.Errorf("no gap named the default install location in tilde form: %+v", det.Gaps)
	}
}

// TestInstallDefaultsToUserLocalBin is decision 5 of the install-experience
// plan: install writes ~/.local/bin/abcd, creating the directory, with no
// privilege escalation anywhere.
func TestInstallDefaultsToUserLocalBin(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(binDir, "abcd")
	fi, err := os.Lstat(target)
	if err != nil {
		t.Fatalf("install did not create %s: %v", target, err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("install wrote a regular file, want a symlink")
	}
	dest, _ := os.Readlink(target)
	if resolveSymlinkDest(target, dest) != resolvePath(pluginBinaryPath(pluginRoot)) {
		t.Errorf("symlink dest = %q, want %q", dest, pluginBinaryPath(pluginRoot))
	}
	// The write is recorded on the receipt through the same note seam as every
	// other apply step, and the receipt scrub owned by iss-177 renders a
	// user-scope write home-relative — so the entry appears in tilde form.
	if !containsPath(res.Writes, "~/.local/bin/abcd") {
		t.Errorf("the PATH entry was not recorded on the receipt: %v", res.Writes)
	}
	if m, _ := detectSignal(t, repo, "install_mode").(string); m != "pinned" {
		t.Errorf("install_mode = %q, want pinned", m)
	}
	_ = home
}

func containsPath(paths []string, want string) bool {
	for _, p := range paths {
		if p == want {
			return true
		}
	}
	return false
}

// detectSignal returns one detection signal for repo.
func detectSignal(t *testing.T, repo, key string) any {
	t.Helper()
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	return det.Signals[key]
}

// TestInstallAdoptsOwnedInstallInPlace proves install never plants a second
// install beside a working one: an owned entry already on PATH is adopted where
// it stands.
func TestInstallAdoptsOwnedInstallInPlace(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	other := filepath.Join(t.TempDir(), "opt", "bin")
	t.Setenv("PATH", other)
	linkOwned(t, filepath.Join(other, "abcd"), pluginRoot)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(home, ".local", "bin", "abcd")); !os.IsNotExist(err) {
		t.Errorf("install planted a second entry at ~/.local/bin beside the adopted one: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(other, "abcd")); err != nil {
		t.Errorf("install disturbed the adopted install: %v", err)
	}
}

// TestInstallRefusesToCreateDanglingSymlink is the shadowing refusal: with no
// binary at <plugin-root>/abcd, install must NOT write a symlink at all — a
// dangling link on PATH shadows whatever else would have answered.
func TestInstallRefusesToCreateDanglingSymlink(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	if err := os.Remove(pluginBinaryPath(pluginRoot)); err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(binDir, "abcd")); !os.IsNotExist(err) {
		t.Fatalf("install created a symlink to a non-existent target: %v", err)
	}
	joined := strings.Join(res.Notes, "\n")
	if !strings.Contains(joined, "does not exist") {
		t.Errorf("the refusal was silent; notes = %v", res.Notes)
	}
}

// TestInstallBinDirUnwritableFailsLoudly pins the system-wide path: an explicit
// --bin-dir abcd cannot write to is an error, never a silent skip and never a
// privilege escalation.
func TestInstallBinDirUnwritableFailsLoudly(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: every directory is writable, so the refusal cannot be observed")
	}
	setupUserScope(t)
	ro := filepath.Join(t.TempDir(), "ro")
	if err := os.Mkdir(ro, 0o555); err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	opts := installOpts()
	opts.BinDir = ro

	_, err := Install(repo, opts, RefusingPrompter{})
	if err == nil {
		t.Fatalf("an unwritable --bin-dir was accepted silently")
	}
	msg := err.Error()
	for _, want := range []string{"not writable", "privilege"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error = %q, want it to carry %q", msg, want)
		}
	}
	if _, err := os.Lstat(filepath.Join(ro, "abcd")); !os.IsNotExist(err) {
		t.Errorf("something was written into the unwritable dir: %v", err)
	}
}

// TestInstallBinDirWritableInstallsThere pins the opt-in: an explicit, writable
// --bin-dir is where the entry lands.
func TestInstallBinDirWritableInstallsThere(t *testing.T) {
	_, pluginRoot := setupUserScope(t)
	dir := filepath.Join(t.TempDir(), "opt", "bin")
	t.Setenv("PATH", dir)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	opts := installOpts()
	opts.BinDir = dir

	if _, err := Install(repo, opts, RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "abcd")
	dest, err := os.Readlink(target)
	if err != nil {
		t.Fatalf("--bin-dir install did not create %s: %v", target, err)
	}
	if resolveSymlinkDest(target, dest) != resolvePath(pluginBinaryPath(pluginRoot)) {
		t.Errorf("symlink dest = %q, want %q", dest, pluginBinaryPath(pluginRoot))
	}
}

// TestUninstallReceiptCarriesNoAbsoluteHomePath is the second half of the
// iss-177 hand-off: `ahoy uninstall` prints the receipt's symlink target
// verbatim, so the target must be rendered in tilde form once the default
// location lives under $HOME.
func TestUninstallReceiptCarriesNoAbsoluteHomePath(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	linkOwned(t, filepath.Join(binDir, "abcd"), pluginRoot)

	receipt, err := Uninstall(managedRepo(t), "")
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Symlink.Removed {
		t.Fatalf("uninstall did not remove the owned install: %+v", receipt.Symlink)
	}
	if strings.Contains(receipt.Symlink.Target, home) {
		t.Errorf("receipt leaks the absolute home path: %q", receipt.Symlink.Target)
	}
	if want := "~/.local/bin/abcd"; receipt.Symlink.Target != want {
		t.Errorf("receipt target = %q, want %q", receipt.Symlink.Target, want)
	}
}

// ---------------------------------------------------------------------------
// review round: what shadows what, and what says so
// ---------------------------------------------------------------------------

// writeForeign plants a binary abcd does not own at path.
func writeForeign(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho stale\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func notesJoined(notes []string) string { return strings.Join(notes, "\n") }

// TestDetectShadowedByForeignCopyOnPath is the population the old README
// one-liner created: it COPIED the binary to /usr/local/bin, and a copy is a
// regular file, so it classifies foreign, is never adopted, and a correct entry
// installed behind it is never what runs. Reporting that as a clean pinned
// install is the lie this gap ends.
func TestDetectShadowedByForeignCopyOnPath(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	stale := filepath.Join(t.TempDir(), "usr-local-bin")
	t.Setenv("PATH", stale+string(os.PathListSeparator)+binDir)
	writeForeign(t, filepath.Join(stale, "abcd"))
	linkOwned(t, filepath.Join(binDir, "abcd"), pluginRoot)

	det, err := Detect(managedRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	g := gapByID(det.Gaps, "symlink.shadowed")
	if g == nil {
		t.Fatalf("a stale copy earlier on PATH produced no symlink.shadowed gap: %+v", det.Gaps)
	}
	if !g.Required {
		t.Errorf("symlink.shadowed must be required: %+v", g)
	}
	if !strings.Contains(g.Detail, filepath.Join(stale, "abcd")) {
		t.Errorf("the gap does not name the occupant: %q", g.Detail)
	}
	if m, _ := det.Signals["install_mode"].(string); m == "pinned" {
		t.Errorf("install_mode = %q — a shadowed entry must not report as a healthy pinned install", m)
	}
}

// TestInstallBehindForeignCopyReportsShadowing is the same defect at the moment
// it is created: install writes a correct entry BEHIND the stale copy, and must
// say so on the result. A clean status with no note is how a machine ends up
// running the old binary forever.
func TestInstallBehindForeignCopyReportsShadowing(t *testing.T) {
	home, _ := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	stale := filepath.Join(t.TempDir(), "usr-local-bin")
	t.Setenv("PATH", stale+string(os.PathListSeparator)+binDir)
	writeForeign(t, filepath.Join(stale, "abcd"))
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(binDir, "abcd")); err != nil {
		t.Fatalf("install wrote no entry at all: %v", err)
	}
	if !strings.Contains(notesJoined(res.Notes), filepath.Join(stale, "abcd")) {
		t.Errorf("install did not report the shadowing binary; notes = %v", res.Notes)
	}
}

// TestDetectDanglingForeignEntryIsDescribed pins the half the first cut missed:
// dangling was computed only for entries abcd owns, so a foreign `abcd` pointing
// at nothing was reported as nothing at all — while still occupying the name.
func TestDetectDanglingForeignEntryIsDescribed(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	stale := filepath.Join(t.TempDir(), "stale")
	t.Setenv("PATH", stale+string(os.PathListSeparator)+binDir)
	if err := os.MkdirAll(stale, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(t.TempDir(), "gone", "abcd"), filepath.Join(stale, "abcd")); err != nil {
		t.Fatal(err)
	}
	linkOwned(t, filepath.Join(binDir, "abcd"), pluginRoot)

	det, err := Detect(managedRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	g := gapByID(det.Gaps, "symlink.shadowed")
	if g == nil {
		t.Fatalf("a dangling foreign entry produced no gap: %+v", det.Gaps)
	}
	if !strings.Contains(g.Detail, "target is gone") {
		t.Errorf("the gap does not describe the occupant as dangling: %q", g.Detail)
	}
}

// TestInstallForeignOccupantAtBinDirIsRefusedLoudly: the foreign bail-out wrote
// nothing and recorded nothing, and symlink.foreign is advisory so it is
// filtered out of Remaining — under an explicit --bin-dir the run reported a
// clean status while the remedy text pointed at a location the user overrode.
func TestInstallForeignOccupantAtBinDirIsRefusedLoudly(t *testing.T) {
	setupUserScope(t)
	dir := filepath.Join(t.TempDir(), "opt", "bin")
	t.Setenv("PATH", dir)
	writeForeign(t, filepath.Join(dir, "abcd"))
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	opts := installOpts()
	opts.BinDir = dir

	res, err := Install(repo, opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	joined := notesJoined(res.Notes)
	if !strings.Contains(joined, dir) || !strings.Contains(joined, "does not own") {
		t.Errorf("the foreign occupant was refused in silence; notes = %v", res.Notes)
	}
}

// TestInstallFreshPathGapIsCarriedOnNotes: "~/.local/bin is not on PATH" is a
// required, non-resolvable gap, so it is excluded from Remaining, absent from
// InstallResult, and printed only by `doctor` — which cannot be run by name on
// the very machine where abcd is not on PATH. Install carries it.
func TestInstallFreshPathGapIsCarriedOnNotes(t *testing.T) {
	home, _ := setupUserScope(t, filepath.Join(t.TempDir(), "elsewhere"))
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	joined := notesJoined(res.Notes)
	if !strings.Contains(joined, `export PATH="$HOME/.local/bin:$PATH"`) {
		t.Errorf("install did not carry the PATH fix; notes = %v", res.Notes)
	}
	if strings.Contains(joined, home) {
		t.Errorf("a note leaks the absolute home path: %v", res.Notes)
	}
}

// TestInstallDanglingEntryWithMissingBinaryRefusesLoudly: the owned-symlink
// early return preceded the source check, so a dangling entry plus a missing
// plugin binary produced status=partial with no note and no reason anywhere.
func TestInstallDanglingEntryWithMissingBinaryRefusesLoudly(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	linkOwned(t, filepath.Join(binDir, "abcd"), pluginRoot)
	if err := os.Remove(pluginBinaryPath(pluginRoot)); err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(notesJoined(res.Notes), "does not exist") {
		t.Errorf("a dangling entry with no binary to repoint at was silent; notes = %v", res.Notes)
	}
}

// TestInstallBinDirOffPathIsDescribedAndRemovable: an entry written outside PATH
// by an explicit --bin-dir is invisible to a PATH scan, so it was written and
// never described, and uninstall could not reach it. Install describes the
// directory it actually wrote, and uninstall takes the same flag.
func TestInstallBinDirOffPathIsDescribedAndRemovable(t *testing.T) {
	setupUserScope(t, filepath.Join(t.TempDir(), "elsewhere"))
	dir := filepath.Join(t.TempDir(), "opt", "bin")
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	opts := installOpts()
	opts.BinDir = dir

	res, err := Install(repo, opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(notesJoined(res.Notes), `export PATH="`+dir+`:$PATH"`) {
		t.Errorf("the --bin-dir entry was written without describing its reachability; notes = %v", res.Notes)
	}
	receipt, err := Uninstall(repo, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Symlink.Removed {
		t.Errorf("uninstall could not reach the --bin-dir entry: %+v", receipt.Symlink)
	}
}

// ---------------------------------------------------------------------------
// iss-2609100506256636: a dangling entry abcd does not own is not a wall
// ---------------------------------------------------------------------------

// linkDanglingForeign plants the exact entry the field report found on a real
// machine: an `abcd` symlink at the PATH target whose destination is a deleted
// Go-test temp tree. The destination is NOT a sibling of the plugin root, so
// strandedSiblingDest does not claim it, and it does not resolve to the plugin
// binary either — the two rungs that make a dangling link "ours". It is
// therefore the one shape the classifier had no rung for: a link abcd cannot
// prove it wrote, that resolves to nothing at all.
func linkDanglingForeign(t *testing.T, path string) string {
	t.Helper()
	dest := filepath.Join(t.TempDir(), "TestAhoyInstallAcceptsPipedAnswers", "001", "abcd")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(dest, path); err != nil {
		t.Fatal(err)
	}
	return dest
}

// TestDetectDanglingEntryAtTargetIsRepairableNotForeign is
// iss-2609100506256636's detection half. A link whose target does not exist
// runs nothing, shadows every later PATH entry, and cannot be anyone's working
// install — so refusing to clear it is a permanent wall with no supported way
// past it: symlink.foreign is `resolvable: false`, so `ahoy install` can never
// finish on that machine.
func TestDetectDanglingEntryAtTargetIsRepairableNotForeign(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	_ = pluginRoot
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	linkDanglingForeign(t, filepath.Join(binDir, "abcd"))

	det, err := Detect(managedRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	if hasGap(det.Gaps, "symlink.foreign") {
		t.Errorf("a link that resolves to nothing was reported as a foreign occupant to hand-resolve: %+v", det.Gaps)
	}
	g := gapByID(det.Gaps, "symlink.dangling")
	if g == nil {
		t.Fatalf("a dangling entry at the PATH target produced no symlink.dangling gap: %+v", det.Gaps)
	}
	if !g.Resolvable {
		t.Errorf("symlink.dangling at the target must be resolvable, or install can never finish: %+v", g)
	}
	if m, _ := det.Signals["install_mode"].(string); m == "pinned" {
		t.Errorf("a dangling entry reported install_mode=pinned")
	}
}

// TestInstallReplacesDanglingEntryAtTarget is the repair half: the install must
// complete and leave an entry that resolves, rather than reporting nothing
// written and no reason why.
func TestInstallReplacesDanglingEntryAtTarget(t *testing.T) {
	home, _ := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	target := filepath.Join(binDir, "abcd")
	linkDanglingForeign(t, target)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	fi, serr := os.Stat(target)
	if serr != nil {
		dest, _ := os.Readlink(target)
		t.Fatalf("install left the dangling entry in place: %s -> %s (%v); notes = %v", target, dest, serr, res.Notes)
	}
	if !fi.Mode().IsRegular() {
		t.Fatalf("PATH entry %s did not resolve to a regular file (mode %v)", target, fi.Mode())
	}
}

// TestDetectLiveForeignSymlinkStaysForeign is the negative control, and the
// half the fix must NOT weaken. A symlink that RESOLVES is somebody's working
// install — an attacker's plant included — and abcd still refuses to clobber
// it. Danglingness is the whole discriminator: "runs nothing" versus "runs
// something", never a guess about who wrote the link.
func TestDetectLiveForeignSymlinkStaysForeign(t *testing.T) {
	home, _ := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	elsewhere := filepath.Join(t.TempDir(), "somebody-elses-abcd")
	writeForeign(t, elsewhere)
	target := filepath.Join(binDir, "abcd")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, target); err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	det, err := Detect(managedRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	if !hasGap(det.Gaps, "symlink.foreign") {
		t.Errorf("a live symlink abcd does not own must stay foreign: %+v", det.Gaps)
	}
	if hasGap(det.Gaps, "symlink.dangling") {
		t.Errorf("a link that resolves is not dangling: %+v", det.Gaps)
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	dest, rerr := os.Readlink(target)
	if rerr != nil || dest != elsewhere {
		t.Fatalf("install clobbered a live symlink abcd does not own: %q (%v), want %q", dest, rerr, elsewhere)
	}
}
