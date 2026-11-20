package ideate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The ideate surface once said that, ideate being optional, "capture friction
// stays at one line", while the capture surface makes a capture one line of
// text and one line of remedy, the remedy required (iss-2610262033262866). The
// ideate chapter's claim is that ideate adds nothing to capture, so it says
// that, and the capture chapter keeps the two-line shape it contradicted.
func TestBriefIdeateClaimsNoCaptureFrictionNotOneLine(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	read := func(rel string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		return strings.Join(strings.Fields(string(b)), " ")
	}
	const ideateRel = ".abcd/development/brief/04-surfaces/18-ideate.md"
	const captureRel = ".abcd/development/brief/04-surfaces/06-capture.md"
	ideate := read(ideateRel)
	if strings.Contains(ideate, "capture friction stays at one line") {
		t.Errorf("%s still says capture friction stays at one line, though %s makes a capture one line of text and one line of remedy", ideateRel, captureRel)
	}
	if !strings.Contains(ideate, "ideate adds no friction to capture") {
		t.Errorf("%s no longer says ideate adds no friction to capture", ideateRel)
	}
	if !strings.Contains(read(captureRel), "is one line of text and one line of remedy") {
		t.Errorf("%s no longer says a capture is one line of text and one line of remedy; recheck the ideate chapter's claim against it", captureRel)
	}
}
