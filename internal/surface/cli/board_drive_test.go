package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/sessionname"
)

// The drive's state as the status surfaces read it (spc-2611141617199772 step 5): the
// stored value is read through the loop, which judges the recorded process against
// this machine's own process table, and a file no drive wrote shows no line and says
// why on stderr.

// storeDriveState writes s as the drive stores it, at the drive's own path under root.
func storeDriveState(t *testing.T, root string, s loop.DriveState) {
	t.Helper()
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	storeDriveStateBody(t, root, string(data))
}

// storeDriveStateBody writes body at the drive's state path under root.
func storeDriveStateBody(t *testing.T, root, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(loop.DriveStateRel))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// thisProcess is this test's own process as a drive records itself: its id and its
// start, read from the machine's process table.
func thisProcess(t *testing.T) loop.DriveProcess {
	t.Helper()
	tab, err := sessionname.Live().ReadTable()
	if err != nil {
		t.Skipf("the process table cannot be read here: %v", err)
	}
	for _, p := range tab.Procs {
		if p.PID == os.Getpid() {
			return loop.DriveProcess{PID: p.PID, StartUnix: tab.ReadAt.Add(-p.Age).Unix(), SlackMS: tab.Took.Milliseconds()}
		}
	}
	t.Skip("this process is not in the process table")
	return loop.DriveProcess{}
}

// goneProcess is a process that ran and was reaped, as a drive that ended without a
// last line leaves its record.
func goneProcess(t *testing.T) loop.DriveProcess {
	t.Helper()
	c := exec.Command("true")
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	return loop.DriveProcess{PID: c.Process.Pid, StartUnix: time.Now().Add(-time.Second).Unix()}
}

// workingAt is a working drive, written at written by process p.
func workingAt(p loop.DriveProcess, written time.Time) loop.DriveState {
	return loop.DriveState{Version: loop.DriveStateVersion, State: loop.DriveWorking, Since: written.Add(-3 * time.Minute),
		On: "iss-9 at implement", Process: p, Written: written, Next: written.Add(10 * time.Minute)}
}

// A recorded process that runs shows the line as the drive stored it; one that is
// gone, or whose id is held by a process that started at another time, reads stopped,
// not heard from since the value was written, and never working, though the drive
// said it would write again later.
func TestTheBoardReadsAGoneDriveAsStopped(t *testing.T) {
	root := managedCheckout(t)
	written := time.Now().UTC().Truncate(time.Second).Add(-2 * time.Minute)
	self := thisProcess(t)
	reused := self
	reused.StartUnix -= 3600
	for _, c := range []struct {
		name string
		p    loop.DriveProcess
		want string
	}{
		{"its process runs", self, "working 5m · on iss-9 at implement · you: nothing to do"},
		{"its process is gone", goneProcess(t), "stopped 2m · on the drive, not heard from since " + written.Format(time.RFC3339) + " · you: needs a look"},
		{"its id is held by a process started an hour after it", reused, "stopped 2m · on the drive, not heard from since " + written.Format(time.RFC3339) + " · you: needs a look"},
	} {
		t.Run(c.name, func(t *testing.T) {
			storeDriveState(t, root, workingAt(c.p, written))
			var errb strings.Builder
			d := boardDrive(root, &errb)
			if d == nil || errb.Len() != 0 {
				t.Fatalf("no drive read: %+v, stderr %q", d, errb.String())
			}
			if got := d.line(); got != c.want {
				t.Fatalf("the line: %q, want %q", got, c.want)
			}
		})
	}
}

// A file no drive wrote shows no line, and stderr says why, masked: none there, a
// newer version, a control code, and a folder in its place.
func TestNoDriveStateLineSaysWhyOnStderr(t *testing.T) {
	root := managedCheckout(t)
	path := filepath.Join(root, filepath.FromSlash(loop.DriveStateRel))
	const good = `"since":"2026-11-25T10:05:00Z","written":"2026-11-25T10:10:00Z","next":"2026-11-25T11:10:00Z"`
	for _, c := range []struct {
		name string
		lay  func(t *testing.T)
		why  string
	}{
		{"none there", func(t *testing.T) {}, "no drive has written its state in this checkout (" + loop.DriveStateRel + ")"},
		{"a newer version", func(t *testing.T) {
			storeDriveStateBody(t, root, `{"version":99,"state":"working",`+good+`}`)
		}, "states version 99, newer than the 1 this abcd reads"},
		{"a control code", func(t *testing.T) {
			storeDriveStateBody(t, root, `{"version":1,"state":"working","on":"iss-1\u001b]0;owned\u0007",`+good+`}`)
		}, "carries a control code in its on"},
		{"a folder in its place", func(t *testing.T) {
			if err := os.MkdirAll(path, 0o700); err != nil {
				t.Fatal(err)
			}
		}, "is not a regular file"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if err := os.RemoveAll(path); err != nil {
				t.Fatal(err)
			}
			c.lay(t)
			var errb strings.Builder
			if d := boardDrive(root, &errb); d != nil {
				t.Fatalf("a line was shown: %+v", d)
			}
			got := errb.String()
			if !strings.HasPrefix(got, "abcd: no drive state line — ") || !strings.Contains(got, c.why) || strings.Count(got, "\n") != 1 {
				t.Fatalf("stderr %q, want one line saying %q", got, c.why)
			}
			if strings.ContainsAny(got, "\x1b\x07") {
				t.Fatalf("stderr carries a control code: %q", got)
			}
		})
	}
}

// Outside a checkout abcd manages the board reads no drive and says nothing, as its
// status block does.
func TestTheBoardReadsNoDriveOutsideAManagedCheckout(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	repo := t.TempDir()
	gitInitAt(t, repo)
	storeDriveState(t, repo, workingAt(loop.DriveProcess{PID: os.Getpid()}, time.Now()))
	var errb strings.Builder
	if d := boardDrive(repo, &errb); d != nil || errb.Len() != 0 {
		t.Fatalf("an unmanaged checkout read %+v, stderr %q", d, errb.String())
	}
}
