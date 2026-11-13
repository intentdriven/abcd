package loop

// remoteid_lock_test.go holds ruling 3 of fix round 5 of
// iss-2611131308266006: a drive's let-go compares before it takes the drain
// lock, and each push of its release, inside the lock, asks a comparison of
// its own; another session waiting on the lock meanwhile is told another
// drain is moving.

import (
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestAComparisonInsideTheDrainLockTellsAWaitingSessionAnotherDrainMoves: a
// drive ends with a judgement's hold to release; its let-go compares, finds
// nothing changed, takes the drain lock and starts the release, whose first
// git asks the comparison again, inside the lock. While that comparison
// runs, another session's drain move waits on the lock for its three
// seconds and is told another drain is moving in this checkout and to back
// off and retry; the release then goes on and the hold is released. Watched
// failing with runHookedContext's ask taken out: no comparison ran inside
// the lock, and the other session was never held.
func TestAComparisonInsideTheDrainLockTellsAWaitingSessionAnotherDrainMoves(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	_, taken := askFirstOnce(t, repo, bare, DrainOptions{})
	id, err := readRemoteIdentity(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	type told struct{ err error }
	waited := make(chan told, 1)
	inside := false
	unset, err := gitutil.SetBeforeHookedGit(func([]string) error {
		if !inside {
			inside = true
			go func() {
				_, err := Drain(repo.Root(), Options{}, DrainOptions{})
				waited <- told{err}
			}()
			select {
			case w := <-waited:
				waited <- w
			case <-time.After(10 * time.Minute):
			}
		}
		return heldBeforeGit(repo.Root(), id)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer unset()
	tm := &terminal{repoRoot: repo.Root()}
	tm.o.Options.remote = id
	tm.letGo()
	if !inside {
		t.Fatal("the release's git asked a comparison inside the drain lock")
	}
	w := <-waited
	r, ok := AsRefusal(w.err)
	if !ok || !r.Contention || r.Reason != "another drain is moving in this checkout" || r.Remedy != "back off and retry" {
		t.Fatalf("the waiting session is told another drain is moving and to back off: %v", w.err)
	}
	if refOf(t, bare, secondIssue) == taken {
		t.Fatal("the release went on once the comparison finished")
	}
}
