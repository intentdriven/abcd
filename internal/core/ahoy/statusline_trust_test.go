package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// trustFixture is a hermetic home with one abcd binary at <tmp>/bin/abcd that
// ~/.abcd.noindex/path-entry records, owned by the test's uid and writable by
// nobody else — the shape a documented install leaves, and the one the status
// line may run. It returns the binary's path.
func trustFixture(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	bin := filepath.Join(t.TempDir(), "bin", binName)
	writeTrustBinary(t, bin)
	recordPathEntry(t, bin)
	return bin
}

func writeTrustBinary(t *testing.T, bin string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// WriteFile's mode passes through the umask; pin it so the fixture says
	// exactly what each case changes.
	if err := os.Chmod(bin, 0o755); err != nil {
		t.Fatal(err)
	}
}

func recordPathEntry(t *testing.T, bin string) {
	t.Helper()
	writeUserPathEntry(t, "path="+bin+"\nbinary_sha256="+strings.Repeat("a", 64)+"\n")
	if err := os.Chmod(userPathEntryPath(), 0o600); err != nil {
		t.Fatal(err)
	}
}

// relink replaces the fixture's binary at link with a symlink to target.
func relink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
}

// TestStatusLineEntryTrust pins the checks the status line's wired binary must
// pass — the ones the plugin's hook shims apply before they run a PATH abcd
// (hooks/hooks.json), plus the plugin cache's versioned directory, which the
// next plugin update deletes. Each refusal names its reason.
func TestStatusLineEntryTrust(t *testing.T) {
	t.Run("the recorded install passes", func(t *testing.T) {
		bin := trustFixture(t)
		if ok, reason := statusLineEntryTrust(bin); !ok {
			t.Errorf("the recorded, owned install was refused: %s", reason)
		}
	})

	cases := []struct {
		name string
		// arrange returns the path to judge, after changing the fixture.
		arrange func(t *testing.T, bin string) string
		reason  string
	}{
		{"a relative path", func(*testing.T, string) string { return "abcd" }, "not an absolute path"},
		{"a binary that is gone", func(t *testing.T, bin string) string {
			if err := os.Remove(bin); err != nil {
				t.Fatal(err)
			}
			return bin
		}, "does not exist"},
		{"a binary path-entry does not record", func(t *testing.T, bin string) string {
			other := filepath.Join(filepath.Dir(bin), "abcd-darwin-arm64")
			writeTrustBinary(t, other)
			return other
		}, "path-entry does not record it"},
		{"no path-entry record at all", func(t *testing.T, bin string) string {
			if err := os.Remove(userPathEntryPath()); err != nil {
				t.Fatal(err)
			}
			return bin
		}, "path-entry records no abcd install"},
		{"a world-writable directory", func(t *testing.T, bin string) string {
			if err := os.Chmod(filepath.Dir(bin), 0o777); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(filepath.Dir(bin), 0o755) })
			return bin
		}, "its directory is world-writable"},
		{"a world-writable binary", func(t *testing.T, bin string) string {
			if err := os.Chmod(bin, 0o757); err != nil {
				t.Fatal(err)
			}
			return bin
		}, "the binary itself is world-writable"},
		{"a group-writable binary", func(t *testing.T, bin string) string {
			if err := os.Chmod(bin, 0o775); err != nil {
				t.Fatal(err)
			}
			return bin
		}, "writable by your group"},
		{"a binary another account owns", func(t *testing.T, bin string) string {
			ownPathAs(t, bin, uint32(os.Getuid())+1)
			return bin
		}, "not owned by you"},
		{"a binary inside the working tree", func(t *testing.T, _ string) string {
			repo := t.TempDir()
			if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
				t.Fatal(err)
			}
			inTree := filepath.Join(repo, "bin", binName)
			writeTrustBinary(t, inTree)
			recordPathEntry(t, inTree)
			sub := filepath.Join(repo, "internal")
			if err := os.Mkdir(sub, 0o755); err != nil {
				t.Fatal(err)
			}
			t.Chdir(sub)
			return inTree
		}, "inside the working tree"},
		{"the plugin cache's versioned directory", func(t *testing.T, _ string) string {
			cached := filepath.Join(os.Getenv("HOME"), ".claude", "plugins", "cache", "intentdriven", "abcd", "0.12.0", binName)
			writeTrustBinary(t, cached)
			recordPathEntry(t, cached)
			return cached
		}, "versioned directory"},
		{"a link into the plugin cache's versioned directory", func(t *testing.T, bin string) string {
			cached := filepath.Join(os.Getenv("HOME"), ".claude", "plugins", "cache", "intentdriven", "abcd", "0123456789ab", binName)
			writeTrustBinary(t, cached)
			if err := os.Remove(bin); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(cached, bin); err != nil {
				t.Fatal(err)
			}
			return bin
		}, "versioned directory"},
		// The recorded entry may be a link (binTargetOwnedSymlink), and
		// whoever can write the directory its TARGET sits in chooses what the
		// line runs as surely as whoever can write the link's own directory.
		{"a link into a world-writable directory", func(t *testing.T, bin string) string {
			open := t.TempDir()
			if err := os.Chmod(open, 0o777); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(open, 0o700) })
			target := filepath.Join(open, binName)
			writeTrustBinary(t, target)
			relink(t, target, bin)
			return bin
		}, "world-writable"},
		{"a link into a working tree", func(t *testing.T, bin string) string {
			repo := t.TempDir()
			if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(repo, "bin", binName)
			writeTrustBinary(t, target)
			relink(t, target, bin)
			t.Chdir(repo)
			return bin
		}, "inside the working tree"},
		// A directory is held to the binary's own standard: writable by its
		// owner alone. A group-writable /usr/local/bin (root:admin 0775) lets
		// every admin replace what the line runs.
		{"a group-writable directory", func(t *testing.T, bin string) string {
			if err := os.Chmod(filepath.Dir(bin), 0o775); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(filepath.Dir(bin), 0o755) })
			return bin
		}, "writable by its group"},
		{"a link into a group-writable directory", func(t *testing.T, bin string) string {
			shared := t.TempDir()
			if err := os.Chmod(shared, 0o775); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(shared, binName)
			writeTrustBinary(t, target)
			relink(t, target, bin)
			return bin
		}, "writable by its group"},
		{"a directory another account owns", func(t *testing.T, bin string) string {
			ownPathAs(t, filepath.Dir(bin), uint32(os.Getuid())+1)
			return bin
		}, "owned by another account"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bin := trustFixture(t)
			p := tc.arrange(t, bin)
			ok, reason := statusLineEntryTrust(p)
			if ok {
				t.Fatalf("statusLineEntryTrust(%s) passed; want a refusal naming %q", p, tc.reason)
			}
			if !strings.Contains(reason, tc.reason) {
				t.Errorf("reason = %q, want it to name %q", reason, tc.reason)
			}
			if home := os.Getenv("HOME"); strings.Contains(reason, home) {
				t.Errorf("reason carries the home path: %q", reason)
			}
		})
	}
}

// ownPathAs makes the owner lookup report uid for p alone, so a test can stand
// a file or directory in for one root or another account owns; every other
// path keeps its real owner.
func ownPathAs(t *testing.T, p string, uid uint32) {
	t.Helper()
	want := resolvePath(p)
	restore := fsutil.SwapOwnerUIDForTest(func(q string) (uint32, error) {
		if resolvePath(q) == want {
			return uid, nil
		}
		return fsutil.OwnerUID(q)
	})
	t.Cleanup(restore)
}

// TestStatusLineEntryTrustDirectoryStandard: every directory the entry is
// reached through is held to the binary's standard — no group or other write
// bit — and may be owned by the caller or by root, so ~/.local/bin (yours,
// 0755) and a system directory (root, 0755) are admitted while a
// group-writable /usr/local/bin (root:admin 0775) is not. Each directory
// refusal names the remedy.
func TestStatusLineEntryTrustDirectoryStandard(t *testing.T) {
	t.Run("a 0755 directory you own is admitted", func(t *testing.T) {
		bin := trustFixture(t)
		if err := os.Chmod(filepath.Dir(bin), 0o755); err != nil {
			t.Fatal(err)
		}
		if ok, reason := statusLineEntryTrust(bin); !ok {
			t.Errorf("a 0755 directory was refused: %s", reason)
		}
	})
	t.Run("a 0755 directory root owns is admitted", func(t *testing.T) {
		bin := trustFixture(t)
		if err := os.Chmod(filepath.Dir(bin), 0o755); err != nil {
			t.Fatal(err)
		}
		ownPathAs(t, filepath.Dir(bin), 0)
		if ok, reason := statusLineEntryTrust(bin); !ok {
			t.Errorf("a root-owned 0755 directory was refused: %s", reason)
		}
	})
	for _, tc := range []struct {
		name string
		mode os.FileMode
		uid  func() uint32
	}{
		{"a 0775 directory root owns", 0o775, func() uint32 { return 0 }},
		{"a 0775 directory you own", 0o775, func() uint32 { return uint32(os.Getuid()) }},
		{"a 0777 directory", 0o777, func() uint32 { return uint32(os.Getuid()) }},
		{"a 0755 directory another account owns", 0o755, func() uint32 { return uint32(os.Getuid()) + 1 }},
	} {
		t.Run(tc.name+" is refused with the remedy", func(t *testing.T) {
			bin := trustFixture(t)
			dir := filepath.Dir(bin)
			if err := os.Chmod(dir, tc.mode); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
			ownPathAs(t, dir, tc.uid())
			ok, reason := statusLineEntryTrust(bin)
			if ok {
				t.Fatalf("statusLineEntryTrust passed a %04o directory owned by uid %d", uint32(tc.mode), tc.uid())
			}
			for _, want := range []string{"its directory", "~/.local/bin", "`abcd ahoy install`"} {
				if !strings.Contains(reason, want) {
					t.Errorf("reason = %q, want it to name %q", reason, want)
				}
			}
		})
	}
}

// TestStatusLineEntryTrustSparesASessionStartedInHome: the working-tree check
// bounds at the git checkout the session stands in. A session started in the
// home folder, which is no checkout, has no working tree for ~/.local/bin to
// sit inside, so the documented install is not refused for where the session
// happened to open.
func TestStatusLineEntryTrustSparesASessionStartedInHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	bin := filepath.Join(home, ".local", "bin", binName)
	writeTrustBinary(t, bin)
	recordPathEntry(t, bin)
	t.Chdir(home)
	if ok, reason := statusLineEntryTrust(bin); !ok {
		t.Errorf("the documented install was refused from a session in the home folder: %s", reason)
	}
	// A home folder that is itself a dotfiles checkout is still the person's
	// own home, not a project that could plant a binary in ~/.local/bin.
	if err := os.Mkdir(filepath.Join(home, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if ok, reason := statusLineEntryTrust(bin); !ok {
		t.Errorf("the documented install was refused because the home folder is a checkout: %s", reason)
	}
}
