package ahoy

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The PATH rung of every hook shim is owned-only: it runs an `abcd` off PATH
// only when `~/.abcd/path-entry` records the path `command -v abcd` printed.
// So the record is not a detail of the owned-copy shape — it is the thing that
// makes ANY install reachable from a hook. Every install path that leaves a
// usable binary on PATH therefore writes it, and `ahoy`'s own classification
// must never report an install the hooks would refuse.

// commandVAbcd returns what a hook shim's `command -v abcd` prints with PATH
// set to exactly binDir — the string the shim compares the record's `path=`
// against, produced by the same shell rather than assumed.
func commandVAbcd(t *testing.T, binDir string) string {
	t.Helper()
	// An absolute shell: these tests narrow PATH to the install directory, so
	// looking `sh` up on PATH would skip the assertion rather than make it.
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("/bin/sh unavailable")
	}
	cmd := exec.Command("/bin/sh", "-c", "command -v abcd")
	cmd.Env = []string{"PATH=" + binDir}
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("command -v abcd found nothing in %s: %v", binDir, err)
	}
	return strings.TrimRight(string(out), "\n")
}

// recordedPathEntry returns the parsed record, failing when there is none.
func recordedPathEntry(t *testing.T) pathEntryRecord {
	t.Helper()
	rec, ok := readPathEntry()
	if !ok {
		raw, err := os.ReadFile(userPathEntryPath())
		t.Fatalf("no usable ~/.abcd/path-entry record after the install; file = %q (%v)", raw, err)
	}
	return rec
}

// assertShimWouldAccept pins the whole contract in one place: the record names
// the entry, and it names it in the exact spelling the shim's `command -v`
// yields, so the shim's string comparison succeeds.
func assertShimWouldAccept(t *testing.T, binDir string) {
	t.Helper()
	rec := recordedPathEntry(t)
	want := commandVAbcd(t, binDir)
	if rec.path != want {
		t.Errorf("path-entry records path=%q, but a hook shim's `command -v abcd` yields %q — the shim compares those two strings, so it will refuse this install", rec.path, want)
	}
	if !hexDigestOK(rec.sha) {
		t.Errorf("path-entry records binary_sha256=%q, which is not a digest readPathEntry accepts", rec.sha)
	}
}

// repoForInstall is an adoptable unmanaged repo.
func repoForInstall(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	return repo
}

// TestPinnedSymlinkInstallRecordsThePathEntry is the headline defect. With no
// verified cache artefact to copy from, install degrades — loudly, and by
// design — to the spc-21 pinned symlink. That symlink is a working `abcd` on
// PATH, and docs/how-to/install.md routes readers to this very path, yet
// nothing recorded it, so every hook refused the binary the guide had just told
// the user to install.
func TestPinnedSymlinkInstallRecordsThePathEntry(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	// No seedDataCache: this is the documented degraded path.

	res, err := Install(repoForInstall(t), installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(binDir, "abcd")
	fi, err := os.Lstat(target)
	if err != nil {
		t.Fatalf("install did not create %s: %v (notes %v)", target, err, res.Notes)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("this test must exercise the degraded pinned-symlink path; got a regular file")
	}
	if dest, _ := os.Readlink(target); resolveSymlinkDest(target, dest) != resolvePath(pluginBinaryPath(pluginRoot)) {
		t.Fatalf("the pinned symlink does not point at the plugin binary")
	}
	assertShimWouldAccept(t, binDir)
	if res.Status != "clean" {
		t.Errorf("status = %q (remaining %v), want clean — an install the hooks accept has nothing left over", res.Status, res.Remaining)
	}
}

// TestDevShimInstallRecordsThePathEntry: `--dev` is a documented install verb
// that leaves a usable `abcd` in the same user directory, so it is recorded on
// the same terms. Nothing else about the entry changes — it stays the shim.
func TestDevShimInstallRecordsThePathEntry(t *testing.T) {
	home, _ := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)

	res, err := Install(repoForInstall(t), devInstallOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(binDir, "abcd")
	if !isDevShimFile(target) {
		t.Fatalf("--dev did not install the shim at %s (notes %v)", target, res.Notes)
	}
	assertShimWouldAccept(t, binDir)
	if res.Status != "clean" {
		t.Errorf("status = %q (remaining %v), want clean", res.Status, res.Remaining)
	}
}

// TestOwnedCopyInstallRecordsThePathEntry re-states the shape that already
// worked, in the same words as the two above, so the three shapes are held to
// one contract rather than to three.
func TestOwnedCopyInstallRecordsThePathEntry(t *testing.T) {
	home, _ := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	seedDataCache(t, cacheArtefact)

	res, err := Install(repoForInstall(t), installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if classifyBinTarget(filepath.Join(binDir, "abcd"), "") == binTargetForeign {
		t.Fatalf("the owned copy did not land (notes %v)", res.Notes)
	}
	assertShimWouldAccept(t, binDir)
}

// TestUnrecordedOwnedEntryIsItsOwnGap covers the machines the release already
// produced: an entry abcd owns, with no record naming it. The board must not
// call that healthy while every hook refuses it, and `ahoy install` must heal
// it — an install with zero actionable gaps never even builds an apply context,
// so without the gap the advertised remedy could not run.
func TestUnrecordedOwnedEntryIsItsOwnGap(t *testing.T) {
	for _, tc := range []struct {
		name  string
		opts  func() InstallOptions
		plant func(t *testing.T, target, pluginRoot string)
		still func(t *testing.T, target string) bool
	}{
		{
			name:  "pinned symlink",
			opts:  installOpts,
			plant: func(t *testing.T, target, pluginRoot string) { linkOwned(t, target, pluginRoot) },
			still: func(t *testing.T, target string) bool {
				fi, err := os.Lstat(target)
				return err == nil && fi.Mode()&os.ModeSymlink != 0
			},
		},
		{
			// The dev shim is healed by the operator's own install verb. A
			// PLAIN install over a shim is a deliberate mode switch to pinned
			// (iss-107), which the gap's fix hint says.
			name: "dev shim",
			opts: devInstallOpts,
			plant: func(t *testing.T, target, pluginRoot string) {
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					t.Fatal(err)
				}
				body := renderDevShim(pluginRoot, pluginBinaryPath(pluginRoot))
				if err := os.WriteFile(target, []byte(body), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			still: func(t *testing.T, target string) bool { return isDevShimFile(target) },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home, pluginRoot := setupUserScope(t)
			binDir := filepath.Join(home, ".local", "bin")
			t.Setenv("PATH", binDir)
			target := filepath.Join(binDir, "abcd")
			tc.plant(t, target, pluginRoot)

			repo := managedRepo(t)
			det, err := Detect(repo)
			if err != nil {
				t.Fatal(err)
			}
			g := gapByID(det.Gaps, "symlink.unrecorded")
			if g == nil {
				t.Fatalf("an owned PATH entry no record names produced no symlink.unrecorded gap; every hook refuses that binary: %+v", det.Gaps)
			}
			if !g.Required || !g.Resolvable {
				t.Errorf("symlink.unrecorded must be required and resolvable: %+v", g)
			}

			// The advertised remedy heals it — and does not change the shape of
			// the install while doing so.
			if _, err := Install(repo, tc.opts(), RefusingPrompter{}); err != nil {
				t.Fatal(err)
			}
			if !tc.still(t, target) {
				t.Fatalf("healing the record changed the install shape at %s", target)
			}
			assertShimWouldAccept(t, binDir)
			det, err = Detect(repo)
			if err != nil {
				t.Fatal(err)
			}
			if hasGap(det.Gaps, "symlink.unrecorded") {
				t.Errorf("symlink.unrecorded survived the install that advertises itself as the fix")
			}
		})
	}
}

// TestUninstallDropsTheRecordItWrote closes the other end. A record that
// outlives the entry it names is worse than no record: the next thing to
// occupy that path — a binary abcd does not own — inherits the ownership claim
// and every hook runs it.
func TestUninstallDropsTheRecordItWrote(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts func() InstallOptions
	}{
		{"pinned symlink", installOpts},
		{"dev shim", devInstallOpts},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home, _ := setupUserScope(t)
			binDir := filepath.Join(home, ".local", "bin")
			t.Setenv("PATH", binDir)
			repo := repoForInstall(t)
			if _, err := Install(repo, tc.opts(), RefusingPrompter{}); err != nil {
				t.Fatal(err)
			}
			recordedPathEntry(t) // the install recorded it
			if _, err := Uninstall(repo, binDir); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Lstat(filepath.Join(binDir, "abcd")); !os.IsNotExist(err) {
				t.Fatalf("uninstall left the entry in place: %v", err)
			}
			if rec, ok := readPathEntry(); ok {
				t.Errorf("uninstall removed the entry but left path-entry vouching for %q; whatever lands there next inherits the claim", rec.path)
			}
		})
	}
}

// TestUninstallKeepsARecordItDoesNotOwn: the removal is scoped to the entry
// being removed, never a blanket delete of the home-scoped record.
func TestUninstallKeepsARecordItDoesNotOwn(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	target := filepath.Join(binDir, "abcd")
	linkOwned(t, target, pluginRoot)
	other := filepath.Join(t.TempDir(), "abcd")
	writeUserPathEntry(t, "path="+other+"\nbinary_sha256="+strings.Repeat("b", 64)+"\n")

	if _, err := Uninstall(managedRepo(t), binDir); err != nil {
		t.Fatal(err)
	}
	rec, ok := readPathEntry()
	if !ok || rec.path != other {
		t.Errorf("uninstalling %s deleted a record naming %s; got ok=%v rec=%+v", target, other, ok, rec)
	}
}

// TestOwnedCopyPredicateExcludesTheDevShim: now that the record names the dev
// shim too, a matching record no longer distinguishes the two, so the copy
// predicate has to say so itself. `abcd update` treats a true answer as
// permission to overwrite the file, and overwriting a shim the operator chose
// with a release binary is a silent mode switch.
func TestOwnedCopyPredicateExcludesTheDevShim(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	if _, err := Install(repoForInstall(t), devInstallOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(binDir, "abcd")
	if !isDevShimFile(target) {
		t.Fatalf("--dev did not install the shim at %s", target)
	}
	if IsOwnedPathCopy(target) {
		t.Errorf("IsOwnedPathCopy reported the dev shim as the owned release copy; `abcd update` would replace it")
	}
	if classifyBinTarget(target, pluginRoot) != binTargetDevShim {
		t.Errorf("the recorded dev shim stopped classifying as the dev shim")
	}
}
