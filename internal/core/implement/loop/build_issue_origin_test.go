package loop

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

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
