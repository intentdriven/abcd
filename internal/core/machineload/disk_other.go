//go:build !darwin && !linux

package machineload

// statVolume reports ErrUnsupported: abcd reads free space on macOS and Linux
// only, as it reads the load (read_other.go), and on any other platform the
// gate says it could not read the disk and never refuses for it.
func statVolume(string) (Volume, error) { return Volume{}, ErrUnsupported }
