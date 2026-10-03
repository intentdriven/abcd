package term

import (
	"os"
	"strconv"
	"strings"

	xterm "golang.org/x/term"
)

// The size a question is drawn at when nothing answers one: the narrow window
// abcd's question layout promises (itd-2610030810350727), 80 by 24.
const (
	defaultColumns = 80
	defaultRows    = 24
)

// Size is the window f is a terminal for, in columns and rows, read through
// golang.org/x/term before each question so a resize between questions is
// honoured (spc-2610030911534855). When f answers no size (it is not a
// terminal, or the terminal reports zero), the width is COLUMNS when that is
// a positive whole number and 80 otherwise, and the height is LINES, else 24.
//
// The descriptor is reached through SyscallConn rather than Fd, as IsTerminal
// does, so the file keeps its non-blocking mode.
func Size(f *os.File, getenv func(string) string) (cols, rows int) {
	if f != nil {
		if rc, err := f.SyscallConn(); err == nil {
			_ = rc.Control(func(fd uintptr) {
				if !isTerminalFd(fd) {
					return
				}
				if w, h, err := xterm.GetSize(int(fd)); err == nil {
					cols, rows = w, h
				}
			})
		}
	}
	if cols <= 0 {
		cols = positive(getenv("COLUMNS"), defaultColumns)
	}
	if rows <= 0 {
		rows = positive(getenv("LINES"), defaultRows)
	}
	return cols, rows
}

// positive reads s as a positive whole number, or answers def.
func positive(s string, def int) int {
	if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && n > 0 {
		return n
	}
	return def
}
