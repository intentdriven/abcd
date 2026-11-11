package claim

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The index of unshown announcements (iss-2610161839527371): a prompt reads
// the index and the announcements it names, never every announcement kept.

// TestAPromptReadsOnlyTheUnshownAnnouncements: with announcements kept and
// shown, a prompt reads back only those taken in since, and none when nothing
// was; `abcd status`'s read still reads them all.
func TestAPromptReadsOnlyTheUnshownAnnouncements(t *testing.T) {
	w := newWorld(t)
	maya := w.clone("5+maya@users.noreply.github.com", "")
	pat := w.clone("9+pat@users.noreply.github.com", "")
	for i := 1; i <= 5; i++ {
		maya.announceAs("merging", fmt.Sprintf("#%d", i))
		w.set(w.clock().Add(time.Second))
	}
	pat.status(nil)
	if got := pat.notes(); !strings.Contains(got, "merging #5") || !strings.Contains(got, "merging #1") {
		t.Fatalf("the first block:\n%s\nwant all five announcements", got)
	}
	w.set(w.clock().Add(time.Second))

	// Nothing new: the prompt reads no kept announcement.
	keptReads.Store(0)
	if got := pat.notes(); got != "" {
		t.Fatalf("a second block:\n%s\nwant nothing: every line was shown", got)
	}
	if n := keptReads.Load(); n != 0 {
		t.Fatalf("a prompt with nothing unshown read %d kept announcements; want none", n)
	}

	// One new announcement: the prompt reads it alone.
	maya.announceAs("merging", "#6")
	pat.status(nil)
	w.set(w.clock().Add(time.Second))
	keptReads.Store(0)
	got := pat.notes()
	if !strings.Contains(got, "merging #6") || strings.Contains(got, "merging #5") {
		t.Fatalf("the block:\n%s\nwant #6 alone", got)
	}
	if n := keptReads.Load(); n != 1 {
		t.Fatalf("a prompt with one unshown announcement among six kept read %d; want 1", n)
	}
	if got := pat.notes(); got != "" {
		t.Fatalf("the block after #6 was shown:\n%s\nwant nothing", got)
	}
	// The status command's read is still the full one.
	if v := pat.read(); len(v.Announcements) != 6 {
		t.Fatalf("`abcd status` reads %d announcements; want all 6", len(v.Announcements))
	}
}

// indexFiles is the index files of unshown announcements p keeps.
func (p *participant) indexFiles() []string {
	var names []string
	for _, e := range statusEntries(p.dir, unshownDir) {
		if strings.HasPrefix(e.Name(), unshownPrefix) {
			names = append(names, e.Name())
		}
	}
	return names
}

// TestAnnouncementsKeptBeforeTheIndexAreStillShown: a checkout whose
// announcements were taken in by a build that kept no index shows each
// unshown one at the next prompt, once, and reads none after.
func TestAnnouncementsKeptBeforeTheIndexAreStillShown(t *testing.T) {
	w := newWorld(t)
	maya := w.clone("5+maya@users.noreply.github.com", "")
	pat := w.clone("9+pat@users.noreply.github.com", "")
	maya.announceAs("merging", "#1")
	pat.status(nil)
	if got := pat.notes(); !strings.Contains(got, "merging #1") {
		t.Fatalf("the first block:\n%s\nwant #1", got)
	}
	w.set(w.clock().Add(time.Second))
	maya.announceAs("merging", "#2")
	pat.status(nil)
	// As an earlier build left it: kept, with no index.
	if err := os.RemoveAll(filepath.Join(pat.dir, filepath.FromSlash(statusFile(unshownDir)))); err != nil {
		t.Fatal(err)
	}
	w.set(w.clock().Add(time.Second))
	got := pat.notes()
	if !strings.Contains(got, "merging #2") || strings.Contains(got, "merging #1") {
		t.Fatalf("the block:\n%s\nwant #2, kept before the index, and not #1 again", got)
	}
	if got := pat.notes(); got != "" {
		t.Fatalf("the next block:\n%s\nwant nothing", got)
	}
	if left := pat.indexFiles(); len(left) != 0 {
		t.Fatalf("index files left once everything is shown: %v; want none", left)
	}
	keptReads.Store(0)
	_ = pat.notes()
	if n := keptReads.Load(); n != 0 {
		t.Fatalf("once the index is built, a prompt with nothing unshown read %d kept announcements; want none", n)
	}
}

// TestALostShownMarkerShowsTheAnnouncementAgain: a block that was never kept
// as shown leaves its announcements in the index, so the next prompt shows
// them again.
func TestALostShownMarkerShowsTheAnnouncementAgain(t *testing.T) {
	w := newWorld(t)
	maya := w.clone("5+maya@users.noreply.github.com", "")
	pat := w.clone("9+pat@users.noreply.github.com", "")
	maya.announceAs("merging", "#1")
	pat.status(nil)
	stretchPromptBudget(t)
	n, err := PromptNotes(t.Context(), pat.dir, w.clock(), pat.opts())
	if err != nil || !strings.Contains(n.Text, "merging #1") {
		t.Fatalf("PromptNotes = %q, %v; want #1", n.Text, err)
	}
	// MarkShown never ran.
	if got := pat.notes(); !strings.Contains(got, "merging #1") {
		t.Fatalf("the next block:\n%s\nwant #1 again: it was never kept as shown", got)
	}
	if got := pat.notes(); got != "" {
		t.Fatalf("the block after it was kept shown:\n%s\nwant nothing", got)
	}
}

// TestAnOlderAnnouncementTakenInLateIsStillCounted: an announcement taken in
// beneath one already shown, for the same item, is counted, not shown as the
// item's news: the prompt reads every kept announcement whenever an unshown
// one is no newer than one that left the index.
func TestAnOlderAnnouncementTakenInLateIsStillCounted(t *testing.T) {
	w := newWorld(t)
	maya := w.clone("5+maya@users.noreply.github.com", "")
	pat := w.clone("9+pat@users.noreply.github.com", "")
	maya.announceAs("merging", "#1")
	pat.status(nil)
	v := pat.read()
	if len(v.Announcements) != 1 {
		t.Fatalf("announcements = %+v; want one", v.Announcements)
	}
	shown := v.Announcements[0]
	if got := pat.notes(); !strings.Contains(got, "merging #1") {
		t.Fatalf("the first block:\n%s\nwant #1", got)
	}
	// An older commit of the same branch, for the same item, kept now with
	// the time and place beneath the one shown, as a later intake keeps what
	// an earlier one left.
	older := strings.Repeat("ab", 20)
	keepSeen(pat.dir, shown.Branch, []string{older}, map[string]readAnnouncement{older: {a: announcement{
		Kind: "asking", Item: "#1", Sender: shown.Sender, SentAt: shown.SentAt.Add(-time.Minute), ExpiresAt: shown.SentAt.Add(29 * time.Minute)}}},
		shown.FirstFetched, shown.order-1)
	w.set(w.clock().Add(time.Second))
	got := pat.notes()
	if strings.Contains(got, "asking #1") || !strings.Contains(got, "- and 1 more: `abcd status` lists them all\n") {
		t.Fatalf("the block:\n%s\nwant the older announcement counted, not shown as #1's news", got)
	}
	if got := pat.notes(); got != "" {
		t.Fatalf("the next block:\n%s\nwant nothing", got)
	}
}

// TestTheIndexIsPrunedWithTheAnnouncementsItNames: an index file no prompt
// removed goes once it was written longer ago than a kept announcement is
// kept, and a temporary file a write cut off left in the index goes too.
func TestTheIndexIsPrunedWithTheAnnouncementsItNames(t *testing.T) {
	w := newWorld(t)
	maya := w.clone("5+maya@users.noreply.github.com", "")
	pat := w.clone("9+pat@users.noreply.github.com", "")
	maya.announceAs("merging", "#1")
	pat.status(nil)
	files := pat.indexFiles()
	if len(files) != 1 {
		t.Fatalf("index files after one intake: %v; want one", files)
	}
	dir := filepath.Join(pat.dir, filepath.FromSlash(statusFile(unshownDir)))
	aged := time.Now().Add(-(MaxAnnouncementLife + seenKeep + time.Hour))
	if err := os.Chtimes(filepath.Join(dir, files[0]), aged, aged); err != nil {
		t.Fatal(err)
	}
	tmp := filepath.Join(dir, tmpPrefix+"0123456789abcdef")
	if err := os.WriteFile(tmp, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(tmp, aged, aged); err != nil {
		t.Fatal(err)
	}
	// A new intake prunes.
	w.set(w.clock().Add(time.Second))
	maya.announceAs("merging", "#2")
	pat.status(nil)
	left := pat.indexFiles()
	if len(left) != 1 || left[0] == files[0] {
		t.Fatalf("index files after the prune: %v; want only the new intake's", left)
	}
	if _, err := os.Lstat(tmp); !os.IsNotExist(err) {
		t.Fatalf("the cut-off temporary file is still in the index (%v)", err)
	}
}
