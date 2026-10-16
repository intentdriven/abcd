package claim

import (
	"context"
	"errors"
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
	pat.status(func(_ context.Context, _, branch string) ([]Push, error) {
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
	lookup := func(_ context.Context, _, branch string) ([]Push, error) {
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

// TestTheForgeIsAskedOfABoundedNumberOfBranchesNewestFirst: one fetch asks
// the forge about at most maxAskedBranches branches, the newest first, and
// says how many it left for the next; a commit the forge answered for is not
// asked about again, so the next fetch asks only the rest.
func TestTheForgeIsAskedOfABoundedNumberOfBranchesNewestFirst(t *testing.T) {
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	var order []string
	for i := 0; i < maxAskedBranches+2; i++ {
		b := "u" + strings.Repeat("x", i)
		w.pushStatus(b, announcementMsg("merging", "#1", b, t0.Add(time.Duration(i)*time.Second), 30*time.Minute), "")
		order = append([]string{b}, order...)
	}
	var asked []string
	lookup := func(_ context.Context, _, branch string) ([]Push, error) {
		asked = append(asked, branch)
		return []Push{{Before: zeroCommit, After: w.statusTip(branch), Pusher: branch}}, nil
	}
	res := pat.status(lookup)
	if strings.Join(asked, " ") != strings.Join(order[:maxAskedBranches], " ") {
		t.Fatalf("asked %v; want the newest %d, newest first: %v", asked, maxAskedBranches, order[:maxAskedBranches])
	}
	if !strings.Contains(res.SendersUnchecked, "2 status branches not asked about") {
		t.Fatalf("unchecked = %q; want the two left named by count", res.SendersUnchecked)
	}
	asked = nil
	if res := pat.status(lookup); res.SendersUnchecked != "" {
		t.Fatalf("the second fetch's unchecked = %q; want none", res.SendersUnchecked)
	}
	if strings.Join(asked, " ") != strings.Join(order[maxAskedBranches:], " ") {
		t.Fatalf("the second fetch asked %v; want only the two left: %v", asked, order[maxAskedBranches:])
	}
	asked = nil
	pat.status(lookup)
	if len(asked) != 0 {
		t.Fatalf("a third fetch asked %v; want nothing asked again", asked)
	}
}

// TestTheForgeIsNotAskedPastItsBudget: a forge that answers slowly uses up
// the budget, and no further branch is asked about; the rest are counted.
func TestTheForgeIsNotAskedPastItsBudget(t *testing.T) {
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	w.pushStatus("aa", announcementMsg("merging", "#1", "aa", t0, 30*time.Minute), "")
	w.pushStatus("bb", announcementMsg("merging", "#2", "bb", t0.Add(time.Second), 30*time.Minute), "")
	old := pusherBudget
	pusherBudget = 100 * time.Millisecond
	t.Cleanup(func() { pusherBudget = old })
	var asked []string
	res := pat.status(func(_ context.Context, _, branch string) ([]Push, error) {
		asked = append(asked, branch)
		time.Sleep(2 * pusherBudget)
		return []Push{}, nil
	})
	if len(asked) != 1 || asked[0] != "bb" {
		t.Fatalf("asked %v; want only the newest before the budget ran out", asked)
	}
	if !strings.Contains(res.SendersUnchecked, "1 status branch not asked about") {
		t.Fatalf("unchecked = %q; want the one left counted", res.SendersUnchecked)
	}
}

// TestAPartialRecordKeepsOnlyAnotherAccountNamed: a record not read in full
// (ErrPartialRecord) keeps another account it names as a pusher, which no
// older push can make less doubtful, and keeps nothing for a commit only the
// owner's pushes cover or no push covers, so each is asked about again; the
// fetch says the record was not read in full.
func TestAPartialRecordKeepsOnlyAnotherAccountNamed(t *testing.T) {
	_, pat, cs := octoBranch(t, 3)
	asked := 0
	lookup := func(_ context.Context, _, branch string) ([]Push, error) {
		if branch != "octo" {
			return nil, nil
		}
		asked++
		return []Push{
			{Before: cs[1], After: cs[2], Pusher: "mallory"},
			{Before: cs[0], After: cs[1], Pusher: "octo"},
		}, ErrPartialRecord
	}
	res := pat.status(lookup)
	if !strings.Contains(res.SendersUnchecked, "abcd-status/octo: the forge's record was not read in full") {
		t.Fatalf("unchecked = %q; want the record said not read in full", res.SendersUnchecked)
	}
	for i, want := range []pusherVerdict{{}, {}, {asked: true, known: true, login: "mallory"}} {
		if got := pusherKept(pat.dir, cs[i]); got != want {
			t.Fatalf("commit %d keeps %+v; want %+v", i, got, want)
		}
	}
	pat.status(lookup)
	if asked != 2 {
		t.Fatalf("the forge was asked %d times; want the next fetch to ask again", asked)
	}
}

// TestEachForgeLookupEndsAtTheBudget: every lookup is asked under what is
// left of pusherBudget as its deadline, so a forge that does not answer holds
// the fetch no longer than the budget, and the branch it left is counted.
func TestEachForgeLookupEndsAtTheBudget(t *testing.T) {
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	w.pushStatus("aa", announcementMsg("merging", "#1", "aa", t0, 30*time.Minute), "")
	w.pushStatus("bb", announcementMsg("merging", "#2", "bb", t0.Add(time.Second), 30*time.Minute), "")
	pat.status(nil)
	old := pusherBudget
	pusherBudget = 300 * time.Millisecond
	t.Cleanup(func() { pusherBudget = old })
	var asked []string
	start := time.Now()
	res := pat.status(func(ctx context.Context, _, branch string) ([]Push, error) {
		asked = append(asked, branch)
		if d, ok := ctx.Deadline(); !ok || d.After(start.Add(2*pusherBudget)) {
			t.Errorf("the lookup for %s was asked with deadline %v (set %v); want the budget's", branch, d, ok)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(20 * time.Second):
			return []Push{}, nil
		}
	})
	if took := time.Since(start); took > 5*time.Second {
		t.Fatalf("a forge that never answered held the fetch %s against a %s budget", took, pusherBudget)
	}
	if len(asked) != 1 || asked[0] != "bb" {
		t.Fatalf("asked %v; want only the newest before the budget ran out", asked)
	}
	if !strings.Contains(res.SendersUnchecked, "1 status branch not asked about") {
		t.Fatalf("unchecked = %q; want the one left counted", res.SendersUnchecked)
	}
}

// TestARecordShortOfTheTipIsAskedAgain: the forge records a push some time
// after it lands, so a record whose page is not full but names no push to
// the tip held here is one the forge has not caught up on yet. It is read as
// a record not read in full: no commit it leaves open is kept as asked, not
// even as covered by no push, the fetch says so, and the next fetch asks
// again.
func TestARecordShortOfTheTipIsAskedAgain(t *testing.T) {
	_, pat, cs := octoBranch(t, 2)
	asked := 0
	lookup := func(_ context.Context, _, branch string) ([]Push, error) {
		if branch != "octo" {
			return nil, nil
		}
		asked++
		return []Push{{Before: zeroCommit, After: cs[0], Pusher: "octo"}}, nil
	}
	res := pat.status(lookup)
	if want := "abcd-status/octo: the forge's record does not yet name the push of the tip held here"; !strings.Contains(res.SendersUnchecked, want) {
		t.Fatalf("unchecked = %q; want %q", res.SendersUnchecked, want)
	}
	for i, c := range cs {
		if got := pusherKept(pat.dir, c); got != (pusherVerdict{}) {
			t.Fatalf("commit %d keeps %+v; want nothing kept", i, got)
		}
	}
	pat.status(lookup)
	if asked != 2 {
		t.Fatalf("the forge was asked %d times; want the next fetch to ask again", asked)
	}
}

// TestATipAnAppPushedReachesTheRecord: the forge's record names an app's
// push as the newest to the tip held here. The record has reached the tip, so
// the fetch does not say it has not, and the commit the app's push carried
// stays claimed, never judged by the app's login. A record read in full keeps
// a verdict for every commit asked about, so the next fetch asks nothing; one
// not read in full is asked about again, and the fetch says so.
func TestATipAnAppPushedReachesTheRecord(t *testing.T) {
	for _, short := range []bool{false, true} {
		w := newWorld(t)
		octo := w.clone("6+octo@users.noreply.github.com", "")
		pat := w.clone("9+pat@users.noreply.github.com", "")
		first := octo.announceAs("merging", "#1").Commit
		octo.announceAs("merging", "#2")
		asked := 0
		lookup := func(_ context.Context, _, branch string) ([]Push, error) {
			if branch != "octo" {
				return nil, nil
			}
			asked++
			tip := w.statusTip("octo")
			pushes := []Push{
				{Before: first, After: tip, Pusher: "hosted-app[bot]", App: true},
				{Before: zeroCommit, After: first, Pusher: "octo"},
			}
			if short {
				return pushes, ErrPartialRecord
			}
			return pushes, nil
		}
		res := pat.status(lookup)
		if strings.Contains(res.SendersUnchecked, "does not yet name") {
			t.Fatalf("short %v: unchecked = %q; want no word that the record has not reached the tip", short, res.SendersUnchecked)
		}
		if got := strings.Contains(res.SendersUnchecked, ErrPartialRecord.Error()); got != short {
			t.Fatalf("short %v: unchecked = %q; want the record not read in full named only when it was not", short, res.SendersUnchecked)
		}
		for _, a := range pat.read().Announcements {
			if a.Commit != first && (a.Check != SenderClaimed || a.Pusher != "") {
				t.Fatalf("short %v: the app's commit reads %s %q; want it claimed", short, a.Check, a.Pusher)
			}
		}
		pat.status(lookup)
		want := 1
		if short {
			want = 2
		}
		if asked != want {
			t.Fatalf("short %v: the forge was asked %d times over two fetches; want %d", short, asked, want)
		}
	}
}

// TestAnAnnouncementReadAfterAnAppsTipIsAskedAbout: bob posts A, then more
// announcements than one intake takes in, the last pushed by an app. The
// first intake stops short of A, and the fetch asks the forge about bob's
// branch, whose tip the app pushed. The next intake reads A, and a later
// fetch at the same tip asks about it: the forge's record judges it.
func TestAnAnnouncementReadAfterAnAppsTipIsAskedAbout(t *testing.T) {
	withIntakeCap(t, 3)
	w := newWorld(t)
	bob := w.clone("7+bob@users.noreply.github.com", "")
	pat := w.clone("9+pat@users.noreply.github.com", "")
	a := bob.announceAs("merging", "#1").Commit
	var cs []string
	for _, item := range []string{"#2", "#3", "#4"} {
		cs = append(cs, bob.announceAs("merging", item).Commit)
	}
	asked := 0
	lookup := func(_ context.Context, _, branch string) ([]Push, error) {
		if branch != "bob" {
			return nil, nil
		}
		asked++
		return []Push{
			{Before: cs[1], After: cs[2], Pusher: "hosted-app[bot]", App: true},
			{Before: zeroCommit, After: cs[1], Pusher: "bob"},
		}, nil
	}
	pat.status(lookup)
	if asked != 1 {
		t.Fatalf("the forge was asked %d times at the first fetch; want once", asked)
	}
	if _, ok := keptAnnouncement(pat.dir, a); ok {
		t.Fatal("the first intake read A; want it left past the cap")
	}
	pat.read()
	if _, ok := keptAnnouncement(pat.dir, a); !ok {
		t.Fatal("the second intake did not read A")
	}
	res := pat.status(lookup)
	found := false
	for _, an := range pat.read().Announcements {
		if an.Commit != a {
			continue
		}
		found = true
		if an.Check != SenderVerified && !strings.Contains(res.SendersUnchecked, "not asked about") {
			t.Fatalf("A reads %s, the forge was asked %d times, and unchecked = %q; want A asked about or counted unchecked",
				an.Check, asked, res.SendersUnchecked)
		}
	}
	if !found {
		t.Fatal("A is not among the announcements read")
	}
}

// TestTheForgeIsAskedAboutEveryBranchInTurn: more branches wait to be asked
// about than one fetch asks, and the forge answers none of them in full, so
// none is ever settled. Each fetch asks first the branches waiting longest,
// and each branch asked goes behind every branch still waiting, so every
// branch is asked by the second fetch.
func TestTheForgeIsAskedAboutEveryBranchInTurn(t *testing.T) {
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	var all []string
	for i := 0; i < maxAskedBranches+2; i++ {
		b := "u" + strings.Repeat("x", i)
		w.pushStatus(b, announcementMsg("merging", "#1", b, t0.Add(time.Duration(i)*time.Second), 30*time.Minute), "")
		all = append(all, b)
	}
	asked := map[string]int{}
	lookup := func(_ context.Context, _, branch string) ([]Push, error) {
		asked[branch]++
		return nil, errors.New("the forge did not answer")
	}
	pat.status(lookup)
	pat.status(lookup)
	for _, b := range all {
		if asked[b] == 0 {
			t.Fatalf("after two fetches %s was never asked about (asked: %v); want every branch asked in turn", b, asked)
		}
	}
}

// TestAnUnreadTipDoesNotReadAsARecordBehind: when the status branches held
// here cannot be read, no tip is known to hold the record against. The fetch
// does not say of every branch that the record has not reached its tip; it
// says the branches could not be read, and keeps nothing the record could
// still change, so the next fetch asks again.
func TestAnUnreadTipDoesNotReadAsARecordBehind(t *testing.T) {
	_, pat, cs := octoBranch(t, 2)
	pat.status(nil)
	old := senderRefs
	senderRefs = func(context.Context, string) ([]statusRef, error) {
		return nil, errors.New("reading the status branches: git failed")
	}
	t.Cleanup(func() { senderRefs = old })
	asked := 0
	lookup := func(_ context.Context, _, branch string) ([]Push, error) {
		if branch != "octo" {
			return nil, nil
		}
		asked++
		return []Push{{Before: zeroCommit, After: cs[1], Pusher: "octo"}}, nil
	}
	res := pat.status(lookup)
	if strings.Contains(res.SendersUnchecked, "does not yet name") {
		t.Fatalf("unchecked = %q; want no word that the record has not reached the tip", res.SendersUnchecked)
	}
	if !strings.Contains(res.SendersUnchecked, "the status branches held here could not be read") {
		t.Fatalf("unchecked = %q; want the failed read of the branches named", res.SendersUnchecked)
	}
	for i, c := range cs {
		if got := pusherKept(pat.dir, c); got != (pusherVerdict{}) {
			t.Fatalf("commit %d keeps %+v; want nothing kept while the tip is unknown", i, got)
		}
	}
	pat.status(lookup)
	if asked != 2 {
		t.Fatalf("the forge was asked %d times; want the next fetch to ask again", asked)
	}
}
