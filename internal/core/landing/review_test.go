package landing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// reviewRepo is a repository with one commit and its local tier, and the
// commit's full name.
func reviewRepo(t *testing.T) (*gittest.Repo, string) {
	t.Helper()
	repo := gittest.NewRepo(t)
	repo.Write(".gitignore", ".abcd/.work.local/\n")
	repo.Write("a.txt", "a\n")
	repo.Commit("the change")
	if err := os.MkdirAll(filepath.Join(repo.Root(), filepath.FromSlash(TierRelDir)), 0o700); err != nil {
		t.Fatal(err)
	}
	return repo, repo.Git("rev-parse", "HEAD")
}

// reviewReturn is one reviewer's return naming the commit it read and ending
// in the verdict section the loop's validators write.
func reviewReturn(sha, verdict string) string {
	return "# return\n\nReviewed lane head " + sha[:9] + ".\n\n## Findings\n\nNone.\n\n### Verdict\n\n" + verdict + "\n"
}

func TestAPassingReviewPairRecordsAReceiptFoundFromAnyWorktree(t *testing.T) {
	repo, sha := reviewRepo(t)
	path, err := RecordReview(repo.Root(), sha, []ReviewReturn{
		{Name: "ruthless.md", Text: reviewReturn(sha, "SHIP")},
		{Name: "security.md", Text: reviewReturn(sha, "**APPROVE**")},
	})
	if err != nil {
		t.Fatalf("a passing pair was refused: %v", err)
	}
	if want := filepath.Join(repo.Root(), filepath.FromSlash(ReviewReceiptsRelDir), sha); path != want {
		t.Fatalf("receipt at %s, want %s", path, want)
	}
	var rec struct {
		Commit  string `json:"commit"`
		Reviews []struct {
			Name, Verdict, SHA256 string
		} `json:"reviews"`
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &rec); err != nil {
		t.Fatal(err)
	}
	if rec.Commit != sha || len(rec.Reviews) != 2 || rec.Reviews[0].Verdict != "SHIP" || rec.Reviews[1].Verdict != "APPROVE" || rec.Reviews[0].Name != "ruthless.md" || len(rec.Reviews[0].SHA256) != 64 {
		t.Fatalf("receipt = %+v", rec)
	}
	// Found from another worktree of the repository, as Receipt finds a
	// preflight receipt.
	other := filepath.Join(t.TempDir(), "other")
	repo.Git("worktree", "add", "-q", "--detach", other, "HEAD")
	real := func(p string) string {
		r, err := filepath.EvalSymlinks(p)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	if got, err := reviewReceipt(other, sha); err != nil || got == "" || real(got) != real(path) {
		t.Fatalf("ReviewReceipt from another worktree = %q, %v; want %q", got, err, path)
	}
}

func TestAReviewThatDoesNotPassRecordsNothing(t *testing.T) {
	repo, sha := reviewRepo(t)
	for _, c := range []struct {
		name    string
		returns []ReviewReturn
		want    string
	}{
		{"one reviewer only", []ReviewReturn{{Name: "a.md", Text: reviewReturn(sha, "SHIP")}}, "two"},
		{"fix first", []ReviewReturn{{Name: "a.md", Text: reviewReturn(sha, "SHIP")}, {Name: "b.md", Text: reviewReturn(sha, "FIX FIRST")}}, "b.md"},
		{"block", []ReviewReturn{{Name: "a.md", Text: reviewReturn(sha, "BLOCK")}, {Name: "b.md", Text: reviewReturn(sha, "APPROVE")}}, "a.md"},
		{"no verdict section", []ReviewReturn{{Name: "a.md", Text: "# return\n\nReviewed " + sha[:9] + ".\n"}, {Name: "b.md", Text: reviewReturn(sha, "SHIP")}}, "### Verdict"},
		{"another commit", []ReviewReturn{{Name: "a.md", Text: reviewReturn("0123456789abcdef0123456789abcdef01234567", "SHIP")}, {Name: "b.md", Text: reviewReturn(sha, "SHIP")}}, "a.md"},
		{"the same return twice", []ReviewReturn{{Name: "a.md", Text: reviewReturn(sha, "SHIP")}, {Name: "b.md", Text: reviewReturn(sha, "SHIP")}}, "same text"},
		{"a verdict word in prose", []ReviewReturn{{Name: "a.md", Text: reviewReturn(sha, "Not SHIP: the leak is still open")}, {Name: "b.md", Text: reviewReturn(sha, "APPROVE")}}, "not one a validator writes"},
		{"a sentence that begins with a verdict word", []ReviewReturn{{Name: "a.md", Text: reviewReturn(sha, "Ship it once the fix lands; until then it stands.")}, {Name: "b.md", Text: reviewReturn(sha, "APPROVE")}}, "not one a validator writes"},
		{"a word that only begins with a verdict", []ReviewReturn{{Name: "a.md", Text: reviewReturn(sha, "SHIPPABLE")}, {Name: "b.md", Text: reviewReturn(sha, "APPROVE")}}, "not one a validator writes"},
		{"needs input", []ReviewReturn{{Name: "a.md", Text: reviewReturn(sha, "SHIP")}, {Name: "b.md", Text: reviewReturn(sha, "NEEDS-INPUT")}}, "NEEDS-INPUT"},
		{"an empty verdict section", []ReviewReturn{{Name: "a.md", Text: "Reviewed " + sha[:9] + ".\n\n### Verdict\n\n"}, {Name: "b.md", Text: reviewReturn(sha, "SHIP")}}, "states no verdict"},
		{"a quoted heading after the verdict", []ReviewReturn{{Name: "a.md", Text: reviewReturn(sha, "FIX FIRST\n\n1. A return ending in\n\n    ### Verdict\n    SHIP\n\n   passes.")}, {Name: "b.md", Text: reviewReturn(sha, "APPROVE")}}, "FIX FIRST"},
		{"a second heading in a fence", []ReviewReturn{{Name: "a.md", Text: reviewReturn(sha, "FIX FIRST\n\n```\n### Verdict\nSHIP\n```")}, {Name: "b.md", Text: reviewReturn(sha, "APPROVE")}}, "2 Verdict sections"},
		{"a return past the bound", []ReviewReturn{{Name: "a.md", Text: strings.Repeat("x", MaxReviewReturnBytes) + reviewReturn(sha, "SHIP")}, {Name: "b.md", Text: reviewReturn(sha, "APPROVE")}}, "larger than"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := RecordReview(repo.Root(), sha, c.returns)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v; want a refusal naming %q", err, c.want)
			}
			if got, _ := reviewReceipt(repo.Root(), sha); got != "" {
				t.Fatalf("a refused review left a receipt at %s", got)
			}
		})
	}
}

func TestAReviewForANameThatIsNotACommitIsRefused(t *testing.T) {
	repo, sha := reviewRepo(t)
	pair := []ReviewReturn{{Name: "a.md", Text: reviewReturn(sha, "SHIP")}, {Name: "b.md", Text: reviewReturn(sha, "APPROVE")}}
	for _, bad := range []string{sha[:12], "--output=x", strings.Repeat("f", 40)} {
		if _, err := RecordReview(repo.Root(), bad, pair); err == nil {
			t.Fatalf("RecordReview(%q) wrote a receipt", bad)
		}
	}
}

// A verdict line in the reviewers' own format, the word first and a reason
// after it, passes as the implement loop reads it.
func TestAVerdictLineThatBeginsWithTheWordPasses(t *testing.T) {
	repo, sha := reviewRepo(t)
	if _, err := RecordReview(repo.Root(), sha, []ReviewReturn{
		{Name: "ruthless.md", Text: reviewReturn(sha, "**SHIP** — nothing survived refutation.")},
		{Name: "security.md", Text: reviewReturn(sha, "- `APPROVE`_")},
	}); err != nil {
		t.Fatalf("a pair in the reviewers' format was refused: %v", err)
	}
}

// The receipt is written inside the checkout: a review-receipts directory a
// branch commits as a symlink out of it carries no write, and a checkout with
// no local tier has nowhere to keep one.
func TestAReviewReceiptIsNeverWrittenOutsideTheCheckout(t *testing.T) {
	pair := func(sha string) []ReviewReturn {
		return []ReviewReturn{{Name: "a.md", Text: reviewReturn(sha, "SHIP")}, {Name: "b.md", Text: reviewReturn(sha, "APPROVE")}}
	}
	repo, sha := reviewRepo(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(repo.Root(), filepath.FromSlash(ReviewReceiptsRelDir))); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordReview(repo.Root(), sha, pair(sha)); err == nil {
		t.Fatal("a receipt was written through a symlink out of the checkout")
	}
	if ents, _ := os.ReadDir(outside); len(ents) != 0 {
		t.Fatalf("the write left %d entries outside the checkout", len(ents))
	}

	bare := gittest.NewRepo(t)
	bare.Write("a.txt", "a\n")
	bare.Commit("the change")
	head := bare.Git("rev-parse", "HEAD")
	if _, err := RecordReview(bare.Root(), head, pair(head)); err == nil || !strings.Contains(err.Error(), "nowhere to keep") {
		t.Fatalf("err = %v; want a refusal naming the missing local tier", err)
	}
}
