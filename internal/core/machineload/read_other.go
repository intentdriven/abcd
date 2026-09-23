//go:build !darwin && !linux

package machineload

// Read reports ErrUnsupported: abcd reads the load and the process table on
// macOS and Linux only. itd-2609231434459890's first scope condition confines
// the load check to macOS and Linux developer machines, and its out-of-scope
// list names every other platform, so on this one the check says it could not
// check and carries on (spc-2609231542463113, section 8).
func Read() (Snapshot, error) { return Snapshot{}, ErrUnsupported }
