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

// TestStalenessCountsEveryCommitOnAMergeHeavyHistory pins what Row.CommitsSince
// documents on the history shape a merge-based workflow makes: the count is
// every commit the default branch holds that the pin does not, branch commits
// and merge commits alike (`rev-list --count <pin>..main`), not the first-parent
// walk a linear history cannot tell apart from it.
func TestStalenessCountsEveryCommitOnAMergeHeavyHistory(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Commit("c0")
	pin := r.Git("rev-parse", "HEAD")
	for i := 1; i <= 3; i++ {
		branch := fmt.Sprintf("f%d", i)
		r.Git("checkout", "-q", "-b", branch)
		r.Commit(branch + "-a")
		r.Commit(branch + "-b")
		r.Git("checkout", "-q", "main")
		r.Git("merge", "-q", "--no-ff", "-m", "merge "+branch, branch)
	}
	r.Write(Dir+"/2026-09-01-merged/00-summary.md", pinnedSummary(pin))

	b, err := Staleness(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	if first := r.Git("rev-list", "--count", "--first-parent", pin+"..main"); first != "3" {
		t.Fatalf("fixture: first-parent count %s, want 3", first)
	}
	// Three branches of two commits each, and a merge commit apiece.
	if len(b.Rows) != 1 || b.Rows[0].CommitsSince == nil || *b.Rows[0].CommitsSince != 9 || b.Rows[0].Stale {
		t.Fatalf("merge-heavy row = %+v, want pinned with 9 commits since and not stale", b.Rows)
	}
}

// TestStalenessWithNoDefaultBranchCountsAgainstHead pins the fallback: where
// no default branch resolves, the board names HEAD and counts against it
// rather than refusing or counting nothing.
func TestStalenessWithNoDefaultBranchCountsAgainstHead(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Commit("c0")
	pin := r.Git("rev-parse", "HEAD")
	for i := 1; i <= 3; i++ {
		r.Commit(fmt.Sprintf("c%d", i))
	}
	r.Git("branch", "-m", "main", "feature")
	r.Write(Dir+"/2026-09-01-headless/00-summary.md", pinnedSummary(pin))

	b, err := Staleness(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	if b.DefaultRef != "HEAD" {
		t.Fatalf("default ref %q, want HEAD", b.DefaultRef)
	}
	if len(b.Rows) != 1 || b.Rows[0].State != StatePinned || b.Rows[0].CommitsSince == nil || *b.Rows[0].CommitsSince != 3 {
		t.Fatalf("row = %+v, want pinned with 3 commits since HEAD", b.Rows)
	}
}

// TestReadShowsUnpinnedWhatTheGateRefuses is the board's half of the parity
// scripts/check-reviews-cases.sh proves from the gate's side: a symlinked
// summary, a summary past maxSummaryBytes, and a NUL byte anywhere in the
// frontmatter block are no pin, and a symlinked folder is not read at all. At
// exactly the cap, and with a NUL in the body past the block, the pin reads.
func TestReadShowsUnpinnedWhatTheGateRefuses(t *testing.T) {
	root := t.TempDir()
	sha := strings.Repeat("f", 40)
	write := func(rel string, body []byte) {
		t.Helper()
		p := filepath.Join(root, Dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	padded := func(n int) []byte {
		b := []byte(pinnedSummary(sha))
		return append(b, []byte(strings.Repeat("a", n-len(b)))...)
	}
	write("2026-09-26-linked/01-real.md", []byte(pinnedSummary(sha)))
	if err := os.Symlink("01-real.md", filepath.Join(root, Dir, "2026-09-26-linked", summaryFile)); err != nil {
		t.Fatal(err)
	}
	write("../elsewhere/00-summary.md", []byte(pinnedSummary(sha)))
	if err := os.Symlink(filepath.Join("..", "elsewhere"), filepath.Join(root, Dir, "2026-09-26-linked-dir")); err != nil {
		t.Fatal(err)
	}
	write("2026-09-26-oversize/00-summary.md", padded(maxSummaryBytes+1))
	write("2026-09-26-at-cap/00-summary.md", padded(maxSummaryBytes))
	write("2026-09-26-nul-value/00-summary.md", []byte("---\nreview_of_commit: "+sha+"\x00\n---\n# S\n"))
	write("2026-09-26-nul-key/00-summary.md", []byte("---\nreview_of\x00_commit: "+sha+"\n---\n# S\n"))
	write("2026-09-26-nul-fence/00-summary.md", []byte("---\nreview_of_commit: "+sha+"\n---\x00\n# S\n"))
	write("2026-09-26-nul-body/00-summary.md", []byte(pinnedSummary(sha)+"\x00\n"))

	got, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	pins := map[string]string{}
	for _, e := range got {
		pins[e.Folder] = e.ReviewOfCommit
	}
	if _, ok := pins["2026-09-26-linked-dir"]; ok {
		t.Fatal("a symlinked review folder was read; the board follows no symlinked entry")
	}
	want := map[string]string{
		"2026-09-26-linked":    "",
		"2026-09-26-oversize":  "",
		"2026-09-26-at-cap":    sha,
		"2026-09-26-nul-value": "",
		"2026-09-26-nul-key":   "",
		"2026-09-26-nul-fence": "",
		"2026-09-26-nul-body":  sha,
	}
	for folder, pin := range want {
		if got, ok := pins[folder]; !ok || got != pin {
			t.Errorf("%s: pin %q (read %v), want %q", folder, got, ok, pin)
		}
	}
}
