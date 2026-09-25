package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/machineload"
)

// loadCaller is the invocation in every fixture: its chain is exempt.
var loadCaller = machineload.Self{PID: 50000, UID: 501}

// loadFixture points the verb at a fixture machine rather than the real one, in
// a committed repository under a temporary HOME, outside any CI marker. It
// returns the run directory.
func loadFixture(t *testing.T, snap machineload.Snapshot, readErr error) string {
	t.Helper()
	_, runDir := implementRepo(t)
	for _, k := range []string{"GITHUB_ACTIONS", "CI", "ABCD_LOAD_CHECKED"} {
		t.Setenv(k, "")
	}
	prevRead, prevSelf := loadReader, loadSelf
	loadReader = func() (machineload.Snapshot, error) { return snap, readErr }
	loadSelf = func() machineload.Self { return loadCaller }
	t.Cleanup(func() { loadReader, loadSelf = prevRead, prevSelf })
	return runDir
}

// machine is a 16-core machine at the given loads holding the caller's chain
// and procs.
func machine(load1, load5, load15 float64, procs ...machineload.Proc) machineload.Snapshot {
	chain := []machineload.Proc{
		{PID: 1, PPID: 0, PGID: 1, UID: 0, Age: 60 * time.Hour, CPU: time.Minute, Name: "launchd"},
		{PID: 49000, PPID: 1, PGID: 49000, UID: 501, Age: 50 * time.Hour, CPU: 50 * time.Hour, Name: "node"},
		{PID: 50000, PPID: 49000, PGID: 49000, UID: 501, Age: time.Second, Name: "abcd"},
	}
	return machineload.Snapshot{Load1: load1, Load5: load5, Load15: load15, HasLoad: true, Cores: 16,
		Procs: append(chain, procs...), HasProcs: true}
}

func stray(pid, pgid int, uid uint32, name string, age time.Duration, share float64) machineload.Proc {
	return machineload.Proc{PID: pid, PPID: 1, PGID: pgid, UID: uid, Age: age, CPU: time.Duration(float64(age) * share), Name: name}
}

// incidentOne is the caller's eight `yes` burners in one group, plus one stray
// in a group that also holds an editor.
func incidentOne() []machineload.Proc {
	var procs []machineload.Proc
	for pid := 41233; pid < 41241; pid++ {
		procs = append(procs, stray(pid, 41230, 501, "yes", 55*time.Hour, 0.99))
	}
	return append(procs,
		stray(41250, 41249, 501, "spin", 2*time.Hour, 1),
		stray(41251, 41249, 501, "editor", 2*time.Hour, 0.01),
	)
}

// TestImplementLoadNamesOwnStraysWithTheRemedy: the caller's own strays are
// named with name, pid, group, age and share; every kill is preceded by its
// re-check; no pattern kill is offered; exit 0.
func TestImplementLoadNamesOwnStraysWithTheRemedy(t *testing.T) {
	loadFixture(t, machine(12.3, 11, 10, incidentOne()...), nil)
	code, out, errOut := implementCLI(t, "implement", "load", "--site", "preflight")
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	for _, want := range []string{
		"LOAD WARNING (preflight): programs that are not abcd's tests are keeping this machine busy.",
		"Your programs at nearly all the CPU they can get, for over 30 min:",
		"yes     pid 41233  group 41230  running 2d 07h  CPU 99% of a core",
		"spin    pid 41250  group 41249  running 2h 00m  CPU 100% of a core",
		"pgrep -g 41230",
		"lists only 41233 41234 41235 41236 41237 41238 41239 41240, then:  kill -- -41230",
		"ps -o pid=,comm= -p 41250",
		"still shows it, then:  kill 41250   (its group holds other programs)",
		"Never by pattern (pkill -f, killall)",
		"Load 12.3 on 16 online cores.",
		"abcd carries on; stopping them is your call.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q\n%s", want, out)
		}
	}
	// Each kill follows its re-check on the same line, and no pattern kill is
	// offered outside the line that forbids it.
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "kill ") && !strings.Contains(line, "Never by pattern") {
			pre, _, _ := strings.Cut(line, "kill ")
			if !strings.Contains(pre, "pgrep -g") && !strings.Contains(pre, "ps -o pid=,comm= -p") {
				t.Errorf("a kill is not preceded by its re-check: %q", line)
			}
		}
		if (strings.Contains(line, "pkill") || strings.Contains(line, "killall")) && !strings.Contains(line, "Never by pattern") {
			t.Errorf("a pattern kill is offered: %q", line)
		}
	}
	if strings.Contains(out, "node") {
		t.Errorf("the caller's own agent host was named:\n%s", out)
	}
}

// TestImplementLoadKeepsOtherAccountsAnonymous: another account's strays appear
// as a count and a CPU total, in text and JSON alike.
func TestImplementLoadKeepsOtherAccountsAnonymous(t *testing.T) {
	const secretName = "zz-private-loop"
	var procs []machineload.Proc
	for i := 0; i < 38; i++ {
		procs = append(procs, stray(70000+i, 70000, 7777, secretName, 96*time.Hour, 0.95))
	}
	loadFixture(t, machine(20, 20, 20, procs...), nil)

	code, out, _ := implementCLI(t, "implement", "load", "--site", "preflight")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out, "Other accounts: 38 programs at nearly all the CPU they can get, for over 30 min, using about 36.1 cores.") {
		t.Fatalf("no anonymous count:\n%s", out)
	}
	code, js, _ := implementCLI(t, "implement", "load", "--site", "preflight", "--json")
	if code != 0 {
		t.Fatalf("--json exit %d", code)
	}
	for _, leak := range []string{secretName, "7777", "70000", "70037"} {
		if strings.Contains(out, leak) || strings.Contains(js, leak) {
			t.Fatalf("another account's %q leaked:\n%s\n%s", leak, out, js)
		}
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal([]byte(js), &doc); err != nil {
		t.Fatal(err)
	}
	var others map[string]any
	if err := json.Unmarshal(doc["other_strays"], &others); err != nil {
		t.Fatal(err)
	}
	var keys []string
	for k := range others {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	if !slices.Equal(keys, []string{"cores", "count"}) {
		t.Fatalf("other_strays keys = %v, want exactly count and cores", keys)
	}
}

// TestImplementLoadWarnsInTheOversubscribedBand: at a load of 40 on 16 cores
// every two-day busy loop gets about 0.4 of a core and uses all of it, so each is
// a stray (ruling H1, iss-2609231947544298): the caller's own are named and the
// other account's counted, and the warning says what a program can get at that
// load, so a stray at 40% of a core reads as the whole of its share.
func TestImplementLoadWarnsInTheOversubscribedBand(t *testing.T) {
	var procs []machineload.Proc
	for i := 0; i < 4; i++ {
		procs = append(procs, stray(41300+i, 41300, 501, "spin", 48*time.Hour, 0.4))
	}
	for i := 0; i < 36; i++ {
		procs = append(procs, stray(70000+i, 70000, 7777, "zsh", 48*time.Hour, 0.4))
	}
	loadFixture(t, machine(40, 40, 40, procs...), nil)
	code, out, _ := implementCLI(t, "implement", "load", "--site", "preflight")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{
		"LOAD WARNING (preflight): programs that are not abcd's tests are keeping this machine busy.",
		"Your programs at nearly all the CPU they can get, for over 30 min:",
		"spin    pid 41300  group 41300  running 2d 00h  CPU 40% of a core",
		"lists only 41300 41301 41302 41303, then:  kill -- -41300",
		"Other accounts: 36 programs at nearly all the CPU they can get, for over 30 min, using about 14.4 cores.",
		"Load 40.0 on 16 online cores: a program can get about 40% of a core.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q\n%s", want, out)
		}
	}

	// Above the extreme limit the share line follows the load line.
	loadFixture(t, machine(440.2, 431, 425.7, stray(41300, 41300, 501, "spin", 48*time.Hour, 0.04)), nil)
	_, out, _ = implementCLI(t, "implement", "load", "--site", "preflight")
	if want := "  Load 440.2 over 1 min (431.0 over 5, 425.7 over 15) is above the extreme limit of 64 (4 x 16 online cores).\n" +
		"  At that load a program can get about 4% of a core.\n"; !strings.Contains(out, want) {
		t.Errorf("output lacks %q\n%s", want, out)
	}
}

// TestImplementLoadIsQuietUnderOwnParallelWork: eight concurrent preflights at a
// load near 42 on 16 cores print one quiet line.
func TestImplementLoadIsQuietUnderOwnParallelWork(t *testing.T) {
	var procs []machineload.Proc
	for i := 0; i < 40; i++ {
		procs = append(procs, stray(90000+i, 90000+i%8, 501, "core.test", time.Duration(1+i%14)*time.Minute, 1))
	}
	loadFixture(t, machine(41.91, 30.2, 22.5, procs...), nil)
	code, out, _ := implementCLI(t, "implement", "load", "--site", "preflight")
	if code != 0 || strings.Contains(out, "LOAD WARNING") {
		t.Fatalf("exit %d, output:\n%s", code, out)
	}
	if want := "load check (preflight): nothing to warn about; load 41.9 on 16 online cores\n"; out != want {
		t.Fatalf("output = %q, want %q", out, want)
	}
}

// TestImplementLoadReportsLoadAndCores: an extreme load gives all three
// averages, the online core count and how the limit was derived; exit 0.
func TestImplementLoadReportsLoadAndCores(t *testing.T) {
	loadFixture(t, machine(440.2, 431, 425.7), nil)
	code, out, _ := implementCLI(t, "implement", "load", "--site", "eval-harness")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{
		"LOAD WARNING (eval-harness): this machine's load is above the extreme limit.",
		"Load 440.2 over 1 min (431.0 over 5, 425.7 over 15) is above the extreme limit of 64 (4 x 16 online cores).",
		"abcd carries on.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q\n%s", want, out)
		}
	}
}

// TestImplementLoadMalformedLimitsIsLoud: an unusable settings file is reported
// even when there is otherwise nothing to warn about.
func TestImplementLoadMalformedLimitsIsLoud(t *testing.T) {
	loadFixture(t, machine(3, 3, 3), nil)
	home, _ := os.UserHomeDir()
	if err := os.MkdirAll(filepath.Join(home, ".abcd"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".abcd", "load-limits"), []byte("# mine\nextreme-load 9\nstray-mins 12\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, _ := implementCLI(t, "implement", "load", "--site", "preflight")
	want := `LOAD CHECK SETTINGS UNUSABLE: ~/.abcd/load-limits line 3: unknown key "stray-mins" (known: stray-minutes, extreme-load); using the defaults for both limits: 30 min, load 64 (4 x 16 online cores)`
	if code != 0 || !strings.Contains(out, want) || !strings.Contains(out, "nothing to warn about") {
		t.Fatalf("exit %d, output:\n%s", code, out)
	}
}

// TestImplementLoadCannotCheck: an unsupported platform and a failed process
// read both print the loud line and exit 0; the second still reports the load.
func TestImplementLoadCannotCheck(t *testing.T) {
	loadFixture(t, machineload.Snapshot{}, machineload.ErrUnsupported)
	code, out, _ := implementCLI(t, "implement", "load", "--site", "preflight")
	if code != 0 || !strings.HasPrefix(out, "LOAD CHECK UNAVAILABLE (preflight): abcd reads load and processes on macOS and Linux only, and this is ") ||
		!strings.Contains(out, "; carrying on without checking the machine's load") {
		t.Fatalf("unsupported: exit %d\n%s", code, out)
	}

	partial := machine(3.25, 2, 1)
	partial.Procs, partial.HasProcs = nil, false
	loadReader = func() (machineload.Snapshot, error) {
		return partial, errors.New("could not read the process table (/bin/ps: exit status 1)")
	}
	code, out, _ = implementCLI(t, "implement", "load", "--site", "preflight")
	want := "LOAD CHECK UNAVAILABLE (preflight): could not read the process table (/bin/ps: exit status 1); carrying on without checking for stray programs; load 3.2 on 16 online cores\n"
	if code != 0 || out != want {
		t.Fatalf("failed process read: exit %d\n%q\nwant\n%q", code, out, want)
	}
}

// TestImplementLoadSkipsOnCI: the CI line names the variable, and nothing is read.
func TestImplementLoadSkipsOnCI(t *testing.T) {
	loadFixture(t, machine(3, 3, 3), nil)
	t.Setenv("GITHUB_ACTIONS", "true")
	loadReader = func() (machineload.Snapshot, error) {
		t.Fatal("the machine was read on a CI runner")
		return machineload.Snapshot{}, nil
	}
	code, out, _ := implementCLI(t, "implement", "load", "--site", "eval-harness")
	want := "load check (eval-harness): skipped on a CI runner (GITHUB_ACTIONS=true): a fresh runner carries no programs left from earlier work, so there is nothing to warn about\n"
	if code != 0 || out != want {
		t.Fatalf("exit %d\n%q", code, out)
	}
}

// TestLoggedEventRendersToThePrintedWarning: inside a live run the warning is
// logged, and the event read back renders byte for byte the block that was
// printed, masked names included.
func TestLoggedEventRendersToThePrintedWarning(t *testing.T) {
	procs := incidentOne()
	procs[0].Name = "clientwidget"
	loadFixture(t, machine(70.5, 60, 50, procs...), nil)
	if err := os.MkdirAll(filepath.Join(".abcd", ".work.local"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(".abcd", ".work.local", "private-names.txt"), []byte("# abcd-banlist: keyed\nclient clientwidget\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mustImplement(t, "implement", "join", "--session", "alpha", "--role", "first")

	code, out, _ := implementCLI(t, "implement", "load", "--site", "preflight")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if strings.Contains(out, "clientwidget") || !strings.Contains(out, "[private name]") {
		t.Fatalf("the private name was printed:\n%s", out)
	}
	run, err := implementRun(runPeek, "")
	if err != nil {
		t.Fatal(err)
	}
	evs, _, err := run.ReadLog()
	if err != nil {
		t.Fatal(err)
	}
	var logged []implement.Event
	for _, e := range evs {
		if e.Event == implement.EventLoad {
			logged = append(logged, e)
		}
	}
	if len(logged) != 1 || logged[0].Session != "alpha" {
		t.Fatalf("load events = %+v", logged)
	}
	back, err := implement.LoadResultFromEvent(logged[0])
	if err != nil {
		t.Fatal(err)
	}
	var block bytes.Buffer
	renderLoadWarning(&block, back)
	if !strings.Contains(out, block.String()) || block.Len() == 0 {
		t.Fatalf("the logged event renders\n%s\nwhich the printed output does not hold byte for byte:\n%s", block.String(), out)
	}
	if strings.Contains(string(logged[0].Fields["own_strays"]), "clientwidget") {
		t.Fatal("the private name reached the run log")
	}
}

// TestRunLogFailureIsLoudAndExitsZero: a log that cannot be written is one loud
// line after the warning, and the exit stays 0.
func TestRunLogFailureIsLoudAndExitsZero(t *testing.T) {
	runDir := loadFixture(t, machine(70.5, 60, 50, incidentOne()...), nil)
	mustImplement(t, "implement", "join", "--session", "alpha", "--role", "first")
	// Every day's log file a directory, so the append fails whatever the date.
	for _, d := range []time.Time{time.Now().UTC(), time.Now().UTC().Add(24 * time.Hour)} {
		p := filepath.Join(runDir, d.Format(time.DateOnly)+".jsonl")
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if err := os.Mkdir(p, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	code, out, _ := implementCLI(t, "implement", "load", "--site", "preflight")
	if code != 0 || !strings.Contains(out, "LOAD WARNING") ||
		!strings.Contains(out, "could not write the load warning to the run log: ") ||
		!strings.Contains(out, "; the warning above stands") {
		t.Fatalf("exit %d\n%s", code, out)
	}
}

// TestImplementLoadRefusesAnUnknownSite: the one non-zero exit is a malformed
// invocation.
func TestImplementLoadRefusesAnUnknownSite(t *testing.T) {
	loadFixture(t, machine(3, 3, 3), nil)
	for _, args := range [][]string{{"implement", "load"}, {"implement", "load", "--site", "ci"}} {
		if code, _, _ := implementCLI(t, args...); code == 0 {
			t.Fatalf("%v exited 0", args)
		}
	}
}
