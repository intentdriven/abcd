package reading

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTheBriefCountsEveryUnrecordedRefusal holds the brief's account of the
// refusals past the identity point to what Ingest does (iss-2611060809210526).
//
// Two refusals there write no refusal record: the prose citation, returned
// unrecorded so the re-worded run is still admitted, and refuseARerun, which
// runs before refuse() so a run that already has an outcome never has it
// overwritten (TestARerunOfACommittedRunIsRefused holds the behaviour). The
// brief once counted only the first, so a reader took every other refusal past
// that point, the rerun among them, to leave a record.
func TestTheBriefCountsEveryUnrecordedRefusal(t *testing.T) {
	const rel = ".abcd/development/brief/04-surfaces/23-reading.md"
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	text := strings.Join(strings.Fields(string(raw)), " ")

	if strings.Contains(text, "One refusal after that point is deliberately unrecorded") {
		t.Errorf("%s still counts one unrecorded refusal past the identity point; the rerun "+
			"refusal writes no refusal record either", rel)
	}
	for _, want := range []string{
		"Two refusals after that point are deliberately unrecorded",
		"already has an outcome",
		"overwrite",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("%s does not say %q; the brief must name the rerun of a run that already "+
				"has an outcome as unrecorded, and why it writes no record", rel, want)
		}
	}
}
