package claim

import (
	"errors"
	"os/exec"
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
