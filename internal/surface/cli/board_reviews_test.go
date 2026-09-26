package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

func reviewLines(text string) []string {
	var out []string
	in := false
	for _, l := range strings.Split(text, "\n") {
		switch {
		case strings.HasPrefix(l, "  reviews:"):
			in = true
			out = append(out, l)
		case in && strings.HasPrefix(l, "    "):
			out = append(out, l)
		default:
			in = false
		}
	}
	return out
}

// TestBoardListsEachReviewWithItsAgeStalestFirst is itd-28 criteria 2, 3 and
// 5 at the front door: every folder under the reviews tree is a row naming
// what it reviewed (the spec id read from the folder's name), the commits the
// default branch has moved since its pin, and a flag past twenty; the JSON
// carries the same rows and the threshold.
func TestBoardListsEachReviewWithItsAgeStalestFirst(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r := gittest.NewRepo(t)
	r.Commit("c0")
	old := r.Git("rev-parse", "HEAD")
	for i := 1; i <= 21; i++ {
		r.Commit(fmt.Sprintf("c%d", i))
	}
	head := r.Git("rev-parse", "HEAD")
	r.Write(".abcd/work/reviews/2026-09-01-spc-7-plan/00-summary.md", "---\nreview_of_commit: "+old+"\n---\n# S\n")
	r.Write(".abcd/work/reviews/2026-07-06-legacy/00-summary.md", "# S\n")
	r.Write(".abcd/work/reviews/"+head+"/docs-currency-reviewer.json", "{}")
	t.Chdir(r.Root())

	stdout, stderr, err := runCLISplit(t)
	if err != nil {
		t.Fatalf("board: %v\n%s", err, stderr)
	}
	lines := reviewLines(stdout)
	if len(lines) != 4 {
		t.Fatalf("want the reviews heading, two review rows and the receipts line, got:\n%s", stdout)
	}
	if !strings.Contains(lines[0], "2 review folders, 1 past 20 commits") || !strings.Contains(lines[0], "main") || !strings.Contains(lines[0], "re-run those marked !") {
		t.Fatalf("heading %q does not name the review count, the flagged count, the branch and the re-run", lines[0])
	}
	for i, want := range [][]string{
		{"!", "21", old[:7], "2026-09-01-spc-7-plan"},
		{"unpinned", "2026-07-06-legacy"},
		{"receipts:", "1 release receipt", "the release it gated is 0 commits behind main", "--json"},
	} {
		for _, w := range want {
			if !strings.Contains(lines[i+1], w) {
				t.Fatalf("line %d %q does not carry %q", i+1, lines[i+1], w)
			}
		}
	}
	if strings.Contains(lines[2], "!") || strings.Contains(lines[3], "!") {
		t.Fatalf("a row at or under the threshold is flagged:\n%s", stdout)
	}

	var got struct {
		Reviews struct {
			Threshold  int    `json:"threshold"`
			DefaultRef string `json:"default_ref"`
			Rows       []struct {
				Folder         string `json:"folder"`
				Kind           string `json:"kind"`
				Spec           string `json:"spec"`
				ReviewOfCommit string `json:"review_of_commit"`
				State          string `json:"state"`
				CommitsSince   *int   `json:"commits_since"`
				Stale          bool   `json:"stale"`
			} `json:"rows"`
		} `json:"reviews"`
	}
	if err := json.Unmarshal(runCLI(t, "--json"), &got); err != nil {
		t.Fatal(err)
	}
	rv := got.Reviews
	if rv.Threshold != 20 || rv.DefaultRef != "main" || len(rv.Rows) != 3 {
		t.Fatalf("reviews payload = %+v", rv)
	}
	first := rv.Rows[0]
	if first.Spec != "spc-7" || first.ReviewOfCommit != old || first.CommitsSince == nil || *first.CommitsSince != 21 || !first.Stale || first.State != "pinned" {
		t.Fatalf("stalest row = %+v", first)
	}
	if last := rv.Rows[2]; last.State != "unpinned" || last.CommitsSince != nil || last.Stale {
		t.Fatalf("legacy row = %+v", last)
	}
}

// TestBoardOmitsReviewsWithNoReviewFolders: no tree, no block — the JSON
// omits the member rather than nulling it.
func TestBoardOmitsReviewsWithNoReviewFolders(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r := gittest.NewRepo(t)
	r.Commit("c0")
	t.Chdir(r.Root())
	stdout, _, err := runCLISplit(t)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout, "reviews:") {
		t.Fatalf("a board with no review folders rendered a reviews block:\n%s", stdout)
	}
	if strings.Contains(string(runCLI(t, "--json")), `"reviews"`) {
		t.Fatal("--json carries a reviews member with no review folders")
	}
}

// TestBoardFoldsReleaseReceiptsIntoOneTruthfulLine: a release receipt gates
// the release it names and is never re-run, and RD002 keeps every one, so each
// release adds a receipt that is permanently past the threshold. The text
// render therefore lists the dated reviews and folds the receipts into one line
// that says how far behind the oldest release it gated is and how many pins
// this history no longer holds; the imperative to re-run appears only for a
// flagged review, and --json still carries every receipt as a row.
func TestBoardFoldsReleaseReceiptsIntoOneTruthfulLine(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r := gittest.NewRepo(t)
	var pins []string
	for i := 0; i <= 30; i++ {
		r.Commit(fmt.Sprintf("c%d", i))
		pins = append(pins, r.Git("rev-parse", "HEAD"))
	}
	head := pins[len(pins)-1]
	for _, p := range []string{pins[0], pins[5], head, strings.Repeat("e", 40)} {
		r.Write(".abcd/work/reviews/"+p+"/docs-currency-reviewer.json", "{}")
	}
	r.Write(".abcd/work/reviews/2026-09-01-spc-7-plan/00-summary.md", "---\nreview_of_commit: "+head+"\n---\n# S\n")
	t.Chdir(r.Root())

	stdout, stderr, err := runCLISplit(t)
	if err != nil {
		t.Fatalf("board: %v\n%s", err, stderr)
	}
	lines := reviewLines(stdout)
	if len(lines) != 3 {
		t.Fatalf("want the heading, one review row and one receipts line, got:\n%s", stdout)
	}
	if !strings.Contains(lines[0], "1 review folder, 0 past 20 commits") || strings.Contains(lines[0], "re-run") {
		t.Fatalf("heading %q: with no flagged review it counts the reviews alone and asks for no re-run", lines[0])
	}
	for _, w := range []string{"receipts:", "4 release receipts", "the oldest release gated is 30 commits behind main", "1 with a pin not in this history", "not re-run", "--json"} {
		if !strings.Contains(lines[2], w) {
			t.Fatalf("receipts line %q does not carry %q", lines[2], w)
		}
	}
	if strings.Contains(stdout, "!") {
		t.Fatalf("a receipt is flagged for a re-run nobody makes:\n%s", stdout)
	}

	var got struct {
		Reviews struct {
			Rows []struct {
				Kind  string `json:"kind"`
				Stale bool   `json:"stale"`
			} `json:"rows"`
		} `json:"reviews"`
	}
	if err := json.Unmarshal(runCLI(t, "--json"), &got); err != nil {
		t.Fatal(err)
	}
	receipts, stale := 0, 0
	for _, row := range got.Reviews.Rows {
		if row.Kind == "receipt" {
			receipts++
			if row.Stale {
				stale++
			}
		}
	}
	if len(got.Reviews.Rows) != 5 || receipts != 4 || stale != 2 {
		t.Fatalf("--json carries %d rows, %d receipts, %d stale; want every receipt as a row (5, 4, 2)", len(got.Reviews.Rows), receipts, stale)
	}
}
