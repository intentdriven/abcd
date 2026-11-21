package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestADirectoryLinkTheRepositoryHoldsIsHeldToTheTree pins the shape round 4
// of iss-2611202034242491 closes: PATH holds ~/alias, ~/alias links to
// R/scripts, and R/scripts is a committed link to /opt/tools/bin, which holds a
// real gitleaks outside R. The written path, the final file and the link
// ~/alias all lie outside R, but the route passes through R/scripts, a link the
// repository holds, so the repository chose which program runs.
func TestADirectoryLinkTheRepositoryHoldsIsHeldToTheTree(t *testing.T) {
	r := newRig(t)
	opt := filepath.Join(t.TempDir(), "opt", "tools", "bin")
	if err := os.MkdirAll(opt, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(opt, "gitleaks"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(opt, filepath.Join(r.repo, "scripts")); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(r.home, "alias")
	if err := os.Symlink(filepath.Join(r.repo, "scripts"), alias); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(alias, "gitleaks")
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	if !WithinTree(p, resolved, r.repo) {
		t.Errorf("WithinTree(%q, %q, %q) = false; the route passes through R/scripts, a link the repository holds", p, resolved, r.repo)
	}
	r.env["PATH"] = alias
	_, err = r.installer().admit("gitleaks")
	if err == nil || !strings.Contains(err.Error(), "inside the repository") {
		t.Errorf("admit = %v; want the program refused as reached through the repository", err)
	}
}
