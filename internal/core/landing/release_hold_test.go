package landing

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gittest"
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

// homeOf is the home the copy at dir keeps its person's holder id in: the
// fixture's home for its own checkout and P's other copies, as every copy of
// P's on this machine shares it, and a home of their own for anyone else, as
// their own machine would be, so a second person's holds are never P's.
func (f *fixture) homeOf(dir string) string {
	f.t.Helper()
	// --local: the fixture's own git calls name an email of their own with -c.
	email := strings.TrimSpace(f.repo.Git("-C", dir, "config", "--local", "user.email"))
	if dir == f.repo.Root() || strings.EqualFold(email, patEmail) {
		return f.home
	}
	home := filepath.Join(f.home, "people", strings.NewReplacer("+", "-", "@", "-").Replace(strings.ToLower(email)))
	if err := os.MkdirAll(home, 0o700); err != nil {
		f.t.Fatal(err)
	}
	return home
}

// heldBy is the holder id the copy at dir's holds name (homeOf).
func (f *fixture) heldBy(dir string) string {
	f.t.Helper()
	data, err := os.ReadFile(abcdhome.Path(f.homeOf(dir), "holder-id"))
	if err != nil {
		f.t.Fatalf("no holder id is kept for the copy at %s: %v", dir, err)
	}
	return strings.TrimSpace(string(data))
}

// cut reserves the release subject from the working copy dir at at, as a
// session starting a cut does, with the release's default lease.
func (f *fixture) cut(dir, subject string, at time.Time) {
	f.t.Helper()
	o := claim.Options{Now: func() time.Time { return at }, Home: f.homeOf(dir)}
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
	q := f.copyOf(quinnEmail)
	f.cut(q, "rel-v0.16.0", at)
	_, err := f.land(f.options())
	r := wantHeld(t, err)
	want := "release v0.16.0 is being cut by " + f.heldBy(q) + ", until " + at.Add(24*time.Hour).Format(time.RFC3339) +
		", and a session cutting a release has the final say on what merges before its tag, so abcd land pushes, arms and queues nothing"
	if r.Reason != want {
		t.Fatalf("reason = %q; want %q", r.Reason, want)
	}
	if r.Fix != "the copy cutting v0.16.0 may merge this itself; otherwise wait until v0.16.0 is tagged, or until rel-v0.16.0, held by "+f.heldBy(q)+" from that copy, is released or the hold lapses at "+
		at.Add(24*time.Hour).Format(time.RFC3339)+" unrenewed, when nothing holds the merge, then run abcd land again" {
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

// TestAnEndedCutHoldsNoLanding: a cut ends without a tag too. Q's release
// reservation, never renewed, holds nothing once its day is out, and one Q
// released (its tombstone) holds nothing at once; either way land pushes and
// arms though the branch abcd-reserved/rel-v0.16.0 is still on origin.
func TestAnEndedCutHoldsNoLanding(t *testing.T) {
	for _, end := range []string{"its day ran out", "its tombstone"} {
		t.Run(end, func(t *testing.T) {
			f := newFixture(t, queueRuleset)
			f.asPat()
			f.forge.prs = f.prFor(f.head())
			f.forge.views = []string{viewBlocked, viewQueued}
			q := f.copyOf(quinnEmail)
			now := time.Now().UTC().Truncate(time.Second)
			if end == "its day ran out" {
				f.cut(q, "rel-v0.16.0", now.Add(-24*time.Hour-time.Minute))
			} else {
				f.cut(q, "rel-v0.16.0", now.Add(-time.Hour))
				o := claim.Options{Now: func() time.Time { return now.Add(-time.Minute) }, Home: f.homeOf(q)}
				if res, err := claim.Reserve(q, claim.Request{Record: "rel-v0.16.0", Release: true}, o); err != nil || res.Act != claim.ActReleased {
					t.Fatalf("releasing the cut: %+v %v", res, err)
				}
			}
			if f.onOrigin(strings.TrimPrefix(claim.RefPrefix, "refs/heads/")+"rel-v0.16.0") == "" {
				t.Fatal("the release's reservation branch is gone from origin; want it still there")
			}
			if _, err := f.land(f.options()); err != nil {
				t.Fatalf("land once %s: %v", end, err)
			}
			if f.onOrigin("feat/x") != f.head() || f.forge.called("pr", "merge", "7", "--auto") != 1 {
				t.Fatalf("origin feat/x %q, forge calls %v: want pushed and armed", f.onOrigin("feat/x"), f.forge.calls)
			}
		})
	}
}

// TestLandArmsNothingWhenTheHoldCannotBeRead: origin stops answering after
// the default branch was fetched (here, during the gate); a hold land cannot
// see may be there, so nothing is pushed, armed or queued. Nobody is known to
// hold anything, so it is a refusal (exit 2), not a hold (exit 3).
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
	var r *Refusal
	if !errors.As(err, &r) || r.Held {
		t.Fatalf("err = %v; want a refusal that is not a hold, since nobody holds the release", err)
	}
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
	if _, err := releaseHold(root, f.home); err == nil {
		t.Fatal("with origin there, nothing holds; want Q's release to hold the landing")
	}
	ref := claim.TrackingPrefix + "rel-v0.16.0"
	tip := f.repo.Git("rev-parse", ref)
	f.repo.Git("remote", "remove", claim.Remote)
	f.repo.Git("update-ref", ref, tip)
	if _, err := releaseHold(root, f.home); err != nil {
		t.Fatalf("with no origin: %v; want nothing held", err)
	}
}

// pushDatedRelease pushes a release reservation for subject onto origin,
// claimed at at by quinn from another copy and committed at at, from a
// scratch clone: a cut reserved on a clock that may run far ahead of this one.
func (f *fixture) pushDatedRelease(subject string, at time.Time) {
	f.t.Helper()
	dir := filepath.Join(f.t.TempDir(), "ahead")
	f.repo.Git("clone", "-q", f.bare, dir)
	msg := filepath.Join(f.t.TempDir(), "msg")
	text := "abcd reservation: " + subject + "\n\nAbcd-Reservation: " + subject + "\nAbcd-Holder: quinn\n" +
		"Abcd-Copy: 9f2c41d07be35a6c80e1f4b2a7d39c55\nAbcd-Claimed-At: " + at.UTC().Format(time.RFC3339) +
		"\nAbcd-Expires-At: " + at.UTC().Add(24*time.Hour).Format(time.RFC3339) + "\n"
	if err := os.WriteFile(msg, []byte(text), 0o600); err != nil {
		f.t.Fatal(err)
	}
	tree := f.repo.Git("-C", dir, "hash-object", "-w", "-t", "tree", "/dev/null")
	date := "@" + strconv.FormatInt(at.Unix(), 10) + " +0000"
	cmd := exec.Command("git", "-C", dir, "-c", "user.name=Quinn", "-c", "user.email=quinn@example.invalid", "-c", "commit.gpgsign=false",
		"commit-tree", tree, "-F", msg)
	cmd.Env = append(gittest.Env(f.t), "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
	out, err := cmd.Output()
	if err != nil {
		f.t.Fatal(err)
	}
	f.repo.Git("-C", dir, "push", "-q", "origin", strings.TrimSpace(string(out))+":"+claim.RefPrefix+subject)
}

// TestLandWithNoLocalTierCountsNoReleaseDatedAhead: a new worktree has no
// local tier, so it cannot keep when it first saw a release reservation (here
// its gate is skipped on a receipt another worktree minted). A release dated
// five years ahead would hold it from every reading, so it is not counted:
// land pushes, and its notice names the release. A release dated within the
// allowance holds the same worktree as any cut does, and no tier is made.
func TestLandWithNoLocalTierCountsNoReleaseDatedAhead(t *testing.T) {
	for _, c := range []struct {
		name string
		at   time.Duration
		held bool
	}{{"dated five years ahead", 5 * 365 * 24 * time.Hour, false}, {"dated a minute ahead", time.Minute, true}} {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t, queueRuleset)
			f.forge.prs = "[]"
			f.forge.head = f.head()
			wt := filepath.Join(t.TempDir(), "wt")
			f.repo.Git("worktree", "add", "-q", "-b", "feat/y", wt, "feat/x")
			if err := f.mint(f.repo.Root()); err != nil {
				t.Fatal(err)
			}
			f.pushDatedRelease("rel-v0.16.0", time.Now().UTC().Truncate(time.Second).Add(c.at))
			res, err := Land(wt, f.options())
			if c.held {
				if r := wantHeld(t, err); !strings.Contains(r.Reason, "release v0.16.0 is being cut by quinn") {
					t.Fatalf("reason = %q; want quinn's release named", r.Reason)
				}
				if len(res.Notices) != 0 || f.onOrigin("feat/y") != "" {
					t.Fatalf("notices %q, origin feat/y %q: want no notice and nothing pushed", res.Notices, f.onOrigin("feat/y"))
				}
			} else {
				if err != nil {
					t.Fatalf("land: %v; want the release dated ahead not counted", err)
				}
				want := "release v0.16.0 is reserved by quinn with a date more than 5 minutes ahead of this clock, and this checkout cannot keep when it first saw it"
				if len(res.Notices) != 1 || !strings.HasPrefix(res.Notices[0], want) || f.onOrigin("feat/y") != f.head() {
					t.Fatalf("notices %q, origin feat/y %q: want one notice naming the release and the branch pushed", res.Notices, f.onOrigin("feat/y"))
				}
			}
			if _, err := os.Stat(filepath.Join(wt, ".abcd", ".work.local")); err == nil {
				t.Fatal("land made the worktree's local tier")
			}
		})
	}
}

// TestLandFromTheCuttingCheckoutIsNotHeldByItsOwnCut: P reserves the release
// from this checkout, then lands a branch checked out in another worktree. The
// cut is judged against the checkout land runs in, whose copy reserved it, so
// it is not held by its own hold.
func TestLandFromTheCuttingCheckoutIsNotHeldByItsOwnCut(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.asPat()
	f.forge.prs = "[]"
	wt := filepath.Join(t.TempDir(), "wt")
	f.repo.Git("worktree", "add", "-q", "-b", "feat/y", wt, "feat/x")
	f.cut(f.repo.Root(), "rel-v0.16.0", time.Now().UTC().Add(-time.Minute))
	o := f.options()
	o.Branch = "feat/y"
	if _, err := f.land(o); err != nil {
		t.Fatalf("land feat/y from the cutting checkout: %v; want it not held", err)
	}
	if f.onOrigin("feat/y") != f.head() {
		t.Fatalf("origin feat/y %q; want %s pushed", f.onOrigin("feat/y"), f.head())
	}
}
