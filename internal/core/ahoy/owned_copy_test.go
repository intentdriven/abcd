package ahoy

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// spc-35: the PATH entry `ahoy install` writes is an abcd-owned REGULAR FILE
// copied out of the persistent data dir's verified cache, never a symlink into
// a directory the harness deletes on every plugin update. Ownership is
// recorded provenance — the data dir's path-entry file names the installed
// path and its SHA-256 — and every promotion out of the cache re-verifies the
// artefact against the recorded hash before anything lands on PATH.

// cacheArtefact is a runnable stand-in for the release binary, distinctive so
// no other fixture bytes can pass for it.
var cacheArtefact = []byte("#!/bin/sh\n# abcd release artefact fixture\nexit 0\n")

// writeUserPathEntry writes the home-scoped provenance record (spc-35 moved it
// out of the harness data dir, which is unreachable from a terminal). The
// parent ~/.abcd is created first, matching writePathEntry.
func writeUserPathEntry(t *testing.T, body string) {
	t.Helper()
	p := userPathEntryPath()
	if p == "" {
		t.Fatal("no home-scoped path-entry location resolved")
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// seedDataCache provisions a persistent data dir holding the verified cache —
// artefact plus binary-meta — and points CLAUDE_PLUGIN_DATA at it.
func seedDataCache(t *testing.T, body []byte) string {
	t.Helper()
	data := t.TempDir()
	seedDataCacheAt(t, data, body)
	t.Setenv("CLAUDE_PLUGIN_DATA", data)
	return data
}

// seedDataCacheAt writes the self-consistent cache (artefact plus binary-meta)
// under data, wherever the caller chose to put it; it sets no environment.
func seedDataCacheAt(t *testing.T, data string, body []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(data, "cache"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cacheAssetPath(data), body, 0o755); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	meta := "release_tag=v9.9.9\nrelease_sha=unknown\nbinary_sha256=" +
		hex.EncodeToString(sum[:]) + "\nfetched_at=2026-08-01T00:00:00Z\n"
	if err := os.WriteFile(cacheMetaPath(data), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestInstallWritesOwnedCopyFromCache is AC 4's install half: with a verified
// cache present, the PATH entry lands as a regular file holding the cache
// artefact's bytes, and the data dir's path-entry records the provenance. The
// plugin root is then DELETED — the harness does exactly this to old roots —
// and the entry keeps executing, which is the whole point of the copy.
func TestInstallWritesOwnedCopyFromCache(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	seedDataCache(t, cacheArtefact)
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
		t.Fatalf("install did not create %s: %v (notes %v)", target, err, res.Notes)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("the PATH entry must be a regular file, not a symlink into a directory the harness deletes")
	}
	if fi.Mode().Perm() != 0o755 {
		t.Errorf("the owned copy must be mode 0755, got %v", fi.Mode().Perm())
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != string(cacheArtefact) {
		t.Errorf("the owned copy must hold the verified cache artefact; got %q (%v)", got, err)
	}
	raw, err := os.ReadFile(userPathEntryPath())
	if err != nil {
		t.Fatalf("install must record the provenance in the data dir's path-entry: %v", err)
	}
	sum := sha256.Sum256(cacheArtefact)
	if !strings.Contains(string(raw), "path="+target) || !strings.Contains(string(raw), hex.EncodeToString(sum[:])) {
		t.Errorf("path-entry must record the installed path and its hash; got %q", raw)
	}
	if m, _ := detectSignal(t, repo, "install_mode").(string); m != "pinned" {
		t.Errorf("install_mode = %q, want pinned", m)
	}

	// The update simulation: the plugin root vanishes, the entry survives.
	if err := os.RemoveAll(pluginRoot); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command(target).Run(); err != nil {
		t.Errorf("the owned copy must keep executing after the plugin root is deleted: %v", err)
	}
	if kind := classifyBinTarget(target, ""); kind != binTargetOwnedCopy {
		t.Errorf("classify = %v after root deletion, want owned copy — ownership is the recorded provenance, not the root", kind)
	}
}

// TestInstallHealsLegacySymlinkToOwnedCopy is AC 4's migration half: the
// symlink a pre-spc-35 install wrote still works today and dies at the next
// plugin update, so it is a heal-able gap — named by detection, replaced by
// the owned copy on `ahoy install`.
func TestInstallHealsLegacySymlinkToOwnedCopy(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	link := filepath.Join(binDir, "abcd")
	linkOwned(t, link, pluginRoot)
	seedDataCache(t, cacheArtefact)

	det, err := Detect(managedRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	g := gapByID(det.Gaps, "symlink.legacy")
	if g == nil {
		t.Fatalf("a legacy owned symlink with a verified cache available produced no symlink.legacy gap: %+v", det.Gaps)
	}
	if !g.Required || !g.Resolvable {
		t.Errorf("symlink.legacy must be required and resolvable: %+v", g)
	}

	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("the healed entry is gone: %v", err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("the legacy symlink must be replaced by the owned copy")
	}
	if got, err := os.ReadFile(link); err != nil || string(got) != string(cacheArtefact) {
		t.Errorf("the healed entry must hold the verified artefact; got %q (%v)", got, err)
	}
	if _, err := os.Stat(userPathEntryPath()); err != nil {
		t.Errorf("healing must record the provenance: %v", err)
	}
}

// TestInstallRefusesCorruptCacheArtefact is AC 5 at the PATH promotion: an
// artefact that stopped matching its recorded binary_sha256 is refused loudly
// and nothing is installed — a tampered cache must never reach PATH.
func TestInstallRefusesCorruptCacheArtefact(t *testing.T) {
	home, _ := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	seedDataCache(t, cacheArtefact)
	if err := os.WriteFile(cacheAssetPath(pluginDataDir("").dir), []byte("tampered"), 0o755); err != nil {
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
		t.Errorf("a mismatching artefact must never be installed: %v", err)
	}
	if _, err := os.Stat(userPathEntryPath()); !os.IsNotExist(err) {
		t.Errorf("no provenance may be recorded for a refused install: %v", err)
	}
	joined := notesJoined(res.Notes)
	if !strings.Contains(joined, "SHA-256") {
		t.Errorf("the refusal must name the checksum mismatch; notes = %v", res.Notes)
	}
}

// TestInstallWithoutCacheDegradesLoudlyToSymlink: no persistent data dir means
// no artefact whose provenance can be recorded, so install falls back to the
// spc-21 pinned symlink — and says so, never silently.
func TestInstallWithoutCacheDegradesLoudlyToSymlink(t *testing.T) {
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
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("without a cache the entry must be the spc-21 symlink: %v (%v)", fi, err)
	}
	dest, _ := os.Readlink(target)
	if resolveSymlinkDest(target, dest) != resolvePath(pluginBinaryPath(pluginRoot)) {
		t.Errorf("symlink dest = %q, want %q", dest, pluginBinaryPath(pluginRoot))
	}
	joined := notesJoined(res.Notes)
	if !strings.Contains(joined, "symlink") {
		t.Errorf("the degradation must be named on the result, never silent; notes = %v", res.Notes)
	}
	// The note says which sources were tried — the environment and the root's
	// stamp — so a reader following the documented terminal instruction learns
	// why it did not land the owned copy (iss-2609012111168716).
	for _, want := range []string{"CLAUDE_PLUGIN_DATA", ".data-dir"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the degradation note must name every source tried (%s); notes = %v", want, res.Notes)
		}
	}
}

// TestInstallRefusesForeignFileDespitePathEntry: ownership is the recorded
// hash, not the recorded path alone. A file at the recorded location that no
// longer matches classifies foreign and is refused exactly as any other
// foreign occupant — whoever changed it owns it now.
func TestInstallRefusesForeignFileDespitePathEntry(t *testing.T) {
	home, _ := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	seedDataCache(t, cacheArtefact)
	target := filepath.Join(binDir, "abcd")
	foreign := []byte("#!/bin/sh\n# hand-built abcd\nexit 0\n")
	writeForeign(t, target)
	if err := os.WriteFile(target, foreign, 0o755); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(cacheArtefact)
	entry := "path=" + target + "\nbinary_sha256=" + hex.EncodeToString(sum[:]) + "\n"
	writeUserPathEntry(t, entry)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	if got, rerr := os.ReadFile(target); rerr != nil || string(got) != string(foreign) {
		t.Fatalf("a file that stopped matching the recorded hash must never be replaced; got %q (%v)", got, rerr)
	}
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !hasGap(det.Gaps, "symlink.foreign") {
		t.Errorf("a file that stopped matching the recorded hash must be described as foreign: %+v", det.Gaps)
	}
	if m, _ := det.Signals["install_mode"].(string); m == "pinned" {
		t.Errorf("a mismatching file must not report as a healthy pinned install")
	}
}

// TestUninstallRemovesOwnedCopyAndPathEntry: abcd removes what abcd owns — the
// copy and its provenance record — and leaves the cache to the harness's
// uninstall-from-all-scopes deletion.
func TestUninstallRemovesOwnedCopyAndPathEntry(t *testing.T) {
	home, _ := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	seedDataCache(t, cacheArtefact)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(binDir, "abcd")
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("precondition: the owned copy must exist: %v", err)
	}

	receipt, err := Uninstall(managedRepo(t), "")
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Symlink.Removed {
		t.Fatalf("uninstall left the owned copy in place: %+v", receipt.Symlink)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Errorf("the owned copy survived uninstall: %v", err)
	}
	if _, err := os.Stat(userPathEntryPath()); !os.IsNotExist(err) {
		t.Errorf("the provenance record survived uninstall: %v", err)
	}
	if _, err := os.Stat(cacheAssetPath(pluginDataDir("").dir)); err != nil {
		t.Errorf("the cache is the harness's to delete, not uninstall's: %v", err)
	}
}

// TestUninstallLeavesForeignFileDespitePathEntry is the destructive twin of
// the install refusal: uninstall never removes a file that stopped matching
// the recorded hash, whatever path-entry says.
func TestUninstallLeavesForeignFileDespitePathEntry(t *testing.T) {
	home, _ := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	seedDataCache(t, cacheArtefact)
	target := filepath.Join(binDir, "abcd")
	writeForeign(t, target)
	sum := sha256.Sum256(cacheArtefact)
	entry := "path=" + target + "\nbinary_sha256=" + hex.EncodeToString(sum[:]) + "\n"
	writeUserPathEntry(t, entry)

	receipt, err := Uninstall(managedRepo(t), "")
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Symlink.Removed {
		t.Fatalf("uninstall removed a file that stopped matching the recorded provenance: %+v", receipt.Symlink)
	}
	if _, err := os.Stat(target); err != nil {
		t.Errorf("the foreign file must be left in place: %v", err)
	}
}

// TestDetectOwnedCopyIsAHealthyInstall: an owned copy on PATH reads as
// installed — no symlink.missing, no symlink.foreign, install_mode pinned —
// even though it is a regular file, because path-entry vouches for it.
func TestDetectOwnedCopyIsAHealthyInstall(t *testing.T) {
	home, _ := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	seedDataCache(t, cacheArtefact)
	target := filepath.Join(binDir, "abcd")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, cacheArtefact, 0o755); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(cacheArtefact)
	entry := "path=" + target + "\nbinary_sha256=" + hex.EncodeToString(sum[:]) + "\n"
	writeUserPathEntry(t, entry)

	det, err := Detect(managedRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"symlink.missing", "symlink.foreign", "symlink.legacy"} {
		if hasGap(det.Gaps, id) {
			t.Errorf("an owned copy reported %s: %+v", id, det.Gaps)
		}
	}
	if m, _ := det.Signals["install_mode"].(string); m != "pinned" {
		t.Errorf("install_mode = %q, want pinned", m)
	}
}

// TestRefreshPathEntryDigest: `abcd update` swaps the owned file after proving
// the release's provenance itself, so it re-records the hash — otherwise the
// entry it just refreshed would classify foreign forever after.
func TestRefreshPathEntryDigest(t *testing.T) {
	home, _ := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	seedDataCache(t, cacheArtefact)
	target := filepath.Join(binDir, "abcd")
	sum := sha256.Sum256(cacheArtefact)
	entry := "path=" + target + "\nbinary_sha256=" + hex.EncodeToString(sum[:]) + "\n"
	writeUserPathEntry(t, entry)
	swapped := []byte("#!/bin/sh\n# updated release\nexit 0\n")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, swapped, 0o755); err != nil {
		t.Fatal(err)
	}
	newSum := sha256.Sum256(swapped)

	RefreshPathEntryDigest(target, hex.EncodeToString(newSum[:]))
	if kind := classifyBinTarget(target, ""); kind != binTargetOwnedCopy {
		t.Errorf("after the digest refresh the swapped file must classify owned, got %v", kind)
	}

	// A refresh for a path the record does not name must change nothing.
	other := filepath.Join(t.TempDir(), "abcd")
	RefreshPathEntryDigest(other, strings.Repeat("0", 64))
	if kind := classifyBinTarget(target, ""); kind != binTargetOwnedCopy {
		t.Errorf("a refresh for an unrelated path must not disturb the record, got %v", kind)
	}
}

// TestTerminalOwnedCopySurvivesClearedHarnessEnv is the regression the harness
// hid (iss-2608210934566230): `ahoy install`, `ahoy uninstall`, and `abcd
// update` run from a plain terminal, where CLAUDE_PLUGIN_DATA and
// CLAUDE_PLUGIN_ROOT are NOT exported. With the provenance record moved home-
// scoped and carrying the plugin root, an owned copy installed under a hook's
// environment still classifies as ours and still resolves its plugin root once
// every harness variable is cleared and abcd is invoked by name through the copy
// — so Detect reports a healthy pinned install (never abcd's own binary as
// foreign), and Uninstall still removes the copy.
func TestTerminalOwnedCopySurvivesClearedHarnessEnv(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	seedDataCache(t, cacheArtefact) // sets CLAUDE_PLUGIN_DATA
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(binDir, "abcd")
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("precondition: the owned copy must be installed: %v", err)
	}

	// Now the terminal: every harness variable is gone, and abcd is invoked by
	// name through the copy (os.Executable reports the copy, whose ancestors hold
	// no plugin layout). Only the home-scoped record can carry ownership and the
	// route home now.
	t.Setenv("CLAUDE_PLUGIN_DATA", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	saved := osExecutable
	t.Cleanup(func() { osExecutable = saved })
	osExecutable = func() (string, error) { return target, nil }

	det, err := Detect(managedRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	if det.PluginRootStatus != "resolved" {
		t.Errorf("the plugin root must resolve from the record in a terminal, got %q", det.PluginRootStatus)
	}
	if hasGap(det.Gaps, "plugin.root_missing") {
		t.Errorf("the plugin root must not read as missing from a terminal: %+v", det.Gaps)
	}
	if hasGap(det.Gaps, "symlink.foreign") {
		t.Errorf("abcd must not report its own installed copy as foreign from a terminal: %+v", det.Gaps)
	}
	if m, _ := det.Signals["install_mode"].(string); m != "pinned" {
		t.Errorf("install_mode = %q from a terminal, want pinned", m)
	}
	if kind := classifyBinTarget(target, pluginRoot); kind != binTargetOwnedCopy {
		t.Errorf("classify = %v from a terminal, want owned copy", kind)
	}

	receipt, err := Uninstall(managedRepo(t), "")
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Symlink.Removed {
		t.Fatalf("uninstall must remove the owned copy from a terminal: %+v", receipt.Symlink)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Errorf("the owned copy survived a terminal uninstall: %v", err)
	}
}

// TestInstallFromTerminalReachesCacheThroughRootStamp is iss-2609012111168716.
// The bootstrap's success notice and docs/how-to/install.md tell the reader to
// run '<plugin-root>/abcd' ahoy install from a terminal, where the harness
// exports no CLAUDE_PLUGIN_DATA. The bootstrap records the data dir it
// provisioned the root from in the root's .data-dir stamp, and install reaches
// the verified cache through it — so the documented instruction lands the
// owned copy rather than the legacy symlink that dies at the next plugin
// update. Every harness variable is cleared and abcd is invoked by the
// absolute plugin-root path the notice prints.
func TestInstallFromTerminalReachesCacheThroughRootStamp(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	data := seedDataCache(t, cacheArtefact)
	if err := os.WriteFile(filepath.Join(pluginRoot, ".data-dir"), []byte("data_dir="+data+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_PLUGIN_DATA", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	saved := osExecutable
	t.Cleanup(func() { osExecutable = saved })
	osExecutable = func() (string, error) { return pluginBinaryPath(pluginRoot), nil }
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
		t.Fatalf("install did not create %s: %v (notes %v)", target, err, res.Notes)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("a terminal install must reach the cache through the root's .data-dir stamp and write the owned copy, not degrade to the legacy symlink; notes %v", res.Notes)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != string(cacheArtefact) {
		t.Errorf("the owned copy must hold the verified cache artefact; got %q (%v)", got, err)
	}
	if _, err := os.Stat(userPathEntryPath()); err != nil {
		t.Errorf("a terminal install must record the provenance: %v", err)
	}
	if joined := notesJoined(res.Notes); strings.Contains(joined, "symlink") {
		t.Errorf("no degradation may be reported when the cache was reached; notes = %v", res.Notes)
	}
}

// TestInstallDegradesLoudlyWhenRootStampIsInvalid: the stamp is a route to
// the cache, never a trust claim, so a recorded path that is not an existing
// absolute directory — or one that holds no verified artefact — is not
// followed. The install still degrades to the symlink, loudly, and the note
// names both sources it tried, so the reader learns why a documented
// instruction did not land the owned copy.
func TestInstallDegradesLoudlyWhenRootStampIsInvalid(t *testing.T) {
	cases := map[string]func(t *testing.T) string{
		"relative": func(t *testing.T) string { return "relative/data" },
		"absent":   func(t *testing.T) string { return filepath.Join(t.TempDir(), "gone") },
		"empty":    func(t *testing.T) string { return t.TempDir() },
	}
	for name, recorded := range cases {
		t.Run(name, func(t *testing.T) {
			home, pluginRoot := setupUserScope(t)
			binDir := filepath.Join(home, ".local", "bin")
			t.Setenv("PATH", binDir)
			if err := os.WriteFile(filepath.Join(pluginRoot, ".data-dir"), []byte("data_dir="+recorded(t)+"\n"), 0o644); err != nil {
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
			target := filepath.Join(binDir, "abcd")
			fi, err := os.Lstat(target)
			if err != nil || fi.Mode()&os.ModeSymlink == 0 {
				t.Fatalf("an unusable stamp must degrade to the spc-21 symlink: %v (%v)", fi, err)
			}
			joined := notesJoined(res.Notes)
			for _, want := range []string{"CLAUDE_PLUGIN_DATA", ".data-dir"} {
				if !strings.Contains(joined, want) {
					t.Errorf("the degradation note must name every source tried (%s); notes = %v", want, res.Notes)
				}
			}
		})
	}
}
