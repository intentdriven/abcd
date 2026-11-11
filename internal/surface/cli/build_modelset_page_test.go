package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildPageCarriesTheModelSet is step 7 of spc-2611101423523725: the build
// command page's review set section carries the one-for-one mapping from the
// old class names, a worked class map for a second provider, and the one line
// that sets a cheaper review. The page wraps its prose, so the section is read
// with its whitespace folded.
func TestBuildPageCarriesTheModelSet(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRootFromTest(t), "commands", "build.md"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(raw)
	const heading = "## The review set"
	start := strings.Index(page, heading)
	if start < 0 {
		t.Fatalf("the build page has no %q section", heading)
	}
	section := page[start+len(heading):]
	if end := strings.Index(section, "\n## "); end >= 0 {
		section = section[:end]
	}
	folded := strings.Join(strings.Fields(section), " ")
	for _, want := range []string{
		"`research`, `top`, `default` and `small`",
		"`top` is now `research`, `middle` is `top`, `below` is `default` and `smallest` is `small`",
		"`models.effort`",
		"`low`, `medium`, `high`, `xhigh` and `max`",
		`{"review": {"reviewers": ["default", "small"]}}`,
		"A class map for a second provider",
		`"research": "openrouter/`,
		`"small": "openrouter/`,
		"`waiting`",
		"`active`",
	} {
		if !strings.Contains(folded, want) {
			t.Errorf("the build page's %q section lacks %q", heading, want)
		}
	}
}
