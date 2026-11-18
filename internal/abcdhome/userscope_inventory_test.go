package abcdhome

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestUserScopeInventoryNamesTheChapterOfEachFolder (iss-2610230851529426):
// the configuration chapter's user-scope inventory of the home folder names
// away/, sessions/, dashboard/ and after-update/, the four folders the binary
// writes that it once left out, each followed by a link to the chapter that
// owns it, and each linked chapter exists and carries the heading its anchor
// names.
func TestUserScopeInventoryNamesTheChapterOfEachFolder(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	brief := filepath.Join(root, ".abcd", "development", "brief")
	b, err := os.ReadFile(filepath.Join(brief, "05-internals", "03-configuration.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	start := strings.Index(text, "**User scope, `~/.abcd.noindex/`**")
	end := strings.Index(text, "**Never the design record.**")
	if start < 0 || end < start {
		t.Fatalf("the user-scope inventory paragraph was not found (start %d, end %d)", start, end)
	}
	inventory := strings.Join(strings.Fields(text[start:end]), " ")
	for _, c := range []struct {
		folder, chapter, heading string
	}{
		{"`away/<root-sha>/`", "26-mode.md", "## The away mode"},
		{"`after-update/`", "21-update.md", "## An update says so once"},
		{"`sessions/names.json`", "08-abcd.md", "## What ships today"},
		{"`dashboard/`", "36-dashboard.md", "## Starting, stopping, reading the state"},
	} {
		anchor := strings.NewReplacer(",", "", " ", "-").Replace(strings.ToLower(strings.TrimPrefix(c.heading, "## ")))
		link := "([`../04-surfaces/" + c.chapter + "`](../04-surfaces/" + c.chapter + "#" + anchor + "))"
		if !strings.Contains(inventory, c.folder+" "+link) {
			t.Errorf("the user-scope inventory does not follow %s with its chapter, %s", c.folder, link)
		}
		page, err := os.ReadFile(filepath.Join(brief, "04-surfaces", c.chapter))
		if err != nil {
			t.Error(err)
			continue
		}
		if !strings.Contains(string(page), "\n"+c.heading+"\n") {
			t.Errorf("%s carries no %q heading for the inventory's anchor", c.chapter, c.heading)
		}
	}
}
