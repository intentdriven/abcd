package loop

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The run's reservation in `implement step` (spc-2610031753359849 spec 2,
// steps 8 and 9): renewed at half its lease before a stage, fenced and
// renewed past the queue's timeout before the landing arms a merge, and
// tombstoned once the lane that completes the run has landed. A reservation
// test runs against the fixture's bare origin, with further clones standing
// in for a second person and a second computer; never the real remote.

// The usernames the fixtures' no-reply addresses name.
const (
	patEmail   = "1234+octo@users.noreply.github.com"
	quinnEmail = "99+quinn@users.noreply.github.com"
)

// reservedRef is the run's reservation branch as origin holds it.
const reservedRef = claim.RefPrefix + "itd-10"

// asPat gives the fixture's checkout P's git identity, a GitHub no-reply
// address, as a person's own git configuration names it.
func asPat(t *testing.T, repo *gittest.Repo) {
	t.Helper()
	repo.Git("config", "user.name", "Pat Example")
	repo.Git("config", "user.email", patEmail)
}

// clockAt is a claim clock fixed at at.
func clockAt(at time.Time) claim.Options {
	return claim.Options{Now: func() time.Time { return at }}
}

// reserveRun reserves the run's key on origin from the fixture's checkout,
// as P at the time at, and records the reservation in the run's state, as a
// start that reserves does.
func reserveRun(t *testing.T, repo *gittest.Repo, runID, key string, at time.Time, footprint ...string) {
	t.Helper()
	asPat(t, repo)
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: key, Footprint: footprint}, clockAt(at)); err != nil {
		t.Fatalf("reserving %s: %v", key, err)
	}
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		st.ReservedAt = &at
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
}

// otherCopy clones origin as another working copy whose git identity is
// email: P's second computer when it is P's address, a second person
// otherwise.
func otherCopy(t *testing.T, bare, email string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "copy")
	gitIn(t, "", "clone", "-q", bare, dir)
	for _, kv := range [][2]string{{"user.name", "Other Example"}, {"user.email", email}, {"gc.auto", "0"},
		{"maintenance.auto", "false"}, {"core.fsmonitor", "false"}} {
		gitIn(t, dir, "config", kv[0], kv[1])
	}
	if err := os.MkdirAll(filepath.Join(dir, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// gitIn runs git in dir ("" for none) under the hermetic test environment.
func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	if dir != "" {
		args = append([]string{"-C", dir}, args...)
	}
	cmd := exec.Command("git", args...)
	cmd.Env = gittest.Env(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// reservation is the lease the tip of origin's reservation branch states.
type reservation struct {
	commit, parent, holder string
	claimed, expires       time.Time
	released               bool
}

// originReservation reads the reservation branch's tip on origin; commit is
// "" when origin has none.
func originReservation(t *testing.T, bare, ref string) reservation {
	t.Helper()
	sha := gitIn(t, bare, "for-each-ref", "--format=%(objectname)", ref)
	if sha == "" {
		return reservation{}
	}
	r := reservation{commit: sha}
	if f := strings.Fields(gitIn(t, bare, "rev-list", "--parents", "-n", "1", sha)); len(f) > 1 {
		r.parent = f[1]
	}
	for _, line := range strings.Split(gitIn(t, bare, "log", "-1", "--format=%B", sha), "\n") {
		k, v, ok := strings.Cut(line, ": ")
		if !ok {
			continue
		}
		switch k {
		case "Abcd-Holder":
			r.holder = v
		case "Abcd-Claimed-At":
			r.claimed, _ = time.Parse(time.RFC3339, v)
		case "Abcd-Expires-At":
			r.expires, _ = time.Parse(time.RFC3339, v)
		case "Abcd-Released-At":
			r.released = true
		}
	}
	return r
}

// recordStages are the run record's stages, in order.
func recordStages(t *testing.T, repo *gittest.Repo, runID string) []string {
	t.Helper()
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range st.Record {
		out = append(out, e.Stage)
	}
	return out
}

// recordNote is the run record's last note at stage, "" when none.
func recordNote(t *testing.T, repo *gittest.Repo, runID, stage string) string {
	t.Helper()
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	note := ""
	for _, e := range st.Record {
		if e.Stage == stage {
			note = e.Note
		}
	}
	return note
}

// TestAStepRenewsAtHalfLease is C40: a step that finds the run's reservation
// half through its lease, as the view reads it, fetches and pushes a renewal
// with the default lease before it performs the stage, and the run record
// logs it as reservation_renewed.
func TestAStepRenewsAtHalfLease(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-61*time.Minute), "internal/widget")
	before := originReservation(t, f.bare, reservedRef)

	res := f.step(t)
	after := originReservation(t, f.bare, reservedRef)
	if after.commit == before.commit || after.parent != before.commit {
		t.Fatalf("the step pushes a renewal as the tip's child: before %+v, after %+v", before, after)
	}
	if after.claimed.Before(now) || after.expires.Sub(after.claimed) != 2*time.Hour || after.holder != "octo" {
		t.Fatalf("the renewal counts the default lease from the step: %+v", after)
	}
	if note := recordNote(t, f.repo, f.runID, StageReservationRenewed); !strings.Contains(note, "itd-10") || !strings.Contains(note, after.expires.Format(time.RFC3339)) {
		t.Fatalf("the run record logs the renewal with its new expiry: %q", note)
	}
	if l := currentLane(t, f.repo, f.runID); l.Stage != StageBrief || res.PerformedStage != StageWorktree {
		t.Fatalf("the stage still runs: %+v %+v", res, l)
	}
}

// TestAStepUnderHalfLeasePushesNothing is C41.
func TestAStepUnderHalfLeasePushesNothing(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-30*time.Minute), "internal/widget")
	before := originReservation(t, f.bare, reservedRef)
	f.step(t)
	if after := originReservation(t, f.bare, reservedRef); after.commit != before.commit {
		t.Fatalf("a step under half its lease pushed %+v over %+v", after, before)
	}
	for _, s := range recordStages(t, f.repo, f.runID) {
		if strings.HasPrefix(s, "reservation") {
			t.Fatalf("a step under half its lease logs %s", s)
		}
	}
}

// TestARenewalFailureOutsideLandIsNamedAndTheStageRuns: a renewal origin
// refuses is named in the step's next move and logged in the run record, and
// the stage still runs: the land fence is where a lost reservation stops the
// run.
func TestARenewalFailureOutsideLandIsNamedAndTheStageRuns(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-61*time.Minute), "internal/widget")
	before := originReservation(t, f.bare, reservedRef)
	hook := "#!/bin/sh\nwhile read old new ref; do case \"$ref\" in refs/heads/abcd-reserved/*) echo 'reservations are closed here' >&2; exit 1;; esac; done\nexit 0\n"
	if err := os.WriteFile(filepath.Join(f.bare, "hooks", "pre-receive"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	res := f.step(t)
	if after := originReservation(t, f.bare, reservedRef); after.commit != before.commit {
		t.Fatalf("a refused renewal moved the reservation: %+v", after)
	}
	if l := currentLane(t, f.repo, f.runID); l.Stage != StageBrief {
		t.Fatalf("the stage runs after a failed renewal: lane at %s", l.Stage)
	}
	if !strings.Contains(res.Next, "could not renew the reservation on itd-10") || !strings.Contains(res.Next, "reservations are closed here") {
		t.Fatalf("the step names the failed renewal with origin's reason: %q", res.Next)
	}
	if note := recordNote(t, f.repo, f.runID, StageReservationRenewFailed); !strings.Contains(note, "reservations are closed here") {
		t.Fatalf("the run record logs the failed renewal: %q", note)
	}
}

// fenced advances the run expecting the land fence's refusal, and checks that
// nothing was armed and the lane still waits to arm.
func (f *landFixture) fenced(t *testing.T) *Refusal {
	t.Helper()
	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	r, ok := AsRefusal(err)
	if !ok || !r.Contention || r.Stage != string(StageLand) {
		t.Fatalf("the fence refuses the arming as contention: %v", err)
	}
	if strings.Contains(f.ghLog(t), "pr merge") {
		t.Fatalf("the fence armed a merge:\n%s", f.ghLog(t))
	}
	if l := currentLane(t, f.repo, f.runID); l.Landing == nil || l.Landing.Merge != "" || l.Landing.Armed {
		t.Fatalf("the lane still waits to arm: %+v", l.Landing)
	}
	return r
}

// TestTheFenceRefusesAReservationNowAnotherPersons is C42: P's reservation
// lapsed and Q took it over; P's lane reaches the arming, and the fence names
// Q and arms nothing.
func TestTheFenceRefusesAReservationNowAnotherPersons(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.toArm(t)
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-130*time.Minute), "internal/widget")
	q := otherCopy(t, f.bare, quinnEmail)
	if res, err := claim.Reserve(q, claim.Request{Record: "itd-10", Footprint: []string{"internal/gadget"}}, clockAt(now.Add(-time.Minute))); err != nil || res.Act != claim.ActTakenOver {
		t.Fatalf("Q takes the lapsed reservation over: %+v %v", res, err)
	}
	qTip := originReservation(t, f.bare, reservedRef)
	r := f.fenced(t)
	want := "the reservation on itd-10 is now quinn's, until " + qTip.expires.Format(time.RFC3339) + ", so this lane does not arm its merge"
	if r.Reason != want {
		t.Fatalf("the fence's reason = %q; want %q", r.Reason, want)
	}
	if got := originReservation(t, f.bare, reservedRef); got.commit != qTip.commit {
		t.Fatalf("the fence moved Q's reservation: %+v", got)
	}
}

// TestTheFenceRefusesAfterTheOtherCopyTookOver is C53: P's second computer
// took the reservation over; the lane on the first does not arm.
func TestTheFenceRefusesAfterTheOtherCopyTookOver(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.toArm(t)
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-30*time.Minute), "internal/widget")
	p2 := otherCopy(t, f.bare, patEmail)
	if res, err := claim.Reserve(p2, claim.Request{Record: "itd-10", Footprint: []string{"internal/widget"}, TakeOver: true}, clockAt(now.Add(-time.Minute))); err != nil || res.Act != claim.ActTakenOver {
		t.Fatalf("P's other copy takes it over: %+v %v", res, err)
	}
	r := f.fenced(t)
	if want := "the reservation on itd-10 moved to your copy on another computer, so this lane does not arm its merge here"; r.Reason != want {
		t.Fatalf("the fence's reason = %q; want %q", r.Reason, want)
	}
}

// TestTheFenceRefusesAReleasedOrUnreadableReservation: a reservation released
// (a tombstone) and one origin cannot be read from both stop the arming, each
// saying so.
func TestTheFenceRefusesAReleasedOrUnreadableReservation(t *testing.T) {
	t.Run("released", func(t *testing.T) {
		f := newLandFixture(t, queueRuleset("MERGE"))
		f.toArm(t)
		now := time.Now().UTC().Truncate(time.Second)
		reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-30*time.Minute), "internal/widget")
		if _, err := claim.Reserve(f.repo.Root(), claim.Request{Record: "itd-10", Release: true}, clockAt(now.Add(-time.Minute))); err != nil {
			t.Fatal(err)
		}
		r := f.fenced(t)
		want := "the reservation on itd-10 was released at " + now.Add(-time.Minute).Format(time.RFC3339) + ", so this lane does not arm its merge"
		if r.Reason != want || !strings.Contains(r.Remedy, "abcd build itd-10") {
			t.Fatalf("the fence says it was released: %+v; want %q", r, want)
		}
	})
	t.Run("fetch failed", func(t *testing.T) {
		f := newLandFixture(t, queueRuleset("MERGE"))
		f.toArm(t)
		now := time.Now().UTC().Truncate(time.Second)
		reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-30*time.Minute), "internal/widget")
		if err := os.Rename(f.bare, f.bare+".gone"); err != nil {
			t.Fatal(err)
		}
		r := f.fenced(t)
		if !strings.HasPrefix(r.Reason, "could not read the reservation on itd-10 from origin: ") ||
			!strings.HasSuffix(r.Reason, ", so the merge is not armed") || r.Remedy != "run abcd implement step again once origin answers" {
			t.Fatalf("the fence names the failed fetch: %+v", r)
		}
	})
}

// queueRulesetTimed is queueRuleset("MERGE") whose merge queue waits minutes
// for its checks.
func queueRulesetTimed(minutes string) string {
	return strings.Replace(queueRuleset("MERGE"), `"grouping_strategy":"ALLGREEN"`,
		`"grouping_strategy":"ALLGREEN","check_response_timeout_minutes":`+minutes, 1)
}

// TestTheFenceRenewsPastTheQueueTimeout is C43: when the reservation holds,
// the fence renews it with max(the default lease, the queue's check timeout
// plus fifteen minutes), clamped to a day, then arms the merge.
func TestTheFenceRenewsPastTheQueueTimeout(t *testing.T) {
	for _, tc := range []struct {
		minutes string
		lease   time.Duration
	}{
		{"45", 2 * time.Hour},
		{"150", 165 * time.Minute},
		{"3000", 24 * time.Hour},
	} {
		t.Run(tc.minutes, func(t *testing.T) {
			f := newLandFixture(t, queueRulesetTimed(tc.minutes))
			f.toArm(t)
			now := time.Now().UTC().Truncate(time.Second)
			reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-30*time.Minute), "internal/widget")
			before := originReservation(t, f.bare, reservedRef)
			f.step(t)
			after := originReservation(t, f.bare, reservedRef)
			if after.parent != before.commit || after.expires.Sub(after.claimed) != tc.lease || after.claimed.Before(now) {
				t.Fatalf("the fence renews with a %s lease: before %+v, after %+v", tc.lease, before, after)
			}
			if l := currentLane(t, f.repo, f.runID); l.Landing == nil || !l.Landing.Armed {
				t.Fatalf("the merge is armed once the fence holds: %+v", l.Landing)
			}
		})
	}
}

// TestTheFenceRenewsWithTheDefaultLeaseWithoutAMirror is C44: with no ruleset
// mirror the fence still runs, renewing with the default lease, and the pull
// request is left open for a person.
func TestTheFenceRenewsWithTheDefaultLeaseWithoutAMirror(t *testing.T) {
	f := newLandFixture(t, "")
	f.toArm(t)
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-30*time.Minute), "internal/widget")
	before := originReservation(t, f.bare, reservedRef)
	f.step(t)
	after := originReservation(t, f.bare, reservedRef)
	if after.parent != before.commit || after.expires.Sub(after.claimed) != 2*time.Hour {
		t.Fatalf("the fence renews with the default lease: before %+v, after %+v", before, after)
	}
	if l := currentLane(t, f.repo, f.runID); l.Landing == nil || l.Landing.Armed || !strings.HasPrefix(l.Landing.Merge, "left open") {
		t.Fatalf("the pull request is left open: %+v", l.Landing)
	}
}

// TestARunHoldingNoReservationIsNeitherRenewedNorFenced: a run whose state
// records no reservation (one started before reservations, or in a checkout
// with no remote named origin) arms as it did, and nothing is pushed under
// the reservation prefix.
func TestARunHoldingNoReservationIsNeitherRenewedNorFenced(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.landedToArmed(t)
	if l.Landing == nil || !l.Landing.Armed {
		t.Fatalf("the merge is armed: %+v", l.Landing)
	}
	if got := originReservation(t, f.bare, reservedRef); got.commit != "" {
		t.Fatalf("a run holding no reservation pushed one: %+v", got)
	}
}

// TestAVersion10StateCarryingAReservationIsRefused: version 10 never wrote a
// run's reservation, so a file of that version carrying `reserved_at` is
// refused, and one without it is read and written back at the current
// version, as a run that holds no reservation.
func TestAVersion10StateCarryingAReservationIsRefused(t *testing.T) {
	root, id, _, _ := startToImplement(t)
	path := filepath.Join(root, filepath.FromSlash(StateRelPath(id)))
	current := stateBytes(t, root, id)
	cur := fmt.Sprintf(`"schema_version": %d,`, SchemaVersion)
	old := strings.Replace(string(current), cur, fmt.Sprintf(`"schema_version": %d,`, schemaVersionUnreserved), 1)
	if old == string(current) || schemaVersionUnreserved != 10 {
		t.Fatalf("the current version is past 10: %s", cur)
	}
	if err := os.WriteFile(path, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	if st, err := ReadState(root, id); err != nil || st.SchemaVersion != SchemaVersion || st.ReservedAt != nil {
		t.Fatalf("a version-10 file is read as the current version, holding no reservation: %+v %v", st.ReservedAt, err)
	}
	carrying := strings.Replace(old, `"record": [`, `"reserved_at": "2026-10-14T12:00:00Z",
  "record": [`, 1)
	if err := os.WriteFile(path, []byte(carrying), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadState(root, id); err == nil || !strings.Contains(err.Error(), "never wrote") {
		t.Fatalf("a version-10 file carrying a reservation must be refused: %v", err)
	}
}
