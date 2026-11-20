package site

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestTheHealthPageIsNotCalledTheStatusPage holds the living record to the
// ruling on iss-2610262036208370 (dcn-2610271112330536): the word status
// names `abcd status`, the shared-remote view, so the site page at
// `/record/health/` is the record health page and is never called the status
// page. The Now / Next / Later block it opens with keeps its internal name
// (the status block, the `status` labels and page switch, statusblock), and
// the site chapter records that it does.
//
// Shipped intents, ADRs, release notes and the changelog are history and say
// what was true when they were written, so only the living prose is read: the
// site chapter the issue was found in, and the plugin page and the
// acknowledgements, which describe the same page.
func TestTheHealthPageIsNotCalledTheStatusPage(t *testing.T) {
	living := []string{
		".abcd/development/brief/04-surfaces/22-site.md",
		"commands/abcd.md",
		"ACKNOWLEDGEMENTS.md",
	}
	statusPage := regexp.MustCompile(`(?i)\bstatus page\b`)
	read := func(rel string) string {
		raw, err := os.ReadFile(filepath.Join(repoRoot(), filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		return strings.Join(strings.Fields(string(raw)), " ")
	}
	for _, rel := range living {
		if loc := statusPage.FindStringIndex(read(rel)); loc != nil {
			t.Errorf("%s calls /record/health/ the status page; it is the record health page, "+
				"and status names `abcd status`", rel)
		}
	}

	site := read(".abcd/development/brief/04-surfaces/22-site.md")
	if !strings.Contains(site, "keeps its internal name") {
		t.Errorf("the site chapter does not record that the Now / Next / Later block keeps its internal name")
	}
}
