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
		"an executable record":   {"100755", ".abcd/work/issues/open/iss-2-a.md", "at mode 100755"},
		"a status folder's case": {"100644", ".abcd/work/issues/Resolved/iss-3-a.md", "spelled unlike"},
		"a record folder's case": {"100644", ".abcd/Development/specs/closed/spc-1-a.md", "spelled unlike"},
	} {
		t.Run(name, func(t *testing.T) {
			w := newDeltaWorld(t)
			base, head := commit(t, w, tc.mode, tc.path)
			why, err := RecordsCommitShape(w.repo.Root(), base, head)
			if err != nil {
				t.Fatal(err)
			}
			if (tc.want == "") != (why == "") || !strings.Contains(why, tc.want) {
				t.Fatalf("want %q, got %q", tc.want, why)
			}
		})
	}
}
