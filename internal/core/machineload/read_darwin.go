//go:build darwin

package machineload

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// psPath is ps by absolute path, so a planted `ps` earlier on PATH is never run.
const psPath = "/bin/ps"

// psTimeout bounds the one ps run; it takes tens of milliseconds for about two
// thousand processes.
const psTimeout = 20 * time.Second

// runPS runs one ps command; a var only so a test can count the runs.
var runPS = func(cmd *exec.Cmd) error { return cmd.Run() }

// Read reads this macOS machine: the load averages from the vm.loadavg sysctl,
// the online core count from hw.activecpu (the value
// sysconf(_SC_NPROCESSORS_ONLN) reports), and the process table from one ps run.
// A load average that reads while the process table does not is returned with
// the error, so the extreme trigger can still apply.
func Read() (Snapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), psTimeout)
	defer cancel()
	return ReadContext(ctx)
}

// ReadContext is Read bound to ctx as well as to ps's own psTimeout: a ctx
// already ended reads nothing and starts no ps, answering ctx's error, and one
// that ends while ps runs kills it. A reader on a deadline (the away mode's
// stop check) reads the machine through it.
func ReadContext(ctx context.Context) (Snapshot, error) {
	var snap Snapshot
	if err := ctx.Err(); err != nil {
		return snap, err
	}
	raw, err := syscall.Sysctl("vm.loadavg")
	if err != nil {
		return snap, fmt.Errorf("could not read the load average (sysctl vm.loadavg: %w)", err)
	}
	if snap.Load1, snap.Load5, snap.Load15, err = parseLoadavgSysctl([]byte(raw)); err != nil {
		return snap, fmt.Errorf("could not read the load average (%w)", err)
	}
	snap.HasLoad = true
	if n, err := syscall.SysctlUint32("hw.activecpu"); err == nil && n > 0 {
		snap.Cores = int(n)
	} else {
		snap.Cores, snap.CoresFallback = runtime.NumCPU(), true
	}

	ctx, cancel := context.WithTimeout(ctx, psTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, psPath, "-axo", "pid=,ppid=,pgid=,uid=,etime=,time=,comm=")
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := ctx.Err(); err != nil {
		return snap, err
	}
	if err := runPS(cmd); err != nil {
		return snap, fmt.Errorf("could not read the process table (%s: %w)", psPath, err)
	}
	procs, err := parseDarwinPS(out.Bytes())
	if err != nil {
		return snap, fmt.Errorf("could not read the process table (%w)", err)
	}
	snap.Procs, snap.HasProcs = procs, true
	return snap, nil
}

// ReadArgs reads the arguments of each of pids, the program name dropped, from
// one ps run (parseDarwinArgs says how the words are split). A pid that has
// exited is absent from the result; ps exits 1 when none of them is left, which
// is an empty result, not an error.
func ReadArgs(pids []int) (map[int][]string, error) {
	if len(pids) == 0 {
		return map[int][]string{}, nil
	}
	list := make([]string, len(pids))
	for i, p := range pids {
		list[i] = strconv.Itoa(p)
	}
	ctx, cancel := context.WithTimeout(context.Background(), psTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, psPath, "-o", "pid=,args=", "-p", strings.Join(list, ","))
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	var exit *exec.ExitError
	if err != nil && !(errors.As(err, &exit) && exit.ExitCode() == 1 && out.Len() == 0) {
		return nil, fmt.Errorf("could not read the arguments of the go processes (%s: %w)", psPath, err)
	}
	return parseDarwinArgs(out.Bytes()), nil
}
