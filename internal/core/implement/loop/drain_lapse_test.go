package loop

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/lease"
)

// A judgement request's bound (spc-2610161044461448 step 2): the drain takes
// its judgement hold once per request, so a host that polls without answering
// never renews it; a request whose hold lapses unanswered is dropped and the
// hold released, and the issue is asked again only after the drain's other
// eligible work.

// TestALapsedJudgementRequestIsDroppedAndAskedAgainOnlyAfterOtherWork: polls
// without an answer push nothing, even when the move's view cannot be read;
// once the hold has lapsed the request is dropped, the hold released, and the
// next issue asked instead; once that one is handed back, the lapsed issue is
// asked again on a new hold.
func TestALapsedJudgementRequestIsDroppedAndAskedAgainOnlyAfterOtherWork(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	c := &clock{t: time.Now().UTC().Truncate(time.Second)}
	o := Options{Now: c.now}
	// A window longer than the lease, so the pace never holds the moves.
	pace := "600/0"
	_, taken := askFirst(t, repo, bare, Options{Now: c.now, Pace: &pace}, DrainOptions{})
	pushes := countPushes(t, repo)

	c.t = c.t.Add(30 * time.Minute)
	was := drainView
	drainView = func(string, time.Time) (claim.Snapshot, error) { return claim.Snapshot{}, errors.New("unreadable") }
	poll, err := Drain(repo.Root(), o, DrainOptions{})
	drainView = was
	if err != nil {
		t.Fatal(err)
	}
	if poll.Judging == nil || poll.Judging.Issue != secondIssue || pushes() != 0 {
		t.Fatalf("a poll without an answer still awaits %s and pushes nothing: judging %+v, %d push(es)", secondIssue, poll.Judging, pushes())
	}
	heldStill(t, bare, secondIssue, taken)

	c.t = c.t.Add(lease.Default)
	lapsed, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	releasedOver(t, bare, secondIssue, taken)
	if lapsed.Judging == nil || lapsed.Judging.Issue != eligibleIssue {
		t.Fatalf("once the request lapses the next issue is asked: %+v", lapsed.Judging)
	}
	if !strings.Contains(lapsed.Next, secondIssue+" was not answered before") {
		t.Fatalf("the move says the request lapsed: %q", lapsed.Next)
	}
	if st, _, _ := readDrain(repo.Root()); len(st.Lapsed) != 1 || st.Lapsed[0].Issue != secondIssue {
		t.Fatalf("the drain records the lapsed request: %+v", st.Lapsed)
	}

	again := judge(t, repo, o, lapsed, JudgementYes, HandBackTrustRule, "the remedy moves a trust boundary")
	if again.Judging == nil || again.Judging.Issue != secondIssue || again.Judging.Hold == nil || again.Judging.Hold.Commit == taken {
		t.Fatalf("with the other work taken the lapsed issue is asked again on a new hold: %+v", again.Judging)
	}
}
