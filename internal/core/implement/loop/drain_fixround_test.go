package loop

import (
	"errors"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// The review round on spc-2610161044461448 steps 1-18: each test here is one
// finding of the two reviews, watched fail before its fix.

// failingEntropy is a minter's entropy that cannot be read: a start that
// mints its run id from it fails after its reservation is made.
type failingEntropy struct{}

func (failingEntropy) Read([]byte) (int, error) { return 0, errors.New("no entropy") }

// trailerOn is the values of key on origin's reservation tip for id.
func trailerOn(t *testing.T, bare, id, key string) []string {
	t.Helper()
	sha := gitIn(t, bare, "for-each-ref", "--format=%(objectname)", claim.RefPrefix+id)
	if sha == "" {
		return nil
	}
	var out []string
	for _, line := range strings.Split(gitIn(t, bare, "log", "-1", "--format=%B", sha), "\n") {
		if v, ok := strings.CutPrefix(line, key+": "); ok {
			out = append(out, v)
		}
	}
	return out
}

// TestAFailedBuildStartOverAHandBackKeepsIt: a person's `abcd build iss-X`
// over a standing hand-back reserves it, as a person's own reservation may;
// when the start then fails, the release its deferred clean-up pushes keeps
// the hand-back, so no drain judges the issue again while its record stands.
func TestAFailedBuildStartOverAHandBackKeepsIt(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Footprint: []string{"commands/drain.md"}, Drain: true}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	if res, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Release: true, HandBack: claim.HandBackDesignFinding}, claim.Options{}); err != nil || res.HandBackUnpublished != "" {
		t.Fatalf("the hand-back = %+v, %v", res, err)
	}
	_, err := Start(repo.Root(), secondIssue, Options{Minter: recordid.Minter{Entropy: failingEntropy{}}})
	if err == nil || !strings.Contains(err.Error(), "no entropy") {
		t.Fatalf("the start = %v; want it to fail minting its run id", err)
	}
	if tomb := originReservation(t, bare, claim.RefPrefix+secondIssue); !tomb.released {
		t.Fatalf("the failed start left its reservation holding: %+v", tomb)
	}
	if kind := trailerOn(t, bare, secondIssue, "Abcd-Handed-Back"); len(kind) != 1 || kind[0] != claim.HandBackDesignFinding {
		t.Fatalf("the failed start's release marks hand-back %v; want the design-finding hand-back kept", kind)
	}
}
