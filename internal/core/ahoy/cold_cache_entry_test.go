package ahoy

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// iss-2609100506263330: `ahoy install` on a COLD cache — no verified release
// artefact in the persistent plugin data directory — used to write the PATH
// entry as a symlink into the versioned plugin root, report success, and leave
// an entry the next plugin update strands. It now refuses the link form and
// names a command the operator can run first (the README install one-liner,
// which fetches the release binary and verifies it against the release's own
// checksums). And a later run recognises an entry ~/.abcd.noindex/path-entry records
// as abcd's even once it dangles, rather than calling it foreign, so the
// repair every owned shape gets is offered for it too.

// installRemedyAnchor is the fragment every cold-cache refusal must carry: the
// address of the one-liner the operator runs first.
const installRemedyAnchor = "github.com/intentdriven/abcd#install"

// gitRepo returns an adoptable repository (a bare .git dir).
func gitRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	return repo
}

// plantDanglingLink writes a symlink at path to a binary that does not exist,
// in a directory that is NOT a sibling of the plugin root — so neither the
// stranded-sibling rule nor the current-root rule can claim it. Only the
// provenance record can. It returns the link's destination.
func plantDanglingLink(t *testing.T, path string) string {
	t.Helper()
	dest := filepath.Join(t.TempDir(), "gone", "abcd")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(dest, path); err != nil {
		t.Fatal(err)
	}
	return dest
}

// assertLinkUntouched fails unless path is still the symlink to dest.
func assertLinkUntouched(t *testing.T, path, dest, when string) {
	t.Helper()
	fi, err := os.Lstat(path)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s: the dangling link at %s was removed or replaced (%v, %v)", when, path, fi, err)
	}
	if got, _ := os.Readlink(path); got != dest {
		t.Fatalf("%s: the link at %s was repointed to %q, want it left at %q", when, path, got, dest)
	}
}

// TestInstallOnColdCacheWritesNoLink is choice B's install half: no verified
// artefact, so nothing is written at the PATH target — not a symlink into the
// plugin root, not anything — and the refusal names the command to run first.
// The run is not reported clean, and no provenance record is written for an
// entry that does not exist.
func TestInstallOnColdCacheWritesNoLink(t *testing.T) {
	home, _ := setupUserScope(t)
	coldCache(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	target := filepath.Join(binDir, "abcd")

	res, err := Install(gitRepo(t), installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("a cold-cache install wrote %s (%v); it must write no entry at all — a link into the plugin root dangles at the next plugin update", target, fi)
	}
	if res.Status == "clean" {
		t.Errorf("a cold-cache install that wrote no PATH entry reported clean; remaining %v", res.Remaining)
	}
	joined := notesJoined(res.Notes)
	if !strings.Contains(joined, installRemedyAnchor) {
		t.Errorf("the refusal must name the command to run first (%s); notes = %v", installRemedyAnchor, res.Notes)
	}
	if !strings.Contains(joined, "ahoy install") {
		t.Errorf("the refusal must say to re-run `abcd ahoy install` afterwards; notes = %v", res.Notes)
	}
	if _, err := os.Lstat(userPathEntryPath()); !os.IsNotExist(err) {
		t.Errorf("no entry was written, so no provenance record may be either: %v", err)
	}
	if m, _ := detectSignal(t, gitRepo(t), "install_mode").(string); m != "" {
		t.Errorf("install_mode = %q after a cold-cache install; nothing is installed", m)
	}
}

// TestColdCacheInstallLeavesNothingAPluginUpdateCanStrand is choice B's update
// half: the harness replaces the plugin root on update. Because the cold
// install wrote no link, there is no entry on PATH afterwards for the update
// to strand — the state the record describes cannot arise.
func TestColdCacheInstallLeavesNothingAPluginUpdateCanStrand(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	coldCache(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)

	if _, err := Install(gitRepo(t), installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	// The plugin update: the old root is replaced by a fresh sibling.
	if err := os.RemoveAll(pluginRoot); err != nil {
		t.Fatal(err)
	}
	for _, e := range scanPathEntries(pluginRoot) {
		if e.dangling {
			t.Fatalf("a plugin update stranded %s, which the cold-cache install wrote", e.path)
		}
		t.Errorf("a cold-cache install left %s on PATH", e.path)
	}
}

// TestInstallOnWarmCacheSurvivesPluginUpdate is the other half of the same
// promise: with a verified cache the entry is the owned copy, and a plugin
// update that replaces the root leaves it resolving and runnable.
func TestInstallOnWarmCacheSurvivesPluginUpdate(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	seedDataCache(t, cacheArtefact)
	target := filepath.Join(binDir, "abcd")

	if _, err := Install(gitRepo(t), installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(pluginRoot); err != nil {
		t.Fatal(err)
	}
	if present, err := fsutil.Exists(target); err != nil || !present {
		t.Fatalf("the entry no longer resolves after the plugin root was replaced (%v, %v)", present, err)
	}
	if err := exec.Command(target).Run(); err != nil {
		t.Errorf("the entry must keep executing after a plugin update: %v", err)
	}
}

// TestDetectLegacyPinOnColdCacheIsAGap is the folded evidence's first ask: a
// live symlink into the CURRENT plugin root works today and dies at the next
// update, and on a cold cache it used to raise nothing at all. It is a gap now
// whatever the cache holds, and on a cold cache its fix hint names the command
// to run first rather than an install that could not act on it.
func TestDetectLegacyPinOnColdCacheIsAGap(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	coldCache(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	link := filepath.Join(binDir, "abcd")
	linkOwned(t, link, pluginRoot)

	det, err := Detect(managedRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	g := gapByID(det.Gaps, "symlink.legacy")
	if g == nil {
		t.Fatalf("a pin into the plugin root on a cold cache raised no symlink.legacy gap: %+v", det.Gaps)
	}
	if !g.Required {
		t.Errorf("symlink.legacy must be required: %+v", g)
	}
	if !strings.Contains(g.FixHint, installRemedyAnchor) {
		t.Errorf("on a cold cache the fix hint must name the command to run first (%s): %q", installRemedyAnchor, g.FixHint)
	}

	// Install on the same cold cache leaves the working pin as it stands and
	// says why, rather than reporting the pin as a clean install.
	res, err := Install(gitRepo(t), installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if dest, err := os.Readlink(link); err != nil || resolveSymlinkDest(link, dest) != resolvePath(pluginBinaryPath(pluginRoot)) {
		t.Fatalf("install on a cold cache must leave the working pin in place (%q, %v)", dest, err)
	}
	if res.Status == "clean" {
		t.Errorf("a pin that dies at the next plugin update was reported clean; notes %v", res.Notes)
	}
	if !strings.Contains(notesJoined(res.Notes), installRemedyAnchor) {
		t.Errorf("install must name the command to run first; notes = %v", res.Notes)
	}
}

// TestRecordedDanglingEntryIsAbcdsAndRepairs is part (C): an entry the
// provenance record names that has since become dangling is abcd's own, not a
// foreign occupant. Bare detection names it with the owned wording and writes
// nothing; the explicit install repairs it, replacing it with the verified
// owned copy; `abcd update` routes it to that repair.
func TestRecordedDanglingEntryIsAbcdsAndRepairs(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	coldCache(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	link := filepath.Join(binDir, "abcd")
	dest := plantDanglingLink(t, link)
	vouchedPathEntry(t, link)
	recBefore, err := os.ReadFile(userPathEntryPath())
	if err != nil {
		t.Fatal(err)
	}

	if kind := classifyBinTarget(link, pluginRoot); kind != binTargetOwnedSymlink {
		t.Errorf("classify = %v; a dangling entry ~/.abcd.noindex/path-entry records is abcd's own", kind)
	}
	if got := ResolveUpdateTarget().Kind; got != UpdateTargetDangling {
		t.Errorf("update target = %q, want %q: `abcd update` must route abcd's own dangling entry to its repair", got, UpdateTargetDangling)
	}

	det, err := Detect(managedRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	if hasGap(det.Gaps, "symlink.foreign") {
		t.Errorf("a recorded dangling entry was reported foreign: %+v", det.Gaps)
	}
	var dangling []Gap
	for _, g := range det.Gaps {
		if g.ID == "symlink.dangling" {
			dangling = append(dangling, g)
		}
	}
	if len(dangling) != 1 {
		t.Fatalf("want exactly one symlink.dangling gap for the one entry, got %d: %+v", len(dangling), det.Gaps)
	}
	if !dangling[0].Resolvable || !strings.Contains(dangling[0].Detail, "abcd-owned") {
		t.Errorf("the gap must carry the owned wording and offer the repair: %+v", dangling[0])
	}
	// Bare ahoy writes nothing.
	assertLinkUntouched(t, link, dest, "bare detection")
	if recAfter, _ := os.ReadFile(userPathEntryPath()); !bytes.Equal(recAfter, recBefore) {
		t.Errorf("bare detection rewrote the provenance record: %q -> %q", recBefore, recAfter)
	}

	// The repair, on a verified cache.
	seedDataCache(t, cacheArtefact)
	if _, err := Install(gitRepo(t), installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Lstat(link)
	if err != nil || !fi.Mode().IsRegular() {
		t.Fatalf("the repair must leave a regular owned copy at %s (%v, %v)", link, fi, err)
	}
	if got, _ := os.ReadFile(link); !bytes.Equal(got, cacheArtefact) {
		t.Errorf("the repaired entry must hold the verified artefact; got %q", got)
	}
	if kind := classifyBinTarget(link, pluginRoot); kind != binTargetOwnedCopy {
		t.Errorf("after the repair classify = %v, want the owned copy (the record must name the new bytes)", kind)
	}
}

// TestRecordedDanglingEntryOnColdCacheIsLeftAndNamed: the repair needs a
// verified artefact. Without one, install touches nothing, keeps the entry
// classified as abcd's, and names the command to run first.
func TestRecordedDanglingEntryOnColdCacheIsLeftAndNamed(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	coldCache(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	link := filepath.Join(binDir, "abcd")
	dest := plantDanglingLink(t, link)
	vouchedPathEntry(t, link)

	res, err := Install(gitRepo(t), installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	assertLinkUntouched(t, link, dest, "cold-cache install")
	if kind := classifyBinTarget(link, pluginRoot); kind != binTargetOwnedSymlink {
		t.Errorf("classify = %v after a cold-cache install; the entry is still abcd's", kind)
	}
	joined := notesJoined(res.Notes)
	if !strings.Contains(joined, installRemedyAnchor) {
		t.Errorf("the cold-cache refusal must name the command to run first; notes = %v", res.Notes)
	}
	if strings.Contains(joined, "does not own") {
		t.Errorf("abcd's own recorded entry was described as one abcd does not own; notes = %v", res.Notes)
	}
}

// TestUnrecordedDanglingEntryIsNotClaimed: the same dangling link with NO
// record naming it is not abcd's. Detection never asserts provenance for it,
// classification stays foreign, and neither bare detection nor a cold-cache
// install touches it.
func TestUnrecordedDanglingEntryIsNotClaimed(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	coldCache(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	link := filepath.Join(binDir, "abcd")
	dest := plantDanglingLink(t, link)
	// A record exists, but for a different entry: it vouches for nothing here.
	vouchedPathEntry(t, filepath.Join(t.TempDir(), "abcd"))

	if kind := classifyBinTarget(link, pluginRoot); kind != binTargetForeign {
		t.Errorf("classify = %v; an unrecorded dangling link is not abcd's", kind)
	}
	det, err := Detect(managedRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range det.Gaps {
		if g.ID == "symlink.dangling" && strings.Contains(g.Detail, "abcd-owned") {
			t.Errorf("an unrecorded dangling link was claimed as abcd-owned: %+v", g)
		}
	}
	assertLinkUntouched(t, link, dest, "bare detection")

	if _, err := Install(gitRepo(t), installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	assertLinkUntouched(t, link, dest, "cold-cache install")
}

// TestDanglingEntryRecordNotTrustedUnlessOwned: the record is honoured only as
// the hook reads it — a regular file this uid owns that group and other cannot
// write. A record failing that vouches for nothing, so the dangling link it
// names stays unclaimed.
func TestDanglingEntryRecordNotTrustedUnlessOwned(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T)
	}{
		{"group-writable", func(t *testing.T) {
			if err := os.Chmod(userPathEntryPath(), 0o664); err != nil {
				t.Fatal(err)
			}
		}},
		{"other-writable", func(t *testing.T) {
			if err := os.Chmod(userPathEntryPath(), 0o646); err != nil {
				t.Fatal(err)
			}
		}},
		{"owned by another uid", func(t *testing.T) {
			restore := fsutil.SwapOwnerUIDForTest(func(string) (uint32, error) {
				return uint32(os.Getuid()) + 1, nil
			})
			t.Cleanup(restore)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home, pluginRoot := setupUserScope(t)
			binDir := filepath.Join(home, ".local", "bin")
			t.Setenv("PATH", binDir)
			link := filepath.Join(binDir, "abcd")
			plantDanglingLink(t, link)
			vouchedPathEntry(t, link)
			tc.setup(t)

			if kind := classifyBinTarget(link, pluginRoot); kind != binTargetForeign {
				t.Errorf("classify = %v; a record this session does not solely own vouches for nothing", kind)
			}
			if got := ResolveUpdateTarget().Kind; got == UpdateTargetDangling {
				t.Errorf("update target = %q on an untrusted record", got)
			}
		})
	}
}

// TestUninstallTakesARecordedDanglingEntryWithItsRecord sweeps the uninstall
// sibling: the owned dangling gap names `ahoy uninstall` as a remedy, so
// uninstall classifies with the same predicate — it removes the link the
// record names together with the record, and leaves an unrecorded one alone.
func TestUninstallTakesARecordedDanglingEntryWithItsRecord(t *testing.T) {
	t.Run("recorded", func(t *testing.T) {
		home, _ := setupUserScope(t)
		binDir := filepath.Join(home, ".local", "bin")
		t.Setenv("PATH", binDir)
		link := filepath.Join(binDir, "abcd")
		plantDanglingLink(t, link)
		vouchedPathEntry(t, link)

		receipt, err := Uninstall(managedRepo(t), "")
		if err != nil {
			t.Fatal(err)
		}
		if !receipt.Symlink.Removed {
			t.Fatalf("uninstall left abcd's own recorded dangling entry in place: %+v", receipt.Symlink)
		}
		if _, err := os.Lstat(link); !os.IsNotExist(err) {
			t.Errorf("the entry is still there: %v", err)
		}
		if _, err := os.Lstat(userPathEntryPath()); !os.IsNotExist(err) {
			t.Errorf("the record outlived the entry it names: %v", err)
		}
	})
	t.Run("unrecorded", func(t *testing.T) {
		home, _ := setupUserScope(t)
		binDir := filepath.Join(home, ".local", "bin")
		t.Setenv("PATH", binDir)
		link := filepath.Join(binDir, "abcd")
		dest := plantDanglingLink(t, link)

		receipt, err := Uninstall(managedRepo(t), "")
		if err != nil {
			t.Fatal(err)
		}
		if receipt.Symlink.Removed {
			t.Fatalf("uninstall removed a dangling link abcd cannot prove it wrote: %+v", receipt.Symlink)
		}
		assertLinkUntouched(t, link, dest, "uninstall")
	})
}

// noPluginRoot leaves nothing for the plugin-root ladder to resolve: both root
// variables empty and an executable whose ancestors hold no plugin layout —
// the machine the owned dangling gap means by "if abcd is gone". It fails the
// test unless the premise holds, so a leak through the ladder cannot turn the
// assertions below into a test of the rooted path.
func noPluginRoot(t *testing.T) {
	t.Helper()
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	exe := filepath.Join(t.TempDir(), "elsewhere", "abcd")
	saved := osExecutable
	t.Cleanup(func() { osExecutable = saved })
	osExecutable = func() (string, error) { return exe, nil }
	if root, ok := resolvePluginRoot(); ok {
		t.Fatalf("premise: no plugin root may resolve, got %q", root)
	}
}

// TestUninstallTakesARecordedDanglingEntryWithNoPluginRoot is the case the
// owned dangling gap's fix hint sends to `ahoy uninstall`: abcd is gone, so no
// plugin root resolves. The record vouches for the link without one, so
// uninstall removes the link together with its record — at the default
// location and wherever else on PATH it sits — and an unrecorded dangling link
// in the same shape stays untouched, its unrelated record with it.
func TestUninstallTakesARecordedDanglingEntryWithNoPluginRoot(t *testing.T) {
	for _, tc := range []struct {
		name      string
		elsewhere bool
	}{{"default location", false}, {"elsewhere on PATH", true}} {
		t.Run("recorded/"+tc.name, func(t *testing.T) {
			home, _ := setupUserScope(t)
			binDir := filepath.Join(home, ".local", "bin")
			link := filepath.Join(binDir, "abcd")
			t.Setenv("PATH", binDir)
			if tc.elsewhere {
				other := filepath.Join(t.TempDir(), "opt-bin")
				link = filepath.Join(other, "abcd")
				t.Setenv("PATH", other+string(os.PathListSeparator)+binDir)
			}
			plantDanglingLink(t, link)
			vouchedPathEntry(t, link)
			noPluginRoot(t)

			receipt, err := Uninstall(managedRepo(t), "")
			if err != nil {
				t.Fatal(err)
			}
			if !receipt.Symlink.Removed {
				t.Fatalf("uninstall left abcd's own recorded dangling entry in place with no plugin root: %+v", receipt.Symlink)
			}
			if _, err := os.Lstat(link); !os.IsNotExist(err) {
				t.Errorf("the entry is still there: %v", err)
			}
			if _, err := os.Lstat(userPathEntryPath()); !os.IsNotExist(err) {
				t.Errorf("the record outlived the entry it names: %v", err)
			}
		})
	}
	t.Run("unrecorded", func(t *testing.T) {
		home, _ := setupUserScope(t)
		binDir := filepath.Join(home, ".local", "bin")
		t.Setenv("PATH", binDir)
		link := filepath.Join(binDir, "abcd")
		dest := plantDanglingLink(t, link)
		vouchedPathEntry(t, filepath.Join(t.TempDir(), "abcd"))
		recBefore, err := os.ReadFile(userPathEntryPath())
		if err != nil {
			t.Fatal(err)
		}
		noPluginRoot(t)

		receipt, err := Uninstall(managedRepo(t), "")
		if err != nil {
			t.Fatal(err)
		}
		if receipt.Symlink.Removed {
			t.Fatalf("uninstall removed a dangling link abcd cannot prove it wrote: %+v", receipt.Symlink)
		}
		assertLinkUntouched(t, link, dest, "uninstall with no plugin root")
		if recAfter, _ := os.ReadFile(userPathEntryPath()); !bytes.Equal(recAfter, recBefore) {
			t.Errorf("uninstall touched a record naming another entry: %q -> %q", recBefore, recAfter)
		}
	})
}
