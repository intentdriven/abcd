package claim

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The prompt hook's one deadline (C12): the status read, its render and the
// marking of what it showed all fit inside the deadline the hook set at its
// start.

// TestTheReadLeavesTheHookTimeToMarkWhatItShowed: given the hook's deadline,
// a read held past it returns at most promptMarkReserve before that deadline,
// so the hook still has time to write the block and mark it, never at the
// deadline itself.
func TestTheReadLeavesTheHookTimeToMarkWhatItShowed(t *testing.T) {
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	release := make(chan struct{})
	hold := func(read func()) {
		<-release
		read()
	}
	statusReadHook.Store(&hold)
	t.Cleanup(func() {
		close(release)
		promptReads.Wait()
		statusReadHook.Store(nil)
	})
	deadline := time.Now().Add(500 * time.Millisecond)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	_, err := PromptNotes(ctx, pat.dir, w.clock(), pat.opts())
	left := time.Until(deadline)
	if !errors.Is(err, ErrStatusUnavailable) {
		t.Fatalf("a held read = %v; want the status unavailable", err)
	}
	if left < promptMarkReserve/2 {
		t.Fatalf("the read returned %s before the hook's deadline; want about %s left to mark and emit", left, promptMarkReserve)
	}
}

// TestNothingIsMarkedPastTheHooksDeadline: a block marked after the deadline
// the hook set keeps nothing, so its lines are shown again rather than the
// hook be held past its second. The read gets the stretched budget and a
// deadline no machine's git outruns, and the notes' own deadline is then
// moved into the past: the test is of the marking, not of how long the read
// took (iss-2610251853204621).
func TestNothingIsMarkedPastTheHooksDeadline(t *testing.T) {
	stretchPromptBudget(t)
	w := newWorld(t)
	maya := w.clone("5+maya@users.noreply.github.com", "")
	pat := w.clone("9+pat@users.noreply.github.com", "")
	maya.announceAs("merging", "#1012")
	pat.status(nil)
	deadline := time.Now().Add(time.Minute)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	n, err := PromptNotes(ctx, pat.dir, w.clock(), pat.opts())
	if err != nil || !strings.Contains(n.Text, "merging #1012") {
		t.Fatalf("PromptNotes = %q, %v", n.Text, err)
	}
	if want := deadline.Add(-promptWriteReserve); !n.by.Equal(want) {
		t.Fatalf("the notes are marked by %v; want the hook's deadline less its write reserve, %v", n.by, want)
	}
	n.by = time.Now().Add(-time.Millisecond)
	n.MarkShown()
	if got := pat.notes(); !strings.Contains(got, "merging #1012") {
		t.Fatalf("after a block marked past the hook's deadline, the block:\n%s\nwant the line again", got)
	}
}

// TestPruningStopsWhenItsContextEnds: the prune runs inside the read, so it
// stops when the read's context ends and removes nothing more.
func TestPruningStopsWhenItsContextEnds(t *testing.T) {
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	keepDoubt(pat.dir, BranchDoubt{Branch: "gone", Reason: DoubtMerged, Tip: strings.Repeat("a", 40), FoundAt: t0.Add(-2 * MaxAnnouncementLife)})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	pruneStatus(ctx, pat.dir, t0, nil)
	if _, err := os.Lstat(filepath.Join(pat.dir, filepath.FromSlash(statusFile(doubtDir+"/"+hashName("gone"))))); err != nil {
		t.Fatalf("a prune whose context had ended removed a doubt: %v", err)
	}
}

// TestAPromptWithNoTimeLeftSaysSoInItsOwnWords: when the hook's deadline
// leaves the read nothing, no read is started, and the block says so in its
// own fixed line, never that a read took too long.
func TestAPromptWithNoTimeLeftSaysSoInItsOwnWords(t *testing.T) {
	w := newWorld(t)
	maya := w.clone("5+maya@users.noreply.github.com", "")
	pat := w.clone("9+pat@users.noreply.github.com", "")
	maya.announceAs("merging", "#1012")
	pat.status(nil)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(promptMarkReserve/2))
	defer cancel()
	n, err := PromptNotes(ctx, pat.dir, w.clock(), pat.opts())
	want := NotesHeading + "\n\nstatus unavailable: no time was left to read it at this prompt; `abcd status` reads it without this prompt's deadline\n"
	if !errors.Is(err, ErrStatusUnavailable) || n.Text != want {
		t.Fatalf("PromptNotes = %q, %v; want %q", n.Text, err, want)
	}
}

// TestTheFoldedDoubtsCountTheAnnouncementsToo: the line that sums up more
// doubts than the block shows one by one counts a doubtful branch and an
// announcement that does not read alike, and says it covers both.
func TestTheFoldedDoubtsCountTheAnnouncementsToo(t *testing.T) {
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	for _, b := range []string{"amy", "bob"} {
		w.pushStatus(b, "not an announcement from "+b+"\n", "")
	}
	tip := w.pushStatus("cal", announcementMsg("merging", "#1", "cal", t0, 30*time.Minute), "")
	side := w.pushStatus("side", announcementMsg("merging", "#3", "cal", t0, 30*time.Minute), "")
	runGit(t, w.bare, "update-ref", "-d", StatusRefPrefix+"side")
	w.pushStatus("cal", announcementMsg("merging", "#2", "cal", t0, 30*time.Minute), "", tip, side)
	pat.status(nil)
	got := pat.notes()
	want := "- 3 doubts about the status branches and their announcements, not shown one by one; `abcd status` lists them\n"
	if !strings.Contains(got, want) {
		t.Fatalf("the block:\n%s\nwant %q", got, want)
	}
}
