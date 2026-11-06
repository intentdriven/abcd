package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"errors"
	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/core/occupancy"
	"github.com/intentdriven/abcd/internal/core/runstop"
	"github.com/intentdriven/abcd/internal/core/sessionname"
	"strconv"
)

// stopCLI runs a verb with HOME at a fixture home and a terminal, or none.
func stopCLI(t *testing.T, home string, terminal bool, args ...string) (string, string, int) {
	t.Helper()
	t.Setenv("HOME", home)
	old := readCaller
	readCaller = func() caller { return caller{terminal: terminal} }
	defer func() { readCaller = old }()
	var out, errb bytes.Buffer
	code := Run(args, &out, &errb)
	return out.String(), errb.String(), code
}

// TestImplementStopIsThePersonsAct: a caller with no terminal is refused and
// changes nothing; a person's stop is written and lifted, and said to reach only
// this account where no folder was chosen.
func TestImplementStopIsThePersonsAct(t *testing.T) {
	home := t.TempDir()
	p := abcdhome.Path(home, runstop.HomeStopFile)
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
	if out, stderr, code := stopCLI(t, home, true, "implement", "stop"); code != 0 || !strings.Contains(out, "each account that chose this shared folder") {
		t.Fatalf("the stop: %d %q %q", code, out, stderr)
	}
	if _, err := os.Lstat(filepath.Join(folder, runstop.StopDirName+"-"+strconv.Itoa(os.Getuid()), runstop.StopFileName)); err != nil {
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
	if err := os.Symlink("/nonexistent", filepath.Join(folder, runstop.StopDirName+"-"+strconv.Itoa(os.Getuid()), runstop.StopFileName)); err != nil {
		t.Fatal(err)
	}
	if _, stderr, code := stopCLI(t, home, true, "implement", "stop"); code != 2 || !strings.Contains(stderr, "no stop was set") {
		t.Fatalf("over a symlink: %d %q", code, stderr)
	}
}

// TestAHostSessionIsNoPerson: a process under a host session is no person.
func TestAHostSessionIsNoPerson(t *testing.T) {
	t.Setenv(hostSessionIDEnv, "a-session")
	if hostSessionReading().state != hostSessionFound {
		t.Fatal("a process with a host session's id reads as a person")
	}
}

// lookedCLI runs a verb at a terminal with each look for a host session answered in turn,
// the last answer standing; it returns how many looks were taken.
func lookedCLI(t *testing.T, home string, looks []hostLook, args ...string) (string, int, int) {
	t.Helper()
	t.Setenv("HOME", home)
	oldTerminal, oldReading := stdinIsTerminal, hostSessionReading
	taken := 0
	stdinIsTerminal = func() bool { return true }
	hostSessionReading = func() hostLook {
		l := looks[min(taken, len(looks)-1)]
		taken++
		return l
	}
	t.Cleanup(func() { stdinIsTerminal, hostSessionReading = oldTerminal, oldReading })
	var out, errb bytes.Buffer
	code := Run(args, &out, &errb)
	return errb.String(), code, taken
}

// cannotTellCLI runs a verb at a terminal where the look for a host session cannot be
// finished.
func cannotTellCLI(t *testing.T, home string, args ...string) (string, int) {
	t.Helper()
	stderr, code, _ := lookedCLI(t, home, []hostLook{{state: hostSessionUnknown, why: "a reason: run it again"}}, args...)
	return stderr, code
}

// TestAnActStandsOnOneReading: a loaded machine can answer one look and not the next, so
// an act is judged by a single look, and a later one cannot let it through.
func TestAnActStandsOnOneReading(t *testing.T) {
	unknownThenFound := []hostLook{{state: hostSessionUnknown, why: "a reason: run it again"}, {state: hostSessionFound}}
	home := t.TempDir()
	if _, stderr, code := stopCLI(t, home, true, "implement", "stop"); code != 0 {
		t.Fatalf("the stop: %d %q", code, stderr)
	}
	stderr, code, taken := lookedCLI(t, home, unknownThenFound, "implement", "stop", "--lift")
	if code != 2 || taken != 1 || !strings.Contains(stderr, "could not be told") {
		t.Fatalf("the lift: %d, %d looks, %q", code, taken, stderr)
	}
	if r := runstop.ReadPinned(home, nil); r.Stop == nil {
		t.Fatalf("the refused lift took the stop away: %+v", r)
	}
	stderr, code, taken = lookedCLI(t, home, unknownThenFound, "ahoy", "shared-folder", t.TempDir())
	if code != 2 || taken != 1 || !strings.Contains(stderr, "could not be told") {
		t.Fatalf("the choice: %d, %d looks, %q", code, taken, stderr)
	}
	stderr, code, taken = lookedCLI(t, t.TempDir(), []hostLook{{state: noHostSession}}, "implement", "stop")
	if code != 0 || taken != 1 {
		t.Fatalf("the stop: %d, %d looks, %q", code, taken, stderr)
	}
}

// TestAMarkersFolderThatCannotBeUsedIsNamed: running it again clears nothing there, so the
// refusal says which folder to fix.
func TestAMarkersFolderThatCannotBeUsedIsNamed(t *testing.T) {
	home := t.TempDir()
	if _, stderr, code := stopCLI(t, home, true, "implement", "stop"); code != 0 {
		t.Fatalf("the stop: %d %q", code, stderr)
	}
	p := abcdhome.Path(home, "occupancy")
	if err := os.WriteFile(p, []byte("not a folder\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tab := sessionname.Table{ReadAt: time.Now(), Procs: []machineload.Proc{{PID: 500, PPID: 1, Name: "zsh", Age: time.Hour}}}
	look := hostSessionAbove(home, tab, 500)
	stderr, code, _ := lookedCLI(t, home, []hostLook{look}, "implement", "stop", "--lift")
	if code != 2 || !strings.Contains(stderr, abcdhome.Display("occupancy")) || !strings.Contains(stderr, "fix that folder") {
		t.Fatalf("the lift: %d %q", code, stderr)
	}
}

// TestAWalkCutShortIsReadOnce: a parent gone while the table was read leaves a gap that
// moves, which tells nothing; a gap in one place is a process hidden from this account,
// which must not hold a person's lift for ever. A real chain lifts.
func TestAWalkCutShortIsReadOnce(t *testing.T) {
	proc := func(pid, ppid int, name string) machineload.Proc {
		return machineload.Proc{PID: pid, PPID: ppid, Name: name, Age: time.Hour}
	}
	table := func(procs ...machineload.Proc) sessionname.Table {
		return sessionname.Table{ReadAt: time.Now(), Procs: procs}
	}
	host := proc(500, 1, "host")
	for _, c := range []struct {
		name   string
		from   int
		tables []sessionname.Table
		want   hostReading
		reads  int
	}{
		{"through a root-owned login", 700, []sessionname.Table{table(proc(700, 600, "zsh"), proc(600, 500, "login"), proc(500, 1, "terminal"))}, noHostSession, 1},
		{"through a tmux server", 700, []sessionname.Table{table(proc(700, 600, "zsh"), proc(600, 1, "tmux"))}, noHostSession, 1},
		{"a container's first process", 1, []sessionname.Table{table(proc(1, 0, "bash"))}, noHostSession, 1},
		{"a parent outside the namespace", 700, []sessionname.Table{table(proc(700, 0, "bash"))}, noHostSession, 1},
		{"a hidden ancestor", 700, []sessionname.Table{table(proc(700, 600, "zsh")), table(proc(700, 600, "zsh"))}, noHostSession, 2},
		{"a parent that ended under the read", 700, []sessionname.Table{table(proc(700, 600, "zsh"), host), table(proc(700, 1, "zsh"), host)}, hostSessionUnknown, 2},
		{"a gap that moved", 700, []sessionname.Table{table(proc(700, 600, "zsh")), table(proc(700, 650, "zsh"))}, hostSessionUnknown, 2},
		{"a from the table lacks", 700, []sessionname.Table{table(host)}, hostSessionUnknown, 1},
		{"a chain with no end", 700, []sessionname.Table{table(proc(700, 600, "zsh"), proc(600, 700, "zsh"))}, hostSessionUnknown, 1},
	} {
		reads := 0
		look := hostSessionLook(t.TempDir(), func() (sessionname.Table, error) {
			tab := c.tables[min(reads, len(c.tables)-1)]
			reads++
			return tab, nil
		}, c.from)
		if look.state != c.want || reads != c.reads {
			t.Errorf("%s: %v after %d reads, want %v after %d (%q)", c.name, look.state, reads, c.want, c.reads, look.why)
		}
		if look.state == hostSessionUnknown && !strings.Contains(look.why, "run it again") {
			t.Errorf("%s: the reason %q names no way on", c.name, look.why)
		}
	}
	reads := 0
	if look := hostSessionLook(t.TempDir(), func() (sessionname.Table, error) {
		reads++
		return sessionname.Table{}, errors.New("no table")
	}, 700); look.state != hostSessionUnknown || reads != 1 {
		t.Errorf("no table: %v after %d reads", look.state, reads)
	}
}

// TestAStopIsSetWhereASessionCannotBeTold: a stop has to work on a machine too loaded to
// read its processes, and costs only pace.
func TestAStopIsSetWhereASessionCannotBeTold(t *testing.T) {
	home := t.TempDir()
	if stderr, code := cannotTellCLI(t, home, "implement", "stop"); code != 0 {
		t.Fatalf("the stop: %d %q", code, stderr)
	}
	if r := runstop.ReadPinned(home, nil); r.Stop == nil {
		t.Fatalf("no stop: %+v", r)
	}
}

// TestALiftIsRefusedWhereASessionCannotBeTold: what opens the runs again is no one's act
// until a person is known to be taking it, and the refusal says to try again.
func TestALiftIsRefusedWhereASessionCannotBeTold(t *testing.T) {
	home := t.TempDir()
	if _, stderr, code := stopCLI(t, home, true, "implement", "stop"); code != 0 {
		t.Fatalf("the stop: %d %q", code, stderr)
	}
	stderr, code := cannotTellCLI(t, home, "implement", "stop", "--lift")
	if code != 2 || !strings.Contains(stderr, "could not be told") || !strings.Contains(stderr, "run it again") {
		t.Fatalf("the lift: %d %q", code, stderr)
	}
	if r := runstop.ReadPinned(home, nil); r.Stop == nil {
		t.Fatalf("the refused lift took the stop away: %+v", r)
	}
}

// TestAFolderIsNotChosenWhereASessionCannotBeTold: choosing says where the stop is read
// from, so it waits as a lift does; reading the choice does not.
func TestAFolderIsNotChosenWhereASessionCannotBeTold(t *testing.T) {
	home := t.TempDir()
	stderr, code := cannotTellCLI(t, home, "ahoy", "shared-folder", t.TempDir())
	if code != 2 || !strings.Contains(stderr, "could not be told") || !strings.Contains(stderr, "run it again") {
		t.Fatalf("the choice: %d %q", code, stderr)
	}
	if _, chosen, _ := runstop.ReadSetting(home); chosen {
		t.Fatal("the refused choice wrote a folder")
	}
	if stderr, code := cannotTellCLI(t, home, "ahoy", "shared-folder"); code != 0 {
		t.Fatalf("the bare form: %d %q", code, stderr)
	}
}

// TestAHostSessionLookThatCannotFinishSaysSo: a process the table does not hold, and
// markers that cannot be read, tell nothing; markers no one has written yet say none.
func TestAHostSessionLookThatCannotFinishSaysSo(t *testing.T) {
	tab := sessionname.Table{ReadAt: time.Now(), Procs: []machineload.Proc{{PID: 500, PPID: 1, Name: "zsh", Age: time.Hour}}}
	if got := hostSessionAbove(t.TempDir(), tab, 500); got.state != noHostSession {
		t.Fatalf("no markers yet: %v", got)
	}
	if got := hostSessionAbove(t.TempDir(), tab, 999); got.state != hostSessionUnknown {
		t.Fatalf("a process the table lacks: %v", got)
	}
	home := t.TempDir()
	p := abcdhome.Path(home, "occupancy")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("not a folder\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := hostSessionAbove(home, tab, 500); got.state != hostSessionUnknown {
		t.Fatalf("markers that cannot be read: %v", got)
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
	if hostSessionAbove(home, tab, 700).state != hostSessionFound {
		t.Fatal("a process with a host session above it reads as a person")
	}
	later := tab
	later.ReadAt = now.Add(1100 * time.Millisecond)
	later.Procs = later.Procs[:1]
	if hostSessionAbove(home, later, 500).state != hostSessionFound {
		t.Fatal("a later reading of the host's start reads as no session")
	}
	if hostSessionAbove(t.TempDir(), tab, 700).state != noHostSession {
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
		{"Write", abcdhome.Path(home, "stop.json")},
		{"Edit", abcdhome.Path(home, "shared-folder.json")},
		{"MultiEdit", abcdhome.Path(home, "shared-folder.json")},
		{"Write", "/Users/Shared/abcd-stop/abcd-stop.json"},
		{"Write", abcdhome.Path(home, "STOP.JSON")},
		{"Edit", filepath.Join(home, strings.ToUpper(abcdhome.Rel()), "SHARED-FOLDER.JSON")},
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
	dir := filepath.Join(folder, runstop.StopDirName+"-"+strconv.Itoa(os.Getuid()+1))
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, runstop.StopFileName), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := runstop.WellKnownFolder
	runstop.WellKnownFolder = func() string { return folder }
	t.Cleanup(func() { runstop.WellKnownFolder = old })
	out, stderr, code := stopCLI(t, t.TempDir(), true, "implement", "stop", "--lift")
	if code != 0 || strings.Contains(out, "no stop was set") || !strings.Contains(out, "cannot reach") {
		t.Fatalf("exit %d, %q, %q", code, out, stderr)
	}
}

// TestAPersonAtTheTerminalIsNoHostSession: both halves count: a terminal under a host
// session is no person, and neither is a caller with no terminal.
func TestAPersonAtTheTerminalIsNoHostSession(t *testing.T) {
	old := stdinIsTerminal
	t.Cleanup(func() { stdinIsTerminal = old })
	stdinIsTerminal = func() bool { return true }
	t.Setenv(hostSessionIDEnv, "a-session")
	if readCaller().person() {
		t.Fatal("a terminal under a host session reads as a person")
	}
	stdinIsTerminal = func() bool { return false }
	t.Setenv(hostSessionIDEnv, "")
	if readCaller().person() {
		t.Fatal("no terminal reads as a person")
	}
}

// TestTheStopSaysItsReach: its answer names the accounts it reaches and that it cannot
// tell which of them chose the folder.
func TestTheStopSaysItsReach(t *testing.T) {
	home := t.TempDir()
	folder := filepath.Join(t.TempDir(), "shared")
	if err := os.Mkdir(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	old := runstop.AccountHome
	runstop.AccountHome = func() (string, error) { return home, nil }
	t.Cleanup(func() { runstop.AccountHome = old })
	if _, err := runstop.WriteSetting(home, folder, ""); err != nil {
		t.Fatal(err)
	}
	out, stderr, code := stopCLI(t, home, true, "implement", "stop")
	if code != 0 || !strings.Contains(out, "cannot tell") || strings.Contains(out, "every run on this machine") {
		t.Fatalf("exit %d, %q, %q", code, out, stderr)
	}
}

// TestARefusalNamesWhatItWrote: a refusal after a call acted does not say nothing was
// written.
func TestARefusalNamesWhatItWrote(t *testing.T) {
	err := loopFail(&bytes.Buffer{}, false, "abcd x", &loop.Refusal{Stage: "reservation", Reason: "r", Remedy: "m", Wrote: "claimed the driver"})
	var ee *exitError
	if !errors.As(err, &ee) || strings.Contains(ee.Msg, "nothing written") || !strings.Contains(ee.Msg, "claimed the driver") {
		t.Fatalf("%v", err)
	}
}

// TestAStopTypedAgainSaysItStood: it changes nothing, and says since when.
func TestAStopTypedAgainSaysItStood(t *testing.T) {
	home := t.TempDir()
	first, stderr, code := stopCLI(t, home, true, "--json", "implement", "stop")
	if code != 0 {
		t.Fatalf("the stop: %d %q", code, stderr)
	}
	var set stopOutput
	if err := json.Unmarshal([]byte(first), &set); err != nil || set.Stood || set.SetAt == nil {
		t.Fatalf("the stop: %q %v", first, err)
	}
	time.Sleep(20 * time.Millisecond)
	again, stderr, code := stopCLI(t, home, true, "--json", "implement", "stop")
	if code != 0 {
		t.Fatalf("again: %d %q", code, stderr)
	}
	var stood stopOutput
	if err := json.Unmarshal([]byte(again), &stood); err != nil || !stood.Stood || stood.SetAt == nil || !stood.SetAt.Equal(*set.SetAt) {
		t.Fatalf("again: %q %v", again, err)
	}
	said, stderr, code := stopCLI(t, home, true, "implement", "stop")
	if code != 0 || !strings.Contains(said, "has stood since "+set.SetAt.UTC().Format(time.RFC3339)) {
		t.Fatalf("the words: %d %q %q", code, said, stderr)
	}
}

// TestTheStopsFlagsNameNoValue: a flag's help takes a backquoted word as the name of its
// value, which is no value a boolean or a run id's flag has.
func TestTheStopsFlagsNameNoValue(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		wrong string
	}{
		{[]string{"implement", "stop", "--help"}, "--lift abcd"},
		{[]string{"implement", "resume", "--help"}, "--take abcd"},
		{[]string{"ahoy", "shared-folder", "--help"}, "--person abcd"},
	} {
		var out, errb bytes.Buffer
		if code := Run(tc.args, &out, &errb); code != 0 {
			t.Fatalf("%v: %d %q", tc.args, code, errb.String())
		}
		if strings.Contains(out.String(), tc.wrong) {
			t.Fatalf("%v names a value:\n%s", tc.args, out.String())
		}
	}
}
