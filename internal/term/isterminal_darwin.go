//go:build darwin

package term

import (
	"syscall"
	"unsafe"
)

// isTerminalFd asks the kernel for fd's terminal attributes (TIOCGETA, the
// request tcgetattr(3) makes); only a terminal answers.
func isTerminalFd(fd uintptr) bool {
	var t syscall.Termios
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCGETA), uintptr(unsafe.Pointer(&t)))
	return errno == 0
}
