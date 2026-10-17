package claim

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The order one intake reads the status branches in, and the commits of each:
// every branch left unread is walked before any branch read since, in the
// order they were left, and a branch already left takes in its oldest unread
// commits beside its newest, so no commit is left unread for more than a
// bounded number of intakes while the commits above a branch's mark stay
// within the walk's limit.

// withIntakeCap lowers maxIntakeCommits to n for the test.
func withIntakeCap(t *testing.T, n int) {
	t.Helper()
	old := maxIntakeCommits
	maxIntakeCommits = n
	t.Cleanup(func() { maxIntakeCommits = old })
}

// refill appends n announcements to abcd-status/<branch> on origin, numbered
// from *next on, and moves *next past them.
func (w *world) refill(branch string, n int, next *int) {
	w.t.Helper()
	var msgs []string
	for i := 0; i < n; i++ {
		msgs = append(msgs, announcementMsg("merging", "#"+strconv.Itoa(*next), branch, t0, 30*time.Minute))
		*next++
	}
	w.pushMany(branch, msgs)
}

// items is each announcement of v as "branch item".
func items(v StatusView) map[string]bool {
	out := map[string]bool{}
	for _, a := range v.Announcements {
		out[a.Branch+" "+a.Item] = true
	}
	return out
}

// TestBranchesLeftUnreadAreReadInTheOrderTheyWereLeft: before each intake a
// branch first by name gains as many new commits as one intake takes in, and
// a new branch holding as many again appears between it and zed, whose one
// announcement waits. Every branch left unread is walked before any branch
// read since, the one left first first, so zed's announcement is read by the
// third intake, however many new names keep appearing ahead of it.
func TestBranchesLeftUnreadAreReadInTheOrderTheyWereLeft(t *testing.T) {
	withIntakeCap(t, 4)
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	w.pushStatus("zed", announcementMsg("merging", "#9999", "zed", t0, 30*time.Minute), "")
	next := 1
	for i := 1; i <= 8; i++ {
		w.refill("aaa", maxIntakeCommits, &next)
		w.refill(fmt.Sprintf("n%03d", i), maxIntakeCommits, &next)
		pat.fetchOnly()
		if items(pat.read())["zed #9999"] {
			if i > 3 {
				t.Fatalf("zed's announcement was read only by intake %d; want it read by the third", i)
			}
			return
		}
	}
	t.Fatal("zed's announcement was not read in 8 intakes: the branches appearing ahead of it kept it unread")
}

// TestABranchsOldestUnreadCommitsAreRead: a branch gains as many new commits
// before every intake as one intake takes in. Once the cap has cut it short,
// it takes in half its share from its newest unread commits and half from its
// oldest, and its read mark moves up past every commit read beneath, so its
// first announcement is read by the second intake, its walk never runs past
// the walk's limit, and once it stops gaining commits every one is read and
// it is marked read at its tip.
func TestABranchsOldestUnreadCommitsAreRead(t *testing.T) {
	withIntakeCap(t, 4)
	old := statusWalkLimit
	statusWalkLimit = 12
	t.Cleanup(func() { statusWalkLimit = old })
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	next := 1
	w.refill("aaa", 6, &next)
	firstAt := 0
	for i := 1; i <= 8; i++ {
		pat.fetchOnly()
		v := pat.read()
		if len(v.Doubts) != 0 {
			t.Fatalf("intake %d: doubts %+v; want none: the read mark moves up past what is read", i, v.Doubts)
		}
		if firstAt == 0 && items(v)["aaa #1"] {
			firstAt = i
		}
		w.refill("aaa", maxIntakeCommits, &next)
	}
	if firstAt == 0 || firstAt > 2 {
		t.Fatalf("aaa's first announcement was read by intake %d (0: never); want it read by the second", firstAt)
	}
	pat.fetchOnly()
	for i := 0; i < 20; i++ {
		if pat.read().BranchesLeft == 0 {
			break
		}
	}
	got := items(pat.read())
	for n := 1; n < next; n++ {
		if !got["aaa #"+strconv.Itoa(n)] {
			t.Fatalf("aaa #%d was never read once aaa stopped gaining commits", n)
		}
	}
	if mark, tip := statusBranches.lastRead(pat.dir, StatusTrackingPrefix+"aaa"), w.statusTip("aaa"); mark != tip {
		t.Fatalf("aaa is read at %q; want its tip %s", mark, tip)
	}
}

// TestABranchWithNothingNewIsNotCountedLeft: once the cap is spent, a branch
// whose tip is the commit it was last read at is passed over without a walk
// and not counted among the branches left.
func TestABranchWithNothingNewIsNotCountedLeft(t *testing.T) {
	withIntakeCap(t, 2)
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	next := 1
	w.refill("bob", 1, &next)
	pat.fetchOnly()
	pat.read()
	w.refill("amy", maxIntakeCommits, &next)
	pat.fetchOnly()
	if v := pat.read(); v.BranchesLeft != 0 {
		t.Fatalf("%d branches counted left; want none: bob holds nothing new", v.BranchesLeft)
	}
}

// TestASplitShareOfOneTakesTheOldest: a branch an intake left unread, given
// a share of one commit by the next, takes its oldest unread commit, and its
// read mark moves up to it, so every walk of a branch left moves its mark.
func TestASplitShareOfOneTakesTheOldest(t *testing.T) {
	withIntakeCap(t, 1)
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	next := 1
	w.refill("aaa", 3, &next)
	oldest := strings.Fields(runGit(t, w.bare, "rev-list", "--reverse", StatusRefPrefix+"aaa"))[0]
	ref := StatusTrackingPrefix + "aaa"
	pat.fetchOnly()
	pat.read()
	if mark := statusBranches.lastRead(pat.dir, ref); mark != "" {
		t.Fatalf("the first intake read aaa at %s; want it left unmarked, its oldest commits unread", mark)
	}
	pat.read()
	if _, ok := keptAnnouncement(pat.dir, oldest); !ok {
		t.Fatal("the second intake did not take in aaa's oldest unread commit")
	}
	if mark := statusBranches.lastRead(pat.dir, ref); mark != oldest {
		t.Fatalf("aaa is read at %q after the second intake; want its oldest commit %s", mark, oldest)
	}
}
