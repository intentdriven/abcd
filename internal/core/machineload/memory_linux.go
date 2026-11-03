//go:build linux

package machineload

import "os"

// ReadMemory reads this Linux machine's memory pressure from
// /proc/pressure/memory with one bounded read (readPressureFS).
func ReadMemory() (Memory, error) {
	return readPressureFS(os.DirFS("/proc"))
}
