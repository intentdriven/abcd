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

// A lane's reservation narrows to the files its branch changed
// (spc-2610161044461448 step 12; iss-2610152125101468; dcn-2610161040385152):
// a drain lane that reserved a directory has its footprint narrowed, at the
// first renewal after the implement stage, to the files its branch changed
// within it, so other accounts may take the rest of the directory, and a
// narrowing never names a path outside the footprint it narrows.

// originFootprint is the footprint the tip of origin's reservation of id names.
func originFootprint(t *testing.T, bare, id string) []string {
	t.Helper()
	sha := gitIn(t, bare, "for-each-ref", "--format=%(objectname)", claim.RefPrefix+id)
	if sha == "" {
		return nil
	}
	var out []string
	for _, line := range strings.Split(gitIn(t, bare, "log", "-1", "--format=%B", sha), "\n") {
		if v, ok := strings.CutPrefix(line, "Abcd-Footprint: "); ok {
			out = append(out, v)
		}
	}
	return out
}

// commitIn commits each of names, written fresh, in the lane's worktree.
func commitIn(t *testing.T, repo *gittest.Repo, l Lane, names ...string) string {
	t.Helper()
	for _, name := range names {
		p := filepath.Join(l.Worktree, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(name+" changed\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		repo.Git("-C", l.Worktree, "add", "--", name)
	}
	repo.Git("-C", l.Worktree, "commit", "-q", "-m", "lane: "+strings.Join(names, ", "))
	return strings.TrimSpace(repo.Git("-C", l.Worktree, "rev-parse", "HEAD"))
}

// TestALanesDirectoryReservationNarrowsToItsChangedFiles: the drain reserves
// the documentation issue with the directory its found_at names; a renewal
// while the implementer works keeps the directory, and the first renewal
// after the implement stage narrows it to the files the branch changed inside
// it, leaving out a file the branch changed outside it.
func TestALanesDirectoryReservationNarrowsToItsChangedFiles(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	repo.Write("docs/guide.md", "the guide\n")
	repo.Write("docs/page.md", "the page\n")
	repo.Commit("the docs")
	issueRepoRule(t, repo)
	fileIssueAt(t, repo, eligibleIssue, capture.SeverityMinor, capture.Category("bug"), issueRemedy, "AGENTS.md")
	fileIssueAt(t, repo, secondIssue, capture.SeverityMinor, capture.Category("documentation"), "Name the flag in the page.", "docs/")
	repo.Commit("the ledger")
	asPat(t, repo)
	for _, k := range []string{"GIT_AUTHOR_NAME", "GIT_COMMITTER_NAME"} {
		t.Setenv(k, "Pat Example")
	}
	for _, k := range []string{"GIT_AUTHOR_EMAIL", "GIT_COMMITTER_EMAIL"} {
		t.Setenv(k, "pat@example.com")
	}
	bare := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, bare)
	repo.Git("remote", "add", "origin", bare)
	repo.Git("push", "-q", "origin", "main")
	repo.Git("fetch", "-q", "origin")

	// A working window long enough for both renewals below.
	pace := "480/10"
	first := drainToLane(t, repo, Options{Pace: &pace}, DrainOptions{})
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	runID := first.Start.RunID
	if fp := originFootprint(t, bare, secondIssue); !slices.Equal(fp, []string{"docs"}) {
		t.Fatalf("the lane reserves the directory its found_at names: %q", fp)
	}
	stages := DefaultStages()
	stepTo(t, repo, runID, stages, StageImplement)
	if res, err := advance(repo.Root(), runID, stages, Options{}); err != nil || res.Awaiting == nil {
		t.Fatalf("the implement stage awaits an implementer: %+v %v", res, err)
	}
	l := currentLane(t, repo, runID)
	sha := commitIn(t, repo, l, "docs/page.md", "README.txt")

	// While the implementer works, a renewal keeps the directory.
	at := originReservation(t, bare, claim.RefPrefix+secondIssue).claimed.Add(61 * time.Minute)
	o := Options{Now: func() time.Time { return at }}
	if _, err := advance(repo.Root(), runID, stages, o); err != nil {
		if r, ok := AsRefusal(err); !ok || !r.Contention {
			t.Fatalf("a step while the implementer works: %v", err)
		}
	}
	renewed := originReservation(t, bare, claim.RefPrefix+secondIssue)
	if !renewed.claimed.Equal(at) {
		t.Fatalf("the step renews while the implementer works: %+v", renewed)
	}
	if fp := originFootprint(t, bare, secondIssue); !slices.Equal(fp, []string{"docs"}) {
		t.Fatalf("a renewal during the implement stage keeps the directory: %q", fp)
	}

	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), runID, l.ID)
	rc := goodReceipt(t, runID, l, dir, sha)
	rc.Resolves = []Resolution{{Issue: secondIssue, Commit: sha, Note: "the page names the flag", Impact: "fix",
		Grounds: "pursued: the page names the flag; shown wrong if it still does not"}}
	if _, err := Receipt(repo.Root(), runID, writeReceipt(t, dir, rc), stages, o); err != nil {
		t.Fatal(err)
	}
	if l := currentLane(t, repo, runID); l.Stage != StageValidate {
		t.Fatalf("the receipt moves the lane past the implement stage: %s", l.Stage)
	}

	// The first renewal after the implement stage narrows the footprint.
	later := at.Add(61 * time.Minute)
	if _, err := advance(repo.Root(), runID, stages, Options{Now: func() time.Time { return later }}); err != nil {
		if r, ok := AsRefusal(err); !ok || !r.Contention {
			t.Fatalf("a step past the implement stage: %v", err)
		}
	}
	tip := originReservation(t, bare, claim.RefPrefix+secondIssue)
	if tip.parent != renewed.commit || tip.released || tip.holder != "octo" || !tip.claimed.Equal(later) {
		t.Fatalf("the renewal is this holder's, over the last one: before %+v, after %+v", renewed, tip)
	}
	fp := originFootprint(t, bare, secondIssue)
	if !slices.Equal(fp, []string{"docs/page.md"}) {
		t.Fatalf("the footprint narrows to the branch's changed files within the directory: %q", fp)
	}
	if !covers([]string{"docs"}, fp) {
		t.Fatalf("a narrowed footprint never names a path outside the first: %q", fp)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(st.Record, func(e Entry) bool {
		return e.Stage == StageReservationRenewed && strings.Contains(e.Note, "narrowed") && strings.Contains(e.Note, "docs/page.md")
	}) {
		t.Fatalf("the run record says the footprint narrowed: %+v", st.Record)
	}
}
