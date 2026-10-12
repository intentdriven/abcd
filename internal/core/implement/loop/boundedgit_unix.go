//go:build unix

package loop

import (
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// stopWithDescendants makes cmd's deadline stop reach the processes git
// started, not git alone, while leaving git where the person's own git runs:
// in abcd's process group, so a signing prompt or a hook that asks on the
// terminal reads it as it would on the person's own commit, and an interrupt
// at the terminal reaches git as it would there. Its Cancel reads the process
// table and collects git's descendants (a hook, the programs that hook
// started, a signing program git ran), then sends SIGTERM to git (on which it
// removes its lock files) and to each of them. The sweep it returns, run once
// Wait has returned, waits until commitStopGrace has passed since the stop
// and then sends SIGKILL to each that is still running, with any process one
// of them started since; it returns at once when none is left, and does
// nothing when the deadline was not reached: a program a hook leaves running
// behind a git that finished in time is the repository's, and is left alone.
//
// A process is known by its pid and its start time together, so a pid the
// system has handed to another program since is never signalled. Out of
// reach: a hook's child that detaches (its parent exits and it is reparented
// to init) before the stop, since nothing then ties it to git; a process a
// hook asks another to start (a gpg-agent's pinentry); and every descendant
// where the process table cannot be read, when git alone is stopped, as
// before.
func stopWithDescendants(cmd *exec.Cmd) (sweep func()) {
	var mu sync.Mutex
	var stopped time.Time
	var tree []procEntry
	cmd.Cancel = func() error {
		desc := descendantsOf(readProcTable(), []int{cmd.Process.Pid})
		mu.Lock()
		stopped, tree = time.Now(), desc
		mu.Unlock()
		err := cmd.Process.Signal(syscall.SIGTERM)
		for _, p := range desc {
			_ = syscall.Kill(p.pid, syscall.SIGTERM)
		}
		if err != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
	return func() {
		mu.Lock()
		at, set := stopped, tree
		mu.Unlock()
		if at.IsZero() || len(set) == 0 {
			return
		}
		deadline := at.Add(commitStopGrace)
		for {
			alive := stillRunning(readProcTable(), set)
			if len(alive) == 0 {
				return
			}
			if !time.Now().Before(deadline) {
				for _, p := range alive {
					_ = syscall.Kill(p.pid, syscall.SIGKILL)
				}
				return
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}

// procEntry is one live process of the table: its pid, its parent's, and its
// start time as ps prints it, which with the pid names the process.
type procEntry struct {
	pid, ppid int
	start     string
}

// psPaths are where ps is looked for; never on PATH, which a repository's
// environment does not choose but abcd need not trust either.
var psPaths = []string{"/bin/ps", "/usr/bin/ps"}

// readProcTable is every live process, zombies left out; nil when ps cannot
// be run or read.
func readProcTable() []procEntry {
	ps := ""
	for _, p := range psPaths {
		if _, err := exec.LookPath(p); err == nil {
			ps = p
			break
		}
	}
	if ps == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, ps, "-A", "-o", "pid=", "-o", "ppid=", "-o", "stat=", "-o", "lstart=")
	c.Env = []string{"LC_ALL=C", "PATH=/usr/bin:/bin"}
	var out bytes.Buffer
	c.Stdout = &out
	if c.Run() != nil {
		return nil
	}
	return parseProcTable(out.String())
}

// parseProcTable reads readProcTable's ps output: pid, ppid, state, and the
// start time as the rest of the line.
func parseProcTable(out string) []procEntry {
	var table []procEntry
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) < 4 || strings.HasPrefix(f[2], "Z") {
			continue
		}
		pid, err1 := strconv.Atoi(f[0])
		ppid, err2 := strconv.Atoi(f[1])
		if err1 != nil || err2 != nil {
			continue
		}
		table = append(table, procEntry{pid: pid, ppid: ppid, start: strings.Join(f[3:], " ")})
	}
	return table
}

// descendantsOf is every process of table below the roots, the roots left
// out.
func descendantsOf(table []procEntry, roots []int) []procEntry {
	children := map[int][]procEntry{}
	for _, p := range table {
		children[p.ppid] = append(children[p.ppid], p)
	}
	var out []procEntry
	seen := map[int]bool{}
	queue := append([]int(nil), roots...)
	for len(queue) > 0 {
		pid := queue[0]
		queue = queue[1:]
		for _, c := range children[pid] {
			if seen[c.pid] {
				continue
			}
			seen[c.pid] = true
			out = append(out, c)
			queue = append(queue, c.pid)
		}
	}
	return out
}

// stillRunning is the processes of set that table still holds, the same pid
// started at the same time, with every process below them.
func stillRunning(table, set []procEntry) []procEntry {
	live := map[procEntry]bool{}
	for _, p := range table {
		live[procEntry{pid: p.pid, start: p.start}] = true
	}
	var alive []procEntry
	var pids []int
	for _, p := range set {
		if live[procEntry{pid: p.pid, start: p.start}] {
			alive = append(alive, p)
			pids = append(pids, p.pid)
		}
	}
	if len(pids) == 0 {
		return nil
	}
	known := map[int]bool{}
	for _, p := range alive {
		known[p.pid] = true
	}
	for _, p := range descendantsOf(table, pids) {
		if !known[p.pid] {
			alive = append(alive, p)
		}
	}
	return alive
}
