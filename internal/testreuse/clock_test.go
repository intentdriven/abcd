package testreuse

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestAClockNoteIsOneLineAppended (iss-2611190439574731): each note is one line
// of three words the receipt script sums, appended so two writers' lines never cut
// into each other, and the file and its directory are made by the first.
func TestAClockNoteIsOneLineAppended(t *testing.T) {
	clock := filepath.Join(t.TempDir(), "preflight-clock", "1763540000-42")
	if err := noteClock(clock, clockWait, "lock", 6*time.Minute+2*time.Second+400*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if err := noteClock(clock, clockStage, "unit", 1500*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if err := noteClock(clock, clockWait, "race", -time.Second); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(clock)
	if err != nil {
		t.Fatal(err)
	}
	if want := "wait lock 362.400\nstage unit 1.500\nwait race 0.000\n"; string(raw) != want {
		t.Errorf("the clock holds %q, want %q", raw, want)
	}
	if err := noteClock("", clockWait, "lock", time.Minute); err != nil {
		t.Errorf("a note with no clock named: %v", err)
	}
}

// TestAClockNoteTheScriptCouldNotReadIsRefused: a relative clock would land
// wherever the writer runs, and a name of more than one word would shift the
// script's columns.
func TestAClockNoteTheScriptCouldNotReadIsRefused(t *testing.T) {
	dir := t.TempDir()
	for _, c := range []struct{ clock, kind, name string }{
		{"relative/clock", clockWait, "lock"},
		{filepath.Join(dir, "c"), "other", "lock"},
		{filepath.Join(dir, "c"), clockWait, "two words"},
		{filepath.Join(dir, "c"), clockWait, ""},
		{filepath.Join(dir, "c"), clockStage, "Unit"},
	} {
		if err := noteClock(c.clock, c.kind, c.name, time.Second); err == nil {
			t.Errorf("noteClock(%q, %q, %q) was accepted", c.clock, c.kind, c.name)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "c")); !os.IsNotExist(err) {
		t.Errorf("a refused note wrote the clock: %v", err)
	}
}

// TestTheClocksFormIsTheReportsForm: the form the receipt script prints, which
// Go's time.ParseDuration also reads: whole seconds, two-digit minutes and
// seconds after a larger unit.
func TestTheClocksFormIsTheReportsForm(t *testing.T) {
	for d, want := range map[time.Duration]string{
		0:                              "0s",
		400 * time.Millisecond:         "0s",
		500 * time.Millisecond:         "1s",
		42 * time.Second:               "42s",
		3*time.Minute + 12*time.Second: "3m12s",
		41*time.Minute + 7*time.Second: "41m07s",
		6*time.Minute + 2*time.Second:  "6m02s",
		time.Hour + 2*time.Minute + 3*time.Second: "1h02m03s",
		-time.Second: "0s",
	} {
		got := formatClock(d)
		if got != want {
			t.Errorf("formatClock(%v) = %q, want %q", d, got, want)
		}
		if parsed, err := time.ParseDuration(got); err != nil || (d >= 0 && parsed != d.Round(time.Second)) {
			t.Errorf("formatClock(%v) = %q, which reads back as %v (%v)", d, got, parsed, err)
		}
	}
}

// readClock sums a clock file's seconds by "<kind> <name>".
func readClock(t *testing.T, clock string) map[string]float64 {
	t.Helper()
	raw, err := os.ReadFile(clock)
	if err != nil {
		t.Fatalf("the clock: %v", err)
	}
	sums := map[string]float64{}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		f := strings.Fields(line)
		if len(f) != 3 {
			t.Fatalf("a clock line is not three words: %q", line)
		}
		s, err := strconv.ParseFloat(f[2], 64)
		if err != nil {
			t.Fatalf("a clock line's seconds: %q", line)
		}
		sums[f[0]+" "+f[1]] += s
	}
	return sums
}

// TestAStageNotesItsTimeApartFromItsWaits (iss-2611190439574731): a stage notes on
// the preflight's clock the time it spent checking, and apart from it the time it
// spent waiting for another check's lock and going back through the race gate, so
// the run's checked time counts neither wait. It also says its own time beside its
// reused line.
func TestAStageNotesItsTimeApartFromItsWaits(t *testing.T) {
	w := newWorld(t)
	race := []string{"-race", "-timeout", "5m", "./..."}
	root := runGit(t, w.repo, "rev-list", "--max-parents=0", "HEAD")
	dir, err := checkDir(w.home, root, os.Geteuid())
	if err != nil {
		t.Fatal(err)
	}
	clock := filepath.Join(t.TempDir(), "preflight-clock", "1763540000-42")
	const gateWait, hold = time.Second, 1500 * time.Millisecond
	waiting := make(chan struct{})
	hook := func(c *Config) {
		c.Stage = "race"
		c.Clock = clock
		c.RaceGate = func() error { time.Sleep(gateWait); return nil }
		c.Stdout = &onLine{w: c.Stdout, line: "check-tests: waiting for another check", seen: waiting}
	}

	// The hold starts once the stage is seen waiting, not when it starts: what
	// the stage does before it reaches the lock (its git calls) takes however
	// long the machine makes it, and no slack here depends on it.
	release := holdLock(t, filepath.Join(dir, lockName))
	began := time.Now()
	got := make(chan result)
	go func() { got <- w.stage(w.repo, hook, race...) }()
	select {
	case <-waiting:
	case r := <-got:
		release()
		t.Fatalf("the stage never waited for the held lock: %s", r)
	case <-time.After(5 * time.Minute):
		t.Fatal("the stage never reached the held lock")
	}
	time.Sleep(hold)
	release()
	r := <-got
	elapsed := time.Since(began).Seconds()
	if r.code != 0 {
		t.Fatalf("the check failed: %s", r)
	}

	// The stage says it is waiting once its first try at the lock has failed, a
	// whole slice, and each later try that fails adds its slice; the try that
	// takes the lock began at most one slice before the release, and its time
	// is not waiting. The first slice makes up for that one, so the lock's wait
	// is at least the hold; the check asks for the hold less a slice, leaving
	// that slice to the scheduler.
	sums := readClock(t, clock)
	lock, gate, stage := sums["wait lock"], sums["wait race"], sums["stage race"]
	if want := (hold - lockSlice).Seconds(); lock < want {
		t.Errorf("the lock's wait noted %.3fs, want at least %.3fs: %v", lock, want, sums)
	}
	if gate < gateWait.Seconds()-0.05 {
		t.Errorf("the race gate's wait noted %.3fs, want at least %v: %v", gate, gateWait, sums)
	}
	if stage <= 0 || stage+lock+gate > elapsed+0.05 {
		t.Errorf("the stage noted %.3fs checking beside %.3fs waiting, in a run of %.3fs: %v", stage, lock+gate, elapsed, sums)
	}
	if !strings.Contains(r.out, "check-tests: race stage checked in ") || !strings.Contains(r.out, "s at the check directory's lock and the race gate)") {
		t.Errorf("the stage does not say its time:\n%s", r)
	}
	if i, j := strings.Index(r.out, "race stage reused "), strings.Index(r.out, "race stage checked in "); i < 0 || j < i {
		t.Errorf("the stage's time is not beside its reused line:\n%s", r)
	}
}

// onLine passes writes on to w and closes seen the first time one holds line.
type onLine struct {
	mu   sync.Mutex
	w    io.Writer
	line string
	seen chan struct{}
	done bool
}

func (o *onLine) Write(p []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	n, err := o.w.Write(p)
	if !o.done && bytes.Contains(p, []byte(o.line)) {
		o.done = true
		close(o.seen)
	}
	return n, err
}

// TestAStageWithNoClockNotesNothing: outside make preflight no clock is named, and
// the stage writes none, though it still says its time.
func TestAStageWithNoClockNotesNothing(t *testing.T) {
	w := newWorld(t)
	if r := w.stage(w.repo, nil, allPackages...); r.code != 0 {
		t.Fatalf("the check failed: %s", r)
	} else if !strings.Contains(r.out, "check-tests: unit stage checked in ") {
		t.Errorf("the stage does not say its time:\n%s", r)
	}
	if _, err := os.Stat(filepath.Join(w.repo, ".abcd", ".work.local", "preflight-clock")); !os.IsNotExist(err) {
		t.Errorf("a stage with no clock wrote one: %v", err)
	}
}

// TestTheTestsAreNotToldTheClock: a test that ran a stage would note its
// fixture's times on the real run's clock, and go test's cache keys a test on
// every variable it reads, so a clock named afresh each run would keep it from
// ever being reused. So no test line is given the variable.
func TestTheTestsAreNotToldTheClock(t *testing.T) {
	w := newWorld(t)
	w.write("clockless/clockless_test.go", `package clockless

import (
	"os"
	"testing"
)

func TestNoClock(t *testing.T) {
	if v := os.Getenv("`+ClockEnv+`"); v != "" {
		t.Fatalf("the test was told the clock: %q", v)
	}
}
`)
	w.commit("a test that must not see the clock")
	clock := filepath.Join(t.TempDir(), "preflight-clock", "1763540000-42")
	t.Setenv(ClockEnv, clock)
	r := w.stage(w.repo, func(c *Config) { c.Clock = clock }, allPackages...)
	if r.code != 0 {
		t.Fatalf("a test line was told the clock: %s", r)
	}
}

// TestTheScriptNotesTheGatesWaits (iss-2611190439574731): make preflight names a
// clock for a run starting now, under the checkout's local tier; the stages' make
// notes the load gate's wait as it starts, from the run's start, and only once,
// so a make started inside the stages adds nothing; the race gate's wait runs
// from its mark. With no clock named, nothing is written.
func TestTheScriptNotesTheGatesWaits(t *testing.T) {
	r := newGitRepo(t)
	r.write(".gitignore", ".abcd/.work.local/\n")
	r.write("a.txt", "one\n")
	r.commit("one")

	before := time.Now().Unix()
	named := receiptScript(t, r.root, "clock", "new")
	dir, base := filepath.Split(named)
	start, err := clockStart(base)
	if real, _ := filepath.EvalSymlinks(r.root); filepath.Clean(dir) != filepath.Join(real, ".abcd", ".work.local", "preflight-clock") && filepath.Clean(dir) != filepath.Join(r.root, ".abcd", ".work.local", "preflight-clock") {
		t.Errorf("clock new names %q, not a clock under the local tier", named)
	}
	if err != nil || start < before || start > time.Now().Unix() {
		t.Errorf("clock new names %q, whose start %d (%v) is not now", named, start, err)
	}
	if _, err := os.Stat(named); !os.IsNotExist(err) {
		t.Errorf("clock new wrote the clock: %v", err)
	}

	clock := filepath.Join(r.root, ".abcd", ".work.local", "preflight-clock", strconv.FormatInt(time.Now().Unix()-372, 10)+"-4242")
	t.Setenv(ClockEnv, clock)
	receiptScript(t, r.root, "clock", "opened")
	receiptScript(t, r.root, "clock", "opened")
	receiptScript(t, r.root, "clock", "mark", "race")
	receiptScript(t, r.root, "clock", "since", "race")
	receiptScript(t, r.root, "clock", "since", "lock")
	sums := readClock(t, clock)
	if load := sums["wait load"]; load < 372 || load > 374 {
		t.Errorf("the load gate's wait noted %vs, want the 372s since the run began, once: %v", load, sums)
	}
	if race := sums["wait race"]; race > 1 {
		t.Errorf("the race gate's wait noted %vs from a mark made just now: %v", race, sums)
	}
	if _, ok := sums["wait lock"]; ok {
		t.Errorf("a wait with no mark was noted: %v", sums)
	}

	t.Setenv(ClockEnv, "")
	receiptScript(t, r.root, "clock", "opened")
	if entries, _ := os.ReadDir(filepath.Dir(clock)); len(entries) != 1 {
		t.Errorf("with no clock named the script wrote one: %v", entries)
	}
}

// clockStart reads a clock name's start, <start>-<pid>.
func clockStart(base string) (int64, error) {
	start, _, _ := strings.Cut(base, "-")
	return strconv.ParseInt(start, 10, 64)
}

// TestTheReceiptLinePrintsTheRunsTimeWithoutItsWaits (iss-2611190439574731): the
// preflight's last lines give each stage's time and the whole run's, beside the
// reuse counts on the receipt line, and the whole run's checked time leaves out
// every wait the clock holds: the load gate's, the race gate's and a stage's wait
// for the check directory's lock. The run began an hour ago, by its clock's name,
// and waited 10m31s of it, so it checked for 49m29s, not an hour. The gates, build
// and vet are what the hour held besides the waits and the two test stages.
func TestTheReceiptLinePrintsTheRunsTimeWithoutItsWaits(t *testing.T) {
	r := newGitRepo(t)
	r.write(".gitignore", ".abcd/.work.local/\n")
	r.write("a.txt", "one\n")
	head := r.commit("one")
	tree := r.git("rev-parse", head+"^{tree}")
	for _, stage := range []string{"unit", "race"} {
		body := "commit " + head + "\ntree " + tree + "\nstage " + stage + "\nmode reuse\nreused 1\npackages 2\nresult pass\n"
		writeFile(t, filepath.Join(r.root, ".abcd", ".work.local", "preflight-reuse", head, stage+"-aaaaaaaaaaaa"), body, 0o644)
	}
	began := time.Now().Unix() - 3600
	clock := filepath.Join(r.root, ".abcd", ".work.local", "preflight-clock", strconv.FormatInt(began, 10)+"-4242")
	writeFile(t, clock, "wait load 600\nstage unit 120.400\nmark race 1763540000\nwait race 30\nwait lock 0.600\nstage race 1500.000\nstage unit 60.000\n", 0o644)

	state := receiptScript(t, r.root, "state")
	t.Setenv(ClockEnv, clock)
	out := receiptScript(t, r.root, "mint", state)
	lines := strings.Split(out, "\n")
	if len(lines) < 6 || !strings.HasPrefix(lines[len(lines)-6], "preflight: push receipt minted for ") || !strings.Contains(lines[len(lines)-5], "the unit stage reused 1 of 2 packages and the race stage 1 of 2") {
		t.Fatalf("the time is not on the last lines, after the receipt line:\n%s", out)
	}
	last := lines[len(lines)-4:]
	if last[1] != "preflight: stage unit: 3m00s" || last[2] != "preflight: stage race: 25m00s" {
		t.Errorf("the stages' times:\n%s", out)
	}
	checked := within(t, last[3], "preflight: checked in ", " (waited 10m31s: load gate 10m00s, race gate 30s, check lock 1s)", 2969, 2971)
	gates := within(t, last[0], "preflight: stage gates: ", "", 1289, 1291)
	if checked-gates != 1680 {
		t.Errorf("the gates' %ds and the test stages' 1680s do not make the checked %ds:\n%s", gates, checked, out)
	}

	// mint run by hand names no clock, and says nothing of time.
	t.Setenv(ClockEnv, "")
	if out := receiptScript(t, r.root, "mint", state); strings.Contains(out, "checked in") || strings.Contains(out, "stage unit:") {
		t.Errorf("a mint with no clock printed a time:\n%s", out)
	}
}

// TestTheLastLineNamesEachKindOfWait (iss-2611190439574731): the parenthesis
// says where the run waited, one field per kind of wait it had, so a run that
// waited ten seconds at the load gate and six minutes behind another checkout's
// race stage does not read as six minutes at the load gate. A kind with no wait
// is left out, a run that waited for nothing says so, and the total and each
// field read back through time.ParseDuration, the fields adding to the total.
func TestTheLastLineNamesEachKindOfWait(t *testing.T) {
	r := newGitRepo(t)
	r.write(".gitignore", ".abcd/.work.local/\n")
	r.write("a.txt", "one\n")
	r.commit("one")
	state := receiptScript(t, r.root, "state")
	for i, c := range []struct{ notes, want string }{
		{"wait load 10\nwait race 362\n", "(waited 6m12s: load gate 10s, race gate 6m02s)"},
		{"wait race 300\nwait load 4\nwait race 62.2\n", "(waited 6m06s: load gate 4s, race gate 6m02s)"},
		{"wait lock 3.5\n", "(waited 4s: check lock 4s)"},
		{"stage unit 60\nwait lock 0.4\nwait load 0\n", "(waited 0s)"},
		{"", "(waited 0s)"},
	} {
		clock := filepath.Join(r.root, ".abcd", ".work.local", "preflight-clock", strconv.FormatInt(time.Now().Unix()-3600, 10)+"-"+strconv.Itoa(100+i))
		writeFile(t, clock, c.notes, 0o644)
		t.Setenv(ClockEnv, clock)
		out := receiptScript(t, r.root, "mint", state, "records-only")
		last := out[strings.LastIndex(out, "\n")+1:]
		open := strings.Index(last, " (")
		if open < 0 || last[open+1:] != c.want {
			t.Errorf("notes %q end on %q, want the parenthesis %q", c.notes, last, c.want)
			continue
		}
		total, fields, _ := strings.Cut(strings.TrimSuffix(strings.TrimPrefix(last[open+1:], "(waited "), ")"), ": ")
		whole, err := time.ParseDuration(total)
		if err != nil {
			t.Errorf("the total %q: %v", total, err)
		}
		var sum time.Duration
		for _, f := range strings.Split(fields, ", ") {
			if f == "" {
				continue
			}
			d, err := time.ParseDuration(f[strings.LastIndex(f, " ")+1:])
			if err != nil {
				t.Errorf("the field %q: %v", f, err)
			}
			sum += d
		}
		if sum != whole {
			t.Errorf("%q: the fields add to %v, not the total %v", last, sum, whole)
		}
	}
}

// TestAGarbledClockNeverCostsTheReceipt (iss-2611190439574731): the clock is a
// report, never a gate. Whatever its file holds, or if it cannot be read at all,
// mint still mints the receipt and exits 0, and the last line is the report with
// every line that is not a note skipped, or one plain line saying no time is
// reported; never an error from the tool that reads it.
func TestAGarbledClockNeverCostsTheReceipt(t *testing.T) {
	r := newGitRepo(t)
	r.write(".gitignore", ".abcd/.work.local/\n")
	r.write("a.txt", "one\n")
	head := r.commit("one")
	tree := r.git("rev-parse", head+"^{tree}")
	for _, stage := range []string{"unit", "race"} {
		body := "commit " + head + "\ntree " + tree + "\nstage " + stage + "\nmode reuse\nreused 1\npackages 2\nresult pass\n"
		writeFile(t, filepath.Join(r.root, ".abcd", ".work.local", "preflight-reuse", head, stage+"-aaaaaaaaaaaa"), body, 0o644)
	}
	state := receiptScript(t, r.root, "state")
	receipt := filepath.Join(r.root, ".abcd", ".work.local", "preflight-receipts", head)
	clocks := filepath.Join(r.root, ".abcd", ".work.local", "preflight-clock")
	hourAgo := strconv.FormatInt(time.Now().Unix()-3600, 10)

	// Two notes the report must read, among lines it must skip.
	garbled := strings.Join([]string{
		"wait load 10",
		"stage unit 60",
		"", "wait", "wait load", "wait Load 5", "wait load -3", "wait load 1e3",
		"wait load 5s", "wait load 5 extra", "stage two words 7", "stage unit 1.2.3",
		"wait load .5", "mark race notanumber", "\x00\xff\xfe binary \x01",
		"wait load " + strings.Repeat("9", 20000),
		"stage unit 60 # a comment", "wait\tload\t\t", "WAIT LOAD 99",
	}, "\n") + "\n"
	reported := []string{"preflight: stage gates: 58m50s", "preflight: stage unit: 1m00s", "preflight: checked in 59m50s (waited 10s: load gate 10s)"}
	noTime := "preflight: no time reported"

	for _, c := range []struct {
		name  string
		make  func(t *testing.T) string // makes the clock, returns its path
		lines []string                  // the last lines, or nil for noTime
	}{
		{"garbled lines", func(t *testing.T) string {
			clock := filepath.Join(clocks, hourAgo+"-1")
			writeFile(t, clock, garbled, 0o644)
			return clock
		}, reported},
		{"no final newline", func(t *testing.T) string {
			clock := filepath.Join(clocks, hourAgo+"-2")
			writeFile(t, clock, "wait load 10\nstage unit 60", 0o644)
			return clock
		}, reported},
		{"an unreadable file", func(t *testing.T) string {
			if os.Geteuid() == 0 {
				t.Skip("root reads a file whatever its mode")
			}
			clock := filepath.Join(clocks, hourAgo+"-3")
			writeFile(t, clock, "wait load 10\nstage unit 60\n", 0o644)
			if err := os.Chmod(clock, 0); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.Chmod(clock, 0o644) })
			return clock
		}, nil},
		{"a directory", func(t *testing.T) string {
			clock := filepath.Join(clocks, hourAgo+"-4")
			if err := os.MkdirAll(filepath.Join(clock, "inside"), 0o755); err != nil {
				t.Fatal(err)
			}
			return clock
		}, nil},
		{"a name with no start", func(t *testing.T) string {
			clock := filepath.Join(clocks, "garbled-5")
			writeFile(t, clock, "wait load 10\n", 0o644)
			return clock
		}, nil},
		{"a start of leading zeros", func(t *testing.T) string {
			clock := filepath.Join(clocks, "0899-6")
			writeFile(t, clock, "wait load 10\n", 0o644)
			return clock
		}, nil},
		{"a start of too many digits", func(t *testing.T) string {
			clock := filepath.Join(clocks, strings.Repeat("9", 25)+"-7")
			writeFile(t, clock, "wait load 10\n", 0o644)
			return clock
		}, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			if err := os.Remove(receipt); err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			clock := c.make(t)
			script, err := filepath.Abs(filepath.Join("..", "..", "scripts", "preflight-receipt.sh"))
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("bash", script, "mint", state)
			var stdout, stderr bytes.Buffer
			cmd.Dir, cmd.Env, cmd.Stdout, cmd.Stderr = r.root, append(gittest.Env(t), ClockEnv+"="+clock), &stdout, &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("mint over the clock exited %v:\n%s%s", err, stdout.String(), stderr.String())
			}
			if stderr.Len() > 0 {
				t.Errorf("mint over the clock said on stderr:\n%s", stderr.String())
			}
			if _, err := os.Stat(receipt); err != nil {
				t.Errorf("no receipt was minted: %v\n%s", err, stdout.String())
			}
			lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
			if c.lines == nil {
				if last := lines[len(lines)-1]; !strings.HasPrefix(last, noTime) {
					t.Errorf("the last line is %q, want %q…:\n%s", last, noTime, stdout.String())
				}
				return
			}
			if len(lines) < len(c.lines) {
				t.Fatalf("too few lines:\n%s", stdout.String())
			}
			got := lines[len(lines)-len(c.lines):]
			for i, want := range c.lines {
				// The run's start is a second from where the test set it, either way.
				if got[i] != want && !closeTo(got[i], want) {
					t.Errorf("line %d is %q, want %q:\n%s", i, got[i], want, stdout.String())
				}
			}
		})
	}

	// clock since reads the last mark of its name and does arithmetic on it, so a
	// mark that is not Unix seconds by the start's rule (no leading zero, which
	// the shell reads as octal, and no more than twelve digits) is refused: it
	// notes nothing and exits 0, even when an earlier mark of that name was sound.
	for _, mark := range []string{"0899", "0", strings.Repeat("9", 13), "1700000000\nmark race 0899"} {
		t.Run("a race mark of "+strconv.Quote(mark), func(t *testing.T) {
			clock := filepath.Join(clocks, hourAgo+"-9")
			body := "wait load 10\nmark race " + mark + "\n"
			writeFile(t, clock, body, 0o644)
			script, err := filepath.Abs(filepath.Join("..", "..", "scripts", "preflight-receipt.sh"))
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("bash", script, "clock", "since", "race")
			var out bytes.Buffer
			cmd.Dir, cmd.Env, cmd.Stdout, cmd.Stderr = r.root, append(gittest.Env(t), ClockEnv+"="+clock), &out, &out
			if err := cmd.Run(); err != nil {
				t.Errorf("clock since race over the mark exited %v:\n%s", err, out.String())
			}
			if out.Len() > 0 {
				t.Errorf("clock since race over the mark said:\n%s", out.String())
			}
			if raw, err := os.ReadFile(clock); err != nil || string(raw) != body {
				t.Errorf("clock since race noted over the mark: the clock holds %q (%v), want %q", raw, err, body)
			}
		})
	}
}

// closeTo reports whether got is want with its first time a second either way.
func closeTo(got, want string) bool {
	re := regexp.MustCompile(`\b[0-9][0-9hms]*s\b`)
	w := re.FindStringIndex(want)
	g := re.FindStringIndex(got)
	if w == nil || g == nil || got[:g[0]] != want[:w[0]] || got[g[1]:] != want[w[1]:] {
		return false
	}
	gd, err1 := time.ParseDuration(got[g[0]:g[1]])
	wd, err2 := time.ParseDuration(want[w[0]:w[1]])
	return err1 == nil && err2 == nil && (gd-wd).Abs() <= time.Second
}

// within reads the duration a line holds between prefix and suffix and fails the
// test unless it is from lo to hi seconds; it returns the seconds.
func within(t *testing.T, line, prefix, suffix string, lo, hi int) int {
	t.Helper()
	if !strings.HasPrefix(line, prefix) || !strings.HasSuffix(line, suffix) {
		t.Fatalf("%q is not %q<time>%q", line, prefix, suffix)
	}
	d, err := time.ParseDuration(strings.TrimSuffix(strings.TrimPrefix(line, prefix), suffix))
	if err != nil {
		t.Fatalf("%q: %v", line, err)
	}
	s := int(d.Seconds())
	if s < lo || s > hi || d != d.Truncate(time.Second) {
		t.Errorf("%q gives %v, want whole seconds from %d to %d", line, d, lo, hi)
	}
	return s
}
