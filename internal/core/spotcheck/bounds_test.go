package spotcheck

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// plantHandBacks puts a typed hand-back for each issue on origin's store
// branch, made at at, and answers their population keys, sorted.
func (c *drawCase) plantHandBacks(at time.Time, issues ...string) []string {
	c.t.Helper()
	files := map[string]string{}
	var keys []string
	for _, issue := range issues {
		f, err := handBack(issue, at).typed().file()
		if err != nil {
			c.t.Fatal(err)
		}
		files[f.path()] = string(f.data)
		keys = append(keys, recordKey(f.name))
	}
	c.plant(files)
	slices.Sort(keys)
	return keys
}

// groupOf is cy's draw of group.
func groupOf(t *testing.T, cy Cycle, group string) GroupDraw {
	t.Helper()
	for _, g := range cy.Groups {
		if g.Group == group {
			return g
		}
	}
	t.Fatalf("cycle %s holds no %s group", cy.ID, group)
	return GroupDraw{}
}

// drawsTwoWindows opens the first window, runs between before it closes,
// draws it, runs after, and draws the second, answering both cycles.
func (c *drawCase) drawsTwoWindows(between, after func()) (Cycle, Cycle) {
	c.t.Helper()
	latest := "1000"
	c.serveRounds(&latest)
	c.openWindow(w0)
	between()
	c.closeWindow()
	if note := c.observe(q1000.Add(time.Hour)); !strings.Contains(note, "drew cycle") {
		c.t.Fatalf("the first draw said %q", note)
	}
	first := c.newestCycle()
	after()
	latest = "202600"
	c.landAt(1, q202600.Add(time.Second))
	if note := c.observe(q202600.Add(time.Hour)); !strings.Contains(note, "drew cycle") {
		c.t.Fatalf("the second draw said %q", note)
	}
	return first, c.newestCycle()
}

// TestARecordPopulationIsListedAndNeverDrawnFromTwice: a window whose
// hand-backs outnumber what a population lists takes the first of them in
// key order and leaves the rest for the next window, so each cycle lists its
// population, every record is in exactly one population, and nothing drawn
// is drawn again.
func TestARecordPopulationIsListedAndNeverDrawnFromTwice(t *testing.T) {
	saved := maxListedKeys
	maxListedKeys = 3
	t.Cleanup(func() { maxListedKeys = saved })
	c := newDrawCase(t)
	var want []string
	first, second := c.drawsTwoWindows(func() {
		want = c.plantHandBacks(w0.Add(time.Hour), "iss-70", "iss-71", "iss-72", "iss-73", "iss-74")
	}, func() {})
	var listed []string
	drawn := map[string]int{}
	for _, cy := range []Cycle{first, second} {
		g := groupOf(t, cy, GroupDrainHandBacks)
		if len(g.Population.Keys) != g.Population.Count || g.Population.SHA256 != "" || g.Population.Count > maxListedKeys {
			t.Fatalf("cycle %s keeps its hand-backs as %+v; want them listed, at most %d", cy.ID, g.Population, maxListedKeys)
		}
		listed = append(listed, g.Population.Keys...)
		for _, k := range g.Drawn {
			drawn[k]++
		}
	}
	slices.Sort(listed)
	if !slices.Equal(listed, want) {
		t.Fatalf("the two populations list %v; want each of %v once", listed, want)
	}
	for k, n := range drawn {
		if n > 1 {
			t.Fatalf("%s was drawn %d times", k, n)
		}
	}
	if got := groupOf(t, first, GroupDrainHandBacks).Population.Keys; !slices.Equal(got, want[:3]) {
		t.Fatalf("the first window took %v; want the first three keys %v", got, want[:3])
	}
}

// TestADrawReadsOnlyTheRecordsNoEarlierCycleListed: once a cycle lists the
// hand-backs on origin's branch, the next draw reads only what arrived since,
// so a branch holding more records than one read takes is drawn, from the
// new records alone.
func TestADrawReadsOnlyTheRecordsNoEarlierCycleListed(t *testing.T) {
	c := newDrawCase(t)
	var fresh []string
	_, second := c.drawsTwoWindows(func() {
		c.plantHandBacks(w0.Add(time.Hour), "iss-80", "iss-81", "iss-82", "iss-83", "iss-84", "iss-85")
	}, func() {
		saved := maxReadFiles
		maxReadFiles = 4
		t.Cleanup(func() { maxReadFiles = saved })
		fresh = c.plantHandBacks(q1000.Add(2*time.Hour), "iss-90", "iss-91")
	})
	if got := groupOf(t, second, GroupDrainHandBacks).Population.Keys; !slices.Equal(got, fresh) {
		t.Fatalf("the second window drew from %v; want only the records that arrived since, %v", got, fresh)
	}
}

// TestASecondCloneBehindOriginIsNotToldTheBranchWasRewritten: two clones of
// one origin under one home share the machine's mark. A clone that has not
// fetched what the other pushed is behind, not rewritten: its observation
// fetches the branch and goes on.
func TestASecondCloneBehindOriginIsNotToldTheBranchWasRewritten(t *testing.T) {
	latest := "1000"
	c := newDrawCase(t)
	c.serveRounds(&latest)
	b := clone(t, c.bare)
	observeB := func(at time.Time) string {
		b.Git("fetch", "-q", "origin", "main")
		return AfterLanding(t.Context(), Observed{Root: b.Root(), Default: "main", Site: SiteLand, Now: at})
	}
	c.openWindow(w0)
	if note := observeB(w0.Add(time.Hour)); strings.Contains(note, "rewritten") || strings.Contains(note, "first window") {
		t.Fatalf("a clone that never fetched the branch said %q", note)
	}
	if got := strings.TrimSpace(b.Git("rev-parse", TrackingRef)); got != c.originTip() {
		t.Fatalf("the clone's copy is %s; want origin's %s fetched", got, c.originTip())
	}
	c.land(2)
	c.closeWindow()
	if note := c.observe(q1000.Add(time.Hour)); !strings.Contains(note, "drew cycle") {
		t.Fatalf("the first clone's draw said %q", note)
	}
	if note := observeB(q1000.Add(2 * time.Hour)); strings.Contains(note, "rewritten") || strings.Contains(note, "drew cycle") {
		t.Fatalf("a clone behind the drawn cycle said %q", note)
	}
	if got := strings.TrimSpace(b.Git("rev-parse", TrackingRef)); got != c.originTip() {
		t.Fatalf("the clone's copy is %s; want origin's %s fetched", got, c.originTip())
	}
}

// TestAnOpeningWithinTheForgesLeadIsKept: an opening written by a clock a few
// seconds ahead of the forge's, whose commit's child the forge then commits
// within those seconds, is still the window's start.
func TestAnOpeningWithinTheForgesLeadIsKept(t *testing.T) {
	c := newDrawCase(t)
	at := w0.Add(3 * time.Second)
	if note := c.observe(at); !strings.Contains(note, "first window opened") {
		t.Fatalf("the first check said %q", note)
	}
	opened := c.originOpening()
	c.landAt(1, w0.Add(time.Second))
	if note := c.observe(w0.Add(time.Hour)); note != "" {
		t.Fatalf("the observation after the child said %q", note)
	}
	if w := c.originOpening(); w.Commit != opened.Commit || !w.At.Equal(at) {
		t.Fatalf("origin's opening is %+v; want %+v kept", w, opened)
	}
}

// TestAPassingReadFailureOfTheBlockIsNoDefaults: a block git cannot read at
// the window's start, for a reason that says nothing of the file, is an
// error, never a window counted by the bundled defaults: the observation
// refuses and draws nothing.
func TestAPassingReadFailureOfTheBlockIsNoDefaults(t *testing.T) {
	c := newDrawCase(t)
	c.r.Write(".abcd/config.json", `{"spotcheck":{"window_days":3}}`+"\n")
	c.r.Git("add", "-A")
	start := c.landAt(1, w0.Add(-time.Hour))
	c.openWindow(w0)
	tree := strings.TrimSpace(c.r.Git("rev-parse", start+"^{tree}"))
	if err := os.Remove(filepath.Join(c.r.Root(), ".git", "objects", tree[:2], tree[2:])); err != nil {
		t.Fatal(err)
	}
	cfg, why, err := windowConfig(context.Background(), c.r.Root(), start)
	if err == nil {
		t.Fatalf("windowConfig over a tree git cannot read = %+v, %q, nil; want an error", cfg, why)
	}
	if !strings.Contains(err.Error(), "was not read") || why != "" {
		t.Fatalf("the error is %v (why %q); want the read named as not made", err, why)
	}
	if note := c.observe(w0.Add(time.Hour)); strings.Contains(note, "bundled defaults") || !strings.Contains(note, "nothing was drawn") {
		t.Fatalf("the observation over a block git cannot read said %q", note)
	}
}

// TestABlockOverTheSizeBoundCountsByTheDefaults: a spotcheck block whose
// file at the window's start is larger than a configuration file may be is a
// file read and refused: the window is counted by the bundled defaults, says
// so, and is drawn.
func TestABlockOverTheSizeBoundCountsByTheDefaults(t *testing.T) {
	latest := "1000"
	c := newDrawCase(t)
	c.serveRounds(&latest)
	c.r.Write(".abcd/config.json", `{"spotcheck":{"window_days":3},"pad":"`+strings.Repeat("x", 1<<20)+`"}`+"\n")
	c.r.Git("add", "-A")
	start := c.landAt(1, w0.Add(-time.Hour))
	c.openWindow(w0)
	cfg, why, err := windowConfig(context.Background(), c.r.Root(), start)
	if err != nil || why == "" || cfg.WindowDays != DefaultWindowDays || cfg.PerGroup != DefaultPerGroup {
		t.Fatalf("windowConfig over a block past the size bound = %+v, %q, %v; want the defaults with a reason", cfg, why, err)
	}
	c.land(2)
	c.closeWindow()
	if note := c.observe(q1000.Add(time.Hour)); !strings.Contains(note, "drew cycle") || !strings.Contains(note, "bundled defaults") {
		t.Fatalf("the draw over a block past the size bound said %q", note)
	}
}

// TestARecordThatDoesNotReadTakesNoPlaceInAWindow: files under record names
// whose content is not the record their name gives take none of a window's
// places: the window is filled from the records that read, and the draw's
// note counts the files that did not.
func TestARecordThatDoesNotReadTakesNoPlaceInAWindow(t *testing.T) {
	saved := maxListedKeys
	maxListedKeys = 2
	t.Cleanup(func() { maxListedKeys = saved })
	latest := "1000"
	c := newDrawCase(t)
	c.serveRounds(&latest)
	c.openWindow(w0)
	unread := map[string]string{}
	for _, issue := range []string{"iss-60", "iss-61", "iss-62"} {
		f, err := handBack(issue, w0.Add(time.Minute)).typed().file()
		if err != nil {
			t.Fatal(err)
		}
		unread[f.path()] = "{}\n"
	}
	c.plant(unread)
	want := c.plantHandBacks(w0.Add(time.Hour), "iss-70", "iss-71")
	c.closeWindow()
	note := c.observe(q1000.Add(time.Hour))
	if !strings.Contains(note, "drew cycle") || !strings.Contains(note, "3 record file(s)") {
		t.Fatalf("the draw over records that do not read said %q", note)
	}
	if strings.Contains(note, "places") {
		t.Fatalf("a window whose places were filled says some were left open: %q", note)
	}
	if got := groupOf(t, c.newestCycle(), GroupDrainHandBacks).Population.Keys; !slices.Equal(got, want) {
		t.Fatalf("the window took %v; want the records that read, %v", got, want)
	}
}

// TestACycleWhoseRecordGroupIsNotListedIsSetAside: a cycle from another
// writer that keeps a record group by its hash, though it verifies in every
// other way, is set aside and its window drawn afresh.
func TestACycleWhoseRecordGroupIsNotListedIsSetAside(t *testing.T) {
	saved := maxListedKeys
	maxListedKeys = 1
	t.Cleanup(func() { maxListedKeys = saved })
	latest := "1000"
	c := newDrawCase(t)
	c.serveRounds(&latest)
	from := c.openWindow(w0)
	line := c.landAt(3, w0.Add(time.Hour))
	c.closeWindow()
	forged := handMade(from, line, ClosedByDays)
	for i, g := range forged.Groups {
		if g.Group == GroupDrainHandBacks {
			forged.Groups[i] = GroupDraw{Group: g.Group, Population: Population{Count: 2, First: "a", Last: "b", SHA256: strings.Repeat("0", 64)},
				Drawn: []string{"a", "b"}}
		}
	}
	c.plant(map[string]string{CyclesDir + "/" + forged.ID + ".json": cycleJSON(t, forged, sevenDays(3))})
	note := c.observe(q1000.Add(time.Hour))
	if !strings.Contains(note, "drew cycle "+cycleID(q1000, line)) || !strings.Contains(note, forged.ID+" on origin does not verify (its "+GroupDrainHandBacks+" population is not listed") {
		t.Fatalf("the observation over a cycle with an unlisted record group said %q", note)
	}
}

// plantUnread puts a file under a hand-back's name for each issue on origin's
// store branch, made at at, whose content is not the record its name gives.
func (c *drawCase) plantUnread(at time.Time, issues ...string) {
	c.t.Helper()
	files := map[string]string{}
	for _, issue := range issues {
		f, err := handBack(issue, at).typed().file()
		if err != nil {
			c.t.Fatal(err)
		}
		files[f.path()] = "{}\n"
	}
	c.plant(files)
}

// TestAGroupWhoseReadBoundEndsAfterARecordIsDrawn: a group whose read bound
// is spent after some of its records read is drawn from the records that
// read, and the note counts the files that did not.
func TestAGroupWhoseReadBoundEndsAfterARecordIsDrawn(t *testing.T) {
	latest := "1000"
	c := newDrawCase(t)
	c.serveRounds(&latest)
	c.openWindow(w0)
	saved := maxReadFiles
	maxReadFiles = 4
	t.Cleanup(func() { maxReadFiles = saved })
	c.plantUnread(w0.Add(time.Minute), "iss-60")
	want := c.plantHandBacks(w0.Add(time.Hour), "iss-70")
	c.plantUnread(w0.Add(2*time.Hour), "iss-80", "iss-81")
	c.closeWindow()
	note := c.observe(q1000.Add(time.Hour))
	if !strings.Contains(note, "drew cycle") || !strings.Contains(note, "1 record file(s)") {
		t.Fatalf("the draw over a group whose read bound ended after a record said %q", note)
	}
	if got := groupOf(t, c.newestCycle(), GroupDrainHandBacks).Population.Keys; !slices.Equal(got, want) {
		t.Fatalf("the window took %v; want the record that read, %v", got, want)
	}
}

// TestAGroupThatListsNoRecordWithinItsReadBoundIsNotDrawn: a group whose
// read bound is spent before any of its files reads as a record refuses the
// draw, naming the count, and writes nothing.
func TestAGroupThatListsNoRecordWithinItsReadBoundIsNotDrawn(t *testing.T) {
	latest := "1000"
	c := newDrawCase(t)
	c.serveRounds(&latest)
	c.openWindow(w0)
	saved := maxReadFiles
	maxReadFiles = 4
	t.Cleanup(func() { maxReadFiles = saved })
	c.plantUnread(w0.Add(time.Minute), "iss-60", "iss-61", "iss-62")
	c.plantHandBacks(w0.Add(time.Hour), "iss-70")
	c.closeWindow()
	tip := c.originTip()
	note := c.observe(q1000.Add(time.Hour))
	if !strings.Contains(note, "listed none of its records") || !strings.Contains(note, "2 record file(s)") || !strings.Contains(note, "nothing was drawn") {
		t.Fatalf("the draw over a group that listed no record said %q", note)
	}
	if c.originTip() != tip {
		t.Fatal("a refused draw moved origin's branch")
	}
}

// TestANoteSaysHowManyPlacesAGroupLeftOpen: a group whose read bound is spent
// on files that do not read, before its window's places are filled, is drawn
// from the records that read, and the note says how many places it drew from
// of how many its window has, beside the count of files that did not read.
func TestANoteSaysHowManyPlacesAGroupLeftOpen(t *testing.T) {
	savedKeys, savedFiles := maxListedKeys, maxReadFiles
	maxListedKeys, maxReadFiles = 3, 4
	t.Cleanup(func() { maxListedKeys, maxReadFiles = savedKeys, savedFiles })
	latest := "1000"
	c := newDrawCase(t)
	c.serveRounds(&latest)
	c.openWindow(w0)
	c.plantUnread(w0.Add(time.Minute), "iss-60")
	want := c.plantHandBacks(w0.Add(time.Hour), "iss-70")
	c.plantUnread(w0.Add(2*time.Hour), "iss-80", "iss-81", "iss-82")
	c.closeWindow()
	note := c.observe(q1000.Add(time.Hour))
	if !strings.Contains(note, "drew cycle") || !strings.Contains(note, GroupDrainHandBacks+" drew from 1 of 3 places") || !strings.Contains(note, "1 record file(s)") {
		t.Fatalf("the draw over a group that left places open said %q", note)
	}
	if strings.Contains(note, GroupFacilitatorDecisions) {
		t.Fatalf("the note names a group whose read bound held: %q", note)
	}
	if got := groupOf(t, c.newestCycle(), GroupDrainHandBacks).Population.Keys; !slices.Equal(got, want) {
		t.Fatalf("the window took %v; want the record that read, %v", got, want)
	}
}
