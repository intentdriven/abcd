package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBriefIndexSplitsDeliveryStateByKind holds the brief index's
// delivery-state note to what the brief does (iss-2610262035572805). The note
// said delivery state lives in the roadmap dashboard, never in brief prose,
// while every surface chapter states each verb's existence as `shipped` or
// `staged` in a table surface_coverage checks. The note now splits the state:
// what exists is stated in the brief, and what is scheduled or in flight is the
// status board's Now / Next / Later block.
func TestBriefIndexSplitsDeliveryStateByKind(t *testing.T) {
	path := filepath.Join("..", "..", "..", ".abcd", "development", "brief", "README.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the brief index: %v", err)
	}
	start := strings.Index(string(raw), "**Delivery-state provenance.**")
	if start < 0 {
		t.Fatalf("%s has no delivery-state provenance note", path)
	}
	var note []string
	for _, line := range strings.Split(string(raw)[start:], "\n") {
		if !strings.HasPrefix(line, ">") && !strings.Contains(line, "**Delivery-state provenance.**") {
			break
		}
		note = append(note, strings.TrimSpace(strings.TrimPrefix(line, ">")))
	}
	text := strings.Join(note, " ")
	for _, banned := range []string{"never in brief prose", "roadmap dashboard"} {
		if strings.Contains(text, banned) {
			t.Errorf("the note still says %q, which the surface chapters' shipped / staged tables contradict", banned)
		}
	}
	for _, want := range []string{"`shipped` / `staged`", "`surface_coverage`", "verification matrix", "Now / Next / Later"} {
		if !strings.Contains(text, want) {
			t.Errorf("the note does not name %q", want)
		}
	}
}
