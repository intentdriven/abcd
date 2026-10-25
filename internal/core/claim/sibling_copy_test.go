package claim

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// A second worktree of the same checkout is its own copy, with its own key in
// its own local tier, but it is on this machine: what holds a reservation from
// it is named as that worktree, never as another computer
// (iss-2610250013552001).

// sibling adds a worktree beside p's checkout, with its own local tier, and
// the path a refusal names it by.
func (p *participant) sibling(t *testing.T) (*participant, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "side")
	p.git("worktree", "add", "-q", "-b", "side", dir)
	if err := os.MkdirAll(filepath.Join(dir, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	return &participant{w: p.w, dir: dir, home: p.home}, fsutil.RedactHome(real)
}

func TestASiblingWorktreesReservationIsNamedAsThatWorktree(t *testing.T) {
	w := newWorld(t)
	main := w.clone("1234+octo@users.noreply.github.com", "")
	side, shown := main.sibling(t)
	if _, err := side.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	_, err := main.reserve("iss-5", "a.go")
	want := "you are already building iss-5 from your worktree " + shown + " on this machine, until 2026-10-03T16:00:00Z; " +
		"renew it from there, or pass --take-over to continue here"
	if !errors.Is(err, ErrBackOff) || err.Error() != want {
		t.Fatalf("the reserve from the main worktree = %v; want %q", err, want)
	}
	_, err = main.do(Request{Record: "iss-5", Release: true})
	want = "iss-5 is held by your worktree " + shown + " on this machine; release it there, or take it over here first; nothing was pushed"
	if !errors.Is(err, ErrRefused) || err.Error() != want {
		t.Fatalf("the release from the main worktree = %v; want %q", err, want)
	}
	res, err := main.do(Request{Record: "iss-5", Footprint: []string{"a.go"}, TakeOver: true})
	if err != nil || res.Replaced == nil || res.Replaced.YourCopy != "your worktree "+shown+" on this machine" {
		t.Fatalf("the take-over = %+v, %v; want it to name the worktree it replaced", res.Replaced, err)
	}
	// A copy on another computer keeps its words.
	other := w.clone("1234+octo@users.noreply.github.com", "")
	_, err = other.reserve("iss-5", "a.go")
	want = "you are already building iss-5 on another computer, until 2026-10-03T16:00:00Z; pass --take-over to continue here"
	if !errors.Is(err, ErrBackOff) || err.Error() != want {
		t.Fatalf("the reserve from another clone = %v; want %q", err, want)
	}
}

func TestASiblingWorktreesCutIsNamedAsThatWorktreeInTheMergeHold(t *testing.T) {
	w := newWorld(t)
	main := w.clone("1234+octo@users.noreply.github.com", "")
	side, shown := main.sibling(t)
	if _, err := side.do(Request{Record: "rel-v0.16.0", Lease: time.Hour}); err != nil {
		t.Fatal(err)
	}
	m, err := FetchMergeHold(main.dir, t0, main.opts())
	if err != nil {
		t.Fatal(err)
	}
	reason := "release v0.16.0 is being cut by your worktree " + shown + " on this machine, until 2026-10-03T15:00:00Z" +
		", and a session cutting a release has the final say on what merges before its tag"
	wait := "merge it from " + shown + ", the worktree cutting v0.16.0, or wait until v0.16.0 is tagged or that copy releases rel-v0.16.0"
	if !m.Held() || m.Reason() != reason || m.Waits() != wait {
		t.Fatalf("held %v, reason %q, waits %q; want reason %q, waits %q", m.Held(), m.Reason(), m.Waits(), reason, wait)
	}
	if m.Holds[0].Sibling != shown {
		t.Fatalf("the hold's sibling = %q; want %q", m.Holds[0].Sibling, shown)
	}
}
