//go:build darwin

package machineload

import (
	"fmt"
	"syscall"
)

// statVolume reads the volume holding path through statfs(2).
func statVolume(path string) (Volume, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return Volume{}, fmt.Errorf("could not read the free space (statfs: %w)", err)
	}
	mount := make([]byte, 0, len(st.Mntonname))
	for _, c := range st.Mntonname {
		if c == 0 {
			break
		}
		mount = append(mount, byte(c))
	}
	return Volume{
		Free:  st.Bavail * uint64(st.Bsize),
		Total: st.Blocks * uint64(st.Bsize),
		ID:    fmt.Sprintf("%d:%d", st.Fsid.Val[0], st.Fsid.Val[1]),
		Mount: string(mount),
	}, nil
}
