package loop

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gittest"
)

// `build next` and `build <itd-N>` honour another person's reservation
// (spc-2610031753359849 spec 2, step 12): the peers check reads the
// reservations as last fetched, from disk alone, and counts another copy's
// as a holder, naming it with the view's age; this copy's own is never one.

// onOrigin puts repo on a local bare origin, as P's checkout, and returns the
// bare repository's path.
func onOrigin(t *testing.T, repo *gittest.Repo) string {
	t.Helper()
	bare := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, bare)
	repo.Git("remote", "add", "origin", bare)
	repo.Git("push", "-q", "origin", "main")
	repo.Git("fetch", "-q", "origin")
	asPat(t, repo)
	return bare
}

// fetchedBy fetches the reservations into root, as `abcd peers` does, so its
// view holds what origin holds now.
func fetchedBy(t *testing.T, root string) {
	t.Helper()
	if _, err := claim.Fetch(root, claim.Options{}); err != nil {
		t.Fatalf("fetching the reservations: %v", err)
	}
}

// TestBuildNextExcludesAnotherPersonsReservationWithTheViewsAge is C38: an
// intent another person reserved, in the view as last fetched, is not a
// candidate; `build next` lists it under excluded at the peers check, naming
// the holder, the expiry and the view's age, and picks among the rest.
func TestBuildNextExcludesAnotherPersonsReservationWithTheViewsAge(t *testing.T) {
	repo := pickRepo(t, map[string][2]string{
		"10": {pickIntent("10", "", settledQuestions, gwt), pickSpec("10", fpSmall)},
		"11": {pickIntent("11", "", settledQuestions, gwt), pickSpec("11", fpSmall)},
	})
	bare := onOrigin(t, repo)
	q := otherCopy(t, bare, quinnEmail)
	if _, err := claim.Reserve(q, claim.Request{Record: "itd-10", Footprint: []string{"internal/core/intent"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	fetchedBy(t, repo.Root())

	set, err := candidates(repo.Root(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Candidates) != 1 || set.Candidates[0].ID != "itd-11" {
		t.Fatalf("only the intent nobody else reserved is a candidate: %+v", set.Candidates)
	}
	i := slices.IndexFunc(set.Excluded, func(e Excluded) bool { return e.ID == "itd-10" })
	if i < 0 {
		t.Fatalf("Q's intent is excluded: %+v", set.Excluded)
	}
	e := set.Excluded[i]
	if e.Check != CheckPeers || !strings.Contains(e.Reason, "reserved by quinn until ") ||
		!strings.Contains(e.Reason, "as of the last fetch under a minute ago") {
		t.Fatalf("the exclusion names the holder, the expiry and the view's age at the peers check: %+v", e)
	}
}

// TestBuildRefusesAnotherPersonsReservationAtThePeersCheck is C39: `build
// <itd-N>` over an intent another person reserved, as last fetched, is
// refused at the peers check as contention, naming the holder, and writes
// nothing.
func TestBuildRefusesAnotherPersonsReservationAtThePeersCheck(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	bare := onOrigin(t, repo)
	q := otherCopy(t, bare, quinnEmail)
	if _, err := claim.Reserve(q, claim.Request{Record: "itd-10", Footprint: []string{"internal/widget"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	fetchedBy(t, repo.Root())
	held := gitIn(t, bare, "for-each-ref", "--format=%(objectname)", reservedRef)

	_, err := Start(repo.Root(), "itd-10", Options{})
	r := mustRefusal(t, err)
	if r.Stage != "check" || r.Check != CheckPeers || !r.Contention {
		t.Fatalf("want the peers check as contention: %+v", r)
	}
	if !strings.Contains(r.Reason, "reserved by quinn until ") || !strings.Contains(r.Reason, "as of the last fetch") {
		t.Fatalf("the refusal names Q's reservation and the view's age: %q", r.Reason)
	}
	if strings.Contains(r.Reason, "take-over") || strings.Contains(r.Remedy, "take-over") {
		t.Fatalf("another person's reservation never suggests --take-over: %+v", r)
	}
	runTierAbsent(t, repo.Root())
	if got := gitIn(t, bare, "for-each-ref", "--format=%(objectname)", reservedRef); got != held {
		t.Fatalf("Q's reservation moved: %s", got)
	}
}

// TestThePeersCheckNamesTakeOverForYourOtherCopy: once the view is fetched,
// `build <itd-N>` over this person's live reservation from another computer
// is refused at the peers check, before the start's own reservation could
// say so, so the peers check names --take-over as the start would.
func TestThePeersCheckNamesTakeOverForYourOtherCopy(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), footprintedSpec)
	bare := onOrigin(t, repo)
	p2 := otherCopy(t, bare, patEmail)
	if _, err := claim.Reserve(p2, claim.Request{Record: "itd-10", Footprint: []string{"internal/widget"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	fetchedBy(t, repo.Root())
	_, err := Start(repo.Root(), "itd-10", Options{})
	r := mustRefusal(t, err)
	if r.Check != CheckPeers || !r.Contention {
		t.Fatalf("want the peers check as contention: %+v", r)
	}
	if !strings.Contains(r.Reason, "your copy on another computer") || !strings.Contains(r.Reason, "pass --take-over to continue here") {
		t.Fatalf("the refusal names your other copy and --take-over: %q", r.Reason)
	}
	if !strings.Contains(r.Remedy, "--take-over") {
		t.Fatalf("the remedy names --take-over: %q", r.Remedy)
	}
	runTierAbsent(t, repo.Root())
}

// TestTheRunsOwnReservationIsNeverAHolder: this copy's own reservation on the
// intent is not a peer's, so the peers check passes over it; this person's
// other copy is a holder, named as theirs on another computer.
func TestTheRunsOwnReservationIsNeverAHolder(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	bare := onOrigin(t, repo)
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: "itd-10", Footprint: []string{"internal/widget"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	chk, err := Check(repo.Root(), "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	if row := chk.Checks[len(chk.Checks)-1]; row.Name != CheckPeers || !row.OK {
		t.Fatalf("this copy's own reservation is no peer's: %+v", row)
	}

	// P's second computer takes it over once it lapses; P's view then
	// names it as P's other copy.
	p2 := otherCopy(t, bare, patEmail)
	if _, err := claim.Reserve(p2, claim.Request{Record: "itd-10", Footprint: []string{"internal/widget"}, TakeOver: true}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	fetchedBy(t, repo.Root())
	chk, err = Check(repo.Root(), "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	row := chk.Checks[len(chk.Checks)-1]
	if row.Name != CheckPeers || row.OK || !strings.Contains(row.Detail, "your copy on another computer") {
		t.Fatalf("P's other copy holds it, named as P's: %+v", row)
	}
}

// `abcd build` reserves, releases and takes over (spc-2610031753359849 spec
// 2, step 13): a new run, and a resumed one, reserves its key at the start,
// after the checks and before the lane opens, with the footprint its spec's
// `## Footprint` packages line names (an issue's found_at path for an issue
// run); `--take-over` passes through; a run whose issue lane is handed back
// releases its reservation (iss-2610142247197128).

// footprintedSpec is spc-1 with a footprint naming two packages.
var footprintedSpec = specWithSteps("") + "\n## Footprint\n\n- packages: internal/widget, commands/\n- tests: the widget\n"

// TestBuildReservesTheIntentWithItsSpecFootprint is C34: a new run reserves
// its intent on origin, as this person from this copy, with the packages its
// spec's footprint names, records it in its state and its record, and says so;
// a resumed run reserves again, renewing its own.
func TestBuildReservesTheIntentWithItsSpecFootprint(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), footprintedSpec)
	bare := onOrigin(t, repo)

	res, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if fp := footprintOf(t, bare, "itd-10"); !slices.Equal(fp, []string{"internal/widget", "commands"}) {
		t.Fatalf("the reservation's footprint is the spec's packages: %v", fp)
	}
	tip := originReservation(t, bare, reservedRef)
	if tip.holder != "octo" || tip.released {
		t.Fatalf("the reservation is P's: %+v", tip)
	}
	st, err := ReadState(repo.Root(), res.RunID)
	if err != nil || st.ReservedAt == nil {
		t.Fatalf("the run records its reservation: %+v %v", st.ReservedAt, err)
	}
	if note := recordNote(t, repo, res.RunID, StageReservation); !strings.Contains(note, "reserved itd-10 on origin") {
		t.Fatalf("the run record says what it reserved: %q", note)
	}
	if !strings.Contains(res.Reservation, "reserved itd-10 on origin") {
		t.Fatalf("the start names the reservation: %q", res.Reservation)
	}

	again, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil || !again.Resumed || again.RunID != res.RunID {
		t.Fatalf("starting again resumes the run: %+v %v", again, err)
	}
	if after := originReservation(t, bare, reservedRef); after.parent != tip.commit || after.released {
		t.Fatalf("the resumed run renews its reservation as the tip's child: before %+v, after %+v", tip, after)
	}
	if !strings.Contains(again.Reservation, "renewed the reservation on itd-10") {
		t.Fatalf("the resumed start names the renewal: %q", again.Reservation)
	}
}

// TestAResumedRunWithoutAReservationReservesIt: a run started where nothing
// was reserved (before reservations, or before its checkout had an origin)
// reserves its key when it is resumed, and records it.
func TestAResumedRunWithoutAReservationReservesIt(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), footprintedSpec)
	res, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if st, _ := ReadState(repo.Root(), res.RunID); st.ReservedAt != nil {
		t.Fatalf("with no origin nothing is reserved: %+v", st.ReservedAt)
	}
	bare := onOrigin(t, repo)
	before, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(res.RunID))))
	if err != nil {
		t.Fatal(err)
	}
	again, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil || !again.Resumed {
		t.Fatalf("starting again resumes the run: %+v %v", again, err)
	}
	if tip := originReservation(t, bare, reservedRef); tip.holder != "octo" || tip.parent != "" {
		t.Fatalf("the resumed run reserves its key: %+v", tip)
	}
	st, err := ReadState(repo.Root(), res.RunID)
	if err != nil || st.ReservedAt == nil {
		t.Fatalf("the resumed run records its reservation: %+v %v", st.ReservedAt, err)
	}
	after, _ := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(res.RunID))))
	if string(after) == string(before) {
		t.Fatal("the state was not written")
	}
}

// TestBuildWithoutAFootprintIsRefusedAtTheReservation is C35: a spec with no
// footprint refuses the start at the reservation, naming the spec and the
// missing section, starts nothing and pushes nothing.
func TestBuildWithoutAFootprintIsRefusedAtTheReservation(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	bare := onOrigin(t, repo)
	_, err := Start(repo.Root(), "itd-10", Options{})
	r := mustRefusal(t, err)
	want := "spc-1 has no footprint under ## Footprint to reserve itd-10 with; fill its packages line, then run abcd build again; nothing was started"
	if r.Stage != StageReservation || r.Contention || r.Reason != want {
		t.Fatalf("want the reservation refusal %q, got %+v", want, r)
	}
	runTierAbsent(t, repo.Root())
	if tip := originReservation(t, bare, reservedRef); tip.commit != "" {
		t.Fatalf("nothing is pushed: %+v", tip)
	}
}

// TestBuildTakeOverPassesThrough: a build over this person's live
// reservation from another computer backs off naming --take-over; with
// TakeOver the start takes it over to this copy and the record says so.
func TestBuildTakeOverPassesThrough(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), footprintedSpec)
	bare := onOrigin(t, repo)
	p2 := otherCopy(t, bare, patEmail)
	if _, err := claim.Reserve(p2, claim.Request{Record: "itd-10", Footprint: []string{"internal/widget"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	held := originReservation(t, bare, reservedRef)

	_, err := Start(repo.Root(), "itd-10", Options{})
	r := mustRefusal(t, err)
	if !r.Contention || !strings.Contains(r.Reason, "pass --take-over to continue here") {
		t.Fatalf("the build backs off naming --take-over: %+v", r)
	}
	runTierAbsent(t, repo.Root())

	res, err := Start(repo.Root(), "itd-10", Options{TakeOver: true})
	if err != nil {
		t.Fatalf("the take-over: %v", err)
	}
	if tip := originReservation(t, bare, reservedRef); tip.parent != held.commit {
		t.Fatalf("the take-over is the tip's child: %+v over %+v", tip, held)
	}
	if note := recordNote(t, repo, res.RunID, StageReservation); !strings.Contains(note, "took over the reservation on itd-10") {
		t.Fatalf("the run record says it took the reservation over: %q", note)
	}
}

// TestBuildNextReservesThePickedIntent: `build next` reaches the same start,
// so the picked intent is reserved with its spec's footprint.
func TestBuildNextReservesThePickedIntent(t *testing.T) {
	repo := pickRepo(t, map[string][2]string{
		"10": {pickIntent("10", "", settledQuestions, gwt), pickSpec("10", fpSmall)},
	})
	bare := onOrigin(t, repo)
	res, err := Next(repo.Root(), Options{}, NextOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Pick == nil || res.Pick.Chosen.ID != "itd-10" {
		t.Fatalf("the pick takes itd-10: %+v", res.Pick)
	}
	if fp := footprintOf(t, bare, "itd-10"); !slices.Equal(fp, []string{"internal/core/intent"}) {
		t.Fatalf("the picked intent is reserved with its spec's footprint: %v", fp)
	}
}

// TestBuildReservesAnIssueWithItsFoundAt: `build <iss-N>` reserves the issue
// with the path its found_at names, as the drain does; one with no found_at
// path is refused at the reservation, starting nothing.
func TestBuildReservesAnIssueWithItsFoundAt(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	if _, err := Start(repo.Root(), secondIssue, Options{}); err != nil {
		t.Fatal(err)
	}
	if fp := footprintOf(t, bare, secondIssue); !slices.Equal(fp, []string{"commands/drain.md"}) {
		t.Fatalf("the issue is reserved with its found_at path: %v", fp)
	}

	repo, bare = reservingDrainRepo(t, "")
	_, err := Start(repo.Root(), secondIssue, Options{})
	r := mustRefusal(t, err)
	if r.Stage != StageReservation || !strings.Contains(r.Reason, secondIssue+" has no found_at path to reserve it with") {
		t.Fatalf("an issue with no found_at is refused at the reservation: %+v", r)
	}
	runTierAbsent(t, repo.Root())
	if fp := footprintOf(t, bare, secondIssue); fp != nil {
		t.Fatalf("nothing is reserved: %v", fp)
	}
}

// TestAHandedBackBuildLaneReleasesItsReservation is iss-2610142247197128: a
// run `build <iss-N>` started, which no drain routes, releases its
// reservation when its lane is handed back, and its record says so.
func TestAHandedBackBuildLaneReleasesItsReservation(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	res, err := Start(repo.Root(), secondIssue, Options{})
	if err != nil {
		t.Fatal(err)
	}
	handBackLaneOf(t, repo, res.RunID, Options{}, LaneHandBack{Kind: HandBackDesignFinding, Reason: "the page needs a decision", Home: "the brief"})
	if tomb := originReservation(t, bare, claim.RefPrefix+secondIssue); !tomb.released || tomb.holder != "octo" {
		t.Fatalf("the handed-back issue's reservation is released: %+v", tomb)
	}
	if note := recordNote(t, repo, res.RunID, StageReservationReleased); !strings.Contains(note, "released the reservation on "+secondIssue) {
		t.Fatalf("the run record says the reservation was released: %q", note)
	}
}

// TestAResumedIssueRunReadsFoundAtWhereverItsIssueIs: a resume is not
// re-judged, since the run's own lane resolves its issue, so `build <iss-N>`
// over a run whose issue has left open/ reads the found_at from the folder it
// is in, and reserves with it, rather than calling it a record with none.
func TestAResumedIssueRunReadsFoundAtWhereverItsIssueIs(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	res, err := Start(repo.Root(), secondIssue, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := capture.Resolve(capture.ResolveRequest{RepoRoot: repo.Root(), ID: secondIssue, Resolution: "the page names the flag",
		Impact: "fix", Grounds: "pursued: the page names the flag; shown wrong if a reader cannot find it"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repo.Root(), ".abcd", "work", "issues", "open", secondIssue+".md")); err == nil {
		t.Fatal("the issue is still open")
	}
	held := originReservation(t, bare, claim.RefPrefix+secondIssue)
	again, err := Start(repo.Root(), secondIssue, Options{})
	if err != nil || !again.Resumed || again.RunID != res.RunID {
		t.Fatalf("the run whose issue left open/ resumes: %+v %v", again, err)
	}
	if tip := originReservation(t, bare, claim.RefPrefix+secondIssue); tip.parent != held.commit || tip.released {
		t.Fatalf("the resume renews the issue's reservation: before %+v, after %+v", held, tip)
	}
	if fp := footprintOf(t, bare, secondIssue); !slices.Equal(fp, []string{"commands/drain.md"}) {
		t.Fatalf("the footprint is the resolved issue's found_at path: %v", fp)
	}
}

// pushRacedReservation pushes holder's live reservation on id straight onto
// origin, footprint fp, as a race between two people lets one in past the
// overlap check: each reservation is its own ref, so the overlap is not
// compare-and-swapped.
func pushRacedReservation(t *testing.T, bare, id, holder string, fp ...string) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	msg := "abcd reservation: " + id + "\n\nAbcd-Reservation: " + id + "\nAbcd-Holder: " + holder +
		"\nAbcd-Copy: 9f2c41d07be35a6c80e1f4b2a7d39c55\n"
	for _, p := range fp {
		msg += "Abcd-Footprint: " + p + "\n"
	}
	msg += "Abcd-Claimed-At: " + stamp(now) + "\nAbcd-Expires-At: " + stamp(now.Add(2*time.Hour)) + "\n"
	dir := filepath.Join(t.TempDir(), "raced")
	gitIn(t, "", "clone", "-q", bare, dir)
	path := filepath.Join(t.TempDir(), "msg")
	if err := os.WriteFile(path, []byte(msg), 0o600); err != nil {
		t.Fatal(err)
	}
	tree := gitIn(t, dir, "hash-object", "-w", "-t", "tree", "/dev/null")
	sha := gitIn(t, dir, "-c", "user.name=Quinn", "-c", "user.email=quinn@example.invalid", "-c", "commit.gpgsign=false",
		"commit-tree", tree, "-F", path)
	gitIn(t, dir, "push", "-q", "origin", sha+":"+claim.RefPrefix+id)
}

// TestAResumeRenewsItsOwnReservationPastAnOverlapARaceLetIn: a resume over
// this copy's own live reservation is a renewal, keeping the tip's footprint
// and term, so it is not judged for overlap; another person's reservation a
// race let in over the same paths would otherwise refuse `abcd build` while
// `implement step` keeps renewing the very same reservation.
func TestAResumeRenewsItsOwnReservationPastAnOverlapARaceLetIn(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), footprintedSpec)
	bare := onOrigin(t, repo)
	res, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	tip := originReservation(t, bare, reservedRef)
	pushRacedReservation(t, bare, "itd-11", "quinn", "internal/widget")

	again, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil || !again.Resumed || again.RunID != res.RunID {
		t.Fatalf("the resume renews its own reservation past the raced overlap: %+v %v", again, err)
	}
	after := originReservation(t, bare, reservedRef)
	if after.parent != tip.commit || after.released || after.expires.Sub(after.claimed) != tip.expires.Sub(tip.claimed) {
		t.Fatalf("the resume renews the tip keeping its term: before %+v, after %+v", tip, after)
	}
	if !strings.Contains(again.Reservation, "renewed the reservation on itd-10") {
		t.Fatalf("the resumed start names the renewal: %q", again.Reservation)
	}

	// A view gone stale: P's other computer took the reservation over, and
	// this copy has not fetched since. The renewal finds it moved, and the
	// resume falls back to the reserve any other resume makes, which backs
	// off naming --take-over, as the start does.
	gitIn(t, bare, "update-ref", "-d", claim.RefPrefix+"itd-11")
	p2 := otherCopy(t, bare, patEmail)
	if _, err := claim.Reserve(p2, claim.Request{Record: "itd-10", Footprint: []string{"internal/widget"}, TakeOver: true}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	_, err = Start(repo.Root(), "itd-10", Options{})
	if r := mustRefusal(t, err); !r.Contention || !strings.Contains(r.Reason, "pass --take-over to continue here") {
		t.Fatalf("a stale view's renewal falls back to the reserve, naming --take-over: %+v", r)
	}
}

// TestBuildNextExcludesAnIntentWithNoFootprintToReserve: in a checkout with
// a remote named origin a start reserves its intent with its spec's
// footprint packages, so an intent whose spec names none would be picked
// only to be refused at the reservation, aborting a series and re-picked on
// every rerun. It is excluded at the check `footprint`, named in the dry
// view, and the pick moves to the next candidate.
func TestBuildNextExcludesAnIntentWithNoFootprintToReserve(t *testing.T) {
	repo := pickRepo(t, map[string][2]string{
		// The readier one by score: a test path, but no packages.
		"10": {pickIntent("10", "", settledQuestions, gwt), pickSpec("10", "- tests: the widget\n")},
		"11": {pickIntent("11", "", settledQuestions, "- it works\n"), pickSpec("11", "- packages: a, b, c, d\n- tests: the widget\n")},
	})
	bare := onOrigin(t, repo)
	set, err := candidates(repo.Root(), "")
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(set.Excluded, func(e Excluded) bool { return e.ID == "itd-10" })
	if i < 0 || set.Excluded[i].Check != CheckFootprint || !strings.Contains(set.Excluded[i].Reason, "spc-10 has no footprint under ## Footprint") {
		t.Fatalf("the intent with no footprint is excluded at the footprint check: %+v", set.Excluded)
	}
	if len(set.Candidates) != 1 || set.Candidates[0].ID != "itd-11" {
		t.Fatalf("only the footprinted intent is a candidate: %+v", set.Candidates)
	}
	res, err := Next(repo.Root(), Options{}, NextOptions{})
	if err != nil {
		t.Fatalf("the pick moves on to the next candidate: %v", err)
	}
	if res.Pick == nil || res.Pick.Chosen.ID != "itd-11" {
		t.Fatalf("the pick takes itd-11: %+v", res.Pick)
	}
	if fp := footprintOf(t, bare, "itd-11"); !slices.Equal(fp, []string{"a", "b", "c", "d"}) {
		t.Fatalf("the picked intent is reserved: %v", fp)
	}
}
