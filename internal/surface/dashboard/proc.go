package dashboard

import "strings"

// executableOf is the executable a process runs, as /proc's link names it,
// read as the same file start recorded. The kernel appends " (deleted)" to
// the link once the file is replaced or removed, as `abcd update` replaces
// the binary a running server was started from, and that process is no less
// the one start launched: its pid and start time still say so.
func executableOf(link string) string {
	return strings.TrimSuffix(link, " (deleted)")
}
