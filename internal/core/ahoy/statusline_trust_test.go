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
			restore := fsutil.SwapOwnerUIDForTest(func(string) (uint32, error) { return uint32(os.Getuid()) + 1, nil })
			t.Cleanup(restore)
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
