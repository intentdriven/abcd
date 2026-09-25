package memory

import (
	"syscall"
	"testing"
)

// TestStoreLockModeGuardMasksTheFileType pins iss-2608261133210491. The
// store-lock guard tested st.Mode&S_IFREG != 0, but the file-type field is an
// enumeration, not a set of flags: a socket (S_IFSOCK) and a symlink (S_IFLNK)
// both carry the S_IFREG bit, so the "regular file" assertion admitted them.
// The type is compared under the S_IFMT mask.
func TestStoreLockModeGuardMasksTheFileType(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode uint32
		want bool
	}{
		{"regular", syscall.S_IFREG | 0o600, true},
		{"socket", syscall.S_IFSOCK | 0o600, false},
		{"symlink", syscall.S_IFLNK | 0o777, false},
		{"directory", syscall.S_IFDIR | 0o700, false},
		{"fifo", syscall.S_IFIFO | 0o600, false},
	} {
		if got := lockModeIsRegular(uint32(tc.mode)); got != tc.want {
			t.Errorf("lockModeIsRegular(%s) = %v, want %v", tc.name, got, tc.want)
		}
	}
}
