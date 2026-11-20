package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNamingDeliveryStatePointsAtTheStatusBlock (iss-2610262031267581): the
// register's note on the reserved-vocabulary table sent readers to "the
// roadmap dashboard" for current delivery state, while the glossary's roadmap
// entry is superseded: the Now / Next / Later status block on the bare `abcd`
// board and the site's Status page replaces the roadmap document and the
// word. The note, and the brief README's delivery-state paragraph, must point
// at the status block and never at the retired name.
func TestNamingDeliveryStatePointsAtTheStatusBlock(t *testing.T) {
	brief := filepath.Join(repoRootFromPackage, ".abcd", "development", "brief")
	for _, rel := range []string{
		filepath.Join("02-constraints", "04-naming.md"),
		"README.md",
	} {
		body, err := os.ReadFile(filepath.Join(brief, rel))
		if err != nil {
			t.Fatal(err)
		}
		var note string
		for _, line := range strings.Split(string(body), "\n") {
			if strings.Contains(line, "delivery state") && strings.Contains(strings.ToLower(line), "current") {
				note = line
				break
			}
		}
		if rel == "README.md" {
			// The README wraps its paragraph; judge the whole file.
			note = strings.Join(strings.Fields(string(body)), " ")
		}
		if note == "" {
			t.Fatalf("%s carries no current-delivery-state note", rel)
		}
		if strings.Contains(note, "roadmap dashboard") {
			t.Errorf("%s still sends readers to the retired roadmap dashboard for delivery state", rel)
		}
		for _, want := range []string{"Now / Next / Later", "bare `abcd` board", "Status page"} {
			if !strings.Contains(note, want) {
				t.Errorf("%s's delivery-state note does not name %q", rel, want)
			}
		}
	}
}
