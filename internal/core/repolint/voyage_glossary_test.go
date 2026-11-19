package repolint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTheVoyageEntryCallsPrivacyHygieneALintRule holds the voyage glossary
// entry to the lint chapter's vocabulary (iss-2610262035510145). The lint
// chapter says the conformance verb applies rules about form, which adr-40
// names a lint, and reserves "audit" for itd-16's hash-chain fidelity surface.
// privacy-hygiene is a rule of this package's lint, so the entry must call it
// a lint rule and say an in-tree voyage would fail abcd's own lint.
func TestTheVoyageEntryCallsPrivacyHygieneALintRule(t *testing.T) {
	const entry = ".abcd/development/brief/glossary/core/voyage.md"
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the test working directory")
		}
		dir = parent
	}
	raw, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(entry)))
	if err != nil {
		t.Fatalf("read %s: %v", entry, err)
	}
	text := strings.Join(strings.Fields(string(raw)), " ")

	for _, stale := range []string{"`privacy-hygiene` audit rule", "fail its own audit"} {
		if strings.Contains(text, stale) {
			t.Errorf("%s says %q; privacy-hygiene is a lint rule and \"audit\" is reserved "+
				"for the hash-chain fidelity surface", entry, stale)
		}
	}
	for _, want := range []string{"`privacy-hygiene` lint rule", "fail its own lint"} {
		if !strings.Contains(text, want) {
			t.Errorf("%s does not say %q", entry, want)
		}
	}
}
