package site

import (
	"html"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/statusblock"
)

// TestStatusPageRendersTheBlockFromTheSameRead is criterion 2: the built
// Status page carries Now, Next and Later, and every row statusblock.Read
// returns for the same repository and the same lane reader is on it, with its
// id and title; the lane row carries its lane state, and the intent in a lane
// is on the block once, under Now (ruling BV2 of 2026-09-29).
func TestStatusPageRendersTheBlockFromTheSameRead(t *testing.T) {
	f := newFixture(t)
	// A second draft, so Later keeps a row while the fixture's first draft is
	// in a lane.
	f.write(".abcd/development/intents/drafts/itd-9-a-second-draft.md", "---\nid: itd-9\nslug: a-second-draft\nspec_id: null\nkind: standalone\n---\n\n# A Second Draft\n\n## Acceptance Criteria\n\n- Given nothing, when nothing, then nothing.\n")
	f.commitAt("2026-03-07T09:00:00+00:00", "feat: a second draft", "None")
	out := t.TempDir()
	lanes := func(string) ([]statusblock.Started, error) {
		return []statusblock.Started{{Intent: "itd-1", Lane: statusblock.Lane{Run: "run-2609290000000001", Lane: "lane-1", Stage: "implement"}}}, nil
	}
	if _, err := Build(Request{RepoRoot: f.Root(), OutDir: out, Stamp: fixtureStamp, Lanes: lanes}); err != nil {
		t.Fatalf("build: %v", err)
	}
	page := outFile(t, out, "record/health/index.html")

	want, err := statusblock.Read(f.Root(), lanes, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(want.Now) == 0 || len(want.Later) == 0 {
		t.Fatalf("precondition: the fixture's block has a Now row and a Later row: %+v", want)
	}
	for _, heading := range []string{">Now<span>", ">Next<span>", ">Later<span>"} {
		if !strings.Contains(page, heading) {
			t.Errorf("the Status page has no %s panel", heading)
		}
	}
	for _, rows := range [][]statusblock.Row{want.Now, want.Next, want.Later} {
		for _, r := range rows {
			if !strings.Contains(page, ">"+r.ID+"<") || !strings.Contains(page, ">"+html.EscapeString(r.Title)+"<") {
				t.Errorf("the Status page lacks the row %s %q", r.ID, r.Title)
			}
		}
	}
	if !strings.Contains(page, "lane-1 · implement") {
		t.Error("the lane row does not carry its lane state")
	}
	block := page[strings.Index(page, ">Now<span>"):strings.Index(page, "References a record the tree does not hold")]
	if n := strings.Count(block, ">itd-1<"); n != 1 {
		t.Errorf("the intent in a lane is on the block %d times, want once, under Now", n)
	}
	if !strings.Contains(page, `<span class="s">draft</span>`) {
		t.Error("the draft row is not marked a draft")
	}
	// The block opens the page, before the checks.
	if strings.Index(page, ">Now<span>") > strings.Index(page, "References a record the tree does not hold") {
		t.Error("the block does not open the Status page")
	}
}

// TestStatusSectionMarksTheHeadAndTheFailingChecks renders the rows the
// fixture does not carry: the head marked next up, and a refused planned
// intent with the gating checks it fails.
func TestStatusSectionMarksTheHeadAndTheFailingChecks(t *testing.T) {
	f := newFixture(t)
	ui, err := LoadUI(f.Root(), "site-src/ui.json")
	if err != nil {
		t.Fatal(err)
	}
	e := &explorer{c: &composer{ui: ui}, status: &statusblock.Block{
		Now:   []statusblock.Row{{ID: "itd-5", Title: "Five & more", Bucket: "planned", NextUp: true}},
		Next:  []statusblock.Row{{ID: "itd-5", Title: "Five & more", Bucket: "planned"}},
		Later: []statusblock.Row{{ID: "itd-6", Title: "Six", Bucket: "planned", Failing: []string{"spec_link", "spec_body"}}},
	}}
	got := e.statusSection()
	for _, want := range []string{
		`<span class="id">itd-5</span><span>Five &amp; more</span><span class="s"><b>next up</b></span>`,
		`<span class="id">itd-6</span><span>Six</span><span class="s">fails spec_link, spec_body</span>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the block lacks\n%s\nin\n%s", want, got)
		}
	}
	if (&explorer{c: &composer{ui: ui}}).statusSection() != "" {
		t.Error("a page with no block renders one")
	}
}

// TestStatusSurfacesNeverSayRoadmap is criterion 5 at the site: the Status
// page does not carry the word.
func TestStatusSurfacesNeverSayRoadmap(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	buildFixture(t, f, out)
	if page := outFile(t, out, "record/health/index.html"); strings.Contains(strings.ToLower(page), "roadmap") {
		t.Error("the Status page says roadmap")
	}
}

// TestStatusPageCarriesNoOrderNote: the head and Next are read in the pick
// order, the one order the block has, so the built Status page leads the block
// with no note about its order — not the record-id note the interface strings
// still declare for a managed repository's ui.json, nor any other.
func TestStatusPageCarriesNoOrderNote(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	buildFixture(t, f, out)
	page := outFile(t, out, "record/health/index.html")
	ui, err := LoadUI(f.Root(), "site-src/ui.json")
	if err != nil {
		t.Fatal(err)
	}
	note := html.EscapeString(ui.Status.OrderRecordID)
	if note == "" {
		t.Fatal("precondition: the fixture declares the retired order note")
	}
	if strings.Contains(page, note) {
		t.Errorf("the Status page names an interim order %q", ui.Status.OrderRecordID)
	}
	block := (&explorer{c: &composer{ui: ui}, status: &statusblock.Block{Order: statusblock.OrderPick}}).statusSection()
	if !strings.HasPrefix(block, `<div class="dash reading status-block">`) {
		t.Errorf("the block does not open on its panels:\n%s", block)
	}
}
