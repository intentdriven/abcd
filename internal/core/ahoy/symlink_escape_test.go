package ahoy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core/identity"
)

// TestInstallRefusesAbcdDirSymlinkEscape is the GHSA-xrf8-4432-gw2f regression:
// a hostile clone commits `.abcd` as a directory symlink (git mode 120000)
// pointing outside the working tree and a CLAUDE.md marker block so the folder
// classifies ManagedRepo with no adopt prompt. The config, rules, and identity
// writers then join cwd/.abcd/... and, on unfixed code, follow the ancestor
// symlink — landing attacker-chosen JSON OUTSIDE the clone.
//
// The fix is two complementary gates: Install refuses a non-real `.abcd` before
// any mutating step, and every repo-.abcd writer resolves through os.OpenRoot so
// a symlinked ancestor is refused rather than followed. This test asserts nothing
// escapes into the symlink target and that the install is refused.
func TestInstallRefusesAbcdDirSymlinkEscape(t *testing.T) {
	setupHermetic(t)

	repo := t.TempDir()
	// A git dir plus a marker block: classify returns ManagedRepo (strong signal),
	// so install proceeds straight to the writers with no adoption prompt.
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "CLAUDE.md"),
		[]byte("# repo\n\n<!-- BEGIN ABCD -->\nrules\n<!-- END ABCD -->\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// The attacker-chosen directory OUTSIDE the clone, and the committed `.abcd`
	// directory symlink that points the writers at it.
	exfil := t.TempDir()
	if err := os.Symlink(exfil, filepath.Join(repo, ".abcd")); err != nil {
		t.Fatal(err)
	}

	opts := InstallOptions{
		Yes: true,
		ValueOverrides: map[string]string{
			"visibility":     "private",
			"docs_target":    "agents_md",
			"oracle_backend": "host-delegated",
			"scan_deep":      "false",
		},
	}
	res, err := Install(repo, opts, RefusingPrompter{})
	if err != nil {
		t.Fatalf("Install returned an error: %v", err)
	}

	// The security assertion: nothing the writers produce may appear in the
	// symlink target. config.json and rules.json escape under plain --yes; the
	// identity pin's config/ subtree is checked too for defence in depth.
	for _, leaked := range []string{
		"config.json",
		"rules.json",
		filepath.FromSlash("config/identity.json"),
	} {
		if _, err := os.Lstat(filepath.Join(exfil, leaked)); err == nil {
			t.Errorf("SECURITY: %s escaped through the .abcd symlink into %s", leaked, exfil)
		}
	}

	// The classify/mutation gate must refuse a non-real .abcd rather than write.
	if res.Status != "refused" {
		t.Errorf("Install status = %q, want %q (a non-real .abcd must gate writes)", res.Status, "refused")
	}

	// And no identity pin may have been written through the symlink either.
	if _, ok, _ := identity.LoadPin(repo); ok {
		t.Errorf("an identity pin was written despite the .abcd symlink hazard")
	}
}
