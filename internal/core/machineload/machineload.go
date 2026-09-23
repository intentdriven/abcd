// Package machineload reads the machine's load and process table and decides,
// from one snapshot, whether programs outside the running work are keeping the
// machine busy (itd-2609231434459890, spc-2609231542463113).
//
// It is a leaf: it imports only the standard library, so the one new primitive
// the load check needs brings no dependency with it. Everything that decides is a
// pure function of a Snapshot, so a two-day burner is a fixture value, never a
// process a test has to leave running. The per-platform readers only fill a
// Snapshot: macOS through sysctl and /bin/ps, Linux through /proc alone, and
// every other platform reports ErrUnsupported so the caller can say it could not
// check (read_other.go).
//
// Nothing here writes to stdout, signals a process or writes a file.
package machineload

import (
	"errors"
	"time"
)

// Proc is one process as the check sees it.
type Proc struct {
	PID  int
	PPID int
	PGID int
	// UID is the effective uid.
	UID uint32
	// Age is how long the process has existed.
	Age time.Duration
	// CPU is the process's cumulative user plus system CPU time.
	CPU time.Duration
	// Name is the executable's base name: never a path, never a command line.
	Name string
}

// Share is the process's lifetime CPU share in cores: cumulative CPU time over
// age, so 1.0 is one core flat out since it started. A multi-threaded process can
// exceed 1.0. A process with no measurable age has no share.
func (p Proc) Share() float64 {
	if p.Age <= 0 {
		return 0
	}
	return p.CPU.Seconds() / p.Age.Seconds()
}

// Snapshot is one reading of the machine.
type Snapshot struct {
	// Load1, Load5 and Load15 are the load averages over one, five and fifteen
	// minutes. They are meaningful only when HasLoad is true.
	Load1, Load5, Load15 float64
	HasLoad              bool
	// Cores is the online core count, the unit the load averages are read in.
	Cores int
	// CoresFallback is true when the online count could not be read and Cores is
	// runtime.NumCPU's count instead.
	CoresFallback bool
	// Procs is the process table. It is meaningful only when HasProcs is true: a
	// load average that reads while the process table does not is still a reading.
	Procs    []Proc
	HasProcs bool
}

// ErrUnsupported is returned by Read on a platform the readers do not cover.
var ErrUnsupported = errors.New("machineload: this platform's load and process table are not read")
