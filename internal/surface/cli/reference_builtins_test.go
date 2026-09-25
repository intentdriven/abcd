package cli

import (
	"strings"
	"testing"
)

// TestReferenceListsTheBuiltinCommands pins the b-4 half of iss-304. The page
// says every user-facing command is listed, but it was generated from
// NewRootCommand before Cobra attaches its own `completion` and `help`, so two
// commands any operator can invoke were missing from it.
func TestReferenceListsTheBuiltinCommands(t *testing.T) {
	page := GenerateReference()
	for _, cmd := range []string{"abcd completion", "abcd completion zsh", "abcd help"} {
		if !strings.Contains(page, "`"+cmd+"`\n") {
			t.Errorf("the CLI reference does not list %q, which the binary answers to", cmd)
		}
	}
}
