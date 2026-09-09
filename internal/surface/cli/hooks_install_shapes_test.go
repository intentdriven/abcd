package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
)

// The two surfaces that must agree about one install. `ahoy` classifies the
// PATH entry and reports whether the machine is installed; the hook shims read
// `~/.abcd/path-entry` and decide whether to RUN that entry. A board that
// reports healthy while every hook refuses is the worst shape this pair can
// take, because neither surface says anything about the other: the user did
// what the install guide told them to and gets a session with no rules loader,
// an UNGUARDED shell guard, and no transcript capture.
//
// So the agreement is asserted end to end and per shape, against the shipped
// hooks manifest rather than a paraphrase of it: `ahoy install` really runs,
// into a sandboxed home, and the real UserPromptSubmit command is then executed
// by /bin/sh against the entry it left behind.

// installShapePluginRoot builds a plugin root `ahoy install` will accept: the
// shipped hooks manifest plus a recording stub for the binary the pinned entry
// points at.
func installShapePluginRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(hooksManifest(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "hooks", "hooks.json"), manifest, 0o644); err != nil {
		t.Fatal(err)
	}
	stub := "#!/bin/sh\ncat >/dev/null\nprintf '%s %s\\n' \"$1\" \"$2\" >> \"$ABCD_CALLS\"\nexit 0\n"
	if err := os.WriteFile(filepath.Join(root, "abcd"), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

// seedInstallShapeCache writes the verified cache `ahoy install` promotes into
// an owned copy: the platform artefact plus a binary-meta naming its digest.
func seedInstallShapeCache(t *testing.T, body []byte) string {
	t.Helper()
	data := t.TempDir()
	cache := filepath.Join(data, "cache")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	asset := filepath.Join(cache, "abcd-"+runtime.GOOS+"-"+runtime.GOARCH)
	if err := os.WriteFile(asset, body, 0o755); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	meta := "release_tag=v9.9.9\nrelease_sha=unknown\nbinary_sha256=" +
		hex.EncodeToString(sum[:]) + "\nfetched_at=2026-08-01T00:00:00Z\n"
	if err := os.WriteFile(filepath.Join(cache, "binary-meta"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
	return data
}

// runAhoyInstall installs into a sandboxed home and returns that home and the
// directory the entry landed in.
func runAhoyInstall(t *testing.T, dev bool, dataDir string) (home, binDir string) {
	t.Helper()
	home = t.TempDir()
	binDir = filepath.Join(home, ".local", "bin")
	root := installShapePluginRoot(t)
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", root)
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_DATA", dataDir)
	t.Setenv("ABCD_BIN_TARGET", "")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+"/usr/bin:/bin")

	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	adopt := true
	res, err := ahoy.Install(repo, ahoy.InstallOptions{
		Adopt: &adopt,
		Yes:   true,
		Dev:   dev,
		ValueOverrides: map[string]string{
			"visibility":     "private",
			"docs_target":    "both",
			"oracle_backend": "host-delegated",
			"scan_deep":      "false",
		},
	}, ahoy.RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	// The board's own verdict. It is asserted here so the hook assertion below
	// cannot pass merely because the board also gave up: the pair only agrees
	// if BOTH say the machine is installed.
	if res.Status != "clean" {
		t.Fatalf("ahoy reports status %q (remaining %v, notes %v); the board must call this install healthy for the disagreement under test to be the one that matters",
			res.Status, res.Remaining, res.Notes)
	}
	if _, err := os.Lstat(filepath.Join(binDir, "abcd")); err != nil {
		t.Fatalf("no PATH entry at %s: %v", filepath.Join(binDir, "abcd"), err)
	}
	return home, binDir
}

// TestHooksRunEveryShapeAhoyInstalls: for every install shape `ahoy install`
// can leave on PATH, the shipped UserPromptSubmit shim accepts that entry. The
// plugin root the shim is given is a DIFFERENT, unprovisionable one — the
// harness garbage-collects the root an install was made from — so the shim has
// no rung left but PATH, which is exactly the documented rescue.
func TestHooksRunEveryShapeAhoyInstalls(t *testing.T) {
	artefact := []byte("#!/bin/sh\ncat >/dev/null\nprintf '%s %s\\n' \"$1\" \"$2\" >> \"$ABCD_CALLS\"\nexit 0\n")
	for _, tc := range []struct {
		name string
		dev  bool
		// data is the persistent plugin data dir; empty means no verified
		// cache, which is the documented degradation to the pinned symlink.
		data func(t *testing.T) string
		// ran reports whether the entry, once accepted, actually executes abcd.
		// The dev shim rebuilds from source on every call and there is no `go`
		// on the shim's PATH, so it fails in its own loud words — which is a
		// different question from whether the hook was willing to run it.
		ran bool
	}{
		{name: "pinned symlink", data: func(*testing.T) string { return "" }, ran: true},
		{name: "owned copy", data: func(t *testing.T) string { return seedInstallShapeCache(t, artefact) }, ran: true},
		{name: "dev shim", dev: true, data: func(*testing.T) string { return "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home, binDir := runAhoyInstall(t, tc.dev, tc.data(t))

			// A second, unprovisionable plugin root: the shim's plugin-root rung
			// answers nothing, so it falls through to PATH.
			gone := hookRoot(t, failingBootstrap, false)
			_, stderr, code := hookRunHome(t, "UserPromptSubmit", gone, binDir, t.TempDir(), home)

			if strings.Contains(stderr, "ignoring the abcd found on PATH") {
				t.Fatalf("the hook refused the entry `ahoy install` just wrote, while ahoy reports the install clean; stderr: %s", stderr)
			}
			if !tc.ran {
				return
			}
			if code != 0 {
				t.Fatalf("UserPromptSubmit exit = %d against an entry ahoy installed; stderr: %s", code, stderr)
			}
			if !strings.Contains(callLog(t, filepath.Join(gone, "calls.log")), "prompt-router") {
				t.Fatalf("the hook did not run the installed abcd; stderr: %s", stderr)
			}
		})
	}
}

// TestUninstallStopsTheHooksRunningTheEntry is the same agreement from the other
// end: once `ahoy uninstall` has removed the entry, nothing it left behind may
// vouch for whatever occupies that path next. A record outliving its entry
// would hand the ownership claim to a foreign binary and every hook would run
// it — the hijack the ownership rung exists to refuse.
func TestUninstallStopsTheHooksRunningTheEntry(t *testing.T) {
	home, binDir := runAhoyInstall(t, false, "")
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ahoy.Uninstall(repo, binDir); err != nil {
		t.Fatal(err)
	}
	// A foreign binary takes the vacated path — the hijack.
	pathStub(t, binDir)

	gone := hookRoot(t, failingBootstrap, false)
	_, stderr, code := hookRunHome(t, "UserPromptSubmit", gone, binDir, t.TempDir(), home)
	if log := callLog(t, filepath.Join(gone, "calls.log")); log != "" {
		t.Fatalf("the hook ran a binary that inherited an uninstalled entry's ownership record: %q", log)
	}
	if !strings.Contains(stderr, pathRefusalUnowned) {
		t.Fatalf("the hook did not refuse the replacement on ownership grounds (exit %d); stderr: %s", code, stderr)
	}
}
