package landing

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestAReceiptACommitCarriesIsRefused: a preflight receipt is the checkout's
// own, so one git tracks came with a commit and is refused rather than counted
// as a preflight that passed.
func TestAReceiptACommitCarriesIsRefused(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write(".gitignore", TierRelDir+"/\n")
	repo.Commit("seed")
	sha := repo.Git("rev-parse", "HEAD")
	rel := ReceiptsRelDir + "/" + sha
	repo.Write(rel, "commit "+sha+"\n")
	if p, err := Receipt(repo.Root(), sha); err != nil || p == "" {
		t.Fatalf("this checkout's own receipt: %q, %v; want it found", p, err)
	}
	repo.Git("add", "-f", "--", rel)
	repo.Git("commit", "-q", "-m", "an unrelated change")
	if p, err := Receipt(repo.Root(), sha); err == nil || p != "" || !strings.Contains(err.Error(), "git tracks "+rel) {
		t.Fatalf("a receipt a commit carries: %q, %v; want it refused as tracked", p, err)
	}
}
