package checklist_test

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/checklist"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/procgroup"
)

// The crash lab (spc-2610230805429429 step 7, the kill half): a writer is
// killed at a random point of its write cycle, again and again, and after
// every kill the reader is asked what it sees. The power-cut half cannot run
// in a test and is owed by a person at each machine.
//
// The writer is this test binary re-executed, so it runs the real path, Update
// with its lock, its read, its scan and its two atomic writes, in a process of
// its own that a SIGKILL can stop anywhere. It leads its own process group, and
// the kill is sent to that group alone, through the pid its own exec handle
// holds while the child is not yet reaped, so a git the writer had started dies
// with it and nothing outside the group is ever signalled.
const (
	// crashWriterEnv names the checkout the re-executed writer writes in; set,
	// TestCrashLabWriterProcess is the writer instead of a skipped test.
	crashWriterEnv = "ABCD_CHECKLIST_CRASH_WRITER"
	// crashStartEnv is the first version that writer writes.
	crashStartEnv = "ABCD_CHECKLIST_CRASH_START"
	// crashKillsEnv sets how many kills the lab makes (crashKillsDefault when
	// unset); the full lab is ABCD_CRASH_LAB_KILLS=1000.
	crashKillsEnv = "ABCD_CRASH_LAB_KILLS"
	// crashSeedEnv replays a run: the seed the lab logs chooses every delay.
	crashSeedEnv = "ABCD_CRASH_LAB_SEED"

	crashKillsDefault = 25
	// crashKillsForWindows is the kill count from which the lab asserts that
	// every window it claims for was hit at least once.
	crashKillsForWindows = 200
	crashKillsMax        = 100000
	// crashVersionsPerWriter spaces each writer's versions apart, so a version
	// read back names the writer that wrote it.
	crashVersionsPerWriter = 100000
	// crashMaxDelay bounds the random wait between a writer's first
	// acknowledged write and its kill: about two write cycles, so the kill
	// lands in every phase of one (the lock, the read, the scan, either temp
	// file and its flush, either rename, the acknowledgement).
	crashMaxDelay = 150 * time.Millisecond
	// crashWriterLife bounds a writer that outlives its test (the parent
	// itself killed), so no writer is left running.
	crashWriterLife = time.Minute
)

// crashVersion is version n of the writer's checklist. Its size changes with
// n, so a file torn at any length could not pass for a whole version.
func crashVersion(n int) checklist.Checklist {
	c := checklist.Checklist{
		Schema:    checklist.Schema,
		UpdatedAt: "2026-11-25T00:00:00Z",
		UpdatedBy: fmt.Sprintf("crash-lab-%d", n),
	}
	states := checklist.States()
	for k := range 20 + n%60 {
		c.Items = append(c.Items, checklist.Item{
			ID:    fmt.Sprintf("v%d-item-%d", n, k),
			Title: fmt.Sprintf("Version %d, item %d: %s", n, k, strings.Repeat("padding ", 15)),
			State: states[k%len(states)],
		})
	}
	return c
}

// crashSecretVersion is version n with a token in one title: the write the
// writer attempts beside every odd version, which must be refused before any
// byte of it reaches the tier. Beside every other version, not once a writer,
// so a kill lands inside a refused write as often as inside a kept one.
func crashSecretVersion(n int, tok string) checklist.Checklist {
	c := crashVersion(n)
	c.Items[0].Title = "Use " + tok
	return c
}

// versionOf is the version a checklist the writer wrote names, or -1.
func versionOf(c checklist.Checklist) int {
	n, err := strconv.Atoi(strings.TrimPrefix(c.UpdatedBy, "crash-lab-"))
	if err != nil || !strings.HasPrefix(c.UpdatedBy, "crash-lab-") {
		return -1
	}
	return n
}

// TestCrashLabWriterProcess is not a test: it is the writer the crash lab
// re-executes. It writes version after version through Update, attempts a
// secret-bearing write before every odd one, printing "secret <n>" before the
// attempt and "refused <n>" once it was refused, and prints "wrote <n>" once
// version n's cycle has returned, until it is killed or its life runs out.
//
// It writes only under the system's temporary directory, where the lab's
// t.TempDir checkout is: an environment that names any other root, by mistake
// or on purpose, never turns a test binary into a writer of a real checklist.
func TestCrashLabWriterProcess(t *testing.T) {
	root := os.Getenv(crashWriterEnv)
	if root == "" {
		t.Skip("the crash lab's writer; runs only when re-executed")
	}
	if tmp := filepath.Clean(os.TempDir()); !filepath.IsAbs(root) || filepath.Clean(root) != root ||
		!strings.HasPrefix(root, tmp+string(filepath.Separator)) {
		fmt.Println("error: the writer's checkout is not under the system's temporary directory")
		os.Exit(1)
	}
	start, err := strconv.Atoi(os.Getenv(crashStartEnv))
	if err != nil {
		fmt.Println("error: no start version")
		os.Exit(1)
	}
	tok := token()
	deadline := time.Now().Add(crashWriterLife)
	for n := start; time.Now().Before(deadline); n++ {
		if n%2 == 1 {
			fmt.Printf("secret %d\n", n)
			err := checklist.Update(root, func(checklist.Result) (checklist.Checklist, error) {
				return crashSecretVersion(n, tok), nil
			})
			var se *checklist.SecretError
			if !errors.As(err, &se) {
				fmt.Printf("error: a secret-bearing write was not refused: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("refused %d\n", n)
		}
		if err := checklist.Update(root, func(checklist.Result) (checklist.Checklist, error) {
			return crashVersion(n), nil
		}); err != nil {
			fmt.Printf("error: version %d: %v\n", n, err)
			os.Exit(1)
		}
		fmt.Printf("wrote %d\n", n)
	}
	os.Exit(0)
}

// crashWriter is one running writer and what it has printed.
type crashWriter struct {
	cmd    *exec.Cmd
	stderr bytes.Buffer
	first  chan struct{} // closed on the first "wrote" line, or on EOF
	done   chan struct{} // closed once stdout reached EOF

	mu     sync.Mutex
	acked  int // the last version it acknowledged, -1 for none
	writes int
	// inSecret is whether its last word was "secret" with no "refused" after:
	// once it is killed, whether the kill fell inside a refused write.
	inSecret bool
	failed   string // an "error:" line it printed
	reaped   bool
}

// startCrashWriter re-executes the test binary as a writer in root, its
// versions starting at start, leading a process group of its own.
func startCrashWriter(t *testing.T, root string, start int) *crashWriter {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestCrashLabWriterProcess$", "-test.count=1",
		"-test.timeout="+(2*crashWriterLife).String())
	// The writer flushes as a shipped binary does, whatever the run's own
	// opt-out: the flush is part of the write cycle the kills land in.
	cmd.Env = append(os.Environ(), crashWriterEnv+"="+root, crashStartEnv+"="+strconv.Itoa(start),
		fsutil.SkipFlushEnv+"=0")
	procgroup.Lead(cmd)
	cmd.WaitDelay = 5 * time.Second
	w := &crashWriter{cmd: cmd, first: make(chan struct{}), done: make(chan struct{}), acked: -1}
	cmd.Stderr = &w.stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// A test that failed before its kill leaves the writer running: kill
		// its group while the handle is still unreaped, and so still its own.
		w.mu.Lock()
		reaped := w.reaped
		w.mu.Unlock()
		if !reaped {
			_ = procgroup.Kill(cmd)
			<-w.done
			_ = cmd.Wait()
		}
	})
	go func() {
		defer close(w.done)
		once := sync.OnceFunc(func() { close(w.first) })
		defer once()
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			line := sc.Text()
			w.mu.Lock()
			if n, ok := strings.CutPrefix(line, "wrote "); ok {
				if v, err := strconv.Atoi(n); err == nil {
					w.acked = v
					w.writes++
				}
			} else if strings.HasPrefix(line, "secret ") {
				w.inSecret = true
			} else if strings.HasPrefix(line, "refused ") {
				w.inSecret = false
			} else if strings.HasPrefix(line, "error:") {
				w.failed = line
			}
			w.mu.Unlock()
			if strings.HasPrefix(line, "wrote ") || strings.HasPrefix(line, "error:") {
				once()
			}
		}
	}()
	return w
}

// killAfter waits for the writer's first acknowledged write, lets it run on
// for delay, and kills its process group. The handle is re-checked first: the
// child is unreaped, so its pid cannot have been reused, and it must still
// lead the group it was started in; only then is the group signalled. It
// returns once the writer is reaped and nothing of its group still runs.
func (w *crashWriter) killAfter(t *testing.T, delay time.Duration) {
	t.Helper()
	select {
	case <-w.first:
	case <-time.After(30 * time.Second):
		t.Fatal("the writer acknowledged no write in 30s")
	}
	w.mu.Lock()
	failed := w.failed
	w.mu.Unlock()
	if failed != "" {
		t.Fatalf("the writer failed before its kill: %s", failed)
	}
	time.Sleep(delay)

	pid := w.cmd.Process.Pid
	if w.cmd.ProcessState != nil {
		t.Fatalf("the writer %d was reaped before its kill; not signalling", pid)
	}
	if pgid, err := syscall.Getpgid(pid); err != nil || pgid != pid {
		t.Fatalf("the writer %d no longer leads its own process group (pgid %d, %v); not signalling", pid, pgid, err)
	}
	if err := procgroup.Kill(w.cmd); err != nil {
		t.Fatalf("killing the writer's group: %v", err)
	}
	<-w.done
	err := w.cmd.Wait()
	w.mu.Lock()
	w.reaped = true
	failed = w.failed
	w.mu.Unlock()
	if failed != "" {
		t.Fatalf("the writer failed: %s", failed)
	}
	if st, ok := w.cmd.ProcessState.Sys().(syscall.WaitStatus); !ok || !st.Signaled() || st.Signal() != syscall.SIGKILL {
		t.Fatalf("the writer ended other than by the kill (%v); stderr: %s", err, w.stderr.String())
	}
	// Clean is what is running: the group is empty once its last member (a git
	// the writer started) is gone. Signal 0 reaches no process.
	for wait := time.Now().Add(5 * time.Second); ; {
		if err := syscall.Kill(-pid, 0); errors.Is(err, syscall.ESRCH) {
			break
		}
		if time.Now().After(wait) {
			t.Fatalf("a process of the killed writer's group %d still runs", pid)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// crashLabKills is how many kills to make: crashKillsEnv, or the default.
func crashLabKills(t *testing.T) int {
	t.Helper()
	v := os.Getenv(crashKillsEnv)
	if v == "" {
		return crashKillsDefault
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 || n > crashKillsMax {
		t.Fatalf("%s must be a whole number from 1 to %d", crashKillsEnv, crashKillsMax)
	}
	return n
}

// crashLabSeed is the seed every delay is drawn from: crashSeedEnv to replay
// a run, or the clock.
func crashLabSeed(t *testing.T) uint64 {
	t.Helper()
	v := os.Getenv(crashSeedEnv)
	if v == "" {
		return uint64(time.Now().UnixNano())
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		t.Fatalf("%s must be an unsigned whole number", crashSeedEnv)
	}
	return n
}

// tempFiles is the set of the atomic writer's temp files in the tier.
func tempFiles(t *testing.T, tier string) map[string]bool {
	t.Helper()
	entries, err := os.ReadDir(tier)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".abcd-tmp-") {
			got[e.Name()] = true
		}
	}
	return got
}

// assertTierHoldsNoSecret fails when any file in the tier, a torn temp file
// included, carries the token's prefix or any eight of its characters in a row:
// a refused write reaches no byte of disk. Eight, not assertNoEcho's four: the
// lab's files run to tens of kilobytes of version numbers, in which a run of
// four of the token's random characters turns up by chance.
func assertTierHoldsNoSecret(t *testing.T, tier, tok string) {
	t.Helper()
	entries, err := os.ReadDir(tier)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		text := string(readFile(t, filepath.Join(tier, e.Name())))
		if strings.Contains(text, tok[:4]) {
			t.Fatalf("%s holds the refused secret's prefix", e.Name())
		}
		for i := 0; i+8 <= len(tok); i++ {
			if strings.Contains(text, tok[i:i+8]) {
				t.Fatalf("%s holds part of the refused secret", e.Name())
			}
		}
	}
}

// A writer killed anywhere in its cycle leaves a list the reader can use: the
// checklist itself, whole, and never older than the last write the writer
// acknowledged (that version, or the next one when the kill fell after its
// rename and before the acknowledgement); last-good whole and no newer than
// it; and no byte of a refused secret anywhere in the tier. A temp file a
// kill left is never read, and a later write removes it once it is old.
func TestCrashLabKilledWriterLeavesAReadableList(t *testing.T) {
	kills := crashLabKills(t)
	seed := crashLabSeed(t)
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	t.Logf("crash lab: %d kills, seed %d (%s=%d replays it)", kills, seed, crashSeedEnv, seed)

	root := newRepo(t)
	tier := abs(root, checklist.TierRelPath)
	tok := token()
	var tornLeft, afterRename, insideRefused, writes int
	before := tempFiles(t, tier)
	began := time.Now()

	for i := range kills {
		w := startCrashWriter(t, root, (i+1)*crashVersionsPerWriter)
		w.killAfter(t, time.Duration(rng.Int64N(int64(crashMaxDelay))))
		w.mu.Lock()
		acked := w.acked
		writes += w.writes
		if w.inSecret {
			insideRefused++
		}
		w.mu.Unlock()

		got := checklist.Read(root)
		if got.Source != checklist.FromFile {
			t.Fatalf("kill %d: the reader fell back to %s after a kill: %s", i, got.Source, got.Reason)
		}
		v := versionOf(got.Checklist)
		switch v {
		case acked:
		case acked + 1:
			afterRename++
		default:
			t.Fatalf("kill %d: the reader saw version %d; the writer had acknowledged %d", i, v, acked)
		}
		if !reflect.DeepEqual(got.Checklist, crashVersion(v)) {
			t.Fatalf("kill %d: version %d was read back changed", i, v)
		}
		lg, err := checklist.Decode(readFile(t, abs(root, checklist.LastGoodRelPath)))
		if err != nil {
			t.Fatalf("kill %d: last-good does not decode after a kill: %v", i, err)
		}
		if lv := versionOf(lg); (lv != acked && lv != acked+1) || lv > v || !reflect.DeepEqual(lg, crashVersion(lv)) {
			t.Fatalf("kill %d: last-good holds version %d; want a whole %d or %d no newer than the file's %d", i, lv, acked, acked+1, v)
		}
		assertTierHoldsNoSecret(t, tier, tok)

		now := tempFiles(t, tier)
		for name := range now {
			if !before[name] {
				tornLeft++
			}
		}
		before = now
	}
	t.Logf("crash lab: %d kills in %s, %d writes acknowledged, %d kills fell after a rename and before its acknowledgement, %d left a torn temp file, %d fell inside a refused secret-bearing write",
		kills, time.Since(began).Round(time.Millisecond), writes, afterRename, tornLeft, insideRefused)
	// A full lab must have reached each window it claims for: a kill after a
	// rename, one between a temp file and its rename, and one inside a refused
	// write. A short run only logs them; on a disk that flushes fast a few kills
	// may miss the narrower windows, and the default run must not flake there.
	if kills >= crashKillsForWindows {
		for _, w := range []struct {
			name string
			n    int
		}{{"after a rename and before its acknowledgement", afterRename}, {"between a temp file and its rename", tornLeft}, {"inside a refused write", insideRefused}} {
			if w.n == 0 {
				t.Errorf("no kill of %d fell %s, so the lab never tested that window", kills, w.name)
			}
		}
	}

	// What the kills left is removed by the next write, once it is old enough
	// that no running write can own it.
	old := time.Now().Add(-2 * time.Minute)
	for name := range before {
		if err := os.Chtimes(filepath.Join(tier, name), old, old); err != nil {
			t.Fatal(err)
		}
	}
	if err := checklist.Update(root, func(checklist.Result) (checklist.Checklist, error) { return crashVersion(1), nil }); err != nil {
		t.Fatalf("the write after the kills: %v", err)
	}
	if left := tempFiles(t, tier); len(left) != 0 {
		t.Errorf("the write after the kills left %d torn temp file(s) in place", len(left))
	}
	if got := checklist.Read(root); got.Source != checklist.FromFile || versionOf(got.Checklist) != 1 {
		t.Errorf("after the kills, a write was not read back: %s, %s", got.Source, got.Reason)
	}
}

// A temp file torn at any length, or one holding a whole list a writer never
// renamed, is never what the reader shows: beside a checklist it reads the
// checklist, and alone it reads nothing. This is every outcome a kill between
// the temp file's creation and its rename can leave: cut at every byte of the
// first and last stretch of the file, where a decoder is likeliest to be fooled
// (nothing, a bare brace, all but the closing bracket), and at a stride
// between, since each read builds a scanner and every byte would take minutes.
func TestCrashLabTornTempIsNeverRead(t *testing.T) {
	root := newRepo(t)
	tier := abs(root, checklist.TierRelPath)
	unrenamed := marshal(t, crashVersion(7))
	torn := filepath.Join(tier, ".abcd-tmp-2611250001")

	for _, beside := range []bool{false, true} {
		if beside {
			if err := checklist.WriteForTest(root, crashVersion(3)); err != nil {
				t.Fatal(err)
			}
		}
		for _, cut := range tornCuts(len(unrenamed)) {
			plant(t, torn, unrenamed[:cut])
			got := checklist.Read(root)
			switch {
			case beside && (got.Source != checklist.FromFile || versionOf(got.Checklist) != 3):
				t.Fatalf("a temp file cut at byte %d of %d beside the checklist was read: %s, %s", cut, len(unrenamed), got.Source, got.Reason)
			case !beside && got.Source != checklist.FromNothing:
				t.Fatalf("a temp file cut at byte %d of %d with no checklist was read as %s: %s", cut, len(unrenamed), got.Source, got.Reason)
			}
		}
	}
}

// tornCuts is where TestCrashLabTornTempIsNeverRead cuts a file of n bytes:
// every length up to 32 and from n-32 to n whole, and every 97th between.
func tornCuts(n int) []int {
	var cuts []int
	for c := 0; c <= n; c++ {
		if c <= 32 || c >= n-32 || c%97 == 0 {
			cuts = append(cuts, c)
		}
	}
	return cuts
}
