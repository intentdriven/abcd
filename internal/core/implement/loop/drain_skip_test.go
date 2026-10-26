package loop

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/capture"
)

// The drain's skip list (spc-2609212015054359 step 4; iss-2610090642385898):
// an issue, or a path, held back for now with its reason, passed over in the
// move's passed entries, and cleared when the drain ends.

func drainStatePresent(t *testing.T, root string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(DrainStateRel)))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	return err == nil
}

// TestAnIssueOnTheSkipListIsPassedOverWithItsReason: the documentation issue,
// which the order takes first, is held back; the move names it in its passed
// entries with the reason and asks the judgement over the next issue, and a
// later move of the same drain still passes it over.
func TestAnIssueOnTheSkipListIsPassedOverWithItsReason(t *testing.T) {
	repo := drainRepo(t)
	res, err := Drain(repo.Root(), Options{}, DrainOptions{Skip: []string{secondIssue}, SkipReason: "a peer's fix is in flight"})
	if err != nil {
		t.Fatal(err)
	}
	p := passedFor(t, res, secondIssue)
	if p.Check != CheckSkipped || !strings.Contains(p.Reason, "a peer's fix is in flight") || !strings.Contains(p.Reason, "--unskip "+secondIssue) {
		t.Fatalf("the held-back issue is passed over with its reason: %+v", p)
	}
	if res.Judging == nil || res.Judging.Issue != eligibleIssue {
		t.Fatalf("the move asks over the next issue in the order instead: %+v", res.Judging)
	}
	if len(res.Skips) != 1 || res.Skips[0].Target != secondIssue || !strings.Contains(res.Next, "held back for now") {
		t.Fatalf("the summary carries the skip list and says what this move held back: %+v / %q", res.Skips, res.Next)
	}
	if res.Counts.Passed < 1 {
		t.Fatalf("the pass-over is counted: %+v", res.Counts)
	}
	again, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if p := passedFor(t, again, secondIssue); p.Check != CheckSkipped {
		t.Fatalf("the skip list holds for the drain's later moves: %+v", p)
	}
}

// TestTheSkipListClearsWhenTheDrainEnds: holding back every eligible issue
// leaves nothing to take, so the drain ends and its skip list clears; the next
// drain takes the documentation issue first again.
func TestTheSkipListClearsWhenTheDrainEnds(t *testing.T) {
	repo := drainRepo(t)
	res, err := Drain(repo.Root(), Options{}, DrainOptions{Skip: []string{secondIssue, eligibleIssue}, SkipReason: "held for the cut"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Stopped != DrainStoppedEmpty || len(res.Skips) != 0 || !strings.Contains(res.Next, "cleared with the drain's end") {
		t.Fatalf("a drain left with only held-back issues ends and clears its list: %+v", res)
	}
	passedFor(t, res, secondIssue)
	passedFor(t, res, eligibleIssue)
	st, _, err := readDrain(repo.Root())
	if err != nil || len(st.Skips) != 0 {
		t.Fatalf("the ended drain's state holds no skip list: %+v %v", st.Skips, err)
	}
	next, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !next.Started || next.Judging == nil || next.Judging.Issue != secondIssue {
		t.Fatalf("the next drain holds nothing back: %+v", next)
	}
}

// TestAPathOnTheSkipListHoldsBackEveryIssueInIt: a path holds back an issue
// whose found_at lies in it, naming the path, and leaves an issue elsewhere
// to be taken.
func TestAPathOnTheSkipListHoldsBackEveryIssueInIt(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	issueRepoRule(t, repo)
	repo.Write("internal/core/capture/eligible.go", "package capture\n")
	repo.Write("docs/page.md", "# A page\n")
	repo.Commit("the files the issues are about")
	fileIssueAt(t, repo, eligibleIssue, capture.SeverityMinor, capture.Category("bug"), issueRemedy, "internal/core/capture/eligible.go")
	fileIssueAt(t, repo, secondIssue, capture.SeverityMinor, capture.Category("bug"), "Name the flag in the page.", "docs/page.md")
	repo.Commit("the ledger")
	res, err := Drain(repo.Root(), Options{}, DrainOptions{Skip: []string{"internal/core/"}, SkipReason: "a fix is in flight there"})
	if err != nil {
		t.Fatal(err)
	}
	if p := passedFor(t, res, eligibleIssue); p.Check != CheckSkipped || !strings.Contains(p.Reason, "in internal/core,") {
		t.Fatalf("an issue in the held-back path is passed over naming it: %+v", p)
	}
	if res.Judging == nil || res.Judging.Issue != secondIssue {
		t.Fatalf("an issue outside the path is still taken: %+v", res.Judging)
	}
	if len(res.Skips) != 1 || res.Skips[0].Target != "internal/core" {
		t.Fatalf("the path is stored clean, its trailing slash dropped: %+v", res.Skips)
	}
}

// TestUnskipTakesAnIssueOffTheList: --unskip lifts a hold before the drain
// ends, and the move no longer passes the issue over.
func TestUnskipTakesAnIssueOffTheList(t *testing.T) {
	repo := drainRepo(t)
	if _, err := Drain(repo.Root(), Options{}, DrainOptions{Skip: []string{secondIssue}, SkipReason: "for now"}); err != nil {
		t.Fatal(err)
	}
	res, err := Drain(repo.Root(), Options{}, DrainOptions{Unskip: []string{secondIssue}})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range res.Passed {
		if p.ID == secondIssue && p.Check == CheckSkipped {
			t.Fatalf("an unskipped issue is not passed over as held back: %+v", p)
		}
	}
	if len(res.Skips) != 0 || !strings.Contains(res.Next, "no longer held back") {
		t.Fatalf("the list is empty and the move says so: %+v / %q", res.Skips, res.Next)
	}
}

// TestASkipWithoutAReasonOrWithABadTargetIsRefusedWritingNothing.
func TestASkipWithoutAReasonOrWithABadTargetIsRefusedWritingNothing(t *testing.T) {
	repo := drainRepo(t)
	for name, d := range map[string]DrainOptions{
		"no reason":      {Skip: []string{secondIssue}},
		"blank reason":   {Skip: []string{secondIssue}, SkipReason: "  "},
		"reason alone":   {SkipReason: "why"},
		"escaping path":  {Skip: []string{"../outside"}, SkipReason: "why"},
		"absolute path":  {Skip: []string{"/etc"}, SkipReason: "why"},
		"bad unskip":     {Unskip: []string{"a/../b"}},
		"too long a why": {Skip: []string{secondIssue}, SkipReason: strings.Repeat("x", maxSkipReasonBytes+1)},
	} {
		if _, err := Drain(repo.Root(), Options{}, d); err == nil {
			t.Fatalf("%s: the move is refused", name)
		} else if _, ok := AsRefusal(err); !ok {
			t.Fatalf("%s: a refusal, not %v", name, err)
		}
		if drainStatePresent(t, repo.Root()) {
			t.Fatalf("%s: nothing is written", name)
		}
	}
}

// TestADrainStateWithASkipEntryTheDrainNeverWritesIsRefused.
func TestADrainStateWithASkipEntryTheDrainNeverWritesIsRefused(t *testing.T) {
	repo := drainRepo(t)
	if _, err := Drain(repo.Root(), Options{}, DrainOptions{Skip: []string{secondIssue}, SkipReason: "for now"}); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(repo.Root(), filepath.FromSlash(DrainStateRel))
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(strings.Replace(string(data), `"target": "`+secondIssue+`"`, `"target": "../outside"`, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readDrain(repo.Root()); err == nil || !strings.Contains(err.Error(), "skip-list entry") {
		t.Fatalf("a skip entry the drain never writes is refused: %v", err)
	}
}
