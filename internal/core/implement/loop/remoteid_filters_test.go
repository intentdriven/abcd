package loop

// remoteid_filters_test.go holds ruling 1 of fix round 5 of
// iss-2611131308266006: an isolated read a drive makes that runs the
// repository's content filters asks the drive's comparison before git runs,
// as the hooked runners do, so a filter changed after a step's comparison is
// not run by it.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// filteredRecordRepo is a checkout holding secondIssue's open record, its
// .md files given a clean filter its settings name, read by the drive as it
// starts (the hold returned); then the filter's program is changed to one
// that leaves mark when it runs, as an agent of the same account could.
// The fixture's home lists the checkout in filter-roots when filtersOn.
func filteredRecordRepo(t *testing.T, filtersOn bool) (repo *gittest.Repo, id *remoteIdentity, mark string) {
	t.Helper()
	repo = briefRepo(t, agentsMarked)
	home := os.Getenv("HOME")
	repo.Write("commands/other.md", "# other\n")
	fileIssueAt(t, repo, secondIssue, capture.SeverityMinor, capture.Category("documentation"), issueRemedy, "commands/other.md")
	repo.Commit("the issue")
	dir := t.TempDir()
	mark = filepath.Join(dir, "changed-filter-ran")
	writeMode(t, filepath.Join(dir, "clean"), "#!/bin/sh\ncat\n", 0o755)
	writeMode(t, filepath.Join(dir, "changed"), "#!/bin/sh\n: > '"+mark+"'\ncat\n", 0o755)
	repo.Git("config", "filter.watch.clean", filepath.Join(dir, "clean"))
	writeMode(t, filepath.Join(repo.Root(), ".git", "info", "attributes"), "*.md filter=watch\n", 0o644)
	if filtersOn {
		mkdirs(t, abcdhome.Path(home))
		writeMode(t, abcdhome.Path(home, "filter-roots"), repo.Root()+"\n", 0o600)
	}
	id, err := readRemoteIdentity(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	repo.Git("config", "filter.watch.clean", filepath.Join(dir, "changed"))
	return repo, id, mark
}

// driveCheck sets the check a drive sets, its comparison of what it holds
// (heldBeforeGit) on root against id, until the test ends or the returned
// clear is called.
func driveCheck(t *testing.T, root string, id *remoteIdentity) (clear func()) {
	t.Helper()
	unset, err := gitutil.SetBeforeHookedGit(func([]string) error { return heldBeforeGit(root, id) })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(unset)
	return unset
}

// TestAFilterChangedAfterTheComparisonIsNotRunByTheHandBacksRead: the
// drain's hand-back reads the checkout's record as git would store it
// (hash-object), through the clean filter the repository names; with that
// filter's program changed since the drive started, the read is not run: the
// step stops at the config stage saying git hash-object was not run and
// naming the setting, and the changed program leaves no mark. With no check
// set the same read runs it, so the fixture is live. Watched failing before
// isolatedGit asked: the changed filter ran and the hand-back went on.
func TestAFilterChangedAfterTheComparisonIsNotRunByTheHandBacksRead(t *testing.T) {
	repo, id, mark := filteredRecordRepo(t, false)
	origin := &DrainOrigin{Ref: "origin/main", Commit: strings.TrimSpace(repo.Git("rev-parse", "HEAD"))}
	clear := driveCheck(t, repo.Root(), id)
	_, err := notHeldAsOrigin(repo.Root(), secondIssue, origin)
	notRun(t, err, "hash-object", "filter.watch.clean")
	absent(t, mark, "the changed filter is not run")

	clear()
	if why, err := notHeldAsOrigin(repo.Root(), secondIssue, origin); err != nil || why != "" {
		t.Fatalf("with no check set the record reads as origin's: %q %v", why, err)
	}
	if _, err := os.Stat(mark); err != nil {
		t.Fatalf("with no check set the read runs the repository's filter, as it always has: %v", err)
	}
}

// TestAFilterChangedAfterTheComparisonIsNotRunByAStatusWithFiltersOn: where
// the person has switched the filters back on for the checkout
// (filter-roots), a status read runs them, so it asks the comparison too: it
// is not run, it says why, and the changed program leaves no mark. With the
// filters off (no declaration) the status asks nothing and runs no filter.
// Watched failing before isolatedGit asked: the changed filter ran.
func TestAFilterChangedAfterTheComparisonIsNotRunByAStatusWithFiltersOn(t *testing.T) {
	t.Run("filters on", func(t *testing.T) {
		repo, id, mark := filteredRecordRepo(t, true)
		touchLater(t, repo)
		driveCheck(t, repo.Root(), id)
		_, err := gitutil.Status(repo.Root(), 1<<20, gitutil.StatusOptions{})
		r, ok := heldStopOf(err)
		if !ok || r.Stage != StageConfig || !strings.Contains(r.Reason, "git status was not run") || !strings.Contains(r.Reason, "filter.watch.clean") {
			t.Fatalf("the status is stopped by the comparison, naming the setting: %v", err)
		}
		absent(t, mark, "the changed filter is not run")
	})
	t.Run("filters off", func(t *testing.T) {
		repo, id, mark := filteredRecordRepo(t, false)
		touchLater(t, repo)
		driveCheck(t, repo.Root(), id)
		if _, err := gitutil.Status(repo.Root(), 1<<20, gitutil.StatusOptions{}); err != nil {
			t.Fatalf("a status that starts no filter asks nothing: %v", err)
		}
		absent(t, mark, "no filter runs with the filters off")
	})
}

// touchLater makes every tracked .md file's saved stat stale, so a status
// reads it again, through its filter.
func touchLater(t *testing.T, repo *gittest.Repo) {
	t.Helper()
	later := time.Now().Add(time.Hour)
	for _, rel := range strings.Fields(repo.Git("ls-files", "*.md")) {
		if err := os.Chtimes(filepath.Join(repo.Root(), filepath.FromSlash(rel)), later, later); err != nil {
			t.Fatal(err)
		}
	}
}
