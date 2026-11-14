package loop

import (
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
)

// A fixture machine with two plain terminals under one terminal program: each tab's
// login process, its shell, and drives started from the shell (iss-2611140902531015).
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
		{PID: termDriveTwo, PPID: termShell, Name: "abcd", Age: 30 * time.Second},
		{PID: otherLogin, PPID: termApp, Name: "login", Age: time.Hour},
		{PID: otherShell, PPID: otherLogin, Name: "-zsh", Age: time.Hour},
		{PID: otherDrive, PPID: otherShell, Name: "abcd", Age: time.Minute},
	}}
	tab.Procs = append(tab.Procs, extra...)
	return tab
}

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
	if e.Stage != StageDriver || !strings.Contains(e.Note, "the recorded driver process 5100 is the terminal this drive was started from") ||
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

// TestAReusedIDIsNotTheDrivesTerminal: a recorded driver holding the terminal's process
// id but not its start time is a process gone, replaced as one, never named as this
// drive's terminal.
func TestAReusedIDIsNotTheDrivesTerminal(t *testing.T) {
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
	if e := rec[len(rec)-1]; !strings.Contains(e.Note, "the driver process 5100 is gone") || strings.Contains(e.Note, "terminal") {
		t.Fatalf("the record's last entry: %+v", e)
	}
}

// TestTwoDrivesFromOneTerminal: the first drive to take the lock takes the run from the
// terminal; the second then waits on that live drive.
func TestTwoDrivesFromOneTerminal(t *testing.T) {
	repo, runID, _, _ := awaitingLane(t)
	home, _ := stopWorld(t, false)
	tab := terminalProcs(time.Now())
	takeOverFrom(t, repo.Root(), runID, home, tab, termShell)
	if err := driveClaims(t, repo.Root(), runID, home, tab, termShell, termDrive); err != nil {
		t.Fatal(err)
	}
	contendsOn(t, driveClaims(t, repo.Root(), runID, home, tab, termShell, termDriveTwo), termDrive)
	if c := controlOf(t, repo.Root(), runID); c.Driver.PID != termDrive {
		t.Fatalf("the second drive took the run: %+v", c.Driver)
	}
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
	full := terminalProcs(tab.ReadAt)
	h, ok := procOf(full, pid)
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
