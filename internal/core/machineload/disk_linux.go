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
	// Block counts are in fragment-size units (f_frsize), as statvfs(3) and
	// df read them; f_bsize is the preferred I/O size, which differs on some
	// FUSE mounts (blockBytes).
	return Volume{
		Free:  blockBytes(uint64(st.Bavail), int64(st.Frsize), int64(st.Bsize)),
		Total: blockBytes(uint64(st.Blocks), int64(st.Frsize), int64(st.Bsize)),
		ID:    volumeID(int64(st.Fsid.X__val[0]), int64(st.Fsid.X__val[1])),
	}, nil
}
