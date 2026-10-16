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

// TestImplementLoadWaitRaceGate: --wait --race waits for another race run to
// exit, and its closing line names the race gate.
func TestImplementLoadWaitRaceGate(t *testing.T) {
	waitFixture(t, machine(3, 3, 3, liveRace), machine(3, 3, 3))
	code, out, errOut := implementCLI(t, "implement", "load", "--wait", "--race")
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

// TestImplementLoadRaceNeedsWait: --race names a gate, so it is refused
// without --wait.
func TestImplementLoadRaceNeedsWait(t *testing.T) {
	loadFixture(t, machine(3, 3, 3), nil)
	code, _, errOut := implementCLI(t, "implement", "load", "--site", "preflight", "--race")
	if code != 2 || !strings.Contains(errOut, "--race is for --wait") {
		t.Fatalf("exit %d, want 2 refusing --race without --wait:\n%s", code, errOut)
	}
}
