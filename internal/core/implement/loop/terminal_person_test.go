//go:build unix

package loop

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// TestHoldingAnIssueBackLeavesAReservationTheDrainNeverTook: the person's own
// claim on it stays; only the drain's own hold is the drain's to release.
func TestHoldingAnIssueBackLeavesAReservationTheDrainNeverTook(t *testing.T) {
	w := newTerminalWorld(t, repeated("noshape", 3)...)
	root := w.repo.Root()
	mine := map[string]string{eligibleIssue: "AGENTS.md", secondIssue: "commands/drain.md", thirdIssue: "commands/other.md"}
	for id, path := range mine {
		if _, err := claim.Reserve(root, claim.Request{Record: id, Footprint: []string{path}}, claim.Options{Now: w.clock.now}); err != nil {
			t.Fatal(err)
		}
	}
	o := w.options(t)
	o.Drain.Max = 1
	w.guardSleeps(t, &o)
	res, err := RunTerminal(context.Background(), root, o)
	if err != nil {
		t.Fatalf("%v\n%s", err, strings.Join(w.lines, "\n"))
	}
	if len(res.HeldBack) == 0 {
		t.Fatalf("an issue is held back: %+v\n%s", res.Ended, strings.Join(w.lines, "\n"))
	}
	if r, ok := claimOf(t, w, res.HeldBack[0]); !ok || r.State != claim.StateLive {
		t.Fatalf("the person's own reservation was released: %+v", r)
	}
}

// TestAPassedOverIssueIsSaidOnceAndNamedAtTheEnd: the person hears why, once, and
// the end names it, under a check of its own.
func TestAPassedOverIssueIsSaidOnceAndNamedAtTheEnd(t *testing.T) {
	w := newTerminalWorld(t)
	o := w.options(t)
	o.Drain.Max = 1
	o.Drain.PassOver = func(_, _ string) string {
		return "it changes a gate file, so it needs a person's own session"
	}
	w.guardSleeps(t, &o)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.PassedOver) < 2 {
		t.Fatalf("the end names each: %+v\n%s", res.PassedOver, strings.Join(w.lines, "\n"))
	}
	for _, p := range res.PassedOver {
		if p.Check != CheckNeedsPerson {
			t.Errorf("a check of its own: %+v", p)
		}
		if n := strings.Count(strings.Join(w.lines, "\n"), p.ID+": it changes a gate file"); n != 1 {
			t.Errorf("%s is said %d times:\n%s", p.ID, n, strings.Join(w.lines, "\n"))
		}
	}
}

// TestTheGateCheckReadsFromTheLastSync: what a default branch merged into the lane
// changed is not the lane's change, and the base stays held as it was.
func TestTheGateCheckReadsFromTheLastSync(t *testing.T) {
	w := newTerminalWorld(t)
	o := oneLane(t, w)
	const merged = "1111111111111111111111111111111111111111"
	for i := range w.stages {
		if w.stages[i].Name == StageValidate {
			run := w.stages[i].Run
			w.stages[i].Run = func(c Context, l *Lane) (Outcome, error) {
				l.Syncs = append(l.Syncs, Sync{At: time.Now().UTC(), Merged: merged, Head: "2222222222222222222222222222222222222222"})
				return run(c, l)
			}
		}
	}
	got := ""
	o.PreflightAllowed = func(_, base string) error {
		got = base
		return nil
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if got != merged {
		t.Fatalf("the check read from %q, want the last sync's %q: %+v\n%s", got, merged, res.Ended, strings.Join(w.lines, "\n"))
	}
}

// TestHeldBackIssuesAreKeptInTheMainCheckout: removing a linked worktree does not
// lose the fact.
func TestHeldBackIssuesAreKeptInTheMainCheckout(t *testing.T) {
	w := newTerminalWorld(t)
	linked := filepath.Join(t.TempDir(), "linked")
	w.repo.Git("worktree", "add", "-q", "-b", "other", linked)
	main, err := filepath.EvalSymlinks(w.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(main, filepath.FromSlash(HeldBackRel)); heldBackFile(linked) != want {
		t.Fatalf("kept at %q, want %q", heldBackFile(linked), want)
	}
}

// TestAWallsReasonReachesTheEnd: what stopped a drive is named at its end.
func TestAWallsReasonReachesTheEnd(t *testing.T) {
	w := newTerminalWorld(t)
	o := oneLane(t, w)
	o.BeforeStage = func() error { return os.ErrInvalid }
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalWall || !strings.Contains(res.Why, os.ErrInvalid.Error()) {
		t.Fatalf("the end names the cause: %+v", res.Why)
	}
}

// TestLettingGoIsFencedOnTheHold: at a drive's end the hold a judgement waited on
// is released while it is still the drain's, and left when it has been renewed.
func TestLettingGoIsFencedOnTheHold(t *testing.T) {
	letGo := func(root string) {
		(&terminal{repoRoot: root}).letGo()
	}
	t.Run("the tip is still the hold", func(t *testing.T) {
		repo, bare := reservingDrainRepo(t, "commands/drain.md")
		_, taken := askFirstOnce(t, repo, bare, DrainOptions{})
		letGo(repo.Root())
		if refOf(t, bare, secondIssue) == taken {
			t.Fatal("the hold was not released")
		}
		snap, err := claim.View(repo.Root(), time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if r, ok := snap.Lookup(secondIssue); ok && r.State == claim.StateLive {
			t.Fatalf("the hold stays live: %+v", r)
		}
	})
	t.Run("the tip moved through a renewal", func(t *testing.T) {
		repo, bare := reservingDrainRepo(t, "commands/drain.md")
		askFirstOnce(t, repo, bare, DrainOptions{})
		if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Footprint: []string{"commands"}}, claim.Options{}); err != nil {
			t.Fatal(err)
		}
		mine := refOf(t, bare, secondIssue)
		letGo(repo.Root())
		heldStill(t, bare, secondIssue, mine)
		snap, err := claim.View(repo.Root(), time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if r, ok := snap.Lookup(secondIssue); !ok || r.State != claim.StateLive {
			t.Fatalf("the renewed reservation does not stay live: %+v", r)
		}
	})
}
