package claim

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// branch is a chain of empty-tree commits under testNamespace in one
// checkout, standing for a branch a fetch brought in.
type branch struct {
	t    *testing.T
	r    *gittest.Repo
	ref  string
	tree string
	n    int
}

func newBranch(t *testing.T, r *gittest.Repo, name string) *branch {
	t.Helper()
	return &branch{t: t, r: r, ref: testNamespace.track(name), tree: r.Git("hash-object", "-w", "-t", "tree", "/dev/null")}
}

// commit writes one commit on parent ("" for a root), without moving the ref.
func (b *branch) commit(parents ...string) string {
	b.t.Helper()
	b.n++
	args := []string{"commit-tree", b.tree, "-m", "c" + strconv.Itoa(b.n)}
	for _, p := range parents {
		args = append(args, "-p", p)
	}
	return b.r.Git(args...)
}

// grow adds n commits on the ref's tip and returns them, oldest first.
func (b *branch) grow(n int) []string {
	b.t.Helper()
	var out []string
	tip := strings.TrimSpace(b.r.Git("for-each-ref", "--format=%(objectname)", b.ref))
	for range n {
		if tip == "" {
			tip = b.commit()
		} else {
			tip = b.commit(tip)
		}
		out = append(out, tip)
	}
	b.r.Git("update-ref", b.ref, tip)
	return out
}

func newest(oldestFirst []string) []string {
	out := slices.Clone(oldestFirst)
	slices.Reverse(out)
	return out
}

// TestTheWalkReadsWhatABranchGainedSinceItWasLastRead: a branch never read is
// walked from its tip; once marked, the next walk reads only what came after,
// newest first, and a branch with nothing new runs no walk at all.
func TestTheWalkReadsWhatABranchGainedSinceItWasLastRead(t *testing.T) {
	r := newCheckout(t, "1234+octo@users.noreply.github.com")
	b := newBranch(t, r, "octo")
	first := b.grow(3)
	h, err := testNamespace.since(r.Root(), b.ref, 10)
	if err != nil || h.Tip != first[2] || h.Since != "" || h.More || h.Rewritten || !slices.Equal(h.Added, newest(first)) {
		t.Fatalf("a first walk = %+v, %v; want all three, newest first", h, err)
	}
	if err := testNamespace.markRead(r.Root(), b.ref, h.Tip); err != nil {
		t.Fatal(err)
	}
	more := b.grow(2)
	h, err = testNamespace.since(r.Root(), b.ref, 10)
	if err != nil || h.Since != first[2] || h.Rewritten || !slices.Equal(h.Added, newest(more)) {
		t.Fatalf("a second walk = %+v, %v; want only the two added", h, err)
	}
	if err := testNamespace.markRead(r.Root(), b.ref, h.Tip); err != nil {
		t.Fatal(err)
	}
	log := traced(t)
	h, err = testNamespace.since(r.Root(), b.ref, 10)
	if err != nil || len(h.Added) != 0 || h.Since != h.Tip {
		t.Fatalf("a walk with nothing new = %+v, %v", h, err)
	}
	if slices.Contains(log.subcommands(), "rev-list") {
		t.Fatalf("a branch with nothing new was walked: %v", log.cmds)
	}
}

// TestTheWalkIsBounded: a branch that gained more than the limit returns the
// newest of them and says there were more; a limit outside 1 to maxWalk is
// refused.
func TestTheWalkIsBounded(t *testing.T) {
	r := newCheckout(t, "1234+octo@users.noreply.github.com")
	b := newBranch(t, r, "octo")
	added := b.grow(5)
	h, err := testNamespace.since(r.Root(), b.ref, 3)
	if err != nil || !h.More || !slices.Equal(h.Added, newest(added)[:3]) {
		t.Fatalf("a walk past its limit = %+v, %v; want the newest three and More", h, err)
	}
	if h, err := testNamespace.since(r.Root(), b.ref, 5); err != nil || h.More || len(h.Added) != 5 {
		t.Fatalf("a walk at its limit = %+v, %v; want all five and no More", h, err)
	}
	for _, limit := range []int{0, -1, maxWalk + 1} {
		if _, err := testNamespace.since(r.Root(), b.ref, limit); err == nil {
			t.Errorf("a walk of %d commits was not refused", limit)
		}
	}
}

// TestARewrittenBranchIsSaid: a tip that no longer holds the commit last read
// is a rewrite, and the walk reads what the new line holds that the old did
// not; a mark naming a commit this checkout no longer holds reads the same,
// walked from the tip.
func TestARewrittenBranchIsSaid(t *testing.T) {
	r := newCheckout(t, "1234+octo@users.noreply.github.com")
	b := newBranch(t, r, "octo")
	base := b.grow(2)
	read := b.grow(1)[0]
	if err := testNamespace.markRead(r.Root(), b.ref, read); err != nil {
		t.Fatal(err)
	}
	r.Git("update-ref", b.ref, base[1])
	fresh := b.grow(2)
	h, err := testNamespace.since(r.Root(), b.ref, 10)
	if err != nil || !h.Rewritten || h.Since != read || !slices.Equal(h.Added, newest(fresh)) {
		t.Fatalf("a rewritten branch = %+v, %v; want Rewritten and the two new commits", h, err)
	}
	const gone = "1111111111111111111111111111111111111111"
	if err := testNamespace.markRead(r.Root(), b.ref, gone); err != nil {
		t.Fatal(err)
	}
	h, err = testNamespace.since(r.Root(), b.ref, 10)
	if err != nil || !h.Rewritten || h.Since != gone || !slices.Equal(h.Added, newest(append(base, fresh...))) {
		t.Fatalf("a mark this checkout no longer holds = %+v, %v; want Rewritten and the walk from the tip", h, err)
	}
}

// TestTheWalkFollowsTheBranchsOwnLine: a merge onto the branch does not pull
// the merged line's history into the walk.
func TestTheWalkFollowsTheBranchsOwnLine(t *testing.T) {
	r := newCheckout(t, "1234+octo@users.noreply.github.com")
	b := newBranch(t, r, "octo")
	own := b.grow(1)[0]
	side := b.commit(b.commit())
	merge := b.commit(own, side)
	r.Git("update-ref", b.ref, merge)
	h, err := testNamespace.since(r.Root(), b.ref, 10)
	if err != nil || !slices.Equal(h.Added, []string{merge, own}) {
		t.Fatalf("a walk over a merge = %+v, %v; want the merge and the branch's own commit", h, err)
	}
}

// TestTheWalkReadsAbsentAndOddBranchesWithoutFailing: an absent ref reads as
// no tip, a tip that is not a commit as unreadable, a ref outside the
// namespace is refused, and a mark that does not read is never read.
func TestTheWalkReadsAbsentAndOddBranchesWithoutFailing(t *testing.T) {
	r := newCheckout(t, "1234+octo@users.noreply.github.com")
	if h, err := testNamespace.since(r.Root(), testNamespace.track("nobody"), 10); err != nil || h.Tip != "" || len(h.Added) != 0 {
		t.Fatalf("an absent branch = %+v, %v; want no tip", h, err)
	}
	blob := r.Git("hash-object", "-w", "--stdin")
	r.Git("update-ref", testNamespace.track("blob"), blob)
	if h, err := testNamespace.since(r.Root(), testNamespace.track("blob"), 10); err != nil || h.Tip != blob || h.Unreadable != reasonNoObject {
		t.Fatalf("a blob tip = %+v, %v; want it unreadable", h, err)
	}
	for _, ref := range []string{TrackingPrefix + "iss-5", testNamespace.tracking, "refs/heads/main"} {
		if _, err := testNamespace.since(r.Root(), ref, 10); err == nil {
			t.Errorf("a walk of %s was not refused", ref)
		}
		if err := testNamespace.markRead(r.Root(), ref, blob); err == nil {
			t.Errorf("a mark on %s was not refused", ref)
		}
	}
	b := newBranch(t, r, "octo")
	added := b.grow(2)
	if err := testNamespace.markRead(r.Root(), b.ref, "not-a-commit"); err == nil {
		t.Fatalf("a mark naming no commit was written")
	}
	mark := filepath.Join(r.Root(), filepath.FromSlash(testNamespace.markRel(b.ref)))
	if err := os.MkdirAll(filepath.Dir(mark), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mark, []byte("garbage\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if h, err := testNamespace.since(r.Root(), b.ref, 10); err != nil || h.Since != "" || h.Rewritten || !slices.Equal(h.Added, newest(added)) {
		t.Fatalf("a mark that does not read = %+v, %v; want the branch read as never read", h, err)
	}
}

// TestACheckoutWithoutItsTierKeepsNoMark: the tier is never made on the way
// to a mark, so such a checkout reads every branch as never read.
func TestACheckoutWithoutItsTierKeepsNoMark(t *testing.T) {
	r := gittest.NewRepo(t)
	b := newBranch(t, r, "octo")
	added := b.grow(1)
	if err := testNamespace.markRead(r.Root(), b.ref, added[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(r.Root(), ".abcd")); !os.IsNotExist(err) {
		t.Fatalf("a mark made the tier: %v", err)
	}
	if h, err := testNamespace.since(r.Root(), b.ref, 10); err != nil || h.Since != "" || len(h.Added) != 1 {
		t.Fatalf("a walk without the tier = %+v, %v", h, err)
	}
}
