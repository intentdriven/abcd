package landing

import (
	"context"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/spotcheck"
)

// The merge gate's decisions recorded for the spot-check (spc-2610301925429250
// step 1): a decision abcd takes as the facilitator, in auto mode over a
// maintainer's pull request, is kept on the store's local copy once per head
// and answer; every other decision waits for a person and is not.

// gateRecordedAt asks the recorded gate about a fixture's head.
func gateRecordedAt(t *testing.T, f *fixture) MergeDecision {
	t.Helper()
	d, err := DecideMergeRecorded(MergeQuery{RepoRoot: f.repo.Root(), Roots: f.options().Roots, Tip: f.repo.Git("rev-parse", "origin/main"), TipName: "origin/main", Default: "main",
		Repo: "o/r", PR: "7", Head: f.head(), Forge: func(args ...string) (string, error) { return f.forge.run("", args...) }}, spotcheck.SiteLand)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func gateRecords(t *testing.T, f *fixture) []spotcheck.Decision {
	t.Helper()
	rec, _, err := spotcheck.Read(context.Background(), f.repo.Root(), spotcheck.LocalRef)
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.Unreadable) != 0 {
		t.Fatalf("unreadable records: %v", rec.Unreadable)
	}
	return rec.Decisions
}

// TestAnAutoDecisionOverAMaintainersPullRequestIsRecorded: the record names
// the site, the pull request and head, the mode and its layer, the role,
// whether a receipt named the head and the gate's answer; the gate asked again
// over the same head with the same answer keeps the one record; and the
// recorded gate asks the forge exactly what the gate alone asks.
func TestAnAutoDecisionOverAMaintainersPullRequestIsRecorded(t *testing.T) {
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	f.forge.head = f.head()
	plain := gateAt(t, f)
	asked := len(f.forge.calls)
	d := gateRecordedAt(t, f)
	if d != plain {
		t.Fatalf("the recorded gate decided %+v; the gate alone %+v", d, plain)
	}
	if got := len(f.forge.calls) - asked; got != asked {
		t.Fatalf("the recorded gate made %d forge calls; the gate alone makes %d", got, asked)
	}
	recs := gateRecords(t, f)
	if len(recs) != 1 {
		t.Fatalf("records: %+v", recs)
	}
	r := recs[0]
	if r.Kind != spotcheck.KindMergeGate || r.Site != spotcheck.SiteLand || r.PR != 7 || r.Head != f.head() || r.Mode != ModeAuto ||
		r.ModeLayer == "" || r.Role != RoleMaintainer || r.Reviewed || r.Arm != d.Arm || r.Why == "" {
		t.Fatalf("the record is %+v", r)
	}
	gateRecordedAt(t, f)
	if n := len(gateRecords(t, f)); n != 1 {
		t.Fatalf("the gate asked again over the same head kept %d records", n)
	}
}

// TestAHumanModeOrContributorDecisionIsNotRecorded: in human mode a
// person's approval decides, and a contributor's pull request waits for a
// maintainer's in either mode; neither is abcd's decision alone, and neither
// is recorded.
func TestAHumanModeOrContributorDecisionIsNotRecorded(t *testing.T) {
	human := newFixtureFiles(t, queueRuleset, nil)
	human.forge.head = human.head()
	gateRecordedAt(t, human)
	contributor := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	contributor.forge.head = contributor.head()
	contributor.forge.roles = map[string]string{"author-a": `{"permission":"read","role_name":"read","user":{"login":"author-a"}}`,
		"maintainer-b": defaultRoles["maintainer-b"]}
	gateRecordedAt(t, contributor)
	for name, f := range map[string]*fixture{"human mode": human, "contributor": contributor} {
		if out := f.repo.Git("for-each-ref", spotcheck.LocalRef); out != "" {
			t.Errorf("%s: a record was written: %s", name, out)
		}
	}
}

// TestARecordThatCannotBeWrittenChangesNothingTheGateDecided: a store that
// cannot be read refuses the write; the gate's answer is the same, its reason
// says the record was not written, and nothing is refused.
func TestARecordThatCannotBeWrittenChangesNothingTheGateDecided(t *testing.T) {
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	f.forge.head = f.head()
	plain := gateAt(t, f)
	blob := strings.TrimSpace(f.repo.Git("hash-object", "-w", "feature.txt"))
	f.repo.Git("update-ref", spotcheck.TrackingRef, blob)
	d := gateRecordedAt(t, f)
	if d.Arm != plain.Arm || d.Method != plain.Method || d.Unread != plain.Unread || !strings.HasPrefix(d.Why, plain.Why) ||
		!strings.Contains(d.Why, "the spot-check's record of this decision was not written") {
		t.Fatalf("the gate decided %+v; alone %+v", d, plain)
	}
	if out := f.repo.Git("for-each-ref", spotcheck.LocalRef); out != "" {
		t.Fatalf("a record was written over an unreadable store: %s", out)
	}
}
