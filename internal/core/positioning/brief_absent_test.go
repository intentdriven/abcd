package positioning

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The identity chapter once said a surface whose candidate files are all
// absent is "skipped rather than reported" (iss-2611040555271283). Check
// reports such a surface with StatusAbsent, `abcd lint identity` prints it with
// that state and its JSON lists it, and only bare `abcd lint` (the
// identity-positioning rule) leaves it out. The chapter is held to that here.
func TestBriefIdentityChapterSaysAbsentSurfaceIsReportedAbsent(t *testing.T) {
	const rel = ".abcd/development/brief/04-surfaces/19-identity.md"
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	// Fold line wrapping so a phrase split across lines still matches.
	body := strings.Join(strings.Fields(string(raw)), " ")
	if strings.Contains(body, "skipped rather than reported") {
		t.Errorf("%s still says an all-absent surface is skipped rather than reported; the identity report lists it as %q", rel, StatusAbsent)
	}
	for _, want := range []string{
		"with the state `" + string(StatusAbsent) + "`",
		"not as drift",
		"bare `abcd lint` stays silent on it",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("%s does not say %q about a surface whose candidate files are all absent", rel, want)
		}
	}
}
