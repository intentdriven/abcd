package claim

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestTheSpotcheckBranchIsAbsentBesideABranchEndingInItsName: an origin that
// holds no spot-check branch, only one whose name ends in the spot-check's
// ref (refs/heads/x/refs/heads/abcd-spotcheck, which `git ls-remote --heads`
// lists for it), answers the spot-check's fetch as a branch origin lacks,
// never as a fetch that failed: only the line naming the branch exactly says
// origin holds it.
func TestTheSpotcheckBranchIsAbsentBesideABranchEndingInItsName(t *testing.T) {
	bare := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, bare)
	r := gittest.NewRepo(t)
	r.Write("a.txt", "a\n")
	r.Commit("the base")
	r.Git("remote", "add", Remote, bare)
	r.Git("push", "-q", Remote, "HEAD:refs/heads/x/"+spotcheckBranchRef)
	found, err := FetchBranch(context.Background(), r.Root(), spotcheckBranchRef, spotcheckTrackingRef)
	if err != nil || found {
		t.Fatalf("FetchBranch = %v, %v; want origin to hold no spot-check branch", found, err)
	}
	if out := r.Git("for-each-ref", spotcheckTrackingRef); out != "" {
		t.Fatalf("a fetch of a branch origin lacks moved the tracking ref: %s", out)
	}
}
