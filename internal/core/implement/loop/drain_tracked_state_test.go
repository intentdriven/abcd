package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// TestADrainStateACommitCarriesIsRefusedAndPublishesNothing: the drain's state
// is this checkout's own. One that reaches a fresh clone through a commit (added
// past the local tier's ignore rule) was written by whoever made the commit, so
// the clone's drain refuses it, naming the file, and publishes none of the
// hand-backs it lists: neither over a released branch nor over an absent one.
func TestADrainStateACommitCarriesIsRefusedAndPublishesNothing(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Footprint: []string{"commands"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: secondIssue, Release: true}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	scratch := t.TempDir()
	if err := os.MkdirAll(filepath.Join(scratch, RunRelDir), 0o755); err != nil {
		t.Fatal(err)
	}
	st := DrainState{SchemaVersion: DrainSchemaVersion, PendingHandBacks: []DrainPendingHandBack{
		{Issue: secondIssue, Kind: HandBackTrustRule, Blob: originBlob(t, bare, secondIssue), Why: "listed by another checkout", At: now},
		{Issue: eligibleIssue, Kind: HandBackTrustRule, Blob: originBlob(t, bare, eligibleIssue), Why: "listed by another checkout", At: now},
	}}
	if err := writeDrain(scratch, st); err != nil {
		t.Fatalf("writeDrain: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(scratch, DrainStateRel))
	if err != nil {
		t.Fatal(err)
	}
	pushFromOtherCopy(t, bare, "an unrelated change", func(dir string) {
		p := filepath.Join(dir, DrainStateRel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, data, 0o644); err != nil {
			t.Fatal(err)
		}
		gitIn(t, dir, "add", "-f", DrainStateRel)
	})

	q := otherCopy(t, bare, quinnEmail)
	if _, err := os.Stat(filepath.Join(q, DrainStateRel)); err != nil {
		t.Fatalf("the clone should check the committed drain state out: %v", err)
	}
	beforeSecond, beforeEligible := refOf(t, bare, secondIssue), refOf(t, bare, eligibleIssue)
	asHomeOf(t, q, func() { _, err = Drain(q, Options{}, DrainOptions{}) })
	r := mustRefusal(t, err)
	if r.Stage != StageDrain || !strings.Contains(r.Reason, "git tracks "+DrainStateRel) || !strings.Contains(r.Remedy, "git rm --cached") {
		t.Fatalf("the drain should refuse the committed state by name and say how to untrack it: %+v", r)
	}
	if got := refOf(t, bare, secondIssue); got != beforeSecond {
		t.Fatalf("%s's branch moved from %s to %s", secondIssue, beforeSecond, got)
	}
	if got := refOf(t, bare, eligibleIssue); got != beforeEligible {
		t.Fatalf("%s's branch moved from %q to %q", eligibleIssue, beforeEligible, got)
	}
	for _, id := range []string{secondIssue, eligibleIssue} {
		if kind, _ := tombstoneTrailers(t, bare, id); kind != "" {
			t.Fatalf("a hand-back over %s was published from a state the clone did not write", id)
		}
	}
	// The same file, untracked in the clone, is the clone's own again and reads.
	gitIn(t, q, "rm", "--cached", "-q", DrainStateRel)
	if _, _, err := readDrain(q); err != nil {
		t.Fatalf("an untracked drain state should read: %v", err)
	}
}
