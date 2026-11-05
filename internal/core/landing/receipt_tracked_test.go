package landing

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestAReceiptACommitCarriesIsRefused: a preflight or review receipt is the
// checkout's own, so one git tracks came with a commit and is refused rather
// than counted as a preflight that passed or a review the pair gave.
func TestAReceiptACommitCarriesIsRefused(t *testing.T) {
	for name, c := range map[string]struct {
		dir  string
		find func(root, sha string) (string, error)
	}{
		"a preflight receipt": {ReceiptsRelDir, Receipt},
		"a review receipt":    {ReviewReceiptsRelDir, reviewReceipt},
	} {
		t.Run(name, func(t *testing.T) {
			repo := gittest.NewRepo(t)
			repo.Write(".gitignore", TierRelDir+"/\n")
			repo.Commit("seed")
			sha := repo.Git("rev-parse", "HEAD")
			rel := c.dir + "/" + sha
			repo.Write(rel, "commit "+sha+"\n")
			if p, err := c.find(repo.Root(), sha); err != nil || p == "" {
				t.Fatalf("this checkout's own receipt: %q, %v; want it found", p, err)
			}
			repo.Git("add", "-f", "--", rel)
			repo.Git("commit", "-q", "-m", "an unrelated change")
			if p, err := c.find(repo.Root(), sha); err == nil || p != "" || !strings.Contains(err.Error(), "git tracks "+rel) {
				t.Fatalf("a receipt a commit carries: %q, %v; want it refused as tracked", p, err)
			}
		})
	}
}

// The merge gate reads a review receipt through every worktree's reading, so a
// receipt a commit carries counts for nothing there, or as a base's full pair,
// however well it reads.
func TestTheGateRefusesAReviewReceiptACommitCarries(t *testing.T) {
	repo, sha := reviewRepo(t)
	rel := ReviewReceiptsRelDir + "/" + sha
	body := `{"commit":"` + sha + `","reviews":[{"name":"a","verdict":"SHIP","sha256":"x"},{"name":"b","verdict":"APPROVE","sha256":"y"}]}`
	repo.Write(rel, body+"\n")
	if p := readReviewProof(repo.Root(), sha, ""); !p.reviewed {
		t.Fatalf("this checkout's own receipt read as %+v", p)
	}
	if err := hasFullPair(repo.Root(), sha); err != nil {
		t.Fatalf("this checkout's own receipt: %v", err)
	}
	repo.Git("add", "-f", "--", rel)
	repo.Git("commit", "-q", "-m", "an unrelated change")
	if p := readReviewProof(repo.Root(), sha, ""); p.reviewed || !strings.Contains(p.unreadable, "git tracks "+rel) {
		t.Fatalf("a receipt a commit carries read as %+v; want it refused as tracked", p)
	}
	if err := hasFullPair(repo.Root(), sha); err == nil || !strings.Contains(err.Error(), "git tracks "+rel) {
		t.Fatalf("err = %v; want a base's tracked receipt refused", err)
	}
}
