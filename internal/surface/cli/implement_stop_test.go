package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/core/occupancy"
	"github.com/intentdriven/abcd/internal/core/runstop"
	"github.com/intentdriven/abcd/internal/core/sessionname"
)

// stopCLI runs a verb with HOME at a fixture home and a terminal, or none.
func stopCLI(t *testing.T, home string, terminal bool, args ...string) (string, string, int) {
	t.Helper()
	t.Setenv("HOME", home)
	old := personAtTerminal
	personAtTerminal = func() bool { return terminal }
	t.Cleanup(func() { personAtTerminal = old })
	var out, errb bytes.Buffer
	code := Run(args, &out, &errb)
	return out.String(), errb.String(), code
}

// TestImplementStopIsThePersonsAct: a caller with no terminal is refused and
// changes nothing; a person's stop is written and lifted, and said to reach only
// this account where no folder was chosen.
func TestImplementStopIsThePersonsAct(t *testing.T) {
	home := t.TempDir()
	p := filepath.Join(home, ".abcd.noindex", runstop.HomeStopFile)
	_, stderr, code := stopCLI(t, home, false, "implement", "stop")
	if code != 2 || !strings.Contains(stderr, "person's own act") {
		t.Fatalf("no terminal: %d %q", code, stderr)
	}
	if _, err := os.Lstat(p); !os.IsNotExist(err) {
		t.Fatalf("a refused stop wrote %v", err)
	}

	out, stderr, code := stopCLI(t, home, true, "implement", "stop")
	if code != 0 {
		t.Fatalf("the person's stop: %d %q", code, stderr)
	}
	if !strings.Contains(out, "this account's runs only") {
		t.Fatalf("the reach: %q", out)
	}
	if r := runstop.ReadPinned(home, nil); r.Stop == nil {
		t.Fatalf("no stop: %+v", r)
	}

	_, stderr, code = stopCLI(t, home, false, "implement", "stop", "--lift")
	if code != 2 || runstop.ReadPinned(home, nil).Stop == nil {
		t.Fatalf("a lift with no terminal: %d %q", code, stderr)
	}
	out, stderr, code = stopCLI(t, home, true, "--json", "implement", "stop", "--lift")
	if code != 0 {
		t.Fatalf("the person's lift: %d %q", code, stderr)
	}
	var got stopOutput
	if err := json.Unmarshal([]byte(out), &got); err != nil || !got.Lifted {
		t.Fatalf("the lift's JSON: %q %v", out, err)
	}
	if r := runstop.ReadPinned(home, nil); r.Stop != nil {
		t.Fatalf("the stop stands: %+v", r)
	}
}

// TestSharedFolderSaysWhatItIsAndChooses: bare, it says what the folder is for,
// that every account reads it and that it is meant for an unshared machine, and
// writes nothing; choosing needs a terminal; a stop then lands in the folder.
func TestSharedFolderSaysWhatItIsAndChooses(t *testing.T) {
	home := t.TempDir()
	folder := t.TempDir()
	out, _, code := stopCLI(t, home, false, "ahoy", "shared-folder")
	if code != 0 {
		t.Fatalf("bare: %d", code)
	}
	for _, want := range []string{"Every account on this machine can read", "no other person shares", "Declining leaves every other part", "none chosen"} {
		if !strings.Contains(out, want) {
			t.Errorf("the walkthrough lacks %q:\n%s", want, out)
		}
	}
	if _, chosen, _ := runstop.ReadSetting(home); chosen {
		t.Fatal("the bare form chose a folder")
	}
	if _, stderr, code := stopCLI(t, home, false, "ahoy", "shared-folder", folder); code != 2 || !strings.Contains(stderr, "person's own act") {
		t.Fatalf("no terminal: %d %q", code, stderr)
	}
	if out, stderr, code := stopCLI(t, home, true, "ahoy", "shared-folder", folder); code != 0 || !strings.Contains(out, "chosen:") {
		t.Fatalf("choosing: %d %q %q", code, out, stderr)
	}
	if out, stderr, code := stopCLI(t, home, true, "implement", "stop"); code != 0 || !strings.Contains(out, "every run on this machine") {
		t.Fatalf("the stop: %d %q %q", code, out, stderr)
	}
	if _, err := os.Lstat(filepath.Join(folder, runstop.StopDirName, runstop.StopFileName)); err != nil {
		t.Fatal(err)
	}
}

// TestImplementStopSaysWhenItDidNotTakeHold: a file no run would honour at the
// stop's path is a refusal, never a stop the person believes they set.
func TestImplementStopSaysWhenItDidNotTakeHold(t *testing.T) {
	home := t.TempDir()
	folder := t.TempDir()
	if _, _, code := stopCLI(t, home, true, "ahoy", "shared-folder", folder); code != 0 {
		t.Fatal("choosing failed")
	}
	if err := os.Symlink("/nonexistent", filepath.Join(folder, runstop.StopDirName, runstop.StopFileName)); err != nil {
		t.Fatal(err)
	}
	if _, stderr, code := stopCLI(t, home, true, "implement", "stop"); code != 2 || !strings.Contains(stderr, "no stop was set") {
		t.Fatalf("over a symlink: %d %q", code, stderr)
	}
}

// TestAHostSessionIsNoPerson: a process under a host session is no person.
func TestAHostSessionIsNoPerson(t *testing.T) {
	t.Setenv(hostSessionIDEnv, "a-session")
	if !underHostSession() {
		t.Fatal("a process with a host session's id reads as a person")
	}
}

// TestAnyAncestorsHostSessionCounts: every process above this one is asked about a
// host session, and each reading of a host's start may differ by a second.
func TestAnyAncestorsHostSessionCounts(t *testing.T) {
	home := t.TempDir()
	now := time.Now()
	tab := sessionname.Table{ReadAt: now, Procs: []machineload.Proc{
		{PID: 500, PPID: 1, Name: "host", Age: time.Hour},
		{PID: 600, PPID: 500, Name: "script", Age: time.Minute},
		{PID: 700, PPID: 600, Name: "zsh", Age: time.Second},
	}}
	host := sessionname.Host{PID: 500, StartUnix: now.Add(-time.Hour).Unix()}
	if err := occupancy.Note(home, "s-one", t.TempDir(), host, tab); err != nil {
		t.Fatal(err)
	}
	if !hostSessionAbove(home, tab, 700) {
		t.Fatal("a process with a host session above it reads as a person")
	}
	later := tab
	later.ReadAt = now.Add(1100 * time.Millisecond)
	later.Procs = later.Procs[:1]
	if !hostSessionAbove(home, later, 500) {
		t.Fatal("a later reading of the host's start reads as no session")
	}
	if hostSessionAbove(t.TempDir(), tab, 700) {
		t.Fatal("a terminal with no session above it reads as a host session")
	}
}

// TestGuardHookDeniesTheFileToolsTheStop: the host's own file tools may not write
// the stop or the setting that says where it is read, which would lift it or put
// the run out of its reach.
func TestGuardHookDeniesTheFileToolsTheStop(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cwd := t.TempDir()
	for _, tc := range []struct{ tool, path string }{
		{"Write", filepath.Join(home, ".abcd.noindex", "stop.json")},
		{"Edit", filepath.Join(home, ".abcd.noindex", "shared-folder.json")},
		{"MultiEdit", filepath.Join(home, ".abcd.noindex", "shared-folder.json")},
		{"Write", "/Users/Shared/abcd-stop/abcd-stop.json"},
		{"Write", filepath.Join(home, ".abcd.noindex", "STOP.JSON")},
		{"Edit", filepath.Join(home, ".ABCD.NOINDEX", "SHARED-FOLDER.JSON")},
	} {
		stdout, stderr, code := runGuard(fileToolPayload(t, tc.tool, cwd, tc.path), "guard", "hook")
		if reason := mustDeny(t, stdout, stderr, code); !strings.Contains(reason, "person") {
			t.Errorf("%s %s: the deny says %q", tc.tool, tc.path, reason)
		}
	}
	stdout, stderr, code := runGuard(fileToolPayload(t, "Write", cwd, filepath.Join(cwd, "stop.json")), "guard", "hook")
	if code != 0 || stdout != "" || stderr != "" {
		t.Errorf("a stop.json of the project's own: exit %d, %q, %q", code, stdout, stderr)
	}
}

// TestTheStopVerbsActOnTheAccountsHome: a stop set or lifted under another HOME
// lands in this account's home, where every run reads it.
func TestTheStopVerbsActOnTheAccountsHome(t *testing.T) {
	account := t.TempDir()
	old := runstop.AccountHome
	runstop.AccountHome = func() (string, error) { return account, nil }
	t.Cleanup(func() { runstop.AccountHome = old })
	elsewhere := t.TempDir()
	if _, stderr, code := stopCLI(t, elsewhere, true, "implement", "stop"); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if got := runstop.ReadPinned(account, nil); got.Stop == nil {
		t.Fatal("the stop did not land in the account's home")
	}
	if got := runstop.ReadPinned(elsewhere, nil); got.Stop != nil {
		t.Fatal("the stop followed the variable")
	}
	if _, stderr, code := stopCLI(t, elsewhere, true, "implement", "stop", "--lift"); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if got := runstop.ReadPinned(account, nil); got.Stop != nil {
		t.Fatal("the lift did not reach the account's home")
	}
}

// TestLiftNamesTheStopItCannotReach: "no stop was set" is never said over a stop that
// stands where another account left it.
func TestLiftNamesTheStopItCannotReach(t *testing.T) {
	folder := t.TempDir()
	dir := filepath.Join(folder, runstop.StopDirName)
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, runstop.StopFileName), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := runstop.WellKnownFolder
	runstop.WellKnownFolder = func() string { return folder }
	t.Cleanup(func() { runstop.WellKnownFolder = old })
	t.Cleanup(runstop.SwapStopDirOwnerForTest(func(string) (int, bool) { return os.Getuid() + 1, true }))
	out, stderr, code := stopCLI(t, t.TempDir(), true, "implement", "stop", "--lift")
	if code != 0 || strings.Contains(out, "no stop was set") || !strings.Contains(out, "cannot reach") {
		t.Fatalf("exit %d, %q, %q", code, out, stderr)
	}
}
