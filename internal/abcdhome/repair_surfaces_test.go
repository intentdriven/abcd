package abcdhome

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repairDocPages are the pages that print the repair for a reader to paste:
// each carries RepairCommand byte for byte, so a reader never meets a second
// spelling of it. The hooks' wrapper and hooks/bootstrap.sh print it inside
// the stop lines, and TestHookWrapperStopsBeforeProvisioning and
// TestBootstrapWritesNothingBesideTheOldHome hold their output to the line
// built here.
var repairDocPages = []string{
	"docs/how-to/install.md",
	"docs/how-to/upgrade-to-v0.13.0.md",
	".abcd/development/brief/05-internals/03-configuration.md",
}

// repairLiveSurfaces are every file that tells a person how to repair today,
// the pages above and the two scripts. Closed records, specs, the decision log
// and the changelog keep the text they were written with: they record what
// was.
var repairLiveSurfaces = append([]string{"hooks/hooks.json", "hooks/bootstrap.sh"}, repairDocPages...)

// TestEveryRepairSurfaceCarriesTheOneCommand (iss-2610050728100598): the glob
// the first repair used, worktrees/*/*, reached only a worktree exactly two
// levels down. No live surface may still print it, and every page that shows
// the repair shows RepairCommand itself.
func TestEveryRepairSurfaceCarriesTheOneCommand(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range repairLiveSurfaces {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "worktrees/*/*") {
			t.Errorf("%s still prints the depth-two repair glob worktrees/*/*", rel)
		}
	}
	for _, rel := range repairDocPages {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), RepairCommand) {
			t.Errorf("%s does not carry the repair command %q", rel, RepairCommand)
		}
	}
}
