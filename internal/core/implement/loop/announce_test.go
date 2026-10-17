package loop

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// The announcement before a lane's push (iss-2610161645224876,
// dcn-2610170655202547): before the landing pushes a lane's branch it posts
// "merging <the lane's record>" through claim.AnnounceBeforePush, the code
// `abcd announce` runs, and pushes only once it is on origin; one that cannot
// be published refuses the push (dcn-2610170704189712).

// noAnnouncement stands in for the announcer in the package's tests, whose
// fixtures land with no person's username to post as; TestMain sets it, and a
// test that proves the announcement takes the real one (realAnnouncer).
func noAnnouncement(string, []string, claim.Options) ([]claim.PushNotice, error) { return nil, nil }

// realAnnouncer puts the real announcer back for one test, wrapped by after,
// which runs once each call returns.
func realAnnouncer(t *testing.T, after func(n []claim.PushNotice, err error)) {
	t.Helper()
	prev := announceBeforePush
	announceBeforePush = func(root string, items []string, o claim.Options) ([]claim.PushNotice, error) {
		n, err := claim.AnnounceBeforePush(root, items, o)
		if after != nil {
			after(n, err)
		}
		return n, err
	}
	t.Cleanup(func() { announceBeforePush = prev })
}

// asOcto gives the run's checkout a person's git identity whose no-reply
// address names the GitHub user octo, so an announcement is sent as octo.
func (f *landFixture) asOcto() {
	f.repo.Git("config", "user.name", "Pat Example")
	f.repo.Git("config", "user.email", "1234+octo@users.noreply.github.com")
}

// colleague clones the fixture's origin as another person's working copy,
// with its local tier.
func (f *landFixture) colleague(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "quinn")
	f.repo.Git("clone", "-q", f.bare, dir)
	if err := os.MkdirAll(filepath.Join(dir, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// statusRead is what `abcd status` run in dir shows now: its fetch of the
// status branches, then its read of them from disk.
func statusRead(t *testing.T, dir string) []claim.Announcement {
	t.Helper()
	o := claim.Options{Home: t.TempDir()}
	if _, err := claim.FetchStatus(dir, nil, o); err != nil {
		t.Fatalf("the colleague's status fetch: %v", err)
	}
	v, err := claim.ReadStatus(context.Background(), dir, time.Now(), o)
	if err != nil {
		t.Fatalf("the colleague's status read: %v", err)
	}
	return v.Announcements
}

// TestALaneIsAnnouncedBeforeItsBranchReachesOrigin is the messaging promise
// at a lane's landing: when the announcement is on origin, a colleague's
// status read shows "merging itd-10" from the person, and origin does not yet
// hold the lane's branch; the push follows, and the step's record says what
// was announced.
func TestALaneIsAnnouncedBeforeItsBranchReachesOrigin(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.asOcto()
	quinn := f.colleague(t)
	var branch, branchThen string
	var seen []claim.Announcement
	calls := 0
	realAnnouncer(t, func(_ []claim.PushNotice, err error) {
		calls++
		if err == nil {
			branchThen, seen = f.remoteBranch(t, branch), statusRead(t, quinn)
		}
	})
	l := f.validated(t)
	branch = l.Branch
	for range 2 {
		f.step(t)
	}
	l = currentLane(t, f.repo, f.runID)
	preflighted(t, l, l.HeadSHA)
	f.step(t)

	if calls != 1 || branchThen != "" {
		t.Fatalf("announcements %d, the lane's branch on origin once announced %q: want one, before the branch was there", calls, branchThen)
	}
	var found bool
	for _, a := range seen {
		found = found || (a.Branch == "octo" && a.Kind == "merging" && a.Item == "itd-10")
	}
	if !found {
		t.Fatalf("the colleague's read once announced: %+v; want octo's merging itd-10", seen)
	}
	if got := f.remoteBranch(t, l.Branch); got != l.HeadSHA {
		t.Fatalf("the lane's head is pushed after the announcement: remote %q, head %s", got, l.HeadSHA)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(recordText(st), "on its preflight receipt; announced merging itd-10 on abcd-status/octo, standing until ") {
		t.Fatalf("the push's record names the announcement:\n%s", recordText(st))
	}
}

// TestALaneWhoseAnnouncementCannotBePublishedIsNotPushed: with no username to
// post as (the forge client answers nothing), the landing refuses the push
// with the reason and pushes nothing; once the person's username can be read,
// the next step announces and pushes.
func TestALaneWhoseAnnouncementCannotBePublishedIsNotPushed(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	realAnnouncer(t, nil)
	l := f.validated(t)
	for range 2 {
		f.step(t)
	}
	l = currentLane(t, f.repo, f.runID)
	preflighted(t, l, l.HeadSHA)
	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	r := mustRefusal(t, err)
	if r.Stage != string(StageLand) || !strings.HasPrefix(r.Reason, "the announcement merging itd-10 could not be published: ") ||
		!strings.HasSuffix(r.Reason, ", so "+l.Branch+" was not pushed: the landing announces before it pushes") ||
		r.Remedy != "settle what it names, then run `abcd implement step` again; nothing was pushed" {
		t.Fatalf("refusal %+v; want the announcement's failure, and nothing pushed", r)
	}
	if got := f.remoteBranch(t, l.Branch); got != "" {
		t.Fatalf("a lane whose announcement failed was pushed: %s", got)
	}
	if got := f.remoteBranch(t, "abcd-status/octo"); got != "" {
		t.Fatalf("an announcement reached origin: %s", got)
	}

	f.asOcto()
	f.step(t)
	if f.remoteBranch(t, "abcd-status/octo") == "" || f.remoteBranch(t, l.Branch) != l.HeadSHA {
		t.Fatalf("the next step announces, then pushes: status %q, lane %q", f.remoteBranch(t, "abcd-status/octo"), f.remoteBranch(t, l.Branch))
	}
}
