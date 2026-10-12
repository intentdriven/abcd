package site

import (
	"html"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
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
		return []statusblock.Started{{Intent: "itd-1", Lanes: []statusblock.Lane{{Run: "run-2609290000000001", Lane: "lane-1", Stage: "implement"}}}}, nil
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

// TestStatusSectionShowsEachRowsTarget is itd-2609212103572513 criterion 4 on
// the site's Status page: a row whose intent names a release shows the target
// after what places it there, under the ui.json label, in Now, Next and
// Later alike; a row with none shows none.
func TestStatusSectionShowsEachRowsTarget(t *testing.T) {
	f := newFixture(t)
	ui, err := LoadUI(f.Root(), "site-src/ui.json")
	if err != nil {
		t.Fatal(err)
	}
	e := &explorer{c: &composer{ui: ui}, status: &statusblock.Block{
		Now:   []statusblock.Row{{ID: "itd-5", Title: "Five", Bucket: "planned", Target: "next", NextUp: true}},
		Next:  []statusblock.Row{{ID: "itd-5", Title: "Five", Bucket: "planned", Target: "next"}, {ID: "itd-4", Title: "Four", Bucket: "planned"}},
		Later: []statusblock.Row{{ID: "itd-6", Title: "Six", Bucket: "planned", Target: "v0.11.0", Failing: []string{"spec_link"}}},
	}}
	got := e.statusSection()
	for _, want := range []string{
		`<span class="id">itd-5</span><span>Five</span><span class="s"><b>next up</b> · target next</span>`,
		`<span class="id">itd-5</span><span>Five</span><span class="s">target next</span>`,
		`<span class="id">itd-4</span><span>Four</span></li>`,
		`<span class="id">itd-6</span><span>Six</span><span class="s">fails spec_link · target v0.11.0</span>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the block lacks\n%s\nin\n%s", want, got)
		}
	}
}

// TestStatusPageCarriesNoReservation (spc-2610031753359849 step 11): the
// board names the reservation another copy holds on a record, but the site's
// Status page is published and shows no checkout's local view, so a live
// reservation this checkout fetched for a record on the page never reaches it.
func TestStatusPageCarriesNoReservation(t *testing.T) {
	f := newFixture(t)
	now := time.Now().UTC()
	msg := "abcd reservation: itd-1\n\nAbcd-Reservation: itd-1\nAbcd-Holder: octo\n" +
		"Abcd-Copy: 9f2c41d07be35a6c80e1f4b2a7d39c55\nAbcd-Footprint: a.go\n" +
		"Abcd-Claimed-At: " + now.Add(-time.Minute).Format(time.RFC3339) + "\n" +
		"Abcd-Expires-At: " + now.Add(time.Hour).Format(time.RFC3339) + "\n"
	tree := f.gitOut("hash-object", "-w", "-t", "tree", "/dev/null")
	sha := f.gitOut("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit-tree", tree, "-m", msg)
	f.gitOut("remote", "add", "origin", filepath.Join(t.TempDir(), "origin.git"))
	f.gitOut("update-ref", "refs/remotes/origin/abcd-reserved/itd-1", sha)
	f.write(".abcd/.work.local/reservation/fetched-at", now.Format(time.RFC3339)+"\n")
	snap, err := claim.View(f.Root(), now)
	if err != nil {
		t.Fatal(err)
	}
	if held := snap.HeldBy("itd-1", now); !strings.HasPrefix(held, "reserved by octo") {
		t.Fatalf("precondition: the checkout's view holds itd-1 for octo, got %q", held)
	}

	out := t.TempDir()
	if _, err := Build(Request{RepoRoot: f.Root(), OutDir: out, Stamp: fixtureStamp}); err != nil {
		t.Fatalf("build: %v", err)
	}
	page := outFile(t, out, "record/health/index.html")
	if !strings.Contains(page, ">itd-1<") {
		t.Fatal("precondition: itd-1 is on the Status page")
	}
	for _, leak := range []string{"reserved by", "octo", "as of the last fetch"} {
		if strings.Contains(page, leak) {
			t.Errorf("the published Status page carries %q, a checkout's local view of the reservations", leak)
		}
	}
}
