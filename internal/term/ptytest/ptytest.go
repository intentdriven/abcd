// Package ptytest opens a pseudo-terminal for tests, with the standard library
// alone: /dev/ptmx and the platform's grant and unlock requests, no module
// (spc-2610030911534855, B6). It is a test helper: no production package
// imports it. The answer loop's restore tests run a child on the terminal end
// and read its attributes before and after; guided connect's hidden key read
// (spc-2610031241482088, TestHiddenKeyRestoresTerminalOnInterrupt) reuses it.
//
// Both ends are opened blocking and outside the runtime's poller, so a child
// handed the terminal end inherits an ordinary blocking descriptor.
package ptytest

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

// Pty is one pseudo-terminal: Master is the controlling end a test types into
// and reads the child's output from, Terminal the end a child takes as its
// stdin, stdout and stderr.
type Pty struct {
	Master   *os.File
	Terminal *os.File

	mu   sync.Mutex
	out  bytes.Buffer
	done chan struct{}
}

// Open opens a pseudo-terminal, sizes it cols by rows, and starts draining the
// master end into a buffer Output and WaitFor read, so a child never blocks
// on a full terminal. Both ends are closed when the test ends.
func Open(t testing.TB, cols, rows int) *Pty {
	t.Helper()
	m, s, err := open()
	if err != nil {
		t.Fatalf("opening a pseudo-terminal: %v", err)
	}
	p := &Pty{Master: m, Terminal: s, done: make(chan struct{})}
	if err := SetSize(s, cols, rows); err != nil {
		m.Close()
		s.Close()
		t.Fatalf("sizing the pseudo-terminal: %v", err)
	}
	go p.drain()
	t.Cleanup(p.Close)
	return p
}

// Close closes the terminal end, then the master end, which ends the drain.
func (p *Pty) Close() {
	p.Terminal.Close()
	p.Master.Close()
}

func (p *Pty) drain() {
	defer close(p.done)
	buf := make([]byte, 4096)
	for {
		n, err := p.Master.Read(buf)
		if n > 0 {
			p.mu.Lock()
			p.out.Write(buf[:n])
			p.mu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

// Output is everything the child has written so far.
func (p *Pty) Output() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.out.String()
}

// WaitFor waits until the output after offset bytes holds sub, and returns
// the offset just past it. It fails the test after timeout.
func (p *Pty) WaitFor(t testing.TB, offset int, sub string, timeout time.Duration) int {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		out := p.Output()
		if offset > len(out) {
			offset = len(out)
		}
		if i := strings.Index(out[offset:], sub); i >= 0 {
			return offset + i + len(sub)
		}
		if time.Now().After(deadline) {
			t.Fatalf("waited %s for %q; the terminal shows:\n%q", timeout, sub, out)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// Type writes s to the master end, as keys typed at the terminal.
func (p *Pty) Type(t testing.TB, s string) {
	t.Helper()
	if _, err := io.WriteString(p.Master, s); err != nil {
		t.Fatalf("typing %q: %v", s, err)
	}
}

// Attrs reads f's terminal attributes, as tcgetattr(3) does.
func Attrs(f *os.File) (syscall.Termios, error) {
	var tio syscall.Termios
	err := ioctl(f, getAttrs, uintptr(unsafe.Pointer(&tio)))
	return tio, err
}

// Same reports whether two sets of terminal attributes are the same, the
// kernel's PENDIN bit aside: a BSD kernel sets it whenever canonical mode is
// entered again, to mark input for retyping, so a terminal restored exactly
// can read back with it set.
func Same(a, b syscall.Termios) bool {
	a.Lflag &^= syscall.PENDIN
	b.Lflag &^= syscall.PENDIN
	return a == b
}

// SetAttrs writes f's terminal attributes, as tcsetattr(3) with TCSANOW does.
func SetAttrs(f *os.File, tio syscall.Termios) error {
	return ioctl(f, setAttrs, uintptr(unsafe.Pointer(&tio)))
}

// winsize is struct winsize, the same on darwin and linux.
type winsize struct {
	Row, Col, X, Y uint16
}

// SetSize sets the window size of the terminal f is an end of.
func SetSize(f *os.File, cols, rows int) error {
	ws := winsize{Row: uint16(rows), Col: uint16(cols)}
	return ioctl(f, syscall.TIOCSWINSZ, uintptr(unsafe.Pointer(&ws)))
}

// ioctl makes one request on f's descriptor.
func ioctl(f *os.File, req, arg uintptr) error {
	rc, err := f.SyscallConn()
	if err != nil {
		return err
	}
	var errno syscall.Errno
	if err := rc.Control(func(fd uintptr) {
		_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, fd, req, arg)
	}); err != nil {
		return err
	}
	if errno != 0 {
		return errno
	}
	return nil
}

// openFile opens path blocking, outside the runtime's poller.
func openFile(path string) (*os.File, error) {
	fd, err := syscall.Open(path, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return os.NewFile(uintptr(fd), path), nil
}
