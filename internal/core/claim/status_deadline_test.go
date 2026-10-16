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
// hook be held past its second.
func TestNothingIsMarkedPastTheHooksDeadline(t *testing.T) {
	w := newWorld(t)
	maya := w.clone("5+maya@users.noreply.github.com", "")
	pat := w.clone("9+pat@users.noreply.github.com", "")
	maya.announceAs("merging", "#1012")
	pat.status(nil)
	deadline := time.Now().Add(400 * time.Millisecond)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	n, err := PromptNotes(ctx, pat.dir, w.clock(), pat.opts())
	if err != nil || !strings.Contains(n.Text, "merging #1012") {
		t.Fatalf("PromptNotes = %q, %v", n.Text, err)
	}
	time.Sleep(time.Until(deadline) + 50*time.Millisecond)
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
