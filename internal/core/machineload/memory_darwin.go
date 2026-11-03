//go:build darwin

package machineload

import (
	"fmt"
	"syscall"
)

// ReadMemory reads this macOS machine's memory pressure level with one
// sysctl, kern.memorystatus_vm_pressure_level, which an ordinary account can
// read (parseMemoryPressureLevel says how its values map).
func ReadMemory() (Memory, error) {
	v, err := syscall.SysctlUint32("kern.memorystatus_vm_pressure_level")
	if err != nil {
		return Memory{}, fmt.Errorf("could not read the memory pressure level (sysctl kern.memorystatus_vm_pressure_level: %w)", err)
	}
	return parseMemoryPressureLevel(v), nil
}
