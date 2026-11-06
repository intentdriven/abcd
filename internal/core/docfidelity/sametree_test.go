package docfidelity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const promoteVerdict = `{"verificationResult": "PROMOTE", "judgeModel": "claude-opus-5-5", "tier": "full", "failing": []}`

// emptyCommit moves HEAD to a new commit with the same tree.
func emptyCommit(t *testing.T, root, msg string) string {
	t.Helper()
	git(t, root, "commit", "-q", "--allow-empty", "-m", msg)
	return git(t, root, "rev-parse", "HEAD")
}

// renameReceipt relabels a saved review's directory and its subject as naming
// another commit, as a review of a commit since garbage collected reads.
func renameReceipt(t *testing.T, root, from, to string) {
	t.Helper()
	dir := filepath.Join(root, ReceiptsDir)
	data, err := os.ReadFile(filepath.Join(dir, from, GateName+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(dir, from)); err != nil {
		t.Fatal(err)
	}
	write(t, root, filepath.Join(ReceiptsDir, to, GateName+".json"), strings.ReplaceAll(string(data), from, to))
}

// TestASameTreeHeadStandsOnTheReviewOfItsTree is iss-2611060603332537: a review
// is of bytes, so a head whose tree is the reviewed commit's own tree object
// meets it, and the review says which commit it stands on.
func TestASameTreeHeadStandsOnTheReviewOfItsTree(t *testing.T) {
	root := armedRepo(t)
	reviewed := git(t, root, "rev-parse", "HEAD")
	if _, _, err := Record(root, []byte(promoteVerdict), at); err != nil {
		t.Fatal(err)
	}
	head := emptyCommit(t, root, "a commit with the reviewed tree")

	got := (SavedReview{Root: root, Commit: head}).Review()
	if got.Status != ReviewMatch || got.Commit != head || got.SameTreeAs != reviewed {
		t.Fatalf("a head with the reviewed tree: %+v; want a match on %s standing on %s", got, head, reviewed)
	}
	v, _, err := Gate(root, tree, []string{"itd-1"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if v.Refuse {
		t.Fatalf("the gate refused a head with the reviewed tree: %v", v.Reasons)
	}
	note := v.Review.TreeNote()
	if !strings.Contains(note, short(reviewed)) || !strings.Contains(note, short(head)) || !strings.Contains(note, "same tree") {
		t.Fatalf("the note states the match: %q", note)
	}
}

// A revert back to the reviewed tree is the reported case: the commit in between
// changed the tree, and the revert restored it.
func TestARevertToTheReviewedTreeStandsOnItsReview(t *testing.T) {
	root := armedRepo(t)
	reviewed := git(t, root, "rev-parse", "HEAD")
	if _, _, err := Record(root, []byte(promoteVerdict), at); err != nil {
		t.Fatal(err)
	}
	write(t, root, "roll.txt", "the roll\n")
	git(t, root, "add", "roll.txt")
	git(t, root, "commit", "-q", "-m", "the roll")
	git(t, root, "revert", "--no-edit", "HEAD")
	head := git(t, root, "rev-parse", "HEAD")

	got := (SavedReview{Root: root, Commit: head}).Review()
	if got.Status != ReviewMatch || got.SameTreeAs != reviewed {
		t.Fatalf("the revert: %+v; want a match standing on %s", got, reviewed)
	}
}

// A commit that changes the tree stays stale, naming the review it does not meet.
func TestADifferentTreeStaysStale(t *testing.T) {
	root := armedRepo(t)
	reviewed := git(t, root, "rev-parse", "HEAD")
	if _, _, err := Record(root, []byte(promoteVerdict), at); err != nil {
		t.Fatal(err)
	}
	write(t, root, "changed.txt", "changed\n")
	git(t, root, "add", "changed.txt")
	git(t, root, "commit", "-q", "-m", "a change")
	head := git(t, root, "rev-parse", "HEAD")

	got := (SavedReview{Root: root, Commit: head}).Review()
	if got.Status != ReviewStale || got.Named != reviewed || got.SameTreeAs != "" {
		t.Fatalf("a different tree: %+v; want stale naming %s", got, reviewed)
	}
}

// A review naming a commit this repository no longer holds says nothing of any
// tree, so it stays refused, as stale, and is never read as a match.
func TestAReviewOfAnUnresolvableCommitStaysRefused(t *testing.T) {
	root := armedRepo(t)
	if _, _, err := Record(root, []byte(promoteVerdict), at); err != nil {
		t.Fatal(err)
	}
	reviewed := git(t, root, "rev-parse", "HEAD")
	const gone = "0123456789012345678901234567890123456789"
	renameReceipt(t, root, reviewed, gone)
	head := emptyCommit(t, root, "a commit with the same tree")

	got := (SavedReview{Root: root, Commit: head}).Review()
	if got.Status != ReviewStale || got.Named != gone || got.SameTreeAs != "" {
		t.Fatalf("a review of a commit no longer held: %+v; want stale naming %s", got, gone)
	}
}

// The head's own review is the review of the head: a HOLD there is not outvoted
// by a PROMOTE saved for another commit with the same tree.
func TestTheHeadsOwnReviewWinsOverASameTreeOne(t *testing.T) {
	root := armedRepo(t)
	if _, _, err := Record(root, []byte(promoteVerdict), at); err != nil {
		t.Fatal(err)
	}
	head := emptyCommit(t, root, "a commit with the reviewed tree")
	hold := `{"verificationResult": "HOLD", "judgeModel": "claude-opus-5-5", "tier": "full", "failing": [` +
		`{"doc": "brief", "chapter": "06-capture.md", "sentence": "s", "evidence": "e", "disposition": "confirmed"}]}`
	if _, _, err := Record(root, []byte(hold), at); err != nil {
		t.Fatal(err)
	}
	got := (SavedReview{Root: root, Commit: head}).Review()
	if got.Status != ReviewHold || got.SameTreeAs != "" {
		t.Fatalf("the head's own HOLD: %+v; want it read, not the same-tree PROMOTE", got)
	}
}

// A same-tree review is read through every check its own commit's review meets,
// so one git tracks stays refused, whichever commit stands on it.
func TestASameTreeReviewGitTracksIsRefused(t *testing.T) {
	root := armedRepo(t)
	if _, _, err := Record(root, []byte(promoteVerdict), at); err != nil {
		t.Fatal(err)
	}
	reviewed := git(t, root, "rev-parse", "HEAD")
	head := git(t, root, "commit-tree", reviewed+"^{tree}", "-p", reviewed, "-m", "the same tree")
	git(t, root, "update-ref", "HEAD", head)
	git(t, root, "add", "-f", ReceiptsDir+"/"+reviewed+"/"+GateName+".json")
	got := (SavedReview{Root: root, Commit: head}).Review()
	if got.Status != ReviewInvalid || !strings.Contains(strings.Join(got.Problems, "\n"), "git tracks") {
		t.Fatalf("a same-tree review git tracks: %+v; want it refused as tracked", got)
	}
}
