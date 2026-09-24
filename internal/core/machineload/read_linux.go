//go:build linux

package machineload

import (
	"encoding/binary"
	"os"
	"runtime"
	"strconv"
)

// Read reads this Linux machine from /proc and /sys alone, so a minimal or
// missing ps does not matter: the load averages from /proc/loadavg, the online
// core count from /sys/devices/system/cpu/online (what glibc's
// sysconf(_SC_NPROCESSORS_ONLN) reads; runtime.NumCPU counts the affinity mask
// and can be smaller), and every numeric /proc/<pid>. A load average that reads
// while the process table does not is returned with the error, so the extreme
// trigger can still apply.
func Read() (Snapshot, error) {
	snap, err := readProcFS(os.DirFS("/proc"), os.DirFS("/sys/devices/system/cpu"), strconv.IntSize/8, binary.NativeEndian)
	if snap.HasLoad && snap.Cores == 0 {
		snap.Cores, snap.CoresFallback = runtime.NumCPU(), true
	}
	return snap, err
}
