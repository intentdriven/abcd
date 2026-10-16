package landing

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// The release hold at `abcd land` (itd-2609151838312703 C7,
// spc-2610150728396006 step 4): before it pushes, arms or queues anything,
// land makes the land fence's own judgement (claim.FetchMergeHold). It
// fetches the reservations and origin's release tags, and refuses as
// contention while a release another session is cutting holds every merge;
// the cutting copy's own landing is its ruling. An origin whose reservations
// cannot be read pushes and arms nothing.

// The usernames the fixtures' no-reply addresses name.
const (
	patEmail   = "1234+octo@users.noreply.github.com"
	quinnEmail = "99+quinn@users.noreply.github.com"
)

// copyOf clones the fixture's origin as another working copy whose git
// identity is email, with its local tier: a second person, or P's second
// computer when it is P's address.
func (f *fixture) copyOf(email string) string {
	f.t.Helper()
	dir := filepath.Join(f.t.TempDir(), "copy")
	f.repo.Git("clone", "-q", f.bare, dir)
	for _, kv := range [][2]string{{"user.name", "Other Example"}, {"user.email", email}, {"gc.auto", "0"},
		{"maintenance.auto", "false"}, {"core.fsmonitor", "false"}} {
		f.repo.Git("-C", dir, "config", kv[0], kv[1])
	}
	if err := os.MkdirAll(filepath.Join(dir, ".abcd", ".work.local"), 0o755); err != nil {
		f.t.Fatal(err)
	}
	return dir
}

// cut reserves the release subject from the working copy dir at at, as a
// session starting a cut does, with the release's default lease.
func (f *fixture) cut(dir, subject string, at time.Time) {
	f.t.Helper()
	o := claim.Options{Now: func() time.Time { return at }, Home: f.home}
	if res, err := claim.Reserve(dir, claim.Request{Record: subject}, o); err != nil || res.Act != claim.ActReserved {
		f.t.Fatalf("reserving %s: %+v %v", subject, res, err)
	}
}

// asPat gives the fixture's checkout P's git identity and its local tier.
func (f *fixture) asPat() {
	f.t.Helper()
	f.repo.Git("config", "user.name", "Pat Example")
	f.repo.Git("config", "user.email", patEmail)
	if err := os.MkdirAll(filepath.Join(f.repo.Root(), ".abcd", ".work.local"), 0o755); err != nil {
		f.t.Fatal(err)
	}
}

// heldNothing fails unless the landing pushed, armed and queued nothing.
func (f *fixture) heldNothing() {
	f.t.Helper()
	if len(f.pushes()) != 0 || f.onOrigin("feat/x") != "" || f.forge.called("pr", "merge") != 0 {
		f.t.Fatalf("pushed %v (origin feat/x %q), forge calls %v: want nothing pushed, armed or queued",
			f.pushes(), f.onOrigin("feat/x"), f.forge.calls)
	}
}

// wantHeld returns the landing's refusal, failing unless it is a hold.
func wantHeld(t *testing.T, err error) *Refusal {
	t.Helper()
	var r *Refusal
	if !errors.As(err, &r) || !r.Held {
		t.Fatalf("err = %v, want a refusal held as contention", err)
	}
	return r
}

// TestLandWaitsWhileAnotherPersonCutsARelease is C7 at `abcd land`: Q
// reserves v0.16.0 after this checkout last fetched; land fetches, names the
// release, Q and until when, and pushes, arms and queues nothing.
func TestLandWaitsWhileAnotherPersonCutsARelease(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.asPat()
	f.forge.prs = f.prFor(f.head())
	f.forge.views = []string{viewBlocked, viewQueued}
	at := time.Now().UTC().Truncate(time.Second).Add(-time.Minute)
	f.cut(f.copyOf(quinnEmail), "rel-v0.16.0", at)
	_, err := f.land(f.options())
	r := wantHeld(t, err)
	want := "release v0.16.0 is being cut by quinn, until " + at.Add(24*time.Hour).Format(time.RFC3339) +
		", and a session cutting a release has the final say on what merges before its tag, so abcd land pushes, arms and queues nothing"
	if r.Reason != want {
		t.Fatalf("reason = %q; want %q", r.Reason, want)
	}
	if r.Fix != "ask quinn to merge it or to release rel-v0.16.0, or wait until v0.16.0 is tagged, then run abcd land again" {
		t.Fatalf("fix = %q", r.Fix)
	}
	f.heldNothing()
}

// TestLandWaitsWhileYourOtherCopyCutsARelease: P cuts from another computer;
// this checkout is another session, so it is held like anyone's.
func TestLandWaitsWhileYourOtherCopyCutsARelease(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.asPat()
	f.forge.prs = f.prFor(f.head())
	f.cut(f.copyOf(patEmail), "rel-v0.16.0", time.Now().UTC().Add(-time.Minute))
	_, err := f.land(f.options())
	if r := wantHeld(t, err); !strings.Contains(r.Reason, "being cut by your copy on another computer") {
		t.Fatalf("reason = %q; want it to name your other copy", r.Reason)
	}
	f.heldNothing()
}

// TestTheCuttingCopyLandsItsOwnBranch: the copy cutting the release lands
// through the hold; the cut is its to rule on.
func TestTheCuttingCopyLandsItsOwnBranch(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.asPat()
	f.forge.prs = f.prFor(f.head())
	f.forge.views = []string{viewBlocked, viewQueued}
	f.cut(f.repo.Root(), "rel-v0.16.0", time.Now().UTC().Add(-time.Minute))
	if _, err := f.land(f.options()); err != nil {
		t.Fatal(err)
	}
	if f.onOrigin("feat/x") != f.head() || f.forge.called("pr", "merge", "7", "--auto") != 1 {
		t.Fatalf("origin feat/x %q, forge calls %v: want pushed and armed", f.onOrigin("feat/x"), f.forge.calls)
	}
}

// TestATaggedCutHoldsNoLanding: once v0.16.0 is tagged on origin the cut is
// done, and land pushes and arms though Q's reservation is still there.
func TestATaggedCutHoldsNoLanding(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.asPat()
	f.forge.prs = f.prFor(f.head())
	f.forge.views = []string{viewBlocked, viewQueued}
	f.cut(f.copyOf(quinnEmail), "rel-v0.16.0", time.Now().UTC().Add(-time.Hour))
	f.repo.Git("-C", f.bare, "tag", "v0.16.0", "main")
	if _, err := f.land(f.options()); err != nil {
		t.Fatal(err)
	}
	if f.onOrigin("feat/x") != f.head() || f.forge.called("pr", "merge", "7", "--auto") != 1 {
		t.Fatalf("origin feat/x %q, forge calls %v: want pushed and armed", f.onOrigin("feat/x"), f.forge.calls)
	}
}

// TestLandArmsNothingWhenTheHoldCannotBeRead: origin stops answering after
// the default branch was fetched (here, during the gate); a hold land cannot
// see may be there, so nothing is pushed, armed or queued.
func TestLandArmsNothingWhenTheHoldCannotBeRead(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.gate = func(dir string) error {
		if err := f.mint(dir); err != nil {
			return err
		}
		return os.Rename(f.bare, f.bare+".gone")
	}
	_, err := f.land(f.options())
	r := wantHeld(t, err)
	if !strings.HasPrefix(r.Reason, "could not read the reservations from origin: ") ||
		!strings.HasSuffix(r.Reason, ", so no release hold could be judged and abcd land pushes, arms and queues nothing") ||
		r.Fix != "run abcd land again once origin answers" {
		t.Fatalf("refusal = %+v; want the failed fetch named", r)
	}
	if err := os.Rename(f.bare+".gone", f.bare); err != nil {
		t.Fatal(err)
	}
	f.heldNothing()
}

// TestACheckoutWithNoOriginIsNotHeldByLand: with no remote named origin
// there are no reservations, so the hold holds nothing, even with a release
// reservation left from before origin was removed. (Land itself refuses such
// a checkout earlier, for want of a default branch on origin.)
func TestACheckoutWithNoOriginIsNotHeldByLand(t *testing.T) {
	f := newFixture(t, "")
	f.cut(f.copyOf(quinnEmail), "rel-v0.16.0", time.Now().UTC().Add(-time.Minute))
	root := f.repo.Root()
	if err := releaseHold(root, f.home); err == nil {
		t.Fatal("with origin there, nothing holds; want Q's release to hold the landing")
	}
	ref := claim.TrackingPrefix + "rel-v0.16.0"
	tip := f.repo.Git("rev-parse", ref)
	f.repo.Git("remote", "remove", claim.Remote)
	f.repo.Git("update-ref", ref, tip)
	if err := releaseHold(root, f.home); err != nil {
		t.Fatalf("with no origin: %v; want nothing held", err)
	}
}
