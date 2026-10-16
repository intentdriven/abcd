package claim

import (
	"strings"
	"testing"
	"time"
)

// How the forge's record of pushes places each commit of a status branch
// (itd-2609151838312703 C17): every first-parent commit a push carried is
// judged by its pusher, a commit two pushes cover takes the one that is not
// the owner, and a push whose extent the walk cannot read vouches for no more
// than it can prove.

// octoBranch is a world where octo sent n announcements, one push each, and
// pat, who reads them; it returns the commits, oldest first.
func octoBranch(t *testing.T, n int) (*world, *participant, []string) {
	t.Helper()
	w := newWorld(t)
	octo := w.clone("6+octo@users.noreply.github.com", "")
	pat := w.clone("9+pat@users.noreply.github.com", "")
	var cs []string
	for i := 1; i <= n; i++ {
		cs = append(cs, octo.announceAs("merging", "#"+strings.Repeat("1", i)).Commit)
	}
	return w, pat, cs
}

// checksAfter is how pat judges each announcement once the forge answered
// pushes for abcd-status/octo: "verified", "claimed", or "doubtful <login>".
func checksAfter(pat *participant, pushes []Push) map[string]string {
	pat.status(func(_, branch string) ([]Push, error) {
		if branch == "octo" {
			return pushes, nil
		}
		return nil, nil
	})
	out := map[string]string{}
	for _, a := range pat.read().Announcements {
		out[a.Commit] = strings.TrimSpace(a.Check + " " + a.Pusher)
	}
	return out
}

// wantChecks fails the test unless got judges each commit as want does.
func wantChecks(t *testing.T, got map[string]string, want map[string]string) {
	t.Helper()
	for c, w := range want {
		if got[c] != w {
			t.Fatalf("checks = %v; want %s %q", got, c, w)
		}
	}
}

// TestAFirstPushByAnotherAccountIsDoubtful: a branch named for octo that
// another account made, with its first push (Before the zero name), reads
// doubtful in every commit that push carried, naming that account.
func TestAFirstPushByAnotherAccountIsDoubtful(t *testing.T) {
	_, pat, cs := octoBranch(t, 2)
	got := checksAfter(pat, []Push{{Before: zeroCommit, After: cs[1], Pusher: "mallory"}})
	wantChecks(t, got, map[string]string{cs[0]: "doubtful mallory", cs[1]: "doubtful mallory"})
}

// TestAForcedPushIsJudgedByWhatItCanProve: a forced push names, as Before, a
// tip that is not on the line it left, so where its commits begin cannot be
// read. Another account's forced push makes every commit the walk reads
// doubtful, since any of them may be its; the owner's vouches for its tip
// alone, and the commits beneath stay claimed.
func TestAForcedPushIsJudgedByWhatItCanProve(t *testing.T) {
	_, pat, cs := octoBranch(t, 3)
	gone := strings.Repeat("e", 40)
	got := checksAfter(pat, []Push{{Before: gone, After: cs[2], Pusher: "mallory"}})
	wantChecks(t, got, map[string]string{cs[0]: "doubtful mallory", cs[1]: "doubtful mallory", cs[2]: "doubtful mallory"})

	_, pat, cs = octoBranch(t, 3)
	got = checksAfter(pat, []Push{{Before: gone, After: cs[2], Pusher: "octo"}})
	wantChecks(t, got, map[string]string{cs[0]: "claimed", cs[1]: "claimed", cs[2]: "verified"})
}

// TestAPushByAnotherAccountOutweighsTheOwnersOverACommit: two pushes that
// both carried a commit (a branch made twice) judge it by the one that is not
// the owner, whichever the record lists first; a commit only the owner's
// carried is verified.
func TestAPushByAnotherAccountOutweighsTheOwnersOverACommit(t *testing.T) {
	_, pat, cs := octoBranch(t, 3)
	got := checksAfter(pat, []Push{
		{Before: zeroCommit, After: cs[2], Pusher: "octo"},
		{Before: zeroCommit, After: cs[1], Pusher: "mallory"},
	})
	wantChecks(t, got, map[string]string{cs[0]: "doubtful mallory", cs[1]: "doubtful mallory", cs[2]: "verified"})
}

// TestAPushPastTheWalkLimitIsJudgedAsFarAsTheWalkReads: a first push that
// carried more commits than one walk reads judges the newest the walk read,
// and the rest stay claimed. All three were read before the limit fell.
func TestAPushPastTheWalkLimitIsJudgedAsFarAsTheWalkReads(t *testing.T) {
	_, pat, cs := octoBranch(t, 3)
	pat.status(nil)
	old := statusWalkLimit
	statusWalkLimit = 2
	t.Cleanup(func() { statusWalkLimit = old })
	got := checksAfter(pat, []Push{{Before: zeroCommit, After: cs[2], Pusher: "octo"}})
	wantChecks(t, got, map[string]string{cs[0]: "claimed", cs[1]: "verified", cs[2]: "verified"})
}

// TestAnAppsPushLeavesItsCommitClaimed: the adapter passes over a push an app
// made, so the record holds the owner's pushes on either side of it; the
// commit the app's push carried is covered by neither and stays claimed.
func TestAnAppsPushLeavesItsCommitClaimed(t *testing.T) {
	_, pat, cs := octoBranch(t, 3)
	got := checksAfter(pat, []Push{
		{Before: cs[1], After: cs[2], Pusher: "octo"},
		// the app's push, cs[0] to cs[1], is not in the record
		{Before: zeroCommit, After: cs[0], Pusher: "octo"},
	})
	wantChecks(t, got, map[string]string{cs[0]: "verified", cs[1]: "claimed", cs[2]: "verified"})
}

// TestAPlacingCutShortKeepsNothing: the forge's record is placed newest push
// first, and a commit an older push also carried takes that push's pusher
// when it is not the owner, so a placing the budget ends partway would leave
// such a commit read as the owner's. Nothing of it is kept: the commits stay
// claimed, the check says so, and the next fetch asks again.
func TestAPlacingCutShortKeepsNothing(t *testing.T) {
	_, pat, cs := octoBranch(t, 2)
	old := pusherBudget
	pusherBudget = 100 * time.Millisecond
	placeHook = func(i int) {
		if i == 1 {
			time.Sleep(2 * pusherBudget)
		}
	}
	t.Cleanup(func() { pusherBudget, placeHook = old, nil })
	asked := 0
	lookup := func(_, branch string) ([]Push, error) {
		if branch != "octo" {
			return nil, nil
		}
		asked++
		return []Push{
			{Before: zeroCommit, After: cs[1], Pusher: "octo"},
			{Before: zeroCommit, After: cs[0], Pusher: "mallory"},
		}, nil
	}
	res := pat.status(lookup)
	if !strings.Contains(res.SendersUnchecked, "abcd-status/octo: ") {
		t.Fatalf("unchecked = %q; want octo's placing named", res.SendersUnchecked)
	}
	for _, a := range pat.read().Announcements {
		if a.Check != SenderClaimed {
			t.Fatalf("%s reads %s after a placing cut short; want claimed", a.Commit, a.Check)
		}
	}
	placeHook = nil
	pat.status(lookup)
	if asked != 2 {
		t.Fatalf("asked %d times; want the cut-short branch asked again", asked)
	}
}
