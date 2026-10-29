//go:build unix

package gitutil

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
// at the terminal reaches git as it would there. Since git is not in a group
// of its own, no one signal reaches the whole tree, so its Cancel freezes the
// tree before it signals it: it sends SIGSTOP to git, so git starts nothing
// more, then reads the process table and sends SIGSTOP to each of git's
// descendants as it finds them (a hook, the programs that hook started, a
// signing program git ran), reading the table again until a read finds none
// it had not stopped, since a descendant can start a process between a read
// and its own stop. It then sends SIGTERM to git (on which it removes its
// lock files) and to each of them, and SIGCONT to each process it stopped, on
// every path, so each acts on the SIGTERM. So a hook git would start, or a
// program a hook would start, while the stop is under way is never started,
// rather than escaping the stop and holding git's output for the whole
// WaitDelay (iss-2610292307181518). The reads share one read's time limit, so
// the stop takes no longer than it did with one. The sweep it returns, run
// once Wait has returned, waits until stopGrace has passed since the stop and
// then stops each process that is still running, with any process one of them
// started since, the same way, and sends each SIGKILL; it returns at once
// when none is left, and does
// nothing when the deadline was not reached: a program a hook leaves running
// behind a git that finished in time is the repository's, and is left alone.
//
// The sweep knows a process by its pid and its start time together, so a pid
// the system has handed to another program since is never killed; the stop
// signals by pid, in the moment after a read, while the processes it signals
// are stopped and cannot exit. Out of
// reach: a hook's child that detaches (its parent exits and it is reparented
// to init) before it is stopped, since nothing then ties it to git; a process
// a descendant starts after the last of the reads; a process a hook asks
// another to start (a gpg-agent's pinentry); and every descendant where the
// process table cannot be read, when git alone is stopped and continued.
//
// A tree stopped when abcd itself dies, before the SIGCONT, is continued only
// where abcd's death leaves its process group orphaned, when the system sends
// each stopped member SIGHUP and then SIGCONT. A process in the group whose
// parent is the shell (the other end of a pipe from abcd) keeps the group from
// being orphaned, and the tree then stays stopped until something continues
// it.
func stopWithDescendants(cmd *exec.Cmd) (sweep func()) {
	var mu sync.Mutex
	var stopped time.Time
	var tree []procEntry
	cmd.Cancel = func() error {
		// A git already reaped refuses the SIGSTOP and is not continued; one
		// that has exited and not been reaped takes it, and the SIGCONT, as a
		// no-op.
		gitStopped := cmd.Process.Signal(syscall.SIGSTOP) == nil
		var desc []procEntry
		defer func() {
			for _, p := range desc {
				_ = syscall.Kill(p.pid, syscall.SIGCONT)
			}
			if gitStopped {
				_ = cmd.Process.Signal(syscall.SIGCONT)
			}
		}()
		stopDescendants(&desc, cmd.Process.Pid)
		if afterSnapshot != nil {
			afterSnapshot()
		}
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
		deadline := at.Add(stopGrace)
		for {
			alive := stillRunning(readProcTable(), set)
			if len(alive) == 0 {
				return
			}
			if !time.Now().Before(deadline) {
				pids := make([]int, 0, len(alive))
				for _, p := range alive {
					_ = syscall.Kill(p.pid, syscall.SIGSTOP)
					pids = append(pids, p.pid)
				}
				stopDescendants(&alive, pids...)
				for _, p := range alive {
					_ = syscall.Kill(p.pid, syscall.SIGKILL)
				}
				return
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}

// stopRounds caps the process-table reads stopDescendants makes.
const stopRounds = 4

// stopDescendants sends SIGSTOP to every descendant of roots the process
// table shows that stopped does not already hold, appending each to stopped
// as it is stopped, so a caller continuing them on every path continues each
// one stopped so far. It reads the table again until a read finds none new,
// at most stopRounds times, all within procTableTimeout, and adds nothing
// where the table cannot be read.
func stopDescendants(stopped *[]procEntry, roots ...int) {
	seen := map[int]bool{}
	for _, p := range *stopped {
		seen[p.pid] = true
	}
	by := time.Now().Add(procTableTimeout)
	for range stopRounds {
		if !time.Now().Before(by) {
			return
		}
		fresh := false
		for _, p := range descendantsOf(readProcTableBy(by), roots) {
			if seen[p.pid] {
				continue
			}
			seen[p.pid] = true
			_ = syscall.Kill(p.pid, syscall.SIGSTOP)
			*stopped = append(*stopped, p)
			fresh = true
		}
		if !fresh {
			return
		}
	}
}

// afterSnapshot, when set, runs in the stop between the reads of git's
// descendants and the signals, so a test can hold the stop open there.
var afterSnapshot func()

// procEntry is one live process of the table: its pid, its parent's, and its
// start time as ps prints it, which with the pid names the process.
type procEntry struct {
	pid, ppid int
	start     string
}

// psPaths are where ps is looked for; never on PATH, which a repository's
// environment does not choose but abcd need not trust either.
var psPaths = []string{"/bin/ps", "/usr/bin/ps"}

// procTableTimeout bounds one read of the process table.
const procTableTimeout = 5 * time.Second

// readProcTable is every live process, zombies left out; nil when ps cannot
// be run or read within procTableTimeout.
func readProcTable() []procEntry {
	return readProcTableBy(time.Now().Add(procTableTimeout))
}

// readProcTableBy is readProcTable with the read bounded by deadline.
func readProcTableBy(deadline time.Time) []procEntry {
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
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
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
