package spotcheck

import (
	"context"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestTheLocalCopyIsPushedAsItIsOnlyWhenItStoresOriginsCopy: the local copy
// is pushed as it is only when its commit stores origin's; a replace ref
// giving the local commit origin's as a parent answers no, so the records are
// laid over origin's copy instead.
func TestTheLocalCopyIsPushedAsItIsOnlyWhenItStoresOriginsCopy(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Commit("base")
	base := r.Git("rev-parse", "HEAD")
	tracking := r.Git("commit-tree", base+"^{tree}", "-p", base, "-m", "origin's copy")
	local := r.Git("commit-tree", base+"^{tree}", "-p", base, "-m", "the local copy")
	r.Git("replace", "--graft", local, base, tracking)
	if err := r.Git("merge-base", "--is-ancestor", tracking, local); err != "" {
		t.Fatalf("git presents origin's copy on the local one: %q", err)
	}
	if ok, err := isAncestor(context.Background(), r.Root(), tracking, local); err != nil || ok {
		t.Fatalf("isAncestor = %v, %v; want no from the stored parents", ok, err)
	}
	if ok, err := isAncestor(context.Background(), r.Root(), base, local); err != nil || !ok {
		t.Fatalf("isAncestor(base) = %v, %v; want yes", ok, err)
	}
}
