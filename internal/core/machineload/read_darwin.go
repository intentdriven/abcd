//go:build darwin

package machineload

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"
)

// psPath is ps by absolute path, so a planted `ps` earlier on PATH is never run.
const psPath = "/bin/ps"

// psTimeout bounds the one ps run; it takes tens of milliseconds for about two
// thousand processes.
const psTimeout = 20 * time.Second

// Read reads this macOS machine: the load averages from the vm.loadavg sysctl,
// the online core count from hw.activecpu (the value
// sysconf(_SC_NPROCESSORS_ONLN) reports), and the process table from one ps run.
// A load average that reads while the process table does not is returned with
// the error, so the extreme trigger can still apply.
func Read() (Snapshot, error) {
	var snap Snapshot
	raw, err := syscall.Sysctl("vm.loadavg")
	if err != nil {
		return snap, fmt.Errorf("could not read the load average (sysctl vm.loadavg: %w)", err)
	}
	if snap.Load1, snap.Load5, snap.Load15, err = ParseLoadavgSysctl([]byte(raw)); err != nil {
		return snap, fmt.Errorf("could not read the load average (%w)", err)
	}
	snap.HasLoad = true
	if n, err := syscall.SysctlUint32("hw.activecpu"); err == nil && n > 0 {
		snap.Cores = int(n)
	} else {
		snap.Cores, snap.CoresFallback = runtime.NumCPU(), true
	}

	ctx, cancel := context.WithTimeout(context.Background(), psTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, psPath, "-axo", "pid=,ppid=,pgid=,uid=,etime=,time=,comm=")
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return snap, fmt.Errorf("could not read the process table (%s: %w)", psPath, err)
	}
	procs, err := parseDarwinPS(out.Bytes())
	if err != nil {
		return snap, fmt.Errorf("could not read the process table (%w)", err)
	}
	snap.Procs, snap.HasProcs = procs, true
	return snap, nil
}
