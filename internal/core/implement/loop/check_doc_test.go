package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildChapterNamesEveryCheck holds the build chapter's checks section to
// the names Check reports: a refusal names its check, so a reader holding one
// must find that name in the section (iss-2611040554115326). A name is found
// bold or in backticks, an underscore read as the space the chapter writes.
func TestBuildChapterNamesEveryCheck(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", ".abcd", "development", "brief", "04-surfaces", "34-build.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the build chapter: %v", err)
	}
	doc := string(raw)
	start := strings.Index(doc, "\n## The checks\n")
	if start < 0 {
		t.Fatalf("%s has no \"## The checks\" section", path)
	}
	section := doc[start+1:]
	if end := strings.Index(section[len("## The checks"):], "\n## "); end >= 0 {
		section = section[:len("## The checks")+end]
	}
	for _, name := range []string{
		CheckKey, CheckReady, CheckOpenQuestions, CheckClaimSections, CheckHold,
		CheckBlocked, CheckSteps, CheckPeers, CheckEligible,
	} {
		found := false
		for _, form := range []string{name, strings.ReplaceAll(name, "_", " ")} {
			if strings.Contains(section, "**"+form+"**") || strings.Contains(section, "`"+form+"`") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("the build chapter's checks section does not name the %q check", name)
		}
	}
}
