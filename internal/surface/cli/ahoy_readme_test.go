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

// TestAhoyInstallReadmeOfferLabelIsTrue: a run without --yes reports the
// README question under optional work it left, and the label over that list
// is true of every id in it. "asked only at a terminal" was false of the
// README question, which no run asks until its question pair is built
// (spc-2610031704338417 step 4), at a terminal or off one.
func TestAhoyInstallReadmeOfferLabelIsTrue(t *testing.T) {
	hermeticRepo(t)
	out, err := runCLIPipedStdin(t, strings.Repeat("y\n", 40), "ahoy", "install", "--adopt",
		"--visibility", "private", "--docs-target", "agents_md",
		"--oracle-backend", "host-delegated", "--scan-deep", "false")
	if err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	s := string(out)
	if !strings.Contains(s, "readme.offered") {
		t.Fatalf("the README question is not reported as left:\n%s", s)
	}
	if strings.Contains(s, "asked only at a terminal:") {
		t.Errorf("the label says the README question is asked at a terminal, which no run does yet:\n%s", s)
	}
	if !strings.Contains(s, "optional, not asked in this run: ") {
		t.Errorf("no label says the optional work was not asked:\n%s", s)
	}
}

// TestAhoyInstallReadmeHintWithoutAConfig: a run that ends with no
// .abcd/config.json (the starter files declined) does not tell the person to
// answer the README question at a terminal or with --readme-parts alone, both
// of which fail until the starter files exist; it names them first
// (iss-2610120505551170).
func TestAhoyInstallReadmeHintWithoutAConfig(t *testing.T) {
	repo := hermeticRepo(t)
	out, err := runCLIPipedStdin(t, strings.Repeat("n\n", 40), "ahoy", "install", "--adopt",
		"--visibility", "private", "--docs-target", "agents_md",
		"--oracle-backend", "host-delegated", "--scan-deep", "false")
	if err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	if _, serr := os.Stat(filepath.Join(repo, ".abcd", "config.json")); !os.IsNotExist(serr) {
		t.Fatalf("declining every question left a config (%v):\n%s", serr, out)
	}
	s := string(out)
	if !strings.Contains(s, "readme.offered") {
		t.Fatalf("the README question is not reported as left:\n%s", s)
	}
	if strings.Contains(s, "the README question is asked only of a person at a terminal, without --yes: run `abcd ahoy install` there and answer it, or answer it with --readme-parts") {
		t.Errorf("the hint names two routes that fail without a config:\n%s", s)
	}
	if !strings.Contains(s, "the README answer is kept in .abcd/config.json, which does not exist yet") {
		t.Errorf("the hint does not name the starter files first:\n%s", s)
	}
}
