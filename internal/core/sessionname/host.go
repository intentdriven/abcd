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
	// Slack is how far StartUnix may be early: the length of the table read
	// it came from, capped at maxReadSlack. The read's moment is taken before
	// the read and each age somewhere inside it, so a start time is early by
	// up to the read's length and never late. Zero is a fast read, and an
	// entry written before the length was recorded.
	Slack time.Duration
}

// Table is one reading of the process table and the moment the read began,
// which is what a process's age is measured from.
type Table struct {
	Procs  []machineload.Proc
	ReadAt time.Time
	// Took is how long the read took: how far, at most, a start time computed
	// from the table is early.
	Took time.Duration
}

// startTolerance is how far two readings of one process's start time may
// differ and still name the same process. macOS reports a process's age in
// whole seconds and Linux in clock ticks against an uptime read a moment
// apart, so one process read twice can differ by a second; a different
// process given the same id starts at least the time it took the old one to
// end and the id to come round, which is far longer.
const startTolerance = 2

// maxReadSlack caps how far a start time is allowed to be early for a slow
// read. It sits above the twenty seconds macOS's ps run is given before it is
// killed (a read that long fails and names nothing), so no read that names a
// session is cut short; and it bounds how long after a host's recorded start a
// different process given the same id can still pass for it, which takes the
// host ending and the id coming round inside that window.
const maxReadSlack = 30 * time.Second

// readSlack is the slack a read of length took gives: took, kept within zero
// and maxReadSlack.
func readSlack(took time.Duration) time.Duration {
	return min(max(took, 0), maxReadSlack)
}

// slackSecs is a slack in whole seconds, rounded up: start times are whole
// seconds.
func slackSecs(slack time.Duration) int64 {
	slack = readSlack(slack)
	return int64((slack + time.Second - 1) / time.Second)
}

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
			return t.host(p), nil
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

// host is the table's reading of p: its start time measured back from the
// moment the read began, early by up to the read's length.
func (t Table) host(p machineload.Proc) Host {
	return Host{PID: p.PID, StartUnix: t.ReadAt.Add(-p.Age).Unix(), Slack: readSlack(t.Took)}
}

// Alive reports whether h is running: its process id is in the table, and the
// process holding it started when h says, within startTolerance widened by
// each reading's slack (C9).
func (t Table) Alive(h Host) bool {
	if h.PID <= 1 {
		return false
	}
	for _, p := range t.Procs {
		if p.PID != h.PID {
			continue
		}
		return sameHost(t.host(p), h)
	}
	return false
}

// sameHost reports whether two readings name one process: the same id, and
// start times within startTolerance of each other, widened by each reading's
// slack on the side it can err. A reading is early by up to its slack and
// never late, so a may sit below b by up to a's slack, and above it by up to
// b's.
func sameHost(a, b Host) bool {
	d := a.StartUnix - b.StartUnix
	return a.PID == b.PID && d >= -(startTolerance+slackSecs(a.Slack)) && d <= startTolerance+slackSecs(b.Slack)
}
