package sessionname

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/machineload"
)

// Host is the process a session's hooks run under: the host program itself,
// not the shell a hook command starts in. A register entry is keyed on it, so
// its liveness is the session's (decision 4).
type Host struct {
	// PID is the host process's id.
	PID int
	// StartUnix is when it started, in Unix seconds: the table's read time
	// minus the process's age. With the pid it tells the process apart from a
	// later one the kernel gave the same id.
	StartUnix int64
}

// Table is one reading of the process table and the moment it was read, which
// is what a process's age is measured from.
type Table struct {
	Procs  []machineload.Proc
	ReadAt time.Time
}

// startTolerance is how far two readings of one process's start time may
// differ and still name the same process. macOS reports a process's age in
// whole seconds and Linux in clock ticks against an uptime read a moment
// apart, so one process read twice can differ by a second; a different
// process given the same id starts at least the time it took the old one to
// end and the id to come round, which is far longer.
const startTolerance = 2

// shells are the programs a hook command runs in. The host walk passes over
// them: they are the plumbing between the host and abcd, and a shell started
// for one hook is gone the moment the hook ends.
var shells = []string{"sh", "bash", "zsh", "dash", "ksh", "fish"}

// maxWalk bounds the host walk. A real chain is three or four deep; a table
// whose parent links loop would otherwise never end it.
const maxWalk = 64

// FindHost walks up from the process from (the hook process's parent) to the
// first process that is not a shell and not abcd, and returns it with its
// start time. A walk that leaves the table, loops, or reaches the machine's
// first process (pid 1, launchd or init) names no host: keyed on pid 1, an
// entry would never end.
func FindHost(t Table, from int) (Host, error) {
	byPID := make(map[int]machineload.Proc, len(t.Procs))
	for _, p := range t.Procs {
		byPID[p.PID] = p
	}
	pid := from
	for range maxWalk {
		if pid <= 1 {
			return Host{}, errors.New("no host process was found above the hook: the walk reached the machine's first process")
		}
		p, ok := byPID[pid]
		if !ok {
			return Host{}, fmt.Errorf("no host process was found above the hook: process %d is not in the process table", pid)
		}
		if !passOver(p.Name) {
			return Host{PID: p.PID, StartUnix: t.ReadAt.Add(-p.Age).Unix()}, nil
		}
		pid = p.PPID
	}
	return Host{}, errors.New("no host process was found above the hook: its parent links do not end")
}

// passOver reports whether the walk passes over a process of this name: a
// shell (a login shell's name carries a leading dash) or abcd itself, under
// its plain name or a release asset's (abcd-darwin-arm64).
func passOver(name string) bool {
	name = strings.TrimPrefix(name, "-")
	return slices.Contains(shells, name) || name == "abcd" || strings.HasPrefix(name, "abcd-")
}

// Alive reports whether h is running: its process id is in the table, and the
// process holding it started when h says, within startTolerance (C9).
func (t Table) Alive(h Host) bool {
	if h.PID <= 1 {
		return false
	}
	for _, p := range t.Procs {
		if p.PID != h.PID {
			continue
		}
		d := t.ReadAt.Add(-p.Age).Unix() - h.StartUnix
		return d >= -startTolerance && d <= startTolerance
	}
	return false
}
