package gitleaks

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRefusesADirectoryLinkTheRepositoryHoldsInTheFolderPath: PATH holds
// ~/alias -> R/scripts, and R/scripts is a committed link to /opt/tools/bin,
// which holds a real gitleaks outside R. The lookup's written path, its final
// target and ~/alias all lie outside R, but the route passes through
// R/scripts, so the repository chose which gitleaks runs.
func TestRefusesADirectoryLinkTheRepositoryHoldsInTheFolderPath(t *testing.T) {
	repo := t.TempDir()
	opt := filepath.Join(t.TempDir(), "opt", "tools", "bin")
	writeExecutable(t, filepath.Join(opt, "gitleaks"))
	if err := os.Symlink(opt, filepath.Join(repo, "scripts")); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(filepath.Join(repo, "scripts"), alias); err != nil {
		t.Fatal(err)
	}
	assertRefused(t, newTrustProbe(filepath.Join(alias, "gitleaks")), repo, Config{Enabled: true})
}
