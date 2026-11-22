package embargo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The embargo chapter says its "What stays the person's" section lists the
// acts the command page hands the person. The page's own list carries six
// items (publishing an advisory the bare listing names ready joined them with
// iss-2611220942278236); the chapter once left out putting back the record details redacted to
// land a fix (iss-2611060809519550), which is a separate item from restoring
// each record into the public ledger. Each act on the page is held here to a
// phrase the chapter's section must carry, and the page's list to exactly
// those acts, so an act added to one side alone fails.
func TestBriefPersonsActsMatchCommandPage(t *testing.T) {
	const (
		chapterRel = ".abcd/development/brief/04-surfaces/37-embargo.md"
		pageRel    = "commands/embargo.md"
	)
	// Each act: a phrase the page's bullet carries, and one the chapter's
	// section carries for the same act.
	acts := []struct{ page, chapter string }{
		{"fixing each vulnerability", "fixing and landing the fixes"},
		{"restoring each record into the public ledger", "restoring each record into the public ledger"},
		{"requesting a CVE", "requesting a CVE"},
		{"putting back any record details they redacted", "putting back any record details redacted"},
		{"every verdict", "Every verdict"},
		{"publishing an advisory the bare listing names ready", "publishing an advisory the bare listing names ready"},
	}

	section := func(rel, heading string) string {
		t.Helper()
		body, err := os.ReadFile(filepath.Join("..", "..", "..", filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := string(body)
		start := strings.Index(text, "\n"+heading+"\n")
		if start < 0 {
			t.Fatalf("%s has no %q section", rel, heading)
		}
		rest := text[start+len(heading)+2:]
		if end := strings.Index(rest, "\n## "); end >= 0 {
			rest = rest[:end]
		}
		if end := strings.Index(rest, "\n<!--"); end >= 0 {
			rest = rest[:end]
		}
		return rest
	}

	page := section(pageRel, "## What stays the person's act")
	// Prose wraps anywhere, so the chapter is read with its whitespace folded.
	chapter := strings.Join(strings.Fields(section(chapterRel, "## What stays the person's")), " ")

	var bullets []string
	for _, line := range strings.Split(page, "\n") {
		if strings.HasPrefix(line, "- ") {
			bullets = append(bullets, line)
		} else if len(bullets) > 0 && strings.HasPrefix(line, "  ") {
			bullets[len(bullets)-1] += " " + strings.TrimSpace(line)
		}
	}
	if len(bullets) != len(acts) {
		t.Errorf("%s lists %d acts the person keeps; this test knows %d: hold the chapter to the new one", pageRel, len(bullets), len(acts))
	}
	for _, a := range acts {
		found := false
		for _, b := range bullets {
			if strings.Contains(b, a.page) {
				found = true
			}
		}
		if !found {
			t.Errorf("%s's list no longer names %q", pageRel, a.page)
		}
		if !strings.Contains(chapter, a.chapter) {
			t.Errorf("%s's section does not name the act the page lists as %q (want %q)", chapterRel, a.page, a.chapter)
		}
	}
}
