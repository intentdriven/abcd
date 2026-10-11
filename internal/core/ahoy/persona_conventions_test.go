package ahoy

import (
	"regexp"
	"strings"
	"testing"
)

// conventionsPersonaCap matches the wording that closes a persona roster: names
// that are "always" or "only" a fixed set, or a ban on "other names".
var conventionsPersonaCap = regexp.MustCompile(`(?i)\balways\b|\bonly\b|other names|\bfixed\b`)

// TestAhoyInstallConventionsKeepThePersonaSequenceOpen pins that the
// working-conventions block the ahoy page's preparation workflow writes into a
// prepared
// repository's AGENTS.md teaches the same open persona sequence the bundled
// INTENTS rule does (iss-2610040758394205). A cap left in that block would
// contradict the injected rule from inside the repository's own conventions
// file, and forbid the fourth persona the issue was captured for.
func TestAhoyInstallConventionsKeepThePersonaSequenceOpen(t *testing.T) {
	rel := ahoyPage
	page := preparationWorkflow(t)
	start := strings.Index(page, "### The working-conventions section")
	if start < 0 {
		t.Fatalf("%s carries no working-conventions section", rel)
	}
	section := page[start:]
	if end := strings.Index(section[len("###"):], "\n### "); end >= 0 {
		section = section[:end+len("###")]
	}

	// The persona bullet, joined across its wrapped lines.
	var bullet string
	for _, item := range strings.Split(section, "\n- ") {
		if strings.Contains(strings.ToLower(item), "persona") &&
			strings.Contains(item, "Alice") {
			bullet = strings.Join(strings.Fields(item), " ")
			break
		}
	}
	if bullet == "" {
		t.Fatalf("%s's working-conventions section teaches no persona names", rel)
	}
	if m := conventionsPersonaCap.FindString(bullet); m != "" {
		t.Errorf("%s's working-conventions section caps the persona names (%q): %s", rel, m, bullet)
	}
	for _, want := range []string{"Alice", "Bob", "Carol", "Dave", "and on", "extends", "roster", "names and roles"} {
		if !strings.Contains(bullet, want) {
			t.Errorf("%s's persona convention does not read as an open sequence with a roster (missing %q): %s",
				rel, want, bullet)
		}
	}
}
