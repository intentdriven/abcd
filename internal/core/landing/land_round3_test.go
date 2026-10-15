package landing

import (
	"strings"
	"testing"
)

// R3-F1: a pull request merged at the head into a branch other than the
// default (a stacked pull request) has not landed: the work is not on the
// default branch, so land gates and pushes as for any branch.
func TestAPullRequestMergedIntoAnotherBaseIsNotLanded(t *testing.T) {
	f := newFixture(t, "")
	f.squashOntoMain()
	f.forge.merged = "[" + mergedEntry(7, f.head(), "feat/parent", false) + "]"
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if res.Landed || len(f.gates) != 1 || f.onOrigin("feat/x") != f.head() {
		t.Fatalf("result %+v, gates %v, origin %q: want an ordinary landing", res, f.gates, f.onOrigin("feat/x"))
	}
}

// R3-F2: a pull request from another repository (a fork's branch of the same
// name) merged at the head proves nothing about this branch.
func TestMergedAtSetsAsideAPullRequestFromAnotherRepository(t *testing.T) {
	sha := strings.Repeat("a", 40)
	ask := func(...string) (string, error) { return "[" + mergedEntry(9, sha, "main", true) + "]", nil }
	n, url, err := mergedAt(ask, "feat/x", "main", sha)
	if err != nil || n != 0 || url != "" {
		t.Fatalf("mergedAt = %d, %q, %v: want 0 for a pull request from another repository", n, url, err)
	}
	ask = func(...string) (string, error) { return "[" + mergedEntry(9, sha, "main", false) + "]", nil }
	if n, url, err := mergedAt(ask, "feat/x", "main", sha); err != nil || n != 9 || url != "https://example.com/o/r/pull/9" {
		t.Fatalf("mergedAt = %d, %q, %v: want #9 from this repository", n, url, err)
	}
}

// R3-F3: a listing the bound filled with no pull request at the head cannot
// prove the branch has not merged (the proof may be past the bound), so land
// refuses the push and says why rather than recreate a deleted branch.
func TestAFullMergedListingWithNoMatchRefusesThePush(t *testing.T) {
	f := newFixture(t, "")
	var entries []string
	for i := range maxMergedListed {
		entries = append(entries, mergedEntry(100+i, strings.Repeat("c", 40), "main", true))
	}
	f.forge.merged = "[" + strings.Join(entries, ",") + "]"
	res, err := f.land(f.options())
	wantRefusal(t, err, "20")
	wantRefusal(t, err, "nothing was gated or pushed")
	if res.Landed || len(f.pushes()) != 0 || len(f.gates) != 0 {
		t.Fatalf("result %+v, pushed %v, gated %v: want a refusal and neither", res, f.pushes(), f.gates)
	}
}

// R3-F6: the queue state is asked of the repository the forge client named
// when origin was checked, through the GraphQL query (gh pr view --json has
// no merge-queue field), and a name that is not owner/name arms nothing.
func TestTheQueueStateIsAskedOfTheForgesOwnRepository(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.forge.nwo = "acme/widgets"
	f.forge.prs = f.prFor(f.head())
	f.forge.views = []string{viewBlocked, viewQueued}
	if _, err := f.land(f.options()); err != nil {
		t.Fatal(err)
	}
	if f.forge.called("api", "graphql") != 2 || f.forge.called("pr", "view") != 0 {
		t.Fatalf("forge calls %v: want the state asked through the API twice, never through pr view", f.forge.calls)
	}

	f = newFixture(t, queueRuleset)
	f.forge.nwo = "acme/widgets/extra"
	f.forge.prs = f.prFor(f.head())
	_, err := f.land(f.options())
	wantRefusal(t, err, "not owner/name")
	if f.forge.called("pr", "merge") != 0 || f.forge.called("api") != 0 {
		t.Fatalf("forge calls %v: want nothing asked or armed", f.forge.calls)
	}
}
