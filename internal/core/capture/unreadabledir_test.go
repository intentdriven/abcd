package capture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// unreadableStatusDir creates the status directory sub under ir and takes
// every permission off it, restoring them when the test ends so the temporary
// directory can be removed. A test that runs as root reads through the mode
// bits, so it is skipped there rather than passing for the wrong reason.
func unreadableStatusDir(t *testing.T, ir string, sub State) string {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("root reads through a mode-000 directory")
	}
	dir := filepath.Join(ir, statusDirName[sub])
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	return dir
}

// An unreadable status directory is not an absent one (iss-2609261631120364).
// The filing-time match that cannot read resolved/ has not compared the
// record, so it says the record set was unread, as it does when the intent
// store cannot be read, instead of reporting the records it happened to reach.
func TestMatchReportsAnUnreadableStatusDirectoryAsUnread(t *testing.T) {
	repo, ir := ledger(t)
	captureText(t, repo, ir, plantedFinding, nil)
	captureText(t, repo, ir, matchFiller1, nil)
	unreadableStatusDir(t, ir, StateResolved)

	res := captureText(t, repo, ir, plantedDouble, bundled())
	if res.Match == nil || !strings.Contains(res.Match.Skipped, "could not be read") {
		t.Fatalf("an unreadable resolved/ did not read as an unread record set: %+v", res.Match)
	}
	if len(res.Match.Matches) != 0 {
		t.Fatalf("an unread record set still linked: %+v", res.Match.Matches)
	}
	if _, err := os.Stat(filepath.Join(repo, res.Path)); err != nil {
		t.Fatalf("the capture was not written: %v", err)
	}
}

// The board says the same thing: capture list and capture status name an
// unreadable status directory in the skipped roster rather than counting it
// as a directory with no records in it. An absent directory stays silent — a
// virgin ledger has none.
func TestListAndStatusReportAnUnreadableStatusDirectory(t *testing.T) {
	repo, ir := ledger(t)
	captureText(t, repo, ir, plantedFinding, nil)
	dir := unreadableStatusDir(t, ir, StateResolved)

	named := func(skipped []SkipRecord) bool {
		for _, sk := range skipped {
			if filepath.Base(sk.Path) == filepath.Base(dir) && sk.Layer == SkipLayerRead {
				return true
			}
		}
		return false
	}
	list, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir})
	if err != nil {
		t.Fatal(err)
	}
	if !named(list.Skipped) {
		t.Fatalf("capture list did not name the unreadable %s: %+v", dir, list.Skipped)
	}
	if len(list.Skipped) != 1 {
		t.Fatalf("the absent wontfix/ was reported too: %+v", list.Skipped)
	}
	st, err := Status(StatusRequest{RepoRoot: repo, IssuesRoot: ir})
	if err != nil {
		t.Fatal(err)
	}
	if !named(st.Skipped) || st.SkippedCount != 1 {
		t.Fatalf("capture status did not count the unreadable %s: %+v", dir, st.Skipped)
	}
}

// A list scoped to resolved/ still reads open/ for the blocked_by projection;
// an unreadable open/ there leaves every dependent looking unblocked, so the
// roster names it even though the rows come from another directory.
func TestAScopedListNamesAnUnreadableOpenDirectory(t *testing.T) {
	repo, ir := ledger(t)
	if err := os.MkdirAll(filepath.Join(ir, statusDirName[StateResolved]), 0o755); err != nil {
		t.Fatal(err)
	}
	dir := unreadableStatusDir(t, ir, StateOpen)
	list, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateResolved})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Skipped) != 1 || filepath.Base(list.Skipped[0].Path) != filepath.Base(dir) {
		t.Fatalf("a resolved/ list did not name the unreadable open/: %+v", list.Skipped)
	}
}
