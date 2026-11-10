package autonomy

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/core/occupancy"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestASessionStateACommitCarriesIsRefused: a session's wake state is this
// copy's own, so one git tracks came with a commit and is refused rather than
// read as the session's.
func TestASessionStateACommitCarriesIsRefused(t *testing.T) {
	root := t.TempDir()
	if err := SaveSession(root, "sess-1", SessionState{SchemaVersion: SchemaVersion}); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSession(context.Background(), root, "sess-1"); err != nil {
		t.Fatalf("this copy's own state: %v", err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "-f", "--", path.Join(SessionsDir, "sess-1.json")}} {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = gittest.Env(t)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if _, err := LoadSession(context.Background(), root, "sess-1"); err == nil || !strings.Contains(err.Error(), "git tracks") {
		t.Fatalf("LoadSession over a tracked state = %v; want it refused as tracked", err)
	}
}

// TestALaneCheckTheBudgetEndedIsACutReading: the question whether git tracks
// the run tier, asked once the check's budget is spent, fails for the budget
// and not for anything tracked, so the reading is cut short, never whole with
// no lanes.
func TestALaneCheckTheBudgetEndedIsACutReading(t *testing.T) {
	f := newFx(t)
	f.writeRun("run-0000000000000001", "itd-1", false, map[string]any{"id": "lane-1", "branch": "feat/lane", "stage": "validate"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := &owedReader{ctx: ctx, root: f.root}
	r.readLanes()
	if !r.cut || len(r.lanes) != 0 {
		t.Fatalf("a lane check the budget ended: cut %v, %d lanes; want a cut reading", r.cut, len(r.lanes))
	}
}

// TestTheStopAndPromptReadsRunNoGitOnASpentDeadline: the session's state, the
// behind note, the copy's resolution and the owed reading all run their git
// under the hook's ctx, so a spent deadline runs none and each read refuses.
func TestTheStopAndPromptReadsRunNoGitOnASpentDeadline(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Commit("seed")
	root := repo.Root()
	if err := SaveSession(root, "sess-1", SessionState{SchemaVersion: SchemaVersion, ChecklistNote: notePrefix + "x"}); err != nil {
		t.Fatal(err)
	}
	calls := countGit(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := LoadSession(ctx, root, "sess-1"); err == nil {
		t.Error("LoadSession read the state on a spent deadline")
	}
	if note, err := TakeChecklistNote(ctx, root, "sess-1"); err == nil || note != "" {
		t.Errorf("TakeChecklistNote took %q on a spent deadline", note)
	}
	if _, _, err := ResolveCopyContext(ctx, root, "sess-1", Options{Home: t.TempDir()}); err == nil {
		t.Error("ResolveCopyContext resolved on a spent deadline")
	}
	if _, err := ReadOwed(ctx, ReadOptions{Root: root, Home: t.TempDir()}); err == nil {
		t.Error("ReadOwed read on a spent deadline")
	}
	if n := calls(); n != 0 {
		t.Errorf("a spent deadline still ran git %d time(s)", n)
	}
}

// hangingIdentityGit puts a git first on PATH that hangs on any call whose
// config key argument is an identity key (user.name, user.email, or the
// identity listing's pattern), logging each such call, and hands every other
// call to the real git. It matches the key argument alone, so a path that
// happens to hold "user" hangs nothing.
func hangingIdentityGit(t *testing.T) (log string) {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not on PATH")
	}
	bin := t.TempDir()
	log = filepath.Join(t.TempDir(), "identity-reads")
	script := "#!/bin/sh\nfor a in \"$@\"; do\n  case \"$a\" in\n    user.name|user.email|'^(user|author|committer)\\.(name|email)$')\n      printf '%s\\n' \"$*\" >> '" + log + "'\n      exec sleep 60;;\n  esac\ndone\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

// TestTheOwedReadsLaterGitCallsAreBound: the owed read's identity reads run
// under its ctx after the root lookup. The first identity read's git hangs
// until the deadline kills it; the reader's holder, read later
// while away, then forks no git at all. Unbound, the first would hang the
// read past its deadline and the second would start a git after it.
func TestTheOwedReadsLaterGitCallsAreBound(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Commit("seed")
	// An identity the environment sets resolves with no git at all, which
	// would pass this test with the identity read unbound.
	for _, k := range []string{"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL"} {
		t.Setenv(k, "")
	}
	log := hangingIdentityGit(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := ReadOwed(ctx, ReadOptions{Root: repo.Root(), Home: t.TempDir(), Record: &Record{Start: time.Now()},
			Forge: func(context.Context, string, ...string) (string, error) { return "", errors.New("no forge here") }})
		done <- err
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("the owed read hung past its deadline on an identity read")
	}
	raw, _ := os.ReadFile(log)
	var identityReads []string
	for _, line := range strings.Split(string(raw), "\n") {
		if line != "" {
			identityReads = append(identityReads, line)
		}
	}
	if len(identityReads) != 1 {
		t.Errorf("identity reads %q, want only the first, killed at the deadline", identityReads)
	}
}

// TestTheFetchStampIsJudgedUnderTheDeadline: the fetch time the owed read
// names is the stamp's, asked of git only under the read's ctx, so a spent
// deadline forks no git for it.
func TestTheFetchStampIsJudgedUnderTheDeadline(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Commit("seed")
	stamp := filepath.Join(repo.Root(), ".abcd", ".work.local", "reservation", "fetched-at")
	if err := os.MkdirAll(filepath.Dir(stamp), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stamp, []byte(time.Now().UTC().Format(time.RFC3339)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	calls := countGit(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := &owedReader{ctx: ctx, root: repo.Root()}
	if got := r.fetchedAt(); got != "an unknown time" {
		t.Errorf("a spent deadline still named the fetch: %q", got)
	}
	if n := calls(); n != 0 {
		t.Errorf("a spent deadline still ran git %d time(s)", n)
	}
}

// TestTheLandingReceiptIsReadUnderTheDeadline: the owed read asks for a
// head's landing receipt under its own ctx, so a spent deadline forks no git
// for it, and the head reads as having none.
func TestTheLandingReceiptIsReadUnderTheDeadline(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write(".gitignore", ".abcd/.work.local/\n")
	repo.Commit("seed")
	head := repo.Git("rev-parse", "HEAD")
	repo.Write(landing.ReceiptsRelDir+"/"+head, "commit "+head+"\n")
	calls := countGit(t)
	if !(&owedReader{ctx: context.Background(), root: repo.Root()}).hasReceipt(head) {
		t.Fatal("a live ctx found no receipt; the test proves nothing")
	}
	if calls() == 0 {
		t.Fatal("a live ctx ran no git; the count proves nothing")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := &owedReader{ctx: ctx, root: repo.Root()}
	if r.hasReceipt(head) {
		t.Error("a spent deadline still found the receipt")
	}
	if !r.cut {
		t.Error("a receipt read the deadline ended is not marked cut short, so it reads as no receipt")
	}
	if n := calls(); n != 0 {
		t.Errorf("a spent deadline still ran git %d time(s)", n)
	}
}

// TestEachOwedReadMarksASpentDeadline: each read of the owed reader that
// passes through cutShort marks the reading cut short once the check's ctx has
// ended, so the verdict fails open rather than judging from what the deadline
// emptied: the author email, the holder, the default branch and the receipt
// folder not read, a receipt not found, the runs' tracked check not made, a
// created-branch list or the other worktrees not listed, the fetch stamp not
// judged, the stack index cut short before it asks for a reflog, the
// reservations as last fetched not read, the live sessions not read, and the
// merge queue's rules and its armed pull requests not read. These are the
// reads named here; the reader's other reads are marked by its git or by
// ranOut, and this test does not hold them.
func TestEachOwedReadMarksASpentDeadline(t *testing.T) {
	f := newFx(t)
	if _, err := claim.CopyKey(f.root); err != nil {
		t.Fatal(err)
	}
	f.receipt(f.head("main"))
	if err := os.MkdirAll(filepath.Join(f.root, runRelDir, "run-0000000000000001"), 0o755); err != nil {
		t.Fatal(err)
	}
	spent, cancel := context.WithCancel(context.Background())
	cancel()
	refused := func(ctx context.Context, _ string, _ ...string) (string, error) { return "", ctx.Err() }
	behind := pullRequest{Number: 1, HeadRefName: "feat/x", BaseRefName: "main", MergeStateStatus: "BEHIND"}
	reads := map[string]func(r *owedReader){
		"the author email":    func(r *owedReader) { r.readEmail() },
		"the holder":          func(r *owedReader) { r.readPerson() },
		"the default branch":  func(r *owedReader) { r.readDefault() },
		"the receipt folder":  func(r *owedReader) { r.readReceiptsMinted() },
		"the landing receipt": func(r *owedReader) { r.hasReceipt(f.head("main")) },
		"the runs' tracked check": func(r *owedReader) {
			r.lanes, r.runDrivers = nil, nil
			r.readLanes()
		},
		"the created branches": func(r *owedReader) { r.recordedBranches() },
		"the other worktrees":  func(r *owedReader) { r.checkedOutElsewhere() },
		"the fetch stamp":      func(r *owedReader) { r.fetchedAt() },
		"the stack index": func(r *owedReader) {
			r.baseNames, r.branches = []string{"main"}, map[string]*localBranch{"main": {name: "main", head: f.head("main")}}
			r.stackIdx()
		},
		"the reservations":      func(r *owedReader) { r.readReservations(ReadOptions{}) },
		"the live sessions":     func(r *owedReader) { r.liveSessions() },
		"the merge queue rules": func(r *owedReader) { r.queueKeepsCurrent(behind) },
		"the armed pull requests": func(r *owedReader) {
			r.queueRead, r.queue = true, queueRule{queue: true}
			r.queueKeepsCurrent(behind)
		},
	}
	for name, read := range reads {
		t.Run(name, func(t *testing.T) {
			r := &owedReader{ctx: spent, root: f.root, home: f.home, defName: "main",
				forge: refused, rulesDeadline: time.Second,
				liveRead: func() ([]occupancy.Session, error) { return nil, spent.Err() }}
			read(r)
			if !r.cut && !r.incomplete {
				t.Errorf("%s, read once the deadline had ended, is not marked cut short", name)
			}
			// The index marks before it asks for a reflog, and so indexes no
			// base: its reflog read would be marked by git alone, so a mark
			// with the base indexed and its reflog asked for holds nothing.
			if name == "the stack index" && (r.stacks == nil || len(r.stacks.heads) > 0) {
				t.Error("the stack index, read once the deadline had ended, still indexed a base and asked for its reflog")
			}
		})
	}
}

// TestAReservationsReadTheDeadlineEndsIsMarkedCutShort: the reservations as
// last fetched are read under the check's ctx (claim.ViewContext), so a spent
// deadline starts no git for them and a git that hangs in the view is stopped
// when the deadline ends; either way the reading is marked cut short and says
// the reservations were not read, rather than counting none as this copy's.
func TestAReservationsReadTheDeadlineEndsIsMarkedCutShort(t *testing.T) {
	f := newFx(t)
	if _, err := claim.CopyKey(f.root); err != nil {
		t.Fatal(err)
	}
	const unread = "the reservations as last fetched were not read before the check's deadline"
	t.Run("spent", func(t *testing.T) {
		calls := countGit(t)
		spent, cancel := context.WithCancel(context.Background())
		cancel()
		r := &owedReader{ctx: spent, root: f.root}
		r.readReservations(ReadOptions{})
		if !r.cut || !slices.ContainsFunc(r.notes, func(n string) bool { return strings.Contains(n, unread) }) {
			t.Errorf("a reservations read once the deadline had ended: cut %v, notes %q; want cut, and the note", r.cut, r.notes)
		}
		if n := calls(); n != 0 {
			t.Errorf("a spent deadline still ran git %d time(s)", n)
		}
	})
	t.Run("hung", func(t *testing.T) {
		o := f.opts()
		o.Reservations = nil
		calls, _ := stallNthGit(t, "", 0, nil)
		calls()
		if whole := f.readWith(o); whole.Incomplete {
			t.Fatalf("with git answering, the reading is whole: %s", describe(whole))
		}
		log := calls()
		at := slices.IndexFunc(log, func(c string) bool {
			return strings.HasSuffix(c, "for-each-ref --format=%(refname) "+claim.TrackingPrefix)
		})
		if at < 0 {
			t.Fatalf("the view never listed the reservations; the test proves nothing: %q", log)
		}
		n := nthWith(log, at, claim.TrackingPrefix)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		_, stalled := stallNthGit(t, claim.TrackingPrefix, n, cancel)
		start := time.Now()
		owed, err := ReadOwed(ctx, o)
		took := time.Since(start)
		if !stalled() {
			t.Fatalf("the view's listing (call %d naming the prefix) never came; the test proves nothing", n)
		}
		if err != nil {
			t.Fatal(err)
		}
		if took >= 60*time.Second {
			t.Errorf("the check waited out the hung git (%s)", took)
		}
		if !owed.Incomplete || !slices.ContainsFunc(owed.Notes, func(n string) bool { return strings.Contains(n, unread) }) {
			t.Errorf("a reservations read the deadline cut short: %s; want it incomplete, saying the reservations were not read", describe(owed))
		}
	})
}

// stallNthGit puts a git first on PATH that logs each call's arguments and, on
// the nth call that has word as an argument, writes to the fifo it makes and
// sleeps far past any deadline; every other call goes to the real git. A
// receive on stalled reports, once ReadOwed is done, whether that call came.
// With n 0 it only logs. The stalled call cancels ctx's cancel, so the test
// ends the check exactly while git hangs, with no deadline to guess.
func stallNthGit(t *testing.T, word string, n int, cancel context.CancelFunc) (calls func() []string, stalled func() bool) {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not on PATH")
	}
	bin, dir := t.TempDir(), t.TempDir()
	log, count, fifo := filepath.Join(dir, "calls"), filepath.Join(dir, "count"), filepath.Join(dir, "stall")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> '" + log + "'\n"
	if n > 0 {
		if err := syscall.Mkfifo(fifo, 0o600); err != nil {
			t.Skipf("no fifo: %v", err)
		}
		script += "for a in \"$@\"; do\n  if [ \"$a\" = '" + word + "' ]; then\n" +
			"    c=$(( $(cat '" + count + "' 2>/dev/null || echo 0) + 1 ))\n    echo $c > '" + count + "'\n" +
			"    if [ $c -eq " + strconv.Itoa(n) + " ]; then printf x > '" + fifo + "'; exec sleep 60; fi\n    break\n  fi\ndone\n"
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
		// A stall that never came leaves the reader waiting: a writer that
		// opens and closes releases it.
		if w, err := os.OpenFile(fifo, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
			_ = w.Close()
		}
		<-done
		return came
	}
}

// nthWith is the position among calls that have word as an argument of the
// call at index i, 0 when calls[i] has none.
func nthWith(calls []string, i int, word string) int {
	n := 0
	for j := 0; j <= i; j++ {
		if slices.Contains(strings.Fields(calls[j]), word) {
			n++
			if j == i {
				return n
			}
		}
	}
	return 0
}

// TestAReceiptReadTheDeadlineCutsShortFailsOpen: a stacked branch whose head
// has a receipt owes nothing; when the check's deadline ends inside that
// receipt read, while git lists the worktrees or checks the receipt is
// tracked, the reading is incomplete, so the stop fails open, and no
// "no receipt names its head" item is made up from the read the deadline
// emptied. The branch is the last one judged and has no local branch, so no
// later git call can mark the cut on the receipt read's behalf.
//
// The test finds the receipt read as the check's last two git calls, which
// has two limits. A git call added after the last branch's judgement (in
// reservationItems or breachNote, say) moves the read off the end, and the
// test then fails loudly on its own check rather than passing. And it covers
// only the stacked route's receipt read: the hand-back route's (handBack)
// has no test at this level, and is held only by hasReceipt's own mark,
// which TestTheLandingReceiptIsReadUnderTheDeadline asserts.
func TestAReceiptReadTheDeadlineCutsShortFailsOpen(t *testing.T) {
	const noReceipt = "no receipt names its head"
	owesPreflight := func(o Owed) bool {
		for _, i := range o.Items {
			if strings.Contains(i.Reason, noReceipt) {
				return true
			}
		}
		return false
	}
	setup := func(t *testing.T) (*fx, ReadOptions) {
		f := newFx(t)
		f.git("branch", "feat/base")
		f.commitOn("feat/base", "base work", "")
		f.push("feat/base")
		f.own("feat/base")
		f.git("branch", "scratch", "feat/base")
		top := f.commitOn("scratch", "top work", "")
		f.git("branch", "-D", "scratch")
		f.receipt(top)
		f.reserveFresh("itd-2")
		p := pr(2, "feat/itd-2-top", "feat/base", "alice")
		p.HeadRefOid = top
		f.forge.prs = []fakePR{pr(1, "feat/base", "main", "alice"), p}
		o := f.opts()
		o.Record = f.awayRecord(false)
		return f, o
	}
	for name, word := range map[string]string{"the worktree listing": "worktree", "the tracked check": "ls-files"} {
		t.Run(name, func(t *testing.T) {
			f, o := setup(t)
			calls, _ := stallNthGit(t, "", 0, nil)
			calls()
			whole := f.readWith(o)
			if whole.Incomplete || owesPreflight(whole) || has(whole, OwedStacked, "feat/itd-2-top") {
				t.Fatalf("with git answering, the receipt is found and nothing is owed: %s", describe(whole))
			}
			// The receipt read of the branch judged last makes the check's last
			// two git calls: its worktree listing, then its tracked check.
			log := calls()
			last := len(log) - 1
			if last < 1 || !slices.Contains(strings.Fields(log[last-1]), "worktree") || !slices.Contains(strings.Fields(log[last]), "ls-files") {
				t.Fatalf("the check did not end on the receipt read's listing and tracked check; the test proves nothing: %q", log)
			}
			at := last
			if word == "worktree" {
				at = last - 1
			}
			n := nthWith(log, at, word)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			_, stalled := stallNthGit(t, word, n, cancel)
			owed, err := ReadOwed(ctx, o)
			if !stalled() {
				t.Fatalf("the receipt read's %s (call %d naming %s) never came; the test proves nothing", name, n, word)
			}
			if err != nil {
				t.Fatal(err)
			}
			if !owed.Incomplete {
				t.Errorf("a receipt read the deadline cut short leaves the reading whole: %s", describe(owed))
			}
			if owesPreflight(owed) {
				t.Errorf("a receipt read the deadline cut short made up an owed preflight: %s", describe(owed))
			}
		})
	}
}

// TestASpentDeadlineMakesUpNoMergeOwed: a branch that has its base in it owes
// no merge, and a check whose deadline has ended cannot ask git whether it
// has, so it owes none either way: neither the stacked branch's merge from its
// base on disk nor, with the pull requests unread, the behind item against the
// default branch as last fetched is made up from a read the deadline ended,
// and the reading is marked cut short.
func TestASpentDeadlineMakesUpNoMergeOwed(t *testing.T) {
	f := newFx(t)
	f.stack()
	top := &localBranch{name: "feat/top", head: f.head("feat/top"), upstream: "refs/remotes/origin/feat/top"}
	spent, cancel := context.WithCancel(context.Background())
	cancel()
	reader := func() *owedReader {
		return &owedReader{ctx: spent, root: f.root, def: "refs/remotes/origin/main", defName: "main",
			defTip: f.head("main"), branches: map[string]*localBranch{"feat/top": top}}
	}
	r := reader()
	r.stacked("feat/top", top.head, "feat/base", f.head("feat/base"), pullRequest{}, false)
	if len(r.items) != 0 || !r.cut {
		t.Errorf("a stacked branch read once the deadline had ended: items %v, cut %v; want none, and cut", r.items, r.cut)
	}
	r = reader()
	r.forgeItems("feat/top", top, top.head, pullRequest{}, false)
	if len(r.items) != 0 || !r.cut {
		t.Errorf("a pushed branch judged on disk once the deadline had ended: items %v, cut %v; want none, and cut", r.items, r.cut)
	}
}

// endsAfter is a check's ctx that reads as live for its first n looks at Err
// and as ended from then on. It has no Done channel, so no git is killed by
// it: it only moves the moment the deadline ends between any two looks a
// read takes, wherever n puts it.
type endsAfter struct {
	context.Context
	n int
}

func (c *endsAfter) Err() error {
	if c.n > 0 {
		c.n--
		return nil
	}
	return context.DeadlineExceeded
}

func (c *endsAfter) Done() <-chan struct{} { return nil }

// TestADeadlineEndingBetweenLooksStillMarks: the ancestry check and the
// receipt read owe a merge or a preflight only on an answer the deadline did
// not end, and owe none once it has, marking the reading cut short. Whichever
// look at the deadline it ends between, an item left out is marked: the
// deadline ending after the look that would mark and before the one that
// leaves the item out is the window this sweeps.
func TestADeadlineEndingBetweenLooksStillMarks(t *testing.T) {
	f := newFx(t)
	f.stack()
	top, main := f.head("feat/top"), f.head("main")
	reads := map[string]func(r *owedReader) bool{
		"the ancestry check": func(r *owedReader) bool { return r.notAncestor(top, main) },
		"the receipt read":   func(r *owedReader) bool { return r.owesPreflight(top) },
	}
	const looks = 64
	for name, owes := range reads {
		t.Run(name, func(t *testing.T) {
			if !owes(&owedReader{ctx: &endsAfter{Context: context.Background(), n: looks}, root: f.root}) {
				t.Fatalf("with the deadline live throughout, %s owes nothing; the sweep proves nothing", name)
			}
			for n := 0; n < looks; n++ {
				r := &owedReader{ctx: &endsAfter{Context: context.Background(), n: n}, root: f.root}
				if !owes(r) && !r.cut {
					t.Errorf("the deadline ending after look %d: %s owes nothing and the reading is not marked cut short", n, name)
				}
			}
		})
	}
}

// TestAnAncestryCheckTheDeadlineEndsMidReadMakesUpNoMerge: the deadline
// ending while git answers the ancestry check, not before it, owes no merge
// from a check git was killed in, and marks the reading cut short. The base
// is in the branch, so git, left to answer, owes nothing either.
func TestAnAncestryCheckTheDeadlineEndsMidReadMakesUpNoMerge(t *testing.T) {
	f := newFx(t)
	f.stack()
	top := &localBranch{name: "feat/top", head: f.head("feat/top"), upstream: "refs/remotes/origin/feat/top"}
	reader := func(ctx context.Context) *owedReader {
		return &owedReader{ctx: ctx, root: f.root, def: "refs/remotes/origin/main", defName: "main",
			defTip: f.head("main"), branches: map[string]*localBranch{"feat/top": top}}
	}
	judges := map[string]func(r *owedReader){
		"the stacked branch's base": func(r *owedReader) {
			r.stacked("feat/top", top.head, "feat/base", f.head("feat/base"), pullRequest{}, false)
		},
		"the default branch on disk": func(r *owedReader) {
			r.forgeItems("feat/top", top, top.head, pullRequest{}, false)
		},
	}
	for name, judge := range judges {
		t.Run(name, func(t *testing.T) {
			r := reader(context.Background())
			judge(r)
			if len(r.items) != 0 || r.cut {
				t.Fatalf("with git answering: items %v, cut %v; want none; the test proves nothing", r.items, r.cut)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			_, stalled := stallNthGit(t, "--is-ancestor", 1, cancel)
			r = reader(ctx)
			judge(r)
			if !stalled() {
				t.Fatal("the ancestry check never reached git; the test proves nothing")
			}
			if len(r.items) != 0 || !r.cut {
				t.Errorf("an ancestry check the deadline ended mid-read: items %v, cut %v; want none, and cut", r.items, r.cut)
			}
		})
	}
}

// TestASpentQueueReadMakesUpNoMergeOrArming: a behind pull request into a
// default branch a merge queue may guard is owed a merge or its arming only
// on what the forge answered: a queue read the check's deadline ended, the
// rules or the armed set, owes neither, and marks the reading cut short.
func TestASpentQueueReadMakesUpNoMergeOrArming(t *testing.T) {
	f := newFx(t)
	spent, cancel := context.WithCancel(context.Background())
	cancel()
	refused := func(ctx context.Context, _ string, _ ...string) (string, error) { return "", ctx.Err() }
	behind := pullRequest{Number: 1, HeadRefName: "feat/x", BaseRefName: "main", MergeStateStatus: "BEHIND"}
	reads := map[string]func(r *owedReader){
		"the merge queue rules":   func(*owedReader) {},
		"the armed pull requests": func(r *owedReader) { r.queueRead, r.queue = true, queueRule{queue: true} },
	}
	for name, given := range reads {
		t.Run(name, func(t *testing.T) {
			r := &owedReader{ctx: spent, root: f.root, def: "refs/remotes/origin/main", defName: "main",
				defTip: f.head("main"), forge: refused, rulesDeadline: time.Second}
			given(r)
			r.forgeItems("feat/x", nil, f.head("main"), behind, true)
			if len(r.items) != 0 || !r.cut {
				t.Errorf("%s unread once the deadline had ended: items %v, cut %v; want none, and cut", name, r.items, r.cut)
			}
		})
	}
}
