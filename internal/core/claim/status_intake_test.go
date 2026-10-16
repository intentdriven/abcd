package claim

import (
	"context"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// The intake's cost (C12): what one intake takes in is bounded before it is
// written, and what it writes is not flushed, so a flood of new announcements
// never holds the prompt hook past its deadline.

// fetchOnly fetches the status branches into p as `abcd announce`'s fetch does,
// with no intake after it, so the next read is the first to take them in.
func (p *participant) fetchOnly() {
	p.w.t.Helper()
	if _, err := fetch(p.dir, p.w.clock(), "nothing was fetched", statusBranches); err != nil {
		p.w.t.Fatalf("fetch: %v", err)
	}
}

// TestAFirstIntakeOfAFloodFitsTheHooksDeadline: with flushing on, as a shipped
// binary writes, the first intake of 300 new announcements runs inside the
// prompt hook's read and answers within the hook's deadline; what it does not
// take in is counted as doubtful, never left to hold the hook.
func TestAFirstIntakeOfAFloodFitsTheHooksDeadline(t *testing.T) {
	t.Setenv(fsutil.SkipFlushEnv, "0")
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	var msgs []string
	for i := 0; i < 300; i++ {
		msgs = append(msgs, announcementMsg("merging", "#"+strconv.Itoa(1+i), "maya", t0.Add(time.Duration(i)*time.Second), 30*time.Minute))
	}
	w.pushMany("maya", msgs)
	pat.fetchOnly()
	t.Cleanup(promptReads.Wait)

	start := time.Now()
	deadline := start.Add(time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	n, err := PromptNotes(ctx, pat.dir, w.clock(), pat.opts())
	if err != nil {
		t.Fatalf("PromptNotes = %v; want the flood read inside the hook's deadline", err)
	}
	n.MarkShown()
	t.Logf("the first intake of 300 announcements, flushing on, took %s", time.Since(start))
	if left := time.Until(deadline); left < 0 {
		t.Fatalf("the first intake of 300 announcements ended %s past the hook's deadline", -left)
	}
	v := pat.read()
	if len(v.Doubts) != 1 || v.Doubts[0].Branch != "maya" || v.Doubts[0].Reason != DoubtMore {
		t.Fatalf("doubts = %+v; want maya doubtful, past what one intake takes in", v.Doubts)
	}
	if got := len(v.Announcements); got != maxIntakeCommits {
		t.Fatalf("%d announcements taken in; want the newest %d", got, maxIntakeCommits)
	}
	if v.Announcements[0].Item != "#300" {
		t.Fatalf("the newest announcement taken in is %s; want #300", v.Announcements[0].Item)
	}
}

// TestTheIntakesCapSpansEveryBranch: the cap counts the new commits of every
// branch together, so a branch read once the cap is spent takes in what is
// left of it, the newest first, and is doubtful; a branch read in full is
// not.
func TestTheIntakesCapSpansEveryBranch(t *testing.T) {
	old := maxIntakeCommits
	maxIntakeCommits = 3
	t.Cleanup(func() { maxIntakeCommits = old })
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	w.pushMany("amy", []string{
		announcementMsg("merging", "#1", "amy", t0, 30*time.Minute),
		announcementMsg("merging", "#2", "amy", t0, 30*time.Minute),
	})
	w.pushMany("bob", []string{
		announcementMsg("merging", "#3", "bob", t0, 30*time.Minute),
		announcementMsg("merging", "#4", "bob", t0, 30*time.Minute),
	})
	pat.status(nil)
	v := pat.read()
	var items []string
	for _, a := range v.Announcements {
		items = append(items, a.Branch+" "+a.Item)
	}
	if len(items) != 3 || !slices.Contains(items, "bob #4") || slices.Contains(items, "bob #3") {
		t.Fatalf("taken in: %q; want amy's two and bob's newest", items)
	}
	if len(v.Doubts) != 1 || v.Doubts[0].Branch != "bob" || v.Doubts[0].Reason != DoubtMore {
		t.Fatalf("doubts = %+v; want bob doubtful, past what one intake takes in", v.Doubts)
	}
}
