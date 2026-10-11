package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ahoyPage is the command page that carries the preparation workflow:
// prepare-this-repo folded into its install section (itd-2610090831227812,
// spc-2610100613109045 step 4), around the binary call that section makes.
var ahoyPage = filepath.Join("..", "..", "..", "commands", "ahoy.md")

// The headings that bound the preparation workflow on the ahoy page: it opens
// the install section and ends where the page turns to what the binary install
// itself does, whose user-scope paths (`~/.local/bin`, the home folder) are
// the binary's own and no asset the workflow applies.
const (
	installHeading       = "## `install`"
	binaryInstallHeading = "### What the binary install does"
)

// readAhoyPage returns the ahoy page's text.
func readAhoyPage(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(ahoyPage)
	if err != nil {
		t.Fatalf("cannot read %s: %v", ahoyPage, err)
	}
	return string(data)
}

// preparationWorkflow returns the part of the ahoy page the host follows to
// prepare a repository: from the install heading to the heading on what the
// binary install does.
func preparationWorkflow(t *testing.T) string {
	t.Helper()
	page := readAhoyPage(t)
	start := strings.Index(page, installHeading)
	if start < 0 {
		t.Fatalf("%s has no %q section", ahoyPage, installHeading)
	}
	rest := page[start:]
	end := strings.Index(rest, binaryInstallHeading)
	if end < 0 {
		t.Fatalf("%s's install section has no %q heading closing the preparation workflow", ahoyPage, binaryInstallHeading)
	}
	return rest[:end]
}

// TestAhoyInstallCarriesThePreparation is A11's page half: `ahoy install` does
// what prepare-this-repo did, and that page is gone. Each phase the retired page
// ran is held to the install section by the words that carry it, so a later
// edit that drops one is named.
func TestAhoyInstallCarriesThePreparation(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "..", "..", "commands", "prepare-this-repo.md")); err == nil {
		t.Errorf("commands/prepare-this-repo.md still exists; it folded into the ahoy page's install section")
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	section := strings.Join(strings.Fields(preparationWorkflow(t)), " ")
	for _, want := range []struct{ phase, text string }{
		{"the ownership refusal", "git remote get-url origin"},
		{"the authority ordering", "Authority ordering"},
		{"the orientation", ".abcd/development/principles/"},
		{"the engine-backed audit", `"${CLAUDE_PLUGIN_ROOT}/abcd" lint --json`},
		{"the gap report", ".abcd/.work.local/scratch/"},
		{"the legacy migration", "never leave both `.work/` and `.abcd/work/`"},
		{"the AGENTS.md merge", "Merge — never overwrite"},
		{"the identity block", "identity init"},
		{"the commit gates", `"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`},
		{"the attribution opt-in", "ahoy install --attribution"},
		{"the working-conventions section", "### The working-conventions section"},
		{"the downstream rule", "### Never commit downstream"},
		{"the definition of done", "### Definition of done"},
	} {
		if !strings.Contains(section, want.text) {
			t.Errorf("the ahoy page's install section lost %s (%q)", want.phase, want.text)
		}
	}
}
