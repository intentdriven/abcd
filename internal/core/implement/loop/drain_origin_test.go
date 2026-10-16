package loop

import (
	"errors"
	"io/fs"
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

// A drain plans every move from origin's default branch (spc-2610161044461448
// step 1; iss-2610160957404827, iss-2610161015448977): each move fetches origin
// and reads one snapshot of the ledger from the default branch it fetched, so a
// checkout behind origin never reserves, judges or opens a lane for an issue
// another account finished there, and an issue new on origin is worked from
// origin's record. A move that cannot read origin plans nothing and waits.

// originIssue is an issue filed on origin alone, ordered before secondIssue.
const originIssue = "iss-2609300000000099"

// pushFromOtherCopy clones origin as Quinn's copy, lets change edit it, and
// pushes the commit to origin's main: another account's work landing.
func pushFromOtherCopy(t *testing.T, bare, msg string, change func(dir string)) {
	t.Helper()
	q := otherCopy(t, bare, quinnEmail)
	change(q)
	gitIn(t, q, "add", "-A")
	gitIn(t, q, "-c", "commit.gpgsign=false", "commit", "-q", "-m", msg)
	gitIn(t, q, "push", "-q", "origin", "HEAD:main")
}

// resolveOnOrigin moves id's record from open/ to resolved/ on origin's main.
func resolveOnOrigin(t *testing.T, bare, id string) {
	t.Helper()
	pushFromOtherCopy(t, bare, "resolve "+id, func(dir string) {
		matches, _ := filepath.Glob(filepath.Join(dir, ".abcd", "work", "issues", "open", id+"-*.md"))
		if len(matches) != 1 {
			t.Fatalf("origin holds no open record of %s", id)
		}
		if err := os.MkdirAll(filepath.Join(dir, ".abcd", "work", "issues", "resolved"), 0o755); err != nil {
			t.Fatal(err)
		}
		gitIn(t, dir, "mv", matches[0], filepath.Join(dir, ".abcd", "work", "issues", "resolved", filepath.Base(matches[0])))
	})
}

// noJudgementRequest fails when the move wrote a judgement request.
func noJudgementRequest(t *testing.T, repo *gittest.Repo) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(repo.Root(), filepath.FromSlash(DrainJudgementRequestRel))); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the move wrote a judgement request: %v", err)
	}
}

// noReservations fails when origin holds any reservation branch.
func noReservations(t *testing.T, bare string) {
	t.Helper()
	if refs := gitIn(t, bare, "for-each-ref", "--format=%(refname)", claim.RefPrefix); refs != "" {
		t.Fatalf("origin holds reservations: %s", refs)
	}
}

// TestABehindClonesMovePassesOverAnIssueResolvedOnOrigin: Quinn resolves the
// head of the queue on origin; Pat's checkout, one commit behind, neither
// reserves, judges nor opens a lane for it, and takes the next issue instead.
func TestABehindClonesMovePassesOverAnIssueResolvedOnOrigin(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	resolveOnOrigin(t, bare, secondIssue)

	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Judging == nil || res.Judging.Issue != eligibleIssue {
		t.Fatalf("the move judges %+v; want the next issue, %s, not %s finished on origin", res.Judging, eligibleIssue, secondIssue)
	}
	if slices.ContainsFunc(res.Dispositions, func(v capture.DrainVerdict) bool { return v.ID == secondIssue }) {
		t.Fatalf("the plan lists %s, which origin's default branch has resolved: %+v", secondIssue, res.Dispositions)
	}
	out := judge(t, repo, Options{}, res, JudgementNo, "", "the remedy changes neither what a user sees nor a trust boundary")
	if out.Lane == nil || out.Lane.Issue != eligibleIssue {
		t.Fatalf("the lane opened is %+v; want %s's", out.Lane, eligibleIssue)
	}
	if fp := footprintOf(t, bare, secondIssue); fp != nil {
		t.Fatalf("%s, resolved on origin, was reserved: %v", secondIssue, fp)
	}
	runs, err := Runs(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(runs, func(r State) bool { return r.Key == secondIssue }) {
		t.Fatalf("a run was opened for %s", secondIssue)
	}
}

// TestAnIssueOpenOnOriginAndAbsentHereIsJudgedAndOpened: Quinn files an issue
// on origin that Pat's checkout has never fetched into its working tree; Pat's
// drain judges it over origin's remedy and opens its lane from origin's record.
func TestAnIssueOpenOnOriginAndAbsentHereIsJudgedAndOpened(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	const remedy = "Name the new flag on the drain page."
	pushFromOtherCopy(t, bare, "file "+originIssue, func(dir string) {
		if _, err := capture.Capture(capture.CaptureRequest{RepoRoot: dir, Text: "The drain page misses a flag (" + originIssue + ")",
			Severity: capture.SeverityMinor, Category: capture.Category("documentation"), Source: "manual-test", Slug: "flag",
			FoundDuring: "the loop's tests", FoundAt: "commands/drain.md", ForceID: originIssue, Remedy: remedy}); err != nil {
			t.Fatalf("capture on origin: %v", err)
		}
	})

	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Judging == nil || res.Judging.Issue != originIssue || res.Judging.RemedySHA256 != remedyDigest(remedy) {
		t.Fatalf("the move judges %+v; want %s over origin's remedy", res.Judging, originIssue)
	}
	req, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(res.Judging.Request)))
	if err != nil || !strings.Contains(string(req), remedy) {
		t.Fatalf("the request does not quote origin's remedy: %v\n%s", err, req)
	}
	out := judge(t, repo, Options{}, res, JudgementNo, "", "the remedy changes neither what a user sees nor a trust boundary")
	if out.Start == nil || out.Lane == nil || out.Lane.Issue != originIssue {
		t.Fatalf("the lane opened is %+v; want %s's", out.Lane, originIssue)
	}
	if fp := footprintOf(t, bare, originIssue); !slices.Equal(fp, []string{"commands/drain.md"}) {
		t.Fatalf("the reservation's footprint is origin's found_at: %v", fp)
	}
	if matches, _ := filepath.Glob(filepath.Join(repo.Root(), ".abcd", "work", "issues", "*", originIssue+"-*.md")); len(matches) != 0 {
		t.Fatalf("the checkout gained a record of %s: %v", originIssue, matches)
	}
	// The lane is cut from origin's default branch, so its brief carries
	// origin's record and remedy.
	stepTo(t, repo, out.Start.RunID, DefaultStages(), StageImplement)
	l := currentLane(t, repo, out.Start.RunID)
	brief, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), out.Start.RunID, l.ID, BriefFileName))
	if err != nil || !strings.Contains(string(brief), remedy) {
		t.Fatalf("the lane's brief does not carry origin's remedy: %v\n%s", err, brief)
	}
}

// TestADrainMoveThatCannotReadOriginWaits: a fetch that fails, a default
// branch the fetch could not read, and no default branch of origin at all each
// plan nothing: the move records an origin outage probe in the drain's state,
// says when it looks again, asks no judgement and reserves nothing.
func TestADrainMoveThatCannotReadOriginWaits(t *testing.T) {
	at := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)
	cases := []struct {
		name, kind, says string
		breakOrigin      func(t *testing.T, repo *gittest.Repo, bare string)
	}{
		{"failed fetch", DrainOriginFetch, "could not fetch",
			func(t *testing.T, repo *gittest.Repo, bare string) {
				repo.Git("remote", "set-url", "origin", filepath.Join(t.TempDir(), "gone.git"))
			}},
		{"default-branch gap", DrainOriginDefaultBranch, "could not read",
			func(t *testing.T, repo *gittest.Repo, bare string) {
				gitIn(t, bare, "branch", "-m", "main", "trunk")
				gitIn(t, bare, "symbolic-ref", "HEAD", "refs/heads/trunk")
			}},
		{"missing origin/HEAD", DrainOriginHead, "git remote set-head origin --auto",
			func(t *testing.T, repo *gittest.Repo, bare string) {
				repo.Git("update-ref", "-d", "refs/remotes/origin/main")
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, bare := reservingDrainRepo(t, "commands/drain.md")
			tc.breakOrigin(t, repo, bare)
			o := Options{Now: func() time.Time { return at }}
			res, err := Drain(repo.Root(), o, DrainOptions{})
			if err != nil {
				t.Fatalf("a move that cannot read origin: %v; want it to wait", err)
			}
			if res.Judging != nil || res.Lane != nil || res.Start != nil || len(res.Dispositions) != 0 {
				t.Fatalf("the move planned: %+v", res)
			}
			noJudgementRequest(t, repo)
			noReservations(t, bare)
			if !strings.Contains(res.Next, tc.says) || !strings.Contains(res.Next, "plans nothing") {
				t.Fatalf("next = %q; want it to say the move plans nothing and %q", res.Next, tc.says)
			}
			st, live, err := readDrain(repo.Root())
			if err != nil || !live || st.OriginWait == nil {
				t.Fatalf("the drain's state records no origin wait: %+v %v %v", st.OriginWait, live, err)
			}
			w := st.OriginWait
			if w.Kind != tc.kind || w.Probes != 1 || !w.Since.Equal(at) || !w.NextProbeAt.After(at) {
				t.Fatalf("the origin wait = %+v; want kind %s, one probe, since %s, next after it", *w, tc.kind, at)
			}
			if res.OriginWait == nil || res.OriginWait.Kind != tc.kind {
				t.Fatalf("the result carries no origin wait: %+v", res.OriginWait)
			}

			// Before the next probe is due, the move looks at nothing and the
			// record is unchanged.
			again, err := Drain(repo.Root(), o, DrainOptions{})
			if err != nil || again.Judging != nil || !strings.Contains(again.Next, w.NextProbeAt.UTC().Format(time.RFC3339)) {
				t.Fatalf("a move before the next probe = %+v, %v; want it to wait naming %s", again, err, w.NextProbeAt)
			}
			if st, _, _ := readDrain(repo.Root()); st.OriginWait == nil || st.OriginWait.Probes != 1 {
				t.Fatalf("a move before the probe is due counted a probe: %+v", st.OriginWait)
			}
		})
	}

	// Once origin answers again, the next move at or after the probe plans
	// from it and the wait is cleared.
	t.Run("recovers", func(t *testing.T) {
		repo, bare := reservingDrainRepo(t, "commands/drain.md")
		repo.Git("remote", "set-url", "origin", filepath.Join(t.TempDir(), "gone.git"))
		o := Options{Now: func() time.Time { return at }}
		if _, err := Drain(repo.Root(), o, DrainOptions{}); err != nil {
			t.Fatal(err)
		}
		repo.Git("remote", "set-url", "origin", bare)
		later := at.Add(2 * time.Minute)
		res, err := Drain(repo.Root(), Options{Now: func() time.Time { return later }}, DrainOptions{})
		if err != nil || res.Judging == nil || res.Judging.Issue != secondIssue {
			t.Fatalf("the move once origin answers = %+v, %v; want it to plan and judge %s", res.Judging, err, secondIssue)
		}
		if st, _, _ := readDrain(repo.Root()); st.OriginWait != nil {
			t.Fatalf("the wait is kept after origin answered: %+v", st.OriginWait)
		}
	})
}

// tierFiles is every file under the checkout's local tier with its content,
// and the checkout's git status: what a write would change.
func tierFiles(t *testing.T, repo *gittest.Repo) map[string]string {
	t.Helper()
	out := map[string]string{"git status": repo.Git("status", "--porcelain", "--untracked-files=all")}
	root := filepath.Join(repo.Root(), ".abcd", ".work.local")
	if err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(p)
		out[p] = string(data)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

// TestTheDryRunNamesTheFinishedIssueTheViewsAgeAndTheCommitsBehind: the dry run
// plans from origin's default branch as last fetched, fetching nothing; it names
// the issue open here that origin finished, how old its view is and how many
// commits the checkout is behind, and writes nothing.
func TestTheDryRunNamesTheFinishedIssueTheViewsAgeAndTheCommitsBehind(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	resolveOnOrigin(t, bare, secondIssue)
	fetchedAt := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)
	if _, err := claim.Fetch(repo.Root(), claim.Options{Now: func() time.Time { return fetchedAt }}); err != nil {
		t.Fatal(err)
	}
	// Origin moves again after the fetch; the dry run must not see it.
	pushFromOtherCopy(t, bare, "file "+originIssue, func(dir string) {
		fileIssueAtRoot(t, dir, originIssue)
	})

	before := tierFiles(t, repo)
	pv, err := DrainDryRun(repo.Root(), fetchedAt.Add(30*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if after := tierFiles(t, repo); len(after) != len(before) {
		t.Fatalf("the dry run wrote: before %v, after %v", before, after)
	} else {
		for k, v := range before {
			if after[k] != v {
				t.Fatalf("the dry run changed %s", k)
			}
		}
	}
	if pv.Origin == nil || pv.Origin.Ref != "origin/main" || pv.Origin.Behind != 1 || pv.Origin.FetchedAt == nil ||
		!pv.Origin.FetchedAt.Equal(fetchedAt) || pv.Origin.AgeMinutes != 30 || !slices.Equal(pv.Origin.Finished, []string{secondIssue}) {
		t.Fatalf("the dry run's origin = %+v; want origin/main, one commit behind, fetched 30 minutes before, %s finished", pv.Origin, secondIssue)
	}
	for _, v := range pv.Dispositions {
		if v.ID == secondIssue || v.ID == originIssue {
			t.Fatalf("the dry run lists %s: it plans from origin as last fetched", v.ID)
		}
	}
	if !slices.ContainsFunc(pv.Dispositions, func(v capture.DrainVerdict) bool { return v.ID == eligibleIssue && v.Outcome == capture.DrainEligible }) {
		t.Fatalf("the dry run does not list %s as eligible: %+v", eligibleIssue, pv.Dispositions)
	}
}

// fileIssueAtRoot files originIssue in the checkout at dir.
func fileIssueAtRoot(t *testing.T, dir, id string) {
	t.Helper()
	if _, err := capture.Capture(capture.CaptureRequest{RepoRoot: dir, Text: "The drain page misses a flag (" + id + ")",
		Severity: capture.SeverityMinor, Category: capture.Category("documentation"), Source: "manual-test", Slug: "flag",
		FoundDuring: "the loop's tests", FoundAt: "commands/drain.md", ForceID: id, Remedy: "Name the flag."}); err != nil {
		t.Fatalf("capture: %v", err)
	}
}
