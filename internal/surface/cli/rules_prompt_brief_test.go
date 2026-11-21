package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The brief describes `abcd rules --prompt` as commands/ahoy.md does
// (iss-2610230851422913): the verb table's row names the form beside the bare
// and DOMAIN ones, and the rules section says what --prompt refuses, exit 2 —
// a DOMAIN beside it, and --fill without it or outside 0 to 100 — and what its
// JSON form carries, so a reader of the brief alone meets the flag the verb
// registers and the refusals the command enforces.
func TestTheBriefsRulesSectionDescribesThePromptForm(t *testing.T) {
	rel := ".abcd/development/brief/04-surfaces/README.md"
	raw, err := os.ReadFile(filepath.Join(testRepoRoot(), filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)

	var row string
	for _, line := range strings.Split(text, "\n") {
		// The verb table's row, not the audience table's earlier one.
		if strings.HasPrefix(line, "| `rules` | Renders ") {
			row = line
			break
		}
	}
	if row == "" {
		t.Fatalf("%s has no verb-table row for `rules`", rel)
	}
	if !strings.Contains(row, "`--prompt`") {
		t.Errorf("%s: the verb table's `rules` row names only the bare and DOMAIN forms, not `--prompt`", rel)
	}

	_, section, found := strings.Cut(text, "\n### The rules verb\n")
	if !found {
		t.Fatalf("%s has no rules-verb section", rel)
	}
	if i := strings.Index(section, "\n### "); i >= 0 {
		section = section[:i]
	}
	_, bullet, found := strings.Cut(section, "\n- **`abcd rules --prompt \"<text>\"`**")
	if !found {
		t.Fatalf("%s: the rules-verb section has no bullet for `abcd rules --prompt \"<text>\"`", rel)
	}
	if i := strings.Index(bullet, "\n- **"); i >= 0 {
		bullet = bullet[:i]
	}
	flat := strings.Join(strings.Fields(bullet), " ")
	for _, want := range []string{
		"a `DOMAIN` beside `--prompt` is refused, exit 2",
		"`--fill` without `--prompt`, or outside 0 to 100, is refused, exit 2",
		"The JSON form carries",
		"`bracket`",
		"`stage`",
		"`domains`",
		"`notes`",
		"`pages`",
		"`disabled`",
		"`unreadable`",
	} {
		if !strings.Contains(flat, want) {
			t.Errorf("%s: the rules-verb section's --prompt bullet does not say %q", rel, want)
		}
	}
}
