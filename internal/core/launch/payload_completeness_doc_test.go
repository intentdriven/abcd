package launch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// skillsChapter is the brief chapter that tells a later intent how to add a
// skill, the payload step among its steps.
const skillsChapter = ".abcd/development/brief/05-internals/08-skills.md"

// TestSkillsChapterNamesThePayloadGuard (iss-2611040555168794): the skills
// chapter said a skill added without the skills directory in the payload's
// include list passes the record gate and is missing from the release.
// TestBundleShipsEveryPluginSurface fails on exactly that tree, and it runs in
// preflight and CI, so such a skill cannot land by the normal route. The
// chapter names the test as the guard and no longer promises the silent drop.
func TestSkillsChapterNamesThePayloadGuard(t *testing.T) {
	root := repoRootForTest(t)
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(skillsChapter)))
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(strings.Fields(string(b)), " ")
	if !strings.Contains(text, "TestBundleShipsEveryPluginSurface") {
		t.Errorf("%s does not name TestBundleShipsEveryPluginSurface as the guard for the payload step", skillsChapter)
	}
	if strings.Contains(text, "missing from the release") {
		t.Errorf("%s still says a skill left out of the include list is missing from the release; TestBundleShipsEveryPluginSurface fails on that tree instead", skillsChapter)
	}
}
