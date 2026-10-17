package claim

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// An unreadable tip is held from its commit date for the default lease (C13,
// C14), including one too big to read whole and one whose date cannot be read
// at all (review finding R3 on spc-2610031753359849 steps 1-3).

// TestATipOverTheObjectBoundIsHeldFromItsCommitDate: a commit past the read's
// bound still gives its committer date, so its hold is not the year 1.
func TestATipOverTheObjectBoundIsHeldFromItsCommitDate(t *testing.T) {
	r := newCheckout(t, "pat@example.com")
	tree := r.Git("hash-object", "-w", "-t", "tree", "/dev/null")
	ref := hostileTip(t, r, "iss-5", "abcd reservation: iss-5\n\n"+strings.Repeat("x", 9000)+"\n", tree, t0)
	got, ok, err := readTip(r.Root(), "iss-5", ref, tree, t0.Add(time.Hour), true)
	if err != nil || !ok || got.Unreadable != reasonObjectTooBig {
		t.Fatalf("tip = %+v, %v, %v; want unreadable: %s", got, ok, err, reasonObjectTooBig)
	}
	if !got.WrittenAt.Equal(t0) || got.DatedFromFetch {
		t.Fatalf("the oversized tip is dated %s (from fetch: %v); want its commit date %s", got.WrittenAt, got.DatedFromFetch, t0)
	}
}

// TestAnUndatedTipIsHeldFromWhenThisCopyFirstFetchedIt: a tip whose date
// cannot be read (here, a ref naming an object that is not a commit) is held
// from when this copy first fetched it, kept in the local tier so a later
// fetch does not restart the hold, and the refusal says so.
func TestAnUndatedTipIsHeldFromWhenThisCopyFirstFetchedIt(t *testing.T) {
	w := newWorld(t)
	q := w.clone("99+quinn@users.noreply.github.com", "")
	blob := strings.TrimSpace(runGitStdin(t, q.dir, "not a commit\n", "hash-object", "-w", "--stdin"))
	q.git("update-ref", trackingName("iss-5"), blob)
	tree := q.git("hash-object", "-w", "-t", "tree", "/dev/null")

	got, ok, err := readTip(q.dir, "iss-5", trackingName("iss-5"), tree, t0, true)
	if err != nil || !ok || got.Unreadable != reasonNoObject || !got.WrittenAt.Equal(t0) || !got.DatedFromFetch {
		t.Fatalf("tip = %+v, %v, %v; want unreadable and dated from the first fetch, %s", got, ok, err, t0)
	}
	got, _, _ = readTip(q.dir, "iss-5", trackingName("iss-5"), tree, t0.Add(time.Hour), true)
	if !got.WrittenAt.Equal(t0) {
		t.Fatalf("a later fetch restarted the hold at %s; want the first fetch, %s", got.WrittenAt, t0)
	}

	w.set(t0.Add(time.Hour))
	p, err := prepare(q.dir, Request{Record: "iss-5", Footprint: []string{"b.go"}}, q.opts())
	if err != nil {
		t.Fatal(err)
	}
	j, err := judge(p, tree)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, err = decide(p, j)
	want := "the reservation on iss-5 is unreadable (" + reasonNoObject + "); its commit date cannot be read, so it is held from when " +
		"this copy first fetched it, until 2026-10-03T16:00:00Z, and can be taken over after 2026-10-03T16:05:00Z"
	if !errors.Is(err, ErrBackOff) || err.Error() != want {
		t.Fatalf("decide = %v; want %q", err, want)
	}
}

// runGitStdin runs git in dir with stdin under the hermetic test environment.
func runGitStdin(t *testing.T, dir, stdin string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = gittest.Env(t)
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(out)
}

// TestAnOversizedTipIsDatedAsStoredNotAsReplaced: a replace ref (one a fetch
// can bring in, under remote.origin.fetch = +refs/replace/*:refs/replace/*)
// standing a small commit dated 2100 in for an oversized tip does not lend the
// tip its date: the tip is judged too big from the object as stored, and is
// held from that object's own commit date.
func TestAnOversizedTipIsDatedAsStoredNotAsReplaced(t *testing.T) {
	r := newCheckout(t, "pat@example.com")
	tree := r.Git("hash-object", "-w", "-t", "tree", "/dev/null")
	ref := hostileTip(t, r, "iss-5", "abcd reservation: iss-5\n\n"+strings.Repeat("x", 9000)+"\n", tree, t0)
	sha := r.Git("rev-parse", ref)
	later := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "msg")
	if err := os.WriteFile(path, []byte("abcd reservation: iss-5\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r.Git("replace", sha, commitAt(t, r, tree, path, later))
	// The replacement is live for a git that honours it: the tip reads as
	// dated 2100.
	if got := r.Git("show", "-s", "--format=%ct", sha); got != strconv.FormatInt(later.Unix(), 10) {
		t.Fatalf("the replace ref is not in effect: show --format=%%ct %s = %q", sha, got)
	}
	got, ok, err := readTip(r.Root(), "iss-5", ref, tree, t0.Add(time.Hour), true)
	if err != nil || !ok || got.Unreadable != reasonObjectTooBig {
		t.Fatalf("tip = %+v, %v, %v; want unreadable: %s", got, ok, err, reasonObjectTooBig)
	}
	if !got.WrittenAt.Equal(t0) || got.DatedFromFetch {
		t.Fatalf("an oversized tip behind a replace ref is dated %s (from fetch: %v); want its own commit date %s, not %s",
			got.WrittenAt, got.DatedFromFetch, t0, later)
	}
}

// TestATipThatIsNotACommitStaysUnreadableBehindAReplaceRef: a tracking ref
// naming a blob that a replace ref stands a commit in for reads as the blob it
// is, unreadable and dated from the first fetch, and the view lists it as
// unreadable rather than refusing the listing. The walk reads its type as
// stored too.
func TestATipThatIsNotACommitStaysUnreadableBehindAReplaceRef(t *testing.T) {
	r := newCheckout(t, "pat@example.com")
	tree := r.Git("hash-object", "-w", "-t", "tree", "/dev/null")
	blob := strings.TrimSpace(runGitStdin(t, r.Root(), "not a commit\n", "hash-object", "-w", "--stdin"))
	r.Git("update-ref", trackingName("iss-5"), blob)
	path := filepath.Join(t.TempDir(), "msg")
	if err := os.WriteFile(path, []byte("abcd reservation: iss-5\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r.Git("replace", "-f", blob, commitAt(t, r, tree, path, t0))
	// The replacement is live for a git that honours it: the blob reads as a
	// commit.
	if got := r.Git("cat-file", "-t", blob); got != "commit" {
		t.Fatalf("the replace ref is not in effect: cat-file -t %s = %q", blob, got)
	}
	got, ok, err := readTip(r.Root(), "iss-5", trackingName("iss-5"), tree, t0, false)
	if err != nil || !ok || got.Commit != blob || got.Unreadable != reasonNoObject || !got.DatedFromFetch {
		t.Fatalf("a blob tip behind a replace ref = %+v, %v, %v; want it unreadable (%s), dated from the fetch", got, ok, err, reasonNoObject)
	}
	snap, err := View(r.Root(), t0)
	if err != nil || len(snap.Reservations) != 1 || snap.Reservations[0].State != StateUnreadable {
		t.Fatalf("the view of a blob tip behind a replace ref = %+v, %v; want it listed as unreadable", snap, err)
	}
	r.Git("update-ref", testNamespace.track("blob"), blob)
	if h, err := testNamespace.since(r.Root(), testNamespace.track("blob"), 10); err != nil || h.Tip != blob || h.Unreadable != reasonNoObject {
		t.Fatalf("a walk of a blob tip behind a replace ref = %+v, %v; want it unreadable", h, err)
	}
}
