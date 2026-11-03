//go:build !darwin && !linux

package machineload

import "runtime"

// ReadMemory reports a MemoryUnsupportedError: abcd reads memory pressure on
// macOS and Linux only, as it reads the load (read_other.go), and on any
// other platform the gate says the memory floor was not checked and admits.
func ReadMemory() (Memory, error) { return memoryUnsupported(runtime.GOOS) }
