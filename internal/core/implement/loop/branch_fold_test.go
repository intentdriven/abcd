package loop

import "testing"

// TestABranchCheckedOutUnderAnotherCaseIsTheLanesOnAFoldingFilesystem: on a
// filesystem that folds case (macOS by default), git resolves a loose ref
// file whatever the case it is named in, so a state naming build/Lane-1 reads
// and deletes the branch build/lane-1. The check that refuses to delete a
// branch another checkout stands on compared the names byte for byte, so a
// worktree on build/lane-1 did not count as standing on build/Lane-1 and its
// HEAD was left naming nothing. Plain `git branch -D` has the same gap, so
// this is help, not a fix of abcd's own route; it changes nothing about how
// the repository works.
func TestABranchCheckedOutUnderAnotherCaseIsTheLanesOnAFoldingFilesystem(t *testing.T) {
	folding := caseFoldingRefs
	t.Cleanup(func() { caseFoldingRefs = folding })

	caseFoldingRefs = func() bool { return true }
	if !sameBranchRef("refs/heads/build/lane-1", "build/Lane-1") {
		t.Fatal("on a folding filesystem a listed branch in another case is the lane's")
	}
	if sameBranchRef("refs/heads/build/lane-2", "build/Lane-1") {
		t.Fatal("a different branch is never the lane's")
	}
	// macOS also folds Unicode normalisation: a state spelling the branch in
	// NFD names the NFC branch a worktree stands on.
	if !sameBranchRef("refs/heads/build/lan\u00e9-1", "build/lane\u0301-1") {
		t.Fatal("on a folding filesystem a listed branch in another normalisation is the lane's")
	}
	caseFoldingRefs = func() bool { return false }
	if sameBranchRef("refs/heads/build/lane-1", "build/Lane-1") {
		t.Fatal("on a case-sensitive filesystem the names differ")
	}
	if !sameBranchRef("refs/heads/build/lane-1", "build/lane-1") {
		t.Fatal("the same name is always the lane's")
	}
}
