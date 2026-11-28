package loop

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/core/sessionname"
)

// The reader of a drive's stored state (spc-2611141617199772 step 5): a status surface
// shows the line a drive stored while that drive runs, reads stopped once it is gone
// (criterion 7), and shows no line, saying why, for a file a drive did not write.

// readerPID is the drive's process in the reader's tests, started readerAge before
// the table is read.
const (
	readerPID = 7311
	readerAge = 3 * time.Hour
)

// readerTable is the machine's process table read at now, holding readerPID started
// age before it.
func readerTable(now time.Time, age time.Duration) sessionname.Table {
	return sessionname.Table{ReadAt: now, Procs: []machineload.Proc{
		{PID: 1, Name: "launchd", Age: 48 * time.Hour},
		{PID: readerPID, PPID: 1, Name: "abcd", Age: age},
	}}
}

// readerProcs has every read of the process table answer tab, or fail where err is
// not nil.
func readerProcs(t *testing.T, tab sessionname.Table, err error) {
	t.Helper()
	old := readProcs
	readProcs = func() (sessionname.Table, error) { return tab, err }
	t.Cleanup(func() { readProcs = old })
}

// storedWorking writes, at written, a working drive whose process is p, due again
// within the renewal tick and its margin, and returns the checkout.
func storedWorking(t *testing.T, p DriveProcess, written time.Time) string {
	t.Helper()
	repo := t.TempDir()
	s := DriveState{State: DriveWorking, Since: written.Add(-5 * time.Minute), On: "iss-4 at implement", Process: p,
		Lanes: DriveLanes{Done: 2, InFlight: 1}}
	if _, err := writeDriveState(repo, s, written, terminalRenewTick+driveStateMargin); err != nil {
		t.Fatal(err)
	}
	return repo
}

// readerProcess is readerPID as the drive recorded itself, readerAge before at.
func readerProcess(at time.Time) DriveProcess {
	return DriveProcess{PID: readerPID, StartUnix: at.Add(-readerAge).Unix(), SlackMS: 1000}
}

// While the recorded process runs, the reader shows the value the drive stored.
func TestADriveStateReadsAsStoredWhileItsDriveRuns(t *testing.T) {
	written := driveAt(10)
	repo := storedWorking(t, readerProcess(written), written)
	now := driveAt(12)
	readerProcs(t, readerTable(now, readerAge+2*time.Minute), nil)
	got, err := ReadDriveState(repo, now)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != DriveWorking || got.On != "iss-4 at implement" || !got.Since.Equal(driveAt(5)) || got.Lanes.Done != 2 {
		t.Fatalf("the stored value read back as %+v", got)
	}
	if line := DriveLine(got, now); line != "working 7m · on iss-4 at implement · you: nothing to do" {
		t.Fatalf("the line: %q", line)
	}
}

// A drive whose recorded process is gone reads stopped, not heard from since the value
// was written, and never working, however recently it wrote and whatever it said.
func TestAGoneDriveReadsStoppedNeverWorking(t *testing.T) {
	written := driveAt(10)
	for _, c := range []struct {
		name string
		tab  func(now time.Time) sessionname.Table
	}{
		{"its id is not in the table", func(now time.Time) sessionname.Table { return sessionname.Table{ReadAt: now} }},
		{"its id is held by a process that started later", func(now time.Time) sessionname.Table { return readerTable(now, time.Minute) }},
		{"its id is held by a process that started earlier", func(now time.Time) sessionname.Table { return readerTable(now, readerAge+time.Hour) }},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo := storedWorking(t, readerProcess(written), written)
			now := driveAt(11)
			readerProcs(t, c.tab(now), nil)
			got, err := ReadDriveState(repo, now)
			if err != nil {
				t.Fatal(err)
			}
			if got.State != DriveStopped || !got.Since.Equal(written) || got.Do != "" || got.Until != nil {
				t.Fatalf("a gone drive read as %+v", got)
			}
			want := "stopped 1m · on the drive, not heard from since 2026-11-25T10:10:00Z · you: needs a look"
			if line := DriveLine(got, now); line != want || strings.Contains(line, DriveWorking) {
				t.Fatalf("the line: %q, want %q", line, want)
			}
		})
	}
}

// A drive that stored its own stopped line keeps it once its process is gone: it is
// the drive's last word, with the command that goes on.
func TestADriveThatStoppedKeepsItsOwnLastLine(t *testing.T) {
	repo := t.TempDir()
	written := driveAt(30)
	s := DriveState{State: DriveStopped, Since: written, On: "its end: until", Do: "run `abcd drain --drive` to go on", Process: readerProcess(written)}
	if _, err := writeDriveState(repo, s, written, terminalRenewEvery); err != nil {
		t.Fatal(err)
	}
	now := driveAt(90)
	readerProcs(t, sessionname.Table{ReadAt: now}, nil)
	got, err := ReadDriveState(repo, now)
	if err != nil {
		t.Fatal(err)
	}
	if line := DriveLine(got, now); line != "stopped 1h00m · on its end: until · you: run `abcd drain --drive` to go on" {
		t.Fatalf("the line: %q", line)
	}
}

// Where the process cannot be judged, because the record names no start or the table
// cannot be read, the drive's own next-write time decides: as stored until it passes,
// stopped from then on.
func TestADriveThatCannotBeJudgedByItsProcessReadsByItsNextWrite(t *testing.T) {
	written := driveAt(10)
	next := written.Add(terminalRenewTick + driveStateMargin)
	for _, c := range []struct {
		name string
		p    DriveProcess
		err  error
	}{
		{"the record names no start", DriveProcess{PID: readerPID}, nil},
		{"the table cannot be read", readerProcess(written), os.ErrPermission},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo := storedWorking(t, c.p, written)
			readerProcs(t, readerTable(next, readerAge), c.err)
			before, err := ReadDriveState(repo, next.Add(-time.Second))
			if err != nil || before.State != DriveWorking {
				t.Fatalf("before its next write: %+v, %v", before, err)
			}
			after, err := ReadDriveState(repo, next.Add(time.Second))
			if err != nil || after.State != DriveStopped || !after.Since.Equal(written) {
				t.Fatalf("past its next write: %+v, %v", after, err)
			}
		})
	}
}

// A file no drive wrote shows no line, and the error says why: none there, not a
// regular file, too large, not JSON, a newer version, no version, a control code in
// any word, a word that is no state, or no written time.
func TestADriveStateFileADriveDidNotWriteShowsNoLine(t *testing.T) {
	readerProcs(t, readerTable(driveAt(11), readerAge), nil)
	write := func(t *testing.T, body string) string {
		t.Helper()
		repo := t.TempDir()
		dir := filepath.Join(repo, filepath.FromSlash(driveStateDir))
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, filepath.FromSlash(DriveStateRel)), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return repo
	}
	const good = `"since":"2026-11-25T10:05:00Z","written":"2026-11-25T10:10:00Z","next":"2026-11-25T11:10:00Z"`
	for _, c := range []struct {
		name, body, why string
	}{
		{"not JSON", "{", "is not a drive's state"},
		{"a newer version", `{"version":2,"state":"working",` + good + `}`, "states version 2, newer than the 1 this abcd reads"},
		{"no version", `{"state":"working",` + good + `}`, "states no version"},
		{"a control code in on", `{"version":1,"state":"working","on":"iss-1\u001b[2J at land",` + good + `}`, "carries a control code in its on"},
		{"a control code in do", `{"version":1,"state":"waiting","do":"lift\u0007 the stop",` + good + `}`, "carries a control code in its do"},
		{"a line separator in on", `{"version":1,"state":"working","on":"iss-1 at land",` + good + `}`, "carries a control code in its on"},
		{"a control code in state", `{"version":1,"state":"work\u0000ing",` + good + `}`, "carries a control code in its state"},
		{"no state word", `{"version":1,"state":"sleeping",` + good + `}`, `states "sleeping", which is none of`},
		{"no written time", `{"version":1,"state":"working","since":"2026-11-25T10:05:00Z"}`, "names no time it was written"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := ReadDriveState(write(t, c.body), driveAt(11))
			if err == nil || !strings.Contains(err.Error(), c.why) || !strings.Contains(err.Error(), DriveStateRel) {
				t.Fatalf("got %v, want a refusal naming %s and saying %q", err, DriveStateRel, c.why)
			}
			if strings.ContainsAny(err.Error(), "\x1b\x07\x00 ") {
				t.Fatalf("the refusal carries a control code: %q", err)
			}
		})
	}
	t.Run("none there", func(t *testing.T) {
		if _, err := ReadDriveState(t.TempDir(), driveAt(11)); !errors.Is(err, ErrNoDriveState) {
			t.Fatalf("got %v, want ErrNoDriveState", err)
		}
	})
	t.Run("a folder in its place", func(t *testing.T) {
		repo := t.TempDir()
		if err := os.MkdirAll(filepath.Join(repo, filepath.FromSlash(DriveStateRel)), 0o700); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadDriveState(repo, driveAt(11)); err == nil || !strings.Contains(err.Error(), "is not a regular file") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("a symlink in its place", func(t *testing.T) {
		repo := write(t, "{}")
		target := filepath.Join(t.TempDir(), "elsewhere.json")
		if err := os.WriteFile(target, []byte(`{"version":1,"state":"working",`+good+`}`), 0o600); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(repo, filepath.FromSlash(DriveStateRel))
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadDriveState(repo, driveAt(11)); err == nil || !strings.Contains(err.Error(), "is not a regular file") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("too large", func(t *testing.T) {
		big := `{"version":1,"state":"working","on":"` + strings.Repeat("a", maxDriveStateBytes) + `",` + good + `}`
		if _, err := ReadDriveState(write(t, big), driveAt(11)); err == nil || !strings.Contains(err.Error(), "is larger than") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("unreadable", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("root reads a file of mode 0")
		}
		repo := write(t, `{"version":1,"state":"working",`+good+`}`)
		if err := os.Chmod(filepath.Join(repo, filepath.FromSlash(DriveStateRel)), 0); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadDriveState(repo, driveAt(11)); err == nil || !strings.Contains(err.Error(), "cannot be read") {
			t.Fatalf("got %v", err)
		}
	})
}
