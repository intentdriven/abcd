package autonomy

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/core/occupancy"
)

// owedread_drive_test.go — a lane a live drive is building is the drive's
// (iss-2611150851542274). A drive started in a plain terminal records its own
// process as its run's driver and no session, so no session's hold names it;
// while that process runs, and is not the stopping check's own or one above
// it, its lanes and the reservation its run is for are not owed here.

const (
	drivePID  = 7001 // a drive in another terminal
	hostPID   = 7101 // the host this check runs under
	checkPID  = 7102 // the check itself
	driveAge  = 100 * time.Second
	hostAge   = 500 * time.Second
	driveLane = "build/drive-lane"
)

// writeDriverProc lays the run's control record naming a driver process pid
// that started age ago, with session as its session ("" for none).
func (f *fx) writeDriverProc(runID, session string, pid int, age time.Duration) {
	f.t.Helper()
	dir := filepath.Join(f.root, ".abcd", ".work.local", "run", runID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		f.t.Fatal(err)
	}
	start := time.Now().Add(-age).Unix()
	raw := `{"schema_version":1,"driver":{"session":"` + session + `","pid":` + strconv.Itoa(pid) +
		`,"start_unix":` + strconv.FormatInt(start, 10) + `,"at":"2026-11-01T00:00:00Z"}}`
	if err := os.WriteFile(filepath.Join(dir, "control.json"), []byte(raw+"\n"), 0o600); err != nil {
		f.t.Fatal(err)
	}
}

// procsOf is a process table holding procs, counting its reads, and this
// check as checkPID under hostPID.
func procsOf(reads *int, procs ...machineload.Proc) func(context.Context) (machineload.Snapshot, error) {
	return func(context.Context) (machineload.Snapshot, error) {
		*reads++
		all := append([]machineload.Proc{
			{PID: checkPID, PPID: hostPID, Age: time.Second, Name: "abcd", UID: me()},
			{PID: hostPID, PPID: 1, Age: hostAge, Name: "abcd", UID: me()},
		}, procs...)
		return machineload.Snapshot{HasProcs: true, Procs: all}, nil
	}
}

// asCheck stands checkPID in as the stopping check's own process.
func asCheck(t *testing.T) {
	old := selfPID
	selfPID = func() int { return checkPID }
	t.Cleanup(func() { selfPID = old })
}

// thisUID is this account's uid, as the process table reads it.
func thisUID() uint32 { return uint32(os.Geteuid()) }

func liveDrive() machineload.Proc {
	return machineload.Proc{PID: drivePID, PPID: 1, Age: driveAge, Name: "abcd-darwin-arm64", UID: me()}
}

// TestALiveDrivesLaneIsNamedNotOwed is the detector: the lane of a run a live
// drive in another terminal records as its driver, and the reservation that
// run is for, are not owed to this session, whether or not it has a session
// id abcd keys on, and a note names the drive once with what it holds.
func TestALiveDrivesLaneIsNamedNotOwed(t *testing.T) {
	asCheck(t)
	f := newFx(t)
	f.laneBranch("run-0000000000000011", "iss-11", driveLane)
	f.writeDriverProc("run-0000000000000011", "", drivePID, driveAge)
	f.dueReservation("iss-11")
	for _, id := range []string{sessMe, ""} {
		reads, live := 0, 0
		o := f.opts()
		o.Record = f.awayRecord(false)
		o.SessionID = id
		o.ProcTable = procsOf(&reads, liveDrive())
		o.LiveSessions = liveAs(&live, occupancy.Session{ID: sessMe, Checkout: f.root})
		owed := f.readWith(o)
		if has(owed, OwedUnpushed, driveLane) {
			t.Errorf("session %q: a live drive's lane is owed here: %s", id, describe(owed))
		}
		for _, it := range owed.Items {
			if it.Record == "iss-11" {
				t.Errorf("session %q: the reservation a live drive's run is for is owed here: %s %s", id, it.Kind, describe(owed))
			}
		}
		notes := strings.Join(owed.Notes, "\n")
		if n := strings.Count(notes, "a live drive (process 7001) holds"); n != 1 {
			t.Errorf("session %q: the drive is named %d times; once:\n%s", id, n, notes)
		}
		for _, want := range []string{driveLane, "iss-11"} {
			if !strings.Contains(notes, want) {
				t.Errorf("session %q: the drive's note does not name %s:\n%s", id, want, notes)
			}
		}
		if reads != 1 {
			t.Errorf("session %q: the process table was read %d times; once a check", id, reads)
		}
		if v := Judge(o.Record, owed, StopInput{SessionID: id}, SessionState{}, f.now); v.Row == 2 || v.Row == 4 || v.Row == 5 {
			t.Errorf("session %q: the stop is held for a live drive's work: row %d: %s", id, v.Row, v.Reason)
		}
	}
}

// TestADrivesLaneStaysOwedWhenItsDriverCannotBeTold: the lane is owed again
// once the drive has ended, when another process now holds its id, when the
// recorded driver is this check's own host, when the walk up from this check
// is cut short, and when the process table cannot be read.
func TestADrivesLaneStaysOwedWhenItsDriverCannotBeTold(t *testing.T) {
	asCheck(t)
	cases := []struct {
		name  string
		pid   int
		age   time.Duration
		table func(*int) func(context.Context) (machineload.Snapshot, error)
	}{
		{"the drive has ended", drivePID, driveAge, func(n *int) func(context.Context) (machineload.Snapshot, error) {
			return procsOf(n)
		}},
		{"another process holds its id", drivePID, driveAge, func(n *int) func(context.Context) (machineload.Snapshot, error) {
			return procsOf(n, machineload.Proc{PID: drivePID, PPID: 1, Age: 5 * time.Second, Name: "abcd", UID: me()})
		}},
		{"the live process is another program", drivePID, driveAge, func(n *int) func(context.Context) (machineload.Snapshot, error) {
			p := liveDrive()
			p.Name = "sleep"
			return procsOf(n, p)
		}},
		{"the live process is another account's", drivePID, driveAge, func(n *int) func(context.Context) (machineload.Snapshot, error) {
			p := liveDrive()
			p.UID = me() + 1
			return procsOf(n, p)
		}},
		{"the driver is this check's host", hostPID, hostAge, func(n *int) func(context.Context) (machineload.Snapshot, error) {
			return procsOf(n, liveDrive())
		}},
		{"the walk up is cut short", drivePID, driveAge, func(n *int) func(context.Context) (machineload.Snapshot, error) {
			return func(context.Context) (machineload.Snapshot, error) {
				*n++
				return machineload.Snapshot{HasProcs: true, Procs: []machineload.Proc{
					{PID: checkPID, PPID: hostPID, Age: time.Second, Name: "abcd", UID: me()}, liveDrive(),
				}}, nil
			}
		}},
		{"the table cannot be read", drivePID, driveAge, func(n *int) func(context.Context) (machineload.Snapshot, error) {
			return func(context.Context) (machineload.Snapshot, error) {
				*n++
				return machineload.Snapshot{}, errors.New("no process table")
			}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFx(t)
			f.laneBranch("run-0000000000000011", "iss-11", driveLane)
			f.writeDriverProc("run-0000000000000011", "", c.pid, c.age)
			f.reserveFresh("iss-11")
			reads, live := 0, 0
			o := f.opts()
			o.Record = f.awayRecord(false)
			o.SessionID = sessMe
			o.ProcTable = c.table(&reads)
			o.LiveSessions = liveAs(&live, occupancy.Session{ID: sessMe, Checkout: f.root})
			owed := f.readWith(o)
			if !has(owed, OwedUnpushed, driveLane) {
				t.Errorf("the lane must stay owed: %s", describe(owed))
			}
			if strings.Contains(strings.Join(owed.Notes, "\n"), "a live drive") {
				t.Errorf("a drive is named where none was told:\n%s", strings.Join(owed.Notes, "\n"))
			}
		})
	}
}

// TestADriverWithASessionIsNeverReadAsADrive: a run whose recorded driver
// names a session is judged by that session alone, so a live process under
// an ended session leaves its lane owed, as before.
func TestADriverWithASessionIsNeverReadAsADrive(t *testing.T) {
	asCheck(t)
	f := newFx(t)
	f.laneBranch("run-0000000000000011", "iss-11", driveLane)
	f.writeDriverProc("run-0000000000000011", "sess-ended", drivePID, driveAge)
	f.reserveFresh("iss-11")
	reads, live := 0, 0
	o := f.opts()
	o.SessionID = sessMe
	o.ProcTable = procsOf(&reads, liveDrive())
	o.LiveSessions = liveAs(&live, occupancy.Session{ID: sessMe, Checkout: f.root})
	owed := f.readWith(o)
	if !has(owed, OwedUnpushed, driveLane) {
		t.Fatalf("a lane whose driving session has ended stays owed: %s", describe(owed))
	}
	if reads != 0 {
		t.Errorf("the process table was read %d times for a driver that names a session", reads)
	}
}
