package loop

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/core/occupancy"
	"github.com/intentdriven/abcd/internal/core/sessionname"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// A fixture machine with two plain terminals under one terminal program: each tab's
// login process, its shell, and a drive started from the shell (iss-2611140902531015).
// termDriveTwo, a second drive from the first tab's shell, is in the table only where a
// test adds it (secondDrive).
const (
	termApp      = 5000
	termLogin    = 5100
	termShell    = 5110
	termDrive    = 5120
	termDriveTwo = 5130
	otherLogin   = 5200
	otherShell   = 5210
	otherDrive   = 5220
)

func terminalProcs(now time.Time, extra ...machineload.Proc) sessionname.Table {
	tab := sessionname.Table{ReadAt: now, Procs: []machineload.Proc{
		{PID: termApp, PPID: 1, Name: "Terminal", Age: 3 * time.Hour},
		{PID: termLogin, PPID: termApp, Name: "login", Age: 2 * time.Hour},
		{PID: termShell, PPID: termLogin, Name: "-zsh", Age: 2 * time.Hour},
		{PID: termDrive, PPID: termShell, Name: "abcd", Age: time.Minute},
		{PID: otherLogin, PPID: termApp, Name: "login", Age: time.Hour},
		{PID: otherShell, PPID: otherLogin, Name: "-zsh", Age: time.Hour},
		{PID: otherDrive, PPID: otherShell, Name: "abcd", Age: time.Minute},
	}}
	tab.Procs = append(tab.Procs, extra...)
	return tab
}

// secondDrive is termDriveTwo, a second drive started from the first tab's shell.
var secondDrive = machineload.Proc{PID: termDriveTwo, PPID: termShell, Name: "abcd", Age: 30 * time.Second}

// standIn has every read of the process table answer tab, the walk start at from, and
// this process be self.
func standIn(t *testing.T, tab sessionname.Table, from, self int) {
	t.Helper()
	oldProcs, oldFrom, oldSelf := readProcs, hereFrom, hereSelf
	readProcs = func() (sessionname.Table, error) { return tab, nil }
	hereFrom = func() int { return from }
	hereSelf = func() int { return self }
	t.Cleanup(func() { readProcs, hereFrom, hereSelf = oldProcs, oldFrom, oldSelf })
}

// takeOverFrom makes the claim `abcd implement resume --take` makes, run from the
// shell named: the take-over's own process sits under that shell.
func takeOverFrom(t *testing.T, repoRoot, runID, home string, tab sessionname.Table, shell int) {
	t.Helper()
	standIn(t, tab, shell, shell+1)
	if err := claimDriver(repoRoot, runID, Options{stopHome: home}, true); err != nil {
		t.Fatalf("the take-over: %v", err)
	}
}

// driveClaims is the claim a drive makes before it steps the run's lane (driveLane).
func driveClaims(t *testing.T, repoRoot, runID, home string, tab sessionname.Table, shell, drive int) error {
	t.Helper()
	standIn(t, tab, shell, drive)
	return ClaimDriver(repoRoot, runID, Options{stopHome: home, driverIsSelf: true})
}

func entriesOf(t *testing.T, repoRoot, runID string) []Entry {
	t.Helper()
	st, err := ReadState(repoRoot, runID)
	if err != nil {
		t.Fatal(err)
	}
	return st.Record
}

// contendsOn fails the test unless err is the driver's contention naming the process.
func contendsOn(t *testing.T, err error, pid int) {
	t.Helper()
	r, ok := AsRefusal(err)
	if !ok || r.Stage != StageDriver || !r.Contention || !strings.Contains(r.Reason, "process "+strconv.Itoa(pid)) {
		t.Fatalf("want the driver's contention over process %d, got %v", pid, err)
	}
}

// TestADriveTakesARunItsOwnTerminalTookOver is the record's sequence: a person takes a
// run over from a plain terminal, which records the terminal's login process as its
// driver, then starts a drive from that same terminal. The drive takes the run, and the
// run's record says why, where it used to wait on the terminal without end.
func TestADriveTakesARunItsOwnTerminalTookOver(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	tab := terminalProcs(time.Now())

	takeOverFrom(t, repo.Root(), runID, home, tab, termShell)
	if c := controlOf(t, repo.Root(), runID); c.Driver == nil || c.Driver.PID != termLogin || c.Driver.Session != "" {
		t.Fatalf("the take-over recorded %+v, not the terminal's login process", c.Driver)
	}
	before := len(entriesOf(t, repo.Root(), runID))

	if err := driveClaims(t, repo.Root(), runID, home, tab, termShell, termDrive); err != nil {
		t.Fatalf("the drive from the take-over's own terminal: %v", err)
	}
	c := controlOf(t, repo.Root(), runID)
	if c.Driver == nil || c.Driver.PID != termDrive || c.Driver.StartUnix == 0 {
		t.Fatalf("the drive did not take the run: %+v", c.Driver)
	}
	rec := entriesOf(t, repo.Root(), runID)
	if len(rec) != before+1 {
		t.Fatalf("the record gained %d entries, want one", len(rec)-before)
	}
	e := rec[len(rec)-1]
	if e.Stage != StageDriver || !strings.Contains(e.Note, "the recorded driver process 5100 is the process this drive's shell runs under, and only shells run beneath it besides this drive") ||
		!strings.Contains(e.Note, "process 5120 took the run") {
		t.Fatalf("the record's last entry: %+v", e)
	}
	// The drive holds it now: its next claim is no change.
	if err := driveClaims(t, repo.Root(), runID, home, tab, termShell, termDrive); err != nil {
		t.Fatal(err)
	}
	if got := len(entriesOf(t, repo.Root(), runID)); got != before+1 {
		t.Fatalf("a second claim by the same drive added to the record: %d", got-before)
	}
}

// TestADriveUnderASessionDoesNotTakeTheRun: a drive beneath a host session that drives
// the run waits on it, whether the session was noted before its claim (the driver names
// it) or only after (the walk from the drive finds it).
func TestADriveUnderASessionDoesNotTakeTheRun(t *testing.T) {
	const host, hostShell, drive = 5300, 5310, 5320
	for _, notedFirst := range []bool{true, false} {
		t.Run(map[bool]string{true: "noted before the claim", false: "noted after it"}[notedFirst], func(t *testing.T) {
			repo, runID, _, _ := awaitingLane(t)
			home, _ := stopWorld(t, false)
			now := time.Now()
			tab := terminalProcs(now,
				machineload.Proc{PID: host, PPID: termShell, Name: "claude", Age: time.Hour},
				machineload.Proc{PID: hostShell, PPID: host, Name: "zsh", Age: time.Second},
				machineload.Proc{PID: drive, PPID: hostShell, Name: "abcd", Age: time.Second})
			note := func() {
				if err := occupancy.Note(home, "s-one", repo.Root(), sessionname.Host{PID: host, StartUnix: now.Add(-time.Hour).Unix()}, tab); err != nil {
					t.Fatal(err)
				}
			}
			if notedFirst {
				note()
			}
			takeOverFrom(t, repo.Root(), runID, home, tab, hostShell)
			if c := controlOf(t, repo.Root(), runID); c.Driver == nil || c.Driver.PID != host || (c.Driver.Session != "") != notedFirst {
				t.Fatalf("the session's take-over recorded %+v", c.Driver)
			}
			if !notedFirst {
				note()
			}
			contendsOn(t, driveClaims(t, repo.Root(), runID, home, tab, hostShell, drive), host)
			if c := controlOf(t, repo.Root(), runID); c.Driver.PID != host {
				t.Fatalf("the drive took a session's run: %+v", c.Driver)
			}
		})
	}
}

// TestADriveBeneathAnotherDriveDoesNotTakeTheRun: a drive started beneath the run's own
// live drive, through an agent that drive launched (its session noted or not) or a shell
// straight under it, waits on the first drive.
func TestADriveBeneathAnotherDriveDoesNotTakeTheRun(t *testing.T) {
	const agent, agentShell, nested, plainShell, plainNested = 5400, 5410, 5420, 5430, 5440
	for _, tc := range []struct {
		name  string
		shell int
		drive int
		noted bool
	}{
		{"through its agent's noted session", agentShell, nested, true},
		{"through its agent, no session noted", agentShell, nested, false},
		{"through a shell under it", plainShell, plainNested, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, runID, _, _ := awaitingLane(t)
			home, _ := stopWorld(t, false)
			now := time.Now()
			tab := terminalProcs(now,
				machineload.Proc{PID: agent, PPID: termDrive, Name: "claude", Age: 10 * time.Second},
				machineload.Proc{PID: agentShell, PPID: agent, Name: "zsh", Age: time.Second},
				machineload.Proc{PID: nested, PPID: agentShell, Name: "abcd", Age: time.Second},
				machineload.Proc{PID: plainShell, PPID: termDrive, Name: "sh", Age: time.Second},
				machineload.Proc{PID: plainNested, PPID: plainShell, Name: "abcd", Age: time.Second})
			if err := driveClaims(t, repo.Root(), runID, home, tab, termShell, termDrive); err != nil {
				t.Fatal(err)
			}
			if tc.noted {
				if err := occupancy.Note(home, "agent-one", repo.Root(), sessionname.Host{PID: agent, StartUnix: now.Add(-10 * time.Second).Unix()}, tab); err != nil {
					t.Fatal(err)
				}
			}
			contendsOn(t, driveClaims(t, repo.Root(), runID, home, tab, tc.shell, tc.drive), termDrive)
			if c := controlOf(t, repo.Root(), runID); c.Driver.PID != termDrive {
				t.Fatalf("the nested drive took the run: %+v", c.Driver)
			}
		})
	}
}

// TestADriveContendsWhereItsTerminalIsNotTheDriver: a drive in another terminal waits on
// the take-over's terminal; so does one whose recorded driver is an ancestor further up
// (a terminal program, or another account's process, is no answer of its walk), and one
// whose comparison is unsure: a walk cut short, or session markers that cannot be read.
func TestADriveContendsWhereItsTerminalIsNotTheDriver(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		name   string
		tab    sessionname.Table
		shell  int
		drive  int
		driver int
		setup  func(t *testing.T, home string)
	}{
		{name: "another terminal", tab: terminalProcs(now), shell: otherShell, drive: otherDrive, driver: termLogin},
		{name: "an ancestor that is not the walk's answer", tab: terminalProcs(now), shell: termShell, drive: termDrive, driver: termApp},
		{name: "a walk cut short", tab: withoutProc(terminalProcs(now), termApp), shell: termShell, drive: termDrive, driver: termLogin},
		{name: "unreadable session markers", tab: terminalProcs(now), shell: termShell, drive: termDrive, driver: termLogin,
			setup: func(t *testing.T, home string) {
				dir := filepath.Join(home, filepath.FromSlash(abcdhome.Rel()))
				if err := os.MkdirAll(dir, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "occupancy"), []byte("not a folder"), 0o600); err != nil {
					t.Fatal(err)
				}
			}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, runID, _, _ := awaitingLane(t)
			home, _ := stopWorld(t, false)
			recordDriver(t, repo.Root(), runID, tc.tab, tc.driver)
			if tc.setup != nil {
				tc.setup(t, home)
			}
			contendsOn(t, driveClaims(t, repo.Root(), runID, home, tc.tab, tc.shell, tc.drive), tc.driver)
			if c := controlOf(t, repo.Root(), runID); c.Driver.PID != tc.driver {
				t.Fatalf("the drive took the run: %+v", c.Driver)
			}
		})
	}
}

// TestAReusedIDIsAGoneDriver: a recorded driver holding the terminal's process id but not
// its start time reads as gone, by the route every claim takes before startedUnder is
// asked, and is replaced as a gone driver, never named as the process this drive's
// shell runs under. It holds that earlier route, which this change leaves as it was.
func TestAReusedIDIsAGoneDriver(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	tab := terminalProcs(time.Now())
	recordDriver(t, repo.Root(), runID, tab, termLogin)
	root, err := os.OpenRoot(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	c := controlOf(t, repo.Root(), runID)
	c.Driver.StartUnix -= int64(24 * time.Hour / time.Second)
	if err := writeControl(root, runID, c); err != nil {
		t.Fatal(err)
	}
	if err := driveClaims(t, repo.Root(), runID, home, tab, termShell, termDrive); err != nil {
		t.Fatal(err)
	}
	rec := entriesOf(t, repo.Root(), runID)
	if e := rec[len(rec)-1]; !strings.Contains(e.Note, "the driver process 5100 is gone") || strings.Contains(e.Note, "runs under") {
		t.Fatalf("the record's last entry: %+v", e)
	}
}

// TestTwoDrivesFromOneTerminal: a drive that takes the run from the terminal holds it, so
// a second drive started from that terminal afterwards waits on the first; and where the
// second is already running when the first claims, it is a process beneath the terminal
// besides the first, so the first waits too, naming it, and neither takes the run.
func TestTwoDrivesFromOneTerminal(t *testing.T) {
	t.Run("the second started after the first took the run", func(t *testing.T) {
		repo, runID, _, _ := awaitingLane(t)
		home, _ := stopWorld(t, false)
		tab := terminalProcs(time.Now())
		takeOverFrom(t, repo.Root(), runID, home, tab, termShell)
		if err := driveClaims(t, repo.Root(), runID, home, tab, termShell, termDrive); err != nil {
			t.Fatal(err)
		}
		both := terminalProcs(tab.ReadAt, secondDrive)
		contendsOn(t, driveClaims(t, repo.Root(), runID, home, both, termShell, termDriveTwo), termDrive)
		if c := controlOf(t, repo.Root(), runID); c.Driver.PID != termDrive {
			t.Fatalf("the second drive took the run: %+v", c.Driver)
		}
	})
	t.Run("both running when the first claims", func(t *testing.T) {
		repo, runID, _, _ := awaitingLane(t)
		home, _ := stopWorld(t, false)
		tab := terminalProcs(time.Now(), secondDrive)
		takeOverFrom(t, repo.Root(), runID, home, tab, termShell)
		for _, pair := range [][2]int{{termDrive, termDriveTwo}, {termDriveTwo, termDrive}} {
			err := driveClaims(t, repo.Root(), runID, home, tab, termShell, pair[0])
			contendsOn(t, err, termLogin)
			keptBy(t, err, pair[1], "abcd", termLogin, "login")
		}
		if c := controlOf(t, repo.Root(), runID); c.Driver.PID != termLogin {
			t.Fatalf("a drive took the run beside another: %+v", c.Driver)
		}
	})
}

// TestAStepUnderTheTakeOversTerminalIsUnchanged: a step that is not a drive claims as
// the host walk names it, so from the take-over's own terminal it is the run's driver
// already, and from another terminal it waits.
func TestAStepUnderTheTakeOversTerminalIsUnchanged(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	tab := terminalProcs(time.Now())
	takeOverFrom(t, repo.Root(), runID, home, tab, termShell)
	before := len(entriesOf(t, repo.Root(), runID))

	standIn(t, tab, termShell, termDrive)
	if err := ClaimDriver(repo.Root(), runID, Options{stopHome: home}); err != nil {
		t.Fatal(err)
	}
	if c := controlOf(t, repo.Root(), runID); c.Driver.PID != termLogin {
		t.Fatalf("a step from the terminal moved the driver: %+v", c.Driver)
	}
	if got := len(entriesOf(t, repo.Root(), runID)); got != before {
		t.Fatalf("a step from the terminal added %d entries", got-before)
	}
	standIn(t, tab, otherShell, otherDrive)
	contendsOn(t, ClaimDriver(repo.Root(), runID, Options{stopHome: home}), termLogin)
}

// recordDriver writes pid, as tab reads it, as the run's driver naming no session.
func recordDriver(t *testing.T, repoRoot, runID string, tab sessionname.Table, pid int) {
	t.Helper()
	h, ok := procOf(tab, pid)
	if !ok {
		t.Fatalf("process %d is not in the fixture", pid)
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	c, err := readControl(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	c.Driver = &LiveDriver{PID: h.PID, StartUnix: h.StartUnix, SlackMS: h.Slack.Milliseconds(), At: time.Now().UTC()}
	if err := writeControl(root, runID, c); err != nil {
		t.Fatal(err)
	}
}

func withoutProc(tab sessionname.Table, pid int) sessionname.Table {
	var kept []machineload.Proc
	for _, p := range tab.Procs {
		if p.PID != pid {
			kept = append(kept, p)
		}
	}
	tab.Procs = kept
	return tab
}

// keptBy fails the test unless err's reason names process pid, called name, as what runs
// beneath the recorded driver besides the drive, and its remedy gives the two ways on the
// code accepts: end that process, or close the terminal the run was taken over from so the
// recorded driver (process driver, called driverName) ends, and start the drive from
// another. It never sends the person to a take-over, which a live driver refuses.
func keptBy(t *testing.T, err error, pid int, name string, driver int, driverName string) {
	t.Helper()
	r, ok := AsRefusal(err)
	want := "process " + strconv.Itoa(pid) + " (" + name + ") runs beneath it besides this drive"
	way := "end process " + strconv.Itoa(pid) + " (" + name + "), and this drive's next look takes the run where nothing else runs beneath it; " +
		"or close the terminal the run was taken over from, so that process " + strconv.Itoa(driver) + " (" + driverName + ") ends, and start the drive from another terminal"
	if !ok || !strings.Contains(r.Reason, want) || r.Remedy != way {
		t.Fatalf("want a refusal naming %q with the way on %q, got %v", want, way, err)
	}
	if strings.Contains(r.Remedy, "resume --take") || strings.Contains(r.Remedy, "second terminal") {
		t.Fatalf("the remedy sends the person to a take-over a live driver refuses: %q", r.Remedy)
	}
}

// runLockHeld reports whether the checkout's run lock is held, as it is through a claim's
// save: another holder of the lock file finds it taken.
func runLockHeld(t *testing.T, repoRoot string) bool {
	t.Helper()
	lock := filepath.Join(repoRoot, filepath.FromSlash(RunRelDir), lockFileName)
	err := fsutil.WithFileLock(lock, time.Millisecond, func() error { return nil })
	if errors.Is(err, fsutil.ErrLockContention) {
		return true
	}
	if err != nil {
		t.Fatal(err)
	}
	return false
}

// A fixture terminal multiplexer: one server process, the nearest above every pane's shell
// that is not a shell, with the drive in the first pane and a second pane's shell.
const (
	muxServer    = 6000
	muxShell     = 6010
	muxDrive     = 6020
	muxPaneShell = 6110
	muxPaneWork  = 6120
)

func muxProcs(now time.Time, extra ...machineload.Proc) sessionname.Table {
	tab := sessionname.Table{ReadAt: now, Procs: []machineload.Proc{
		{PID: muxServer, PPID: 1, Name: "tmux", Age: 5 * time.Hour},
		{PID: muxShell, PPID: muxServer, Name: "zsh", Age: 4 * time.Hour},
		{PID: muxDrive, PPID: muxShell, Name: "abcd", Age: time.Minute},
		{PID: muxPaneShell, PPID: muxServer, Name: "zsh", Age: 3 * time.Hour},
	}}
	tab.Procs = append(tab.Procs, extra...)
	return tab
}

// TestADriveWaitsOnAStepBackgroundedInItsShell: a step started in the background from the
// drive's own shell runs beneath the take-over's terminal beside the drive, so the drive
// waits, naming that step and the way on, and the take-over's claim stands. Each way on
// leads on: with the step ended, the same claim takes the run; with the take-over's
// terminal closed instead, a drive from another terminal replaces its gone driver.
func TestADriveWaitsOnAStepBackgroundedInItsShell(t *testing.T) {
	const step = 5150
	stepProc := machineload.Proc{PID: step, PPID: termShell, Name: "abcd", Age: 10 * time.Second}
	for _, way := range []string{"the step ended", "the terminal closed"} {
		t.Run(way, func(t *testing.T) {
			repo, runID, _, _ := awaitingLane(t)
			home, _ := stopWorld(t, false)
			tab := terminalProcs(time.Now(), stepProc)
			takeOverFrom(t, repo.Root(), runID, home, tab, termShell)
			err := driveClaims(t, repo.Root(), runID, home, tab, termShell, termDrive)
			contendsOn(t, err, termLogin)
			keptBy(t, err, step, "abcd", termLogin, "login")
			if c := controlOf(t, repo.Root(), runID); c.Driver.PID != termLogin {
				t.Fatalf("the drive took the run beside a step: %+v", c.Driver)
			}
			if way == "the step ended" {
				if err := driveClaims(t, repo.Root(), runID, home, withoutProc(tab, step), termShell, termDrive); err != nil {
					t.Fatalf("the same claim with the step ended: %v", err)
				}
				if c := controlOf(t, repo.Root(), runID); c.Driver.PID != termDrive {
					t.Fatalf("the drive did not take the run once the step ended: %+v", c.Driver)
				}
				return
			}
			closed := tab
			for _, pid := range []int{termLogin, termShell, termDrive, step} {
				closed = withoutProc(closed, pid)
			}
			if err := driveClaims(t, repo.Root(), runID, home, closed, otherShell, otherDrive); err != nil {
				t.Fatalf("a drive from another terminal once the take-over's closed: %v", err)
			}
			if c := controlOf(t, repo.Root(), runID); c.Driver.PID != otherDrive {
				t.Fatalf("the other terminal's drive did not take the run: %+v", c.Driver)
			}
			rec := entriesOf(t, repo.Root(), runID)
			if e := rec[len(rec)-1]; !strings.Contains(e.Note, "the driver process 5100 is gone") {
				t.Fatalf("the record's last entry: %+v", e)
			}
		})
	}
}

// TestADriveInASharedTerminalProcess: under a terminal multiplexer, every pane's shell has
// the one server process above it, so a take-over from one pane records the server. A
// drive in another pane takes the run only while the other panes sit at their shells: a
// step running in a second pane, or a session host there that left no marker, keeps it
// waiting, named.
func TestADriveInASharedTerminalProcess(t *testing.T) {
	for _, tc := range []struct {
		name string
		work string // what runs in the second pane; "" is nothing but its shell
	}{
		{"a second pane idle at its shell", ""},
		{"a second pane running a step", "abcd"},
		{"a second pane running a session host with no marker", "claude"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, runID, _, _ := awaitingLane(t)
			home, _ := stopWorld(t, false)
			var extra []machineload.Proc
			if tc.work != "" {
				extra = append(extra, machineload.Proc{PID: muxPaneWork, PPID: muxPaneShell, Name: tc.work, Age: time.Minute})
			}
			tab := muxProcs(time.Now(), extra...)
			takeOverFrom(t, repo.Root(), runID, home, tab, muxPaneShell)
			if c := controlOf(t, repo.Root(), runID); c.Driver == nil || c.Driver.PID != muxServer {
				t.Fatalf("the take-over from the second pane recorded %+v, not the server", c.Driver)
			}
			err := driveClaims(t, repo.Root(), runID, home, tab, muxShell, muxDrive)
			if tc.work == "" {
				if err != nil {
					t.Fatalf("the drive beside an idle pane: %v", err)
				}
				if c := controlOf(t, repo.Root(), runID); c.Driver.PID != muxDrive {
					t.Fatalf("the drive did not take the run: %+v", c.Driver)
				}
				rec := entriesOf(t, repo.Root(), runID)
				if e := rec[len(rec)-1]; !strings.Contains(e.Note, "the recorded driver process 6000 is the process this drive's shell runs under") {
					t.Fatalf("the record's last entry: %+v", e)
				}
				return
			}
			contendsOn(t, err, muxServer)
			keptBy(t, err, muxPaneWork, tc.work, muxServer, "tmux")
			if c := controlOf(t, repo.Root(), runID); c.Driver.PID != muxServer {
				t.Fatalf("the drive took the run: %+v", c.Driver)
			}
			// The named process ended, the same claim takes the run.
			if err := driveClaims(t, repo.Root(), runID, home, withoutProc(tab, muxPaneWork), muxShell, muxDrive); err != nil {
				t.Fatalf("the same claim with process %d ended: %v", muxPaneWork, err)
			}
			if c := controlOf(t, repo.Root(), runID); c.Driver.PID != muxDrive {
				t.Fatalf("the drive did not take the run once the second pane's work ended: %+v", c.Driver)
			}
		})
	}
}

// TestADriveUnderAHostWithNoMarkerWaitsOnItsAgent: a session host that left no marker
// records itself, by the host walk, when a step under it claims the run. A drive started
// through the host's own shell walks to that same host and finds no marker; the host's
// agent is in its call beside the drive's shell, so the drive waits, naming it.
func TestADriveUnderAHostWithNoMarkerWaitsOnItsAgent(t *testing.T) {
	const host, agent, hostShell, step, drive = 5300, 5305, 5310, 5315, 5320
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	tab := terminalProcs(time.Now(),
		machineload.Proc{PID: host, PPID: termShell, Name: "claude", Age: time.Hour},
		machineload.Proc{PID: agent, PPID: host, Name: "node", Age: time.Minute},
		machineload.Proc{PID: hostShell, PPID: host, Name: "zsh", Age: time.Second},
		machineload.Proc{PID: drive, PPID: hostShell, Name: "abcd", Age: time.Second})
	standIn(t, tab, hostShell, step)
	if err := ClaimDriver(repo.Root(), runID, Options{stopHome: home}); err != nil {
		t.Fatal(err)
	}
	if c := controlOf(t, repo.Root(), runID); c.Driver == nil || c.Driver.PID != host || c.Driver.Session != "" {
		t.Fatalf("the host's step recorded %+v", c.Driver)
	}
	err := driveClaims(t, repo.Root(), runID, home, tab, hostShell, drive)
	contendsOn(t, err, host)
	keptBy(t, err, agent, "node", host, "claude")
	if c := controlOf(t, repo.Root(), runID); c.Driver.PID != host {
		t.Fatalf("the drive took a host's run: %+v", c.Driver)
	}
}

// TestADriveWaitsWhereASessionMarkerCannotBeRead: one file in the markers' folder that
// does not read as a marker may be the marker of a session above the drive, so the drive
// waits on the take-over's terminal and says why, though the folder lists, naming the
// folder and the file to remove or repair; with the file removed, the same claim takes
// the run.
func TestADriveWaitsWhereASessionMarkerCannotBeRead(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	now := time.Now()
	tab := terminalProcs(now)
	takeOverFrom(t, repo.Root(), runID, home, tab, termShell)
	// A readable marker of a session elsewhere makes the folder, as a session start does.
	if err := occupancy.Note(home, "s-elsewhere", repo.Root(), sessionname.Host{PID: otherLogin, StartUnix: now.Add(-time.Hour).Unix()}, tab); err != nil {
		t.Fatal(err)
	}
	unread := filepath.Join(home, filepath.FromSlash(abcdhome.Rel("occupancy")), "s-unread.json")
	if err := os.WriteFile(unread, []byte("not a marker\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := driveClaims(t, repo.Root(), runID, home, tab, termShell, termDrive)
	contendsOn(t, err, termLogin)
	folder := abcdhome.Display("occupancy")
	r, _ := AsRefusal(err)
	if !strings.Contains(r.Reason, "1 file(s) in "+folder+" could not be read as session markers") {
		t.Fatalf("the refusal does not say a marker could not be read: %v", err)
	}
	if want := "remove or repair each file in " + folder +
		" that cannot be read as a session marker, s-unread.json first; this drive reads them again at its next look"; r.Remedy != want {
		t.Fatalf("the remedy: %q, want %q", r.Remedy, want)
	}
	if c := controlOf(t, repo.Root(), runID); c.Driver.PID != termLogin {
		t.Fatalf("the drive took the run past an unread marker: %+v", c.Driver)
	}
	if err := os.Remove(unread); err != nil {
		t.Fatal(err)
	}
	if err := driveClaims(t, repo.Root(), runID, home, tab, termShell, termDrive); err != nil {
		t.Fatalf("the same claim with the file removed: %v", err)
	}
	if c := controlOf(t, repo.Root(), runID); c.Driver.PID != termDrive {
		t.Fatalf("the drive did not take the run once the file was removed: %+v", c.Driver)
	}
}

// TestADrivesPassingClaimLeavesTheTerminalsClaim: a drive's claim in passing, made for one
// save over a run that is not its lane, never takes the run from the terminal a take-over
// recorded: the control record is left byte for byte as it stood.
func TestADrivesPassingClaimLeavesTheTerminalsClaim(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	tab := terminalProcs(time.Now())
	takeOverFrom(t, repo.Root(), runID, home, tab, termShell)
	stateAtVersion(t, repo.Root(), runID, schemaVersionPrevious)
	path := filepath.Join(repo.Root(), filepath.FromSlash(controlRel(runID)))
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	standIn(t, tab, termShell, termDrive)
	letGo, err := claimInPassing(repo.Root(), runID, Options{stopHome: home, driverIsSelf: true})
	letGo()
	// The passing claim is refused as any claim over a live driver is, with the plain
	// reason and remedy: it never asks what runs beneath the terminal.
	contendsOn(t, err, termLogin)
	if r, _ := AsRefusal(err); strings.Contains(r.Reason, "runs beneath it") || strings.Contains(r.Reason, "could not be read") ||
		!strings.HasPrefix(r.Remedy, "back off: only one driver steps a run") {
		t.Fatalf("the passing claim was judged by what runs beneath the terminal: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("the passing claim changed the control record:\nbefore %s\nafter %s", before, after)
	}
}

// TestADriveJudgesTheTableReadUnderTheLock: the table read outside the run's lock shows
// nothing beneath the terminal but the drive; the one read under the lock shows a step
// started meanwhile, and the drive waits on it. A reading under the lock that fails makes
// the drive wait too, saying its next look reads the table again, and that next look,
// with the table read, takes the run. The stand-in answers by whether the run's lock is
// held, so the test does not depend on how many readings the claim makes on either side.
func TestADriveJudgesTheTableReadUnderTheLock(t *testing.T) {
	const step = 5150
	for _, tc := range []struct {
		name  string
		fails bool
	}{
		{"a step started meanwhile", false},
		{"the reading under the lock fails", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, runID, _, _ := awaitingLane(t)
			home, _ := stopWorld(t, false)
			now := time.Now()
			clean := terminalProcs(now)
			takeOverFrom(t, repo.Root(), runID, home, clean, termShell)
			later := terminalProcs(now, machineload.Proc{PID: step, PPID: termShell, Name: "abcd", Age: time.Second})
			standIn(t, clean, termShell, termDrive)
			underLock := 0
			readProcs = func() (sessionname.Table, error) {
				if !runLockHeld(t, repo.Root()) {
					return clean, nil
				}
				underLock++
				if tc.fails {
					return sessionname.Table{}, errors.New("the stand-in table cannot be read")
				}
				return later, nil
			}
			err := ClaimDriver(repo.Root(), runID, Options{stopHome: home, driverIsSelf: true})
			if underLock == 0 {
				t.Fatal("the claim read no table under the run's lock")
			}
			contendsOn(t, err, termLogin)
			if tc.fails {
				r, _ := AsRefusal(err)
				if !strings.Contains(r.Reason, "could not be read again under the run's lock") {
					t.Fatalf("the refusal does not say the table could not be read: %v", err)
				}
				if want := "nothing to do: this drive reads the process table again at its next look"; r.Remedy != want {
					t.Fatalf("the remedy: %q, want %q", r.Remedy, want)
				}
			} else {
				keptBy(t, err, step, "abcd", termLogin, "login")
			}
			if c := controlOf(t, repo.Root(), runID); c.Driver.PID != termLogin {
				t.Fatalf("the drive took the run: %+v", c.Driver)
			}
			if !tc.fails {
				return
			}
			// The next look reads the table under the lock and takes the run.
			readProcs = func() (sessionname.Table, error) { return clean, nil }
			if err := ClaimDriver(repo.Root(), runID, Options{stopHome: home, driverIsSelf: true}); err != nil {
				t.Fatalf("the next look with the table read: %v", err)
			}
			if c := controlOf(t, repo.Root(), runID); c.Driver.PID != termDrive {
				t.Fatalf("the next look did not take the run: %+v", c.Driver)
			}
		})
	}
}

// TestTheListingSaysATerminalsRunIsTakenByADriveFromIt: `implement resume` lists a run
// whose live driver names no session with the way a drive takes it; a run a session drives
// keeps the plain line.
func TestTheListingSaysATerminalsRunIsTakenByADriveFromIt(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	tab := terminalProcs(time.Now())
	takeOverFrom(t, repo.Root(), runID, home, tab, termShell)
	next := func() string {
		t.Helper()
		l, err := ListRuns(repo.Root(), runID, Options{stopHome: home})
		if err != nil || len(l.Runs) != 1 {
			t.Fatalf("the listing: %+v, %v", l, err)
		}
		return l.Runs[0].Next
	}
	if got := next(); !strings.Contains(got, "names no session") || !strings.Contains(got, "leave it, unless you start `abcd drain --drive` from that driver's own terminal with nothing else running there, which takes it") {
		t.Fatalf("the listing of a terminal's run: %q", got)
	}
	root, err := os.OpenRoot(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	c := controlOf(t, repo.Root(), runID)
	c.Driver.Session = "s-one"
	if err := writeControl(root, runID, c); err != nil {
		t.Fatal(err)
	}
	if got := next(); got != "its driver is alive: leave it" {
		t.Fatalf("the listing of a session's run: %q", got)
	}
}
