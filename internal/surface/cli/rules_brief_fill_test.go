package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The rules verb section of the surfaces brief describes each form of
// `abcd rules`. Its --prompt bullet describes --fill, the flag that sets the
// context bracket --prompt shows: the range it takes, and its refusal at
// exit 2 without --prompt or outside that range, as rulesCmd enforces it
// (iss-2611060809025632).
func TestTheRulesBriefsPromptBulletDescribesFill(t *testing.T) {
	rel := ".abcd/development/brief/04-surfaces/README.md"
	raw, err := os.ReadFile(filepath.Join(testRepoRoot(), filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	_, section, found := strings.Cut(string(raw), "\n### The rules verb\n")
	if !found {
		t.Fatalf("%s has no rules verb section", rel)
	}
	if i := strings.Index(section, "\n### "); i >= 0 {
		section = section[:i]
	}
	_, bullet, found := strings.Cut(section, "\n- **`abcd rules --prompt")
	if !found {
		t.Fatalf("%s: the rules verb section has no --prompt bullet", rel)
	}
	if i := strings.Index(bullet, "\n- "); i >= 0 {
		bullet = bullet[:i]
	}
	flat := strings.Join(strings.Fields(bullet), " ")
	for _, want := range []string{
		"`--fill <percent-used>`",
		"0 to 100",
		"`--fill` without `--prompt`",
		"exit 2",
	} {
		if !strings.Contains(flat, want) {
			t.Errorf("%s: the rules verb section's --prompt bullet does not say %q", rel, want)
		}
	}
}
