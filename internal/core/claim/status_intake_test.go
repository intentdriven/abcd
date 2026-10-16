package claim

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
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
// take in is left for the next intake, never left to hold the hook.
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
	// The read is a second intake, which takes in the next newest.
	v := pat.read()
	if len(v.Doubts) != 0 {
		t.Fatalf("doubts = %+v; want none: the cap leaves the rest for the next intake", v.Doubts)
	}
	if got := len(v.Announcements); got != 2*maxIntakeCommits {
		t.Fatalf("%d announcements taken in after two intakes; want %d", got, 2*maxIntakeCommits)
	}
	if v.Announcements[0].Item != "#300" {
		t.Fatalf("the newest announcement taken in is %s; want #300", v.Announcements[0].Item)
	}
}

// TestTheIntakesCapSpansEveryBranch: the cap counts the new commits of every
// branch together, so a branch read once the cap is spent takes in what is
// left of it, the newest first, and the next intake takes in the rest; no
// branch is doubtful for the cap alone.
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
	// The fetch takes in three (amy's two and bob's newest); the read is a
	// second intake, which takes in bob's other.
	pat.status(nil)
	v := pat.read()
	var items []string
	for _, a := range v.Announcements {
		items = append(items, a.Branch+" "+a.Item)
	}
	if len(items) != 4 || !slices.Contains(items, "bob #3") {
		t.Fatalf("taken in after two intakes: %q; want all four, bob #3 among them", items)
	}
	if len(v.Doubts) != 0 {
		t.Fatalf("doubts = %+v; want none: the cap leaves the rest for the next intake", v.Doubts)
	}
}

// TestABranchTheCapReachesWithNothingLeftIsReadNext: once the cap is spent,
// a later branch takes in nothing and is not marked read, so its
// announcement is shown after the next intake rather than passed over.
func TestABranchTheCapReachesWithNothingLeftIsReadNext(t *testing.T) {
	old := maxIntakeCommits
	maxIntakeCommits = 2
	t.Cleanup(func() { maxIntakeCommits = old })
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	w.pushMany("amy", []string{
		announcementMsg("merging", "#1", "amy", t0, 30*time.Minute),
		announcementMsg("merging", "#2", "amy", t0, 30*time.Minute),
	})
	w.pushMany("bob", []string{announcementMsg("merging", "#3", "bob", t0, 30*time.Minute)})
	w.pushMany("carl", []string{announcementMsg("merging", "#9", "carl", t0, 30*time.Minute)})
	pat.status(nil)
	pat.status(nil)
	var items []string
	for _, a := range pat.read().Announcements {
		items = append(items, a.Branch+" "+a.Item)
	}
	for _, want := range []string{"amy #1", "amy #2", "bob #3", "carl #9"} {
		if !slices.Contains(items, want) {
			t.Fatalf("after two intakes: %q; want %q among them", items, want)
		}
	}
}

// TestATemporaryFileACutOffWriteLeftIsPruned: a hook cut off mid-write leaves
// its temporary file behind in the status state; the next intake that takes
// something in removes every one older than a minute, and leaves a newer
// one, which a write running beside it may still rename into place.
func TestATemporaryFileACutOffWriteLeftIsPruned(t *testing.T) {
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	w.pushStatus("maya", announcementMsg("merging", "#1", "maya", t0, 30*time.Minute), "")
	pat.status(nil)
	put := func(rel string, age time.Duration) string {
		t.Helper()
		p := filepath.Join(pat.dir, filepath.FromSlash(statusFile(rel)))
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("{"), 0o600); err != nil {
			t.Fatal(err)
		}
		at := time.Now().Add(-age)
		if err := os.Chtimes(p, at, at); err != nil {
			t.Fatal(err)
		}
		return p
	}
	stale := []string{
		put(seenDir+"/.tmp-0123456789abcdef", time.Hour),
		put(doubtDir+"/.tmp-0123456789abcdef", time.Hour),
		put(shownDir+"/.tmp-0123456789abcdef", time.Hour),
		put("read/.tmp-0123456789abcdef", time.Hour),
		put(seenDir+"/.abcd-tmp-0123456789abcdef", time.Hour),
	}
	fresh := put(seenDir+"/.tmp-fedcba9876543210", 0)
	w.pushStatus("quinn", announcementMsg("merging", "#2", "quinn", t0, 30*time.Minute), "")
	pat.status(nil)
	for _, p := range stale {
		if _, err := os.Lstat(p); !os.IsNotExist(err) {
			t.Fatalf("%s survived the intake: %v", p, err)
		}
	}
	if _, err := os.Lstat(fresh); err != nil {
		t.Fatalf("a temporary file under a minute old was removed: %v", err)
	}
}

// TestABranchAheadThatSpendsTheCapDoesNotHideTheNext: a branch first in
// git's order holding as many new commits as one intake takes in spends that
// intake; the branch after it is not walked and not marked read, the block
// says one branch is not read yet (never that `abcd status` reads it in
// full), and the next intake takes in its announcement, even when the branch
// ahead gained as many again in between: it starts where the last one's cap
// ran out.
func TestABranchAheadThatSpendsTheCapDoesNotHideTheNext(t *testing.T) {
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	flood := func(from int) {
		var msgs []string
		for i := 0; i < maxIntakeCommits; i++ {
			msgs = append(msgs, announcementMsg("merging", "#"+strconv.Itoa(from+i), "aaa", t0, 30*time.Minute))
		}
		w.pushMany("aaa", msgs)
	}
	flood(1)
	w.pushStatus("bob", announcementMsg("merging", "#9999", "bob", t0, 30*time.Minute), "")
	pat.fetchOnly()
	log := traced(t)
	first := pat.notes()
	traceGit = nil
	if log.anyArgContains(StatusTrackingPrefix + "bob") {
		t.Fatal("bob was walked once the intake's cap was spent")
	}
	if strings.Contains(first, "#9999") || strings.Contains(first, "(doubtful)") {
		t.Fatalf("the first block:\n%s\nwant bob neither shown nor doubtful", first)
	}
	notYet := "- 1 status branch not read yet: one read takes in at most 100 new commits; the next prompt or `abcd status` reads on\n"
	if !strings.Contains(first, notYet) {
		t.Fatalf("the first block:\n%s\nwant %q", first, notYet)
	}
	if strings.Contains(first, "reads it in full") {
		t.Fatalf("the first block:\n%s\nsays `abcd status` reads it in full", first)
	}
	flood(1001)
	pat.fetchOnly()
	v := pat.read()
	var items []string
	for _, a := range v.Announcements {
		items = append(items, a.Branch+" "+a.Item)
	}
	if !slices.Contains(items, "bob #9999") || v.BranchesLeft != 1 {
		t.Fatalf("after the second intake: %d left, taken in %q; want bob's announcement, and aaa left", v.BranchesLeft, items)
	}
}

// TestACapSpentElsewhereLeavesNoLastingDoubt: a branch the cap cut short
// reads as not read yet, never as a doubtful branch, and once a later intake
// has taken in the rest the line is gone and the branch is marked read at
// its tip.
func TestACapSpentElsewhereLeavesNoLastingDoubt(t *testing.T) {
	old := maxIntakeCommits
	maxIntakeCommits = 2
	t.Cleanup(func() { maxIntakeCommits = old })
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	tip := w.pushMany("bob", []string{
		announcementMsg("merging", "#1", "bob", t0, 30*time.Minute),
		announcementMsg("merging", "#2", "bob", t0, 30*time.Minute),
		announcementMsg("merging", "#3", "bob", t0, 30*time.Minute),
	})
	pat.fetchOnly()
	if v := pat.read(); v.BranchesLeft != 1 || len(v.Doubts) != 0 {
		t.Fatalf("after one intake: %d branches left, doubts %+v; want bob left and no doubt", v.BranchesLeft, v.Doubts)
	}
	if v := pat.read(); v.BranchesLeft != 0 || len(v.Doubts) != 0 || len(v.Announcements) != 3 {
		t.Fatalf("after two intakes: %d branches left, doubts %+v, %d announcements; want all three, none left", v.BranchesLeft, v.Doubts, len(v.Announcements))
	}
	if got := statusBranches.lastRead(pat.dir, StatusTrackingPrefix+"bob"); got != tip {
		t.Fatalf("bob is read at %q; want its tip %s", got, tip)
	}
}

// TestKeepingManyDoubtsFitsThePrompt: an intake that finds each of
// maxStatusBranches branches doubtful again, each kept already for the same
// reason at the same tip, tells so from that branch's own file alone, so
// keeping them all stays well inside the prompt hook's read.
func TestKeepingManyDoubtsFitsThePrompt(t *testing.T) {
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	now := w.clock()
	tip := strings.Repeat("a", 40)
	var ds []BranchDoubt
	for i := 0; i < maxStatusBranches; i++ {
		d := BranchDoubt{Branch: "user" + strconv.Itoa(i), Reason: DoubtMore, Tip: tip, FoundAt: now}
		keepDoubt(pat.dir, d)
		ds = append(ds, d)
	}
	start := time.Now()
	for _, d := range ds {
		keepDoubt(pat.dir, d)
	}
	took := time.Since(start)
	t.Logf("keeping %d doubts kept already took %s", len(ds), took)
	if took > 300*time.Millisecond {
		t.Fatalf("keeping %d doubts kept already took %s; want it well inside the prompt's read, under 300 ms", len(ds), took)
	}
	if got := len(keptDoubts(pat.dir, now)); got != len(ds) {
		t.Fatalf("%d doubts kept; want %d", got, len(ds))
	}
}

