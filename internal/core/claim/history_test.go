package claim

import (
	"os"
	"path"
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
	if err != nil || !slices.Equal(h.Added, []string{merge, own}) || !h.Merged {
		t.Fatalf("a walk over a merge = %+v, %v; want the merge and the branch's own commit, and Merged", h, err)
	}
}

// TestAMergeOverUnreadCommitsSaysMerged: commits pushed on the mark and
// brought in by a merge whose first parent is the mark are off the tip's
// first-parent line, so the walk lists only the merge, and says so.
func TestAMergeOverUnreadCommitsSaysMerged(t *testing.T) {
	r := newCheckout(t, "1234+octo@users.noreply.github.com")
	b := newBranch(t, r, "octo")
	m := b.grow(2)[1]
	if err := testNamespace.markRead(r.Root(), b.ref, m); err != nil {
		t.Fatal(err)
	}
	v1 := b.commit(m)
	v2 := b.commit(v1)
	x := b.commit(m, v2)
	r.Git("update-ref", b.ref, x)
	h, err := testNamespace.since(r.Root(), b.ref, 10)
	if err != nil || !slices.Equal(h.Added, []string{x}) || h.More || h.Rewritten || !h.Merged {
		t.Fatalf("a merge over unread commits = %+v, %v; want only the merge, and Merged", h, err)
	}
	// A line with no merge on it says nothing of the kind.
	if err := testNamespace.markRead(r.Root(), b.ref, x); err != nil {
		t.Fatal(err)
	}
	b.grow(2)
	if h, err := testNamespace.since(r.Root(), b.ref, 10); err != nil || len(h.Added) != 2 || h.Merged {
		t.Fatalf("a walk over plain commits = %+v, %v; want two and no Merged", h, err)
	}
}

// TestAMergeWhoseParentsPassTheListingsBoundStillReads: a commit with more
// parents than the walk's listing has room for does not fail the walk; the
// line is read without its parents, and the window says Merged.
func TestAMergeWhoseParentsPassTheListingsBoundStillReads(t *testing.T) {
	r := newCheckout(t, "1234+octo@users.noreply.github.com")
	b := newBranch(t, r, "octo")
	base := b.grow(1)[0]
	parents := []string{base}
	for range 8 {
		parents = append(parents, b.commit())
	}
	octopus := b.commit(parents...)
	r.Git("update-ref", b.ref, octopus)
	h, err := testNamespace.since(r.Root(), b.ref, 1)
	if err != nil || !slices.Equal(h.Added, []string{octopus}) || !h.More || !h.Merged {
		t.Fatalf("an octopus past the listing's bound = %+v, %v; want the octopus, More and Merged", h, err)
	}
}

// TestAFileNamedLikeACommitIsNotReadAsAPath: a file in the work tree named
// exactly as the tip or the mark is never taken for a path by either walk.
func TestAFileNamedLikeACommitIsNotReadAsAPath(t *testing.T) {
	r := newCheckout(t, "1234+octo@users.noreply.github.com")
	b := newBranch(t, r, "octo")
	read := b.grow(2)[1]
	r.Write(read, "planted\n")
	h, err := testNamespace.since(r.Root(), b.ref, 10)
	if err != nil || h.Tip != read || len(h.Added) != 2 {
		t.Fatalf("a tip named by a file = %+v, %v; want the branch walked", h, err)
	}
	if err := testNamespace.markRead(r.Root(), b.ref, read); err != nil {
		t.Fatal(err)
	}
	r.Git("update-ref", "-d", b.ref)
	fresh := b.grow(2)
	h, err = testNamespace.since(r.Root(), b.ref, 10)
	if err != nil || !h.Rewritten || !slices.Equal(h.Added, newest(fresh)) {
		t.Fatalf("a mark named by a file = %+v, %v; want Rewritten and the new line", h, err)
	}
}

// TestABranchReplacedByAnUnrelatedLineWithinTheLimitIsRewritten: a tip whose
// line reaches its root inside the window without meeting the mark's line is
// a rewrite, and every commit of the new line is added.
func TestABranchReplacedByAnUnrelatedLineWithinTheLimitIsRewritten(t *testing.T) {
	r := newCheckout(t, "1234+octo@users.noreply.github.com")
	b := newBranch(t, r, "octo")
	read := b.grow(3)[2]
	if err := testNamespace.markRead(r.Root(), b.ref, read); err != nil {
		t.Fatal(err)
	}
	r.Git("update-ref", "-d", b.ref)
	fresh := b.grow(2)
	h, err := testNamespace.since(r.Root(), b.ref, 10)
	if err != nil || !h.Rewritten || h.More || h.Since != read || !slices.Equal(h.Added, newest(fresh)) {
		t.Fatalf("an unrelated line = %+v, %v; want Rewritten and both its commits", h, err)
	}
}

// TestHasCommitReadsOnlyTheObjectsType: whether the mark is held is read from
// the object's type, never by peeling it, which reads the whole object; a
// commit is held, and a missing object or one of another type is not.
func TestHasCommitReadsOnlyTheObjectsType(t *testing.T) {
	r := newCheckout(t, "1234+octo@users.noreply.github.com")
	b := newBranch(t, r, "octo")
	c := b.grow(1)[0]
	blob := r.Git("hash-object", "-w", "--stdin")
	log := traced(t)
	for sha, want := range map[string]bool{c: true, blob: false, b.tree: false, "1111111111111111111111111111111111111111": false} {
		if got, err := hasCommit(r.Root(), sha); err != nil || got != want {
			t.Errorf("hasCommit(%s) = %v, %v; want %v", sha, got, err, want)
		}
	}
	for _, cmd := range log.cmds {
		if cmd[0] != "cat-file" || (cmd[1] != "-t" && cmd[1] != "-e") {
			t.Errorf("hasCommit ran git %q; want only a type or existence read", cmd)
		}
	}
	if log.anyArgContains("^{") {
		t.Errorf("hasCommit peeled an object: %q", log.cmds)
	}
	if got, err := hasCommit(r.Root(), "not-a-commit"); err != nil || got {
		t.Errorf("hasCommit of a non-name = %v, %v; want false", got, err)
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

// TestAMarkedWalkReadsNoFurtherThanItsWindow: a branch that gained more than
// the limit since its mark is answered from walks that each stop at limit+1
// commits from where they start. A walk that excludes the mark (`^mark`), or
// asks whether the mark is in the tip's history, reads every commit between
// the two whatever --max-count says, so a writer who pushed a million commits
// would make the walk read a million.
func TestAMarkedWalkReadsNoFurtherThanItsWindow(t *testing.T) {
	r := newCheckout(t, "1234+octo@users.noreply.github.com")
	b := newBranch(t, r, "octo")
	read := b.grow(2)[1]
	if err := testNamespace.markRead(r.Root(), b.ref, read); err != nil {
		t.Fatal(err)
	}
	b.grow(6)
	log := traced(t)
	if _, err := testNamespace.since(r.Root(), b.ref, 3); err != nil {
		t.Fatal(err)
	}
	walks := 0
	for _, c := range log.cmds {
		switch c[0] {
		case "for-each-ref", "cat-file":
		case "rev-list":
			walks++
			if !slices.Contains(c, "--first-parent") || !slices.Contains(c, "--max-count=4") {
				t.Errorf("a walk without its bound: %q", c)
			}
			for _, a := range c[1:] {
				if strings.HasPrefix(a, "^") || strings.HasPrefix(a, "--not") || strings.Contains(a, "..") {
					t.Errorf("a walk that excludes a commit, which reads all between: %q", c)
				}
			}
		default:
			t.Errorf("the walk ran git %q", c)
		}
	}
	if walks == 0 || walks > 2 {
		t.Fatalf("the walk ran %d rev-lists; want one or two: %q", walks, log.cmds)
	}
}

// TestAMarkedBranchPastItsLimitSaysMore: what came after the mark, past the
// limit, is the newest limit commits and More; exactly the limit is all of it.
func TestAMarkedBranchPastItsLimitSaysMore(t *testing.T) {
	r := newCheckout(t, "1234+octo@users.noreply.github.com")
	b := newBranch(t, r, "octo")
	read := b.grow(2)[1]
	if err := testNamespace.markRead(r.Root(), b.ref, read); err != nil {
		t.Fatal(err)
	}
	added := b.grow(5)
	h, err := testNamespace.since(r.Root(), b.ref, 3)
	if err != nil || h.Since != read || !h.More || h.Rewritten || !slices.Equal(h.Added, newest(added)[:3]) {
		t.Fatalf("a marked walk past its limit = %+v, %v; want the newest three and More", h, err)
	}
	if err := testNamespace.markRead(r.Root(), b.ref, h.Tip); err != nil {
		t.Fatal(err)
	}
	added = b.grow(3)
	h, err = testNamespace.since(r.Root(), b.ref, 3)
	if err != nil || h.More || h.Rewritten || !slices.Equal(h.Added, newest(added)) {
		t.Fatalf("a marked walk at its limit = %+v, %v; want all three and no More", h, err)
	}
}

// TestABranchMovedBackIsRewrittenWithNothingAdded: a tip that is an ancestor
// of the mark holds nothing the mark does not.
func TestABranchMovedBackIsRewrittenWithNothingAdded(t *testing.T) {
	r := newCheckout(t, "1234+octo@users.noreply.github.com")
	b := newBranch(t, r, "octo")
	base := b.grow(2)
	read := b.grow(1)[0]
	if err := testNamespace.markRead(r.Root(), b.ref, read); err != nil {
		t.Fatal(err)
	}
	r.Git("update-ref", b.ref, base[0])
	h, err := testNamespace.since(r.Root(), b.ref, 10)
	if err != nil || !h.Rewritten || h.More || h.Since != read || h.Tip != base[0] || len(h.Added) != 0 {
		t.Fatalf("a branch moved back = %+v, %v; want Rewritten and nothing added", h, err)
	}
}

// TestADivergedPushReadsAsRewritten: a branch force-pushed to origin onto
// another line, then fetched, reads as rewritten, with only the new line's
// own commit added.
func TestADivergedPushReadsAsRewritten(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	trunk := w.seed.Git("rev-parse", "main")
	tree := w.seed.Git("hash-object", "-w", "-t", "tree", "/dev/null")
	first := w.seed.Git("commit-tree", tree, "-p", trunk, "-m", "first")
	w.seed.Git("push", "-q", "origin", first+":refs/heads/abcd-test/octo")
	if _, err := fetch(p.dir, t0, "nothing was read", testNamespace); err != nil {
		t.Fatal(err)
	}
	ref := testNamespace.track("octo")
	h, err := testNamespace.since(p.dir, ref, 10)
	if err != nil || h.Tip != first || !slices.Equal(h.Added, []string{first, trunk}) {
		t.Fatalf("a first walk = %+v, %v", h, err)
	}
	if err := testNamespace.markRead(p.dir, ref, h.Tip); err != nil {
		t.Fatal(err)
	}
	other := w.seed.Git("commit-tree", tree, "-p", trunk, "-m", "other")
	w.seed.Git("push", "-q", "-f", "origin", other+":refs/heads/abcd-test/octo")
	if _, err := fetch(p.dir, t0, "nothing was read", testNamespace); err != nil {
		t.Fatal(err)
	}
	h, err = testNamespace.since(p.dir, ref, 10)
	if err != nil || h.Tip != other || h.Since != first || !h.Rewritten || h.More || !slices.Equal(h.Added, []string{other}) {
		t.Fatalf("a diverged push = %+v, %v; want Rewritten and the one new commit", h, err)
	}
}

// TestMarkingTheSameCommitTwiceLeavesNoTempFile: a second mark at the same
// commit replaces the first and leaves the marks' directory holding the mark
// alone.
func TestMarkingTheSameCommitTwiceLeavesNoTempFile(t *testing.T) {
	r := newCheckout(t, "1234+octo@users.noreply.github.com")
	b := newBranch(t, r, "octo")
	tip := b.grow(1)[0]
	for range 2 {
		if err := testNamespace.markRead(r.Root(), b.ref, tip); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(filepath.Join(r.Root(), filepath.FromSlash(testNamespace.readDir())))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if want := path.Base(testNamespace.markRel(b.ref)); !slices.Equal(names, []string{want}) {
		t.Fatalf("the marks' directory holds %q; want only %q", names, want)
	}
	if got := testNamespace.lastRead(r.Root(), b.ref); got != tip {
		t.Fatalf("the mark reads %q; want %q", got, tip)
	}
}

// TestWhatTheBoundLeavesUnsaid pins the three readings the bound costs. A
// rewrite whose new line runs past the limit before it meets the old reads as
// More, not Rewritten; a mark the tip reaches only through a merge's second
// parent is off the tip's first-parent line, so reads as rewritten (and
// Merged); and a tip moved back more than limit commits below the mark, with
// more than limit commits beneath it, reads as More, not Rewritten, with
// Added holding commits already read.
func TestWhatTheBoundLeavesUnsaid(t *testing.T) {
	r := newCheckout(t, "1234+octo@users.noreply.github.com")
	b := newBranch(t, r, "octo")
	base := b.grow(1)[0]
	read := b.grow(1)[0]
	if err := testNamespace.markRead(r.Root(), b.ref, read); err != nil {
		t.Fatal(err)
	}
	r.Git("update-ref", b.ref, base)
	fresh := b.grow(5)
	h, err := testNamespace.since(r.Root(), b.ref, 3)
	if err != nil || h.Rewritten || !h.More || !slices.Equal(h.Added, newest(fresh)[:3]) {
		t.Fatalf("a rewrite longer than the window = %+v, %v; want More, not Rewritten", h, err)
	}
	merge := b.commit(base, read)
	r.Git("update-ref", b.ref, merge)
	h, err = testNamespace.since(r.Root(), b.ref, 10)
	if err != nil || !h.Rewritten || !h.Merged || !slices.Equal(h.Added, []string{merge}) {
		t.Fatalf("a mark reached through a second parent = %+v, %v; want Rewritten, Merged and the merge", h, err)
	}

	far := newBranch(t, r, "far")
	line := far.grow(10)
	if err := testNamespace.markRead(r.Root(), far.ref, line[9]); err != nil {
		t.Fatal(err)
	}
	r.Git("update-ref", far.ref, line[4])
	h, err = testNamespace.since(r.Root(), far.ref, 3)
	if err != nil || h.Rewritten || !h.More || h.Merged || !slices.Equal(h.Added, newest(line[2:5])) {
		t.Fatalf("a tip moved back past the window = %+v, %v; want More, not Rewritten, and three commits already read", h, err)
	}
}
