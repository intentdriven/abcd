package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAhoyInstallReadmePartsFlag: --readme-parts reaches the core from the CLI
// front door. A value naming a part abcd does not write is refused, exit 2,
// naming the parts it does, before anything is written; a good value is
// stored and its parts written under --yes; the text render of a --yes run
// without it names the flag; and uninstall's text names what it did with
// README.md's parts.
func TestAhoyInstallReadmePartsFlag(t *testing.T) {
	repo := hermeticRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "LICENSE"), []byte("MIT License\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	base := []string{"ahoy", "install", "--yes", "--adopt",
		"--visibility", "private", "--docs-target", "agents_md",
		"--oracle-backend", "host-delegated", "--scan-deep", "false"}

	out, err := runCLIErr(t, append(base, "--readme-parts", "licence,wiki")...)
	if err == nil {
		t.Fatalf("an unknown part was not refused:\n%s", out)
	}
	if !strings.Contains(err.Error(), "release-badge, build-badge, abcd-badge, licence, links, contents") {
		t.Errorf("the refusal does not name the known parts: %v", err)
	}
	if _, serr := os.Stat(filepath.Join(repo, ".abcd")); !os.IsNotExist(serr) {
		t.Errorf("a refused flag wrote .abcd/: %v", serr)
	}

	out = runCLI(t, base...)
	if !strings.Contains(string(out), "readme.offered") || !strings.Contains(string(out), "--readme-parts licence|links|contents|release-badge|build-badge|abcd-badge|none") {
		t.Errorf("a --yes run without the flag does not name it:\n%s", out)
	}

	out = runCLI(t, append(base, "--readme-parts", "licence")...)
	readme, err := os.ReadFile(filepath.Join(repo, "README.md"))
	if err != nil {
		t.Fatalf("the chosen part was not written: %v\n%s", err, out)
	}
	if !strings.Contains(string(readme), "<!-- BEGIN ABCD:licence ") {
		t.Errorf("README.md holds no licence part:\n%s", readme)
	}
	if !strings.Contains(string(out), "readme.parts: not set -> licence") {
		t.Errorf("the stored answer is not echoed:\n%s", out)
	}

	out = runCLI(t, "ahoy", "uninstall")
	if !strings.Contains(string(out), "  readme: licence removed; README.md, which abcd created, deleted\n") {
		t.Errorf("uninstall does not say what it did with README.md:\n%s", out)
	}
}
