package reviews

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

func pinnedSummary(sha string) string {
	return "---\nreview_of_commit: " + sha + "\n---\n# Review summary\n"
}

// TestPinReadsTheFrontmatterKeyAndNothingElse pins the one reading of the key:
// a bare full sha in the leading frontmatter block. Everything else — a
// quoted value, an abbreviated sha, the key in the body, no block at all — is
// no pin, which is exactly what the reviews-charter gate refuses, so the board
// and the gate cannot disagree about a folder.
func TestPinReadsTheFrontmatterKeyAndNothingElse(t *testing.T) {
	full := strings.Repeat("a", 40)
	cases := []struct {
		name, text, want string
	}{
		{"bare sha", pinnedSummary(full), full},
		{"sha256 repository", pinnedSummary(strings.Repeat("b", 64)), strings.Repeat("b", 64)},
		{"crlf", "---\r\nreview_of_commit: " + full + "\r\n---\r\n# S\r\n", full},
		{"trailing comment", "---\nreview_of_commit: " + full + " # the tree read\n---\n", full},
		{"quoted", "---\nreview_of_commit: \"" + full + "\"\n---\n", ""},
		{"abbreviated", pinnedSummary("abcdef1"), ""},
		{"uppercase", pinnedSummary(strings.Repeat("A", 40)), ""},
		{"in the body", "# Summary\n\nreview_of_commit: " + full + "\n", ""},
		{"unclosed block", "---\nreview_of_commit: " + full + "\n# S\n", ""},
		{"no frontmatter", "# Summary\n", ""},
	}
	for _, c := range cases {
		if got := Pin([]byte(c.text)); got != c.want {
			t.Errorf("%s: Pin = %q, want %q", c.name, got, c.want)
		}
	}
}

// TestReadListsEveryFolderWithItsScopeAndPin: a dated review names its scope
// and, when the scope carries one, the spec it reviewed; a sha-keyed receipt
// directory is pinned by its own name and scoped by the gates it holds; the
// charter README and anything that is not a directory are not reviews.
func TestReadListsEveryFolderWithItsScopeAndPin(t *testing.T) {
	root := t.TempDir()
	sha := strings.Repeat("c", 40)
	receipt := strings.Repeat("d", 40)
	write := func(rel, body string) {
		t.Helper()
		p := filepath.Join(root, Dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("README.md", "# Reviews\n")
	write("2026-09-26-spc-2609211854150455-rp-reviews/00-summary.md", pinnedSummary(sha))
	write("2026-07-06-plan-consistency/00-summary.md", "# Legacy summary\n")
	write(receipt+"/docs-currency-reviewer.json", "{}")
	write(receipt+"/iss35-brief-surface-crosscheck.json", "{}")

	got, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 folders, got %d: %+v", len(got), got)
	}
	by := map[string]Entry{}
	for _, f := range got {
		by[f.Folder] = f
	}
	spec := by["2026-09-26-spc-2609211854150455-rp-reviews"]
	if spec.Kind != KindReview || spec.Spec != "spc-2609211854150455" || spec.Scope != "spc-2609211854150455-rp-reviews" || spec.ReviewOfCommit != sha {
		t.Fatalf("spec review read as %+v", spec)
	}
	legacy := by["2026-07-06-plan-consistency"]
	if legacy.Kind != KindReview || legacy.Spec != "" || legacy.Scope != "plan-consistency" || legacy.ReviewOfCommit != "" {
		t.Fatalf("legacy review read as %+v", legacy)
	}
	rc := by[receipt]
	if rc.Kind != KindReceipt || rc.ReviewOfCommit != receipt || rc.Scope != "docs-currency-reviewer, iss35-brief-surface-crosscheck" {
		t.Fatalf("receipt read as %+v", rc)
	}
}

// TestReadWithNoReviewsTreeIsEmpty: a repository without the tree has no rows
// and no error — the board omits the block rather than failing.
func TestReadWithNoReviewsTreeIsEmpty(t *testing.T) {
	got, err := Read(t.TempDir())
	if err != nil || len(got) != 0 {
		t.Fatalf("Read = %+v, %v", got, err)
	}
}

// TestStalenessCountsTheDefaultBranchSinceThePin is criteria 2 and 5 against
// real history: the count is the commits the default branch holds past the
// pin, a count past the threshold is flagged and one at it is not, a pin git
// does not know is unreachable rather than fresh, an unpinned folder is named
// as such, and the rows come stalest first.
func TestStalenessCountsTheDefaultBranchSinceThePin(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Commit("c0")
	pins := []string{r.Git("rev-parse", "HEAD")}
	for i := 1; i <= StaleAfter+1; i++ {
		r.Commit(fmt.Sprintf("c%d", i))
		pins = append(pins, r.Git("rev-parse", "HEAD"))
	}
	past := pins[0]                 // 21 commits behind main
	at := pins[1]                   // 20 commits behind: not past the threshold
	fresh := pins[len(pins)-1]      // main itself
	gone := strings.Repeat("e", 40) // no such object
	r.Write(Dir+"/2026-09-01-past/00-summary.md", pinnedSummary(past))
	r.Write(Dir+"/2026-09-02-at/00-summary.md", pinnedSummary(at))
	r.Write(Dir+"/2026-09-03-fresh/00-summary.md", pinnedSummary(fresh))
	r.Write(Dir+"/2026-07-06-legacy/00-summary.md", "# no pin\n")
	r.Write(Dir+"/"+gone+"/gate.json", "{}")

	b, err := Staleness(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	if b.Threshold != StaleAfter || b.DefaultRef != "main" {
		t.Fatalf("threshold %d, default ref %q", b.Threshold, b.DefaultRef)
	}
	order := make([]string, 0, len(b.Rows))
	for _, row := range b.Rows {
		order = append(order, row.Folder)
	}
	want := []string{"2026-09-01-past", "2026-09-02-at", "2026-09-03-fresh", gone, "2026-07-06-legacy"}
	if strings.Join(order, " ") != strings.Join(want, " ") {
		t.Fatalf("order %v, want %v", order, want)
	}
	check := func(i int, state string, since int, stale bool) {
		t.Helper()
		row := b.Rows[i]
		if row.State != state || row.Stale != stale {
			t.Fatalf("%s: state %q stale %v, want %q %v", row.Folder, row.State, row.Stale, state, stale)
		}
		if since < 0 {
			if row.CommitsSince != nil {
				t.Fatalf("%s: commits_since %d, want none", row.Folder, *row.CommitsSince)
			}
			return
		}
		if row.CommitsSince == nil || *row.CommitsSince != since {
			t.Fatalf("%s: commits_since %v, want %d", row.Folder, row.CommitsSince, since)
		}
	}
	check(0, StatePinned, StaleAfter+1, true)
	check(1, StatePinned, StaleAfter, false)
	check(2, StatePinned, 0, false)
	check(3, StateUnreachable, -1, false)
	check(4, StateUnpinned, -1, false)
	if b.StaleCount() != 1 {
		t.Fatalf("StaleCount = %d, want 1", b.StaleCount())
	}
}
