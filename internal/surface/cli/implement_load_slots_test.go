package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/machineload"
)

// otherPreflightProcs is another account's make preflight in its unit stage: make
// at pid, its go test beneath it.
func otherPreflightProcs(pid int) []machineload.Proc {
	return []machineload.Proc{
		{PID: pid, PPID: 1, PGID: pid, UID: 502, Age: 4 * time.Minute, Name: "make"},
		{PID: pid + 1, PPID: pid, PGID: pid, UID: 502, Age: time.Minute, Name: "go"},
	}
}

// preflightArgs reads every make as `make preflight` and every go as a plain
// go test.
func preflightArgs(t *testing.T, snap machineload.Snapshot) {
	t.Helper()
	names := map[int]string{}
	for _, p := range snap.Procs {
		names[p.PID] = p.Name
	}
	prev := loadArgsReader
	loadArgsReader = func(pids []int) (map[int][]string, error) {
		out := map[int][]string{}
		for _, p := range pids {
			switch names[p] {
			case "make":
				out[p] = []string{"preflight"}
			case "go":
				out[p] = []string{"test", "./..."}
			}
		}
		return out, nil
	}
	t.Cleanup(func() { loadArgsReader = prev })
}

// TestImplementLoadWaitAdmitsASecondPreflightWithTwoSlots: with
// ABCD_LOAD_PREFLIGHTS=2 the first gate opens beside one other preflight in
// its test stages, says so, and reports the slots in --json.
func TestImplementLoadWaitAdmitsASecondPreflightWithTwoSlots(t *testing.T) {
	snap := machine(3, 3, 3, otherPreflightProcs(80000)...)
	waitFixture(t, snap)
	preflightArgs(t, snap)
	t.Setenv(implement.PreflightSlotsEnv, "2")
	code, out, errOut := implementCLI(t, "implement", "load", "--wait", "--timeout", "1m")
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	want := "fewer than 2 other preflights or gated commands are in their test stages (2 at once, set by " + implement.PreflightSlotsEnv + ")"
	if !strings.Contains(out, want) {
		t.Fatalf("output lacks %q:\n%s", want, out)
	}

	// The first wait's hold would keep the second shut for its settle.
	home, _ := os.UserHomeDir()
	if err := os.Remove(abcdhome.Path(home, implement.WaitHoldFileName)); err != nil {
		t.Fatal(err)
	}
	code, out, errOut = implementCLI(t, "implement", "load", "--wait", "--json", "--timeout", "1m")
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	var res implement.WaitResult
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("not one JSON document: %v\n%s", err, out)
	}
	if res.Gate != implement.GateTests || res.Slots != 2 || res.SlotsSource != implement.SlotsEnv {
		t.Fatalf("result = %+v, want the tests gate with 2 slots from the environment", res)
	}
}

// TestImplementLoadWaitSlotsAreNotTheRepositorys: a checkout's
// .abcd/config.json setting load.preflights changes nothing: one preflight in
// its tests still shuts the gate.
func TestImplementLoadWaitSlotsAreNotTheRepositorys(t *testing.T) {
	snap := machine(3, 3, 3, otherPreflightProcs(80000)...)
	waitFixture(t, snap)
	preflightArgs(t, snap)
	if err := os.MkdirAll(".abcd", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(".abcd", "config.json"), []byte(`{"load": {"preflights": 4}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := implementCLI(t, "implement", "load", "--wait", "--timeout", "1m")
	if code != 1 || !strings.Contains(out, "another make preflight is running (pid 80000)") {
		t.Fatalf("exit %d, want 1 still shut on the other preflight:\n%s", code, out)
	}
}

// TestImplementLoadWaitRefusesAnUnusableSlotSetting: a bad
// ABCD_LOAD_PREFLIGHTS refuses before any wait, naming it.
func TestImplementLoadWaitRefusesAnUnusableSlotSetting(t *testing.T) {
	waitFixture(t, machine(3, 3, 3))
	t.Setenv(implement.PreflightSlotsEnv, "9")
	code, _, errOut := implementCLI(t, "implement", "load", "--wait")
	if code != 1 || !strings.Contains(errOut, implement.PreflightSlotsEnv) || !strings.Contains(errOut, "nothing was waited for") {
		t.Fatalf("exit %d, want 1 naming %s:\n%s", code, implement.PreflightSlotsEnv, errOut)
	}
}

// TestImplementLoadWaitRaceGate: with two slots, --race-gate waits for
// another race run to exit, and its closing line names the race gate.
func TestImplementLoadWaitRaceGate(t *testing.T) {
	waitFixture(t, machine(3, 3, 3, liveRace), machine(3, 3, 3))
	t.Setenv(implement.PreflightSlotsEnv, "2")
	code, out, errOut := implementCLI(t, "implement", "load", "--race-gate")
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	for _, want := range []string{
		"waiting for capacity: a go test -race run is live (pid 70000)",
		"race gate open after 35s: load 3.0 is below 32 (2 x 16 online cores), and no other go test -race run or race waiter ahead is live;",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output lacks %q:\n%s", want, out)
		}
	}
}

// TestImplementLoadRaceGateOpensAtOnceWithOneSlot: at one slot, the default,
// the race gate opens at once beside a live race run and a loaded machine,
// and says why.
func TestImplementLoadRaceGateOpensAtOnceWithOneSlot(t *testing.T) {
	waitFixture(t, machine(40, 40, 40, liveRace))
	code, out, errOut := implementCLI(t, "implement", "load", "--race-gate", "--timeout", "1m")
	want := "race gate open at once: one preflight at a time may be in its test stages, so the race stage follows its tests at once\n"
	if code != 0 || out != want {
		t.Fatalf("exit %d, want 0 and exactly %q:\n%s%s", code, want, out, errOut)
	}
}

// TestImplementLoadRaceGateTakesNoWait: --race-gate waits on its own, and is
// refused beside --wait, which an older abcd reads as a first gate's waiter,
// and beside --site; --timeout without either is refused.
func TestImplementLoadRaceGateTakesNoWait(t *testing.T) {
	loadFixture(t, machine(3, 3, 3), nil)
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"implement", "load", "--wait", "--race-gate"}, "--race-gate waits on its own, so it takes no --wait"},
		{[]string{"implement", "load", "--race-gate", "--site", "preflight"}, "takes no --site"},
		{[]string{"implement", "load", "--site", "preflight", "--timeout", "1m"}, "--timeout is for --wait or --race-gate"},
	} {
		code, _, errOut := implementCLI(t, c.args...)
		if code != 2 || !strings.Contains(errOut, c.want) {
			t.Errorf("%q: exit %d, want 2 with %q:\n%s", c.args, code, c.want, errOut)
		}
	}
}

// TestImplementLoadWaitNamesAnUnreadableMachineConfig: a group-writable
// ~/.abcd.noindex/config.json no longer stops the first gate, and so make
// preflight: the gate takes one preflight at a time and says so once on
// stderr, naming the file, the cause and the fix.
func TestImplementLoadWaitNamesAnUnreadableMachineConfig(t *testing.T) {
	waitFixture(t, machine(3, 3, 3))
	home, _ := os.UserHomeDir()
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := abcdhome.Path(home, "config.json")
	if err := os.WriteFile(cfg, []byte(`{"load": {"preflights": 2}}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(cfg, 0o660); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := implementCLI(t, "implement", "load", "--wait", "--timeout", "1m")
	file := abcdhome.Display("config.json")
	note := "abcd implement load --wait: the machine configuration " + file + " cannot be read for load.preflights"
	for _, want := range []string{
		note,
		"writable by group or other",
		"so one preflight at a time applies",
		"keep " + file + " a regular file you own that nobody else can write (chmod go-w " + file + ")",
	} {
		if code != 0 || !strings.Contains(errOut, want) {
			t.Errorf("exit %d, want 0 with %q on stderr:\n%s", code, want, errOut)
		}
	}
	if strings.Count(errOut, note) != 1 || !strings.Contains(out, "load gate open") {
		t.Errorf("want the note once and the gate open:\nstdout %s\nstderr %s", out, errOut)
	}
}

// TestImplementLoadWaitGatesARaceCommandAsARaceRun: with two slots a live race
// run does not shut the first gate by itself, but `--wait -- go test -race`
// gates a race run, so it stays shut while one is live and never starts its
// command; the plain form opens.
func TestImplementLoadWaitGatesARaceCommandAsARaceRun(t *testing.T) {
	race := machineload.Proc{PID: 70000, PPID: 1, PGID: 70000, UID: 502, Age: 3 * time.Minute, Name: "go"}
	waitFixture(t, machine(3, 3, 3, race))
	t.Setenv(implement.PreflightSlotsEnv, "2")
	code, out, errOut := implementCLI(t, "implement", "load", "--wait", "--timeout", "1m", "--", "go", "test", "-race", "./no/such/package")
	if code != 1 || !strings.Contains(out, "a go test -race run is live (pid 70000)") {
		t.Fatalf("race command: exit %d, want 1 still on the race run:\n%s%s", code, out, errOut)
	}
	code, out, errOut = implementCLI(t, "implement", "load", "--wait", "--timeout", "1m")
	if code != 0 || !strings.Contains(out, "load gate open") {
		t.Fatalf("plain form: exit %d, want 0 and open:\n%s%s", code, out, errOut)
	}
}

// TestImplementLoadRaceGateTakesNoCommand: --race-gate gates the race stage
// that follows it and never runs a command, so a command after `--` is
// refused with a fixed message before the machine is read or the slots are
// resolved, at one slot as above it, and nothing is run.
func TestImplementLoadRaceGateTakesNoCommand(t *testing.T) {
	loadFixture(t, machine(3, 3, 3), nil)
	prev := loadReader
	loadReader = func() (machineload.Snapshot, error) {
		t.Fatal("the race gate read the machine for a command it refuses")
		return machineload.Snapshot{}, nil
	}
	t.Cleanup(func() { loadReader = prev })
	marker := filepath.Join(t.TempDir(), "ran")
	const want = "abcd implement load: --race-gate takes no command: it gates the race stage that follows it, so a command to run goes after `--` with --wait (nothing waited for)"
	for _, slots := range []string{"1", "2"} {
		t.Setenv(implement.PreflightSlotsEnv, slots)
		code, out, errOut := implementCLI(t, "implement", "load", "--race-gate", "--", "touch", marker)
		if code != 2 || !strings.Contains(errOut, want) || out != "" {
			t.Errorf("%s slot(s): exit %d, want 2 with %q:\nstdout %s\nstderr %s", slots, code, want, out, errOut)
		}
		if _, err := os.Stat(marker); err == nil {
			t.Fatalf("%s slot(s): the refused command ran", slots)
		}
	}
}
