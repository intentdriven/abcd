package dashboard

import (
	"errors"
	"io/fs"
	"strings"
	"syscall"
)

// processGone reports whether err, from a read of a process's kernel entry,
// means the process is gone. An entry already removed reads as ENOENT, and a
// read that races the exit, while the entry still exists, fails with ESRCH:
// both are a process that is no longer there.
func processGone(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ESRCH)
}

// executableOf is the executable a process runs, as /proc's link names it,
// read as the same file start recorded. The kernel appends " (deleted)" to
// the link once the file is replaced or removed, as `abcd update` replaces
// the binary a running server was started from, and that process is no less
// the one start launched: its pid and start time still say so.
func executableOf(link string) string {
	return strings.TrimSuffix(link, " (deleted)")
}
