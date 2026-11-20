package repolint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTheScopePageShipsAnIntentOnlyWithItsLastOpenSpec holds the scope page
// to the 1:n relation between an intent and its specs (iss-2610262031147190).
// `spec close` ships the intent only when no open spec still names it, and a
// `--remainder` close leaves it planned, so the page must not say that closing
// "its linked spec" is what ships the intent.
func TestTheScopePageShipsAnIntentOnlyWithItsLastOpenSpec(t *testing.T) {
	const page = ".abcd/development/brief/01-product/04-scope.md"
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
	raw, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(page)))
	if err != nil {
		t.Fatalf("read %s: %v", page, err)
	}
	text := string(raw)
	for _, stale := range []string{
		"when its linked spec is closed",
		"closing the spec is what ships the intent",
	} {
		if strings.Contains(text, stale) {
			t.Errorf("%s says %q; an intent ships only when the last open spec naming it closes", page, stale)
		}
	}
	for _, want := range []string{
		"last open spec",
		"`spec close`",
		"`--remainder`",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("%s does not say %q: the shipping sentence must name the last open spec, the verb and the --remainder close", page, want)
		}
	}
}
