package lint

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestBriefEditProvenanceNamesThePredecessorLineage pins the provenance line
// of shipped itd-69 (iss-2610262044537770). It was drawn out of a brief edit
// in the predecessor lineage, whose spc-75 is a retired predecessor-store id
// (.abcd/development/specs/README.md). The line must not credit the
// brief-change derivation gate: itd-61, which records that gate, is still a
// draft and has given abcd no such gate, so a shipped record cannot have come
// out of it. And spc-75 carries the predecessor-store qualifier the record
// puts on every predecessor id. Draft itd-70 carries the same line; a draft
// claims no delivery, and it joins this list when its line is reworded.
func TestBriefEditProvenanceNamesThePredecessorLineage(t *testing.T) {
	records := []string{
		".abcd/development/intents/shipped/itd-69-plugin-metadata-lockstep-update.md",
	}
	gate := regexp.MustCompile(`(?i)derivation\s+gate`)
	qualified := regexp.MustCompile(`spc-75\s+\(predecessor store\)`)
	for _, rel := range records {
		raw, err := os.ReadFile(filepath.Join(repoRootFromPackage, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		text := string(raw)
		if !strings.Contains(text, "Drawn out from a human brief edit") {
			t.Fatalf("%s: the provenance line is gone; this test pins its wording", rel)
		}
		if gate.MatchString(text) {
			t.Errorf("%s credits the brief-change derivation gate, which draft itd-61 has not delivered", rel)
		}
		if !strings.Contains(text, "predecessor lineage") {
			t.Errorf("%s: the provenance line must say the brief edit was in the predecessor lineage", rel)
		}
		if strings.Contains(text, "spc-75") && !qualified.MatchString(text) {
			t.Errorf("%s cites spc-75 without the predecessor-store qualifier", rel)
		}
	}
}
