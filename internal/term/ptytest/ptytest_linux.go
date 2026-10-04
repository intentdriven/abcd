//go:build linux

package ptytest

import (
	"os"
	"strconv"
	"syscall"
	"unsafe"
)

// getAttrs and setAttrs are tcgetattr's and tcsetattr's (TCSANOW) requests
// on linux.
const (
	getAttrs = syscall.TCGETS
	setAttrs = syscall.TCSETS
)

// open opens /dev/ptmx and the terminal end it names: unlockpt(3) is the
// TIOCSPTLCK request with zero, ptsname(3) is TIOCGPTN, and grantpt(3) has
// nothing to do on devpts.
func open() (master, terminal *os.File, err error) {
	m, err := openFile("/dev/ptmx")
	if err != nil {
		return nil, nil, err
	}
	var unlock int32
	if err := ioctl(m, syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&unlock))); err != nil {
		m.Close()
		return nil, nil, err
	}
	var n uint32
	if err := ioctl(m, syscall.TIOCGPTN, uintptr(unsafe.Pointer(&n))); err != nil {
		m.Close()
		return nil, nil, err
	}
	s, err := openFile("/dev/pts/" + strconv.FormatUint(uint64(n), 10))
	if err != nil {
		m.Close()
		return nil, nil, err
	}
	return m, s, nil
}

// unreadRequest asks how many bytes the master end has still to read: on
// linux the master's own read buffer, FIONREAD, which the syscall package
// names TIOCINQ. (TIOCOUTQ on a pty counts nothing: a write on the terminal
// end goes straight to the master's buffer.)
const unreadRequest = syscall.TIOCINQ
