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
		t.Fatalf("want the reviews heading and three rows, got:\n%s", stdout)
	}
	if !strings.Contains(lines[0], "1 past 20 commits") || !strings.Contains(lines[0], "main") {
		t.Fatalf("heading %q does not name the flagged count and the branch", lines[0])
	}
	for i, want := range [][]string{
		{"!", "21", old[:7], "2026-09-01-spc-7-plan"},
		{"0", head[:7], "docs-currency-reviewer"},
		{"unpinned", "2026-07-06-legacy"},
	} {
		for _, w := range want {
			if !strings.Contains(lines[i+1], w) {
				t.Fatalf("row %d %q does not carry %q", i+1, lines[i+1], w)
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
