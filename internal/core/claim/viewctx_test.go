package claim

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// The reservations' view a hook reads on a deadline (iss-2611100548341325):
// ViewContext reads what View reads, starts no git once its ctx has ended,
// and a git that hangs anywhere on its route is stopped when the ctx ends,
// answering ctx's error and no snapshot read in part.

// viewOnEveryRoute is a copy whose view starts git on every reader of its
// route: the fetch stamp's tracked check, the reservation and release-tag
// listings, the empty tree's name, origin's default branch and the records
// on it, this copy's kept key, the tip batch, a tip too big for the batch
// read on its own, a tip dated ahead whose first sighting is kept, and the
// generated files a live footprint is judged against.
func viewOnEveryRoute(t *testing.T) *participant {
	t.Helper()
	w := newWorld(t)
	w.declare("gen.md linguist-generated\n", "gen.md")
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := p.reserve("iss-1", "a.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := q.reserve("iss-2", "b.go"); err != nil {
		t.Fatal(err)
	}
	w.finishOnMain("iss-2", "resolved")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	w.tagOrigin("v0.16.0")
	ahead := t0.Add(30 * 24 * time.Hour)
	w.pushTipAt("iss-5", reservationText("iss-5", ahead, ahead.Add(time.Hour)), ahead)
	w.pushTipAt("iss-8", reservationText("iss-8", t0, t0.Add(time.Hour))+strings.Repeat("x", maxObjectBytes), t0)
	if _, err := Fetch(p.dir, p.opts()); err != nil {
		t.Fatal(err)
	}
	return p
}

// gitOnPath puts a git first on PATH that logs each call and, on call n
// (counting from 1; 0 for none), signals and then sleeps for stallFor in
// place of git; every other call runs the real git. The returned calls
// empties the log and answers it; stalled, once the read is done, whether
// call n came. The stall cancels cancel, so the ctx ends exactly while git
// hangs.
func gitOnPath(t *testing.T, n int, cancel context.CancelFunc) (calls func() int, stalled func() bool) {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not on PATH")
	}
	bin, dir := t.TempDir(), t.TempDir()
	log, fifo := filepath.Join(dir, "calls"), filepath.Join(dir, "stall")
	script := "#!/bin/sh\necho x >> '" + log + "'\n"
	if n > 0 {
		if err := syscall.Mkfifo(fifo, 0o600); err != nil {
			t.Skipf("no fifo: %v", err)
		}
		script += "if [ $(wc -l < '" + log + "') -eq " + strconv.Itoa(n) + " ]; then printf x > '" + fifo + "'; exec sleep " +
			strconv.Itoa(int(stallFor/time.Second)) + "; fi\n"
	}
	script += "exec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	calls = func() int {
		raw, _ := os.ReadFile(log)
		_ = os.WriteFile(log, nil, 0o600)
		return strings.Count(string(raw), "\n")
	}
	if n == 0 {
		return calls, func() bool { return false }
	}
	came, done := false, make(chan struct{})
	go func() {
		defer close(done)
		b, _ := os.ReadFile(fifo)
		came = len(b) > 0
		cancel()
	}()
	return calls, func() bool {
		if w, err := os.OpenFile(fifo, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
			_ = w.Close()
		}
		<-done
		return came
	}
}

// stallFor is how long a stalled git sleeps: a read that returns before it
// has passed was not held by that git.
const stallFor = 60 * time.Second

// TestViewContextReadsWhatViewReads: one body, so a live ctx reads the same
// snapshot the context-free view does.
func TestViewContextReadsWhatViewReads(t *testing.T) {
	p := viewOnEveryRoute(t)
	want, err := View(p.dir, p.w.clock())
	if err != nil {
		t.Fatal(err)
	}
	if len(want.Reservations) < 4 || want.Generated.files == nil {
		t.Fatalf("the fixture's view holds %d reservations and generated %v; the test proves nothing", len(want.Reservations), want.Generated.files)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	got, err := ViewContext(ctx, p.dir, p.w.clock())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ViewContext = %+v\nwant View's %+v", got, want)
	}
}

// TestViewContextStartsNoGitOnceEnded: a ctx already ended starts no git and
// answers its error, with no reservation in the snapshot.
func TestViewContextStartsNoGitOnceEnded(t *testing.T) {
	p := viewOnEveryRoute(t)
	calls, _ := gitOnPath(t, 0, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	snap, err := ViewContext(ctx, p.dir, p.w.clock())
	if !errors.Is(err, context.Canceled) || len(snap.Reservations) != 0 {
		t.Errorf("a spent ctx = %d reservations, %v; want none and ctx's error", len(snap.Reservations), err)
	}
	if n := calls(); n != 0 {
		t.Errorf("a spent ctx still started git %d time(s)", n)
	}
}

// TestViewContextStopsAHungGitOnEveryRoute: each git the view starts, hung in
// turn, is stopped when the ctx ends, and the view answers ctx's error and no
// snapshot read in part, whichever reader the hang was in.
func TestViewContextStopsAHungGitOnEveryRoute(t *testing.T) {
	p := viewOnEveryRoute(t)
	calls, _ := gitOnPath(t, 0, nil)
	calls()
	live, cancel := context.WithTimeout(context.Background(), time.Hour)
	if _, err := ViewContext(live, p.dir, p.w.clock()); err != nil {
		t.Fatal(err)
	}
	cancel()
	total := calls()
	if total < 12 {
		t.Fatalf("the view started git %d times; the fixture does not reach every reader", total)
	}
	for n := 1; n <= total; n++ {
		t.Run("call "+strconv.Itoa(n), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			_, stalled := gitOnPath(t, n, cancel)
			start := time.Now()
			snap, err := ViewContext(ctx, p.dir, p.w.clock())
			took := time.Since(start)
			if !stalled() {
				t.Fatalf("call %d never came; the test proves nothing", n)
			}
			if took >= stallFor {
				t.Errorf("the view waited out the hung git (%s)", took)
			}
			if !errors.Is(err, context.Canceled) || len(snap.Reservations) != 0 || snap.Generated.files != nil {
				t.Errorf("a view whose call %d hung until the ctx ended = %d reservations, %v; want none and ctx's error", n, len(snap.Reservations), err)
			}
		})
	}
}
