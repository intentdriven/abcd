package implement

import (
	"encoding/json"
	"errors"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// carryWords is the gated command of the drive lanes' retry form.
var carryWords = []string{"go", "test", "-trimpath", "-count=1", "-timeout", "7m", "./internal/core/spec"}

// carryReq is one try of a narrow wait in the retry form: a new waiter at pid, in
// the lane dir, gating words, giving up after timeout.
func carryReq(home string, pid int, clock *fakeClock, dir string, words []string, timeout time.Duration, p poll) WaitRequest {
	req := narrowReq(home, pid, clock, p)
	req.Dir, req.Words, req.Timeout = dir, words, timeout
	return req
}

// overduePoll is a whole-run waiter queued past WaitNarrowYieldAfter with a whole
// slot free, which no narrow run ever gets past: the stand-aside alone shuts the
// narrow gate.
func overduePoll(t *testing.T, home string) poll {
	t.Helper()
	writeMachineConfig(t, home, `{"load": {"preflights": 2}}`)
	p := poll{load: 30}
	wholeWaiter(&p, 50001, WaitNarrowYieldAfter+time.Minute)
	runningPreflight(&p, 80000)
	return p
}

// retryUntilOpen runs the retry form: each try a new waiter, a few seconds after
// the last gave up, as a lane's harness runs it again. It returns the try that
// opened, from zero, and its result, or -1 after tries.
func retryUntilOpen(t *testing.T, tries int, try func(i int) WaitRequest) (int, WaitResult) {
	t.Helper()
	var res WaitResult
	for i := range tries {
		req := try(i)
		var err error
		res, err = WaitLoad(req)
		if err != nil {
			t.Fatalf("try %d: %v", i, err)
		}
		if res.Status == WaitOpen {
			return i, res
		}
		if res.Status != WaitTimedOut {
			t.Fatalf("try %d: result = %+v, want it open or timed out", i, res)
		}
		req.Sleep(5 * time.Second)
	}
	return -1, res
}

// TestARetriedNarrowWaitReachesTheStandAsideCap is the drive lane's form: a narrow
// wait of two minutes, run again each time it gives up, while an earlier whole run
// stays overdue. Each try is a new waiter, so the stand-aside it spends must carry
// to the next, or no try ever reaches WaitNarrowYieldCap and the lane stands aside
// for ever. The third try passes the cap and runs ahead of the whole run.
func TestARetriedNarrowWaitReachesTheStandAsideCap(t *testing.T) {
	home := t.TempDir()
	p := overduePoll(t, home)
	dir := t.TempDir()
	clock := &fakeClock{now: waitT0}
	var tryStart time.Time
	opened, res := retryUntilOpen(t, 6, func(i int) WaitRequest {
		tryStart = clock.Now()
		return carryReq(home, 60000+i, clock, dir, carryWords, 2*time.Minute, p)
	})
	if opened != 2 {
		t.Fatalf("opened at try %d (%+v), want the third: two tries of two minutes spend four of the cap's five", opened, res)
	}
	if !slices.Equal(res.WaitedOn, []string{WaitOnWholeRun}) {
		t.Fatalf("result = %+v, want it to have waited on the whole run alone", res)
	}
	if got := clock.Now().Sub(tryStart); got < time.Minute || got > time.Minute+2*testJitter {
		t.Fatalf("the third try opened after %s, want the cap's last minute", got)
	}
	if _, err := os.Lstat(abcdhome.Path(home, WaitNarrowCarryFileName)); err != nil {
		t.Fatalf("the carried budget file: %v", err)
	}
	if got := readCarryFile(t, home); len(got.Entries) != 0 {
		t.Errorf("entries after the open = %+v, want the opened wait's dropped", got.Entries)
	}
}

// TestARetriedNarrowWaitReachesTheQueueCap: the queue cap's clock carries too, so
// narrow waiters queued ahead that alone keep a lane shut are passed after
// WaitNarrowQueueCap counted across its tries, though each try is shorter.
func TestARetriedNarrowWaitReachesTheQueueCap(t *testing.T) {
	home := t.TempDir()
	writeMachineConfig(t, home, `{"load": {"narrow_runs": 2}}`)
	p := poll{load: 3}
	for i := range 5 {
		narrowRun(&p, 51000+i*10, time.Hour, false)
	}
	dir := t.TempDir()
	clock := &fakeClock{now: waitT0}
	opened, res := retryUntilOpen(t, 6, func(i int) WaitRequest {
		return carryReq(home, 60000+i, clock, dir, carryWords, 50*time.Second, p)
	})
	if opened != 2 || !slices.Equal(res.WaitedOn, []string{WaitOnWaiter}) {
		t.Fatalf("opened at try %d (%+v), want the third, past the stuck queue: two tries of 50s spend 100s of the cap's 120s", opened, res)
	}
}

// TestACarriedBudgetIsTheLanesAlone: a different command, a different lane's
// worktree, or the same lane in another run of a drive (its stage's write root)
// starts with nothing carried; the same command in the same lane takes it up.
func TestACarriedBudgetIsTheLanesAlone(t *testing.T) {
	home := t.TempDir()
	p := overduePoll(t, home)
	dir := t.TempDir()
	clock := &fakeClock{now: waitT0}
	stage := func(root string) func(string) string {
		return func(k string) string {
			if k == runner.StageWritesEnv {
				return root
			}
			return ""
		}
	}
	first := carryReq(home, 60000, clock, dir, carryWords, 4*time.Minute, p)
	first.Getenv = stage("/runs/r1/lane-1")
	if res, err := WaitLoad(first); err != nil || res.Status != WaitTimedOut {
		t.Fatalf("first try: %+v, %v", res, err)
	}
	clock.Sleep(5 * time.Second)
	other := slices.Clone(carryWords)
	other[len(other)-1] = "./internal/core/claim"
	for name, c := range map[string]struct {
		dir   string
		words []string
		stage string
	}{
		"another command":     {dir, other, "/runs/r1/lane-1"},
		"another lane":        {t.TempDir(), carryWords, "/runs/r1/lane-1"},
		"the lane in another": {dir, carryWords, "/runs/r2/lane-1"},
		"no stage":            {dir, carryWords, ""},
	} {
		// Two minutes: with the four carried it would open after one.
		req := carryReq(home, 61000, &fakeClock{now: clock.Now()}, c.dir, c.words, 2*time.Minute, p)
		req.Getenv = stage(c.stage)
		res, err := WaitLoad(req)
		if err != nil || res.Status != WaitTimedOut || !slices.Equal(res.Still, []string{WaitOnWholeRun}) {
			t.Errorf("%s: %+v, %v; want it standing aside the whole two minutes, nothing carried", name, res, err)
		}
	}
	same := carryReq(home, 62000, clock, dir, carryWords, 2*time.Minute, p)
	same.Getenv = stage("/runs/r1/lane-1")
	if res, err := WaitLoad(same); err != nil || res.Status != WaitOpen {
		t.Fatalf("the same lane's next try: %+v, %v; want it open past the carried cap", res, err)
	}
}

// TestACarriedBudgetThatCannotBeTrustedIsIgnored: an entry past its lapse, a file
// that does not parse, one that is not a regular file and one another account owns
// each carry nothing, and the wait goes on as a first try would.
func TestACarriedBudgetThatCannotBeTrustedIsIgnored(t *testing.T) {
	for name, spoil := range map[string]func(t *testing.T, home string, clock *fakeClock){
		"lapsed": func(t *testing.T, home string, clock *fakeClock) {
			// Past one timeout since the last try gave up, it is not a retry.
			clock.Sleep(2*time.Minute + time.Second)
		},
		"unparsable": func(t *testing.T, home string, clock *fakeClock) {
			if err := os.WriteFile(abcdhome.Path(home, WaitNarrowCarryFileName), []byte(`{"entries": [`), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"a symlink": func(t *testing.T, home string, clock *fakeClock) {
			path := abcdhome.Path(home, WaitNarrowCarryFileName)
			if err := os.Rename(path, path+".real"); err != nil {
				t.Fatal(err)
			}
			// Relative, so it resolves inside the home and only the refusal stops it.
			if err := os.Symlink(WaitNarrowCarryFileName+".real", path); err != nil {
				t.Fatal(err)
			}
		},
		"a directory": func(t *testing.T, home string, clock *fakeClock) {
			path := abcdhome.Path(home, WaitNarrowCarryFileName)
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(path, 0o700); err != nil {
				t.Fatal(err)
			}
		},
		"another account's": func(t *testing.T, home string, clock *fakeClock) {
			prev := carryOwner
			carryOwner = func(os.FileInfo) (uint32, error) { return uint32(os.Geteuid()) + 1, nil }
			t.Cleanup(func() { carryOwner = prev })
		},
		"unknown owner": func(t *testing.T, home string, clock *fakeClock) {
			prev := carryOwner
			carryOwner = func(os.FileInfo) (uint32, error) { return 0, errors.New("no owner") }
			t.Cleanup(func() { carryOwner = prev })
		},
		"open to others": func(t *testing.T, home string, clock *fakeClock) {
			if err := os.Chmod(abcdhome.Path(home, WaitNarrowCarryFileName), 0o622); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			p := overduePoll(t, home)
			dir := t.TempDir()
			clock := &fakeClock{now: waitT0}
			if res, err := WaitLoad(carryReq(home, 60000, clock, dir, carryWords, 2*time.Minute, p)); err != nil || res.Status != WaitTimedOut {
				t.Fatalf("first try: %+v, %v", res, err)
			}
			if got := readCarryFile(t, home); len(got.Entries) != 1 {
				t.Fatalf("entries after the first try = %+v, want its own", got.Entries)
			}
			spoil(t, home, clock)
			res, err := WaitLoad(carryReq(home, 61000, clock, dir, carryWords, 4*time.Minute, p))
			if err != nil || res.Status != WaitTimedOut || !slices.Equal(res.Still, []string{WaitOnWholeRun}) {
				t.Fatalf("%+v, %v; want the wait to stand aside its whole four minutes, nothing carried", res, err)
			}
		})
	}
}

// TestTheCarriedBudgetFileIsBounded: lapsed entries are pruned on each write, and
// no more than maxCarryEntries are kept, the latest.
func TestTheCarriedBudgetFileIsBounded(t *testing.T) {
	home := t.TempDir()
	root, err := fsutil.EnsureHomeScope(home, abcdhome.Rel(), abcdhome.DirMode)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	seen := make([]int, 3*maxCarrySeen)
	for i := range seen {
		seen[i] = 100000 + i
	}
	entry := func(key string, last time.Time) narrowCarry {
		return narrowCarry{Key: key, First: last.Add(-time.Minute), Timeout: time.Hour, Last: last, Aside: time.Minute, Queue: time.Minute, QueueSeen: seen}
	}
	storeCarry(root, entry("lapsed", waitT0), waitT0)
	t1 := waitT0.Add(2 * time.Hour)
	for i := range maxCarryEntries + 5 {
		storeCarry(root, entry(narrowCarryKey("/lane", "", []string{"go", "test", string(rune('a' + i))}), t1.Add(time.Duration(i)*time.Second)), t1.Add(time.Duration(i)*time.Second))
	}
	got := readCarryFile(t, home)
	if len(got.Entries) != maxCarryEntries {
		t.Fatalf("%d entries kept, want %d", len(got.Entries), maxCarryEntries)
	}
	for _, e := range got.Entries {
		if e.Key == "lapsed" {
			t.Errorf("the lapsed entry is still kept: %+v", e)
		}
		if len(e.QueueSeen) > maxCarrySeen {
			t.Errorf("an entry keeps %d waiters seen, want at most %d", len(e.QueueSeen), maxCarrySeen)
		}
	}
	if !slices.ContainsFunc(got.Entries, func(e narrowCarry) bool { return e.Last.Equal(t1.Add(time.Duration(maxCarryEntries+4) * time.Second)) }) {
		t.Error("the latest entry was dropped, want the oldest dropped first")
	}
	fi, err := os.Lstat(abcdhome.Path(home, WaitNarrowCarryFileName))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != abcdhome.FileMode || fi.Size() > maxCarryBytes {
		t.Errorf("file mode %o, size %d; want %o and at most %d bytes", fi.Mode().Perm(), fi.Size(), abcdhome.FileMode, maxCarryBytes)
	}
}

func readCarryFile(t *testing.T, home string) narrowCarryFile {
	t.Helper()
	raw, err := os.ReadFile(abcdhome.Path(home, WaitNarrowCarryFileName))
	if err != nil {
		t.Fatal(err)
	}
	var f narrowCarryFile
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	return f
}
