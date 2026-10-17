package loop

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/core/lease"
)

// The land fence and `abcd land` make one release-hold judgement
// (claim.FetchMergeHold): the same release holds both, or neither, and each
// names it in the same words before its own last clause. They part on
// purpose in one place, pinned here: a reservations fetch that fails.

// landHold is `abcd land`'s release hold judged in root, as a *landing.Refusal
// (nil when nothing holds the landing).
func landHold(t *testing.T, root string) *landing.Refusal {
	t.Helper()
	_, err := landing.ReleaseHoldForTest(root, "")
	if err == nil {
		return nil
	}
	var r *landing.Refusal
	if !errors.As(err, &r) {
		t.Fatalf("land's release hold: %v; want a *landing.Refusal", err)
	}
	return r
}

// fenceHold is the land fence's release hold for a run holding no
// reservation, judged in root at now (nil when nothing holds the lane).
func fenceHold(t *testing.T, root string, now time.Time) *Refusal {
	t.Helper()
	_, _, err := landFence(Context{RepoRoot: root, Now: now}, Lane{ID: "lane-1"}, fenceLease)
	if err == nil {
		return nil
	}
	r, ok := AsRefusal(err)
	if !ok {
		t.Fatalf("the fence: %v; want a *Refusal", err)
	}
	return r
}

// TestAFailedFetchIsContentionAtTheFenceAndARefusalAtLand pins an intended
// difference: when the reservations cannot be fetched, both arm nothing, but
// the land fence reports it as contention (exit 3), which a run or a drain
// backs off from and retries on its next step, while `abcd land` refuses it
// (Held false, exit 2), since nobody is known to hold anything and a person
// ran the verb.
func TestAFailedFetchIsContentionAtTheFenceAndARefusalAtLand(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	root := f.repo.Root()
	if err := os.Rename(f.bare, f.bare+".gone"); err != nil {
		t.Fatal(err)
	}
	const named = "could not read the reservations from origin: "
	fence := fenceHold(t, root, time.Now().UTC().Truncate(time.Second))
	if fence == nil || !fence.Contention || !strings.HasPrefix(fence.Reason, named) {
		t.Fatalf("the fence: %+v; want contention naming the failed fetch", fence)
	}
	land := landHold(t, root)
	if land == nil || land.Held || !strings.HasPrefix(land.Reason, named) {
		t.Fatalf("land: %+v; want a refusal that is not a hold, naming the failed fetch", land)
	}
}

// TestTheFenceAndLandJudgeOneReleaseAlike: Q cuts v0.16.0, and the fence and
// land in P's checkout both hold, naming the release, Q and until when in the
// same words; once Q releases it, neither holds.
func TestTheFenceAndLandJudgeOneReleaseAlike(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	root := f.repo.Root()
	asPat(t, f.repo)
	now := time.Now().UTC().Truncate(time.Second)
	q := otherCopy(t, f.bare, quinnEmail)
	reserveRelease(t, q, "rel-v0.16.0", now.Add(-time.Minute))

	fence, land := fenceHold(t, root, now), landHold(t, root)
	if fence == nil || !fence.Contention || land == nil || !land.Held {
		t.Fatalf("fence %+v, land %+v; want both held", fence, land)
	}
	shared := "release v0.16.0 is being cut by quinn, until " + stamp(now.Add(-time.Minute).Add(lease.Max)) +
		", and a session cutting a release has the final say on what merges before its tag, so "
	if !strings.HasPrefix(fence.Reason, shared) || !strings.HasPrefix(land.Reason, shared) {
		t.Fatalf("fence %q, land %q; want both to start %q", fence.Reason, land.Reason, shared)
	}

	if _, err := claim.Reserve(q, claim.Request{Record: "rel-v0.16.0", Release: true}, clockAt(now)); err != nil {
		t.Fatal(err)
	}
	if fence, land := fenceHold(t, root, now), landHold(t, root); fence != nil || land != nil {
		t.Fatalf("once released: fence %+v, land %+v; want neither held", fence, land)
	}
}

// TestTheFenceStopsHoldingWhenAReleasesDayRunsOut: a release reserved with
// its default lease and never renewed holds the fence until 24 hours from
// its claim, and not after, with no tag and no release to end it.
func TestTheFenceStopsHoldingWhenAReleasesDayRunsOut(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	root := f.repo.Root()
	now := time.Now().UTC().Truncate(time.Second)
	q := otherCopy(t, f.bare, quinnEmail)
	reserveRelease(t, q, "rel-v0.16.0", now)
	for _, c := range []struct {
		at   time.Duration
		held bool
	}{{time.Hour, true}, {lease.Max - time.Second, true}, {lease.Max, false}, {lease.Max + time.Hour, false}} {
		r := fenceHold(t, root, now.Add(c.at))
		if (r != nil) != c.held || (c.held && (!r.Contention || !strings.Contains(r.Reason, "until "+stamp(now.Add(lease.Max))))) {
			t.Fatalf("at +%s: %+v; want held %v", c.at, r, c.held)
		}
	}
}
