package machineload

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"strconv"
	"strings"
)

// Memory pressure is the load gate's third resource (itd-2611012048006057,
// spc-2611020732419632): on 2026-11-01 a 128 GB development machine ran out
// of application memory while several agent sessions built and reviewed at
// once, and nothing in the gate held a test stage back. The reading is the
// operating system's own pressure signal, the one it acts on before it runs
// out: macOS's pressure level (kern.memorystatus_vm_pressure_level), and on
// Linux the share of recent time some task stalled waiting for memory (the
// `some avg10` of /proc/pressure/memory, the kernel's pressure stall
// information, from 4.20). This file only reads and parses; the floor and
// the judgement live with the gate (internal/core/implement,
// load_memory.go).

// The platforms a Memory reading comes from.
const (
	MemoryDarwin = "darwin"
	MemoryLinux  = "linux"
)

// macOS's pressure levels, as a Memory names them.
const (
	MemoryNormal   = "normal"
	MemoryWarn     = "warn"
	MemoryUrgent   = "urgent"
	MemoryCritical = "critical"
)

// MemoryLevelRank orders macOS's pressure levels, normal lowest: a level is
// at or above another when its rank is. An unknown level ranks as critical,
// so a level abcd does not know never reads as below a floor.
func MemoryLevelRank(level string) int {
	switch level {
	case MemoryNormal:
		return 1
	case MemoryWarn:
		return 2
	case MemoryUrgent:
		return 3
	}
	return 4
}

// Memory is one reading of the machine's memory pressure.
type Memory struct {
	// Platform is MemoryDarwin or MemoryLinux.
	Platform string
	// Level is macOS's pressure level: MemoryNormal, MemoryWarn, MemoryUrgent
	// or MemoryCritical; "" on Linux.
	Level string
	// StallPct is Linux's share of the last ten seconds in which some task
	// stalled waiting for memory, a percentage; zero on macOS.
	StallPct float64
	// Raw is the value as the system gave it, for the report.
	Raw string
}

// MemoryUnsupportedError is a memory reading abcd cannot take on this
// machine: another platform, or a Linux kernel without the pressure stall
// information. It is errors.Is ErrUnsupported, and its message says why.
type MemoryUnsupportedError struct{ Why string }

func (e *MemoryUnsupportedError) Error() string { return e.Why }
func (e *MemoryUnsupportedError) Unwrap() error { return ErrUnsupported }

// memoryUnsupported is the reading on a platform with neither signal.
func memoryUnsupported(goos string) (Memory, error) {
	return Memory{}, &MemoryUnsupportedError{Why: "abcd reads memory pressure on macOS and Linux only, and this is " + goos}
}

// parseMemoryPressureLevel maps macOS's kern.memorystatus_vm_pressure_level,
// the dispatch levels the kernel reports to an ordinary account: 1 normal, 2
// warn, 4 critical, and 3, which xnu does not emit today (it reports its
// urgent level as warn), urgent. Any other value reads as critical, the
// highest level, so a value abcd does not know shuts the gate at either
// floor rather than opening it.
func parseMemoryPressureLevel(v uint32) Memory {
	level := MemoryCritical
	switch v {
	case 1:
		level = MemoryNormal
	case 2:
		level = MemoryWarn
	case 3:
		level = MemoryUrgent
	}
	return Memory{Platform: MemoryDarwin, Level: level, Raw: strconv.FormatUint(uint64(v), 10)}
}

// maxPressureBytes bounds the read of /proc/pressure/memory: two lines of
// about sixty bytes each.
const maxPressureBytes = 4 << 10

// parsePressureMemory reads Linux's /proc/pressure/memory:
//
//	some avg10=0.00 avg60=0.00 avg300=0.00 total=0
//	full avg10=0.00 avg60=0.00 avg300=0.00 total=0
//
// and returns the `some` line's avg10, a percentage from 0 to 100. A file
// without a `some` line, or whose `some` line carries no readable avg10, is
// an error naming what is missing.
func parsePressureMemory(raw []byte) (Memory, error) {
	sc := bufio.NewScanner(bytes.NewReader(raw))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] != "some" {
			continue
		}
		for _, f := range fields[1:] {
			k, v, ok := strings.Cut(f, "=")
			if !ok || k != "avg10" {
				continue
			}
			pct, err := strconv.ParseFloat(v, 64)
			if err != nil || math.IsNaN(pct) || pct < 0 || pct > 100 {
				return Memory{}, fmt.Errorf("/proc/pressure/memory's some avg10 is not a percentage (%q)", boundText(v))
			}
			return Memory{Platform: MemoryLinux, StallPct: pct, Raw: boundText(strings.Join(fields, " "))}, nil
		}
		return Memory{}, errors.New("/proc/pressure/memory's some line carries no avg10")
	}
	return Memory{}, errors.New("/proc/pressure/memory carries no some line")
}

// readPressureFS reads pressure/memory from proc (a view of /proc) with one
// bounded read. A missing file is a MemoryUnsupportedError naming the kernel
// requirement: the pressure stall information came in Linux 4.20, and a
// kernel may be built without it.
func readPressureFS(proc fs.FS) (Memory, error) {
	f, err := proc.Open("pressure/memory")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Memory{}, &MemoryUnsupportedError{Why: "/proc/pressure/memory is absent: the kernel's pressure stall information needs Linux 4.20 or later, built with it (CONFIG_PSI) and not turned off (psi=0)"}
		}
		return Memory{}, fmt.Errorf("could not read /proc/pressure/memory (%w)", err)
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, maxPressureBytes+1))
	if err != nil {
		return Memory{}, fmt.Errorf("could not read /proc/pressure/memory (%w)", err)
	}
	if len(raw) > maxPressureBytes {
		return Memory{}, fmt.Errorf("/proc/pressure/memory is larger than %d KiB", maxPressureBytes>>10)
	}
	return parsePressureMemory(raw)
}

// boundText cuts a value read from the system for a message.
func boundText(s string) string {
	if len(s) > 80 {
		return s[:77] + "..."
	}
	return s
}
