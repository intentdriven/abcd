package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/implement"
)

// sharedRun opens the shared run state for repo under the test's HOME and joins
// session to it.
func sharedRun(t *testing.T, root, sha, session string) *implement.Run {
	t.Helper()
	run, err := implement.Open(strings.TrimSpace(sha))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := run.Join(session, implement.RoleFirst, "", "", 0); err != nil {
		t.Fatal(err)
	}
	return run
}

// TestAStartForASessionClaimsTheIntentWhereAnotherCheckoutSeesIt: a build
// started for a joined session claims its intent in the shared run state, so a
// second build of the same intent from another checkout of the repository —
// one whose lane has neither moved nor claimed anything yet — is refused as
// contention naming the session, not started as a duplicate
// (iss-2609252050506863).
func TestAStartForASessionClaimsTheIntentWhereAnotherCheckoutSeesIt(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	sha := repo.Git("rev-list", "--max-parents=0", "HEAD")
	run := sharedRun(t, repo.Root(), sha, "host-a")

	first, err := Start(repo.Root(), "itd-10", Options{Session: "host-a"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Claim == nil || first.Claim.Claim.Session != "host-a" || first.Claim.Claim.Lane != first.RunID {
		t.Fatalf("the start reports no claim for its session: %+v", first.Claim)
	}
	claims, err := run.Claims()
	if err != nil || len(claims) != 1 || claims[0].Record != "itd-10" || claims[0].Session != "host-a" || !claims[0].Live {
		t.Fatalf("shared claims = %+v, %v; want host-a's live claim on itd-10", claims, err)
	}

	// Another checkout of the same repository: same root commit, its own tier.
	other := filepath.Join(t.TempDir(), "second")
	repo.Git("worktree", "add", "-q", "-b", "second", other)
	if err := os.MkdirAll(filepath.Join(other, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err = Start(other, "itd-10", Options{})
	r := mustRefusal(t, err)
	if r.Check != CheckPeers || !r.Contention || !strings.Contains(r.Reason, "host-a") {
		t.Fatalf("a second build from another checkout = %+v; want the peers check naming host-a", r)
	}
	runTierAbsent(t, other)

	// The session's own start again resumes; it is not its own peer.
	again, err := Start(repo.Root(), "itd-10", Options{Session: "host-a"})
	if err != nil || !again.Resumed || again.RunID != first.RunID {
		t.Fatalf("resume = %+v, %v", again, err)
	}
}

// TestAStartForASessionIsNotRefusedByItsOwnClaim: a session that claimed the
// intent itself (`implement claim`) before building it is not a peer of its own
// build; the claim is renewed for the run.
func TestAStartForASessionIsNotRefusedByItsOwnClaim(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	sha := repo.Git("rev-list", "--max-parents=0", "HEAD")
	run := sharedRun(t, repo.Root(), sha, "host-a")
	if _, err := run.Claim(implement.ClaimRequest{Session: "host-a", Record: "itd-10", Lane: "mine"}); err != nil {
		t.Fatal(err)
	}
	res, err := Start(repo.Root(), "itd-10", Options{Session: "host-a"})
	if err != nil {
		t.Fatalf("a session's own claim refused its build: %v", err)
	}
	if res.Claim == nil || !res.Claim.Renewed {
		t.Fatalf("claim = %+v; want the session's claim renewed for the run", res.Claim)
	}
	// Without the session named, the same claim is a peer's.
	repo2 := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	sha2 := repo2.Git("rev-list", "--max-parents=0", "HEAD")
	run2 := sharedRun(t, repo2.Root(), sha2, "host-b")
	if _, err := run2.Claim(implement.ClaimRequest{Session: "host-b", Record: "itd-10", Lane: "mine"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Start(repo2.Root(), "itd-10", Options{}); mustRefusal(t, err).Check != CheckPeers {
		t.Fatalf("an unnamed start past a live claim: %v", err)
	}
}

// TestAStartForASessionThatHasNotJoinedWritesNothing: the claim is the
// session's, so a session the shared run does not hold is refused before the
// run is created.
func TestAStartForASessionThatHasNotJoinedWritesNothing(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	sha := repo.Git("rev-list", "--max-parents=0", "HEAD")
	sharedRun(t, repo.Root(), sha, "host-a")
	_, err := Start(repo.Root(), "itd-10", Options{Session: "ghost"})
	if r := mustRefusal(t, err); r.Stage != StageClaim || r.Contention || !strings.Contains(r.Reason, "ghost") {
		t.Fatalf("a start for an unjoined session = %+v; want the claim step refused naming it", r)
	}
	runTierAbsent(t, repo.Root())
	if _, err := Start(repo.Root(), "itd-10", Options{Session: "../x"}); err == nil {
		t.Fatal("a start for a session that is not a name succeeded")
	}
	runTierAbsent(t, repo.Root())
}
