package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The reserve chapter's Exits section lists what exits 3 and puts every other
// refusal at exit 2, so a back-off it leaves out reads as an exit 2. A release
// reserved while another release is still being cut backs off with
// claim.ErrBackOff, which reserveRefusal maps to exit 3, the same person's
// other release included (iss-2611040555429917).
func TestTheReserveBriefsExitsListAnotherReleaseBeingCut(t *testing.T) {
	rel := ".abcd/development/brief/04-surfaces/37-reserve.md"
	raw, err := os.ReadFile(filepath.Join(testRepoRoot(), filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	_, section, found := strings.Cut(string(raw), "\n## Exits\n")
	if !found {
		t.Fatalf("%s has no Exits section", rel)
	}
	if i := strings.Index(section, "\n## "); i >= 0 {
		section = section[:i]
	}
	flat := strings.Join(strings.Fields(section), " ")
	exit3, _, found := strings.Cut(flat, "Exit 2 ")
	if !found {
		t.Fatalf("%s: the Exits section has no exit 2 sentence", rel)
	}
	if !strings.Contains(exit3, "another release is still being cut") {
		t.Errorf("%s: the Exits section's exit 3 cases do not name a release reserved while another release is still being cut", rel)
	}
}
