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
)

// `abcd build <iss-N>` judges the issue from origin's default branch after a
// fetch, as the drain's own move does (iss-2611101316317556): a checkout
// behind origin neither refuses an issue origin holds open nor builds one
// origin has finished, and when its own ledger disagrees the eligible row says
// the checkout is behind and by how many commits. Where origin cannot be read
// the build judges from the checkout's own ledger, as it did before, and says
// why; a checkout with no remote named origin is judged from its working tree
// (TestAnIssueKeyOpensOneLaneWhoseBriefIsTheRecordAndItsRemedy).

// eligibleRow is the start's or the refusal's eligible row.
func eligibleRow(t *testing.T, rows []CheckRow) CheckRow {
	t.Helper()
	i := slices.IndexFunc(rows, func(c CheckRow) bool { return c.Name == CheckEligible })
	if i < 0 {
		t.Fatalf("no eligible row among %+v", rows)
	}
	return rows[i]
}

// TestABuildOfAnIssueOnOriginAloneIsJudgedFromOrigin: Quinn files an issue on
// origin that Pat's checkout has never fetched; Pat's `abcd build` of it is
// not refused, reserves it with origin's found_at, and says the checkout is
// one commit behind origin/main.
func TestABuildOfAnIssueOnOriginAloneIsJudgedFromOrigin(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	pushFromOtherCopy(t, bare, "file "+originIssue, func(dir string) {
		if _, err := capture.Capture(capture.CaptureRequest{RepoRoot: dir, Text: "The drain page misses a flag (" + originIssue + ")",
			Severity: capture.SeverityMinor, Category: capture.Category("tech-debt"), Source: "manual-test", Slug: "flag",
			FoundDuring: "the loop's tests", FoundAt: "commands/drain.md", ForceID: originIssue, Remedy: "Name the new flag on the drain page."}); err != nil {
			t.Fatalf("capture on origin: %v", err)
		}
	})

	start, err := Start(repo.Root(), originIssue, Options{})
	if err != nil {
		t.Fatalf("a build of an issue origin holds open is refused: %v", err)
	}
	row := eligibleRow(t, start.Checks)
	for _, want := range []string{originIssue + " is eligible under", "read from origin/main at",
		"this checkout is 1 commit(s) behind origin/main, and its own ledger does not hold " + originIssue + " open"} {
		if !row.OK || !strings.Contains(row.Detail, want) {
			t.Fatalf("the eligible row = %+v; want it to say %q", row, want)
		}
	}
	if fp := footprintOf(t, bare, originIssue); !slices.Equal(fp, []string{"commands/drain.md"}) {
		t.Fatalf("the reservation's footprint is origin's found_at: %v", fp)
	}
	if matches, _ := filepath.Glob(filepath.Join(repo.Root(), ".abcd", "work", "issues", "*", originIssue+"-*.md")); len(matches) != 0 {
		t.Fatalf("the checkout gained a record of %s: %v", originIssue, matches)
	}

	// Pat's build again, a killed session's or the land fence's "reserve it
	// again with abcd build", resumes the run and renews its reservation with
	// origin's found_at, which this checkout's ledger still does not hold.
	held := originReservation(t, bare, claim.RefPrefix+originIssue)
	again, err := Start(repo.Root(), originIssue, Options{})
	if err != nil || !again.Resumed || again.RunID != start.RunID {
		t.Fatalf("a second build of the issue only origin holds resumes its run: %+v %v", again, err)
	}
	if tip := originReservation(t, bare, claim.RefPrefix+originIssue); tip.parent != held.commit || tip.released {
		t.Fatalf("the resume renews the issue's reservation: before %+v, after %+v", held, tip)
	}
	if fp := footprintOf(t, bare, originIssue); !slices.Equal(fp, []string{"commands/drain.md"}) {
		t.Fatalf("the resumed reservation's footprint is origin's found_at: %v", fp)
	}
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if got := declaredPaths(repo.Root(), st, "nothing"); !slices.Equal(got, []string{"commands/drain.md"}) {
		t.Fatalf("a walled stage of the run declares origin's found_at: %v", got)
	}
}

// TestABuildOfAnIssueOnOriginIsJudgedFromItsTipAsStored: origin's default
// branch holds an issue whose found_at an earlier commit named otherwise, and
// this checkout holds a refs/replace entry standing a tag of that earlier
// commit in for origin's tip. The build judges the issue from the tip as
// stored: it reserves the tip's found_at, a walled stage of the run declares
// it, and the eligible row counts the commits behind the tip as stored.
func TestABuildOfAnIssueOnOriginIsJudgedFromItsTipAsStored(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	pushFromOtherCopy(t, bare, "file "+originIssue, func(dir string) {
		if _, err := capture.Capture(capture.CaptureRequest{RepoRoot: dir, Text: "The drain page misses a flag (" + originIssue + ")",
			Severity: capture.SeverityMinor, Category: capture.Category("tech-debt"), Source: "manual-test", Slug: "flag",
			FoundDuring: "the loop's tests", FoundAt: "AGENTS.md", ForceID: originIssue, Remedy: "Name the new flag on the drain page."}); err != nil {
			t.Fatalf("capture on origin: %v", err)
		}
	})
	earlier := gitIn(t, bare, "rev-parse", "main")
	pushFromOtherCopy(t, bare, "point "+originIssue+" at the drain page", func(dir string) {
		matches, _ := filepath.Glob(filepath.Join(dir, ".abcd", "work", "issues", "open", originIssue+"-*.md"))
		if len(matches) != 1 {
			t.Fatalf("origin holds no open record of %s", originIssue)
		}
		body, err := os.ReadFile(matches[0])
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(string(body), "\n")
		i := slices.IndexFunc(lines, func(l string) bool { return strings.HasPrefix(l, "found_at:") })
		if i < 0 {
			t.Fatalf("the record names no found_at:\n%s", body)
		}
		lines[i] = `found_at: "commands/drain.md"`
		if err := os.WriteFile(matches[0], []byte(strings.Join(lines, "\n")), 0o600); err != nil {
			t.Fatal(err)
		}
	})
	tip := gitIn(t, bare, "rev-parse", "main")
	repo.Git("fetch", "-q", "origin")
	repo.Git("-c", "tag.gpgsign=false", "tag", "-a", "-m", "stand-in", "stand-in", earlier)
	standIn := repo.Git("rev-parse", "refs/tags/stand-in")
	repo.Git("update-ref", "-d", "refs/tags/stand-in")
	repo.Git("update-ref", "refs/replace/"+tip, standIn)

	start, err := Start(repo.Root(), originIssue, Options{})
	if err != nil {
		t.Fatalf("a build of an issue origin holds open is refused: %v", err)
	}
	row := eligibleRow(t, start.Checks)
	if !row.OK || !strings.Contains(row.Detail, "read from origin/main at "+shortSHA(tip)) {
		t.Fatalf("the eligible row = %+v; want it read from origin's tip %s", row, shortSHA(tip))
	}
	if want := "this checkout is 2 commit(s) behind origin/main"; !strings.Contains(row.Detail, want) {
		t.Fatalf("the eligible row = %+v; want it to count the tip's stored history: %q", row, want)
	}
	if fp := footprintOf(t, bare, originIssue); !slices.Equal(fp, []string{"commands/drain.md"}) {
		t.Fatalf("the reservation's footprint is the tip's found_at: %v", fp)
	}
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if got := declaredPaths(repo.Root(), st, "nothing"); !slices.Equal(got, []string{"commands/drain.md"}) {
		t.Fatalf("a walled stage of the run declares the tip's found_at: %v", got)
	}
}

// TestABuildOfAnIssueFinishedOnOriginIsRefusedSayingTheCheckoutIsBehind:
// Quinn resolves an issue on origin that Pat's checkout still holds open; the
// build refuses it as origin's ledger reads, says the checkout is behind, and
// reserves nothing.
func TestABuildOfAnIssueFinishedOnOriginIsRefusedSayingTheCheckoutIsBehind(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	resolveOnOrigin(t, bare, secondIssue)

	_, err := Start(repo.Root(), secondIssue, Options{})
	r := mustRefusal(t, err)
	for _, want := range []string{secondIssue + " is not an open issue on origin/main at",
		"this checkout is 1 commit(s) behind origin/main, and its own ledger still holds " + secondIssue + " open"} {
		if r.Check != CheckEligible || !strings.Contains(r.Reason, want) {
			t.Fatalf("the refusal = %+v; want it at the eligible check saying %q", r, want)
		}
	}
	if row := eligibleRow(t, r.Checks); row.OK || row.Detail != r.Reason {
		t.Fatalf("the refusal's rows carry the eligible row as its reason: %+v", row)
	}
	noReservations(t, bare)
	runTierAbsent(t, repo.Root())
}

// TestABuildThatCannotFetchOriginJudgesFromTheCheckoutAndSaysSo: with origin
// gone, the fetch fails within its bound and the build judges the issue from
// this checkout's own ledger, as it did before, and its eligible row says the
// fetch failed and that the ledger read may be behind origin.
func TestABuildThatCannotFetchOriginJudgesFromTheCheckoutAndSaysSo(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	repo.Git("remote", "set-url", "origin", filepath.Join(t.TempDir(), "gone.git"))

	_, err := Start(repo.Root(), originIssue, Options{})
	r := mustRefusal(t, err)
	for _, want := range []string{originIssue + " is not an open issue in this checkout's ledger",
		"could not fetch origin", "so it is judged from this checkout's own ledger, which may be behind origin"} {
		if r.Check != CheckEligible || !strings.Contains(r.Reason, want) {
			t.Fatalf("the refusal = %+v; want it at the eligible check saying %q", r, want)
		}
	}
	runTierAbsent(t, repo.Root())
}

// TestABuildThatCannotReadOriginsDefaultBranchRecordsItInTheRun: origin
// answers but no longer holds the branch this checkout tracks as its default;
// the build judges the issue from this checkout's own ledger, its eligible row
// says why, and so does the run's own record, which outlives the row.
func TestABuildThatCannotReadOriginsDefaultBranchRecordsItInTheRun(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	gitIn(t, bare, "branch", "-m", "main", "trunk-renamed")

	start, err := Start(repo.Root(), secondIssue, Options{})
	if err != nil {
		t.Fatalf("a build that cannot read origin's default branch judges the checkout's ledger: %v", err)
	}
	const why = "could not read origin's default branch"
	if row := eligibleRow(t, start.Checks); !row.OK || !strings.Contains(row.Detail, why) ||
		!strings.Contains(row.Detail, "so it is judged from this checkout's own ledger, which may be behind origin") {
		t.Fatalf("the eligible row = %+v; want it to say %q", row, why)
	}
	if note := recordNote(t, repo, start.RunID, CheckEligible); !strings.Contains(note, why) ||
		!strings.Contains(note, "judged from this checkout's own ledger") {
		t.Fatalf("the run's record says the issue was judged from the checkout's own ledger: %q", note)
	}
}

// TestABuildInACheckoutNamingNoDefaultBranchOfOriginSaysSo: a checkout that
// tracks no default branch of origin (no origin/HEAD, and none of the
// conventional names) has no commit of origin to judge the issue from; the
// build judges this checkout's own ledger and its row and record say why.
func TestABuildInACheckoutNamingNoDefaultBranchOfOriginSaysSo(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	gitIn(t, bare, "branch", "-m", "main", "release")
	repo.Git("update-ref", "--no-deref", "-d", "refs/remotes/origin/HEAD")
	repo.Git("update-ref", "-d", "refs/remotes/origin/main")

	start, err := Start(repo.Root(), secondIssue, Options{})
	if err != nil {
		t.Fatalf("a build in a checkout naming no default branch of origin judges the checkout's ledger: %v", err)
	}
	const why = "this checkout names no default branch of origin"
	if row := eligibleRow(t, start.Checks); !row.OK || !strings.Contains(row.Detail, why) {
		t.Fatalf("the eligible row = %+v; want it to say %q", row, why)
	}
	if note := recordNote(t, repo, start.RunID, CheckEligible); !strings.Contains(note, why) {
		t.Fatalf("the run's record says why origin was not read: %q", note)
	}
}

// TestABuildOfAnIssueOnOriginWithNoFoundAtIsRefusedAtTheReservation: origin's
// record names no found_at path, and this checkout holds none at all, so the
// start is refused at the reservation as one of the checkout's own would be,
// starting nothing and reserving nothing.
func TestABuildOfAnIssueOnOriginWithNoFoundAtIsRefusedAtTheReservation(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	pushFromOtherCopy(t, bare, "file "+originIssue, func(dir string) {
		if _, err := capture.Capture(capture.CaptureRequest{RepoRoot: dir, Text: "The drain page misses a flag (" + originIssue + ")",
			Severity: capture.SeverityMinor, Category: capture.Category("tech-debt"), Source: "manual-test", Slug: "flag",
			FoundDuring: "the loop's tests", ForceID: originIssue, Remedy: "Name the new flag on the drain page."}); err != nil {
			t.Fatalf("capture on origin: %v", err)
		}
	})

	_, err := Start(repo.Root(), originIssue, Options{})
	r := mustRefusal(t, err)
	if r.Stage != StageReservation || !strings.Contains(r.Reason, originIssue+" has no found_at path to reserve it with") ||
		!strings.HasSuffix(r.Reason, "nothing was started") {
		t.Fatalf("an origin record with no found_at is refused at the reservation: %+v", r)
	}
	runTierAbsent(t, repo.Root())
	noReservations(t, bare)
}

// TestABuildOfAPromotedIssueOnOriginAloneIsAskedAgainFromOrigin: an issue
// linked to an intent, which a drain hands back as promoted, sits on origin
// alone; a person's build asks the plan again with promoted set aside from
// origin's ledger, not the checkout's, so it starts, reserved with origin's
// found_at.
func TestABuildOfAPromotedIssueOnOriginAloneIsAskedAgainFromOrigin(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	pushFromOtherCopy(t, bare, "file "+originIssue, func(dir string) {
		if _, err := capture.Capture(capture.CaptureRequest{RepoRoot: dir, Text: "The audit of itd-2609300000000201 could not decide ac-1",
			Severity: capture.SeverityMinor, Category: capture.Category("inconsistency"), Source: "agent-finding", Slug: "owed",
			FoundDuring: "abcd intent audit ingest", FoundAt: "commands/drain.md", ForceID: originIssue, Remedy: "re-run the audit",
			RelatedIntents: []string{"itd-2609300000000201"}}); err != nil {
			t.Fatalf("capture on origin: %v", err)
		}
	})

	start, err := Start(repo.Root(), originIssue, Options{})
	if err != nil {
		t.Fatalf("a person's build of a promoted issue only origin holds starts: %v", err)
	}
	row := eligibleRow(t, start.Checks)
	for _, want := range []string{"itd-2609300000000201", "promoted", "read from origin/main at"} {
		if !row.OK || !strings.Contains(row.Detail, want) {
			t.Fatalf("the eligible row = %+v; want it to say %q", row, want)
		}
	}
	if fp := footprintOf(t, bare, originIssue); !slices.Equal(fp, []string{"commands/drain.md"}) {
		t.Fatalf("the reservation's footprint is origin's found_at: %v", fp)
	}
}

// TestABuildOfARecordNotYetOnOriginSaysTheCheckoutIsNotBehind: Pat captures an
// issue here and has not pushed it; origin's default branch, which the lane is
// cut from, holds no such record, so the build refuses it and says the
// checkout is not behind, but holds it open in its own ledger.
func TestABuildOfARecordNotYetOnOriginSaysTheCheckoutIsNotBehind(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	fileIssueAt(t, repo, originIssue, capture.SeverityMinor, capture.Category("bug"), issueRemedy, "commands/drain.md")
	repo.Commit("file " + originIssue + " here")

	_, err := Start(repo.Root(), originIssue, Options{})
	r := mustRefusal(t, err)
	for _, want := range []string{originIssue + " is not an open issue on origin/main at",
		"this checkout is not behind origin/main, but its own ledger holds " + originIssue + " open"} {
		if r.Check != CheckEligible || !strings.Contains(r.Reason, want) {
			t.Fatalf("the refusal = %+v; want it at the eligible check saying %q", r, want)
		}
	}
	runTierAbsent(t, repo.Root())
	noReservations(t, bare)
}

// TestABuildOfARecordNotYetOnOriginFromABehindCheckoutSaysItHasNotReachedOrigin:
// Pat files an issue in a checkout that is also behind origin/main, which
// another account moved, and origin holds it neither open nor finished. The refusal
// says the record has not reached origin and to land it there first, and does
// not blame the checkout being behind, which would read as origin having
// finished the issue.
func TestABuildOfARecordNotYetOnOriginFromABehindCheckoutSaysItHasNotReachedOrigin(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	pushFromOtherCopy(t, bare, "another account's change", func(dir string) {
		if err := os.WriteFile(filepath.Join(dir, "other.txt"), []byte("another account's change\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	})
	fileIssueAt(t, repo, originIssue, capture.SeverityMinor, capture.Category("bug"), issueRemedy, "commands/drain.md")
	repo.Commit("file " + originIssue + " here")

	_, err := Start(repo.Root(), originIssue, Options{})
	r := mustRefusal(t, err)
	for _, want := range []string{originIssue + " is not an open issue on origin/main at",
		"origin/main holds " + originIssue + " neither open nor finished, while this checkout's own ledger holds it open: the record has not reached origin/main, so land it there first",
		"this checkout is also 1 commit(s) behind origin/main"} {
		if r.Check != CheckEligible || !strings.Contains(r.Reason, want) {
			t.Fatalf("the refusal = %+v; want it at the eligible check saying %q", r, want)
		}
	}
	if strings.Contains(r.Reason, "its own ledger still holds") {
		t.Fatalf("the refusal blames the checkout being behind for a record origin never held: %s", r.Reason)
	}
	runTierAbsent(t, repo.Root())
	noReservations(t, bare)
}

// TestADrainLaneLeavingItsLandingReadsItsFoundAtFromOrigin
// (iss-2611010953085251): the drain judged the documentation issue from
// origin's default branch, and the checkout the run lives in holds no record
// of it. Origin's default branch moves while the lane is at its landing, so
// the landing's sync sends the lane back to a fresh round and widens its
// narrowed reservation back to the issue's found_at, which it reads from
// origin's record as last fetched rather than refusing because this
// checkout's ledger lacks it.
func TestADrainLaneLeavingItsLandingReadsItsFoundAtFromOrigin(t *testing.T) {
	f := newNarrowFixture(t)
	f.implementOnce(t, "docs/page.md")
	f.toLandOnce(t)
	matches, _ := filepath.Glob(filepath.Join(f.repo.Root(), ".abcd", "work", "issues", "*", secondIssue+"-*.md"))
	if len(matches) != 1 {
		t.Fatalf("the checkout holds one record of %s: %v", secondIssue, matches)
	}
	f.repo.Git("rm", "-q", "--", matches[0])
	f.repo.Commit("this checkout's ledger without " + secondIssue)
	pushFromOtherCopy(t, f.bare, "another account's change", func(dir string) {
		if err := os.WriteFile(filepath.Join(dir, "other.txt"), []byte("another account's change\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	})

	due := f.tip(t).claimed.Add(61 * time.Minute)
	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{Now: func() time.Time { return due }})
	l := currentLane(t, f.repo, f.runID)
	if l.Stage != StageValidate || len(l.Syncs) != 1 {
		t.Fatalf("the landing's sync sends the lane back to a fresh round: stage %s, syncs %d, step %v", l.Stage, len(l.Syncs), err)
	}
	if r, ok := AsRefusal(err); ok && strings.Contains(r.Reason, "in no status folder") {
		t.Fatalf("the step is refused for the checkout's ledger lacking the record: %v", err)
	}
	if fp := f.footprint(t); !slices.Contains(fp, "docs") {
		t.Fatalf("the lane leaving its landing holds origin's found_at again: %q", fp)
	}
}
