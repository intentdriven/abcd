package landing

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The announcement before the push (iss-2610161645224876, dcn-2610170655202547):
// before abcd land pushes a branch, it posts "merging <item>" for each issue,
// intent or pull request the branch's own commits name in a Refs:, Resolves:
// or Delivers: trailer, through the code `abcd announce` runs, and pushes only
// once each announcement is on origin. A colleague's status read therefore
// shows the announcement before the branch exists there.

// publicRecord is the issue the fixture's default branch carries, so an
// announcement may name it.
const publicRecord = ".abcd/work/issues/open/iss-5-the-widget.md"

// announcing makes the fixture P's checkout with its local tier, puts iss-5 on
// origin's default branch, and adds a commit to feat/x whose message ends in
// trailer (say "Refs: iss-5"), so the branch is work for that item.
func (f *fixture) announcing(trailer string) {
	f.t.Helper()
	f.asPat()
	f.repo.Git("checkout", "-q", "main")
	f.repo.Write(publicRecord, "---\nid: iss-5\n---\n# The widget\n")
	f.repo.Commit("the record")
	f.repo.Git("push", "-q", "origin", "main")
	f.repo.Git("fetch", "-q", "origin")
	f.repo.Git("checkout", "-q", "feat/x")
	f.repo.Write("fix.txt", "the fix\n")
	f.repo.Git("add", "fix.txt")
	f.repo.Git("commit", "-q", "-m", "fix: the widget\n\n"+trailer)
}

// statusTip is P's status branch on the bare origin, "" when it has none:
// the branch is named by P's holder id, and a home that keeps none has sent
// nothing under one.
func (f *fixture) statusTip() string {
	f.t.Helper()
	data, err := os.ReadFile(abcdhome.Path(f.home, "holder-id"))
	if errors.Is(err, os.ErrNotExist) {
		return ""
	}
	if err != nil {
		f.t.Fatal(err)
	}
	return f.onOrigin("abcd-status/" + strings.TrimSpace(string(data)))
}

// colleagueRead is what a colleague's `abcd status` shows at this moment, run
// from the working copy dir: it fetches the status branches as that verb does
// and reads them back from disk.
func (f *fixture) colleagueRead(dir string) []claim.Announcement {
	f.t.Helper()
	o := claim.Options{Home: f.t.TempDir()}
	if _, err := claim.FetchStatus(dir, nil, o); err != nil {
		f.t.Fatalf("the colleague's status fetch: %v", err)
	}
	v, err := claim.ReadStatus(context.Background(), dir, time.Now(), o)
	if err != nil {
		f.t.Fatalf("the colleague's status read: %v", err)
	}
	return v.Announcements
}

// TestAColleagueSeesTheAnnouncementBeforeTheBranchIsOnOrigin is the messaging
// promise at abcd land: at the moment land's push starts, a colleague's status
// read already shows P's "merging iss-5", and origin does not hold feat/x yet.
func TestAColleagueSeesTheAnnouncementBeforeTheBranchIsOnOrigin(t *testing.T) {
	f := newFixture(t, "")
	f.announcing("Refs: iss-5")
	quinn := f.copyOf(quinnEmail)
	var seen []claim.Announcement
	branchThen, pushes := "unread", 0
	f.beforeGit = func(args []string) {
		if len(args) == 0 || args[0] != "push" {
			return
		}
		pushes++
		branchThen = f.onOrigin("feat/x")
		seen = f.colleagueRead(quinn)
	}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatalf("land: %v", err)
	}
	if pushes != 1 || branchThen != "" {
		t.Fatalf("land's pushes %d, feat/x on origin as the push began %q: want one push of a branch origin did not hold", pushes, branchThen)
	}
	var found bool
	for _, a := range seen {
		if a.Branch == f.heldBy(f.repo.Root()) && a.Kind == "merging" && a.Item == "iss-5" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the colleague's read as land's push began: %+v; want P's merging iss-5 under P's holder id", seen)
	}
	if f.onOrigin("feat/x") != f.head() {
		t.Fatalf("feat/x on origin is %q; want the head %s pushed after the announcement", f.onOrigin("feat/x"), f.head())
	}
	s := step(t, res, "announce")
	if s.Status != StatusDone || !strings.Contains(s.Note, "announced merging iss-5 on abcd-status/"+f.heldBy(f.repo.Root())) {
		t.Fatalf("announce step %+v; want it done, naming merging iss-5 on P's holder id's status branch", s)
	}
}

// statusCommits counts the commits on P's status branch on origin.
func (f *fixture) statusCommits() int {
	f.t.Helper()
	tip := f.statusTip()
	if tip == "" {
		return 0
	}
	out, err := gitutil.Run(f.bare, "rev-list", "--count", tip)
	if err != nil {
		f.t.Fatal(err)
	}
	n, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		f.t.Fatal(err)
	}
	return n
}

// TestLandPushesNothingWhenItsAnnouncementCannotBePublished: an embargo store
// that cannot be checked refuses the announcement, so land refuses its push
// with the reason, and neither the branch nor an announcement reaches origin
// (dcn-2610170704189712).
func TestLandPushesNothingWhenItsAnnouncementCannotBePublished(t *testing.T) {
	f := newFixture(t, "")
	f.announcing("Resolves: iss-5")
	o := f.options()
	o.Announce.Embargo = func(_, _, _ string, _ []string) (bool, error) { return false, errors.New("the store is unreadable") }
	_, err := f.land(o)
	wantRefusal(t, err, "the announcement merging iss-5 could not be published: an announcement about iss-5 is refused: "+
		"the embargo store on this machine could not be checked: the store is unreadable")
	wantRefusal(t, err, "so feat/x was not pushed: abcd land announces before it pushes; settle what it names, then run abcd land again; the branch was not pushed, and an announcement already sent stands")
	if len(f.pushes()) != 0 || f.onOrigin("feat/x") != "" || f.statusTip() != "" {
		t.Fatalf("pushes %v, feat/x %q, status %q: want nothing pushed", f.pushes(), f.onOrigin("feat/x"), f.statusTip())
	}
}

// TestLandRunAgainDoesNotAnnounceTwice: a second land while the first
// announcement stands sends none, and says the one standing.
func TestLandRunAgainDoesNotAnnounceTwice(t *testing.T) {
	f := newFixture(t, "")
	f.announcing("Delivers: iss-5")
	if _, err := f.land(f.options()); err != nil {
		t.Fatal(err)
	}
	if f.statusCommits() != 1 {
		t.Fatalf("status commits after one land = %d; want 1", f.statusCommits())
	}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	s := step(t, res, "announce")
	if f.statusCommits() != 1 || s.Status != StatusSkipped || !strings.Contains(s.Note, "merging iss-5 was already announced on abcd-status/"+f.heldBy(f.repo.Root())) {
		t.Fatalf("status commits %d, announce step %+v; want the standing one named and none sent", f.statusCommits(), s)
	}
}

// TestLandPassesOverARecordNotYetPublic: a record only the branch holds is
// not announced (an announcement names only a public record); land says so
// and still pushes.
func TestLandPassesOverARecordNotYetPublic(t *testing.T) {
	f := newFixture(t, "")
	f.announcing("Resolves: iss-9")
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	s := step(t, res, "announce")
	if s.Status != StatusSkipped || s.Note != "did not announce iss-9: it is not on origin's default branch yet, and an announcement names only a public record" {
		t.Fatalf("announce step %+v", s)
	}
	if f.onOrigin("feat/x") != f.head() || f.statusTip() != "" {
		t.Fatalf("feat/x %q, status %q: want the branch pushed and no announcement", f.onOrigin("feat/x"), f.statusTip())
	}
}

// TestLandWithNoItemAnnouncesNothingAndSaysSo: a branch whose commits name
// no item has nothing to announce; land says so and pushes, asking nothing of
// the status branches.
func TestLandWithNoItemAnnouncesNothingAndSaysSo(t *testing.T) {
	f := newFixture(t, "")
	f.announcing("Refs: spc-2610150728396006")
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	s := step(t, res, "announce")
	if s.Status != StatusSkipped || s.Note != "feat/x's commits name no issue, intent or pull request in a Refs:, Resolves: or Delivers: trailer, so there is nothing to announce" {
		t.Fatalf("announce step %+v", s)
	}
	if f.onOrigin("feat/x") != f.head() || f.statusTip() != "" {
		t.Fatalf("feat/x %q, status %q: want the branch pushed and no announcement", f.onOrigin("feat/x"), f.statusTip())
	}
}

// TestLandReadsABareNumberInATrailerAsTextNotAPullRequest: "Refs: spc-9
// step 3" names a spec and its step, and no pull request #3, so there is
// nothing to announce; only "#N" names a pull request in a trailer.
func TestLandReadsABareNumberInATrailerAsTextNotAPullRequest(t *testing.T) {
	f := newFixture(t, "")
	f.announcing("Refs: spc-2610150728396006 step 3")
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	s := step(t, res, "announce")
	if s.Status != StatusSkipped || strings.Contains(s.Note, "#3") || f.statusTip() != "" {
		t.Fatalf("announce step %+v, status %q; want nothing announced, no #3", s, f.statusTip())
	}
}

// TestLandWithPushOffAnnouncesNothing: with land.push off nothing is pushed,
// so nothing is announced either.
func TestLandWithPushOffAnnouncesNothing(t *testing.T) {
	f := newFixture(t, "")
	f.announcing("Refs: iss-5")
	o := f.options()
	o.NoPush, o.NoEnqueue = true, false
	f.forge.prs = f.prFor(f.head())
	res, _ := f.land(o)
	for _, s := range res.Steps {
		if s.Name == "announce" {
			t.Fatalf("an announce step with push off: %+v", s)
		}
	}
	if f.statusTip() != "" {
		t.Fatalf("an announcement with push off: %s", f.statusTip())
	}
}
