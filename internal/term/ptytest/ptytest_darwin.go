//go:build darwin

package ptytest

import (
	"bytes"
	"os"
	"syscall"
	"unsafe"
)

// getAttrs and setAttrs are tcgetattr's and tcsetattr's (TCSANOW) requests
// on darwin.
const (
	getAttrs = syscall.TIOCGETA
	setAttrs = syscall.TIOCSETA
)

// open opens /dev/ptmx and the terminal end it names: grantpt(3) and
// unlockpt(3) are the TIOCPTYGRANT and TIOCPTYUNLK requests, and ptsname(3)
// is TIOCPTYGNAME.
func open() (master, terminal *os.File, err error) {
	m, err := openFile("/dev/ptmx")
	if err != nil {
		return nil, nil, err
	}
	if err := ioctl(m, syscall.TIOCPTYGRANT, 0); err != nil {
		m.Close()
		return nil, nil, err
	}
	if err := ioctl(m, syscall.TIOCPTYUNLK, 0); err != nil {
		m.Close()
		return nil, nil, err
	}
	var name [128]byte
	if err := ioctl(m, syscall.TIOCPTYGNAME, uintptr(unsafe.Pointer(&name[0]))); err != nil {
		m.Close()
		return nil, nil, err
	}
	if i := bytes.IndexByte(name[:], 0); i >= 0 {
		s, err := openFile(string(name[:i]))
		if err != nil {
			m.Close()
			return nil, nil, err
		}
		return m, s, nil
	}
	m.Close()
	return nil, nil, syscall.EINVAL
}

// unreadRequest asks how many bytes the master end has still to read. On
// darwin FIONREAD on the master counts the terminal's input queue, the bytes
// typed and not yet read by the child, and reads 0 while the child's output
// waits: a probe with the drain held showed FIONREAD 0 and TIOCOUTQ 14 for 14
// bytes written. The child's output waits in the terminal's output queue, so
// darwin's request is TIOCOUTQ, which counts it from either end.
const unreadRequest = syscall.TIOCOUTQ
