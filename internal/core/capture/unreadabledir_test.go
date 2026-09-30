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
// A capture never gets that far: its mint cannot tell a free id from one the
// unreadable directory holds, so it faults naming the directory
// (iss-2609261241121312). The quoted-text intent create writes nothing into
// the ledger and still matches against it, so the unread outcome is what it
// files with: the content comes back unlinked.
func TestMatchReportsAnUnreadableStatusDirectoryAsUnread(t *testing.T) {
	repo, ir := ledger(t)
	captureText(t, repo, ir, plantedFinding, nil)
	captureText(t, repo, ir, matchFiller1, nil)
	unreadableStatusDir(t, ir, StateResolved)

	if _, err := testCapture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: plantedDouble, Severity: SeverityMinor,
		Category: "bug", Source: "user-observation", FoundDuring: "t", Match: bundled(),
	}); err == nil || !strings.Contains(err.Error(), statusDirName[StateResolved]+"/") {
		t.Fatalf("a capture over an unreadable resolved/ was not refused naming it: %v", err)
	}
	if _, err := MatchCandidates(repo, *bundled()); err == nil || !strings.Contains(err.Error(), statusDirName[StateResolved]+"/") {
		t.Fatalf("the intent create's candidate set read an unreadable resolved/ as empty: %v", err)
	}
	const content = "---\nid: planted\n---\n"
	got, o := matchAndLink(repo, ir, *bundled(), plantedDouble, nil, content, map[string]any{})
	if o == nil || !strings.Contains(o.Skipped, "could not be read") {
		t.Fatalf("an unreadable resolved/ did not read as an unread record set: %+v", o)
	}
	if len(o.Matches) != 0 {
		t.Fatalf("an unread record set still linked: %+v", o.Matches)
	}
	if got != content {
		t.Fatalf("an unread match changed the content it was to file:\n%s", got)
	}
}

// The board says the same thing: capture list and capture status name an
// unreadable status directory rather than counting it as a directory with no
// records in it. The one mechanism is readStatusDir's (iss-2609261241121312):
// the directory is a fault naming it, ledger-relatively, so neither surface
// reports a count for a ledger it did not read. An absent directory stays
// silent — a virgin ledger has none.
func TestListAndStatusReportAnUnreadableStatusDirectory(t *testing.T) {
	repo, ir := ledger(t)
	captureText(t, repo, ir, plantedFinding, nil)
	unreadableStatusDir(t, ir, StateResolved)

	named := func(err error) bool {
		return err != nil && strings.Contains(err.Error(), statusDirName[StateResolved]+"/") &&
			!strings.Contains(err.Error(), statusDirName[StateWontfix]+"/") && !strings.Contains(err.Error(), repo)
	}
	if list, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir}); !named(err) {
		t.Fatalf("capture list did not name the unreadable resolved/ alone: %v (%d rows)", err, len(list.Issues))
	}
	if st, err := Status(StatusRequest{RepoRoot: repo, IssuesRoot: ir}); !named(err) {
		t.Fatalf("capture status did not name the unreadable resolved/ alone: %v (%d resolved)", err, st.ResolvedCount)
	}
}

// A list scoped to resolved/ still reads open/ for the blocked_by projection;
// an unreadable open/ there leaves every dependent looking unblocked, so the
// list names it even though the rows come from another directory.
func TestAScopedListNamesAnUnreadableOpenDirectory(t *testing.T) {
	repo, ir := ledger(t)
	if err := os.MkdirAll(filepath.Join(ir, statusDirName[StateResolved]), 0o755); err != nil {
		t.Fatal(err)
	}
	unreadableStatusDir(t, ir, StateOpen)
	_, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateResolved})
	if err == nil || !strings.Contains(err.Error(), statusDirName[StateOpen]+"/") {
		t.Fatalf("a resolved/ list did not name the unreadable open/: %v", err)
	}
}
