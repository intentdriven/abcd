package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/machineload"
)

// No test in this package reads the real machine's memory pressure: the one
// init among its tests, in implement_load_disk_test.go, stubs
// implement.ReadMemory with macOS at its normal level.

// withMemory makes every memory read return readings in turn, the last
// repeating, for the test's length.
func withMemory(t *testing.T, readings ...machineload.Memory) {
	t.Helper()
	prev := implement.ReadMemory
	i := 0
	implement.ReadMemory = func() (machineload.Memory, error) {
		m := readings[min(i, len(readings)-1)]
		i++
		return m, nil
	}
	t.Cleanup(func() { implement.ReadMemory = prev })
}

// TestImplementLoadWaitNamesMemoryPressure: a waiter held by memory pressure
// names the level and the floor in its waiting line, and opens once it falls.
func TestImplementLoadWaitNamesMemoryPressure(t *testing.T) {
	waitFixture(t, machine(3, 3, 3))
	warn := machineload.Memory{Platform: machineload.MemoryDarwin, Level: machineload.MemoryWarn, Raw: "2"}
	normal := machineload.Memory{Platform: machineload.MemoryDarwin, Level: machineload.MemoryNormal, Raw: "1"}
	withMemory(t, warn, normal)
	code, out, errOut := implementCLI(t, "implement", "load", "--wait", "--timeout", "5m")
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	if !strings.Contains(out, "waiting for capacity: memory pressure warn (floor warn)") || !strings.Contains(out, "load gate open after") {
		t.Fatalf("stdout:\n%s", out)
	}
}

// TestImplementLoadRaceGateAtOneSlotSaysItWaitedOnMemory: at one slot the
// race gate is held by memory pressure like any gate, and once it falls the
// closing line says the gate opened after its wait, naming memory, never
// that it opened at once.
func TestImplementLoadRaceGateAtOneSlotSaysItWaitedOnMemory(t *testing.T) {
	waitFixture(t, machine(3, 3, 3))
	warn := machineload.Memory{Platform: machineload.MemoryDarwin, Level: machineload.MemoryWarn, Raw: "2"}
	normal := machineload.Memory{Platform: machineload.MemoryDarwin, Level: machineload.MemoryNormal, Raw: "1"}
	withMemory(t, warn, normal)
	code, out, errOut := implementCLI(t, "implement", "load", "--race-gate", "--timeout", "5m")
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	if !strings.Contains(out, "waiting for capacity: memory pressure warn (floor warn)") ||
		!strings.Contains(out, "race gate open after ") ||
		!strings.Contains(out, ": one preflight runs at a time, and what it waited on (memory) no longer holds the race stage back\n") ||
		strings.Contains(out, "at once") {
		t.Fatalf("stdout:\n%s", out)
	}
}

// TestImplementLoadWaitTimesOutOnMemory: pressure that outlasts --timeout
// ends the wait as a load wait ends, exiting 1 and naming memory, in text
// and in JSON.
func TestImplementLoadWaitTimesOutOnMemory(t *testing.T) {
	waitFixture(t, machine(3, 3, 3))
	withMemory(t, machineload.Memory{Platform: machineload.MemoryLinux, StallPct: 14.2, Raw: "some avg10=14.20"})
	code, out, errOut := implementCLI(t, "implement", "load", "--wait", "--timeout", "1m")
	if code != 1 || !strings.Contains(out, "LOAD GATE TIMED OUT after 1m0s: exit 1, nothing ran: memory stalled 14.2% over 10s (floor 10%)") {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	code, out, _ = implementCLI(t, "--json", "implement", "load", "--wait", "--timeout", "1m")
	var res struct {
		Status string   `json:"status"`
		Still  []string `json:"still"`
		Memory struct {
			StallPct      float64 `json:"stall_pct"`
			FloorStallPct int     `json:"floor_stall_pct"`
			AtFloor       bool    `json:"at_floor"`
		} `json:"memory"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("exit %d, json: %v\n%s", code, err, out)
	}
	if code != 1 || res.Status != implement.WaitTimedOut || len(res.Still) != 1 || res.Still[0] != implement.WaitOnMemory ||
		res.Memory.StallPct != 14.2 || res.Memory.FloorStallPct != 10 || !res.Memory.AtFloor {
		t.Fatalf("exit %d, json %+v", code, res)
	}
}

// TestImplementLoadWaitSaysMemoryWasNotChecked: a waiter that cannot read
// memory opens as before, and says once on stderr that the memory floor was
// not checked, and why.
func TestImplementLoadWaitSaysMemoryWasNotChecked(t *testing.T) {
	waitFixture(t, machine(3, 3, 3))
	prev := implement.ReadMemory
	implement.ReadMemory = func() (machineload.Memory, error) {
		return machineload.Memory{}, &machineload.MemoryUnsupportedError{Why: "/proc/pressure/memory is absent: the kernel's pressure stall information needs Linux 4.20 or later"}
	}
	t.Cleanup(func() { implement.ReadMemory = prev })
	code, out, errOut := implementCLI(t, "implement", "load", "--wait", "--timeout", "1m")
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	want := "abcd implement load --wait: the memory floor was not checked: /proc/pressure/memory is absent"
	if strings.Count(errOut, want) != 1 {
		t.Fatalf("stderr does not say once %q:\n%s", want, errOut)
	}
}

// TestLoadCheckRendersTheMemoryWarning: the load check prints a reading at
// the floor as a warning beside the load's own line, a note when memory was
// not checked, and nothing more below the floor.
func TestLoadCheckRendersTheMemoryWarning(t *testing.T) {
	res := implement.LoadResult{Status: implement.LoadOK, Site: implement.SitePreflight, Load: &implement.LoadAverages{One: 3}, Cores: 16,
		Memory: &implement.MemoryCheck{Platform: machineload.MemoryDarwin, Level: machineload.MemoryCritical, FloorLevel: machineload.MemoryWarn, AtFloor: true}}
	var b bytes.Buffer
	renderLoad(&b, res)
	if !strings.Contains(b.String(), "load check (preflight): nothing to warn about") ||
		!strings.Contains(b.String(), "MEMORY PRESSURE WARNING (preflight): memory pressure critical (floor warn)") {
		t.Fatalf("no memory warning:\n%s", b.String())
	}
	res.Memory.AtFloor, res.Memory.Level = false, machineload.MemoryNormal
	b.Reset()
	renderLoad(&b, res)
	if strings.Contains(b.String(), "MEMORY") || strings.Contains(b.String(), "memory") {
		t.Fatalf("a warning below the floor:\n%s", b.String())
	}
	res.Memory, res.MemoryNote = nil, "the memory floor was not checked: sysctl refused"
	b.Reset()
	renderLoad(&b, res)
	if !strings.Contains(b.String(), "memory check (preflight): the memory floor was not checked: sysctl refused") {
		t.Fatalf("no not-checked line:\n%s", b.String())
	}
}

// TestImplementLoadCheckWarnsOnMemoryAndExitsZero: the check run end to end
// at memory pressure warns and still exits 0.
func TestImplementLoadCheckWarnsOnMemoryAndExitsZero(t *testing.T) {
	loadFixture(t, machine(3, 3, 3), nil)
	withMemory(t, machineload.Memory{Platform: machineload.MemoryDarwin, Level: machineload.MemoryWarn, Raw: "2"})
	code, out, errOut := implementCLI(t, "implement", "load", "--site", "preflight")
	if code != 0 || !strings.Contains(out, "MEMORY PRESSURE WARNING (preflight)") {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
}
