package loop

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/claim"
)

// A drain's hand-backs are published on origin (spc-2610161044461448 step 3;
// iss-2610161015370582; dcn-2610161040365515): the release of a handed-back
// issue's reservation carries its kind and the blob id of its record on
// origin's default branch, so another account's drain passes the issue over
// while that record is unchanged; and a promotion this checkout cannot make,
// not holding the record as origin does, publishes that tombstone alone.

// tombstoneTrailers are the hand-back kind and record blob origin's
// reservation tip on id carries, "" for each it does not.
func tombstoneTrailers(t *testing.T, bare, id string) (kind, blob string) {
	t.Helper()
	sha := refOf(t, bare, id)
	if sha == "" {
		return "", ""
	}
	for _, line := range strings.Split(gitIn(t, bare, "log", "-1", "--format=%B", sha), "\n") {
		if v, ok := strings.CutPrefix(line, "Abcd-Handed-Back: "); ok {
			kind = v
		}
		if v, ok := strings.CutPrefix(line, "Abcd-Record: "); ok {
			blob = v
		}
	}
	return kind, blob
}

// originBlob is the blob of id's open record on origin's main.
func originBlob(t *testing.T, bare, id string) string {
	t.Helper()
	for _, line := range strings.Split(gitIn(t, bare, "ls-tree", "-r", "main", "--", ".abcd/work/issues/open"), "\n") {
		meta, file, ok := strings.Cut(line, "\t")
		if ok && strings.HasPrefix(filepath.Base(file), id+"-") {
			return strings.Fields(meta)[2]
		}
	}
	t.Fatalf("origin's main holds no open record of %s", id)
	return ""
}

// TestEveryKindTheLoopHandsBackIsATombstoneKind: each kind a lane or the
// judgement hands an issue back as, and a lane stopped after its fix rounds,
// is a kind the hand-back tombstone may name.
func TestEveryKindTheLoopHandsBackIsATombstoneKind(t *testing.T) {
	for _, k := range laneHandBackKinds {
		if got := tombstoneKind(HandBack{Kind: k.kind}); got != k.kind || !slices.Contains(claim.HandBackKinds, got) {
			t.Fatalf("a lane's %s hand-back is tombstoned as %q", k.kind, got)
		}
	}
	if got := tombstoneKind(HandBack{Verdict: VerdictUnachievable, Round: 3, FixRounds: 2}); got != claim.HandBackUnachievable {
		t.Fatalf("a lane stopped after its fix rounds is tombstoned as %q", got)
	}
}

// TestOneClonesHandBackIsPassedOverByAnotherClonesDrain: Pat's drain hands
// the documentation issue back on a yes, and the tombstone names the kind and
// origin's blob of the record; Quinn's drain, on another clone, passes it over
// naming the hand-back, pushes nothing over it, and writes no judgement
// request.
func TestOneClonesHandBackIsPassedOverByAnotherClonesDrain(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	next := judge(t, repo, Options{}, first, JudgementYes, HandBackTrustRule, "may the page name a token?")
	if kind, blob := tombstoneTrailers(t, bare, secondIssue); kind != HandBackTrustRule || blob != originBlob(t, bare, secondIssue) {
		t.Fatalf("the yes's tombstone names %q over %q; want trust-rule over origin's record", kind, blob)
	}
	if r := routeOf(t, next.HandBacks, secondIssue); !strings.Contains(r.Wrote, "marking it handed back as trust-rule") {
		t.Fatalf("the route says the hand-back is published: %q", r.Wrote)
	}
	tomb := refOf(t, bare, secondIssue)

	q := otherCopy(t, bare, quinnEmail)
	var res DrainResult
	asHomeOf(t, q, func() { res, err = Drain(q, Options{}, DrainOptions{}) })
	if err != nil {
		t.Fatal(err)
	}
	if p := passedFor(t, res, secondIssue); p.Check != CheckReservation ||
		!strings.Contains(p.Reason, secondIssue+" was handed back by "+heldBy(t, patEmail)+" as trust-rule") || !strings.Contains(p.Reason, "no judgement") {
		t.Fatalf("Quinn's drain passes the hand-back over, naming it: %+v", p)
	}
	// Pat's drain went on to reserve the next issue for its own judgement, so
	// Quinn's passes that over too, and has nothing to ask.
	if p := passedFor(t, res, eligibleIssue); !strings.Contains(p.Reason, eligibleIssue+" is reserved by "+heldBy(t, patEmail)) {
		t.Fatalf("Quinn's drain passes over the issue Pat holds: %+v", p)
	}
	if res.Judging != nil {
		t.Fatalf("Quinn's drain asks a judgement: %+v", res.Judging)
	}
	if _, err := os.Stat(filepath.Join(q, filepath.FromSlash(DrainJudgementRequestRel))); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Quinn's drain wrote a judgement request: %v", err)
	}
	if got := refOf(t, bare, secondIssue); got != tomb {
		t.Fatalf("Quinn's drain pushed over the hand-back: %s, was %s", got, tomb)
	}
}

// TestALanesHandBackIsPublishedWithItsKind: a lane handed back as a design
// finding has its reservation released by a tombstone naming that kind and
// origin's blob of the record.
func TestALanesHandBackIsPublishedWithItsKind(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	first := drainToLane(t, repo, Options{}, DrainOptions{})
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	handBackLaneOf(t, repo, first.Start.RunID, Options{}, LaneHandBack{Kind: HandBackDesignFinding, Reason: "the page needs a decision", Home: "the brief"})
	if kind, blob := tombstoneTrailers(t, bare, secondIssue); kind != HandBackDesignFinding || blob != originBlob(t, bare, secondIssue) {
		t.Fatalf("the lane's tombstone names %q over %q; want design-finding over origin's record", kind, blob)
	}
	if note := recordNote(t, repo, first.Start.RunID, StageReservationReleased); !strings.Contains(note, "marking it handed back as design-finding") {
		t.Fatalf("the run record says the hand-back is published: %q", note)
	}
}

// TestAPromotionTheCheckoutCannotMakePublishesTheTombstoneAlone: a yes of
// kind user-visible over an issue this checkout does not hold as origin's
// default branch does, absent here or held here as another record, promotes
// nothing here; the move goes on, publishes the hand-back on origin, and the
// route says why nothing was promoted.
func TestAPromotionTheCheckoutCannotMakePublishesTheTombstoneAlone(t *testing.T) {
	drafts := func(t *testing.T, root string) int {
		n, _ := os.ReadDir(filepath.Join(root, ".abcd", "development", "intents", "drafts"))
		return len(n)
	}
	for name, setup := range map[string]func(t *testing.T, bare string) (issue, why string){
		"absent here": func(t *testing.T, bare string) (string, string) {
			pushFromOtherCopy(t, bare, "file "+originIssue, func(dir string) {
				if _, err := capture.Capture(capture.CaptureRequest{RepoRoot: dir, Text: "The drain page misses a flag (" + originIssue + ")",
					Severity: capture.SeverityMinor, Category: capture.Category("tech-debt"), Source: "manual-test", Slug: "flag",
					FoundDuring: "the loop's tests", FoundAt: "commands/drain.md", ForceID: originIssue, Remedy: "Name the new flag on the drain page."}); err != nil {
					t.Fatalf("capture on origin: %v", err)
				}
			})
			return originIssue, "this checkout holds no record of " + originIssue
		},
		"another record here": func(t *testing.T, bare string) (string, string) {
			pushFromOtherCopy(t, bare, "note on "+secondIssue, func(dir string) {
				path := filepath.Join(dir, filepath.FromSlash(recordRel(t, dir, secondIssue)))
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, append(data, []byte("\nA note added on origin.\n")...), 0o644); err != nil {
					t.Fatal(err)
				}
			})
			return secondIssue, "this checkout's record of " + secondIssue + " is not the one origin's default branch holds"
		},
	} {
		t.Run(name, func(t *testing.T) {
			repo, bare := reservingDrainRepo(t, "commands/drain.md")
			issue, why := setup(t, bare)
			before := drafts(t, repo.Root())
			first, err := Drain(repo.Root(), Options{}, DrainOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if first.Judging == nil || first.Judging.Issue != issue {
				t.Fatalf("the move asks over %s: %+v", issue, first.Judging)
			}
			next := judge(t, repo, Options{}, first, JudgementYes, HandBackUserVisible, "the page gains a flag users read")
			r := routeOf(t, next.Routed, issue)
			if r.Draft != "" || !strings.Contains(r.Wrote, why) || !strings.Contains(r.Wrote, "so it is not promoted here") ||
				!strings.Contains(r.Wrote, "marking it handed back as user-visible") {
				t.Fatalf("the route says nothing was promoted here, and why, and that the hand-back is published: %+v", r)
			}
			if kind, blob := tombstoneTrailers(t, bare, issue); kind != HandBackUserVisible || blob != originBlob(t, bare, issue) {
				t.Fatalf("the tombstone names %q over %q; want user-visible over origin's record", kind, blob)
			}
			if got := drafts(t, repo.Root()); got != before {
				t.Fatalf("a draft was minted here: %d, was %d", got, before)
			}
		})
	}
}

// recordRel is id's open record in the checkout at root, repository-relative.
func recordRel(t *testing.T, root, id string) string {
	t.Helper()
	matches, _ := filepath.Glob(filepath.Join(root, ".abcd", "work", "issues", "open", id+"-*.md"))
	if len(matches) != 1 {
		t.Fatalf("%s holds no open record of %s", root, id)
	}
	rel, err := filepath.Rel(root, matches[0])
	if err != nil {
		t.Fatal(err)
	}
	return filepath.ToSlash(rel)
}

// TestAPersonsOwnBuildOverAHandBackGoesAheadWithANote: Quinn's drain hands the
// documentation issue back; Pat's own `abcd build` of it is not a drain's, so
// it reserves the issue and starts, and its start and run record name the
// hand-back it was made over.
func TestAPersonsOwnBuildOverAHandBackGoesAheadWithANote(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	q := otherCopy(t, bare, quinnEmail)
	if _, err := claim.Reserve(q, claim.Request{Record: secondIssue, Footprint: []string{"commands/drain.md"}, Drain: true}, as(t, q, claim.Options{})); err != nil {
		t.Fatal(err)
	}
	if _, err := claim.Reserve(q, claim.Request{Record: secondIssue, Release: true, HandBack: claim.HandBackSecondPackage}, as(t, q, claim.Options{})); err != nil {
		t.Fatal(err)
	}
	res, err := Start(repo.Root(), secondIssue, Options{})
	if err != nil {
		t.Fatalf("Pat's own build over Quinn's hand-back: %v", err)
	}
	want := secondIssue + " was handed back by " + heldBy(t, quinnEmail) + " as second-package at "
	if !strings.Contains(res.Reservation, want) || !strings.Contains(recordNote(t, repo, res.RunID, StageReservation), want) {
		t.Fatalf("the start names the hand-back: %q", res.Reservation)
	}
	if r := originReservation(t, bare, claim.RefPrefix+secondIssue); r.released || r.holder != heldBy(t, patEmail) {
		t.Fatalf("Pat holds the issue: %+v", r)
	}
}
