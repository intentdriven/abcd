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
