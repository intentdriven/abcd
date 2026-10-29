//go:build linux

package machineload

import (
	"fmt"
	"syscall"
)

// statVolume reads the volume holding path through statfs(2). Linux names no
// mount point there, and the gate does not parse the mount table for one: the
// path read names the volume well enough.
func statVolume(path string) (Volume, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return Volume{}, fmt.Errorf("could not read the free space (statfs: %w)", err)
	}
	return Volume{
		Free:  uint64(st.Bavail) * uint64(st.Bsize),
		Total: uint64(st.Blocks) * uint64(st.Bsize),
		ID:    fmt.Sprintf("%d:%d", st.Fsid.X__val[0], st.Fsid.X__val[1]),
	}, nil
}
