package claim

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The reservations' view a hook reads on a deadline (iss-2611100548341325):
// ViewContext reads what View reads, starts no git once its ctx has ended,
// and a git that hangs at any of the sites the fixtures reach is stopped when
// the ctx ends, answering ctx's error and no snapshot read in part.

// viewOnEveryRoute is a copy whose view starts git at the sites viewSites
// names: the tracked checks (the fetch stamp, this copy's kept key, and the
// kept first sightings of a tip dated ahead and of a tip that is not a
// commit, the last read both as its date and as its first sighting), the
// reservation and release-tag listings, the empty tree's name,
// origin's default-branch listing, the records on that branch, the tip
// batch's listing, headers and bodies, the own reads of a tip too big for the
// batch (its ref, its size and its date) and of the tip that is not a commit
// (its ref), and the generated files a live footprint is judged against
// (the tree listing, the root attributes' entry and blob). crowded fills
// origin's default-branch listing past its cap, so the default branch is
// named by symbolic-ref and a question per ref as well. It does not reach a
// small commit read on its own (the batch reads every one), which
// TestABoundTipReadOnItsOwnStopsAHungGit reaches.
func viewOnEveryRoute(t *testing.T, crowded bool) *participant {
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
	// A tip that is not a commit, its first sighting kept as a fetch keeps
	// one, so the view's read of it asks its tracked check.
	blob := strings.TrimSpace(runGitStdin(t, p.dir, "not a commit\n", "hash-object", "-w", "--stdin"))
	p.git("update-ref", trackingName("iss-9"), blob)
	tree, err := emptyTree(p.dir)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok, err := readTip(p.dir, "iss-9", trackingName("iss-9"), tree, t0, true); err != nil || !ok || got.Unreadable != reasonNoObject {
		t.Fatalf("the blob tip = %+v, %v, %v; want it unreadable, its first sighting kept", got, ok, err)
	}
	if crowded {
		sha := strings.TrimSpace(p.git("rev-parse", "refs/remotes/origin/main"))
		var lines strings.Builder
		for i := range 2000 {
			fmt.Fprintf(&lines, "create refs/remotes/origin/trunk/b%04d %s\n", i, sha)
		}
		runGitStdin(t, p.dir, lines.String(), "update-ref", "--stdin")
	}
	return p
}

// viewSite is one place the view starts git, matched on the subcommand's
// argv as git was called, and how many times the fixture makes it.
type viewSite struct {
	name  string
	match func(argv string) bool
	times int
}

// argvEnds matches a call whose argv ends with s.
func argvEnds(s string) func(string) bool {
	return func(a string) bool { return strings.HasSuffix(a, s) }
}

// argvHas matches a call whose argv holds s.
func argvHas(s string) func(string) bool {
	return func(a string) bool { return strings.Contains(a, s) }
}

// viewSites are the sites viewOnEveryRoute's view reaches, each with the
// times it is made; crowded adds the default branch's two fallback questions.
func viewSites(crowded bool) []viewSite {
	own := func(id string) string {
		return "for-each-ref --format=%(refname) %(objectname) %(objecttype) " + TrackingPrefix + id
	}
	sites := []viewSite{
		// The tip that is not a commit asks its kept first sighting twice: as
		// its date (dated) and as its first sighting (seen).
		{"the tracked checks (fetch stamp, kept key, the kept first sightings of the tip ahead and twice of the tip not a commit)", argvEnds(" ls-files -z"), 5},
		{"the reservation listing", argvEnds(" for-each-ref --format=%(refname) " + TrackingPrefix), 1},
		{"the release-tag listing", argvEnds(" for-each-ref --format=%(refname) " + ReleaseTagsTracking), 1},
		{"the empty tree's name", argvEnds(" hash-object -t tree --stdin"), 1},
		{"origin's default-branch listing", argvHas(" for-each-ref --format=%(refname) %(symref) "), 1},
		{"the records on origin's default branch", argvHas(" ls-tree -r -z --full-tree refs/remotes/origin/main -- "), 1},
		{"the tip batch's listing", argvEnds(" " + own("")), 1},
		{"the tip batch's headers", argvHas(" cat-file --batch-check="), 1},
		{"the tip batch's bodies", argvEnds(" cat-file --batch"), 1},
		{"the own ref read of the tip too big for the batch", argvEnds(" " + own("iss-8")), 1},
		{"the own ref read of the tip that is not a commit", argvEnds(" " + own("iss-9")), 1},
		{"the size of the tip too big for the batch", argvHas(" cat-file -s "), 1},
		{"the date of the tip too big for the batch", argvHas(" show -s --format=%ct "), 1},
		{"the generated files' tree listing", argvEnds(" ls-tree -r -z --full-tree refs/remotes/origin/main"), 1},
		{"the root attributes' entry", argvEnds(" ls-tree -l -z --full-tree refs/remotes/origin/main -- .gitattributes"), 1},
		{"the root attributes' blob", argvHas(" cat-file blob "), 1},
	}
	if crowded {
		sites = append(sites,
			viewSite{"the default branch's symbolic-ref question", argvEnds(" symbolic-ref --quiet refs/remotes/origin/HEAD"), 1},
			viewSite{"the default branch's question per ref", argvEnds(" rev-parse --verify --quiet refs/remotes/origin/main^{commit} --"), 1})
	}
	return sites
}

// checkSites fails t for each site log does not hold the times named, and
// for each call no site names (git's own version probe aside).
func checkSites(t *testing.T, log []string, sites []viewSite) {
	t.Helper()
	claimed := make([]bool, len(log))
	for _, s := range sites {
		n := 0
		for i, a := range log {
			if s.match(a) {
				n++
				claimed[i] = true
			}
		}
		if n != s.times {
			t.Errorf("%s: made %d time(s), want %d", s.name, n, s.times)
		}
	}
	for i, a := range log {
		if !claimed[i] && !strings.HasSuffix(a, " version") && a != "version" {
			t.Errorf("a git call no site names: %s", a)
		}
	}
}

// gitOnPath puts a git first on PATH that logs each call's argv and, on call
// n (counting from 1; 0 for none), signals and then sleeps for stallFor in
// place of git; every other call runs the real git. The returned calls
// empties the log and answers it, one argv per call; stalled, once the read
// is done, whether call n came. The stall calls cancel, so a ctx it cancels
// ends exactly while git hangs.
func gitOnPath(t *testing.T, n int, cancel context.CancelFunc) (calls func() []string, stalled func() bool) {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not on PATH")
	}
	bin, dir := t.TempDir(), t.TempDir()
	log, count, fifo := filepath.Join(dir, "calls"), filepath.Join(dir, "count"), filepath.Join(dir, "stall")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> '" + log + "'\necho x >> '" + count + "'\n"
	if n > 0 {
		if err := syscall.Mkfifo(fifo, 0o600); err != nil {
			t.Skipf("no fifo: %v", err)
		}
		script += "if [ $(wc -l < '" + count + "') -eq " + strconv.Itoa(n) + " ]; then printf x > '" + fifo + "'; exec sleep " +
			strconv.Itoa(int(stallFor/time.Second)) + "; fi\n"
	}
	script += "exec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	calls = func() []string {
		raw, _ := os.ReadFile(log)
		_ = os.WriteFile(log, nil, 0o600)
		var out []string
		for _, l := range strings.Split(string(raw), "\n") {
			if l != "" {
				out = append(out, l)
			}
		}
		return out
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
// snapshot the context-free view does, a default-branch listing past its cap
// included (crowded), where both forms refuse the listing whole.
func TestViewContextReadsWhatViewReads(t *testing.T) {
	for _, crowded := range []bool{false, true} {
		t.Run("crowded "+strconv.FormatBool(crowded), func(t *testing.T) {
			p := viewOnEveryRoute(t, crowded)
			want, err := View(p.dir, p.w.clock())
			if err != nil {
				t.Fatal(err)
			}
			if len(want.Reservations) < 5 || want.Generated.files == nil {
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
		})
	}
}

// TestViewContextStartsNoGitOnceEnded: a ctx already ended starts no git and
// answers its error, with no reservation in the snapshot.
func TestViewContextStartsNoGitOnceEnded(t *testing.T) {
	p := viewOnEveryRoute(t, false)
	calls, _ := gitOnPath(t, 0, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	snap, err := ViewContext(ctx, p.dir, p.w.clock())
	if !errors.Is(err, context.Canceled) || len(snap.Reservations) != 0 {
		t.Errorf("a spent ctx = %d reservations, %v; want none and ctx's error", len(snap.Reservations), err)
	}
	if n := calls(); len(n) != 0 {
		t.Errorf("a spent ctx still started git %d time(s): %q", len(n), n)
	}
}

// TestViewContextStopsAHungGitOnEveryRoute: the bound view makes each site
// viewSites names (the crowded fixture, so the default branch's fallback
// questions too), and each git it starts, hung in turn, is stopped when the
// ctx ends, the view answering ctx's error and no snapshot read in part,
// whichever reader the hang was in.
func TestViewContextStopsAHungGitOnEveryRoute(t *testing.T) {
	p := viewOnEveryRoute(t, true)
	calls, _ := gitOnPath(t, 0, nil)
	calls()
	live, cancel := context.WithTimeout(context.Background(), time.Hour)
	if _, err := ViewContext(live, p.dir, p.w.clock()); err != nil {
		t.Fatal(err)
	}
	cancel()
	log := calls()
	checkSites(t, log, viewSites(true))
	if t.Failed() {
		t.Fatalf("the bound view did not make the sites named; the hangs below would prove less than they say:\n  %s", strings.Join(log, "\n  "))
	}
	for n := 1; n <= len(log); n++ {
		t.Run("call "+strconv.Itoa(n), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			_, stalled := gitOnPath(t, n, cancel)
			start := time.Now()
			snap, err := ViewContext(ctx, p.dir, p.w.clock())
			took := time.Since(start)
			if !stalled() {
				t.Fatalf("call %d (%s) never came; the test proves nothing", n, log[n-1])
			}
			if took >= stallFor {
				t.Errorf("the view waited out the hung git (%s)", took)
			}
			if !errors.Is(err, context.Canceled) || len(snap.Reservations) != 0 || snap.Generated.files != nil {
				t.Errorf("a view whose call %d (%s) hung until the ctx ended = %d reservations, %v; want none and ctx's error", n, log[n-1], len(snap.Reservations), err)
			}
		})
	}
}

// boundTipOnItsOwn is a copy holding one small reservation tip, and the batch
// budget that leaves it out of the batch, so the bound reader reads it on its
// own: its ref, its size, and its commit.
func boundTipOnItsOwn(t *testing.T) (p *participant, refs []string, tree string) {
	t.Helper()
	w := newWorld(t)
	p = w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.reserve("iss-1", "a.go"); err != nil {
		t.Fatal(err)
	}
	tree, err := emptyTree(p.dir)
	if err != nil {
		t.Fatal(err)
	}
	return p, []string{TrackingPrefix + "iss-1"}, tree
}

// ownReadSites are the sites the bound reader makes for a small tip past a
// budget of 0: the batch's listing and headers, then the tip's own ref, size
// and commit.
var ownReadSites = []viewSite{
	{"the tip batch's listing", argvEnds(" for-each-ref --format=%(refname) %(objectname) %(objecttype) " + TrackingPrefix), 1},
	{"the tip batch's headers", argvHas(" cat-file --batch-check="), 1},
	{"the tip's own ref read", argvEnds(" for-each-ref --format=%(refname) %(objectname) %(objecttype) " + TrackingPrefix + "iss-1"), 1},
	{"the tip's size", argvHas(" cat-file -s "), 1},
	{"the tip's commit", argvHas(" cat-file commit "), 1},
}

// TestABoundTipReadOnItsOwnStopsAHungGit: a small commit past the batch's
// budget is read on its own through the bound viewGit (its ref, its size and
// its commit), and each git that read starts, hung in turn, is stopped when
// the ctx ends, the read answering ctx's error rather than a tip.
func TestABoundTipReadOnItsOwnStopsAHungGit(t *testing.T) {
	p, refs, tree := boundTipOnItsOwn(t)
	calls, _ := gitOnPath(t, 0, nil)
	calls()
	live, cancel := context.WithTimeout(context.Background(), time.Hour)
	got := readSeenTipsWithin(viewGit{ctx: live}, p.dir, refs, tree, t0, false, 0)
	cancel()
	if !got[0].ok || got[0].err != nil || got[0].t.Unreadable != "" {
		t.Fatalf("a live ctx read the small tip as %+v; the test proves nothing", got[0])
	}
	log := calls()
	checkSites(t, log, ownReadSites)
	if t.Failed() {
		t.Fatalf("the bound reader did not make the sites named:\n  %s", strings.Join(log, "\n  "))
	}
	for n := 1; n <= len(log); n++ {
		t.Run("call "+strconv.Itoa(n), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			_, stalled := gitOnPath(t, n, cancel)
			start := time.Now()
			got := readSeenTipsWithin(viewGit{ctx: ctx}, p.dir, refs, tree, t0, false, 0)
			took := time.Since(start)
			if !stalled() {
				t.Fatalf("call %d (%s) never came; the test proves nothing", n, log[n-1])
			}
			if took >= stallFor {
				t.Errorf("the read waited out the hung git (%s)", took)
			}
			if got[0].ok || !errors.Is(got[0].err, context.Canceled) {
				t.Errorf("a tip read whose call %d (%s) hung until the ctx ended = %+v; want ctx's error and no tip", n, log[n-1], got[0])
			}
		})
	}
}

// TestATipReadItsDeadlineEndsIsNoObjectTooBig: the hook ends its ctx by a
// deadline, whose error reads "context deadline exceeded". A commit read that
// deadline ends answers the deadline's error, never a tip made unreadable as
// too big and dated from the fetch, which matched "exceeded" in the cap's
// refusal.
func TestATipReadItsDeadlineEndsIsNoObjectTooBig(t *testing.T) {
	p, refs, tree := boundTipOnItsOwn(t)
	calls, _ := gitOnPath(t, 0, nil)
	calls()
	live, cancel := context.WithTimeout(context.Background(), time.Hour)
	readSeenTipsWithin(viewGit{ctx: live}, p.dir, refs, tree, t0, false, 0)
	cancel()
	log := calls()
	n := slices.IndexFunc(log, argvHas(" cat-file commit ")) + 1
	if n == 0 {
		t.Fatalf("the bound reader never read the tip's commit; the test proves nothing:\n  %s", strings.Join(log, "\n  "))
	}
	// The deadline falls when the hung commit read signals, so it falls
	// while that read hangs however long the reads before it took.
	ctx := deadlineAtSignal{context.Background(), make(chan struct{})}
	_, stalled := gitOnPath(t, n, sync.OnceFunc(func() { close(ctx.done) }))
	start := time.Now()
	got := readSeenTipsWithin(viewGit{ctx: ctx}, p.dir, refs, tree, t0, false, 0)
	took := time.Since(start)
	if !stalled() {
		t.Fatalf("the commit read (call %d) never came; the test proves nothing", n)
	}
	if took >= stallFor {
		t.Errorf("the read waited out the hung git (%s)", took)
	}
	if got[0].ok || !errors.Is(got[0].err, context.DeadlineExceeded) || got[0].t.Unreadable != "" {
		t.Errorf("a commit read the deadline ended = %+v; want the deadline's error, no tip and nothing unreadable", got[0])
	}
}

// deadlineAtSignal is a ctx that ends as a deadline does, its error
// context.DeadlineExceeded, when done is closed: the stall's own signal, not a
// time guessed before the read began.
type deadlineAtSignal struct {
	context.Context
	done chan struct{}
}

func (c deadlineAtSignal) Done() <-chan struct{} { return c.done }

func (c deadlineAtSignal) Err() error {
	select {
	case <-c.done:
		return context.DeadlineExceeded
	default:
		return nil
	}
}

// TestACapRefusalIsNotAContextsEnd: overCap reads the refusal of each capped
// runner this package's readers fold into an object too big, both as stored
// reads and the bound one, from a real git past a tiny cap, so gitutil
// rewording it fails here; and a context's end, whose error also says
// "exceeded", is never one.
func TestACapRefusalIsNotAContextsEnd(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("README.md", "a project\n")
	r.Commit("a")
	live, cancel := context.WithCancel(context.Background())
	defer cancel()
	for name, read := range map[string]func() (string, error){
		"storedGit":    func() (string, error) { return storedGit(r.Root(), 8, "cat-file", "commit", "HEAD") },
		"storedGitCtx": func() (string, error) { return storedGitCtx(live, r.Root(), 8, "cat-file", "commit", "HEAD") },
	} {
		_, capped := read()
		if capped == nil || !overCap(capped) {
			t.Errorf("%s past its cap: overCap(%v) = false; want the cap's refusal read as one", name, capped)
		}
	}
	for _, err := range []error{
		fmt.Errorf("git cat-file commit x: %w", context.DeadlineExceeded),
		fmt.Errorf("git cat-file commit x: %w", context.Canceled),
		errors.New("git cat-file commit x: exit status 128"),
		nil,
	} {
		if overCap(err) {
			t.Errorf("overCap(%v) = true; want it read as no cap refusal", err)
		}
	}
}
