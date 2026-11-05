package landing

import (
	"strings"
	"testing"
)

// TestARecordsCommitHoldsMarkdownOfOneMode: a relink of another record is its own; code,
// another mode and a status folder spelled in another case go to a person.
func TestARecordsCommitHoldsMarkdownOfOneMode(t *testing.T) {
	commit := func(t *testing.T, w *deltaWorld, mode, path string) (string, string) {
		t.Helper()
		base := w.head()
		blob := strings.TrimSpace(w.repo.Git("rev-parse", "HEAD:docs/how-to/a.md"))
		w.repo.Git("update-index", "--add", "--cacheinfo", mode+","+blob+","+path)
		w.repo.Git("commit", "-q", "-m", "chore(record): land lane-1 of run-1")
		return base, w.head()
	}
	for name, tc := range map[string]struct{ mode, path, want string }{
		"a record":               {"100644", ".abcd/work/issues/resolved/iss-1-a.md", ""},
		"a relinked plan":        {"100644", ".abcd/development/plans/p.md", ""},
		"a relinked decision":    {"100644", ".abcd/development/decisions/adrs/1.md", ""},
		"code":                   {"100644", "internal/a.go", "no markdown record"},
		"a script in a folder":   {"100644", ".abcd/work/issues/open/run.sh", "no markdown record"},
		"an executable record":   {"100755", ".abcd/work/issues/open/iss-1-a.md", "at mode 100755"},
		"a status folder's case": {"100644", ".abcd/work/issues/Resolved/iss-1-a.md", "spelled unlike"},
		"a root folder's case":   {"100644", ".abcd/Development/specs/closed/spc-1-a.md", "spelled unlike"},
		"a draft's folder case":  {"100644", ".abcd/development/intents/Drafts/itd-1-a.md", "spelled unlike"},
		"a kept folder's case":   {"100644", ".abcd/development/intents/SUPERSEDED/itd-1-a.md", "spelled unlike"},
		"another issue's move":   {"100644", ".abcd/work/issues/resolved/iss-2-a.md", "iss-2-a.md"},
	} {
		t.Run(name, func(t *testing.T) {
			w := newDeltaWorld(t)
			w.repo.Write(".abcd/development/intents/drafts/itd-0-a.md", "x\n")
			w.repo.Write(".abcd/development/intents/superseded/itd-0-b.md", "x\n")
			w.repo.Commit("the folders")
			base, head := commit(t, w, tc.mode, tc.path)
			why, err := RecordsCommitShape(w.repo.Root(), base, head, []string{"iss-1"})
			if err != nil {
				t.Fatal(err)
			}
			if (tc.want == "") != (why == "") || !strings.Contains(why, tc.want) {
				t.Fatalf("want %q, got %q", tc.want, why)
			}
		})
	}
}

// TestARecordsCommitIsReadAsPlainText: a NUL would show the pair a binary file.
func TestARecordsCommitIsReadAsPlainText(t *testing.T) {
	w := newDeltaWorld(t)
	base := w.head()
	w.repo.Write(".abcd/work/issues/resolved/iss-1-a.md", "a\x00b\n")
	w.repo.Commit("the record")
	why, err := RecordsCommitShape(w.repo.Root(), base, w.head(), []string{"iss-1"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(why, "iss-1-a.md") {
		t.Fatalf("it parks: %q", why)
	}
}

// TestARelinkOfAnotherIssuesRecordIsNoMove: a record changed where it stands stays open
// to the landing's own relink.
func TestARelinkOfAnotherIssuesRecordIsNoMove(t *testing.T) {
	w := newDeltaWorld(t)
	w.repo.Write(".abcd/work/issues/open/iss-2-a.md", "one\n")
	w.repo.Commit("the record")
	base := w.head()
	w.repo.Write(".abcd/work/issues/open/iss-2-a.md", "two\n")
	w.repo.Commit("the relink")
	why, err := RecordsCommitShape(w.repo.Root(), base, w.head(), []string{"iss-1"})
	if err != nil || why != "" {
		t.Fatalf("it stands: %q %v", why, err)
	}
}
