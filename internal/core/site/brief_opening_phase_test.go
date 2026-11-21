package site

// The press release the site opens on states no remedy in a retired unit
// (iss-2610262032073819). Its Open Questions said transcript-noise mitigation
// would move "into an earlier phase", but itd-2609211913453478 retired phases
// in favour of dependencies and shelves, and Pass B ships only as a declared
// exemption (itd-88), so the bullet asked a question about a plan that no
// longer exists.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBriefOpeningNamesNoRetiredPhase(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(), filepath.FromSlash(BriefOpeningRelPath)))
	if err != nil {
		t.Fatal(err)
	}
	doc := strings.ToLower(string(raw))
	for _, stale := range []string{"earlier phase", "later phase", "pass b signal density"} {
		if strings.Contains(doc, stale) {
			t.Errorf("%s still says %q: phases are retired (itd-2609211913453478) and Pass B ships only as a declared exemption (itd-88)", BriefOpeningRelPath, stale)
		}
	}
}
